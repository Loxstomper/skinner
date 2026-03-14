package bd

import (
	"encoding/json"
	"testing"
	"time"
)

// Sample bd show --json output with full dependency data.
const sampleShowJSON = `{
  "id": "skinner-zz3",
  "title": "Define Issue struct and JSON parsing",
  "description": "Create internal/bd/issue.go with Issue struct matching bd JSON schema.",
  "status": "in_progress",
  "priority": 2,
  "issue_type": "task",
  "assignee": "Lochie Ashcroft",
  "owner": "17447236+Loxstomper@users.noreply.github.com",
  "created_at": "2026-03-14T12:51:52Z",
  "created_by": "Lochie Ashcroft",
  "updated_at": "2026-03-14T13:02:57Z",
  "dependencies": [
    {
      "id": "skinner-5mq",
      "title": "bd CLI integration package",
      "description": "New internal/bd package.",
      "status": "closed",
      "priority": 1,
      "issue_type": "feature",
      "assignee": "Lochie Ashcroft",
      "owner": "17447236+Loxstomper@users.noreply.github.com",
      "created_at": "2026-03-14T12:49:58Z",
      "created_by": "Lochie Ashcroft",
      "updated_at": "2026-03-14T12:57:34Z",
      "closed_at": "2026-03-14T12:57:34Z",
      "close_reason": "Feature placeholder",
      "dependency_type": "parent-child"
    }
  ],
  "dependents": [
    {
      "id": "skinner-9wn",
      "title": "Build in-memory issue graph",
      "status": "open",
      "priority": 2,
      "issue_type": "task",
      "owner": "17447236+Loxstomper@users.noreply.github.com",
      "created_at": "2026-03-14T12:51:52Z",
      "updated_at": "2026-03-14T12:51:52Z",
      "dependency_type": "blocks"
    },
    {
      "id": "skinner-pid",
      "title": "Implement bd CLI executor",
      "status": "open",
      "priority": 2,
      "issue_type": "task",
      "owner": "17447236+Loxstomper@users.noreply.github.com",
      "created_at": "2026-03-14T12:51:52Z",
      "updated_at": "2026-03-14T12:51:52Z",
      "dependency_type": "blocks"
    }
  ],
  "parent": "skinner-5mq"
}`

// Sample bd list --json output with compact dependencies.
const sampleListJSON = `[
  {
    "id": "skinner-ev3",
    "title": "Tasks view scaffold and lifecycle",
    "description": "Core tasks view overlay.",
    "status": "open",
    "priority": 1,
    "issue_type": "feature",
    "owner": "17447236+Loxstomper@users.noreply.github.com",
    "created_at": "2026-03-14T12:50:03Z",
    "created_by": "Lochie Ashcroft",
    "updated_at": "2026-03-14T12:50:03Z",
    "dependencies": [
      {
        "issue_id": "skinner-ev3",
        "depends_on_id": "skinner-5mq",
        "type": "blocks",
        "created_at": "2026-03-14T22:51:21Z",
        "created_by": "Lochie Ashcroft",
        "metadata": "{}"
      }
    ],
    "dependency_count": 2,
    "dependent_count": 3,
    "comment_count": 0,
    "parent": "skinner-dfe"
  },
  {
    "id": "skinner-a5o",
    "title": "Add tasks view state to Model",
    "status": "open",
    "priority": 2,
    "issue_type": "task",
    "owner": "17447236+Loxstomper@users.noreply.github.com",
    "created_at": "2026-03-14T12:52:45Z",
    "created_by": "Lochie Ashcroft",
    "updated_at": "2026-03-14T12:52:45Z",
    "dependency_count": 0,
    "dependent_count": 1,
    "comment_count": 0,
    "parent": "skinner-ev3"
  }
]`

// Minimal issue with only required fields.
const sampleMinimalJSON = `{
  "id": "skinner-min",
  "title": "Minimal issue",
  "status": "open",
  "priority": 3,
  "issue_type": "task",
  "created_at": "2026-03-14T10:00:00Z",
  "updated_at": "2026-03-14T10:00:00Z"
}`

