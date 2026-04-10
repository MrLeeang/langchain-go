# langchain-go

A Go implementation of LangChain-style agents for building AI applications with large language models (LLMs). This library provides a flexible framework for creating ReAct-style agents that can use tools, maintain conversation history, and interact with various LLM providers.

## Features

- 🤖 **ReAct Agent Framework** - Implements the ReAct (Reasoning + Acting) pattern for intelligent tool usage
- 🛠️ **MCP Tool Integration** - Seamless integration with Model Context Protocol (MCP) tools
- 💾 **Flexible Memory Management** - Pluggable memory backends (in-memory, Redis, database, etc.)
- 🌊 **Streaming Support** - Real-time streaming responses for better UX
- 🔌 **OpenAI-compatible APIs** - Chat and embeddings via the official OpenAI Go SDK
- 🎯 **Simple API** - Clean, intuitive API design following Go best practices
- 📦 **Modular Design** - Well-structured packages for easy extension

## Installation

```bash
go get github.com/MrLeeang/langchain-go
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/MrLeeang/langchain-go/agents"
    "github.com/MrLeeang/langchain-go/llms"
)

func main() {
    ctx := context.Background()
    
    // Create LLM instance
    llm := llms.NewOpenAIModel(llms.Config{
        BaseURL: "https://api.openai.com/v1",
        APIKey:  os.Getenv("OPENAI_API_KEY"),
        Model:   "gpt-3.5-turbo",
    })
    
    // Create agent
    agent := agents.CreateReactAgent(ctx, llm)
    
    // Run the agent
    response, err := agent.Run("What is the capital of France?")
    if err != nil {
        panic(err)
    }
    
    fmt.Println(response)
}
```

### With Tools and Memory

```go
import (
    "github.com/MrLeeang/langchain-go/agents"
    "github.com/MrLeeang/langchain-go/llms"
    "github.com/MrLeeang/langchain-go/mcp"
    "github.com/MrLeeang/langchain-go/memory"
)

// Initialize MCP tools
tools, _ := mcp.InitializeMCP(ctx, []*mcp.Config{{
    Name:      "my-server",
    Transport: "sse",
    URL:       "http://localhost:8080/sse",
}})

// Create agent with tools and memory
agent := agents.CreateReactAgent(ctx, llm,
    agents.WithTools(tools),
    agents.WithMemory(memory.NewBufferMemory()),
    agents.WithConversationID("user-123"),
    agents.WithMaxIterations(10),
).WithPrompt("You are a helpful assistant.")
```

## Core Concepts

### Agents

Agents are the core abstraction that orchestrate LLM interactions and tool usage. They follow the ReAct pattern:

1. **Think** - The agent analyzes the user's request
2. **Act** - If needed, the agent calls tools to gather information
3. **Observe** - The agent processes tool results
4. **Respond** - The agent provides a final answer

### LLM Providers

The library supports multiple LLM providers through a unified interface:

- **OpenAI & compatible HTTP APIs** (DeepSeek, local gateways, etc.)

### Memory

Memory manages conversation history. You can use:

- **BufferMemory** - In-memory storage (default, session-based)
- **Custom Memory** - Implement the `memory.Memory` interface for Redis, databases, etc.

### Tools (MCP)

Tools extend agent capabilities through the Model Context Protocol (MCP). Agents can:

- Call external APIs
- Access databases
- Execute shell commands
- Interact with any MCP-compatible service

## Usage Examples

### Streaming Responses

```go
agent := agents.CreateReactAgent(ctx, llm)

ch := agent.Stream("Explain quantum computing")

for resp := range ch {
    if resp.Error != nil {
        log.Fatal(resp.Error)
    }
    fmt.Print(resp.Content)
    if resp.Done {
        break
    }
}
```

### Custom Memory Backend

```go
// Implement memory.Memory interface
type MyMemory struct {
    // your storage implementation
}

func (m *MyMemory) LoadMessages(ctx context.Context, id string) ([]llms.ChatCompletionMessage, error) {
    // load from your storage
}

func (m *MyMemory) SaveMessages(ctx context.Context, id string, msgs []llms.ChatCompletionMessage) error {
    // save to your storage
}

func (m *MyMemory) ClearMessages(ctx context.Context, id string) error {
    // clear from your storage
}

// Use custom memory
agent := agents.CreateReactAgent(ctx, llm,
    agents.WithMemory(&MyMemory{}),
    agents.WithConversationID("user-123"),
)
```

## Examples

Check out the `examples/` directory for complete examples:

- `simple-run/` - Basic agent usage
- `stream-example/` - Streaming responses
- `buffer-memory/` - In-memory conversation history
- `redis-memory/` - Custom Redis memory implementation
- `agent-tools/` - Agent with MCP tools

Run any example:

```bash
go run ./examples/simple-run
```

## API Reference

### Agents

- `agents.CreateReactAgent(ctx, llm, opts...)` - Create a new ReAct agent
- `agent.Run(message)` - Execute agent and get response
- `agent.Stream(message)` - Get streaming response via channel
- `agent.WithPrompt(prompt)` - Add custom system prompt
- `agent.SetConversationID(id)` - Switch conversation thread

### Agent Options

- `agents.WithTools(tools)` - Set tools for the agent
- `agents.WithMemory(mem)` - Set memory backend
- `agents.WithConversationID(id)` - Set conversation ID
- `agents.WithMaxIterations(n)` - Set max tool-calling iterations

### LLMs

- `llms.NewOpenAIModel(config)` - Create OpenAI-compatible LLM

### Memory

- `memory.NewBufferMemory()` - Create in-memory storage
- Implement `memory.Memory` interface for custom backends

### MCP Tools

- `mcp.InitializeMCP(ctx, configs)` - Initialize MCP servers and get tools

## Configuration

### LLM Config

```go
llms.Config{
    BaseURL: "https://api.openai.com/v1",
    APIKey:  "sk-...",
    Model:   "gpt-3.5-turbo",
}
```

### MCP Config

```go
mcp.Config{
    Name:      "my-server",
    Transport: "sse",           // or "streamable_http", "stdio"
    URL:       "http://localhost:8080/sse",
    Command:   "",              // for stdio transport
    Args:      []string{},      // for stdio transport
    Disabled:  false,
}
```

## Project Structure

```
langchain-go/
├── agents/          # Agent implementation (ReAct pattern)
├── llms/            # LLM provider implementations
├── memory/          # Memory management interfaces and implementations
├── mcp/             # Model Context Protocol integration
└── examples/        # Example code
```

## Requirements

- Go 1.24 or later

## License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Acknowledgments

- Inspired by [LangChain](https://github.com/langchain-ai/langchain)
- Built with [openai-go](https://github.com/openai/openai-go) (official SDK)
- MCP integration via [mcp-go](https://github.com/mark3labs/mcp-go)
