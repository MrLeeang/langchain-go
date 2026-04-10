package llms

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/ssestream"
	"github.com/openai/openai-go/v3/shared"
	"github.com/tidwall/gjson"
)

// Config is options for [NewOpenAIModel] (OpenAI or compatible HTTP API).
type Config struct {
	BaseURL string // e.g. https://api.openai.com/v1
	APIKey  string
	Model   string

	// Thinking enables provider-specific extended thinking where supported (e.g. via chat_template_kwargs).
	Thinking bool
}

// OpenAIModel implements [LLM], [ChatStreamer], and [Embedder] using github.com/openai/openai-go/v3.
type OpenAIModel struct {
	client   openai.Client
	model    string
	thinking bool
}

// NewOpenAIModel builds a client. BaseURL/APIKey/Model come from cfg.
func NewOpenAIModel(cfg Config) *OpenAIModel {
	opts := []option.RequestOption{option.WithAPIKey(cfg.APIKey)}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}

	return &OpenAIModel{
		client:   openai.NewClient(opts...),
		model:    cfg.Model,
		thinking: cfg.Thinking,
	}
}

// NewOpenAIModelWithParams is shorthand for three string fields.
func NewOpenAIModelWithParams(baseURL, apiKey, model string) *OpenAIModel {
	return NewOpenAIModel(Config{BaseURL: baseURL, APIKey: apiKey, Model: model})
}

// Deprecated: use [NewOpenAIModelWithParams].
func NewOpenaiModel(baseURL, apiKey, model string) *OpenAIModel {
	return NewOpenAIModelWithParams(baseURL, apiKey, model)
}

// Chat calls POST /chat/completions (non-streaming).
func (m *OpenAIModel) Chat(ctx context.Context, messages []ChatCompletionMessage) (ChatCompletionResponse, error) {
	return m.ChatWithTools(ctx, messages, nil)
}

// ChatWithTools is like [OpenAIModel.Chat] but registers native OpenAI function tools when non-empty.
func (m *OpenAIModel) ChatWithTools(ctx context.Context, messages []ChatCompletionMessage, tools []openai.ChatCompletionToolUnionParam) (ChatCompletionResponse, error) {
	params := openai.ChatCompletionNewParams{
		Messages: openaiMessageParams(messages),
		Model:    shared.ChatModel(m.model),
	}
	if len(tools) > 0 {
		params.Tools = tools
	}
	m.applyThinkingParams(&params)

	resp, err := m.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return ChatCompletionResponse{}, err
	}
	return completionFromSDK(resp), nil
}

// ChatStream calls POST /chat/completions with stream=true.
func (m *OpenAIModel) ChatStream(ctx context.Context, messages []ChatCompletionMessage) (*ChatCompletionStream, error) {
	return m.ChatStreamWithTools(ctx, messages, nil)
}

// ChatStreamWithTools is like [OpenAIModel.ChatStream] with optional tools (tool_calls deltas require client-side assembly).
func (m *OpenAIModel) ChatStreamWithTools(ctx context.Context, messages []ChatCompletionMessage, tools []openai.ChatCompletionToolUnionParam) (*ChatCompletionStream, error) {
	params := openai.ChatCompletionNewParams{
		Messages: openaiMessageParams(messages),
		Model:    shared.ChatModel(m.model),
	}
	if len(tools) > 0 {
		params.Tools = tools
	}
	m.applyThinkingParams(&params)

	stream := m.client.Chat.Completions.NewStreaming(ctx, params)
	if stream.Err() != nil {
		return nil, stream.Err()
	}
	return newChatCompletionStream(stream), nil
}

func (m *OpenAIModel) applyThinkingParams(params *openai.ChatCompletionNewParams) {
	if m.thinking {
		return
	}
	ex := map[string]any{"enable_thinking": false}
	if strings.HasPrefix(m.model, "kimi-") {
		ex["thinking"] = map[string]any{"type": "disabled"}
	}

	ex["chat_template_kwargs"] = map[string]any{"enable_thinking": false}

	params.SetExtraFields(ex)
}

// Embeddings calls POST /embeddings.
func (m *OpenAIModel) Embeddings(ctx context.Context, inputs []string) ([][]float32, error) {
	resp, err := m.client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: openai.EmbeddingModel(m.model),
		Input: openai.EmbeddingNewParamsInputUnion{OfArrayOfStrings: inputs},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("未返回嵌入数据")
	}
	out := make([][]float32, len(resp.Data))
	for i, row := range resp.Data {
		vec := make([]float32, len(row.Embedding))
		for j, v := range row.Embedding {
			vec[j] = float32(v)
		}
		out[i] = vec
	}
	return out, nil
}

