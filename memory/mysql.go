package memory

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	openai "github.com/sashabaranov/go-openai"
)

// MySQLMemory is a memory implementation that uses MySQL database to store and retrieve conversation history.
// It implements the Memory interface and provides persistent storage with SQL capabilities.
//
// Example:
//
//	db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=true")
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer db.Close()
//
//	mem := memory.NewMySQLMemory(db, 24*time.Hour) // 24 hour TTL
type MySQLMemory struct {
	db     *sql.DB
	ttl    time.Duration
	prefix string
}

// MySQLConfig holds configuration for MySQLMemory.
type MySQLConfig struct {
	// DB is the database connection.
	// If nil, a new connection will be created using DSN.
	DB *sql.DB

	// DSN is the data source name for MySQL connection.
	// Used only if DB is nil.
	DSN string

	// TTL is the time-to-live for stored messages.
	// Zero means no expiration (handled by cleanup job).
	TTL time.Duration

	// TablePrefix is the prefix for all table names. Default is "langchain_".
	TablePrefix string
}

// NewMySQLMemory creates a new MySQLMemory instance with the given database connection and TTL.
//
// Example:
//
//	mem := memory.NewMySQLMemory(db, 24*time.Hour)
func NewMySQLMemory(db *sql.DB, ttl time.Duration) *MySQLMemory {
	return &MySQLMemory{
		db:     db,
		ttl:    ttl,
		prefix: "langchain_",
	}
}

// NewMySQLMemoryWithConfig creates a new MySQLMemory instance with configuration options.
//
// Example:
//
//	mem, err := memory.NewMySQLMemoryWithConfig(memory.MySQLConfig{
//	    DSN: "user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=true",
//	    TTL: 24 * time.Hour,
//	    TablePrefix: "myapp_",
//	})
func NewMySQLMemoryWithConfig(cfg MySQLConfig) (*MySQLMemory, error) {
	var db *sql.DB

	if cfg.DB != nil {
		db = cfg.DB
	} else {
		var err error
		db, err = sql.Open("mysql", cfg.DSN)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
		}
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping MySQL: %w", err)
	}

	prefix := cfg.TablePrefix
	if prefix == "" {
		prefix = "langchain_"
	}

	// Create tables if they don't exist
	if err := createTables(ctx, db, prefix); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return &MySQLMemory{
		db:     db,
		ttl:    cfg.TTL,
		prefix: prefix,
	}, nil
}

// createTables creates the necessary tables for storing conversation messages.
func createTables(ctx context.Context, db *sql.DB, prefix string) error {
	// Create messages table
	messagesTable := prefix + "messages"
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			conversation_id VARCHAR(255) NOT NULL,
			role VARCHAR(20) NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NULL,
			INDEX idx_conversation_id (conversation_id),
			INDEX idx_expires_at (expires_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`, messagesTable)

	_, err := db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create messages table: %w", err)
	}

	return nil
}

// getConversationID returns a conversation ID, using a default if empty.
func (m *MySQLMemory) getConversationID(conversationID string) string {
	if conversationID == "" {
		return "default"
	}
	return conversationID
}

// getTableName returns the table name for messages.
func (m *MySQLMemory) getTableName() string {
	return m.prefix + "messages"
}

// LoadMessages loads conversation history for the given conversation ID.
// It returns messages in chronological order (oldest first).
func (m *MySQLMemory) LoadMessages(ctx context.Context, conversationID string) ([]openai.ChatCompletionMessage, error) {
	tableName := m.getTableName()
	convID := m.getConversationID(conversationID)

	query := fmt.Sprintf(`
		SELECT role, content 
		FROM %s 
		WHERE conversation_id = ? 
			AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY created_at ASC
	`, tableName)

	rows, err := m.db.QueryContext(ctx, query, convID)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []openai.ChatCompletionMessage
	for rows.Next() {
		var role, content string
		if err := rows.Scan(&role, &content); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		messages = append(messages, openai.ChatCompletionMessage{
			Role:    role,
			Content: content,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

// SaveMessages saves messages to the conversation history.
// Each message is stored as a separate row with optional expiration.
func (m *MySQLMemory) SaveMessages(ctx context.Context, conversationID string, messages []openai.ChatCompletionMessage) error {
	if len(messages) == 0 {
		return nil
	}

	tableName := m.getTableName()
	convID := m.getConversationID(conversationID)

	// Begin a transaction for atomic insert
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare insert statement
	stmt, err := tx.PrepareContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (conversation_id, role, content, expires_at) 
		VALUES (?, ?, ?)
	`, tableName))
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Calculate expiration time
	var expiresAt interface{}
	if m.ttl > 0 {
		expiresAt = time.Now().Add(m.ttl)
	}

	// Insert each message
	for _, msg := range messages {
		_, err := stmt.ExecContext(ctx, convID, msg.Role, msg.Content, expiresAt)
		if err != nil {
			return fmt.Errorf("failed to insert message: %w", err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ClearMessages clears all messages for the given conversation ID.
func (m *MySQLMemory) ClearMessages(ctx context.Context, conversationID string) error {
	tableName := m.getTableName()
	convID := m.getConversationID(conversationID)

	query := fmt.Sprintf("DELETE FROM %s WHERE conversation_id = ?", tableName)
	_, err := m.db.ExecContext(ctx, query, convID)
	if err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}

	return nil
}

// Close closes the database connection.
// This is optional but recommended for proper resource cleanup.
func (m *MySQLMemory) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

// GetDB returns the underlying database connection.
// This can be useful for advanced operations or debugging.
func (m *MySQLMemory) GetDB() *sql.DB {
	return m.db
}

// CleanupExpiredMessages removes expired messages from all conversations.
// This should be called periodically (e.g., every hour) to prevent table bloat.
func (m *MySQLMemory) CleanupExpiredMessages(ctx context.Context) error {
	tableName := m.getTableName()

	query := fmt.Sprintf(`
		DELETE FROM %s 
		WHERE expires_at IS NOT NULL AND expires_at <= NOW()
	`, tableName)

	_, err := m.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired messages: %w", err)
	}

	return nil
}
