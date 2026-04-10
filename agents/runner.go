package agents

import (
	"context"
	"fmt"
	"time"

	"github.com/MrLeeang/langchain-go/llms"
)

// Run processes a user message and returns the agent's response.
// It handles tool calling iteratively until a final answer is reached or max iterations are exceeded.
func (a *Agent) Run(message string) (string, error) {
	a.ResetTokenUsage()
	a.ResetDuration()

	a.LoadMessages(message)

	// Cancel any previous run/stream if still active
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}

	// Create a cancellable context so that Stop() can interrupt this run.
	ctx, cancel := context.WithCancel(a.ctx)
	a.cancel = cancel
	defer func() {
		if a.cancel != nil {
			a.cancel()
			a.cancel = nil
		}
	}()

	return a.RunWithContext(ctx, message)
}

// RunWithContext processes a user message with a custom context and returns the agent's response.
func (a *Agent) RunWithContext(ctx context.Context, message string) (string, error) {
	a.StartTime = time.Now()
	defer func() {
		a.EndTime = time.Now()
		a.Duration = a.EndTime.Sub(a.StartTime)
	}()

	userMsg := llms.ChatCompletionMessage{
		Role:    llms.ChatMessageRoleUser,
		Content: message,
	}
	a.messages = append(a.messages, userMsg)

	defer func() {
		// 生成Assistant概要消息，然后作为Assistant消息保存起来
		// 创建概要生成器（使用已有的大模型）
		// summarizer := NewSummarizer(SummarizerConfig{
		// 	LLM:       a.GetLLM(), // 复用现有大模型
		// 	MaxTokens: 500,
		// })

		// // 生成概要
		// summary, err := summarizer.GenerateSummaryWithContext(ctx, a.messages[a.historyMessageIndex:])

		// if err != nil {
		// 	// 如果生成概要失败，记录错误但不影响正常流程
		// 	fmt.Println("Error generating summary:", err)
		// 	return
		// }

		// // 将生成的概要作为Assistant消息保存到对话中
		// summaryMsg := openai.ChatCompletionMessage{
		// 	Role:    openai.ChatMessageRoleAssistant,
		// 	Content: fmt.Sprintf("Conversation Summary:\n%s", summary),
		// }
		// a.messages = append(a.messages, summaryMsg)

		// // 可选：将概要消息保存到内存中，以便后续查询使用
		// if a.mem != nil && a.conversationID != "" {
		// 	if err := a.mem.SaveMessages(ctx, a.conversationID, []openai.ChatCompletionMessage{summaryMsg}); err != nil {
		// 		fmt.Println("Error saving summary to memory:", err)
		// 	}
		// }

		if a.mem != nil && a.conversationID != "" {
			// user message already saved to memory in handleStreamResponse
			if err := a.mem.SaveMessages(ctx, a.conversationID, a.messages[a.historyMessageIndex:]); err != nil {
				fmt.Println("Error saving messages to memory:", err)
			}
		}
	}()

	iterations := 0
	for iterations < a.maxIter {
		iterations++

		// If the context has been cancelled (via Stop or parent ctx),
		// abort early.
		if err := ctx.Err(); err != nil {
			return "", err
		}

		resp, err := a.completeLLMTurn(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get LLM response: %w", err)
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("no response from LLM")
		}

		assistantMsg := resp.Choices[0].Message
		a.messages = append(a.messages, assistantMsg)

		a.CalculateCompletionTokenUsage(resp.Usage)

		if len(assistantMsg.ToolCalls) > 0 {
			if err := a.executeNativeToolCalls(ctx, nil, assistantMsg.ToolCalls); err != nil {
				return "", err
			}
			continue
		}

		return assistantMsg.Content, nil
	}

	return "", fmt.Errorf("max iterations (%d) exceeded", a.maxIter)
}

// completeLLMTurn uses OpenAI native tools when the LLM is [*llms.OpenAIModel] and MCP tools are configured.
func (a *Agent) completeLLMTurn(ctx context.Context) (llms.ChatCompletionResponse, error) {
	if om, ok := a.llm.(*llms.OpenAIModel); ok && len(a.tools) > 0 {
		return om.ChatWithTools(ctx, a.messages, OpenAICompletionTools(a.tools))
	}
	return a.llm.Chat(ctx, a.messages)
}
