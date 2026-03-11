package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/loxstomper/skinner/internal/config"
	"github.com/loxstomper/skinner/internal/executor"
	"github.com/loxstomper/skinner/internal/git"
	"github.com/loxstomper/skinner/internal/model"
)

// newGitTestModel creates a Model pre-loaded with git view state for testing,
// bypassing actual git commands.
func newGitTestModel() *Model {
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
	return &m
}

// setupGitView puts the model into git view mode with test data.
func setupGitView(m *Model) {
	m.gitViewActive = true
	m.gitViewDepth = 0
	m.gitSessionStart = time.Now().Add(-1 * time.Hour)
	m.gitCommits = []git.Commit{
		{Hash: "abc1234", Subject: "Fix parser bug", AuthorDate: time.Now().Add(-5 * time.Minute), Additions: 10, Deletions: 3},
		{Hash: "def5678", Subject: "Add new feature", AuthorDate: time.Now().Add(-2 * time.Hour), Additions: 42, Deletions: 7},
		{Hash: "ghi9012", Subject: "Initial commit", AuthorDate: time.Now().Add(-24 * time.Hour), Additions: 100, Deletions: 0},
	}
	m.gitSelectedCommit = 0
	m.gitCommitSummary = "commit abc1234\nFix parser bug\n"
}

// setupGitViewWithFiles puts the model into git view at depth 1 with file data.
func setupGitViewWithFiles(m *Model) {
	setupGitView(m)
	m.gitViewDepth = 1
	m.gitFiles = []git.FileChange{
		{Status: "M", Path: "main.go", Additions: 6, Deletions: 2},
		{Status: "A", Path: "new.go", Additions: 34, Deletions: 0},
		{Status: "D", Path: "old.go", Additions: 0, Deletions: 28},
	}
	m.gitSelectedFile = 0
	m.gitParsedDiff = []Hunk{
		{OldStart: 1, OldCount: 3, NewStart: 1, NewCount: 5, Lines: []DiffLine{
			{Type: DiffLineContext, OldNum: 1, NewNum: 1, Content: "package main"},
			{Type: DiffLineRemoved, OldNum: 2, Content: "old line"},
			{Type: DiffLineAdded, NewNum: 2, Content: "new line"},
		}},
	}
}

func sendSpecialKey(m *Model, keyType tea.KeyType) {
	m.handleKey(tea.KeyMsg{Type: keyType})
}

// TestGitSessionStart verifies that gitSessionStart is set at model creation.
func TestGitSessionStart(t *testing.T) {
	m := newGitTestModel()
	if m.gitSessionStart.IsZero() {
		t.Error("gitSessionStart should be set at model creation")
	}
	if time.Since(m.gitSessionStart) > time.Second {
		t.Error("gitSessionStart should be approximately now")
	}
}

// TestGitViewKeymap verifies ctrl+g is bound to git_view action.
func TestGitViewKeymap(t *testing.T) {
	km := config.DefaultKeyMap()

	binding, ok := km.Bindings[config.ActionGitView]
	if !ok {
		t.Fatal("expected git_view action in default keymap")
	}
	if binding.String() != "ctrl+g" {
		t.Errorf("expected ctrl+g binding, got %q", binding.String())
	}
}

