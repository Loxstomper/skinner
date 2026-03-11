package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/loxstomper/skinner/internal/config"
	"github.com/loxstomper/skinner/internal/executor"
	"github.com/loxstomper/skinner/internal/model"
)

// newFileExplorerTestModel creates a Model for file explorer tests.
func newFileExplorerTestModel(workDir string) *Model {
	fake := &executor.FakeExecutor{}
	sess := model.Session{
		Mode:          "idle",
		MaxIterations: 0,
		StartTime:     time.Now(),
	}
	cfg := config.DefaultConfig()
	th := testTheme()
	m := NewModel(sess, cfg, "", th, false, false, fake)
	m.width = 120
	m.height = 30
	m.workDir = workDir
	return &m
}

// setupFileExplorerTree sets up a file explorer with test tree data,
// bypassing actual filesystem walk.
func setupFileExplorerTree(m *Model) {
	m.fileExplorerActive = true
	m.fileExplorerDepth = 0
	m.fileExplorerTree = NewFileTreeView([]*FileNode{
		{Name: "cmd", Path: "cmd", IsDir: true, Children: []*FileNode{
			{Name: "main.go", Path: "cmd/main.go"},
		}},
		{Name: "internal", Path: "internal", IsDir: true, Children: []*FileNode{
			{Name: "tui", Path: "internal/tui", IsDir: true, Children: []*FileNode{
				{Name: "root.go", Path: "internal/tui/root.go"},
				{Name: "view.go", Path: "internal/tui/view.go"},
			}},
		}},
		{Name: "go.mod", Path: "go.mod"},
		{Name: "README.md", Path: "README.md"},
	})
	m.filePreviewScroll = 0
	m.filePreviewHScroll = 0
}

// TestFileExplorerEnterExit verifies entering and exiting file explorer preserves state.
func TestFileExplorerEnterExit(t *testing.T) {
	m := newFileExplorerTestModel(t.TempDir())

	// Record initial state
	initialFocus := m.focusedPane
	initialIterCursor := m.iterList.Cursor

	// Enter file explorer
	setupFileExplorerTree(m)

	if !m.fileExplorerActive {
		t.Fatal("expected file explorer to be active")
	}
	if m.fileExplorerDepth != 0 {
		t.Errorf("expected depth 0, got %d", m.fileExplorerDepth)
	}
	if m.fileExplorerTree == nil {
		t.Fatal("expected file explorer tree to be non-nil")
	}

	// Exit file explorer
	m.exitFileExplorer()

	if m.fileExplorerActive {
		t.Error("expected file explorer to be inactive after exit")
	}
	if m.fileExplorerTree != nil {
		t.Error("expected tree to be nil after exit")
	}

	// Verify original state is preserved
	if m.focusedPane != initialFocus {
		t.Errorf("expected focus %d preserved, got %d", initialFocus, m.focusedPane)
	}
	if m.iterList.Cursor != initialIterCursor {
		t.Errorf("expected iter cursor %d preserved, got %d", initialIterCursor, m.iterList.Cursor)
	}
}

// TestFileExplorerEscAtDepth0Exits verifies escape at depth 0 exits file explorer.
func TestFileExplorerEscAtDepth0Exits(t *testing.T) {
	m := newFileExplorerTestModel(t.TempDir())
	setupFileExplorerTree(m)

	sendSpecialKey(m, tea.KeyEscape)

	if m.fileExplorerActive {
		t.Error("escape at depth 0 should exit file explorer")
	}
}

// TestFileExplorerEscAtDepth1GoesBack verifies escape at depth 1 returns to depth 0.
func TestFileExplorerEscAtDepth1GoesBack(t *testing.T) {
	m := newFileExplorerTestModel(t.TempDir())
	setupFileExplorerTree(m)
	m.fileExplorerDepth = 1
	m.filePreviewScroll = 10
	m.filePreviewHScroll = 5

	sendSpecialKey(m, tea.KeyEscape)

	if m.fileExplorerDepth != 0 {
		t.Errorf("expected depth 0 after esc, got %d", m.fileExplorerDepth)
	}
	if m.fileExplorerActive != true {
		t.Error("file explorer should still be active after esc at depth 1")
	}
	if m.filePreviewScroll != 0 {
		t.Errorf("expected preview scroll reset to 0, got %d", m.filePreviewScroll)
	}
	if m.filePreviewHScroll != 0 {
		t.Errorf("expected preview hscroll reset to 0, got %d", m.filePreviewHScroll)
	}
}

