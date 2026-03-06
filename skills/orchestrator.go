package skills

import (
	"context"
	"fmt"
	"strings"
)

// Orchestrator handles skill-based task orchestration.
// It can execute skills by following their defined steps and coordinating tool calls.
type Orchestrator struct {
	skills []Skill
}

// NewOrchestrator creates a new skill orchestrator with the given skills.
func NewOrchestrator(skills []Skill) *Orchestrator {
	return &Orchestrator{
		skills: skills,
	}
}

// FindSkill finds a skill by name.
func (o *Orchestrator) FindSkill(name string) *Skill {
	for i := range o.skills {
		if strings.EqualFold(o.skills[i].Name, name) {
			return &o.skills[i]
		}
	}
	return nil
}

// ListSkills returns all available skill names.
func (o *Orchestrator) ListSkills() []string {
	names := make([]string, len(o.skills))
	for i, skill := range o.skills {
		names[i] = skill.Name
	}
	return names
}

// GetSkillContent returns the full content of a skill by name.
func (o *Orchestrator) GetSkillContent(name string) (string, error) {
	skill := o.FindSkill(name)
	if skill == nil {
		return "", fmt.Errorf("skill not found: %s", name)
	}
	return skill.Content, nil
}

// ExecuteSkill executes a skill by following its steps.
// This method can be used to guide the agent's execution flow.
// It returns a formatted instruction string that can be added to the agent's context.
func (o *Orchestrator) ExecuteSkill(ctx context.Context, skillName string, params map[string]interface{}) (string, error) {
	skill := o.FindSkill(skillName)
	if skill == nil {
		return "", fmt.Errorf("skill not found: %s", skillName)
	}

	return skill.Content, nil
}

// SuggestSkills analyzes the user query and suggests relevant skills.
// This is a simple implementation that matches keywords.
func (o *Orchestrator) SuggestSkills(query string) []Skill {
	var suggestions []Skill
	queryLower := strings.ToLower(query)

	for _, skill := range o.skills {
		// Check if skill name or description contains query keywords
		if strings.Contains(strings.ToLower(skill.Name), queryLower) ||
			strings.Contains(strings.ToLower(skill.Description), queryLower) {
			suggestions = append(suggestions, skill)
		}
	}

	return suggestions
}

// GetSkillInstructions returns formatted instructions for a skill that can be
// directly injected into the agent's context or prompt.
func (o *Orchestrator) GetSkillInstructions(skillName string) (string, error) {
	skill := o.FindSkill(skillName)
	if skill == nil {
		return "", fmt.Errorf("skill not found: %s", skillName)
	}

	return skill.Content, nil
}
