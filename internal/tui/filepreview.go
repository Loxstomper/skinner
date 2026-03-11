package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/loxstomper/skinner/internal/theme"
)

// FilePreviewProps contains the data needed to render a file preview.
type FilePreviewProps struct {
	Path            string // relative path from CWD
	Dir             string // working directory (CWD)
	Width           int
	Height          int
	Scroll          int
	HScroll         int
	ShowLineNumbers bool
	ThemeName       string // for chroma style selection
	Theme           theme.Theme
}

// FilePreviewResult contains the render output and metadata.
type FilePreviewResult struct {
	Content    string
	TotalLines int
}

// RenderFilePreview renders a file preview with title bar, syntax highlighting,
// and scroll support. Markdown files are rendered via glamour; source code via
// chroma; binary files show a placeholder message.
func RenderFilePreview(props FilePreviewProps) FilePreviewResult {
	if props.Width < 1 || props.Height < 1 {
		return FilePreviewResult{}
	}

	// Title bar: relative path centered, bold
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(props.Theme.Foreground)).
		Bold(true).
		Width(props.Width).
		Align(lipgloss.Center)
	title := titleStyle.Render(props.Path)

	contentHeight := props.Height - 1
	if contentHeight < 1 {
		return FilePreviewResult{
			Content: lipgloss.NewStyle().Width(props.Width).Height(props.Height).Render(title),
		}
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(props.Theme.ForegroundDim))

	// Empty path — directory or no selection
	if props.Path == "" {
		return buildPreviewResult(title, nil, contentHeight, props.Width)
	}

	fullPath := filepath.Join(props.Dir, props.Path)

	// Check if file exists
	info, err := os.Stat(fullPath)
	if err != nil {
		msg := dimStyle.Render("  File not found")
		return buildPreviewResult(title, []string{msg}, contentHeight, props.Width)
	}

	// Directory — empty preview
	if info.IsDir() {
		return buildPreviewResult(title, nil, contentHeight, props.Width)
	}

	// Binary check
	if IsBinary(fullPath) {
		msg := dimStyle.Render("  Binary file — preview not available")
		return buildPreviewResult(title, []string{msg}, contentHeight, props.Width)
	}

	// Read file content
	data, err := os.ReadFile(fullPath)
	if err != nil {
		msg := dimStyle.Render("  File not found")
		return buildPreviewResult(title, []string{msg}, contentHeight, props.Width)
	}

	content := string(data)

	// Markdown: glamour rendering, no line numbers
	if isMarkdown(props.Path) {
		return renderMarkdownPreview(title, content, props)
	}

	// Source code: chroma syntax highlighting
	return renderSourcePreview(title, content, props)
}

// renderMarkdownPreview renders markdown content via glamour.
func renderMarkdownPreview(title, content string, props FilePreviewProps) FilePreviewResult {
	contentHeight := props.Height - 1
	rendered := renderMarkdown(content, props.Width)

	contentLines := strings.Split(rendered, "\n")
	for len(contentLines) > 0 && contentLines[len(contentLines)-1] == "" {
		contentLines = contentLines[:len(contentLines)-1]
	}
	totalLines := len(contentLines)

	scroll := clampPreviewScrollVal(props.Scroll, totalLines, contentHeight)
	end := scroll + contentHeight
	if end > totalLines {
		end = totalLines
	}

	var visible []string
	if scroll < totalLines {
		visible = contentLines[scroll:end]
	}

	result := buildPreviewResult(title, visible, contentHeight, props.Width)
	result.TotalLines = totalLines
	return result
}

// renderSourcePreview renders source code with chroma syntax highlighting.
func renderSourcePreview(title, content string, props FilePreviewProps) FilePreviewResult {
	contentHeight := props.Height - 1
	sourceLines := strings.Split(content, "\n")
	if len(sourceLines) > 0 && sourceLines[len(sourceLines)-1] == "" {
		sourceLines = sourceLines[:len(sourceLines)-1]
	}
	totalLines := len(sourceLines)

	scroll := clampPreviewScrollVal(props.Scroll, totalLines, contentHeight)
	end := scroll + contentHeight
	if end > totalLines {
		end = totalLines
	}

	var visibleSource []string
	if scroll < totalLines {
		visibleSource = sourceLines[scroll:end]
	}

	lexer := getLexer(props.Path)
	chromaStyle := getChromaStyle(props.ThemeName)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(props.Theme.ForegroundDim))

	var gutterWidth int
	if props.ShowLineNumbers {
		maxNum := scroll + len(visibleSource)
		gutterWidth = digitCount(maxNum) + 1
		if gutterWidth < 4 {
			gutterWidth = 4
		}
	}

	var rendered []string
	for i, line := range visibleSource {
		lineNum := scroll + i + 1

		var gutterStr string
		if props.ShowLineNumbers {
			numStr := fmt.Sprintf("%*d ", gutterWidth-1, lineNum)
			gutterStr = dimStyle.Render(numStr)
		}

		displayLine := applyHScroll(line, props.HScroll)
		styledLine := renderPreviewLine(displayLine, lexer, chromaStyle, props.Theme)
		rendered = append(rendered, gutterStr+styledLine)
	}

	result := buildPreviewResult(title, rendered, contentHeight, props.Width)
	result.TotalLines = totalLines
	return result
}

// renderPreviewLine renders a single source line with chroma syntax highlighting.
func renderPreviewLine(line string, lexer chroma.Lexer, style *chroma.Style, th theme.Theme) string {
	spans := tokenizeLine(line, lexer, style)

	var b strings.Builder
	for _, span := range spans {
		if span.Fg != "" {
			s := lipgloss.NewStyle().Foreground(lipgloss.Color(span.Fg))
			b.WriteString(s.Render(span.Text))
		} else {
			s := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Foreground))
			b.WriteString(s.Render(span.Text))
		}
	}
	return b.String()
}

// applyHScroll trims characters from the left of a line for horizontal scrolling.
func applyHScroll(line string, hscroll int) string {
	if hscroll <= 0 {
		return line
	}
	runes := []rune(line)
	if hscroll >= len(runes) {
		return ""
	}
	return string(runes[hscroll:])
}

// buildPreviewResult builds the final preview output with title and padding.
func buildPreviewResult(title string, contentLines []string, contentHeight, width int) FilePreviewResult {
	lines := []string{title}
	lines = append(lines, contentLines...)

	totalHeight := contentHeight + 1
	for len(lines) < totalHeight {
		lines = append(lines, "")
	}

	return FilePreviewResult{
		Content:    lipgloss.NewStyle().Width(width).Height(totalHeight).Render(strings.Join(lines, "\n")),
		TotalLines: len(contentLines),
	}
}

// clampPreviewScrollVal ensures scroll is within valid bounds.
func clampPreviewScrollVal(scroll, totalLines, viewHeight int) int {
	maxScroll := totalLines - viewHeight
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

// isMarkdown returns true if the file path has a markdown extension.
func isMarkdown(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".markdown" || ext == ".mkd"
}

// ClampFilePreviewScroll ensures the file preview scroll doesn't exceed bounds.
func ClampFilePreviewScroll(scroll, totalLines, viewHeight int) int {
	contentHeight := viewHeight - 1
	if contentHeight < 1 {
		return 0
	}
	return clampPreviewScrollVal(scroll, totalLines, contentHeight)
}
