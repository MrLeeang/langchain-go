package builtin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultMaxReadBytes = 1024 * 1024 // 1 MiB

// Config controls built-in filesystem tools.
type Config struct {
	// Root is the workspace directory; all paths must resolve inside it.
	// Empty means the current working directory at tool execution time.
	Root string

	// MaxReadBytes caps how many bytes read_file returns (default 1 MiB).
	MaxReadBytes int64

	// AllowWrite enables write_file (default false).
	AllowWrite bool
}

func (c *Config) normalized() Config {
	out := *c
	if out.MaxReadBytes <= 0 {
		out.MaxReadBytes = defaultMaxReadBytes
	}
	return out
}

func (c *Config) workspaceRoot() (string, error) {
	root := c.Root
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get working directory: %w", err)
		}
		root = wd
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve workspace root: %w", err)
	}
	return abs, nil
}

func (c *Config) resolvePath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("path is empty")
	}
	root, err := c.workspaceRoot()
	if err != nil {
		return "", err
	}

	var abs string
	if filepath.IsAbs(path) {
		abs, err = filepath.Abs(path)
	} else {
		abs, err = filepath.Abs(filepath.Join(root, path))
	}
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}

	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", fmt.Errorf("path outside workspace: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q is outside workspace root %q", path, root)
	}
	return abs, nil
}
