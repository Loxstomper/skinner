# Implementation Plan: Decompose for Testability

## Context

The codebase had a monolithic `internal/tui/tui.go` (1,276 lines) containing all business logic, subprocess management, rendering, navigation, and formatting. The specs (`specs/architecture.md`) define a layered architecture with 7 packages and 13+ files, each independently testable. This plan decomposed the code bottom-up in incremental steps, keeping `make check` green at each step. Steps 1–8 are now complete.

## Remaining Steps

### Step 9: Integration test with fake executor

**`tui/integration_test.go`:**
- Construct root `Model` with `FakeExecutor` + canned event sequence
- Send `WindowSizeMsg`, process messages through `Update`
- Assert `View()` at stages: after tool uses, after results, after iteration end
- Test key nav: tab, j/k, enter expand/collapse, v toggle view mode
- Test multi-iteration: list grows, cursor follows

---

### Step 10: Update specs and documentation

- Update `specs/architecture.md` to match any deviations (e.g., `ShouldStartNext` parameter, event type naming, executor sentinel behavior)
- Update this file (`IMPLEMENTATION_PLAN.md`) to reflect completed work
- Run `make check` — verify all tests pass, no lint errors

## Key Files

| File | Role |
|------|------|
| `internal/tui/root.go` | Root TUI model (~430 lines, thin coordinator) |
| `internal/tui/header.go` | Header bar component (100 lines, pure `RenderHeader`) |
| `internal/tui/iterlist.go` | Iteration list component (167 lines, `IterList` struct) |
| `internal/tui/timeline.go` | Message timeline component (439 lines, `Timeline` struct) |
| `internal/tui/format.go` | Pure formatting helpers |
| `internal/tui/cursor.go` | Flat cursor math |
| `internal/tui/autofollow.go` | Auto-follow state machine |
| `internal/session/session.go` | Business logic controller |
| `internal/executor/claude.go` | Subprocess abstraction |
| `internal/model/model.go` | Data types (shared across layers) |
| `internal/parser/parser.go` | JSON parsing (consumed only by executor) |
| `specs/architecture.md` | Target architecture reference |

## Verification

After each step: `make check` (vet + lint + tests).
After step 9: manual smoke test — `make run` with a prompt file to verify identical behavior.
After step 10: full spec review pass.

## Previous Work (Completed)

- **Step 1: Extract format helpers** — `FormatDuration`, `FormatDurationValue`, `FormatTokens`, `ToolIcon`, `GroupSummaryUnit`, `IsKnownTool` moved to `tui/format.go` with full test coverage in `tui/format_test.go`.
- **Step 2: Extract cursor math** — `FlatCursorCount`, `FlatToItem`, `ItemToFlat`, `FlatCursorLineRange`, `ItemLineCount`, `TotalLines` extracted to `tui/cursor.go` as standalone functions. 26 tests in `tui/cursor_test.go`.
- **Context window percentage in header** — `ContextWindow` field added to `ModelPricing`, latest usage tracked per assistant event, header centred with `ctx N%` display color-coded by threshold.
- **Lint fixes** — golangci-lint v2 config migration, fixed `errcheck`, `gocritic`, and `nilerr` warnings.
- **Config tests** — `TestDefaultPricing`, `TestLoadConfig_ContextWindowFromTOML`, `TestLoadConfig_NoConfigFile` added.
- **Step 3: Extract auto-follow** — `AutoFollow` struct in `tui/autofollow.go` with `NewAutoFollow()`, `OnManualMove()`, `JumpToEnd()`, `OnNewItem()`, `Following()`. 8 tests in `tui/autofollow_test.go`. `autoFollowLeft`/`autoFollowRight` bools replaced with `AutoFollow` instances in `tui.go`.
- **Step 4: Add tests for model, parser, theme** — `model/model_test.go` (5 test fns, ~22 cases), `parser/parser_test.go` (10 test fns, ~50 cases), `theme/theme_test.go` (2 test fns, ~9 cases). All packages now have full test coverage.
- **Step 5: Create session controller** — `session/events.go` (5 event types + Event interface), `session/session.go` (Controller with 10 methods), `session/session_test.go` (23 tests covering grouping, result matching, usage/cost, iteration lifecycle, full end-to-end).
- **Step 6: Create executor** — `executor/executor.go` (Executor interface), `executor/claude.go` (ClaudeExecutor with readEvents), `executor/fake.go` (FakeExecutor), `executor/executor_test.go` (13 tests). Added `AssistantBatchEvent` and `SubprocessExitEvent` to `session/events.go`.
- **Step 7: Wire controller + executor into TUI** — Replaced inline business logic with `session.Controller` calls, replaced subprocess code with `executor.Executor`. Message types wrap `session.*Event`. `parser` import removed from `tui`. `main.go` constructs `ClaudeExecutor`. `tui.go` reduced to ~730 lines.
- **Step 8: Split TUI rendering into component files** — Decomposed `tui.go` (880 lines) into 4 files matching `specs/architecture.md`: `root.go` (432 lines, thin coordinator owning Controller, Executor, IterList, Timeline, focus, dimensions, global keys), `header.go` (100 lines, pure `RenderHeader(HeaderProps) string` function), `iterlist.go` (167 lines, `IterList` struct with Cursor, AutoFollow, Update/View/OnNewIteration/JumpToTop/JumpToBottom), `timeline.go` (439 lines, `Timeline` struct with Cursor, Scroll, AutoFollow, Update/View/OnNewItems/JumpToTop/JumpToBottom/ResetPosition, plus `renderTextBlockLines`, `renderToolCallLine`, `renderGroupHeaderLine`). Tests: `header_test.go` (5 test fns covering duration/tokens, context % thresholds, cost visibility, iteration progress, status icons), `iterlist_test.go` (11 test fns covering cursor movement, bounds, auto-follow, OnNewIteration, jump, view with 0/1/N iterations, page navigation), `timeline_test.go` (16 test fns covering cursor movement, bounds, enter toggle for text blocks and groups, auto-follow, OnNewItems, jump, reset, view with empty/tool calls/text blocks/compact/group collapsed/expanded, scroll). `make check` green. Old `tui.go` deleted.
