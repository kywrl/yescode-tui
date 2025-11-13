package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"yescode-cli/internal/api"
)

type focusArea int

const (
	focusProviders focusArea = iota
	focusAlternatives
)

// Model wires Bubble Tea with the YesCode API client.
type Model struct {
	client *api.Client

	profile             *api.Profile
	providers           []api.ProviderBucket
	providerIdx         int
	altIdx              int
	focus               focusArea
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
	Up        key.Binding
	Down      key.Binding
	Left      key.Binding
	Right     key.Binding
	Enter     key.Binding
	Refresh   key.Binding
	Balance   key.Binding
	Quit      key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Left, k.Right, k.Enter, k.Refresh, k.Balance, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Enter, k.Refresh, k.Balance, k.Quit},
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
		key.WithKeys("left", "h", "tab"),
		key.WithHelp("←/h/tab", "切换焦点"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l", "tab"),
		key.WithHelp("→/l/tab", "切换焦点"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "选择"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "刷新"),
	),
	Balance: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "切换余额偏好"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "退出"),
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

	return &Model{
		client:       client,
		focus:        focusProviders,
		providerData: make(map[int]*providerState),
		spinner:      s,
		help:         h,
		keys:         keys,
	}
}

// Init triggers the first batch of API calls.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		loadProfileCmd(m.client),
		loadProvidersCmd(m.client),
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
		m.status = fmt.Sprintf("欢迎 %s", msg.profile.Username)
		m.updateReady()
	case providersLoadedMsg:
		m.providers = msg.response.Providers
		if m.providerIdx >= len(m.providers) {
			m.providerIdx = 0
		}
		if len(m.providers) > 0 {
			cmds = append(cmds, m.queueProviderDetailLoad(m.currentProviderID()))
			m.status = fmt.Sprintf("已加载 %d 个提供商分组", len(m.providers))
		}
		m.updateReady()
	case alternativesLoadedMsg:
		state := m.ensureProviderState(msg.providerID)
		state.alternatives = msg.alternatives
		state.alternativesLoaded = true
		state.loadingAlternatives = false
		state.lastError = nil
		m.syncAltIdx(msg.providerID)
	case selectionLoadedMsg:
		state := m.ensureProviderState(msg.providerID)
		state.selection = msg.selection
		state.selectionLoaded = true
		state.loadingSelection = false
		state.lastError = nil
		m.syncAltIdx(msg.providerID)
	case switchCompletedMsg:
		state := m.ensureProviderState(msg.providerID)
		state.selection = msg.selection
		state.selectionLoaded = true
		state.switching = false
		state.lastError = nil
		m.syncAltIdx(msg.providerID)
		m.status = fmt.Sprintf("已切换到 %s", msg.selection.SelectedAlternative.DisplayName)
	case preferenceUpdatedMsg:
		if m.profile != nil {
			m.profile.BalancePreference = msg.preference
		}
		m.preferenceSwitching = false
		m.status = fmt.Sprintf("余额偏好已切换为 %s", describePreference(msg.preference))
	case preferenceFailedMsg:
		m.preferenceSwitching = false
		m.err = msg.err
		m.status = fmt.Sprintf("余额偏好切换失败: %v", msg.err)
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
	case errMsg:
		m.err = msg.err
		m.status = msg.err.Error()
	}

	// 更新 spinner
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the TUI.
func (m *Model) View() string {
	if !m.ready {
		return lipgloss.NewStyle().Padding(1, 2).Render(
			fmt.Sprintf("加载账号与提供商信息中，请稍候... %s", m.spinner.View()),
		)
	}

	var sections []string
	sections = append(sections, m.renderProfile())
	sections = append(sections, m.help.View(m.keys))
	sections = append(sections, m.renderPanels())

	if m.status != "" {
		sections = append(sections, statusStyle.Render(m.status))
	}

	return strings.Join(sections, "\n\n")
}

func (m *Model) updateReady() {
	m.ready = m.profile != nil && len(m.providers) > 0
}


