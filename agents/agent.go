package agents

import (
	"context"
	"time"

	"github.com/MrLeeang/langchain-go/llms"
	"github.com/MrLeeang/langchain-go/mcp"
	"github.com/MrLeeang/langchain-go/memory"
	"github.com/MrLeeang/langchain-go/skills"
)

// Agent represents a ReAct-style agent that can use tools to answer questions.
// It maintains a conversation history and can iteratively use tools to gather information.
type Agent struct {
	ctx                 context.Context
	cancel              context.CancelFunc
	llm                 llms.LLM
	tools               []mcp.Tool
	messages            []llms.ChatCompletionMessage
	historyMessageIndex int
	maxHistoryTokens    int
	Prompt              string
	maxIter             int
	mem                 memory.Memory
	conversationID      string
	TotalTokens         int
	PromptTokens        int
	CompletionTokens    int
	Duration            time.Duration
	StartTime           time.Time
	EndTime             time.Time
	debug               bool
	// registeredSkills lists skill metadata (name, description, path) injected into the system prompt so the model can read the full .md via tools such as read_file.
	registeredSkills []skills.Skill
}

// CreateReactAgent creates a new ReAct-style agent with the given LLM.
//
// The agent uses a ReAct (Reasoning + Acting) approach where it can:
// 1. Think about what to do
// 2. Use tools to gather information
// 3. Provide final answers based on tool results
//
// Example:
//
//	llm := llms.NewOpenAIModel(llms.Config{...})
//	tools, _ := mcp.InitializeMCP(ctx, configs)
//	agent := agents.CreateReactAgent(ctx, llm,
//	    agents.WithTools(tools),
//	    agents.WithMemory(memory.NewBufferMemory()),
//	)
func CreateReactAgent(ctx context.Context, llm llms.LLM, opts ...AgentOption) *Agent {
	agent := &Agent{
		ctx:              ctx,
		llm:              llm,
		tools:            []mcp.Tool{}, // Default to empty tools
		messages:         []llms.ChatCompletionMessage{},
		maxIter:          10,
		mem:              memory.NewBufferMemory(), // Default memory implementation
		maxHistoryTokens: 32000,
	}

	// Apply options
	for _, opt := range opts {
		opt(agent)
	}

	return agent
}

func (a *Agent) SetMessages(messages []llms.ChatCompletionMessage) {
	a.messages = append(a.messages, messages...)
}

func (a *Agent) GetMessages() []llms.ChatCompletionMessage {
	return a.messages
}

// GetLLM returns the underlying LLM instance
func (a *Agent) GetLLM() llms.LLM {
	if a == nil {
		return nil
	}
	return a.llm
}
