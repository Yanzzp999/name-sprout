# Name Sprout

Name Sprout 是一款基于 [Bubble Tea](https://github.com/charmbracelet/bubbletea) 的终端命名助手，当前默认集成 Google Gemini API。通过命令行参数一次性提交描述与命名类型（函数 / 变量 / 项目），程序生成候选列表后，会在 TUI 中展示结果并允许用户快速复制。整体架构预留了扩展接口，后续可无缝支持 ChatGPT、Cursor 等不同大模型提供方。

## 快速开始

1. **准备环境**
   - Go 1.24 及以上版本（`go env GOTOOLCHAIN` 会自动切换到 go1.24.8）。
   - Gemini API Key（可在 [ai.google.dev](https://ai.google.dev/) 获取）。

2. **填写配置**
   - 编辑仓库根目录的 `config.yaml`，将 `providers.gemini.api_key` 替换为真实的 API Key。
   - 如需调整生成数量，可修改 `app.max_suggestions`。

3. **构建 & 运行**
   ```bash
   go build -o namesprout ./cmd/namesprout
   ./namesprout -f "为一个Go开源库取函数名"
   ./namesprout -v "用于存储数据库连接的变量名"
   ./namesprout -p "描述一个AI命名助手的项目"
   ```
   可选参数：
   - `--config` 指定自定义配置路径。
   - `--no-alt-screen` 禁用备用屏幕（方便与其它终端工具搭配使用）。
   - `-f / -v / -p` 分别代表函数、变量、项目命名，三者必须且只能选择一个。

4. **TUI 操作**
   - `↑ ↓`：在候选列表中移动光标。
   - `Enter / C`：复制当前选中的名称。
   - `R`：重新向模型请求一组候选。
   - `Ctrl+C / Q / Esc`：退出程序。

## 配置结构

```yaml
app:
  default_provider: gemini
  max_suggestions: 5
providers:
  gemini:
    type: gemini
    api_key: "YOUR_GEMINI_API_KEY"
    model: "models/gemini-1.5-pro"
```

- `app.default_provider`：启动时使用的默认提供方名称。
- `app.max_suggestions`：单次生成的目标数量。
- `providers`：以“名称”为 key，value 中 `type` 表示实现类型，其余字段作为特定 Provider 的参数。

## 项目结构

```
cmd/namesprout      # 程序入口，负责解析配置与启动 Bubble Tea
internal/app        # 应用上下文，统一管理配置与 Provider 实例
internal/config     # YAML 配置解析与校验
internal/providers  # Provider 接口、注册中心，以及具体的 Gemini 实现
internal/ui         # 终端界面模型，包含交互逻辑与样式
config.yaml         # 默认配置文件
```

`internal/providers/provider.go` 定义了统一的接口：

```go
type Provider interface {
    Name() string
    GenerateNames(ctx context.Context, req Request) ([]string, error)
}
```

任何新的模型提供方只需实现上述接口，并在 `internal/providers/registry.go` 中注册，即可通过配置启用。主程序会读取默认提供方并调用其 `Warmup` 方法（若实现），再进入候选展示界面，便于扩展到 ChatGPT、Cursor 或自建模型。

## 后续扩展建议

- **多提供方参数面板**：在 TUI 中为不同模型提供方暴露额外参数（温度、提示词模板等）。
- **命名类型自定义**：允许用户通过配置文件添加新的命名类型，并在界面中动态展示。
- **历史记录与导出**：记录最近一次生成结果，支持导出 JSON / Markdown。
- **多语言输出**：结合配置或快捷键快速切换输出语言与风格。
- **单元测试覆盖**：为配置解析、Provider 调用与提示模板添加针对性测试，保证扩展时的稳定性。

欢迎按照需求逐步迭代，Name Sprout 可以成为一个灵活的终端命名助手平台。
