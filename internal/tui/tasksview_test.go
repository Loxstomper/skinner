package tui

import (
	"errors"
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

	// Tab 1: All
	m.tasksViewTab = 1
	m.tasksViewRefilter()
	if len(m.tasksViewFiltered) != 4 {
		t.Errorf("All tab: expected 4 issues, got %d", len(m.tasksViewFiltered))
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
	m.tasksViewFiltered = []*bd.Issue{
		{ID: "t-1"}, {ID: "t-2"}, {ID: "t-3"},
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
