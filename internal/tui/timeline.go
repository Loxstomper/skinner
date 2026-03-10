package tui

import (
	"fmt"
	"strings"

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

	// Sub-scroll state: when a tool call's expanded content exceeds 40% of
	// pane height, the user can enter sub-scroll mode to scroll within it.
	// SubScrollIdx is the flat cursor index of the tool call in sub-scroll
	// mode, or -1 when inactive.
	SubScrollIdx    int
	SubScrollOffset int

	// CountBuffer accumulates digit keys for vim count+jump motions.
	// When j/k is pressed, the buffer is consumed as the move count.
	CountBuffer string
}

// NewTimeline creates a new Timeline with auto-follow enabled.
func NewTimeline() Timeline {
	return Timeline{
		AutoFollow:   NewAutoFollow(),
		SubScrollIdx: -1,
	}
}

// TimelineProps contains the data needed to render the timeline.
type TimelineProps struct {
	Items       []model.TimelineItem
	Width       int
	Height      int
	Focused     bool
	CompactView bool
	LineNumbers bool
	Theme       theme.Theme
	WorkDir     string // CWD for path trimming in tool call summaries
}

// renderedLine pairs a rendered text line with its flat cursor index.
type renderedLine struct {
	text        string
	flatIdx     int  // flat cursor position (-1 for continuation lines of text blocks)
	highlighted bool // true when highlight background is baked into the text
}

// InSubScroll returns true when the timeline is in sub-scroll mode.
func (tl *Timeline) InSubScroll() bool {
	return tl.SubScrollIdx >= 0
}

// AccumulateDigit adds a digit to the count buffer. Leading zeros are ignored.
func (tl *Timeline) AccumulateDigit(digit rune) {
	if tl.CountBuffer == "" && digit == '0' {
		return // ignore leading zero
	}
	tl.CountBuffer += string(digit)
}

// ConsumeCount returns the accumulated count (minimum 1) and clears the buffer.
func (tl *Timeline) ConsumeCount() int {
	if tl.CountBuffer == "" {
		return 1
	}
	n := 0
	for _, c := range tl.CountBuffer {
		n = n*10 + int(c-'0')
	}
	tl.CountBuffer = ""
	if n < 1 {
		return 1
	}
	return n
}

// ClearCount clears the count buffer without consuming it.
func (tl *Timeline) ClearCount() {
	tl.CountBuffer = ""
}

// HandleAction processes a resolved action for the timeline.
func (tl *Timeline) HandleAction(action string, props TimelineProps) {
	tl.HandleActionWithCount(action, 1, props)
}

// HandleActionWithCount processes a resolved action with a count multiplier.
func (tl *Timeline) HandleActionWithCount(action string, count int, props TimelineProps) {
	// When in sub-scroll mode, route navigation to the sub-scroll handler.
	if tl.InSubScroll() {
		tl.handleSubScrollAction(action, props)
		return
	}

	maxPos := FlatCursorCount(props.Items) - 1
	atEnd := func() bool { return tl.Cursor >= maxPos }

	switch action {
	case "move_down":
		tl.Cursor += count
		if tl.Cursor > maxPos {
			tl.Cursor = maxPos
		}
		if tl.Cursor < 0 {
			tl.Cursor = 0
		}
		tl.ensureCursorVisible(props)
		tl.AutoFollow.OnManualMove(atEnd())

	case "move_up":
		tl.Cursor -= count
		if tl.Cursor < 0 {
			tl.Cursor = 0
		}
		tl.ensureCursorVisible(props)
		tl.AutoFollow.OnManualMove(atEnd())

	case "jump_bottom":
		if maxPos >= 0 {
			tl.Cursor = maxPos
			tl.scrollToBottom(props)
		}
		tl.AutoFollow.JumpToEnd()

	case "jump_top":
		tl.Cursor = 0
		tl.Scroll = 0
		tl.AutoFollow.OnManualMove(false)

	case "page_down":
		tl.Scroll += props.Height
		tl.clampScroll(props)
		tl.clampCursorToViewport(props)
		total := TotalLines(props.Items, props.CompactView)
		tl.AutoFollow.OnManualMove(tl.Scroll+props.Height >= total)

	case "page_up":
		tl.Scroll -= props.Height
		if tl.Scroll < 0 {
			tl.Scroll = 0
		}
		tl.clampCursorToViewport(props)
		tl.AutoFollow.OnManualMove(false)

	case "expand":
		tl.handleEnter(props)
	}
}

