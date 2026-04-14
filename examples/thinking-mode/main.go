package main

import (
	"context"
	"fmt"
	"os"

	"github.com/MrLeeang/langchain-go/agents"
	"github.com/MrLeeang/langchain-go/llms"
)

// This example demonstrates how to enable thinking mode with stream output.
func main() {
	ctx := context.Background()

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = "your-api-key-here" // Replace with your actual API key
	}

	// Thinking mode is provider/model dependent.
	// If your backend supports it, set Thinking: true.
	llm := llms.NewOpenAIModel(llms.Config{
		BaseURL:  "https://api.openai.com/v1",
		APIKey:   apiKey,
		Model:    "gpt-3.5-turbo",
		Thinking: true,
	})

	agent := agents.CreateReactAgent(ctx, llm).
		WithPrompt("You are a helpful assistant. Think carefully before answering.")

	ch := agent.Stream("Compare TCP and UDP in a concise way.")
	fmt.Println("Streaming response:")
	fmt.Println("===================")

	for resp := range ch {
		if resp.Error != nil {
			fmt.Printf("\nError: %v\n", resp.Error)
			break
		}
		if resp.ReasoningContent != "" {
			fmt.Print(resp.ReasoningContent)
		}
		if resp.Content != "" {
			fmt.Print(resp.Content)
		}
		if resp.Done {
			fmt.Println("\n[Stream completed]")
			break
		}
	}
}
