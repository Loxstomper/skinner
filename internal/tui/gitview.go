package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/loxstomper/skinner/internal/config"
	"github.com/loxstomper/skinner/internal/git"
)

// enterGitView activates the git view, loading the commit list.
func (m *Model) enterGitView() {
	commits, err := git.LogCommits(200)
	if err != nil {
		m.statusFlash = "git log failed: " + err.Error()
		return
	}
	m.gitViewActive = true
	m.gitViewDepth = 0
	m.gitCommits = commits
	m.gitSelectedCommit = 0
	m.gitCommitScroll = 0
	m.gitFiles = nil
	m.gitSelectedFile = 0
	m.gitFileScroll = 0
	m.gitDiffScroll = 0
	m.gitDiffHScroll = 0
	m.gitParsedDiff = nil
	m.gitCommitSummary = ""
	m.gitDiffContent = ""

	// Load commit summary for the first commit
	if len(commits) > 0 {
		m.loadCommitSummary()
	}
}

// exitGitView deactivates the git view, restoring the previous view state.
func (m *Model) exitGitView() {
	m.gitViewActive = false
	m.gitViewDepth = 0
	m.gitCommits = nil
	m.gitFiles = nil
	m.gitParsedDiff = nil
	m.gitCommitSummary = ""
	m.gitDiffContent = ""
}

// handleGitViewKey routes key actions when the git view is active.
func (m *Model) handleGitViewKey(action string) (tea.Model, tea.Cmd) {
	switch action {
	case config.ActionQuit:
		m.activeModal = modalQuitConfirm
		return m, nil

	case config.ActionHelp:
		m.activeModal = modalHelp
		return m, nil

	case config.ActionGitView:
		// ctrl+g while in git view: exit
		m.exitGitView()
		return m, nil

	case config.ActionEscape:
		switch m.gitViewDepth {
		case 0:
			m.exitGitView()
		case 1:
			// Back to commit list
			m.gitViewDepth = 0
			m.gitFiles = nil
			m.gitSelectedFile = 0
			m.gitFileScroll = 0
			m.gitDiffScroll = 0
			m.gitDiffHScroll = 0
			m.gitParsedDiff = nil
			m.gitDiffContent = ""
			m.loadCommitSummary()
		case 2:
			// Exit sub-scroll, back to file list
			m.gitViewDepth = 1
			m.gitDiffScroll = 0
		}
		return m, nil

	case config.ActionExpand:
		switch m.gitViewDepth {
		case 0:
			// Drill into commit → file list
			if len(m.gitCommits) > 0 {
				m.gitViewDepth = 1
				m.loadFileList()
			}
		case 1:
			// Enter sub-scroll on diff
			if len(m.gitFiles) > 0 {
				m.gitViewDepth = 2
			}
		}
		return m, nil

	case config.ActionMoveDown:
		m.gitViewMoveDown()
		return m, nil

	case config.ActionMoveUp:
		m.gitViewMoveUp()
		return m, nil

	case config.ActionJumpTop:
		m.gitViewJumpTop()
		return m, nil

	case config.ActionJumpBottom:
		m.gitViewJumpBottom()
		return m, nil

	case config.ActionFocusLeft:
		// h key: horizontal scroll left at depth 1/2
		if m.gitViewDepth >= 1 {
			if m.gitDiffHScroll > 0 {
				m.gitDiffHScroll--
			}
		}
		return m, nil

	case config.ActionFocusRight:
		// l key: horizontal scroll right at depth 1/2
		if m.gitViewDepth >= 1 {
			m.gitDiffHScroll++
		}
		return m, nil

	case config.ActionToggleLeftPane:
		// [ key still works in git view for hiding/showing left pane
		if m.effectiveLayout() == "bottom" {
			m.bottomBarVisible = !m.bottomBarVisible
		} else {
			m.leftPaneVisible = !m.leftPaneVisible
		}
		return m, nil
	}

	return m, nil
}

