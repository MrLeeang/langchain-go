package agents

import (
	"context"

	"github.com/MrLeeang/langchain-go/skills"
)

// GetOrchestrator returns a skill orchestrator instance for the agent's skills.
// This can be used to execute skills programmatically or get skill instructions.
func (a *Agent) GetOrchestrator() *skills.Orchestrator {
	return skills.NewOrchestrator(a.skillsList)
}

// GetSkills returns the list of skills configured for this agent.
func (a *Agent) GetSkills() []skills.Skill {
	return a.skillsList
}

// ExecuteSkill executes a skill by name with optional parameters.
// This adds skill instructions to the conversation context.
func (a *Agent) ExecuteSkill(ctx context.Context, skillName string, params map[string]interface{}) (string, error) {
	orchestrator := a.GetOrchestrator()
	return orchestrator.ExecuteSkill(ctx, skillName, params)
}

// SuggestSkills suggests relevant skills based on the user query.
func (a *Agent) SuggestSkills(query string) []skills.Skill {
	orchestrator := a.GetOrchestrator()
	return orchestrator.SuggestSkills(query)
}

