package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"yescode-cli/internal/api"
)

type focusArea int

const (
	focusProviders focusArea = iota
	focusAlternatives
)

type tabIndex int

const (
	tabProfile tabIndex = iota
	tabProviders
	tabBalancePreference
)

// Model wires Bubble Tea with the YesCode API client.
type Model struct {
	client *api.Client

	profile             *api.Profile
	providers           []api.ProviderBucket
	providerIdx         int
	altIdx              int
	balancePreferenceIdx int
	focus               focusArea
	currentTab          tabIndex
	ready               bool
	status              string
	err                 error
	width               int
	height              int
	providerData        map[int]*providerState
	preferenceSwitching bool
	spinner             spinner.Model
	help                help.Model
	keys                keyMap
	profileViewport     viewport.Model
	providersLoaded     bool
	loadingProviders    bool
}

type providerState struct {
	alternatives        []api.AlternativeOption
	selection           *api.ProviderSelection
	alternativesLoaded  bool
	selectionLoaded     bool
	loadingAlternatives bool
	loadingSelection    bool
	switching           bool
	lastError           error
}

// keyMap defines key bindings for the app
type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Left    key.Binding
	Right   key.Binding
	Tab     key.Binding
	Enter   key.Binding
	Refresh key.Binding
	Tab1    key.Binding
	Tab2    key.Binding
	Tab3    key.Binding
	Quit    key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab, k.Up, k.Down, k.Left, k.Right, k.Enter, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab, k.Tab1, k.Tab2, k.Tab3},
		{k.Up, k.Down, k.Left, k.Right},
		{k.Enter, k.Refresh, k.Quit},
	}
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "上移"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "下移"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "切换焦点"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "切换焦点"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "切换标签页"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "选择"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "刷新"),
	),
	Tab1: key.NewBinding(
		key.WithKeys("1"),
		key.WithHelp("1", "个人资料"),
	),
	Tab2: key.NewBinding(
		key.WithKeys("2"),
		key.WithHelp("2", "提供商"),
	),
	Tab3: key.NewBinding(
		key.WithKeys("3"),
		key.WithHelp("3", "余额使用偏好"),
	),
	Quit: key.NewBinding(
		key.WithKeys("esc", "ctrl+c"),
		key.WithHelp("esc", "退出"),
	),
}

type profileLoadedMsg struct {
	profile *api.Profile
}

type providersLoadedMsg struct {
	response *api.ProvidersResponse
}

type alternativesLoadedMsg struct {
	providerID   int
	alternatives []api.AlternativeOption
}

type selectionLoadedMsg struct {
	providerID int
	selection  *api.ProviderSelection
}

type switchCompletedMsg struct {
	providerID int
	selection  *api.ProviderSelection
}

type preferenceUpdatedMsg struct {
	preference string
}

type preferenceFailedMsg struct {
	err error
}

type providerLoadFailedMsg struct {
	providerID int
	target     string
	err        error
}

type errMsg struct {
	err error
}

type clearStatusMsg struct{}


// NewModel constructs the root Bubble Tea model.
func NewModel(client *api.Client) *Model {
	// 创建 spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(primaryColor)

	// 创建 help
	h := help.New()
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(primaryColor)
	h.Styles.ShortDesc = helpStyle
	h.Styles.FullKey = lipgloss.NewStyle().Foreground(primaryColor)
	h.Styles.FullDesc = helpStyle

	// 创建 viewport
	vp := viewport.New(0, 20)

	return &Model{
		client:           client,
		focus:            focusProviders,
		providerData:     make(map[int]*providerState),
		spinner:          s,
		help:             h,
		keys:             keys,
		profileViewport:  vp,
		ready:            true,
		status:           "正在加载个人资料...",
	}
}

// Init triggers the first batch of API calls.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		loadProfileCmd(m.client),
		m.spinner.Tick,
	)
}

