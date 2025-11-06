package agents

import (
	"context"
	"fmt"

	"github.com/MrLeeang/langchain-go/memory"
	openai "github.com/sashabaranov/go-openai"
)

// Run processes a user message and returns the agent's response.
// It handles tool calling iteratively until a final answer is reached or max iterations are exceeded.
func (a *Agent) Run(message string) (string, error) {
	a.ResetTokenUsage()
	if a.mem != nil {
		if milvusMem, ok := a.mem.(memory.MilvusMemoryInterface); ok {
			milvusMem.SetQuery(message)
		}
	}

	return a.RunWithContext(a.ctx, message)
}

// RunWithContext processes a user message with a custom context and returns the agent's response.
func (a *Agent) RunWithContext(ctx context.Context, message string) (string, error) {
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
			// Log error but continue - memory save failures shouldn't block execution
			// In production, you might want to log this
		}
	}

	iterations := 0
	for iterations < a.maxIter {
		iterations++

		resp, err := a.llm.Chat(ctx, a.messages)
		if err != nil {
			return "", fmt.Errorf("failed to get LLM response: %w", err)
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("no response from LLM")
		}

		output := resp.Choices[0].Message.Content
		assistantMsg := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: output,
		}
		a.messages = append(a.messages, assistantMsg)

		a.CalculateCompletionTokenUsage(output)

		// Save assistant message to memory
		if a.mem != nil && a.conversationID != "" {
			if err := a.mem.SaveMessages(ctx, a.conversationID, []openai.ChatCompletionMessage{assistantMsg}); err != nil {
				// Log error but continue
				fmt.Println("error", err)
			}
		}

		result, shouldContinue, err := a.parseLLMResponse(ctx, output)
		if err != nil {
			return "", err
		}

		if !shouldContinue {
			return result, nil
		}
	}

	return "", fmt.Errorf("max iterations (%d) exceeded", a.maxIter)
}
