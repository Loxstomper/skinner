package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/loxstomper/skinner/internal/bd"
)

// tasksViewRenderDetail renders the full issue detail pane with all sections.
func (m *Model) tasksViewRenderDetail(width, height int) string {
	if len(m.tasksViewFiltered) == 0 || m.tasksViewCursor >= len(m.tasksViewFiltered) {
		return strings.Repeat("\n", max(0, height-1))
	}

	issue := m.tasksViewFiltered[m.tasksViewCursor]

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.ForegroundDim))
	fgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Foreground))
	boldStyle := fgStyle.Bold(true)
	separator := dimStyle.Render(strings.Repeat("─", width))

	var sections []string

	// 1. Title + priority badge.
	badge := fmt.Sprintf("[P%d]", issue.Priority)
	badgeStyled := m.priorityStyle(issue.Priority).Bold(true).Render(badge)
	titleStyled := boldStyle.Render(issue.Title)
	sections = append(sections, badgeStyled+" "+titleStyled)

	// Separator after title.
	sections = append(sections, separator)

	// 2. Meta line.
	meta := issue.IssueType + "  " + statusIcon(issue.Status) + " " + issue.Status
	if issue.Assignee != "" {
		meta += "  assigned: " + issue.Assignee
	}
	if issue.Parent != "" {
		meta += "  parent: " + issue.Parent
	}
	sections = append(sections, dimStyle.Render(meta))

	// 3. Labels.
	if len(issue.Labels) > 0 {
		sections = append(sections, dimStyle.Render("Labels: "+strings.Join(issue.Labels, ", ")))
	}

	// 4. Description.
	if issue.Description != "" {
		sections = append(sections, separator)
		// Soft-wrap description to width.
		sections = append(sections, fgStyle.Width(width).Render(issue.Description))
	}

	// 5. Blocking Dependencies.
	if m.tasksViewGraph != nil {
		depNodes := m.tasksViewGraph.WalkBlockingDeps(issue.ID, 3)
		if len(depNodes) > 0 {
			sections = append(sections, separator)
			sections = append(sections, boldStyle.Render("Blocked by"))
			sections = append(sections, renderDepTree(depNodes, dimStyle, "├── ", "└── ", "│   ", "    "))
		}
	}

	// 6. Dependents.
	if m.tasksViewGraph != nil {
		depNodes := m.tasksViewGraph.WalkBlockingDependents(issue.ID, 3)
		if len(depNodes) > 0 {
			sections = append(sections, "")
			sections = append(sections, boldStyle.Render("Blocks"))
			sections = append(sections, renderDepTree(depNodes, dimStyle, "├── ", "└── ", "│   ", "    "))
		}
	}

	// 7. Related (non-blocking deps).
	if m.tasksViewGraph != nil {
		nonBlocking := m.tasksViewGraph.NonBlockingDeps(issue.ID)
		if len(nonBlocking) > 0 {
			sections = append(sections, "")
			sections = append(sections, dimStyle.Render("Related"))
			for _, dep := range nonBlocking {
				depType := dep.DepType()
				id := dep.DependsOnID
				if id == "" {
					id = dep.ID
				}
				title := dep.Title
				// Try to look up title from graph if not in dependency data.
				if title == "" && m.tasksViewGraph != nil {
					if looked := m.tasksViewGraph.ByID[id]; looked != nil {
						title = looked.Title
						if dep.IssueType == "" {
							dep.IssueType = looked.IssueType
						}
					}
				}
				line := fmt.Sprintf("  %s: %s", depType, id)
				if title != "" {
					line += "  " + title
				}
				if dep.IssueType != "" {
					line += " (" + dep.IssueType + ")"
				}
				sections = append(sections, dimStyle.Render(line))
			}
		}
	}

	// 8. Gates.
	if len(issue.Gates) > 0 {
		sections = append(sections, separator)
		sections = append(sections, boldStyle.Render("Gates"))
		for _, gate := range issue.Gates {
			icon := "☐"
			if gate.Complete {
				icon = "☑"
			}
			line := fmt.Sprintf("  %s %s", icon, gate.Type)
			if gate.Detail != "" {
				line += " " + gate.Detail
			}
			sections = append(sections, dimStyle.Render(line))
		}
	}

	// 9. Timestamps.
	sections = append(sections, separator)
	tsLabel := dimStyle.Render
	tsValue := fgStyle.Render
	sections = append(sections,
		tsLabel("Created  ")+tsValue(issue.CreatedAt.Format("2006-01-02 15:04")))
	sections = append(sections,
		tsLabel("Updated  ")+tsValue(issue.UpdatedAt.Format("2006-01-02 15:04")))
	if !issue.ClosedAt.IsZero() {
		closed := tsLabel("Closed   ") + tsValue(issue.ClosedAt.Format("2006-01-02 15:04"))
		if issue.CloseReason != "" {
			closed += "  " + dimStyle.Render(fmt.Sprintf("%q", issue.CloseReason))
		}
		sections = append(sections, closed)
	}

	content := strings.Join(sections, "\n")

	// Apply scroll for depth 1.
	lines := strings.Split(content, "\n")
	if m.tasksViewDepth == 1 && m.tasksViewScroll > 0 {
		if m.tasksViewScroll >= len(lines) {
			m.tasksViewScroll = max(0, len(lines)-1)
		}
		lines = lines[m.tasksViewScroll:]
	}

	// Truncate to height.
	if len(lines) > height {
		lines = lines[:height]
	}

	return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
}

// priorityStyle returns a style colored by priority level.
func (m *Model) priorityStyle(priority int) lipgloss.Style {
	switch priority {
	case 0:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.StatusError))
	case 1:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.StatusRunning))
	case 2:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Foreground))
	default: // 3, 4
		return lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.ForegroundDim))
	}
}

// renderDepTree renders a dependency tree as indented lines with tree-drawing characters.
func renderDepTree(nodes []*bd.DepNode, dimStyle lipgloss.Style, branch, last, cont, space string) string {
	var lines []string
	for i, node := range nodes {
		isLast := i == len(nodes)-1
		prefix := branch
		if isLast {
			prefix = last
		}
		icon := statusIcon(node.Issue.Status)
		label := fmt.Sprintf("%s %s  %s", icon, node.Issue.ID, node.Issue.Title)
		if node.Cycle {
			label += " " + dimStyle.Render("(cycle)")
		}
		lines = append(lines, dimStyle.Render(prefix)+label)

		// Recurse into children.
		if len(node.Children) > 0 {
			childTree := renderDepTree(node.Children, dimStyle, branch, last, cont, space)
			// Indent child lines with continuation or space prefix.
			childIndent := cont
			if isLast {
				childIndent = space
			}
			for _, cl := range strings.Split(childTree, "\n") {
				if cl != "" {
					lines = append(lines, dimStyle.Render(childIndent)+cl)
				}
			}
		}
	}
	return strings.Join(lines, "\n")
}
