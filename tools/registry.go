package tools

import (
	"fmt"
	"sync"
)

// Registry stores tools by name and supports automatic registration of Go functions.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// DefaultRegistry is a process-wide registry for package-level Register calls.
var DefaultRegistry = NewRegistry()

// Register adds a tool. Returns an error if the name already exists.
func (r *Registry) Register(tool Tool) error {
	if tool == nil {
		return fmt.Errorf("tool is nil")
	}
	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name is empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %q already registered", name)
	}
	r.tools[name] = tool
	return nil
}

// MustRegister panics if Register fails.
func (r *Registry) MustRegister(tool Tool) {
	if err := r.Register(tool); err != nil {
		panic(err)
	}
}

// RegisterOrReplace adds or replaces a tool by name.
func (r *Registry) RegisterOrReplace(tool Tool) error {
	if tool == nil {
		return fmt.Errorf("tool is nil")
	}
	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name is empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[name] = tool
	return nil
}

// RegisterFunc wraps fn as a FuncTool and registers it.
func (r *Registry) RegisterFunc(fn any, opts ...FuncOption) error {
	t, err := NewFuncTool(fn, opts...)
	if err != nil {
		return err
	}
	return r.Register(t)
}

// MustRegisterFunc panics if RegisterFunc fails.
func (r *Registry) MustRegisterFunc(fn any, opts ...FuncOption) {
	if err := r.RegisterFunc(fn, opts...); err != nil {
		panic(err)
	}
}

// RegisterFuncs registers multiple functions. Keys in names map override inferred names.
func (r *Registry) RegisterFuncs(fns map[string]any, descriptions map[string]string) error {
	for key, fn := range fns {
		var opts []FuncOption
		if key != "" {
			opts = append(opts, WithName(key))
		}
		if desc, ok := descriptions[key]; ok && desc != "" {
			opts = append(opts, WithDescription(desc))
		}
		if err := r.RegisterFunc(fn, opts...); err != nil {
			return fmt.Errorf("register %q: %w", key, err)
		}
	}
	return nil
}

// Get returns a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// Tools returns all registered tools in stable name order.
func (r *Registry) Tools() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sortStrings(names)
	out := make([]Tool, 0, len(names))
	for _, name := range names {
		out = append(out, r.tools[name])
	}
	return out
}

// Len returns the number of registered tools.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// Register is a convenience for DefaultRegistry.Register.
func Register(tool Tool) error {
	return DefaultRegistry.Register(tool)
}

// RegisterFunc registers fn on DefaultRegistry.
func RegisterFunc(fn any, opts ...FuncOption) error {
	return DefaultRegistry.RegisterFunc(fn, opts...)
}

func sortStrings(ss []string) {
	for i := 0; i < len(ss); i++ {
		for j := i + 1; j < len(ss); j++ {
			if ss[j] < ss[i] {
				ss[i], ss[j] = ss[j], ss[i]
			}
		}
	}
}
