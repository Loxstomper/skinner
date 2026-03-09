# Implementation Plan: Interactive Run Start

## Completed

### 1. Add `Run` struct and session phase to model ✓

**File:** `internal/model/model.go`

- Added `SessionPhase` type with constants: `PhaseIdle`, `PhaseRunning`, `PhaseFinished`
- Added `Run` struct: `PromptName`, `PromptFile`, `StartIndex`, `MaxIterations`
- Added `Runs []Run`, `Phase SessionPhase`, `AccumulatedDuration time.Duration` fields to `Session`
- Tests added for `SessionPhase` constants, `Run` struct, and `Session` runs/phase

### 5. Update header for idle state ✓

**File:** `internal/tui/header.go`

- Added `Phase SessionPhase` field to `HeaderProps`
- When phase is `Idle`: renders `⏱ --` and `Idle` only, omits tokens/cost/context/rate limits/iteration counter
- When phase is `Running`/`Finished`: renders as before
- Wired `controller.Phase()` into `HeaderProps` in `root.go`
- All existing tests updated to set `Phase: model.PhaseRunning` (zero value is `PhaseIdle`)
- Tests added: idle phase, running phase full stats, finished phase full stats

### 2. Add session phase and run methods to controller ✓

**File:** `internal/session/session.go`

- Added `Phase() SessionPhase` method
- Added `StartRun(promptName, promptFile string, maxIterations int)` method
- Updated `CompleteIteration()` to set phase to `Finished` when run exhausted or on failure
- Updated `ShouldStartNext()` with per-run limits (backward compatible with legacy no-run path)
- Tests: `Phase()`, `StartRun()`, multi-run, per-run limits, phase transitions, accumulated duration

**Design note:** `ShouldStartNext()` has two paths — legacy (no runs, uses global `MaxIterations`) and run-based (checks `Phase == PhaseRunning` and per-run `MaxIterations` relative to `Run.StartIndex`). `CompleteIteration()` only manages phase transitions when runs are present. This keeps backward compatibility with existing TUI code that doesn't use `StartRun()` yet.

### 3. Update CLI arg parsing for idle mode ✓

**File:** `cmd/skinner/main.go`

- Default mode changed from "build" to "idle" when no positional args
- Added explicit `build` keyword handling (previously only `plan` was explicit)
- `--exit` validates that both mode and iteration count are provided
- Prompt file only read from disk when mode is not "idle"

### 4. Update `NewModel` to support idle startup ✓

**File:** `internal/tui/root.go`

- `Init()` now checks `session.Mode == "idle"` — if so, only starts tick (for prompt file scanning), no iteration spawn
- For non-idle modes, `Init()` creates the first `Run` via `controller.StartRun()` before spawning iterations
- All existing integration tests pass (they use `Mode: "build"`)

## Remaining

## 6. Update session timer for pause/resume

**File:** `internal/tui/root.go`

- Track accumulated duration on the session model
- When phase transitions from `Running` to `Finished`: record elapsed time into `AccumulatedDuration`, stop tick
- When phase transitions from `Finished` to `Running`: resume from `AccumulatedDuration`, restart tick
- `SessionDuration` in header props = `AccumulatedDuration` + (time since current run started, if running)

## 7. Add run modal type and rendering

**File:** `internal/tui/modal.go`

- Add `modalRunConfig` to the `modalType` enum
- Add `RenderRunModal(width, height int, th theme.Theme, value string, cursorPos int) string` function
- Centered modal with "Iterations:" label, input field, "enter to start" / "esc to cancel" hints

## 8. Add run modal state and key handling to root model

**File:** `internal/tui/root.go`

- Add fields: `runModalValue string`, `runModalLastValue string` (pre-fill memory, default `"10"`)
- Add `handleRunModalKey(key string)` method:
  - Digits: append to `runModalValue` (or replace if fully selected)
  - Backspace: delete last digit
  - Enter: parse value, close modal(s), start run
  - Escape: close modal, return to previous context
- On enter: read prompt file fresh from disk, call `controller.StartRun()`, start iteration loop

## 9. Wire `r` key in prompt picker

**File:** `internal/tui/root.go`

- In `handleKey()`, when focused pane is `promptsPane` and key is `r`:
  - If phase is `Running`: ignore
  - Otherwise: set `activeModal = modalRunConfig`, pre-fill with last value, record selected prompt file

## 10. Wire `r` key in prompt read modal

**File:** `internal/tui/root.go`

- In `handlePromptModalKey()`, when key is `r`:
  - If phase is `Running`: ignore
  - Otherwise: set `activeModal = modalRunConfig`, pre-fill with last value, use the currently viewed prompt file

## 11. Update prompt read modal footer

**File:** `internal/tui/promptmodal.go`

- Add `Running bool` field to `PromptModalProps`
- Footer: `e to edit · r to run · esc to close` when not running
- Footer: `e to edit · esc to close` when running (hide `r to run`)

## 12. Add run separators to iteration list

**File:** `internal/tui/iterlist.go`

- Add `Runs []model.Run` field to `IterListProps`
- In `View()`: before rendering each iteration, check if it crosses a run boundary (iteration index == `Run.StartIndex` for any run after the first)
- Render separator: `── PROMPTNAME ──` padded with `─` to full width
- Separator uses `ForegroundDim` for the line, bold `Foreground` for the name
- Separator lines are not selectable — cursor skips them
- Account for separator lines in scroll offset and height calculations

## 13. Update root model to pass run data through

**File:** `internal/tui/root.go`

- Pass `controller.Session.Runs` into `IterListProps`
- Pass `controller.Phase()` into `HeaderProps`
- Pass running state into `PromptModalProps`

## 14. Add `r` to keybindings config

**File:** `internal/config/config.go`

- Add `Run` action to `KeyMap` with default `"r"`
- Add `run` key to the `[keybindings]` config section

## 15. Update help modal to show `r` key

**File:** `internal/tui/modal.go`

- Add "Run prompt" / `r` entry under the Actions section of `RenderHelpModal`

## 16. Tests

- ✓ **model:** Test `Run` struct, `SessionPhase` constants
- ✓ **session:** Test `StartRun()`, `Phase()` transitions, `ShouldStartNext()` with per-run limits, multi-run iteration numbering
- ✓ **header:** Test idle state rendering (shows `⏱ --` and `Idle`), running/finished phase full stats
- **iterlist:** Test run separator rendering, cursor skipping separators
- **root:** Test `r` key disabled during Running phase, modal open/close flow, pre-fill memory
- **promptmodal:** Test footer with/without running state
- **CLI:** Test idle mode (no args), `--exit` validation
