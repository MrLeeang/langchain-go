package agents

import (
	"github.com/MrLeeang/langchain-go/memory"

	openai "github.com/sashabaranov/go-openai"
)

// SetConversationID sets the conversation ID for this agent instance.
// This can be used to switch between different conversation threads.
// The conversation history for the new ID will be loaded from memory.
//
// Note: This method preserves system messages and replaces conversation history.
func (a *Agent) SetConversationID(conversationID string) *Agent {
	// Extract system messages (they should be at the beginning)
	systemMessages := []openai.ChatCompletionMessage{}
	for _, msg := range a.messages {
		if msg.Role == openai.ChatMessageRoleSystem {
			systemMessages = append(systemMessages, msg)
		} else {
			break
		}
	}

	a.conversationID = conversationID

	// Reload conversation history for the new ID
	if a.mem != nil {
		if history, err := a.mem.LoadMessages(a.ctx, conversationID); err == nil {
			// Reset messages with system messages first, then history
			a.messages = append(systemMessages, history...)
		} else {
			// If loading fails, just keep system messages
			a.messages = systemMessages
		}
	} else {
		// No memory, just keep system messages
		a.messages = systemMessages
	}
	return a
}

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