// Update handles Bubble Tea messages.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
	case tea.KeyMsg:
		cmd := m.handleKey(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case profileLoadedMsg:
		m.profile = msg.profile
		m.status = ""
	case providersLoadedMsg:
		m.providers = msg.response.Providers
		m.providersLoaded = true
		m.loadingProviders = false
		if m.providerIdx >= len(m.providers) {
			m.providerIdx = 0
		}
		// 立即清除加载状态消息
		if strings.Contains(m.status, "加载提供商列表中") {
			m.status = ""
		}
		if len(m.providers) > 0 {
			cmds = append(cmds, m.queueProviderDetailLoad(m.currentProviderID()))
		}
	case alternativesLoadedMsg:
		state := m.ensureProviderState(msg.providerID)
		state.alternatives = msg.alternatives
		state.alternativesLoaded = true
		state.loadingAlternatives = false
		state.lastError = nil
		m.syncAltIdx(msg.providerID)
		// 检查是否所有加载都完成，立即清除加载状态消息
		if state.alternativesLoaded && state.selectionLoaded && strings.Contains(m.status, "加载提供商") {
			m.status = ""
		}
	case selectionLoadedMsg:
		state := m.ensureProviderState(msg.providerID)
		state.selection = msg.selection
		state.selectionLoaded = true
		state.loadingSelection = false
		state.lastError = nil
		m.syncAltIdx(msg.providerID)
		// 检查是否所有加载都完成，立即清除加载状态消息
		if state.alternativesLoaded && state.selectionLoaded && strings.Contains(m.status, "加载提供商") {
			m.status = ""
		}
	case switchCompletedMsg:
		state := m.ensureProviderState(msg.providerID)
		state.selection = msg.selection
		state.selectionLoaded = true
		state.switching = false
		state.lastError = nil
		m.syncAltIdx(msg.providerID)
		m.status = fmt.Sprintf("已切换到 %s", msg.selection.SelectedAlternative.DisplayName)
		cmds = append(cmds, clearStatusAfterDelay(2))
	case preferenceUpdatedMsg:
		if m.profile != nil {
			m.profile.BalancePreference = msg.preference
		}
		m.preferenceSwitching = false
		m.syncBalancePreferenceIdx()
		m.status = fmt.Sprintf("余额偏好已切换为 %s", describePreference(msg.preference))
		cmds = append(cmds, clearStatusAfterDelay(2))
	case preferenceFailedMsg:
		m.preferenceSwitching = false
		m.err = msg.err
		m.status = fmt.Sprintf("余额偏好切换失败: %v", msg.err)
		cmds = append(cmds, clearStatusAfterDelay(3))
	case providerLoadFailedMsg:
		state := m.ensureProviderState(msg.providerID)
		switch msg.target {
		case "alternatives":
			state.loadingAlternatives = false
		case "selection":
			state.loadingSelection = false
		case "switch":
			state.switching = false
		}
		state.lastError = msg.err
		m.err = msg.err
		m.status = fmt.Sprintf("提供商 %d: %v", msg.providerID, msg.err)
		cmds = append(cmds, clearStatusAfterDelay(3))
	case errMsg:
		m.err = msg.err
		m.status = msg.err.Error()
		// 如果是加载提供商失败，重置加载状态
		if m.loadingProviders {
			m.loadingProviders = false
		}
		cmds = append(cmds, clearStatusAfterDelay(3))
	case clearStatusMsg:
		m.status = ""
		m.err = nil
	}

	// 更新 spinner
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the TUI.
func (m *Model) View() string {
	var sections []string
	sections = append(sections, m.help.View(m.keys))

	// 添加 tab header
	sections = append(sections, m.renderTabHeader())

	// 根据当前 tab 渲染不同内容
	if m.currentTab == tabProfile {
		sections = append(sections, m.renderProfileTab())
	} else if m.currentTab == tabProviders {
		sections = append(sections, m.renderPanels())
	} else if m.currentTab == tabBalancePreference {
		sections = append(sections, m.renderBalancePreferenceTab())
	}

	// 始终渲染状态栏区域，保持视图高度一致
	statusText := ""
	if m.status != "" {
		statusText = m.status
		// 如果状态消息表示正在进行中，添加 spinner
		if strings.Contains(statusText, "中...") || strings.Contains(statusText, "加载") {
			statusText = fmt.Sprintf("%s %s", statusText, m.spinner.View())
		}
	}
	sections = append(sections, statusStyle.Render(statusText))

	return strings.Join(sections, "\n\n")
}


