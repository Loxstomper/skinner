package tui

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/loxstomper/skinner/internal/config"
	"github.com/loxstomper/skinner/internal/executor"
	"github.com/loxstomper/skinner/internal/model"
	"github.com/loxstomper/skinner/internal/session"
	"github.com/loxstomper/skinner/internal/theme"
)

// Bubble Tea messages — thin wrappers around session events

type assistantBatchMsg struct{ session.AssistantBatchEvent }
type toolResultMsg struct{ session.ToolResultEvent }
type usageMsg struct{ session.UsageEvent }
type iterationEndMsg struct{}
type subprocessExitMsg struct{ err error }
type tickMsg time.Time
type promptEditorDoneMsg struct{ err error }

// Pane focus

type paneID int

const (
	iterationsPane paneID = iota
	promptsPane
	rightPane
)

// isLeftPane returns true if the pane is in the left column.
func isLeftPane(p paneID) bool {
	return p == iterationsPane || p == promptsPane
}

// Model is the Bubble Tea model for the TUI.
type Model struct {
	controller    *session.Controller
	exec          executor.Executor
	config        config.Config
	promptContent string
	theme         theme.Theme

	// Sub-components
	iterList   IterList
	promptList PromptList
	timeline   Timeline

	// Working directory for prompt file scanning
	workDir string

	// UI state
	width, height   int
	focusedPane     paneID
	compactView     bool
	leftPaneVisible bool
	lineNumbers     bool

	// KeyMap-driven sequence state machine (replaces hardcoded gPending)
	pendingAction string

	// Event channel for bridging executor events to Bubble Tea
	eventCh chan tea.Msg

	// Exit when done
	exitOnComplete bool

	// Modal state
	activeModal modalType
	lastCtrlCAt time.Time

	// Prompt read modal state
	promptModalFile    string // filename (e.g. "PROMPT_BUILD.md")
	promptModalContent string // cached file content
	promptModalScroll  int    // scroll offset within modal

	// Quit state
	quitting bool
}

func NewModel(sess model.Session, cfg config.Config, promptContent string, th theme.Theme, compactView bool, exitOnComplete bool, exec executor.Executor) Model {
	sessionPtr := &sess
	ctrl := session.NewController(sessionPtr, cfg, nil)

	// Use current working directory for prompt file scanning
	workDir, _ := os.Getwd()

	return Model{
		controller:      ctrl,
		exec:            exec,
		config:          cfg,
		promptContent:   promptContent,
		theme:           th,
		compactView:     compactView,
		leftPaneVisible: true,
		lineNumbers:     cfg.LineNumbers,
		exitOnComplete:  exitOnComplete,
		eventCh:         make(chan tea.Msg, 100),
		focusedPane:     iterationsPane,
		iterList:        NewIterList(),
		promptList:      NewPromptList(workDir),
		timeline:        NewTimeline(),
		workDir:         workDir,
	}
}

// Session returns a pointer to the controller's session for read access.
func (m *Model) Session() *model.Session {
	return m.controller.Session
}

