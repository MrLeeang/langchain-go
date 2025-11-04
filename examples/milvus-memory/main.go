package main

import (
	"context"
	"fmt"
	"os"

	"github.com/MrLeeang/langchain-go/agents"
	"github.com/MrLeeang/langchain-go/llms"
	"github.com/MrLeeang/langchain-go/memory"
)

// This example demonstrates how to use MilvusMemory (vector store memory)
// to store and retrieve conversation history using semantic similarity search.
// MilvusMemory stores conversation messages as embeddings in a vector database,
// allowing you to retrieve relevant past conversations based on semantic similarity.
func main() {
	ctx := context.Background()

	// Get API key from environment variable
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Warning: OPENAI_API_KEY not set. Please set it to use this example.")
		apiKey = "your-api-key-here" // Replace with your actual API key
	}

	// Create LLM instance for chat
	llm := llms.NewOpenAIModel(llms.Config{
		BaseURL: "https://api.openai.com/v1",
		APIKey:  apiKey,
		Model:   "gpt-3.5-turbo",
	})

	// Create embedding model for generating vector embeddings
	// Note: You can use the same OpenAI client, but typically you'd use a dedicated embedding model
	embeddingLLM := llms.NewOpenAIModel(llms.Config{
		BaseURL: "https://api.openai.com/v1",
		APIKey:  apiKey,
		Model:   "text-embedding-ada-002", // OpenAI embedding model
	})

	memory.NewEmbedderWrapperFromEmbeddings(embeddingLLM)

	// Wrap the embedding model to match EmbedderInterface
	embedder := memory.NewEmbedderWrapperFromEmbeddings(embeddingLLM)

	// Get Milvus connection details from environment or use defaults
	milvusAddress := os.Getenv("MILVUS_ADDRESS")
	if milvusAddress == "" {
		milvusAddress = "localhost"
	}

	milvusPort := 19530 // Default Milvus port

	fmt.Println("Creating MilvusMemory...")
	fmt.Printf("Milvus Address: %s:%d\n", milvusAddress, milvusPort)
	fmt.Println("Note: Make sure Milvus is running and accessible.")

	// Create MilvusMemory with query-based loading enabled
	// This allows semantic search to retrieve relevant past conversations
	milvusMem, err := memory.NewMilvusMemory(memory.MilvusConfig{
		Address:                 milvusAddress,
		Port:                    milvusPort,
		CollectionName:          "langchain_memory_example",
		EmbeddingDim:            1536, // text-embedding-ada-002 dimension
		Embedder:                embedder,
		EnableQueryBasedLoading: true, // Enable semantic search
		MaxRelevantMessages:     5,    // Retrieve top 5 relevant messages
	})
	if err != nil {
		fmt.Printf("Error creating MilvusMemory: %v\n", err)
		fmt.Println("\nMake sure:")
		fmt.Println("1. Milvus is running (docker run -d -p 19530:19530 milvusdb/milvus:latest)")
		fmt.Println("2. Milvus address and port are correct")
		fmt.Println("3. Network connectivity to Milvus is available")
		return
	}
	defer milvusMem.Close()

	fmt.Println("MilvusMemory created successfully!")
	fmt.Println()

	// Create agent with Milvus memory
	agent := agents.CreateReactAgent(ctx, llm,
		agents.WithMemory(milvusMem),
		agents.WithConversationID("example-conversation"),
	).WithPrompt("You are a helpful assistant that remembers past conversations using semantic search.")

	// First interaction - store some information
	fmt.Println("=== First Interaction ===")
	fmt.Println("Question: My name is Alice and I love programming in Python.")
	input := "My name is Alice and I love programming in Python."
	milvusMem.SetQuery(input)
	response1, err := agent.Run(input)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Response: %s\n\n", response1)

	// Second interaction - store different information
	fmt.Println("=== Second Interaction ===")
	fmt.Println("Question: I also enjoy reading science fiction books.")
	response2, err := agent.Run("I also enjoy reading science fiction books.")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Response: %s\n\n", response2)

	// Third interaction - semantic search should retrieve relevant past conversations
	fmt.Println("=== Third Interaction (Semantic Search) ===")
	fmt.Println("Question: What programming language do I like?")
	fmt.Println("Note: This query should retrieve the first conversation about Python programming.")
	response3, err := agent.Run("What programming language do I like?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Response: %s\n\n", response3)

	// Fourth interaction - test another semantic query
	fmt.Println("=== Fourth Interaction (Semantic Search) ===")
	fmt.Println("Question: What are my hobbies?")
	fmt.Println("Note: This query should retrieve conversations about both Python and books.")
	response4, err := agent.Run("What are my hobbies?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Response: %s\n\n", response4)

	// Demonstrate GetRelevantMessages directly
	fmt.Println("=== Direct Semantic Search ===")
	fmt.Println("Searching for messages relevant to: 'programming'")
	relevantMessages, err := milvusMem.GetRelevantMessages(ctx, "example-conversation", "programming", 3)
	if err != nil {
		fmt.Printf("Error retrieving relevant messages: %v\n", err)
	} else {
		fmt.Printf("Found %d relevant message(s):\n", len(relevantMessages))
		for i, msg := range relevantMessages {
			fmt.Printf("  %d. [%s] %s\n", i+1, msg.Role, msg.Content)
		}
	}
	fmt.Println()

	// Demonstrate SummarizeMessages
	fmt.Println("=== Conversation Summary ===")
	summary, err := milvusMem.SummarizeMessages(ctx, "example-conversation")
	if err != nil {
		fmt.Printf("Error generating summary: %v\n", err)
	} else {
		fmt.Printf("Summary: %s\n", summary)
	}
	fmt.Println()

	fmt.Println("=== Example Complete ===")
	fmt.Println("MilvusMemory allows you to:")
	fmt.Println("1. Store conversation history as vector embeddings")
	fmt.Println("2. Retrieve relevant past conversations using semantic search")
	fmt.Println("3. Maintain context across multiple conversation threads")
	fmt.Println("4. Scale to large conversation histories efficiently")
}
