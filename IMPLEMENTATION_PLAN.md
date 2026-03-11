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

### 7. Git view rendering: left pane ✅

`RenderGitCommitList` and `RenderGitFileList` in `internal/tui/gitrender.go`. Bottom layout variants implemented in task 12. Session highlight coloring uses `DiffSessionCommit` with `AuthorDate.After(SessionStart)` check. Tests verify rendering with session-aware data.

### 8. Git view rendering: right pane ✅

Commit summary via `renderGitCommitSummary` with `colorizeStatLine()` applying `DiffAdded`/`DiffRemoved` to `+`/`-` characters in stat lines. Diff rendering via `RenderDiff()` with full side-by-side/unified modes. Tests cover rendering at all depths.

### 9. Git view navigation and scrolling ✅

All navigation implemented in `internal/tui/gitview.go` and `internal/tui/root.go`:
- `j`/`k`/arrows: scroll commit list (depth 0), file list (depth 1), diff (depth 2)
- `h`/`l`: horizontal scroll diff content (depth 1/2 only)
- `enter`: drill into commit → file list → sub-scroll
- `esc`: pop depth; at depth 0, exit git view
- `gg`/`G`/`Home`/`End`: jump navigation
- `pgup`/`pgdn`: page-sized scrolling at all depths via `gitViewPageUp()`/`gitViewPageDown()`
- Mouse wheel: 3-line scrolling at all depths via `gitViewScrollBy()`, routed from `handleMouse()` when `gitViewActive`

40 tests in `gitview_test.go` covering all key/mouse navigation, depth transitions, clamping, page scrolling, mouse routing, and auto-follow behavior.

### 10. Live commit list updates ✅

Implemented 5-second polling via `gitTickCmd()` → `gitRefreshCmd()` → `gitRefreshMsg` async pipeline. `enterGitView()` now returns `gitTickCmd()` to start the polling loop. `mergeGitCommits()` replaces the commit list, tracking selection by hash when user has manually scrolled (`gitAutoFollow=false`). Auto-follow (stay at index 0) re-enables on `gg`/Home jump-to-top. Polling stops automatically when git view exits (`gitTickMsg` returns nil when inactive). 10 tests covering auto-follow, manual scroll preservation, hash-gone clamping, empty merge, move up/down disabling auto-follow, jump-top re-enabling, and tick/refresh message routing.

### 11. Keybinding and config updates ✅

Already fully implemented: `ActionGitView` constant with `ctrl+g` binding in `internal/config/keymap.go`, included in `AllActions()`. Tests in `gitview_test.go` verify keymap binding and presence in AllActions. Specs updated.

### 12. Bottom layout integration ✅

Implemented git view bottom bar in `internal/tui/gitrender.go`:
- `GitBottomBarHeight = 3` (1 divider + 2 content rows) — separate from regular `BottomBarHeight = 9`
- `gitContentHeight()` calculates right pane height accounting for git-specific bottom bar
- `renderGitBottomBar()` renders a single section: "Commits" at depth 0, "Files" at depth 1-2
- `renderGitView()` appends bottom bar in bottom layout mode, uses `gitContentHeight()` for proper sizing
- `[` toggle works in git view for bottom bar visibility
- Auto-scroll-to-selection added to both `RenderGitCommitList` and `RenderGitFileList` — keeps selected item visible in the 2-line bottom bar window (also fixes scroll behavior in side layout)
- `gitViewPageDown`/`gitViewPageUp` use `gitContentHeight()` instead of `rightPaneHeight()`

10 tests in `gitview_test.go` covering bottom bar rendering (commits/files), hidden bar, content height calculation, toggle, auto-scroll-to-selection, and side layout unchanged.

## Status

All 12 tasks completed. All specs fully implemented. All tests passing.

## Notes

- `h`/`l` are repurposed for horizontal scroll only at depth 1/2 in git view. At depth 0, they retain normal focus-switching behavior.
- Chroma style is selected based on active theme name (solarized-dark → `solarized-dark`, etc.)
- The git view does not use glamour — diff rendering is custom. Chroma is used only for token-level syntax coloring.
- `sergi/go-diff` is used only for intra-line character diffing, not for computing the file diff itself (git handles that).
