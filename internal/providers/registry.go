package providers

import (
	"fmt"
	"sort"
	"sync"

	"github.com/yanzzp/name-sprout/internal/config"
)

// Factory 负责基于配置构造 Provider。
type Factory func(name string, settings config.ProviderSettings) (Provider, error)

var (
	mu        sync.RWMutex
	registry  = make(map[string]Factory)
	typeNames = make(map[string]string)
)

// Register 将新的 Provider 类型注册进全局表。
func Register(providerType string, factory Factory, displayName string) {
	mu.Lock()
	defer mu.Unlock()
	registry[providerType] = factory
	typeNames[providerType] = displayName
}

// New 通过配置实例化一个 Provider。
func New(name string, settings config.ProviderSettings) (Provider, error) {
	mu.RLock()
	factory, ok := registry[settings.Type]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("未注册提供方类型: %s", settings.Type)
	}
	return factory(name, settings)
}

// Types 返回当前已注册的提供方类型列表，方便 UI 展示。
func Types() []string {
	mu.RLock()
	defer mu.RUnlock()

	types := make([]string, 0, len(registry))
	for key := range registry {
		types = append(types, key)
	}
	sort.Strings(types)
	return types
}

// DisplayName 返回友好的类型名称。
func DisplayName(providerType string) string {
	mu.RLock()
	defer mu.RUnlock()
	if name, ok := typeNames[providerType]; ok {
		return name
	}
	return providerType
}
