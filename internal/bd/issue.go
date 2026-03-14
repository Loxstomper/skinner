// Package bd provides types and utilities for interacting with the bd (beads) issue tracker.
package bd

import "time"

// Issue represents a beads issue matching the bd JSON schema.
type Issue struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Status      string   `json:"status"`
	Priority    int      `json:"priority"`
	IssueType   string   `json:"issue_type"`
	Assignee    string   `json:"assignee,omitempty"`
	Owner       string   `json:"owner,omitempty"`
	CreatedBy   string   `json:"created_by,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	Parent      string   `json:"parent,omitempty"`

	Dependencies []Dependency `json:"dependencies,omitempty"`
	Dependents   []Dependency `json:"dependents,omitempty"`

	// Counts from bd list format (when full dependency data is not included).
	DependencyCount int `json:"dependency_count,omitempty"`
	DependentCount  int `json:"dependent_count,omitempty"`

	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ClosedAt    time.Time `json:"closed_at,omitempty"`
	CloseReason string    `json:"close_reason,omitempty"`

	Gates       []Gate         `json:"gates,omitempty"`
	RelatesTo   []string       `json:"relates_to,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	ExternalRef string         `json:"external_ref,omitempty"`

	CommentCount int `json:"comment_count,omitempty"`
}

// Dependency represents a dependency relationship between issues.
// In bd show --json format, it contains full issue data plus a dependency_type field.
// In bd list --json format, it uses compact fields (issue_id, depends_on_id, type).
type Dependency struct {
	// Full issue data (populated from bd show --json).
	ID          string `json:"id,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	Priority    int    `json:"priority,omitempty"`
	IssueType   string `json:"issue_type,omitempty"`
	Assignee    string `json:"assignee,omitempty"`
	Owner       string `json:"owner,omitempty"`
	CreatedBy   string `json:"created_by,omitempty"`

	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
	ClosedAt    time.Time `json:"closed_at,omitempty"`
	CloseReason string    `json:"close_reason,omitempty"`

	// DependencyType is used in bd show --json format ("blocks", "parent-child", etc.).
	DependencyType string `json:"dependency_type,omitempty"`

	// Compact fields from bd list --json format.
	IssueID     string `json:"issue_id,omitempty"`
	DependsOnID string `json:"depends_on_id,omitempty"`
	Type        string `json:"type,omitempty"`
	MetadataRaw string `json:"metadata,omitempty"`
}

// DepType returns the dependency type, checking both the show and list format fields.
func (d Dependency) DepType() string {
	if d.DependencyType != "" {
		return d.DependencyType
	}
	return d.Type
}

// Gate represents a gate condition on an issue.
type Gate struct {
	Type     string `json:"type"`
	Detail   string `json:"detail,omitempty"`
	Status   string `json:"status"`
	Complete bool   `json:"complete"`
}
