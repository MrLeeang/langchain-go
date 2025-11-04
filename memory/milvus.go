package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
	openai "github.com/sashabaranov/go-openai"
)

type MilvusMemoryInterface interface {
	SetQuery(query string)
}

// MilvusMemory is a memory implementation that uses Milvus vector database
// to store and retrieve conversation messages based on embeddings.
// It implements both Memory and ConversationMemory interfaces.
type MilvusMemory struct {
	milvusClient   client.Client
	embedder       EmbedderInterface
	collectionName string
	embeddingDim   int
	// EnableQueryBasedLoading enables query-based loading in LoadMessages.
	// When enabled, LoadMessages will use the latest user input to retrieve relevant messages.
	EnableQueryBasedLoading bool
	// MaxRelevantMessages limits the number of relevant messages to retrieve when using query-based loading.
	MaxRelevantMessages int
	// latestUserInput stores the latest user input for automatic query-based loading
	latestUserInput string
	mutex           sync.RWMutex
}

// EmbedderInterface defines the interface for generating embeddings.
// This allows flexibility in using different embedding models.
type EmbedderInterface interface {
	// Embeddings creates embeddings for the given input strings.
	// Returns a slice of embedding vectors, one for each input string.
	Embeddings(ctx context.Context, inputs []string) ([][]float32, error)
}

// MilvusConfig holds configuration for MilvusMemory.
type MilvusConfig struct {
	// MilvusClient is the Milvus client instance.
	// If nil, a new client will be created using Address and Port.
	MilvusClient client.Client

	// Address is the Milvus server address (used if MilvusClient is nil).
	Address string

	// Port is the Milvus server port (used if MilvusClient is nil).
	Port int

	// CollectionName is the name of the Milvus collection to use.
	// If empty, a default name will be used.
	CollectionName string

	// EmbeddingDim is the dimension of embedding vectors.
	// Common values: 1536 (text-embedding-ada-002), 768, etc.
	EmbeddingDim int

	// Embedder is the embedding model interface.
	Embedder EmbedderInterface

	// EnableQueryBasedLoading enables query-based loading in LoadMessages.
	// When enabled, LoadMessages will use vector similarity search to retrieve
	// relevant messages based on the current user query instead of loading all messages.
	EnableQueryBasedLoading bool

	// MaxRelevantMessages limits the number of relevant messages to retrieve
	// when using query-based loading. Default is 10.
	MaxRelevantMessages int
}

// NewMilvusMemory creates a new MilvusMemory instance.
//
// Example:
//
//	embedder := &MyEmbedder{} // implements EmbedderInterface
//	mem := memory.NewMilvusMemory(memory.MilvusConfig{
//	    Address:        "localhost",
//	    Port:           19530,
//	    CollectionName: "conversation_memory",
//	    EmbeddingDim:   1536,
//	    Embedder:       embedder,
//	})
func NewMilvusMemory(cfg MilvusConfig) (*MilvusMemory, error) {
	var milvusClient client.Client
	var err error

	if cfg.MilvusClient != nil {
		milvusClient = cfg.MilvusClient
	} else {
		address := cfg.Address
		if address == "" {
			address = "localhost"
		}
		port := cfg.Port
		if port == 0 {
			port = 19530
		}

		milvusClient, err = client.NewDefaultGrpcClient(context.Background(), fmt.Sprintf("%s:%d", address, port))
		if err != nil {
			return nil, fmt.Errorf("failed to create Milvus client: %w", err)
		}
	}

	collectionName := cfg.CollectionName
	if collectionName == "" {
		collectionName = "langchain_memory"
	}

	if cfg.EmbeddingDim == 0 {
		return nil, fmt.Errorf("embedding dimension must be specified")
	}

	if cfg.Embedder == nil {
		return nil, fmt.Errorf("embedder must be provided")
	}

	maxRelevant := cfg.MaxRelevantMessages
	if maxRelevant <= 0 {
		maxRelevant = 10 // Default limit
	}

	mem := &MilvusMemory{
		milvusClient:            milvusClient,
		embedder:                cfg.Embedder,
		collectionName:          collectionName,
		embeddingDim:            cfg.EmbeddingDim,
		EnableQueryBasedLoading: cfg.EnableQueryBasedLoading,
		MaxRelevantMessages:     maxRelevant,
	}

	// Ensure collection exists
	if err := mem.ensureCollection(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure collection: %w", err)
	}

	return mem, nil
}