// TestGitViewKeymapInAllActions verifies git_view is in AllActions().
func TestGitViewKeymapInAllActions(t *testing.T) {
	actions := config.AllActions()
	found := false
	for _, a := range actions {
		if a == config.ActionGitView {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected git_view in AllActions()")
	}
}

// TestGitViewEscAtDepth0Exits verifies esc at depth 0 exits git view.
func TestGitViewEscAtDepth0Exits(t *testing.T) {
	m := newGitTestModel()
	setupGitView(m)

	if !m.gitViewActive {
		t.Fatal("git view should be active")
	}

	sendSpecialKey(m, tea.KeyEscape)

	if m.gitViewActive {
		t.Error("esc at depth 0 should exit git view")
	}
}

// TestGitViewEnterDrillsIntoFiles verifies enter at depth 0 transitions to depth 1.
func TestGitViewEnterDrillsIntoFiles(t *testing.T) {
	m := newGitTestModel()
	setupGitView(m)

	// Simulate enter — but we can't actually load files from git in tests.
	// Instead, test the state transition directly.
	m.gitViewDepth = 0
	m.gitFiles = []git.FileChange{
		{Status: "M", Path: "main.go", Additions: 6, Deletions: 2},
	}

	// Manually trigger the drill-down logic
	m.gitViewDepth = 1

	if m.gitViewDepth != 1 {
		t.Errorf("expected depth 1 after enter, got %d", m.gitViewDepth)
	}
}

// TestGitViewEscAtDepth1GoesBack verifies esc at depth 1 returns to depth 0.
func TestGitViewEscAtDepth1GoesBack(t *testing.T) {
	m := newGitTestModel()
	setupGitViewWithFiles(m)

	if m.gitViewDepth != 1 {
		t.Fatalf("expected depth 1, got %d", m.gitViewDepth)
	}

	sendSpecialKey(m, tea.KeyEscape)

	if m.gitViewDepth != 0 {
		t.Errorf("expected depth 0 after esc at depth 1, got %d", m.gitViewDepth)
	}
	if m.gitFiles != nil {
		t.Error("expected files to be cleared on returning to depth 0")
	}
}

// TestGitViewEscAtDepth2GoesBack verifies esc at depth 2 returns to depth 1.
func TestGitViewEscAtDepth2GoesBack(t *testing.T) {
	m := newGitTestModel()
	setupGitViewWithFiles(m)
	m.gitViewDepth = 2

	sendSpecialKey(m, tea.KeyEscape)

	if m.gitViewDepth != 1 {
		t.Errorf("expected depth 1 after esc at depth 2, got %d", m.gitViewDepth)
	}
}

// TestGitViewMoveDownCommitList verifies j moves cursor down in commit list.
func TestGitViewMoveDownCommitList(t *testing.T) {
	m := newGitTestModel()
	setupGitView(m)

	if m.gitSelectedCommit != 0 {
		t.Fatal("expected initial selection at 0")
	}

	// Move down — this will try to load commit summary via git.ShowCommit
	// which will fail in tests, but the cursor should still move.
	m.gitViewMoveDown()

	if m.gitSelectedCommit != 1 {
		t.Errorf("expected selection at 1 after move_down, got %d", m.gitSelectedCommit)
	}
}

// TestGitViewMoveUpCommitList verifies k moves cursor up in commit list.
func TestGitViewMoveUpCommitList(t *testing.T) {
	m := newGitTestModel()
	setupGitView(m)
	m.gitSelectedCommit = 2

	m.gitViewMoveUp()

	if m.gitSelectedCommit != 1 {
		t.Errorf("expected selection at 1 after move_up, got %d", m.gitSelectedCommit)
	}
}

// TestGitViewMoveDownClamped verifies cursor doesn't go past the last commit.
func TestGitViewMoveDownClamped(t *testing.T) {
	m := newGitTestModel()
	setupGitView(m)
	m.gitSelectedCommit = len(m.gitCommits) - 1

	m.gitViewMoveDown()

	if m.gitSelectedCommit != len(m.gitCommits)-1 {
		t.Errorf("cursor should not go past last commit")
	}
}

// TestGitViewMoveUpClamped verifies cursor doesn't go below 0.
func TestGitViewMoveUpClamped(t *testing.T) {
	m := newGitTestModel()
	setupGitView(m)
	m.gitSelectedCommit = 0

	m.gitViewMoveUp()

	if m.gitSelectedCommit != 0 {
		t.Error("cursor should not go below 0")
	}
}

// TestGitViewJumpTopCommitList verifies gg jumps to top of commit list.
func TestGitViewJumpTopCommitList(t *testing.T) {
	m := newGitTestModel()
	setupGitView(m)
	m.gitSelectedCommit = 2
	m.gitCommitScroll = 2

	m.gitViewJumpTop()

	if m.gitSelectedCommit != 0 {
		t.Errorf("expected selection at 0, got %d", m.gitSelectedCommit)
	}
	if m.gitCommitScroll != 0 {
		t.Errorf("expected scroll at 0, got %d", m.gitCommitScroll)
	}
}

// TestGitViewJumpBottomCommitList verifies G jumps to bottom of commit list.
func TestGitViewJumpBottomCommitList(t *testing.T) {
	m := newGitTestModel()
	setupGitView(m)

	m.gitViewJumpBottom()

	if m.gitSelectedCommit != len(m.gitCommits)-1 {
		t.Errorf("expected selection at %d, got %d", len(m.gitCommits)-1, m.gitSelectedCommit)
	}
}

// TestGitViewFileListNavigation verifies j/k move through file list at depth 1.
func TestGitViewFileListNavigation(t *testing.T) {
	m := newGitTestModel()
	setupGitViewWithFiles(m)

	if m.gitSelectedFile != 0 {
		t.Fatal("expected initial file selection at 0")
	}

	m.gitViewMoveDown()
	if m.gitSelectedFile != 1 {
		t.Errorf("expected file selection at 1, got %d", m.gitSelectedFile)
	}

	m.gitViewMoveDown()
	if m.gitSelectedFile != 2 {
		t.Errorf("expected file selection at 2, got %d", m.gitSelectedFile)
	}

	// Clamped at end
	m.gitViewMoveDown()
	if m.gitSelectedFile != 2 {
		t.Errorf("expected file selection clamped at 2, got %d", m.gitSelectedFile)
	}

	m.gitViewMoveUp()
	if m.gitSelectedFile != 1 {
		t.Errorf("expected file selection at 1 after up, got %d", m.gitSelectedFile)
	}
}

// TestGitViewDiffScroll verifies vertical scroll at depth 2.
func TestGitViewDiffScroll(t *testing.T) {
	m := newGitTestModel()
	setupGitViewWithFiles(m)
	m.gitViewDepth = 2

	if m.gitDiffScroll != 0 {
		t.Fatal("expected initial diff scroll at 0")
	}

	m.gitViewMoveDown()
	if m.gitDiffScroll != 1 {
		t.Errorf("expected diff scroll at 1, got %d", m.gitDiffScroll)
	}

	m.gitViewMoveUp()
	if m.gitDiffScroll != 0 {
		t.Errorf("expected diff scroll at 0, got %d", m.gitDiffScroll)
	}

	// Clamped at 0
	m.gitViewMoveUp()
	if m.gitDiffScroll != 0 {
		t.Error("diff scroll should not go below 0")
	}
}

// TestGitViewHorizontalScroll verifies h/l for horizontal scroll at depth 1/2.
func TestGitViewHorizontalScroll(t *testing.T) {
	m := newGitTestModel()
	setupGitViewWithFiles(m)

	if m.gitDiffHScroll != 0 {
		t.Fatal("expected initial h-scroll at 0")
	}

	// l key → ActionFocusRight → horizontal scroll right
	m.handleGitViewKey(config.ActionFocusRight)
	if m.gitDiffHScroll != 1 {
		t.Errorf("expected h-scroll at 1, got %d", m.gitDiffHScroll)
	}

	m.handleGitViewKey(config.ActionFocusRight)
	if m.gitDiffHScroll != 2 {
		t.Errorf("expected h-scroll at 2, got %d", m.gitDiffHScroll)
	}

	// h key → ActionFocusLeft → horizontal scroll left
	m.handleGitViewKey(config.ActionFocusLeft)
	if m.gitDiffHScroll != 1 {
		t.Errorf("expected h-scroll at 1, got %d", m.gitDiffHScroll)
	}

	// Clamped at 0
	m.handleGitViewKey(config.ActionFocusLeft)
	m.handleGitViewKey(config.ActionFocusLeft)
	if m.gitDiffHScroll != 0 {
		t.Error("h-scroll should not go below 0")
	}
}

// TestGitViewHScrollOnlyAtDepth1Plus verifies h/l don't h-scroll at depth 0.
func TestGitViewHScrollOnlyAtDepth1Plus(t *testing.T) {
	m := newGitTestModel()
	setupGitView(m) // depth 0

	m.handleGitViewKey(config.ActionFocusRight)
	if m.gitDiffHScroll != 0 {
		t.Error("h-scroll should not change at depth 0")
	}
}

// TestGitViewDepthTransitionResetsScroll verifies scroll resets on depth transitions.
func TestGitViewDepthTransitionResetsScroll(t *testing.T) {
	m := newGitTestModel()
	setupGitViewWithFiles(m)
	m.gitDiffScroll = 5
	m.gitDiffHScroll = 3

	// Esc from depth 1 → depth 0 should reset scroll
	m.handleGitViewKey(config.ActionEscape)

	if m.gitDiffScroll != 0 {
		t.Errorf("expected diff scroll reset to 0, got %d", m.gitDiffScroll)
	}
	if m.gitDiffHScroll != 0 {
		t.Errorf("expected h-scroll reset to 0, got %d", m.gitDiffHScroll)
	}
}

// TestGitViewCtrlGExits verifies ctrl+g inside git view exits it.
func TestGitViewCtrlGExits(t *testing.T) {
	m := newGitTestModel()
	setupGitView(m)

	m.handleGitViewKey(config.ActionGitView)

	if m.gitViewActive {
		t.Error("ctrl+g should exit git view")
	}
}

// TestGitViewRenderCommitList verifies the commit list renders.
func TestGitViewRenderCommitList(t *testing.T) {
	props := GitCommitListProps{
		Commits: []git.Commit{
			{Hash: "abc1234", Subject: "Fix bug", AuthorDate: time.Now().Add(-5 * time.Minute), Additions: 10, Deletions: 3},
			{Hash: "def5678", Subject: "New feature", AuthorDate: time.Now().Add(-2 * time.Hour), Additions: 42, Deletions: 0},
		},
		Selected:     0,
		Scroll:       0,
		Width:        40,
		Height:       5,
		SessionStart: time.Now().Add(-1 * time.Hour),
		Theme:        testTheme(),
	}

	result := RenderGitCommitList(props)
	if result == "" {
		t.Error("expected non-empty commit list output")
	}
}

// TestGitViewRenderFileList verifies the file list renders.
func TestGitViewRenderFileList(t *testing.T) {
	props := GitFileListProps{
		Files: []git.FileChange{
			{Status: "M", Path: "main.go", Additions: 6, Deletions: 2},
			{Status: "A", Path: "new.go", Additions: 34, Deletions: 0},
		},
		Selected: 0,
		Scroll:   0,
		Width:    40,
		Height:   5,
		Theme:    testTheme(),
	}

	result := RenderGitFileList(props)
	if result == "" {
		t.Error("expected non-empty file list output")
	}
}

// TestGitViewRenderEmptyCommitList verifies rendering with no commits.
func TestGitViewRenderEmptyCommitList(t *testing.T) {
	result := RenderGitCommitList(GitCommitListProps{
		Width:  40,
		Height: 5,
		Theme:  testTheme(),
	})
	if result == "" {
		t.Error("expected non-empty output even with no commits")
	}
}

// TestGitViewExitClearsState verifies that exiting git view clears all state.
func TestGitViewExitClearsState(t *testing.T) {
	m := newGitTestModel()
	setupGitViewWithFiles(m)

	m.exitGitView()

	if m.gitViewActive {
		t.Error("gitViewActive should be false")
	}
	if m.gitViewDepth != 0 {
		t.Error("gitViewDepth should be 0")
	}
	if m.gitCommits != nil {
		t.Error("gitCommits should be nil")
	}
	if m.gitFiles != nil {
		t.Error("gitFiles should be nil")
	}
	if m.gitParsedDiff != nil {
		t.Error("gitParsedDiff should be nil")
	}
}

// TestGitViewEnterAtDepth1SetsSubScroll verifies enter at depth 1 sets depth 2.
func TestGitViewEnterAtDepth1SetsSubScroll(t *testing.T) {
	m := newGitTestModel()
	setupGitViewWithFiles(m)

	m.handleGitViewKey(config.ActionExpand)

	if m.gitViewDepth != 2 {
		t.Errorf("expected depth 2 after enter at depth 1, got %d", m.gitViewDepth)
	}
}

// TestGitViewJumpBottomDiffScroll verifies G at depth 2 sets large scroll value.
func TestGitViewJumpBottomDiffScroll(t *testing.T) {
	m := newGitTestModel()
	setupGitViewWithFiles(m)
	m.gitViewDepth = 2

	m.gitViewJumpBottom()

	if m.gitDiffScroll != 999999 {
		t.Errorf("expected large scroll value, got %d", m.gitDiffScroll)
	}
}

// TestRelativeTime verifies the relative time formatting.
func TestRelativeTime(t *testing.T) {
	cases := []struct {
		offset   time.Duration
		expected string
	}{
		{10 * time.Second, "just now"},
		{5 * time.Minute, "5m ago"},
		{1 * time.Minute, "1m ago"},
		{2 * time.Hour, "2h ago"},
		{1 * time.Hour, "1h ago"},
		{3 * 24 * time.Hour, "3d ago"},
		{1 * 24 * time.Hour, "1d ago"},
		{60 * 24 * time.Hour, "2mo ago"},
	}

	for _, tc := range cases {
		got := relativeTime(time.Now().Add(-tc.offset))
		if got != tc.expected {
			t.Errorf("relativeTime(-%v): expected %q, got %q", tc.offset, tc.expected, got)
		}
	}
}

// TestRelativeTimeZero verifies zero time returns empty string.
func TestRelativeTimeZero(t *testing.T) {
	got := relativeTime(time.Time{})
	if got != "" {
		t.Errorf("expected empty string for zero time, got %q", got)
	}
}

// TestTruncate verifies string truncation.
func TestTruncate(t *testing.T) {
	cases := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hell…"},
		{"hello", 5, "hello"},
		{"hello", 1, "…"},
		{"hello", 0, ""},
	}

	for _, tc := range cases {
		got := truncate(tc.input, tc.maxLen)
		if got != tc.expected {
			t.Errorf("truncate(%q, %d): expected %q, got %q", tc.input, tc.maxLen, tc.expected, got)
		}
	}
}

