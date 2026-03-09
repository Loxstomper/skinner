package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/loxstomper/skinner/internal/model"
	"github.com/loxstomper/skinner/internal/theme"
)

// IterList is the left pane component that displays the iteration list.
// It owns its cursor position, scroll offset, and auto-follow state.
type IterList struct {
	Cursor     int
	Scroll     int
	AutoFollow AutoFollow
}

// NewIterList creates a new IterList starting at position 0 with auto-follow enabled.
func NewIterList() IterList {
	return IterList{
		AutoFollow: NewAutoFollow(),
	}
}

// IterListProps contains the data needed to render the iteration list.
type IterListProps struct {
	Iterations []model.Iteration
	Runs       []model.Run
	Width      int
	Height     int
	Focused    bool
	Theme      theme.Theme
}

// HandleAction processes a resolved action for the iteration list.
func (il *IterList) HandleAction(action string, props IterListProps) {
	count := len(props.Iterations)
	atEnd := func() bool { return il.Cursor == count-1 }

	switch action {
	case "move_down":
		if il.Cursor < count-1 {
			il.Cursor++
		}
		il.ensureCursorVisible(props)
		il.AutoFollow.OnManualMove(atEnd())

	case "move_up":
		if il.Cursor > 0 {
			il.Cursor--
		}
		il.ensureCursorVisible(props)
		il.AutoFollow.OnManualMove(atEnd())

	case "jump_bottom":
		if count > 0 {
			il.Cursor = count - 1
		}
		il.ensureCursorVisible(props)
		il.AutoFollow.JumpToEnd()

	case "jump_top":
		il.Cursor = 0
		il.Scroll = 0
		il.AutoFollow.OnManualMove(atEnd())

	case "page_down":
		il.Cursor += props.Height
		if il.Cursor >= count {
			il.Cursor = count - 1
		}
		if il.Cursor < 0 {
			il.Cursor = 0
		}
		il.ensureCursorVisible(props)
		il.AutoFollow.OnManualMove(atEnd())

	case "page_up":
		il.Cursor -= props.Height
		if il.Cursor < 0 {
			il.Cursor = 0
		}
		il.ensureCursorVisible(props)
		il.AutoFollow.OnManualMove(atEnd())
	}
}

// View renders the iteration list, showing only the visible slice based on scroll offset.
// Run separators are inserted at boundaries between runs (after the first run).
func (il *IterList) View(props IterListProps) string {
	style := lipgloss.NewStyle().Width(props.Width).Height(props.Height)
	highlight := lipgloss.NewStyle().Background(lipgloss.Color(props.Theme.Highlight))

	// Build separator lookup: iteration index → prompt name
	separatorAt := make(map[int]string)
	for i := 1; i < len(props.Runs); i++ {
		separatorAt[props.Runs[i].StartIndex] = props.Runs[i].PromptName
	}

	// Build all display lines (iterations + separators)
	var allLines []string
	for i, iter := range props.Iterations {
		// Insert separator before this iteration if it starts a new run
		if name, ok := separatorAt[i]; ok {
			allLines = append(allLines, renderRunSeparator(name, props.Width, props.Theme))
		}

		var statusIcon string
		var statusColor, iterColor string
		switch iter.Status {
		case model.IterationRunning:
			statusIcon = "⟳"
			statusColor = props.Theme.StatusRunning
			iterColor = props.Theme.IterRunning
		case model.IterationCompleted:
			statusIcon = "✓"
			statusColor = props.Theme.StatusSuccess
			iterColor = props.Theme.IterSuccess
		case model.IterationFailed:
			statusIcon = "✗"
			statusColor = props.Theme.StatusError
			iterColor = props.Theme.IterError
		}

		var dur string
		if iter.Status == model.IterationRunning {
			// Show live elapsed time for running iterations
			elapsed := time.Since(iter.StartTime)
			dur = FormatDurationValue(elapsed)
		} else {
			dur = FormatDurationValue(iter.Duration)
		}
		styledIcon := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render(statusIcon)
		iterText := fmt.Sprintf("  Iter %d  ", iter.Index+1)
		metaText := fmt.Sprintf("  (%s)", dur)

		iterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(iterColor))
		line := iterStyle.Render(iterText) + styledIcon + iterStyle.Render(metaText)

		if i == il.Cursor {
			displayWidth := lipgloss.Width(line)
			if displayWidth < props.Width {
				line += strings.Repeat(" ", props.Width-displayWidth)
			}
			line = highlight.Render(line)
		}
		allLines = append(allLines, line)
	}

	// Apply scroll: render only the visible slice
	start := il.Scroll
	if start >= len(allLines) {
		start = len(allLines) - 1
	}
	if start < 0 {
		start = 0
	}
	end := start + props.Height
	if end > len(allLines) {
		end = len(allLines)
	}

	visible := allLines[start:end]
	content := strings.Join(visible, "\n")
	return style.Render(content)
}