// ensureCollection creates the Milvus collection if it doesn't exist.
func (m *MilvusMemory) ensureCollection(ctx context.Context) error {
	// Check if collection exists
	exists, err := m.milvusClient.HasCollection(ctx, m.collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if exists {
		return nil
	}

	// Define schema - store as Q&A pairs (user_input, llm_output)
	schema := &entity.Schema{
		CollectionName: m.collectionName,
		Description:    "LangChain conversation memory storage (Q&A pairs)",
		AutoID:         true,
		Fields: []*entity.Field{
			{
				Name:       "id",
				DataType:   entity.FieldTypeInt64,
				PrimaryKey: true,
				AutoID:     true,
			},
			{
				Name:     "conversation_id",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "256",
				},
			},
			{
				Name:     "user_input",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "65535",
				},
			},
			{
				Name:     "llm_output",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "65535",
				},
			},
			{
				Name:     "embedding",
				DataType: entity.FieldTypeFloatVector,
				TypeParams: map[string]string{
					"dim": fmt.Sprintf("%d", m.embeddingDim),
				},
			},
			{
				Name:     "timestamp",
				DataType: entity.FieldTypeInt64,
			},
		},
	}

	// Create collection
	err = m.milvusClient.CreateCollection(ctx, schema, entity.DefaultShardNumber)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	// Create index for embedding field
	index, err := entity.NewIndexHNSW(entity.L2, 16, 200)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	err = m.milvusClient.CreateIndex(ctx, m.collectionName, "embedding", index, false)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// Load collection
	err = m.milvusClient.LoadCollection(ctx, m.collectionName, false)
	if err != nil {
		return fmt.Errorf("failed to load collection: %w", err)
	}

	return nil
}

// getConversationID returns the conversation ID, using default if empty.
func (m *MilvusMemory) getConversationID(conversationID string) string {
	if conversationID != "" {
		return conversationID
	}

	return "default"
}

// LoadMessages loads conversation history for the given conversation ID.
// If EnableQueryBasedLoading is true, it will use the latest user input
// (captured from SaveMessages) to retrieve relevant messages via vector similarity search.
// Otherwise, it returns all messages in chronological order.
func (m *MilvusMemory) LoadMessages(ctx context.Context, conversationID string) ([]openai.ChatCompletionMessage, error) {
	// If query-based loading is enabled, use the latest user input as query
	if m.EnableQueryBasedLoading {
		m.mutex.RLock()
		query := m.latestUserInput
		m.mutex.RUnlock()

		if query != "" {
			// Use semantic search with the latest user input
			return m.GetRelevantMessages(ctx, conversationID, query, m.MaxRelevantMessages)
		}
		// If no user input has been saved yet, fall back to loading all messages
	}

	// Default behavior: load all messages
	return m.loadAllMessages(ctx, conversationID)
}

// SetQuery manually sets a query for context-aware message loading.
// This is useful when you want to use a specific query instead of the latest user input.
// The query will be used to retrieve semantically relevant messages from history.
//
// Example:
//
//	mem.SetQuery("How to use Python?")
//	messages, _ := mem.LoadMessages(ctx, "conv-123")
func (m *MilvusMemory) SetQuery(query string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.latestUserInput = query
}

// loadAllMessages loads all messages for the conversation ID in chronological order.
func (m *MilvusMemory) loadAllMessages(ctx context.Context, conversationID string) ([]openai.ChatCompletionMessage, error) {
	convID := m.getConversationID(conversationID)

	// Query by conversation_id, ordered by timestamp
	expr := fmt.Sprintf("conversation_id == \"%s\"", convID)

	results, err := m.milvusClient.Query(
		ctx,
		m.collectionName,
		[]string{},
		expr,
		[]string{"user_input", "llm_output", "timestamp"},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query Milvus: %w", err)
	}

	return m.assembleMessagesFromColumns(results)
}

// assembleMessagesFromColumns converts Milvus query results to messages.
func (m *MilvusMemory) assembleMessagesFromColumns(results []entity.Column) ([]openai.ChatCompletionMessage, error) {
	messages := make([]openai.ChatCompletionMessage, 0)

	var userInputCol, llmOutputCol *entity.ColumnVarChar

	// Extract columns
	for _, col := range results {
		switch col.Name() {
		case "user_input":
			userInputCol = col.(*entity.ColumnVarChar)
		case "llm_output":
			llmOutputCol = col.(*entity.ColumnVarChar)
		}
	}

	if userInputCol == nil || llmOutputCol == nil {
		return messages, nil
	}

	// Assemble messages from Q&A pairs
	for i := 0; i < userInputCol.Len(); i++ {
		userInputVal, _ := userInputCol.Get(i)
		llmOutputVal, _ := llmOutputCol.Get(i)

		userInput, ok1 := userInputVal.(string)
		llmOutput, ok2 := llmOutputVal.(string)

		if !ok1 || userInput == "" || !ok2 || llmOutput == "" {
			continue
		}

		// Add user message
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: userInput,
		})

		// Add assistant message
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: llmOutput,
		})
	}

	return messages, nil
}

