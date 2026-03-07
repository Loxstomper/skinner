# Implementation Plan

## Features

1. **Expandable tool call detail** — press Enter on any tool call to show command/output/diff below it
2. **Edit unified diff** — expanded Edit tool calls show -/+ colored diff of old_string vs new_string
3. **Remove tool call count from left pane** — iteration lines show `(Ns)` instead of `(M calls, Ns)`

Specs updated: `tui-layout.md`, `keybindings.md`, `tool-call-groups.md`, `stream-json-format.md`.

## Tasks

### 1–2. Data pipeline + expanded content rendering  ✅ DONE

Raw input/result content flows through parser → executor → session → model. Expansion rendering in `expand.go` with 33 tests: `expandedContentLines` (per tool type), `renderEditDiff`, `toolCallLineCount`, `renderExpandedContentLine`.

### 3. Cursor system updates — `internal/tui/cursor.go`  ✅ DONE

`ItemLineCount`, `FlatCursorLineRange`, and `LineToFlatCursor` now use `toolCallLineCount()` for ToolCall items and group children, supporting multi-line expanded tool calls. 8 new tests cover expanded standalone tool calls, groups with expanded children, line-to-cursor mapping for content lines, and roundtrip consistency.

### 4. Timeline rendering and interaction — `internal/tui/timeline.go`

- [ ] **`View()`**: after rendering a ToolCall line, if `Expanded`, render content lines via `expandedContentLines` + `renderExpandedContentLine` with `flatIdx: -1` (no highlight). Same for group children when their `Expanded` is true (with extra 2-space indent).
- [ ] **`handleEnter()`**: add `*model.ToolCall` case to toggle `Expanded`; change `*model.ToolCallGroup` `childIdx >= 0` branch from no-op to toggling `child.Expanded`
- [ ] **tests**: update `timeline_test.go` — test Enter toggles standalone ToolCall expansion, Enter toggles group child expansion, View output includes expanded content lines, Edit shows diff lines, truncation renders footer

### 5. Left pane — remove tool call count — `internal/tui/iterlist.go`

- [ ] Change format string from `"  (%d calls, %s)"` to `"  (%s)"`, remove `callCount` variable
- [ ] **tests**: update `iterlist_test.go` — verify format is `(Ns)` without call count

### 6. Integration tests — `internal/tui/integration_test.go`

- [ ] Update any assertions that check for call count text in the left pane
- [ ] Add integration test for expanding a standalone tool call via Enter and verifying expanded content appears in the rendered view
