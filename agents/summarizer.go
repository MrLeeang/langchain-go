package agents

import (
	"context"
	"fmt"

	"github.com/MrLeeang/langchain-go/llms"
)

// SummarizerConfig holds configuration for the summarizer
type SummarizerConfig struct {
	LLM       llms.LLM
	MaxTokens int
	Language  string // "en" or "zh" for different prompt templates
}

// Summarizer is responsible for generating summaries using the LLM
type Summarizer struct {
	llm       llms.LLM
	maxTokens int
	language  string
}

// NewSummarizer creates a new Summarizer instance
func NewSummarizer(config SummarizerConfig) *Summarizer {
	if config.LLM == nil {
		return nil
	}

	language := config.Language
	if language == "" {
		language = "en"
	}

	maxTokens := config.MaxTokens
	if maxTokens == 0 {
		maxTokens = 500
	}

	return &Summarizer{
		llm:       config.LLM,
		maxTokens: maxTokens,
		language:  language,
	}
}

// GenerateSummaryWithContext generates a summary with additional context
func (s *Summarizer) GenerateSummaryWithContext(ctx context.Context, msgs []llms.ChatCompletionMessage) (string, error) {
	if s == nil || s.llm == nil {
		return "", fmt.Errorf("summarizer or LLM not initialized")
	}

	systemPrompt := s.buildSummaryPrompt()

	messages := []llms.ChatCompletionMessage{
		{
			Role:    llms.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
	}

	for _, msg := range msgs {
		if msg.Role == llms.ChatMessageRoleSystem {
			continue
		}

		messages = append(messages, msg)
	}

	messages = append(messages, llms.ChatCompletionMessage{
		Role:    llms.ChatMessageRoleUser,
		Content: "Please provide a summary of the above conversation.",
	})

	resp, err := s.llm.Chat(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to generate summary: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	return resp.Choices[0].Message.Content, nil
}

// buildSummaryPrompt returns the appropriate summary prompt based on language
func (s *Summarizer) buildSummaryPrompt() string {
	if s.language == "zh" {
		return s.buildChineseSummaryPrompt()
	}
	return s.buildEnglishSummaryPrompt()
}

// buildEnglishSummaryPrompt returns the English summary prompt
func (s *Summarizer) buildEnglishSummaryPrompt() string {
	return fmt.Sprintf(`You are an expert summarizer. Your task is to create clear, concise, and informative summaries.

Instructions:
1. Extract the key information and main points from the provided content
2. Organize the summary in a logical structure
3. Use clear and simple language
4. Keep the summary within approximately %d tokens
5. Focus on the most important information
6. Maintain objectivity and accuracy
7. Include relevant details that provide context
8. If there are commands, code, sql statements, etc., please keep the original text format and do not modify it.

Provide only the summary without any preamble or explanation.`, s.maxTokens)
}

// buildChineseSummaryPrompt returns the Chinese summary prompt
func (s *Summarizer) buildChineseSummaryPrompt() string {
	return fmt.Sprintf(`你是一位专业的总结专家。你的任务是创建清晰、简洁和信息丰富的摘要。

要求：
1. 从提供的内容中提取关键信息和要点
2. 以逻辑清晰的结构组织摘要
3. 使用简明扼要的语言
4. 保持摘要在约 %d 个 token 以内
5. 重点关注最重要的信息
6. 保持客观性和准确性
7. 包含提供上下文的相关细节
8. 如果有命令、代码、sql语句等，请保持原文本格式，不要进行任何修改。

仅提供摘要，不需要前言或解释。`, s.maxTokens)
}

// SetMaxTokens updates the max tokens for summary generation
func (s *Summarizer) SetMaxTokens(maxTokens int) {
	if s != nil && maxTokens > 0 {
		s.maxTokens = maxTokens
	}
}

// SetLanguage updates the language for summary prompts
func (s *Summarizer) SetLanguage(language string) {
	if s != nil && (language == "en" || language == "zh") {
		s.language = language
	}
}
