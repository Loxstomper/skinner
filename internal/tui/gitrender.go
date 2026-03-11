package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/loxstomper/skinner/internal/git"
	"github.com/loxstomper/skinner/internal/theme"
)

// GitBottomBarHeight is the total height of the git bottom bar: 1 divider + 2 content rows.
const GitBottomBarHeight = 3

// gitContentHeight returns the height available for the main content area in git view,
// accounting for the git-specific bottom bar height instead of the regular bottom bar.
func (m *Model) gitContentHeight() int {
	h := m.height - 1 // subtract header
	if h < 1 {
		return 20
	}
	if m.effectiveLayout() == "bottom" && m.bottomBarVisible {
		h -= GitBottomBarHeight
		if h < 1 {
			h = 1
		}
	}
	return h
}

// renderGitView renders the full git view (left pane + right pane).
func (m *Model) renderGitView() string {
	paneHeight := m.height - 1
	leftWidth := m.leftPaneWidth()
	rightWidth := m.rightPaneWidth()
	rightHeight := m.gitContentHeight()
	isBottom := m.effectiveLayout() == "bottom" && m.bottomBarVisible

	// Right pane content
	var right string
	switch m.gitViewDepth {
	case 0:
		isSessionCommit := false
		if m.gitSelectedCommit < len(m.gitCommits) {
			c := m.gitCommits[m.gitSelectedCommit]
			isSessionCommit = !m.gitSessionStart.IsZero() && c.AuthorDate.After(m.gitSessionStart)
		}
		right = renderGitCommitSummary(m.gitCommitSummary, rightWidth, rightHeight, m.theme, isSessionCommit)
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

	// Build the result
	var result string

	// Left pane content (side layout only)
	if leftWidth > 0 {
		var left string
		switch m.gitViewDepth {
		case 0:
			left = RenderGitCommitList(GitCommitListProps{
				Commits:          m.gitCommits,
				Selected:         m.gitSelectedCommit,
				Scroll:           m.gitCommitScroll,
				Width:            leftWidth,
				Height:           paneHeight,
				SessionStart:     m.gitSessionStart,
				Theme:            m.theme,
				TotalAdditions:   m.gitTotalAdditions,
				TotalDeletions:   m.gitTotalDeletions,
				TotalStatsLoaded: m.gitTotalStatsLoaded,
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

		result = lipgloss.JoinHorizontal(lipgloss.Top, left, separator, right)
	} else {
		result = right
	}

	// Bottom bar (bottom layout only)
	if isBottom {
		bottomBar := m.renderGitBottomBar(rightWidth)
		result = lipgloss.JoinVertical(lipgloss.Left, result, bottomBar)
	}

	return result
}

// renderGitBottomBar renders the git view bottom bar with either Commits or Files section.
func (m *Model) renderGitBottomBar(width int) string {
	th := m.theme

	var label string
	var content string

	switch m.gitViewDepth {
	case 0:
		label = "Commits"
		content = RenderGitCommitList(GitCommitListProps{
			Commits:          m.gitCommits,
			Selected:         m.gitSelectedCommit,
			Scroll:           m.gitCommitScroll,
			Width:            width,
			Height:           bottomBarSectionHeight,
			SessionStart:     m.gitSessionStart,
			Theme:            th,
			TotalAdditions:   m.gitTotalAdditions,
			TotalDeletions:   m.gitTotalDeletions,
			TotalStatsLoaded: m.gitTotalStatsLoaded,
		})
	case 1, 2:
		label = "Files"
		content = RenderGitFileList(GitFileListProps{
			Files:    m.gitFiles,
			Selected: m.gitSelectedFile,
			Scroll:   m.gitFileScroll,
			Width:    width,
			Height:   bottomBarSectionHeight,
			Theme:    th,
		})
	}

	divider := renderLabeledDivider(label, width, th)
	return lipgloss.JoinVertical(lipgloss.Left, divider, content)
}

// GitCommitListProps are the props for rendering the commit list.
type GitCommitListProps struct {
	Commits          []git.Commit
	Selected         int
	Scroll           int
	Width            int
	Height           int
	SessionStart     time.Time
	Theme            theme.Theme
	TotalAdditions   int
	TotalDeletions   int
	TotalStatsLoaded bool
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

	// Header line with total stats
	headerLine := renderCommitListHeader(props.Width, props.TotalStatsLoaded, props.TotalAdditions, props.TotalDeletions, dimStyle, addedStyle, removedStyle)
	lines = append(lines, headerLine)

	// Reduce available height by 1 for the header line
	commitHeight := props.Height - 1
	if commitHeight < 0 {
		commitHeight = 0
	}

	visible := props.Commits
	scroll := props.Scroll

	// Auto-scroll to keep selected item visible
	if commitHeight > 0 && props.Selected >= scroll+commitHeight {
		scroll = props.Selected - commitHeight + 1
	}
	if props.Selected < scroll {
		scroll = props.Selected
	}

	// Clamp scroll
	if scroll > len(visible)-commitHeight {
		scroll = len(visible) - commitHeight
	}
	if scroll < 0 {
		scroll = 0
	}

	end := scroll + commitHeight
	if end > len(visible) {
		end = len(visible)
	}

	for i := scroll; i < end; i++ {
		c := visible[i]
		isSelected := i == props.Selected
		isSessionCommit := !props.SessionStart.IsZero() && c.AuthorDate.After(props.SessionStart)

		hash := c.Hash
		if len(hash) > 3 {
			hash = hash[:3]
		}

		if isSelected {
			// Selected row: hash subject +N -N (stats replace relTime)
			addStr := addedStyle.Render("+" + git.FormatStatNumber(c.Additions))
			delStr := removedStyle.Render("-" + git.FormatStatNumber(c.Deletions))
			statsText := addStr + " " + delStr
			statsPlain := "+" + git.FormatStatNumber(c.Additions) + " -" + git.FormatStatNumber(c.Deletions)

			subject := c.Subject
			// 3 hash + 1 space + 1 space + statsPlainLen
			maxSubject := props.Width - 3 - 1 - 1 - len(statsPlain)
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

			row := highlightStyle.Render(fmt.Sprintf("%s %s %s",
				dimStyle.Render(hash), subjectRendered, statsText))
			lines = append(lines, truncateWidth(row, props.Width))
		} else {
			// Unselected row: hash subject relTime (no stats)
			relTime := relativeTime(c.AuthorDate)

			subject := c.Subject
			// 3 hash + 1 space + 1 space + relTimeLen
			maxSubject := props.Width - 3 - 1 - 1 - len(relTime)
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

			row := fmt.Sprintf("%s %s %s", dimStyle.Render(hash), subjectRendered, dimStyle.Render(relTime))
			lines = append(lines, truncateWidth(row, props.Width))
		}
	}

	result := strings.Join(lines, "\n")
	return padToHeight(result, props.Width, props.Height)
}

// renderCommitListHeader renders the centered stats divider header line.
func renderCommitListHeader(width int, loaded bool, additions, deletions int, dimStyle, addedStyle, removedStyle lipgloss.Style) string {
	var statsStr string
	if !loaded {
		statsStr = " ... "
	} else {
		addStr := addedStyle.Render("+" + git.FormatStatNumber(additions))
		delStr := removedStyle.Render("-" + git.FormatStatNumber(deletions))
		statsStr = " " + addStr + " " + delStr + " "
	}

	// Calculate plain text width for centering
	var statsPlainLen int
	if !loaded {
		statsPlainLen = 5 // " ... "
	} else {
		statsPlainLen = 1 + 1 + len(git.FormatStatNumber(additions)) + 1 + 1 + len(git.FormatStatNumber(deletions)) + 1
	}

	remaining := width - statsPlainLen
	if remaining < 0 {
		remaining = 0
	}
	leftDashes := remaining / 2
	rightDashes := remaining - leftDashes

	return dimStyle.Render(strings.Repeat("─", leftDashes)) + statsStr + dimStyle.Render(strings.Repeat("─", rightDashes))
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

	// Auto-scroll to keep selected item visible
	if props.Height > 0 && props.Selected >= scroll+props.Height {
		scroll = props.Selected - props.Height + 1
	}
	if props.Selected < scroll {
		scroll = props.Selected
	}

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
// It extracts the subject line from the git show output and renders it as a bold header
// above a horizontal rule, followed by the rest of the commit details.
func renderGitCommitSummary(summary string, width, height int, th theme.Theme, isSessionCommit bool) string {
	if summary == "" {
		return padToHeight("No commit selected", width, height)
	}

	lines := strings.Split(summary, "\n")
	addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.DiffAdded))
	removedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.DiffRemoved))

	// Extract subject line: first non-blank line after the "Date:" line
	subject := ""
	subjectIdx := -1
	pastDate := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Date:") {
			pastDate = true
			continue
		}
		if pastDate && trimmed != "" {
			subject = trimmed
			subjectIdx = i
			break
		}
	}

	var result []string

	// Render subject header if found
	if subject != "" {
		subjectColor := lipgloss.Color(th.Foreground)
		if isSessionCommit {
			subjectColor = lipgloss.Color(th.DiffSessionCommit)
		}
		subjectStyle := lipgloss.NewStyle().Bold(true).Foreground(subjectColor)
		result = append(result, subjectStyle.Render(subject))

		// Horizontal rule
		ruleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(th.ForegroundDim))
		result = append(result, ruleStyle.Render(strings.Repeat("─", width)))
	}

	// Append remaining lines, skipping the subject line to avoid repetition
	for i, line := range lines {
		if i == subjectIdx {
			continue
		}
		// Color the +/- in stat summary lines
		if strings.Contains(line, "insertion") || strings.Contains(line, "deletion") || strings.Contains(line, "file") {
			result = append(result, colorizeStatLine(line, addedStyle, removedStyle))
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
