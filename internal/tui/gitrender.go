package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/loxstomper/skinner/internal/git"
	"github.com/loxstomper/skinner/internal/theme"
)

// renderGitView renders the full git view (left pane + right pane).
func (m *Model) renderGitView() string {
	paneHeight := m.height - 1
	leftWidth := m.leftPaneWidth()
	rightWidth := m.rightPaneWidth()
	rightHeight := m.rightPaneHeight()

	// Right pane content
	var right string
	switch m.gitViewDepth {
	case 0:
		right = renderGitCommitSummary(m.gitCommitSummary, rightWidth, rightHeight, m.theme)
	case 1, 2:
		switch {
		case len(m.gitParsedDiff) > 0:
			filePath := ""
			if m.gitSelectedFile < len(m.gitFiles) {
				filePath = m.gitFiles[m.gitSelectedFile].Path
			}
			right = RenderDiff(DiffViewProps{
				Hunks:     m.gitParsedDiff,
				FilePath:  filePath,
				Theme:     m.theme,
				ThemeName: m.config.ThemeName,
				Width:     rightWidth,
				HScroll:   m.gitDiffHScroll,
			})
			// Apply vertical scroll
			lines := strings.Split(right, "\n")
			if m.gitDiffScroll >= len(lines) {
				m.gitDiffScroll = len(lines) - 1
				if m.gitDiffScroll < 0 {
					m.gitDiffScroll = 0
				}
			}
			if m.gitDiffScroll > 0 && m.gitDiffScroll < len(lines) {
				lines = lines[m.gitDiffScroll:]
			}
			if len(lines) > rightHeight {
				lines = lines[:rightHeight]
			}
			right = strings.Join(lines, "\n")
		case m.gitDiffContent != "":
			right = m.gitDiffContent
		default:
			right = "No diff available"
		}
	}
	// Pad right pane to full height
	right = padToHeight(right, rightWidth, rightHeight)

	// Left pane content
	if leftWidth > 0 {
		var left string
		switch m.gitViewDepth {
		case 0:
			left = RenderGitCommitList(GitCommitListProps{
				Commits:      m.gitCommits,
				Selected:     m.gitSelectedCommit,
				Scroll:       m.gitCommitScroll,
				Width:        leftWidth,
				Height:       paneHeight,
				SessionStart: m.gitSessionStart,
				Theme:        m.theme,
			})
		case 1, 2:
			left = RenderGitFileList(GitFileListProps{
				Files:    m.gitFiles,
				Selected: m.gitSelectedFile,
				Scroll:   m.gitFileScroll,
				Width:    leftWidth,
				Height:   paneHeight,
				Theme:    m.theme,
			})
		}

		sepLines := make([]string, paneHeight)
		for i := range sepLines {
			sepLines[i] = "│"
		}
		separator := lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.theme.ForegroundDim)).
			Render(strings.Join(sepLines, "\n"))

		return lipgloss.JoinHorizontal(lipgloss.Top, left, separator, right)
	}

	return right
}

// GitCommitListProps are the props for rendering the commit list.
type GitCommitListProps struct {
	Commits      []git.Commit
	Selected     int
	Scroll       int
	Width        int
	Height       int
	SessionStart time.Time
	Theme        theme.Theme
}

// RenderGitCommitList renders the commit list for the left pane.
func RenderGitCommitList(props GitCommitListProps) string {
	if len(props.Commits) == 0 {
		return padToHeight("No commits", props.Width, props.Height)
	}

	th := props.Theme
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.ForegroundDim))
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Foreground))
	sessionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.DiffSessionCommit))
	highlightStyle := lipgloss.NewStyle().Background(lipgloss.Color(th.Highlight))
	addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.DiffAdded))
	removedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.DiffRemoved))

	var lines []string
	visible := props.Commits
	scroll := props.Scroll

	// Clamp scroll
	if scroll > len(visible)-props.Height {
		scroll = len(visible) - props.Height
	}
	if scroll < 0 {
		scroll = 0
	}

	end := scroll + props.Height
	if end > len(visible) {
		end = len(visible)
	}

	for i := scroll; i < end; i++ {
		c := visible[i]
		isSelected := i == props.Selected
		isSessionCommit := !props.SessionStart.IsZero() && c.AuthorDate.After(props.SessionStart)

		hash := dimStyle.Render(truncate(c.Hash, 7))
		relTime := dimStyle.Render(relativeTime(c.AuthorDate))

		subject := c.Subject
		maxSubject := props.Width - 7 - 1 - len(relTime) - 6 // hash + space + time + stats
		if maxSubject < 5 {
			maxSubject = 5
		}
		subject = truncate(subject, maxSubject)

		var subjectRendered string
		if isSessionCommit {
			subjectRendered = sessionStyle.Render(subject)
		} else {
			subjectRendered = normalStyle.Render(subject)
		}

		stats := ""
		if c.Additions > 0 {
			stats += addedStyle.Render(fmt.Sprintf("+%d", c.Additions))
		}
		if c.Deletions > 0 {
			if stats != "" {
				stats += " "
			}
			stats += removedStyle.Render(fmt.Sprintf("-%d", c.Deletions))
		}

		var row string
		if isSelected {
			row = highlightStyle.Render(fmt.Sprintf("%s %s %s %s",
				dimStyle.Render(truncate(c.Hash, 7)),
				func() string {
					if isSessionCommit {
						return sessionStyle.Render(subject)
					}
					return normalStyle.Render(subject)
				}(),
				relTime, stats))
		} else {
			row = fmt.Sprintf("%s %s %s %s", hash, subjectRendered, relTime, stats)
		}

		lines = append(lines, truncateWidth(row, props.Width))
	}

	result := strings.Join(lines, "\n")
	return padToHeight(result, props.Width, props.Height)
}

