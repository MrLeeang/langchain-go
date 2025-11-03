package mcp

import "context"

// Tool is the interface that all tools must implement.
// It provides a standard way for agents to interact with external tools.
type Tool interface {
	// Name returns the name of the tool.
	Name() string

	// Description returns a description of what the tool does.
	// This is used to help the LLM understand when and how to use the tool.
	Description() string

	// Call executes the tool with the given input and returns the result.
	// The input can be any type, typically a map[string]interface{} or a JSON-serializable structure.
	// Returns the result as a string and any error that occurred during execution.
	Call(ctx context.Context, input interface{}) (string, error)
}
