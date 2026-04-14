# langchain-go

一个面向 Go 的 LangChain 风格 Agent 框架，聚焦 **ReAct + MCP Tools + 多种 Memory**，用于构建可调用工具、可持久化上下文、支持流式输出的 LLM 应用。

## 功能总览

- **ReAct Agent 执行循环**：支持多轮「思考 -> 工具调用 -> 观察 -> 回复」
- **OpenAI 兼容模型接入**：支持 OpenAI 及兼容 `/v1` 协议的网关/私有部署
- **原生工具调用**：将 MCP Tool 自动映射为 OpenAI `tools` (function calling)
- **流式输出**：支持文本增量输出、推理内容增量输出、工具调用过程透出
- **多种 Memory 实现**
  - `BufferMemory`：内存会话
  - `RedisMemory`：Redis 持久化，支持 TTL、限量读取
  - `MilvusMemory`：向量记忆，支持语义检索相关历史
  - `FileMemory`：JSON 文件持久化
  - 自定义实现 `memory.Memory` 接口
- **Skills 能力注入**：支持加载 Markdown 技能文档，注入系统提示让模型按技能执行
- **Token / 时长统计**：内置 prompt/completion/total token 与耗时统计
- **可中断执行**：支持通过 `agent.Stop()` 取消正在运行的 `Run/Stream`
- **上下文窗口压缩**：长对话可自动压缩历史，减少 token 占用

## 版本要求

- Go `1.24+`

## 安装

```bash
go get github.com/MrLeeang/langchain-go
```

## 快速开始

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/MrLeeang/langchain-go/agents"
	"github.com/MrLeeang/langchain-go/llms"
)

func main() {
	ctx := context.Background()

	llm := llms.NewOpenAIModel(llms.Config{
		BaseURL: "https://api.openai.com/v1",
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   "gpt-3.5-turbo",
	})

	agent := agents.CreateReactAgent(ctx, llm).
		WithPrompt("You are a helpful assistant.")

	resp, err := agent.Run("What is the capital of France?")
	if err != nil {
		panic(err)
	}

	fmt.Println(resp)
}
```

## 核心概念

### 1) Agent

`agents.Agent` 是执行核心，负责：

- 组装系统提示与历史消息
- 发起 LLM 对话（普通/流式）
- 识别并执行工具调用
- 将结果写回 Memory
- 统计 token 与耗时

核心创建方式：

```go
agent := agents.CreateReactAgent(ctx, llm, opts...)
```

### 2) LLM

当前内置 `llms.OpenAIModel`，通过 `llms.Config` 配置：

```go
llm := llms.NewOpenAIModel(llms.Config{
	BaseURL: "https://api.openai.com/v1",
	APIKey:  "sk-xxx",
	Model:   "gpt-4o-mini",
	Thinking: false, // 可选，兼容部分支持思考模式的后端
})
```

除聊天外，也支持 Embeddings（`Embeddings(ctx, []string)`）。

### 3) Tools（MCP）

通过 `mcp.InitializeMCP` 初始化 MCP 服务并获取工具列表，Agent 会自动将其作为 function tools 提供给模型。

支持的传输方式：

- `sse`
- `streamable_http`
- `stdio`

### 4) Memory

所有记忆实现都遵循统一接口：

```go
type Memory interface {
	LoadMessages(ctx context.Context, conversationID string) ([]llms.ChatCompletionMessage, error)
	SaveMessages(ctx context.Context, conversationID string, messages []llms.ChatCompletionMessage) error
	ClearMessages(ctx context.Context, conversationID string) error
}
```

### 5) Skills

可通过 `skills.LoadDirectory` 或 `skills.LoadFiles` 加载 Markdown 技能文档，并使用 `agents.WithSkills(...)` 注入。  
Agent 会将技能元信息注入系统提示，模型可按路径读取技能内容后执行。

## 常用 API

### Agent 创建与选项

- `agents.CreateReactAgent(ctx, llm, opts...)`
- `agents.WithTools(tools []mcp.Tool)`
- `agents.WithSkills(skills []skills.Skill)`
- `agents.WithMemory(mem memory.Memory)`
- `agents.WithConversationID(id string)`
- `agents.WithMaxIterations(n int)`
- `agents.WithDebug(debug bool)`
- `agents.WithMaxWindowTokens(tokens int)`

### Agent 方法

- `agent.Run(message string) (string, error)`
- `agent.RunWithContext(ctx, message)`
- `agent.Stream(message string) <-chan agents.StreamResponse`
- `agent.StreamWithContext(ctx, message)`
- `agent.WithPrompt(prompt string) *Agent`
- `agent.Stop()`：中断当前执行
- `agent.ClearHistory()`：清空当前会话历史
- `agent.GetMetadata()`：获取 token 与时间信息

### 统计相关

- `agent.GetPromptTokens()`
- `agent.GetCompletionTokens()`
- `agent.GetTotalTokens()`
- `agent.GetDuration()`

## 配置示例

### LLM 配置

```go
llms.Config{
	BaseURL: "https://api.openai.com/v1",
	APIKey:  os.Getenv("OPENAI_API_KEY"),
	Model:   "gpt-4o-mini",
	Thinking: false,
}
```

### MCP 配置

```go
configs := []*mcp.Config{
	{
		Name:      "my-server",
		Transport: "sse", // sse | streamable_http | stdio
		URL:       "http://localhost:8080/sse",
		Disabled:  false,
	},
	{
		Name:      "local-stdio",
		Transport: "stdio",
		Command:   "node",
		Args:      []string{"./server.js"},
	},
}
```

### Memory 配置（Redis）

```go
mem, err := memory.NewRedisMemoryWithConfig(memory.RedisConfig{
	Address:   "localhost",
	Port:      6379,
	Password:  "",
	DB:        0,
	TTL:       24 * time.Hour,
	KeyPrefix: "langchain:memory:",
})
```

### Memory 配置（File）

```go
mem := memory.NewFileMemory("./data/memory.json")
```

## 示例目录（完整）

仓库内可直接运行的示例位于 `examples/`：

- `examples/simple-run`：最简 `Run` 调用
- `examples/stream-example`：流式输出
- `examples/buffer-memory`：内存会话
- `examples/file-memory`：文件持久化会话（JSON）
- `examples/redis-memory`：Redis 持久化会话
- `examples/milvus-memory`：Milvus 语义记忆
- `examples/agent-tools`：MCP 工具调用
- `examples/skills`：Skills + MCP + Memory 组合
- `examples/metadata`：运行元数据与 Token 统计
- `examples/stop-stream`：流式输出中断（`agent.Stop()`）
- `examples/thinking-mode`：开启 `Thinking: true` 的思考模式示例

运行示例：

```bash
go run ./examples/simple-run
go run ./examples/stream-example
go run ./examples/agent-tools
go run ./examples/file-memory
go run ./examples/thinking-mode
```

## 典型使用模式

### 1) Agent + Tools + Memory

```go
tools, err := mcp.InitializeMCP(ctx, configs)
if err != nil {
	panic(err)
}

