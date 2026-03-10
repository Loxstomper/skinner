# Implementation Plan: Bottom Layout & Path Trimming

## ~~1. Add `Layout` config field~~ ✅ DONE

Implemented: `Layout string` on Config struct, parsed from `view.layout` TOML, default `"auto"`, invalid values fall back to default. Tests cover all three valid values, default, and invalid fallback.

## ~~2. Add path trimming utility~~ ✅ DONE

Implemented: `TrimPath(path, cwd)` and `TrimSummaryPath(summary, toolName, cwd)` in `format.go`. Rules: strip CWD+/ prefix first, then $HOME+/ → ~/. Tests cover CWD stripping, HOME fallback, non-dir-boundary prefixes, trailing slash, and paths outside both.

## ~~3. Apply path trimming to tool call summaries~~ ✅ DONE

Implemented: Added `WorkDir` field to `TimelineProps`, passed from `Model.workDir`. `renderToolCallLine` applies `TrimSummaryPath` to summaries for Read/Edit/Write/Grep/Glob. No separate expanded headers needed — paths appear only in summary lines. Note: expand.go does not show path headers separately, so no changes needed there.

## ~~4. Add layout mode to TUI model~~ ✅ DONE

Implemented: `effectiveLayout()` method returns `"side"` or `"bottom"` based on `config.Layout` and terminal width (auto threshold: 80 columns). `updateLayoutForSize()` called on every `WindowSizeMsg` to set `leftPaneVisible` and handle layout transitions. Added `bottomBarVisible` field for bottom bar toggle. Updated `ActionToggleLeftPane` to toggle bottom bar in bottom mode. Updated `ActionFocusToggle` with bottom-layout cycle order (Timeline → Plans → Iterations → Prompts → Timeline). Updated `ActionFocusLeft`/`ActionFocusRight` for bottom layout navigation. Updated `rightPaneWidth()` (full width in bottom) and `rightPaneHeight()` (subtracts `BottomBarHeight=9` when bar visible). Focus preserved across layout switches. Tests in `layout_test.go` cover all modes, threshold, resize, pane dimensions, toggle, focus cycle, and focus preservation.

## ~~5. Implement bottom bar rendering~~ ✅ DONE

Implemented: `renderBottomBar()` method in root.go renders Plans, Iterations, and Prompts sections with labeled dividers. `ViewBottom()` added to IterList (no run separators), PlanList (no title), and PromptList (no title) — each renders 2 compact rows. `View()` uses `rightPaneHeight()` for right pane height (accounts for bottom bar). `renderLabeledDivider()` renders `── Label ────` divider lines. Tests in layout_test.go cover: View contains bottom bar sections, hidden bar, no left pane in bottom mode, no bottom bar in side mode, ViewBottom methods.

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
