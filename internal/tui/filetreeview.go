package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/loxstomper/skinner/internal/theme"
	"github.com/sahilm/fuzzy"
)

// FileTreeView is the left pane component that displays a navigable file tree.
// It owns cursor position, scroll offset, and the tree data.
type FileTreeView struct {
	Cursor int
	Scroll int
	roots  []*FileNode
	rows   []FlatRow // cached flattened visible rows

	// Fuzzy search state
	searching      bool
	searchQuery    string
	searchResults  []searchResult // ranked matches
	searchCursor   int
	preSearchState fileTreeState // snapshot for cancel
}

// searchResult holds a fuzzy match result with its matched node.
type searchResult struct {
	Node         *FileNode
	Path         string
	MatchedIndex []int // character indices that matched
}

// fileTreeState captures tree state for search cancel restore.
type fileTreeState struct {
	cursor        int
	scroll        int
	expandedPaths map[string]bool
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
// When in search mode, it renders the search results with an input bar at the bottom.
func (ftv *FileTreeView) View(props FileTreeViewProps) string {
	style := lipgloss.NewStyle().Width(props.Width).Height(props.Height)

	if ftv.searching {
		return ftv.viewSearch(props, style)
	}

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

// viewSearch renders the search mode: ranked result list + input bar at bottom.
func (ftv *FileTreeView) viewSearch(props FileTreeViewProps, style lipgloss.Style) string {
	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(props.Theme.Foreground))
	highlight := lipgloss.NewStyle().Background(lipgloss.Color(props.Theme.Highlight))

	// Reserve 1 line for the search input bar at the bottom
	listHeight := props.Height - 1
	if listHeight < 0 {
		listHeight = 0
	}

	// Build result lines
	var resultLines []string
	for i, sr := range ftv.searchResults {
		line := "  " + nameStyle.Render(sr.Path)
		if i == ftv.searchCursor {
			displayWidth := lipgloss.Width(line)
			if displayWidth < props.Width {
				line += strings.Repeat(" ", props.Width-displayWidth)
			}
			line = highlight.Render(line)
		}
		resultLines = append(resultLines, line)
	}

	// If no results and there's a query, show a message
	if len(resultLines) == 0 && ftv.searchQuery != "" {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(props.Theme.ForegroundDim))
		resultLines = append(resultLines, "  "+dimStyle.Render("No matches"))
	}

	// Scroll results to keep cursor visible
	start := 0
	if ftv.searchCursor >= listHeight {
		start = ftv.searchCursor - listHeight + 1
	}
	end := start + listHeight
	if end > len(resultLines) {
		end = len(resultLines)
	}
	if start > len(resultLines) {
		start = len(resultLines)
	}

	visible := resultLines[start:end]

	// Pad to fill the list area
	for len(visible) < listHeight {
		visible = append(visible, "")
	}

	// Search input bar at the bottom
	inputBar := nameStyle.Render("/ " + ftv.searchQuery + "█")
	visible = append(visible, inputBar)

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

// IsSearching returns whether the tree view is in fuzzy search mode.
func (ftv *FileTreeView) IsSearching() bool {
	return ftv.searching
}

// SearchQuery returns the current search query.
func (ftv *FileTreeView) SearchQuery() string {
	return ftv.searchQuery
}

// EnterSearch activates fuzzy search mode, saving the current state.
func (ftv *FileTreeView) EnterSearch() {
	ftv.searching = true
	ftv.searchQuery = ""
	ftv.searchResults = nil
	ftv.searchCursor = 0
	ftv.preSearchState = fileTreeState{
		cursor:        ftv.Cursor,
		scroll:        ftv.Scroll,
		expandedPaths: collectExpandedPaths(ftv.roots),
	}
}

// CancelSearch exits search mode and restores the pre-search state.
func (ftv *FileTreeView) CancelSearch() {
	ftv.searching = false
	ftv.searchQuery = ""
	ftv.searchResults = nil
	ftv.searchCursor = 0

	// Restore pre-search cursor and scroll
	ftv.Cursor = ftv.preSearchState.cursor
	ftv.Scroll = ftv.preSearchState.scroll

	// Restore expand state: collapse all, then re-expand saved paths
	collapseAll(ftv.roots)
	restoreExpandedPaths(ftv.roots, ftv.preSearchState.expandedPaths)
	ftv.rebuildRows()
	ftv.clampCursor()
}

// ConfirmSearch exits search mode and navigates to the selected result.
// Returns the selected node (for preview update), or nil.
func (ftv *FileTreeView) ConfirmSearch() *FileNode {
	if !ftv.searching || len(ftv.searchResults) == 0 {
		ftv.CancelSearch()
		return nil
	}

	selected := ftv.searchResults[ftv.searchCursor]
	ftv.searching = false
	ftv.searchQuery = ""
	ftv.searchResults = nil
	ftv.searchCursor = 0

	// Expand all parent directories of the selected file
	expandPathToNode(ftv.roots, selected.Path)
	ftv.rebuildRows()

	// Move cursor to the selected file
	for i, row := range ftv.rows {
		if row.Node.Path == selected.Path {
			ftv.Cursor = i
			break
		}
	}
	ftv.ensureCursorVisible(0)

	return selected.Node
}

// HandleSearchKey processes a key press during fuzzy search.
// Returns: "confirm" if enter was pressed, "cancel" if escape, "" otherwise.
func (ftv *FileTreeView) HandleSearchKey(key string) string {
	switch key {
	case "enter":
		return "confirm"
	case "esc":
		return "cancel"
	case "backspace":
		if len(ftv.searchQuery) > 0 {
			ftv.searchQuery = ftv.searchQuery[:len(ftv.searchQuery)-1]
			ftv.updateSearchResults()
		}
	case "up", "ctrl+p":
		if ftv.searchCursor > 0 {
			ftv.searchCursor--
		}
	case "down", "ctrl+n":
		if ftv.searchCursor < len(ftv.searchResults)-1 {
			ftv.searchCursor++
		}
	default:
		// Only accept printable single characters
		if len(key) == 1 && key[0] >= ' ' && key[0] <= '~' {
			ftv.searchQuery += key
			ftv.updateSearchResults()
		}
	}
	return ""
}

// SearchSelectedNode returns the currently highlighted search result node.
func (ftv *FileTreeView) SearchSelectedNode() *FileNode {
	if !ftv.searching || len(ftv.searchResults) == 0 {
		return nil
	}
	if ftv.searchCursor < 0 || ftv.searchCursor >= len(ftv.searchResults) {
		return nil
	}
	return ftv.searchResults[ftv.searchCursor].Node
}

// updateSearchResults runs fuzzy matching against all file paths.
func (ftv *FileTreeView) updateSearchResults() {
	ftv.searchResults = nil
	ftv.searchCursor = 0

	if ftv.searchQuery == "" {
		return
	}

	// Collect all file paths
	var allFiles []fileEntry
	collectAllFiles(ftv.roots, &allFiles)

	// Build string slice for fuzzy matching
	paths := make([]string, len(allFiles))
	for i, f := range allFiles {
		paths[i] = f.path
	}

	// Run fuzzy match
	matches := fuzzy.Find(ftv.searchQuery, paths)

	for _, m := range matches {
		ftv.searchResults = append(ftv.searchResults, searchResult{
			Node:         allFiles[m.Index].node,
			Path:         allFiles[m.Index].path,
			MatchedIndex: m.MatchedIndexes,
		})
	}
}

// fileEntry pairs a file path with its node for fuzzy matching.
type fileEntry struct {
	path string
	node *FileNode
}

// collectAllFiles recursively collects all file (non-dir) paths from the tree.
func collectAllFiles(nodes []*FileNode, out *[]fileEntry) {
	for _, n := range nodes {
		if n.IsDir {
			collectAllFiles(n.Children, out)
		} else {
			*out = append(*out, fileEntry{path: n.Path, node: n})
		}
	}
}

// expandPathToNode expands all parent directories along the path to a file.
func expandPathToNode(roots []*FileNode, targetPath string) {
	parts := strings.Split(targetPath, "/")
	if len(parts) <= 1 {
		return // root-level file, no parents to expand
	}

	// Build each parent path and expand it
	nodes := roots
	for i := 0; i < len(parts)-1; i++ {
		prefix := strings.Join(parts[:i+1], "/")
		for _, n := range nodes {
			if n.Path == prefix && n.IsDir {
				n.Expanded = true
				nodes = n.Children
				break
			}
		}
	}
}

// collapseAll collapses all directories in the tree.
func collapseAll(nodes []*FileNode) {
	for _, n := range nodes {
		if n.IsDir {
			n.Expanded = false
			collapseAll(n.Children)
		}
	}
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