// handleSubScrollAction processes actions while in sub-scroll mode.
func (tl *Timeline) handleSubScrollAction(action string, props TimelineProps) {
	tc := tl.subScrollToolCall(props)
	if tc == nil {
		tl.ExitSubScroll()
		return
	}
	contentLen := len(expandedContentLines(tc))
	maxViewport := subScrollViewportHeight(contentLen, props.Height)

	switch action {
	case "move_down":
		if tl.SubScrollOffset < contentLen-maxViewport {
			tl.SubScrollOffset++
		}
	case "move_up":
		if tl.SubScrollOffset > 0 {
			tl.SubScrollOffset--
		}
	case "jump_top":
		tl.SubScrollOffset = 0
	case "jump_bottom":
		maxOffset := contentLen - maxViewport
		if maxOffset < 0 {
			maxOffset = 0
		}
		tl.SubScrollOffset = maxOffset
	case "expand":
		// Enter on sub-scroll: collapse and exit
		tc.Expanded = false
		tl.ExitSubScroll()
		tl.ensureCursorVisible(props)
	}
	// escape is handled in root.go before reaching here
}

// subScrollToolCall returns the ToolCall currently in sub-scroll mode, or nil.
func (tl *Timeline) subScrollToolCall(props TimelineProps) *model.ToolCall {
	if tl.SubScrollIdx < 0 {
		return nil
	}
	itemIdx, childIdx := FlatToItem(props.Items, tl.SubScrollIdx)
	if itemIdx >= len(props.Items) {
		return nil
	}
	switch it := props.Items[itemIdx].(type) {
	case *model.ToolCall:
		return it
	case *model.ToolCallGroup:
		if childIdx >= 0 && childIdx < len(it.Children) {
			return it.Children[childIdx]
		}
	}
	return nil
}

// ExitSubScroll leaves sub-scroll mode, returning to timeline navigation.
func (tl *Timeline) ExitSubScroll() {
	tl.SubScrollIdx = -1
	tl.SubScrollOffset = 0
}

// subScrollViewportHeight returns the capped viewport height for sub-scroll.
// Content exceeding 40% of pane height is capped at 70% of pane height.
func subScrollViewportHeight(contentLines, paneHeight int) int {
	threshold := paneHeight * 40 / 100
	if contentLines <= threshold {
		return contentLines
	}
	cap := paneHeight * 70 / 100
	if cap < 1 {
		cap = 1
	}
	if contentLines < cap {
		return contentLines
	}
	return cap
}

// subScrollEnabled returns true if the expanded content lines exceed the
// inline threshold (40% of pane height) and sub-scroll would be active.
func subScrollEnabled(contentLines, paneHeight int) bool {
	threshold := paneHeight * 40 / 100
	return contentLines > threshold
}

// handleEnter toggles expand/collapse on the selected item. If the item is
// already expanded and its content exceeds the inline threshold, enter
// sub-scroll mode instead of collapsing.
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
		if it.Expanded {
			// Already expanded: enter sub-scroll if content is large enough
			content := expandedContentLines(it)
			if subScrollEnabled(len(content), props.Height) {
				tl.SubScrollIdx = tl.Cursor
				tl.SubScrollOffset = 0
				return
			}
			// Content is small — collapse
			it.Expanded = false
		} else {
			it.Expanded = true
		}
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
			child := it.Children[childIdx]
			if child.Expanded {
				// Already expanded: enter sub-scroll if large enough
				content := expandedContentLines(child)
				if subScrollEnabled(len(content), props.Height) {
					tl.SubScrollIdx = tl.Cursor
					tl.SubScrollOffset = 0
					return
				}
				child.Expanded = false
			} else {
				child.Expanded = true
			}
			tl.ensureCursorVisible(props)
		}
	}
}

// gutterWidth is the width of the line number gutter (3-char number + 1 space).
const gutterWidth = 4