// TestFileExplorerDepthTransition verifies enter on a file transitions to depth 1.
func TestFileExplorerDepthTransition(t *testing.T) {
	m := newFileExplorerTestModel(t.TempDir())
	setupFileExplorerTree(m)

	// Move cursor to a file (go.mod is at index after expanded dirs)
	// The tree starts with first-level dirs expanded, so:
	// 0: cmd/  (expanded)
	// 1:   main.go
	// 2: internal/ (expanded)
	// 3:   tui/ (collapsed)
	// 4: go.mod
	// 5: README.md
	m.fileExplorerTree.Cursor = 4 // go.mod

	// Press enter
	sendSpecialKey(m, tea.KeyEnter)

	if m.fileExplorerDepth != 1 {
		t.Errorf("expected depth 1 after enter on file, got %d", m.fileExplorerDepth)
	}
}

// TestFileExplorerEnterOnDirToggles verifies enter on a dir toggles expand/collapse.
func TestFileExplorerEnterOnDirToggles(t *testing.T) {
	m := newFileExplorerTestModel(t.TempDir())
	setupFileExplorerTree(m)

	// Cursor starts at index 0 (cmd/, expanded)
	m.fileExplorerTree.Cursor = 0
	node := m.fileExplorerTree.SelectedNode()
	if !node.Expanded {
		t.Fatal("cmd/ should be expanded initially")
	}

	// Press enter → collapse
	sendSpecialKey(m, tea.KeyEnter)

	node = m.fileExplorerTree.SelectedNode()
	if node.Expanded {
		t.Error("cmd/ should be collapsed after enter")
	}
	if m.fileExplorerDepth != 0 {
		t.Errorf("depth should remain 0 after toggling dir, got %d", m.fileExplorerDepth)
	}
}

// TestFileExplorerNavigation verifies j/k move cursor in the tree.
func TestFileExplorerNavigation(t *testing.T) {
	m := newFileExplorerTestModel(t.TempDir())
	setupFileExplorerTree(m)

	if m.fileExplorerTree.Cursor != 0 {
		t.Fatalf("expected initial cursor 0, got %d", m.fileExplorerTree.Cursor)
	}

	// j moves down
	m.handleFileExplorerKey(config.ActionMoveDown)
	if m.fileExplorerTree.Cursor != 1 {
		t.Errorf("expected cursor 1 after move_down, got %d", m.fileExplorerTree.Cursor)
	}

	// k moves up
	m.handleFileExplorerKey(config.ActionMoveUp)
	if m.fileExplorerTree.Cursor != 0 {
		t.Errorf("expected cursor 0 after move_up, got %d", m.fileExplorerTree.Cursor)
	}
}

// TestFileExplorerPreviewScrollReset verifies cursor movement resets preview scroll.
func TestFileExplorerPreviewScrollReset(t *testing.T) {
	m := newFileExplorerTestModel(t.TempDir())
	setupFileExplorerTree(m)
	m.filePreviewScroll = 15
	m.filePreviewHScroll = 5

	m.handleFileExplorerKey(config.ActionMoveDown)

	if m.filePreviewScroll != 0 {
		t.Errorf("expected preview scroll reset to 0, got %d", m.filePreviewScroll)
	}
	if m.filePreviewHScroll != 0 {
		t.Errorf("expected preview hscroll reset to 0, got %d", m.filePreviewHScroll)
	}
}

