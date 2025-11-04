package main

import (
	"context"
	"fmt"
	"os"

	"github.com/MrLeeang/langchain-go/agents"
	"github.com/MrLeeang/langchain-go/llms"
	"github.com/MrLeeang/langchain-go/memory"
)

// This example demonstrates how to use the default BufferMemory
// to maintain conversation history across multiple interactions.
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

	// Create buffer memory (in-memory storage)
	mem := memory.NewBufferMemory()

	// Create agent with memory and conversation ID
	// The conversation ID helps maintain separate conversation threads
	agent := agents.CreateReactAgent(ctx, llm,
		agents.WithMemory(mem),
		agents.WithConversationID("user-123"),
	).WithPrompt("You are a helpful assistant that remembers the conversation.")

	// First message
	fmt.Println("Question 1: What is AI?")
	response1, err := agent.Run("What is AI?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Response: %s\n\n", response1)

	// Second message - agent will remember the previous conversation
	fmt.Println("Question 2: Can you explain it more simply?")
	response2, err := agent.Run("Can you explain it more simply?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Response: %s\n\n", response2)

	// Check what conversations are stored
	conversations := mem.GetConversations()
	fmt.Printf("Stored conversations: %v\n", conversations)
}
