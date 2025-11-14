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

	"yescode-tui/internal/api"
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

// UI layout constants
const (
	defaultViewportHeight  = 20
	defaultPanelHeight     = 10
	minPanelWidth          = 30
	viewportWidthMargin    = 4
	profileRefreshInterval = 5 * time.Second
	statusClearDelay       = 2 * time.Second
	errorClearDelay        = 3 * time.Second
)

// UI element positions (calculated relative to View() output)
type uiLayout struct {
	titleLineY        int // Title line Y position
	helpLineY         int // Help hint line Y position
	tabHeaderY        int // Tab header line Y position
	contentStartY     int // Content area start Y position
	panelInnerOffsetY int // Y offset for panel inner content (border + padding)
	panelInnerOffsetX int // X offset for panel inner content (border + padding)
}

// getUILayout calculates UI element positions based on View() structure.
func getUILayout() uiLayout {
	return uiLayout{
		titleLineY:        0, // Title at Y=0
		helpLineY:         2, // Help hint at Y=2 (title + blank line)
		tabHeaderY:        4, // Tab header at Y=4 (title + blank + help + blank)
		contentStartY:     6, // Content starts at Y=6 (after tab header + blank)
		panelInnerOffsetY: 2, // Panel has 1 line border + 1 line padding
		panelInnerOffsetX: 3, // Panel has left border (1) + left padding (2)
	}
}

// Model wires Bubble Tea with the YesCode API client.
type Model struct {
	client *api.Client

	profile                 *api.Profile
	providers               []api.ProviderBucket
	providerIdx             int
	altIdx                  int
	balancePreferenceIdx    int
	focus                   focusArea
	currentTab              tabIndex
	ready                   bool
	status                  string
	err                     error
	width                   int
	height                  int
	providerData            map[int]*providerState
	preferenceSwitching     bool
	spinner                 spinner.Model
	help                    help.Model
	keys                    keyMap
	profileViewport         viewport.Model
	providersLoaded         bool
	loadingProviders        bool
	loadingProfile          bool
	manualRefreshingProfile bool
	showHelpDialog          bool
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
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Tab      key.Binding
	ShiftTab key.Binding
	Enter    key.Binding
	Refresh  key.Binding
	Tab1     key.Binding
	Tab2     key.Binding
	Tab3     key.Binding
	Help     key.Binding
	Quit     key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Left, k.Right, k.Enter, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab, k.ShiftTab, k.Tab1, k.Tab2, k.Tab3},
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
		key.WithHelp("tab", "下一标签页"),
	),
	ShiftTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "上一标签页"),
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
		key.WithHelp("1", "用户资料"),
	),
	Tab2: key.NewBinding(
		key.WithKeys("2"),
		key.WithHelp("2", "提供商"),
	),
	Tab3: key.NewBinding(
		key.WithKeys("3"),
		key.WithHelp("3", "余额使用偏好"),
	),
	Help: key.NewBinding(
		key.WithKeys("?", "？"),
		key.WithHelp("?", "帮助"),
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

type profileRefreshTickMsg struct{}

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
	vp := viewport.New(0, defaultViewportHeight)

	return &Model{
		client:          client,
		focus:           focusProviders,
		providerData:    make(map[int]*providerState),
		spinner:         s,
		help:            h,
		keys:            keys,
		profileViewport: vp,
		ready:           true,
		loadingProfile:  true,
	}
}

// Init triggers the first batch of API calls.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		loadProfileCmd(m.client),
		m.spinner.Tick,
		profileRefreshTicker(),
	)
}

