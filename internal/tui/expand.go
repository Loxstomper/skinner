package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/loxstomper/skinner/internal/model"
	"github.com/loxstomper/skinner/internal/theme"
)

// sideBySideMinWidth is the minimum terminal width for side-by-side diff layout.
const sideBySideMinWidth = 120

// expandedContentLines returns the content lines to display below an expanded
// tool call. The lines are plain text (not styled) — styling is applied by
// renderExpandedContentLine. Returns nil if no content is available or the tool
// call is not expanded. All content lines are returned without truncation;
// sub-scroll (3.2) handles viewport management for large content.
func expandedContentLines(tc *model.ToolCall) []string {
	if !tc.Expanded {
		return nil
	}

	var lines []string

	switch tc.Name {
	case "Bash":
		lines = bashExpandedLines(tc)
	case "Edit":
		lines = renderEditDiff(tc.RawInput)
	case "Write":
		lines = writeExpandedLines(tc)
	case "Read", "Grep", "Glob", "Task":
		lines = resultContentLines(tc)
	default:
		lines = resultContentLines(tc)
	}

	if len(lines) == 0 {
		return nil
	}

	return lines
}

// bashExpandedLines returns "$ command" followed by the command output.
func bashExpandedLines(tc *model.ToolCall) []string {
	var lines []string

	// Add "$ command" header
	if tc.RawInput != nil {
		if cmd, ok := tc.RawInput["command"].(string); ok && cmd != "" {
			lines = append(lines, "$ "+cmd)
		}
	}

	// Add output from result
	if tc.ResultContent != "" {
		resultLines := strings.Split(tc.ResultContent, "\n")
		lines = append(lines, resultLines...)
	}

	return lines
}

// writeExpandedLines returns the content that was written (from tool input).
func writeExpandedLines(tc *model.ToolCall) []string {
	if tc.RawInput != nil {
		if content, ok := tc.RawInput["content"].(string); ok && content != "" {
			return strings.Split(content, "\n")
		}
	}
	return nil
}

// resultContentLines returns lines from the tool result content.
func resultContentLines(tc *model.ToolCall) []string {
	if tc.ResultContent == "" {
		return nil
	}
	return strings.Split(tc.ResultContent, "\n")
}

// renderEditDiff splits old_string into "-" prefixed lines and new_string into
// "+" prefixed lines, producing a simple unified diff view.
func renderEditDiff(rawInput map[string]interface{}) []string {
	if rawInput == nil {
		return nil
	}

	oldStr, _ := rawInput["old_string"].(string)
	newStr, _ := rawInput["new_string"].(string)

	if oldStr == "" && newStr == "" {
		return nil
	}

	var lines []string

	if oldStr != "" {
		for _, l := range strings.Split(oldStr, "\n") {
			lines = append(lines, "-"+l)
		}
	}

	if newStr != "" {
		for _, l := range strings.Split(newStr, "\n") {
			lines = append(lines, "+"+l)
		}
	}

	return lines
}

// renderEditDiffStyled produces pre-styled diff lines for an Edit tool call.
// It chooses between unified (width < 120) and side-by-side (width >= 120)
// layout based on the available width. Returns nil if no diff content exists.
func renderEditDiffStyled(rawInput map[string]interface{}, width int, th theme.Theme) []string {
	if rawInput == nil {
		return nil
	}

	oldStr, _ := rawInput["old_string"].(string)
	newStr, _ := rawInput["new_string"].(string)

	if oldStr == "" && newStr == "" {
		return nil
	}

	var oldLines, newLines []string
	if oldStr != "" {
		oldLines = strings.Split(oldStr, "\n")
	}
	if newStr != "" {
		newLines = strings.Split(newStr, "\n")
	}

	if width >= sideBySideMinWidth {
		return renderSideBySideDiff(oldLines, newLines, width, th)
	}
	return renderUnifiedDiffStyled(oldLines, newLines, width, th)
}

// renderUnifiedDiffStyled produces styled unified diff lines with line numbers.
// Format: "  {linenum} -{content}" in red, "  {linenum} +{content}" in green.
func renderUnifiedDiffStyled(oldLines, newLines []string, width int, th theme.Theme) []string {
	indent := "    " // 4-space indent for expanded content
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.StatusError))
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.StatusSuccess))

	var result []string
	lineNum := 1

	for _, l := range oldLines {
		gutter := fmt.Sprintf("%4d ", lineNum)
		content := indent + gutter + "-" + l
		if len(content) > width {
			content = content[:width-1] + "…"
		}
		result = append(result, errorStyle.Render(content))
		lineNum++
	}

	lineNum = 1
	for _, l := range newLines {
		gutter := fmt.Sprintf("%4d ", lineNum)
		content := indent + gutter + "+" + l
		if len(content) > width {
			content = content[:width-1] + "…"
		}
		result = append(result, successStyle.Render(content))
		lineNum++
	}

	return result
}

