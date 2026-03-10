# Implementation Plan: Mouse Wheel Scrolling in Plan Content View

All tasks completed.

## Completed

- [x] Update specs
  - [x] `specs/plan-files.md` — add Plan Content View mouse support subsection
  - [x] `specs/mouse.md` — note right-pane scroll targets plan content in plan mode

- [x] Update `internal/tui/root.go` `handleMouse` — in both wheel-up and wheel-down `default` branches, check `m.rightPaneMode == planMode`: if true, adjust `m.planViewScroll` by `∓mouseScrollLines` and clamp with `ClampPlanScroll`; otherwise scroll timeline as before

- [x] Add integration test `TestIntegration_PlanMouseWheelScrollsPlanContent` — covers scroll down/up, clamping at 0 and max, and focus switching to right pane
