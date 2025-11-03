package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/MrLeeang/langchain-go/agents"
	"github.com/MrLeeang/langchain-go/llms"

	openai "github.com/sashabaranov/go-openai"
)

// RedisMemory is a custom memory implementation using Redis.
// This is an example of how to implement a custom memory backend.
type RedisMemory struct {
	// In a real implementation, this would be a Redis client
	// For this example, we'll use a simple in-memory map as a demonstration
	storage map[string][]byte
	ttl     time.Duration
}

// NewRedisMemory creates a new Redis memory instance.
// In production, you would pass an actual Redis client here.
func NewRedisMemory(ttl time.Duration) *RedisMemory {
	return &RedisMemory{
		storage: make(map[string][]byte),
		ttl:     ttl,
	}
}

// LoadMessages loads conversation history from Redis.
func (m *RedisMemory) LoadMessages(ctx context.Context, conversationID string) ([]openai.ChatCompletionMessage, error) {
	key := fmt.Sprintf("conversation:%s:messages", conversationID)
	
	data, exists := m.storage[key]
	if !exists {
		return []openai.ChatCompletionMessage{}, nil
	}

	var messages []openai.ChatCompletionMessage
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	return messages, nil
}

// SaveMessages saves messages to Redis.
func (m *RedisMemory) SaveMessages(ctx context.Context, conversationID string, messages []openai.ChatCompletionMessage) error {
	key := fmt.Sprintf("conversation:%s:messages", conversationID)

	// Load existing messages
	existing, err := m.LoadMessages(ctx, conversationID)
	if err != nil {
		return err
	}

	// Append new messages
	existing = append(existing, messages...)

	// Serialize and save
	data, err := json.Marshal(existing)
	if err != nil {
		return fmt.Errorf("failed to marshal messages: %w", err)
	}

	// In a real Redis implementation, you would use:
	// return redisClient.Set(ctx, key, data, m.ttl).Err()
	m.storage[key] = data
	return nil
}

// ClearMessages clears all messages for the conversation ID.
func (m *RedisMemory) ClearMessages(ctx context.Context, conversationID string) error {
	key := fmt.Sprintf("conversation:%s:messages", conversationID)
	
	// In a real Redis implementation, you would use:
	// return redisClient.Del(ctx, key).Err()
	delete(m.storage, key)
	return nil
}

// This example demonstrates how to use a custom memory implementation (Redis)
// to persist conversation history across application restarts.
func main() {
	ctx := context.Background()

	// Get API key from environment variable
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = "your-api-key-here" // Replace with your actual API key
	}

	// Create LLM instance
	llm := llms.NewOpenAIChatModel(llms.Config{
		BaseURL: "https://api.openai.com/v1",
		APIKey:  apiKey,
		Model:   "gpt-3.5-turbo",
	})

	// Create custom Redis memory with 24 hour TTL
	// In production, you would use: redis.NewClient(&redis.Options{...})
	redisMem := NewRedisMemory(24 * time.Hour)

	// Create agent with Redis memory
	agent := agents.CreateReactAgent(ctx, llm,
		agents.WithMemory(redisMem),
		agents.WithConversationID("user-456"),
	).WithPrompt("You are a helpful assistant.")

	// First interaction
	fmt.Println("First interaction:")
	fmt.Println("==================")
	response1, err := agent.Run("My name is Alice. Remember this.")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Response: %s\n\n", response1)

	// Second interaction - memory should persist
	fmt.Println("Second interaction (should remember name):")
	fmt.Println("==========================================")
	response2, err := agent.Run("What's my name?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Response: %s\n\n", response2)

	// Note: In a real Redis implementation, the conversation history
	// would persist even after the application restarts
	fmt.Println("Note: In production with real Redis, conversation history")
	fmt.Println("would persist across application restarts.")
}

