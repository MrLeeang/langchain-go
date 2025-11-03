package llms

// LLMChatModelConfig holds configuration for creating an OpenAI chat model.
type Config struct {
	// BaseURL is the base URL of the API endpoint.
	// Examples: "https://api.openai.com/v1", "https://api.deepseek.com/v1"
	BaseURL string

	// APIKey is the API key for authentication.
	APIKey string

	// Model is the model name to use.
	// Examples: "gpt-4", "deepseek-chat", "deepseek-reasoner"
	Model string
}
