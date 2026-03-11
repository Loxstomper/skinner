package tui

import (
	"strings"
	"testing"

	"github.com/loxstomper/skinner/internal/theme"
)

// makeTestNodes creates a simple tree for testing without filesystem access.
//
//	cmd/
//	  main.go
//	internal/
//	  tui/
//	    root.go
//	  app.go
//	go.mod
//	README.md
func makeTestNodes() []*FileNode {
	return []*FileNode{
		{
			Name: "cmd", Path: "cmd", IsDir: true,
			Children: []*FileNode{
				{Name: "main.go", Path: "cmd/main.go"},
			},
		},
		{
			Name: "internal", Path: "internal", IsDir: true,
			Children: []*FileNode{
				{
					Name: "tui", Path: "internal/tui", IsDir: true,
					Children: []*FileNode{
						{Name: "root.go", Path: "internal/tui/root.go"},
					},
				},
				{Name: "app.go", Path: "internal/app.go"},
			},
		},
		{Name: "go.mod", Path: "go.mod"},
		{Name: "README.md", Path: "README.md"},
	}
}

func fileTreeTestTheme() theme.Theme {
	return theme.Theme{
		Foreground:    "#ffffff",
		ForegroundDim: "#888888",
		Highlight:     "#333333",
		DiffAdded:     "#00ff00",
		DiffRemoved:   "#ff0000",
	}
}

func testProps(w, h int) FileTreeViewProps {
	return FileTreeViewProps{
		Width:   w,
		Height:  h,
		Focused: true,
		Theme:   fileTreeTestTheme(),
	}
}

func TestNewFileTreeView_FirstLevelExpanded(t *testing.T) {
	nodes := makeTestNodes()
	ftv := NewFileTreeView(nodes)

	// First-level dirs should be expanded
	for _, n := range ftv.Roots() {
		if n.IsDir && !n.Expanded {
			t.Errorf("first-level dir %q should be expanded", n.Name)
		}
	}

	// Nested dirs should NOT be expanded
	for _, child := range nodes[1].Children {
		if child.IsDir && child.Expanded {
			t.Errorf("nested dir %q should not be expanded", child.Name)
		}
	}
}

func TestNewFileTreeView_VisibleRows(t *testing.T) {
	nodes := makeTestNodes()
	ftv := NewFileTreeView(nodes)
	rows := ftv.VisibleRows()

	// With first level expanded:
	// cmd/ (depth 0), main.go (depth 1), internal/ (depth 0), tui/ (depth 1), app.go (depth 1), go.mod (depth 0), README.md (depth 0)
	expected := []struct {
		name  string
		depth int
	}{
		{"cmd", 0},
		{"main.go", 1},
		{"internal", 0},
		{"tui", 1},
		{"app.go", 1},
		{"go.mod", 0},
		{"README.md", 0},
	}

	if len(rows) != len(expected) {
		t.Fatalf("expected %d rows, got %d", len(expected), len(rows))
	}
	for i, e := range expected {
		if rows[i].Node.Name != e.name || rows[i].Depth != e.depth {
			t.Errorf("row %d: expected %s@%d, got %s@%d", i, e.name, e.depth, rows[i].Node.Name, rows[i].Depth)
		}
	}
}

func TestFileTreeView_MoveDown(t *testing.T) {
	ftv := NewFileTreeView(makeTestNodes())
	props := testProps(40, 10)

	ftv.HandleAction("move_down", props)
	if ftv.Cursor != 1 {
		t.Errorf("expected cursor 1, got %d", ftv.Cursor)
	}

	ftv.HandleAction("move_down", props)
	if ftv.Cursor != 2 {
		t.Errorf("expected cursor 2, got %d", ftv.Cursor)
	}
}

func TestFileTreeView_MoveUp(t *testing.T) {
	ftv := NewFileTreeView(makeTestNodes())
	props := testProps(40, 10)

	ftv.Cursor = 3
	ftv.HandleAction("move_up", props)
	if ftv.Cursor != 2 {
		t.Errorf("expected cursor 2, got %d", ftv.Cursor)
	}
}

func TestFileTreeView_MoveUpAtTop(t *testing.T) {
	ftv := NewFileTreeView(makeTestNodes())
	props := testProps(40, 10)

	ftv.HandleAction("move_up", props)
	if ftv.Cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", ftv.Cursor)
	}
}

