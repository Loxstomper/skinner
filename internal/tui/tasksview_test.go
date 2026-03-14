package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/loxstomper/skinner/internal/bd"
	"github.com/loxstomper/skinner/internal/session"
)

func newTasksTestModel() *Model {
	m := newTestModel([]session.Event{}, 1)
	return m
}

func TestEnterTasksViewSetsState(t *testing.T) {
	m := newTasksTestModel()
	cmd := m.enterTasksView()

	if !m.tasksViewActive {
		t.Error("expected tasksViewActive to be true")
	}
	if !m.tasksViewLoading {
		t.Error("expected tasksViewLoading to be true")
	}
	if m.tasksViewDepth != 0 {
		t.Errorf("expected depth 0, got %d", m.tasksViewDepth)
	}
	if m.tasksViewTab != 0 {
		t.Errorf("expected tab 0 (Ready), got %d", m.tasksViewTab)
	}
	if m.tasksViewExpanded == nil {
		t.Error("expected tasksViewExpanded to be initialized")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for async data fetch")
	}
}

func TestExitTasksViewClearsState(t *testing.T) {
	m := newTasksTestModel()
	m.enterTasksView()
	m.tasksViewGraph = bd.NewGraph(nil)
	m.tasksViewFiltered = []*bd.Issue{{ID: "test-1"}}
	m.tasksViewVisibleRows = []tasksViewRow{{issue: &bd.Issue{ID: "test-1"}}}
	m.tasksViewSearchActive = true
	m.tasksViewSearchQuery = "foo"

	m.exitTasksView()

	if m.tasksViewActive {
		t.Error("expected tasksViewActive to be false")
	}
	if m.tasksViewGraph != nil {
		t.Error("expected tasksViewGraph to be nil")
	}
	if m.tasksViewFiltered != nil {
		t.Error("expected tasksViewFiltered to be nil")
	}
	if m.tasksViewVisibleRows != nil {
		t.Error("expected tasksViewVisibleRows to be nil")
	}
	if m.tasksViewExpanded != nil {
		t.Error("expected tasksViewExpanded to be nil")
	}
	if m.tasksViewSearchActive {
		t.Error("expected tasksViewSearchActive to be false")
	}
	if m.tasksViewSearchQuery != "" {
		t.Error("expected tasksViewSearchQuery to be empty")
	}
}

func TestTasksViewLoadingRender(t *testing.T) {
	m := newTasksTestModel()
	m.tasksViewActive = true
	m.tasksViewLoading = true

	view := m.renderTasksView()

	if !strings.Contains(view, "Loading issues...") {
		t.Errorf("expected loading text, got %q", view)
	}
}

func TestTasksViewErrorRender(t *testing.T) {
	m := newTasksTestModel()
	m.tasksViewActive = true
	m.tasksViewError = errors.New("bd not found")

	view := m.renderTasksView()

	if !strings.Contains(view, "Could not load issues: bd not found") {
		t.Errorf("expected error message, got %q", view)
	}
	if !strings.Contains(view, "r:retry") {
		t.Errorf("expected retry hint, got %q", view)
	}
	if !strings.Contains(view, "q:back") {
		t.Errorf("expected back hint, got %q", view)
	}
}

func TestTasksViewDataMsgSuccess(t *testing.T) {
	m := newTasksTestModel()
	m.tasksViewActive = true
	m.tasksViewLoading = true

	issues := []bd.Issue{
		{ID: "test-1", Title: "First issue", Status: "open", Priority: 1},
		{ID: "test-2", Title: "Second issue", Status: "in_progress", Priority: 2},
	}
	graph := bd.NewGraph(issues)

	msg := tasksViewDataMsg{graph: graph}
	m.Update(msg)

	if m.tasksViewLoading {
		t.Error("expected loading to be false after data msg")
	}
	if m.tasksViewGraph == nil {
		t.Error("expected graph to be set")
	}
	if m.tasksViewError != nil {
		t.Errorf("expected no error, got %v", m.tasksViewError)
	}
}

