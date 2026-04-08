package agents

import (
	"fmt"

	"github.com/MrLeeang/langchain-go/mcp"
	"github.com/MrLeeang/langchain-go/skills"
)

// buildSystemPrompt constructs the system prompt for the agent.
// It introduces tools and skills, but intentionally does NOT include
// the full detailed steps from skills documents. The detailed steps
// are only injected after the LLM explicitly selects a skill.
func buildSystemPrompt(tools []mcp.Tool, skillsList []skills.Skill) string {
	prompt := `You are an AI assistant. When you need external tools or skills to complete user requests, you must use them according to the following rules.

In this environment you have access to a set of tools you can use to answer the user's question.
You can use one tool per message, and will receive the result of that tool use in the user's response. You use tools step-by-step to accomplish a given task, with each tool use informed by the result of the previous tool use.

You must output according to the following requirements:

Core concept (do NOT mix them):
- Skill ("use_skill") = select a reusable workflow. After selecting a skill, you will receive the skill's detailed instructions in the next message. Skills are NOT executed directly.
- Tool ("call_tool") = execute an external function immediately. After calling a tool, you will receive the tool result and then you can continue.

Important Notice!:
The output format for skill selection and tool calling must strictly adhere to the following JSON structure in one line, no line breaks. Do NOT add any additional text before or after the JSON. Do NOT wrap the JSON in markdown code blocks (no backticks or code fences).

1) To select a skill, return ONLY the following JSON object:
   {"action":"use_skill","skill":"<skill_name>","args":{}}
   - After you return this JSON, you will receive the detailed steps for the selected skill in a new message.
   - Example: {"action":"use_skill","skill":"Skill Name"}

2) To call a tool, return ONLY the following JSON object:
   {"action":"call_tool","tool":"<tool_name>","args":{...}}
   - Example: {"action":"call_tool","tool":"execute_command","args":{"command":"ping 8.8.8.8"}}
   
   Tool Use Examples:
	   User: create a new session with python code
	   Assistant: {"action":"call_tool","tool":"create_session","args":{"session_id":"1234567890"}}
	

	   User: close the session
	   Assistant: {"action":"call_tool","tool":"close_session","args":{"session_id":"1234567890"}}

	   User: scan the target host with nmap
	   Assistant: {"action":"call_tool","tool":"nmap_scan","args":{"target":"192.168.1.1"}}
`

	// Add high-level skills information to the prompt (name + description + usage tips only)
	if len(skillsList) > 0 {
		prompt += "\n\nAvailable skills:\n\n"
		for _, skill := range skillsList {
			prompt += fmt.Sprintf("name: %s", skill.Name)
			if skill.Description != "" {
				prompt += fmt.Sprintf("\nDescription: %s", skill.Description)
			}
			prompt += "\n\n"
		}

		prompt += `
Skills Use Rules
1. A workflow begins when you select a skill and ends when you have completed all steps for that user request. Each new user request starts a completely new workflow.
2. When you need a predefined workflow, select the most suitable skill using "use_skill".
3. Do not execute tools when selecting a skill; you will receive workflow instructions first.
4. When using skills, please strictly follow the detailed steps of the skill.
5. Do not re-select the same skill repeatedly within the same workflow.
6. After receiving skill instructions, follow them exactly using "call_tool" as needed.`
	}

	// Add tools information to the prompt
	if len(tools) > 0 {
		prompt += "\n\nAvailable tools:\n\n"
		for _, tool := range tools {
			prompt += tool.Description()

			prompt += "\n\n"

		}

		prompt += `
Tools Use Rules
1. Always use the right arguments. Use actual values, not variable names.
2. Call a tool only when needed.
3. If no tool call is needed, answer directly in natural language (no JSON required).
4. Never re-do a tool call with the exact same parameters.`
	}

	return prompt
}

// WithPrompt adds a custom system prompt to the agent.
// This can be used to customize the agent's behavior or add additional instructions.
func (a *Agent) WithPrompt(prompt string) *Agent {
	// a.messages = append(a.messages, openai.ChatCompletionMessage{
	// 	Role:    openai.ChatMessageRoleSystem,
	// 	Content: prompt,
	// })

	a.messages[0].Content += "\n\nUser Instructions\n" + prompt
	a.Prompt = prompt

	if a.debug {
		fmt.Printf("System prompt set to:\n%s\n", a.messages[0].Content)
	}

	return a
}
