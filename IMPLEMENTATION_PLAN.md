# Implementation Plan

## Phase 1 — Foundation & Quick Wins ✅ ALL DONE

- ~~1.1 Configurable Keymappings~~ — `internal/config/keymap.go` with `KeyMap`, `Resolve()`, `ParseKeyBinding()`, TOML overrides, 22 tests
- ~~1.2 Full Row Highlight~~ — already implemented in `timeline.go` `renderWithLines()`
- ~~1.3 Verify `--exit` Flag~~ — confirmed no hang bug; `subprocessExitMsg` → `tea.Quit` path works correctly; `--exit` only applies with `maxIterations > 0`; 4 integration tests added
- ~~1.4 Auto-Hide Left Pane~~ — `leftPaneVisible` toggle, auto-hide < 80 cols, focus auto-switch, 7 integration tests

## Phase 2 — Modals ✅ ALL DONE

- ~~2.1 Modal Infrastructure~~ — `modal.go`: `modalType` enum, `centerOverlay()`, modal routing in `handleKey()`
- ~~2.2 Quit Confirmation Modal~~ — q/ctrl+c shows modal, y quits, n/esc dismisses, double ctrl+c force-quits, `--exit` bypasses; 10 tests
- ~~2.3 Help Modal~~ — `?` opens, any key dismisses, 4 sections reflecting `KeyMap` bindings; 7 tests
- **Note**: Bubble Tea uses `"esc"` not `"escape"` — `normalizeKeyName()` and `DisplayString()` handle this

## Phase 3 — Content & Display ✅ ALL DONE

### ~~3.1 Remove Truncation from Expanded Content~~ ✅ DONE

Implemented in `expand.go`:
- Removed `maxExpandedLines` constant and `truncateLines()` function
- `expandedContentLines()` now returns all lines without truncation
- `toolCallLineCount()` already uses actual content length via `expandedContentLines()` (no separate cap)
- Updated tests: `TestExpandedContentLines_FullContent` verifies all 30 lines returned, `TestToolCallLineCount_ExpandedLargeContent` expects 31 (1 header + 30 content), `TestTimeline_View_ExpandedFullContent` verifies full content renders and no truncation footer appears

### ~~3.2 Sub-Scroll for Expanded Content~~ ✅ DONE

Implemented in `timeline.go` and `expand.go`:
- `SubScrollIdx` (flat cursor index, -1 = inactive) and `SubScrollOffset` on `Timeline`
- `subScrollViewportHeight()` and `subScrollEnabled()` implement adaptive sizing: content ≤ 40% of pane inline, > 40% capped at 70%
- `handleEnter()` detects already-expanded tool call with large content → enters sub-scroll
- `handleSubScrollAction()` routes j/k/gg/G within expanded content, enter collapses and exits
- `root.go` intercepts escape to exit sub-scroll; q/? still show modals; all other keys ignored
- `appendExpandedLines()` renders capped viewport with `│` border and `[current/total]` indicator in `ForegroundDim`
- `toolCallLineCountCapped()` for sub-scroll-aware scroll management via `effectiveTotalLines()`/`effectiveLineRange()`
- 17 tests: viewport height calc, enter/exit sub-scroll, move up/down, jump top/bottom, clamp, indicator rendering, group children, cursor stability, reset clears sub-scroll

### ~~3.3 Relative Line Numbers~~ ✅ DONE

Implemented in `root.go` and `timeline.go`:
- `lineNumbers` bool on `Model`, initialized from `config.LineNumbers` (default `true`)
- `LineNumbers` field on `TimelineProps`, passed from `root.go`
- `ActionToggleLineNumbers` (`#` key) handler in `handleKey()` toggles at runtime
- `gutterWidth` constant (4 columns: 3-char right-aligned number + 1 space)
- `renderWithLines()` prepends gutter to each visible line: cursor line shows `  0 ` in `Highlight` color, others show relative distance in `ForegroundDim`, expanded content lines get blank gutter
- `View()` reserves gutter width from content area when line numbers enabled
- `appendExpandedLines()` takes explicit `availWidth` parameter for correct gutter-aware layout
- 6 unit tests + 1 integration test: relative numbers render correctly, disabled mode has no gutter, expanded content shares parent number, cursor at zero, width reduction, `#` toggle

### ~~3.4 Vim Count+Jump Motions~~ ✅ DONE

