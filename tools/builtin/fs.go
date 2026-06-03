package builtin

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/MrLeeang/langchain-go/v2/tools"
)

// Set provides built-in filesystem tools bound to a [Config].
type Set struct {
	cfg Config
}

// NewSet returns a built-in tool set for the given config.
func NewSet(cfg Config) *Set {
	return &Set{cfg: cfg.normalized()}
}

// Tools returns all built-in tools for this set.
func (s *Set) Tools() ([]tools.Tool, error) {
	reg := tools.NewRegistry()
	if err := s.Register(reg); err != nil {
		return nil, err
	}
	return reg.Tools(), nil
}

// Register adds built-in tools to reg.
func (s *Set) Register(reg *tools.Registry) error {
	if reg == nil {
		return fmt.Errorf("registry is nil")
	}
	if err := reg.RegisterFunc(s.readFile, tools.WithName("read_file"), tools.WithDescription(
		"Read a text file from the workspace. Use absolute paths or paths relative to the workspace root. "+
			"For large files, set offset (1-based line) and limit (max lines).")); err != nil {
		return err
	}
	if err := reg.RegisterFunc(s.listDir, tools.WithName("list_dir"), tools.WithDescription(
		"List files and directories under a workspace path.")); err != nil {
		return err
	}
	if err := reg.RegisterFunc(s.fileInfo, tools.WithName("file_info"), tools.WithDescription(
		"Return metadata (size, mode, mod time) for a file or directory in the workspace.")); err != nil {
		return err
	}
	if s.cfg.AllowWrite {
		if err := reg.RegisterFunc(s.writeFile, tools.WithName("write_file"), tools.WithDescription(
			"Write text content to a file in the workspace. Creates parent directories if needed.")); err != nil {
			return err
		}
	}
	return nil
}

type readFileArgs struct {
	Path   string `json:"path" description:"File path (absolute or relative to workspace root)" required:"true"`
	Offset int    `json:"offset" description:"1-based starting line (optional, default 1)"`
	Limit  int    `json:"limit" description:"Maximum number of lines to return (optional, 0 = all within size cap)"`
}

func (s *Set) readFile(_ context.Context, args readFileArgs) (string, error) {
	abs, err := s.cfg.resolvePath(args.Path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%q is a directory", args.Path)
	}
	if info.Size() > s.cfg.MaxReadBytes && args.Offset <= 1 && args.Limit == 0 {
		return "", fmt.Errorf("file size %d exceeds max read %d bytes; use offset/limit or a smaller file", info.Size(), s.cfg.MaxReadBytes)
	}

	f, err := os.Open(abs)
	if err != nil {
		return "", err
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, s.cfg.MaxReadBytes+1))
	if err != nil {
		return "", err
	}
	if int64(len(data)) > s.cfg.MaxReadBytes {
		return "", fmt.Errorf("file exceeds max read size of %d bytes", s.cfg.MaxReadBytes)
	}

	content := string(data)
	if args.Offset > 0 || args.Limit > 0 {
		content = sliceLines(content, args.Offset, args.Limit)
	}
	return content, nil
}

type listDirArgs struct {
	Path string `json:"path" description:"Directory path (absolute or relative to workspace root)" required:"true"`
}

func (s *Set) listDir(_ context.Context, args listDirArgs) (string, error) {
	abs, err := s.cfg.resolvePath(args.Path)
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		b.WriteString(name)
		b.WriteByte('\n')
	}
	out := strings.TrimSuffix(b.String(), "\n")
	if out == "" {
		return "(empty directory)", nil
	}
	return out, nil
}

type fileInfoArgs struct {
	Path string `json:"path" description:"File or directory path" required:"true"`
}

func (s *Set) fileInfo(_ context.Context, args fileInfoArgs) (string, error) {
	abs, err := s.cfg.resolvePath(args.Path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	kind := "file"
	if info.IsDir() {
		kind = "directory"
	}
	return fmt.Sprintf("path: %s\nkind: %s\nsize: %d\nmode: %s\nmodified: %s",
		abs, kind, info.Size(), info.Mode(), info.ModTime().Format("2006-01-02T15:04:05Z07:00")), nil
}

type writeFileArgs struct {
	Path    string `json:"path" description:"File path to write" required:"true"`
	Content string `json:"content" description:"Text content to write" required:"true"`
}

func (s *Set) writeFile(_ context.Context, args writeFileArgs) (string, error) {
	if !s.cfg.AllowWrite {
		return "", fmt.Errorf("write_file is disabled; enable Config.AllowWrite")
	}
	abs, err := s.cfg.resolvePath(args.Path)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(abs, []byte(args.Content), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("wrote %d bytes to %s", len(args.Content), abs), nil
}

func sliceLines(content string, offset, limit int) string {
	lines := strings.Split(content, "\n")
	if offset < 1 {
		offset = 1
	}
	start := offset - 1
	if start >= len(lines) {
		return ""
	}
	end := len(lines)
	if limit > 0 && start+limit < end {
		end = start + limit
	}
	return strings.Join(lines[start:end], "\n")
}

// Tools is a shortcut for NewSet(cfg).Tools().
func Tools(cfg Config) ([]tools.Tool, error) {
	return NewSet(cfg).Tools()
}

// Register adds built-in tools to reg.
func Register(reg *tools.Registry, cfg Config) error {
	return NewSet(cfg).Register(reg)
}

// DefaultTools returns built-in tools using the current working directory as Root.
func DefaultTools() ([]tools.Tool, error) {
	return Tools(Config{})
}
