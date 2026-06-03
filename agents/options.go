package agents

import (
	"fmt"

	"github.com/MrLeeang/langchain-go/v2/memory"
	"github.com/MrLeeang/langchain-go/v2/skills"
	"github.com/MrLeeang/langchain-go/v2/tools"
	"github.com/MrLeeang/langchain-go/v2/tools/builtin"
)

// AgentOption is a function type for configuring an Agent.
type AgentOption func(*Agent)

// WithTools sets the tools that the agent can use (MCP tools, function tools, or both via [tools.Merge]).
// If not provided, the agent will be created without tools.
func WithTools(toolList []tools.Tool) AgentOption {
	return func(a *Agent) {
		a.tools = toolList
	}
}

// WithRegistry registers all tools from a [tools.Registry].
func WithRegistry(reg *tools.Registry) AgentOption {
	return func(a *Agent) {
		if reg != nil {
			a.tools = tools.Merge(a.tools, reg.Tools())
		}
	}
}

// WithToolRouter registers all tools from a [tools.Router] (prefix-based routing).
func WithToolRouter(router *tools.Router) AgentOption {
	return func(a *Agent) {
		if router != nil {
			a.tools = tools.Merge(a.tools, router.Tools())
		}
	}
}

// WithBuiltinTools registers built-in tools (read_file, list_dir, file_info, and optionally write_file).
// root is the workspace directory; empty uses the process working directory.
func WithBuiltinTools(root string) AgentOption {
	return WithBuiltinToolsConfig(builtin.Config{Root: root})
}

// WithBuiltinToolsConfig registers built-in tools with full [builtin.Config].
func WithBuiltinToolsConfig(cfg builtin.Config) AgentOption {
	return func(a *Agent) {
		builtins, err := builtin.Tools(cfg)
		if err != nil {
			panic(fmt.Sprintf("builtin tools: %v", err))
		}
		a.tools = tools.Merge(a.tools, builtins)
	}
}

// WithSkills registers skill metadata (from skills.LoadFiles / LoadDirectory) into the system prompt.
// The agent does not load file contents itself: the model should use read_file (or the MCP file tool) with the given path to read the full Markdown playbook.
func WithSkills(s []skills.Skill) AgentOption {
	return func(a *Agent) {
		a.registeredSkills = s
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

// WithDebug sets the debug mode for the agent.
// Default is false.
func WithDebug(debug bool) AgentOption {
	return func(a *Agent) {
		a.debug = debug
	}
}

// WithMaxWindowTokens sets the maximum number of tokens in the window.
// Default is 32000.
func WithMaxWindowTokens(maxWindowTokens int) AgentOption {
	return func(a *Agent) {
		a.maxWindowTokens = maxWindowTokens
	}
}
