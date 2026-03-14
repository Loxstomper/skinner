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
	m.tasksViewTab = 0 // Ready tab
	m.tasksViewFiltered = nil
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
		}
		return m, nil

	case config.ActionJumpBottom:
		if m.tasksViewDepth == 1 {
			// Will be clamped during render.
			m.tasksViewScroll = 9999
		} else if len(m.tasksViewFiltered) > 0 {
			m.tasksViewCursor = len(m.tasksViewFiltered) - 1
		}
		return m, nil

	case "search":
		m.tasksViewSearchActive = true
		m.tasksViewSearchQuery = ""
		return m, nil
	}

	// Handle unbound keys by raw key value for tasks-specific actions.
	if m.tasksViewDepth == 0 {
		switch key {
		case "H": // Previous tab.
			if m.tasksViewTab > 0 {
				m.tasksViewTab--
				m.tasksViewCursor = 0
				m.tasksViewSearchActive = false
				m.tasksViewSearchQuery = ""
				m.tasksViewRefilter()
			}
			return m, nil

		case "L": // Next tab.
			if m.tasksViewTab < 3 {
				m.tasksViewTab++
				m.tasksViewCursor = 0
				m.tasksViewSearchActive = false
				m.tasksViewSearchQuery = ""
				m.tasksViewRefilter()
			}
			return m, nil

		case " ": // Toggle expand/collapse.
			if !m.tasksViewFlatMode && len(m.tasksViewFiltered) > 0 && m.tasksViewCursor < len(m.tasksViewFiltered) {
				issue := m.tasksViewFiltered[m.tasksViewCursor]
				m.tasksViewExpanded[issue.ID] = !m.tasksViewExpanded[issue.ID]
			}
			return m, nil

		case "f": // Toggle flat/tree mode.
			m.tasksViewFlatMode = !m.tasksViewFlatMode
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
		// Keep filtered results.
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
		} else {
			m.tasksViewSearchActive = false
			m.tasksViewRefilter()
		}
		return true
	default:
		if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
			m.tasksViewSearchQuery += key
			m.tasksViewRefilter()
			return true
		}
	}
	return false
}

// tasksViewMoveCursor moves the cursor by delta, clamping to bounds.
func (m *Model) tasksViewMoveCursor(delta int) {
	if len(m.tasksViewFiltered) == 0 {
		return
	}
	m.tasksViewCursor += delta
	if m.tasksViewCursor < 0 {
		m.tasksViewCursor = 0
	}
	if m.tasksViewCursor >= len(m.tasksViewFiltered) {
		m.tasksViewCursor = len(m.tasksViewFiltered) - 1
	}
}

// tasksViewRefilter rebuilds the filtered issue list based on current tab and search.
func (m *Model) tasksViewRefilter() {
	if m.tasksViewGraph == nil {
		m.tasksViewFiltered = nil
		return
	}

	var issues []*bd.Issue
	switch m.tasksViewTab {
	case 0: // Ready - open issues (for now, all open; proper ready filtering requires bd ready data)
		for _, issue := range m.tasksViewGraph.Issues {
			if issue.Status == "open" {
				issues = append(issues, issue)
			}
		}
	case 1: // All (excluding closed)
		for _, issue := range m.tasksViewGraph.Issues {
			if issue.Status != "closed" {
				issues = append(issues, issue)
			}
		}
	case 2: // Blocked
		issues = m.tasksViewGraph.FilterByStatus("blocked")
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
	if m.tasksViewCursor >= len(m.tasksViewFiltered) {
		m.tasksViewCursor = max(0, len(m.tasksViewFiltered)-1)
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
	case 0: // Ready (open status)
		for _, issue := range m.tasksViewGraph.Issues {
			if issue.Status == "open" {
				issues = append(issues, issue)
			}
		}
	case 1: // All (excluding closed)
		for _, issue := range m.tasksViewGraph.Issues {
			if issue.Status != "closed" {
				issues = append(issues, issue)
			}
		}
	case 2: // Blocked
		issues = m.tasksViewGraph.FilterByStatus("blocked")
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

// renderTasksViewList renders the left pane issue list.
func (m *Model) renderTasksViewList(width, height int) string {
	if len(m.tasksViewFiltered) == 0 {
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.theme.ForegroundDim)).
			Width(width).
			Height(height).
			Align(lipgloss.Center, lipgloss.Center)
		return style.Render("No issues")
	}

	var lines []string
	for i, issue := range m.tasksViewFiltered {
		if i >= height {
			break
		}
		icon := statusIcon(issue.Status)
		line := fmt.Sprintf("%s %d %s  %s", icon, issue.Priority, issue.ID, issue.Title)

		// Truncate to fit width.
		if len(line) > width {
			line = line[:width-1] + "…"
		}
		// Pad to full width.
		for len(line) < width {
			line += " "
		}

		if i == m.tasksViewCursor {
			style := lipgloss.NewStyle().
				Background(lipgloss.Color(m.theme.Highlight)).
				Width(width)
			line = style.Render(line)
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
			listRow := paneRow - tabHeaderHeight
			if listRow >= 0 && listRow < len(m.tasksViewFiltered) {
				m.tasksViewCursor = listRow
				m.tasksViewDepth = 0
			}
		}
	}
	return m, nil
}
