package agents

import (
	"context"
	"time"

	"github.com/MrLeeang/langchain-go/llms"
	"github.com/MrLeeang/langchain-go/mcp"
	"github.com/MrLeeang/langchain-go/memory"
	"github.com/MrLeeang/langchain-go/skills"

	openai "github.com/sashabaranov/go-openai"
)

// Agent represents a ReAct-style agent that can use tools to answer questions.
// It maintains a conversation history and can iteratively use tools to gather information.
type Agent struct {
	ctx                 context.Context
	cancel              context.CancelFunc
	llm                 llms.LLM
	tools               []mcp.Tool
	skillsList          []skills.Skill
	messages            []openai.ChatCompletionMessage
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
	maxBufferSize       int
	debug               bool
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
		messages:         []openai.ChatCompletionMessage{},
		maxIter:          10,
		mem:              memory.NewBufferMemory(), // Default memory implementation
		maxBufferSize:    200,
		maxHistoryTokens: 32000,
	}

	// Apply options
	for _, opt := range opts {
		opt(agent)
	}

	// build system prompt
	if len(agent.tools) > 0 || len(agent.skillsList) > 0 {
		systemPrompt := buildSystemPrompt(agent.tools, agent.skillsList)
		agent.messages = append(agent.messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		})
	}

	return agent
}

func (a *Agent) SetMessages(messages []openai.ChatCompletionMessage) {
	a.messages = append(a.messages, messages...)
}

func (a *Agent) GetMessages() []openai.ChatCompletionMessage {
	return a.messages
}

// GetLLM returns the underlying LLM instance
func (a *Agent) GetLLM() llms.LLM {
	if a == nil {
		return nil
	}
	return a.llm
}
