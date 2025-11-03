package main

import (
	"context"
	"fmt"
	"os"

	"github.com/MrLeeang/langchain-go/agents"
	"github.com/MrLeeang/langchain-go/llms"
)

// This example demonstrates how to use streaming responses with the agent.
// Streaming allows you to see the response as it's being generated.
func main() {
	ctx := context.Background()

	// Get API key from environment variable
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = "your-api-key-here" // Replace with your actual API key
	}

	// Create LLM instance that supports streaming
	llm := llms.NewOpenAIChatModel(llms.Config{
		BaseURL: "https://api.openai.com/v1",
		APIKey:  apiKey,
		Model:   "gpt-3.5-turbo",
	})

	// Create agent with a custom prompt
	agent := agents.CreateReactAgent(ctx, llm).
		WithPrompt("You are a helpful assistant. Answer concisely.")

	// Use streaming to get real-time responses
	ch := agent.Stream("Write a short poem about programming")

	fmt.Println("Streaming response:")
	fmt.Println("===================")

	// Read from the stream channel
	for resp := range ch {
		if resp.Error != nil {
			fmt.Printf("\nError: %v\n", resp.Error)
			break
		}

		// Print content as it arrives
		fmt.Print(resp.Content)

		// Check if stream is complete
		if resp.Done {
			fmt.Println("\n[Stream completed]")
			break
		}
	}
}

