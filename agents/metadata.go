package agents

import "time"

// AgentMetadata contains metadata about the agent's execution, including
// conversation ID, token usage, and timing information.
type AgentMetadata struct {
	ConversationID   string        `json:"conversation_id"`
	TotalTokens      int           `json:"total_tokens"`
	PromptTokens     int           `json:"prompt_tokens"`
	CompletionTokens int           `json:"completion_tokens"`
	Duration         time.Duration `json:"duration"`
	StartTime        time.Time     `json:"start_time"`
	EndTime          time.Time     `json:"end_time"`
}

// GetMetadata returns the metadata containing conversation ID, token usage, and timing information.
func (a *Agent) GetMetadata() AgentMetadata {
	return AgentMetadata{
		ConversationID:   a.conversationID,
		TotalTokens:      a.TotalTokens,
		PromptTokens:     a.PromptTokens,
		CompletionTokens: a.CompletionTokens,
		Duration:         a.Duration,
		StartTime:        a.StartTime,
		EndTime:          a.EndTime,
	}
}