// View renders the timeline.
func (tl *Timeline) View(props TimelineProps) string {
	style := lipgloss.NewStyle().Width(props.Width).Height(props.Height)

	if len(props.Items) == 0 {
		return style.Render("  No activity yet...")
	}

	// Reserve gutter space when line numbers are enabled.
	contentWidth := props.Width
	if props.LineNumbers {
		contentWidth -= gutterWidth
		if contentWidth < 20 {
			contentWidth = 20
		}
	}

	iconWidth := 2 // icon + space
	nameWidth := 6
	durWidth := 8

	var summaryWidth, childSummaryWidth int
	if props.CompactView {
		summaryWidth = contentWidth - iconWidth - durWidth - 5
		if summaryWidth < 10 {
			summaryWidth = 10
		}
		childSummaryWidth = summaryWidth - 2
		if childSummaryWidth < 10 {
			childSummaryWidth = 10
		}
	} else {
		summaryWidth = contentWidth - iconWidth - nameWidth - durWidth - 7
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
		// Determine if this flat position is the highlighted cursor row.
		hlBg := ""
		isCursor := props.Focused && flatPos == tl.Cursor
		if isCursor {
			hlBg = props.Theme.Highlight
		}

		switch it := item.(type) {
		case *model.TextBlock:
			textLines := renderTextBlockLines(it, contentWidth, props.CompactView, props.Theme, hlBg)
			for _, l := range textLines {
				lines = append(lines, renderedLine{text: l, flatIdx: flatPos, highlighted: isCursor})
			}
			flatPos++
		case *model.ToolCall:
			l := renderToolCallLine(it, nameWidth, summaryWidth, durWidth, props.CompactView, props.Theme, hlBg, props.WorkDir)
			lines = append(lines, renderedLine{text: l, flatIdx: flatPos, highlighted: isCursor})
			if it.Expanded {
				lines = tl.appendExpandedLines(lines, it, flatPos, "", props, contentWidth)
			}
			flatPos++
		case *model.ToolCallGroup:
			l := renderGroupHeaderLine(it, nameWidth, summaryWidth, durWidth, props.CompactView, props.Theme, hlBg)
			lines = append(lines, renderedLine{text: l, flatIdx: flatPos, highlighted: isCursor})
			flatPos++
			if it.Expanded {
				for ci := range it.Children {
					child := it.Children[ci]
					childHlBg := ""
					childIsCursor := props.Focused && flatPos == tl.Cursor
					if childIsCursor {
						childHlBg = props.Theme.Highlight
					}
					cl := renderToolCallLine(child, nameWidth, childSummaryWidth, durWidth, props.CompactView, props.Theme, childHlBg, props.WorkDir)
					if childHlBg != "" {
						cl = lipgloss.NewStyle().Background(lipgloss.Color(childHlBg)).Render("  ") + cl
					} else {
						cl = "  " + cl
					}
					lines = append(lines, renderedLine{text: cl, flatIdx: flatPos, highlighted: childIsCursor})
					if child.Expanded {
						lines = tl.appendExpandedLines(lines, child, flatPos, "  ", props, contentWidth)
					}
					flatPos++
				}
			}
		}
	}

	return tl.renderWithLines(lines, props)
}

