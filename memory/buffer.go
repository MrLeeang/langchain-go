package memory

import (
	"context"
	"sync"

	openai "github.com/sashabaranov/go-openai"
)

// BufferMemory is a simple in-memory implementation of the Memory interface.
// It stores conversation history in memory and is suitable for single-session
// or short-lived conversations.
//
// This is the default memory implementation when no custom memory is provided.
type BufferMemory struct {
	mu            sync.RWMutex
	conversations map[string][]openai.ChatCompletionMessage
}

// NewBufferMemory creates a new BufferMemory instance.
func NewBufferMemory() *BufferMemory {
	return &BufferMemory{
		conversations: make(map[string][]openai.ChatCompletionMessage),
	}
}

// LoadMessages loads conversation history for the given conversation ID.
func (m *BufferMemory) LoadMessages(ctx context.Context, conversationID string) ([]openai.ChatCompletionMessage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	id := m.getConversationID(conversationID)
	messages := m.conversations[id]

	// Return a copy to prevent external modifications
	result := make([]openai.ChatCompletionMessage, len(messages))
	copy(result, messages)
	return result, nil
}

// SaveMessages saves messages to the conversation history.
func (m *BufferMemory) SaveMessages(ctx context.Context, conversationID string, messages []openai.ChatCompletionMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := m.getConversationID(conversationID)
	m.conversations[id] = append(m.conversations[id], messages...)
	return nil
}

// ClearMessages clears all messages for the given conversation ID.
func (m *BufferMemory) ClearMessages(ctx context.Context, conversationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := m.getConversationID(conversationID)
	delete(m.conversations, id)
	return nil
}

// getConversationID returns the conversation ID, using a default if empty.
func (m *BufferMemory) getConversationID(conversationID string) string {
	if conversationID == "" {
		return "default"
	}
	return conversationID
}

// GetConversations returns all conversation IDs stored in memory.
// This is useful for debugging or administrative purposes.
func (m *BufferMemory) GetConversations() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.conversations))
	for id := range m.conversations {
		ids = append(ids, id)
	}
	return ids
}