// gitViewMoveDown handles j/down in git view.
func (m *Model) gitViewMoveDown() {
	switch m.gitViewDepth {
	case 0:
		// Scroll commit list
		if m.gitSelectedCommit < len(m.gitCommits)-1 {
			m.gitSelectedCommit++
			m.loadCommitSummary()
		}
	case 1:
		// Scroll file list (left pane focused)
		if m.gitSelectedFile < len(m.gitFiles)-1 {
			m.gitSelectedFile++
			m.gitDiffScroll = 0
			m.gitDiffHScroll = 0
			m.loadFileDiff()
		}
	case 2:
		// Scroll diff vertically
		m.gitDiffScroll++
	}
}

// gitViewMoveUp handles k/up in git view.
func (m *Model) gitViewMoveUp() {
	switch m.gitViewDepth {
	case 0:
		if m.gitSelectedCommit > 0 {
			m.gitSelectedCommit--
			m.loadCommitSummary()
		}
	case 1:
		if m.gitSelectedFile > 0 {
			m.gitSelectedFile--
			m.gitDiffScroll = 0
			m.gitDiffHScroll = 0
			m.loadFileDiff()
		}
	case 2:
		if m.gitDiffScroll > 0 {
			m.gitDiffScroll--
		}
	}
}

// gitViewJumpTop handles gg/Home in git view.
func (m *Model) gitViewJumpTop() {
	switch m.gitViewDepth {
	case 0:
		m.gitSelectedCommit = 0
		m.gitCommitScroll = 0
		m.loadCommitSummary()
	case 1:
		m.gitSelectedFile = 0
		m.gitFileScroll = 0
		m.gitDiffScroll = 0
		m.gitDiffHScroll = 0
		m.loadFileDiff()
	case 2:
		m.gitDiffScroll = 0
	}
}

// gitViewJumpBottom handles G/End in git view.
func (m *Model) gitViewJumpBottom() {
	switch m.gitViewDepth {
	case 0:
		if len(m.gitCommits) > 0 {
			m.gitSelectedCommit = len(m.gitCommits) - 1
			m.loadCommitSummary()
		}
	case 1:
		if len(m.gitFiles) > 0 {
			m.gitSelectedFile = len(m.gitFiles) - 1
			m.gitDiffScroll = 0
			m.gitDiffHScroll = 0
			m.loadFileDiff()
		}
	case 2:
		// Jump to bottom of diff — clamped during rendering
		m.gitDiffScroll = 999999
	}
}

// loadCommitSummary loads the commit summary for the currently selected commit.
func (m *Model) loadCommitSummary() {
	if m.gitSelectedCommit >= len(m.gitCommits) {
		return
	}
	sha := m.gitCommits[m.gitSelectedCommit].Hash
	summary, err := git.ShowCommit(sha)
	if err != nil {
		m.gitCommitSummary = "Error loading commit: " + err.Error()
		return
	}
	m.gitCommitSummary = summary
}

// loadFileList loads the file list for the currently selected commit.
func (m *Model) loadFileList() {
	if m.gitSelectedCommit >= len(m.gitCommits) {
		return
	}
	sha := m.gitCommits[m.gitSelectedCommit].Hash
	files, err := git.DiffTreeFiles(sha)
	if err != nil {
		m.statusFlash = "git diff-tree failed: " + err.Error()
		m.gitViewDepth = 0
		return
	}
	m.gitFiles = files
	m.gitSelectedFile = 0
	m.gitFileScroll = 0
	m.gitDiffScroll = 0
	m.gitDiffHScroll = 0
	m.loadFileDiff()
}

// loadFileDiff loads the diff for the currently selected file.
func (m *Model) loadFileDiff() {
	if m.gitSelectedCommit >= len(m.gitCommits) || m.gitSelectedFile >= len(m.gitFiles) {
		m.gitParsedDiff = nil
		m.gitDiffContent = ""
		return
	}
	sha := m.gitCommits[m.gitSelectedCommit].Hash
	path := m.gitFiles[m.gitSelectedFile].Path
	diff, err := git.FileDiff(sha, path)
	if err != nil {
		m.gitDiffContent = "Error loading diff: " + err.Error()
		m.gitParsedDiff = nil
		return
	}
	m.gitDiffContent = diff
	m.gitParsedDiff = ParseUnifiedDiff(diff)
}
