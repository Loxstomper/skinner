# Implementation Plan: Interactive Run Start

## Completed

1. **Run struct and session phase** — `SessionPhase` type, `Run` struct, phase/run fields on `Session` ✓
2. **Session controller methods** — `Phase()`, `StartRun()`, per-run limits in `ShouldStartNext()`/`CompleteIteration()` ✓
3. **CLI idle mode** — Default to idle when no args, `--exit` validation ✓
4. **Idle startup in TUI** — `Init()` skips iteration spawn in idle mode, creates `Run` via `StartRun()` otherwise ✓
5. **Idle header rendering** — `Phase` field on `HeaderProps`, idle shows `⏱ --` and `Idle` only ✓
6. **Session timer pause/resume** — `sessionDuration()` with accumulated duration across runs ✓
7. **Run modal rendering** — `modalRunConfig` type, `RenderRunModal` with iterations label, input field, hints ✓
8. **Run modal state and keys** — `runModalValue`/`runModalLastValue`/`runModalSelected` fields, `handleRunModalKey` for digits/backspace/enter/escape, pre-fill memory (default "10") ✓
9. **Wire `r` in prompt picker** — `ActionRun` in `handleKey()` opens run modal when not running ✓
10. **Wire `r` in prompt read modal** — `handlePromptModalKey` opens run modal for viewed prompt ✓
11. **Prompt modal footer** — `Running bool` on `PromptModalProps`, hides "r to run" when running ✓
12. **Run separators in iteration list** — `Runs` field on `IterListProps`, separator rendering at run boundaries, cursor skips separators, scroll accounting for display rows vs iteration indices ✓
13. **Root model passes run data** — `controller.Session.Runs` passed into `IterListProps` in both `View()` and `iterListProps()`, all method call sites updated ✓
14. **Keybinding config** — `ActionRun = "run"` with default `r`, added to `AllActions()` ✓
15. **Help modal** — "Run prompt" entry in Actions section of `RenderHelpModal` ✓

16. **Tests** — All test categories complete ✓

## All items completed.
