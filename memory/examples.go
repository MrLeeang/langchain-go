package memory

// This file contains example implementations of the Memory interface
// to help users understand how to create custom memory backends.

// Example: Database-backed memory implementation
//
// type DatabaseMemory struct {
//     db *sql.DB
// }
//
// func NewDatabaseMemory(db *sql.DB) *DatabaseMemory {
//     return &DatabaseMemory{db: db}
// }
//
// func (m *DatabaseMemory) LoadMessages(ctx context.Context, conversationID string) ([]openai.ChatCompletionMessage, error) {
//     query := `SELECT role, content FROM messages WHERE conversation_id = $1 ORDER BY created_at ASC`
//     rows, err := m.db.QueryContext(ctx, query, conversationID)
//     if err != nil {
//         return nil, err
//     }
//     defer rows.Close()
//
//     var messages []openai.ChatCompletionMessage
//     for rows.Next() {
//         var role, content string
//         if err := rows.Scan(&role, &content); err != nil {
//             return nil, err
//         }
//         messages = append(messages, openai.ChatCompletionMessage{
//             Role:    role,
//             Content: content,
//         })
//     }
//     return messages, rows.Err()
// }
//
// func (m *DatabaseMemory) SaveMessages(ctx context.Context, conversationID string, messages []openai.ChatCompletionMessage) error {
//     query := `INSERT INTO messages (conversation_id, role, content, created_at) VALUES ($1, $2, $3, NOW())`
//     for _, msg := range messages {
//         _, err := m.db.ExecContext(ctx, query, conversationID, msg.Role, msg.Content)
//         if err != nil {
//             return err
//         }
//     }
//     return nil
// }
//
// func (m *DatabaseMemory) ClearMessages(ctx context.Context, conversationID string) error {
//     _, err := m.db.ExecContext(ctx, `DELETE FROM messages WHERE conversation_id = $1`, conversationID)
//     return err
// }

// Example: Redis-backed memory implementation
//
// type RedisMemory struct {
//     client *redis.Client
//     ttl    time.Duration
// }
//
// func NewRedisMemory(client *redis.Client, ttl time.Duration) *RedisMemory {
//     return &RedisMemory{client: client, ttl: ttl}
// }
//
// func (m *RedisMemory) LoadMessages(ctx context.Context, conversationID string) ([]openai.ChatCompletionMessage, error) {
//     key := fmt.Sprintf("conversation:%s:messages", conversationID)
//     data, err := m.client.Get(ctx, key).Result()
//     if err == redis.Nil {
//         return []openai.ChatCompletionMessage{}, nil
//     }
//     if err != nil {
//         return nil, err
//     }
//
//     var messages []openai.ChatCompletionMessage
//     if err := json.Unmarshal([]byte(data), &messages); err != nil {
//         return nil, err
//     }
//     return messages, nil
// }
//
// func (m *RedisMemory) SaveMessages(ctx context.Context, conversationID string, messages []openai.ChatCompletionMessage) error {
//     key := fmt.Sprintf("conversation:%s:messages", conversationID)
//     
//     // Load existing messages
//     existing, err := m.LoadMessages(ctx, conversationID)
//     if err != nil {
//         return err
//     }
//     
//     // Append new messages
//     existing = append(existing, messages...)
//     
//     // Serialize and save
//     data, err := json.Marshal(existing)
//     if err != nil {
//         return err
//     }
//     
//     return m.client.Set(ctx, key, data, m.ttl).Err()
// }
//
// func (m *RedisMemory) ClearMessages(ctx context.Context, conversationID string) error {
//     key := fmt.Sprintf("conversation:%s:messages", conversationID)
//     return m.client.Del(ctx, key).Err()
// }

