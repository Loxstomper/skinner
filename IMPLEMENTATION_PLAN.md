# Implementation Plan

## Phase 1 — Foundation & Quick Wins

### ~~1.1 Configurable Keymappings~~ ✅ DONE

Implemented in `internal/config/keymap.go`:
- `KeyMap` type with `Bindings map[string]KeyBinding` and `Resolve()` method for sequence-aware key dispatch
- `ParseKeyBinding()` supports single keys, modifiers (`ctrl+c`), and sequences (`g g`)
- `[keybindings]` TOML section parsed in `LoadConfig()`, merging overrides with defaults
- `Config` now includes `KeyMap` and `LineNumbers` fields
- `root.go` `handleKey()` uses `KeyMap.Resolve()` instead of hardcoded switch cases
- `timeline.go` and `iterlist.go` refactored from `Update(tea.KeyMsg)` to `HandleAction(string)` — actions dispatched by root
- Arrow keys (`←`/`→`/`↑`/`↓`) always active as alternates via `HasAlternateArrowKey()`
- `ctrl+c` always quits (not configurable per spec)
- 22 tests in `keymap_test.go`: default bindings, parsing, resolve, remapping, sequence abort, TOML overrides, line_numbers config

### ~~1.2 Full Row Highlight~~ ✅ DONE (previously implemented)

Full-width row highlighting already works in `timeline.go` `renderWithLines()`.

### 1.3 Fix `--exit` Flag
- [ ] Debug why `--exit` hangs after iterations complete — investigate `subprocessExitMsg` handling in `root.go`
- [ ] Ensure `tea.Quit` is returned reliably after final iteration with `exitOnComplete`
- [ ] Verify no pending tick or event commands keep the program alive
- [ ] Tests: integration test that `--exit` model returns `tea.Quit` after last iteration completes

### 1.4 Auto-Hide Left Pane
- [ ] Add `leftPaneVisible` bool to `Model` in `root.go`, default `true`
- [ ] On `WindowSizeMsg`, set `leftPaneVisible = false` when width < 80 columns
- [ ] Add `[` keybind (via `KeyMap`) to toggle `leftPaneVisible`
- [ ] Update `View()` to render full-width right pane when left pane is hidden
- [ ] When left pane is hidden and focused, auto-switch focus to right pane
- [ ] Tests: pane hidden below 80 cols, toggle with `[`, focus auto-switches

## Phase 2 — Modals

### ~~2.1 Modal Infrastructure~~ ✅ DONE

Implemented in `internal/tui/modal.go` and `root.go`:
- `modalType` enum (`modalNone`, `modalQuitConfirm`, `modalHelp`)
- `activeModal` and `lastCtrlCAt` fields added to `Model`
- `handleKey()` checks `m.activeModal != modalNone` and routes to `handleModalKey()` — all normal keybindings are blocked while a modal is active
- `View()` renders the modal overlay instead of the normal TUI when a modal is active
- `centerOverlay()` utility centers any rendered block horizontally and vertically in the terminal
- `RenderQuitConfirmModal()` renders the quit confirmation with theme-aware styling

### ~~2.2 Quit Confirmation Modal~~ ✅ DONE

Implemented in `root.go` and `modal.go`:
- `q` (via `ActionQuit`) opens quit confirmation modal instead of quitting directly
- `ctrl+c` (single) shows the quit confirmation modal, records `lastCtrlCAt` timestamp
- `ctrl+c` (double within 500ms) force-quits immediately — no modal shown
- Modal keys: `y` → kill subprocess + `tea.Quit`; `n` or `esc` → dismiss modal; all others ignored
- `--exit` bypasses modal entirely (existing `exitOnComplete` path unchanged)
- 10 integration tests in `modal_test.go`: q shows modal, y quits, n dismisses, esc dismisses, other keys ignored, blocks navigation, single ctrl+c shows modal, double ctrl+c force-quits, expired ctrl+c window, --exit bypasses

**Bug fixed**: Bubble Tea uses `"esc"` not `"escape"` for the Escape key string. Updated `DefaultKeyMap()` to use `"esc"` and added `normalizeKeyName()` to `ParseKeyBinding()` so user config `escape = "escape"` still works. Added `DisplayString()` to `KeyBinding` for user-friendly display (maps `"esc"` back to `"escape"`).

### 2.3 Help Modal
- [ ] On `?` (via `KeyMap`): set `activeModal = help`
- [ ] Modal view: centered overlay listing all actions with their resolved key bindings from `KeyMap`
- [ ] Render sections: Navigation, Focus, Actions, View, Global
- [ ] Any key press dismisses the modal
- [ ] Tests: `?` opens modal, any key closes, displayed bindings match configured `KeyMap`

## Phase 3 — Content & Display

### 3.1 Remove Truncation from Expanded Content
- [ ] Remove `maxExpandedLines` constant and `truncateLines()` calls from `expand.go`
- [ ] Update `expandedContentLines()` to return all lines
- [ ] Update `toolCallLineCount()` in `cursor.go` to use actual content length (no cap)
- [ ] Update tests: remove assertions for "... N more lines ..." truncation, add tests for full content display

### 3.2 Sub-Scroll for Expanded Content
- [ ] Add `subScrollMode` bool and `subScrollOffset` int to `Timeline`
- [ ] Track which tool call is in sub-scroll mode (by flat cursor index)
- [ ] On `enter` for already-expanded tool call: enter sub-scroll mode
- [ ] Adaptive sizing: if content ≤ 40% of pane height, show inline; if > 40%, cap viewport at 70% of pane height
- [ ] In sub-scroll mode: `j`/`k` scroll within expanded content, `gg`/`G` jump within content
- [ ] `escape` exits sub-scroll mode, returns to timeline navigation
- [ ] Render scroll position indicator `[current/total]` in `ForegroundDim`
- [ ] Render subtle border around expanded area when in sub-scroll mode
- [ ] `q` in sub-scroll shows quit confirmation (not escape behavior)
- [ ] Tests: enter/exit sub-scroll, scroll within content, adaptive threshold, escape returns to timeline

### 3.3 Relative Line Numbers
- [ ] Add `lineNumbers` bool to `Model` (from config `view.line_numbers`, default `true`)
- [ ] Add `#` keybind (via `KeyMap`) to toggle `lineNumbers`
- [ ] In `timeline.go` `View()`, render a 4-column gutter with relative line numbers when enabled
- [ ] Line 0 = cursor position (rendered with `Highlight` color), others show distance in `ForegroundDim`
- [ ] Expanded content lines share their parent's line number (not individually numbered)
- [ ] Tests: gutter renders relative numbers, toggle with `#`, cursor at 0

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
- [ ] Add integration test: left pane auto-hide on narrow terminal
- [ ] Add integration test: sub-scroll enter/navigate/exit
- [ ] Add integration test: count+jump motions
- [ ] Add integration test: expand shows full content (no truncation)
- [ ] Add integration test: configurable keybindings apply end-to-end
