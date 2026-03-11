package tui

import (
	"os"
	"path/filepath"
	"testing"
)

// createTestTree sets up a temp directory with a known file structure for testing.
// Returns the root path. Structure:
//
//	root/
//	  .git/          (should be skipped)
//	    config
//	  cmd/
//	    main.go
//	  internal/
//	    app.go
//	    util.go
//	  go.mod
//	  README.md
//	  Makefile
func createTestTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	dirs := []string{
		".git",
		"cmd",
		"internal",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	files := map[string]string{
		".git/config":      "[core]",
		"cmd/main.go":      "package main",
		"internal/app.go":  "package internal",
		"internal/util.go": "package internal",
		"go.mod":           "module test",
		"README.md":        "# Test",
		"Makefile":         "all:",
	}
	for path, content := range files {
		if err := os.WriteFile(filepath.Join(root, path), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

func TestBuildFileTree_SkipsGitDir(t *testing.T) {
	root := createTestTree(t)
	nodes := BuildFileTree(root)

	for _, n := range nodes {
		if n.Name == ".git" {
			t.Error("BuildFileTree should skip .git directory")
		}
	}
}

func TestBuildFileTree_SortOrder(t *testing.T) {
	root := createTestTree(t)
	nodes := BuildFileTree(root)

	// Dirs should come first, then files, each sorted case-insensitively
	var names []string
	for _, n := range nodes {
		names = append(names, n.Name)
	}

	// Expected: dirs first (cmd, internal), then files (go.mod, Makefile, README.md)
	// Dirs: cmd, internal (alphabetical)
	// Files: go.mod, Makefile, README.md (case-insensitive: go.mod < makefile < readme.md)
	expected := []string{"cmd", "internal", "go.mod", "Makefile", "README.md"}
	if len(names) != len(expected) {
		t.Fatalf("expected %d nodes, got %d: %v", len(expected), len(names), names)
	}
	for i, name := range expected {
		if names[i] != name {
			t.Errorf("position %d: expected %q, got %q (full: %v)", i, name, names[i], names)
		}
	}
}

func TestBuildFileTree_DirChildren(t *testing.T) {
	root := createTestTree(t)
	nodes := BuildFileTree(root)

	// Find internal/ dir
	var internalNode *FileNode
	for _, n := range nodes {
		if n.Name == "internal" {
			internalNode = n
			break
		}
	}
	if internalNode == nil {
		t.Fatal("internal/ directory not found")
	}
	if !internalNode.IsDir {
		t.Error("internal/ should be marked as dir")
	}
	if len(internalNode.Children) != 2 {
		t.Fatalf("internal/ should have 2 children, got %d", len(internalNode.Children))
	}

	// Children should be sorted: app.go, util.go
	if internalNode.Children[0].Name != "app.go" {
		t.Errorf("expected first child app.go, got %s", internalNode.Children[0].Name)
	}
	if internalNode.Children[1].Name != "util.go" {
		t.Errorf("expected second child util.go, got %s", internalNode.Children[1].Name)
	}
}

func TestBuildFileTree_RelativePaths(t *testing.T) {
	root := createTestTree(t)
	nodes := BuildFileTree(root)

	// Top-level files should have simple names as paths
	for _, n := range nodes {
		if !n.IsDir && n.Path != n.Name {
			t.Errorf("top-level file %q should have path=%q, got path=%q", n.Name, n.Name, n.Path)
		}
	}

	// Nested files should have relative paths
	var cmdNode *FileNode
	for _, n := range nodes {
		if n.Name == "cmd" {
			cmdNode = n
			break
		}
	}
	if cmdNode == nil {
		t.Fatal("cmd/ not found")
	}
	if len(cmdNode.Children) == 0 {
		t.Fatal("cmd/ should have children")
	}
	mainGo := cmdNode.Children[0]
	if mainGo.Path != filepath.Join("cmd", "main.go") {
		t.Errorf("expected path cmd/main.go, got %s", mainGo.Path)
	}
}

func TestFlattenVisible_AllCollapsed(t *testing.T) {
	root := createTestTree(t)
	nodes := BuildFileTree(root)

	// All dirs collapsed by default
	rows := FlattenVisible(nodes)

	// Should only show top-level: cmd/, internal/, go.mod, Makefile, README.md
	if len(rows) != 5 {
		t.Fatalf("expected 5 rows when all collapsed, got %d", len(rows))
	}
	for _, r := range rows {
		if r.Depth != 0 {
			t.Errorf("all rows should be depth 0 when collapsed, got depth %d for %s", r.Depth, r.Node.Name)
		}
	}
}

func TestFlattenVisible_Expanded(t *testing.T) {
	root := createTestTree(t)
	nodes := BuildFileTree(root)

	// Expand internal/
	for _, n := range nodes {
		if n.Name == "internal" {
			n.Expanded = true
		}
	}

	rows := FlattenVisible(nodes)

	// Should show: cmd/, internal/, internal/app.go, internal/util.go, go.mod, Makefile, README.md
	if len(rows) != 7 {
		t.Fatalf("expected 7 rows, got %d", len(rows))
	}

	// Verify internal's children are at depth 1
	if rows[2].Node.Name != "app.go" || rows[2].Depth != 1 {
		t.Errorf("expected app.go at depth 1, got %s at depth %d", rows[2].Node.Name, rows[2].Depth)
	}
	if rows[3].Node.Name != "util.go" || rows[3].Depth != 1 {
		t.Errorf("expected util.go at depth 1, got %s at depth %d", rows[3].Node.Name, rows[3].Depth)
	}
}

func TestFlattenVisible_NestedExpand(t *testing.T) {
	// Create a deeper structure
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "a", "b"), 0o755)
	_ = os.WriteFile(filepath.Join(root, "a", "b", "c.go"), []byte("pkg"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "a", "x.go"), []byte("pkg"), 0o644)

	nodes := BuildFileTree(root)

	// Expand a/ and a/b/
	nodes[0].Expanded = true             // a/
	nodes[0].Children[0].Expanded = true // a/b/

	rows := FlattenVisible(nodes)

	// a/ (depth 0), b/ (depth 1), c.go (depth 2), x.go (depth 1)
	expected := []struct {
		name  string
		depth int
	}{
		{"a", 0},
		{"b", 1},
		{"c.go", 2},
		{"x.go", 1},
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

func TestIsBinary(t *testing.T) {
	dir := t.TempDir()

	// Text file
	textPath := filepath.Join(dir, "text.go")
	_ = os.WriteFile(textPath, []byte("package main\nfunc main() {}\n"), 0o644)
	if IsBinary(textPath) {
		t.Error("text file should not be detected as binary")
	}

	// Binary file (contains null bytes)
	binPath := filepath.Join(dir, "binary.dat")
	_ = os.WriteFile(binPath, []byte{0x89, 0x50, 0x4E, 0x47, 0x00, 0x00, 0x01}, 0o644)
	if !IsBinary(binPath) {
		t.Error("binary file should be detected as binary")
	}

	// Empty file
	emptyPath := filepath.Join(dir, "empty")
	_ = os.WriteFile(emptyPath, []byte{}, 0o644)
	if IsBinary(emptyPath) {
		t.Error("empty file should not be detected as binary")
	}

	// Non-existent file
	if IsBinary(filepath.Join(dir, "nonexistent")) {
		t.Error("non-existent file should not be detected as binary")
	}
}

func TestSymlinkDetection(t *testing.T) {
	root := t.TempDir()

	// Create a real file and a symlink to it
	realFile := filepath.Join(root, "real.txt")
	_ = os.WriteFile(realFile, []byte("hello"), 0o644)
	symlinkFile := filepath.Join(root, "link.txt")
	if err := os.Symlink(realFile, symlinkFile); err != nil {
		t.Skip("symlinks not supported on this platform")
	}

	nodes := BuildFileTree(root)
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}

	// Find the symlink node (sorted: link.txt, real.txt)
	var linkNode *FileNode
	for _, n := range nodes {
		if n.Name == "link.txt" {
			linkNode = n
		}
	}
	if linkNode == nil {
		t.Fatal("link.txt not found")
	}
	if !linkNode.IsSymlink {
		t.Error("link.txt should be detected as symlink")
	}
	if linkNode.SymlinkTarget != realFile {
		t.Errorf("symlink target should be %q, got %q", realFile, linkNode.SymlinkTarget)
	}
	if linkNode.IsDir {
		t.Error("symlink to file should not be marked as dir")
	}
}

func TestSymlinkToDir(t *testing.T) {
	root := t.TempDir()

	// Create a real dir and a symlink to it
	realDir := filepath.Join(root, "realdir")
	_ = os.Mkdir(realDir, 0o755)
	_ = os.WriteFile(filepath.Join(realDir, "file.go"), []byte("pkg"), 0o644)
	symlinkDir := filepath.Join(root, "linkdir")
	if err := os.Symlink(realDir, symlinkDir); err != nil {
		t.Skip("symlinks not supported on this platform")
	}

	nodes := BuildFileTree(root)

	var linkNode *FileNode
	for _, n := range nodes {
		if n.Name == "linkdir" {
			linkNode = n
		}
	}
	if linkNode == nil {
		t.Fatal("linkdir not found")
	}
	if !linkNode.IsSymlink {
		t.Error("linkdir should be detected as symlink")
	}
	if !linkNode.IsDir {
		t.Error("symlink to directory should be marked as dir")
	}
}

// --- Git status tests ---

func TestParsePorcelain(t *testing.T) {
	output := ` M go.mod
?? new_file.go
A  added.go
 D deleted.go
MM both.go
`
	m := parsePorcelain(output)

	tests := []struct {
		path   string
		status GitStatus
	}{
		{"go.mod", GitStatusModified},
		{"new_file.go", GitStatusUntracked},
		{"added.go", GitStatusAdded},
		{"deleted.go", GitStatusDeleted},
		{"both.go", GitStatusModified},
	}
	for _, tt := range tests {
		got, ok := m[tt.path]
		if !ok {
			t.Errorf("path %q not found in status map", tt.path)
			continue
		}
		if got != tt.status {
			t.Errorf("path %q: expected status %q, got %q", tt.path, tt.status, got)
		}
	}
}

func TestApplyGitStatus_Files(t *testing.T) {
	root := createTestTree(t)
	nodes := BuildFileTree(root)

	porcelain := ` M go.mod
?? cmd/main.go
`
	ApplyGitStatus(nodes, porcelain)

	// Find go.mod
	for _, n := range nodes {
		if n.Name == "go.mod" {
			if n.Status != GitStatusModified {
				t.Errorf("go.mod should be Modified, got %q", n.Status)
			}
		}
	}

	// Find cmd/main.go
	for _, n := range nodes {
		if n.Name == "cmd" {
			if len(n.Children) == 0 {
				t.Fatal("cmd/ should have children")
			}
			child := n.Children[0]
			if child.Status != GitStatusUntracked {
				t.Errorf("cmd/main.go should be Untracked, got %q", child.Status)
			}
		}
	}
}

func TestApplyGitStatus_DirectoryInheritance(t *testing.T) {
	root := createTestTree(t)
	nodes := BuildFileTree(root)

	// internal/ has two files, one modified and one deleted
	// D should win (higher priority)
	porcelain := ` M internal/app.go
 D internal/util.go
`
	ApplyGitStatus(nodes, porcelain)

	for _, n := range nodes {
		if n.Name == "internal" {
			if n.Status != GitStatusDeleted {
				t.Errorf("internal/ should inherit Deleted (highest priority), got %q", n.Status)
			}
		}
	}
}

func TestApplyGitStatus_DirectoryInheritancePriority(t *testing.T) {
	// Test all priorities: D > M > A > ?
	root := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "dir"), 0o755)
	_ = os.WriteFile(filepath.Join(root, "dir", "a.go"), []byte("a"), 0o644)
	_ = os.WriteFile(filepath.Join(root, "dir", "b.go"), []byte("b"), 0o644)

	nodes := BuildFileTree(root)

	// A and ? → A wins
	ApplyGitStatus(nodes, "A  dir/a.go\n?? dir/b.go\n")
	if nodes[0].Status != GitStatusAdded {
		t.Errorf("A > ?: expected Added, got %q", nodes[0].Status)
	}

	// Reset
	nodes[0].Children[0].Status = GitStatusNone
	nodes[0].Children[1].Status = GitStatusNone
	nodes[0].Status = GitStatusNone

	// M and A → M wins
	ApplyGitStatus(nodes, " M dir/a.go\nA  dir/b.go\n")
	if nodes[0].Status != GitStatusModified {
		t.Errorf("M > A: expected Modified, got %q", nodes[0].Status)
	}
}

func TestFindParent(t *testing.T) {
	root := createTestTree(t)
	nodes := BuildFileTree(root)

	// Root-level file has no parent
	parent := FindParent(nodes, "go.mod")
	if parent != nil {
		t.Error("root-level file should have nil parent")
	}

	// Nested file should find parent dir
	parent = FindParent(nodes, filepath.Join("cmd", "main.go"))
	if parent == nil {
		t.Fatal("cmd/main.go should have parent")
	}
	if parent.Name != "cmd" {
		t.Errorf("parent should be cmd, got %s", parent.Name)
	}
}

func TestBuildFileTree_EmptyDir(t *testing.T) {
	root := t.TempDir()
	nodes := BuildFileTree(root)
	if len(nodes) != 0 {
		t.Errorf("empty dir should produce 0 nodes, got %d", len(nodes))
	}
}

func TestBuildFileTree_OnlyGitDir(t *testing.T) {
	root := t.TempDir()
	_ = os.Mkdir(filepath.Join(root, ".git"), 0o755)
	nodes := BuildFileTree(root)
	if len(nodes) != 0 {
		t.Errorf("dir with only .git should produce 0 nodes, got %d", len(nodes))
	}
}
