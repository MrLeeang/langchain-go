package agents

import (
	"github.com/MrLeeang/langchain-go/mcp"

	openai "github.com/sashabaranov/go-openai"
)

// buildSystemPrompt constructs the system prompt for the agent.
func buildSystemPrompt(tools []mcp.Tool) string {
	prompt := `You are an AI assistant. When you need external tools to complete a user request, you must return ONLY a valid JSON object (without any additional explanations) in the following format:
1) To call a tool, return:
{"action":"call_tool","tool":"<tool_name>","args":{...}}
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
