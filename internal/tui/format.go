package tui

import (
	"fmt"
	"os"
	"strings"
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

// FormatTokens formats token counts with appropriate unit suffix:
// >= 1B → "N.NG", >= 1M → "N.NM", >= 1k → "N.Nk", < 1k → raw number.
func FormatTokens(tokens int64) string {
	switch {
	case tokens >= 1_000_000_000:
		return fmt.Sprintf("%.1fG", float64(tokens)/1_000_000_000.0)
	case tokens >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(tokens)/1_000_000.0)
	case tokens >= 1000:
		return fmt.Sprintf("%.1fk", float64(tokens)/1000.0)
	default:
		return fmt.Sprintf("%d", tokens)
	}
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

// TrimPath strips the CWD prefix or replaces $HOME with ~/ to shorten displayed paths.
// Rules applied in order: (1) strip cwd+"/" prefix, (2) replace $HOME+"/" with "~/".
func TrimPath(path, cwd string) string {
	// Rule 1: strip CWD prefix
	if cwd != "" {
		prefix := cwd
		if !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		if strings.HasPrefix(path, prefix) {
			return path[len(prefix):]
		}
	}

	// Rule 2: replace $HOME with ~/
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		prefix := home
		if !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}
		if strings.HasPrefix(path, prefix) {
			return "~/" + path[len(prefix):]
		}
	}

	return path
}

// TrimSummaryPath applies path trimming to a tool call summary based on tool name.
func TrimSummaryPath(summary, toolName, cwd string) string {
	switch toolName {
	case "Read", "Edit", "Write":
		return TrimPath(summary, cwd)
	case "Grep":
		// Summary format: "pattern" in path — trim the path part
		if idx := strings.LastIndex(summary, " in "); idx >= 0 {
			pathPart := summary[idx+4:]
			return summary[:idx+4] + TrimPath(pathPart, cwd)
		}
		return summary
	case "Glob":
		// Pattern may contain an absolute path prefix
		return TrimPath(summary, cwd)
	default:
		return summary
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
