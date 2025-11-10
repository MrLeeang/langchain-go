package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/MrLeeang/langchain-go/llms"

	openai "github.com/sashabaranov/go-openai"
)

// StreamResponse represents a single chunk of streamed content from the agent.
type StreamResponse struct {

	// ReasoningContent is the reasoning content in this chunk.
	ReasoningContent string

	// Content is the text content in this chunk.
	Content string

	// Done indicates whether the stream is complete.
	Done bool

	// Error contains any error that occurred during streaming.
	Error error
}

// Stream processes a user message and returns a channel that streams the agent's response.
// This method supports streaming for direct answers but requires complete responses for tool calls.
//
// Example:
//
//	ch := agent.Stream("What's the weather like?")
//	for resp := range ch {
//	    if resp.Error != nil {
//	        log.Printf("Error: %v", resp.Error)
//	        break
//	    }
//	    fmt.Print(resp.Content)
//	    if resp.Done {
//	        break
//	    }
//	}
func (a *Agent) Stream(message string) <-chan StreamResponse {
	a.ResetTokenUsage()

	a.LoadMessages(message)

	return a.StreamWithContext(a.ctx, message)
}

// StreamWithContext processes a user message with a custom context and returns a channel that streams the response.
func (a *Agent) StreamWithContext(ctx context.Context, message string) <-chan StreamResponse {
	ch := make(chan StreamResponse, 10)

	go func() {

		defer func() {
			time.Sleep(1 * time.Second)
			close(ch)
		}()

		userMsg := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: message,
		}
		a.messages = append(a.messages, userMsg)

		// Calculate prompt token usage for all messages
		for _, msg := range a.messages {
			if msg.Role == openai.ChatMessageRoleUser || msg.Role == openai.ChatMessageRoleSystem {
				a.CalculatePromptTokenUsage(msg.Content)
			}
		}

		// Save user message to memory
		if a.mem != nil && a.conversationID != "" {
			if err := a.mem.SaveMessages(ctx, a.conversationID, []openai.ChatCompletionMessage{userMsg}); err != nil {
				// Log error but continue
			}
		}

		iterations := 0
		for iterations < a.maxIter {
			iterations++

			// Check if LLM supports streaming
			streamer, ok := a.llm.(llms.ChatStreamer)
			if !ok {
				// Fallback to non-streaming if LLM doesn't support streaming
				resp, err := a.llm.Chat(ctx, a.messages)
				if err != nil {
					ch <- StreamResponse{Error: fmt.Errorf("failed to get LLM response: %w", err)}
					return
				}

				if len(resp.Choices) == 0 {
					ch <- StreamResponse{Error: fmt.Errorf("no response from LLM")}
					return
				}

				output := resp.Choices[0].Message.Content
				result, shouldContinue, err := a.handleLLMResponse(ctx, output)
				if err != nil {
					ch <- StreamResponse{Error: err}
					return
				}

				if !shouldContinue {
					ch <- StreamResponse{Done: true}
					return
				}

				if result != "" {
					ch <- StreamResponse{Content: result}
				}
			}

			// Use streaming
			stream, err := streamer.ChatStream(ctx, a.messages)
			if err != nil {
				ch <- StreamResponse{Error: fmt.Errorf("failed to create stream: %w", err)}

				return
			}

			var fullContent string

			buffer := ""
			isAssistantContent := false
			toolJSONStartFound := false

			for {
				response, err := stream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}

				if err != nil {
					stream.Close()
					ch <- StreamResponse{Error: fmt.Errorf("stream error: %w", err)}
					return
				}

				if len(response.Choices) > 0 && response.Choices[0].Delta.ReasoningContent != "" {
					ch <- StreamResponse{ReasoningContent: response.Choices[0].Delta.ReasoningContent}
				}

				if len(response.Choices) > 0 && response.Choices[0].Delta.Content != "" {
					content := response.Choices[0].Delta.Content
					fullContent += content

					if a.debug {
						ch <- StreamResponse{Content: content}
						continue
					}

					// If we've already determined it's plain assistant text, stream through
					if isAssistantContent {
						ch <- StreamResponse{Content: content}
						continue
					}

					// Accumulate and detect tool JSON that may start mid-stream
					buffer += content

					// If we haven't started parsing JSON yet, look for the start marker
					if !toolJSONStartFound {
						flag := `{"action"`

						if len(buffer) > len(flag) {
							idx := strings.Index(buffer, flag)
							if idx != -1 {
								// found JSON
								if idx > 0 {
									ch <- StreamResponse{Content: buffer[:idx]}
								}

								// Keep only the JSON part in buffer going forward
								buffer = buffer[idx:]
								toolJSONStartFound = true

							} else {
								// not found, output one character to slide
								bufferRunes := []rune(buffer)
								flagRunes := []rune(flag)

								if len(bufferRunes) > len(flagRunes) {
									ch <- StreamResponse{Content: string(bufferRunes[:len(bufferRunes)-len(flagRunes)])}
									buffer = string(bufferRunes[len(bufferRunes)-len(flagRunes):])
								}

							}
						}

					}

					// If we're inside a JSON tool payload, track braces until complete
					if toolJSONStartFound {
						// tool use found, return the tool use
					}
				}
			}

			if len(buffer) > 0 && !toolJSONStartFound {
				// stream end but buffer is not empty, output the buffer
				ch <- StreamResponse{Content: buffer}
				buffer = ""
			}

			stream.Close()

			// Process the complete response (handleStreamResponse will save the message)
			if fullContent != "" {
				// if a.debug {
				// 	fmt.Println("\n=============fullContent before handleStreamResponse============")
				// 	fmt.Printf("fullContent length: %d\n", len(fullContent))
				// 	fmt.Printf("fullContent preview (first 200 chars): %s\n", func() string {
				// 		if len(fullContent) > 200 {
				// 			return fullContent[:200] + "..."
				// 		}
				// 		return fullContent
				// 	}())
				// 	fmt.Println("=============fullContent before handleStreamResponse============")
				// }

				a.CalculateCompletionTokenUsage(fullContent)

				// Check if this is a final answer or tool call
				// handleStreamResponse will save the assistant message to memory
				_, shouldContinue, err := a.handleStreamResponse(ctx, ch, fullContent)
				if err != nil {
					ch <- StreamResponse{Error: err}
					return
				}

				if !shouldContinue {
					// Final answer - already streamed, just mark as done
					ch <- StreamResponse{Done: true}
					return
				}

				// Tool call detected and handled - continue iteration
				// The tool result has already been sent to channel in handleStreamResponse
			}
		}

		ch <- StreamResponse{Error: fmt.Errorf("max iterations (%d) exceeded", a.maxIter)}
	}()

	return ch
}

