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

func CountTokens(text string) int {

	enc, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return 1000 // if encoding fails, return a large number to be safe
	}

	tokens := enc.Encode(text, nil, nil)

	return len(tokens)
}
