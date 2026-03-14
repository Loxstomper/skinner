package tui

import (
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/loxstomper/skinner/internal/config"
)

const fileExplorerRefreshInterval = 5 * time.Second

type fileExplorerEditorDoneMsg struct{ err error }

// enterFileExplorer activates the file explorer, building the tree from CWD.
// Returns a tea.Cmd to start the 5-second refresh timer.
func (m *Model) enterFileExplorer() tea.Cmd {
	roots := BuildFileTree(m.workDir)

	// Apply git status
	porcelain, err := gitStatusPorcelain(m.workDir)
	if err == nil {
		ApplyGitStatus(roots, porcelain)
	}

	m.fileExplorerActive = true
	m.fileExplorerDepth = 0
	m.fileExplorerTree = NewFileTreeView(roots)
	m.filePreviewScroll = 0
	m.filePreviewHScroll = 0

	return fileExplorerTickCmd()
}

// exitFileExplorer deactivates the file explorer, restoring the previous view.
func (m *Model) exitFileExplorer() {
	m.fileExplorerActive = false
	m.fileExplorerDepth = 0
	m.fileExplorerTree = nil
	m.filePreviewScroll = 0
	m.filePreviewHScroll = 0
}

// handleFileExplorerKey routes key actions when the file explorer is active.
func (m *Model) handleFileExplorerKey(action string) (tea.Model, tea.Cmd) {
	switch action {
	case config.ActionQuit:
		m.activeModal = modalQuitConfirm
		return m, nil

	case config.ActionHelp:
		m.activeModal = modalHelp
		m.helpModalScroll = 0
		return m, nil

	case config.ActionFileExplorer:
		// f while in file explorer: exit
		m.exitFileExplorer()
		return m, nil

	case config.ActionEscape:
		switch m.fileExplorerDepth {
		case 0:
			m.exitFileExplorer()
		case 1:
			// Back to tree navigation
			m.fileExplorerDepth = 0
			m.filePreviewScroll = 0
			m.filePreviewHScroll = 0
		}
		return m, nil

	case config.ActionToggleLeftPane:
		// [ key still works in file explorer for hiding/showing left pane
		if m.effectiveLayout() == "bottom" {
			m.bottomBarVisible = !m.bottomBarVisible
		} else {
			m.leftPaneVisible = !m.leftPaneVisible
		}
		return m, nil

	case config.ActionToggleLineNumbers:
		m.lineNumbers = !m.lineNumbers
		return m, nil
	}

	if m.fileExplorerDepth == 0 {
		return m.handleFileExplorerDepth0(action)
	}
	return m.handleFileExplorerDepth1(action)
}

// handleFileExplorerSearchKey handles raw key input during fuzzy search mode.
// Returns true if the key was consumed by search.
func (m *Model) handleFileExplorerSearchKey(key string) bool {
	if m.fileExplorerTree == nil || !m.fileExplorerTree.IsSearching() {
		return false
	}

	result := m.fileExplorerTree.HandleSearchKey(key)
	switch result {
	case "confirm":
		node := m.fileExplorerTree.ConfirmSearch()
		if node != nil {
			m.filePreviewScroll = 0
			m.filePreviewHScroll = 0
		}
	case "cancel":
		m.fileExplorerTree.CancelSearch()
	}

	return true
}

// handleFileExplorerDepth0 handles tree navigation at depth 0.
func (m *Model) handleFileExplorerDepth0(action string) (tea.Model, tea.Cmd) {
	props := m.fileTreeViewProps()

	switch action {
	case config.ActionMoveDown:
		m.fileExplorerTree.HandleAction("move_down", props)
		m.filePreviewScroll = 0
		m.filePreviewHScroll = 0

	case config.ActionMoveUp:
		m.fileExplorerTree.HandleAction("move_up", props)
		m.filePreviewScroll = 0
		m.filePreviewHScroll = 0

	case config.ActionJumpTop:
		m.fileExplorerTree.HandleAction("jump_top", props)
		m.filePreviewScroll = 0
		m.filePreviewHScroll = 0

	case config.ActionJumpBottom:
		m.fileExplorerTree.HandleAction("jump_bottom", props)
		m.filePreviewScroll = 0
		m.filePreviewHScroll = 0

	case "page_down":
		m.fileExplorerTree.HandleAction("page_down", props)
		m.filePreviewScroll = 0
		m.filePreviewHScroll = 0

	case "page_up":
		m.fileExplorerTree.HandleAction("page_up", props)
		m.filePreviewScroll = 0
		m.filePreviewHScroll = 0

	case config.ActionExpand:
		// enter: toggle dir or enter depth 1
		if m.fileExplorerTree.HandleAction("expand", props) {
			m.fileExplorerDepth = 1
			m.filePreviewScroll = 0
			m.filePreviewHScroll = 0
		}

	case config.ActionFocusLeft:
		// h: tree-specific left navigation
		m.fileExplorerTree.HandleAction("focus_left", props)

	case config.ActionFocusRight:
		// l: tree-specific right navigation, may signal depth 2
		if m.fileExplorerTree.HandleAction("focus_right", props) {
			m.fileExplorerDepth = 1
			m.filePreviewScroll = 0
			m.filePreviewHScroll = 0
		}

	case config.ActionEditPlan:
		// e: open editor for selected file
		return m, m.launchFileExplorerEditor()

	case "search":
		// / activates fuzzy search in the file tree
		if m.fileExplorerTree != nil {
			m.fileExplorerTree.EnterSearch()
		}
	}

	return m, nil
}