func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyCtrlC:
		return tea.Quit
	}

	switch msg.String() {
	case "q", "esc":
		return tea.Quit
	case "tab":
		if m.focus == focusProviders {
			m.focus = focusAlternatives
		} else {
			m.focus = focusProviders
		}
	case "left":
		m.focus = focusProviders
	case "right":
		m.focus = focusAlternatives
	case "r":
		return m.refreshCurrentProvider()
	case "b":
		return m.toggleBalancePreference()
	case "enter":
		if m.focus == focusAlternatives {
			return m.switchSelection()
		}
	case "up", "k":
		return m.moveSelection(-1)
	case "down", "j":
		return m.moveSelection(1)
	}
	return nil
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
	target := nextPreference(m.profile.BalancePreference)
	if target == m.profile.BalancePreference {
		return nil
	}
	m.preferenceSwitching = true
	m.status = fmt.Sprintf("切换余额偏好到 %s...", describePreference(target))
	return updatePreferenceCmd(m.client, target)
}

func (m *Model) queueProviderDetailLoad(providerID int) tea.Cmd {
	if providerID == 0 {
		return nil
	}
	state := m.ensureProviderState(providerID)
	var cmds []tea.Cmd
	if !state.alternativesLoaded && !state.loadingAlternatives {
		state.loadingAlternatives = true
		cmds = append(cmds, loadAlternativesCmd(m.client, providerID))
	}
	if !state.selectionLoaded && !state.loadingSelection {
		state.loadingSelection = true
		cmds = append(cmds, loadSelectionCmd(m.client, providerID))
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

func (m *Model) renderProfile() string {
	if m.profile == nil {
		return "账号信息加载中..."
	}
	lines := []string{
		titleStyle.Render(fmt.Sprintf("%s (%s)", m.profile.Username, m.profile.Email)),
		fmt.Sprintf("总余额: $%.2f | 订阅余额: $%.2f | 按需: $%.2f", m.profile.Balance, m.profile.SubscriptionBalance, m.profile.PayAsYouGoBalance),
		fmt.Sprintf("偏好: %s | 本周消费: $%.2f | 本月消费: $%.2f", describePreference(m.profile.BalancePreference), m.profile.CurrentWeekSpend, m.profile.CurrentMonthSpend),
	}
	if m.profile.SubscriptionPlan.Name != "" {
		plan := m.profile.SubscriptionPlan
		lines = append(lines,
			fmt.Sprintf("订阅: %s ($%.2f) 截止 %s | 每日额度: $%.2f | 周限额: $%.2f | 月限额: $%.2f",
				plan.Name,
				plan.Price,
				m.profile.SubscriptionExpiry,
				plan.DailyBalance,
				plan.WeeklyLimit,
				plan.MonthlySpendLimit))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) renderHelp() string {
	return helpStyle.Render("↑↓/jk: 移动  |  Tab/←→: 切换焦点  |  Enter: 切换提供商  |  b: 切换余额偏好  |  r: 刷新  |  q: 退出")
}

func (m *Model) renderPanels() string {
	left := m.renderProvidersPanel()
	right := m.renderAlternativesPanel()
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m *Model) renderProvidersPanel() string {
	var lines []string

	// 添加面板标题
	lines = append(lines, titleStyle.Render("提供商"))
	lines = append(lines, "")  // 空行分隔

	if len(m.providers) == 0 {
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
	return style.Width(m.panelWidth()).Render(content)
}

func (m *Model) renderAlternativesPanel() string {
	var lines []string

	// 添加面板标题
	lines = append(lines, titleStyle.Render("替代方案"))
	lines = append(lines, "")  // 空行分隔

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
	return style.Width(m.panelWidth()).Render(content)
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
	primaryColor         = lipgloss.Color("#00B894")
	panelStyle           = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(1, 2).BorderForeground(lipgloss.Color("#666666"))
	activeBorder         = lipgloss.RoundedBorder()
	titleStyle           = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
	helpStyle            = lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA"))
	statusStyle          = lipgloss.NewStyle().Foreground(primaryColor)
	activeIndicatorStyle = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
	selectedItemStyle    = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
)

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

func describePreference(pref string) string {
	switch pref {
	case "subscription_first":
		return "订阅优先"
	case "payg_only":
		return "仅按需"
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
