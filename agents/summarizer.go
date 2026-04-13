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
	return fmt.Sprintf(`You are a professional conversation memory manager. From the following conversation history, extract key information worth remembering long-term, and organize the output into four categories. Ignore trivial chatter, temporary content, or outdated information.

## Extraction Rules
1. **Important decisions and reasons**: Explicit choices, commitments, directional decisions made by the user, along with the rationale or context behind those decisions.
2. **Lessons learned**: Reusable insights from successes or failures, pitfalls to avoid, methods proven effective.
3. **Preferences and habits**: Repeatedly expressed likes, dislikes, work/life habits, communication style, or tool preferences.
4. **Project progress and status**: Key milestones, completed items, unresolved blockers, next-stage goals.

## Special Formatting Requirement
- If the conversation contains **commands, code, SQL statements**, or any other executable or structured text, you **must preserve the original text format** exactly, including:
  - All indentation (spaces or tabs)
  - Line breaks
  - Letter case
  - Punctuation
  - Comments
- Do not modify, translate, reflow, grammar-check, or reformat these contents in any way.
- If you need to provide context or explanation in the output, place the original content inside a Markdown code block . The characters inside the code block must be identical to the original.

## Output Length Limit
- Keep the entire summary within **%d tokens**.
- Prioritize the most core, long‑term valuable information across the four categories. If the limit is exceeded, add a note at the end: "(some details omitted due to length limit)".

## Output Format
If a category has no new content to extract, omit that category. Describe each piece of information under each category with concise phrases or short sentences, avoiding redundancy.

Please output the long-term memory summary。`, s.maxTokens)
}

// buildChineseSummaryPrompt returns the Chinese summary prompt
func (s *Summarizer) buildChineseSummaryPrompt() string {
	return fmt.Sprintf(`你是一个专业的会话记忆管理器。请从以下对话历史中，提取值得长期记住的关键信息，并按照四个类别组织输出。忽略琐碎的闲聊、临时性内容或已经过时的信息。

## 提取规则
1. **重要决定和原因**：用户明确做出的选择、承诺、方向性决策，以及做出该决定的理由或背景。
2. **学到的经验教训**：从成功或失败中总结出的可复用的认知、需要避免的坑、验证过有效的方法。
3. **偏好和习惯**：用户反复表露的喜好、厌恶、工作/生活习惯、沟通风格或工具使用倾向。
4. **项目进展和状态**：项目当前的关键里程碑、已完成事项、待解决阻塞点、下一阶段目标。

## 特殊格式要求
- 如果对话中出现**命令、代码、SQL 语句**等任何可执行或结构化文本，在提取时**必须保持原始文本格式**，包括：
  - 所有缩进（空格或制表符）
  - 换行位置
  - 大小写
  - 标点符号
  - 注释内容
- 不得对这些内容进行任何修改、翻译、重排、语法润色或格式转换。
- 如需在输出中提供上下文说明，请将原始内容放入 Markdown 代码块中，代码块内的字符必须与原文一字不差。

## 输出长度限制
- 请控制整个摘要的输出不超过 **%d tokens**。
- 优先保留四类中最核心、最具长期价值的信息。若超出限制，可在末尾注明“（因长度限制省略了部分细节）”。

## 输出格式
如果某类没有新内容可提取，则省略该类。每类下的每条信息用简洁的短语或短句描述，避免冗余。

请输出长期记忆摘要。`, s.maxTokens)
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
