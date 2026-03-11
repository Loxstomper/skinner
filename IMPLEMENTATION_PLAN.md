# Implementation Plan: Git View

## Overview

Read-only git history viewer (`ctrl+g`) with side-by-side diffs, syntax highlighting, and intra-line change emphasis. See [specs/git-view.md](specs/git-view.md).

## Dependencies

- [ ] Add `github.com/sergi/go-diff` for character-level intra-line diffing
- [ ] Add `github.com/alecthomas/chroma/v2` as direct dependency (currently indirect via glamour)

## Tasks

### 1. Theme: add diff color roles ✅

Added 8 new fields to `Theme` struct: `DiffAdded`, `DiffRemoved`, `DiffAddedBg`, `DiffRemovedBg`, `DiffAddedEmphasis`, `DiffRemovedEmphasis`, `DiffLineNumber`, `DiffSessionCommit`. All 4 built-in themes populated per spec. Tests verify non-empty on all themes.

### 2. Git data layer: commit list and diff loading

New package `internal/git` with functions:
- `LogCommits(limit int) ([]Commit, error)` — runs `git log`, returns slice of `Commit{Hash, Subject, AuthorDate, Additions, Deletions}`
- `DiffTreeFiles(sha string) ([]FileChange, error)` — runs `git diff-tree --no-commit-id -r --numstat <sha>`, returns `FileChange{Status, Path, Additions, Deletions}`
- `ShowCommit(sha string) (string, error)` — runs `git show --stat <sha>`, returns full commit message + stats
- `FileDiff(sha, path string) (string, error)` — runs `git diff --diff-algorithm=histogram <sha>~1 <sha> -- <path>`, returns unified diff

All functions shell out to git CLI. No domain logic beyond parsing output.

**Tests**: `internal/git/git_test.go` — test output parsing with canned git output (no real git repo required; test the parsers, not git itself).

### 3. Diff parser: unified diff to structured hunks

New file `internal/tui/diffparse.go`:
- Parse unified diff string into `[]Hunk` where each hunk has `[]DiffLine{Type, OldNum, NewNum, Content}`
- Pair lines for side-by-side: context lines on both sides, removed+added blocks paired row-by-row, unequal blocks padded with blank lines
- Types: `DiffLineContext`, `DiffLineAdded`, `DiffLineRemoved`

**Tests**: `internal/tui/diffparse_test.go` — table-driven tests: parse a unified diff string, verify hunk structure, line pairing, and padding for unequal add/remove blocks.

### 4. Intra-line highlighting

New file `internal/tui/diffhighlight.go`:
- `IntraLineChanges(oldLine, newLine string) ([]CharRange, []CharRange)` — uses `sergi/go-diff` Myers algorithm to compute character-level differences
- Returns ranges of changed characters for each side

**Tests**: `internal/tui/diffhighlight_test.go` — test character range computation: identical lines (no ranges), single word change, multiple changes, entire line changed.

### 5. Diff renderer: side-by-side and unified rendering

New file `internal/tui/diffview.go`:
- `RenderDiff(props DiffViewProps) string` — renders parsed hunks as styled output
- Side-by-side when width >= 80, unified when < 80
- Line number gutters (non-scrolling), code content (horizontally scrollable)
- Applies diff line backgrounds (`DiffAddedBg`/`DiffRemovedBg`) via theme
- Applies intra-line emphasis (`DiffAddedEmphasis`/`DiffRemovedEmphasis`) from task 4
- Syntax highlighting via chroma token iterator, using theme-mapped chroma style

`DiffViewProps` includes: parsed hunks, file path (for language detection), theme, pane width, horizontal scroll offset.

**Tests**: `internal/tui/diffview_test.go` — test side-by-side vs unified switching at width threshold, line number rendering, horizontal scroll offset clipping.

### 6. Git view model state

