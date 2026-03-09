# Plan Files — Implementation Plan

## All Tasks Complete ✅

1. ~~Add `github.com/charmbracelet/glamour` dependency~~ ✅
2. ~~Create `internal/tui/planlist.go`~~ ✅ — PlanList component
3. ~~Create `internal/tui/planlist_test.go`~~ ✅ — 20 tests
4. ~~Create `internal/tui/planview.go`~~ ✅ — RenderPlanView with glamour rendering
5. ~~Create `internal/tui/planview_test.go`~~ ✅ — 11 tests
6-11. ~~Integrate plan pane into root.go~~ ✅ — paneID, focus cycle, layout, right pane mode, nav, editor, mouse
12. ~~Integration tests~~ ✅ — 9 new tests
13. ~~Help modal~~ ✅ — Added "Edit plan file" (e) to Actions section
14-17. ~~Spec verification~~ ✅ — Fixed prompt-files.md h/← description to include plan mode behavior

## Notes

- Glamour keeps `# ` prefix in terminal output — expected behavior.
- `planScrollPositions` map initialized but per-file tab-away/back persistence not wired. Scroll resets on cursor change per spec.