// Update handles Bubble Tea messages.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowResize(msg)
	case tea.KeyMsg:
		if cmd := m.handleKey(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case tea.MouseMsg:
		if cmd := m.handleMouse(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	case profileLoadedMsg:
		m.handleProfileLoaded(msg)
	case profileRefreshTickMsg:
		cmds = append(cmds, m.handleProfileRefreshTick()...)
	case providersLoadedMsg:
		cmds = append(cmds, m.handleProvidersLoaded(msg)...)
	case alternativesLoadedMsg:
		m.handleAlternativesLoaded(msg)
	case selectionLoadedMsg:
		m.handleSelectionLoaded(msg)
	case switchCompletedMsg:
		cmds = append(cmds, m.handleSwitchCompleted(msg)...)
	case preferenceUpdatedMsg:
		cmds = append(cmds, m.handlePreferenceUpdated(msg)...)
	case preferenceFailedMsg:
		cmds = append(cmds, m.handlePreferenceFailed(msg)...)
	case providerLoadFailedMsg:
		cmds = append(cmds, m.handleProviderLoadFailed(msg)...)
	case errMsg:
		cmds = append(cmds, m.handleError(msg)...)
	case clearStatusMsg:
		m.handleClearStatus()
	}

	// 更新 spinner
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// handleWindowResize updates dimensions when the window is resized.
func (m *Model) handleWindowResize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height
	m.help.Width = msg.Width
}

// handleProfileLoaded processes successful profile load.
func (m *Model) handleProfileLoaded(msg profileLoadedMsg) {
	m.profile = msg.profile
	m.loadingProfile = false
	m.manualRefreshingProfile = false
	m.status = ""
}

// handleProfileRefreshTick handles periodic profile refresh.
func (m *Model) handleProfileRefreshTick() []tea.Cmd {
	var cmds []tea.Cmd
	// 只在profile tab时自动刷新（不显示loading）
	if m.currentTab == tabProfile {
		cmds = append(cmds, loadProfileCmd(m.client))
	}
	// 继续下一个tick
	cmds = append(cmds, profileRefreshTicker())
	return cmds
}

// handleProvidersLoaded processes provider list load.
func (m *Model) handleProvidersLoaded(msg providersLoadedMsg) []tea.Cmd {
	var cmds []tea.Cmd
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
	return cmds
}

// handleAlternativesLoaded processes alternatives load.
func (m *Model) handleAlternativesLoaded(msg alternativesLoadedMsg) {
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
}

// handleSelectionLoaded processes selection load.
func (m *Model) handleSelectionLoaded(msg selectionLoadedMsg) {
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
}

// handleSwitchCompleted processes provider switch completion.
func (m *Model) handleSwitchCompleted(msg switchCompletedMsg) []tea.Cmd {
	state := m.ensureProviderState(msg.providerID)
	state.selection = msg.selection
	state.selectionLoaded = true
	state.switching = false
	state.lastError = nil
	m.syncAltIdx(msg.providerID)
	m.status = fmt.Sprintf("已切换到 %s", msg.selection.SelectedAlternative.DisplayName)
	return []tea.Cmd{clearStatusAfter(statusClearDelay)}
}

// handlePreferenceUpdated processes preference update success.
func (m *Model) handlePreferenceUpdated(msg preferenceUpdatedMsg) []tea.Cmd {
	if m.profile != nil {
		m.profile.BalancePreference = msg.preference
	}
	m.preferenceSwitching = false
	m.syncBalancePreferenceIdx()
	m.status = fmt.Sprintf("余额偏好已切换为 %s", describePreference(msg.preference))
	return []tea.Cmd{clearStatusAfter(statusClearDelay)}
}

// handlePreferenceFailed processes preference update failure.
func (m *Model) handlePreferenceFailed(msg preferenceFailedMsg) []tea.Cmd {
	m.preferenceSwitching = false
	m.err = msg.err
	m.status = fmt.Sprintf("余额偏好切换失败: %v", msg.err)
	return []tea.Cmd{clearStatusAfter(errorClearDelay)}
}

// handleProviderLoadFailed processes provider load failures.
func (m *Model) handleProviderLoadFailed(msg providerLoadFailedMsg) []tea.Cmd {
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
	return []tea.Cmd{clearStatusAfter(errorClearDelay)}
}

