# TUI Layout

## Overview

Two-pane layout: a left sidebar listing iterations and a right main pane showing a message timeline for the selected iteration. All colors are driven by the active theme — see [theme.md](theme.md).

## Header Bar

A single-line header pinned to the top of the TUI, spanning the full terminal width. Colored with `ForegroundDim` text on the default background.

```
              ⏱ 14m32s   ↑42.1k ↓8.3k tokens   ctx 62%   ~$1.24              Iter 3/10 ⟳
```

**Centre** (centred in the space left of the iteration indicator):
- **Session duration** — `⏱` followed by total wallclock time since `skinner` started. Updates every second. Format follows [duration-tracking.md](duration-tracking.md) rules.
- **Token counts** — `↑` input tokens (including cache read and cache creation) and `↓` output tokens. Formatted with `k` suffix for thousands (e.g. `42.1k`), no suffix under 1000 (e.g. `850`).
- **Context window usage** — `ctx N%` showing how full the current context window is. Calculated as `(input_tokens + cache_read_input_tokens) / context_window * 100` from the most recent `assistant` event's `message.usage`. The denominator (`context_window`) is a per-model value from the pricing config (see [config.md](config.md)). Omitted entirely until the first `assistant` event is received or if the model is not in the pricing table. Colored by threshold:
  - Normal (`ForegroundDim`) — 0–69%
  - Warning (`StatusRunning`) — 70–89%
  - Critical (`StatusError`) — 90%+
- **Estimated cost** — `~$` followed by the accumulated cost estimate. See [stream-json-format.md](stream-json-format.md) for calculation. Omitted entirely if the model is not in the pricing table.

**Right side** (right-aligned):
- **Iteration progress** — `Iter N` (unlimited mode) or `Iter N/M` (when max iterations is set).
- **Status icon** — `⟳` while an iteration is running, `✓` when the session has finished all iterations, `✗` if the last iteration failed. Colored per theme (`StatusRunning`/`StatusSuccess`/`StatusError`).

## Focus Model

One pane is focused at a time. The focused pane has a visual indicator (brighter border or highlight). All movement keys operate on the focused pane. See [keybindings.md](keybindings.md) for controls.

## Left Pane — Iteration List

A vertical list of all iterations in the current session. Each entry shows:

```
 ████████████████████████████████
   Iter 1  ✓  (23 calls, 2m14s)    ← highlighted row
 ████████████████████████████████
   Iter 2  ✓  (17 calls, 1m48s)
   Iter 3  ⟳  (5 calls, 0m32s...)
```

- **Cursor**: The selected iteration row is highlighted with the theme's `Highlight` background color.
- **Status icon**: `✓` completed (`StatusSuccess`), `⟳` running (`StatusRunning`), `✗` failed (`StatusError`). Colored per theme.
- **Iteration text**: colored per state (`IterRunning`, `IterSuccess`, `IterError`).
- **Call count**: total number of tool calls in that iteration.
- **Duration**: total wallclock time of the iteration. Shown with `...` suffix while still running.

**Scrolling**: When iterations exceed the viewport height, the list scrolls to keep the cursor visible. Moving the cursor beyond the viewport edge adjusts the scroll offset. The view renders only the visible slice of iterations.

**Auto-follow**: During a run, the cursor auto-follows to the latest iteration. If the user manually selects a previous iteration, auto-follow pauses. Selecting the latest iteration re-enables auto-follow.

## Right Pane — Message Timeline

When an iteration is selected, show its messages as a scrollable timeline. Messages are rendered in the order they appear in the stream. There are two types of items:

### Text Blocks

Claude's reasoning and responses (the `text` content blocks from `assistant` events). Displayed in the theme's `TextBlock` color.

- **Full view**: show up to **3 lines**. If the text exceeds 3 lines, truncate with `…`.
- **Compact view**: show up to **1 line**. If the text exceeds 1 line, truncate with `…`.
- When the cursor is on a text block, pressing `enter` toggles expand/collapse (both view modes).

### Tool Call Rows

Each tool call is a single row containing:

1. **Icon** — a Nerd Font icon identifying the tool type. Colored per state (`ToolNameRunning`/`ToolNameSuccess`/`ToolNameError`). See [stream-json-format.md](stream-json-format.md) for the icon table.
2. **Tool name** — left-aligned, fixed width. Colored per state. *(Full view only — hidden in compact view.)*
3. **Arg summary** — truncated to fit available terminal width (see [stream-json-format.md](stream-json-format.md) for summary extraction per tool). Always dim (`ToolSummary`).
4. **Line count metadata** — for Read, Edit, and Write only. Shown in parentheses after the arg summary, dim (`ToolSummary`). See [stream-json-format.md](stream-json-format.md) for extraction rules. Blank while the call is in progress (Read metadata comes from the result, so it is only available after completion; Edit and Write metadata comes from the input, so it is available immediately but should still only be shown after completion for visual consistency).
5. **Result indicator** — `✓` success or `✗` error, colored per state. Blank while in progress.
6. **Duration** — right-aligned, colored per state (`DurationRunning`/`DurationSuccess`/`DurationError`). Shows `...` while the call is in progress, then the final duration.

For unknown tools (not in the icon table), use the fallback icon `` (`f059`, question-circle) and always show the tool name regardless of view mode.

### Tool Call Groups

Consecutive tool calls of the same type within a single `assistant` event are displayed as a collapsible group. Groups expand automatically while in progress and collapse on completion. See [tool-call-groups.md](tool-call-groups.md) for full grouping rules, expand/collapse behavior, and cursor navigation.

### View Modes

The right pane supports two view modes, toggled at runtime with `v` (see [keybindings.md](keybindings.md)) or set in the config file (see [config.md](config.md)):

**Full view** (default) — icon + tool name + summary; text blocks show up to 3 lines; groups are collapsed:

```
  Looking at the test failures to understand
  what's going wrong with the parser module.
  The error suggests a type mismatch...
   Read   4 files                              ✓   2.1s
  The test expects a return value of 42 but
  the function returns a string. I need to
  fix the return type.
   Edit   src/main.go (+3/-1)                 ✓   0.3s
   Bash   go test ./...                       ✗   4.5s
  Tests still failing — different error now…
   Read   src/main.go (85 lines)              ✓   0.8s
```

**Compact view** — icon + summary only (no tool name); text blocks show 1 line; groups are collapsed:

```
  Looking at the test failures to understand...
   4 files                                     ✓   2.1s
  The test expects a return value of 42 but...
   src/main.go (+3/-1)                        ✓   0.3s
   go test ./...                              ✗   4.5s
  Tests still failing — different error now…
   src/main.go (85 lines)                     ✓   0.8s
```

See [tool-call-groups.md](tool-call-groups.md) for expanded group examples in both view modes.

### Cursor

The right pane has a visible cursor that highlights the current item (text block or tool call row). Move with `j`/`k`/`↑`/`↓` when the right pane is focused.

### Auto-follow

When showing the **currently running iteration**: auto-scroll follows new messages as they arrive (cursor stays at bottom). If the user manually scrolls up, auto-follow pauses. Moving the cursor back to the bottom (including via `G` / `End`) re-enables it.

When showing a **completed iteration**: cursor starts at the top, no auto-scroll.
