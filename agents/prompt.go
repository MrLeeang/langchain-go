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
	b.WriteString(`You are an AI assistant. Use the provided function tools when they help answer the user.

Skills (workflow playbooks) are Markdown files on disk, often with YAML front matter (---) containing name: and description: lines, followed by steps and guidance.

When a user request matches a skill, load that playbook with the read_file tool (or the file-reading tool your environment exposes — use its actual name and parameters from the tool list). Pass the exact path given in the registered list below.

After reading a skill file, follow its instructions and use other tools as needed. You may read multiple skill files if relevant. Do not invent file paths; if unsure, ask the user.
`)
	if len(skills) > 0 {
		b.WriteString("\n## Available skills\n")
		b.WriteString("Use your file tool with the path on the line below each entry to load the full document.\n\n")
		for i, s := range skills {
			desc := s.Description
			if desc == "" {
				desc = "(no description)"
			}
			fmt.Fprintf(&b, "%d. **%s** — %s\n   Path: `%s`\n\n", i+1, s.Name, desc, s.Path)
		}
	}
	return b.String()
}

// WithPrompt adds a custom system prompt to the agent.
// This can be used to customize the agent's behavior or add additional instructions.
func (a *Agent) WithPrompt(prompt string) *Agent {
	a.Prompt = prompt

	return a
}
