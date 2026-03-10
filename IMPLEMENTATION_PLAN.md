# Implementation Plan: Plan Mode

Interactive Claude CLI session launched from Skinner via `p` key. Suspends TUI, runs `sh -c` with configurable command, resumes on exit.

Spec: `specs/plan-mode.md`

## Tasks

- [x] Add `ActionPlanMode` constant to `internal/config/keymap.go`
- [x] Add `PlanCommand` field to `internal/config/config.go`
- [x] Add config tests in `internal/config/config_test.go`
  - Note: TOML parser uses `strings.Trim` which strips matching quote chars from both ends — test values must avoid embedded quotes at boundaries

- [ ] Add `planModeDoneMsg` and handler in `internal/tui/root.go`
  - Add `planModeDoneMsg struct{ err error }` message type
  - Add `launchPlanMode() tea.Cmd` method: uses `tea.ExecProcess(exec.Command("sh", "-c", m.config.PlanCommand), ...)` returning `planModeDoneMsg`
  - Handle `ActionPlanMode` in `handleKey` switch: ignore if `m.controller.Phase() == model.PhaseRunning`, otherwise return `m.launchPlanMode()`
  - Handle `planModeDoneMsg` in `Update`: if `msg.err != nil`, set status flash message; rescan plan files

- [ ] Add status flash message to Model
  - Add `statusFlash string` and `statusFlashCleared bool` fields to `Model` struct
  - On `planModeDoneMsg` with error: set `statusFlash` to `fmt.Sprintf("plan command failed (exit %v)", msg.err)` and `statusFlashCleared = false`
  - Clear `statusFlash` on next keypress (in `handleKey`, check and clear before processing)
  - Pass `statusFlash` through `HeaderProps` to `RenderHeader` for display

- [ ] Update `internal/tui/header.go`
  - Add `StatusFlash string` field to `HeaderProps`
  - In `RenderHeader`, when `StatusFlash != ""`: render flash text in `StatusError` color in the header bar

- [ ] Update `internal/tui/help.go`
  - Add `ActionPlanMode` entry to help modal rendering (in Actions section)

- [ ] Add integration tests in `internal/tui/integration_test.go`
  - `TestIntegration_PlanModeDisabledDuringRun`: press `p` while `PhaseRunning`, verify no `tea.Exec` cmd returned
  - `TestIntegration_PlanModeKeyReturnsExecCmd`: press `p` while idle, verify a `tea.Cmd` is returned (can't fully test `tea.Exec` in unit tests, but verify non-nil cmd)
  - `TestIntegration_PlanModeErrorSetsFlash`: send `planModeDoneMsg{err: errors.New("exit 1")}`, verify `statusFlash` is set
  - `TestIntegration_PlanModeFlashClearsOnKeypress`: set flash, send keypress, verify flash cleared

- [ ] Update specs (already done in prior conversation)
  - [x] `specs/plan-mode.md` — created
  - [x] `specs/keybindings.md` — added `p` key
  - [x] `specs/config.md` — added `[plan]` section and `plan_mode` keybinding
  - [x] `specs/help-modal.md` — added plan mode row
  - [x] `specs/README.md` — added plan-mode entry
