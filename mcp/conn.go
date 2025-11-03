package mcp

import (
	"fmt"

	mcpxport "github.com/mark3labs/mcp-go/client/transport"
)

// ConnSpec holds connection specifications for an MCP server.
type ConnSpec struct {
	Name      string
	Transport string
	Endpoint  string
	Command   string
	Args      []string
}

// newTransportFromSpec creates a transport interface from a connection specification.
func newTransportFromSpec(spec ConnSpec) (mcpxport.Interface, error) {
	switch TransportType(spec.Transport) {
	case TransportSSE:
		if spec.Endpoint == "" {
			return nil, fmt.Errorf("endpoint is required for sse transport")
		}
		return mcpxport.NewSSE(spec.Endpoint)
	case TransportStreamableHTTP:
		if spec.Endpoint == "" {
			return nil, fmt.Errorf("endpoint is required for streamable_http transport")
		}
		return mcpxport.NewStreamableHTTP(spec.Endpoint)
	case TransportStdio:
		if spec.Command == "" {
			return nil, fmt.Errorf("command is required for stdio transport")
		}
		tr := mcpxport.NewStdio(spec.Command, spec.Args, []string{}...)
		return tr, nil
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", spec.Transport)
	}
}
