package agents

import (
	"github.com/MrLeeang/langchain-go/llms"
	"github.com/pkoukk/tiktoken-go"
)

func (a *Agent) GetTotalTokens() int {
	return a.TotalTokens
}

func (a *Agent) GetPromptTokens() int {
	return a.PromptTokens
}

func (a *Agent) GetCompletionTokens() int {
	return a.CompletionTokens
}

func (a *Agent) GetTokenUsage() map[string]int {
	return map[string]int{
		"total_tokens":      a.TotalTokens,
		"prompt_tokens":     a.PromptTokens,
		"completion_tokens": a.CompletionTokens,
	}
}

func (a *Agent) ResetTokenUsage() {
	a.TotalTokens = 0
	a.PromptTokens = 0
	a.CompletionTokens = 0
}

func (a *Agent) AddTokenUsage(totalTokens int, promptTokens int, completionTokens int) {
	a.TotalTokens += totalTokens
	a.PromptTokens += promptTokens
	a.CompletionTokens += completionTokens
}

// Calculate completion token usage from response
func (a *Agent) CalculateCompletionTokenUsage(usage llms.ChatUsage) {
	a.CompletionTokens += usage.CompletionTokens
	a.PromptTokens += usage.PromptTokens
	a.TotalTokens += usage.TotalTokens
}

type TokenCounter struct {
	encoder *tiktoken.Tiktoken
}

func NewTokenCounter() (*TokenCounter, error) {
	// DeepSeek 使用 cl100k_base 编码，与 GPT-4 相同
	enc, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return nil, err
	}
	return &TokenCounter{encoder: enc}, nil
}

func (tc *TokenCounter) CountTokens(text string) int {
	tokens := tc.encoder.Encode(text, nil, nil)
	return len(tokens)
}