// handleStreamResponse handles a complete LLM response in streaming context.
// It returns the result, whether to continue, and any error.
// For tool calls, it executes the tool and sends the result through the channel.
func (a *Agent) handleStreamResponse(ctx context.Context, ch chan<- StreamResponse, response string) (string, bool, error) {
	var resp struct {
		Action string                 `json:"action"`
		Tool   string                 `json:"tool,omitempty"`
		Args   map[string]interface{} `json:"args,omitempty"`
		Answer string                 `json:"answer,omitempty"`
	}

	// Save assistant message to memory first
	if response != "" {
		assistantMsg := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: response,
		}
		a.messages = append(a.messages, assistantMsg)

		if a.mem != nil && a.conversationID != "" {
			_ = a.mem.SaveMessages(ctx, a.conversationID, []openai.ChatCompletionMessage{assistantMsg})
		}
	}

	// if a.debug {
	// 	fmt.Println("\n=============handleStreamResponse============")
	// 	fmt.Println("handleStreamResponse: ", response)
	// 	fmt.Println("=============handleStreamResponse============")
	// }

	// find call_tool action in response
	callToolJson := ""
	idx := strings.Index(response, `{"action":"call_tool"`)
	if idx != -1 {
		callToolJson = response[idx:]
	}

	if callToolJson == "" {
		// if a.debug {
		// 	fmt.Println("\ncallToolJson is empty: ", response)
		// }
		return response, false, nil
	}

	callToolJson = thoroughlyCleanJSON(callToolJson)

	if callToolJson != "" {
		err := json.Unmarshal([]byte(callToolJson), &resp)
		if err != nil {
			// if a.debug {
			// 	fmt.Println("=============unmarshalling============")
			// 	fmt.Println("error unmarshalling callToolJson: ", err, ", callToolJson: ", callToolJson)
			// 	fmt.Println("=============unmarshalling============")
			// }
			return response, false, nil
		}
	}

	switch resp.Action {
	case "final_answer":
		return resp.Answer, false, nil

	case "call_tool":
		if resp.Tool == "" {
			// if a.debug {
			// 	fmt.Println("\ntool name is required for call_tool action")
			// }
			return "", false, fmt.Errorf("tool name is required for call_tool action")
		}

		callToolResult := a.newCallToolResult(resp.Tool, resp.Args)

		// Send notification about tool call, json string

		ch <- StreamResponse{Content: "\n"}
		ch <- StreamResponse{Content: callToolJson}

		tool := a.findTool(resp.Tool)
		if tool == nil {
			ch <- StreamResponse{Content: "\n"}
			callToolResult.Error = true
			callToolResult.Message = fmt.Sprintf("tool '%s' not found", resp.Tool)
			ch <- StreamResponse{Content: callToolResult.String()}
			return "", false, fmt.Errorf("tool not found: %s", resp.Tool)
		}

		toolResult, err := tool.Call(ctx, resp.Args)
		if err != nil {
			callToolResult.Error = true
			callToolResult.Message = fmt.Sprintf("tool call failed for %s: %v", resp.Tool, err)
			ch <- StreamResponse{Content: "\n"}
			ch <- StreamResponse{Content: callToolResult.String()}
			return "", false, fmt.Errorf("tool call failed for %s: %w", resp.Tool, err)
		}

		// Send tool result through channel
		ch <- StreamResponse{Content: "\n"}
		callToolResult.Result = toolResult
		ch <- StreamResponse{Content: callToolResult.String()}
		ch <- StreamResponse{Content: "\n"}

		// Add tool result to conversation and continue
		toolMessage := fmt.Sprintf("Tool %s returned: %s", resp.Tool, toolResult)
		msg := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: toolMessage,
		}
		a.messages = append(a.messages, msg)

		// Save tool message to memory
		if a.mem != nil && a.conversationID != "" {
			_ = a.mem.SaveMessages(ctx, a.conversationID, []openai.ChatCompletionMessage{msg})
		}

		return "", true, nil

	default:
		// Unknown action, treat as final answer
		return response, false, nil
	}
}

// handleLLMResponse handles a complete LLM response (non-streaming fallback).
func (a *Agent) handleLLMResponse(ctx context.Context, output string) (string, bool, error) {
	assistantMsg := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: output,
	}
	a.messages = append(a.messages, assistantMsg)

	// Save assistant message to memory
	if a.mem != nil && a.conversationID != "" {
		_ = a.mem.SaveMessages(ctx, a.conversationID, []openai.ChatCompletionMessage{assistantMsg})
	}

	return a.parseLLMResponse(ctx, output)
}

// call tool result data
type callToolResult struct {
	Action  string                 `json:"action"`
	Tool    string                 `json:"tool"`
	Args    map[string]interface{} `json:"args"`
	Result  string                 `json:"result"`
	Error   bool                   `json:"error"`
	Message string                 `json:"message"`
}

func (c *callToolResult) String() string {
	json, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return string(json)
}

func (a *Agent) newCallToolResult(tool string, args map[string]interface{}) *callToolResult {
	return &callToolResult{
		Action:  "tool_result",
		Tool:    tool,
		Args:    args,
		Result:  "",
		Error:   false,
		Message: "",
	}
}