// appendExpandedLines adds expanded content lines for a tool call, applying
// sub-scroll viewport capping when in sub-scroll mode.
func (tl *Timeline) appendExpandedLines(lines []renderedLine, tc *model.ToolCall, flatPos int, indent string, props TimelineProps, availWidth int) []renderedLine {
	allContent := expandedContentLines(tc)
	if len(allContent) == 0 {
		return lines
	}

	inSubScroll := tl.SubScrollIdx == flatPos
	cw := availWidth - len(indent)
	if cw < 10 {
		cw = 10
	}

	// For Edit tool calls, use pre-styled diff rendering with line numbers
	// and adaptive layout (unified vs side-by-side based on width).
	isEditDiff := tc.Name == "Edit" && tc.RawInput != nil

	// For Edit diffs, pre-render styled lines and use their count for
	// sub-scroll calculations (side-by-side may have fewer lines than unified).
	var editStyledLines []string
	contentLen := len(allContent)
	if isEditDiff {
		editStyledLines = renderEditDiffStyled(tc.RawInput, cw, props.Theme)
		if editStyledLines != nil {
			contentLen = len(editStyledLines)
		}
	}

	if inSubScroll && subScrollEnabled(contentLen, props.Height) {
		// Sub-scroll mode: show capped viewport with border and indicator
		vpHeight := subScrollViewportHeight(contentLen, props.Height)
		offset := tl.SubScrollOffset
		// Clamp offset
		maxOffset := contentLen - vpHeight
		if maxOffset < 0 {
			maxOffset = 0
		}
		if offset > maxOffset {
			offset = maxOffset
			tl.SubScrollOffset = offset
		}

		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(props.Theme.ForegroundDim))
		borderChar := dimStyle.Render("│")

		for i := 0; i < vpHeight; i++ {
			contentIdx := offset + i
			var rendered string
			if isEditDiff && contentIdx < len(editStyledLines) {
				rendered = indent + borderChar + " " + editStyledLines[contentIdx]
			} else if contentIdx < len(allContent) {
				rendered = indent + borderChar + " " + renderExpandedContentLine(allContent[contentIdx], tc.Name, cw-3, props.Theme)
			}

			// Last line: append scroll indicator
			if i == vpHeight-1 {
				indicator := fmt.Sprintf("[%d/%d]", offset+vpHeight, contentLen)
				styledIndicator := dimStyle.Render(indicator)
				lineWidth := lipgloss.Width(rendered)
				indicatorWidth := lipgloss.Width(styledIndicator)
				padding := availWidth - lineWidth - indicatorWidth
				if padding > 0 {
					rendered += strings.Repeat(" ", padding) + styledIndicator
				} else {
					rendered += " " + styledIndicator
				}
			}

			lines = append(lines, renderedLine{text: rendered, flatIdx: -1})
		}
	} else {
		// Normal inline display
		if isEditDiff && editStyledLines != nil {
			// Pre-styled diff lines with line numbers and adaptive layout
			for _, sl := range editStyledLines {
				rendered := indent + sl
				lines = append(lines, renderedLine{text: rendered, flatIdx: -1})
			}
		} else {
			for _, cl := range allContent {
				rendered := indent + renderExpandedContentLine(cl, tc.Name, cw, props.Theme)
				lines = append(lines, renderedLine{text: rendered, flatIdx: -1})
			}
		}
	}

	return lines
}

// renderWithLines applies scroll, cursor highlighting, and optional line number gutter.
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
	hlBgStyle := lipgloss.NewStyle().Background(lipgloss.Color(props.Theme.Highlight))
	highlightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(props.Theme.Highlight))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(props.Theme.ForegroundDim))

	var rendered []string
	for _, line := range visible {
		text := line.text

		// Prepend gutter with relative line numbers when enabled.
		if props.LineNumbers {
			var gutter string
			if line.flatIdx >= 0 {
				rel := line.flatIdx - tl.Cursor
				if rel < 0 {
					rel = -rel
				}
				num := fmt.Sprintf("%3d ", rel)
				if rel == 0 {
					if line.highlighted {
						gutter = lipgloss.NewStyle().
							Foreground(lipgloss.Color(props.Theme.Highlight)).
							Background(lipgloss.Color(props.Theme.Highlight)).
							Render(num)
					} else {
						gutter = highlightStyle.Render(num)
					}
				} else {
					gutter = dimStyle.Render(num)
				}
			} else {
				// Expanded content lines: blank gutter
				gutter = dimStyle.Render("    ")
			}
			text = gutter + text
		}

		// For highlighted rows, the background is already baked into each
		// segment. We just need to pad to full width with the same background.
		if line.highlighted {
			displayWidth := lipgloss.Width(text)
			if displayWidth < props.Width {
				text += hlBgStyle.Render(strings.Repeat(" ", props.Width-displayWidth))
			}
		}
		rendered = append(rendered, text)
	}

	// Overlay pending count buffer in bottom-right corner.
	if tl.CountBuffer != "" && len(rendered) > 0 {
		lastIdx := len(rendered) - 1
		countStr := dimStyle.Render(tl.CountBuffer)
		countWidth := lipgloss.Width(countStr)
		lastLine := rendered[lastIdx]
		lastLineWidth := lipgloss.Width(lastLine)
		padding := props.Width - lastLineWidth - countWidth
		if padding > 0 {
			rendered[lastIdx] = lastLine + strings.Repeat(" ", padding) + countStr
		} else {
			// Overwrite the end of the last line with the count indicator.
			rendered[lastIdx] = lastLine + " " + countStr
		}
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
	tl.ExitSubScroll()
}

// ensureCursorVisible adjusts scroll to keep the cursor in view.
func (tl *Timeline) ensureCursorVisible(props TimelineProps) {
	lineStart, lc := tl.effectiveLineRange(props)
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
	total := tl.effectiveTotalLines(props)
	if total > props.Height {
		tl.Scroll = total - props.Height
	} else {
		tl.Scroll = 0
	}
}

