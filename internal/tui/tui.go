package tui

import (
	"context"
	"fmt"
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

	// UI state
	selectedIter  int
	scrollOffset  int
	width, height int
	focusedPane   paneID
	rightCursor   int

	// View mode
	compactView bool

	// Auto-follow
	autoFollowLeft  AutoFollow
	autoFollowRight AutoFollow

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
		controller:      ctrl,
		exec:            exec,
		config:          cfg,
		promptContent:   promptContent,
		theme:           th,
		compactView:     compactView,
		exitOnComplete:  exitOnComplete,
		eventCh:         make(chan tea.Msg, 100),
		focusedPane:     leftPane,
		autoFollowLeft:  NewAutoFollow(),
		autoFollowRight: NewAutoFollow(),
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
			iter := &m.controller.Session.Iterations[idx]
			if m.selectedIter == idx && m.autoFollowRight.Following() {
				m.rightCursor = FlatCursorCount(iter.Items) - 1
				m.scrollToBottom()
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
				// Find the item index of this group
				for i, item := range iter.Items {
					if item == group {
						cursorOnGroup := m.isCursorOnGroup(i)
						if m.selectedIter != idx || !cursorOnGroup {
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
			m.jumpToTop()
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

	case "G", "end":
		m.jumpToBottom()

	case "home":
		m.jumpToTop()

	case "j", "down":
		if m.focusedPane == leftPane {
			if m.selectedIter < len(m.controller.Session.Iterations)-1 {
				m.selectedIter++
				m.rightCursor = 0
				m.scrollOffset = 0
			}
			m.autoFollowLeft.OnManualMove(m.selectedIter == len(m.controller.Session.Iterations)-1)
		} else {
			items := m.selectedItems()
			maxPos := FlatCursorCount(items) - 1
			if m.rightCursor < maxPos {
				m.rightCursor++
				m.ensureCursorVisible()
			}
			m.autoFollowRight.OnManualMove(m.rightCursor >= maxPos)
		}

	case "k", "up":
		if m.focusedPane == leftPane {
			if m.selectedIter > 0 {
				m.selectedIter--
				m.rightCursor = 0
				m.scrollOffset = 0
			}
			m.autoFollowLeft.OnManualMove(m.selectedIter == len(m.controller.Session.Iterations)-1)
		} else {
			if m.rightCursor > 0 {
				m.rightCursor--
				m.ensureCursorVisible()
			}
			m.autoFollowRight.OnManualMove(m.rightCursor >= FlatCursorCount(m.selectedItems())-1)
		}

	case "pgdown":
		if m.focusedPane == leftPane {
			m.selectedIter += m.height
			if m.selectedIter >= len(m.controller.Session.Iterations) {
				m.selectedIter = len(m.controller.Session.Iterations) - 1
			}
			if m.selectedIter < 0 {
				m.selectedIter = 0
			}
			m.rightCursor = 0
			m.scrollOffset = 0
			m.autoFollowLeft.OnManualMove(m.selectedIter == len(m.controller.Session.Iterations)-1)
		} else {
			m.scrollOffset += m.rightPaneHeight()
			m.clampScroll()
			total := TotalLines(m.selectedItems(), m.compactView)
			m.autoFollowRight.OnManualMove(m.scrollOffset+m.rightPaneHeight() >= total)
		}

	case "pgup":
		if m.focusedPane == leftPane {
			m.selectedIter -= m.height
			if m.selectedIter < 0 {
				m.selectedIter = 0
			}
			m.rightCursor = 0
			m.scrollOffset = 0
			m.autoFollowLeft.OnManualMove(m.selectedIter == len(m.controller.Session.Iterations)-1)
		} else {
			m.scrollOffset -= m.rightPaneHeight()
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}
			m.autoFollowRight.OnManualMove(false)
		}

	case "v":
		m.compactView = !m.compactView

	case "enter":
		if m.focusedPane == leftPane {
			m.focusedPane = rightPane
		} else if m.selectedIter < len(m.controller.Session.Iterations) {
			iter := &m.controller.Session.Iterations[m.selectedIter]
			itemIdx, childIdx := FlatToItem(iter.Items, m.rightCursor)
			if itemIdx < len(iter.Items) {
				switch it := iter.Items[itemIdx].(type) {
				case *model.TextBlock:
					it.Expanded = !it.Expanded
					m.ensureCursorVisible()
				case *model.ToolCallGroup:
					if childIdx == -1 {
						// On group header: toggle expand/collapse
						it.ManualToggle = true
						if it.Expanded {
							// Collapsing: move cursor to header position
							it.Expanded = false
							m.rightCursor = ItemToFlat(iter.Items, itemIdx)
						} else {
							it.Expanded = true
						}
						m.ensureCursorVisible()
					}
					// On child row: no-op
				}
			}
		}
	}

	return m, nil
}

func (m *Model) jumpToTop() {
	if m.focusedPane == leftPane {
		m.selectedIter = 0
		m.rightCursor = 0
		m.scrollOffset = 0
		m.autoFollowLeft.OnManualMove(m.selectedIter == len(m.controller.Session.Iterations)-1)
	} else {
		m.rightCursor = 0
		m.scrollOffset = 0
		m.autoFollowRight.OnManualMove(false)
	}
}

func (m *Model) jumpToBottom() {
	if m.focusedPane == leftPane {
		if len(m.controller.Session.Iterations) > 0 {
			m.selectedIter = len(m.controller.Session.Iterations) - 1
			m.rightCursor = 0
			m.scrollOffset = 0
		}
		m.autoFollowLeft.JumpToEnd()
	} else {
		maxPos := FlatCursorCount(m.selectedItems())
		if maxPos > 0 {
			m.rightCursor = maxPos - 1
			m.scrollToBottom()
		}
		m.autoFollowRight.JumpToEnd()
	}
}

func (m *Model) viewHeader() string {
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.ForegroundDim))

	// Build centre content: duration, tokens, context %, cost
	dur := FormatDurationValue(time.Since(m.controller.Session.StartTime))
	inputTokens := m.controller.Session.InputTokens + m.controller.Session.CacheReadTokens + m.controller.Session.CacheCreationTokens
	outputTokens := m.controller.Session.OutputTokens

	centreText := fmt.Sprintf("⏱ %s   ↑%s ↓%s tokens", dur, FormatTokens(inputTokens), FormatTokens(outputTokens))
	centreRendered := dim.Render(centreText)

	// Context window percentage
	if m.controller.HasKnownModel() && m.controller.LastModel() != "" {
		if pricing, ok := m.config.Pricing[m.controller.LastModel()]; ok && pricing.ContextWindow > 0 {
			pct := int((m.controller.Session.LastInputTokens + m.controller.Session.LastCacheReadTokens) * 100 / int64(pricing.ContextWindow))
			ctxText := fmt.Sprintf("   ctx %d%%", pct)
			var ctxColor string
			switch {
			case pct >= 90:
				ctxColor = m.theme.StatusError
			case pct >= 70:
				ctxColor = m.theme.StatusRunning
			default:
				ctxColor = m.theme.ForegroundDim
			}
			centreRendered += lipgloss.NewStyle().Foreground(lipgloss.Color(ctxColor)).Render(ctxText)
		}
	}

	// Cost
	if m.controller.HasKnownModel() {
		centreRendered += dim.Render(fmt.Sprintf("   ~$%.2f", m.controller.Session.TotalCost))
	}

	// Right side: iteration progress + status icon
	iterCount := len(m.controller.Session.Iterations)
	if iterCount == 0 {
		iterCount = 1
	}
	var iterText string
	if m.controller.Session.MaxIterations > 0 {
		iterText = fmt.Sprintf("Iter %d/%d", iterCount, m.controller.Session.MaxIterations)
	} else {
		iterText = fmt.Sprintf("Iter %d", iterCount)
	}

	var statusIcon, statusColor string
	if idx := m.controller.RunningIterationIdx(); idx >= 0 {
		statusIcon = "⟳"
		statusColor = m.theme.StatusRunning
	} else if iterCount > 0 {
		lastIter := m.controller.Session.Iterations[len(m.controller.Session.Iterations)-1]
		if lastIter.Status == model.IterationFailed {
			statusIcon = "✗"
			statusColor = m.theme.StatusError
		} else {
			statusIcon = "✓"
			statusColor = m.theme.StatusSuccess
		}
	}

	styledStatusIcon := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render(statusIcon)
	rightRendered := dim.Render(iterText+" ") + styledStatusIcon + dim.Render(" ")

	// Centre the stats in the space to the left of the right-aligned iteration indicator
	centreWidth := lipgloss.Width(centreRendered)
	rightWidth := lipgloss.Width(rightRendered)
	availableWidth := m.width - rightWidth
	leftPad := (availableWidth - centreWidth) / 2
	if leftPad < 1 {
		leftPad = 1
	}
	gap := m.width - leftPad - centreWidth - rightWidth
	if gap < 0 {
		gap = 0
	}

	return strings.Repeat(" ", leftPad) + centreRendered + strings.Repeat(" ", gap) + rightRendered
}

