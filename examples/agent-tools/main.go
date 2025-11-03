package main

import (
	"context"
	"fmt"
	"os"

	"github.com/MrLeeang/langchain-go/agents"
	"github.com/MrLeeang/langchain-go/llms"
	"github.com/MrLeeang/langchain-go/mcp"
	"github.com/MrLeeang/langchain-go/memory"
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

	// Configure MCP servers
	configs := []*mcp.Config{
		{
			Name:      "my-mcp-server",
			Transport: "sse",
			URL:       "http://localhost:8080/sse",
			Disabled:  false,
		},
		// You can add more MCP server configurations here
	}

	// Initialize MCP tools
	tools, err := mcp.InitializeMCP(ctx, configs)
	if err != nil {
		fmt.Printf("Warning: Failed to initialize MCP tools: %v\n", err)
		fmt.Println("Continuing without tools...")
		tools = []mcp.Tool{}
	}

	// Create LLM instance
	llm := llms.NewOpenAIChatModel(llms.Config{
		BaseURL: "https://api.openai.com/v1",
		APIKey:  apiKey,
		Model:   "gpt-3.5-turbo",
	})

	// Create memory for conversation history
	mem := memory.NewBufferMemory()

	// Create agent with tools and memory
	agent := agents.CreateReactAgent(ctx, llm,
		agents.WithTools(tools),
		agents.WithMemory(mem),
		agents.WithConversationID("tools-chat"),
		agents.WithMaxIterations(5), // Limit tool-calling iterations
	).WithPrompt("You are a helpful assistant that can use tools to help users.")

	fmt.Printf("Agent created with %d tools\n", len(tools))
	fmt.Println("============================")

	// Ask a question that might require tool usage
	response, err := agent.Run("What tools are available to me?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", response)
}

