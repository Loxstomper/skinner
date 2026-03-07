# Implementation Plan

## Features

1. **Expandable tool call detail** — press Enter on any tool call to show command/output/diff below it
2. **Edit unified diff** — expanded Edit tool calls show -/+ colored diff of old_string vs new_string
3. **Remove tool call count from left pane** — iteration lines show `(Ns)` instead of `(M calls, Ns)`

Specs updated: `tui-layout.md`, `keybindings.md`, `tool-call-groups.md`, `stream-json-format.md`.

## Completed Tasks

### 1–2. Data pipeline + expanded content rendering  ✅ DONE

Raw input/result content flows through parser → executor → session → model. Expansion rendering in `expand.go` with 33 tests: `expandedContentLines` (per tool type), `renderEditDiff`, `toolCallLineCount`, `renderExpandedContentLine`.

### 3. Cursor system updates — `internal/tui/cursor.go`  ✅ DONE

`ItemLineCount`, `FlatCursorLineRange`, and `LineToFlatCursor` now use `toolCallLineCount()` for ToolCall items and group children, supporting multi-line expanded tool calls. 8 new tests cover expanded standalone tool calls, groups with expanded children, line-to-cursor mapping for content lines, and roundtrip consistency.

### 4. Timeline rendering and interaction — `internal/tui/timeline.go`  ✅ DONE

`View()` renders expanded content lines (via `expandedContentLines` + `renderExpandedContentLine`) below standalone ToolCall rows and group child rows. Group child content gets extra 2-space indent. Content lines use `flatIdx: -1` (no cursor highlight). `handleEnter()` toggles `Expanded` on standalone `*model.ToolCall` items and on group children (`childIdx >= 0`). 7 new tests: Enter toggles standalone ToolCall, Enter toggles group child, View shows expanded Bash command+output, View shows Edit diff lines, View shows expanded group child content, truncation footer renders, collapsed tool call hides content.

### 5. Left pane — remove tool call count  ✅ DONE

Removed `callCount` variable and changed format from `"(%d calls, %s)"` to `"(%s)"` in `iterlist.go`. Added `TestIterList_View_FormatShowsDurationOnly` — verifies `(2m14s)` format and absence of "calls" text.

### 6. Integration tests  ✅ DONE

No integration test assertions referenced call count text (no changes needed). Added `TestIntegration_ExpandToolCallShowsContent` — creates a Bash tool call with RawInput command and ResultContent, verifies Enter toggles Expanded, and checks that `"$ go test"` and command output appear/disappear in rendered View.

## All features implemented.