// handleFileExplorerDepth1 handles scrollable preview at depth 1.
func (m *Model) handleFileExplorerDepth1(action string) (tea.Model, tea.Cmd) {
	switch action {
	case config.ActionMoveDown:
		m.filePreviewScroll++

	case config.ActionMoveUp:
		if m.filePreviewScroll > 0 {
			m.filePreviewScroll--
		}

	case config.ActionFocusLeft:
		// h: scroll left
		if m.filePreviewHScroll > 0 {
			m.filePreviewHScroll--
		}

	case config.ActionFocusRight:
		// l: scroll right
		m.filePreviewHScroll++

	case config.ActionJumpTop:
		m.filePreviewScroll = 0

	case config.ActionJumpBottom:
		// Set to a large value — will be clamped during rendering
		m.filePreviewScroll = 999999

	case "page_down":
		pageSize := m.rightPaneHeight() - 2
		if pageSize < 1 {
			pageSize = 1
		}
		m.filePreviewScroll += pageSize

	case "page_up":
		pageSize := m.rightPaneHeight() - 2
		if pageSize < 1 {
			pageSize = 1
		}
		m.filePreviewScroll -= pageSize
		if m.filePreviewScroll < 0 {
			m.filePreviewScroll = 0
		}

	case config.ActionEditPlan:
		// e: open editor
		return m, m.launchFileExplorerEditor()
	}

	return m, nil
}

