package agents

import (
	"fmt"

	"github.com/MrLeeang/langchain-go/mcp"
	"github.com/MrLeeang/langchain-go/skills"

	openai "github.com/sashabaranov/go-openai"
)

// buildSystemPrompt constructs the system prompt for the agent.
// It introduces tools and skills, but intentionally does NOT include
// the full detailed steps from skills documents. The detailed steps
// are only injected after the LLM explicitly selects a skill.
func buildSystemPrompt(tools []mcp.Tool, skillsList []skills.Skill) string {
	prompt := `
You are an AI assistant. When you need external tools or skills to complete user requests, 
you must output according to the following requirements:

1) To select and apply a skill, return ONLY the following JSON object (without markdown code blocks):
   {"action":"use_skill","skill":"<skill_name>","args":{...}}
   - Do NOT wrap the JSON in markdown code blocks (no backticks or code fences).
   - Do NOT include the detailed steps of the skill yourself.
   - After you return this JSON, you will receive the detailed steps for the selected skill
     in a new message, and then you should follow those steps to continue the task.

2) To call a tool directly, return ONLY the following JSON object (without markdown code blocks):
   {"action":"call_tool","tool":"<tool_name>","args":{...}}
   - Do NOT wrap the JSON in markdown code blocks (no backticks or code fences).
`

	// Add high-level skills information to the prompt (name + description + usage tips only)
	if len(skillsList) > 0 {
		prompt += "\n\nAvailable skills for task orchestration (high-level overview only):\n"
		for _, skill := range skillsList {
			prompt += fmt.Sprintf("- %s", skill.Name)
			if skill.Description != "" {
				prompt += fmt.Sprintf(": %s", skill.Description)
			}
			prompt += "\n"
			// Add usage tips if available
			if len(skill.UsageTips) > 0 {
				for _, tip := range skill.UsageTips {
					prompt += fmt.Sprintf("  Usage: %s\n", tip)
				}
			}
		}

		prompt += `

When appropriate, choose the most suitable skill using the "use_skill" action.
You do NOT need to remember the detailed steps; they will be provided to you after selection.
`
	}

	// Add tools information to the prompt
	if len(tools) > 0 {
		prompt += "\n\nAvailable tools:\n"
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
