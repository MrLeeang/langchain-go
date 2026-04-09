package agents

import (
	"fmt"

	"github.com/MrLeeang/langchain-go/memory"
	openai "github.com/sashabaranov/go-openai"
)

// GetMemory returns the memory implementation used by this agent.
func (a *Agent) GetMemory() memory.Memory {
	return a.mem
}

// ClearHistory clears the conversation history for the current conversation ID.
func (a *Agent) ClearHistory() error {
	if a.mem != nil && a.conversationID != "" {
		return a.mem.ClearMessages(a.ctx, a.conversationID)
	}
	return nil
}

func (a *Agent) LoadMessages(latestUserInput string) {
	// clean tmp messages
	if a.mem != nil && a.conversationID != "" {

		// build system prompt
		if len(a.tools) > 0 || len(a.skillsList) > 0 {
			systemPrompt := buildSystemPrompt(a.tools, a.skillsList)
			a.messages = append(a.messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			})

			if a.Prompt != "" {
				a.messages[0].Content += "\n\n# User Instructions\n" + a.Prompt
			}
		}

		// Check if this is MilvusMemory with query-based loading enabled
		// If so, skip loading here - it will be loaded when we have the user query
		if milvusMem, ok := a.mem.(*memory.MilvusMemory); ok {

			a.messages = []openai.ChatCompletionMessage{}

			milvusMem.SetQuery(latestUserInput)

			if history, err := a.mem.LoadMessages(a.ctx, a.conversationID); err == nil && len(history) > 0 {
				a.messages = append(a.messages, history...)
			}
		} else {
			if history, err := a.mem.LoadMessages(a.ctx, a.conversationID); err == nil && len(history) > 0 {

				// 计算history的token数量
				tokenCounter, err := NewTokenCounter()
				if err != nil {
					a.messages = append(a.messages, history...)
					a.historyMessageIndex = len(a.messages)
					return
				}

				tokenCount := 0
				historyIndex := 0

				for index := len(history) - 1; index >= 0; index-- {
					tokenCount += tokenCounter.CountTokens(history[index].Content)
					if tokenCount > a.maxHistoryTokens {
						if index == 0 {
							historyIndex = 1
						} else {
							historyIndex = index + 1
						}
						break
					}
				}

				if historyIndex == 0 {
					a.messages = append(a.messages, history...)
				} else {

					// 触发压缩

					// 从 historyIndex 开始向后找第一个 User 消息
					originalIndex := historyIndex
					for historyIndex < len(history) && history[historyIndex].Role != openai.ChatMessageRoleUser {
						historyIndex++
					}

					// 如果没找到 User 消息，使用原始位置
					if historyIndex >= len(history) {
						historyIndex = originalIndex
					}

					// 生成Assistant概要消息，然后作为Assistant消息保存起来
					// 创建概要生成器（使用已有的大模型）
					summarizer := NewSummarizer(SummarizerConfig{
						LLM:       a.GetLLM(), // 复用现有大模型
						MaxTokens: 1000,
					})

					// 生成概要
					summary, err := summarizer.GenerateSummaryWithContext(a.ctx, history[:historyIndex])

					if err != nil {
						// 如果生成概要失败，记录错误但不影响正常流程
						fmt.Println("Error generating summary:", err)
					} else {
						// 将生成的概要作为Assistant消息保存到对话中
						summaryMsg := openai.ChatCompletionMessage{
							Role:    openai.ChatMessageRoleAssistant,
							Content: fmt.Sprintf("[System Note: Automatic summary of previous conversation]\n\n%s", summary),
						}
						a.messages = append(a.messages, summaryMsg)
					}

					// 确保摘要后跟的是 User 消息
					if history[historyIndex].Role != openai.ChatMessageRoleUser {
						a.messages = append(a.messages, openai.ChatCompletionMessage{
							Role:    openai.ChatMessageRoleUser,
							Content: "Continue from the previous conversation.",
						})
					}

					a.messages = append(a.messages, history[historyIndex:]...)

				}

			}
		}
	}

	a.historyMessageIndex = len(a.messages)
}
