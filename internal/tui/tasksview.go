package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/loxstomper/skinner/internal/bd"
	"github.com/loxstomper/skinner/internal/config"
)

// tasksViewRow represents a single display row in the issue list,
// including tree-rendering metadata.
type tasksViewRow struct {
	issue  *bd.Issue
	depth  int    // tree depth level (0 = root)
	prefix string // tree connector prefix (e.g., "├ ", "│ └ ")
}

// tasksViewDataMsg carries the result of async bd data fetching.
type tasksViewDataMsg struct {
	graph *bd.Graph
	err   error
}

// enterTasksView activates the tasks view and starts async data loading.
func (m *Model) enterTasksView() tea.Cmd {
	m.tasksViewActive = true
	m.tasksViewDepth = 0
	m.tasksViewLoading = true
	m.tasksViewError = nil
	m.tasksViewGraph = nil
	m.tasksViewCursor = 0
	m.tasksViewScroll = 0
	m.tasksViewListScroll = 0
	m.tasksViewTab = 0 // Ready tab
	m.tasksViewFiltered = nil
	m.tasksViewVisibleRows = nil
	m.tasksViewExpanded = make(map[string]bool)
	m.tasksViewFlatMode = false
	m.tasksViewSearchActive = false
	m.tasksViewSearchQuery = ""

	return m.fetchTasksData()
}

// exitTasksView deactivates the tasks view, restoring the previous view.
func (m *Model) exitTasksView() {
	m.tasksViewActive = false
	m.tasksViewDepth = 0
	m.tasksViewGraph = nil
	m.tasksViewFiltered = nil
	m.tasksViewVisibleRows = nil
	m.tasksViewExpanded = nil
	m.tasksViewLoading = false
	m.tasksViewError = nil
	m.tasksViewSearchActive = false
	m.tasksViewSearchQuery = ""
}

// fetchTasksData returns a tea.Cmd that fetches bd data asynchronously.
func (m *Model) fetchTasksData() tea.Cmd {
	return func() tea.Msg {
		client := bd.NewClient()
		ctx := context.Background()

		issues, err := client.List(ctx, bd.ListOpts{})
		if err != nil {
			return tasksViewDataMsg{err: err}
		}

		graph := bd.NewGraph(issues)
		return tasksViewDataMsg{graph: graph}
	}
}

// handleTasksViewKey routes key actions when the tasks view is active.
func (m *Model) handleTasksViewKey(action, key string) (tea.Model, tea.Cmd) {
	// Handle search mode input first.
	if m.tasksViewSearchActive {
		return m.handleTasksViewSearchKey(action)
	}

	switch action {
	case config.ActionQuit:
		m.exitTasksView()
		return m, nil

	case config.ActionEscape:
		if m.tasksViewDepth > 0 {
			m.tasksViewDepth = 0
			m.tasksViewScroll = 0
			return m, nil
		}
		m.exitTasksView()
		return m, nil

	case config.ActionExpand:
		if m.tasksViewDepth == 0 {
			m.tasksViewDepth = 1
			m.tasksViewScroll = 0
		}
		return m, nil

	case config.ActionMoveDown:
		if m.tasksViewDepth == 1 {
			m.tasksViewScroll++
		} else {
			m.tasksViewMoveCursor(1)
		}
		return m, nil

	case config.ActionMoveUp:
		if m.tasksViewDepth == 1 {
			if m.tasksViewScroll > 0 {
				m.tasksViewScroll--
			}
		} else {
			m.tasksViewMoveCursor(-1)
		}
		return m, nil

	case config.ActionJumpTop:
		if m.tasksViewDepth == 1 {
			m.tasksViewScroll = 0
		} else {
			m.tasksViewCursor = 0
			m.tasksViewListScroll = 0
		}
		return m, nil

	case config.ActionJumpBottom:
		if m.tasksViewDepth == 1 {
			// Will be clamped during render.
			m.tasksViewScroll = 9999
		} else if len(m.tasksViewVisibleRows) > 0 {
			m.tasksViewCursor = len(m.tasksViewVisibleRows) - 1
		}
		return m, nil

	case "search":
		m.tasksViewSearchActive = true
		m.tasksViewSearchQuery = ""
		m.tasksViewRebuildVisible()
		return m, nil
	}

	// Handle unbound keys by raw key value for tasks-specific actions.
	if m.tasksViewDepth == 0 {
		switch key {
		case "H": // Previous tab.
			if m.tasksViewTab > 0 {
				m.tasksViewTab--
				m.tasksViewCursor = 0
				m.tasksViewListScroll = 0
				m.tasksViewSearchActive = false
				m.tasksViewSearchQuery = ""
				m.tasksViewRefilter()
			}
			return m, nil

		case "L": // Next tab.
			if m.tasksViewTab < 3 {
				m.tasksViewTab++
				m.tasksViewCursor = 0
				m.tasksViewListScroll = 0
				m.tasksViewSearchActive = false
				m.tasksViewSearchQuery = ""
				m.tasksViewRefilter()
			}
			return m, nil

		case " ": // Toggle expand/collapse.
			if !m.tasksViewFlatMode && len(m.tasksViewVisibleRows) > 0 && m.tasksViewCursor < len(m.tasksViewVisibleRows) {
				issue := m.tasksViewVisibleRows[m.tasksViewCursor].issue
				m.tasksViewExpanded[issue.ID] = !m.tasksViewIsExpanded(issue.ID)
				m.tasksViewRebuildVisible()
			}
			return m, nil

		case "f": // Toggle flat/tree mode.
			m.tasksViewFlatMode = !m.tasksViewFlatMode
			m.tasksViewRebuildVisible()
			return m, nil

		case "r": // Refresh.
			m.tasksViewLoading = true
			m.tasksViewError = nil
			return m, m.fetchTasksData()
		}
	}

	return m, nil
}

