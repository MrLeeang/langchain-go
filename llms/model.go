package llms

import (
	openai "github.com/sashabaranov/go-openai"
)

// ChatUsage is a unified token usage struct aligned with OpenAI's Usage.
type ChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatMessage represents a chat message content in a unified way.
// ReasoningContent is optional and used by models that stream reasoning separately.
type ChatMessage struct {
	Role             string `json:"role"`
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoning_content,omitempty"`
}

// ChatChoice represents a single choice in a completion response.
type ChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// ChatResponse is a unified completion response structure inspired by OpenAI.
type ChatResponse struct {
	ID      string       `json:"id"`
	Model   string       `json:"model"`
	Choices []ChatChoice `json:"choices"`
	Usage   ChatUsage    `json:"usage"`
}

// ChatStreamDelta represents streamed delta content.
type ChatStreamDelta struct {
	Content          string `json:"content,omitempty"`
	ReasoningContent string `json:"reasoning_content,omitempty"`
}

// ChatStreamChoice represents a streamed choice delta.
type ChatStreamChoice struct {
	Index        int             `json:"index"`
	Delta        ChatStreamDelta `json:"delta"`
	FinishReason string          `json:"finish_reason,omitempty"`
}

// ChatStreamChunk is a unified streaming response chunk.
// Usage may be present only in the last chunk when supported by the provider.
type ChatStreamChunk struct {
	ID      string             `json:"id"`
	Model   string             `json:"model"`
	Choices []ChatStreamChoice `json:"choices"`
	Usage   *ChatUsage         `json:"usage,omitempty"`
}

// ToChatResponse converts an OpenAI ChatCompletionResponse to the unified ChatResponse.
func ToChatResponse(resp openai.ChatCompletionResponse) ChatResponse {
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
			FinishReason: string(ch.FinishReason),
		})
	}
	return cr
}

// ToChatStreamChunk converts an OpenAI ChatCompletionStreamResponse to the unified ChatStreamChunk.
func ToChatStreamChunk(resp openai.ChatCompletionStreamResponse) ChatStreamChunk {
	chunk := ChatStreamChunk{
		ID:    resp.ID,
		Model: resp.Model,
	}
	// Usage may be included only in the last chunk if requested
	if resp.Usage != nil {
		chunk.Usage = &ChatUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}
	chunk.Choices = make([]ChatStreamChoice, 0, len(resp.Choices))
	for _, ch := range resp.Choices {
		delta := ChatStreamDelta{
			Content:          ch.Delta.Content,
			ReasoningContent: ch.Delta.ReasoningContent,
		}
		chunk.Choices = append(chunk.Choices, ChatStreamChoice{
			Index:        ch.Index,
			Delta:        delta,
			FinishReason: string(ch.FinishReason),
		})
	}
	return chunk
}
