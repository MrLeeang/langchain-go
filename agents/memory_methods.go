package agents

import (
	"fmt"

	"github.com/MrLeeang/langchain-go/llms"
	"github.com/MrLeeang/langchain-go/memory"
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
		systemPrompt := buildSystemPrompt(a.registeredSkills)
		a.messages = []llms.ChatCompletionMessage{
			{
				Role:    llms.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
		}

		if a.Prompt != "" {
			a.messages[0].Content += "\n\n# User Instructions\n" + a.Prompt
		}

		if a.debug {
			fmt.Printf("System prompt set to:\n%s\n", a.messages[0].Content)
		}

		// Check if this is MilvusMemory with query-based loading enabled
		// If so, skip loading here - it will be loaded when we have the user query
		if milvusMem, ok := a.mem.(*memory.MilvusMemory); ok {

			milvusMem.SetQuery(latestUserInput)

			if history, err := a.mem.LoadMessages(a.ctx, a.conversationID); err == nil && len(history) > 0 {
				a.messages = append(a.messages, history...)
			}
		} else {
			if history, err := a.mem.LoadMessages(a.ctx, a.conversationID); err == nil && len(history) > 0 {

				tokenCount := 0
				historyIndex := 0

				for index := len(history) - 1; index >= 0; index-- {
					tokenCount += CountTokens(history[index].Content)
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
					for historyIndex < len(history) && history[historyIndex].Role != llms.ChatMessageRoleUser {
						historyIndex++
					}

					// 如果没找到 User 消息，使用原始位置
					if historyIndex >= len(history) {
						historyIndex = originalIndex
					}

					// generate summary for messages before historyIndex to save tokens, and keep the conversation context in the remaining messages
					// initialize summarizer with the same LLM as the agent, and a reasonable max token limit for summaries
					summarizer := NewSummarizer(SummarizerConfig{
						LLM:       a.GetLLM(), // 复用现有大模型
						MaxTokens: 2000,
					})

					// generate summary with context (only summarize the messages that are being removed to save tokens)
					summary, err := summarizer.GenerateSummaryWithContext(a.ctx, history[:historyIndex])

					if err != nil {
						// if summary generation fails, we can choose to either skip saving the summary or save an error message as a system note
						fmt.Println("Error generating summary:", err)
					} else {
						// save the summary as an Assistant message in the conversation history, so it can be used in future interactions
						summaryMsg := llms.ChatCompletionMessage{
							Role:    llms.ChatMessageRoleAssistant,
							Content: fmt.Sprintf("[System Note: Automatic summary of previous conversation]\n\n%s", summary),
						}
						a.messages = append(a.messages, summaryMsg)
					}

					// find the first User message in the remaining history to ensure the conversation flow is correct
					if history[historyIndex].Role != llms.ChatMessageRoleUser {
						a.messages = append(a.messages, llms.ChatCompletionMessage{
							Role:    llms.ChatMessageRoleUser,
							Content: "Continue from the previous conversation.",
						})
					}

					a.messages = append(a.messages, history[historyIndex:]...)

					// clear memory and save the new messages with summary
					if err := a.mem.ClearMessages(a.ctx, a.conversationID); err != nil {
						fmt.Println("Error clearing memory:", err)
					} else {
						if len(a.messages) > 1 {
							// saveMessages expects messages without system prompt, so we skip the first message
							if err := a.mem.SaveMessages(a.ctx, a.conversationID, a.messages[1:]); err != nil {
								fmt.Println("Error saving messages to memory:", err)
							}
						}
					}

				}
			}
		}
	}

	a.historyMessageIndex = len(a.messages)
}
