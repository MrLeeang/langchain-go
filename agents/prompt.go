package agents

import (
	"github.com/MrLeeang/langchain-go/mcp"

	openai "github.com/sashabaranov/go-openai"
)

// buildSystemPrompt constructs the system prompt for the agent.
func buildSystemPrompt(tools []mcp.Tool) string {
	// 	prompt := `You are an AI assistant.When you need external tools to complete a user request, you must return ONLY a valid JSON object (without any additional explanations) in the following format:
	// 1) To call a tool, return:
	// {"action":"call_tool","tool":"<tool_name>","args":{...}}
	// 2) Directly output the answer
	// `

	// prompt := `
	// You are an AI assistant. When you need external tools to complete user requests, you must output according to the following requirements:
	// 1) To call the tool, please return:
	// Please use natural language to describe the intended use of the tool,then return the following JSON object:
	// {"action":"call_tool","tool":"<tool_name>","args":{...}}
	// example:
	// 我将使用Nmap对192.168.2.235进行快速端口扫描。
	// {"action":"call_tool","tool":"nmap","args":{"target":"192.168.2.235","ports":"1-1024"}}
	// 2) Directly output the answer

	// When generating the task execution process, it is important to ensure the continuity of task execution and avoid interruptions in task execution.
	// `

	prompt := `
	# 通用Agent元提示词

	## 核心指令
	你是一个自主的任务执行Agent，必须保持完整的"思考-规划-执行"循环，确保任务连续性。

	## 执行框架
	**完整输出结构（禁止中断）：**

	**思考阶段**：分析当前状态，规划下一步行动
	**行动阶段**：根据需要调用工具或直接输出
	**观察阶段**：评估结果并更新任务状态

	保持任务连贯性：每个行动都基于之前的结果，并为后续步骤铺路。

	工具调用格式：
	【行动理由和上下文】
	{"action":"call_tool","tool":"工具名","args":{}}

	## 具体规范
	1. **禁止孤立思考** - 思考必须立即跟随行动
	2. **强制上下文连接** - 每个步骤都要引用前置结果
	3. **完整输出保证** - 单次响应必须包含完整循环
	4. **进度标记** - 明确标识任务阶段和完成度
	5. **严禁重复输出** -每个思考步骤只输出一次，不要重复相同内容

	## 在您的场景中应该这样输出：
	【状态回顾】开始对192.168.2.235进行赏金猎人攻击，当前处于初始阶段
	【当前分析】需要先进行目标情报收集以了解攻击面，选择analyze_target_intelligence工具
	【行动执行】调用目标分析工具：
	{"action":"call_tool","tool":"analyze_target_intelligence","args":{"target":"192.168.2.235"}}
	`

	if len(tools) > 0 {
		prompt += "\n\nAvailable tools (use in the following format):\n"
		for _, tool := range tools {
			prompt += tool.Description() + "\n"
		}
	}

	return prompt
}

// WithPrompt adds a custom system prompt to the agent.
// This can be used to customize the agent's behavior or add additional instructions.
func (a *Agent) WithPrompt(prompt string) *Agent {
	a.messages = append(a.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: prompt,
	})
	return a
}

// WithDebug sets the debug mode for the agent.
// Default is false.
func WithDebug(debug bool) AgentOption {
	return func(a *Agent) {
		a.debug = debug
	}
}
