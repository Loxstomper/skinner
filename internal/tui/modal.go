package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/loxstomper/skinner/internal/theme"
)

// modalType identifies which modal is currently displayed.
type modalType int

const (
	modalNone modalType = iota
	modalQuitConfirm
	modalHelp
)

// RenderQuitConfirmModal renders a centered quit confirmation overlay.
func RenderQuitConfirmModal(width, height int, th theme.Theme) string {
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(th.ForegroundDim)).
		Padding(1, 3)

	textStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.Foreground))

	highlightStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.Highlight))

	body := textStyle.Render("Are you sure you want") + "\n" +
		textStyle.Render("to quit?") + "\n" +
		"\n" +
		highlightStyle.Render("y") + textStyle.Render(" - yes    ") +
		highlightStyle.Render("n") + textStyle.Render(" - cancel")

	modal := borderStyle.Render(body)

	return centerOverlay(modal, width, height)
}

// centerOverlay places a rendered block in the center of the terminal.
func centerOverlay(content string, termWidth, termHeight int) string {
	lines := strings.Split(content, "\n")
	contentHeight := len(lines)

	// Determine the max line width (visible width).
	contentWidth := 0
	for _, line := range lines {
		w := lipgloss.Width(line)
		if w > contentWidth {
			contentWidth = w
		}
	}

	// Vertical centering: pad above with empty lines.
	topPad := (termHeight - contentHeight) / 2
	if topPad < 0 {
		topPad = 0
	}

	// Horizontal centering: pad each line with spaces.
	leftPad := (termWidth - contentWidth) / 2
	if leftPad < 0 {
		leftPad = 0
	}

	var sb strings.Builder
	padding := strings.Repeat(" ", leftPad)

	for i := 0; i < topPad; i++ {
		sb.WriteString(strings.Repeat(" ", termWidth))
		sb.WriteByte('\n')
	}

	for i, line := range lines {
		sb.WriteString(padding)
		sb.WriteString(line)
		// Pad right to fill the full width.
		lineW := lipgloss.Width(line)
		if remaining := termWidth - leftPad - lineW; remaining > 0 {
			sb.WriteString(strings.Repeat(" ", remaining))
		}
		if i < len(lines)-1 {
			sb.WriteByte('\n')
		}
	}

	// Fill remaining rows below the modal.
	bottomRows := termHeight - topPad - contentHeight
	for i := 0; i < bottomRows; i++ {
		sb.WriteByte('\n')
		sb.WriteString(strings.Repeat(" ", termWidth))
	}

	return sb.String()
}