// handleTasksViewSearchKey handles action-based keys during fuzzy search mode.
func (m *Model) handleTasksViewSearchKey(action string) (tea.Model, tea.Cmd) {
	switch action {
	case config.ActionEscape:
		m.tasksViewSearchActive = false
		m.tasksViewSearchQuery = ""
		m.tasksViewRefilter()
		return m, nil

	case config.ActionExpand: // enter
		m.tasksViewSearchActive = false
		// Keep filtered results, rebuild visible for tree mode.
		m.tasksViewRebuildVisible()
		return m, nil

	case config.ActionMoveDown:
		m.tasksViewMoveCursor(1)
		return m, nil

	case config.ActionMoveUp:
		m.tasksViewMoveCursor(-1)
		return m, nil
	}

	return m, nil
}

// handleTasksViewSearchRawKey handles raw character input during search.
// Returns true if the key was consumed.
func (m *Model) handleTasksViewSearchRawKey(key string) bool {
	switch key {
	case "backspace":
		if len(m.tasksViewSearchQuery) > 0 {
			m.tasksViewSearchQuery = m.tasksViewSearchQuery[:len(m.tasksViewSearchQuery)-1]
			m.tasksViewRefilter()
			m.tasksViewCursor = 0
		} else {
			m.tasksViewSearchActive = false
			m.tasksViewRefilter()
		}
		return true
	default:
		if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
			m.tasksViewSearchQuery += key
			m.tasksViewRefilter()
			m.tasksViewCursor = 0
			return true
		}
	}
	return false
}

// tasksViewMoveCursor moves the cursor by delta, clamping to visible row bounds.
func (m *Model) tasksViewMoveCursor(delta int) {
	if len(m.tasksViewVisibleRows) == 0 {
		return
	}
	m.tasksViewCursor += delta
	if m.tasksViewCursor < 0 {
		m.tasksViewCursor = 0
	}
	if m.tasksViewCursor >= len(m.tasksViewVisibleRows) {
		m.tasksViewCursor = len(m.tasksViewVisibleRows) - 1
	}
}

// tasksViewIsExpanded returns whether an issue node is expanded.
// Defaults to true (expanded) if not explicitly set.
func (m *Model) tasksViewIsExpanded(id string) bool {
	expanded, ok := m.tasksViewExpanded[id]
	if !ok {
		return true // default: all expanded
	}
	return expanded
}

// tasksViewSelectedIssue returns the currently selected issue, or nil.
func (m *Model) tasksViewSelectedIssue() *bd.Issue {
	if m.tasksViewCursor < len(m.tasksViewVisibleRows) {
		return m.tasksViewVisibleRows[m.tasksViewCursor].issue
	}
	return nil
}

