package providers

import (
	"context"
	"fmt"
)

// NameKind 表示希望生成的命名类型。
type NameKind string

const (
	NameKindFunction NameKind = "function"
	NameKindVariable NameKind = "variable"
	NameKindProject  NameKind = "project"
)

// AllNameKinds 列出当前支持的命名类型，供 UI 和校验使用。
var AllNameKinds = []NameKind{
	NameKindFunction,
	NameKindVariable,
	NameKindProject,
}

// NamingStyle 表示生成名称时的命名格式。
type NamingStyle string

const (
	NamingStyleLowerCamel NamingStyle = "lower_camel"
	NamingStylePascal     NamingStyle = "pascal_case"
	NamingStyleSnake      NamingStyle = "snake_case"
	NamingStyleKebab      NamingStyle = "kebab_case"
)

// AllNamingStyles 列出当前支持的命名格式，供 UI 和校验使用。
var AllNamingStyles = []NamingStyle{
	NamingStyleLowerCamel,
	NamingStylePascal,
	NamingStyleSnake,
	NamingStyleKebab,
}

// ParseNameKind 将用户输入的字面值转换为枚举。
func ParseNameKind(raw string) (NameKind, error) {
	kind := NameKind(raw)
	for _, candidate := range AllNameKinds {
		if candidate == kind {
			return kind, nil
		}
	}
	return "", fmt.Errorf("不支持的命名类型: %s", raw)
}

// ParseNamingStyle 将用户输入的字面值转换为枚举形式。
func ParseNamingStyle(raw string) (NamingStyle, error) {
	style := NamingStyle(raw)
	for _, candidate := range AllNamingStyles {
		if candidate == style {
			return style, nil
		}
	}
	return "", fmt.Errorf("不支持的命名格式: %s", raw)
}

// Request 聚合用于请求大模型生成名称的上下文信息。
type Request struct {
	Description       string
	Kind              NameKind
	Count             int
	Language          string
	Tone              string
	NamingStyle       NamingStyle
	NamingStyleLabel  string
	NamingStylePrompt string
}

// Provider 定义不同模型提供方需要实现的接口。
type Provider interface {
	Name() string
	GenerateNames(ctx context.Context, req Request) ([]string, error)
}

// ModelReporter 可选接口，允许 Provider 暴露底层模型标识。
type ModelReporter interface {
	ModelIdentifier() string
}

// Initializer 用于声明 Provider 支持启动前的健康检查。
type Initializer interface {
	Warmup(ctx context.Context) error
}
