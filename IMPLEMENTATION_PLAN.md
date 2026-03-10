# Implementation Plan: Bottom Layout & Path Trimming

## ~~1. Add `Layout` config field~~ ✅ DONE

Implemented: `Layout string` on Config struct, parsed from `view.layout` TOML, default `"auto"`, invalid values fall back to default. Tests cover all three valid values, default, and invalid fallback.

## ~~2. Add path trimming utility~~ ✅ DONE

Implemented: `TrimPath(path, cwd)` and `TrimSummaryPath(summary, toolName, cwd)` in `format.go`. Rules: strip CWD+/ prefix first, then $HOME+/ → ~/. Tests cover CWD stripping, HOME fallback, non-dir-boundary prefixes, trailing slash, and paths outside both.

## ~~3. Apply path trimming to tool call summaries~~ ✅ DONE

Implemented: Added `WorkDir` field to `TimelineProps`, passed from `Model.workDir`. `renderToolCallLine` applies `TrimSummaryPath` to summaries for Read/Edit/Write/Grep/Glob. No separate expanded headers needed — paths appear only in summary lines. Note: expand.go does not show path headers separately, so no changes needed there.

## 4. Add layout mode to TUI model

- **File**: `internal/tui/root.go`
- Add `layoutMode string` field to `Model` (resolved from config + terminal width)
- Add `effectiveLayout() string` method — returns `"side"` or `"bottom"` based on config and current width
- Call on init and on every `tea.WindowSizeMsg`

## 5. Implement bottom bar rendering

- **File**: `internal/tui/root.go`
- In `View()`, branch on `effectiveLayout()`:
  - `"side"`: existing rendering path (unchanged)
  - `"bottom"`: render header, then main area (full width), then bottom bar
- Bottom bar renderer: 3 labeled dividers + 2-line sections for plans, iterations, prompts
- **File**: `internal/tui/iterlist.go`
- Add `ViewBottom(props)` method — renders 2 visible rows (no run separators), reuses existing item rendering
- **File**: `internal/tui/planlist.go`
- Add `ViewBottom(props)` method — renders 2 visible rows with label divider
- **File**: `internal/tui/promptlist.go`
- Add `ViewBottom(props)` method — renders 2 visible rows with label divider
- Layout height helpers: `BottomBarHeight() int` returning 9 (3 dividers + 6 content)

## 6. Update focus cycling for bottom layout

- **File**: `internal/tui/root.go`
- In `ActionFocusToggle` handler, branch on `effectiveLayout()`:
  - Side: Plans → Iterations → Prompts → Timeline (existing)
  - Bottom: Timeline → Plans → Iterations → Prompts → Timeline
- Update `ActionFocusLeft`/`ActionFocusRight` for bottom layout:
  - `h`/`←`: main area → last-focused bottom bar section
  - `l`/`→`: bottom bar → main area

## 7. Update mouse handling for bottom layout

- **File**: `internal/tui/root.go`
- In `handleMouse()`, branch on `effectiveLayout()`:
  - Side: existing X-coordinate logic
  - Bottom: use Y-coordinate to determine main area vs bottom bar section
- Add `bottomBarSectionAtRow(y int) paneID` helper — maps Y to plans/iterations/prompts based on divider positions
- **File**: `internal/tui/root.go`
- Update scroll and click handlers to use the new targeting

## 8. Update `[` toggle for bottom layout

- **File**: `internal/tui/root.go`
- `ActionToggleLeftPane` handler: in bottom mode, toggle bottom bar visibility instead of left pane
- Add `bottomBarVisible bool` field (or reuse `leftPaneVisible`)

## 9. Update pane dimension calculations

- **File**: `internal/tui/root.go`
- `rightPaneWidth()`: in bottom mode, always full terminal width (no left pane)
- `rightPaneHeight()`: in bottom mode, `height - 1 - bottomBarHeight` (header + bottom bar)
- When bottom bar is hidden, main area gets full height

## 10. Tests for bottom layout

- **File**: `internal/tui/integration_test.go`
- Add helper to create test model with bottom layout config
- **File**: `internal/tui/root_test.go` (or new `bottom_layout_test.go`)
- Test `effectiveLayout()` returns correct mode for auto/side/bottom at various widths
- Test `View()` output contains bottom bar sections when in bottom mode
- Test focus cycling order in bottom mode
- Test `[` toggles bottom bar
- Test focus preserved across layout switch (auto mode resize)
- **File**: `internal/tui/iterlist_test.go`
- Test `ViewBottom()` renders 2 rows, no run separators
- **File**: `internal/tui/mouse_test.go` (or existing test file)
- Test Y-coordinate section targeting in bottom mode

## 11. Tests for path trimming

- **File**: `internal/tui/format_test.go`
- `TrimPath` unit tests (covered in task 2)
- **File**: `internal/tui/timeline_test.go`
- Integration test: tool call with absolute path renders trimmed in summary
- Test expanded view header shows trimmed path

## 12. Update specs (already done)

- `specs/bottom-layout.md` — created
- `specs/path-trimming.md` — created
- `specs/README.md` — updated
- `specs/tui-layout.md` — updated
- `specs/config.md` — updated
- `specs/keybindings.md` — updated
- `specs/mouse.md` — updated
- `specs/stream-json-format.md` — updated