func TestTasksViewDataMsgError(t *testing.T) {
	m := newTasksTestModel()
	m.tasksViewActive = true
	m.tasksViewLoading = true

	msg := tasksViewDataMsg{err: errors.New("connection refused")}
	m.Update(msg)

	if m.tasksViewLoading {
		t.Error("expected loading to be false after error msg")
	}
	if m.tasksViewError == nil {
		t.Error("expected error to be set")
	}
	if m.tasksViewGraph != nil {
		t.Error("expected graph to be nil on error")
	}
}

func TestTasksViewDataMsgIgnoredWhenInactive(t *testing.T) {
	m := newTasksTestModel()
	m.tasksViewActive = false

	issues := []bd.Issue{{ID: "test-1", Status: "open"}}
	msg := tasksViewDataMsg{graph: bd.NewGraph(issues)}
	m.Update(msg)

	if m.tasksViewGraph != nil {
		t.Error("expected graph to remain nil when tasks view inactive")
	}
}

func TestTasksViewEscapeExits(t *testing.T) {
	m := newTasksTestModel()
	m.enterTasksView()
	m.tasksViewLoading = false

	m.handleTasksViewKey("escape", "")

	if m.tasksViewActive {
		t.Error("expected tasks view to exit on escape at depth 0")
	}
}

func TestTasksViewEscapeAtDepth1GoesBack(t *testing.T) {
	m := newTasksTestModel()
	m.enterTasksView()
	m.tasksViewLoading = false
	m.tasksViewDepth = 1

	m.handleTasksViewKey("escape", "")

	if !m.tasksViewActive {
		t.Error("expected tasks view to remain active")
	}
	if m.tasksViewDepth != 0 {
		t.Errorf("expected depth 0, got %d", m.tasksViewDepth)
	}
}

func TestTasksViewQuitExits(t *testing.T) {
	m := newTasksTestModel()
	m.enterTasksView()
	m.tasksViewLoading = false

	m.handleTasksViewKey("quit", "")

	if m.tasksViewActive {
		t.Error("expected tasks view to exit on quit")
	}
}

func TestTasksViewRefreshKey(t *testing.T) {
	m := newTasksTestModel()
	m.enterTasksView()
	m.tasksViewLoading = false
	m.tasksViewGraph = bd.NewGraph(nil)

	_, cmd := m.handleTasksViewKey("", "r")

	if !m.tasksViewLoading {
		t.Error("expected loading to be true after refresh")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd for refresh fetch")
	}
}

func TestTasksViewRefilterByTab(t *testing.T) {
	m := newTasksTestModel()
	issues := []bd.Issue{
		{ID: "t-1", Status: "open", Priority: 1},
		{ID: "t-2", Status: "in_progress", Priority: 2},
		{ID: "t-3", Status: "blocked", Priority: 1},
		{ID: "t-4", Status: "closed", Priority: 3},
	}
	m.tasksViewGraph = bd.NewGraph(issues)

	// Tab 0: Ready (open issues)
	m.tasksViewTab = 0
	m.tasksViewRefilter()
	if len(m.tasksViewFiltered) != 1 {
		t.Errorf("Ready tab: expected 1 issue, got %d", len(m.tasksViewFiltered))
	}

	// Tab 1: All (regardless of status)
	m.tasksViewTab = 1
	m.tasksViewRefilter()
	if len(m.tasksViewFiltered) != 4 {
		t.Errorf("All tab: expected 4 issues (all regardless of status), got %d", len(m.tasksViewFiltered))
	}

	// Tab 2: Blocked
	m.tasksViewTab = 2
	m.tasksViewRefilter()
	if len(m.tasksViewFiltered) != 1 {
		t.Errorf("Blocked tab: expected 1 issue, got %d", len(m.tasksViewFiltered))
	}

	// Tab 3: In Progress
	m.tasksViewTab = 3
	m.tasksViewRefilter()
	if len(m.tasksViewFiltered) != 1 {
		t.Errorf("InProgress tab: expected 1 issue, got %d", len(m.tasksViewFiltered))
	}
}

