package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/loxstomper/skinner/internal/bd"
)

func setupDetailModel(issues []bd.Issue) *Model {
	m := newTasksTestModel()
	m.tasksViewActive = true
	m.tasksViewLoading = false
	g := bd.NewGraph(issues)
	m.tasksViewGraph = g
	m.tasksViewTab = 1 // All tab (excludes closed)
	m.tasksViewRefilter()
	return m
}

func TestDetailRenderEmptyFiltered(t *testing.T) {
	m := newTasksTestModel()
	m.tasksViewFiltered = nil
	m.tasksViewVisibleRows = nil
	result := m.tasksViewRenderDetail(80, 30)
	// Should return empty lines, not panic.
	if len(result) == 0 {
		t.Error("expected non-empty output for empty filtered list")
	}
}

func TestDetailRenderPriorityBadge(t *testing.T) {
	tests := []struct {
		priority int
		badge    string
	}{
		{0, "[P0]"},
		{1, "[P1]"},
		{2, "[P2]"},
		{3, "[P3]"},
		{4, "[P4]"},
	}

	for _, tt := range tests {
		issues := []bd.Issue{
			{ID: "test-1", Title: "Test issue", Status: "open", Priority: tt.priority,
				IssueType: "task", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		}
		m := setupDetailModel(issues)
		result := m.tasksViewRenderDetail(80, 30)
		if !strings.Contains(result, tt.badge) {
			t.Errorf("priority %d: expected badge %q in output", tt.priority, tt.badge)
		}
	}
}

func TestDetailRenderTitle(t *testing.T) {
	issues := []bd.Issue{
		{ID: "test-1", Title: "My Test Title", Status: "open", Priority: 2,
			IssueType: "task", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	m := setupDetailModel(issues)
	result := m.tasksViewRenderDetail(80, 30)
	if !strings.Contains(result, "My Test Title") {
		t.Error("expected title in output")
	}
}

func TestDetailRenderSeparator(t *testing.T) {
	issues := []bd.Issue{
		{ID: "test-1", Title: "Test", Status: "open", Priority: 2,
			IssueType: "task", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	m := setupDetailModel(issues)
	result := m.tasksViewRenderDetail(80, 30)
	if !strings.Contains(result, "─") {
		t.Error("expected horizontal rule separator in output")
	}
}

func TestDetailRenderMetaLine(t *testing.T) {
	issues := []bd.Issue{
		{ID: "test-1", Title: "Test", Status: "in_progress", Priority: 1,
			IssueType: "feature", Assignee: "Alice", Parent: "parent-1",
			CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	m := setupDetailModel(issues)
	result := m.tasksViewRenderDetail(80, 30)
	if !strings.Contains(result, "feature") {
		t.Error("expected issue type in meta line")
	}
	if !strings.Contains(result, "in_progress") {
		t.Error("expected status in meta line")
	}
	if !strings.Contains(result, "assigned: Alice") {
		t.Error("expected assignee in meta line")
	}
	if !strings.Contains(result, "parent: parent-1") {
		t.Error("expected parent in meta line")
	}
}

func TestDetailRenderLabels(t *testing.T) {
	issues := []bd.Issue{
		{ID: "test-1", Title: "Test", Status: "open", Priority: 2,
			IssueType: "task", Labels: []string{"backend", "v2"},
			CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	m := setupDetailModel(issues)
	result := m.tasksViewRenderDetail(80, 30)
	if !strings.Contains(result, "Labels: backend, v2") {
		t.Errorf("expected labels line, got:\n%s", result)
	}
}

func TestDetailRenderLabelsHiddenWhenEmpty(t *testing.T) {
	issues := []bd.Issue{
		{ID: "test-1", Title: "Test", Status: "open", Priority: 2,
			IssueType: "task", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	m := setupDetailModel(issues)
	result := m.tasksViewRenderDetail(80, 30)
	if strings.Contains(result, "Labels:") {
		t.Error("expected no labels line when labels are empty")
	}
}

func TestDetailRenderDescription(t *testing.T) {
	issues := []bd.Issue{
		{ID: "test-1", Title: "Test", Status: "open", Priority: 2,
			IssueType: "task", Description: "A detailed description of the task.",
			CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	m := setupDetailModel(issues)
	result := m.tasksViewRenderDetail(80, 30)
	if !strings.Contains(result, "A detailed description of the task.") {
		t.Error("expected description in output")
	}
}

func TestDetailRenderDescriptionSeparatorAbsent(t *testing.T) {
	// With description: should have a separator before it.
	withDesc := []bd.Issue{
		{ID: "test-1", Title: "Test", Status: "open", Priority: 2,
			IssueType: "task", Description: "Some text",
			CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	mWith := setupDetailModel(withDesc)
	resultWith := mWith.tasksViewRenderDetail(80, 30)

	// Without description: should have fewer separators.
	noDesc := []bd.Issue{
		{ID: "test-1", Title: "Test", Status: "open", Priority: 2,
			IssueType: "task", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	mNo := setupDetailModel(noDesc)
	resultNo := mNo.tasksViewRenderDetail(80, 30)

	// The version with description should be longer (has extra separator + desc text).
	if len(resultNo) >= len(resultWith) {
		t.Error("expected output with description to be longer than without")
	}
}

func TestDetailRenderBlockingDeps(t *testing.T) {
	issues := []bd.Issue{
		{
			ID: "child-1", Title: "Child task", Status: "blocked", Priority: 1,
			IssueType: "task",
			Dependencies: []bd.Dependency{
				{DependsOnID: "blocker-1", Type: "blocks"},
			},
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
		{
			ID: "blocker-1", Title: "Blocker task", Status: "in_progress", Priority: 1,
			IssueType: "task",
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
	}
	m := setupDetailModel(issues)
	m.tasksViewCursor = 0 // child-1

	result := m.tasksViewRenderDetail(80, 40)
	if !strings.Contains(result, "Blocked by") {
		t.Error("expected 'Blocked by' header")
	}
	if !strings.Contains(result, "blocker-1") {
		t.Error("expected blocker issue ID in dep tree")
	}
	if !strings.Contains(result, "Blocker task") {
		t.Error("expected blocker issue title in dep tree")
	}
}

func TestDetailRenderBlockingDepsHiddenWhenNone(t *testing.T) {
	issues := []bd.Issue{
		{ID: "test-1", Title: "Test", Status: "open", Priority: 2,
			IssueType: "task", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	m := setupDetailModel(issues)
	result := m.tasksViewRenderDetail(80, 30)
	if strings.Contains(result, "Blocked by") {
		t.Error("expected no 'Blocked by' when there are no blocking deps")
	}
}

func TestDetailRenderDependents(t *testing.T) {
	issues := []bd.Issue{
		{
			ID: "blocker-1", Title: "Blocker task", Status: "in_progress", Priority: 1,
			IssueType: "task",
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
		{
			ID: "child-1", Title: "Child task", Status: "blocked", Priority: 1,
			IssueType: "task",
			Dependencies: []bd.Dependency{
				{DependsOnID: "blocker-1", Type: "blocks"},
			},
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
	}
	m := setupDetailModel(issues)
	m.tasksViewCursor = 0 // blocker-1

	result := m.tasksViewRenderDetail(80, 40)
	if !strings.Contains(result, "Blocks") {
		t.Error("expected 'Blocks' header for dependents")
	}
	if !strings.Contains(result, "child-1") {
		t.Error("expected dependent issue ID")
	}
}

func TestDetailRenderDependentsHiddenWhenNone(t *testing.T) {
	issues := []bd.Issue{
		{ID: "test-1", Title: "Test", Status: "open", Priority: 2,
			IssueType: "task", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	m := setupDetailModel(issues)
	result := m.tasksViewRenderDetail(80, 30)
	if strings.Contains(result, "Blocks") {
		t.Error("expected no 'Blocks' when there are no dependents")
	}
}

func TestDetailRenderRelated(t *testing.T) {
	issues := []bd.Issue{
		{
			ID: "test-1", Title: "Test task", Status: "open", Priority: 2,
			IssueType: "task",
			Dependencies: []bd.Dependency{
				{DependsOnID: "related-1", Type: "discovered-from"},
			},
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
		{
			ID: "related-1", Title: "Related task", Status: "open", Priority: 2,
			IssueType: "feature",
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
	}
	m := setupDetailModel(issues)
	m.tasksViewCursor = 0

	result := m.tasksViewRenderDetail(80, 40)
	if !strings.Contains(result, "Related") {
		t.Error("expected 'Related' header")
	}
	if !strings.Contains(result, "discovered-from") {
		t.Error("expected dependency type in related section")
	}
	if !strings.Contains(result, "related-1") {
		t.Error("expected related issue ID")
	}
}

func TestDetailRenderRelatedHiddenWhenNone(t *testing.T) {
	issues := []bd.Issue{
		{ID: "test-1", Title: "Test", Status: "open", Priority: 2,
			IssueType: "task", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	m := setupDetailModel(issues)
	result := m.tasksViewRenderDetail(80, 30)
	if strings.Contains(result, "Related") {
		t.Error("expected no 'Related' when there are no non-blocking deps")
	}
}

func TestDetailRenderGates(t *testing.T) {
	issues := []bd.Issue{
		{
			ID: "test-1", Title: "Test", Status: "open", Priority: 2,
			IssueType: "task",
			Gates: []bd.Gate{
				{Type: "code-review", Status: "pending", Complete: false},
				{Type: "tests-pass", Status: "complete", Complete: true},
			},
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
	}
	m := setupDetailModel(issues)
	result := m.tasksViewRenderDetail(80, 30)
	if !strings.Contains(result, "Gates") {
		t.Error("expected 'Gates' header")
	}
	if !strings.Contains(result, "☐") {
		t.Error("expected unchecked gate icon")
	}
	if !strings.Contains(result, "☑") {
		t.Error("expected checked gate icon")
	}
	if !strings.Contains(result, "code-review") {
		t.Error("expected gate type")
	}
}

func TestDetailRenderGatesHiddenWhenNone(t *testing.T) {
	issues := []bd.Issue{
		{ID: "test-1", Title: "Test", Status: "open", Priority: 2,
			IssueType: "task", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	m := setupDetailModel(issues)
	result := m.tasksViewRenderDetail(80, 30)
	if strings.Contains(result, "Gates") {
		t.Error("expected no 'Gates' when there are no gates")
	}
}

func TestDetailRenderTimestamps(t *testing.T) {
	created := time.Date(2026, 3, 14, 12, 49, 0, 0, time.UTC)
	updated := time.Date(2026, 3, 14, 12, 57, 0, 0, time.UTC)
	issues := []bd.Issue{
		{ID: "test-1", Title: "Test", Status: "open", Priority: 2,
			IssueType: "task", CreatedAt: created, UpdatedAt: updated},
	}
	m := setupDetailModel(issues)
	result := m.tasksViewRenderDetail(80, 30)
	if !strings.Contains(result, "Created") {
		t.Error("expected 'Created' timestamp label")
	}
	if !strings.Contains(result, "2026-03-14 12:49") {
		t.Error("expected created timestamp value")
	}
	if !strings.Contains(result, "Updated") {
		t.Error("expected 'Updated' timestamp label")
	}
	if !strings.Contains(result, "2026-03-14 12:57") {
		t.Error("expected updated timestamp value")
	}
}

func TestDetailRenderClosedTimestamp(t *testing.T) {
	closed := time.Date(2026, 3, 14, 13, 0, 0, 0, time.UTC)
	issues := []bd.Issue{
		{ID: "test-1", Title: "Test", Status: "closed", Priority: 2,
			IssueType: "task",
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
			ClosedAt: closed, CloseReason: "All tasks completed"},
	}
	m := setupDetailModel(issues)
	m.tasksViewTab = 1 // All tab excludes closed; set up manually.
	issue := m.tasksViewGraph.Issues[0]
	m.tasksViewFiltered = []*bd.Issue{issue}
	m.tasksViewVisibleRows = []tasksViewRow{{issue: issue, depth: 0}}
	m.tasksViewCursor = 0

	result := m.tasksViewRenderDetail(80, 30)
	if !strings.Contains(result, "Closed") {
		t.Error("expected 'Closed' timestamp label")
	}
	if !strings.Contains(result, "2026-03-14 13:00") {
		t.Error("expected closed timestamp value")
	}
	if !strings.Contains(result, "All tasks completed") {
		t.Error("expected close reason in output")
	}
}

func TestDetailRenderScrollDepth1(t *testing.T) {
	issues := []bd.Issue{
		{ID: "test-1", Title: "Test", Status: "open", Priority: 2,
			IssueType: "task",
			Description: "Line 1\nLine 2\nLine 3\nLine 4\nLine 5",
			CreatedAt:   time.Now(), UpdatedAt: time.Now()},
	}
	m := setupDetailModel(issues)
	m.tasksViewDepth = 1
	m.tasksViewScroll = 2

	result := m.tasksViewRenderDetail(80, 5)
	// The title should be scrolled past (it's in the first few lines).
	// Exact content depends on rendering, but scroll should reduce visible content.
	lines := strings.Split(result, "\n")
	if len(lines) > 6 { // 5 height + possible trailing
		t.Errorf("expected at most ~5 lines after scroll, got %d", len(lines))
	}
}

func TestDetailRenderScrollClamp(t *testing.T) {
	issues := []bd.Issue{
		{ID: "test-1", Title: "Test", Status: "open", Priority: 2,
			IssueType: "task", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	m := setupDetailModel(issues)
	m.tasksViewDepth = 1
	m.tasksViewScroll = 9999 // Way past end.

	// Should not panic, scroll should be clamped.
	result := m.tasksViewRenderDetail(80, 30)
	if result == "" {
		t.Error("expected non-empty output after scroll clamp")
	}
}

func TestDetailRenderCycleDetection(t *testing.T) {
	// Create a cycle: A blocks B, B blocks A.
	issues := []bd.Issue{
		{
			ID: "a", Title: "Issue A", Status: "blocked", Priority: 1,
			IssueType: "task",
			Dependencies: []bd.Dependency{
				{DependsOnID: "b", Type: "blocks"},
			},
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
		{
			ID: "b", Title: "Issue B", Status: "blocked", Priority: 1,
			IssueType: "task",
			Dependencies: []bd.Dependency{
				{DependsOnID: "a", Type: "blocks"},
			},
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
	}
	m := setupDetailModel(issues)
	m.tasksViewCursor = 0 // Issue A

	result := m.tasksViewRenderDetail(80, 40)
	if !strings.Contains(result, "(cycle)") {
		t.Error("expected (cycle) marker in dependency tree")
	}
}

func TestDetailRenderTransitiveDeps(t *testing.T) {
	// A blocked by B, B blocked by C.
	issues := []bd.Issue{
		{
			ID: "a", Title: "Issue A", Status: "blocked", Priority: 1,
			IssueType: "task",
			Dependencies: []bd.Dependency{
				{DependsOnID: "b", Type: "blocks"},
			},
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
		{
			ID: "b", Title: "Issue B", Status: "blocked", Priority: 1,
			IssueType: "task",
			Dependencies: []bd.Dependency{
				{DependsOnID: "c", Type: "blocks"},
			},
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
		{
			ID: "c", Title: "Issue C", Status: "open", Priority: 2,
			IssueType: "task",
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
	}
	m := setupDetailModel(issues)
	m.tasksViewCursor = 0 // Issue A

	result := m.tasksViewRenderDetail(80, 40)
	if !strings.Contains(result, "Blocked by") {
		t.Error("expected 'Blocked by' header")
	}
	// Should show B (direct) and C (transitive).
	if !strings.Contains(result, "b") {
		t.Error("expected direct blocker 'b' in tree")
	}
	if !strings.Contains(result, "c") {
		t.Error("expected transitive blocker 'c' in tree")
	}
}

func TestDetailRenderTreeChars(t *testing.T) {
	// A blocked by B and C (two direct blockers).
	issues := []bd.Issue{
		{
			ID: "a", Title: "Issue A", Status: "blocked", Priority: 1,
			IssueType: "task",
			Dependencies: []bd.Dependency{
				{DependsOnID: "b", Type: "blocks"},
				{DependsOnID: "c", Type: "blocks"},
			},
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
		{
			ID: "b", Title: "Issue B", Status: "open", Priority: 2,
			IssueType: "task", CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
		{
			ID: "c", Title: "Issue C", Status: "open", Priority: 2,
			IssueType: "task", CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
	}
	m := setupDetailModel(issues)
	m.tasksViewCursor = 0

	result := m.tasksViewRenderDetail(80, 40)
	// Should have tree-drawing characters.
	if !strings.Contains(result, "├── ") {
		t.Error("expected ├── tree branch character")
	}
	if !strings.Contains(result, "└── ") {
		t.Error("expected └── tree last-branch character")
	}
}

func TestPriorityStyle(t *testing.T) {
	m := newTasksTestModel()
	// Just verify it doesn't panic for all priorities.
	for p := 0; p <= 4; p++ {
		style := m.priorityStyle(p)
		_ = style.Render("test")
	}
}

func TestRenderDepTreeEmpty(t *testing.T) {
	m := newTasksTestModel()
	dimStyle := m.priorityStyle(3) // Just need any style.
	result := renderDepTree(nil, dimStyle, "├── ", "└── ", "│   ", "    ")
	if result != "" {
		t.Errorf("expected empty string for nil nodes, got %q", result)
	}
}
