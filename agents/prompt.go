package agents

import (
	"fmt"
	"strings"

	"github.com/MrLeeang/langchain-go/skills"
)

// buildSystemPrompt is used when the agent has MCP tools registered (OpenAI `tools` API).
// registered lists skills to advertise by name, description, and path; the model loads full content via read_file (or equivalent).
func buildSystemPrompt(skills []skills.Skill) string {
	var b strings.Builder
	b.WriteString(`You are an AI assistant. Use the provided function tools when they help answer the user.`)
	if len(skills) > 0 {

		b.WriteString(`
## Skills
Before replying: scan <available_skills> <description> entries.
- If exactly one skill clearly applies: read its full document at <path> with "read_file" tool, then follow it.
- If multiple could apply: choose the most specific one, then read/follow it.
- If none clearly apply: do not read any skill.
Skills (workflow playbooks) are Markdown files on disk, often with YAML front matter (---) containing name: and description: lines, followed by steps and guidance.
After reading a skill file, follow its instructions and use other tools as needed. You may read multiple skill files if relevant. Do not invent file paths; if unsure, ask the user.
When a skill file references a relative path, resolve it against the skill <path> and use that absolute path in tool commands.
`)

		b.WriteString("\n<available_skills>\n")
		for _, s := range skills {
			desc := s.Description
			if desc == "" {
				desc = "(no description)"
			}

			fmt.Fprintf(&b, "<skill>\n<name>%s</name>\n<description>%s</description>\n<path>%s</path>\n</skill>\n", s.Name, desc, s.Path)
		}

		b.WriteString("</available_skills>")
	}
	return b.String()
}

// WithPrompt adds a custom system prompt to the agent.
// This can be used to customize the agent's behavior or add additional instructions.
func (a *Agent) WithPrompt(prompt string) *Agent {
	a.Prompt = prompt

	return a
}
