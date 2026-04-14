package llms

// Chat message roles (OpenAI-compatible).
const (
	ChatMessageRoleSystem    = "system"
	ChatMessageRoleUser      = "user"
	ChatMessageRoleAssistant = "assistant"
	ChatMessageRoleTool      = "tool"
)

// ChatUsage is token usage for a completion or embedding call.
type ChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatToolCall is one function tool invocation from the assistant (OpenAI tool_calls).
type ChatToolCall struct {
	ID        string
	Name      string
	Arguments string // JSON object as produced by the model
}

// ChatCompletionMessage is one turn in a chat request or history.
type ChatCompletionMessage struct {
	Role             string
	Content          string
	ReasoningContent string
	// ToolCalls is set on assistant messages when the model requests tool execution.
	ToolCalls []ChatToolCall
	// ToolCallID is set on role "tool" messages (required by the API when replying to ToolCalls).
	ToolCallID string
}

// ChatCompletionChoice is one candidate in a non-streaming response.
type ChatCompletionChoice struct {
	Index        int
	Message      ChatCompletionMessage
	FinishReason string
}

// ChatCompletionResponse is a full non-streaming chat completion result.
type ChatCompletionResponse struct {
	ID      string
	Model   string
	Choices []ChatCompletionChoice
	Usage   ChatUsage
}

// ChatCompletionStreamDelta is one streamed fragment of an assistant message.
type ChatCompletionStreamDelta struct {
	Content          string
	ReasoningContent string
	// ToolCalls are incremental tool_call deltas for this chunk (concatenate per Index client-side).
	ToolCalls []ChatCompletionStreamToolCallDelta
}

// ChatCompletionStreamToolCallDelta is one entry in delta.tool_calls from the wire format.
type ChatCompletionStreamToolCallDelta struct {
	Index             int
	ID                string
	Type              string
	NameFragment      string
	ArgumentsFragment string
}

// ChatCompletionStreamChoice is one choice in a stream chunk.
type ChatCompletionStreamChoice struct {
	Index        int
	Delta        ChatCompletionStreamDelta
	FinishReason string
}

// ChatCompletionStreamResponse is one SSE chunk from a streaming completion.
type ChatCompletionStreamResponse struct {
	ID      string
	Model   string
	Choices []ChatCompletionStreamChoice
	Usage   *ChatUsage
}

// --- JSON-oriented aliases (e.g. for APIs or logging) ---

// ChatMessage is a message in a unified JSON-shaped completion.
type ChatMessage struct {
	Role             string `json:"role"`
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoning_content,omitempty"`
}

// ChatChoice is one choice in ChatResponse.
type ChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// ChatResponse is a JSON-friendly completion snapshot.
type ChatResponse struct {
	ID      string       `json:"id"`
	Model   string       `json:"model"`
	Choices []ChatChoice `json:"choices"`
	Usage   ChatUsage    `json:"usage"`
}

// ChatStreamDelta is delta content in ChatStreamChunk.
type ChatStreamDelta struct {
	Content          string                        `json:"content,omitempty"`
	ReasoningContent string                        `json:"reasoning_content,omitempty"`
	ToolCalls        []ChatStreamToolCallDeltaJSON `json:"tool_calls,omitempty"`
}

// ChatStreamToolCallDeltaJSON is a JSON-serializable tool_call delta fragment.
type ChatStreamToolCallDeltaJSON struct {
	Index     int    `json:"index"`
	ID        string `json:"id,omitempty"`
	Type      string `json:"type,omitempty"`
	Name      string `json:"name_fragment,omitempty"`
	Arguments string `json:"arguments_fragment,omitempty"`
}

// ChatStreamChoice is one streamed choice in ChatStreamChunk.
type ChatStreamChoice struct {
	Index        int             `json:"index"`
	Delta        ChatStreamDelta `json:"delta"`
	FinishReason string          `json:"finish_reason,omitempty"`
}

// ChatStreamChunk is a JSON-friendly streaming chunk.
type ChatStreamChunk struct {
	ID      string             `json:"id"`
	Model   string             `json:"model"`
	Choices []ChatStreamChoice `json:"choices"`
	Usage   *ChatUsage         `json:"usage,omitempty"`
}

// ToChatResponse maps ChatCompletionResponse to ChatResponse.
func ToChatResponse(resp ChatCompletionResponse) ChatResponse {
	cr := ChatResponse{
		ID:    resp.ID,
		Model: resp.Model,
		Usage: ChatUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}
	cr.Choices = make([]ChatChoice, 0, len(resp.Choices))
	for _, ch := range resp.Choices {
		cr.Choices = append(cr.Choices, ChatChoice{
			Index: ch.Index,
			Message: ChatMessage{
				Role:    ch.Message.Role,
				Content: ch.Message.Content,
			},
			FinishReason: ch.FinishReason,
		})
	}
	return cr
}

// ToChatStreamChunk maps ChatCompletionStreamResponse to ChatStreamChunk.
func ToChatStreamChunk(resp ChatCompletionStreamResponse) ChatStreamChunk {
	chunk := ChatStreamChunk{
		ID:    resp.ID,
		Model: resp.Model,
	}
	if resp.Usage != nil {
		chunk.Usage = &ChatUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}
	chunk.Choices = make([]ChatStreamChoice, 0, len(resp.Choices))
	for _, ch := range resp.Choices {
		tcdr := make([]ChatStreamToolCallDeltaJSON, 0, len(ch.Delta.ToolCalls))
		for _, t := range ch.Delta.ToolCalls {
			tcdr = append(tcdr, ChatStreamToolCallDeltaJSON{
				Index:     t.Index,
				ID:        t.ID,
				Type:      t.Type,
				Name:      t.NameFragment,
				Arguments: t.ArgumentsFragment,
			})
		}
		chunk.Choices = append(chunk.Choices, ChatStreamChoice{
			Index: ch.Index,
			Delta: ChatStreamDelta{
				Content:          ch.Delta.Content,
				ReasoningContent: ch.Delta.ReasoningContent,
				ToolCalls:        tcdr,
			},
			FinishReason: ch.FinishReason,
		})
	}
	return chunk
}
