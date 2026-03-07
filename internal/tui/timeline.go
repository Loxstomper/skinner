package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/loxstomper/skinner/internal/model"
	"github.com/loxstomper/skinner/internal/theme"
)

// Timeline is the right pane component that displays the message timeline.
// It owns its cursor position, scroll offset, and auto-follow state.
type Timeline struct {
	Cursor     int
	Scroll     int
	AutoFollow AutoFollow
}

// NewTimeline creates a new Timeline with auto-follow enabled.
func NewTimeline() Timeline {
	return Timeline{
		AutoFollow: NewAutoFollow(),
	}
}

// TimelineProps contains the data needed to render the timeline.
type TimelineProps struct {
	Items       []model.TimelineItem
	Width       int
	Height      int
	Focused     bool
	CompactView bool
	Theme       theme.Theme
}

// renderedLine pairs a rendered text line with its flat cursor index.
type renderedLine struct {
	text    string
	flatIdx int // flat cursor position (-1 for continuation lines of text blocks)
}

// Update handles key events for the timeline.
func (tl *Timeline) Update(msg tea.KeyMsg, props TimelineProps) tea.Cmd {
	key := msg.String()
	maxPos := FlatCursorCount(props.Items) - 1
	atEnd := func() bool { return tl.Cursor >= maxPos }

	switch key {
	case "j", "down":
		if tl.Cursor < maxPos {
			tl.Cursor++
			tl.ensureCursorVisible(props)
		}
		tl.AutoFollow.OnManualMove(atEnd())

	case "k", "up":
		if tl.Cursor > 0 {
			tl.Cursor--
			tl.ensureCursorVisible(props)
		}
		tl.AutoFollow.OnManualMove(atEnd())

	case "g":
		// gg handled by root — root calls JumpToTop

	case "G", "end":
		if maxPos >= 0 {
			tl.Cursor = maxPos
			tl.scrollToBottom(props)
		}
		tl.AutoFollow.JumpToEnd()

	case "home":
		tl.Cursor = 0
		tl.Scroll = 0
		tl.AutoFollow.OnManualMove(false)

	case "pgdown":
		tl.Scroll += props.Height
		tl.clampScroll(props)
		tl.clampCursorToViewport(props)
		total := TotalLines(props.Items, props.CompactView)
		tl.AutoFollow.OnManualMove(tl.Scroll+props.Height >= total)

	case "pgup":
		tl.Scroll -= props.Height
		if tl.Scroll < 0 {
			tl.Scroll = 0
		}
		tl.clampCursorToViewport(props)
		tl.AutoFollow.OnManualMove(false)

	case "enter":
		tl.handleEnter(props)
	}

	return nil
}

// handleEnter toggles expand/collapse on the selected item.
func (tl *Timeline) handleEnter(props TimelineProps) {
	itemIdx, childIdx := FlatToItem(props.Items, tl.Cursor)
	if itemIdx >= len(props.Items) {
		return
	}
	switch it := props.Items[itemIdx].(type) {
	case *model.TextBlock:
		it.Expanded = !it.Expanded
		tl.ensureCursorVisible(props)
	case *model.ToolCall:
		it.Expanded = !it.Expanded
		tl.ensureCursorVisible(props)
	case *model.ToolCallGroup:
		if childIdx == -1 {
			// On group header: toggle expand/collapse
			it.ManualToggle = true
			if it.Expanded {
				// Collapsing: move cursor to header position
				it.Expanded = false
				tl.Cursor = ItemToFlat(props.Items, itemIdx)
			} else {
				it.Expanded = true
			}
			tl.ensureCursorVisible(props)
		} else if childIdx >= 0 && childIdx < len(it.Children) {
			// On child row: toggle child tool call expansion
			it.Children[childIdx].Expanded = !it.Children[childIdx].Expanded
			tl.ensureCursorVisible(props)
		}
	}
}