// Issue with all optional fields populated.
const sampleFullJSON = `{
  "id": "skinner-full",
  "title": "Full issue with all fields",
  "description": "This issue has every optional field populated.",
  "status": "closed",
  "priority": 0,
  "issue_type": "bug",
  "assignee": "Alice",
  "owner": "alice@example.com",
  "created_by": "Bob",
  "labels": ["backend", "v2", "urgent"],
  "parent": "skinner-parent",
  "created_at": "2026-03-14T08:00:00Z",
  "updated_at": "2026-03-14T09:00:00Z",
  "closed_at": "2026-03-14T10:00:00Z",
  "close_reason": "Fixed in commit abc123",
  "gates": [
    {"type": "code-review", "status": "complete", "complete": true},
    {"type": "tests-pass", "status": "pending", "complete": false}
  ],
  "relates_to": ["skinner-rel1", "skinner-rel2"],
  "metadata": {"source": "automated", "version": 2},
  "external_ref": "JIRA-1234",
  "comment_count": 5
}`

func TestParseShowJSON(t *testing.T) {
	var issue Issue
	if err := json.Unmarshal([]byte(sampleShowJSON), &issue); err != nil {
		t.Fatalf("failed to parse bd show JSON: %v", err)
	}

	if issue.ID != "skinner-zz3" {
		t.Errorf("ID = %q, want %q", issue.ID, "skinner-zz3")
	}
	if issue.Title != "Define Issue struct and JSON parsing" {
		t.Errorf("Title = %q, want %q", issue.Title, "Define Issue struct and JSON parsing")
	}
	if issue.Status != "in_progress" {
		t.Errorf("Status = %q, want %q", issue.Status, "in_progress")
	}
	if issue.Priority != 2 {
		t.Errorf("Priority = %d, want %d", issue.Priority, 2)
	}
	if issue.IssueType != "task" {
		t.Errorf("IssueType = %q, want %q", issue.IssueType, "task")
	}
	if issue.Assignee != "Lochie Ashcroft" {
		t.Errorf("Assignee = %q, want %q", issue.Assignee, "Lochie Ashcroft")
	}
	if issue.Parent != "skinner-5mq" {
		t.Errorf("Parent = %q, want %q", issue.Parent, "skinner-5mq")
	}

	// Check dependencies (bd show format with full issue data).
	if len(issue.Dependencies) != 1 {
		t.Fatalf("len(Dependencies) = %d, want 1", len(issue.Dependencies))
	}
	dep := issue.Dependencies[0]
	if dep.ID != "skinner-5mq" {
		t.Errorf("dep.ID = %q, want %q", dep.ID, "skinner-5mq")
	}
	if dep.DependencyType != "parent-child" {
		t.Errorf("dep.DependencyType = %q, want %q", dep.DependencyType, "parent-child")
	}
	if dep.DepType() != "parent-child" {
		t.Errorf("dep.DepType() = %q, want %q", dep.DepType(), "parent-child")
	}
	if dep.Status != "closed" {
		t.Errorf("dep.Status = %q, want %q", dep.Status, "closed")
	}
	if dep.CloseReason != "Feature placeholder" {
		t.Errorf("dep.CloseReason = %q, want %q", dep.CloseReason, "Feature placeholder")
	}

	// Check dependents.
	if len(issue.Dependents) != 2 {
		t.Fatalf("len(Dependents) = %d, want 2", len(issue.Dependents))
	}
	if issue.Dependents[0].ID != "skinner-9wn" {
		t.Errorf("dependent[0].ID = %q, want %q", issue.Dependents[0].ID, "skinner-9wn")
	}
	if issue.Dependents[0].DepType() != "blocks" {
		t.Errorf("dependent[0].DepType() = %q, want %q", issue.Dependents[0].DepType(), "blocks")
	}
	if issue.Dependents[1].ID != "skinner-pid" {
		t.Errorf("dependent[1].ID = %q, want %q", issue.Dependents[1].ID, "skinner-pid")
	}
}

