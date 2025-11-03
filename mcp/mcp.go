package mcp

import (
	"context"
	"fmt"

	client "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// InitializeMCP initializes MCP servers based on the provided configurations
// and returns a list of available tools.
//
// It validates each configuration, establishes connections to MCP servers,
// and enumerates their available tools. If any server fails to initialize,
// the function returns an error immediately. Disabled configurations are skipped.
//
// The function creates temporary connections to enumerate tools, then closes them.
// Each tool maintains its own connection specification and will create a new
// connection when called.
//
// Example:
//
//	configs := []*mcp.Config{
//	    {
//	        Name:      "my-server",
//	        Transport: "sse",
//	        URL:       "http://localhost:8080/sse",
//	    },
//	}
//	tools, err := mcp.InitializeMCP(ctx, configs)
//	if err != nil {
//	    log.Fatal(err)
//	}
func InitializeMCP(ctx context.Context, configs []*Config) ([]Tool, error) {
	var tools []Tool

	for _, cfg := range configs {
		if cfg.Disabled {
			continue
		}

		if err := cfg.Validate(); err != nil {
			return nil, fmt.Errorf("invalid config for %s: %w", cfg.Name, err)
		}

		spec := ConnSpec{
			Name:      cfg.Name,
			Transport: cfg.Transport,
			Endpoint:  cfg.URL,
			Command:   cfg.Command,
			Args:      cfg.Args,
		}

		transport, err := newTransportFromSpec(spec)
		if err != nil {
			return nil, fmt.Errorf("failed to create transport for %s: %w", cfg.Name, err)
		}

		c := client.NewClient(transport)
		if err := c.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start MCP client for %s: %w", cfg.Name, err)
		}
		defer c.Close()

		if _, err := c.Initialize(ctx, mcp.InitializeRequest{
			Params: mcp.InitializeParams{
				ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
				ClientInfo:      mcp.Implementation{Name: "langchain-go", Version: "0.1.0"},
			},
		}); err != nil {
			return nil, fmt.Errorf("failed to initialize MCP client for %s: %w", cfg.Name, err)
		}

		toolsList, err := c.ListTools(ctx, mcp.ListToolsRequest{})
		if err != nil {
			return nil, fmt.Errorf("failed to list tools from %s: %w", cfg.Name, err)
		}

		for _, rt := range toolsList.Tools {
			if rt.Name == "" {
				continue
			}

			tools = append(tools, NewMCPTool(
				spec,
				rt.Name,
				rt.Description,
				rt.InputSchema,
			))
		}
	}

	return tools, nil
}