// tasksViewRefilter rebuilds the filtered issue list based on current tab and search.
func (m *Model) tasksViewRefilter() {
	if m.tasksViewGraph == nil {
		m.tasksViewFiltered = nil
		m.tasksViewVisibleRows = nil
		return
	}

	var issues []*bd.Issue
	switch m.tasksViewTab {
	case 0: // Ready - open issues with no unresolved blocking dependencies
		blocked := m.tasksViewGraph.FilterBlocked()
		blockedSet := make(map[string]bool, len(blocked))
		for _, b := range blocked {
			blockedSet[b.ID] = true
		}
		for _, issue := range m.tasksViewGraph.Issues {
			if issue.Status == "open" && !blockedSet[issue.ID] {
				issues = append(issues, issue)
			}
		}
	case 1: // All (regardless of status)
		issues = make([]*bd.Issue, len(m.tasksViewGraph.Issues))
		copy(issues, m.tasksViewGraph.Issues)
	case 2: // Blocked
		issues = m.tasksViewGraph.FilterBlocked()
	case 3: // In Progress
		issues = m.tasksViewGraph.FilterByStatus("in_progress")
	}

	if m.tasksViewSearchQuery != "" {
		// Intersect with search results.
		searchResults := m.tasksViewGraph.FuzzySearch(m.tasksViewSearchQuery)
		searchSet := make(map[string]bool, len(searchResults))
		for _, r := range searchResults {
			searchSet[r.ID] = true
		}
		var filtered []*bd.Issue
		for _, issue := range issues {
			if searchSet[issue.ID] {
				filtered = append(filtered, issue)
			}
		}
		issues = filtered
	}

	m.tasksViewFiltered = issues
	m.tasksViewRebuildVisible()
	if m.tasksViewCursor >= len(m.tasksViewVisibleRows) {
		m.tasksViewCursor = max(0, len(m.tasksViewVisibleRows)-1)
	}
}

// tasksViewRebuildVisible builds the display-ordered visible rows from the
// filtered issue list, respecting tree/flat mode and expand/collapse state.
func (m *Model) tasksViewRebuildVisible() {
	// Remember current selection to preserve it after rebuild.
	var prevID string
	if m.tasksViewCursor < len(m.tasksViewVisibleRows) {
		prevID = m.tasksViewVisibleRows[m.tasksViewCursor].issue.ID
	}

	if m.tasksViewFlatMode || m.tasksViewSearchActive {
		// Flat mode: all filtered issues at depth 0, no tree prefixes.
		rows := make([]tasksViewRow, len(m.tasksViewFiltered))
		for i, issue := range m.tasksViewFiltered {
			rows[i] = tasksViewRow{issue: issue, depth: 0}
		}
		m.tasksViewVisibleRows = rows
	} else {
		// Tree mode: walk from roots, respecting expansion and filter.
		if m.tasksViewGraph == nil {
			m.tasksViewVisibleRows = nil
			return
		}

		filteredSet := make(map[string]bool, len(m.tasksViewFiltered))
		for _, issue := range m.tasksViewFiltered {
			filteredSet[issue.ID] = true
		}

		var rows []tasksViewRow
		added := make(map[string]bool)

		// markDescendants recursively marks all descendants of a collapsed parent
		// so they are not treated as orphans.
		var markDescendants func(parentID string, visited map[string]bool)
		markDescendants = func(parentID string, visited map[string]bool) {
			for _, child := range m.tasksViewGraph.Children[parentID] {
				if !visited[child.ID] {
					visited[child.ID] = true
					markDescendants(child.ID, visited)
				}
			}
		}

		var walk func(issues []*bd.Issue, depth int, parentPrefix string)
		walk = func(issues []*bd.Issue, depth int, parentPrefix string) {
			// Collect only filtered issues at this level.
			var visible []*bd.Issue
			for _, issue := range issues {
				if filteredSet[issue.ID] && !added[issue.ID] {
					visible = append(visible, issue)
				}
			}

			for i, issue := range visible {
				added[issue.ID] = true
				isLast := i == len(visible)-1

				var prefix string
				if depth > 0 {
					if isLast {
						prefix = parentPrefix + "└ "
					} else {
						prefix = parentPrefix + "├ "
					}
				}

				rows = append(rows, tasksViewRow{
					issue:  issue,
					depth:  depth,
					prefix: prefix,
				})

				// Recurse into children if expanded; mark as skipped if collapsed.
				children := m.tasksViewGraph.Children[issue.ID]
				if m.tasksViewIsExpanded(issue.ID) {
					if len(children) > 0 {
						var childPrefix string
						if depth > 0 {
							if isLast {
								childPrefix = parentPrefix + "  "
							} else {
								childPrefix = parentPrefix + "│ "
							}
						}
						walk(children, depth+1, childPrefix)
					}
				} else {
					// Mark all descendants as added so they're not treated as orphans.
					markDescendants(issue.ID, added)
				}
			}
		}

		walk(m.tasksViewGraph.Roots, 0, "")

		// Add orphans: filtered issues not reached via root tree walk
		// (e.g., children whose parents were filtered out).
		for _, issue := range m.tasksViewFiltered {
			if !added[issue.ID] {
				added[issue.ID] = true
				rows = append(rows, tasksViewRow{issue: issue, depth: 0})
			}
		}

		m.tasksViewVisibleRows = rows
	}

	// Try to restore cursor to same issue.
	if prevID != "" {
		for i, row := range m.tasksViewVisibleRows {
			if row.issue.ID == prevID {
				m.tasksViewCursor = i
				return
			}
		}
	}
	// Clamp cursor.
	if m.tasksViewCursor >= len(m.tasksViewVisibleRows) {
		m.tasksViewCursor = max(0, len(m.tasksViewVisibleRows)-1)
	}
}

