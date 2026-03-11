package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GitStatus represents a file's git status indicator.
type GitStatus string

const (
	GitStatusNone      GitStatus = ""
	GitStatusModified  GitStatus = "M"
	GitStatusAdded     GitStatus = "A"
	GitStatusDeleted   GitStatus = "D"
	GitStatusUntracked GitStatus = "?"
)

// gitStatusPriority returns priority for inheritance (higher = more important).
func gitStatusPriority(s GitStatus) int {
	switch s {
	case GitStatusDeleted:
		return 4
	case GitStatusModified:
		return 3
	case GitStatusAdded:
		return 2
	case GitStatusUntracked:
		return 1
	default:
		return 0
	}
}

// FileNode represents a file or directory in the tree.
type FileNode struct {
	Name          string
	Path          string // relative path from tree root
	IsDir         bool
	IsSymlink     bool
	SymlinkTarget string // display target for symlinks
	Status        GitStatus
	Children      []*FileNode
	Expanded      bool
}

// FlatRow is a visible row in the flattened tree, with depth for indentation.
type FlatRow struct {
	Node  *FileNode
	Depth int
}

// BuildFileTree walks root recursively and returns the top-level children.
// Skips .git/ directories. Follows symlinks for type detection.
func BuildFileTree(root string) []*FileNode {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}

	var nodes []*FileNode
	for _, e := range entries {
		name := e.Name()
		if name == ".git" {
			continue
		}

		fullPath := filepath.Join(root, name)
		relPath, _ := filepath.Rel(root, fullPath)

		node := &FileNode{
			Name: name,
			Path: relPath,
		}

		// Check if entry is a symlink
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			node.IsSymlink = true
			target, err := os.Readlink(fullPath)
			if err == nil {
				node.SymlinkTarget = target
			}
			// Follow symlink to determine if it's a dir
			resolved, err := os.Stat(fullPath)
			if err == nil && resolved.IsDir() {
				node.IsDir = true
			}
		} else if e.IsDir() {
			node.IsDir = true
		}

		if node.IsDir {
			node.Children = buildFileTreeRecursive(fullPath, root)
		}

		nodes = append(nodes, node)
	}

	sortNodes(nodes)
	return nodes
}

// buildFileTreeRecursive walks a subdirectory.
func buildFileTreeRecursive(dir, root string) []*FileNode {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var nodes []*FileNode
	for _, e := range entries {
		name := e.Name()
		if name == ".git" {
			continue
		}

		fullPath := filepath.Join(dir, name)
		relPath, _ := filepath.Rel(root, fullPath)

		node := &FileNode{
			Name: name,
			Path: relPath,
		}

		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			node.IsSymlink = true
			target, err := os.Readlink(fullPath)
			if err == nil {
				node.SymlinkTarget = target
			}
			resolved, err := os.Stat(fullPath)
			if err == nil && resolved.IsDir() {
				node.IsDir = true
			}
		} else if e.IsDir() {
			node.IsDir = true
		}

		if node.IsDir {
			node.Children = buildFileTreeRecursive(fullPath, root)
		}

		nodes = append(nodes, node)
	}

	sortNodes(nodes)
	return nodes
}

// sortNodes sorts dirs first (case-insensitive), then files (case-insensitive).
func sortNodes(nodes []*FileNode) {
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].IsDir != nodes[j].IsDir {
			return nodes[i].IsDir // dirs first
		}
		return strings.ToLower(nodes[i].Name) < strings.ToLower(nodes[j].Name)
	})
}

// FlattenVisible returns the visible rows from the tree, respecting expanded state.
func FlattenVisible(roots []*FileNode) []FlatRow {
	var rows []FlatRow
	for _, node := range roots {
		flattenNode(node, 0, &rows)
	}
	return rows
}

func flattenNode(node *FileNode, depth int, rows *[]FlatRow) {
	*rows = append(*rows, FlatRow{Node: node, Depth: depth})
	if node.IsDir && node.Expanded {
		for _, child := range node.Children {
			flattenNode(child, depth+1, rows)
		}
	}
}

// IsBinary checks if a file is binary by reading the first 512 bytes
// and checking for null bytes.
func IsBinary(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil || n == 0 {
		return false
	}

	for _, b := range buf[:n] {
		if b == 0 {
			return true
		}
	}
	return false
}

// ApplyGitStatus parses `git status --porcelain` output and applies status
// indicators to matching nodes in the tree. Directories inherit the
// highest-priority status from their children.
func ApplyGitStatus(roots []*FileNode, porcelainOutput string) {
	statusMap := parsePorcelain(porcelainOutput)
	for _, node := range roots {
		applyStatusRecursive(node, statusMap)
	}
}

// parsePorcelain parses `git status --porcelain` output into a map of
// relative path → GitStatus.
func parsePorcelain(output string) map[string]GitStatus {
	m := make(map[string]GitStatus)
	for _, line := range strings.Split(output, "\n") {
		if len(line) < 4 {
			continue
		}
		// Porcelain format: XY PATH
		// X = index status, Y = working tree status
		xy := line[:2]
		path := strings.TrimSpace(line[3:])
		// Remove trailing slash for directories
		path = strings.TrimRight(path, "/")

		var status GitStatus
		switch {
		case xy == "??":
			status = GitStatusUntracked
		case xy[0] == 'D' || xy[1] == 'D':
			status = GitStatusDeleted
		case xy[0] == 'M' || xy[1] == 'M':
			status = GitStatusModified
		case xy[0] == 'A' || xy[1] == 'A':
			status = GitStatusAdded
		default:
			// Other statuses (rename, copy, etc.) treated as modified
			status = GitStatusModified
		}
		m[path] = status
	}
	return m
}

// applyStatusRecursive sets git status on nodes and computes directory inheritance.
func applyStatusRecursive(node *FileNode, statusMap map[string]GitStatus) {
	if !node.IsDir {
		if s, ok := statusMap[node.Path]; ok {
			node.Status = s
		}
		return
	}

	// Process children first
	var maxPriority int
	var inheritedStatus GitStatus
	for _, child := range node.Children {
		applyStatusRecursive(child, statusMap)
		p := gitStatusPriority(child.Status)
		if p > maxPriority {
			maxPriority = p
			inheritedStatus = child.Status
		}
	}
	node.Status = inheritedStatus
}

// FindParent returns the parent FileNode for a given node path, or nil if root-level.
func FindParent(roots []*FileNode, targetPath string) *FileNode {
	parentDir := filepath.Dir(targetPath)
	if parentDir == "." {
		return nil // root-level node
	}
	return findNodeByPath(roots, parentDir)
}

// findNodeByPath finds a node by its relative path.
func findNodeByPath(roots []*FileNode, path string) *FileNode {
	for _, node := range roots {
		if node.Path == path {
			return node
		}
		if node.IsDir {
			if found := findNodeByPath(node.Children, path); found != nil {
				return found
			}
		}
	}
	return nil
}
