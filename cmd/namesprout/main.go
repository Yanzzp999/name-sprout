package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/yanzzp/name-sprout/internal/app"
	"github.com/yanzzp/name-sprout/internal/config"
	"github.com/yanzzp/name-sprout/internal/prompts"
	"github.com/yanzzp/name-sprout/internal/providers"
	"github.com/yanzzp/name-sprout/internal/ui"
)

func main() {
	var (
		cfgPath     = flag.String("config", "config.yaml", "配置文件路径")
		disableAlt  = flag.Bool("no-alt-screen", false, "禁用备用屏幕渲染")
		showVersion = flag.Bool("version", false, "打印版本信息")
		caseFlag    = flag.String("style", "", "指定命名格式（lowerCamelCase / PascalCase / snake_case / kebab-case）")
		funcFlag    = flag.Bool("f", false, "生成函数名称")
		varFlag     = flag.Bool("v", false, "生成变量名称")
		projectFlag = flag.Bool("p", false, "生成项目名称")
	)
	flag.Parse()

	if *showVersion {
		fmt.Println("Name Sprout TUI")
		return
	}

	modeCount := 0
	if *funcFlag {
		modeCount++
	}
	if *varFlag {
		modeCount++
	}
	if *projectFlag {
		modeCount++
	}
	if modeCount != 1 {
		fmt.Fprintln(os.Stderr, "请使用且仅使用 -f、-v 或 -p 指定命名类型。")
		os.Exit(1)
	}

	description := strings.TrimSpace(strings.Join(flag.Args(), " "))
	if description == "" {
		fmt.Fprintln(os.Stderr, "请在参数中提供命名描述，例如：namesprout -f \"为一个Go库取函数名\"")
		os.Exit(1)
	}

	configPath := resolveConfigPath(*cfgPath)

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败：%v\n", err)
		os.Exit(1)
	}

	promptPath := cfg.App.NamingPromptFile
	if !filepath.IsAbs(promptPath) {
		base := filepath.Dir(cfg.Source())
		promptPath = filepath.Join(base, promptPath)
	}

	namingPrompts, err := prompts.LoadNamingPrompts(promptPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载命名提示配置失败：%v\n", err)
		os.Exit(1)
	}

	appCtx, err := app.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化应用失败：%v\n", err)
		os.Exit(1)
	}

	providerName, providerSettings := cfg.DefaultProvider()
	provider, err := appCtx.Provider(providerName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取模型提供方失败：%v\n", err)
		os.Exit(1)
	}

	var kind providers.NameKind
	switch {
	case *funcFlag:
		kind = providers.NameKindFunction
	case *varFlag:
		kind = providers.NameKindVariable
	case *projectFlag:
		kind = providers.NameKindProject
	default:
		// 理论上不会触发，防御性处理。
		kind = providers.NameKindFunction
	}

	kindDefinition, _ := namingPrompts.KindDefinition(kind)

	var (
		namingStyle providers.NamingStyle
		definition  prompts.NamingPromptDefinition
		ok          bool
	)

	if rawStyle := strings.TrimSpace(*caseFlag); rawStyle != "" {
		if namingStyle, definition, ok = namingPrompts.Lookup(rawStyle); !ok {
			fmt.Fprintf(os.Stderr, "不支持的命名格式：%s\n", rawStyle)
			os.Exit(1)
		}
	} else if kindDefinition.DefaultStyle != "" {
		namingStyle = kindDefinition.DefaultStyle
		if definition, ok = namingPrompts.Definition(namingStyle); !ok {
			fmt.Fprintf(os.Stderr, "命名提示配置中缺少默认命名格式：%s\n", namingStyle)
			os.Exit(1)
		}
	} else {
		namingStyle, err = providers.ParseNamingStyle(cfg.App.DefaultNamingStyle)
		if err != nil {
			fmt.Fprintf(os.Stderr, "配置中的默认命名格式无效：%v\n", err)
			os.Exit(1)
		}
		definition, ok = namingPrompts.Definition(namingStyle)
		if !ok {
			fmt.Fprintf(os.Stderr, "命名提示配置中缺少默认命名格式：%s\n", namingStyle)
			os.Exit(1)
		}
	}
	if definition.Label == "" {
		definition.Label = string(namingStyle)
	}

	kindLabel := kindDefinition.Label
	if strings.TrimSpace(kindLabel) == "" {
		kindLabel = string(kind)
	}

	req := providers.Request{
		Description:       description,
		Kind:              kind,
		Count:             cfg.App.MaxSuggestions,
		KindLabel:         kindLabel,
		KindPrompt:        kindDefinition.Prompt,
		NamingStyle:       namingStyle,
		NamingStyleLabel:  definition.Label,
		NamingStylePrompt: definition.Prompt,
	}

	model, err := ui.NewModel(providerName, provider, providerSettings, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建 UI 模型失败：%v\n", err)
		os.Exit(1)
	}

	options := []tea.ProgramOption{}
	if !*disableAlt {
		options = append(options, tea.WithAltScreen())
	}

	if err := tea.NewProgram(model, options...).Start(); err != nil {
		fmt.Fprintf(os.Stderr, "运行 TUI 失败：%v\n", err)
		os.Exit(1)
	}
}

func resolveConfigPath(path string) string {
	if path == "" {
		path = "config.yaml"
	}

	if filepath.IsAbs(path) {
		return path
	}

	if _, err := os.Stat(path); err == nil {
		return path
	}

	exePath, err := os.Executable()
	if err != nil {
		return path
	}

	exeDir := filepath.Dir(exePath)
	candidate := filepath.Join(exeDir, path)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}

	return path
}