// TestFileExplorerDepth1Scrolling verifies j/k scroll the preview at depth 1.
func TestFileExplorerDepth1Scrolling(t *testing.T) {
	m := newFileExplorerTestModel(t.TempDir())
	setupFileExplorerTree(m)
	m.fileExplorerDepth = 1

	m.handleFileExplorerKey(config.ActionMoveDown)
	if m.filePreviewScroll != 1 {
		t.Errorf("expected scroll 1 after j at depth 1, got %d", m.filePreviewScroll)
	}

	m.handleFileExplorerKey(config.ActionMoveDown)
	if m.filePreviewScroll != 2 {
		t.Errorf("expected scroll 2 after second j, got %d", m.filePreviewScroll)
	}

	m.handleFileExplorerKey(config.ActionMoveUp)
	if m.filePreviewScroll != 1 {
		t.Errorf("expected scroll 1 after k, got %d", m.filePreviewScroll)
	}
}

// TestFileExplorerDepth1HorizontalScroll verifies h/l scroll horizontally at depth 1.
func TestFileExplorerDepth1HorizontalScroll(t *testing.T) {
	m := newFileExplorerTestModel(t.TempDir())
	setupFileExplorerTree(m)
	m.fileExplorerDepth = 1

	m.handleFileExplorerKey(config.ActionFocusRight)
	if m.filePreviewHScroll != 1 {
		t.Errorf("expected hscroll 1 after l at depth 1, got %d", m.filePreviewHScroll)
	}

	m.handleFileExplorerKey(config.ActionFocusLeft)
	if m.filePreviewHScroll != 0 {
		t.Errorf("expected hscroll 0 after h at depth 1, got %d", m.filePreviewHScroll)
	}

	// h at 0 stays at 0
	m.handleFileExplorerKey(config.ActionFocusLeft)
	if m.filePreviewHScroll != 0 {
		t.Errorf("expected hscroll 0 (no negative), got %d", m.filePreviewHScroll)
	}
}

// TestFileExplorerQuitAndHelp verifies q and ? work in file explorer.
func TestFileExplorerQuitAndHelp(t *testing.T) {
	m := newFileExplorerTestModel(t.TempDir())
	setupFileExplorerTree(m)

	m.handleFileExplorerKey(config.ActionQuit)
	if m.activeModal != modalQuitConfirm {
		t.Error("q should show quit confirmation")
	}
	m.activeModal = modalNone

	m.handleFileExplorerKey(config.ActionHelp)
	if m.activeModal != modalHelp {
		t.Error("? should show help modal")
	}
}

// TestFileExplorerToggleLeftPane verifies [ key toggles left pane.
func TestFileExplorerToggleLeftPane(t *testing.T) {
	m := newFileExplorerTestModel(t.TempDir())
	setupFileExplorerTree(m)

	if !m.leftPaneVisible {
		t.Fatal("left pane should be visible initially")
	}

	m.handleFileExplorerKey(config.ActionToggleLeftPane)
	if m.leftPaneVisible {
		t.Error("left pane should be hidden after toggle")
	}

	m.handleFileExplorerKey(config.ActionToggleLeftPane)
	if !m.leftPaneVisible {
		t.Error("left pane should be visible after second toggle")
	}
}

// TestFileExplorerToggleLineNumbers verifies # toggles line numbers.
func TestFileExplorerToggleLineNumbers(t *testing.T) {
	m := newFileExplorerTestModel(t.TempDir())
	setupFileExplorerTree(m)

	initial := m.lineNumbers
	m.handleFileExplorerKey(config.ActionToggleLineNumbers)
	if m.lineNumbers == initial {
		t.Error("# should toggle line numbers")
	}
}