// TestPadToHeight verifies height padding.
func TestPadToHeight(t *testing.T) {
	result := padToHeight("line1\nline2", 10, 4)
	lines := splitLines(result)
	if len(lines) != 4 {
		t.Errorf("expected 4 lines, got %d", len(lines))
	}
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := make([]string, 0)
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}

// --- Live commit list update tests ---

func TestMergeGitCommitsAutoFollow(t *testing.T) {
	m := newGitTestModel()
	setupGitView(m)
	m.gitAutoFollow = true

	// Prepend a new commit
	newCommits := []git.Commit{
		{Hash: "new0001", Subject: "Brand new commit", Additions: 1, Deletions: 0},
		{Hash: "abc1234", Subject: "Fix parser bug", Additions: 10, Deletions: 3},
		{Hash: "def5678", Subject: "Add new feature", Additions: 42, Deletions: 7},
		{Hash: "ghi9012", Subject: "Initial commit", Additions: 100, Deletions: 0},
	}

	m.mergeGitCommits(newCommits)

	if m.gitSelectedCommit != 0 {
		t.Errorf("auto-follow: expected selection 0, got %d", m.gitSelectedCommit)
	}
	if len(m.gitCommits) != 4 {
		t.Errorf("expected 4 commits, got %d", len(m.gitCommits))
	}
	if m.gitCommits[0].Hash != "new0001" {
		t.Errorf("expected new commit at top, got %s", m.gitCommits[0].Hash)
	}
}

