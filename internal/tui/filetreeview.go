package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/loxstomper/skinner/internal/theme"
)

// FileTreeView is the left pane component that displays a navigable file tree.
// It owns cursor position, scroll offset, and the tree data.
type FileTreeView struct {
	Cursor int
	Scroll int
	roots  []*FileNode
	rows   []FlatRow // cached flattened visible rows
}

// FileTreeViewProps contains the data needed to render the file tree.
type FileTreeViewProps struct {
	Width   int
	Height  int
	Focused bool
	Theme   theme.Theme
}

// NewFileTreeView creates a FileTreeView with the first level expanded.
func NewFileTreeView(roots []*FileNode) *FileTreeView {
	// Expand first level (CWD direct children that are dirs)
	for _, node := range roots {
		if node.IsDir {
			node.Expanded = true
		}
	}
	ftv := &FileTreeView{roots: roots}
	ftv.rebuildRows()
	return ftv
}

// Roots returns the tree roots.
func (ftv *FileTreeView) Roots() []*FileNode {
	return ftv.roots
}

// SetRoots replaces the tree data, preserving cursor and expand state where possible.
func (ftv *FileTreeView) SetRoots(roots []*FileNode) {
	ftv.roots = roots
	ftv.rebuildRows()
	ftv.clampCursor()
}

// SelectedNode returns the currently selected node, or nil if empty.
func (ftv *FileTreeView) SelectedNode() *FileNode {
	if len(ftv.rows) == 0 {
		return nil
	}
	if ftv.Cursor < 0 || ftv.Cursor >= len(ftv.rows) {
		return nil
	}
	return ftv.rows[ftv.Cursor].Node
}

// VisibleRows returns the current flattened visible rows.
func (ftv *FileTreeView) VisibleRows() []FlatRow {
	return ftv.rows
}

// HandleAction processes a navigation action. Returns true if the action
// signals entering depth 2 (scrollable preview).
func (ftv *FileTreeView) HandleAction(action string, props FileTreeViewProps) bool {
	count := len(ftv.rows)
	if count == 0 {
		return false
	}

	switch action {
	case "move_down":
		if ftv.Cursor < count-1 {
			ftv.Cursor++
		}
		ftv.ensureCursorVisible(props.Height)

	case "move_up":
		if ftv.Cursor > 0 {
			ftv.Cursor--
		}
		ftv.ensureCursorVisible(props.Height)

	case "jump_top":
		ftv.Cursor = 0
		ftv.Scroll = 0

	case "jump_bottom":
		ftv.Cursor = count - 1
		ftv.ensureCursorVisible(props.Height)

	case "page_down":
		ftv.Cursor += props.Height
		if ftv.Cursor >= count {
			ftv.Cursor = count - 1
		}
		ftv.ensureCursorVisible(props.Height)

	case "page_up":
		ftv.Cursor -= props.Height
		if ftv.Cursor < 0 {
			ftv.Cursor = 0
		}
		ftv.ensureCursorVisible(props.Height)

	case "expand":
		return ftv.handleEnter()

	case "focus_left":
		return ftv.handleLeft()

	case "focus_right":
		return ftv.handleRight()
	}

	return false
}

// handleEnter: enter on dir → toggle expand/collapse; enter on file → signal depth 2.
func (ftv *FileTreeView) handleEnter() bool {
	node := ftv.SelectedNode()
	if node == nil {
		return false
	}
	if node.IsDir {
		node.Expanded = !node.Expanded
		ftv.rebuildRows()
		ftv.clampCursor()
		return false
	}
	// File: enter depth 2
	return true
}

// handleLeft implements h/← tree navigation.
func (ftv *FileTreeView) handleLeft() bool {
	node := ftv.SelectedNode()
	if node == nil {
		return false
	}

	if node.IsDir && node.Expanded {
		// Expanded dir → collapse it
		node.Expanded = false
		ftv.rebuildRows()
		ftv.clampCursor()
		return false
	}

	// File or collapsed dir → collapse parent, cursor to parent
	parent := FindParent(ftv.roots, node.Path)
	if parent == nil {
		// Root-level item — no-op per spec
		return false
	}
	parent.Expanded = false
	ftv.rebuildRows()
	// Move cursor to parent
	for i, row := range ftv.rows {
		if row.Node == parent {
			ftv.Cursor = i
			break
		}
	}
	ftv.ensureCursorVisible(0) // will use clamp
	return false
}

// handleRight implements l/→ tree navigation.
func (ftv *FileTreeView) handleRight() bool {
	node := ftv.SelectedNode()
	if node == nil {
		return false
	}

	if node.IsDir {
		if !node.Expanded {
			// Collapsed dir → expand it
			node.Expanded = true
			ftv.rebuildRows()
			return false
		}
		// Expanded dir → cursor to first child
		if len(node.Children) > 0 {
			for i, row := range ftv.rows {
				if row.Node == node.Children[0] {
					ftv.Cursor = i
					break
				}
			}
			ftv.ensureCursorVisible(0)
		}
		return false
	}

	// File → signal enter depth 2
	return true
}

