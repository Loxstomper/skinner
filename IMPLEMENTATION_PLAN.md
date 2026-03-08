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

## Phase 3 — Content & Display

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

### 3.4 Vim Count+Jump Motions
- [ ] Add `countBuffer` (string or int accumulator) to `Timeline`
- [ ] Digit keys `1`-`9` append to buffer; `0` ignored as leading digit
- [ ] On `j`/`k`: consume buffer as count, move cursor by count, clear buffer
- [ ] Any non-digit, non-j/k key clears the buffer
- [ ] Display pending count in bottom-right corner of right pane in `ForegroundDim`
- [ ] Tests: `5j` moves 5 items, `12k` moves 12, no-prefix moves 1, buffer clears on other keys

### 3.5 Full Diffs with Adaptive Layout
- [ ] Update `renderEditDiff()` in `expand.go` to show full diff (no truncation — already done in 3.1)
- [ ] Add line numbers to diff output gutter
- [ ] Detect terminal width: if ≥ 120 cols, render side-by-side; otherwise unified
- [ ] Side-by-side: left column (old, red) | divider | right column (new, green), each half-width
- [ ] Unified: current format with line numbers, `-` red, `+` green
- [ ] Color applied in both layouts
- [ ] Tests: unified below 120 cols, side-by-side at 120+, line numbers present, colors applied

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
- [ ] Add integration test: count+jump motions
- [x] Add integration test: expand shows full content (no truncation)
- [ ] Add integration test: configurable keybindings apply end-to-end