// TestFileExplorerMergePreservesState verifies tree refresh preserves expand/cursor state.
func TestFileExplorerMergePreservesState(t *testing.T) {
	m := newFileExplorerTestModel(t.TempDir())
	setupFileExplorerTree(m)

	// Expand internal/tui/ and select root.go
	rows := m.fileExplorerTree.VisibleRows()
	// Find "tui" node and expand it
	for i, row := range rows {
		if row.Node.Name == "tui" {
			m.fileExplorerTree.Cursor = i
			row.Node.Expanded = true
			break
		}
	}

	// Rebuild rows after expansion
	m.fileExplorerTree.SetRoots(m.fileExplorerTree.Roots())

	// Move cursor to root.go
	rows = m.fileExplorerTree.VisibleRows()
	for i, row := range rows {
		if row.Node.Name == "root.go" {
			m.fileExplorerTree.Cursor = i
			break
		}
	}

	selectedPath := m.fileExplorerTree.SelectedNode().Path

	// Simulate a refresh with new tree data (same structure)
	newRoots := []*FileNode{
		{Name: "cmd", Path: "cmd", IsDir: true, Children: []*FileNode{
			{Name: "main.go", Path: "cmd/main.go"},
		}},
		{Name: "internal", Path: "internal", IsDir: true, Children: []*FileNode{
			{Name: "tui", Path: "internal/tui", IsDir: true, Children: []*FileNode{
				{Name: "root.go", Path: "internal/tui/root.go"},
				{Name: "view.go", Path: "internal/tui/view.go"},
			}},
		}},
		{Name: "go.mod", Path: "go.mod"},
		{Name: "README.md", Path: "README.md"},
	}

	m.mergeFileExplorerTree(newRoots, "")

	// Verify tui/ is still expanded
	tuiNode := findNodeByPath(m.fileExplorerTree.Roots(), "internal/tui")
	if tuiNode == nil {
		t.Fatal("expected to find internal/tui node")
	}
	if !tuiNode.Expanded {
		t.Error("internal/tui should still be expanded after merge")
	}

	// Verify cursor is still on root.go
	selectedNode := m.fileExplorerTree.SelectedNode()
	if selectedNode == nil || selectedNode.Path != selectedPath {
		var got string
		if selectedNode != nil {
			got = selectedNode.Path
		}
		t.Errorf("expected cursor on %q, got %q", selectedPath, got)
	}
}

// TestFileExplorerMergeWithGitStatus verifies refresh applies git status.
func TestFileExplorerMergeWithGitStatus(t *testing.T) {
	m := newFileExplorerTestModel(t.TempDir())
	setupFileExplorerTree(m)

	newRoots := []*FileNode{
		{Name: "go.mod", Path: "go.mod"},
	}

	m.mergeFileExplorerTree(newRoots, " M go.mod\n")

	node := m.fileExplorerTree.SelectedNode()
	if node == nil {
		t.Fatal("expected a selected node")
	}
	if node.Status != GitStatusModified {
		t.Errorf("expected Modified status, got %q", node.Status)
	}
}

