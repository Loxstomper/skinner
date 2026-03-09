# Plan Files — Implementation Plan

## Tasks

1. ~~Add `github.com/charmbracelet/glamour` dependency~~ ✅ Done

2. ~~Create `internal/tui/planlist.go`~~ ✅ Done

3. ~~Create `internal/tui/planlist_test.go`~~ ✅ Done — 20 tests

4. ~~Create `internal/tui/planview.go`~~ ✅ Done — `RenderPlanView(PlanViewProps)` with glamour rendering, title bar, scroll, file-not-found, `ClampPlanScroll()`, `renderMarkdown()`. Note: glamour keeps `# ` prefix in terminal output (not stripped).

5. ~~Create `internal/tui/planview_test.go`~~ ✅ Done — 11 tests covering title, glamour output, file-not-found, empty filename, scroll clamping, word wrap, zero size, height-one edge case

6-11. ~~Integrate plan pane into root.go~~ ✅ Done — All of the following implemented:
   - `plansPane` in paneID enum, `rightPaneModeType` (planMode/timelineMode)
   - `planList PlanList`, `planViewScroll`, `planViewTotalLines`, `planScrollPositions` fields
   - Focus cycle: Plans → Iterations → Prompts → Timeline → Plans
   - `ActionFocusLeft`: from plan content → plans, from timeline → iterations
   - Right pane switches between plan content view and timeline based on mode
   - All nav keys (j/k, gg/G, pgup/pgdn) route to plan list or plan view scroll
   - `e` key launches `$EDITOR` for plan files from plan list or plan content view
   - `planEditorDoneMsg` rescans files and restores plan content focus
   - Mouse: plan section detection, click/scroll handling, iteration row offset adjustment
   - `iterListHeight()` accounts for plan section (5 rows) + extra divider (1 row)
   - Left pane layout: planView + divider + iterView + divider + promptView
   - Tick rescans both plan and prompt files
   - Integration tests updated: 4-pane tab cycle, mouse click Y offsets for plan section
   - Note: `planScrollPositions` map initialized but per-file save/restore not yet wired (scroll resets on cursor change)

12. Remaining integration tests
    - Plan selection swaps right pane content
    - Focus on iterations restores timeline
    - Scroll persistence when tabbing away and back
    - `e` key triggers editor command

13. Update `internal/tui/modal.go` — help modal
    - Add plan-related keybindings to help modal display
    - Update focus cycle description

14. Update `specs/plan-files.md` — already created, verify completeness after implementation

15. Update `specs/keybindings.md` — already updated, verify after implementation

16. Update `specs/tui-layout.md` — already updated, verify after implementation

17. Update `specs/prompt-files.md` — already updated, verify after implementation
