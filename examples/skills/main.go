package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MrLeeang/langchain-go/v2/agents"
	"github.com/MrLeeang/langchain-go/v2/llms"
	"github.com/MrLeeang/langchain-go/v2/memory"
	"github.com/MrLeeang/langchain-go/v2/skills"
)

// This example demonstrates how to use an agent with MCP tools.
// The agent can use external tools to gather information and answer questions.
func main() {
	ctx := context.Background()

	// Get API key from environment variable
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = "your-api-key-here" // Replace with your actual API key
	}

	// Load local skills from examples/skills/skills/*
	skillList, err := skills.LoadDirectory(filepath.Join(".", "examples", "skills", "skills"))
	if err != nil {
		fmt.Printf("Warning: Failed to load skills: %v\n", err)
		fmt.Println("Continuing without skills...")
		skillList = nil
	}

	// Create LLM instance
	llm := llms.NewOpenAIModel(llms.Config{
		BaseURL: "https://api.openai.com/v1",
		APIKey:  apiKey,
		Model:   "gpt-3.5-turbo",
	})

	// Create memory for conversation history
	mem := memory.NewBufferMemory()

	// Create agent with tools and memory
	agent := agents.CreateReactAgent(ctx, llm,
		agents.WithBuiltinTools("."),
		agents.WithSkills(skillList),
		agents.WithMemory(mem),
		agents.WithConversationID("skills-chat"),
		agents.WithMaxIterations(20), // Limit tool-calling iterations
	).WithPrompt("You are a helpful assistant that can use tools to help users.")

	fmt.Println("============================")

	// Ask a question that might require tool usage
	response, err := agent.Run("domain baidu.com is online?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", response)
}