func (m *Model) Init() tea.Cmd {
	if m.controller.Session.Mode == "idle" {
		// Idle mode: no auto-start, just tick for prompt file scanning
		return tickCmd()
	}

	// Non-idle mode: create the first Run and start iterating
	promptName := strings.ToUpper(m.controller.Session.Mode)
	m.controller.StartRun(
		promptName,
		m.controller.Session.PromptFile,
		m.controller.Session.MaxIterations,
	)

	return tea.Batch(
		m.spawnIteration(),
		tickCmd(),
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if msg.Width < 80 {
			m.leftPaneVisible = false
			if isLeftPane(m.focusedPane) {
				m.focusedPane = rightPane
			}
		} else {
			m.leftPaneVisible = true
		}
		return m, nil

	case tickMsg:
		// Rescan prompt files on each tick (1s interval)
		m.promptList.ScanFiles(m.workDir)
		return m, tickCmd()

	case assistantBatchMsg:
		m.controller.ProcessAssistantBatch(msg.Events)
		if idx := m.controller.RunningIterationIdx(); idx >= 0 {
			if m.iterList.SelectedIndex() == idx {
				iter := &m.controller.Session.Iterations[idx]
				m.timeline.OnNewItems(m.timelineProps(iter.Items))
			}
		}
		return m, waitForEvent(m.eventCh)

	case usageMsg:
		m.controller.ProcessUsage(msg.UsageEvent)
		return m, waitForEvent(m.eventCh)

	case toolResultMsg:
		group := m.controller.ProcessToolResult(msg.ToolResultEvent)
		if group != nil && group.Status() != model.ToolCallRunning && !group.ManualToggle {
			idx := m.controller.RunningIterationIdx()
			if idx >= 0 {
				iter := &m.controller.Session.Iterations[idx]
				for i, item := range iter.Items {
					if item == group {
						cursorOnGroup := m.isCursorOnGroup(iter.Items, i)
						if m.iterList.SelectedIndex() != idx || !cursorOnGroup {
							group.Expanded = false
						}
						break
					}
				}
			}
		}
		return m, waitForEvent(m.eventCh)

	case iterationEndMsg:
		return m, waitForEvent(m.eventCh)

	case subprocessExitMsg:
		m.controller.CompleteIteration(msg.err)

		if !m.quitting && m.controller.ShouldStartNext() {
			return m, m.spawnIteration()
		}
		if m.exitOnComplete {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil

	case promptEditorDoneMsg:
		// Editor exited; rescan prompt files in case the file was modified
		m.promptList.ScanFiles(m.workDir)
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)
	}

	return m, nil
}

