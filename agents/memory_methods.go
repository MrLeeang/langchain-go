package agents

import (
	"github.com/MrLeeang/langchain-go/memory"
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
