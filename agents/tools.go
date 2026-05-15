package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/MrLeeang/langchain-go/v2/llms"
	"github.com/MrLeeang/langchain-go/v2/mcp"
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
	return t.Description()
}

func functionParametersForTool(t mcp.Tool) shared.FunctionParameters {
	return normalizeFunctionParameters(t.ArgumentsSchema())
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

type callTool struct {
	Action string `json:"action"`
	Tool   string `json:"tool"`
	Args   any    `json:"args"`
}

func (c *callTool) String() string {
	json, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return string(json)
}

func newCallTool(tool string, args any) *callTool {
	return &callTool{
		Action: "call_tool",
		Tool:   tool,
		Args:   args,
	}
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
	type prepared struct {
		tc   llms.ChatToolCall
		tool mcp.Tool
		args map[string]interface{}
	}
	preparedCalls := make([]prepared, len(calls))

	for i, tc := range calls {
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
		preparedCalls[i] = prepared{tc: tc, tool: tool, args: args}
	}

	results := make([]string, len(calls))
	var wg sync.WaitGroup
	for i := range preparedCalls {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			p := preparedCalls[i]
			tc := p.tc
			tool := p.tool
			args := p.args

			if ch != nil {
				ct := newCallTool(tc.Name, args)
				if a.debug {
					ch <- StreamResponse{Content: "\n"}
					ch <- StreamResponse{Content: ct.String()}
					ch <- StreamResponse{Content: "\n"}
				}
				ch <- StreamResponse{ToolCall: ct}
			}

			callToolResult := newCallToolResult(tc.Name, args)

			result, err := tool.Call(ctx, args)
			if err != nil {
				result = "tool call failed for " + tc.Name + ": " + err.Error()
				callToolResult.Error = true
				callToolResult.Message = result
			}

			if ch != nil {
				callToolResult.Result = result
				if a.debug {
					ch <- StreamResponse{Content: "\n"}
					ch <- StreamResponse{Content: callToolResult.String()}
					ch <- StreamResponse{Content: "\n"}
				}
				ch <- StreamResponse{ToolCallResult: callToolResult}
			}

			results[i] = result
		}(i)
	}
	wg.Wait()

	for i, tc := range calls {
		a.messages = append(a.messages, llms.ChatCompletionMessage{
			Role:       llms.ChatMessageRoleTool,
			ToolCallID: tc.ID,
			Content:    results[i],
		})
	}
	return nil
}
