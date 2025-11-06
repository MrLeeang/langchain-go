package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/MrLeeang/langchain-go/llms"
	"github.com/MrLeeang/langchain-go/memory"

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
	if a.mem != nil {
		if milvusMem, ok := a.mem.(memory.MilvusMemoryInterface); ok {
			milvusMem.SetQuery(message)
		}
	}
	return a.StreamWithContext(a.ctx, message)
}

// StreamWithContext processes a user message with a custom context and returns a channel that streams the response.
func (a *Agent) StreamWithContext(ctx context.Context, message string) <-chan StreamResponse {
	ch := make(chan StreamResponse, 10)

	go func() {
		defer close(ch)

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
				a.handleLLMResponse(ctx, ch, output)
				return
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

					// If we've already determined it's plain assistant text, stream through
					if isAssistantContent {
						ch <- StreamResponse{Content: content}
						continue
					}

					// Accumulate and detect tool JSON that may start mid-stream
					buffer += content

					// If we haven't started parsing JSON yet, look for the start marker
					if !toolJSONStartFound {
						idx := strings.Index(buffer, `{"action"`)
						fmt.Println("idx:", idx, "len(buffer):", len(buffer))
						if idx == -1 {
							// No JSON start yet; if buffer is getting long, flush progressively
							if len(buffer) > a.useToolDataLength {
								ch <- StreamResponse{Content: buffer}
								buffer = ""
								isAssistantContent = true
							}
							continue
						}

						// Emit any narration before the JSON
						if idx > 0 {
							ch <- StreamResponse{Content: buffer[:idx]}
						}
						// Keep only the JSON part in buffer going forward
						buffer = buffer[idx:]
						toolJSONStartFound = true
					}

					// If we're inside a JSON tool payload, track braces until complete
					if toolJSONStartFound {
						// tool use found, return the tool use
					}
				}
			}

			stream.Close()

			// Save assistant message to memory
			if fullContent != "" {

				assistantMsg := openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: fullContent,
				}
				a.messages = append(a.messages, assistantMsg)

				if a.mem != nil && a.conversationID != "" {
					_ = a.mem.SaveMessages(ctx, a.conversationID, []openai.ChatCompletionMessage{assistantMsg})
				}

				a.CalculateCompletionTokenUsage(fullContent)

				// Check if this is a final answer or tool call
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

	// 找到response中的action为call_tool的json对象
	callToolJson := ""
	idx := strings.Index(response, `{"action":"call_tool"`)
	if idx != -1 {
		callToolJson = response[idx:]
	}

	if callToolJson == "" {
		return response, false, nil
	}

	if callToolJson != "" {
		err := json.Unmarshal([]byte(callToolJson), &resp)
		if err != nil {
			return response, false, nil
		}
	}

	switch resp.Action {
	case "final_answer":
		return resp.Answer, false, nil

	case "call_tool":
		if resp.Tool == "" {
			return "", false, fmt.Errorf("tool name is required for call_tool action")
		}

		// Send notification about tool call, json string

		ch <- StreamResponse{Content: "\n"}
		argsJson, _ := json.Marshal(resp.Args)
		argsJsonStr := string(argsJson)
		ch <- StreamResponse{Content: fmt.Sprintf(`{"action":"call_tool","tool":"%s","args":%s}`, resp.Tool, argsJsonStr)}

		tool := a.findTool(resp.Tool)
		if tool == nil {
			ch <- StreamResponse{Content: "\n"}
			ch <- StreamResponse{Content: fmt.Sprintf(`{"action":"error","message":"tool '%s' not found"}`, resp.Tool)}
			return "", false, fmt.Errorf("tool not found: %s", resp.Tool)
		}

		toolResult, err := tool.Call(ctx, resp.Args)
		if err != nil {
			errorMsg := fmt.Sprintf(`{"action":"error","message":"Tool '%s' call failed: %v"}`, resp.Tool, err)
			ch <- StreamResponse{Content: "\n"}
			ch <- StreamResponse{Content: errorMsg}
			return "", false, fmt.Errorf("tool call failed for %s: %w", resp.Tool, err)
		}

		// Send tool result through channel
		ch <- StreamResponse{Content: "\n"}
		resultMsg := fmt.Sprintf(`{"action":"tool_result","tool":"%s","result":%s, "args":%s}`, resp.Tool, toolResult, argsJsonStr)
		ch <- StreamResponse{Content: resultMsg}
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
func (a *Agent) handleLLMResponse(ctx context.Context, ch chan<- StreamResponse, output string) {
	assistantMsg := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: output,
	}
	a.messages = append(a.messages, assistantMsg)

	// Save assistant message to memory
	if a.mem != nil && a.conversationID != "" {
		_ = a.mem.SaveMessages(ctx, a.conversationID, []openai.ChatCompletionMessage{assistantMsg})
	}

	result, shouldContinue, err := a.parseLLMResponse(ctx, output)
	if err != nil {
		ch <- StreamResponse{Error: err}
		return
	}

	if !shouldContinue {
		if result != "" {
			ch <- StreamResponse{Content: result}
		}
		ch <- StreamResponse{Done: true}
		return
	}

	// Tool call detected - use handleStreamResponse to properly handle it
	_, _, err = a.handleStreamResponse(ctx, ch, output)
	if err != nil {
		ch <- StreamResponse{Error: err}
		return
	}
	ch <- StreamResponse{Done: true}
}
