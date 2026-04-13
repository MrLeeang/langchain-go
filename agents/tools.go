package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MrLeeang/langchain-go/llms"
	"github.com/MrLeeang/langchain-go/mcp"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

// chatStream starts a chat completion stream with optional native tools when the LLM is [*llms.OpenAIModel].
func (a *Agent) chatStream(ctx context.Context) (*llms.ChatCompletionStream, error) {
	om, ok := a.llm.(*llms.OpenAIModel)
	if !ok {
		return nil, fmt.Errorf("streaming requires *llms.OpenAIModel")
	}
	var toolParams []openai.ChatCompletionToolUnionParam
	if len(a.tools) > 0 {
		toolParams = OpenAICompletionTools(a.tools)
	}
	return om.ChatStreamWithTools(ctx, a.messages, toolParams)
}

// OpenAICompletionTools builds OpenAI Chat Completions `tools` from MCP tools (function definitions).
func OpenAICompletionTools(tools []mcp.Tool) []openai.ChatCompletionToolUnionParam {
	out := make([]openai.ChatCompletionToolUnionParam, 0, len(tools))
	for _, t := range tools {
		out = append(out, openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
			Name:        t.Name(),
			Description: openai.String(toolModelDescription(t)),
			Parameters:  functionParametersForTool(t),
		}))
	}
	return out
}

func toolModelDescription(t mcp.Tool) string {
	if mt, ok := t.(*mcp.MCPTool); ok {
		return mt.ModelDescription()
	}
	return t.Description()
}

func functionParametersForTool(t mcp.Tool) shared.FunctionParameters {
	if mt, ok := t.(*mcp.MCPTool); ok {
		return normalizeFunctionParameters(mt.ArgumentsSchema())
	}
	return defaultObjectParameters()
}

func defaultObjectParameters() shared.FunctionParameters {
	return shared.FunctionParameters{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func normalizeFunctionParameters(schema any) shared.FunctionParameters {
	if schema == nil {
		return defaultObjectParameters()
	}
	switch v := schema.(type) {
	case map[string]any:
		return v
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return defaultObjectParameters()
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			return defaultObjectParameters()
		}
		return m
	}
}

// findTool finds a tool by name.
func (a *Agent) findTool(name string) mcp.Tool {
	for _, t := range a.tools {
		if t.Name() == name {
			return t
		}
	}
	return nil
}

type callToolResult struct {
	Action  string `json:"action"`
	Tool    string `json:"tool"`
	Args    any    `json:"args"`
	Result  string `json:"result"`
	Error   bool   `json:"error"`
	Message string `json:"message"`
}

func (c *callToolResult) String() string {
	json, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return string(json)
}

func newCallToolResult(tool string, args any) *callToolResult {
	return &callToolResult{
		Action:  "tool_result",
		Tool:    tool,
		Args:    args,
		Result:  "",
		Error:   false,
		Message: "",
	}
}

func (a *Agent) executeNativeToolCalls(ctx context.Context, ch chan<- StreamResponse, calls []llms.ChatToolCall) error {
	for _, tc := range calls {
		if strings.TrimSpace(tc.Name) == "" {
			return fmt.Errorf("tool call has empty function name (tool_call_id=%q)", tc.ID)
		}
		tool := a.findTool(tc.Name)
		if tool == nil {
			return fmt.Errorf("tool not found: %s", tc.Name)
		}
		var args map[string]interface{}
		if tc.Arguments != "" && tc.Arguments != "null" {
			if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
				return fmt.Errorf("invalid tool arguments JSON for %s: %w", tc.Name, err)
			}
		}
		if args == nil {
			args = map[string]interface{}{}
		}

		if ch != nil {
			// send json message to channel
			jsonMessage, err := json.Marshal(map[string]interface{}{
				"action": "call_tool",
				"tool":   tc.Name,
				"args":   args,
			})
			if err == nil {
				ch <- StreamResponse{Content: "\n"}
				ch <- StreamResponse{Content: string(jsonMessage)}
				ch <- StreamResponse{Content: "\n"}
			}
		}

		callToolResult := newCallToolResult(tc.Name, args)

		result, err := tool.Call(ctx, args)
		if err != nil {
			result = "tool call failed for " + tc.Name + ": " + err.Error()
			callToolResult.Error = true
			callToolResult.Message = result
		} else {
			runes := []rune(result)

			if len(runes) > 1000 && tc.Name != "read_file" {
				// truncate result to 1000 characters to avoid overwhelming the model with too much tool output, which can lead to context window issues and degraded performance. The full result is still included in the tool_result message sent to the channel and added to the agent's messages, so the model can access it if needed.
				result = string(runes[:1000]) + "...(truncated)"
			}
		}

		if ch != nil {
			// send json message to channel
			callToolResult.Result = result

			if err == nil {
				ch <- StreamResponse{Content: "\n"}
				ch <- StreamResponse{Content: callToolResult.String()}
				ch <- StreamResponse{Content: "\n"}
			}
		}

		a.messages = append(a.messages, llms.ChatCompletionMessage{
			Role:       llms.ChatMessageRoleTool,
			ToolCallID: tc.ID,
			Content:    result,
		})
	}
	return nil
}