// handleError processes general errors.
func (m *Model) handleError(msg errMsg) []tea.Cmd {
	m.err = msg.err
	m.status = msg.err.Error()

	// 如果是加载提供商失败，重置加载状态
	if m.loadingProviders {
		m.loadingProviders = false
	}
	// 如果是加载用户资料失败，重置加载状态
	if m.loadingProfile {
		m.loadingProfile = false
		m.manualRefreshingProfile = false
	}
	return []tea.Cmd{clearStatusAfter(errorClearDelay)}
}

// handleClearStatus clears status and error messages.
func (m *Model) handleClearStatus() {
	m.status = ""
	m.err = nil
}

// View renders the TUI.
func (m *Model) View() string {
	var sections []string

	// Material Design 风格应用标题
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Width(m.width).
		Align(lipgloss.Center)

	sections = append(sections, titleStyle.Render("◆ YesCode TUI ◆"))

	// 简洁的帮助提示
	helpHintStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Width(m.width).
		Align(lipgloss.Center)
	sections = append(sections, helpHintStyle.Render("支持鼠标操作 · Enter 确认 · Esc 退出 · 输入 ? 查看完整操作帮助"))

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

	// 如果正在手动刷新用户资料，显示刷新状态
	if m.manualRefreshingProfile && m.currentTab == tabProfile {
		statusText = fmt.Sprintf("刷新中... %s", m.spinner.View())
	} else if m.status != "" {
		statusText = m.status
		// 如果状态消息表示正在进行中，添加 spinner
		if strings.Contains(statusText, "中...") || strings.Contains(statusText, "加载") {
			statusText = fmt.Sprintf("%s %s", statusText, m.spinner.View())
		}
	}
	sections = append(sections, statusStyle.Render(statusText))

	mainView := strings.Join(sections, "\n\n")

	// 如果帮助对话框打开，只显示对话框，隐藏主页面
	if m.showHelpDialog {
		dialog := m.renderHelpDialog()
		// 将对话框居中放置在全屏空间中
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	return mainView
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	// Handle Ctrl+C first
	if msg.Type == tea.KeyCtrlC {
		return tea.Quit
	}

	key := msg.String()

	// Handle quit and help
	if cmd := m.handleQuitAndHelp(key); cmd != nil {
		return cmd
	}

	// Handle tab switching
	if cmd := m.handleTabSwitch(key); cmd != nil {
		return cmd
	}

	// Handle focus switching (left/right)
	m.handleFocusSwitch(key)

	// Handle refresh
	if cmd := m.handleRefresh(key); cmd != nil {
		return cmd
	}

	// Handle enter
	if cmd := m.handleEnter(key); cmd != nil {
		return cmd
	}

	// Handle navigation (up/down)
	if cmd := m.handleNavigation(key); cmd != nil {
		return cmd
	}

	return nil
}

// handleQuitAndHelp handles Esc and ? keys.
func (m *Model) handleQuitAndHelp(key string) tea.Cmd {
	switch key {
	case "esc":
		// 如果帮助对话框打开，关闭它；否则退出程序
		if m.showHelpDialog {
			m.showHelpDialog = false
			return nil
		}
		return tea.Quit
	case "?", "？":
		// 切换帮助对话框显示状态
		m.showHelpDialog = !m.showHelpDialog
		return nil
	}
	return nil
}

// handleTabSwitch handles tab switching keys (1, 2, 3, tab, shift+tab).
func (m *Model) handleTabSwitch(key string) tea.Cmd {
	switch key {
	case "1":
		m.currentTab = tabProfile
		return nil
	case "2":
		m.currentTab = tabProviders
		m.focus = focusProviders
		return m.ensureProvidersLoaded()
	case "3":
		m.currentTab = tabBalancePreference
		m.syncBalancePreferenceIdx()
		return nil
	case "tab":
		return m.switchToNextTab()
	case "shift+tab":
		return m.switchToPrevTab()
	}
	return nil
}

// switchToNextTab switches to the next tab.
func (m *Model) switchToNextTab() tea.Cmd {
	m.currentTab = (m.currentTab + 1) % 3
	return m.handleTabChanged()
}

// switchToPrevTab switches to the previous tab.
func (m *Model) switchToPrevTab() tea.Cmd {
	m.currentTab = (m.currentTab - 1 + 3) % 3
	return m.handleTabChanged()
}

// handleTabChanged handles post-tab-switch logic.
func (m *Model) handleTabChanged() tea.Cmd {
	if m.currentTab == tabProviders {
		m.focus = focusProviders
		return m.ensureProvidersLoaded()
	} else if m.currentTab == tabBalancePreference {
		m.syncBalancePreferenceIdx()
	}
	return nil
}

// handleFocusSwitch handles left/right focus switching.
func (m *Model) handleFocusSwitch(key string) {
	if m.currentTab != tabProviders {
		return
	}

	switch key {
	case "left", "h":
		m.focus = focusProviders
	case "right", "l":
		m.focus = focusAlternatives
		// 切换到右栏时，同步游标到当前激活项
		m.syncAltIdx(m.currentProviderID())
	}
}

// handleRefresh handles refresh key (r).
func (m *Model) handleRefresh(key string) tea.Cmd {
	if key != "r" {
		return nil
	}

	switch m.currentTab {
	case tabProfile:
		return m.refreshProfile()
	case tabProviders:
		return m.refreshCurrentProvider()
	}
	return nil
}

// handleEnter handles enter key actions.
func (m *Model) handleEnter(key string) tea.Cmd {
	if key != "enter" {
		return nil
	}

	switch m.currentTab {
	case tabProviders:
		if m.focus == focusAlternatives {
			return m.switchSelection()
		}
	case tabBalancePreference:
		return m.toggleBalancePreference()
	}
	return nil
}

// handleNavigation handles up/down navigation keys.
func (m *Model) handleNavigation(key string) tea.Cmd {
	delta := 0
	switch key {
	case "up", "k":
		delta = -1
	case "down", "j":
		delta = 1
	default:
		return nil
	}

	// Profile tab: scroll viewport
	if m.currentTab == tabProfile {
		if delta < 0 {
			m.profileViewport.LineUp(1)
		} else {
			m.profileViewport.LineDown(1)
		}
		return nil
	}

	// Balance preference tab: move between two options
	if m.currentTab == tabBalancePreference {
		m.balancePreferenceIdx = clampIndex(m.balancePreferenceIdx+delta, 2)
		return nil
	}

	// Providers tab: move selection
	return m.moveSelection(delta)
}

func (m *Model) handleMouse(msg tea.MouseMsg) tea.Cmd {
	x, y := msg.X, msg.Y

	// 处理滚轮滚动
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		return m.handleMouseWheel(-1)
	case tea.MouseButtonWheelDown:
		return m.handleMouseWheel(1)
	}

	// 只处理左键点击
	if msg.Button != tea.MouseButtonLeft || msg.Action != tea.MouseActionPress {
		return nil
	}

	layout := getUILayout()

	// 点击标签页
	if y == layout.tabHeaderY {
		return m.handleTabClick(x)
	}

	// 点击内容区域
	if y >= layout.contentStartY {
		return m.handleContentClick(x, y)
	}

	return nil
}

