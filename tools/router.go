package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// Router resolves tool names across multiple registries using optional prefixes.
//
// Example: prefix "math" + tool "add" is exposed to the LLM as "math_add".
// Lookup tries the full name first, then strips known prefixes.
type Router struct {
	mu       sync.RWMutex
	prefixes []routeEntry
	fallback *Registry
}

type routeEntry struct {
	prefix   string
	registry *Registry
}

// NewRouter creates a router. Use Route to attach registries; Fallback receives unmatched names.
func NewRouter() *Router {
	return &Router{}
}

// Fallback sets the registry used when no prefix route matches.
func (rt *Router) Fallback(reg *Registry) *Router {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.fallback = reg
	return rt
}

// Route registers a registry under a prefix. Tool names are exposed as "{prefix}_{toolName}".
func (rt *Router) Route(prefix string, reg *Registry) *Router {
	prefix = normalizePrefix(prefix)
	if prefix == "" {
		panic("router: prefix must not be empty")
	}
	if reg == nil {
		panic("router: registry is nil")
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.prefixes = append(rt.prefixes, routeEntry{prefix: prefix, registry: reg})
	return rt
}

// Resolve finds a tool by LLM-facing name.
func (rt *Router) Resolve(name string) (Tool, bool) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	if rt.fallback != nil {
		if t, ok := rt.fallback.Get(name); ok {
			return t, true
		}
	}

	for _, e := range rt.prefixes {
		if t, ok := e.registry.Get(name); ok {
			return t, true
		}
	}

	for _, e := range rt.prefixes {
		p := e.prefix + "_"
		if strings.HasPrefix(name, p) {
			local := strings.TrimPrefix(name, p)
			if t, ok := e.registry.Get(local); ok {
				return t, true
			}
		}
	}

	return nil, false
}

// Tools returns all tools with routed names for the LLM.
func (rt *Router) Tools() []Tool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	var out []Tool
	seen := make(map[string]struct{})

	if rt.fallback != nil {
		for _, t := range rt.fallback.Tools() {
			if _, ok := seen[t.Name()]; ok {
				continue
			}
			seen[t.Name()] = struct{}{}
			out = append(out, t)
		}
	}

	for _, e := range rt.prefixes {
		for _, t := range e.registry.Tools() {
			exposed := e.prefix + "_" + t.Name()
			if _, ok := seen[exposed]; ok {
				continue
			}
			seen[exposed] = struct{}{}
			out = append(out, &routedTool{prefix: e.prefix, base: t, exposed: exposed})
		}
	}
	return out
}

// RegisterAll registers every routed tool into dest (typically a main Registry).
func (rt *Router) RegisterAll(dest *Registry) error {
	for _, t := range rt.Tools() {
		if err := dest.RegisterOrReplace(t); err != nil {
			return err
		}
	}
	return nil
}

// routedTool wraps a base tool with a prefixed exposed name.
type routedTool struct {
	prefix  string
	base    Tool
	exposed string
}

func (r *routedTool) Name() string { return r.exposed }

func (r *routedTool) Description() string {
	return fmt.Sprintf("[%s] %s", r.prefix, r.base.Description())
}

func (r *routedTool) ArgumentsSchema() any { return r.base.ArgumentsSchema() }

func (r *routedTool) Call(ctx context.Context, input any) (string, error) {
	return r.base.Call(ctx, input)
}

func normalizePrefix(prefix string) string {
	return strings.Trim(strings.TrimSpace(prefix), "_")
}
