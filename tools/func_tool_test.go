package tools

import (
	"context"
	"testing"
)

type echoArgs struct {
	Text string `json:"text" required:"true"`
}

func echo(_ context.Context, args echoArgs) (string, error) {
	return "echo:" + args.Text, nil
}

func TestFuncTool(t *testing.T) {
	tool, err := NewFuncTool(echo, WithName("echo"), WithDescription("echo text"))
	if err != nil {
		t.Fatal(err)
	}
	if tool.Name() != "echo" {
		t.Fatalf("name = %s", tool.Name())
	}
	out, err := tool.Call(context.Background(), map[string]any{"text": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "echo:hi" {
		t.Fatalf("out = %q", out)
	}
}

func TestInferFuncName(t *testing.T) {
	tool, err := NewFuncTool(echo)
	if err != nil {
		t.Fatal(err)
	}
	if tool.Name() != "echo" {
		t.Fatalf("inferred name = %s", tool.Name())
	}
}
