# Implementation Plan

## 1. Fix edit line spec example

The spec example `(+3/-1)` is impossible with the described algorithm. The algorithm computes `added = new_lines - old_lines` and `removed = old_lines - new_lines` — these are mutually exclusive, so you can never have both positive at the same time.

### Tasks

- [ ] Update `specs/stream-json-format.md` line 146: change the example from `(+3/-1)` to `(+3)` and adjust the surrounding text to clarify that additions and removals are mutually exclusive except in the net-zero case
- [ ] Update `specs/tui-layout.md` lines 95 and 107: change `(+3/-1)` to `(+2/-2)` (a valid net-zero example) in both Full view and Compact view examples

## 2. Document `--exit` flag

The `--exit` flag exists in `cmd/skinner/main.go` but is not documented.

### Tasks

- [ ] Add `--exit` to the CLI arguments table in `specs/iteration-loop.md`
- [ ] Document its behavior: when set, the TUI quits automatically after all iterations complete (or the last iteration fails), rather than remaining open for browsing

## 3. PgUp/PgDown cursor adjustment in timeline

After page scrolling, the cursor may be outside the visible viewport. The highlighted row won't be visible. Note: IterList pgup/pgdown already uses `ensureCursorVisible` (added with left pane scroll support), so the iterlist side is done — this task is timeline-only.

### Tasks

- [ ] In `internal/tui/timeline.go`, after `pgdown` scroll adjustment, clamp cursor into the visible range: if cursor is above viewport, move cursor to the first visible flat position; if cursor is below viewport, move cursor to the last visible flat position
- [ ] Same for `pgup`
- [ ] Add a `LineToFlatCursor(items []model.TimelineItem, line int, compactView bool) int` helper in `internal/tui/cursor.go` that maps a rendered line number to the flat cursor position that owns that line (inverse of `FlatCursorLineRange`)
- [ ] Add tests in `internal/tui/cursor_test.go` for `LineToFlatCursor`
- [ ] Add tests in `internal/tui/timeline_test.go` verifying cursor moves into viewport after pgdown/pgup
- [ ] Update `specs/keybindings.md` pgdn/pgup description to note that the cursor moves into the visible viewport after page scroll

## 4. Mouse scrolling and clicking

Add mouse support for both panes: scroll wheel scrolls the pane under the pointer, click selects rows and switches pane focus.

### Tasks

- [ ] Create `specs/mouse.md` spec covering: mouse mode, scroll behavior (3 lines per tick), click to focus pane, click to select row, auto-follow pausing on mouse interaction, click on empty space does nothing, click on collapsed group header just selects (doesn't expand)
- [ ] Add `mouse.md` to the specs table in `specs/README.md`
- [ ] Enable mouse mode in `cmd/skinner/main.go`: add `tea.WithMouseCellMotion()` option to `tea.NewProgram`
- [ ] Add `tea.MouseMsg` handling in root `Update()` in `internal/tui/root.go`:
  - Determine target pane by comparing `msg.X` against left pane width (32)
  - Subtract header height (1) from `msg.Y` to get pane-relative Y
  - Ignore events where pane-relative Y is negative (click on header)
  - On wheel up/down: switch focus to target pane, adjust that pane's scroll by 3 lines, clamp scroll, pause auto-follow
  - On click (`tea.MouseActionPress` with `tea.MouseButtonLeft`): switch focus to target pane, map pane-relative Y to a row, move cursor to that row, pause auto-follow
- [ ] Add mouse scroll handling to `IterList`: new method `ScrollBy(delta int, props IterListProps)` that adjusts scroll and clamps
- [ ] Add mouse scroll handling to `Timeline`: new method `ScrollBy(delta int, props TimelineProps)` that adjusts scroll and clamps
- [ ] Add mouse click handling to `IterList`: new method `ClickRow(row int, props IterListProps)` that sets cursor to `scroll + row` if valid, resets timeline position (via return value or callback)
- [ ] Add mouse click handling to `Timeline`: new method `ClickRow(row int, props TimelineProps)` that maps `scroll + row` to the flat cursor position using `LineToFlatCursor` (from task 3), sets cursor if valid
- [ ] Both click handlers ignore clicks beyond the last item (do nothing)
- [ ] Both scroll and click handlers call `AutoFollow.OnManualMove()` to pause auto-follow
- [ ] Add tests in `internal/tui/iterlist_test.go` for mouse scroll and click
- [ ] Add tests in `internal/tui/timeline_test.go` for mouse scroll and click
- [ ] Add integration test in `internal/tui/integration_test.go` verifying mouse click switches pane focus

## 5. Token format units (k, M, G)

Extend `FormatTokens()` to support M and G suffixes for millions and billions.

### Tasks

- [ ] Update `FormatTokens()` in `internal/tui/format.go` to add thresholds: `>= 1,000,000,000` → `N.NG`, `>= 1,000,000` → `N.NM`, `>= 1,000` → `N.Nk` (existing), `< 1,000` → raw (existing)
- [ ] Update tests in `internal/tui/format_test.go` to cover M and G cases
- [ ] Update `specs/tui-layout.md` header section token format description to mention M and G suffixes