func TestTasksViewCursorMovement(t *testing.T) {
	m := newTasksTestModel()
	m.tasksViewVisibleRows = []tasksViewRow{
		{issue: &bd.Issue{ID: "t-1"}},
		{issue: &bd.Issue{ID: "t-2"}},
		{issue: &bd.Issue{ID: "t-3"}},
	}

	m.tasksViewMoveCursor(1)
	if m.tasksViewCursor != 1 {
		t.Errorf("expected cursor 1, got %d", m.tasksViewCursor)
	}

	m.tasksViewMoveCursor(5) // clamp
	if m.tasksViewCursor != 2 {
		t.Errorf("expected cursor clamped to 2, got %d", m.tasksViewCursor)
	}

	m.tasksViewMoveCursor(-10) // clamp
	if m.tasksViewCursor != 0 {
		t.Errorf("expected cursor clamped to 0, got %d", m.tasksViewCursor)
	}
}

func TestTasksViewSearchToggle(t *testing.T) {
	m := newTasksTestModel()
	m.enterTasksView()
	m.tasksViewLoading = false

	// Activate search
	m.handleTasksViewKey("search", "")
	if !m.tasksViewSearchActive {
		t.Error("expected search to be active")
	}

	// Escape cancels search
	m.handleTasksViewSearchKey("escape")
	if m.tasksViewSearchActive {
		t.Error("expected search to be inactive after escape")
	}
}

func TestTasksViewSearchRawKeyInput(t *testing.T) {
	m := newTasksTestModel()
	m.tasksViewSearchActive = true
	m.tasksViewGraph = bd.NewGraph([]bd.Issue{
		{ID: "t-1", Title: "hello world", Status: "open"},
	})
	m.tasksViewTab = 1

	consumed := m.handleTasksViewSearchRawKey("h")
	if !consumed {
		t.Error("expected key to be consumed")
	}
	if m.tasksViewSearchQuery != "h" {
		t.Errorf("expected query 'h', got %q", m.tasksViewSearchQuery)
	}

	m.handleTasksViewSearchRawKey("backspace")
	if m.tasksViewSearchQuery != "" {
		t.Errorf("expected empty query after backspace, got %q", m.tasksViewSearchQuery)
	}
}

func TestTasksViewSearchInputBarRendered(t *testing.T) {
	m := newTasksTestModel()
	m.tasksViewActive = true
	m.tasksViewSearchActive = true
	m.tasksViewSearchQuery = "bug"
	m.tasksViewGraph = bd.NewGraph([]bd.Issue{
		{ID: "t-1", Title: "fix login bug", Status: "open", Priority: 1},
	})
	m.tasksViewTab = 1
	m.tasksViewRefilter()

	list := m.renderTasksViewList(32, 10)

	if !strings.Contains(list, "/bug█") {
		t.Errorf("expected search input bar with '/bug█', got %q", list)
	}
}

func TestTasksViewSearchInputBarHiddenWhenInactive(t *testing.T) {
	m := newTasksTestModel()
	m.tasksViewActive = true
	m.tasksViewSearchActive = false
	m.tasksViewGraph = bd.NewGraph([]bd.Issue{
		{ID: "t-1", Title: "fix login bug", Status: "open", Priority: 1},
	})
	m.tasksViewTab = 1
	m.tasksViewRefilter()

	list := m.renderTasksViewList(32, 10)

	if strings.Contains(list, "/") && strings.Contains(list, "█") {
		t.Error("expected no search input bar when search inactive")
	}
}

