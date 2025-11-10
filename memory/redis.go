package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	openai "github.com/sashabaranov/go-openai"
)

// RedisMemory is a memory implementation that uses Redis to store and retrieve conversation history.
// It implements the Memory interface and provides persistent storage with TTL support.
//
// Example:
//
//	rdb := redis.NewClient(&redis.Options{
//	    Addr: "localhost:6379",
//	})
//	mem := memory.NewRedisMemory(rdb, 24*time.Hour) // 24 hour TTL
type RedisMemory struct {
	client *redis.Client
	ttl    time.Duration
	prefix string
}

// RedisConfig holds configuration for RedisMemory.
type RedisConfig struct {
	// Client is the Redis client instance.
	// If nil, a new client will be created using Address and Port.
	Client *redis.Client

	// Address is the Redis server address (used if Client is nil).
	Address string

	// Port is the Redis server port (used if Client is nil).
	Port int

	// Password is the Redis password (used if Client is nil).
	Password string

	// DB is the Redis database number (used if Client is nil).
	DB int

	// TTL is the time-to-live for stored messages. Zero means no expiration.
	TTL time.Duration

	// KeyPrefix is the prefix for all Redis keys. Default is "langchain:memory:".
	KeyPrefix string
}

// NewRedisMemory creates a new RedisMemory instance with the given Redis client and TTL.
//
// Example:
//
//	rdb := redis.NewClient(&redis.Options{
//	    Addr: "localhost:6379",
//	})
//	mem := memory.NewRedisMemory(rdb, 24*time.Hour)
func NewRedisMemory(client *redis.Client, ttl time.Duration) *RedisMemory {
	return &RedisMemory{
		client: client,
		ttl:    ttl,
		prefix: "langchain:memory:",
	}
}

// NewRedisMemoryWithConfig creates a new RedisMemory instance with configuration options.
//
// Example:
//
//	mem, err := memory.NewRedisMemoryWithConfig(memory.RedisConfig{
//	    Address: "localhost",
//	    Port:    6379,
//	    TTL:     24 * time.Hour,
//	    KeyPrefix: "myapp:memory:",
//	})
func NewRedisMemoryWithConfig(cfg RedisConfig) (*RedisMemory, error) {
	var client *redis.Client

	if cfg.Client != nil {
		client = cfg.Client
	} else {
		address := cfg.Address
		if address == "" {
			address = "localhost"
		}

		port := cfg.Port
		if port == 0 {
			port = 6379
		}

		client = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", address, port),
			Password: cfg.Password,
			DB:       cfg.DB,
		})

		// Test connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Ping(ctx).Err(); err != nil {
			return nil, fmt.Errorf("failed to connect to Redis: %w", err)
		}
	}

	prefix := cfg.KeyPrefix
	if prefix == "" {
		prefix = "langchain:memory:"
	}

	return &RedisMemory{
		client: client,
		ttl:    cfg.TTL,
		prefix: prefix,
	}, nil
}

// getConversationID returns the conversation ID, using default if empty.
func (m *RedisMemory) getConversationID(conversationID string) string {
	if conversationID == "" {
		return "default"
	}
	return conversationID
}

// getKey returns the Redis key for the given conversation ID.
func (m *RedisMemory) getKey(conversationID string) string {
	return m.prefix + "conversation:" + m.getConversationID(conversationID) + ":messages"
}

// LoadMessages loads conversation history for the given conversation ID.
// Uses Redis List (LRANGE) for efficient loading.
func (m *RedisMemory) LoadMessages(ctx context.Context, conversationID string) ([]openai.ChatCompletionMessage, error) {
	key := m.getKey(conversationID)

	// Get all messages from the list (0 to -1 means all elements)
	data, err := m.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get messages from Redis: %w", err)
	}

	if len(data) == 0 {
		return []openai.ChatCompletionMessage{}, nil
	}

	// Unmarshal each message from the list
	messages := make([]openai.ChatCompletionMessage, 0, len(data))
	for _, item := range data {
		var msg openai.ChatCompletionMessage
		if err := json.Unmarshal([]byte(item), &msg); err != nil {
			// Skip invalid messages but continue processing
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// SaveMessages saves messages to the conversation history.
// Uses Redis List (RPUSH) for efficient incremental appending.
// Each message is stored as a separate list element, avoiding the need to
// load and rewrite the entire conversation history.
func (m *RedisMemory) SaveMessages(ctx context.Context, conversationID string, messages []openai.ChatCompletionMessage) error {
	if len(messages) == 0 {
		return nil
	}

	key := m.getKey(conversationID)

	// Serialize each message and push to the list
	pipe := m.client.Pipeline()
	for _, msg := range messages {
		data, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal message: %w", err)
		}
		pipe.RPush(ctx, key, data)
	}

	// Execute all pushes in a pipeline for better performance
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to save messages to Redis: %w", err)
	}

	// Set TTL on the list if configured
	if m.ttl > 0 {
		if err := m.client.Expire(ctx, key, m.ttl).Err(); err != nil {
			// Log but don't fail - TTL setting is best effort
			// In production, you might want to log this
		}
	}

	return nil
}

// ClearMessages clears all messages for the given conversation ID.
func (m *RedisMemory) ClearMessages(ctx context.Context, conversationID string) error {
	key := m.getKey(conversationID)

	err := m.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete messages from Redis: %w", err)
	}

	return nil
}

// Close closes the Redis client connection.
// This is optional but recommended for proper resource cleanup.
func (m *RedisMemory) Close() error {
	if m.client != nil {
		return m.client.Close()
	}
	return nil
}

// GetClient returns the underlying Redis client.
// This can be useful for advanced operations or debugging.
func (m *RedisMemory) GetClient() *redis.Client {
	return m.client
}

// LoadMessagesWithLimit loads a limited number of messages from the conversation history.
// This is useful for long conversations where you only need the most recent messages.
// The messages are returned in chronological order (oldest first).
//
// Parameters:
//   - conversationID: The conversation ID
//   - limit: Maximum number of messages to load. If 0 or negative, loads all messages.
//
// Example:
//
//	// Load only the last 10 messages
//	messages, _ := mem.LoadMessagesWithLimit(ctx, "conv-123", 10)
func (m *RedisMemory) LoadMessagesWithLimit(ctx context.Context, conversationID string, limit int) ([]openai.ChatCompletionMessage, error) {
	key := m.getKey(conversationID)

	var data []string
	var err error

	if limit > 0 {
		// Get list length first
		listLen, err := m.client.LLen(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get list length: %w", err)
		}

		// Calculate range: get last 'limit' messages
		start := int64(0)
		if listLen > int64(limit) {
			start = listLen - int64(limit)
		}
		end := listLen - 1

		data, err = m.client.LRange(ctx, key, start, end).Result()
	} else {
		// Load all messages
		data, err = m.client.LRange(ctx, key, 0, -1).Result()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get messages from Redis: %w", err)
	}

	if len(data) == 0 {
		return []openai.ChatCompletionMessage{}, nil
	}

	// Unmarshal each message from the list
	messages := make([]openai.ChatCompletionMessage, 0, len(data))
	for _, item := range data {
		var msg openai.ChatCompletionMessage
		if err := json.Unmarshal([]byte(item), &msg); err != nil {
			// Skip invalid messages but continue processing
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// GetMessageCount returns the number of messages stored for the given conversation ID.
func (m *RedisMemory) GetMessageCount(ctx context.Context, conversationID string) (int64, error) {
	key := m.getKey(conversationID)
	count, err := m.client.LLen(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get message count: %w", err)
	}
	return count, nil
}