// effectiveTotalLines returns total rendered lines, accounting for sub-scroll
// viewport capping on the active sub-scroll item.
func (tl *Timeline) effectiveTotalLines(props TimelineProps) int {
	if !tl.InSubScroll() {
		return TotalLines(props.Items, props.CompactView)
	}
	return tl.totalLinesWithCap(props)
}

// effectiveLineRange returns the line range for the cursor item, accounting
// for sub-scroll viewport capping.
func (tl *Timeline) effectiveLineRange(props TimelineProps) (lineStart int, lineCount int) {
	if !tl.InSubScroll() {
		return FlatCursorLineRange(props.Items, tl.Cursor, props.CompactView)
	}
	return tl.lineRangeWithCap(props)
}

// totalLinesWithCap computes total lines with the sub-scroll item capped.
func (tl *Timeline) totalLinesWithCap(props TimelineProps) int {
	total := 0
	flatPos := 0
	for _, item := range props.Items {
		switch it := item.(type) {
		case *model.TextBlock:
			total += ItemLineCount(it, props.CompactView)
			flatPos++
		case *model.ToolCall:
			if flatPos == tl.SubScrollIdx {
				total += toolCallLineCountCapped(it, props.Height)
			} else {
				total += toolCallLineCount(it)
			}
			flatPos++
		case *model.ToolCallGroup:
			total++ // header
			flatPos++
			if it.Expanded {
				for _, child := range it.Children {
					if flatPos == tl.SubScrollIdx {
						total += toolCallLineCountCapped(child, props.Height)
					} else {
						total += toolCallLineCount(child)
					}
					flatPos++
				}
			}
		}
	}
	return total
}