mem := memory.NewBufferMemory()

agent := agents.CreateReactAgent(ctx, llm,
	agents.WithTools(tools),
	agents.WithMemory(mem),
	agents.WithConversationID("user-123"),
	agents.WithMaxIterations(10),
).WithPrompt("You are a helpful assistant.")
```

### 2) 流式输出

```go
ch := agent.Stream("Explain RAG in simple words")
for resp := range ch {
	if resp.Error != nil {
		panic(resp.Error)
	}
	if resp.ReasoningContent != "" {
		fmt.Print(resp.ReasoningContent)
	}
	if resp.Content != "" {
		fmt.Print(resp.Content)
	}
	if resp.Done {
		break
	}
}
```

### 3) 自定义 Memory

```go
type MyMemory struct{}

func (m *MyMemory) LoadMessages(ctx context.Context, id string) ([]llms.ChatCompletionMessage, error) {
	return nil, nil
}
func (m *MyMemory) SaveMessages(ctx context.Context, id string, msgs []llms.ChatCompletionMessage) error {
	return nil
}
func (m *MyMemory) ClearMessages(ctx context.Context, id string) error {
	return nil
}
```

## 项目结构

```text
langchain-go/
├── agents/      # ReAct Agent 主流程、流式处理、工具执行、统计与中断
├── llms/        # OpenAI 兼容 LLM 封装（聊天 + 流式 + 向量）
├── mcp/         # MCP 配置、连接、工具枚举与调用
├── memory/      # Buffer / Redis / Milvus / File Memory
├── skills/      # Skills 加载与 Front Matter 解析
├── examples/    # 官方示例
└── examples-my/ # 自定义示例
```

## 常见问题

### 1) 模型报鉴权或地址错误

检查：

- `BaseURL` 是否带 `/v1`
- `APIKey` 是否有效
- `Model` 名称是否被当前服务支持

### 2) MCP 工具初始化失败

检查：

- `Transport` 与配置字段是否匹配（`stdio` 需 `Command`）
- `URL` 是否可访问
- MCP 服务是否已启动并可 `ListTools`

### 3) 历史对话未生效

检查：

- 是否设置了 `agents.WithConversationID(...)`
- Memory 实例是否正确注入 `agents.WithMemory(...)`
- 外部存储（Redis/Milvus）是否可连接

## License

Apache License 2.0，见 [LICENSE](LICENSE)。

## 致谢

- [openai-go](https://github.com/openai/openai-go)
- [mcp-go](https://github.com/mark3labs/mcp-go)
