# Implementation Plan: Git View

## Overview

Read-only git history viewer (`ctrl+g`) with side-by-side diffs, syntax highlighting, and intra-line change emphasis. See [specs/git-view.md](specs/git-view.md).

## Dependencies

- [x] Add `github.com/sergi/go-diff` for character-level intra-line diffing
- [x] Add `github.com/alecthomas/chroma/v2` as direct dependency (currently indirect via glamour)

## Tasks

### 1. Theme: add diff color roles ✅

Added 8 new fields to `Theme` struct: `DiffAdded`, `DiffRemoved`, `DiffAddedBg`, `DiffRemovedBg`, `DiffAddedEmphasis`, `DiffRemovedEmphasis`, `DiffLineNumber`, `DiffSessionCommit`. All 4 built-in themes populated per spec. Tests verify non-empty on all themes.

### 2. Git data layer: commit list and diff loading ✅

Implemented `internal/git` package with 4 public functions (`LogCommits`, `DiffTreeFiles`, `ShowCommit`, `FileDiff`) that shell out to git CLI, plus exported parse functions (`ParseLogOutput`, `ParseDiffTreeOutput`) for testability. `DiffTreeFiles` runs two git commands (`--numstat` and `--name-status`) merged by index position (tree order). Handles binary files (numstat `-\t-`), renames (`R100` normalized to `R`), and merge commits (no numstat). Tests use canned output — 17 test cases covering all parsers.

### 3. Diff parser: unified diff to structured hunks ✅

Implemented `internal/tui/diffparse.go` with types `DiffLineType`, `DiffLine`, `Hunk`, `SideBySideLine`. Two main functions: `ParseUnifiedDiff` parses unified diff strings into `[]Hunk` with `@@ ` header extraction and line number tracking; `PairLines` converts hunks into `[]SideBySideLine` for side-by-side rendering with context on both sides, removed+added blocks paired row-by-row, and nil padding for unequal blocks. Handles `\ No newline at end of file` markers, missing count in hunk headers, and multi-hunk diffs. 15 tests covering parsing, line numbering, pairing, padding, and edge cases.

### 4. Intra-line highlighting ✅

Implemented `internal/tui/diffhighlight.go` with `CharRange` type and `IntraLineChanges` function using `sergi/go-diff` Myers algorithm with semantic cleanup. Handles identical lines (nil ranges), single word changes, multiple changes, entire line changes, pure insertions, and pure deletions. Uses rune-based positions for Unicode correctness. 8 tests covering all cases plus range validity checks across multiple inputs.

### 5. Diff renderer: side-by-side and unified rendering ✅

Implemented `internal/tui/diffview.go` with `RenderDiff(props DiffViewProps) string` function. Side-by-side rendering at width >= 80, unified at < 80. Features: line number gutters (non-scrolling), horizontally scrollable code content, diff line backgrounds (`DiffAddedBg`/`DiffRemovedBg`), intra-line emphasis overlays (`DiffAddedEmphasis`/`DiffRemovedEmphasis`), and chroma syntax highlighting with theme-mapped styles (solarized-dark/light, monokai, nord). Character-level rendering pipeline: chroma tokenization → foreground colors → diff background → emphasis overlay → horizontal scroll clipping. Added chroma/v2 as direct dependency. 12 tests covering side-by-side vs unified switching, line numbers, horizontal scroll, empty input, prefixes, padding, syntax highlighting, and intra-line emphasis.

### 6. Git view model state ✅

Added git view state fields to `Model` in `internal/tui/root.go`: `gitViewActive`, `gitViewDepth` (0=commit list, 1=file list, 2=sub-scroll), `gitCommits`, `gitSelectedCommit`, `gitFiles`, `gitSelectedFile`, scroll positions (`gitCommitScroll`, `gitFileScroll`, `gitDiffScroll`, `gitDiffHScroll`), `gitSessionStart` (set at `NewModel`), `gitParsedDiff`, `gitCommitSummary`, `gitDiffContent`.

Wired `ctrl+g` to `ActionGitView` in keymap. New files: `internal/tui/gitview.go` (state machine: `enterGitView`, `exitGitView`, `handleGitViewKey`, navigation methods, data loading), `internal/tui/gitrender.go` (rendering: `RenderGitCommitList`, `RenderGitFileList`, `renderGitCommitSummary`, helpers). `View()` in `root.go` renders git view overlay when active. Navigation: `esc` pops depth (0→exit, 1→0, 2→1), `enter` drills down (0→1→2), `j/k` scroll lists/diff, `h/l` horizontal scroll at depth 1/2, `gg`/`G` jump. 25 tests in `gitview_test.go` covering state transitions, clamping, scroll, rendering.

### 7. Git view rendering: left pane (partially done in task 6)

`RenderGitCommitList` and `RenderGitFileList` are implemented in `internal/tui/gitrender.go` (done in task 6). Remaining:
- `ViewBottom()` variants for bottom layout rendering
- More detailed rendering tests (session highlight verification, exact format checks, scroll/cursor edge cases)

### 8. Git view rendering: right pane (partially done in task 6)

Commit summary and diff rendering are wired in `renderGitView()` in `gitrender.go` (done in task 6). Remaining:
- Stat line coloring in commit summary (currently basic +/- coloring)
- Integration tests verifying correct content at each depth with realistic data

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