func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyCtrlC:
		return tea.Quit
	}

	switch msg.String() {
	case "esc":
		return tea.Quit
	case "1":
		m.currentTab = tabProfile
	case "2":
		m.currentTab = tabProviders
		m.focus = focusProviders
		return m.ensureProvidersLoaded()
	case "3":
		m.currentTab = tabBalancePreference
		m.syncBalancePreferenceIdx()
	case "tab":
		// Tab 键切换到下一个 tab
		m.currentTab = (m.currentTab + 1) % 3
		if m.currentTab == tabProviders {
			m.focus = focusProviders
			return m.ensureProvidersLoaded()
		} else if m.currentTab == tabBalancePreference {
			m.syncBalancePreferenceIdx()
		}
	case "left":
		if m.currentTab == tabProviders {
			m.focus = focusProviders
		}
	case "right":
		if m.currentTab == tabProviders {
			m.focus = focusAlternatives
		}
	case "r":
		if m.currentTab == tabProfile {
			return m.refreshProfile()
		} else if m.currentTab == tabProviders {
			return m.refreshCurrentProvider()
		}
	case "enter":
		if m.currentTab == tabProviders && m.focus == focusAlternatives {
			return m.switchSelection()
		} else if m.currentTab == tabBalancePreference {
			return m.toggleBalancePreference()
		}
	case "up", "k":
		if m.currentTab == tabProfile {
			m.profileViewport.LineUp(1)
			return nil
		} else if m.currentTab == tabBalancePreference {
			// 余额偏好tab：在两个选项之间移动
			if m.balancePreferenceIdx > 0 {
				m.balancePreferenceIdx--
			}
			return nil
		}
		return m.moveSelection(-1)
	case "down", "j":
		if m.currentTab == tabProfile {
			m.profileViewport.LineDown(1)
			return nil
		} else if m.currentTab == tabBalancePreference {
			// 余额偏好tab：在两个选项之间移动
			if m.balancePreferenceIdx < 1 {
				m.balancePreferenceIdx++
			}
			return nil
		}
		return m.moveSelection(1)
	}
	return nil
}

func (m *Model) ensureProvidersLoaded() tea.Cmd {
	// 如果已经加载或正在加载，不重复请求
	if m.providersLoaded || m.loadingProviders {
		return nil
	}
	m.loadingProviders = true
	m.status = "加载提供商列表中..."
	return loadProvidersCmd(m.client)
}

func (m *Model) moveSelection(delta int) tea.Cmd {
	if len(m.providers) == 0 {
		return nil
	}

	if m.focus == focusProviders {
		m.providerIdx = clampIndex(m.providerIdx+delta, len(m.providers))
		m.syncAltIdx(m.currentProviderID())
		return m.queueProviderDetailLoad(m.currentProviderID())
	} else {
		state := m.ensureProviderState(m.currentProviderID())
		if len(state.alternatives) == 0 {
			return nil
		}
		m.altIdx = clampIndex(m.altIdx+delta, len(state.alternatives))
	}
	return nil
}


func (m *Model) refreshProfile() tea.Cmd {
	m.profile = nil
	m.status = "正在刷新个人资料..."
	return loadProfileCmd(m.client)
}

func (m *Model) refreshCurrentProvider() tea.Cmd {
	if len(m.providers) == 0 {
		return nil
	}
	state := m.ensureProviderState(m.currentProviderID())
	state.alternativesLoaded = false
	state.loadingAlternatives = false
	state.selectionLoaded = false
	state.loadingSelection = false
	return m.queueProviderDetailLoad(m.currentProviderID())
}