func TestTasksViewSearchResetsCursor(t *testing.T) {
	m := newTasksTestModel()
	m.tasksViewSearchActive = true
	m.tasksViewGraph = bd.NewGraph([]bd.Issue{
		{ID: "t-1", Title: "alpha", Status: "open"},
		{ID: "t-2", Title: "beta", Status: "open"},
		{ID: "t-3", Title: "gamma", Status: "open"},
	})
	m.tasksViewTab = 1
	m.tasksViewRefilter()
	m.tasksViewCursor = 2

	// Typing resets cursor to 0.
	m.handleTasksViewSearchRawKey("a")
	if m.tasksViewCursor != 0 {
		t.Errorf("expected cursor reset to 0 on search input, got %d", m.tasksViewCursor)
	}

	// Set cursor again, backspace should also reset.
	m.tasksViewCursor = 1
	m.handleTasksViewSearchRawKey("backspace")
	if m.tasksViewCursor != 0 {
		t.Errorf("expected cursor reset to 0 on backspace, got %d", m.tasksViewCursor)
	}
}

func TestTasksViewTabSwitchingClearsSearch(t *testing.T) {
	m := newTasksTestModel()
	m.enterTasksView()
	m.tasksViewLoading = false
	m.tasksViewGraph = bd.NewGraph([]bd.Issue{
		{ID: "t-1", Title: "test issue", Status: "open"},
	})
	// Set search state that would be left over from a confirmed search.
	m.tasksViewSearchActive = false
	m.tasksViewSearchQuery = "test"

	// Switch tab should clear leftover search query.
	m.handleTasksViewKey("", "L")
	if m.tasksViewSearchQuery != "" {
		t.Errorf("expected search query cleared on tab switch, got %q", m.tasksViewSearchQuery)
	}
}

func TestTasksViewTabSwitching(t *testing.T) {
	m := newTasksTestModel()
	m.enterTasksView()
	m.tasksViewLoading = false
	m.tasksViewGraph = bd.NewGraph([]bd.Issue{
		{ID: "t-1", Status: "open"},
	})

	// Switch to next tab
	m.handleTasksViewKey("", "L")
	if m.tasksViewTab != 1 {
		t.Errorf("expected tab 1, got %d", m.tasksViewTab)
	}

	// Switch to previous tab
	m.handleTasksViewKey("", "H")
	if m.tasksViewTab != 0 {
		t.Errorf("expected tab 0, got %d", m.tasksViewTab)
	}

	// Can't go below 0
	m.handleTasksViewKey("", "H")
	if m.tasksViewTab != 0 {
		t.Errorf("expected tab still 0, got %d", m.tasksViewTab)
	}
}

func TestTasksViewTabHeaderFormat(t *testing.T) {
	m := newTasksTestModel()
	m.tasksViewGraph = bd.NewGraph([]bd.Issue{
		{ID: "t-1", Status: "open", Priority: 1},
		{ID: "t-2", Status: "in_progress", Priority: 2},
		{ID: "t-3", Status: "blocked", Priority: 1},
	})
	m.tasksViewTab = 0

	header := m.renderTasksViewTabHeader(80)

	// Active tab should use bracket format [Ready 1].
	if !strings.Contains(header, "[Ready 1]") {
		t.Errorf("expected active tab with brackets, got %q", header)
	}
	// Inactive tabs should not use brackets.
	if strings.Contains(header, "[All") {
		t.Errorf("expected inactive tabs without brackets, got %q", header)
	}
	// Should have q:back hint.
	if !strings.Contains(header, "q:back") {
		t.Errorf("expected q:back hint, got %q", header)
	}
	// Should have separator line.
	if !strings.Contains(header, "─") {
		t.Errorf("expected separator line, got %q", header)
	}
}

func TestTasksViewTabCountIncludesClosed(t *testing.T) {
	m := newTasksTestModel()
	m.tasksViewGraph = bd.NewGraph([]bd.Issue{
		{ID: "t-1", Status: "open"},
		{ID: "t-2", Status: "in_progress"},
		{ID: "t-3", Status: "closed"},
	})

	// All tab should include all issues regardless of status.
	allCount := m.tasksViewTabCount(1)
	if allCount != 3 {
		t.Errorf("All tab count: expected 3 (all issues), got %d", allCount)
	}
}