// OnNewIteration updates auto-follow state when a new iteration is added.
// If auto-follow is active, the cursor moves to the latest iteration.
func (il *IterList) OnNewIteration(count int, height int, runs []model.Run) {
	il.AutoFollow.OnNewItem()
	if il.AutoFollow.Following() && count > 0 {
		il.Cursor = count - 1
		il.clampScroll(totalDisplayLines(count, runs), height)
		il.ensureCursorVisibleSimple(count, height, runs)
	}
}

// SelectedIndex returns the currently selected iteration index.
func (il *IterList) SelectedIndex() int {
	return il.Cursor
}

// JumpToTop moves the cursor to the first iteration.
func (il *IterList) JumpToTop() {
	il.Cursor = 0
	il.Scroll = 0
	il.AutoFollow.OnManualMove(false)
}

// JumpToBottom moves the cursor to the last iteration and resumes auto-follow.
func (il *IterList) JumpToBottom(count int, height int, runs []model.Run) {
	if count > 0 {
		il.Cursor = count - 1
		il.ensureCursorVisibleSimple(count, height, runs)
	}
	il.AutoFollow.JumpToEnd()
}

// ensureCursorVisible adjusts scroll so the cursor row is within the viewport.
func (il *IterList) ensureCursorVisible(props IterListProps) {
	il.ensureCursorVisibleSimple(len(props.Iterations), props.Height, props.Runs)
}

// ensureCursorVisibleSimple adjusts scroll using count, height, and runs directly.
func (il *IterList) ensureCursorVisibleSimple(count int, height int, runs []model.Run) {
	row := displayRowForIter(il.Cursor, runs)
	if row < il.Scroll {
		il.Scroll = row
	}
	if row >= il.Scroll+height {
		il.Scroll = row - height + 1
	}
	il.clampScroll(totalDisplayLines(count, runs), height)
}

// ScrollBy adjusts the scroll offset by delta lines (positive = down, negative = up).
// It clamps the result and pauses auto-follow.
func (il *IterList) ScrollBy(delta int, count int, height int, runs []model.Run) {
	il.Scroll += delta
	il.clampScroll(totalDisplayLines(count, runs), height)
	il.AutoFollow.OnManualMove(false)
}

// ClickRow handles a mouse click on the given pane-relative row.
// It converts the display row to an iteration index (skipping separators),
// and returns true if the cursor changed.
func (il *IterList) ClickRow(row int, count int, height int, runs []model.Run) bool {
	displayRow := il.Scroll + row
	isSep, iterIdx := displayRowToIterIndex(displayRow, count, runs)
	if isSep || iterIdx < 0 || iterIdx >= count {
		return false
	}
	il.Cursor = iterIdx
	il.AutoFollow.OnManualMove(il.Cursor == count-1)
	return true
}

// clampScroll ensures scroll doesn't exceed the maximum valid offset.
// displayCount is the total number of display lines (iterations + separators).
func (il *IterList) clampScroll(displayCount int, height int) {
	maxScroll := displayCount - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if il.Scroll > maxScroll {
		il.Scroll = maxScroll
	}
	if il.Scroll < 0 {
		il.Scroll = 0
	}
}

// separatorsBefore returns the number of run separators that appear before
// the given iteration index. The first run never has a separator.
func separatorsBefore(iterIndex int, runs []model.Run) int {
	n := 0
	for i := 1; i < len(runs); i++ {
		if runs[i].StartIndex <= iterIndex {
			n++
		}
	}
	return n
}

// totalSeparators returns the total number of run separator lines.
func totalSeparators(runs []model.Run) int {
	if len(runs) <= 1 {
		return 0
	}
	return len(runs) - 1
}

// displayRowForIter converts an iteration index to a display row index,
// accounting for separator lines above it.
func displayRowForIter(iterIndex int, runs []model.Run) int {
	return iterIndex + separatorsBefore(iterIndex, runs)
}

// totalDisplayLines returns the total number of display lines (iterations + separators).
func totalDisplayLines(iterCount int, runs []model.Run) int {
	return iterCount + totalSeparators(runs)
}

// displayRowToIterIndex converts a display row back to an iteration index.
// Returns (true, -1) if the display row is a separator line.
func displayRowToIterIndex(displayRow int, iterCount int, runs []model.Run) (isSeparator bool, iterIndex int) {
	seps := 0
	for i := 1; i < len(runs); i++ {
		sepRow := runs[i].StartIndex + seps
		if displayRow == sepRow {
			return true, -1
		}
		if displayRow < sepRow {
			break
		}
		seps++
	}
	idx := displayRow - seps
	if idx < 0 || idx >= iterCount {
		return false, -1
	}
	return false, idx
}

// renderRunSeparator renders a "── NAME ────────" separator line.
func renderRunSeparator(name string, width int, th theme.Theme) string {
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.ForegroundDim))
	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Foreground)).Bold(true)

	styledName := nameStyle.Render(name)
	prefix := "── "
	suffix := " "

	usedWidth := lipgloss.Width(prefix) + lipgloss.Width(styledName) + lipgloss.Width(suffix)
	trailing := width - usedWidth
	if trailing < 0 {
		trailing = 0
	}

	return dimStyle.Render(prefix) + styledName + dimStyle.Render(suffix+strings.Repeat("─", trailing))
}