func (m *Model) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 || m.height == 0 {
		return "Starting..."
	}

	header := m.viewHeader()

	paneHeight := m.height - 1 // subtract 1 for the header line

	leftWidth := 32
	rightWidth := m.width - leftWidth - 1 // 1 for separator
	if rightWidth < 20 {
		rightWidth = 20
	}

	left := m.renderLeftPane(leftWidth, paneHeight)
	right := m.renderRightPane(rightWidth, paneHeight)

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

// Left pane: iteration list

func (m *Model) renderLeftPane(width, height int) string {
	style := lipgloss.NewStyle().Width(width).Height(height)

	highlight := lipgloss.NewStyle().Background(lipgloss.Color(m.theme.Highlight))

	var lines []string
	for i, iter := range m.controller.Session.Iterations {
		var statusIcon string
		var statusColor, iterColor string
		switch iter.Status {
		case model.IterationRunning:
			statusIcon = "⟳"
			statusColor = m.theme.StatusRunning
			iterColor = m.theme.IterRunning
		case model.IterationCompleted:
			statusIcon = "✓"
			statusColor = m.theme.StatusSuccess
			iterColor = m.theme.IterSuccess
		case model.IterationFailed:
			statusIcon = "✗"
			statusColor = m.theme.StatusError
			iterColor = m.theme.IterError
		}

		dur := FormatDuration(iter.Duration, iter.Status == model.IterationRunning)
		callCount := iter.ToolCallCount()

		styledIcon := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render(statusIcon)
		iterText := fmt.Sprintf("  Iter %d  ", iter.Index+1)
		metaText := fmt.Sprintf("  (%d calls, %s)", callCount, dur)

		iterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(iterColor))
		line := iterStyle.Render(iterText) + styledIcon + iterStyle.Render(metaText)

		if i == m.selectedIter {
			displayWidth := lipgloss.Width(line)
			if displayWidth < width {
				line += strings.Repeat(" ", width-displayWidth)
			}
			line = highlight.Render(line)
		}
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	return style.Render(content)
}