const ctrlCForceQuitWindow = 500 * time.Millisecond

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	km := &m.config.KeyMap

	// ctrl+c: double within 500ms force-quits, single shows quit modal.
	if key == "ctrl+c" {
		now := time.Now()
		if !m.lastCtrlCAt.IsZero() && now.Sub(m.lastCtrlCAt) < ctrlCForceQuitWindow {
			// Double ctrl+c: force quit immediately.
			m.quitting = true
			_ = m.exec.Kill()
			return m, tea.Quit
		}
		m.lastCtrlCAt = now
		m.activeModal = modalQuitConfirm
		return m, nil
	}

	// When a modal is active, route keys to the modal handler.
	if m.activeModal != modalNone {
		return m.handleModalKey(msg)
	}

	// Resolve the key through the KeyMap, handling sequence state.
	action, newPending := km.Resolve(key, m.pendingAction)
	m.pendingAction = newPending

	// If we just set a pending action (sequence started), wait for next key.
	if action == "" && newPending != "" {
		return m, nil
	}

	// If no action matched via KeyMap, check arrow key alternates.
	// Arrow keys are always active alongside their letter equivalents per spec.
	if action == "" {
		if alt := config.HasAlternateArrowKey(key); alt != "" {
			action, _ = km.Resolve(alt, "")
		}
	}

	// Also handle "home" as an alternate for jump_top (always active).
	if action == "" && key == "home" {
		action = config.ActionJumpTop
	}
	// "end" as an alternate for jump_bottom (always active).
	if action == "" && key == "end" {
		action = config.ActionJumpBottom
	}
	// pgup/pgdn — delegate directly to focused component.
	if action == "" && key == "pgup" {
		action = "page_up"
	}
	if action == "" && key == "pgdown" {
		action = "page_down"
	}

	// Intercept digit keys for count buffer when right pane is focused
	// and no action was resolved yet.
	if action == "" && newPending == "" && m.focusedPane == rightPane && !m.timeline.InSubScroll() {
		if len(key) == 1 && key[0] >= '0' && key[0] <= '9' {
			m.timeline.AccumulateDigit(rune(key[0]))
			return m, nil
		}
	}

	// Any resolved action (or unrecognized key) clears the count buffer,
	// except for move_down/move_up which consume it.
	clearCount := true

	// When in sub-scroll mode, only allow specific actions.
	if m.timeline.InSubScroll() {
		m.timeline.ClearCount()
		switch action {
		case config.ActionQuit:
			m.activeModal = modalQuitConfirm
		case config.ActionHelp:
			m.activeModal = modalHelp
		case config.ActionEscape:
			m.timeline.ExitSubScroll()
		case config.ActionMoveDown, config.ActionMoveUp,
			config.ActionJumpTop, config.ActionJumpBottom,
			config.ActionExpand:
			m.timeline.HandleAction(action, m.currentTimelineProps())
		}
		// All other keys are ignored in sub-scroll mode.
		return m, nil
	}

	switch action {
	case config.ActionQuit:
		m.timeline.ClearCount()
		m.activeModal = modalQuitConfirm
		return m, nil

	case config.ActionHelp:
		m.timeline.ClearCount()
		m.activeModal = modalHelp
		return m, nil

	case config.ActionToggleLeftPane:
		m.leftPaneVisible = !m.leftPaneVisible
		if !m.leftPaneVisible && isLeftPane(m.focusedPane) {
			m.focusedPane = rightPane
		}

	case config.ActionFocusToggle:
		if !m.leftPaneVisible {
			// Can't toggle to hidden pane.
			break
		}
		// Cycle: iterations → prompts → timeline → iterations
		switch m.focusedPane {
		case iterationsPane:
			m.focusedPane = promptsPane
		case promptsPane:
			m.focusedPane = rightPane
		case rightPane:
			m.focusedPane = iterationsPane
		}

	case config.ActionFocusLeft:
		if m.leftPaneVisible {
			// h from right pane goes to last focused left sub-pane
			if m.focusedPane == rightPane {
				m.focusedPane = iterationsPane
			}
		}

	case config.ActionFocusRight:
		m.focusedPane = rightPane

	case config.ActionToggleView:
		m.compactView = !m.compactView

	case config.ActionToggleLineNumbers:
		m.lineNumbers = !m.lineNumbers

	case config.ActionExpand:
		switch m.focusedPane {
		case iterationsPane:
			m.focusedPane = rightPane
		case promptsPane:
			if f := m.promptList.SelectedFile(); f != "" {
				content, err := ReadFileContent(m.workDir, f)
				if err == nil {
					m.promptModalFile = f
					m.promptModalContent = content
					m.promptModalScroll = 0
					m.activeModal = modalPromptRead
				}
			}
		default:
			m.timeline.HandleAction("expand", m.currentTimelineProps())
		}

	case config.ActionJumpTop:
		switch m.focusedPane {
		case iterationsPane:
			m.iterList.JumpToTop()
			m.timeline.ResetPosition()
		case promptsPane:
			m.promptList.HandleAction("jump_top", m.promptListProps())
		default:
			m.timeline.JumpToTop()
		}

	case config.ActionJumpBottom:
		switch m.focusedPane {
		case iterationsPane:
			m.iterList.JumpToBottom(len(m.controller.Session.Iterations), m.iterListHeight())
			m.timeline.ResetPosition()
		case promptsPane:
			m.promptList.HandleAction("jump_bottom", m.promptListProps())
		default:
			m.timeline.JumpToBottom(m.currentTimelineProps())
		}

	case config.ActionMoveDown, config.ActionMoveUp:
		switch m.focusedPane {
		case iterationsPane:
			oldCursor := m.iterList.Cursor
			m.iterList.HandleAction(action, m.iterListProps())
			if m.iterList.Cursor != oldCursor {
				m.timeline.ResetPosition()
			}
		case promptsPane:
			m.promptList.HandleAction(action, m.promptListProps())
		default:
			count := m.timeline.ConsumeCount()
			clearCount = false // ConsumeCount already cleared
			m.timeline.HandleActionWithCount(action, count, m.currentTimelineProps())
		}

	case "page_up", "page_down":
		switch m.focusedPane {
		case iterationsPane:
			oldCursor := m.iterList.Cursor
			m.iterList.HandleAction(action, m.iterListProps())
			if m.iterList.Cursor != oldCursor {
				m.timeline.ResetPosition()
			}
		case promptsPane:
			m.promptList.HandleAction(action, m.promptListProps())
		default:
			m.timeline.HandleAction(action, m.currentTimelineProps())
		}
	}

	if clearCount {
		m.timeline.ClearCount()
	}

	return m, nil
}

