# Gopilot

一个基于OpenAI的Go语言AI助手，提供智能代码分析和文件操作功能。采用模块化架构设计，支持插件化工具系统和并发执行。

## 功能特性

-  基于大模型的智能对话
-  文件读取和目录列表
-  代码搜索（使用ripgrep）
-  文件编辑和创建
-  Bash命令执行
-  可扩展的插件化工具系统
-  并发工具执行与性能优化
-  多步推理链支持
-  结构化日志系统

## 快速开始

### 环境要求

- Go 1.23.4 或更高版本
- OpenAI API密钥
- ripgrep (用于代码搜索功能)

### 安装

1. 克隆项目：
```bash
git clone <repository-url>
cd gocopilot
```

2. 安装依赖：
```bash
go mod tidy
```

3. 配置环境变量：
```bash
cp .env.example .env
```

编辑 `.env` 文件，添加你的OpenAI API配置：
```env
OPENAI_API_KEY=your_openai_api_key_here
OPENAI_API_BASE_URL=https://api.openai.com/v1  # 可选，默认为OpenAI官方API
MODEL=gpt-4  # 可选，默认为gpt-4
MEMORY_CAPACITY=40  # 对话历史容量
MAX_CONCURRENCY=5  # 最大并发工具执行数
MAX_TOKENS=1024  # 最大响应token数
```

### 运行

```bash
# 开发模式运行（推荐用于工具调用）
go run ./cmd/gocopilot

# 构建二进制文件
go build ./cmd/gocopilot
./gocopilot

# 启用详细日志
go run ./cmd/gocopilot -verbose

# 启用多步推理模式
go run ./cmd/gocopilot -reasoning

# 同时启用详细日志和推理模式
go run ./cmd/gocopilot -verbose -reasoning
```

## 项目结构

```
gocopilot/
├── cmd/
│   └── gocopilot/
│       └── main.go          # CLI入口点
├── internal/
│   ├── agent/
│   │   ├── agent.go         # 智能代理核心逻辑
│   │   ├── executor.go      # 并发工具执行器
│   │   ├── memory.go        # 对话历史管理
│   │   ├── reasoning.go     # 多步推理链
│   │   └── types.go         # 接口定义
│   ├── tools/
│   │   ├── tools.go         # 工具定义和实现
│   │   ├── registry.go      # 工具注册系统
│   │   └── builtin.go       # 内置工具注册
│   ├── config/
│   │   └── config.go        # 配置管理
│   └── logger/
│       ├── logger.go        # 结构化日志
│       └── noop.go          # 空日志实现
├── go.mod                   # Go模块定义
├── go.sum                   # 依赖校验和
├── .env.example             # 环境变量示例
└── LICENSE                  # 许可证文件
```

## 可用工具

### 1. 文件读取 (`read_file`)
读取指定路径的文件内容。

### 2. 目录列表 (`list_files`)
列出指定目录下的文件和子目录。

### 3. Bash命令执行 (`bash`)
执行shell命令并返回输出。

### 4. 文件编辑 (`edit_file`)
搜索并替换文件中的文本内容。

### 5. 代码搜索 (`code_search`)
使用ripgrep在代码库中搜索模式。

## 使用示例

启动程序后，你可以与Gocopilot进行交互：

```
Chat with Gocopilot (use 'ctrl-c' to quit)
You: 帮我读取main.go文件
Gocopilot: 好的，我来帮你读取main.go文件
tool: read_file({"path": "cmd/gocopilot/main.go"})
result: [文件内容...]
Gocopilot: 这是main.go文件的内容...
```

## 开发指南

### 添加新工具

1. 在 [`internal/tools/tools.go`](internal/tools/tools.go) 中定义输入结构：
```go
type NewToolInput struct {
    Param string `json:"param" jsonschema_description:"参数描述"`
}
```

2. 创建工具函数（支持结构化日志）：
```go
func NewTool(input json.RawMessage, log interface{ Debug(format string, args ...interface{}); Error(format string, args ...interface{}); Warn(format string, args ...interface{}) }) (string, error) {
    // 工具实现，使用log进行结构化日志记录
    log.Debug("Executing new tool with param: %s", param)
    // ...
}
```

3. 注册工具定义：
```go
var NewToolDefinition = ToolDefinition{
    Name:        "new_tool",
    Description: "工具描述",
    InputSchema: GenerateSchema[NewToolInput](),
    Function:    NewTool,
}
```

4. 在 [`internal/tools/builtin.go`](internal/tools/builtin.go) 中添加工具到内置工具列表。

工具系统会自动处理JSON schema生成、并发执行和错误处理。

### 测试

项目目前没有测试文件。当添加测试时：

运行所有测试：
```bash
go test ./...
```

运行特定测试：
```bash
go test ./internal/agent -run TestName
```

### 代码规范

- 使用 `gofmt` 格式化代码
- 遵循Go标准命名约定
- 为导出的标识符提供完整的文档注释

## 配置选项

### 环境变量

- `OPENAI_API_KEY`: OpenAI API密钥（必需）
- `OPENAI_API_BASE_URL`: OpenAI API基础URL（可选）
- `MODEL`: 使用的模型名称（可选，默认：gpt-4）
- `MEMORY_CAPACITY`: 对话历史容量（可选，默认：40）
- `MAX_CONCURRENCY`: 最大并发工具执行数（可选，默认：5）
- `MAX_TOKENS`: 最大响应token数（可选，默认：1024）
- `SYSTEM_MESSAGE`: 系统提示消息（可选）

### 命令行参数

- `-verbose`: 启用详细日志输出
- `-reasoning`: 启用多步推理模式

## 故障排除

### 常见问题

1. **API密钥错误**
   - 检查 `.env` 文件中的 `OPENAI_API_KEY` 是否正确设置
   - 确保API密钥有足够的配额

2. **代码搜索功能不可用**
   - 确保系统已安装 `ripgrep` (rg)
   - 在Windows上，可以通过 `scoop install ripgrep` 安装

3. **文件权限问题**
   - 确保程序对工作目录有读写权限

4. **工具调用问题**
   - 确保工具参数格式正确
   - 检查文件路径和权限

### 调试模式

使用 `-verbose` 标志启用详细日志：
```bash
go run ./cmd/gocopilot -verbose
```

### 输出模式

Gocopilot使用批量输出模式，提供稳定的工具调用体验。这是经过优化的版本，移除了stream模式以解决工具调用问题。

### 架构说明

Gocopilot采用模块化架构设计：

- **Agent Core**: 管理对话流程、工具执行和推理链
- **Tool System**: 插件化工具注册和执行系统，支持并发执行
- **Memory Management**: 智能对话历史管理，支持系统消息
- **Configuration**: 统一配置管理，支持环境变量和默认值
- **Logging**: 结构化日志系统，支持多级别日志输出

系统通过接口抽象实现组件解耦，便于扩展和测试。

## 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开Pull Request

## 许可证

本项目基于MIT许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 致谢

- [OpenAI Go SDK](https://github.com/openai/openai-go) - OpenAI API客户端
- [jsonschema](https://github.com/invopop/jsonschema) - JSON Schema生成
- [ripgrep](https://github.com/BurntSushi/ripgrep) - 快速代码搜索工具

## 相关文档

- [CLAUDE.md](CLAUDE.md) - Claude Code开发指南
- [架构设计](internal/agent/types.go) - 核心接口定义