func openaiMessageParams(msgs []ChatCompletionMessage) []openai.ChatCompletionMessageParamUnion {
	out := make([]openai.ChatCompletionMessageParamUnion, 0, len(msgs))
	for _, msg := range msgs {
		switch msg.Role {
		case ChatMessageRoleSystem:
			out = append(out, openai.ChatCompletionMessageParamUnion{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Content: openai.ChatCompletionSystemMessageParamContentUnion{
						OfString: openai.String(msg.Content),
					},
				},
			})
		case ChatMessageRoleAssistant:
			ap := openai.ChatCompletionAssistantMessageParam{}
			if msg.Content != "" {
				ap.Content.OfString = openai.String(msg.Content)
			}
			for _, c := range msg.ToolCalls {
				ap.ToolCalls = append(ap.ToolCalls, openai.ChatCompletionMessageToolCallUnionParam{
					OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
						ID: c.ID,
						Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
							Name:      c.Name,
							Arguments: c.Arguments,
						},
					},
				})
			}
			out = append(out, openai.ChatCompletionMessageParamUnion{OfAssistant: &ap})
		case ChatMessageRoleTool:
			out = append(out, openai.ChatCompletionMessageParamUnion{
				OfTool: &openai.ChatCompletionToolMessageParam{
					ToolCallID: msg.ToolCallID,
					Content: openai.ChatCompletionToolMessageParamContentUnion{
						OfString: openai.String(msg.Content),
					},
				},
			})
		default:
			out = append(out, openai.ChatCompletionMessageParamUnion{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: openai.String(msg.Content),
					},
				},
			})
		}
	}
	return out
}

// chatToolCallsFromOpenAIMessageUnions maps SDK assistant tool_calls to [ChatToolCall].
// OpenAI-compatible providers sometimes omit tool `type` in streamed deltas; then
// [openai.ChatCompletionMessageToolCallUnion.AsAny] is nil even though function.name was merged.
// Some APIs also place `name` next to `index` instead of under `function`.
func chatToolCallsFromOpenAIMessageUnions(calls []openai.ChatCompletionMessageToolCallUnion) []ChatToolCall {
	out := make([]ChatToolCall, 0, len(calls))
	for _, u := range calls {
		tc := chatToolCallFromUnion(u)
		if tc.Name == "" {
			continue
		}
		out = append(out, tc)
	}
	return out
}

func chatToolCallFromUnion(u openai.ChatCompletionMessageToolCallUnion) ChatToolCall {
	if any := u.AsAny(); any != nil {
		switch v := any.(type) {
		case openai.ChatCompletionMessageFunctionToolCall:
			return ChatToolCall{
				ID:        v.ID,
				Name:      v.Function.Name,
				Arguments: v.Function.Arguments,
			}
		case openai.ChatCompletionMessageCustomToolCall:
			return ChatToolCall{
				ID:        v.ID,
				Name:      v.Custom.Name,
				Arguments: v.Custom.Input,
			}
		}
	}
	if u.Function.Name != "" || u.Function.Arguments != "" {
		return ChatToolCall{
			ID:        u.ID,
			Name:      u.Function.Name,
			Arguments: u.Function.Arguments,
		}
	}
	if u.Custom.Name != "" || u.Custom.Input != "" {
		return ChatToolCall{
			ID:        u.ID,
			Name:      u.Custom.Name,
			Arguments: u.Custom.Input,
		}
	}
	raw := u.RawJSON()
	if raw == "" {
		return ChatToolCall{ID: u.ID}
	}
	id := gjson.Get(raw, "id").String()
	if id == "" {
		id = u.ID
	}
	name := gjson.Get(raw, "function.name").String()
	if name == "" {
		name = gjson.Get(raw, "name").String()
	}
	if name == "" {
		name = gjson.Get(raw, "custom.name").String()
	}
	args := gjson.Get(raw, "function.arguments").String()
	if args == "" {
		args = gjson.Get(raw, "arguments").String()
	}
	if args == "" {
		args = gjson.Get(raw, "custom.input").String()
	}
	return ChatToolCall{ID: id, Name: name, Arguments: args}
}

