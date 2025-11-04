package main

import (
	"context"
	"fmt"
	"os"

	"github.com/MrLeeang/langchain-go/agents"
	"github.com/MrLeeang/langchain-go/llms"
)

// This example demonstrates the simplest usage of the agent with Run method.
// It shows how to create an agent and get a simple response.
func main() {
	ctx := context.Background()

	// Get API key from environment variable
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = "your-api-key-here" // Replace with your actual API key
	}

	// Create LLM instance
	llm := llms.NewOpenAIModel(llms.Config{
		BaseURL: "https://api.openai.com/v1", // or "https://api.deepseek.com/v1"
		APIKey:  apiKey,
		Model:   "gpt-3.5-turbo", // or "deepseek-chat"
	})

	// Create agent without tools (simplest case)
	agent := agents.CreateReactAgent(ctx, llm)

	// Run the agent and get response
	response, err := agent.Run("What is the capital of France?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Response:", response)
}