func TestParseListJSON(t *testing.T) {
	var issues []Issue
	if err := json.Unmarshal([]byte(sampleListJSON), &issues); err != nil {
		t.Fatalf("failed to parse bd list JSON: %v", err)
	}

	if len(issues) != 2 {
		t.Fatalf("len(issues) = %d, want 2", len(issues))
	}

	issue := issues[0]
	if issue.ID != "skinner-ev3" {
		t.Errorf("ID = %q, want %q", issue.ID, "skinner-ev3")
	}
	if issue.Priority != 1 {
		t.Errorf("Priority = %d, want %d", issue.Priority, 1)
	}
	if issue.DependencyCount != 2 {
		t.Errorf("DependencyCount = %d, want 2", issue.DependencyCount)
	}
	if issue.DependentCount != 3 {
		t.Errorf("DependentCount = %d, want 3", issue.DependentCount)
	}
	if issue.Parent != "skinner-dfe" {
		t.Errorf("Parent = %q, want %q", issue.Parent, "skinner-dfe")
	}

	// Check compact dependencies (bd list format).
	if len(issue.Dependencies) != 1 {
		t.Fatalf("len(Dependencies) = %d, want 1", len(issue.Dependencies))
	}
	dep := issue.Dependencies[0]
	if dep.IssueID != "skinner-ev3" {
		t.Errorf("dep.IssueID = %q, want %q", dep.IssueID, "skinner-ev3")
	}
	if dep.DependsOnID != "skinner-5mq" {
		t.Errorf("dep.DependsOnID = %q, want %q", dep.DependsOnID, "skinner-5mq")
	}
	if dep.Type != "blocks" {
		t.Errorf("dep.Type = %q, want %q", dep.Type, "blocks")
	}
	if dep.DepType() != "blocks" {
		t.Errorf("dep.DepType() = %q, want %q", dep.DepType(), "blocks")
	}

	// Second issue has no dependencies.
	issue2 := issues[1]
	if issue2.ID != "skinner-a5o" {
		t.Errorf("ID = %q, want %q", issue2.ID, "skinner-a5o")
	}
	if len(issue2.Dependencies) != 0 {
		t.Errorf("len(Dependencies) = %d, want 0", len(issue2.Dependencies))
	}
}

func TestParseMissingOptionalFields(t *testing.T) {
	var issue Issue
	if err := json.Unmarshal([]byte(sampleMinimalJSON), &issue); err != nil {
		t.Fatalf("failed to parse minimal JSON: %v", err)
	}

	if issue.ID != "skinner-min" {
		t.Errorf("ID = %q, want %q", issue.ID, "skinner-min")
	}
	if issue.Assignee != "" {
		t.Errorf("Assignee = %q, want empty", issue.Assignee)
	}
	if issue.Description != "" {
		t.Errorf("Description = %q, want empty", issue.Description)
	}
	if issue.Parent != "" {
		t.Errorf("Parent = %q, want empty", issue.Parent)
	}
	if issue.Labels != nil {
		t.Errorf("Labels = %v, want nil", issue.Labels)
	}
	if len(issue.Dependencies) != 0 {
		t.Errorf("len(Dependencies) = %d, want 0", len(issue.Dependencies))
	}
	if len(issue.Dependents) != 0 {
		t.Errorf("len(Dependents) = %d, want 0", len(issue.Dependents))
	}
	if len(issue.Gates) != 0 {
		t.Errorf("len(Gates) = %d, want 0", len(issue.Gates))
	}
	if !issue.ClosedAt.IsZero() {
		t.Errorf("ClosedAt = %v, want zero", issue.ClosedAt)
	}
	if issue.CloseReason != "" {
		t.Errorf("CloseReason = %q, want empty", issue.CloseReason)
	}
	if issue.ExternalRef != "" {
		t.Errorf("ExternalRef = %q, want empty", issue.ExternalRef)
	}
}

func TestParseFullIssue(t *testing.T) {
	var issue Issue
	if err := json.Unmarshal([]byte(sampleFullJSON), &issue); err != nil {
		t.Fatalf("failed to parse full JSON: %v", err)
	}

	if issue.ID != "skinner-full" {
		t.Errorf("ID = %q, want %q", issue.ID, "skinner-full")
	}
	if issue.Priority != 0 {
		t.Errorf("Priority = %d, want 0", issue.Priority)
	}
	if issue.IssueType != "bug" {
		t.Errorf("IssueType = %q, want %q", issue.IssueType, "bug")
	}

	// Labels.
	if len(issue.Labels) != 3 {
		t.Fatalf("len(Labels) = %d, want 3", len(issue.Labels))
	}
	if issue.Labels[0] != "backend" {
		t.Errorf("Labels[0] = %q, want %q", issue.Labels[0], "backend")
	}

	// Close reason.
	if issue.CloseReason != "Fixed in commit abc123" {
		t.Errorf("CloseReason = %q, want %q", issue.CloseReason, "Fixed in commit abc123")
	}
	if issue.ClosedAt.IsZero() {
		t.Error("ClosedAt should not be zero for closed issue")
	}

	// Gates.
	if len(issue.Gates) != 2 {
		t.Fatalf("len(Gates) = %d, want 2", len(issue.Gates))
	}
	if issue.Gates[0].Type != "code-review" {
		t.Errorf("Gates[0].Type = %q, want %q", issue.Gates[0].Type, "code-review")
	}
	if !issue.Gates[0].Complete {
		t.Error("Gates[0].Complete should be true")
	}
	if issue.Gates[1].Complete {
		t.Error("Gates[1].Complete should be false")
	}

	// RelatesTo.
	if len(issue.RelatesTo) != 2 {
		t.Fatalf("len(RelatesTo) = %d, want 2", len(issue.RelatesTo))
	}

	// Metadata.
	if issue.Metadata == nil {
		t.Fatal("Metadata should not be nil")
	}
	if issue.Metadata["source"] != "automated" {
		t.Errorf("Metadata[source] = %v, want %q", issue.Metadata["source"], "automated")
	}

	// ExternalRef.
	if issue.ExternalRef != "JIRA-1234" {
		t.Errorf("ExternalRef = %q, want %q", issue.ExternalRef, "JIRA-1234")
	}

	// CommentCount.
	if issue.CommentCount != 5 {
		t.Errorf("CommentCount = %d, want 5", issue.CommentCount)
	}
}