func (m *Model) switchSelection() tea.Cmd {
	if len(m.providers) == 0 {
		return nil
	}
	state := m.ensureProviderState(m.currentProviderID())
	if state.switching || state.loadingAlternatives || len(state.alternatives) == 0 {
		return nil
	}
	if m.altIdx >= len(state.alternatives) {
		return nil
	}
	target := state.alternatives[m.altIdx].Alternative
	if state.selection != nil && state.selection.SelectedAlternativeID == target.ID {
		m.status = fmt.Sprintf("已在使用 %s", target.DisplayName)
		return nil
	}

	state.switching = true
	m.status = fmt.Sprintf("切换到 %s 中...", target.DisplayName)
	return switchProviderCmd(m.client, m.currentProviderID(), target.ID)
}


func (m *Model) toggleBalancePreference() tea.Cmd {
	if m.profile == nil || m.preferenceSwitching {
		return nil
	}

	// 根据选中的索引确定目标偏好
	var target string
	if m.balancePreferenceIdx == 0 {
		target = "subscription_first"
	} else {
		target = "payg_only"
	}

	// 如果已经是当前偏好，不需要切换
	if target == m.profile.BalancePreference {
		return nil
	}

	m.preferenceSwitching = true
	m.status = fmt.Sprintf("切换余额偏好到 %s...", describePreference(target))
	return updatePreferenceCmd(m.client, target)
}

func (m *Model) syncBalancePreferenceIdx() {
	if m.profile == nil {
		m.balancePreferenceIdx = 0
		return
	}

	// 根据当前的 BalancePreference 设置索引
	if m.profile.BalancePreference == "payg_only" {
		m.balancePreferenceIdx = 1
	} else {
		m.balancePreferenceIdx = 0
	}
}