// View renders the timeline.
func (tl *Timeline) View(props TimelineProps) string {
	style := lipgloss.NewStyle().Width(props.Width).Height(props.Height)

	if len(props.Items) == 0 {
		return style.Render("  No activity yet...")
	}

	iconWidth := 2 // icon + space
	nameWidth := 6
	durWidth := 8

	var summaryWidth, childSummaryWidth int
	if props.CompactView {
		summaryWidth = props.Width - iconWidth - durWidth - 5
		if summaryWidth < 10 {
			summaryWidth = 10
		}
		childSummaryWidth = summaryWidth - 2
		if childSummaryWidth < 10 {
			childSummaryWidth = 10
		}
	} else {
		summaryWidth = props.Width - iconWidth - nameWidth - durWidth - 7
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
	for _, item := range props.Items {
		switch it := item.(type) {
		case *model.TextBlock:
			textLines := renderTextBlockLines(it, props.Width, props.CompactView, props.Theme)
			for _, l := range textLines {
				lines = append(lines, renderedLine{text: l, flatIdx: flatPos})
			}
			flatPos++
		case *model.ToolCall:
			l := renderToolCallLine(it, nameWidth, summaryWidth, durWidth, props.CompactView, props.Theme)
			lines = append(lines, renderedLine{text: l, flatIdx: flatPos})
			// Render expanded content lines for standalone tool calls
			if it.Expanded {
				for _, cl := range expandedContentLines(it) {
					rendered := renderExpandedContentLine(cl, it.Name, props.Width, props.Theme)
					lines = append(lines, renderedLine{text: rendered, flatIdx: -1})
				}
			}
			flatPos++
		case *model.ToolCallGroup:
			l := renderGroupHeaderLine(it, nameWidth, summaryWidth, durWidth, props.CompactView, props.Theme)
			lines = append(lines, renderedLine{text: l, flatIdx: flatPos})
			flatPos++
			if it.Expanded {
				for ci := range it.Children {
					child := it.Children[ci]
					cl := renderToolCallLine(child, nameWidth, childSummaryWidth, durWidth, props.CompactView, props.Theme)
					cl = "  " + cl
					lines = append(lines, renderedLine{text: cl, flatIdx: flatPos})
					// Render expanded content lines for group children with extra indent
					if child.Expanded {
						for _, el := range expandedContentLines(child) {
							rendered := "  " + renderExpandedContentLine(el, child.Name, props.Width-2, props.Theme)
							lines = append(lines, renderedLine{text: rendered, flatIdx: -1})
						}
					}
					flatPos++
				}
			}
		}
	}

	return tl.renderWithLines(lines, props)
}

// renderWithLines applies scroll and cursor highlighting.
func (tl *Timeline) renderWithLines(lines []renderedLine, props TimelineProps) string {
	style := lipgloss.NewStyle().Width(props.Width).Height(props.Height)

	start := tl.Scroll
	if start >= len(lines) {
		start = len(lines) - 1
	}
	if start < 0 {
		start = 0
	}
	end := start + props.Height
	if end > len(lines) {
		end = len(lines)
	}

	visible := lines[start:end]
	highlight := lipgloss.NewStyle().Background(lipgloss.Color(props.Theme.Highlight))

	var rendered []string
	for _, line := range visible {
		text := line.text
		if props.Focused && line.flatIdx >= 0 && line.flatIdx == tl.Cursor {
			displayWidth := lipgloss.Width(text)
			if displayWidth < props.Width {
				text += strings.Repeat(" ", props.Width-displayWidth)
			}
			text = highlight.Render(text)
		}
		rendered = append(rendered, text)
	}

	content := strings.Join(rendered, "\n")
	return style.Render(content)
}

// OnNewItems updates the timeline when new items arrive during auto-follow.
func (tl *Timeline) OnNewItems(props TimelineProps) {
	tl.AutoFollow.OnNewItem()
	if tl.AutoFollow.Following() {
		maxPos := FlatCursorCount(props.Items) - 1
		if maxPos >= 0 {
			tl.Cursor = maxPos
		}
		tl.scrollToBottom(props)
	}
}

// JumpToTop moves the cursor to the first item.
func (tl *Timeline) JumpToTop() {
	tl.Cursor = 0
	tl.Scroll = 0
	tl.AutoFollow.OnManualMove(false)
}

// JumpToBottom moves the cursor to the last item and resumes auto-follow.
func (tl *Timeline) JumpToBottom(props TimelineProps) {
	maxPos := FlatCursorCount(props.Items)
	if maxPos > 0 {
		tl.Cursor = maxPos - 1
		tl.scrollToBottom(props)
	}
	tl.AutoFollow.JumpToEnd()
}

// ResetPosition resets cursor and scroll to the beginning.
func (tl *Timeline) ResetPosition() {
	tl.Cursor = 0
	tl.Scroll = 0
}

// ensureCursorVisible adjusts scroll to keep the cursor in view.
func (tl *Timeline) ensureCursorVisible(props TimelineProps) {
	lineStart, lc := FlatCursorLineRange(props.Items, tl.Cursor, props.CompactView)
	lineEnd := lineStart + lc
	if lineStart < tl.Scroll {
		tl.Scroll = lineStart
	}
	if lineEnd > tl.Scroll+props.Height {
		tl.Scroll = lineEnd - props.Height
	}
}

// scrollToBottom sets scroll so the last line is visible.
func (tl *Timeline) scrollToBottom(props TimelineProps) {
	total := TotalLines(props.Items, props.CompactView)
	if total > props.Height {
		tl.Scroll = total - props.Height
	} else {
		tl.Scroll = 0
	}
}

// clampCursorToViewport moves the cursor into the visible viewport after
// page scrolling. If the cursor is above the viewport, it moves to the first
// visible flat position; if below, to the last visible flat position.
func (tl *Timeline) clampCursorToViewport(props TimelineProps) {
	lineStart, lc := FlatCursorLineRange(props.Items, tl.Cursor, props.CompactView)
	lineEnd := lineStart + lc

	viewStart := tl.Scroll
	viewEnd := tl.Scroll + props.Height

	if lineEnd <= viewStart {
		// Cursor is above viewport — move to first visible position
		tl.Cursor = LineToFlatCursor(props.Items, viewStart, props.CompactView)
	} else if lineStart >= viewEnd {
		// Cursor is below viewport — move to last visible position
		tl.Cursor = LineToFlatCursor(props.Items, viewEnd-1, props.CompactView)
	}
}

// ScrollBy adjusts the scroll offset by delta lines (positive = down, negative = up).
// It clamps the result and pauses auto-follow.
func (tl *Timeline) ScrollBy(delta int, props TimelineProps) {
	tl.Scroll += delta
	if tl.Scroll < 0 {
		tl.Scroll = 0
	}
	tl.clampScroll(props)
	tl.AutoFollow.OnManualMove(false)
}

// ClickRow handles a mouse click on the given pane-relative row.
// It maps scroll+row to the flat cursor position and sets the cursor if valid.
// Returns true if the cursor changed.
func (tl *Timeline) ClickRow(row int, props TimelineProps) bool {
	line := tl.Scroll + row
	total := TotalLines(props.Items, props.CompactView)
	if line < 0 || line >= total {
		return false
	}
	tl.Cursor = LineToFlatCursor(props.Items, line, props.CompactView)
	maxPos := FlatCursorCount(props.Items) - 1
	tl.AutoFollow.OnManualMove(tl.Cursor >= maxPos)
	return true
}

// clampScroll ensures scroll doesn't exceed the maximum.
func (tl *Timeline) clampScroll(props TimelineProps) {
	total := TotalLines(props.Items, props.CompactView)
	maxScroll := total - props.Height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if tl.Scroll > maxScroll {
		tl.Scroll = maxScroll
	}
}

// renderTextBlockLines renders a text block as one or more display lines.
func renderTextBlockLines(tb *model.TextBlock, width int, compactView bool, th theme.Theme) []string {
	textLines := strings.Split(tb.Text, "\n")

	maxLines := 3
	if compactView {
		maxLines = 1
	}
	if !tb.Expanded && len(textLines) > maxLines {
		textLines = textLines[:maxLines]
		textLines[maxLines-1] += "…"
	}

	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.TextBlock))

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

