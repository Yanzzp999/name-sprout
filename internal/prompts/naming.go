package prompts

import (
	"fmt"
	"os"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"

	"github.com/yanzzp/name-sprout/internal/providers"
)

// NamingPromptDefinition 描述单个命名格式的提示词配置。
type NamingPromptDefinition struct {
	Label   string   `yaml:"label"`
	Prompt  string   `yaml:"prompt"`
	Aliases []string `yaml:"aliases"`
}

// KindPromptDefinition 描述命名类型的补充提示。
type KindPromptDefinition struct {
	Label        string                `yaml:"label"`
	Prompt       string                `yaml:"prompt"`
	DefaultStyle providers.NamingStyle `yaml:"default_style"`
}

type namingPromptFile struct {
	Styles map[string]NamingPromptDefinition `yaml:"styles"`
	Kinds  map[string]KindPromptDefinition   `yaml:"kinds"`
}

// NamingPrompts 管理命名格式与提示词的映射关系。
type NamingPrompts struct {
	definitions     map[providers.NamingStyle]NamingPromptDefinition
	aliases         map[string]providers.NamingStyle
	kindDefinitions map[providers.NameKind]KindPromptDefinition
}

// LoadNamingPrompts 从指定路径读取命名提示词配置。
func LoadNamingPrompts(path string) (*NamingPrompts, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取命名提示配置失败: %w", err)
	}

	var file namingPromptFile
	if err := yaml.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("解析命名提示配置失败: %w", err)
	}

	if len(file.Styles) == 0 {
		return nil, fmt.Errorf("命名提示配置为空")
	}

	lib := &NamingPrompts{
		definitions:     make(map[providers.NamingStyle]NamingPromptDefinition),
		aliases:         make(map[string]providers.NamingStyle),
		kindDefinitions: make(map[providers.NameKind]KindPromptDefinition),
	}

	for key, def := range file.Styles {
		style, err := providers.ParseNamingStyle(string(providers.NamingStyle(key)))
		if err != nil {
			return nil, fmt.Errorf("不支持的命名格式 %q: %w", key, err)
		}
		if strings.TrimSpace(def.Prompt) == "" {
			return nil, fmt.Errorf("命名格式 %q 的 prompt 不能为空", key)
		}
		lib.definitions[style] = def

		lib.addAlias(style, string(style))
		if def.Label != "" {
			lib.addAlias(style, def.Label)
		}
		for _, alias := range def.Aliases {
			lib.addAlias(style, alias)
		}
	}

	for key, def := range file.Kinds {
		kind, err := providers.ParseNameKind(key)
		if err != nil {
			return nil, fmt.Errorf("不支持的命名类型 %q: %w", key, err)
		}
		if strings.TrimSpace(def.Prompt) == "" {
			return nil, fmt.Errorf("命名类型 %q 的 prompt 不能为空", key)
		}
		if raw := strings.TrimSpace(string(def.DefaultStyle)); raw != "" {
			style, err := providers.ParseNamingStyle(raw)
			if err != nil {
				return nil, fmt.Errorf("命名类型 %q 的 default_style 无效: %w", key, err)
			}
			def.DefaultStyle = style
		}
		lib.kindDefinitions[kind] = def
	}

	return lib, nil
}

// Definition 返回指定命名格式的提示定义。
func (n *NamingPrompts) Definition(style providers.NamingStyle) (NamingPromptDefinition, bool) {
	def, ok := n.definitions[style]
	return def, ok
}

// KindDefinition 返回指定命名类型的提示定义。
func (n *NamingPrompts) KindDefinition(kind providers.NameKind) (KindPromptDefinition, bool) {
	def, ok := n.kindDefinitions[kind]
	return def, ok
}

// Lookup 根据别名或关键字查找命名格式定义。
func (n *NamingPrompts) Lookup(raw string) (providers.NamingStyle, NamingPromptDefinition, bool) {
	style, ok := n.aliases[normalizeAlias(raw)]
	if !ok {
		return "", NamingPromptDefinition{}, false
	}
	def, ok := n.definitions[style]
	if !ok {
		return "", NamingPromptDefinition{}, false
	}
	return style, def, true
}

func (n *NamingPrompts) addAlias(style providers.NamingStyle, alias string) {
	normalized := normalizeAlias(alias)
	if normalized == "" {
		return
	}
	n.aliases[normalized] = style
}

func normalizeAlias(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = strings.ToLower(raw)

	var b strings.Builder
	for _, r := range raw {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			b.WriteRune(r)
		}
	}
	return b.String()
}
