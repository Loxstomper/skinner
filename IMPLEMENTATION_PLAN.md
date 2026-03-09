# Plan Files — Implementation Plan

## Tasks

1. ~~Add `github.com/charmbracelet/glamour` dependency~~ — needed when planview is built

2. ~~Create `internal/tui/planlist.go`~~ ✅ Done — `PlanList` component with `ScanFiles`, `HandleAction`, `View`, `ClickRow`, `ScrollBy`, `IsInPlanSection`, `PlanSectionRow`, `PlanDisplayName`

3. ~~Create `internal/tui/planlist_test.go`~~ ✅ Done — 20 tests covering discovery, sorting, display name, navigation, scrolling, click, empty state, section detection

4. Create `internal/tui/planview.go` — plan content view for right pane
   - `RenderPlanView(PlanViewProps)` — glamour-render markdown with `auto` style, word-wrap to pane width
   - Title bar with centered filename
   - Scroll state: `planViewScroll int`
   - Navigation: j/k, gg/G, pgup/pgdn
   - File-not-found handling: dimmed message

5. Create `internal/tui/planview_test.go` — tests for plan content view
   - Title bar rendering with centered filename
   - Glamour rendering produces output
   - Scroll position clamping
   - Word wrap respects pane width
   - File-not-found dimmed message

6. Update `internal/tui/root.go` — integrate plan pane
   - Add `plansPane` to `paneID` enum (before `iterationsPane`)
   - Add `planList PlanList` field to `Model`
   - Add `rightPaneMode` state tracking which left pane last had focus (`planMode` vs `timelineMode`)
   - Add `planViewScroll int` and `planViewContent string` fields
   - Add `planScrollPositions map[string]int` for per-file scroll persistence

7. Update `internal/tui/root.go` — focus cycle
   - `ActionFocusToggle` (tab): Plans → Iterations → Prompts → Timeline → Plans
   - `ActionFocusLeft` (h/←): from timeline → iterations; from plan content → plans
   - `ActionFocusRight` (l/→): from any left pane → right pane
   - When focus enters plans pane, set `rightPaneMode = planMode`
   - When focus enters iterations or prompts pane, set `rightPaneMode = timelineMode`

8. Update `internal/tui/root.go` — left pane layout
   - `View()`: render planList + divider + iterList + divider + promptList
   - Subtract plan section height (5 rows) and extra divider (1 row) from iteration list height
   - Tick handler: call `planList.ScanFiles()` alongside `promptList.ScanFiles()`

9. Update `internal/tui/root.go` — right pane rendering
   - When `rightPaneMode == planMode`: render `RenderPlanView()` instead of timeline
   - When `rightPaneMode == timelineMode`: render timeline as before
   - On plan cursor change: reset scroll to top, cache rendered content
   - On focus change to plans: save current plan scroll, load new plan scroll

10. Update `internal/tui/root.go` — plan pane key handling
    - Route navigation keys to `planList.HandleAction()` when plans pane focused
    - Route navigation keys to plan view scroll when right pane focused in plan mode
    - `e` key: launch `$EDITOR` with selected plan file (from plan list or plan content view)
    - On editor return: re-read and re-render plan file, restore plan content focus

11. Update `internal/tui/root.go` — mouse support
    - Detect clicks/scrolls above the first divider → target plan list
    - Adjust existing prompt section detection to account for new plan section + divider

12. Update `internal/tui/integration_test.go` — integration tests
    - Tab cycles through all four panes
    - Plan selection swaps right pane content
    - Focus on iterations restores timeline
    - Scroll persistence when tabbing away and back
    - Scroll reset when switching between plans
    - `e` key triggers editor command

13. Update `internal/tui/modal.go` — help modal
    - Add plan-related keybindings to help modal display
    - Update focus cycle description

14. Update `specs/plan-files.md` — already created, verify completeness after implementation

15. Update `specs/keybindings.md` — already updated, verify after implementation

16. Update `specs/tui-layout.md` — already updated, verify after implementation

17. Update `specs/prompt-files.md` — already updated, verify after implementation
