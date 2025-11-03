package memory

import (
	"context"

	openai "github.com/sashabaranov/go-openai"
)

// Memory is the interface that all memory implementations must satisfy.
// It provides methods for loading and saving conversation history.
//
// Implementations can use various storage backends:
//   - In-memory storage (for single-session conversations)
//   - Database storage (for persistent conversations)
//   - Redis storage (for distributed systems)
//   - File-based storage (for simple persistence)
type Memory interface {
	// LoadMessages loads conversation history for the given conversation ID.
	// It returns the messages that should be prepended to the conversation.
	// Returns an empty slice if no history exists.
	//
	// The conversation ID can be used to distinguish between different
	// conversation threads. If empty, implementations should handle it
	// according to their storage strategy (e.g., use a default ID or
	// return empty history).
	LoadMessages(ctx context.Context, conversationID string) ([]openai.ChatCompletionMessage, error)

	// SaveMessages saves a message to the conversation history.
	// This is called for each user message and assistant response.
	//
	// Parameters:
	//   - conversationID: The ID of the conversation thread
	//   - messages: The new messages to save (typically 1-2 messages per call)
	SaveMessages(ctx context.Context, conversationID string, messages []openai.ChatCompletionMessage) error

	// ClearMessages clears all messages for the given conversation ID.
	// This is useful for starting fresh conversations or cleaning up old data.
	ClearMessages(ctx context.Context, conversationID string) error
}

// ConversationMemory is an optional interface for memory implementations
// that support advanced features like message retrieval and summarization.
type ConversationMemory interface {
	Memory

	// GetRelevantMessages retrieves relevant messages from history based on a query.
	// This is useful for RAG-like retrieval where you want to find contextually
	// relevant past conversations to include in the current context.
	GetRelevantMessages(ctx context.Context, conversationID string, query string, limit int) ([]openai.ChatCompletionMessage, error)

	// SummarizeMessages creates a summary of the conversation history.
	// This can be used to compress long conversations into a shorter summary
	// to save tokens while preserving context.
	SummarizeMessages(ctx context.Context, conversationID string) (string, error)
}
