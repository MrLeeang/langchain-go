package agents

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MrLeeang/langchain-go/mcp"

	openai "github.com/sashabaranov/go-openai"
)

// parseLLMResponse parses the LLM response and determines whether to call a tool or return a final answer.
// It returns the result, whether to continue iterating, and any error.
func (a *Agent) parseLLMResponse(ctx context.Context, response string) (string, bool, error) {
	var resp struct {
		Action string                 `json:"action"`
		Tool   string                 `json:"tool,omitempty"`
		Args   map[string]interface{} `json:"args,omitempty"`
		Answer string                 `json:"answer,omitempty"`
	}

	if err := json.Unmarshal([]byte(response), &resp); err != nil {
		// If JSON parsing fails, treat the response as a final answer
		return response, false, nil
	}

	switch resp.Action {
	case "final_answer":
		return resp.Answer, false, nil

	case "call_tool":
		if resp.Tool == "" {
			return "", false, fmt.Errorf("tool name is required for call_tool action")
		}

		tool := a.findTool(resp.Tool)
		if tool == nil {
			return "", false, fmt.Errorf("tool not found: %s", resp.Tool)
		}

		toolResult, err := tool.Call(ctx, resp.Args)
		if err != nil {
			return "", false, fmt.Errorf("tool call failed for %s: %w", resp.Tool, err)
		}

		// Add tool result to conversation and continue
		toolMessage := fmt.Sprintf("Tool %s returned: %s", resp.Tool, toolResult)
		msg := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: toolMessage,
		}
		a.messages = append(a.messages, msg)

		// Save tool message to memory (optional, but helps maintain full conversation history)
		if a.mem != nil && a.conversationID != "" {
			_ = a.mem.SaveMessages(a.ctx, a.conversationID, []openai.ChatCompletionMessage{msg})
		}

		return "", true, nil

	default:
		// Unknown action, treat as final answer
		return response, false, nil
	}
}

// findTool finds a tool by name.
func (a *Agent) findTool(name string) mcp.Tool {
	for _, tool := range a.tools {
		if tool.Name() == name {
			return tool
		}
	}
	return nil
}
