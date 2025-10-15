package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"google.golang.org/genai"

	"github.com/yanzzp/name-sprout/internal/config"
	"github.com/yanzzp/name-sprout/internal/providers"
)

const (
	providerType               = "gemini"
	defaultModel               = "models/gemini-1.5-pro"
	defaultTemperature float32 = 0.7
)

type geminiProvider struct {
	name        string
	model       string
	client      *genai.Client
	once        sync.Once
	init        error
	apiKey      string
	temperature float32
	topK        *float32
}

// Register Gemini provider when package initializes.
func init() {
	providers.Register(providerType, newGeminiProvider, "Google Gemini")
}

func newGeminiProvider(name string, settings config.ProviderSettings) (providers.Provider, error) {
	if settings.APIKey == "" {
		return nil, errors.New("Gemini 配置缺少 api_key")
	}
	model := settings.Model
	if model == "" {
		model = defaultModel
	}

	temperature := defaultTemperature
	if settings.Temperature != nil {
		temperature = *settings.Temperature
	}

	var topK *float32
	if settings.TopK != nil {
		if *settings.TopK <= 0 {
			return nil, errors.New("Gemini 配置的 top_k 必须大于 0")
		}
		value := *settings.TopK
		topK = &value
	}

	return &geminiProvider{
		name:        name,
		model:       model,
		apiKey:      settings.APIKey,
		temperature: temperature,
		topK:        topK,
	}, nil
}

func (p *geminiProvider) Name() string {
	return p.name
}

func (p *geminiProvider) GenerateNames(ctx context.Context, req providers.Request) ([]string, error) {
	if err := p.ensureClient(ctx); err != nil {
		return nil, err
	}

	count := req.Count
	if count <= 0 {
		count = 5
	}
	if count > 12 {
		count = 12
	}

	prompt := buildPrompt(req, count)

	config := &genai.GenerateContentConfig{
		Temperature:      genai.Ptr[float32](p.temperature),
		ResponseMIMEType: "application/json",
	}
	if p.topK != nil {
		config.TopK = p.topK
	}

	resp, err := p.client.Models.GenerateContent(ctx, p.model, genai.Text(prompt), config)
	if err != nil {
		return nil, fmt.Errorf("调用 Gemini API 失败: %w", err)
	}

	if resp.PromptFeedback != nil && resp.PromptFeedback.BlockReason != "" {
		return nil, fmt.Errorf("Gemini 拒绝了请求: %s", resp.PromptFeedback.BlockReason)
	}

	text := collectText(resp)
	if text == "" {
		return nil, errors.New("Gemini 返回结果为空")
	}

	names, err := parseNamesFromJSON(text)
	if err != nil {
		// Fallback: 尝试基于换行分割
		return fallbackNames(text, count), nil
	}

	return names, nil
}

func (p *geminiProvider) ensureClient(ctx context.Context) error {
	p.once.Do(func() {
		client, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  p.apiKey,
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			p.init = fmt.Errorf("创建 Gemini 客户端失败: %w", err)
			return
		}
		p.client = client
	})
	return p.init
}

func (p *geminiProvider) Warmup(ctx context.Context) error {
	return p.ensureClient(ctx)
}

func (p *geminiProvider) ModelIdentifier() string {
	return p.model
}

func buildPrompt(req providers.Request, count int) string {
	var b strings.Builder
	b.WriteString("你是一名经验丰富的命名顾问，需要基于用户提供的背景信息生成高质量的名称。\n")
	b.WriteString("请遵循以下规则：\n")
	b.WriteString("- 输出 JSON 对象，结构为 {\"names\": [\"名称1\", \"名称2\", ...]}。\n")
	b.WriteString("- 名称需满足命名类型要求，同时保持易读易记。\n")
	b.WriteString("- 避免输出额外解释或 Markdown。\n\n")

	b.WriteString("命名任务信息：\n")
	b.WriteString(fmt.Sprintf("- 命名类型：%s\n", req.Kind))
	b.WriteString(fmt.Sprintf("- 名称数量：%d\n", count))
	if req.NamingStyleLabel != "" {
		b.WriteString(fmt.Sprintf("- 命名格式：%s\n", req.NamingStyleLabel))
	} else if req.NamingStyle != "" {
		b.WriteString(fmt.Sprintf("- 命名格式：%s\n", req.NamingStyle))
	}
	if req.Language != "" {
		b.WriteString(fmt.Sprintf("- 输出语言：%s\n", req.Language))
	}
	if req.Tone != "" {
		b.WriteString(fmt.Sprintf("- 风格倾向：%s\n", req.Tone))
	}
	if req.Description != "" {
		b.WriteString(fmt.Sprintf("- 详细描述：%s\n", req.Description))
	}
	if prompt := strings.TrimSpace(req.NamingStylePrompt); prompt != "" {
		b.WriteString("\n命名格式要求：\n")
		b.WriteString(prompt)
		b.WriteString("\n")
	}
	b.WriteString("\n请直接返回 JSON，对名称进行去重，并确保每个名称不超过 32 个字符。")
	return b.String()
}

func collectText(resp *genai.GenerateContentResponse) string {
	if resp == nil {
		return ""
	}

	var sb strings.Builder
	for _, cand := range resp.Candidates {
		if cand == nil || cand.Content == nil {
			continue
		}
		for _, part := range cand.Content.Parts {
			if part == nil || part.Thought {
				continue
			}
			if part.Text != "" {
				if sb.Len() > 0 {
					sb.WriteRune('\n')
				}
				sb.WriteString(part.Text)
			}
		}
	}
	return strings.TrimSpace(sb.String())
}

type namesEnvelope struct {
	Names []string `json:"names"`
}

func parseNamesFromJSON(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("空响应")
	}

	var (
		envelope namesEnvelope
		err      error
	)

	// 允许模型返回数组形式。
	if strings.HasPrefix(raw, "[") {
		err = json.Unmarshal([]byte(raw), &envelope.Names)
	} else {
		err = json.Unmarshal([]byte(raw), &envelope)
	}
	if err != nil {
		return nil, err
	}

	dedup := make(map[string]struct{})
	result := make([]string, 0, len(envelope.Names))
	for _, name := range envelope.Names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := dedup[name]; ok {
			continue
		}
		dedup[name] = struct{}{}
		result = append(result, name)
	}

	if len(result) == 0 {
		return nil, errors.New("解析后没有有效名称")
	}

	return result, nil
}

func fallbackNames(raw string, count int) []string {
	lines := strings.Split(raw, "\n")
	dedup := make(map[string]struct{})
	result := make([]string, 0, count)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimLeft(line, "-0123456789. ")
		if line == "" || len(result) >= count {
			continue
		}
		if _, ok := dedup[line]; ok {
			continue
		}
		dedup[line] = struct{}{}
		result = append(result, line)
	}

	return result
}

// Close 释放底层资源，方便未来在 UI 退出时调用。
func (p *geminiProvider) Close() error { return nil }
