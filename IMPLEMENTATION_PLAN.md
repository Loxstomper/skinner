# Implementation Plan: Bottom Layout & Path Trimming

All tasks complete. Summary of what was implemented:

1. **Layout config field** — `Layout string` on Config, parsed from `view.layout` TOML
2. **Path trimming utility** — `TrimPath` and `TrimSummaryPath` in `format.go`
3. **Path trimming in summaries** — `WorkDir` on `TimelineProps`, applied in `renderToolCallLine`
4. **Layout mode** — `effectiveLayout()`, auto threshold at 80 cols, focus cycling, pane dimensions
5. **Bottom bar rendering** — `renderBottomBar()`, `ViewBottom()` on list components
6. **Focus cycling** — bottom-layout cycle order (part of task 4)
7. **Mouse handling** — Y-coordinate section targeting for bottom bar
8. **Toggle support** — `[` toggles bottom bar in bottom mode (part of task 4)
9. **Pane dimensions** — `rightPaneWidth()`/`rightPaneHeight()` (part of task 4)
10. **Bottom layout tests** — `layout_test.go` with mouse, rendering, and focus tests
11. **Path trimming integration tests** — 6 tests in `timeline_test.go` covering CWD trimming, HOME fallback, no-WorkDir passthrough, Edit/Grep tools, and group children
12. **Specs** — all specs updated

## Notes

- `expand.go` does not render separate path headers, so no expanded-view path trimming tests were needed. Paths only appear in the summary line rendered by `renderToolCallLine`.
- Unit tests for `TrimPath`/`TrimSummaryPath` are in `format_test.go` (covered in task 2).