func TestFileTreeView_MoveDownAtBottom(t *testing.T) {
	ftv := NewFileTreeView(makeTestNodes())
	props := testProps(40, 10)
	rows := ftv.VisibleRows()

	ftv.Cursor = len(rows) - 1
	ftv.HandleAction("move_down", props)
	if ftv.Cursor != len(rows)-1 {
		t.Errorf("cursor should stay at bottom, got %d", ftv.Cursor)
	}
}

func TestFileTreeView_JumpTop(t *testing.T) {
	ftv := NewFileTreeView(makeTestNodes())
	props := testProps(40, 10)

	ftv.Cursor = 4
	ftv.HandleAction("jump_top", props)
	if ftv.Cursor != 0 {
		t.Errorf("expected cursor 0, got %d", ftv.Cursor)
	}
	if ftv.Scroll != 0 {
		t.Errorf("expected scroll 0, got %d", ftv.Scroll)
	}
}

func TestFileTreeView_JumpBottom(t *testing.T) {
	ftv := NewFileTreeView(makeTestNodes())
	props := testProps(40, 10)
	rowCount := len(ftv.VisibleRows())

	ftv.HandleAction("jump_bottom", props)
	if ftv.Cursor != rowCount-1 {
		t.Errorf("expected cursor %d, got %d", rowCount-1, ftv.Cursor)
	}
}

func TestFileTreeView_PageDown(t *testing.T) {
	ftv := NewFileTreeView(makeTestNodes())
	props := testProps(40, 3) // Small viewport
	rowCount := len(ftv.VisibleRows())

	ftv.HandleAction("page_down", props)
	if ftv.Cursor != 3 {
		t.Errorf("expected cursor 3, got %d", ftv.Cursor)
	}

	// Page down again — should clamp to last row
	ftv.HandleAction("page_down", props)
	if ftv.Cursor != 6 {
		t.Errorf("expected cursor 6, got %d", ftv.Cursor)
	}

	ftv.HandleAction("page_down", props)
	if ftv.Cursor != rowCount-1 {
		t.Errorf("expected cursor %d, got %d", rowCount-1, ftv.Cursor)
	}
}

func TestFileTreeView_PageUp(t *testing.T) {
	ftv := NewFileTreeView(makeTestNodes())
	props := testProps(40, 3)

	ftv.Cursor = 5
	ftv.HandleAction("page_up", props)
	if ftv.Cursor != 2 {
		t.Errorf("expected cursor 2, got %d", ftv.Cursor)
	}

	ftv.HandleAction("page_up", props)
	if ftv.Cursor != 0 {
		t.Errorf("expected cursor 0, got %d", ftv.Cursor)
	}
}

func TestFileTreeView_ScrollToCursor(t *testing.T) {
	ftv := NewFileTreeView(makeTestNodes())
	props := testProps(40, 3) // Only 3 rows visible

	// Move cursor past viewport
	ftv.Cursor = 5
	ftv.ensureCursorVisible(props.Height)

	if ftv.Scroll > ftv.Cursor || ftv.Cursor >= ftv.Scroll+props.Height {
		t.Errorf("cursor %d should be visible in scroll range [%d, %d)", ftv.Cursor, ftv.Scroll, ftv.Scroll+props.Height)
	}
}

func TestFileTreeView_EnterOnDir(t *testing.T) {
	nodes := makeTestNodes()
	// Start with internal/tui collapsed
	ftv := NewFileTreeView(nodes)
	props := testProps(40, 10)

	// Cursor on tui/ (row 3, collapsed)
	ftv.Cursor = 3
	enterDepth2 := ftv.HandleAction("expand", props)
	if enterDepth2 {
		t.Error("enter on dir should not signal depth 2")
	}
	// tui/ should now be expanded
	if !nodes[1].Children[0].Expanded {
		t.Error("tui/ should be expanded after enter")
	}

	// Enter again to collapse
	ftv.HandleAction("expand", props)
	if nodes[1].Children[0].Expanded {
		t.Error("tui/ should be collapsed after second enter")
	}
}

func TestFileTreeView_EnterOnFile(t *testing.T) {
	ftv := NewFileTreeView(makeTestNodes())
	props := testProps(40, 10)

	// Cursor on main.go (row 1)
	ftv.Cursor = 1
	enterDepth2 := ftv.HandleAction("expand", props)
	if !enterDepth2 {
		t.Error("enter on file should signal depth 2")
	}
}

func TestFileTreeView_LeftOnExpandedDir(t *testing.T) {
	nodes := makeTestNodes()
	ftv := NewFileTreeView(nodes)
	props := testProps(40, 10)

	// Cursor on cmd/ (row 0, expanded)
	ftv.Cursor = 0
	ftv.HandleAction("focus_left", props)

	if nodes[0].Expanded {
		t.Error("cmd/ should be collapsed after h")
	}
}

