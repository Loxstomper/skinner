# Plan Files — Implementation Plan

## Completed

1. ~~Add `github.com/charmbracelet/glamour` dependency~~ ✅
2. ~~Create `internal/tui/planlist.go`~~ ✅ — PlanList component
3. ~~Create `internal/tui/planlist_test.go`~~ ✅ — 20 tests
4. ~~Create `internal/tui/planview.go`~~ ✅ — RenderPlanView with glamour rendering
5. ~~Create `internal/tui/planview_test.go`~~ ✅ — 11 tests
6-11. ~~Integrate plan pane into root.go~~ ✅ — paneID, focus cycle, layout, right pane mode, nav, editor, mouse
12. ~~Integration tests~~ ✅ — 9 new tests: pane swapping, focus restore, scroll reset, editor key, nav, mouse click
13. ~~Help modal~~ ✅ — Added "Edit plan file" (e) to Actions section

## Remaining

14. Verify `specs/plan-files.md` completeness after implementation
15. Verify `specs/keybindings.md` after implementation
16. Verify `specs/tui-layout.md` after implementation
17. Verify `specs/prompt-files.md` after implementation

## Notes

- `planScrollPositions` map is initialized but per-file save/restore not yet wired — scroll resets on cursor change (matches spec: "scroll position resets to the top" when moving between plans). Tab-away/back persistence could be added later.
- Glamour keeps `# ` prefix in terminal output (not stripped) — this is expected behavior.
