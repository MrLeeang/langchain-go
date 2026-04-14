package agents

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/MrLeeang/langchain-go/llms"
)

// streamToolCallBuffer accumulates one tool_call across streamed chunks (by tool index).
type streamToolCallBuffer struct {
	id, typ, name, args string
}

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
// Assistant text is streamed token-wise. Tool calls are accumulated per chunk index (name and
// arguments concatenated across chunks); when finish_reason is tool_calls, the stream round
// ends early, tools execute, then the outer loop continues for the model's next reply.
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
	a.ResetDuration()

	a.LoadMessages(message)

	// Cancel any previous run/stream if still active
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}

	// Create a cancellable context so that Stop() can interrupt this stream.
	ctx, cancel := context.WithCancel(a.ctx)
	a.cancel = cancel

	return a.StreamWithContext(ctx, message)
}

// StreamWithContext processes a user message with a custom context and returns a channel that streams the response.
func (a *Agent) StreamWithContext(ctx context.Context, message string) <-chan StreamResponse {
	ch := make(chan StreamResponse, 10)

	go func() {
		a.StartTime = time.Now()

		defer func() {
			a.EndTime = time.Now()
			a.Duration = a.EndTime.Sub(a.StartTime)
			time.Sleep(1 * time.Second)
			close(ch)

			if a.mem != nil && a.conversationID != "" {
				// user message already saved to memory in handleStreamResponse
				if err := a.mem.SaveMessages(ctx, a.conversationID, a.messages[a.historyMessageIndex:]); err != nil {
					fmt.Println("Error saving messages to memory:", err)
				}
			}
		}()

		userMsg := llms.ChatCompletionMessage{
			Role:    llms.ChatMessageRoleUser,
			Content: message,
		}
		a.messages = append(a.messages, userMsg)

		iterations := 0
		for iterations < a.maxIter {
			iterations++
			finishReason := ""

			if err := ctx.Err(); err != nil {

				if err == context.Canceled {
					ch <- StreamResponse{Done: true}
					return
				}

				ch <- StreamResponse{Error: err, Done: true}
				return
			}

			stream, err := a.chatStream(ctx)
			if err != nil {
				ch <- StreamResponse{Error: fmt.Errorf("failed to create stream: %w", err), Done: true}
				return
			}

			toolCallsBuffer := make(map[int]*streamToolCallBuffer)
			var fullContent strings.Builder
			var reasoningContent strings.Builder
			for {
				response, err := stream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}

				if err == context.Canceled {
					stream.Close()
					ch <- StreamResponse{Done: true}
					assistantMsg := llms.ChatCompletionMessage{
						Role:             llms.ChatMessageRoleAssistant,
						Content:          fullContent.String(),
						ReasoningContent: reasoningContent.String(),
					}
					a.messages = append(a.messages, assistantMsg)
					return
				}

				if err != nil {
					stream.Close()
					ch <- StreamResponse{Error: fmt.Errorf("stream error: %w", err), Done: true}
					return
				}

				if len(response.Choices) == 0 {
					continue
				}

				ch0 := response.Choices[0]
				delta := ch0.Delta
				if ch0.FinishReason != "" {
					finishReason = ch0.FinishReason
				}

				if delta.ReasoningContent != "" {
					reasoningContent.WriteString(delta.ReasoningContent)
					ch <- StreamResponse{ReasoningContent: delta.ReasoningContent}
				}

				if delta.Content != "" {
					fullContent.WriteString(delta.Content)
					ch <- StreamResponse{Content: delta.Content}
				}

				for _, tc := range delta.ToolCalls {
					idx := tc.Index
					buf, exists := toolCallsBuffer[idx]
					if !exists {
						buf = &streamToolCallBuffer{}
						toolCallsBuffer[idx] = buf
					}
					if tc.ID != "" {
						buf.id = tc.ID
					}
					if tc.Type != "" {
						buf.typ = tc.Type
					}
					buf.name += tc.NameFragment
					buf.args += tc.ArgumentsFragment

					if buf.name != "" {
						if a.debug {
							ch <- StreamResponse{Content: fmt.Sprintf("\n[工具调用中: %s, 参数: %s]\n", buf.name, buf.args)}
						}
					}
				}

				if response.Usage != nil {
					a.CalculateCompletionTokenUsage(*response.Usage)
				}

				if strings.EqualFold(ch0.FinishReason, "tool_calls") {
					if a.debug {
						fmt.Println("\n[模型请求调用工具，流结束]")
					}
					break
				}
			}

			stream.Close()

			assistantMsg := llms.ChatCompletionMessage{
				Role:             llms.ChatMessageRoleAssistant,
				Content:          fullContent.String(),
				ReasoningContent: reasoningContent.String(),
				ToolCalls:        toolCallsSortedFromBuffer(toolCallsBuffer),
			}

			if strings.EqualFold(finishReason, "tool_calls") && len(assistantMsg.ToolCalls) == 0 {
				ch <- StreamResponse{
					Error: fmt.Errorf("model finished with tool_calls but no function name was accumulated from stream deltas"),
					Done:  true,
				}
				return
			}

			if a.debug {
				fmt.Println("\n=============stream accumulated assistant============")
				fmt.Printf("Content: %q tool_calls: %d\n", assistantMsg.Content, len(assistantMsg.ToolCalls))
				fmt.Println("=============stream accumulated assistant============")
			}

			a.messages = append(a.messages, assistantMsg)

			if len(assistantMsg.ToolCalls) > 0 {
				if err := a.executeNativeToolCalls(ctx, ch, assistantMsg.ToolCalls); err != nil {
					ch <- StreamResponse{Error: err, Done: true}
					return
				}
				continue
			}

			ch <- StreamResponse{Done: true}
			return
		}

		ch <- StreamResponse{Error: fmt.Errorf("max iterations (%d) exceeded", a.maxIter), Done: true}
	}()

	return ch
}

func toolCallsSortedFromBuffer(m map[int]*streamToolCallBuffer) []llms.ChatToolCall {
	if len(m) == 0 {
		return nil
	}
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	out := make([]llms.ChatToolCall, 0, len(keys))
	for _, k := range keys {
		b := m[k]
		if strings.TrimSpace(b.name) == "" {
			continue
		}
		out = append(out, llms.ChatToolCall{
			ID:        b.id,
			Name:      b.name,
			Arguments: b.args,
		})
	}
	return out
}
