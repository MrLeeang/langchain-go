package llms

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	openai "github.com/sashabaranov/go-openai"
)

// OllamaModel is an implementation of the LLM interface using Ollama's API.
// Ollama is a tool for running large language models locally.
type OllamaModel struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewOllamaModel creates a new Ollama chat model instance.
//
// Example:
//
//	llm := llms.NewOllamaModel(llms.Config{
//	    BaseURL: "http://localhost:11434",
//	    Model:   "llama2",
//	})
func NewOllamaModel(cfg Config) *OllamaModel {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	return &OllamaModel{
		baseURL: baseURL,
		model:   cfg.Model,
		client:  &http.Client{},
	}
}

// Chat sends a chat completion request to Ollama and returns the response.
func (m *OllamaModel) Chat(ctx context.Context, messages []openai.ChatCompletionMessage) (openai.ChatCompletionResponse, error) {
	// Convert OpenAI messages to Ollama format
	ollamaMessages := make([]map[string]interface{}, 0, len(messages))
	for _, msg := range messages {
		ollamaMessages = append(ollamaMessages, map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	// Prepare Ollama API request
	reqBody := map[string]interface{}{
		"model":    m.model,
		"messages": ollamaMessages,
		"stream":   false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return openai.ChatCompletionResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request to Ollama
	url := fmt.Sprintf("%s/api/chat", m.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return openai.ChatCompletionResponse{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return openai.ChatCompletionResponse{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return openai.ChatCompletionResponse{}, fmt.Errorf("ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse Ollama response
	var ollamaResp struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		Done               bool   `json:"done"`
		DoneReason         string `json:"done_reason"`
		PromptEvalCount    int    `json:"prompt_eval_count"`
		EvalCount          int    `json:"eval_count"`
		TotalDuration      int    `json:"total_duration"`
		LoadDuration       int    `json:"load_duration"`
		PromptEvalDuration int    `json:"prompt_eval_duration"`
		EvalDuration       int    `json:"eval_duration"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return openai.ChatCompletionResponse{}, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert Ollama response to OpenAI format
	return openai.ChatCompletionResponse{
		ID: "ollama-" + m.model,
		Choices: []openai.ChatCompletionChoice{
			{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    ollamaResp.Message.Role,
					Content: ollamaResp.Message.Content,
				},
				FinishReason: "stop",
			},
		},
		Model: m.model,
		Usage: openai.Usage{
			TotalTokens:      ollamaResp.PromptEvalCount + ollamaResp.EvalCount,
			PromptTokens:     ollamaResp.PromptEvalCount,
			CompletionTokens: ollamaResp.EvalCount,
		},
	}, nil
}

// Embeddings creates embeddings for the given input using the embedding model.
// return the embedding vector of the input.
func (m *OllamaModel) Embeddings(ctx context.Context, inputs []string) ([]float32, error) {
	// TODO: implement ollama embeddings
	return nil, errors.New("ollama embeddings not yet implemented")
}

// ChatStream sends a chat completion request and returns a stream of responses.
// Note: Ollama streaming API format is different from OpenAI's format.
// Currently, this method is not implemented. The agent will automatically
// fallback to non-streaming Chat method if streaming is not supported.
//
// To enable streaming in the future, an adapter would need to be created
// to convert Ollama's streaming format to OpenAI's ChatCompletionStream format.
func (m *OllamaModel) ChatStream(ctx context.Context, messages []openai.ChatCompletionMessage) (*openai.ChatCompletionStream, error) {
	// Return error to indicate streaming is not supported
	// The agent will automatically fallback to Chat() method
	return nil, errors.New("ollama streaming not yet implemented - will fallback to non-streaming")
}