Add to `Model` in `internal/tui/root.go`:
- `gitViewActive bool` — whether git view is showing
- `gitViewDepth int` — 0=commit list, 1=file list, 2=sub-scroll
- `gitCommits []git.Commit` — cached commit list
- `gitSelectedCommit int` — cursor in commit list
- `gitFiles []git.FileChange` — file list for selected commit
- `gitSelectedFile int` — cursor in file list
- `gitCommitScroll int`, `gitFileScroll int`, `gitDiffScroll int`, `gitDiffHScroll int` — scroll positions
- `gitSessionStart time.Time` — set at launch for session commit highlighting
- `gitParsedDiff []Hunk` — cached parsed diff for selected file
- Preserved run view state (existing fields already remain untouched)

Wire `ctrl+g` in keymap (`internal/config/keymap.go`) to a new `"git_view"` action. Wire `esc` to pop back through depths.

**Tests**: Verify `ctrl+g` toggles `gitViewActive`, `esc` at depth 0 exits git view, `enter`/`esc` navigates depths correctly.

### 7. Git view rendering: left pane

New file `internal/tui/gitlist.go`:
- `RenderGitCommitList(props) string` — renders commit rows: hash, subject, relative time, +/- counts. Session commits use `DiffSessionCommit` color.
- `RenderGitFileList(props) string` — renders file change rows: status, filename, +/- counts.
- Both support cursor highlighting with `Highlight` background, scroll offset, and `ViewBottom()` for bottom layout.

**Tests**: `internal/tui/gitlist_test.go` — test commit list rendering (session highlight, format), file list rendering (status letters, colors), scroll/cursor positioning.

### 8. Git view rendering: right pane

Wire into `View()` in `internal/tui/root.go`:
- When `gitViewActive` and depth 0: right pane calls `RenderCommitSummary()` (commit message + stats from `ShowCommit`)
- When `gitViewActive` and depth 1/2: right pane calls `RenderDiff()` from task 5

**Tests**: Integration tests in `internal/tui/gitview_test.go` — test full view output at each depth, verify correct content appears in left/right panes.

### 9. Git view navigation and scrolling

In `handleKey()` (`internal/tui/root.go`), when `gitViewActive`:
- `j`/`k`/mouse: scroll commit list (depth 0) or file list (depth 1, left pane) or diff (depth 1/2, right pane)
- `h`/`l`: horizontal scroll diff content (depth 1/2 only, right pane focused)
- `enter`: drill into commit → file list; enter sub-scroll on diff
- `esc`: pop depth; at depth 0, exit git view
- `gg`/`G`/`pgup`/`pgdn`: jump navigation

**Tests**: Test key sequences for depth transitions, scroll state changes, h/l horizontal scroll, esc at each depth.

### 10. Live commit list updates

Add a `tea.Tick` command (5-second interval) that re-runs `LogCommits()` when git view is active. Merge new commits into the list. Auto-follow (scroll to top) unless user has manually scrolled.

**Tests**: Test that new commits appear at top, auto-follow behavior, manual scroll pauses auto-follow.

### 11. Keybinding and config updates

- Add `"git_view"` action to default keymap with `ctrl+g` binding
- Update `internal/config/keymap.go` with the new action
- Update `specs/keybindings.md` (already done)
- Update `specs/git-view.md` (already done)

**Tests**: `config/keymap_test.go` — verify `ctrl+g` resolves to `"git_view"`.

### 12. Bottom layout integration

Ensure git view works in bottom layout mode:
- Commit list / file list renders in bottom bar sections when bottom layout active
- Same drill-down navigation applies
- Width threshold for side-by-side uses full-width right pane in bottom mode

**Tests**: Add cases to `layout_test.go` verifying git view renders correctly in bottom layout.

## Task Order

Dependencies: 1 → 2 → 3 → 4 → 5 → 7,8 → 6,9 → 10 → 11,12

Tasks 1 and 2 can be done in parallel. Tasks 3 and 4 can be done in parallel. Tasks 7 and 8 can be done in parallel after 5. Tasks 11 and 12 can be done in parallel at the end.

## Notes

- `h`/`l` are repurposed for horizontal scroll only at depth 1/2 in git view. At depth 0, they retain normal focus-switching behavior.
- Chroma style is selected based on active theme name (solarized-dark → `solarized-dark`, etc.)
- The git view does not use glamour — diff rendering is custom. Chroma is used only for token-level syntax coloring.
- `sergi/go-diff` is used only for intra-line character diffing, not for computing the file diff itself (git handles that).