// handleModalKey routes key presses to the active modal.
func (m *Model) handleModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if m.activeModal == modalQuitConfirm {
		switch key {
		case "y":
			m.quitting = true
			_ = m.exec.Kill()
			return m, tea.Quit
		case "n", "esc":
			m.activeModal = modalNone
			return m, nil
		}
		// All other keys are ignored in quit modal.
		return m, nil
	}

	if m.activeModal == modalHelp {
		// Any key dismisses the help modal.
		m.activeModal = modalNone
		return m, nil
	}

	if m.activeModal == modalPromptRead {
		return m.handlePromptModalKey(key)
	}

	return m, nil
}

// handlePromptModalKey handles key presses in the prompt read modal.
func (m *Model) handlePromptModalKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.activeModal = modalNone
		return m, nil
	case "e":
		// Launch $EDITOR for the prompt file
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}
		filePath := m.workDir + "/" + m.promptModalFile
		m.activeModal = modalNone
		c := tea.ExecProcess(exec.Command(editor, filePath), func(err error) tea.Msg {
			return promptEditorDoneMsg{err: err}
		})
		return m, c
	case "j", "down":
		maxScroll := PromptModalMaxScroll(m.promptModalContent, m.height)
		if m.promptModalScroll < maxScroll {
			m.promptModalScroll++
		}
	case "k", "up":
		if m.promptModalScroll > 0 {
			m.promptModalScroll--
		}
	case "pgdown":
		maxScroll := PromptModalMaxScroll(m.promptModalContent, m.height)
		m.promptModalScroll += 10
		if m.promptModalScroll > maxScroll {
			m.promptModalScroll = maxScroll
		}
	case "pgup":
		m.promptModalScroll -= 10
		if m.promptModalScroll < 0 {
			m.promptModalScroll = 0
		}
	}
	return m, nil
}

const mouseScrollLines = 3

func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Ignore events on the header row
	paneRow := msg.Y - 1
	if paneRow < 0 {
		return m, nil
	}

	// Determine target pane by X coordinate
	leftWidth := m.leftPaneWidth()
	isLeft := leftWidth > 0 && msg.X < leftWidth

	// For left pane, determine if the click is in the iterations or prompts section
	targetPane := rightPane
	if isLeft {
		paneHeight := m.rightPaneHeight()
		if IsInPromptSection(paneRow, paneHeight) {
			targetPane = promptsPane
		} else {
			targetPane = iterationsPane
		}
	}

	switch {
	case msg.Button == tea.MouseButtonWheelUp:
		m.focusedPane = targetPane
		switch targetPane {
		case iterationsPane:
			count := len(m.controller.Session.Iterations)
			m.iterList.ScrollBy(-mouseScrollLines, count, m.iterListHeight())
		case promptsPane:
			m.promptList.ScrollBy(-mouseScrollLines)
		default:
			m.timeline.ScrollBy(-mouseScrollLines, m.currentTimelineProps())
		}

	case msg.Button == tea.MouseButtonWheelDown:
		m.focusedPane = targetPane
		switch targetPane {
		case iterationsPane:
			count := len(m.controller.Session.Iterations)
			m.iterList.ScrollBy(mouseScrollLines, count, m.iterListHeight())
		case promptsPane:
			m.promptList.ScrollBy(mouseScrollLines)
		default:
			m.timeline.ScrollBy(mouseScrollLines, m.currentTimelineProps())
		}

	case msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress:
		m.focusedPane = targetPane
		switch targetPane {
		case iterationsPane:
			count := len(m.controller.Session.Iterations)
			oldCursor := m.iterList.Cursor
			if m.iterList.ClickRow(paneRow, count, m.iterListHeight()) {
				if m.iterList.Cursor != oldCursor {
					m.timeline.ResetPosition()
				}
			}
		case promptsPane:
			paneHeight := m.rightPaneHeight()
			promptRow := PromptSectionRow(paneRow, paneHeight)
			m.promptList.ClickRow(promptRow)
		default:
			props := m.currentTimelineProps()
			if m.timeline.InSubScroll() {
				m.timeline.ClickRowSubScroll(paneRow, props)
			} else {
				m.timeline.ClickRow(paneRow, props)
			}
		}
	}

	return m, nil
}

