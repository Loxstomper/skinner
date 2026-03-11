# Implementation Plan: Git View Commit Stats Redesign

## Summary

Remove per-commit addition/deletion stats from unselected rows in the git view commit list. Add a total repository stats header line at the top of the left pane (loaded asynchronously). Show stats only on the selected commit row, replacing the relative time. Shorten hash display from 7 to 3 characters.

## Tasks

### 1. Update spec (already done)
- [x] Update `specs/git-view.md` with new commit list format, header line, 3-char hash, selected-row stats

### 2. Add `FormatStatNumber` helper to `internal/git/git.go`
- [x] Add a function to format numbers with K/M suffixes: `<1000` raw, `1K–9.9K` one decimal, `10K+` whole K, same for M
- [x] Add tests in `internal/git/git_test.go` for all formatting thresholds (16 test cases covering all boundaries)

### 3. Add `TotalStats` function to `internal/git/git.go`
- [x] Add function that runs `git log --shortstat --no-merges` and parses the output
- [x] Parse lines matching `N files changed, N insertions(+), N deletions(-)` and accumulate totals
- [x] Accept a `context.Context` for cancellation
- [x] Add `ParseShortstatLine(line string) (additions, deletions int, ok bool)` as a testable helper
- [x] Add tests in `internal/git/git_test.go` for parsing various shortstat formats (insertions only, deletions only, both, binary-only commits)

### 4. Add Bubble Tea message types for async stats
- [x] Add `gitTotalStatsMsg struct{ Additions, Deletions int }` to `internal/tui/root.go`
- [x] Add `gitTotalStatsCmd` that calls `git.TotalStats()` in a goroutine
- [x] Add state fields to Model: `gitTotalAdditions int`, `gitTotalDeletions int`, `gitTotalStatsLoaded bool`, `gitTotalStatsCancel context.CancelFunc`
- [x] Handle `gitTotalStatsMsg` in the Update switch — store totals and set loaded flag

### 5. Wire up async stats lifecycle
- [x] In `enterGitView()`: create a context, store cancel func, fire `gitTotalStatsCmd`
- [x] When exiting git view (esc at depth 0): call cancel func, reset stats state
- [x] No caching — re-entering git view re-runs the command

### 6. Update `GitCommitListProps` and `RenderGitCommitList`
- [x] Add fields to `GitCommitListProps`: `TotalAdditions int`, `TotalDeletions int`, `TotalStatsLoaded bool`
- [x] Render header line as first row: `────── +1.2K -4.2K ──────` centered, or `────── ... ──────` while loading
- [x] Use `DiffAdded`/`DiffRemoved` colors for stats, `ForegroundDim` for `─` divider
- [x] Reduce available height by 1 to account for header line
- [x] Truncate hash to 3 characters (from 7) — uses direct slice, not `truncate()` to avoid ellipsis
- [x] Remove stats from unselected rows: just `hash subject relTime`
- [x] On selected row: replace `relTime` with `+N -N` stats (show `+0 -0` for zero-stat commits)
- [x] Update `maxSubject` width calculation to account for shorter hash and removed stats column
- [x] Extract `renderCommitListHeader()` helper for the centered divider line

### 7. Update render call sites
- [x] Update `renderGitView()` side layout call to pass `TotalAdditions`, `TotalDeletions`, `TotalStatsLoaded`
- [x] Update `renderGitBottomBar()` bottom layout call to pass the same fields

### 8. Update tests
- [x] Add `TestCommitListHeaderLoading` — verifies "..." shown while loading
- [x] Add `TestCommitListHeaderLoaded` — verifies formatted stats in header
- [x] Add `TestCommitListThreeCharHash` — verifies 3-char hash without ellipsis
- [x] Add `TestCommitListUnselectedNoStats` — verifies no +N/-N on unselected rows
- [x] Add `TestCommitListSelectedRowShowsStats` — verifies +N -N on selected row
- [x] Add `TestCommitListSelectedRowZeroStats` — verifies +0 -0 for zero-stat commits
- [x] Add `TestGitTotalStatsMsgStoresValues` — verifies msg updates model state
- [x] Add `TestGitTotalStatsResetOnExit` — verifies stats cleared on exit
- [x] All existing tests pass with updated code (no changes needed to existing tests)

### 9. Run `make check`
- [x] All tests pass, linting clean, vet clean
