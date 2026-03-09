# Implementation Plan

## ~~1. Remove `...` suffix from left sidebar running durations~~ ✓ DONE

## ~~2. Remove `✓`/`✗` result indicators from right pane tool calls~~ ✓ DONE

## ~~3. Add prompt file picker to left pane~~ ✓ DONE

## 4. Prompt read modal

### 4a. Spec: add to `specs/prompt-files.md`

- [ ] Document read modal behavior:
  - Full-screen centered overlay composed on top of entire TUI (same pattern as help/quit modals)
  - Title bar shows full filename (e.g. `PROMPT_foo.md`)
  - Plain text content with absolute line numbers in a dimmed gutter
  - Scrollable with j/k/arrows/pgup/pgdn
  - Footer hint: `e to edit · esc to close`
  - `esc` dismisses modal
  - `e` suspends TUI, opens `$EDITOR` (fallback `vi`), on exit modal dismisses, TUI resumes

### 4b. Modal implementation

- [ ] Add `modalPromptRead` to `modalType` enum in `internal/tui/modal.go`
- [ ] Create `internal/tui/promptmodal.go`: `RenderPromptReadModal` function
  - Accepts file path, scroll offset, terminal width/height, theme
  - Reads file content, renders with line number gutter (dimmed)
  - Bordered modal ~80% terminal width/height, centered via `centerOverlay`
  - Title injected into top border (same pattern as help modal)
  - Footer with `e to edit · esc to close`
- [ ] Create `internal/tui/promptmodal_test.go`: tests for rendering, line numbers, scroll bounds, long content

### 4c. Modal state and key handling

- [ ] Add prompt modal state to root model: active file path, scroll offset, content lines
- [ ] Handle Enter on prompt list item: load file, set `modalPromptRead`, reset scroll
- [ ] Handle keys while `modalPromptRead` is active:
  - `j`/`k`/arrows/pgup/pgdn: scroll content
  - `esc`: dismiss modal
  - `e`: launch `$EDITOR` via `tea.ExecProcess`, on return dismiss modal
- [ ] All other keys blocked while modal is open

### 4d. Editor integration

- [ ] Implement `$EDITOR` launch: read `$EDITOR` env var, fallback to `vi`
- [ ] Use `tea.ExecProcess` to suspend TUI and run editor with the prompt file path
- [ ] On editor exit: dismiss modal, TUI resumes normally (tool calls that arrived during editing are already in the model)

## 5. Integration tests

- [ ] Add integration test: prompt list renders files, shows empty state
- [ ] Add integration test: prompt read modal opens on Enter, shows content with line numbers
- [ ] Add integration test: `esc` dismisses prompt read modal
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
