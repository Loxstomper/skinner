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

	// gg state machine
	gPending bool

	// Event channel for bridging executor events to Bubble Tea
	eventCh chan tea.Msg

	// Exit when done
	exitOnComplete bool

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
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// gg state machine: if gPending and key is "g", jump to top
	if m.gPending {
		m.gPending = false
		if key == "g" {
			if m.focusedPane == leftPane {
				m.iterList.JumpToTop()
				m.timeline.ResetPosition()
			} else {
				m.timeline.JumpToTop()
			}
			return m, nil
		}
		// Any other key: clear gPending, fall through to normal handling
	}

	switch key {
	case "q", "ctrl+c":
		m.quitting = true
		_ = m.exec.Kill()
		return m, tea.Quit

	case "tab":
		if m.focusedPane == leftPane {
			m.focusedPane = rightPane
		} else {
			m.focusedPane = leftPane
		}

	case "h", "left":
		m.focusedPane = leftPane

	case "l", "right":
		m.focusedPane = rightPane

	case "g":
		m.gPending = true

	case "v":
		m.compactView = !m.compactView

	case "enter":
		if m.focusedPane == leftPane {
			m.focusedPane = rightPane
		} else {
			m.timeline.Update(msg, m.currentTimelineProps())
		}

	case "G", "end":
		if m.focusedPane == leftPane {
			m.iterList.JumpToBottom(len(m.controller.Session.Iterations))
			m.timeline.ResetPosition()
		} else {
			m.timeline.JumpToBottom(m.currentTimelineProps())
		}

	case "home":
		if m.focusedPane == leftPane {
			m.iterList.JumpToTop()
			m.timeline.ResetPosition()
		} else {
			m.timeline.JumpToTop()
		}

	default:
		// Delegate navigation keys to the focused component
		if m.focusedPane == leftPane {
			oldCursor := m.iterList.Cursor
			m.iterList.Update(msg, m.iterListProps())
			if m.iterList.Cursor != oldCursor {
				m.timeline.ResetPosition()
			}
		} else {
			m.timeline.Update(msg, m.currentTimelineProps())
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
	return header + "\n" + panes
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
	m.iterList.OnNewIteration(len(m.controller.Session.Iterations))
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
