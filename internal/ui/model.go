package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yanzzp/name-sprout/internal/providers"
)

const (
	defaultTimeout = 45 * time.Second
	initTimeout    = 8 * time.Second
)

type suggestionsMsg struct {
	names []string
	err   error
}

// Model 展示命名候选并允许复制。
type Model struct {
	providerName string
	provider     providers.Provider
	request      providers.Request
	modelName    string
	showDetails  bool

	spinner spinner.Model

	suggestions []string
	cursor      int
	loading     bool
	err         error
	status      string
}

// NewModel 创建用于展示命名结果的 TUI 模型。
func NewModel(providerName string, provider providers.Provider, req providers.Request) (*Model, error) {
	if err := warmupProvider(provider); err != nil {
		return nil, fmt.Errorf("提供方初始化失败：%w", err)
	}

	modelName := ""
	if reporter, ok := provider.(providers.ModelReporter); ok {
		modelName = reporter.ModelIdentifier()
	}

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	model := &Model{
		providerName: providerName,
		provider:     provider,
		request:      req,
		modelName:    modelName,
		spinner:      sp,
		loading:      true,
		status:       "正在等待模型响应...",
	}

	return model, nil
}

// Init 启动时立刻触发一次名称生成。
func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, generateCmd(m.provider, m.request))
}

// Update 处理 Bubble Tea 消息。
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case suggestionsMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.status = "生成失败，请检查配置或稍后重试。"
			return m, nil
		}
		m.err = nil
		m.status = fmt.Sprintf("生成完成，共 %d 个候选。使用 ↑↓ 选择，Enter/C 复制。", len(msg.names))
		m.suggestions = msg.names
		m.cursor = 0
		return m, nil
	}

	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "esc":
		return m, tea.Quit
	case "r", "R":
		if !m.loading {
			m.loading = true
			m.err = nil
			m.status = "正在等待模型响应..."
			m.suggestions = nil
			m.cursor = 0
			return m, tea.Batch(m.spinner.Tick, generateCmd(m.provider, m.request))
		}
	case "enter":
		if m.focusOnResults() {
			return m, m.copySelected()
		}
	case "c", "C":
		if m.focusOnResults() {
			return m, m.copySelected()
		}
	case "up":
		m.moveCursor(-1)
	case "down":
		m.moveCursor(1)
	case "i", "I":
		m.showDetails = !m.showDetails
	}

	return m, nil
}

func (m *Model) focusOnResults() bool {
	return !m.loading && len(m.suggestions) > 0
}

// View 渲染界面。
func (m *Model) View() string {
	var sections []string

	sections = append(sections, titleStyle.Render("🌱 Name Sprout"))

	toggle := "▶ 详情 (I)"
	if m.showDetails {
		toggle = "▼ 详情 (I)"
	}
	sections = append(sections, faintStyle.Render(toggle))

	if m.showDetails {
		providerLine := fmt.Sprintf(
			"提供方: %s  模型: %s",
			infoStyle.Render(m.providerName),
			infoStyle.Render(m.modelDisplay()),
		)
		meta := []string{
			providerLine,
			fmt.Sprintf("命名方式: %s", infoStyle.Render(string(m.request.Kind))),
		}
		if label := strings.TrimSpace(m.request.NamingStyleLabel); label != "" {
			meta = append(meta, fmt.Sprintf("命名格式: %s (%s)", infoStyle.Render(label), faintStyle.Render(string(m.request.NamingStyle))))
		} else if m.request.NamingStyle != "" {
			meta = append(meta, fmt.Sprintf("命名格式: %s", infoStyle.Render(string(m.request.NamingStyle))))
		}
		meta = append(meta, fmt.Sprintf("描述: %s", infoStyle.Render(m.request.Description)))
		sections = append(sections, strings.Join(meta, "\n"))
	}

	if m.loading {
		sections = append(sections, fmt.Sprintf("%s %s", m.spinner.View(), m.status))
	} else if m.err != nil {
		sections = append(sections, errStyle.Render(m.err.Error()))
		if m.status != "" {
			sections = append(sections, faintStyle.Render(m.status))
		}
	} else {
		if m.status != "" {
			sections = append(sections, faintStyle.Render(m.status))
		}
	}

	if m.focusOnResults() {
		var rows []string
		for i, name := range m.suggestions {
			prefix := "  "
			style := listItemStyle
			if i == m.cursor {
				prefix = "▶ "
				style = selectedItemStyle
			}
			rows = append(rows, prefix+style.Render(name))
		}
		sections = append(sections, strings.Join(rows, "\n"))
	} else if !m.loading && len(m.suggestions) == 0 {
		sections = append(sections, errStyle.Render("未获取到任何候选结果。"))
	}

	help := faintStyle.Render("操作：↑↓ 选择  Enter/C 复制  R 重新生成  I 切换详情  Q 退出")
	sections = append(sections, help)

	return lipgloss.NewStyle().Padding(1, 2).Render(strings.Join(sections, "\n\n"))
}

func (m *Model) modelDisplay() string {
	if strings.TrimSpace(m.modelName) == "" {
		return "未配置"
	}
	return m.modelName
}

func (m *Model) moveCursor(delta int) {
	if !m.focusOnResults() {
		return
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = len(m.suggestions) - 1
	} else if m.cursor >= len(m.suggestions) {
		m.cursor = 0
	}
}

func (m *Model) copySelected() tea.Cmd {
	if !m.focusOnResults() {
		return nil
	}
	name := m.suggestions[m.cursor]
	if err := clipboard.WriteAll(name); err != nil {
		m.err = fmt.Errorf("复制失败: %w", err)
		return nil
	}
	m.status = fmt.Sprintf("已复制：%s", name)
	return nil
}

func generateCmd(p providers.Provider, req providers.Request) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()
		names, err := p.GenerateNames(ctx, req)
		return suggestionsMsg{names: names, err: err}
	}
}

func warmupProvider(p providers.Provider) error {
	ctx, cancel := context.WithTimeout(context.Background(), initTimeout)
	defer cancel()
	if initializer, ok := p.(providers.Initializer); ok {
		if err := initializer.Warmup(ctx); err != nil {
			return err
		}
	}
	return nil
}

var (
	titleStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	infoStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("111")).Bold(true)
	faintStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)
	listItemStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("247"))
	selectedItemStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("63")).Bold(true)
)