func (m *Model) handleMouseWheel(delta int) tea.Cmd {
	if m.currentTab == tabProfile {
		// Profile tab: 滚动 viewport
		if delta < 0 {
			m.profileViewport.LineUp(1)
		} else {
			m.profileViewport.LineDown(1)
		}
		return nil
	} else if m.currentTab == tabProviders || m.currentTab == tabBalancePreference {
		// 其他 tab: 上下移动选择
		return m.moveSelection(delta)
	}
	return nil
}

func (m *Model) handleTabClick(x int) tea.Cmd {
	// 计算标签页位置
	// 使用 lipgloss 的宽度计算，更准确地处理中文字符
	tab1Text := "1. 用户资料"
	tab2Text := "2. 提供商"

	// activeTabStyle: padding(0,2) + marginRight(1)
	// 中文字符通常占 2 个宽度单位
	tab1Width := lipgloss.Width(activeTabStyle.Render(tab1Text))
	tab2Width := lipgloss.Width(activeTabStyle.Render(tab2Text))

	tab1End := tab1Width
	tab2End := tab1End + tab2Width

	if x < tab1End {
		m.currentTab = tabProfile
	} else if x < tab2End {
		m.currentTab = tabProviders
		m.focus = focusProviders
		return m.ensureProvidersLoaded()
	} else {
		m.currentTab = tabBalancePreference
		m.syncBalancePreferenceIdx()
	}
	return nil
}

