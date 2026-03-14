package bd

import (
	"testing"
	"time"
)

func makeIssues() []Issue {
	now := time.Now()
	return []Issue{
		{
			ID:        "root-1",
			Title:     "Root feature",
			Status:    "open",
			Priority:  1,
			IssueType: "feature",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "child-1",
			Title:     "Child task one",
			Status:    "in_progress",
			Priority:  2,
			IssueType: "task",
			Parent:    "root-1",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "child-2",
			Title:     "Child task two",
			Status:    "open",
			Priority:  2,
			IssueType: "task",
			Parent:    "root-1",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "root-2",
			Title:     "Independent bug",
			Status:    "blocked",
			Priority:  0,
			IssueType: "bug",
			CreatedAt: now,
			UpdatedAt: now,
			Dependencies: []Dependency{
				{DependsOnID: "child-1", Type: "blocks", IssueID: "root-2"},
			},
		},
		{
			ID:        "closed-1",
			Title:     "Closed chore",
			Status:    "closed",
			Priority:  3,
			IssueType: "chore",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

func TestNewGraphIndexes(t *testing.T) {
	g := NewGraph(makeIssues())

	if len(g.Issues) != 5 {
		t.Errorf("len(Issues) = %d, want 5", len(g.Issues))
	}

	if len(g.ByID) != 5 {
		t.Errorf("len(ByID) = %d, want 5", len(g.ByID))
	}

	if g.ByID["root-1"] == nil {
		t.Error("ByID[root-1] is nil")
	}
	if g.ByID["child-1"] == nil {
		t.Error("ByID[child-1] is nil")
	}
}

func TestNewGraphRoots(t *testing.T) {
	g := NewGraph(makeIssues())

	if len(g.Roots) != 3 {
		t.Fatalf("len(Roots) = %d, want 3 (root-1, root-2, closed-1)", len(g.Roots))
	}

	rootIDs := make(map[string]bool)
	for _, r := range g.Roots {
		rootIDs[r.ID] = true
	}
	for _, id := range []string{"root-1", "root-2", "closed-1"} {
		if !rootIDs[id] {
			t.Errorf("expected %s in roots", id)
		}
	}
}

func TestNewGraphChildren(t *testing.T) {
	g := NewGraph(makeIssues())

	children := g.Children["root-1"]
	if len(children) != 2 {
		t.Fatalf("len(Children[root-1]) = %d, want 2", len(children))
	}

	childIDs := make(map[string]bool)
	for _, c := range children {
		childIDs[c.ID] = true
	}
	if !childIDs["child-1"] || !childIDs["child-2"] {
		t.Errorf("expected child-1 and child-2 as children of root-1, got %v", childIDs)
	}

	// root-2 has no children.
	if len(g.Children["root-2"]) != 0 {
		t.Errorf("len(Children[root-2]) = %d, want 0", len(g.Children["root-2"]))
	}
}

func TestWalkBlockingDeps(t *testing.T) {
	g := NewGraph(makeIssues())

	// root-2 is blocked by child-1.
	deps := g.WalkBlockingDeps("root-2", 3)
	if len(deps) != 1 {
		t.Fatalf("len(deps) = %d, want 1", len(deps))
	}
	if deps[0].Issue.ID != "child-1" {
		t.Errorf("dep.Issue.ID = %q, want child-1", deps[0].Issue.ID)
	}
	if deps[0].Depth != 1 {
		t.Errorf("dep.Depth = %d, want 1", deps[0].Depth)
	}
}

func TestWalkBlockingDependents(t *testing.T) {
	g := NewGraph(makeIssues())

	// child-1 blocks root-2.
	deps := g.WalkBlockingDependents("child-1", 3)
	if len(deps) != 1 {
		t.Fatalf("len(deps) = %d, want 1", len(deps))
	}
	if deps[0].Issue.ID != "root-2" {
		t.Errorf("dep.Issue.ID = %q, want root-2", deps[0].Issue.ID)
	}
}

func TestWalkBlockingDepsTransitive(t *testing.T) {
	now := time.Now()
	issues := []Issue{
		{ID: "a", Title: "A", Status: "open", CreatedAt: now, UpdatedAt: now,
			Dependencies: []Dependency{{DependsOnID: "b", Type: "blocks", IssueID: "a"}}},
		{ID: "b", Title: "B", Status: "open", CreatedAt: now, UpdatedAt: now,
			Dependencies: []Dependency{{DependsOnID: "c", Type: "blocks", IssueID: "b"}}},
		{ID: "c", Title: "C", Status: "open", CreatedAt: now, UpdatedAt: now},
	}
	g := NewGraph(issues)

	deps := g.WalkBlockingDeps("a", 3)
	if len(deps) != 1 {
		t.Fatalf("len(deps) = %d, want 1", len(deps))
	}
	if deps[0].Issue.ID != "b" {
		t.Errorf("dep.Issue.ID = %q, want b", deps[0].Issue.ID)
	}
	// b should have c as a child dep.
	if len(deps[0].Children) != 1 {
		t.Fatalf("len(deps[0].Children) = %d, want 1", len(deps[0].Children))
	}
	if deps[0].Children[0].Issue.ID != "c" {
		t.Errorf("deps[0].Children[0].Issue.ID = %q, want c", deps[0].Children[0].Issue.ID)
	}
	if deps[0].Children[0].Depth != 2 {
		t.Errorf("Depth = %d, want 2", deps[0].Children[0].Depth)
	}
}

func TestWalkBlockingDepsDepthCap(t *testing.T) {
	now := time.Now()
	issues := []Issue{
		{ID: "a", Title: "A", Status: "open", CreatedAt: now, UpdatedAt: now,
			Dependencies: []Dependency{{DependsOnID: "b", Type: "blocks", IssueID: "a"}}},
		{ID: "b", Title: "B", Status: "open", CreatedAt: now, UpdatedAt: now,
			Dependencies: []Dependency{{DependsOnID: "c", Type: "blocks", IssueID: "b"}}},
		{ID: "c", Title: "C", Status: "open", CreatedAt: now, UpdatedAt: now,
			Dependencies: []Dependency{{DependsOnID: "d", Type: "blocks", IssueID: "c"}}},
		{ID: "d", Title: "D", Status: "open", CreatedAt: now, UpdatedAt: now},
	}
	g := NewGraph(issues)

	// With maxDepth=2, should only go a->b->c, not to d.
	deps := g.WalkBlockingDeps("a", 2)
	if len(deps) != 1 {
		t.Fatalf("len(deps) = %d, want 1", len(deps))
	}
	if len(deps[0].Children) != 1 {
		t.Fatalf("len(b.Children) = %d, want 1", len(deps[0].Children))
	}
	// c should have no children because depth cap reached.
	if len(deps[0].Children[0].Children) != 0 {
		t.Errorf("c should have no children at maxDepth=2, got %d", len(deps[0].Children[0].Children))
	}
}

func TestWalkBlockingDepsCycle(t *testing.T) {
	now := time.Now()
	issues := []Issue{
		{ID: "a", Title: "A", Status: "open", CreatedAt: now, UpdatedAt: now,
			Dependencies: []Dependency{{DependsOnID: "b", Type: "blocks", IssueID: "a"}}},
		{ID: "b", Title: "B", Status: "open", CreatedAt: now, UpdatedAt: now,
			Dependencies: []Dependency{{DependsOnID: "a", Type: "blocks", IssueID: "b"}}},
	}
	g := NewGraph(issues)

	deps := g.WalkBlockingDeps("a", 3)
	if len(deps) != 1 {
		t.Fatalf("len(deps) = %d, want 1", len(deps))
	}
	if deps[0].Issue.ID != "b" {
		t.Errorf("dep.Issue.ID = %q, want b", deps[0].Issue.ID)
	}
	// b's dependency on a should be marked as a cycle.
	if len(deps[0].Children) != 1 {
		t.Fatalf("len(b.Children) = %d, want 1", len(deps[0].Children))
	}
	if !deps[0].Children[0].Cycle {
		t.Error("expected cycle=true for a in b's children")
	}
}

func TestWalkBlockingDepsNoDeps(t *testing.T) {
	g := NewGraph(makeIssues())

	// root-1 has no blocking deps.
	deps := g.WalkBlockingDeps("root-1", 3)
	if len(deps) != 0 {
		t.Errorf("len(deps) = %d, want 0", len(deps))
	}
}

func TestNonBlockingDeps(t *testing.T) {
	now := time.Now()
	issues := []Issue{
		{ID: "a", Title: "A", Status: "open", CreatedAt: now, UpdatedAt: now,
			Dependencies: []Dependency{
				{DependsOnID: "b", Type: "blocks", IssueID: "a"},
				{DependsOnID: "c", Type: "discovered-from", IssueID: "a"},
				{DependsOnID: "d", Type: "parent-child", IssueID: "a"},
			}},
		{ID: "b", Title: "B", Status: "open", CreatedAt: now, UpdatedAt: now},
		{ID: "c", Title: "C", Status: "open", CreatedAt: now, UpdatedAt: now},
		{ID: "d", Title: "D", Status: "open", CreatedAt: now, UpdatedAt: now},
	}
	g := NewGraph(issues)

	nonBlocking := g.NonBlockingDeps("a")
	if len(nonBlocking) != 1 {
		t.Fatalf("len(nonBlocking) = %d, want 1 (discovered-from only)", len(nonBlocking))
	}
	if nonBlocking[0].DepType() != "discovered-from" {
		t.Errorf("type = %q, want discovered-from", nonBlocking[0].DepType())
	}
}

func TestFilterByStatus(t *testing.T) {
	g := NewGraph(makeIssues())

	open := g.FilterByStatus("open")
	if len(open) != 2 {
		t.Errorf("len(open) = %d, want 2", len(open))
	}

	blocked := g.FilterByStatus("blocked")
	if len(blocked) != 1 {
		t.Errorf("len(blocked) = %d, want 1", len(blocked))
	}

	closed := g.FilterByStatus("closed")
	if len(closed) != 1 {
		t.Errorf("len(closed) = %d, want 1", len(closed))
	}

	none := g.FilterByStatus("nonexistent")
	if len(none) != 0 {
		t.Errorf("len(nonexistent) = %d, want 0", len(none))
	}
}

func TestFuzzySearch(t *testing.T) {
	g := NewGraph(makeIssues())

	tests := []struct {
		query string
		want  int
	}{
		{"", 5},           // Empty query returns all.
		{"Root", 2},       // Matches "Root feature" and "root-2" ID (Independent bug with root-2 ID).
		{"task", 2},       // Matches "Child task one" and "Child task two".
		{"bug", 1},        // Matches "Independent bug".
		{"child", 2},      // Matches child-1 and child-2 titles.
		{"CHORE", 1},      // Case-insensitive match.
		{"nonexistent", 0}, // No matches.
		{"root-1", 1},     // Match by ID.
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			results := g.FuzzySearch(tt.query)
			if len(results) != tt.want {
				ids := make([]string, len(results))
				for i, r := range results {
					ids[i] = r.ID
				}
				t.Errorf("FuzzySearch(%q) returned %d results %v, want %d", tt.query, len(results), ids, tt.want)
			}
		})
	}
}

func TestEmptyGraph(t *testing.T) {
	g := NewGraph(nil)

	if len(g.Issues) != 0 {
		t.Errorf("len(Issues) = %d, want 0", len(g.Issues))
	}
	if len(g.Roots) != 0 {
		t.Errorf("len(Roots) = %d, want 0", len(g.Roots))
	}

	deps := g.WalkBlockingDeps("nonexistent", 3)
	if len(deps) != 0 {
		t.Errorf("len(deps) = %d, want 0", len(deps))
	}

	results := g.FuzzySearch("anything")
	if len(results) != 0 {
		t.Errorf("len(results) = %d, want 0", len(results))
	}

	filtered := g.FilterByStatus("open")
	if len(filtered) != 0 {
		t.Errorf("len(filtered) = %d, want 0", len(filtered))
	}
}
