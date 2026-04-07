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
	prompt := `You are an AI assistant. When you need external tools or skills to complete user requests.

In this environment you have access to a set of tools you can use to answer the user's question. \\
You can use one tool per message, and will receive the result of that tool use in the user's response. You use tools step-by-step to accomplish a given task, with each tool use informed by the result of the previous tool use.

you must output according to the following requirements:

Core concept (do NOT mix them):
- Skill ("use_skill") = select a reusable workflow. After selecting a skill, you will receive the skill's detailed instructions in the next message. Skills are NOT executed directly.
- Tool ("call_tool") = execute an external function immediately. After calling a tool, you will receive the tool result and then you can continue.

Important Notice!:
The output format of the skill (use_skill) and calling tool (call_tool) must strictly adhere to the following JSON structure in one line, no line breaks, and can only output JSON without adding any additional text or markdown format
The output format of the skill (use_skill) and calling tool (call_tool) must strictly adhere to the following JSON structure in one line, no line breaks, and can only output JSON without adding any additional text or markdown format
The output format of the skill (use_skill) and calling tool (call_tool) must strictly adhere to the following JSON structure in one line, no line breaks, and can only output JSON without adding any additional text or markdown format

1) To select and apply a skill, return ONLY the following JSON object (without markdown code blocks):
   {"action":"use_skill","skill":"<skill_name>","args":{...}}
   - Do NOT wrap the JSON in markdown code blocks (no backticks or code fences).
   - Do NOT include the detailed steps of the skill yourself.
   - After you return this JSON, you will receive the detailed steps for the selected skill
     in a new message, and then you should follow those steps to continue the task.
   - json must be output in one line, no line breaks

   examples (Skill selection):
	   # search host
       User: search for google
       Assistant: {"action":"use_skill","skill":"search-host"}

2) To call a tool directly, return ONLY the following JSON object (without markdown code blocks):
   {"action":"call_tool","tool":"<tool_name>","args":{...}}
   - Do NOT wrap the JSON in markdown code blocks (no backticks or code fences).
   - json must be output in one line, no line breaks
   
   examples (Tool execution):
       # create session
	   User: create a new session with python code
	   Assistant: {"action":"call_tool","tool":"create_session","args":{"session_id":"1234567890"}}
	

	   # close session
	   User: close the session
	   Assistant: {"action":"call_tool","tool":"close_session","args":{"session_id":"1234567890"}}

	   # nmap scan
	   User: scan the target host with nmap
	   Assistant: {"action":"call_tool","tool":"nmap_scan","args":{"target":"192.168.1.1"}}
`

	// Add high-level skills information to the prompt (name + description + usage tips only)
	if len(skillsList) > 0 {
		prompt += "\n\nAvailable skills for workflow orchestration (high-level overview only; selecting is done via \"use_skill\"):\n\n"
		for _, skill := range skillsList {
			prompt += fmt.Sprintf("name: %s", skill.Name)
			if skill.Description != "" {
				prompt += fmt.Sprintf(": %s", skill.Description)
			}
			prompt += "\n\n"
		}

		prompt += `

When you need a predefined workflow, choose the most suitable skill using the "use_skill" action.
Do not execute tools when selecting a skill; you will receive workflow instructions first, then call tools in later turns if needed.
`
	}

	// Add tools information to the prompt
	if len(tools) > 0 {
		prompt += "\n\nAvailable tools (execute immediately via \"call_tool\"):\n"
		for _, tool := range tools {
			prompt += tool.Description() + "\n"
		}
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

	a.messages[0].Content += "\n\n# User Instructions\n" + prompt
	a.Prompt = prompt

	if a.debug {
		fmt.Printf("System prompt set to:\n%s\n", a.messages[0].Content)
	}

	return a
}