func TestTasksViewTabCountWithSearch(t *testing.T) {
	m := newTasksTestModel()
	m.tasksViewGraph = bd.NewGraph([]bd.Issue{
		{ID: "t-1", Title: "fix login bug", Status: "open"},
		{ID: "t-2", Title: "add dashboard", Status: "open"},
		{ID: "t-3", Title: "fix logout", Status: "in_progress"},
	})
	m.tasksViewSearchQuery = "fix"

	// Ready tab (open) with search "fix" should match only t-1.
	readyCount := m.tasksViewTabCount(0)
	if readyCount != 1 {
		t.Errorf("Ready tab with search: expected 1, got %d", readyCount)
	}

	// All tab with search "fix" should match t-1 and t-3.
	allCount := m.tasksViewTabCount(1)
	if allCount != 2 {
		t.Errorf("All tab with search: expected 2, got %d", allCount)
	}
}

func TestTasksViewTreeModeRendersConnectors(t *testing.T) {
	m := newTasksTestModel()
	m.tasksViewActive = true
	m.tasksViewExpanded = make(map[string]bool)
	issues := []bd.Issue{
		{ID: "root-1", Title: "Root Issue", Status: "open", Priority: 1, IssueType: "epic"},
		{ID: "child-1", Title: "Child One", Status: "in_progress", Priority: 2, IssueType: "task", Parent: "root-1"},
		{ID: "child-2", Title: "Child Two", Status: "blocked", Priority: 1, IssueType: "bug", Parent: "root-1"},
	}
	m.tasksViewGraph = bd.NewGraph(issues)
	m.tasksViewTab = 1
	m.tasksViewFlatMode = false
	m.tasksViewRefilter()

	list := m.renderTasksViewList(50, 20)

	// Should contain tree connectors for children.
	if !strings.Contains(list, "├") {
		t.Errorf("expected ├ tree connector in tree mode, got:\n%s", list)
	}
	if !strings.Contains(list, "└") {
		t.Errorf("expected └ tree connector in tree mode, got:\n%s", list)
	}
}

func TestTasksViewFlatModeNoConnectors(t *testing.T) {
	m := newTasksTestModel()
	m.tasksViewActive = true
	m.tasksViewExpanded = make(map[string]bool)
	issues := []bd.Issue{
		{ID: "root-1", Title: "Root Issue", Status: "open", Priority: 1, IssueType: "epic"},
		{ID: "child-1", Title: "Child One", Status: "in_progress", Priority: 2, IssueType: "task", Parent: "root-1"},
	}
	m.tasksViewGraph = bd.NewGraph(issues)
	m.tasksViewTab = 1
	m.tasksViewFlatMode = true
	m.tasksViewRefilter()

	list := m.renderTasksViewList(50, 20)

	// Flat mode should not have tree connectors.
	if strings.Contains(list, "├") || strings.Contains(list, "└") || strings.Contains(list, "│") {
		t.Errorf("expected no tree connectors in flat mode, got:\n%s", list)
	}
}

func TestTasksViewCollapse(t *testing.T) {
	m := newTasksTestModel()
	m.tasksViewActive = true
	m.tasksViewExpanded = make(map[string]bool)
	issues := []bd.Issue{
		{ID: "root-1", Title: "Root", Status: "open", Priority: 1, IssueType: "epic"},
		{ID: "child-1", Title: "Child", Status: "open", Priority: 2, IssueType: "task", Parent: "root-1"},
	}
	m.tasksViewGraph = bd.NewGraph(issues)
	m.tasksViewTab = 1
	m.tasksViewFlatMode = false
	m.tasksViewRefilter()

	// Initially expanded: should see both root and child.
	if len(m.tasksViewVisibleRows) != 2 {
		t.Fatalf("expected 2 visible rows when expanded, got %d", len(m.tasksViewVisibleRows))
	}

	// Collapse root.
	m.tasksViewCursor = 0
	m.handleTasksViewKey("", " ")

	// Should only see root now.
	if len(m.tasksViewVisibleRows) != 1 {
		t.Errorf("expected 1 visible row after collapse, got %d", len(m.tasksViewVisibleRows))
	}

	// List should show collapse indicator.
	list := m.renderTasksViewList(50, 20)
	if !strings.Contains(list, "▶") {
		t.Errorf("expected ▶ collapse indicator, got:\n%s", list)
	}

	// Expand again.
	m.handleTasksViewKey("", " ")
	if len(m.tasksViewVisibleRows) != 2 {
		t.Errorf("expected 2 visible rows after re-expand, got %d", len(m.tasksViewVisibleRows))
	}
}

