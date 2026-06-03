package builtin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/MrLeeang/langchain-go/v2/tools"
)

func TestReadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(path, []byte("line1\nline2\nline3"), 0o644); err != nil {
		t.Fatal(err)
	}

	set := NewSet(Config{Root: dir})
	reg := tools.NewRegistry()
	if err := set.Register(reg); err != nil {
		t.Fatal(err)
	}
	tool, ok := reg.Get("read_file")
	if !ok {
		t.Fatal("read_file not registered")
	}

	out, err := tool.Call(context.Background(), map[string]any{"path": "hello.txt", "offset": 2, "limit": 1})
	if err != nil {
		t.Fatal(err)
	}
	if out != "line2" {
		t.Fatalf("got %q", out)
	}
}

func TestPathOutsideRoot(t *testing.T) {
	dir := t.TempDir()
	set := NewSet(Config{Root: dir})
	reg := tools.NewRegistry()
	_ = set.Register(reg)
	tool, _ := reg.Get("read_file")

	_, err := tool.Call(context.Background(), map[string]any{"path": "/etc/passwd"})
	if err == nil {
		t.Fatal("expected error for path outside root")
	}
}

func TestWriteFileDisabled(t *testing.T) {
	dir := t.TempDir()
	set := NewSet(Config{Root: dir, AllowWrite: false})
	reg := tools.NewRegistry()
	_ = set.Register(reg)
	if _, ok := reg.Get("write_file"); ok {
		t.Fatal("write_file should not be registered when AllowWrite is false")
	}
}

func TestWriteFileEnabled(t *testing.T) {
	dir := t.TempDir()
	set := NewSet(Config{Root: dir, AllowWrite: true})
	reg := tools.NewRegistry()
	_ = set.Register(reg)
	tool, _ := reg.Get("write_file")

	_, err := tool.Call(context.Background(), map[string]any{"path": "out.txt", "content": "ok"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "out.txt"))
	if err != nil || string(data) != "ok" {
		t.Fatalf("file content = %q err=%v", data, err)
	}
}