// lineRangeWithCap computes the line range for the cursor with sub-scroll capping.
func (tl *Timeline) lineRangeWithCap(props TimelineProps) (lineStart int, lineCount int) {
	line := 0
	pos := 0
	for _, item := range props.Items {
		switch it := item.(type) {
		case *model.TextBlock:
			lc := ItemLineCount(it, props.CompactView)
			if pos == tl.Cursor {
				return line, lc
			}
			line += lc
			pos++
		case *model.ToolCall:
			var lc int
			if pos == tl.SubScrollIdx {
				lc = toolCallLineCountCapped(it, props.Height)
			} else {
				lc = toolCallLineCount(it)
			}
			if pos == tl.Cursor {
				return line, lc
			}
			line += lc
			pos++
		case *model.ToolCallGroup:
			if pos == tl.Cursor {
				return line, 1
			}
			line++
			pos++
			if it.Expanded {
				for _, child := range it.Children {
					var clc int
					if pos == tl.SubScrollIdx {
						clc = toolCallLineCountCapped(child, props.Height)
					} else {
						clc = toolCallLineCount(child)
					}
					if pos == tl.Cursor {
						return line, clc
					}
					line += clc
					pos++
				}
			}
		}
	}
	return line, 1
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
// If the row was already selected, it triggers expand/collapse (like Enter).
// Returns true if the click was handled.
func (tl *Timeline) ClickRow(row int, props TimelineProps) bool {
	line := tl.Scroll + row
	total := TotalLines(props.Items, props.CompactView)
	if line < 0 || line >= total {
		return false
	}
	oldCursor := tl.Cursor
	tl.Cursor = LineToFlatCursor(props.Items, line, props.CompactView)
	maxPos := FlatCursorCount(props.Items) - 1
	tl.AutoFollow.OnManualMove(tl.Cursor >= maxPos)
	if tl.Cursor == oldCursor {
		tl.handleEnter(props)
	}
	return true
}

// ClickRowSubScroll handles a mouse click while in sub-scroll mode.
// It determines what was clicked relative to the sub-scrolled item:
//   - Summary row: exit sub-scroll and collapse the item
//   - Expanded content area: no-op
//   - Any other row: exit sub-scroll and select the clicked row
func (tl *Timeline) ClickRowSubScroll(row int, props TimelineProps) {
	line := tl.Scroll + row
	total := tl.effectiveTotalLines(props)
	if line < 0 || line >= total {
		return
	}

	// Find the sub-scrolled item's line range (with viewport cap).
	subStart, subCount := tl.lineRangeWithCap(props)

	switch {
	case line == subStart:
		// Clicked the summary row: collapse and exit sub-scroll.
		tc := tl.subScrollToolCall(props)
		if tc != nil {
			tc.Expanded = false
		}
		tl.ExitSubScroll()
		tl.ensureCursorVisible(props)
	case line > subStart && line < subStart+subCount:
		// Clicked inside expanded content area: no-op.
	default:
		// Clicked another row: resolve flat cursor using capped geometry
		// (before exiting sub-scroll, since the line layout changes after exit).
		targetCursor := tl.lineToFlatCursorWithCap(line, props)
		tl.ExitSubScroll()
		tl.Cursor = targetCursor
		maxPos := FlatCursorCount(props.Items) - 1
		tl.AutoFollow.OnManualMove(tl.Cursor >= maxPos)
		tl.ensureCursorVisible(props)
	}
}

// lineToFlatCursorWithCap maps a rendered line to a flat cursor position
// using the capped line count for the sub-scrolled item.
func (tl *Timeline) lineToFlatCursorWithCap(line int, props TimelineProps) int {
	currentLine := 0
	flatPos := 0
	for _, item := range props.Items {
		switch it := item.(type) {
		case *model.TextBlock:
			lc := ItemLineCount(it, props.CompactView)
			if line < currentLine+lc {
				return flatPos
			}
			currentLine += lc
			flatPos++
		case *model.ToolCall:
			var lc int
			if flatPos == tl.SubScrollIdx {
				lc = toolCallLineCountCapped(it, props.Height)
			} else {
				lc = toolCallLineCount(it)
			}
			if line < currentLine+lc {
				return flatPos
			}
			currentLine += lc
			flatPos++
		case *model.ToolCallGroup:
			if line < currentLine+1 {
				return flatPos
			}
			currentLine++
			flatPos++
			if it.Expanded {
				for _, child := range it.Children {
					var clc int
					if flatPos == tl.SubScrollIdx {
						clc = toolCallLineCountCapped(child, props.Height)
					} else {
						clc = toolCallLineCount(child)
					}
					if line < currentLine+clc {
						return flatPos
					}
					currentLine += clc
					flatPos++
				}
			}
		}
	}
	total := FlatCursorCount(props.Items)
	if total > 0 {
		return total - 1
	}
	return 0
}

// clampScroll ensures scroll doesn't exceed the maximum.
func (tl *Timeline) clampScroll(props TimelineProps) {
	total := tl.effectiveTotalLines(props)
	maxScroll := total - props.Height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if tl.Scroll > maxScroll {
		tl.Scroll = maxScroll
	}
}

// renderTextBlockLines renders a text block as one or more display lines.
// When highlightBg is non-empty, each line gets the background color baked in.
func renderTextBlockLines(tb *model.TextBlock, width int, compactView bool, th theme.Theme, highlightBg string) []string {
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
	if highlightBg != "" {
		textStyle = textStyle.Background(lipgloss.Color(highlightBg))
	}

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

// renderToolCallLine renders a single tool call row. When highlightBg is
// non-empty, each segment gets the background color baked in so that inner
// ANSI resets don't interrupt the highlight.
//
//nolint:unparam // nameWidth is kept as a parameter for consistency with renderGroupHeaderLine
func renderToolCallLine(tc *model.ToolCall, nameWidth, summaryWidth, durWidth int, compactView bool, th theme.Theme, highlightBg string, workDir string) string {
	icon := ToolIcon(tc.Name)
	isKnown := IsKnownTool(tc.Name)

	// Build token attribution string if tokens are available.
	tokenStr := ""
	if tc.InputTokens > 0 || tc.CacheReadTokens > 0 {
		tokenStr = fmt.Sprintf("[↑%s ⚡%s]", FormatTokens(tc.InputTokens), FormatTokens(tc.CacheReadTokens))
	}
	tokenWidth := len(tokenStr)
	if tokenWidth > 0 {
		tokenWidth++ // account for leading space
	}

	summary := TrimSummaryPath(tc.Summary, tc.Name, workDir)
	lineInfo := ""
	if tc.LineInfo != "" && tc.Status != model.ToolCallRunning {
		lineInfo = " " + tc.LineInfo
	}
	combined := summary + lineInfo
	// Reduce summary width to make room for token info.
	adjSummaryWidth := summaryWidth - tokenWidth
	if adjSummaryWidth < 10 {
		adjSummaryWidth = 10
	}
	if len(combined) > adjSummaryWidth {
		combined = combined[:adjSummaryWidth-3] + "..."
	}
	combined = fmt.Sprintf("%-*s", adjSummaryWidth, combined)

	var nameColor, durColor string
	switch tc.Status {
	case model.ToolCallDone:
		nameColor = th.ToolNameSuccess
		durColor = th.DurationSuccess
	case model.ToolCallError:
		nameColor = th.ToolNameError
		durColor = th.DurationError
	default:
		nameColor = th.ToolNameRunning
		durColor = th.DurationRunning
	}

	dur := FormatDuration(tc.Duration, tc.Status == model.ToolCallRunning)
	dur = fmt.Sprintf("%*s", durWidth, dur)

	// Helper to build a style with foreground and optional highlight background.
	styled := func(fg, text string) string {
		s := lipgloss.NewStyle().Foreground(lipgloss.Color(fg))
		if highlightBg != "" {
			s = s.Background(lipgloss.Color(highlightBg))
		}
		return s.Render(text)
	}

	styledIcon := styled(nameColor, icon)
	styledSummary := styled(th.ToolSummary, combined)
	styledDur := styled(durColor, dur)

	var styledTokens string
	if tokenStr != "" {
		styledTokens = " " + styled(th.ForegroundDim, tokenStr)
	}

	// When highlighting, plain spaces between segments need the background too.
	space := " "
	indent := "  "
	if highlightBg != "" {
		bgStyle := lipgloss.NewStyle().Background(lipgloss.Color(highlightBg))
		space = bgStyle.Render(" ")
		indent = bgStyle.Render("  ")
	}

	showName := !compactView || !isKnown
	if showName {
		name := fmt.Sprintf("%-*s", nameWidth, tc.Name)
		if len(tc.Name) > nameWidth {
			name = tc.Name[:nameWidth]
		}
		styledName := styled(nameColor, name)
		return fmt.Sprintf("%s%s%s%s%s%s%s%s%s", indent, styledIcon, space, styledName, space, styledSummary, styledTokens, space, styledDur)
	}

	return fmt.Sprintf("%s%s%s%s%s%s%s", indent, styledIcon, space, styledSummary, styledTokens, space, styledDur)
}

// renderGroupHeaderLine renders a tool call group header row. When highlightBg
// is non-empty, each segment gets the background color baked in.
func renderGroupHeaderLine(g *model.ToolCallGroup, nameWidth, summaryWidth, durWidth int, compactView bool, th theme.Theme, highlightBg string) string {
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

	var nameColor, durColor string
	switch status {
	case model.ToolCallDone:
		nameColor = th.ToolNameSuccess
		durColor = th.DurationSuccess
	case model.ToolCallError:
		nameColor = th.ToolNameError
		durColor = th.DurationError
	default:
		nameColor = th.ToolNameRunning
		durColor = th.DurationRunning
	}

	dur := FormatDuration(g.GroupDuration(), status == model.ToolCallRunning)
	dur = fmt.Sprintf("%*s", durWidth, dur)

	styled := func(fg, text string) string {
		s := lipgloss.NewStyle().Foreground(lipgloss.Color(fg))
		if highlightBg != "" {
			s = s.Background(lipgloss.Color(highlightBg))
		}
		return s.Render(text)
	}

	styledIcon := styled(nameColor, icon)
	styledSummary := styled(th.ToolSummary, summary)
	styledDur := styled(durColor, dur)

	space := " "
	indent := "  "
	if highlightBg != "" {
		bgStyle := lipgloss.NewStyle().Background(lipgloss.Color(highlightBg))
		space = bgStyle.Render(" ")
		indent = bgStyle.Render("  ")
	}

	showName := !compactView || !isKnown
	if showName {
		name := fmt.Sprintf("%-*s", nameWidth, g.ToolName)
		if len(g.ToolName) > nameWidth {
			name = g.ToolName[:nameWidth]
		}
		styledName := styled(nameColor, name)
		return fmt.Sprintf("%s%s%s%s%s%s%s%s", indent, styledIcon, space, styledName, space, styledSummary, space, styledDur)
	}

	return fmt.Sprintf("%s%s%s%s%s%s", indent, styledIcon, space, styledSummary, space, styledDur)
}
