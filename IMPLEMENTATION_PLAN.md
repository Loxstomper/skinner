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

## ~~6. Update focus cycling for bottom layout~~ ✅ DONE (implemented as part of task 4)

## ~~7. Update mouse handling for bottom layout~~ ✅ DONE

Implemented: `handleMouse()` branches on `effectiveLayout()`. Bottom layout uses Y-coordinate targeting: events above `rightPaneHeight()` target the timeline/plan view; events in the bottom bar region map to Plans (offset 1-2), Iterations (offset 4-5), or Prompts (offset 7-8) content rows. Divider line clicks (offsets 0, 3, 6) are ignored. `handleBottomBarClick()` helper handles click row mapping: plans/prompts use `ClickRow(contentRow+1)` to compensate for missing title row; iterations pass `nil` runs to skip separator logic. Scroll events in the bottom bar use `bottomBarSectionHeight` and `nil` runs. 15 tests in `layout_test.go` cover: main area clicks/scrolls, section targeting, divider ignoring, item selection, plan scroll reset, timeline reset, hidden bar fallback, and header ignoring.

## ~~8. Update `[` toggle for bottom layout~~ ✅ DONE (implemented as part of task 4)

## ~~9. Update pane dimension calculations~~ ✅ DONE (implemented as part of task 4)

## ~~10. Tests for bottom layout~~ ✅ DONE

All tests implemented in `layout_test.go`: effectiveLayout modes, threshold, resize, pane dimensions, toggle, focus cycle, focus preservation, View output, ViewBottom methods, and mouse handling (15 mouse tests).

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
