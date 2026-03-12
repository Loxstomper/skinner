# Thinking Indicator

## Overview

A transient row displayed at the bottom of the right pane timeline when Claude is processing and no output is visible. Provides feedback that the iteration is alive and shows how long the wait has been.

## Display

```
🧠 Thinking... (4.2s)
```

- **Emoji**: `🧠` brain.
- **Text**: `Thinking...` in `ForegroundDim`.
- **Duration**: wallclock time since the thinking state began, using the standard duration format (see [duration-tracking.md](duration-tracking.md)). Colored with `DurationRunning`.

Same appearance in both full and compact view modes.

## When to Show

The thinking indicator appears when an iteration is running AND there is no visible activity. Specifically, show it when:

1. **Iteration just started** — no events have arrived yet from the subprocess.
2. **Between API calls** — all tool results from the current `assistant` batch have been received (all `user` events matching the batch's `tool_use` IDs), and no new `assistant` event has arrived.

Do NOT show the thinking indicator when:

- A tool call is in progress (the running tool call row already signals activity).
- A tool call group is still expanding with new calls from the current `assistant` event.
- The iteration is complete (a `result` event has been received).

## Timer

The timer starts counting from the moment the thinking state begins:

- **Iteration start**: when the subprocess is launched.
- **Between API calls**: when the last pending tool result for the current batch is received.

The timer resets each time a new thinking state begins. It stops (and the row disappears) when the next `assistant` event arrives.

## Behavior

- **Not a cursor target** — the thinking row is ephemeral UI chrome, not a timeline item. The cursor cannot be placed on it and `j`/`k` navigation skips it. It does not have a line number in the gutter.
- **Auto-follow** — when auto-follow is active, the viewport scrolls to keep the thinking row visible, just as it would for any new content appearing at the bottom.
- **Disappearance** — when the next `assistant` event arrives, the thinking row is removed instantly and replaced by the new content (text block or tool call). No animation or transition.
