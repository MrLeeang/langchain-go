package mcp

import (
	"github.com/MrLeeang/langchain-go/v2/tools"
)

// Tool is the interface that all MCP-backed tools implement.
// It is an alias of [tools.Tool] so MCP and function tools can be used together.
type Tool = tools.Tool
