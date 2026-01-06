package main

import (
	"context"
	"fmt"
	"os"

	"github.com/MrLeeang/langchain-go/agents"
	"github.com/MrLeeang/langchain-go/llms"
	"github.com/MrLeeang/langchain-go/mcp"
	"github.com/MrLeeang/langchain-go/memory"
	"github.com/MrLeeang/langchain-go/skills"
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

	// skills
	skills, err := skills.LoadFiles([]string{"./examples-my/skill/search-host.md"})
	if err != nil {
		panic(err)
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
		agents.WithTools(tools),
		agents.WithSkills(skills),
		agents.WithMemory(mem),
		agents.WithConversationID("skills-chat"),
		agents.WithMaxIterations(20), // Limit tool-calling iterations
	).WithPrompt("You are a helpful assistant that can use tools to help users.")

	fmt.Printf("Agent created with %d tools\n", len(tools))
	fmt.Println("============================")

	// Ask a question that might require tool usage
	response, err := agent.Run("domain baidu.com is online?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", response)
}