// renderTasksView renders the full tasks view overlay.
func (m *Model) renderTasksView() string {
	contentHeight := m.height - 1 // minus header

	if m.tasksViewLoading {
		return m.renderTasksViewCentered("Loading issues...", contentHeight)
	}
	if m.tasksViewError != nil {
		return m.renderTasksViewError(contentHeight)
	}
	if m.tasksViewGraph == nil {
		return m.renderTasksViewCentered("No data", contentHeight)
	}

	// Tab header.
	tabHeader := m.renderTasksViewTabHeader(m.width)

	// Content area below tabs.
	paneHeight := contentHeight - 2 // subtract tab header + separator

	// Left pane (32 chars) + right pane.
	leftWidth := 32
	rightWidth := m.width - leftWidth - 1 // 1 for separator

	leftContent := m.renderTasksViewList(leftWidth, paneHeight)
	rightContent := m.tasksViewRenderDetail(rightWidth, paneHeight)

	sep := strings.Repeat("│\n", paneHeight)
	sep = strings.TrimSuffix(sep, "\n")
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.ForegroundDim))

	panes := lipgloss.JoinHorizontal(lipgloss.Top,
		leftContent,
		sepStyle.Render(sep),
		rightContent,
	)

	return tabHeader + "\n" + panes
}

// renderTasksViewCentered renders centered text for loading states.
func (m *Model) renderTasksViewCentered(text string, height int) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.ForegroundDim)).
		Width(m.width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center)
	return style.Render(text)
}

// renderTasksViewError renders the error state with styled message and hint.
func (m *Model) renderTasksViewError(height int) string {
	errStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.StatusError))
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.ForegroundDim))

	errMsg := errStyle.Render(fmt.Sprintf("Could not load issues: %v", m.tasksViewError))
	hint := hintStyle.Render("r:retry  q:back")
	content := errMsg + "\n" + hint

	container := lipgloss.NewStyle().
		Width(m.width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center)
	return container.Render(content)
}

// renderTasksViewTabHeader renders the tab bar with counts and right-aligned hint.
func (m *Model) renderTasksViewTabHeader(width int) string {
	tabs := []struct {
		label string
		count int
	}{
		{"Ready", m.tasksViewTabCount(0)},
		{"All", m.tasksViewTabCount(1)},
		{"Blocked", m.tasksViewTabCount(2)},
		{"In Progress", m.tasksViewTabCount(3)},
	}

	activeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.Foreground)).
		Bold(true)
	inactiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.ForegroundDim))
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.ForegroundDim))

	var parts []string
	tabTextLen := 1 // leading space
	for i, tab := range tabs {
		var label string
		if i == m.tasksViewTab {
			label = fmt.Sprintf("[%s %d]", tab.label, tab.count)
			parts = append(parts, activeStyle.Render(label))
		} else {
			label = fmt.Sprintf("%s %d", tab.label, tab.count)
			parts = append(parts, inactiveStyle.Render(label))
		}
		tabTextLen += len(label)
		if i < len(tabs)-1 {
			tabTextLen += 2 // separator "  "
		}
	}

	tabLine := " " + strings.Join(parts, "  ")

	// Right-align q:back hint.
	hint := "q:back"
	padding := width - tabTextLen - len(hint) - 1 // 1 for trailing space
	if padding > 0 {
		tabLine += strings.Repeat(" ", padding) + hintStyle.Render(hint)
	}

	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.ForegroundDim)).
		Render(strings.Repeat("─", width))

	return tabLine + "\n" + separator
}

