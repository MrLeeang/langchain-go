package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/MrLeeang/langchain-go/agents"
	"github.com/MrLeeang/langchain-go/llms"
	"github.com/MrLeeang/langchain-go/memory"
)

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
	llm := llms.NewOpenAIModel(llms.Config{
		BaseURL: "https://api.openai.com/v1",
		APIKey:  apiKey,
		Model:   "gpt-3.5-turbo",
	})

	// Create custom Redis memory with 24 hour TTL
	// In production, you would use: redis.NewClient(&redis.Options{...})
	mem, err := memory.NewRedisMemoryWithConfig(memory.RedisConfig{
		Address:   "localhost",
		Port:      6379,
		TTL:       24 * time.Hour,
		KeyPrefix: "langchain:memory:",
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Create agent with Redis memory
	agent := agents.CreateReactAgent(ctx, llm,
		agents.WithMemory(mem),
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
