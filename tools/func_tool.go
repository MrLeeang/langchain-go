package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"unicode"
)

// FuncTool wraps a Go function as an agent tool with auto-generated JSON Schema parameters.
//
// Supported signatures:
//   - func(ctx context.Context, args T) (string, error)
//   - func(args T) (string, error)
//   - func(ctx context.Context, args T) (R, error) — R is JSON-marshaled to string
//   - func(args T) (R, error)
//
// T may be a struct (schema from json tags), map[string]any, or a scalar (wrapped as {"value": ...}).
type FuncTool struct {
	name        string
	description string
	schema      map[string]any
	fn          reflect.Value
	hasContext  bool
	argType     reflect.Type
}

// FuncOption configures RegisterFunc / NewFuncTool.
type FuncOption func(*funcToolConfig)

type funcToolConfig struct {
	name        string
	description string
}

// WithName sets the tool name exposed to the LLM.
func WithName(name string) FuncOption {
	return func(c *funcToolConfig) { c.name = name }
}

// WithDescription sets the tool description for the LLM.
func WithDescription(description string) FuncOption {
	return func(c *funcToolConfig) { c.description = description }
}

// NewFuncTool builds a FuncTool from fn and options.
func NewFuncTool(fn any, opts ...FuncOption) (*FuncTool, error) {
	if fn == nil {
		return nil, fmt.Errorf("fn is nil")
	}
	var cfg funcToolConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	v := reflect.ValueOf(fn)
	if v.Kind() != reflect.Func {
		return nil, fmt.Errorf("fn must be a function, got %T", fn)
	}
	ft := v.Type()
	if ft.NumOut() < 1 || ft.NumOut() > 2 {
		return nil, fmt.Errorf("fn must return (string, error) or (result, error), got %d results", ft.NumOut())
	}
	if ft.NumOut() == 2 && ft.Out(1) != errorType {
		return nil, fmt.Errorf("last return must be error")
	}

	hasCtx := false
	argIdx := 0
	if ft.NumIn() > 0 && ft.In(0) == contextType {
		hasCtx = true
		argIdx = 1
	}

	var argType reflect.Type
	switch ft.NumIn() - argIdx {
	case 0:
		argType = nil
	case 1:
		argType = ft.In(argIdx)
	default:
		return nil, fmt.Errorf("fn must accept at most one argument besides context, got %d", ft.NumIn()-argIdx)
	}

	name := cfg.name
	if name == "" {
		name = inferFuncName(v)
	}
	desc := cfg.description
	if desc == "" {
		desc = name
	}

	var schema map[string]any
	if argType == nil {
		schema = defaultObjectSchema()
	} else {
		schema = SchemaFromType(argType)
	}

	return &FuncTool{
		name:        name,
		description: desc,
		schema:      schema,
		fn:          v,
		hasContext:  hasCtx,
		argType:     argType,
	}, nil
}

var (
	contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType   = reflect.TypeOf((*error)(nil)).Elem()
)

func (t *FuncTool) Name() string        { return t.name }
func (t *FuncTool) Description() string { return t.description }
func (t *FuncTool) ArgumentsSchema() any {
	return t.schema
}

func (t *FuncTool) Call(ctx context.Context, input any) (string, error) {
	in := make([]reflect.Value, 0, 2)
	if t.hasContext {
		in = append(in, reflect.ValueOf(ctx))
	}
	if t.argType != nil {
		arg, err := DecodeInput(t.argType, input)
		if err != nil {
			return "", err
		}
		in = append(in, arg)
	}

	out := t.fn.Call(in)
	if len(out) == 2 && !out[1].IsNil() {
		return "", out[1].Interface().(error)
	}

	result := out[0].Interface()
	switch v := result.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v), nil
		}
		return string(b), nil
	}
}

func inferFuncName(v reflect.Value) string {
	pc := v.Pointer()
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "tool"
	}
	full := fn.Name()
	if i := strings.LastIndex(full, "."); i >= 0 {
		full = full[i+1:]
	}
	return toSnakeCase(full)
}

func toSnakeCase(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