func (m *Model) handleContentClick(x, y int) tea.Cmd {
	layout := getUILayout()
	contentY := y - layout.contentStartY

	switch m.currentTab {
	case tabProviders:
		return m.handleProvidersClick(x, contentY)
	case tabBalancePreference:
		return m.handleBalancePreferenceClick(contentY)
	}
	return nil
}

func (m *Model) handleProvidersClick(x, contentY int) tea.Cmd {
	if len(m.providers) == 0 {
		return nil
	}

	layout := getUILayout()
	// 面板内部列表项的 Y 位置需要减去面板的边框和内边距
	listItemY := contentY - layout.panelInnerOffsetY

	// 左右面板以屏幕中心分界
	if x < m.width/2 {
		// 点击左侧提供商列表
		m.focus = focusProviders
		if listItemY >= 0 && listItemY < len(m.providers) {
			m.providerIdx = listItemY
			return m.queueProviderDetailLoad(m.currentProviderID())
		}
	} else {
		// 点击右侧备选方案列表
		m.focus = focusAlternatives
		state := m.ensureProviderState(m.currentProviderID())
		if state.alternativesLoaded {
			if listItemY >= 0 && listItemY < len(state.alternatives) {
				m.altIdx = listItemY
				// 直接确认切换
				return m.switchSelection()
			} else {
				// 点击空白区域，同步游标到当前激活项
				m.syncAltIdx(m.currentProviderID())
			}
		}
	}
	return nil
}