// GitFileListProps are the props for rendering the file list.
type GitFileListProps struct {
	Files    []git.FileChange
	Selected int
	Scroll   int
	Width    int
	Height   int
	Theme    theme.Theme
}

// RenderGitFileList renders the file list for the left pane.
func RenderGitFileList(props GitFileListProps) string {
	if len(props.Files) == 0 {
		return padToHeight("No files changed", props.Width, props.Height)
	}

	th := props.Theme
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.ForegroundDim))
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.Foreground))
	highlightStyle := lipgloss.NewStyle().Background(lipgloss.Color(th.Highlight))
	addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.DiffAdded))
	removedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.DiffRemoved))

	var lines []string
	scroll := props.Scroll

	if scroll > len(props.Files)-props.Height {
		scroll = len(props.Files) - props.Height
	}
	if scroll < 0 {
		scroll = 0
	}

	end := scroll + props.Height
	if end > len(props.Files) {
		end = len(props.Files)
	}

	for i := scroll; i < end; i++ {
		f := props.Files[i]
		isSelected := i == props.Selected

		status := dimStyle.Render(fmt.Sprintf("%-2s", f.Status))
		name := normalStyle.Render(truncate(f.Path, props.Width-10))

		stats := ""
		if f.Additions > 0 {
			stats += addedStyle.Render(fmt.Sprintf("+%d", f.Additions))
		}
		if f.Deletions > 0 {
			if stats != "" {
				stats += " "
			}
			stats += removedStyle.Render(fmt.Sprintf("-%d", f.Deletions))
		}

		var row string
		if isSelected {
			row = highlightStyle.Render(fmt.Sprintf("%s %s %s", status, name, stats))
		} else {
			row = fmt.Sprintf("%s %s %s", status, name, stats)
		}

		lines = append(lines, truncateWidth(row, props.Width))
	}

	result := strings.Join(lines, "\n")
	return padToHeight(result, props.Width, props.Height)
}

// renderGitCommitSummary renders the commit summary for the right pane at depth 0.
func renderGitCommitSummary(summary string, width, height int, th theme.Theme) string {
	if summary == "" {
		return padToHeight("No commit selected", width, height)
	}

	// Apply basic coloring to stat lines (+/-)
	lines := strings.Split(summary, "\n")
	addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.DiffAdded))
	removedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.DiffRemoved))

	var result []string
	for _, line := range lines {
		// Color the +/- in stat summary lines
		if strings.Contains(line, "insertion") || strings.Contains(line, "deletion") || strings.Contains(line, "file") {
			// This is a stat summary line — color the + and - counts
			colored := colorizeStatLine(line, addedStyle, removedStyle)
			result = append(result, colored)
		} else {
			result = append(result, line)
		}
	}

	if len(result) > height {
		result = result[:height]
	}

	return padToHeight(strings.Join(result, "\n"), width, height)
}

// colorizeStatLine applies DiffAdded/DiffRemoved colors to +/- in stat lines.
func colorizeStatLine(line string, addedStyle, removedStyle lipgloss.Style) string {
	var result strings.Builder
	for _, ch := range line {
		switch ch {
		case '+':
			result.WriteString(addedStyle.Render("+"))
		case '-':
			result.WriteString(removedStyle.Render("-"))
		default:
			result.WriteRune(ch)
		}
	}
	return result.String()
}

// padToHeight pads a rendered string to fill the given height with blank lines.
func padToHeight(s string, width, height int) string {
	lines := strings.Split(s, "\n")
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

// truncate truncates a string to maxLen, adding "…" if truncated.
func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	return string(runes[:maxLen-1]) + "…"
}

// truncateWidth truncates rendered output to a visual width.
func truncateWidth(s string, width int) string {
	w := lipgloss.Width(s)
	if w <= width {
		return s
	}
	// Fall back to rune truncation as approximate
	runes := []rune(stripAnsi(s))
	if len(runes) > width {
		return string(runes[:width])
	}
	return s
}

// stripAnsi removes ANSI escape codes from a string.
func stripAnsi(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '~' {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

// relativeTime formats a time as a human-readable relative string.
func relativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", h)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	default:
		months := int(d.Hours() / 24 / 30)
		if months <= 1 {
			return "1mo ago"
		}
		return fmt.Sprintf("%dmo ago", months)
	}
}
