package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// AppConfig 描述与界面和业务相关的基础配置。
type AppConfig struct {
	DefaultProvider    string `yaml:"default_provider"`
	MaxSuggestions     int    `yaml:"max_suggestions"`
	DefaultNamingStyle string `yaml:"default_naming_style"`
	NamingPromptFile   string `yaml:"naming_prompt_file"`
}

// ProviderSettings 抽象出不同模型提供方的通用配置字段。
// Options 预留给未来扩展，例如自定义 base_url、组织ID等。
type ProviderSettings struct {
	Type        string            `yaml:"type"`
	APIKey      string            `yaml:"api_key"`
	Model       string            `yaml:"model"`
	Endpoint    string            `yaml:"endpoint"`
	Temperature *float32          `yaml:"temperature"`
	TopK        *float32          `yaml:"top_k"`
	Options     map[string]string `yaml:"options"`
}

// Config 代表整份应用配置。
type Config struct {
	App       AppConfig                   `yaml:"app"`
	Providers map[string]ProviderSettings `yaml:"providers"`
	source    string
}

// Load 从指定路径读取并解析配置文件。
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	cfg.source = path
	cfg.setDefaults()

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// setDefaults 填充缺省值，避免在业务逻辑中散落常量。
func (c *Config) setDefaults() {
	if c.App.MaxSuggestions <= 0 {
		c.App.MaxSuggestions = 5
	}
	if c.App.DefaultNamingStyle == "" {
		c.App.DefaultNamingStyle = "lower_camel"
	}
	if c.App.NamingPromptFile == "" {
		c.App.NamingPromptFile = "prompts/naming.yaml"
	}
	if c.Providers == nil {
		c.Providers = make(map[string]ProviderSettings)
	}
	for name, settings := range c.Providers {
		updated := false
		if settings.Type == "" {
			settings.Type = name
			updated = true
		}
		if settings.Options == nil {
			settings.Options = make(map[string]string)
			updated = true
		}
		if updated {
			c.Providers[name] = settings
		}
	}
}

// Validate 校验关键字段是否就绪。
func (c *Config) Validate() error {
	if c.App.DefaultProvider == "" {
		return errors.New("配置缺少 app.default_provider 字段")
	}

	settings, ok := c.Providers[c.App.DefaultProvider]
	if !ok {
		return fmt.Errorf("未找到默认提供方 %q 的配置", c.App.DefaultProvider)
	}

	if settings.Type == "" {
		return fmt.Errorf("提供方 %q 缺少 type 字段", c.App.DefaultProvider)
	}

	return nil
}

// DefaultProvider 返回默认提供方的配置。
func (c *Config) DefaultProvider() (string, ProviderSettings) {
	return c.App.DefaultProvider, c.Providers[c.App.DefaultProvider]
}

// Provider 获取指定名称的提供方配置。
func (c *Config) Provider(name string) (ProviderSettings, bool) {
	settings, ok := c.Providers[name]
	return settings, ok
}

// Source 返回配置文件来源，方便调试信息展示。
func (c *Config) Source() string {
	return c.source
}
