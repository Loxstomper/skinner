package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Commit represents a git commit with diff stats.
type Commit struct {
	Hash       string
	Subject    string
	AuthorDate time.Time
	Additions  int
	Deletions  int
}

// FileChange represents a file changed in a commit.
type FileChange struct {
	Status    string // M, A, D, R
	Path      string
	Additions int
	Deletions int
}

const commitSep = "---COMMIT_SEP---"

// LogCommits returns the most recent commits on the current branch.
func LogCommits(limit int) ([]Commit, error) {
	cmd := exec.Command("git", "log",
		"--format="+commitSep+"%n%h%n%s%n%aI",
		"--numstat",
		fmt.Sprintf("-n%d", limit),
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}
	return ParseLogOutput(string(out))
}

// ParseLogOutput parses the output of git log with our custom format and --numstat.
// Each commit block is separated by the commitSep marker, followed by hash, subject,
// ISO date, and optional numstat lines.
func ParseLogOutput(output string) ([]Commit, error) {
	parts := strings.Split(output, commitSep+"\n")
	var commits []Commit
	for _, part := range parts {
		part = strings.TrimRight(part, "\n")
		if part == "" {
			continue
		}
		lines := strings.SplitN(part, "\n", 4)
		if len(lines) < 3 {
			continue
		}
		hash := lines[0]
		subject := lines[1]
		dateStr := lines[2]
		authorDate, _ := time.Parse(time.RFC3339, dateStr)

		var additions, deletions int
		if len(lines) > 3 {
			additions, deletions = sumNumstat(lines[3])
		}

		commits = append(commits, Commit{
			Hash:       hash,
			Subject:    subject,
			AuthorDate: authorDate,
			Additions:  additions,
			Deletions:  deletions,
		})
	}
	return commits, nil
}

// sumNumstat totals additions and deletions from numstat output lines.
// Each line has the format: "additions\tdeletions\tpath".
// Binary files show "-\t-\tpath" and are skipped.
func sumNumstat(numstatBlock string) (int, int) {
	var additions, deletions int
	for _, line := range strings.Split(numstatBlock, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "\t", 3)
		if len(fields) < 3 {
			continue
		}
		if a, err := strconv.Atoi(fields[0]); err == nil {
			additions += a
		}
		if d, err := strconv.Atoi(fields[1]); err == nil {
			deletions += d
		}
	}
	return additions, deletions
}

// DiffTreeFiles returns the files changed in a commit with status and diff stats.
// It runs two git commands: --numstat for additions/deletions and --name-status for
// the change type. Both commands list files in tree order so results are merged by index.
func DiffTreeFiles(sha string) ([]FileChange, error) {
	numstatCmd := exec.Command("git", "diff-tree", "--no-commit-id", "-r", "--numstat", sha)
	numstatOut, err := numstatCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff-tree numstat: %w", err)
	}

	statusCmd := exec.Command("git", "diff-tree", "--no-commit-id", "-r", "--name-status", sha)
	statusOut, err := statusCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff-tree name-status: %w", err)
	}

	return ParseDiffTreeOutput(string(numstatOut), string(statusOut)), nil
}

// ParseDiffTreeOutput merges numstat and name-status output into FileChanges.
// Both outputs list files in tree order, so they are merged by index position.
func ParseDiffTreeOutput(numstat, nameStatus string) []FileChange {
	numstatLines := nonEmptyLines(numstat)
	statusLines := nonEmptyLines(nameStatus)

	var changes []FileChange
	for i, statusLine := range statusLines {
		fields := strings.SplitN(statusLine, "\t", 3)
		if len(fields) < 2 {
			continue
		}
		status := fields[0]
		path := fields[len(fields)-1] // Last field is the (new) path for renames
		if len(status) > 1 {
			status = status[:1] // Normalize R100 -> R
		}

		var additions, deletions int
		if i < len(numstatLines) {
			nf := strings.SplitN(numstatLines[i], "\t", 3)
			if len(nf) >= 2 {
				additions, _ = strconv.Atoi(nf[0])
				deletions, _ = strconv.Atoi(nf[1])
			}
		}

		changes = append(changes, FileChange{
			Status:    status,
			Path:      path,
			Additions: additions,
			Deletions: deletions,
		})
	}
	return changes
}

// nonEmptyLines splits text into lines and returns only non-empty ones.
func nonEmptyLines(s string) []string {
	var result []string
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

// ShowCommit returns the full commit message and stats for display.
func ShowCommit(sha string) (string, error) {
	cmd := exec.Command("git", "show", "--stat", sha)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git show: %w", err)
	}
	return string(out), nil
}

// FileDiff returns the unified diff for a single file in a commit.
func FileDiff(sha, path string) (string, error) {
	cmd := exec.Command("git", "diff", "--diff-algorithm=histogram",
		sha+"~1", sha, "--", path)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff: %w", err)
	}
	return string(out), nil
}