var currentUserInput string

// SaveMessages saves messages to the conversation history.
// It pairs user messages with assistant messages and stores them as Q&A pairs in Milvus.
func (m *MilvusMemory) SaveMessages(ctx context.Context, conversationID string, messages []openai.ChatCompletionMessage) error {
	if len(messages) == 0 {
		return nil
	}

	convID := m.getConversationID(conversationID)

	// Pair user and assistant messages
	// We need to match each user message with its corresponding assistant message
	// For simplicity, we'll pair them in order: user -> assistant

	// First, collect pairs from the incoming messages
	type QAPair struct {
		userInput string
		llmOutput string
		timestamp int64
	}

	var pairs []QAPair

	for _, msg := range messages {
		switch msg.Role {
		case openai.ChatMessageRoleUser:
			// Save the latest user input for query-based loading
			m.mutex.Lock()
			m.latestUserInput = msg.Content
			m.mutex.Unlock()

			// If we have a previous user input without output, we'll store it with empty output
			// Otherwise, start a new pair
			currentUserInput = msg.Content
		case openai.ChatMessageRoleAssistant:
			// If we have a user input, pair it with this assistant response
			if currentUserInput != "" && msg.Content != "" {
				pairs = append(pairs, QAPair{
					userInput: currentUserInput,
					llmOutput: msg.Content,
					timestamp: time.Now().UnixNano(),
				})
				currentUserInput = "" // Reset after pairing
			}
		case openai.ChatMessageRoleSystem:
			// System messages are handled separately, skip for now
			continue
		}
	}

	if len(pairs) == 0 {
		return nil
	}

	// Prepare data for insertion
	conversationIDs := make([]string, len(pairs))
	userInputs := make([]string, len(pairs))
	llmOutputs := make([]string, len(pairs))
	timestamps := make([]int64, len(pairs))

	for i, pair := range pairs {
		conversationIDs[i] = convID
		userInputs[i] = pair.userInput
		llmOutputs[i] = pair.llmOutput
		timestamps[i] = pair.timestamp
	}

	// Generate embeddings for the Q&A pairs
	// Combine user input and LLM output for better semantic representation
	texts := make([]string, len(pairs))
	for i, pair := range pairs {
		// Full Q&A pair
		texts[i] = fmt.Sprintf("Q: %s\nA: %s", pair.userInput, pair.llmOutput)
	}

	embeddings, err := m.embedder.Embeddings(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	if len(embeddings) != len(pairs) {
		return fmt.Errorf("embedding count mismatch: expected %d, got %d", len(pairs), len(embeddings))
	}

	// Convert embeddings to [][]float32 for insertion
	embeddingVectors := make([][]float32, len(embeddings))
	for i, emb := range embeddings {
		embeddingVectors[i] = emb
	}

	// Create entity columns for insertion
	insertData := []entity.Column{
		entity.NewColumnVarChar("conversation_id", conversationIDs),
		entity.NewColumnVarChar("user_input", userInputs),
		entity.NewColumnVarChar("llm_output", llmOutputs),
		entity.NewColumnFloatVector("embedding", m.embeddingDim, embeddingVectors),
		entity.NewColumnInt64("timestamp", timestamps),
	}

	_, err = m.milvusClient.Insert(ctx, m.collectionName, "", insertData...)
	if err != nil {
		return fmt.Errorf("failed to insert into Milvus: %w", err)
	}

	return nil
}

// ClearMessages clears all messages for the given conversation ID.
func (m *MilvusMemory) ClearMessages(ctx context.Context, conversationID string) error {
	convID := m.getConversationID(conversationID)

	expr := fmt.Sprintf("conversation_id == \"%s\"", convID)

	err := m.milvusClient.Delete(ctx, m.collectionName, "", expr)
	if err != nil {
		return fmt.Errorf("failed to delete from Milvus: %w", err)
	}

	return nil
}

// GetRelevantMessages retrieves relevant messages from history based on a query.
// It uses vector similarity search to find the most relevant Q&A pairs and assembles them.
func (m *MilvusMemory) GetRelevantMessages(ctx context.Context, conversationID string, query string, limit int) ([]openai.ChatCompletionMessage, error) {
	convID := m.getConversationID(conversationID)

	// Generate embedding for query
	embeddings, err := m.embedder.Embeddings(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	if len(embeddings) == 0 || len(embeddings[0]) == 0 {
		return nil, fmt.Errorf("empty embedding generated")
	}

	queryVector := embeddings[0]

	// Convert query vector to entity.Vector
	vectors := []entity.Vector{entity.FloatVector(queryVector)}

	// Create default search parameters
	searchParam, err := entity.NewIndexFlatSearchParam()
	if err != nil {
		return nil, fmt.Errorf("failed to create search param: %w", err)
	}

	// Search for similar Q&A pairs
	searchResults, err := m.milvusClient.Search(
		ctx,
		m.collectionName,
		[]string{},
		"conversation_id == \""+convID+"\"",
		[]string{"user_input", "llm_output"},
		vectors,
		"embedding",
		entity.L2,
		limit,
		searchParam,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search Milvus: %w", err)
	}

	// Convert Q&A pairs to messages
	messages := make([]openai.ChatCompletionMessage, 0)
	for _, result := range searchResults {
		// Extract fields from result columns
		var userInputCol, llmOutputCol *entity.ColumnVarChar
		for _, col := range result.Fields {
			switch col.Name() {
			case "user_input":
				userInputCol = col.(*entity.ColumnVarChar)
			case "llm_output":
				llmOutputCol = col.(*entity.ColumnVarChar)
			}
		}

		if userInputCol != nil {
			var llmOutputColActual *entity.ColumnVarChar
			if llmOutputCol != nil {
				llmOutputColActual = llmOutputCol
			}

			// Assemble messages from Q&A pairs
			for i := 0; i < userInputCol.Len(); i++ {
				userInputVal, _ := userInputCol.Get(i)
				userInput, ok := userInputVal.(string)
				if !ok || userInput == "" {
					continue
				}

				// Add user message
				messages = append(messages, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleUser,
					Content: userInput,
				})

				// Add assistant message if available
				if llmOutputColActual != nil {
					llmOutputVal, _ := llmOutputColActual.Get(i)
					if llmOutput, ok := llmOutputVal.(string); ok && llmOutput != "" {
						messages = append(messages, openai.ChatCompletionMessage{
							Role:    openai.ChatMessageRoleAssistant,
							Content: llmOutput,
						})
					}
				}
			}
		}
	}

	return messages, nil
}

// SummarizeMessages creates a summary of the conversation history.
// This is a placeholder implementation - in a production system, you might want
// to use an LLM to generate the summary.
func (m *MilvusMemory) SummarizeMessages(ctx context.Context, conversationID string) (string, error) {
	messages, err := m.LoadMessages(ctx, conversationID)
	if err != nil {
		return "", err
	}

	if len(messages) == 0 {
		return "", nil
	}

	// Simple summary: concatenate first few messages
	// In production, you might want to use an LLM to generate a proper summary
	summary := fmt.Sprintf("Conversation with %d messages. Topics discussed: ", len(messages))

	// Add first few message contents
	for i, msg := range messages {
		if i >= 3 {
			break
		}
		if len(msg.Content) > 100 {
			summary += msg.Content[:100] + "... "
		} else {
			summary += msg.Content + " "
		}
	}

	return summary, nil
}

// Close closes the Milvus client connection.
func (m *MilvusMemory) Close() error {
	if m.milvusClient != nil {
		return m.milvusClient.Close()
	}
	return nil
}

// EmbedderWrapper wraps embedding models that have an Embeddings method.
// It handles both single-input methods and batch processing.
type EmbedderWrapper struct {
	// embedderSingle supports models with Embeddings(ctx, []string) ([][]float32, error)
	embedderSingle interface {
		Embeddings(ctx context.Context, inputs []string) ([][]float32, error)
	}

	model string
}

// NewEmbedderWrapper creates a wrapper for embedding models.
// It can work with models that have either:
// - Embeddings(ctx, []string) ([]float32, error) method
// - CreateEmbeddings(ctx, req) (openai.EmbeddingResponse, error) method
//
// Example:
//
//	// Using OpenAIModel with Embeddings method
//	openaiModel := llms.NewOpenAIModel(llms.Config{...})
//	embedder := memory.NewEmbedderWrapperFromEmbeddings(openaiModel)
func NewEmbedderWrapperFromEmbeddings(embedder interface {
	Embeddings(ctx context.Context, inputs []string) ([][]float32, error)
}) *EmbedderWrapper {
	return &EmbedderWrapper{embedderSingle: embedder}
}

// Embeddings implements EmbedderInterface.
// It handles both single-input and batch processing.
func (w *EmbedderWrapper) Embeddings(ctx context.Context, inputs []string) ([][]float32, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	// Fallback to single-input method (less efficient, processes one by one)
	if w.embedderSingle != nil {

		embedding, err := w.embedderSingle.Embeddings(ctx, inputs)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding for input: %w", err)
		}

		return embedding, nil
	}

	return nil, fmt.Errorf("no embedder configured")
}
