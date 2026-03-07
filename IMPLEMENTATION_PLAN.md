# Implementation Plan: Decompose for Testability

## Context

The codebase has a monolithic `internal/tui/tui.go` (1,276 lines) containing all business logic, subprocess management, rendering, navigation, and formatting. The specs (`specs/architecture.md`) define a layered architecture with 7 packages and 13+ files, each independently testable. Only `internal/config` has tests today. This plan decomposes the code bottom-up in incremental steps, keeping `make check` green at each step.

## Steps

### ~~Step 1: Extract pure formatting helpers → `tui/format.go`~~ ✅ DONE

Extracted `FormatDuration`, `FormatDurationValue`, `FormatTokens`, `ToolIcon`, `GroupSummaryUnit`, and new `IsKnownTool` into `tui/format.go`. All call sites in `tui.go` updated to exported names. Originals deleted. `tui/format_test.go` has table-driven tests covering all functions. `tui.go` reduced from 1,276 to ~1,220 lines.

---

### ~~Step 2: Extract flat cursor math → `tui/cursor.go`~~ ✅ DONE

Extracted all 6 cursor functions to standalone functions in `tui/cursor.go`: `FlatCursorCount`, `FlatToItem`, `ItemToFlat`, `FlatCursorLineRange`, `ItemLineCount`, `TotalLines`. Added `selectedItems()` helper to Model. All call sites updated. `tui/cursor_test.go` has 26 tests covering empty, standalone, expanded/collapsed groups, mixed items, text blocks, compact view, out-of-range, and roundtrip consistency. `tui.go` reduced from ~1,220 to ~1,080 lines.

---

~~Step 3: Extract auto-follow state machine → `tui/autofollow.go`~~ ✅ DONE

Extracted `AutoFollow` struct with `NewAutoFollow()`, `OnManualMove(atEnd bool)`, `JumpToEnd()`, `OnNewItem()`, `Following()` into `tui/autofollow.go`. Replaced `autoFollowLeft bool` / `autoFollowRight bool` fields in `Model` with `AutoFollow` instances. All 12 call sites updated. `tui/autofollow_test.go` has 8 tests covering: starts following, manual move pauses, move-at-end keeps, resume at end, jump resumes, new item doesn't resume, new item doesn't disrupt, full pause-jump-pause sequence. `tui.go` reduced from ~1,080 to ~1,060 lines.

---

### ~~Step 4: Add tests for `model`, `parser`, `theme`~~ ✅ DONE

Added comprehensive table-driven tests for all three packages:

**`model/model_test.go`:** 5 test functions covering `ToolCallGroup.Status()` (8 cases: all-running, all-done, mixed running+done, error+done, error+running, all-error, single), `GroupDuration()` (5 cases: running returns 0, single child, overlapping span, sequential, error children), `CompletedCount()` (4 cases), `ToolCallCount()` (2 cases), `Iteration.ToolCallCount()` (5 cases: empty, standalone, groups, mixed, text blocks ignored).

**`parser/parser_test.go`:** 10 test functions covering `ParseStreamEvent` with empty/invalid/unknown input, result type, assistant tool_use with usage, text events, empty text skipped, mixed content, user tool_result (success + error), `extractToolSummary` (11 cases per tool type), `extractToolUseLineInfo` (7 cases for Edit/Write), `extractToolResultLineInfo` (5 cases), `truncate` (3 cases).

**`theme/theme_test.go`:** 2 test functions covering `LookupTheme` (existing themes with field validation, missing, empty name) and `ThemeNames` (count, sorted order, roundtrip lookup).

---

### ~~Step 5: Create `internal/session` — business logic controller~~ ✅ DONE

Created `internal/session` package with two files:

**`session/events.go`** — typed event definitions: `Event` interface, `ToolUseEvent`, `ToolResultEvent`, `TextEvent`, `UsageEvent`, `IterationEndEvent`, `AssistantBatchEvent`, `SubprocessExitEvent`. The last two were added in Step 6 to support the executor layer.

**`session/session.go`** — `Controller` struct with `NewController()`, `ProcessAssistantBatch()`, `ProcessToolResult()`, `ProcessUsage()`, `StartIteration()`, `CompleteIteration()`, `ShouldStartNext()`, `RunningIterationIdx()`, `HasKnownModel()`, `LastModel()`. Injectable `Clock func() time.Time` (defaults to `time.Now`).

