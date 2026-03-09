# Implementation Plan: Interactive Run Start

## Completed

### 1. Add `Run` struct and session phase to model ✓

**File:** `internal/model/model.go`

- Added `SessionPhase` type with constants: `PhaseIdle`, `PhaseRunning`, `PhaseFinished`
- Added `Run` struct: `PromptName`, `PromptFile`, `StartIndex`, `MaxIterations`
- Added `Runs []Run`, `Phase SessionPhase`, `AccumulatedDuration time.Duration` fields to `Session`
- Tests added for `SessionPhase` constants, `Run` struct, and `Session` runs/phase

### 2. Add session phase and run methods to controller ✓

**File:** `internal/session/session.go`

- Added `Phase() SessionPhase` method
- Added `StartRun(promptName, promptFile string, maxIterations int)` method
- Updated `CompleteIteration()` to set phase to `Finished` when run exhausted or on failure
- Updated `ShouldStartNext()` with per-run limits (backward compatible with legacy no-run path)
- Tests: `Phase()`, `StartRun()`, multi-run, per-run limits, phase transitions, accumulated duration

**Design note:** `ShouldStartNext()` has two paths — legacy (no runs, uses global `MaxIterations`) and run-based (checks `Phase == PhaseRunning` and per-run `MaxIterations` relative to `Run.StartIndex`). `CompleteIteration()` only manages phase transitions when runs are present. This keeps backward compatibility with existing TUI code that doesn't use `StartRun()` yet.

## Remaining

## 3. Update CLI arg parsing for idle mode

**File:** `cmd/skinner/main.go`

- When no positional args: set mode to `"idle"`, no prompt file, no max iterations
- Validate `--exit` requires both a mode (`plan`/`build`) and a numeric iteration count — print error and exit otherwise
- Pass idle mode through to `NewModel` so the TUI knows not to auto-start

## 4. Update `NewModel` to support idle startup

**File:** `internal/tui/root.go`

- When mode is `"idle"`: don't call `startIteration()` in `Init()`, set session phase to `Idle`
- When mode is `"build"`/`"plan"`: start immediately as before (create first `Run` via controller)
- Session timer: only start ticking when phase transitions to `Running`

## 5. Update header for idle state

**File:** `internal/tui/header.go`

- Add `Phase SessionPhase` field to `HeaderProps`
- When phase is `Idle`: render `⏱ --` and `Idle` only, omit tokens/cost/context/rate limits/iteration counter
- When phase is `Running`/`Finished`: render as before

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
- **header:** Test idle state rendering (shows `⏱ --` and `Idle`)
- **iterlist:** Test run separator rendering, cursor skipping separators
- **root:** Test `r` key disabled during Running phase, modal open/close flow, pre-fill memory
- **promptmodal:** Test footer with/without running state
- **CLI:** Test idle mode (no args), `--exit` validation