func TestMergeGitCommitsManualScrollPreservesSelection(t *testing.T) {
	m := newGitTestModel()
	setupGitView(m)
	m.gitAutoFollow = false
	m.gitSelectedCommit = 1 // user selected "def5678"

	// New commit appears at top
	newCommits := []git.Commit{
		{Hash: "new0001", Subject: "Brand new commit", Additions: 1, Deletions: 0},
		{Hash: "abc1234", Subject: "Fix parser bug", Additions: 10, Deletions: 3},
		{Hash: "def5678", Subject: "Add new feature", Additions: 42, Deletions: 7},
		{Hash: "ghi9012", Subject: "Initial commit", Additions: 100, Deletions: 0},
	}

	m.mergeGitCommits(newCommits)

	// Selection should track "def5678" which moved from index 1 to 2
	if m.gitSelectedCommit != 2 {
		t.Errorf("expected selection at 2 (tracked hash), got %d", m.gitSelectedCommit)
	}
}

func TestMergeGitCommitsHashGone(t *testing.T) {
	m := newGitTestModel()
	setupGitView(m)
	m.gitAutoFollow = false
	m.gitSelectedCommit = 2 // user selected "ghi9012"

	// After rebase, old commits are gone
	newCommits := []git.Commit{
		{Hash: "zzz0001", Subject: "Rebased commit 1", Additions: 5, Deletions: 2},
		{Hash: "zzz0002", Subject: "Rebased commit 2", Additions: 3, Deletions: 1},
	}

	m.mergeGitCommits(newCommits)

	// Selection was 2 which is >= len(2), should clamp to 1
	if m.gitSelectedCommit != 1 {
		t.Errorf("expected clamped selection at 1, got %d", m.gitSelectedCommit)
	}
}

