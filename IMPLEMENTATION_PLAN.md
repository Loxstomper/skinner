# Implementation Plan

## ~~1. Remove `...` suffix from left sidebar running durations~~ ✓ DONE

## ~~2. Remove `✓`/`✗` result indicators from right pane tool calls~~ ✓ DONE

## ~~3. Add prompt file picker to left pane~~ ✓ DONE

## ~~4. Prompt read modal~~ ✓ DONE

## 5. Integration tests

- [x] Add integration test: prompt read modal opens on Enter, shows content with line numbers
- [x] Add integration test: `esc` dismisses prompt read modal
- [x] Add integration test: Tab cycles through all three focus targets (updated existing test)

## Completed

### Previous

- Fix tool call row highlight (per-segment background)
- Click-to-expand and sub-scroll exit via mouse click
- Integration tests for help modal, sub-scroll, and custom keybindings
- Rate limit window display with placeholder values in header
- Per-tool-call token attribution with inline display
- Remove `...` suffix from left sidebar running durations (updated iterlist.go, specs, tests)
- Fix staticcheck lint warnings in timeline_test.go (WriteString→Fprintf)
- Remove `✓`/`✗` result indicators from tool call rows (timeline.go, spec, tests)
- Prompt file picker in left pane: component (`promptlist.go`), 3-pane focus model (`iterationsPane`/`promptsPane`/`rightPane`), left pane layout split (iter list + divider + prompt list), spec (`specs/prompt-files.md`), updated specs (keybindings, tui-layout, mouse, README), 22 unit tests, updated integration tests for 3-pane Tab cycle
- Prompt read modal: `promptmodal.go` renderer with line-number gutter, `modalPromptRead` type, state management in root model (file/content/scroll), key handling (j/k/pgup/pgdn/esc), `$EDITOR` launch via `tea.ExecProcess`, 15 unit+integration tests, updated spec

## Design Decisions

### Focus model: 3-pane cycle
- `h`/`←` from timeline always goes to `iterationsPane` (not "last focused left sub-pane") for simplicity
- Tab cycles: iterationsPane → promptsPane → rightPane → iterationsPane
- `paneID` constants renamed: `leftPane` → `iterationsPane`, added `promptsPane`

## Known Issues

### Integration test timeouts
`TestIntegration_HelpModal_EnterDismisses` and `TestIntegration_ExitFlag_SingleIteration`
occasionally hang on `bubbletea.Tick` channel receives. They do eventually pass
within the 90s Go test timeout but slow down the test suite. The `make check`
target completes successfully despite the delay.