// launchFileExplorerEditor opens $EDITOR for the currently selected file.
func (m *Model) launchFileExplorerEditor() tea.Cmd {
	if m.fileExplorerTree == nil {
		return nil
	}
	node := m.fileExplorerTree.SelectedNode()
	if node == nil || node.IsDir {
		return nil
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	filePath := m.workDir + "/" + node.Path
	return tea.ExecProcess(exec.Command(editor, filePath), func(err error) tea.Msg {
		return fileExplorerEditorDoneMsg{err: err}
	})
}

// fileTreeViewProps builds FileTreeViewProps from current model state.
func (m *Model) fileTreeViewProps() FileTreeViewProps {
	return FileTreeViewProps{
		Width:   m.fileExplorerLeftWidth(),
		Height:  m.rightPaneHeight(),
		Focused: m.fileExplorerDepth == 0,
		Theme:   m.theme,
	}
}

// fileExplorerLeftWidth returns the left pane width for the file explorer.
func (m *Model) fileExplorerLeftWidth() int {
	if !m.leftPaneVisible {
		return 0
	}
	return leftPaneFixedWidth
}

// fileExplorerTickCmd starts the 5-second refresh timer for the file explorer.
func fileExplorerTickCmd() tea.Cmd {
	return tea.Tick(fileExplorerRefreshInterval, func(t time.Time) tea.Msg {
		return fileExplorerTickMsg{}
	})
}

// fileExplorerRefreshCmd re-walks the filesystem and runs git status in the background.
func fileExplorerRefreshCmd(workDir string) tea.Cmd {
	return func() tea.Msg {
		roots := BuildFileTree(workDir)
		porcelain, _ := gitStatusPorcelain(workDir)
		return fileExplorerRefreshMsg{roots: roots, porcelainOutput: porcelain}
	}
}

// gitStatusPorcelain runs `git status --porcelain` and returns the output.
func gitStatusPorcelain(dir string) (string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// renderFileExplorer renders the file explorer two-pane layout.
func (m *Model) renderFileExplorer() string {
	paneHeight := m.height - 1
	leftWidth := m.fileExplorerLeftWidth()
	rightWidth := m.rightPaneWidth()
	rightHeight := m.rightPaneHeight()

	// Right pane: file preview
	var selectedPath string
	if m.fileExplorerTree != nil {
		if node := m.fileExplorerTree.SelectedNode(); node != nil && !node.IsDir {
			selectedPath = node.Path
		}
	}

	preview := RenderFilePreview(FilePreviewProps{
		Path:            selectedPath,
		Dir:             m.workDir,
		Width:           rightWidth,
		Height:          rightHeight,
		Scroll:          m.filePreviewScroll,
		HScroll:         m.filePreviewHScroll,
		ShowLineNumbers: m.lineNumbers,
		ThemeName:       m.config.ThemeName,
		Theme:           m.theme,
		Cache:           m.renderCache,
	})
	// Clamp scroll based on actual content
	m.filePreviewScroll = ClampFilePreviewScroll(m.filePreviewScroll, preview.TotalLines, rightHeight)
	right := preview.Content

	if leftWidth > 0 && m.fileExplorerTree != nil {
		left := m.fileExplorerTree.View(FileTreeViewProps{
			Width:   leftWidth,
			Height:  paneHeight,
			Focused: m.fileExplorerDepth == 0,
			Theme:   m.theme,
		})

		sepLines := make([]string, paneHeight)
		for i := range sepLines {
			sepLines[i] = "│"
		}
		separator := lipgloss.NewStyle().
			Foreground(lipgloss.Color(m.theme.ForegroundDim)).
			Render(strings.Join(sepLines, "\n"))

		return lipgloss.JoinHorizontal(lipgloss.Top, left, separator, right)
	}

	return right
}

// mergeFileExplorerTree updates the file tree with refreshed data, preserving
// expand/collapse state and cursor position.
func (m *Model) mergeFileExplorerTree(newRoots []*FileNode, porcelainOutput string) {
	if m.fileExplorerTree == nil {
		return
	}

	// Preserve expand state from old tree
	expandedPaths := collectExpandedPaths(m.fileExplorerTree.Roots())

	// Remember the selected path so we can restore cursor position
	var selectedPath string
	if node := m.fileExplorerTree.SelectedNode(); node != nil {
		selectedPath = node.Path
	}

	// Apply git status to new roots
	if porcelainOutput != "" {
		ApplyGitStatus(newRoots, porcelainOutput)
	}

	// Restore expand state in new tree
	restoreExpandedPaths(newRoots, expandedPaths)

	m.fileExplorerTree.SetRoots(newRoots)

	// Restore cursor position to the same file if possible
	if selectedPath != "" {
		rows := m.fileExplorerTree.VisibleRows()
		for i, row := range rows {
			if row.Node.Path == selectedPath {
				m.fileExplorerTree.Cursor = i
				break
			}
		}
	}
}

// handleFileExplorerMouse handles mouse events in file explorer mode.
func (m *Model) handleFileExplorerMouse(msg tea.MouseMsg, paneRow int) (tea.Model, tea.Cmd) {
	leftWidth := m.fileExplorerLeftWidth()
	inLeftPane := leftWidth > 0 && msg.X < leftWidth

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if inLeftPane {
			m.fileExplorerTree.ScrollBy(-mouseScrollLines, m.rightPaneHeight())
			m.fileExplorerDepth = 0
		} else {
			m.filePreviewScroll -= mouseScrollLines
			if m.filePreviewScroll < 0 {
				m.filePreviewScroll = 0
			}
			if m.fileExplorerDepth == 0 {
				m.fileExplorerDepth = 1
			}
		}
	case tea.MouseButtonWheelDown:
		if inLeftPane {
			m.fileExplorerTree.ScrollBy(mouseScrollLines, m.rightPaneHeight())
			m.fileExplorerDepth = 0
		} else {
			m.filePreviewScroll += mouseScrollLines
			if m.fileExplorerDepth == 0 {
				m.fileExplorerDepth = 1
			}
		}
	case tea.MouseButtonLeft:
		if msg.Action == tea.MouseActionPress && inLeftPane {
			m.fileExplorerDepth = 0
			if m.fileExplorerTree.ClickRow(paneRow) {
				m.fileExplorerDepth = 1
			}
			m.filePreviewScroll = 0
			m.filePreviewHScroll = 0
		}
	}

	return m, nil
}

// collectExpandedPaths returns a set of paths that are expanded in the tree.
func collectExpandedPaths(nodes []*FileNode) map[string]bool {
	paths := make(map[string]bool)
	var walk func([]*FileNode)
	walk = func(nodes []*FileNode) {
		for _, n := range nodes {
			if n.IsDir && n.Expanded {
				paths[n.Path] = true
				walk(n.Children)
			}
		}
	}
	walk(nodes)
	return paths
}

// restoreExpandedPaths applies saved expand state to a new tree.
func restoreExpandedPaths(nodes []*FileNode, paths map[string]bool) {
	for _, n := range nodes {
		if n.IsDir {
			if paths[n.Path] {
				n.Expanded = true
			}
			restoreExpandedPaths(n.Children, paths)
		}
	}
}
