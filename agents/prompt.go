package agents

import (
	"github.com/MrLeeang/langchain-go/mcp"

	openai "github.com/sashabaranov/go-openai"
)

// buildSystemPrompt constructs the system prompt for the agent.
func buildSystemPrompt(tools []mcp.Tool) string {
	// 	prompt := `You are an AI assistant.When you need external tools to complete a user request, you must return ONLY a valid JSON object (without any additional explanations) in the following format:
	// 1) To call a tool, return:
	// {"action":"call_tool","tool":"<tool_name>","args":{...}}
	// 2) Directly output the answer
	// `

	prompt := `
	You are an AI assistant. When you need external tools to complete user requests, you must output according to the following requirements:
	1) To call the tool, please return:
	Please use natural language to describe the intended use of the tool,don't more than 60 words,then return the following JSON object:
	{"action":"call_tool","tool":"<tool_name>","args":{...}}
	example:
	我将使用Nmap对192.168.2.235进行快速端口扫描。
	{"action":"call_tool","tool":"nmap","args":{"target":"192.168.2.235","ports":"1-1024"}}
	2) Directly output the answer
	`

	if len(tools) > 0 {
		prompt += "\n\nAvailable tools (use in the following format):\n"
		for _, tool := range tools {
			prompt += tool.Description() + "\n"
		}
	}

	return prompt
}

// WithPrompt adds a custom system prompt to the agent.
// This can be used to customize the agent's behavior or add additional instructions.
func (a *Agent) WithPrompt(prompt string) *Agent {
	a.messages = append(a.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: prompt,
	})
	return a
}

// WithDebug sets the debug mode for the agent.
// Default is false.
func WithDebug(debug bool) AgentOption {
	return func(a *Agent) {
		a.debug = debug
	}
}
