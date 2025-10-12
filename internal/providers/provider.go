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

// Request 聚合用于请求大模型生成名称的上下文信息。
type Request struct {
	Description string
	Kind        NameKind
	Count       int
	Language    string
	Tone        string
}

// Provider 定义不同模型提供方需要实现的接口。
type Provider interface {
	Name() string
	GenerateNames(ctx context.Context, req Request) ([]string, error)
}

// Initializer 用于声明 Provider 支持启动前的健康检查。
type Initializer interface {
	Warmup(ctx context.Context) error
}