func TestFileTreeView_LeftOnFile(t *testing.T) {
	nodes := makeTestNodes()
	ftv := NewFileTreeView(nodes)
	props := testProps(40, 10)

	// Cursor on main.go (row 1, child of cmd/)
	ftv.Cursor = 1
	ftv.HandleAction("focus_left", props)

	// cmd/ should be collapsed, cursor should be on cmd/
	if nodes[0].Expanded {
		t.Error("cmd/ should be collapsed after h on child file")
	}
	if ftv.SelectedNode().Name != "cmd" {
		t.Errorf("cursor should be on cmd/, got %s", ftv.SelectedNode().Name)
	}
}

func TestFileTreeView_LeftOnRootFile(t *testing.T) {
	ftv := NewFileTreeView(makeTestNodes())
	props := testProps(40, 10)

	// Cursor on go.mod (row 5, root level)
	ftv.Cursor = 5
	ftv.HandleAction("focus_left", props)

	// No-op — cursor should stay
	if ftv.Cursor != 5 {
		t.Errorf("h on root file should be no-op, cursor moved to %d", ftv.Cursor)
	}
}

func TestFileTreeView_RightOnCollapsedDir(t *testing.T) {
	nodes := makeTestNodes()
	ftv := NewFileTreeView(nodes)
	props := testProps(40, 10)

	// Cursor on tui/ (row 3, collapsed)
	ftv.Cursor = 3
	enterDepth2 := ftv.HandleAction("focus_right", props)

	if enterDepth2 {
		t.Error("l on collapsed dir should not signal depth 2")
	}
	if !nodes[1].Children[0].Expanded {
		t.Error("tui/ should be expanded after l")
	}
}

func TestFileTreeView_RightOnExpandedDir(t *testing.T) {
	nodes := makeTestNodes()
	ftv := NewFileTreeView(nodes)
	props := testProps(40, 10)

	// Cursor on cmd/ (row 0, expanded)
	ftv.Cursor = 0
	enterDepth2 := ftv.HandleAction("focus_right", props)

	if enterDepth2 {
		t.Error("l on expanded dir should not signal depth 2")
	}
	// Cursor should move to first child (main.go, row 1)
	if ftv.Cursor != 1 {
		t.Errorf("cursor should move to first child (row 1), got %d", ftv.Cursor)
	}
	if ftv.SelectedNode().Name != "main.go" {
		t.Errorf("cursor should be on main.go, got %s", ftv.SelectedNode().Name)
	}
}

func TestFileTreeView_RightOnFile(t *testing.T) {
	ftv := NewFileTreeView(makeTestNodes())
	props := testProps(40, 10)

	// Cursor on main.go (row 1)
	ftv.Cursor = 1
	enterDepth2 := ftv.HandleAction("focus_right", props)

	if !enterDepth2 {
		t.Error("l on file should signal depth 2")
	}
}

func TestFileTreeView_ViewContainsTreeStructure(t *testing.T) {
	ftv := NewFileTreeView(makeTestNodes())
	props := testProps(50, 10)

	output := ftv.View(props)

	// Check that key elements are present
	if !strings.Contains(output, "▼") {
		t.Error("output should contain ▼ for expanded dirs")
	}
	if !strings.Contains(output, "▶") {
		t.Error("output should contain ▶ for collapsed dirs")
	}
	if !strings.Contains(output, "cmd/") {
		t.Error("output should contain cmd/")
	}
	if !strings.Contains(output, "main.go") {
		t.Error("output should contain main.go")
	}
	if !strings.Contains(output, "internal/") {
		t.Error("output should contain internal/")
	}
}

func TestFileTreeView_ViewGitStatus(t *testing.T) {
	nodes := makeTestNodes()
	// Set git status on some files
	nodes[2].Status = GitStatusModified              // go.mod
	nodes[0].Children[0].Status = GitStatusUntracked // cmd/main.go
	nodes[0].Status = GitStatusUntracked             // inherited

	ftv := NewFileTreeView(nodes)
	props := testProps(50, 10)

	output := ftv.View(props)

	if !strings.Contains(output, "M") {
		t.Error("output should contain M for modified file")
	}
	if !strings.Contains(output, "?") {
		t.Error("output should contain ? for untracked file")
	}
}

