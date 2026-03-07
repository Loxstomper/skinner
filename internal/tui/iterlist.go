package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/loxstomper/skinner/internal/model"
	"github.com/loxstomper/skinner/internal/theme"
)

// IterList is the left pane component that displays the iteration list.
// It owns its cursor position and auto-follow state.
type IterList struct {
	Cursor     int
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
		il.AutoFollow.OnManualMove(atEnd())

	case "k", "up":
		if il.Cursor > 0 {
			il.Cursor--
		}
		il.AutoFollow.OnManualMove(atEnd())

	case "g":
		// gg handled by root — root calls JumpToTop
	case "G", "end":
		if count > 0 {
			il.Cursor = count - 1
		}
		il.AutoFollow.JumpToEnd()

	case "home":
		il.Cursor = 0
		il.AutoFollow.OnManualMove(atEnd())

	case "pgdown":
		il.Cursor += props.Height
		if il.Cursor >= count {
			il.Cursor = count - 1
		}
		if il.Cursor < 0 {
			il.Cursor = 0
		}
		il.AutoFollow.OnManualMove(atEnd())

	case "pgup":
		il.Cursor -= props.Height
		if il.Cursor < 0 {
			il.Cursor = 0
		}
		il.AutoFollow.OnManualMove(atEnd())

	case "enter":
		// Enter on left pane focuses right pane — handled by root
	}

	return nil
}

// View renders the iteration list.
func (il *IterList) View(props IterListProps) string {
	style := lipgloss.NewStyle().Width(props.Width).Height(props.Height)
	highlight := lipgloss.NewStyle().Background(lipgloss.Color(props.Theme.Highlight))

	var lines []string
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

		dur := FormatDuration(iter.Duration, iter.Status == model.IterationRunning)
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
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	return style.Render(content)
}

// OnNewIteration updates auto-follow state when a new iteration is added.
// If auto-follow is active, the cursor moves to the latest iteration.
func (il *IterList) OnNewIteration(count int) {
	il.AutoFollow.OnNewItem()
	if il.AutoFollow.Following() && count > 0 {
		il.Cursor = count - 1
	}
}

// SelectedIndex returns the currently selected iteration index.
func (il *IterList) SelectedIndex() int {
	return il.Cursor
}

// JumpToTop moves the cursor to the first iteration.
func (il *IterList) JumpToTop() {
	il.Cursor = 0
	il.AutoFollow.OnManualMove(false)
}

// JumpToBottom moves the cursor to the last iteration and resumes auto-follow.
func (il *IterList) JumpToBottom(count int) {
	if count > 0 {
		il.Cursor = count - 1
	}
	il.AutoFollow.JumpToEnd()
}