// View renders the file tree with the visible slice based on scroll offset.
func (ftv *FileTreeView) View(props FileTreeViewProps) string {
	style := lipgloss.NewStyle().Width(props.Width).Height(props.Height)

	if len(ftv.rows) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(props.Theme.ForegroundDim))
		return style.Render("  " + emptyStyle.Render("No files"))
	}

	highlight := lipgloss.NewStyle().Background(lipgloss.Color(props.Theme.Highlight))
	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(props.Theme.Foreground))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(props.Theme.ForegroundDim))
	addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(props.Theme.DiffAdded))
	removedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(props.Theme.DiffRemoved))

	var allLines []string
	for i, row := range ftv.rows {
		line := ftv.renderRow(row, props.Width, nameStyle, dimStyle, addedStyle, removedStyle)

		if i == ftv.Cursor && props.Focused {
			displayWidth := lipgloss.Width(line)
			if displayWidth < props.Width {
				line += strings.Repeat(" ", props.Width-displayWidth)
			}
			line = highlight.Render(line)
		}
		allLines = append(allLines, line)
	}

	// Apply scroll
	start := ftv.Scroll
	if start >= len(allLines) {
		start = len(allLines) - 1
	}
	if start < 0 {
		start = 0
	}
	end := start + props.Height
	if end > len(allLines) {
		end = len(allLines)
	}

	visible := allLines[start:end]
	content := strings.Join(visible, "\n")
	return style.Render(content)
}

// renderRow renders a single tree row with indent, icon, name, symlink info, and git status.
func (ftv *FileTreeView) renderRow(row FlatRow, width int, nameStyle, dimStyle, addedStyle, removedStyle lipgloss.Style) string {
	node := row.Node
	indent := strings.Repeat("  ", row.Depth)

	// Tree icon
	var icon string
	if node.IsDir {
		if node.Expanded {
			icon = "▼ "
		} else {
			icon = "▶ "
		}
	} else {
		icon = "  "
	}

	// Name with trailing slash for dirs
	name := node.Name
	if node.IsDir {
		name += "/"
	}

	// Symlink indicator
	var symlinkSuffix string
	if node.IsSymlink {
		symlinkSuffix = fmt.Sprintf(" → %s 🔗", node.SymlinkTarget)
	}

	// Git status indicator
	var statusStr string
	var statusStyle lipgloss.Style
	switch node.Status {
	case GitStatusModified:
		statusStr = "M"
		statusStyle = dimStyle
	case GitStatusAdded:
		statusStr = "A"
		statusStyle = addedStyle
	case GitStatusDeleted:
		statusStr = "D"
		statusStyle = removedStyle
	case GitStatusUntracked:
		statusStr = "?"
		statusStyle = addedStyle
	}

	// Build the left part (indent + icon + name + symlink)
	leftPart := indent + nameStyle.Render(icon+name)
	if symlinkSuffix != "" {
		leftPart += dimStyle.Render(symlinkSuffix)
	}

	if statusStr == "" {
		return leftPart
	}

	// Right-align the status indicator
	leftWidth := lipgloss.Width(leftPart)
	styledStatus := statusStyle.Render(statusStr)
	statusWidth := lipgloss.Width(styledStatus)
	gap := width - leftWidth - statusWidth
	if gap < 1 {
		gap = 1
	}
	return leftPart + strings.Repeat(" ", gap) + styledStatus
}

// ScrollBy adjusts the scroll offset by delta lines for mouse scrolling.
func (ftv *FileTreeView) ScrollBy(delta int, height int) {
	ftv.Scroll += delta
	ftv.clampScroll(height)
}

// ClickRow handles a mouse click on a pane-relative row.
// Returns true if the click signals entering depth 2 (file selected).
func (ftv *FileTreeView) ClickRow(row int) bool {
	idx := ftv.Scroll + row
	if idx < 0 || idx >= len(ftv.rows) {
		return false
	}
	ftv.Cursor = idx
	node := ftv.rows[idx].Node
	if node.IsDir {
		node.Expanded = !node.Expanded
		ftv.rebuildRows()
		ftv.clampCursor()
		return false
	}
	return false // Click selects; enter required for depth 2
}

// rebuildRows regenerates the flattened visible rows from the tree.
func (ftv *FileTreeView) rebuildRows() {
	ftv.rows = FlattenVisible(ftv.roots)
}

// clampCursor ensures cursor is within valid bounds.
func (ftv *FileTreeView) clampCursor() {
	if ftv.Cursor >= len(ftv.rows) {
		ftv.Cursor = len(ftv.rows) - 1
	}
	if ftv.Cursor < 0 {
		ftv.Cursor = 0
	}
}

// ensureCursorVisible adjusts scroll so cursor is within the viewport.
func (ftv *FileTreeView) ensureCursorVisible(height int) {
	if height <= 0 {
		return
	}
	if ftv.Cursor < ftv.Scroll {
		ftv.Scroll = ftv.Cursor
	}
	if ftv.Cursor >= ftv.Scroll+height {
		ftv.Scroll = ftv.Cursor - height + 1
	}
	ftv.clampScroll(height)
}

// clampScroll ensures scroll doesn't exceed the maximum valid offset.
func (ftv *FileTreeView) clampScroll(height int) {
	maxScroll := len(ftv.rows) - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if ftv.Scroll > maxScroll {
		ftv.Scroll = maxScroll
	}
	if ftv.Scroll < 0 {
		ftv.Scroll = 0
	}
}