func (m *Model) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 || m.height == 0 {
		return "Starting..."
	}

	header := RenderHeader(m.headerProps())

	paneHeight := m.height - 1 // subtract 1 for the header line
	leftWidth := m.leftPaneWidth()
	rightWidth := m.rightPaneWidth()

	right := m.timeline.View(TimelineProps{
		Items:       m.selectedItems(),
		Width:       rightWidth,
		Height:      paneHeight,
		Focused:     m.focusedPane == rightPane,
		CompactView: m.compactView,
		LineNumbers: m.lineNumbers,
		Theme:       m.theme,
	})

	var panes string
	if leftWidth > 0 {
		iterHeight := m.iterListHeight()
		promptHeight := PromptListTotalHeight()

		iterView := m.iterList.View(IterListProps{
			Iterations: m.controller.Session.Iterations,
			Width:      leftWidth,
			Height:     iterHeight,
			Focused:    m.focusedPane == iterationsPane,
			Theme:      m.theme,
		})

		// Horizontal divider between iterations and prompts
		divider := lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.theme.ForegroundDim)).
			Width(leftWidth).
			Render(strings.Repeat("─", leftWidth))

		promptView := m.promptList.View(PromptListProps{
			Width:   leftWidth,
			Height:  promptHeight,
			Focused: m.focusedPane == promptsPane,
			Theme:   m.theme,
		})

		left := lipgloss.JoinVertical(lipgloss.Left, iterView, divider, promptView)

		sepLines := make([]string, paneHeight)
		for i := range sepLines {
			sepLines[i] = "│"
		}
		separator := lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.theme.ForegroundDim)).
			Render(strings.Join(sepLines, "\n"))

		panes = lipgloss.JoinHorizontal(lipgloss.Top, left, separator, right)
	} else {
		panes = right
	}

	view := header + "\n" + panes

	// Render modal overlay if active.
	switch m.activeModal {
	case modalQuitConfirm:
		view = RenderQuitConfirmModal(m.width, m.height, m.theme)
	case modalHelp:
		view = RenderHelpModal(m.width, m.height, m.theme, &m.config.KeyMap)
	case modalPromptRead:
		view = RenderPromptReadModal(PromptModalProps{
			Filename: m.promptModalFile,
			Content:  m.promptModalContent,
			Scroll:   m.promptModalScroll,
			Width:    m.width,
			Height:   m.height,
			Theme:    m.theme,
		})
	}

	return view
}

// headerProps builds the HeaderProps from current model state.
func (m *Model) headerProps() HeaderProps {
	sess := m.controller.Session
	inputTokens := sess.InputTokens + sess.CacheReadTokens + sess.CacheCreationTokens

	contextPercent := -1
	if m.controller.HasKnownModel() && m.controller.LastModel() != "" {
		if pricing, ok := m.config.Pricing[m.controller.LastModel()]; ok && pricing.ContextWindow > 0 {
			contextPercent = int((sess.LastInputTokens + sess.LastCacheReadTokens) * 100 / int64(pricing.ContextWindow))
		}
	}

	// Determine session status from current state
	var sessionStatus model.IterationStatus
	switch {
	case m.controller.RunningIterationIdx() >= 0:
		sessionStatus = model.IterationRunning
	case len(sess.Iterations) > 0 && sess.Iterations[len(sess.Iterations)-1].Status == model.IterationFailed:
		sessionStatus = model.IterationFailed
	default:
		sessionStatus = model.IterationCompleted
	}

	return HeaderProps{
		SessionDuration: time.Since(sess.StartTime),
		InputTokens:     inputTokens,
		OutputTokens:    sess.OutputTokens,
		ContextPercent:  contextPercent,
		TotalCost:       sess.TotalCost,
		HasKnownModel:   m.controller.HasKnownModel(),
		RateLimit:       sess.RateLimit,
		IterationCount:  len(sess.Iterations),
		MaxIterations:   sess.MaxIterations,
		SessionStatus:   sessionStatus,
		Width:           m.width,
		Theme:           m.theme,
	}
}

