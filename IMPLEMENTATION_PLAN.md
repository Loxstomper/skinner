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

## Remaining Tasks

### 3. Pass thinking state to TimelineProps

**Files:** `internal/tui/root.go`, `internal/tui/timeline.go`

Add two fields to `TimelineProps`:
- `ThinkingStartTime time.Time` — zero value means not thinking.
- `IsThinking bool` — convenience flag.

In `Model.timelineProps()` and the inline `TimelineProps{}` in `View()`, populate these from the selected iteration (only if the selected iteration is the running one).

### 4. Render thinking row in Timeline.View()

**Files:** `internal/tui/timeline.go`

At the end of the rendered timeline content (after all real items), if `props.IsThinking`:

- Render a line: `🧠 Thinking... (duration)` where duration is `time.Since(props.ThinkingStartTime)` formatted with `FormatDurationValue()`.
- Style: "Thinking..." in `ForegroundDim`, duration in `DurationRunning` color.
- This row does NOT count toward the item list — it is appended after all cursor-targetable items, outside the cursor/scroll system.

### 5. Auto-follow keeps thinking row visible

**Files:** `internal/tui/timeline.go`

When auto-follow is active and the thinking row is rendered, ensure the scroll offset accounts for the extra line so the thinking row is within the viewport. This should work naturally if the thinking row is appended after the last item and the auto-follow logic scrolls to show the bottom of content.

### 7. Unit tests for thinking row rendering

**Files:** `internal/tui/timeline_test.go`

Test cases:
- `IsThinking=true` with a `ThinkingStartTime` → output contains "🧠" and "Thinking...".
- `IsThinking=false` → no thinking row in output.
- Thinking row does not affect cursor item count (cursor count matches `len(Items)`).
