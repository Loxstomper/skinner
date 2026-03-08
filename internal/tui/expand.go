package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/loxstomper/skinner/internal/model"
	"github.com/loxstomper/skinner/internal/theme"
)

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

// toolCallLineCount returns the number of display lines a tool call occupies.
// Returns 1 if collapsed, or 1 + number of content lines if expanded.
func toolCallLineCount(tc *model.ToolCall) int {
	if !tc.Expanded {
		return 1
	}
	content := expandedContentLines(tc)
	return 1 + len(content)
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
