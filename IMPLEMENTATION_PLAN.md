# Implementation Plan: Thinking Indicator

Spec: [specs/thinking-indicator.md](specs/thinking-indicator.md)

## Overview

Add a transient "🧠 Thinking..." row with a live timer to the bottom of the right pane timeline when Claude is processing and no output is visible. The row is not a cursor target — it's ephemeral UI chrome that disappears when the next `assistant` event arrives.

## Completed Tasks

### ~~1. Track thinking state in session Controller~~ ✅

Added `ThinkingStartTime` field to `model.Iteration`. Controller sets it in `StartIteration()` and `ProcessToolResult()` (when all tools complete), clears it in `ProcessAssistantBatch()` and `CompleteIteration()`.

### ~~2. Add helper to check for running tool calls~~ ✅

Added `Iteration.HasRunningToolCall() bool` and `Iteration.IsThinking() bool` methods.

### ~~6. Unit tests for thinking state tracking~~ ✅

Full test coverage in `model_test.go` (HasRunningToolCall, IsThinking) and `session_test.go` (7 thinking state lifecycle tests).

### ~~3. Pass thinking state to TimelineProps~~ ✅

Added `IsThinking bool` and `ThinkingStartTime time.Time` fields to `TimelineProps`. Populated via `populateThinkingState()` helper in root.go — only sets thinking state when the selected iteration is the running one and `iter.IsThinking()` returns true. Both `timelineProps()` and the inline `TimelineProps{}` in `View()` call this helper.

### ~~4. Render thinking row in Timeline.View()~~ ✅

After all real items in `View()`, if `props.IsThinking`, appends a line: `🧠 Thinking... (duration)` with "Thinking..." in `ForegroundDim` and duration in `DurationRunning` color. The row uses `flatIdx: -1` so it's not a cursor target.

### ~~5. Auto-follow keeps thinking row visible~~ ✅

`effectiveTotalLines()` adds 1 when `props.IsThinking`, so `scrollToBottom()` accounts for the thinking row. Auto-follow naturally keeps it in viewport.

### ~~7. Unit tests for thinking row rendering~~ ✅

Three test cases in `timeline_test.go`:
- `TestTimeline_ThinkingRowShown`: IsThinking=true → output contains "🧠" and "Thinking...".
- `TestTimeline_ThinkingRowHidden`: IsThinking=false → no thinking row.
- `TestTimeline_ThinkingRowDoesNotAffectCursorCount`: cursor count unchanged by thinking state.

## All tasks complete ✅