func (m *Model) queueProviderDetailLoad(providerID int) tea.Cmd {
	if providerID == 0 {
		return nil
	}
	state := m.ensureProviderState(providerID)
	var cmds []tea.Cmd
	var loading bool
	if !state.alternativesLoaded && !state.loadingAlternatives {
		state.loadingAlternatives = true
		cmds = append(cmds, loadAlternativesCmd(m.client, providerID))
		loading = true
	}
	if !state.selectionLoaded && !state.loadingSelection {
		state.loadingSelection = true
		cmds = append(cmds, loadSelectionCmd(m.client, providerID))
		loading = true
	}
	if loading {
		m.status = fmt.Sprintf("加载提供商 %d 详情中...", providerID)
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (m *Model) syncAltIdx(providerID int) {
	if providerID == 0 || providerID != m.currentProviderID() {
		return
	}
	state := m.ensureProviderState(providerID)
	if state.selection != nil && len(state.alternatives) > 0 {
		if idx := m.findAlternativeIndex(state.alternatives, state.selection.SelectedAlternativeID); idx >= 0 {
			m.altIdx = idx
			return
		}
	}
	if len(state.alternatives) == 0 {
		m.altIdx = 0
		return
	}
	m.altIdx = clampIndex(m.altIdx, len(state.alternatives))
}

func (m *Model) findAlternativeIndex(alts []api.AlternativeOption, id int) int {
	for i, alt := range alts {
		if alt.Alternative.ID == id {
			return i
		}
	}
	return -1
}

func (m *Model) currentProviderID() int {
	if len(m.providers) == 0 {
		return 0
	}
	return m.providers[clampIndex(m.providerIdx, len(m.providers))].Provider.ID
}

func (m *Model) ensureProviderState(providerID int) *providerState {
	if providerID == 0 {
		return &providerState{}
	}
	state, ok := m.providerData[providerID]
	if !ok {
		state = &providerState{}
		m.providerData[providerID] = state
	}
	return state
}

func clampIndex(idx, length int) int {
	if idx < 0 {
		return 0
	}
	if idx >= length {
		return length - 1
	}
	return idx
}

// contentHeight 返回内容区域的固定高度
func (m *Model) contentHeight() int {
	return 20
}

func (m *Model) renderPanels() string {
	left := m.renderProvidersPanel()
	right := m.renderAlternativesPanel()

	// 水平拼接左右两个面板
	panels := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	return panels
}

func (m *Model) renderProvidersPanel() string {
	var lines []string

	if m.loadingProviders {
		lines = append(lines, fmt.Sprintf("加载提供商列表中... %s", m.spinner.View()))
	} else if len(m.providers) == 0 {
		lines = append(lines, "没有可用提供商")
	} else {
		for i, bucket := range m.providers {
			prefix := "  "
			if i == m.providerIdx {
				prefix = "→ "
			}
			lines = append(lines,
				fmt.Sprintf("%s%s%s%s",
					prefix,
					translateProviderDisplayName(bucket.Provider.DisplayName),
					formatSourceSuffix(bucket.Source),
					formatTypeSuffix(bucket.Provider.Type),
				),
			)
		}
	}

	content := strings.Join(lines, "\n")

	style := panelStyle.Copy()
	if m.focus == focusProviders {
		style = style.Copy().BorderStyle(activeBorder).BorderForeground(primaryColor)
	}
	return style.Width(m.panelWidth()).Height(10).Render(content)
}

func (m *Model) renderAlternativesPanel() string {
	var lines []string

	if len(m.providers) == 0 {
		lines = append(lines, "尚未选择提供商")
	} else {
		state := m.ensureProviderState(m.currentProviderID())

		switch {
		case state.loadingAlternatives:
			lines = append(lines, fmt.Sprintf("替代方案加载中... %s", m.spinner.View()))
		case state.lastError != nil:
			lines = append(lines, fmt.Sprintf("加载失败: %v (按 r 重试)", state.lastError))
		case len(state.alternatives) == 0:
			lines = append(lines, "没有可切换的替代方案")
		default:
			for i, alt := range state.alternatives {
				prefix := "  "
				if i == m.altIdx {
					prefix = "→ "
				}

				// 检查是否为当前选中项
				isCurrentSelection := state.selection != nil && state.selection.SelectedAlternativeID == alt.Alternative.ID

				// 构建行内容
				lineText := fmt.Sprintf("%s%s ×%.2f",
					prefix,
					alt.Alternative.DisplayName,
					alt.Alternative.RateMultiplier,
				)

				// 如果是当前选中项，应用绿色样式
				if isCurrentSelection {
					lineText = selectedItemStyle.Render(lineText + " [当前]")
				}

				lines = append(lines, lineText)
			}
		}
	}

	content := strings.Join(lines, "\n")

	style := panelStyle.Copy()
	if m.focus == focusAlternatives {
		style = style.Copy().BorderStyle(activeBorder).BorderForeground(primaryColor)
	}
	return style.Width(m.panelWidth()).Height(10).Render(content)
}

func (m *Model) panelWidth() int {
	if m.width <= 0 {
		return 50
	}
	w := m.width/2 - 3
	if w < 30 {
		return 30
	}
	return w
}


func formatSourceSuffix(source string) string {
	label := translateSourceLabel(source)
	if label == "" {
		return ""
	}
	return fmt.Sprintf(" (%s)", label)
}

func translateSourceLabel(source string) string {
	switch source {
	case "subscription":
		return "订阅"
	case "pay_as_you_go":
		return "按需"
	default:
		return source
	}
}

func translateProviderDisplayName(name string) string {
	switch name {
	case "PAYGO APIs (payg)":
		return "PAYGO APIs (按需)"
	default:
		return name
	}
}

func formatTypeSuffix(providerType string) string {
	providerType = strings.TrimSpace(providerType)
	if providerType == "" {
		return ""
	}
	return fmt.Sprintf(" [%s]", providerType)
}

func formatAlternativeTypeSuffix(providerType string) string {
	// 不再显示类型后缀
	return ""
}

var (
	primaryColor      = lipgloss.Color("#00B894")
	panelStyle        = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(1, 2).BorderForeground(lipgloss.Color("#666666"))
	activeBorder      = lipgloss.RoundedBorder()
	titleStyle        = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
	helpStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA"))
	statusStyle       = lipgloss.NewStyle().Foreground(primaryColor)
	selectedItemStyle = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
	activeTabStyle    = lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Padding(0, 2).Border(lipgloss.RoundedBorder(), false, false, true, false).BorderForeground(primaryColor)
	inactiveTabStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Padding(0, 2)
)

func (m *Model) renderTabHeader() string {
	tabs := []string{}

	// Tab 1: 个人资料
	if m.currentTab == tabProfile {
		tabs = append(tabs, activeTabStyle.Render("个人资料"))
	} else {
		tabs = append(tabs, inactiveTabStyle.Render("个人资料"))
	}

	// Tab 2: 提供商
	if m.currentTab == tabProviders {
		tabs = append(tabs, activeTabStyle.Render("提供商"))
	} else {
		tabs = append(tabs, inactiveTabStyle.Render("提供商"))
	}

	// Tab 3: 余额使用偏好
	if m.currentTab == tabBalancePreference {
		tabs = append(tabs, activeTabStyle.Render("余额使用偏好"))
	} else {
		tabs = append(tabs, inactiveTabStyle.Render("余额使用偏好"))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

func (m *Model) renderProfileTab() string {
	if m.profile == nil {
		return ""
	}

	var lines []string

	// 账户信息
	lines = append(lines, titleStyle.Render("账户信息"))
	lines = append(lines, fmt.Sprintf("用户名: %s", m.profile.Username))
	lines = append(lines, fmt.Sprintf("邮箱: %s", m.profile.Email))
	lines = append(lines, "")

	// 余额概览
	lines = append(lines, titleStyle.Render("余额概览"))
	lines = append(lines, fmt.Sprintf("订阅余额: $%.2f", m.profile.SubscriptionBalance))
	lines = append(lines, fmt.Sprintf("按需余额: $%.2f", m.profile.PayAsYouGoBalance))
	lines = append(lines, fmt.Sprintf("总余额: $%.2f", m.profile.Balance))
	lines = append(lines, fmt.Sprintf("余额使用偏好: %s", describePreference(m.profile.BalancePreference)))

	// 订阅计划（如果存在）
	if m.profile.SubscriptionPlan.Name != "" {
		lines = append(lines, "")
		lines = append(lines, titleStyle.Render("订阅计划"))
		plan := m.profile.SubscriptionPlan
		lines = append(lines, fmt.Sprintf("计划: %s ($%.2f)", plan.Name, plan.Price))

		// 优化截止日期显示
		if m.profile.SubscriptionExpiry != "" {
			expiryDate := m.formatDate(m.profile.SubscriptionExpiry)
			lines = append(lines, fmt.Sprintf("到期: %s", expiryDate))
		}

		lines = append(lines, fmt.Sprintf("每日额度: $%.2f", plan.DailyBalance))

		// 本周消费（带百分比）
		weekPercent := 0.0
		if plan.WeeklyLimit > 0 {
			weekPercent = (m.profile.CurrentWeekSpend / plan.WeeklyLimit) * 100
		}
		lines = append(lines, fmt.Sprintf("本周: $%.2f / $%.2f (%.1f%%)",
			m.profile.CurrentWeekSpend, plan.WeeklyLimit, weekPercent))

		// 本月消费（带百分比）
		monthPercent := 0.0
		if plan.MonthlySpendLimit > 0 {
			monthPercent = (m.profile.CurrentMonthSpend / plan.MonthlySpendLimit) * 100
		}
		lines = append(lines, fmt.Sprintf("本月: $%.2f / $%.2f (%.1f%%)",
			m.profile.CurrentMonthSpend, plan.MonthlySpendLimit, monthPercent))
	} else {
		// 如果没有订阅计划，仍然显示消费统计
		lines = append(lines, "")
		lines = append(lines, titleStyle.Render("消费统计"))
		lines = append(lines, fmt.Sprintf("本周消费: $%.2f", m.profile.CurrentWeekSpend))
		lines = append(lines, fmt.Sprintf("本月消费: $%.2f", m.profile.CurrentMonthSpend))
	}

	content := strings.Join(lines, "\n")

	// 更新 viewport 的内容和尺寸
	m.profileViewport.SetContent(content)
	m.profileViewport.Height = m.contentHeight()
	if m.width > 0 {
		m.profileViewport.Width = m.width - 4 // 减去一些边距
	}

	var output []string

	// viewport 主内容
	output = append(output, m.profileViewport.View())

	// 底部滚动指示器
	var bottomParts []string
	if !m.profileViewport.AtBottom() {
		moreIndicator := lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Render("↓ 更多内容")
		bottomParts = append(bottomParts, moreIndicator)
	}

	if len(bottomParts) > 0 {
		output = append(output, strings.Join(bottomParts, "  "))
	}

	return strings.Join(output, "\n")
}

// formatDate 优化日期显示的可读性
func (m *Model) formatDate(dateStr string) string {
	// 尝试解析常见的日期格式
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			// 返回更友好的格式：2024年1月15日
			return t.Format("2006年1月2日")
		}
	}

	// 如果解析失败，返回原始字符串
	return dateStr
}

func (m *Model) renderBalancePreferenceTab() string {
	if m.profile == nil {
		return "加载中..."
	}

	var lines []string

	// 优先订阅选项 (索引0)
	prefix := "  "
	if m.balancePreferenceIdx == 0 {
		prefix = "→ "
	}
	label := "优先订阅"
	if m.profile.BalancePreference == "subscription_first" {
		label = label + " [当前]"
		lines = append(lines, selectedItemStyle.Render(prefix+label))
	} else {
		lines = append(lines, prefix+label)
	}
	lines = append(lines, "    先使用订阅余额，然后使用按需付费。OPUS使用限制适用。")
	lines = append(lines, "")

	// 仅按需付费选项 (索引1)
	prefix = "  "
	if m.balancePreferenceIdx == 1 {
		prefix = "→ "
	}
	label = "仅按需付费"
	if m.profile.BalancePreference == "payg_only" {
		label = label + " [当前]"
		lines = append(lines, selectedItemStyle.Render(prefix+label))
	} else {
		lines = append(lines, prefix+label)
	}
	lines = append(lines, "    始终使用按需付费余额。无OPUS使用限制。")

	return strings.Join(lines, "\n")
}

func loadProfileCmd(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		profile, err := client.GetProfile(context.Background())
		if err != nil {
			return errMsg{err: err}
		}
		return profileLoadedMsg{profile: profile}
	}
}

func loadProvidersCmd(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.GetAvailableProviders(context.Background())
		if err != nil {
			return errMsg{err: err}
		}
		return providersLoadedMsg{response: resp}
	}
}

