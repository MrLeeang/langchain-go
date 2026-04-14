package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/MrLeeang/langchain-go/llms"
)

// FileMemory persists conversation history in a single JSON file on disk.
// All conversations are stored under one file, keyed by conversation ID (same semantics as [BufferMemory]).
//
// The file format is:
//
//	{"conversations":{"<id>":[{"role":"user","content":"..."},...]}}
//
// Writes are atomic (temp file + rename). The parent directory is created on first save if missing.
type FileMemory struct {
	mu       sync.Mutex
	filePath string
}

// NewFileMemory creates a [FileMemory] that reads/writes path.
// path should be an absolute or relative path to a JSON file (e.g. "./data/chat_memory.json").
func NewFileMemory(path string) *FileMemory {
	return &FileMemory{filePath: path}
}

func (m *FileMemory) LoadMessages(ctx context.Context, conversationID string) ([]llms.ChatCompletionMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	store, err := m.readStoreLocked()
	if err != nil {
		return nil, err
	}
	id := normalizeConversationID(conversationID)
	raw := store.Conversations[id]
	if len(raw) == 0 {
		return nil, nil
	}
	out := make([]llms.ChatCompletionMessage, len(raw))
	for i := range raw {
		out[i] = storedToLLM(raw[i])
	}
	return out, nil
}

func (m *FileMemory) SaveMessages(ctx context.Context, conversationID string, messages []llms.ChatCompletionMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(messages) == 0 {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	store, err := m.readStoreLocked()
	if err != nil {
		return err
	}
	if store.Conversations == nil {
		store.Conversations = make(map[string][]storedMessage)
	}
	id := normalizeConversationID(conversationID)
	store.Conversations[id] = append(store.Conversations[id], llmToStored(messages)...)
	return m.writeStoreLocked(store)
}

func (m *FileMemory) ClearMessages(ctx context.Context, conversationID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	store, err := m.readStoreLocked()
	if err != nil {
		return err
	}
	id := normalizeConversationID(conversationID)
	delete(store.Conversations, id)
	return m.writeStoreLocked(store)
}

// --- on-disk model (JSON-friendly; llms types lack json tags) ---

type fileStore struct {
	Conversations map[string][]storedMessage `json:"conversations"`
}

type storedMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCalls  []storedToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type storedToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func normalizeConversationID(conversationID string) string {
	if conversationID == "" {
		return "default"
	}
	return conversationID
}

func llmToStored(msgs []llms.ChatCompletionMessage) []storedMessage {
	out := make([]storedMessage, 0, len(msgs))
	for _, msg := range msgs {

		// if system message, skip
		if msg.Role == llms.ChatMessageRoleSystem {
			continue
		}

		sm := storedMessage{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}
		if len(msg.ToolCalls) > 0 {
			sm.ToolCalls = make([]storedToolCall, 0, len(msg.ToolCalls))
			for _, tc := range msg.ToolCalls {
				sm.ToolCalls = append(sm.ToolCalls, storedToolCall{
					ID:        tc.ID,
					Name:      tc.Name,
					Arguments: tc.Arguments,
				})
			}
		}
		out = append(out, sm)
	}
	return out
}

func storedToLLM(sm storedMessage) llms.ChatCompletionMessage {
	msg := llms.ChatCompletionMessage{
		Role:       sm.Role,
		Content:    sm.Content,
		ToolCallID: sm.ToolCallID,
	}
	if len(sm.ToolCalls) > 0 {
		msg.ToolCalls = make([]llms.ChatToolCall, 0, len(sm.ToolCalls))
		for _, tc := range sm.ToolCalls {
			msg.ToolCalls = append(msg.ToolCalls, llms.ChatToolCall{
				ID:        tc.ID,
				Name:      tc.Name,
				Arguments: tc.Arguments,
			})
		}
	}
	return msg
}

func (m *FileMemory) readStoreLocked() (fileStore, error) {
	var store fileStore
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fileStore{Conversations: make(map[string][]storedMessage)}, nil
		}
		return fileStore{}, err
	}
	if len(data) == 0 {
		return fileStore{Conversations: make(map[string][]storedMessage)}, nil
	}
	if err := json.Unmarshal(data, &store); err != nil {
		return fileStore{}, fmt.Errorf("memory file %s: invalid JSON: %w", m.filePath, err)
	}
	if store.Conversations == nil {
		store.Conversations = make(map[string][]storedMessage)
	}
	return store, nil
}

func (m *FileMemory) writeStoreLocked(store fileStore) error {
	dir := filepath.Dir(m.filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create memory dir %s: %w", dir, err)
		}
	}

	payload, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".langchain-memory-*.json")
	if err != nil {
		return fmt.Errorf("temp file for memory: %w", err)
	}
	tmpPath := tmp.Name()
	_, werr := tmp.Write(payload)
	cerr := tmp.Close()
	if werr != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write temp memory file: %w", errors.Join(werr, cerr))
	}
	if cerr != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp memory file: %w", cerr)
	}

	if err := os.Rename(tmpPath, m.filePath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace memory file %s: %w", m.filePath, err)
	}
	return nil
}
