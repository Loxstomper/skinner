package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
	Width      int
	Height     int
	Focused    bool
	Theme      theme.Theme
}

// Update handles key events for the iteration list.
// Returns a command (always nil for now) and whether the right pane cursor
// should be reset (when the selected iteration changes).
func (il *IterList) Update(msg tea.KeyMsg, props IterListProps) tea.Cmd {
	key := msg.String()
	count := len(props.Iterations)
	atEnd := func() bool { return il.Cursor == count-1 }

	switch key {
	case "j", "down":
		if il.Cursor < count-1 {
			il.Cursor++
		}
		il.ensureCursorVisible(props)
		il.AutoFollow.OnManualMove(atEnd())

	case "k", "up":
		if il.Cursor > 0 {
			il.Cursor--
		}
		il.ensureCursorVisible(props)
		il.AutoFollow.OnManualMove(atEnd())

	case "g":
		// gg handled by root — root calls JumpToTop
	case "G", "end":
		if count > 0 {
			il.Cursor = count - 1
		}
		il.ensureCursorVisible(props)
		il.AutoFollow.JumpToEnd()

	case "home":
		il.Cursor = 0
		il.Scroll = 0
		il.AutoFollow.OnManualMove(atEnd())

	case "pgdown":
		il.Cursor += props.Height
		if il.Cursor >= count {
			il.Cursor = count - 1
		}
		if il.Cursor < 0 {
			il.Cursor = 0
		}
		il.ensureCursorVisible(props)
		il.AutoFollow.OnManualMove(atEnd())

	case "pgup":
		il.Cursor -= props.Height
		if il.Cursor < 0 {
			il.Cursor = 0
		}
		il.ensureCursorVisible(props)
		il.AutoFollow.OnManualMove(atEnd())

	case "enter":
		// Enter on left pane focuses right pane — handled by root
	}

	return nil
}

// View renders the iteration list, showing only the visible slice based on scroll offset.
func (il *IterList) View(props IterListProps) string {
	style := lipgloss.NewStyle().Width(props.Width).Height(props.Height)
	highlight := lipgloss.NewStyle().Background(lipgloss.Color(props.Theme.Highlight))

	// Build all lines
	var allLines []string
	for i, iter := range props.Iterations {
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
			// Show live elapsed time with ... suffix per duration-tracking spec
			elapsed := time.Since(iter.StartTime)
			dur = FormatDurationValue(elapsed) + "..."
		} else {
			dur = FormatDurationValue(iter.Duration)
		}
		callCount := iter.ToolCallCount()

		styledIcon := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render(statusIcon)
		iterText := fmt.Sprintf("  Iter %d  ", iter.Index+1)
		metaText := fmt.Sprintf("  (%d calls, %s)", callCount, dur)

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
func (il *IterList) OnNewIteration(count int, height int) {
	il.AutoFollow.OnNewItem()
	if il.AutoFollow.Following() && count > 0 {
		il.Cursor = count - 1
		il.clampScroll(count, height)
		il.ensureCursorVisibleSimple(count, height)
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
func (il *IterList) JumpToBottom(count int, height int) {
	if count > 0 {
		il.Cursor = count - 1
		il.ensureCursorVisibleSimple(count, height)
	}
	il.AutoFollow.JumpToEnd()
}

// ensureCursorVisible adjusts scroll so the cursor row is within the viewport.
func (il *IterList) ensureCursorVisible(props IterListProps) {
	il.ensureCursorVisibleSimple(len(props.Iterations), props.Height)
}

// ensureCursorVisibleSimple adjusts scroll using count and height directly.
func (il *IterList) ensureCursorVisibleSimple(count int, height int) {
	if il.Cursor < il.Scroll {
		il.Scroll = il.Cursor
	}
	if il.Cursor >= il.Scroll+height {
		il.Scroll = il.Cursor - height + 1
	}
	il.clampScroll(count, height)
}

// ScrollBy adjusts the scroll offset by delta lines (positive = down, negative = up).
// It clamps the result, pauses auto-follow, and returns whether the scroll changed.
func (il *IterList) ScrollBy(delta int, count int, height int) {
	il.Scroll += delta
	il.clampScroll(count, height)
	il.AutoFollow.OnManualMove(false)
}

// ClickRow handles a mouse click on the given pane-relative row.
// It sets the cursor to scroll+row if valid, and returns true if the cursor changed.
func (il *IterList) ClickRow(row int, count int, height int) bool {
	target := il.Scroll + row
	if target < 0 || target >= count {
		return false
	}
	il.Cursor = target
	il.AutoFollow.OnManualMove(il.Cursor == count-1)
	return true
}

// clampScroll ensures scroll doesn't exceed the maximum valid offset.
func (il *IterList) clampScroll(count int, height int) {
	maxScroll := count - height
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