// tasksViewTabCount returns the issue count for a given tab,
// applying the current search filter if active.
func (m *Model) tasksViewTabCount(tab int) int {
	if m.tasksViewGraph == nil {
		return 0
	}

	// Build the base set of issues for this tab.
	var issues []*bd.Issue
	switch tab {
	case 0: // Ready - open issues with no unresolved blocking dependencies
		blocked := m.tasksViewGraph.FilterBlocked()
		blockedSet := make(map[string]bool, len(blocked))
		for _, b := range blocked {
			blockedSet[b.ID] = true
		}
		for _, issue := range m.tasksViewGraph.Issues {
			if issue.Status == "open" && !blockedSet[issue.ID] {
				issues = append(issues, issue)
			}
		}
	case 1: // All (regardless of status)
		issues = make([]*bd.Issue, len(m.tasksViewGraph.Issues))
		copy(issues, m.tasksViewGraph.Issues)
	case 2: // Blocked
		issues = m.tasksViewGraph.FilterBlocked()
	case 3: // In Progress
		issues = m.tasksViewGraph.FilterByStatus("in_progress")
	}

	// If search is active, intersect with search results.
	if m.tasksViewSearchQuery != "" {
		searchResults := m.tasksViewGraph.FuzzySearch(m.tasksViewSearchQuery)
		searchSet := make(map[string]bool, len(searchResults))
		for _, r := range searchResults {
			searchSet[r.ID] = true
		}
		count := 0
		for _, issue := range issues {
			if searchSet[issue.ID] {
				count++
			}
		}
		return count
	}

	return len(issues)
}

// renderTasksViewList renders the left pane issue list with status icons,
// type colors, tree connectors, and viewport scrolling.
func (m *Model) renderTasksViewList(width, height int) string {
	var lines []string
	listHeight := height

	// Show search input bar when search is active.
	if m.tasksViewSearchActive {
		inputStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.theme.Foreground))
		raw := "/" + m.tasksViewSearchQuery + "█"
		if len(raw) < width {
			raw += strings.Repeat(" ", width-len(raw))
		}
		lines = append(lines, inputStyle.Render(raw))
		listHeight--
	}

	if len(m.tasksViewVisibleRows) == 0 {
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.theme.ForegroundDim)).
			Width(width).
			Height(listHeight).
			Align(lipgloss.Center, lipgloss.Center)
		emptyContent := style.Render("No issues")
		if m.tasksViewSearchActive {
			return strings.Join(lines, "\n") + "\n" + emptyContent
		}
		return emptyContent
	}

	// Ensure cursor is visible in the list viewport.
	if m.tasksViewListScroll > m.tasksViewCursor {
		m.tasksViewListScroll = m.tasksViewCursor
	}
	if m.tasksViewCursor >= m.tasksViewListScroll+listHeight {
		m.tasksViewListScroll = m.tasksViewCursor - listHeight + 1
	}
	if m.tasksViewListScroll < 0 {
		m.tasksViewListScroll = 0
	}

	end := m.tasksViewListScroll + listHeight
	if end > len(m.tasksViewVisibleRows) {
		end = len(m.tasksViewVisibleRows)
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.ForegroundDim))

	for i := m.tasksViewListScroll; i < end; i++ {
		row := m.tasksViewVisibleRows[i]
		issue := row.issue

		// Build the collapse indicator for parent nodes.
		collapse := ""
		if !m.tasksViewFlatMode && !m.tasksViewSearchActive && m.tasksViewGraph != nil {
			if children := m.tasksViewGraph.Children[issue.ID]; len(children) > 0 {
				if !m.tasksViewIsExpanded(issue.ID) {
					collapse = "▶"
				}
			}
		}

		icon := statusIcon(issue.Status)
		priStr := fmt.Sprintf("%d", issue.Priority)

		// Calculate available width for the title.
		// Plain-text structure: {prefix}{collapse}{icon} {pri} {id}  {title}
		metaLen := len(row.prefix) + len(collapse) + len(icon) + 1 + len(priStr) + 1 + len(issue.ID) + 2
		titleAvail := width - metaLen
		title := issue.Title
		if titleAvail <= 0 {
			title = ""
		} else {
			titleRunes := []rune(title)
			if len(titleRunes) > titleAvail {
				if titleAvail > 1 {
					title = string(titleRunes[:titleAvail-1]) + "…"
				} else {
					title = "…"
				}
			}
		}

		// Build styled line.
		var sb strings.Builder
		if row.prefix != "" {
			sb.WriteString(dimStyle.Render(row.prefix))
		}
		if collapse != "" {
			sb.WriteString(dimStyle.Render(collapse))
		}
		sb.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.statusIconColor(issue.Status))).
			Render(icon))
		sb.WriteString(" ")
		sb.WriteString(priStr)
		sb.WriteString(" ")
		sb.WriteString(issue.ID)
		sb.WriteString("  ")
		sb.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.issueTypeColor(issue.IssueType))).
			Render(title))

		line := sb.String()

		// Pad to full width.
		lineWidth := lipgloss.Width(line)
		if lineWidth < width {
			line += strings.Repeat(" ", width-lineWidth)
		}

		// Highlight selected row (only when list is focused).
		if i == m.tasksViewCursor && m.tasksViewDepth == 0 {
			line = lipgloss.NewStyle().
				Background(lipgloss.Color(m.theme.Highlight)).
				Width(width).
				Render(line)
		}

		lines = append(lines, line)
	}

	// Pad remaining height.
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	return strings.Join(lines, "\n")
}