// renderToolCallLine renders a single tool call row.
func renderToolCallLine(tc *model.ToolCall, nameWidth, summaryWidth, durWidth int, compactView bool, th theme.Theme) string {
	icon := ToolIcon(tc.Name)
	isKnown := IsKnownTool(tc.Name)

	summary := tc.Summary
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
		nameColor = th.ToolNameSuccess
		durColor = th.DurationSuccess
		resultColor = th.StatusSuccess
	case model.ToolCallError:
		result = "✗"
		nameColor = th.ToolNameError
		durColor = th.DurationError
		resultColor = th.StatusError
	default:
		result = " "
		nameColor = th.ToolNameRunning
		durColor = th.DurationRunning
	}

	dur := FormatDuration(tc.Duration, tc.Status == model.ToolCallRunning)
	dur = fmt.Sprintf("%*s", durWidth, dur)

	styledIcon := lipgloss.NewStyle().Foreground(lipgloss.Color(nameColor)).Render(icon)
	styledSummary := lipgloss.NewStyle().Foreground(lipgloss.Color(th.ToolSummary)).Render(combined)
	styledResult := lipgloss.NewStyle().Foreground(lipgloss.Color(resultColor)).Render(result)
	styledDur := lipgloss.NewStyle().Foreground(lipgloss.Color(durColor)).Render(dur)

	showName := !compactView || !isKnown
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

// renderGroupHeaderLine renders a tool call group header row.
func renderGroupHeaderLine(g *model.ToolCallGroup, nameWidth, summaryWidth, durWidth int, compactView bool, th theme.Theme) string {
	icon := ToolIcon(g.ToolName)
	isKnown := IsKnownTool(g.ToolName)

	status := g.Status()

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
		nameColor = th.ToolNameSuccess
		durColor = th.DurationSuccess
		resultColor = th.StatusSuccess
	case model.ToolCallError:
		result = "✗"
		nameColor = th.ToolNameError
		durColor = th.DurationError
		resultColor = th.StatusError
	default:
		result = " "
		nameColor = th.ToolNameRunning
		durColor = th.DurationRunning
	}

	dur := FormatDuration(g.GroupDuration(), status == model.ToolCallRunning)
	dur = fmt.Sprintf("%*s", durWidth, dur)

	styledIcon := lipgloss.NewStyle().Foreground(lipgloss.Color(nameColor)).Render(icon)
	styledSummary := lipgloss.NewStyle().Foreground(lipgloss.Color(th.ToolSummary)).Render(summary)
	styledResult := lipgloss.NewStyle().Foreground(lipgloss.Color(resultColor)).Render(result)
	styledDur := lipgloss.NewStyle().Foreground(lipgloss.Color(durColor)).Render(dur)

	showName := !compactView || !isKnown
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