// Right pane: message timeline

type renderedLine struct {
	text    string
	flatIdx int // flat cursor position (-1 for continuation lines of text blocks)
}

func (m *Model) renderRightPane(width, height int) string {
	style := lipgloss.NewStyle().Width(width).Height(height)

	if m.selectedIter >= len(m.controller.Session.Iterations) {
		return style.Render("")
	}

	iter := m.controller.Session.Iterations[m.selectedIter]
	items := iter.Items

	if len(items) == 0 {
		return style.Render("  No activity yet...")
	}

	iconWidth := 2 // icon + space
	nameWidth := 6
	durWidth := 8

	var summaryWidth, childSummaryWidth int
	if m.compactView {
		summaryWidth = width - iconWidth - durWidth - 5
		if summaryWidth < 10 {
			summaryWidth = 10
		}
		// Child rows have 2 extra indent spaces
		childSummaryWidth = summaryWidth - 2
		if childSummaryWidth < 10 {
			childSummaryWidth = 10
		}
	} else {
		summaryWidth = width - iconWidth - nameWidth - durWidth - 7
		if summaryWidth < 10 {
			summaryWidth = 10
		}
		childSummaryWidth = summaryWidth - 2
		if childSummaryWidth < 10 {
			childSummaryWidth = 10
		}
	}

	var lines []renderedLine
	flatPos := 0
	for _, item := range items {
		switch it := item.(type) {
		case *model.TextBlock:
			textLines := m.renderTextBlockLines(it, width)
			for _, l := range textLines {
				lines = append(lines, renderedLine{text: l, flatIdx: flatPos})
			}
			flatPos++
		case *model.ToolCall:
			l := m.renderToolCallLine(it, nameWidth, summaryWidth, durWidth)
			lines = append(lines, renderedLine{text: l, flatIdx: flatPos})
			flatPos++
		case *model.ToolCallGroup:
			l := m.renderGroupHeaderLine(it, nameWidth, summaryWidth, durWidth)
			lines = append(lines, renderedLine{text: l, flatIdx: flatPos})
			flatPos++
			if it.Expanded {
				for ci := range it.Children {
					child := it.Children[ci]
					cl := m.renderToolCallLine(child, nameWidth, childSummaryWidth, durWidth)
					// Prepend 2 extra spaces for child indent
					cl = "  " + cl
					lines = append(lines, renderedLine{text: cl, flatIdx: flatPos})
					flatPos++
				}
			}
		}
	}

	return m.renderRightPaneWithLines(lines, width, height)
}