// renderSideBySideDiff produces styled side-by-side diff lines.
// Left column shows old content (red), right column shows new content (green),
// separated by a vertical divider. Each column gets half the available width.
func renderSideBySideDiff(oldLines, newLines []string, width int, th theme.Theme) []string {
	indent := "    " // 4-space indent for expanded content
	indentLen := len(indent)

	// Each side gets half the remaining width after indent, minus 3 for " │ " center divider
	usableWidth := width - indentLen
	if usableWidth < 20 {
		usableWidth = 20
	}
	halfWidth := (usableWidth - 3) / 2  // 3 for " │ "
	gutterW := 5                        // "  42 " — line number gutter per side
	contentW := halfWidth - gutterW - 2 // -2 for "│ " border+space per side
	if contentW < 5 {
		contentW = 5
	}

	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.StatusError))
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.StatusSuccess))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.ForegroundDim))

	// Compute number of rows: max of old and new line counts
	rows := len(oldLines)
	if len(newLines) > rows {
		rows = len(newLines)
	}

	var result []string
	for i := 0; i < rows; i++ {
		// Left side (old)
		var leftPart string
		if i < len(oldLines) {
			gutter := fmt.Sprintf("%4d ", i+1)
			text := truncateToWidth(oldLines[i], contentW)
			padded := padToWidth(text, contentW)
			leftPart = gutter + "│ " + padded
		} else {
			leftPart = strings.Repeat(" ", gutterW) + "│ " + strings.Repeat(" ", contentW)
		}

		// Right side (new)
		var rightPart string
		if i < len(newLines) {
			gutter := fmt.Sprintf("%4d ", i+1)
			text := truncateToWidth(newLines[i], contentW)
			rightPart = gutter + "│ " + text
		} else {
			rightPart = strings.Repeat(" ", gutterW) + "│"
		}

		// Style each part: left in red (or dim if empty), right in green (or dim if empty)
		var styledLeft, styledRight string
		if i < len(oldLines) {
			styledLeft = errorStyle.Render(leftPart)
		} else {
			styledLeft = dimStyle.Render(leftPart)
		}
		divider := dimStyle.Render(" │ ")
		if i < len(newLines) {
			styledRight = successStyle.Render(rightPart)
		} else {
			styledRight = dimStyle.Render(rightPart)
		}

		result = append(result, indent+styledLeft+divider+styledRight)
	}

	return result
}

// truncateToWidth truncates a string to the given width, adding "…" if truncated.
func truncateToWidth(s string, w int) string {
	if len(s) <= w {
		return s
	}
	if w <= 1 {
		return "…"
	}
	return s[:w-1] + "…"
}

// padToWidth pads a string with spaces to the given width.
func padToWidth(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

// toolCallLineCount returns the number of display lines a tool call occupies.
// Returns 1 if collapsed, or 1 + number of content lines if expanded.
// This returns the full (uncapped) count; see toolCallLineCountCapped for
// sub-scroll viewport-aware counting.
func toolCallLineCount(tc *model.ToolCall) int {
	if !tc.Expanded {
		return 1
	}
	content := expandedContentLines(tc)
	return 1 + len(content)
}

// toolCallLineCountCapped returns the display line count for a tool call,
// capping the expanded content height when it exceeds the sub-scroll
// threshold (40% of paneHeight). Used by scroll management functions when
// sub-scroll is active for this tool call.
func toolCallLineCountCapped(tc *model.ToolCall, paneHeight int) int {
	if !tc.Expanded {
		return 1
	}
	content := expandedContentLines(tc)
	contentLen := len(content)
	vpHeight := subScrollViewportHeight(contentLen, paneHeight)
	return 1 + vpHeight
}

// renderExpandedContentLine renders a single expanded content line with proper
// indentation and coloring. Edit diff lines use red/green; all others use dim.
func renderExpandedContentLine(line, toolName string, width int, th theme.Theme) string {
	indent := "    " // 4-space indent

	// Determine color based on tool type and line content
	var color string
	switch {
	case toolName == "Edit" && len(line) > 0 && line[0] == '-':
		color = th.StatusError
	case toolName == "Edit" && len(line) > 0 && line[0] == '+':
		color = th.StatusSuccess
	default:
		color = th.ForegroundDim
	}

	styled := indent + line
	// Truncate to width if needed
	if len(styled) > width {
		styled = styled[:width-1] + "…"
	}

	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(styled)
}