// iterListHeight returns the height available for the iteration list,
// accounting for the prompt list section and divider at the bottom.
func (m *Model) iterListHeight() int {
	paneHeight := m.rightPaneHeight()
	// Subtract prompt section (5 rows) + divider (1 row)
	h := paneHeight - PromptListTotalHeight() - 1
	if h < 1 {
		h = 1
	}
	return h
}

// iterListProps builds IterListProps from current state.
func (m *Model) iterListProps() IterListProps {
	return IterListProps{
		Iterations: m.controller.Session.Iterations,
		Width:      m.leftPaneWidth(),
		Height:     m.iterListHeight(),
		Focused:    m.focusedPane == iterationsPane,
		Theme:      m.theme,
	}
}

// promptListProps builds PromptListProps from current state.
func (m *Model) promptListProps() PromptListProps {
	return PromptListProps{
		Width:   m.leftPaneWidth(),
		Height:  PromptListTotalHeight(),
		Focused: m.focusedPane == promptsPane,
		Theme:   m.theme,
	}
}

// currentTimelineProps builds TimelineProps for the currently selected iteration.
func (m *Model) currentTimelineProps() TimelineProps {
	return m.timelineProps(m.selectedItems())
}

// timelineProps builds TimelineProps for a given set of items.
func (m *Model) timelineProps(items []model.TimelineItem) TimelineProps {
	return TimelineProps{
		Items:       items,
		Width:       m.rightPaneWidth(),
		Height:      m.rightPaneHeight(),
		Focused:     m.focusedPane == rightPane,
		CompactView: m.compactView,
		LineNumbers: m.lineNumbers,
		Theme:       m.theme,
	}
}

// Subprocess management — delegates to executor

func (m *Model) spawnIteration() tea.Cmd {
	m.controller.StartIteration()
	m.iterList.OnNewIteration(len(m.controller.Session.Iterations), m.iterListHeight())
	if m.iterList.AutoFollow.Following() {
		m.timeline.ResetPosition()
	}

	ch := m.eventCh

	eventCh, err := m.exec.Start(context.Background(), m.promptContent)
	if err != nil {
		return func() tea.Msg {
			return subprocessExitMsg{err: err}
		}
	}

	go func() {
		for evt := range eventCh {
			switch e := evt.(type) {
			case session.AssistantBatchEvent:
				ch <- assistantBatchMsg{e}
			case session.UsageEvent:
				ch <- usageMsg{e}
			case session.ToolResultEvent:
				ch <- toolResultMsg{e}
			case session.IterationEndEvent:
				ch <- iterationEndMsg{}
			case session.SubprocessExitEvent:
				ch <- subprocessExitMsg{err: e.Err}
			}
		}
	}()

	return waitForEvent(ch)
}

func waitForEvent(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

// Helpers

const leftPaneFixedWidth = 32

func (m *Model) leftPaneWidth() int {
	if m.leftPaneVisible {
		return leftPaneFixedWidth
	}
	return 0
}

func (m *Model) rightPaneWidth() int {
	lpw := m.leftPaneWidth()
	if lpw == 0 {
		return m.width
	}
	rw := m.width - lpw - 1 // 1 for separator
	if rw < 20 {
		rw = 20
	}
	return rw
}

func (m *Model) rightPaneHeight() int {
	if m.height > 1 {
		return m.height - 1
	}
	return 20
}

// selectedItems returns the timeline items for the currently selected iteration.
func (m *Model) selectedItems() []model.TimelineItem {
	idx := m.iterList.SelectedIndex()
	if idx >= len(m.controller.Session.Iterations) {
		return nil
	}
	return m.controller.Session.Iterations[idx].Items
}

// isCursorOnGroup returns true if the timeline cursor is on the group header or any of its children.
func (m *Model) isCursorOnGroup(items []model.TimelineItem, itemIdx int) bool {
	curItemIdx, _ := FlatToItem(items, m.timeline.Cursor)
	return curItemIdx == itemIdx
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
