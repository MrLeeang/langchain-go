package main

import (
	"context"
	"fmt"
	"os"

	"github.com/MrLeeang/langchain-go/agents"
	"github.com/MrLeeang/langchain-go/llms"
)

// This example demonstrates execution metadata and token usage APIs.
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
		WithPrompt("You are a concise assistant.")

	resp, err := agent.Run("Summarize what an LLM agent is in 2 lines.")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Response:")
	fmt.Println(resp)
	fmt.Println()

	metadata := agent.GetMetadata()
	fmt.Printf("ConversationID: %s\n", metadata.ConversationID)
	fmt.Printf("PromptTokens: %d\n", metadata.PromptTokens)
	fmt.Printf("CompletionTokens: %d\n", metadata.CompletionTokens)
	fmt.Printf("TotalTokens: %d\n", metadata.TotalTokens)
	fmt.Printf("Duration: %s\n", metadata.Duration)
}
