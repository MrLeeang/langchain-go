package llms

import (
	"context"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAIChatModel is an implementation of the LLM interface using OpenAI's API.
// It can be used with any OpenAI-compatible API endpoint (e.g., DeepSeek, Anthropic via proxy, etc.).
type OpenAIChatModel struct {
	client *openai.Client
	model  string
}

// NewOpenAIChatModel creates a new OpenAI-compatible chat model instance using a config struct.
//
// Example:
//
//	llm := llms.NewOpenAIChatModel(llms.Config{
//	    BaseURL: "https://api.deepseek.com/v1",
//	    APIKey:  "sk-...",
//	    Model:   "deepseek-chat",
//	})
func NewOpenAIChatModel(cfg Config) *OpenAIChatModel {
	config := openai.DefaultConfig(cfg.APIKey)
	config.BaseURL = cfg.BaseURL
	client := openai.NewClientWithConfig(config)
	return &OpenAIChatModel{
		client: client,
		model:  cfg.Model,
	}
}

// Chat sends a chat completion request to the LLM and returns the response.
func (m *OpenAIChatModel) Chat(ctx context.Context, messages []openai.ChatCompletionMessage) (openai.ChatCompletionResponse, error) {
	req := openai.ChatCompletionRequest{
		Model:    m.model,
		Messages: messages,
	}

	return m.client.CreateChatCompletion(ctx, req)
}

// ChatStream sends a chat completion request and returns a stream of responses.
// This allows you to receive responses incrementally as they are generated.
func (m *OpenAIChatModel) ChatStream(ctx context.Context, messages []openai.ChatCompletionMessage) (*openai.ChatCompletionStream, error) {
	req := openai.ChatCompletionRequest{
		Model:    m.model,
		Messages: messages,
		Stream:   true,
	}

	return m.client.CreateChatCompletionStream(ctx, req)
}

// NewOpenAIChatModelWithParams creates a new OpenAI chat model using individual parameters.
// This is a convenience function for backward compatibility.
//
// Deprecated: Use NewOpenAIChatModel with OpenAIChatModelConfig instead.
func NewOpenAIChatModelWithParams(baseURL, apiKey, model string) *OpenAIChatModel {
	return NewOpenAIChatModel(Config{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
	})
}

// NewOpenaiChatModel is deprecated. Use NewOpenAIChatModel instead.
// Deprecated: Use NewOpenAIChatModel with OpenAIChatModelConfig for better naming consistency.
func NewOpenaiChatModel(BaseURL, apiKey, model string) *OpenAIChatModel {
	return NewOpenAIChatModelWithParams(BaseURL, apiKey, model)
}