func (m *Model) handleBalancePreferenceClick(contentY int) tea.Cmd {
	// 余额偏好页面布局：
	// 第一个选项：包括标题行(0) + 两行说明(1-2)
	// 空行(3)
	// 第二个选项：包括标题行(4) + 两行说明(5-6)

	const (
		option1Start = 0
		option1End   = 2
		option2Start = 4
		option2End   = 6
	)

	var targetIdx int
	if contentY >= option1Start && contentY <= option1End {
		targetIdx = 0
	} else if contentY >= option2Start && contentY <= option2End {
		targetIdx = 1
	} else {
		return nil
	}

	if m.balancePreferenceIdx != targetIdx {
		m.balancePreferenceIdx = targetIdx
		return m.toggleBalancePreference()
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
	m.loadingProfile = true
	m.manualRefreshingProfile = true
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

	// 如果数据已经加载完成，立即同步游标位置到当前激活项
	if state.alternativesLoaded && state.selectionLoaded {
		m.syncAltIdx(providerID)
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
	return defaultViewportHeight
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
		lines = append(lines, fmt.Sprintf("加载中... %s", m.spinner.View()))
	} else if len(m.providers) == 0 {
		lines = append(lines, "暂无可用提供商")
	} else {
		for i, bucket := range m.providers {
			prefix := "  "
			if i == m.providerIdx {
				prefix = "▶ "
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
	return style.Width(m.panelWidth()).Height(defaultPanelHeight).Render(content)
}

func (m *Model) renderAlternativesPanel() string {
	var lines []string

	if len(m.providers) == 0 {
		lines = append(lines, "请先选择提供商")
	} else {
		state := m.ensureProviderState(m.currentProviderID())

		switch {
		case state.loadingAlternatives:
			lines = append(lines, fmt.Sprintf("加载中... %s", m.spinner.View()))
		case state.lastError != nil:
			errorStyle := lipgloss.NewStyle().Foreground(errorColor)
			lines = append(lines, errorStyle.Render(fmt.Sprintf("⚠ 错误：%v", state.lastError)))
			lines = append(lines, "")
			lines = append(lines, "按 r 键重试")
		case len(state.alternatives) == 0:
			lines = append(lines, "无可切换方案")
		default:
			for i, alt := range state.alternatives {
				prefix := "  "
				if i == m.altIdx {
					prefix = "▶ "
				}

				// 检查是否为当前选中项
				isCurrentSelection := state.selection != nil && state.selection.SelectedAlternativeID == alt.Alternative.ID

				// 构建行内容
				lineText := fmt.Sprintf("%s%s ×%.2f",
					prefix,
					alt.Alternative.DisplayName,
					alt.Alternative.RateMultiplier,
				)

				// 如果是当前选中项，添加标记
				if isCurrentSelection {
					checkStyle := lipgloss.NewStyle().Foreground(successColor)
					lineText = selectedItemStyle.Render(lineText) + " " + checkStyle.Render("✓")
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
	return style.Width(m.panelWidth()).Height(defaultPanelHeight).Render(content)
}

func (m *Model) panelWidth() int {
	if m.width <= 0 {
		return 50
	}
	w := m.width/2 - 3
	if w < minPanelWidth {
		return minPanelWidth
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
	case "pay_as_you_go", "payg":
		return "按需"
	default:
		return source
	}
}

func translateProviderDisplayName(name string) string {
	// 不翻译提供商名称，保持原样
	return name
}

func formatTypeSuffix(providerType string) string {
	providerType = strings.TrimSpace(providerType)
	if providerType == "" {
		return ""
	}
	return fmt.Sprintf(" [%s]", providerType)
}

var (
	// Material Design 风格配色
	primaryColor   = lipgloss.Color("#2196F3") // Material Blue
	secondaryColor = lipgloss.Color("#1976D2") // Dark Blue
	accentColor    = lipgloss.Color("#FF4081") // Pink Accent
	mutedColor     = lipgloss.Color("#9E9E9E") // Grey
	successColor   = lipgloss.Color("#4CAF50") // Green
	errorColor     = lipgloss.Color("#F44336") // Red
	warningColor   = lipgloss.Color("#FF9800") // Orange

	panelStyle        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).BorderForeground(mutedColor)
	activeBorder      = lipgloss.RoundedBorder()
	titleStyle        = lipgloss.NewStyle().Bold(true).Foreground(primaryColor)
	helpStyle         = lipgloss.NewStyle().Foreground(mutedColor)
	statusStyle       = lipgloss.NewStyle().Foreground(primaryColor)
	selectedItemStyle = lipgloss.NewStyle().Bold(true).Foreground(accentColor)
	activeTabStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(primaryColor).Padding(0, 2).MarginRight(1)
	inactiveTabStyle  = lipgloss.NewStyle().Foreground(mutedColor).Padding(0, 2).MarginRight(1)
)

func (m *Model) renderTabHeader() string {
	tabs := []string{}

	// Tab 1: 用户资料
	if m.currentTab == tabProfile {
		tabs = append(tabs, activeTabStyle.Render("1. 用户资料"))
	} else {
		tabs = append(tabs, inactiveTabStyle.Render("1. 用户资料"))
	}

	// Tab 2: 提供商
	if m.currentTab == tabProviders {
		tabs = append(tabs, activeTabStyle.Render("2. 提供商"))
	} else {
		tabs = append(tabs, inactiveTabStyle.Render("2. 提供商"))
	}

	// Tab 3: 余额使用偏好
	if m.currentTab == tabBalancePreference {
		tabs = append(tabs, activeTabStyle.Render("3. 余额使用偏好"))
	} else {
		tabs = append(tabs, inactiveTabStyle.Render("3. 余额使用偏好"))
	}

	tabsRow := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	return tabsRow
}

func (m *Model) renderProfileTab() string {
	// 只在首次加载（profile为空且不是手动刷新）时显示内容区加载状态
	// 手动刷新时在状态栏显示，内容区保持不变
	if m.profile == nil && !m.manualRefreshingProfile {
		return fmt.Sprintf("加载中... %s", m.spinner.View())
	}

	// 如果profile还是nil（不应该发生，但防御性处理）
	if m.profile == nil {
		return ""
	}

	// 构建内容
	var lines []string
	lines = append(lines, m.renderAccountInfo()...)
	lines = append(lines, "")
	lines = append(lines, m.renderBalanceOverview()...)

	if m.profile.SubscriptionPlan.Name != "" {
		lines = append(lines, "")
		lines = append(lines, m.renderSubscriptionPlan()...)
	} else {
		lines = append(lines, "")
		lines = append(lines, m.renderSpendingStats()...)
	}

	content := strings.Join(lines, "\n")
	m.setupProfileViewport(content)

	// 构建输出
	var output []string
	output = append(output, m.profileViewport.View())

	if scrollIndicator := m.renderScrollIndicator(); scrollIndicator != "" {
		output = append(output, scrollIndicator)
	}

	return strings.Join(output, "\n")
}

// renderAccountInfo renders account information section.
func (m *Model) renderAccountInfo() []string {
	return []string{
		titleStyle.Render("账户信息"),
		fmt.Sprintf("  用户名：%s", m.profile.Username),
		fmt.Sprintf("  邮箱：%s", m.profile.Email),
	}
}

// renderBalanceOverview renders balance overview section.
func (m *Model) renderBalanceOverview() []string {
	return []string{
		titleStyle.Render("余额概览"),
		fmt.Sprintf("  ● 订阅余额：$%.2f", m.profile.SubscriptionBalance),
		fmt.Sprintf("  ● 按需余额：$%.2f", m.profile.PayAsYouGoBalance),
		fmt.Sprintf("  ● 总余额：$%.2f", m.profile.Balance),
		fmt.Sprintf("  ● 余额偏好：%s", describePreference(m.profile.BalancePreference)),
	}
}

// renderSubscriptionPlan renders subscription plan details.
func (m *Model) renderSubscriptionPlan() []string {
	plan := m.profile.SubscriptionPlan
	lines := []string{
		titleStyle.Render("订阅计划"),
		fmt.Sprintf("  ● 计划：%s ($%.2f)", plan.Name, plan.Price),
	}

	// 优化截止日期显示
	if m.profile.SubscriptionExpiry != "" {
		expiryDate := m.formatDate(m.profile.SubscriptionExpiry)
		lines = append(lines, fmt.Sprintf("  ● 到期：%s", expiryDate))
	}

	lines = append(lines, fmt.Sprintf("  ● 每日额度：$%.2f", plan.DailyBalance))

	// 本周消费（带百分比）
	weekPercent := 0.0
	if plan.WeeklyLimit > 0 {
		weekPercent = (m.profile.CurrentWeekSpend / plan.WeeklyLimit) * 100
	}
	lines = append(lines, fmt.Sprintf("  ● 本周：$%.2f / $%.2f (%.1f%%)",
		m.profile.CurrentWeekSpend, plan.WeeklyLimit, weekPercent))

	// 本月消费（带百分比）
	monthPercent := 0.0
	if plan.MonthlySpendLimit > 0 {
		monthPercent = (m.profile.CurrentMonthSpend / plan.MonthlySpendLimit) * 100
	}
	lines = append(lines, fmt.Sprintf("  ● 本月：$%.2f / $%.2f (%.1f%%)",
		m.profile.CurrentMonthSpend, plan.MonthlySpendLimit, monthPercent))

	return lines
}

// renderSpendingStats renders spending statistics when no subscription plan exists.
func (m *Model) renderSpendingStats() []string {
	return []string{
		titleStyle.Render("消费统计"),
		fmt.Sprintf("  ● 本周消费：$%.2f", m.profile.CurrentWeekSpend),
		fmt.Sprintf("  ● 本月消费：$%.2f", m.profile.CurrentMonthSpend),
	}
}

// setupProfileViewport configures the viewport with content and dimensions.
func (m *Model) setupProfileViewport(content string) {
	m.profileViewport.SetContent(content)
	m.profileViewport.Height = m.contentHeight()
	if m.width > 0 {
		m.profileViewport.Width = m.width - viewportWidthMargin
	}
}

// renderScrollIndicator returns a scroll indicator if more content is available.
func (m *Model) renderScrollIndicator() string {
	if m.profileViewport.AtBottom() {
		return ""
	}
	return lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true).
		Render("▼ 更多内容")
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
		prefix = "▶ "
	}
	label := "优先订阅"
	if m.profile.BalancePreference == "subscription_first" {
		checkStyle := lipgloss.NewStyle().Foreground(successColor)
		lines = append(lines, selectedItemStyle.Render(prefix+label)+" "+checkStyle.Render("✓"))
	} else {
		lines = append(lines, prefix+label)
	}
	lines = append(lines, "    先使用订阅余额，然后使用按需付费")
	lines = append(lines, "    OPUS 使用限制适用")
	lines = append(lines, "")

	// 仅按需付费选项 (索引1)
	prefix = "  "
	if m.balancePreferenceIdx == 1 {
		prefix = "▶ "
	}
	label = "仅按需付费"
	if m.profile.BalancePreference == "payg_only" {
		checkStyle := lipgloss.NewStyle().Foreground(successColor)
		lines = append(lines, selectedItemStyle.Render(prefix+label)+" "+checkStyle.Render("✓"))
	} else {
		lines = append(lines, prefix+label)
	}
	lines = append(lines, "    始终使用按需付费余额")
	lines = append(lines, "    无 OPUS 使用限制")

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

func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

func profileRefreshTicker() tea.Cmd {
	return tea.Tick(profileRefreshInterval, func(time.Time) tea.Msg {
		return profileRefreshTickMsg{}
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

func (m *Model) renderHelpDialog() string {
	// 样式定义 - 使用主题色
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(primaryColor)  // 主蓝色标题
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(accentColor) // 浅蓝色章节标题
	normalStyle := lipgloss.NewStyle()                                     // 默认文字色
	hintStyle := lipgloss.NewStyle().Foreground(mutedColor).Italic(true)   // 灰色提示

	// 帮助内容
	helpContent := []string{
		titleStyle.Render("操作帮助"),
		"",
		sectionStyle.Render("鼠标操作"),
		normalStyle.Render("  点击标签页        直接切换标签"),
		normalStyle.Render("  点击列表项        选择提供商或备选方案"),
		normalStyle.Render("  滚轮滚动         滚动内容或移动选择"),
		"",
		sectionStyle.Render("标签页切换"),
		normalStyle.Render("  Tab / Shift+Tab  前后切换标签页"),
		normalStyle.Render("  1 / 2 / 3        直接跳转到指定标签页"),
		"",
		sectionStyle.Render("导航操作"),
		normalStyle.Render("  ↑↓ 或 k/j        上下移动"),
		normalStyle.Render("  ←→ 或 h/l        切换焦点（提供商标签页）"),
		normalStyle.Render("  Enter           确认选择"),
		normalStyle.Render("  r               刷新当前视图"),
		"",
		sectionStyle.Render("其他"),
		normalStyle.Render("  ?               显示/隐藏帮助"),
		normalStyle.Render("  Esc             关闭帮助或退出程序"),
		normalStyle.Render("  Ctrl+C          退出程序"),
		"",
		hintStyle.Render("按 Esc 或 ? 键关闭此帮助"),
	}

	content := strings.Join(helpContent, "\n")

	// 对话框样式 - 无背景色，主题色边框
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor). // 使用主题蓝色作为边框
		Padding(2, 3).
		Width(60).
		Align(lipgloss.Left)

	return dialogStyle.Render(content)
}
