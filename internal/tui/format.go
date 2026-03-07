package tui

import (
	"fmt"
	"time"
)

// FormatDuration returns "..." for running items or the formatted duration value.
func FormatDuration(d time.Duration, running bool) string {
	if running {
		return "..."
	}
	return FormatDurationValue(d)
}

// FormatDurationValue formats a duration as "N.Ns" (<60s) or "NmNNs" (>=60s).
func FormatDurationValue(d time.Duration) string {
	secs := d.Seconds()
	if secs < 60 {
		return fmt.Sprintf("%.1fs", secs)
	}
	mins := int(secs) / 60
	remainSecs := int(secs) % 60
	return fmt.Sprintf("%dm%02ds", mins, remainSecs)
}

// FormatTokens formats token counts: raw number if <1000, "N.Nk" otherwise.
func FormatTokens(tokens int64) string {
	if tokens < 1000 {
		return fmt.Sprintf("%d", tokens)
	}
	return fmt.Sprintf("%.1fk", float64(tokens)/1000.0)
}

// ToolIcon returns a Nerd Font icon for the given tool name.
func ToolIcon(name string) string {
	switch name {
	case "Read":
		return "\uf02d"
	case "Edit":
		return "\uf044"
	case "Write":
		return "\uf0c7"
	case "Bash":
		return "\uf120"
	case "Grep":
		return "\uf002"
	case "Glob":
		return "\uf07b"
	case "Task":
		return "\uf085"
	default:
		return "\uf059"
	}
}

// GroupSummaryUnit returns the plural noun for a group of tool calls (e.g. "files", "edits").
func GroupSummaryUnit(toolName string) string {
	switch toolName {
	case "Read", "Write":
		return "files"
	case "Edit":
		return "edits"
	case "Bash":
		return "commands"
	case "Grep":
		return "searches"
	case "Glob":
		return "globs"
	case "Task":
		return "tasks"
	default:
		return "calls"
	}
}

// IsKnownTool returns true if the tool name is one of the built-in Claude tools
// that gets a dedicated icon and compact-mode display.
func IsKnownTool(name string) bool {
	switch name {
	case "Read", "Edit", "Write", "Bash", "Grep", "Glob", "Task":
		return true
	default:
		return false
	}
}