func loadAlternativesCmd(client *api.Client, providerID int) tea.Cmd {
	return func() tea.Msg {
		alts, err := client.GetProviderAlternatives(context.Background(), providerID)
		if err != nil {
			return providerLoadFailedMsg{providerID: providerID, target: "alternatives", err: err}
		}
		return alternativesLoadedMsg{providerID: providerID, alternatives: alts}
	}
}

func loadSelectionCmd(client *api.Client, providerID int) tea.Cmd {
	return func() tea.Msg {
		selection, err := client.GetProviderSelection(context.Background(), providerID)
		if err != nil {
			return providerLoadFailedMsg{providerID: providerID, target: "selection", err: err}
		}
		return selectionLoadedMsg{providerID: providerID, selection: selection}
	}
}

func switchProviderCmd(client *api.Client, providerID, alternativeID int) tea.Cmd {
	return func() tea.Msg {
		selection, err := client.SwitchProvider(context.Background(), providerID, alternativeID)
		if err != nil {
			return providerLoadFailedMsg{providerID: providerID, target: "switch", err: err}
		}
		return switchCompletedMsg{providerID: providerID, selection: selection}
	}
}

func updatePreferenceCmd(client *api.Client, preference string) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.UpdateBalancePreference(context.Background(), preference)
		if err != nil {
			return preferenceFailedMsg{err: err}
		}
		return preferenceUpdatedMsg{preference: resp.BalancePreference}
	}
}

func clearStatusAfterDelay(seconds int) tea.Cmd {
	return tea.Tick(time.Duration(seconds)*time.Second, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

func describePreference(pref string) string {
	switch pref {
	case "subscription_first":
		return "优先订阅"
	case "payg_only":
		return "仅按需付费"
	default:
		if pref == "" {
			return "未知"
		}
		return pref
	}
}

func nextPreference(current string) string {
	switch current {
	case "subscription_first":
		return "payg_only"
	case "payg_only":
		return "subscription_first"
	default:
		return "subscription_first"
	}
}
