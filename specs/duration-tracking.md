# Duration Tracking

## Tool Call Duration

Each tool call's duration is measured as wallclock time between the `assistant` event (tool_use) and the corresponding `user` event (tool_result).

### Algorithm

1. On receiving an `assistant` event containing a `tool_use` content block:
   - Record `tool_use.id`, `tool_use.name`, `tool_use.input`, and `start_time = now()`.
2. On receiving a `user` event containing a `tool_result` content block:
   - Look up the pending tool call by `tool_result.tool_use_id`.
   - Compute `duration = now() - start_time`.
   - Mark the tool call as complete with its duration.

While a tool call is in progress (started but no result yet), display `...` in the duration column.

## Tool Call Group Duration

When tool calls are grouped (see [tool-call-groups.md](tool-call-groups.md)), the group header row displays the wallclock span from the first child's start time to the last child's result time. Individual child rows retain their own durations. While any child is still in progress, the group header shows `...`.

## Iteration Duration

Each iteration's total duration is measured from subprocess start to subprocess exit.

- While running, show elapsed time with a `...` suffix.
- On completion, show the final total duration.

## Session Duration

Total wallclock time since the `skinner` process started. Displayed in the header bar (see [tui-layout.md](tui-layout.md)). Updates every second via a Bubble Tea tick.

## Display Format

- Durations under 60s: `1.2s`, `45.0s`
- Durations 60s and above: `1m14s`, `2m03s`
- In-progress: `...` (no time shown)