func (m *Model) renderRightPaneWithLines(lines []renderedLine, width, height int) string {
	style := lipgloss.NewStyle().Width(width).Height(height)

	// Apply scroll
	start := m.scrollOffset
	if start >= len(lines) {
		start = len(lines) - 1
	}
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > len(lines) {
		end = len(lines)
	}

	visible := lines[start:end]

	highlight := lipgloss.NewStyle().Background(lipgloss.Color(m.theme.Highlight))

	var rendered []string
	for _, line := range visible {
		text := line.text
		if m.focusedPane == rightPane && line.flatIdx >= 0 && line.flatIdx == m.rightCursor {
			displayWidth := lipgloss.Width(text)
			if displayWidth < width {
				text += strings.Repeat(" ", width-displayWidth)
			}
			text = highlight.Render(text)
		}
		rendered = append(rendered, text)
	}

	content := strings.Join(rendered, "\n")
	return style.Render(content)
}

func (m *Model) renderTextBlockLines(tb *model.TextBlock, width int) []string {
	textLines := strings.Split(tb.Text, "\n")

	maxLines := 3
	if m.compactView {
		maxLines = 1
	}
	if !tb.Expanded && len(textLines) > maxLines {
		textLines = textLines[:maxLines]
		textLines[maxLines-1] += "…"
	}

	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.TextBlock))

	var result []string
	for _, l := range textLines {
		line := "  " + l
		if len(line) > width {
			line = line[:width-1] + "…"
		}
		result = append(result, textStyle.Render(line))
	}
	return result
}

func (m *Model) renderToolCallLine(tc *model.ToolCall, nameWidth, summaryWidth, durWidth int) string {
	icon := ToolIcon(tc.Name)
	isKnown := IsKnownTool(tc.Name)

	summary := tc.Summary
	// Append line info metadata after summary for completed calls
	lineInfo := ""
	if tc.LineInfo != "" && tc.Status != model.ToolCallRunning {
		lineInfo = " " + tc.LineInfo
	}
	combined := summary + lineInfo
	if len(combined) > summaryWidth {
		combined = combined[:summaryWidth-3] + "..."
	}
	combined = fmt.Sprintf("%-*s", summaryWidth, combined)

	var nameColor, durColor, resultColor string
	var result string
	switch tc.Status {
	case model.ToolCallDone:
		result = "✓"
		nameColor = m.theme.ToolNameSuccess
		durColor = m.theme.DurationSuccess
		resultColor = m.theme.StatusSuccess
	case model.ToolCallError:
		result = "✗"
		nameColor = m.theme.ToolNameError
		durColor = m.theme.DurationError
		resultColor = m.theme.StatusError
	default:
		result = " "
		nameColor = m.theme.ToolNameRunning
		durColor = m.theme.DurationRunning
	}

	dur := FormatDuration(tc.Duration, tc.Status == model.ToolCallRunning)
	dur = fmt.Sprintf("%*s", durWidth, dur)

	styledIcon := lipgloss.NewStyle().Foreground(lipgloss.Color(nameColor)).Render(icon)
	styledSummary := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.ToolSummary)).Render(combined)
	styledResult := lipgloss.NewStyle().Foreground(lipgloss.Color(resultColor)).Render(result)
	styledDur := lipgloss.NewStyle().Foreground(lipgloss.Color(durColor)).Render(dur)

	showName := !m.compactView || !isKnown
	if showName {
		name := fmt.Sprintf("%-*s", nameWidth, tc.Name)
		if len(tc.Name) > nameWidth {
			name = tc.Name[:nameWidth]
		}
		styledName := lipgloss.NewStyle().Foreground(lipgloss.Color(nameColor)).Render(name)
		return fmt.Sprintf("  %s %s %s %s %s", styledIcon, styledName, styledSummary, styledResult, styledDur)
	}

	return fmt.Sprintf("  %s %s %s %s", styledIcon, styledSummary, styledResult, styledDur)
}

