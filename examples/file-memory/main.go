package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MrLeeang/langchain-go/agents"
	"github.com/MrLeeang/langchain-go/llms"
	"github.com/MrLeeang/langchain-go/memory"
)

// This example demonstrates FileMemory, which persists chat history in a JSON file.
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

	memPath := filepath.Join(".", "tmp", "file-memory.json")
	mem := memory.NewFileMemory(memPath)

	agent := agents.CreateReactAgent(ctx, llm,
		agents.WithMemory(mem),
		agents.WithConversationID("file-memory-demo"),
	).WithPrompt("You are a helpful assistant that keeps memory.")

	fmt.Println("Question 1: My favorite color is blue. Remember this.")
	resp1, err := agent.Run("My favorite color is blue. Remember this.")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Response: %s\n\n", resp1)

	fmt.Println("Question 2: What is my favorite color?")
	resp2, err := agent.Run("What is my favorite color?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Response: %s\n\n", resp2)

	fmt.Printf("Memory file written to: %s\n", memPath)
}
