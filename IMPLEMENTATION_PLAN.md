# Implementation Plan: Plan Mode

Interactive Claude CLI session launched from Skinner via `p` key. Suspends TUI, runs `sh -c` with configurable command, resumes on exit.

Spec: `specs/plan-mode.md`

## Status: Complete

All tasks implemented and tested.

## Completed Tasks

- [x] Add `ActionPlanMode` constant to `internal/config/keymap.go`
- [x] Add `PlanCommand` field to `internal/config/config.go`
- [x] Add config tests in `internal/config/config_test.go`
  - Note: TOML parser uses `strings.Trim` which strips matching quote chars from both ends — test values must avoid embedded quotes at boundaries
- [x] Add `planModeDoneMsg` and handler in `internal/tui/root.go`
- [x] Add status flash message to Model (statusFlash field, cleared on keypress)
- [x] Update `internal/tui/header.go` (StatusFlash in HeaderProps, rendered in StatusError color)
- [x] Update `internal/tui/help.go` (ActionPlanMode entry in help modal — implemented in modal.go)
- [x] Add integration tests in `internal/tui/integration_test.go` (6 tests covering disabled-during-run, exec cmd, error flash, flash clear, success no flash, flash renders in header)
- [x] Update specs (plan-mode.md, keybindings.md, config.md, help-modal.md, README.md)
