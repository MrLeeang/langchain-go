package llms

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

// OpenAIModel is an implementation of the LLM interface using OpenAI's API.
// It can be used with any OpenAI-compatible API endpoint (e.g., DeepSeek, Anthropic via proxy, etc.).
type OpenAIModel struct {
	client *openai.Client
	model  string
}

// NewOpenAIModel creates a new OpenAI-compatible chat model instance using a config struct.
//
// Example:
//
//	llm := llms.NewOpenAIModel(llms.Config{
//	    BaseURL: "https://api.deepseek.com/v1",
//	    APIKey:  "sk-...",
//	    Model:   "deepseek-chat",
//	})
func NewOpenAIModel(cfg Config) *OpenAIModel {
	config := openai.DefaultConfig(cfg.APIKey)
	config.BaseURL = cfg.BaseURL
	client := openai.NewClientWithConfig(config)
	return &OpenAIModel{
		client: client,
		model:  cfg.Model,
	}
}

// Chat sends a chat completion request to the LLM and returns the response.
func (m *OpenAIModel) Chat(ctx context.Context, messages []openai.ChatCompletionMessage) (openai.ChatCompletionResponse, error) {
	req := openai.ChatCompletionRequest{
		Model:    m.model,
		Messages: messages,
	}

	return m.client.CreateChatCompletion(ctx, req)
}

// ChatStream sends a chat completion request and returns a stream of responses.
// This allows you to receive responses incrementally as they are generated.
func (m *OpenAIModel) ChatStream(ctx context.Context, messages []openai.ChatCompletionMessage) (*openai.ChatCompletionStream, error) {
	req := openai.ChatCompletionRequest{
		Model:    m.model,
		Messages: messages,
		Stream:   true,
	}

	return m.client.CreateChatCompletionStream(ctx, req)
}

// Embeddings creates embeddings for the given input using the embedding model.
// return the embedding vector of the input.
func (m *OpenAIModel) Embeddings(ctx context.Context, inputs []string) ([][]float32, error) {

	req := openai.EmbeddingRequest{
		Model: openai.EmbeddingModel(m.model),
		Input: inputs,
	}

	resp, err := m.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, err
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("未返回嵌入数据")
	}

	results := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		results[i] = data.Embedding
	}

	return results, nil
}

// NewOpenAIModelWithParams creates a new OpenAI chat model using individual parameters.
// This is a convenience function for backward compatibility.
//
// Deprecated: Use NewOpenAIModel with OpenAIModelConfig instead.
func NewOpenAIModelWithParams(baseURL, apiKey, model string) *OpenAIModel {
	return NewOpenAIModel(Config{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
	})
}

// NewOpenAIModel is deprecated. Use NewOpenAIModel instead.
// Deprecated: Use NewOpenAIModel with OpenAIModelConfig for better naming consistency.
func NewOpenaiModel(BaseURL, apiKey, model string) *OpenAIModel {
	return NewOpenAIModelWithParams(BaseURL, apiKey, model)
}
