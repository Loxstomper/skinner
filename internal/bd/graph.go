package bd

import "strings"

// Graph holds an indexed collection of issues for fast lookup and traversal.
type Graph struct {
	// Issues in bd's original sort order.
	Issues []*Issue
	// ByID maps issue ID to issue pointer.
	ByID map[string]*Issue
	// Children maps parent ID to child issues.
	Children map[string][]*Issue
	// Roots are issues with no parent.
	Roots []*Issue

	// blockingDeps maps issue ID to IDs of issues that block it.
	blockingDeps map[string][]string
	// blockedBy maps issue ID to IDs of issues it blocks (reverse index).
	blockedBy map[string][]string
	// nonBlockingDeps maps issue ID to non-blocking dependencies.
	nonBlockingDeps map[string][]Dependency
}

// DepNode represents a node in a transitive dependency tree.
type DepNode struct {
	Issue    *Issue
	Type     string
	Children []*DepNode
	Depth    int
	Cycle    bool
}

// NewGraph builds a graph from a flat list of issues.
func NewGraph(issues []Issue) *Graph {
	g := &Graph{
		ByID:            make(map[string]*Issue, len(issues)),
		Children:        make(map[string][]*Issue),
		blockingDeps:    make(map[string][]string),
		blockedBy:       make(map[string][]string),
		nonBlockingDeps: make(map[string][]Dependency),
	}

	// Store pointers and build ID index.
	for i := range issues {
		issue := &issues[i]
		g.Issues = append(g.Issues, issue)
		g.ByID[issue.ID] = issue
	}

	// Build parent-child tree and dependency indexes.
	for _, issue := range g.Issues {
		if issue.Parent != "" {
			g.Children[issue.Parent] = append(g.Children[issue.Parent], issue)
		} else {
			g.Roots = append(g.Roots, issue)
		}

		for _, dep := range issue.Dependencies {
			depType := dep.DepType()
			if depType == "blocks" {
				// This issue is blocked by the dependency target.
				targetID := dep.DependsOnID
				if targetID == "" {
					targetID = dep.ID
				}
				if targetID != "" {
					g.blockingDeps[issue.ID] = append(g.blockingDeps[issue.ID], targetID)
					g.blockedBy[targetID] = append(g.blockedBy[targetID], issue.ID)
				}
			} else if depType != "parent-child" {
				g.nonBlockingDeps[issue.ID] = append(g.nonBlockingDeps[issue.ID], dep)
			}
		}
	}

	return g
}

// WalkBlockingDeps returns the transitive blocking dependency tree for the given issue.
// Walks dependencies of type "blocks" up to maxDepth levels, with cycle detection.
func (g *Graph) WalkBlockingDeps(id string, maxDepth int) []*DepNode {
	visited := map[string]bool{id: true}
	return g.walkDeps(id, maxDepth, 0, visited, g.blockingDeps)
}

// WalkBlockingDependents returns the transitive tree of issues blocked by the given issue.
// Walks in the reverse direction up to maxDepth levels, with cycle detection.
func (g *Graph) WalkBlockingDependents(id string, maxDepth int) []*DepNode {
	visited := map[string]bool{id: true}
	return g.walkDeps(id, maxDepth, 0, visited, g.blockedBy)
}

func (g *Graph) walkDeps(id string, maxDepth, depth int, visited map[string]bool, index map[string][]string) []*DepNode {
	targetIDs := index[id]
	if len(targetIDs) == 0 {
		return nil
	}

	var nodes []*DepNode
	for _, targetID := range targetIDs {
		issue := g.ByID[targetID]
		if issue == nil {
			continue
		}

		node := &DepNode{
			Issue: issue,
			Type:  "blocks",
			Depth: depth + 1,
		}

		if visited[targetID] {
			node.Cycle = true
			nodes = append(nodes, node)
			continue
		}

		visited[targetID] = true
		if depth+1 < maxDepth {
			node.Children = g.walkDeps(targetID, maxDepth, depth+1, visited, index)
		}
		nodes = append(nodes, node)
	}

	return nodes
}

// NonBlockingDeps returns non-blocking dependencies for the given issue
// (relates_to, discovered-from, tracks, etc.). Excludes "blocks" and "parent-child".
func (g *Graph) NonBlockingDeps(id string) []Dependency {
	return g.nonBlockingDeps[id]
}

// FilterByStatus returns issues matching the given status.
func (g *Graph) FilterByStatus(status string) []*Issue {
	var result []*Issue
	for _, issue := range g.Issues {
		if issue.Status == status {
			result = append(result, issue)
		}
	}
	return result
}

// FuzzySearch returns issues whose title, ID, or description contain the query
// as a case-insensitive substring.
func (g *Graph) FuzzySearch(query string) []*Issue {
	if query == "" {
		return g.Issues
	}

	q := strings.ToLower(query)
	var result []*Issue
	for _, issue := range g.Issues {
		if strings.Contains(strings.ToLower(issue.Title), q) ||
			strings.Contains(strings.ToLower(issue.ID), q) ||
			strings.Contains(strings.ToLower(issue.Description), q) {
			result = append(result, issue)
		}
	}
	return result
}
