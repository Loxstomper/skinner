package tui

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/loxstomper/skinner/internal/config"
	"github.com/loxstomper/skinner/internal/model"
	"github.com/loxstomper/skinner/internal/parser"
	"github.com/loxstomper/skinner/internal/theme"
)

// Bubble Tea messages

type assistantBatchMsg struct {
	Events []interface{} // parser.ToolUseEvent and parser.TextEvent only (UsageEvent sent separately)
}
type toolResultMsg parser.ToolResultEvent
type usageMsg parser.UsageEvent
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
	session       model.Session
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

	// Cost tracking
	hasKnownModel bool
	lastModel     string

	// Auto-follow
	autoFollowLeft  AutoFollow
	autoFollowRight AutoFollow

	// gg state machine
	gPending bool

	// Subprocess
	cmd     *exec.Cmd
	eventCh chan tea.Msg

	// Exit when done
	exitOnComplete bool

	// Quit state
	quitting bool
}

func NewModel(session model.Session, cfg config.Config, promptContent string, th theme.Theme, compactView bool, exitOnComplete bool) Model {
	return Model{
		session:         session,
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
		if idx := m.runningIterationIdx(); idx >= 0 {
			iter := &m.session.Iterations[idx]

			// Collect runs of consecutive same-name ToolUseEvents into groups.
			// Runs of 1 become standalone ToolCalls; runs of 2+ become ToolCallGroups.
			type toolRun struct {
				name   string
				events []parser.ToolUseEvent
			}
			var pending []interface{} // *model.TextBlock or *toolRun

			var currentRun *toolRun
			flushRun := func() {
				if currentRun != nil {
					pending = append(pending, currentRun)
					currentRun = nil
				}
			}

			for _, evt := range msg.Events {
				switch e := evt.(type) {
				case parser.ToolUseEvent:
					if currentRun != nil && currentRun.name == e.Name {
						currentRun.events = append(currentRun.events, e)
					} else {
						flushRun()
						currentRun = &toolRun{name: e.Name, events: []parser.ToolUseEvent{e}}
					}
				case parser.TextEvent:
					flushRun()
					pending = append(pending, &model.TextBlock{Text: e.Text})
				}
			}
			flushRun()

			// Convert pending items to timeline items
			now := time.Now()
			for _, p := range pending {
				switch v := p.(type) {
				case *model.TextBlock:
					iter.Items = append(iter.Items, v)
				case *toolRun:
					if len(v.events) == 1 {
						e := v.events[0]
						iter.Items = append(iter.Items, &model.ToolCall{
							ID:        e.ID,
							Name:      e.Name,
							Summary:   e.Summary,
							LineInfo:  e.LineInfo,
							StartTime: now,
							Status:    model.ToolCallRunning,
						})
					} else {
						group := &model.ToolCallGroup{
							ToolName:     v.name,
							Expanded:     true,
							ManualToggle: false,
						}
						for _, e := range v.events {
							group.Children = append(group.Children, &model.ToolCall{
								ID:        e.ID,
								Name:      e.Name,
								Summary:   e.Summary,
								LineInfo:  e.LineInfo,
								StartTime: now,
								Status:    model.ToolCallRunning,
							})
						}
						iter.Items = append(iter.Items, group)
					}
				}
			}

			if m.selectedIter == idx && m.autoFollowRight.Following() {
				m.rightCursor = FlatCursorCount(iter.Items) - 1
				m.scrollToBottom()
			}
		}
		return m, waitForEvent(m.eventCh)

	case usageMsg:
		m.session.InputTokens += msg.InputTokens
		m.session.OutputTokens += msg.OutputTokens
		m.session.CacheReadTokens += msg.CacheReadInputTokens
		m.session.CacheCreationTokens += msg.CacheCreationInputTokens
		m.session.LastInputTokens = msg.InputTokens
		m.session.LastCacheReadTokens = msg.CacheReadInputTokens
		if pricing, ok := m.config.Pricing[msg.Model]; ok {
			m.hasKnownModel = true
			m.lastModel = msg.Model
			m.session.TotalCost += float64(msg.InputTokens) * pricing.Input
			m.session.TotalCost += float64(msg.OutputTokens) * pricing.Output
			m.session.TotalCost += float64(msg.CacheReadInputTokens) * pricing.CacheRead
			m.session.TotalCost += float64(msg.CacheCreationInputTokens) * pricing.CacheCreate
		}
		return m, waitForEvent(m.eventCh)

	case toolResultMsg:
		if idx := m.runningIterationIdx(); idx >= 0 {
			iter := &m.session.Iterations[idx]
			for i, item := range iter.Items {
				if tc, ok := item.(*model.ToolCall); ok && tc.ID == msg.ToolUseID {
					m.applyToolResult(tc, msg)
					break
				}
				if group, ok := item.(*model.ToolCallGroup); ok {
					found := false
					for _, child := range group.Children {
						if child.ID == msg.ToolUseID {
							m.applyToolResult(child, msg)
							found = true
							break
						}
					}
					if found {
						// Check if the group just completed (all children done)
						if group.Status() != model.ToolCallRunning && !group.ManualToggle {
							cursorOnGroup := m.isCursorOnGroup(i)
							if m.selectedIter != idx || !cursorOnGroup {
								group.Expanded = false
							}
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
		m.cmd = nil
		if idx := m.runningIterationIdx(); idx >= 0 {
			iter := &m.session.Iterations[idx]
			iter.Duration = time.Since(iter.StartTime)
			if msg.err != nil {
				iter.Status = model.IterationFailed
			} else {
				iter.Status = model.IterationCompleted
			}
		}

		if m.shouldStartNext() {
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
		if m.cmd != nil && m.cmd.Process != nil {
			_ = m.cmd.Process.Kill()
		}
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
			if m.selectedIter < len(m.session.Iterations)-1 {
				m.selectedIter++
				m.rightCursor = 0
				m.scrollOffset = 0
			}
			m.autoFollowLeft.OnManualMove(m.selectedIter == len(m.session.Iterations)-1)
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
			m.autoFollowLeft.OnManualMove(m.selectedIter == len(m.session.Iterations)-1)
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
			if m.selectedIter >= len(m.session.Iterations) {
				m.selectedIter = len(m.session.Iterations) - 1
			}
			if m.selectedIter < 0 {
				m.selectedIter = 0
			}
			m.rightCursor = 0
			m.scrollOffset = 0
			m.autoFollowLeft.OnManualMove(m.selectedIter == len(m.session.Iterations)-1)
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
			m.autoFollowLeft.OnManualMove(m.selectedIter == len(m.session.Iterations)-1)
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
		} else if m.selectedIter < len(m.session.Iterations) {
			iter := &m.session.Iterations[m.selectedIter]
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
		m.autoFollowLeft.OnManualMove(m.selectedIter == len(m.session.Iterations)-1)
	} else {
		m.rightCursor = 0
		m.scrollOffset = 0
		m.autoFollowRight.OnManualMove(false)
	}
}

func (m *Model) jumpToBottom() {
	if m.focusedPane == leftPane {
		if len(m.session.Iterations) > 0 {
			m.selectedIter = len(m.session.Iterations) - 1
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
	dur := FormatDurationValue(time.Since(m.session.StartTime))
	inputTokens := m.session.InputTokens + m.session.CacheReadTokens + m.session.CacheCreationTokens
	outputTokens := m.session.OutputTokens

	centreText := fmt.Sprintf("⏱ %s   ↑%s ↓%s tokens", dur, FormatTokens(inputTokens), FormatTokens(outputTokens))
	centreRendered := dim.Render(centreText)

	// Context window percentage
	if m.hasKnownModel && m.lastModel != "" {
		if pricing, ok := m.config.Pricing[m.lastModel]; ok && pricing.ContextWindow > 0 {
			pct := int((m.session.LastInputTokens + m.session.LastCacheReadTokens) * 100 / int64(pricing.ContextWindow))
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
	if m.hasKnownModel {
		centreRendered += dim.Render(fmt.Sprintf("   ~$%.2f", m.session.TotalCost))
	}

	// Right side: iteration progress + status icon
	iterCount := len(m.session.Iterations)
	if iterCount == 0 {
		iterCount = 1
	}
	var iterText string
	if m.session.MaxIterations > 0 {
		iterText = fmt.Sprintf("Iter %d/%d", iterCount, m.session.MaxIterations)
	} else {
		iterText = fmt.Sprintf("Iter %d", iterCount)
	}

	var statusIcon, statusColor string
	if idx := m.runningIterationIdx(); idx >= 0 {
		statusIcon = "⟳"
		statusColor = m.theme.StatusRunning
	} else if iterCount > 0 {
		lastIter := m.session.Iterations[len(m.session.Iterations)-1]
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
	for i, iter := range m.session.Iterations {
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

	if m.selectedIter >= len(m.session.Iterations) {
		return style.Render("")
	}

	iter := m.session.Iterations[m.selectedIter]
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

// Subprocess management

func (m *Model) spawnIteration() tea.Cmd {
	iter := model.Iteration{
		Index:     len(m.session.Iterations),
		Status:    model.IterationRunning,
		StartTime: time.Now(),
	}
	m.session.Iterations = append(m.session.Iterations, iter)
	if m.autoFollowLeft.Following() {
		m.selectedIter = len(m.session.Iterations) - 1
		m.rightCursor = 0
		m.scrollOffset = 0
	}

	cmd := exec.Command("claude",
		"-p",
		"--dangerously-skip-permissions",
		"--output-format=stream-json",
		"--verbose",
	)
	cmd.Stdin = strings.NewReader(m.promptContent)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return func() tea.Msg {
			return subprocessExitMsg{err: err}
		}
	}

	if err := cmd.Start(); err != nil {
		return func() tea.Msg {
			return subprocessExitMsg{err: err}
		}
	}

	m.cmd = cmd
	ch := m.eventCh

	// Goroutine reads stdout and sends parsed events to channel
	go func() {
		readEvents(stdout, ch)
		err := cmd.Wait()
		ch <- subprocessExitMsg{err: err}
	}()

	return waitForEvent(ch)
}

func readEvents(r io.Reader, ch chan<- tea.Msg) {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		events, err := parser.ParseStreamEvent(line)
		if err != nil {
			continue
		}

		// Collect ToolUseEvent and TextEvent from assistant events into a batch.
		// UsageEvent, ToolResultEvent, and IterationEndEvent are sent individually.
		var batch []interface{}
		for _, evt := range events {
			switch e := evt.(type) {
			case parser.UsageEvent:
				ch <- usageMsg(e)
			case parser.ToolUseEvent:
				batch = append(batch, e)
			case parser.TextEvent:
				batch = append(batch, e)
			case parser.ToolResultEvent:
				ch <- toolResultMsg(e)
			case parser.IterationEndEvent:
				ch <- iterationEndMsg{}
			}
		}
		if len(batch) > 0 {
			ch <- assistantBatchMsg{Events: batch}
		}
	}
}

func waitForEvent(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

// Helpers

func (m *Model) applyToolResult(tc *model.ToolCall, msg toolResultMsg) {
	tc.Duration = time.Since(tc.StartTime)
	tc.IsError = msg.IsError
	if msg.IsError {
		tc.Status = model.ToolCallError
	} else {
		tc.Status = model.ToolCallDone
	}
	if msg.LineInfo != "" && tc.LineInfo == "" && tc.Name == "Read" {
		tc.LineInfo = msg.LineInfo
	}
}

func (m *Model) runningIterationIdx() int {
	for i, iter := range m.session.Iterations {
		if iter.Status == model.IterationRunning {
			return i
		}
	}
	return -1
}

func (m *Model) shouldStartNext() bool {
	if m.quitting {
		return false
	}
	count := len(m.session.Iterations)
	if m.session.MaxIterations > 0 && count >= m.session.MaxIterations {
		return false
	}
	return true
}

func (m *Model) rightPaneHeight() int {
	if m.height > 1 {
		return m.height - 1 // subtract 1 for the header line
	}
	return 20
}

// selectedItems returns the timeline items for the currently selected iteration,
// or nil if the selection is out of range.
func (m *Model) selectedItems() []model.TimelineItem {
	if m.selectedIter >= len(m.session.Iterations) {
		return nil
	}
	return m.session.Iterations[m.selectedIter].Items
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
