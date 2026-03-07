# Implementation Plan: Decompose for Testability

## Context

The codebase had a monolithic `internal/tui/tui.go` (1,276 lines) containing all business logic, subprocess management, rendering, navigation, and formatting. The specs (`specs/architecture.md`) define a layered architecture with 7 packages and 13+ files, each independently testable. This plan decomposed the code bottom-up in incremental steps, keeping `make check` green at each step. Steps 1–10 are now complete.

## Known Issues / Future Work

### Left pane scroll/viewport for many iterations
The left pane (`iterlist.go`) renders all iterations without scroll logic. If there are more iterations than fit in the available height, Lipgloss truncates the overflow but there is no scroll offset to control which iterations are visible. The cursor can move beyond the visible area but the view won't follow. A viewport/scroll offset similar to the timeline's approach is needed.

### PgDown/PgUp in timeline doesn't adjust cursor to viewport
After page scrolling in the timeline, the cursor may be above/below the viewport. The highlighted row won't be visible. Consider moving the cursor into the visible viewport after page scroll.

### `--exit` flag is undocumented
An `--exit` CLI flag exists in `main.go` that causes the TUI to quit when all iterations complete. This is not documented in `specs/iteration-loop.md`. Should be added to the spec or removed.

### Edit line info spec example `(+3/-1)` is impossible
The spec example `(+3/-1)` in `stream-json-format.md` cannot be produced by the algorithm described (counting newlines in old_string vs new_string). The algorithm produces net additions OR net removals, never both positive simultaneously. The only case where both appear is net zero `(+N/-N)`. The spec example is misleading.

## Key Files

| File | Role |
|------|------|
| `internal/tui/root.go` | Root TUI model (~430 lines, thin coordinator) |
| `internal/tui/header.go` | Header bar component (100 lines, pure `RenderHeader`) |
| `internal/tui/iterlist.go` | Iteration list component (~170 lines, `IterList` struct) |
| `internal/tui/timeline.go` | Message timeline component (439 lines, `Timeline` struct) |
| `internal/tui/format.go` | Pure formatting helpers |
| `internal/tui/cursor.go` | Flat cursor math |
| `internal/tui/autofollow.go` | Auto-follow state machine |
| `internal/tui/integration_test.go` | Integration tests with FakeExecutor (18 tests) |
| `internal/session/session.go` | Business logic controller |
| `internal/executor/claude.go` | Subprocess abstraction |
| `internal/model/model.go` | Data types (shared across layers) |
| `internal/parser/parser.go` | JSON parsing (consumed only by executor) |
| `specs/architecture.md` | Target architecture reference |

## Verification

After each step: `make check` (vet + lint + tests).

## Completed Work

- **Step 1: Extract format helpers** — `FormatDuration`, `FormatDurationValue`, `FormatTokens`, `ToolIcon`, `GroupSummaryUnit`, `IsKnownTool` moved to `tui/format.go` with full test coverage.
- **Step 2: Extract cursor math** — `FlatCursorCount`, `FlatToItem`, `ItemToFlat`, `FlatCursorLineRange`, `ItemLineCount`, `TotalLines` extracted to `tui/cursor.go`. 26 tests.
- **Context window percentage in header** — `ContextWindow` field, latest usage tracking, color-coded `ctx N%` display.
- **Lint fixes** — golangci-lint v2 config migration.
- **Config tests** — `TestDefaultPricing`, `TestLoadConfig_ContextWindowFromTOML`, `TestLoadConfig_NoConfigFile`.
- **Step 3: Extract auto-follow** — `AutoFollow` struct with state machine. 8 tests.
- **Step 4: Add tests for model, parser, theme** — Full test coverage for all packages.
- **Step 5: Create session controller** — `Controller` with 10 methods, 23 tests.
- **Step 6: Create executor** — `Executor` interface, `ClaudeExecutor`, `FakeExecutor`, 13 tests.
- **Step 7: Wire controller + executor into TUI** — Replaced inline business logic and subprocess code.
- **Step 8: Split TUI rendering into component files** — `root.go`, `header.go`, `iterlist.go`, `timeline.go` with full test suites.
- **Step 9: Integration test with fake executor** — 18 integration tests covering full TUI loop.
- **Step 10: Spec review and bug fixes** — Fixed iteration duration to show live elapsed time with `...` suffix (was showing just `...`). Updated `specs/duration-tracking.md` to clarify tool call vs iteration in-progress format. Updated `specs/architecture.md` to document `AssistantBatchEvent` and `SubprocessExitEvent`. Added 2 new iterlist tests for duration display. Documented known issues for future work.
