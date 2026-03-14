package bd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
)

// ErrNotFound indicates the bd binary was not found in PATH.
var ErrNotFound = errors.New("bd: binary not found")

// ErrNoBeadsDir indicates no .beads directory exists in the current project.
var ErrNoBeadsDir = errors.New("bd: no .beads directory found")

// Client wraps the bd CLI for programmatic access.
type Client struct {
	// BinPath is the path to the bd binary. Defaults to "bd".
	BinPath string
}

// NewClient returns a Client with the default bd binary path.
func NewClient() *Client {
	return &Client{BinPath: "bd"}
}

// ListOpts configures filtering for the List method.
type ListOpts struct {
	Status   string // Filter by status (e.g. "blocked", "in_progress").
	Type     string // Filter by issue type.
	Assignee string // Filter by assignee.
}

// List returns issues matching the given filters.
// With no filters, returns all issues.
func (c *Client) List(ctx context.Context, opts ListOpts) ([]Issue, error) {
	args := []string{"list", "--json", "--limit", "0"}
	if opts.Status != "" {
		args = append(args, "--status", opts.Status)
	}
	if opts.Type != "" {
		args = append(args, "--type", opts.Type)
	}
	if opts.Assignee != "" {
		args = append(args, "--assignee", opts.Assignee)
	}

	var issues []Issue
	if err := c.run(ctx, args, &issues); err != nil {
		return nil, err
	}
	return issues, nil
}

// Ready returns issues that are ready to work on (unblocked, open).
func (c *Client) Ready(ctx context.Context) ([]Issue, error) {
	var issues []Issue
	if err := c.run(ctx, []string{"ready", "--json"}, &issues); err != nil {
		return nil, err
	}
	return issues, nil
}

// Show returns the full detail for a single issue.
func (c *Client) Show(ctx context.Context, id string) (*Issue, error) {
	// bd show --json returns an array with one element.
	var issues []Issue
	if err := c.run(ctx, []string{"show", id, "--json"}, &issues); err != nil {
		return nil, err
	}
	if len(issues) == 0 {
		return nil, fmt.Errorf("bd show: no issue returned for %s", id)
	}
	return &issues[0], nil
}

// run executes a bd command and unmarshals its JSON output into dest.
func (c *Client) run(ctx context.Context, args []string, dest any) error {
	bin := c.BinPath
	if bin == "" {
		bin = "bd"
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	out, err := cmd.Output()
	if err != nil {
		// Check for binary not found.
		if errors.Is(err, exec.ErrNotFound) {
			return ErrNotFound
		}
		// Check for exit error with stderr.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr := string(exitErr.Stderr)
			return fmt.Errorf("bd %s: exit %d: %s", args[0], exitErr.ExitCode(), stderr)
		}
		return fmt.Errorf("bd %s: %w", args[0], err)
	}

	if err := json.Unmarshal(out, dest); err != nil {
		return fmt.Errorf("bd %s: invalid JSON: %w", args[0], err)
	}
	return nil
}