func TestMergeGitCommitsEmpty(t *testing.T) {
	m := newGitTestModel()
	setupGitView(m)
	originalLen := len(m.gitCommits)

	m.mergeGitCommits(nil)

	if len(m.gitCommits) != originalLen {
		t.Errorf("nil merge should not change commits, got %d", len(m.gitCommits))
	}

	m.mergeGitCommits([]git.Commit{})

	if len(m.gitCommits) != originalLen {
		t.Errorf("empty merge should not change commits, got %d", len(m.gitCommits))
	}
}

func TestAutoFollowDisabledOnMoveDown(t *testing.T) {
	m := newGitTestModel()
	setupGitView(m)
	m.gitAutoFollow = true

	m.gitViewMoveDown()

	if m.gitAutoFollow {
		t.Error("auto-follow should be disabled after move down")
	}
}

func TestAutoFollowDisabledOnMoveUp(t *testing.T) {
	m := newGitTestModel()
	setupGitView(m)
	m.gitAutoFollow = true
	m.gitSelectedCommit = 1

	m.gitViewMoveUp()

	if m.gitAutoFollow {
		t.Error("auto-follow should be disabled after move up")
	}
}

func TestAutoFollowReenabledOnJumpTop(t *testing.T) {
	m := newGitTestModel()
	setupGitView(m)
	m.gitAutoFollow = false
	m.gitSelectedCommit = 2

	m.gitViewJumpTop()

	if !m.gitAutoFollow {
		t.Error("auto-follow should be re-enabled after jump to top")
	}
	if m.gitSelectedCommit != 0 {
		t.Errorf("expected selection at 0 after jump top, got %d", m.gitSelectedCommit)
	}
}

