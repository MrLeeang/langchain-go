package llms

import (
	"context"

	openai "github.com/sashabaranov/go-openai"
)

// LLM is the interface that all language models must implement.
// It provides a standard way for agents to interact with different LLM providers.
type LLM interface {
	// Chat sends a chat completion request to the LLM and returns the response.
	// The messages parameter contains the conversation history.
	Chat(ctx context.Context, messages []openai.ChatCompletionMessage) (openai.ChatCompletionResponse, error)
}

// ChatStreamer is an optional interface for LLMs that support streaming responses.
type ChatStreamer interface {
	// ChatStream sends a chat completion request and returns a stream of responses.
	ChatStream(ctx context.Context, messages []openai.ChatCompletionMessage) (*openai.ChatCompletionStream, error)
}
