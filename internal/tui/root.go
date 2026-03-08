package tui

import (
	"context"
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

// Pane focus

type paneID int

const (
	leftPane paneID = iota
	rightPane
)

// Model is the Bubble Tea model for the TUI.
type Model struct {
	controller    *session.Controller
	exec          executor.Executor
	config        config.Config
	promptContent string
	theme         theme.Theme

	// Sub-components
	iterList IterList
	timeline Timeline

	// UI state
	width, height int
	focusedPane   paneID
	compactView   bool

	// KeyMap-driven sequence state machine (replaces hardcoded gPending)
	pendingAction string

	// Event channel for bridging executor events to Bubble Tea
	eventCh chan tea.Msg

	// Exit when done
	exitOnComplete bool

	// Modal state
	activeModal modalType
	lastCtrlCAt time.Time

	// Quit state
	quitting bool
}

func NewModel(sess model.Session, cfg config.Config, promptContent string, th theme.Theme, compactView bool, exitOnComplete bool, exec executor.Executor) Model {
	sessionPtr := &sess
	ctrl := session.NewController(sessionPtr, cfg, nil)
	return Model{
		controller:     ctrl,
		exec:           exec,
		config:         cfg,
		promptContent:  promptContent,
		theme:          th,
		compactView:    compactView,
		exitOnComplete: exitOnComplete,
		eventCh:        make(chan tea.Msg, 100),
		focusedPane:    leftPane,
		iterList:       NewIterList(),
		timeline:       NewTimeline(),
	}
}

// Session returns a pointer to the controller's session for read access.
func (m *Model) Session() *model.Session {
	return m.controller.Session
}

func (m *Model) Init() tea.Cmd {
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
		return m, nil

	case tickMsg:
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

	switch action {
	case config.ActionQuit:
		m.activeModal = modalQuitConfirm
		return m, nil

	case config.ActionFocusToggle:
		if m.focusedPane == leftPane {
			m.focusedPane = rightPane
		} else {
			m.focusedPane = leftPane
		}

	case config.ActionFocusLeft:
		m.focusedPane = leftPane

	case config.ActionFocusRight:
		m.focusedPane = rightPane

	case config.ActionToggleView:
		m.compactView = !m.compactView

	case config.ActionExpand:
		if m.focusedPane == leftPane {
			m.focusedPane = rightPane
		} else {
			m.timeline.HandleAction("expand", m.currentTimelineProps())
		}

	case config.ActionJumpTop:
		if m.focusedPane == leftPane {
			m.iterList.JumpToTop()
			m.timeline.ResetPosition()
		} else {
			m.timeline.JumpToTop()
		}

	case config.ActionJumpBottom:
		if m.focusedPane == leftPane {
			m.iterList.JumpToBottom(len(m.controller.Session.Iterations), m.rightPaneHeight())
			m.timeline.ResetPosition()
		} else {
			m.timeline.JumpToBottom(m.currentTimelineProps())
		}

	case config.ActionMoveDown, config.ActionMoveUp,
		"page_up", "page_down":
		if m.focusedPane == leftPane {
			oldCursor := m.iterList.Cursor
			m.iterList.HandleAction(action, m.iterListProps())
			if m.iterList.Cursor != oldCursor {
				m.timeline.ResetPosition()
			}
		} else {
			m.timeline.HandleAction(action, m.currentTimelineProps())
		}
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
	}

	// All other keys are ignored while a modal is open.
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
	leftWidth := 32
	targetPane := leftPane
	if msg.X >= leftWidth {
		targetPane = rightPane
	}

	switch {
	case msg.Button == tea.MouseButtonWheelUp:
		m.focusedPane = targetPane
		if targetPane == leftPane {
			count := len(m.controller.Session.Iterations)
			m.iterList.ScrollBy(-mouseScrollLines, count, m.rightPaneHeight())
		} else {
			m.timeline.ScrollBy(-mouseScrollLines, m.currentTimelineProps())
		}

	case msg.Button == tea.MouseButtonWheelDown:
		m.focusedPane = targetPane
		if targetPane == leftPane {
			count := len(m.controller.Session.Iterations)
			m.iterList.ScrollBy(mouseScrollLines, count, m.rightPaneHeight())
		} else {
			m.timeline.ScrollBy(mouseScrollLines, m.currentTimelineProps())
		}

	case msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress:
		m.focusedPane = targetPane
		if targetPane == leftPane {
			count := len(m.controller.Session.Iterations)
			oldCursor := m.iterList.Cursor
			if m.iterList.ClickRow(paneRow, count, m.rightPaneHeight()) {
				if m.iterList.Cursor != oldCursor {
					m.timeline.ResetPosition()
				}
			}
		} else {
			m.timeline.ClickRow(paneRow, m.currentTimelineProps())
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

	leftWidth := 32
	rightWidth := m.width - leftWidth - 1 // 1 for separator
	if rightWidth < 20 {
		rightWidth = 20
	}

	left := m.iterList.View(IterListProps{
		Iterations: m.controller.Session.Iterations,
		Width:      leftWidth,
		Height:     paneHeight,
		Focused:    m.focusedPane == leftPane,
		Theme:      m.theme,
	})

	right := m.timeline.View(TimelineProps{
		Items:       m.selectedItems(),
		Width:       rightWidth,
		Height:      paneHeight,
		Focused:     m.focusedPane == rightPane,
		CompactView: m.compactView,
		Theme:       m.theme,
	})

	sepLines := make([]string, paneHeight)
	for i := range sepLines {
		sepLines[i] = "│"
	}
	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.ForegroundDim)).
		Render(strings.Join(sepLines, "\n"))

	panes := lipgloss.JoinHorizontal(lipgloss.Top, left, separator, right)
	view := header + "\n" + panes

	// Render modal overlay if active.
	if m.activeModal == modalQuitConfirm {
		view = RenderQuitConfirmModal(m.width, m.height, m.theme)
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
		IterationCount:  len(sess.Iterations),
		MaxIterations:   sess.MaxIterations,
		SessionStatus:   sessionStatus,
		Width:           m.width,
		Theme:           m.theme,
	}
}

// iterListProps builds IterListProps from current state.
func (m *Model) iterListProps() IterListProps {
	return IterListProps{
		Iterations: m.controller.Session.Iterations,
		Width:      32,
		Height:     m.rightPaneHeight(),
		Focused:    m.focusedPane == leftPane,
		Theme:      m.theme,
	}
}

// currentTimelineProps builds TimelineProps for the currently selected iteration.
func (m *Model) currentTimelineProps() TimelineProps {
	return m.timelineProps(m.selectedItems())
}

// timelineProps builds TimelineProps for a given set of items.
func (m *Model) timelineProps(items []model.TimelineItem) TimelineProps {
	rightWidth := m.width - 32 - 1
	if rightWidth < 20 {
		rightWidth = 20
	}
	return TimelineProps{
		Items:       items,
		Width:       rightWidth,
		Height:      m.rightPaneHeight(),
		Focused:     m.focusedPane == rightPane,
		CompactView: m.compactView,
		Theme:       m.theme,
	}
}

// Subprocess management — delegates to executor

func (m *Model) spawnIteration() tea.Cmd {
	m.controller.StartIteration()
	m.iterList.OnNewIteration(len(m.controller.Session.Iterations), m.rightPaneHeight())
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
