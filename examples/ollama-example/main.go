package main

import (
	"context"
	"fmt"

	"github.com/MrLeeang/langchain-go/agents"
	"github.com/MrLeeang/langchain-go/llms"
	"github.com/MrLeeang/langchain-go/memory"
)

// This example demonstrates how to use Ollama (local LLM) with the agent.
// Ollama allows you to run large language models locally without API keys.
//
// Prerequisites:
// 1. Install Ollama: https://ollama.ai
// 2. Pull a model: ollama pull llama2 (or mistral, codellama, etc.)
func main() {
	ctx := context.Background()

	// Create Ollama LLM instance
	// No API key needed! Ollama runs locally.
	llm := llms.NewOllamaModel(llms.Config{
		BaseURL: "http://localhost:11434/api", // Default Ollama endpoint
		Model:   "llama2",                     // Or "mistral", "codellama", etc.
	})

	// Create memory for conversation history
	mem := memory.NewBufferMemory()

	// Create agent with Ollama
	agent := agents.CreateReactAgent(ctx, llm,
		agents.WithMemory(mem),
		agents.WithConversationID("ollama-chat"),
	).WithPrompt("You are a helpful assistant.")

	fmt.Println("Using Ollama (local LLM)")
	fmt.Println("========================")

	// Simple question
	response, err := agent.Run("Explain quantum computing in simple terms.")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("\nMake sure Ollama is running and the model is pulled:")
		fmt.Println("  ollama serve")
		fmt.Println("  ollama pull llama2")
		return
	}

	fmt.Printf("Response: %s\n", response)
}
