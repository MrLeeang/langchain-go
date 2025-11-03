package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// MCPTool represents a tool provided by an MCP server.
// It implements the Tool interface.
type MCPTool struct {
	conn       ConnSpec
	remoteName string
	remoteDesc string
	argsSchema interface{}
}

// NewMCPTool creates a new MCPTool instance.
func NewMCPTool(conn ConnSpec, remoteName, remoteDesc string, argsSchema interface{}) *MCPTool {
	return &MCPTool{
		conn:       conn,
		remoteName: remoteName,
		remoteDesc: remoteDesc,
		argsSchema: argsSchema,
	}
}

// Name returns the name of the tool.
func (t *MCPTool) Name() string {
	return t.remoteName
}

// Description returns a formatted description of the tool including its name,
// description, and argument schema.
func (t *MCPTool) Description() string {
	argsJSON, _ := json.Marshal(t.argsSchema)
	return fmt.Sprintf("\nname: %s, desc: %s, args_schema: %s", t.remoteName, t.remoteDesc, string(argsJSON))
}

// Call executes the tool with the given input.
// It creates a new MCP client connection, initializes it, and calls the tool.
// The input should be a map[string]interface{} or JSON-serializable structure.
func (t *MCPTool) Call(ctx context.Context, input interface{}) (string, error) {
	transport, err := newTransportFromSpec(t.conn)
	if err != nil {
		return "", fmt.Errorf("failed to create transport: %w", err)
	}

	c := mcpclient.NewClient(transport)
	if err := c.Start(ctx); err != nil {
		return "", fmt.Errorf("failed to start MCP client: %w", err)
	}
	defer c.Close()

	initReq := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcp.Implementation{Name: "langchain-go-mcp-tool", Version: "0.1.0"},
		},
	}
	if _, err := c.Initialize(ctx, initReq); err != nil {
		return "", fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	result, err := c.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      t.remoteName,
			Arguments: input,
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to call tool: %w", err)
	}

	// Extract text content from the result
	for _, part := range result.Content {
		switch v := part.(type) {
		case mcp.TextContent:
			return v.Text, nil
		case *mcp.TextContent:
			return v.Text, nil
		case mcp.ImageContent:
			return v.Data, nil
		case *mcp.ImageContent:
			return v.Data, nil
		case mcp.AudioContent:
			return v.Data, nil
		case *mcp.AudioContent:
			return v.Data, nil
		}
	}

	// Return empty string if no content found
	return "", nil
}
