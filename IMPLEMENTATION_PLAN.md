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
- Add fields to `GitCommitListProps`: `TotalAdditions int`, `TotalDeletions int`, `TotalStatsLoaded bool`
- Render header line as first row: `────── +12.3K -4.2K ──────` centered, or `────── ... ──────` while loading
- Use `DiffAdded`/`DiffRemoved` colors for stats, `ForegroundDim` for `─` divider
- Reduce available height by 1 to account for header line
- Truncate hash to 3 characters (from 7)
- Remove stats from unselected rows: just `hash subject relTime`
- On selected row: replace `relTime` with `+N -N` stats (show `+0 -0` for zero-stat commits)
- Update `maxSubject` width calculation to account for shorter hash and removed stats column

### 7. Update render call sites
- Update all places that construct `GitCommitListProps` to pass the new total stats fields
- This includes the main View render path and the bottom layout render path

### 8. Update tests
- Update `internal/tui/gitview_test.go`:
  - Update commit list rendering tests to expect 3-char hashes, no per-row stats
  - Add tests for header line rendering (loading state and loaded state)
  - Add tests for selected row showing stats instead of time
  - Add tests for `gitTotalStatsMsg` handling (stores values, sets loaded flag)
  - Add tests for stats lifecycle (cancelled on exit, reset on re-enter)
- Update `internal/git/git_test.go`:
  - Add tests for `FormatStatNumber`
  - Add tests for `ParseShortstatLine`
  - Add tests for `TotalStats` (if feasible with test fixtures)
- Verify existing tests still pass with updated expectations

### 9. Run `make check`
- Ensure all tests pass, linting clean, vet clean