func (m *Model) renderGroupHeaderLine(g *model.ToolCallGroup, nameWidth, summaryWidth, durWidth int) string {
	icon := ToolIcon(g.ToolName)
	isKnown := IsKnownTool(g.ToolName)

	status := g.Status()

	// Build summary: "3/8 files" while in progress, "8 files" when complete
	total := len(g.Children)
	completed := g.CompletedCount()
	unit := GroupSummaryUnit(g.ToolName)
	var summary string
	if status == model.ToolCallRunning {
		summary = fmt.Sprintf("%d/%d %s", completed, total, unit)
	} else {
		summary = fmt.Sprintf("%d %s", total, unit)
	}
	if len(summary) > summaryWidth {
		summary = summary[:summaryWidth]
	}
	summary = fmt.Sprintf("%-*s", summaryWidth, summary)

	var nameColor, durColor, resultColor string
	var result string
	switch status {
	case model.ToolCallDone:
		result = "✓"
		nameColor = m.theme.ToolNameSuccess
		durColor = m.theme.DurationSuccess
		resultColor = m.theme.StatusSuccess
	case model.ToolCallError:
		result = "✗"
		nameColor = m.theme.ToolNameError
		durColor = m.theme.DurationError
		resultColor = m.theme.StatusError
	default:
		result = " "
		nameColor = m.theme.ToolNameRunning
		durColor = m.theme.DurationRunning
	}

	dur := FormatDuration(g.GroupDuration(), status == model.ToolCallRunning)
	dur = fmt.Sprintf("%*s", durWidth, dur)

	styledIcon := lipgloss.NewStyle().Foreground(lipgloss.Color(nameColor)).Render(icon)
	styledSummary := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.ToolSummary)).Render(summary)
	styledResult := lipgloss.NewStyle().Foreground(lipgloss.Color(resultColor)).Render(result)
	styledDur := lipgloss.NewStyle().Foreground(lipgloss.Color(durColor)).Render(dur)

	showName := !m.compactView || !isKnown
	if showName {
		name := fmt.Sprintf("%-*s", nameWidth, g.ToolName)
		if len(g.ToolName) > nameWidth {
			name = g.ToolName[:nameWidth]
		}
		styledName := lipgloss.NewStyle().Foreground(lipgloss.Color(nameColor)).Render(name)
		return fmt.Sprintf("  %s %s %s %s %s", styledIcon, styledName, styledSummary, styledResult, styledDur)
	}

	return fmt.Sprintf("  %s %s %s %s", styledIcon, styledSummary, styledResult, styledDur)
}

// Subprocess management — delegates to executor

func (m *Model) spawnIteration() tea.Cmd {
	m.controller.StartIteration()
	if m.autoFollowLeft.Following() {
		m.selectedIter = len(m.controller.Session.Iterations) - 1
		m.rightCursor = 0
		m.scrollOffset = 0
	}

	ch := m.eventCh

	// Start the executor and bridge its events to Bubble Tea messages
	eventCh, err := m.exec.Start(context.Background(), m.promptContent)
	if err != nil {
		return func() tea.Msg {
			return subprocessExitMsg{err: err}
		}
	}

	// Bridge goroutine: reads session.Event from executor, wraps as tea.Msg
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
		return m.height - 1 // subtract 1 for the header line
	}
	return 20
}

// selectedItems returns the timeline items for the currently selected iteration,
// or nil if the selection is out of range.
func (m *Model) selectedItems() []model.TimelineItem {
	if m.selectedIter >= len(m.controller.Session.Iterations) {
		return nil
	}
	return m.controller.Session.Iterations[m.selectedIter].Items
}

func (m *Model) ensureCursorVisible() {
	items := m.selectedItems()
	lineStart, lc := FlatCursorLineRange(items, m.rightCursor, m.compactView)
	lineEnd := lineStart + lc
	if lineStart < m.scrollOffset {
		m.scrollOffset = lineStart
	}
	visible := m.rightPaneHeight()
	if lineEnd > m.scrollOffset+visible {
		m.scrollOffset = lineEnd - visible
	}
}

func (m *Model) scrollToBottom() {
	total := TotalLines(m.selectedItems(), m.compactView)
	visible := m.rightPaneHeight()
	if total > visible {
		m.scrollOffset = total - visible
	} else {
		m.scrollOffset = 0
	}
}

func (m *Model) clampScroll() {
	total := TotalLines(m.selectedItems(), m.compactView)
	maxScroll := total - m.rightPaneHeight()
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scrollOffset > maxScroll {
		m.scrollOffset = maxScroll
	}
}

// isCursorOnGroup returns true if the flat cursor is on the group header or any of its children.
func (m *Model) isCursorOnGroup(itemIdx int) bool {
	curItemIdx, _ := FlatToItem(m.selectedItems(), m.rightCursor)
	return curItemIdx == itemIdx
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