func TestParseTimestamps(t *testing.T) {
	var issue Issue
	if err := json.Unmarshal([]byte(sampleShowJSON), &issue); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	expectedCreated := time.Date(2026, 3, 14, 12, 51, 52, 0, time.UTC)
	if !issue.CreatedAt.Equal(expectedCreated) {
		t.Errorf("CreatedAt = %v, want %v", issue.CreatedAt, expectedCreated)
	}

	expectedUpdated := time.Date(2026, 3, 14, 13, 2, 57, 0, time.UTC)
	if !issue.UpdatedAt.Equal(expectedUpdated) {
		t.Errorf("UpdatedAt = %v, want %v", issue.UpdatedAt, expectedUpdated)
	}

	// ClosedAt should be zero (issue not closed).
	if !issue.ClosedAt.IsZero() {
		t.Errorf("ClosedAt = %v, want zero", issue.ClosedAt)
	}
}

func TestDependencyDepType(t *testing.T) {
	tests := []struct {
		name string
		dep  Dependency
		want string
	}{
		{
			name: "show format with dependency_type",
			dep:  Dependency{DependencyType: "blocks"},
			want: "blocks",
		},
		{
			name: "list format with type",
			dep:  Dependency{Type: "parent-child"},
			want: "parent-child",
		},
		{
			name: "both set prefers dependency_type",
			dep:  Dependency{DependencyType: "blocks", Type: "parent-child"},
			want: "blocks",
		},
		{
			name: "neither set",
			dep:  Dependency{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.dep.DepType(); got != tt.want {
				t.Errorf("DepType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIssueRoundTrip(t *testing.T) {
	var original Issue
	if err := json.Unmarshal([]byte(sampleFullJSON), &original); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var roundtripped Issue
	if err := json.Unmarshal(data, &roundtripped); err != nil {
		t.Fatalf("unmarshal roundtripped: %v", err)
	}

	if original.ID != roundtripped.ID {
		t.Errorf("ID mismatch: %q vs %q", original.ID, roundtripped.ID)
	}
	if original.Priority != roundtripped.Priority {
		t.Errorf("Priority mismatch: %d vs %d", original.Priority, roundtripped.Priority)
	}
	if len(original.Labels) != len(roundtripped.Labels) {
		t.Errorf("Labels length mismatch: %d vs %d", len(original.Labels), len(roundtripped.Labels))
	}
	if len(original.Gates) != len(roundtripped.Gates) {
		t.Errorf("Gates length mismatch: %d vs %d", len(original.Gates), len(roundtripped.Gates))
	}
}

func TestParseDependencyWithClosedAt(t *testing.T) {
	// Dependency in bd show format with closed_at timestamp.
	depJSON := `{
		"id": "skinner-closed",
		"title": "Closed dependency",
		"status": "closed",
		"priority": 1,
		"issue_type": "task",
		"created_at": "2026-03-14T08:00:00Z",
		"updated_at": "2026-03-14T09:00:00Z",
		"closed_at": "2026-03-14T10:00:00Z",
		"close_reason": "Done",
		"dependency_type": "blocks"
	}`

	var dep Dependency
	if err := json.Unmarshal([]byte(depJSON), &dep); err != nil {
		t.Fatalf("failed to parse dependency JSON: %v", err)
	}

	if dep.ID != "skinner-closed" {
		t.Errorf("ID = %q, want %q", dep.ID, "skinner-closed")
	}
	if dep.ClosedAt.IsZero() {
		t.Error("ClosedAt should not be zero")
	}
	if dep.CloseReason != "Done" {
		t.Errorf("CloseReason = %q, want %q", dep.CloseReason, "Done")
	}
	if dep.DepType() != "blocks" {
		t.Errorf("DepType() = %q, want %q", dep.DepType(), "blocks")
	}
}