// TestFileExplorerRender verifies the render function produces output.
func TestFileExplorerRender(t *testing.T) {
	dir := t.TempDir()
	// Create a test file
	if err := os.WriteFile(filepath.Join(dir, "hello.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	m := newFileExplorerTestModel(dir)
	m.fileExplorerActive = true
	m.fileExplorerDepth = 0
	m.fileExplorerTree = NewFileTreeView(BuildFileTree(dir))

	result := m.renderFileExplorer()

	if result == "" {
		t.Error("expected non-empty render output")
	}
	if len(result) < 10 {
		t.Error("render output too short")
	}
}

// TestFileExplorerRenderWithoutLeftPane verifies render works with hidden left pane.
func TestFileExplorerRenderWithoutLeftPane(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test.go"), []byte("package test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	m := newFileExplorerTestModel(dir)
	m.fileExplorerActive = true
	m.fileExplorerDepth = 0
	m.fileExplorerTree = NewFileTreeView(BuildFileTree(dir))
	m.leftPaneVisible = false

	result := m.renderFileExplorer()

	if result == "" {
		t.Error("expected non-empty render output even without left pane")
	}
}

// TestFileExplorerFKeyToggle verifies f key enters and exits file explorer.
func TestFileExplorerFKeyToggle(t *testing.T) {
	m := newFileExplorerTestModel(t.TempDir())
	setupFileExplorerTree(m)

	if !m.fileExplorerActive {
		t.Fatal("file explorer should be active")
	}

	// f again should exit
	m.handleFileExplorerKey(config.ActionFileExplorer)

	if m.fileExplorerActive {
		t.Error("f should exit file explorer when already active")
	}
}

// TestFileExplorerJumpTopBottom verifies gg and G at depth 0.
func TestFileExplorerJumpTopBottom(t *testing.T) {
	m := newFileExplorerTestModel(t.TempDir())
	setupFileExplorerTree(m)

	m.handleFileExplorerKey(config.ActionJumpBottom)
	rows := m.fileExplorerTree.VisibleRows()
	if m.fileExplorerTree.Cursor != len(rows)-1 {
		t.Errorf("expected cursor at bottom (%d), got %d", len(rows)-1, m.fileExplorerTree.Cursor)
	}

	m.handleFileExplorerKey(config.ActionJumpTop)
	if m.fileExplorerTree.Cursor != 0 {
		t.Errorf("expected cursor at top (0), got %d", m.fileExplorerTree.Cursor)
	}
}

// TestFileExplorerDepth1JumpTopBottom verifies gg and G at depth 1.
func TestFileExplorerDepth1JumpTopBottom(t *testing.T) {
	m := newFileExplorerTestModel(t.TempDir())
	setupFileExplorerTree(m)
	m.fileExplorerDepth = 1
	m.filePreviewScroll = 50

	m.handleFileExplorerKey(config.ActionJumpTop)
	if m.filePreviewScroll != 0 {
		t.Errorf("expected scroll 0 after gg at depth 1, got %d", m.filePreviewScroll)
	}

	m.handleFileExplorerKey(config.ActionJumpBottom)
	if m.filePreviewScroll != 999999 {
		t.Errorf("expected large scroll after G at depth 1, got %d", m.filePreviewScroll)
	}
}

// TestFileExplorerKeyRouting verifies keys are routed to file explorer when active.
func TestFileExplorerKeyRouting(t *testing.T) {
	m := newFileExplorerTestModel(t.TempDir())
	setupFileExplorerTree(m)

	// Verify key routing goes through handleFileExplorerKey
	initialCursor := m.fileExplorerTree.Cursor
	m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if m.fileExplorerTree.Cursor == initialCursor {
		t.Error("j key should move cursor in file explorer mode")
	}
}

// TestCollectExpandedPaths verifies expanded path collection.
func TestCollectExpandedPaths(t *testing.T) {
	nodes := []*FileNode{
		{Name: "a", Path: "a", IsDir: true, Expanded: true, Children: []*FileNode{
			{Name: "b", Path: "a/b", IsDir: true, Expanded: true},
			{Name: "c", Path: "a/c", IsDir: true, Expanded: false},
		}},
		{Name: "d", Path: "d", IsDir: true, Expanded: false},
	}

	paths := collectExpandedPaths(nodes)

	if !paths["a"] {
		t.Error("expected 'a' in expanded paths")
	}
	if !paths["a/b"] {
		t.Error("expected 'a/b' in expanded paths")
	}
	if paths["a/c"] {
		t.Error("'a/c' should not be in expanded paths")
	}
	if paths["d"] {
		t.Error("'d' should not be in expanded paths")
	}
}

// TestRestoreExpandedPaths verifies expanded state restoration.
func TestRestoreExpandedPaths(t *testing.T) {
	nodes := []*FileNode{
		{Name: "a", Path: "a", IsDir: true, Children: []*FileNode{
			{Name: "b", Path: "a/b", IsDir: true},
			{Name: "c", Path: "a/c", IsDir: true},
		}},
		{Name: "d", Path: "d", IsDir: true},
	}

	paths := map[string]bool{"a": true, "a/b": true}
	restoreExpandedPaths(nodes, paths)

	if !nodes[0].Expanded {
		t.Error("'a' should be expanded")
	}
	if !nodes[0].Children[0].Expanded {
		t.Error("'a/b' should be expanded")
	}
	if nodes[0].Children[1].Expanded {
		t.Error("'a/c' should not be expanded")
	}
	if nodes[1].Expanded {
		t.Error("'d' should not be expanded")
	}
}
