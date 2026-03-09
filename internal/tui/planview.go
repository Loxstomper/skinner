package tui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/loxstomper/skinner/internal/theme"
)

// PlanViewProps contains the data needed to render the plan content view.
type PlanViewProps struct {
	Filename string // e.g. "IMPLEMENTATION_PLAN.md"
	Dir      string // working directory containing the file
	Width    int
	Height   int
	Scroll   int
	Focused  bool
	Theme    theme.Theme
}

// RenderPlanView renders the plan content view for the right pane.
// It shows a title bar with the centered filename, followed by glamour-rendered
// markdown content. Returns the rendered string and the total number of content lines.
func RenderPlanView(props PlanViewProps) (string, int) {
	if props.Width < 1 || props.Height < 1 {
		return "", 0
	}

	// Title bar
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(props.Theme.Foreground)).
		Bold(true).
		Width(props.Width).
		Align(lipgloss.Center)
	title := titleStyle.Render(props.Filename)

	contentHeight := props.Height - 1 // subtract title row
	if contentHeight < 1 {
		return lipgloss.NewStyle().Width(props.Width).Height(props.Height).Render(title), 0
	}

	// Handle empty filename
	if props.Filename == "" {
		dimStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(props.Theme.ForegroundDim))
		msg := dimStyle.Render("  No plan selected")
		lines := []string{title, msg}
		for len(lines) < props.Height {
			lines = append(lines, "")
		}
		return lipgloss.NewStyle().Width(props.Width).Height(props.Height).
			Render(strings.Join(lines, "\n")), 0
	}

	// Read file content
	filePath := filepath.Join(props.Dir, props.Filename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		dimStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(props.Theme.ForegroundDim))
		msg := dimStyle.Render("  File not found")
		lines := []string{title, msg}
		for len(lines) < props.Height {
			lines = append(lines, "")
		}
		return lipgloss.NewStyle().Width(props.Width).Height(props.Height).
			Render(strings.Join(lines, "\n")), 0
	}

	// Render markdown with glamour
	rendered := renderMarkdown(string(data), props.Width)

	// Split into lines for scrolling
	contentLines := strings.Split(rendered, "\n")
	// Remove trailing empty line that glamour often adds
	for len(contentLines) > 0 && contentLines[len(contentLines)-1] == "" {
		contentLines = contentLines[:len(contentLines)-1]
	}
	totalLines := len(contentLines)

	// Apply scroll
	scroll := props.Scroll
	if scroll >= totalLines {
		scroll = totalLines - 1
	}
	if scroll < 0 {
		scroll = 0
	}
	end := scroll + contentHeight
	if end > totalLines {
		end = totalLines
	}

	var visible []string
	if scroll < totalLines {
		visible = contentLines[scroll:end]
	}

	// Build output
	lines := []string{title}
	lines = append(lines, visible...)

	// Pad to fill height
	for len(lines) < props.Height {
		lines = append(lines, "")
	}

	return lipgloss.NewStyle().Width(props.Width).Height(props.Height).
		Render(strings.Join(lines, "\n")), totalLines
}

// renderMarkdown renders markdown content using glamour with auto style.
func renderMarkdown(content string, width int) string {
	// glamour word-wraps to width; subtract 2 for glamour's own margins
	renderWidth := width
	if renderWidth < 10 {
		renderWidth = 10
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(renderWidth),
	)
	if err != nil {
		return content // fallback to raw content
	}

	rendered, err := renderer.Render(content)
	if err != nil {
		return content // fallback to raw content
	}

	return strings.TrimSpace(rendered)
}

// ClampPlanScroll ensures the plan view scroll doesn't exceed bounds.
func ClampPlanScroll(scroll, totalLines, viewHeight int) int {
	contentHeight := viewHeight - 1 // subtract title row
	if contentHeight < 1 {
		return 0
	}
	maxScroll := totalLines - contentHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}
	return scroll
}
