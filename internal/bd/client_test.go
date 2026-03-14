package bd

import (
	"context"
	"errors"
	"os/exec"
	"testing"
)

func hasBd() bool {
	_, err := exec.LookPath("bd")
	return err == nil
}

func TestClientList(t *testing.T) {
	if !hasBd() {
		t.Skip("bd binary not available")
	}

	c := NewClient()
	issues, err := c.List(context.Background(), ListOpts{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}

	if len(issues) == 0 {
		t.Skip("no issues in project, skipping field validation")
	}

	// Verify basic fields are populated.
	issue := issues[0]
	if issue.ID == "" {
		t.Error("issue.ID is empty")
	}
	if issue.Title == "" {
		t.Error("issue.Title is empty")
	}
	if issue.Status == "" {
		t.Error("issue.Status is empty")
	}
	if issue.CreatedAt.IsZero() {
		t.Error("issue.CreatedAt is zero")
	}
}

func TestClientListWithStatusFilter(t *testing.T) {
	if !hasBd() {
		t.Skip("bd binary not available")
	}

	c := NewClient()
	issues, err := c.List(context.Background(), ListOpts{Status: "in_progress"})
	if err != nil {
		t.Fatalf("List(status=in_progress) error: %v", err)
	}

	for _, issue := range issues {
		if issue.Status != "in_progress" {
			t.Errorf("expected status in_progress, got %q for %s", issue.Status, issue.ID)
		}
	}
}

func TestClientReady(t *testing.T) {
	if !hasBd() {
		t.Skip("bd binary not available")
	}

	c := NewClient()
	issues, err := c.Ready(context.Background())
	if err != nil {
		t.Fatalf("Ready() error: %v", err)
	}

	// Ready issues should all be open status.
	for _, issue := range issues {
		if issue.Status != "open" {
			t.Errorf("ready issue %s has status %q, expected open", issue.ID, issue.Status)
		}
	}
}

func TestClientShow(t *testing.T) {
	if !hasBd() {
		t.Skip("bd binary not available")
	}

	// First get an issue ID from list.
	c := NewClient()
	issues, err := c.List(context.Background(), ListOpts{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(issues) == 0 {
		t.Skip("no issues available for Show test")
	}

	id := issues[0].ID
	issue, err := c.Show(context.Background(), id)
	if err != nil {
		t.Fatalf("Show(%s) error: %v", id, err)
	}

	if issue.ID != id {
		t.Errorf("Show returned ID %q, want %q", issue.ID, id)
	}
	if issue.Title == "" {
		t.Error("Show returned empty title")
	}
}

func TestClientMissingBinary(t *testing.T) {
	c := &Client{BinPath: "bd-nonexistent-binary-12345"}
	_, err := c.List(context.Background(), ListOpts{})
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestClientContextCancellation(t *testing.T) {
	if !hasBd() {
		t.Skip("bd binary not available")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	c := NewClient()
	_, err := c.List(ctx, ListOpts{})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}
