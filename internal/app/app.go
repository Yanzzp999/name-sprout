package app

import (
	"fmt"
	"sort"
	"sync"

	"github.com/yanzzp/name-sprout/internal/config"
	"github.com/yanzzp/name-sprout/internal/providers"
)

// App 负责管理配置和 Provider 实例的生命周期。
type App struct {
	cfg        *config.Config
	mu         sync.Mutex
	providers  map[string]providers.Provider
	providerID []string
}

// New 构造应用上下文。
func New(cfg *config.Config) (*App, error) {
	if cfg == nil {
		return nil, fmt.Errorf("cfg 不能为空")
	}
	names := make([]string, 0, len(cfg.Providers))
	for name := range cfg.Providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return &App{
		cfg:        cfg,
		providers:  make(map[string]providers.Provider),
		providerID: names,
	}, nil
}

// Config 返回底层配置。
func (a *App) Config() *config.Config {
	return a.cfg
}

// ProviderNames 返回全部可用的 Provider 名称。
func (a *App) ProviderNames() []string {
	return append([]string(nil), a.providerID...)
}

// DefaultProviderName 返回默认 Provider 名称。
func (a *App) DefaultProviderName() string {
	return a.cfg.App.DefaultProvider
}

// Provider 获取或创建指定 Provider 实例。
func (a *App) Provider(name string) (providers.Provider, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if p, ok := a.providers[name]; ok {
		return p, nil
	}

	settings, ok := a.cfg.Provider(name)
	if !ok {
		return nil, fmt.Errorf("未知的 Provider: %s", name)
	}

	instance, err := providers.New(name, settings)
	if err != nil {
		return nil, err
	}

	a.providers[name] = instance
	return instance, nil
}
