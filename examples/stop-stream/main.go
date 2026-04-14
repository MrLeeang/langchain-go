package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/MrLeeang/langchain-go/agents"
	"github.com/MrLeeang/langchain-go/llms"
)

// This example demonstrates how to stop a running stream with agent.Stop().
func main() {
	ctx := context.Background()

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = "your-api-key-here" // Replace with your actual API key
	}

	llm := llms.NewOpenAIModel(llms.Config{
		BaseURL: "https://api.openai.com/v1",
		APIKey:  apiKey,
		Model:   "gpt-3.5-turbo",
	})

	agent := agents.CreateReactAgent(ctx, llm).
		WithPrompt("You are a helpful assistant.")

	go func() {
		time.Sleep(2 * time.Second)
		fmt.Println("\n[Stopping stream now]")
		agent.Stop()
	}()

	ch := agent.Stream("Write a very long tutorial about distributed systems.")
	for resp := range ch {
		if resp.Error != nil {
			fmt.Printf("\nError: %v\n", resp.Error)
			break
		}
		if resp.Content != "" {
			fmt.Print(resp.Content)
		}
		if resp.Done {
			fmt.Println("\n[Stream finished]")
			break
		}
	}
}