// statusIcon returns the unicode status icon for an issue status.
func statusIcon(status string) string {
	switch status {
	case "open":
		return "●"
	case "in_progress":
		return "◐"
	case "blocked":
		return "◇"
	case "closed":
		return "✓"
	default:
		return "◌"
	}
}

// statusIconColor returns the theme color for a status icon.
func (m *Model) statusIconColor(status string) string {
	switch status {
	case "open":
		return m.theme.ForegroundDim
	case "in_progress":
		return m.theme.StatusRunning
	case "blocked":
		return m.theme.StatusError
	case "closed":
		return m.theme.StatusSuccess
	default:
		return m.theme.Foreground
	}
}

// issueTypeColor returns the theme color for an issue type's title.
func (m *Model) issueTypeColor(issueType string) string {
	switch issueType {
	case "bug":
		return m.theme.StatusError
	case "feature":
		return m.theme.StatusSuccess
	case "task":
		return m.theme.Foreground
	case "epic":
		return "#d33682" // solarized magenta
	case "chore":
		return "#b58900" // solarized yellow
	case "decision":
		return "#2aa198" // solarized cyan
	default:
		return m.theme.Foreground
	}
}

// handleTasksViewMouse handles mouse events in tasks view.
func (m *Model) handleTasksViewMouse(msg tea.MouseMsg, paneRow int) (tea.Model, tea.Cmd) {
	const mouseScrollLines = 3
	leftWidth := 32

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if msg.X > leftWidth {
			// Right pane scroll.
			m.tasksViewDepth = 1
			if m.tasksViewScroll > 0 {
				m.tasksViewScroll -= mouseScrollLines
				if m.tasksViewScroll < 0 {
					m.tasksViewScroll = 0
				}
			}
		} else {
			m.tasksViewMoveCursor(-mouseScrollLines)
		}
	case tea.MouseButtonWheelDown:
		if msg.X > leftWidth {
			m.tasksViewDepth = 1
			m.tasksViewScroll += mouseScrollLines
		} else {
			m.tasksViewMoveCursor(mouseScrollLines)
		}
	case tea.MouseButtonLeft:
		if msg.X <= leftWidth {
			// Click in left pane selects issue.
			// paneRow is relative to content area (after header).
			tabHeaderHeight := 2
			searchBarHeight := 0
			if m.tasksViewSearchActive {
				searchBarHeight = 1
			}
			listRow := paneRow - tabHeaderHeight - searchBarHeight + m.tasksViewListScroll
			if listRow >= 0 && listRow < len(m.tasksViewVisibleRows) {
				m.tasksViewCursor = listRow
				m.tasksViewDepth = 0
			}
		}
	}
	return m, nil
}