func completionFromSDK(resp *openai.ChatCompletion) ChatCompletionResponse {
	if resp == nil {
		return ChatCompletionResponse{}
	}
	out := ChatCompletionResponse{
		ID:    resp.ID,
		Model: resp.Model,
		Usage: ChatUsage{
			PromptTokens:     int(resp.Usage.PromptTokens),
			CompletionTokens: int(resp.Usage.CompletionTokens),
			TotalTokens:      int(resp.Usage.TotalTokens),
		},
	}
	out.Choices = make([]ChatCompletionChoice, 0, len(resp.Choices))
	for _, ch := range resp.Choices {
		msg := ChatCompletionMessage{
			Role:    string(ch.Message.Role),
			Content: ch.Message.Content,
		}
		msg.ToolCalls = chatToolCallsFromOpenAIMessageUnions(ch.Message.ToolCalls)
		out.Choices = append(out.Choices, ChatCompletionChoice{
			Index:        int(ch.Index),
			Message:      msg,
			FinishReason: ch.FinishReason,
		})
	}
	return out
}

// ChatCompletionStream adapts the SDK SSE stream to Recv/Close used by agents.
type ChatCompletionStream struct {
	s *ssestream.Stream[openai.ChatCompletionChunk]
}

func newChatCompletionStream(s *ssestream.Stream[openai.ChatCompletionChunk]) *ChatCompletionStream {
	if s == nil {
		return nil
	}
	return &ChatCompletionStream{s: s}
}

// Recv returns the next chunk, or io.EOF after a normal end.
func (cs *ChatCompletionStream) Recv() (ChatCompletionStreamResponse, error) {
	if cs == nil || cs.s == nil {
		return ChatCompletionStreamResponse{}, io.EOF
	}
	if !cs.s.Next() {
		if err := cs.s.Err(); err != nil {
			return ChatCompletionStreamResponse{}, err
		}
		return ChatCompletionStreamResponse{}, io.EOF
	}
	return streamChunkFromSDK(cs.s.Current()), nil
}

// Close releases the response body.
func (cs *ChatCompletionStream) Close() error {
	if cs == nil || cs.s == nil {
		return nil
	}
	return cs.s.Close()
}

func streamChunkFromSDK(chunk openai.ChatCompletionChunk) ChatCompletionStreamResponse {
	out := ChatCompletionStreamResponse{
		ID:    chunk.ID,
		Model: chunk.Model,
	}

	raw := chunk.RawJSON()

	if u := gjson.Get(raw, "usage"); u.Exists() && u.Type != gjson.Null {
		out.Usage = &ChatUsage{
			PromptTokens:     int(u.Get("prompt_tokens").Int()),
			CompletionTokens: int(u.Get("completion_tokens").Int()),
			TotalTokens:      int(u.Get("total_tokens").Int()),
		}
	}
	if out.Usage == nil {
		if u := gjson.Get(raw, "choices.0.usage"); u.Exists() && u.Type != gjson.Null {
			out.Usage = &ChatUsage{
				PromptTokens:     int(u.Get("prompt_tokens").Int()),
				CompletionTokens: int(u.Get("completion_tokens").Int()),
				TotalTokens:      int(u.Get("total_tokens").Int()),
			}
		}
	}

	out.Choices = make([]ChatCompletionStreamChoice, 0, len(chunk.Choices))
	for _, ch := range chunk.Choices {
		reasoning := gjson.Get(raw, fmt.Sprintf("choices.%d.delta.reasoning_content", ch.Index)).String()
		tcd := make([]ChatCompletionStreamToolCallDelta, 0, len(ch.Delta.ToolCalls))
		for _, dt := range ch.Delta.ToolCalls {
			idx := int(dt.Index)
			if idx < 0 {
				idx = 0
			}
			tcd = append(tcd, ChatCompletionStreamToolCallDelta{
				Index:             idx,
				ID:                dt.ID,
				Type:              dt.Type,
				NameFragment:      dt.Function.Name,
				ArgumentsFragment: dt.Function.Arguments,
			})
		}
		out.Choices = append(out.Choices, ChatCompletionStreamChoice{
			Index: int(ch.Index),
			Delta: ChatCompletionStreamDelta{
				Content:          ch.Delta.Content,
				ReasoningContent: reasoning,
				ToolCalls:        tcd,
			},
			FinishReason: ch.FinishReason,
		})
	}

	// Fallback for OpenAI-compatible backends that omit `usage` but expose
	// token counters in `timings.prompt_n` / `timings.predicted_n`.
	// Apply only when this chunk indicates finish to avoid counting partial values.
	if out.Usage == nil {
		finished := false
		for _, ch := range out.Choices {
			if ch.FinishReason != "" {
				finished = true
				break
			}
		}
		if finished {
			promptN := int(gjson.Get(raw, "timings.prompt_n").Int())
			predictedN := int(gjson.Get(raw, "timings.predicted_n").Int())
			if promptN > 0 || predictedN > 0 {
				out.Usage = &ChatUsage{
					PromptTokens:     promptN,
					CompletionTokens: predictedN,
					TotalTokens:      promptN + predictedN,
				}
			}
		}
	}

	return out
}
