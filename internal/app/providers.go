package app

// 注册默认 Provider 实现，在包加载时触发 init。
import (
	_ "github.com/yanzzp/name-sprout/internal/providers/gemini"
)
