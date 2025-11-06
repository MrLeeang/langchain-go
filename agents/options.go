package agents

import (
	"github.com/MrLeeang/langchain-go/mcp"
	"github.com/MrLeeang/langchain-go/memory"
)

// AgentOption is a function type for configuring an Agent.
type AgentOption func(*Agent)

// WithTools sets the tools that the agent can use.
// If not provided, the agent will be created without tools.
func WithTools(tools []mcp.Tool) AgentOption {
	return func(a *Agent) {
		a.tools = tools
	}
}

// WithMaxIterations sets the maximum number of tool-calling iterations.
// Default is 10.
func WithMaxIterations(maxIter int) AgentOption {
	return func(a *Agent) {
		a.maxIter = maxIter
	}
}

// WithMemory sets a custom memory implementation for the agent.
// If not provided, a default BufferMemory will be used.
//
// Example:
//
//	// Use custom database-backed memory
//	customMemory := NewDatabaseMemory(db)
//	agent := agents.CreateReactAgent(ctx, llm,
//	    agents.WithTools(tools),
//	    agents.WithMemory(customMemory),
//	)
func WithMemory(mem memory.Memory) AgentOption {
	return func(a *Agent) {
		a.mem = mem
	}
}

// WithConversationID sets the conversation ID for this agent instance.
// This ID is used by the memory implementation to identify the conversation thread.
// If not set, the memory implementation will use a default ID.
func WithConversationID(conversationID string) AgentOption {
	return func(a *Agent) {
		a.conversationID = conversationID
	}
}

// WithUseToolDataLength sets the length of the data that will be used to call the tool.
// Default is 200.
func WithUseToolDataLength(useToolDataLength int) AgentOption {
	return func(a *Agent) {
		a.useToolDataLength = useToolDataLength
	}
}