Implemented in `timeline.go` and `root.go`:
- `CountBuffer string` on `Timeline` struct accumulates digit keys for vim count+jump motions
- `AccumulateDigit()` appends digits, ignoring leading zeros per spec
- `ConsumeCount()` returns accumulated count (min 1) and clears buffer
- `ClearCount()` clears buffer without consuming
- `HandleActionWithCount()` processes move_down/move_up with count multiplier, clamping at boundaries
- `HandleAction()` delegates to `HandleActionWithCount()` with count=1 for backward compatibility
- `root.go` intercepts digit keys `0-9` when right pane focused (not in sub-scroll, not in modal)
- On move_down/move_up: consumes count buffer as jump distance
- All other resolved actions clear the count buffer
- `renderWithLines()` overlays pending count in bottom-right corner of right pane in `ForegroundDim`
- 11 unit tests: digit accumulation, leading zero, consume empty/with value, clear, move down/up with count, no-count default, count display, no display when empty
- 7 integration tests: 5j moves 5, 12k moves 12, no-prefix moves 1, buffer clears on other keys, leading zero ignored, digits only on right pane, pending count visible in view

### ~~3.5 Full Diffs with Adaptive Layout~~ ✅ DONE

Implemented in `expand.go` and `timeline.go`:
- `renderEditDiffStyled()` produces pre-styled diff lines, choosing layout based on available width
- `renderUnifiedDiffStyled()` (width < 120): line numbers in gutter (`%4d `), `-` lines in red (`StatusError`), `+` lines in green (`StatusSuccess`)
- `renderSideBySideDiff()` (width ≥ 120): left column (old, red) | `│` divider (dim) | right column (new, green), each with line numbers and half the available width
- `appendExpandedLines()` in `timeline.go` detects Edit tool calls and uses `renderEditDiffStyled` for pre-rendered output, bypassing generic `renderExpandedContentLine` per-line styling
- Sub-scroll integration: uses styled line count for viewport calculations (handles side-by-side producing fewer lines than unified)
- Helper functions: `truncateToWidth()`, `padToWidth()` for column formatting
- `sideBySideMinWidth` constant (120) controls layout threshold
- `renderEditDiff()` retained for backward-compatible plain-text line counting used by scroll management
- 19 new tests: unified line numbers, side-by-side layout/divider/line numbers, nil/empty/only-old/only-new edge cases, truncation/padding helpers, unified content verification, side-by-side basic/uneven layouts
- 3 integration tests: unified with line numbers at width 80, side-by-side at width 140, side-by-side row count verification

## Phase 4 — Token Data

### 4.1 Per-Tool-Call Token Attribution
- [ ] In `session.go` `ProcessAssistantBatch()`: when creating `ToolCall` items, divide the turn's `message.usage` token counts equally across tool calls in that turn
- [ ] Add `InputTokens` and `CacheReadTokens` fields to `model.ToolCall`
- [ ] In `timeline.go`, render `[↑N ⚡N]` inline on each tool call row in `ForegroundDim`
- [ ] Use `FormatTokens()` for display formatting
- [ ] Tests: token attribution divides evenly, rendering shows formatted counts

### 4.2 Rate Limit Window Usage (Placeholder)
- [ ] Add `RateLimitInfo` struct with `FiveHourPercent` and `WeeklyPercent` fields (both `*float64`, nil = unknown)
- [ ] Add `RateLimitInfo` field to `Model`
- [ ] In `header.go`, render `5h: N%  wk: N%` area (or `5h: --  wk: --` when nil)
- [ ] Color thresholds: normal < 70%, warning 70-89%, critical 90%+
- [ ] Leave data fetching unimplemented — display `--` permanently for now
- [ ] Add `// TODO: implement rate limit data fetching at iteration start` comment
- [ ] Tests: header renders placeholder `--` values, renders percentages when set, color thresholds

## Phase 5 — Specs & Test Hygiene

### 5.1 Spec Verification
- [ ] Review all specs in `specs/` against implementation for consistency
- [ ] Verify cross-references between specs are correct
- [ ] Ensure examples in specs match actual rendering

### 5.2 Integration Tests for New Features
- [ ] Add integration test: help modal open/close
- [x] Add integration test: left pane auto-hide on narrow terminal
- [ ] Add integration test: sub-scroll enter/navigate/exit
- [x] Add integration test: count+jump motions
- [x] Add integration test: expand shows full content (no truncation)
- [ ] Add integration test: configurable keybindings apply end-to-end
