package tools

import "context"

// Tool is the interface that all agent tools must implement (MCP tools, function tools, etc.).
type Tool interface {
	Name() string
	Description() string
	ArgumentsSchema() any
	Call(ctx context.Context, input any) (string, error)
}

// Merge combines multiple tool slices; later duplicates override earlier ones by name.
func Merge(slices ...[]Tool) []Tool {
	byName := make(map[string]Tool)
	var order []string
	for _, slice := range slices {
		for _, t := range slice {
			if t == nil {
				continue
			}
			name := t.Name()
			if _, ok := byName[name]; !ok {
				order = append(order, name)
			}
			byName[name] = t
		}
	}
	out := make([]Tool, 0, len(order))
	for _, name := range order {
		out = append(out, byName[name])
	}
	return out
}
