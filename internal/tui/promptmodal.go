package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/loxstomper/skinner/internal/theme"
)

// PromptModalProps holds the parameters for rendering a prompt read modal.
type PromptModalProps struct {
	Filename string
	Content  string
	Scroll   int
	Width    int
	Height   int
	Theme    theme.Theme
}

// RenderPromptReadModal renders a centered modal showing prompt file content
// with line numbers in a dimmed gutter.
func RenderPromptReadModal(props PromptModalProps) string {
	th := props.Theme

	// Modal dimensions: ~80% of terminal
	modalWidth := props.Width * 80 / 100
	if modalWidth < 40 {
		modalWidth = 40
	}
	modalHeight := props.Height * 80 / 100
	if modalHeight < 10 {
		modalHeight = 10
	}

	// Border + padding consume space: 2 (left/right border) + 2 (left padding) + 2 (right padding)
	borderH := 2  // top + bottom border
	paddingV := 2 // top + bottom padding (1 each)
	paddingH := 4 // left + right padding (2 each)
	borderW := 2  // left + right border

	innerWidth := modalWidth - borderW - paddingH
	if innerWidth < 20 {
		innerWidth = 20
	}
	// Content height: modal height minus border, padding, and footer line
	contentHeight := modalHeight - borderH - paddingV - 1 // -1 for footer blank line
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Split content into lines
	lines := strings.Split(props.Content, "\n")
	// Remove trailing empty line from final newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	// Clamp scroll
	scroll := props.Scroll
	maxScroll := len(lines) - contentHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}

	// Determine gutter width from max line number
	totalLines := len(lines)
	if totalLines == 0 {
		totalLines = 1
	}
	gutterWidth := len(fmt.Sprintf("%d", totalLines))
	if gutterWidth < 2 {
		gutterWidth = 2
	}
	// Gutter format: right-aligned number + space separator
	gutterFmt := fmt.Sprintf("%%%dd ", gutterWidth)
	gutterTotal := gutterWidth + 1 // number + space

	textWidth := innerWidth - gutterTotal
	if textWidth < 10 {
		textWidth = 10
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.ForegroundDim))
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Foreground))

	// Build visible lines
	var contentLines []string
	endLine := scroll + contentHeight
	if endLine > len(lines) {
		endLine = len(lines)
	}

	for i := scroll; i < endLine; i++ {
		lineNum := i + 1 // 1-based
		gutter := dimStyle.Render(fmt.Sprintf(gutterFmt, lineNum))

		// Truncate line if too wide
		line := lines[i]
		if len(line) > textWidth {
			line = line[:textWidth]
		}
		text := textStyle.Render(line)
		contentLines = append(contentLines, gutter+text)
	}

	// Pad remaining content rows
	for len(contentLines) < contentHeight {
		gutter := dimStyle.Render(strings.Repeat(" ", gutterTotal))
		contentLines = append(contentLines, gutter)
	}

	// Footer
	footer := dimStyle.Render("e to edit · esc to close")
	footerWidth := len("e to edit · esc to close")
	footerPad := (innerWidth - footerWidth) / 2
	if footerPad < 0 {
		footerPad = 0
	}

	// Combine body: content lines + blank + footer
	body := strings.Join(contentLines, "\n") + "\n" +
		strings.Repeat(" ", footerPad) + footer

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.Foreground)).
		Bold(true)
	title := titleStyle.Render(" " + props.Filename + " ")

	// Border style
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(th.ForegroundDim)).
		Padding(1, 2).
		Width(modalWidth - borderW) // set inner width

	modal := borderStyle.Render(body)

	// Inject title into the top border line
	modalLines := strings.Split(modal, "\n")
	if len(modalLines) > 0 {
		topBorder := modalLines[0]
		borderWidth := lipgloss.Width(topBorder)
		titleWidth := lipgloss.Width(title)
		insertPos := (borderWidth - titleWidth) / 2
		if insertPos > 1 {
			modalLines[0] = replaceInLine(topBorder, title, insertPos)
		}
	}
	modal = strings.Join(modalLines, "\n")

	return centerOverlay(modal, props.Width, props.Height)
}

// PromptModalMaxScroll returns the maximum scroll offset for given content and modal height.
func PromptModalMaxScroll(content string, termHeight int) int {
	lines := strings.Split(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	modalHeight := termHeight * 80 / 100
	if modalHeight < 10 {
		modalHeight = 10
	}
	contentHeight := modalHeight - 2 - 2 - 1 // border + padding + footer
	if contentHeight < 1 {
		contentHeight = 1
	}

	maxScroll := len(lines) - contentHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	return maxScroll
}