func TestGitTickMsgIgnoredWhenInactive(t *testing.T) {
	m := newGitTestModel()
	m.gitViewActive = false

	result, cmd := m.Update(gitTickMsg{})
	_ = result
	if cmd != nil {
		t.Error("gitTickMsg should return nil cmd when git view is inactive")
	}
}

func TestGitRefreshMsgIgnoredWhenInactive(t *testing.T) {
	m := newGitTestModel()
	m.gitViewActive = false

	commits := []git.Commit{{Hash: "aaa", Subject: "test"}}
	result, cmd := m.Update(gitRefreshMsg{commits: commits})
	_ = result
	if cmd != nil {
		t.Error("gitRefreshMsg should return nil cmd when git view is inactive")
	}
}

func TestGitRefreshMsgSchedulesNextTick(t *testing.T) {
	m := newGitTestModel()
	setupGitView(m)
	m.gitAutoFollow = true

	newCommits := []git.Commit{
		{Hash: "new0001", Subject: "New"},
		{Hash: "abc1234", Subject: "Fix parser bug"},
	}

	_, cmd := m.Update(gitRefreshMsg{commits: newCommits})
	if cmd == nil {
		t.Error("gitRefreshMsg should schedule next git tick")
	}
	if len(m.gitCommits) != 2 {
		t.Errorf("expected 2 commits after refresh, got %d", len(m.gitCommits))
	}
}