**`session/session_test.go`** — 23 tests (8 for batch processing covering single/grouped/mixed/text-breaks/empty/no-iteration, 6 for tool result matching covering standalone/group-child/error/Read-lineinfo/Edit-keeps-lineinfo/not-found, 3 for usage covering known-model-cost/unknown-model/accumulation, 4 for iteration lifecycle covering start/complete-success/complete-failed/no-running, table-driven tests for `ShouldStartNext` (7 cases) and `RunningIterationIdx` (5 cases), plus default-clock and full end-to-end lifecycle tests).

---

### ~~Step 6: Create `internal/executor` — subprocess abstraction~~ ✅ DONE

Created `internal/executor` package with four files:

**`executor/executor.go`** — `Executor` interface with `Start(ctx, prompt) (<-chan session.Event, error)` and `Kill() error`.

**`executor/claude.go`** — `ClaudeExecutor` that spawns `claude -p --dangerously-skip-permissions --output-format=stream-json --verbose`, pipes prompt to stdin, reads stdout line-by-line via `bufio.Scanner` (10MB buffer), converts `parser.*Event` → `session.*Event`. ToolUseEvent and TextEvent from a single assistant message are grouped into `session.AssistantBatchEvent`; UsageEvent, ToolResultEvent, and IterationEndEvent are sent individually. Channel is closed after subprocess exits and a `SubprocessExitEvent` is sent. Only package that imports `parser`.

**`executor/fake.go`** — `FakeExecutor` with pre-loaded `[]session.Event`, optional `Delay`, records `Prompt`. Supports context cancellation for delayed delivery.

**`session/events.go`** — Added `AssistantBatchEvent` (wraps `[]Event` for batch processing) and `SubprocessExitEvent` (signals subprocess exit with optional error).

**`executor/executor_test.go`** — 13 tests: FakeExecutor (delivers events, records prompt, empty events, delay+cancel, Kill no-op), interface compliance (both impls), readEvents (assistant batch with usage, tool result success/error, iteration end, skips invalid, multiple lines, tool summary extraction, empty input).

---

### Step 7: Wire controller + executor into TUI

Modify `tui.go`:
- Add `session.Controller` and `executor.Executor` fields to `Model`
- Remove `eventCh`, `cmd`, `hasKnownModel`, `lastModel` fields
- Replace inline business logic in `Update` with controller calls
- Replace subprocess code with executor calls
- Bridge executor channel → Bubble Tea messages: the executor sends `session.Event` values (including `AssistantBatchEvent`, `UsageEvent`, `ToolResultEvent`, `IterationEndEvent`, `SubprocessExitEvent`). The TUI adapter goroutine wraps each into a `tea.Msg` and sends to Bubble Tea.
- The existing `assistantBatchMsg`, `toolResultMsg`, `usageMsg`, `iterationEndMsg`, `subprocessExitMsg` types become thin wrappers around session events.
- Remove `parser` import (now internal to executor)

Modify `cmd/skinner/main.go`:
- Construct `ClaudeExecutor` and pass to `NewModel`

Verify: `make check` still green.

---

### Step 8: Split TUI rendering into component files

**`tui/header.go`** — `HeaderProps` struct + `RenderHeader(HeaderProps) string` pure function. Extract from `viewHeader`.

**`tui/iterlist.go`** — `IterList` struct (cursor + AutoFollow) + `IterListProps`. Extract `renderLeftPane` → `View`, left-pane key handling → `Update`, auto-follow hook `OnNewIteration`.

**`tui/timeline.go`** — `Timeline` struct (cursor + scroll + AutoFollow) + `TimelineProps`. Extract `renderRightPane*`, `renderTextBlockLines`, `renderToolCallLine`, `renderGroupHeaderLine` → `View`, right-pane key handling → `Update`, scroll helpers, `OnNewItems`.

**Rename `tui.go` → `tui/root.go`** — thin coordinator: owns Controller, Executor, IterList, Timeline, focus, dimensions, global keys (q, ctrl+c, tab, h/l, v, gg/G). `View()` calls `RenderHeader` + `IterList.View` + `Timeline.View`.

**Tests:**
- `header_test.go` — `RenderHeader` with various props, assert substrings (duration, tokens, cost, iteration count, status icon)
- `iterlist_test.go` — cursor movement, auto-follow, rendering with 0/1/N iterations
- `timeline_test.go` — cursor movement, enter toggle, scroll, expand/collapse, rendering

---

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
| `internal/tui/tui.go` | Monolith being decomposed (~1,220 lines) |
| `internal/tui/format.go` | Pure formatting helpers (extracted Step 1) |
| `internal/session/session.go` | **New** — business logic controller |
| `internal/executor/claude.go` | **New** — subprocess abstraction |
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