func TestFileTreeView_ViewSymlink(t *testing.T) {
	nodes := []*FileNode{
		{Name: "link.txt", Path: "link.txt", IsSymlink: true, SymlinkTarget: "../real.txt"},
	}
	ftv := NewFileTreeView(nodes)
	props := testProps(60, 10)

	output := ftv.View(props)

	if !strings.Contains(output, "→") {
		t.Error("output should contain → for symlink target")
	}
	if !strings.Contains(output, "🔗") {
		t.Error("output should contain 🔗 for symlink indicator")
	}
}

func TestFileTreeView_EmptyTree(t *testing.T) {
	ftv := NewFileTreeView(nil)
	props := testProps(40, 10)

	output := ftv.View(props)
	if !strings.Contains(output, "No files") {
		t.Error("empty tree should show 'No files'")
	}
}

func TestFileTreeView_SelectedNode(t *testing.T) {
	ftv := NewFileTreeView(makeTestNodes())

	node := ftv.SelectedNode()
	if node == nil {
		t.Fatal("selected node should not be nil")
	}
	if node.Name != "cmd" {
		t.Errorf("initial selected node should be cmd, got %s", node.Name)
	}
}

func TestFileTreeView_EmptySelectedNode(t *testing.T) {
	ftv := NewFileTreeView(nil)
	if ftv.SelectedNode() != nil {
		t.Error("empty tree should return nil selected node")
	}
}

func TestFileTreeView_ScrollBy(t *testing.T) {
	ftv := NewFileTreeView(makeTestNodes())
	ftv.ScrollBy(2, 3)

	if ftv.Scroll != 2 {
		t.Errorf("expected scroll 2, got %d", ftv.Scroll)
	}

	// Scroll past max
	ftv.ScrollBy(100, 3)
	maxScroll := len(ftv.VisibleRows()) - 3
	if ftv.Scroll != maxScroll {
		t.Errorf("expected scroll clamped to %d, got %d", maxScroll, ftv.Scroll)
	}
}

func TestFileTreeView_ClickRow(t *testing.T) {
	nodes := makeTestNodes()
	ftv := NewFileTreeView(nodes)

	// Click on row 1 (main.go)
	enterDepth2 := ftv.ClickRow(1)
	if enterDepth2 {
		t.Error("click on file should not enter depth 2")
	}
	if ftv.Cursor != 1 {
		t.Errorf("expected cursor 1, got %d", ftv.Cursor)
	}

	// Click on row 0 (cmd/ — expanded) → should collapse
	ftv.ClickRow(0)
	if nodes[0].Expanded {
		t.Error("click on expanded dir should collapse it")
	}
}

func TestFileTreeView_HandleActionEmpty(t *testing.T) {
	ftv := NewFileTreeView(nil)
	props := testProps(40, 10)

	// All actions should be no-ops on empty tree
	actions := []string{"move_down", "move_up", "jump_top", "jump_bottom", "page_down", "page_up", "expand", "focus_left", "focus_right"}
	for _, action := range actions {
		enterDepth2 := ftv.HandleAction(action, props)
		if enterDepth2 {
			t.Errorf("action %q on empty tree should not signal depth 2", action)
		}
	}
}

func TestFileTreeView_LeftOnCollapsedDir(t *testing.T) {
	nodes := makeTestNodes()
	ftv := NewFileTreeView(nodes)
	props := testProps(40, 10)

	// First collapse cmd/ so it's a collapsed dir
	nodes[0].Expanded = false
	ftv.rebuildRows()

	// Cursor on cmd/ (collapsed, root level)
	ftv.Cursor = 0
	ftv.HandleAction("focus_left", props)

	// Root-level collapsed dir → no-op
	if ftv.Cursor != 0 {
		t.Errorf("h on root collapsed dir should be no-op, cursor moved to %d", ftv.Cursor)
	}
}

func TestFileTreeView_LeftCollapsesParentFromNestedFile(t *testing.T) {
	nodes := makeTestNodes()
	ftv := NewFileTreeView(nodes)
	props := testProps(40, 10)

	// Expand tui/ so we can navigate to root.go
	nodes[1].Children[0].Expanded = true
	ftv.rebuildRows()

	// Find root.go in rows
	var rootGoIdx int
	for i, row := range ftv.VisibleRows() {
		if row.Node.Name == "root.go" {
			rootGoIdx = i
			break
		}
	}

	ftv.Cursor = rootGoIdx
	ftv.HandleAction("focus_left", props)

	// tui/ should be collapsed, cursor on tui/
	if nodes[1].Children[0].Expanded {
		t.Error("tui/ should be collapsed after h on root.go")
	}
	if ftv.SelectedNode().Name != "tui" {
		t.Errorf("cursor should be on tui/, got %s", ftv.SelectedNode().Name)
	}
}
