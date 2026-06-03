package tools

import (
	"context"
	"testing"
)

func add(_ context.Context, args struct {
	A int `json:"a"`
	B int `json:"b"`
}) (string, error) {
	return "ok", nil
}

func TestRouterPrefix(t *testing.T) {
	math := NewRegistry()
	if err := math.RegisterFunc(add, WithName("add"), WithDescription("add numbers")); err != nil {
		t.Fatal(err)
	}

	rt := NewRouter().Route("math", math)
	tools := rt.Tools()
	if len(tools) != 1 || tools[0].Name() != "math_add" {
		t.Fatalf("tools = %+v", tools[0].Name())
	}

	tool, ok := rt.Resolve("math_add")
	if !ok {
		t.Fatal("resolve failed")
	}
	if _, err := tool.Call(context.Background(), map[string]any{"a": 1, "b": 2}); err != nil {
		t.Fatal(err)
	}
}
