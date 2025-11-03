package mcp

import "fmt"

// TransportType represents the type of transport to use for MCP connections.
type TransportType string

const (
	// TransportSSE represents Server-Sent Events transport.
	TransportSSE TransportType = "sse"
	// TransportStreamableHTTP represents streamable HTTP transport.
	TransportStreamableHTTP TransportType = "streamable_http"
	// TransportStdio represents stdio transport.
	TransportStdio TransportType = "stdio"
)

// Config holds configuration for an MCP server connection.
type Config struct {
	// Name is a unique identifier for this MCP server configuration.
	Name string

	// URL is the endpoint URL (required for SSE and streamable_http transports).
	URL string

	// Transport specifies the transport type: "sse", "streamable_http", or "stdio".
	Transport string

	// Description is an optional description of this MCP server.
	Description string

	// TimeoutSec specifies the timeout in seconds for operations (not currently used).
	TimeoutSec int

	// Disabled indicates whether this MCP server should be skipped during initialization.
	Disabled bool

	// Command is the command to run (required for stdio transport).
	Command string

	// Args are the command arguments (used for stdio transport).
	Args []string
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("config name is required")
	}

	if c.Transport == "" {
		return fmt.Errorf("transport type is required")
	}

	switch TransportType(c.Transport) {
	case TransportSSE, TransportStreamableHTTP:
		if c.URL == "" {
			return fmt.Errorf("URL is required for %s transport", c.Transport)
		}
	case TransportStdio:
		if c.Command == "" {
			return fmt.Errorf("command is required for stdio transport")
		}
	default:
		return fmt.Errorf("unsupported transport type: %s", c.Transport)
	}

	return nil
}