func TestTasksViewIsExpanded(t *testing.T) {
	m := newTasksTestModel()
	m.tasksViewExpanded = make(map[string]bool)

	// Default should be expanded.
	if !m.tasksViewIsExpanded("unknown-id") {
		t.Error("expected default to be expanded")
	}

	m.tasksViewExpanded["test-id"] = false
	if m.tasksViewIsExpanded("test-id") {
		t.Error("expected explicitly collapsed to return false")
	}

	m.tasksViewExpanded["test-id"] = true
	if !m.tasksViewIsExpanded("test-id") {
		t.Error("expected explicitly expanded to return true")
	}
}

func TestTasksViewSelectedIssue(t *testing.T) {
	m := newTasksTestModel()

	// No visible rows → nil.
	if m.tasksViewSelectedIssue() != nil {
		t.Error("expected nil when no visible rows")
	}

	issue := &bd.Issue{ID: "test-1", Title: "Test"}
	m.tasksViewVisibleRows = []tasksViewRow{{issue: issue}}
	m.tasksViewCursor = 0

	selected := m.tasksViewSelectedIssue()
	if selected == nil || selected.ID != "test-1" {
		t.Error("expected selected issue to be test-1")
	}
}

func TestTasksViewStatusIconColor(t *testing.T) {
	m := newTasksTestModel()
	tests := []struct {
		status string
		want   string
	}{
		{"open", m.theme.ForegroundDim},
		{"in_progress", m.theme.StatusRunning},
		{"blocked", m.theme.StatusError},
		{"closed", m.theme.StatusSuccess},
		{"unknown", m.theme.Foreground},
	}
	for _, tt := range tests {
		got := m.statusIconColor(tt.status)
		if got != tt.want {
			t.Errorf("statusIconColor(%q) = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestTasksViewIssueTypeColor(t *testing.T) {
	m := newTasksTestModel()
	tests := []struct {
		issueType string
		want      string
	}{
		{"bug", m.theme.StatusError},
		{"feature", m.theme.StatusSuccess},
		{"task", m.theme.Foreground},
		{"epic", "#d33682"},
		{"chore", "#b58900"},
		{"decision", "#2aa198"},
		{"unknown", m.theme.Foreground},
	}
	for _, tt := range tests {
		got := m.issueTypeColor(tt.issueType)
		if got != tt.want {
			t.Errorf("issueTypeColor(%q) = %q, want %q", tt.issueType, got, tt.want)
		}
	}
}

func TestTasksViewListViewportScroll(t *testing.T) {
	m := newTasksTestModel()
	m.tasksViewActive = true
	m.tasksViewExpanded = make(map[string]bool)

	// Create more issues than fit in the viewport.
	var issues []bd.Issue
	for i := 0; i < 20; i++ {
		issues = append(issues, bd.Issue{
			ID: fmt.Sprintf("t-%d", i), Title: fmt.Sprintf("Issue %d", i),
			Status: "open", Priority: 2, IssueType: "task",
		})
	}
	m.tasksViewGraph = bd.NewGraph(issues)
	m.tasksViewTab = 1
	m.tasksViewRefilter()

	// Move cursor past viewport height.
	m.tasksViewCursor = 15

	// Render with small height — should adjust scroll.
	list := m.renderTasksViewList(40, 5)
	_ = list

	// After render, list scroll should have adjusted.
	if m.tasksViewListScroll == 0 {
		t.Error("expected list scroll to adjust when cursor is past viewport")
	}
	if m.tasksViewListScroll > m.tasksViewCursor {
		t.Errorf("list scroll %d should not exceed cursor %d", m.tasksViewListScroll, m.tasksViewCursor)
	}
}

func TestTasksViewRebuildVisiblePreservesCursor(t *testing.T) {
	m := newTasksTestModel()
	m.tasksViewExpanded = make(map[string]bool)
	issues := []bd.Issue{
		{ID: "t-1", Title: "First", Status: "open", Priority: 1, IssueType: "task"},
		{ID: "t-2", Title: "Second", Status: "open", Priority: 2, IssueType: "task"},
		{ID: "t-3", Title: "Third", Status: "open", Priority: 3, IssueType: "task"},
	}
	m.tasksViewGraph = bd.NewGraph(issues)
	m.tasksViewTab = 1
	m.tasksViewRefilter()

	// Set cursor to second issue.
	m.tasksViewCursor = 1

	// Rebuild visible rows — cursor should stay on same issue.
	m.tasksViewRebuildVisible()
	if m.tasksViewCursor != 1 {
		t.Errorf("expected cursor preserved at 1, got %d", m.tasksViewCursor)
	}
	if m.tasksViewVisibleRows[m.tasksViewCursor].issue.ID != "t-2" {
		t.Errorf("expected cursor on t-2, got %s", m.tasksViewVisibleRows[m.tasksViewCursor].issue.ID)
	}
}

func TestTasksViewJumpBottomUsesVisibleRows(t *testing.T) {
	m := newTasksTestModel()
	m.tasksViewExpanded = make(map[string]bool)
	issues := []bd.Issue{
		{ID: "t-1", Title: "First", Status: "open", Priority: 1, IssueType: "task"},
		{ID: "t-2", Title: "Second", Status: "open", Priority: 2, IssueType: "task"},
		{ID: "t-3", Title: "Third", Status: "open", Priority: 3, IssueType: "task"},
	}
	m.tasksViewGraph = bd.NewGraph(issues)
	m.tasksViewTab = 1
	m.tasksViewRefilter()

	m.handleTasksViewKey("jump_bottom", "")
	if m.tasksViewCursor != 2 {
		t.Errorf("expected cursor at last row (2), got %d", m.tasksViewCursor)
	}
}

func TestTasksViewHighlightDepthBehavior(t *testing.T) {
	m := newTasksTestModel()
	m.tasksViewActive = true
	m.tasksViewExpanded = make(map[string]bool)
	issues := []bd.Issue{
		{ID: "t-1", Title: "Test", Status: "open", Priority: 1, IssueType: "task"},
	}
	m.tasksViewGraph = bd.NewGraph(issues)
	m.tasksViewTab = 1
	m.tasksViewRefilter()

	// Verify render doesn't panic at either depth and produces output.
	m.tasksViewDepth = 0
	list0 := m.renderTasksViewList(40, 10)
	if !strings.Contains(list0, "t-1") {
		t.Error("expected issue ID in depth 0 list")
	}

	m.tasksViewDepth = 1
	list1 := m.renderTasksViewList(40, 10)
	if !strings.Contains(list1, "t-1") {
		t.Error("expected issue ID in depth 1 list")
	}
}

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		status string
		icon   string
	}{
		{"open", "●"},
		{"in_progress", "◐"},
		{"blocked", "◇"},
		{"closed", "✓"},
		{"unknown", "◌"},
	}
	for _, tt := range tests {
		got := statusIcon(tt.status)
		if got != tt.icon {
			t.Errorf("statusIcon(%q) = %q, want %q", tt.status, got, tt.icon)
		}
	}
}
