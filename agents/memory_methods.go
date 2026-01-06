package agents

import (
	"github.com/MrLeeang/langchain-go/memory"
	openai "github.com/sashabaranov/go-openai"
)

// GetMemory returns the memory implementation used by this agent.
func (a *Agent) GetMemory() memory.Memory {
	return a.mem
}

// ClearHistory clears the conversation history for the current conversation ID.
func (a *Agent) ClearHistory() error {
	if a.mem != nil && a.conversationID != "" {
		return a.mem.ClearMessages(a.ctx, a.conversationID)
	}
	return nil
}

func (a *Agent) ReloadMessages(latestUserInput string) {
	// clean tmp messages
	if a.mem != nil && a.conversationID != "" {
		// Check if this is MilvusMemory with query-based loading enabled
		// If so, skip loading here - it will be loaded when we have the user query
		if milvusMem, ok := a.mem.(*memory.MilvusMemory); ok {

			a.messages = []openai.ChatCompletionMessage{}

			if len(a.tools) > 0 || len(a.skillsList) > 0 {
				systemPrompt := buildSystemPrompt(a.tools, a.skillsList)
				a.messages = append(a.messages, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				})
			}

			milvusMem.SetQuery(latestUserInput)

			if history, err := a.mem.LoadMessages(a.ctx, a.conversationID); err == nil && len(history) > 0 {
				a.messages = append(a.messages, history...)
			}
		}
	}
}
