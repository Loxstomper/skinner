# TUI Layout

## Overview

Two-pane layout: a left sidebar listing iterations and a right main pane showing a message timeline for the selected iteration. All colors are driven by the active theme — see [theme.md](theme.md).

## Header Bar

A single-line header pinned to the top of the TUI, spanning the full terminal width. Colored with `ForegroundDim` text on the default background.

```
              ⏱ 14m32s   ↑42.1k ↓8.3k tokens   ctx 62%   ~$1.24   5h: 34%  wk: 12%   Iter 3/10 ⟳
```

**Centre** (centred in the space left of the iteration indicator):
- **Session duration** — `⏱` followed by total wallclock time since `skinner` started. Updates every second. Format follows [duration-tracking.md](duration-tracking.md) rules.
- **Token counts** — `↑` input tokens (including cache read and cache creation) and `↓` output tokens. Formatted with `G` suffix for billions (e.g. `1.5G`), `M` for millions (e.g. `12.3M`), `k` for thousands (e.g. `42.1k`), no suffix under 1000 (e.g. `850`).
- **Context window usage** — `ctx N%` showing how full the current context window is. Calculated as `(input_tokens + cache_read_input_tokens) / context_window * 100` from the most recent `assistant` event's `message.usage`. The denominator (`context_window`) is a per-model value from the pricing config (see [config.md](config.md)). Omitted entirely until the first `assistant` event is received or if the model is not in the pricing table. Colored by threshold:
  - Normal (`ForegroundDim`) — 0–69%
  - Warning (`StatusRunning`) — 70–89%
  - Critical (`StatusError`) — 90%+
- **Estimated cost** — `~$` followed by the accumulated cost estimate. See [stream-json-format.md](stream-json-format.md) for calculation. Omitted entirely if the model is not in the pricing table.
- **Rate limit windows** — `5h: N%` and `wk: N%` showing current utilization of the 5-hour and weekly API token windows. See [token-usage.md](token-usage.md) for data source and styling. Displays `--` until data is fetched.

**Right side** (right-aligned):
- **Iteration progress** — `Iter N` (unlimited mode) or `Iter N/M` (when max iterations is set).
- **Status icon** — `⟳` while an iteration is running, `✓` when the session has finished all iterations, `✗` if the last iteration failed. Colored per theme (`StatusRunning`/`StatusSuccess`/`StatusError`).

## Responsive Layout

The two-pane layout adapts to terminal width:

| Terminal width | Left pane | Right pane |
|----------------|-----------|------------|
| ≥ 80 columns | Visible | Visible |
| < 80 columns | Hidden | Full width |

When the left pane is hidden, it can be toggled with `[` (see [keybindings.md](keybindings.md)). Pressing `[` on a wide terminal also toggles the left pane off/on.

## Focus Model

One pane is focused at a time. The focused pane has a visual indicator (brighter border or highlight). All movement keys operate on the focused pane. See [keybindings.md](keybindings.md) for controls.

## Left Pane — Iteration List

A vertical list of all iterations in the current session. Each entry shows:

```
 ████████████████████████████████
   Iter 1  ✓  (2m14s)               ← highlighted row
 ████████████████████████████████
   Iter 2  ✓  (1m48s)
   Iter 3  ⟳  (0m32s)
```

- **Cursor**: The selected iteration row is highlighted with the theme's `Highlight` background color.
- **Status icon**: `✓` completed (`StatusSuccess`), `⟳` running (`StatusRunning`), `✗` failed (`StatusError`). Colored per theme.
- **Iteration text**: colored per state (`IterRunning`, `IterSuccess`, `IterError`).
- **Duration**: total wallclock time of the iteration. Running and completed iterations both show plain duration values; the `⟳` icon and color distinguish running state.

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
5. **Token counts** — approximate cached vs real input tokens attributed to this tool call, shown as `[↑N ⚡N]` in `ForegroundDim`. See [token-usage.md](token-usage.md) for attribution logic.
6. **Result indicator** — `✓` success or `✗` error, colored per state. Blank while in progress.
7. **Duration** — right-aligned, colored per state (`DurationRunning`/`DurationSuccess`/`DurationError`). Shows `...` while the call is in progress, then the final duration.

For unknown tools (not in the icon table), use the fallback icon `` (`f059`, question-circle) and always show the tool name regardless of view mode.

### Expandable Tool Call Detail

Pressing `enter` on any tool call row (standalone or group child) toggles an expanded detail view below the summary row. The expanded view shows tool-specific content **without truncation** — full content is always available. See [sub-scroll.md](sub-scroll.md) for how large content is handled via adaptive sizing and sub-scroll.

| Tool  | Expanded content                                                    |
|-------|---------------------------------------------------------------------|
| Bash  | `$ command` header line, then full command output                   |
| Edit  | Full diff of `old_string` → `new_string` (see below)               |
| Read  | Full file contents from the tool result                             |
| Write | Full `content` that was written (from the tool input)               |
| Grep  | Full search results from the tool result                            |
| Glob  | Full matched file list from the tool result                         |
| Task  | Full task output from the tool result                               |
| Other | Full tool result content, if available                              |

**Styling**: Expanded content lines are indented by 4 spaces and rendered in dim text (`ForegroundDim`), except for Edit diffs which use colored diff styling (see below).

**Cursor behavior**: Expanded content lines are **not individually selectable** — the tool call remains a single cursor position regardless of expansion state. However, the expanded lines count toward the item's display height for scroll calculations (the cursor "covers" the header + all expanded lines, similar to multi-line text blocks). Pressing `enter` on an already-expanded tool call enters sub-scroll mode — see [sub-scroll.md](sub-scroll.md).

**Highlighting**: When the cursor is on a tool call row, the **entire row** is highlighted with the theme's `Highlight` background, padded to the full width of the right pane regardless of content length. Because tool call rows are composed of multiple individually-styled segments (icon, name, summary, tokens, result, duration), the `Highlight` background must be applied to each segment individually rather than wrapping the concatenated string — wrapping would cause inner ANSI reset codes to clear the background after the first segment. The same per-segment approach applies to group header rows. The expanded content lines below are not highlighted.

#### Edit Diff Format

When an Edit tool call is expanded, the detail view shows a diff with line numbers. The layout adapts to terminal width:

**Unified diff** (terminal width < 120 columns):

- Lines from `old_string` are prefixed with `-` and colored red (`StatusError`).
- Lines from `new_string` are prefixed with `+` and colored green (`StatusSuccess`).
- Line numbers are shown in the gutter.
- Full diff is displayed — no truncation.

```
   Edit   src/main.go (+2/-1)                    ✓   0.3s
      42  -    return "hello"
      42  +    name := "world"
      43  +    return fmt.Sprintf("hello, %s", name)
```

**Side-by-side diff** (terminal width ≥ 120 columns):

- Left column: old content with line numbers, removals colored red (`StatusError`).
- Right column: new content with line numbers, additions colored green (`StatusSuccess`).
- A vertical divider separates the two columns.
- Each column gets half the available width.
- Full diff is displayed — no truncation.

```
   Edit   src/main.go (+2/-1)                                          ✓   0.3s
      42 │ return "hello"                    │ 42 │ name := "world"
         │                                   │ 43 │ return fmt.Sprintf("hello, %s", name)
```

#### Bash Expanded Example

```
   Bash   Run test suite                          ✗   4.5s
      $ go test ./...
      --- FAIL: TestParser (0.01s)
          parser_test.go:42: expected 42, got "42"
      FAIL
```

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
   Edit   src/main.go (+2/-2)                 ✓   0.3s
   Bash   go test ./...                       ✗   4.5s
  Tests still failing — different error now…
   Read   src/main.go (85 lines)              ✓   0.8s
```

**Compact view** — icon + summary only (no tool name); text blocks show 1 line; groups are collapsed:

```
  Looking at the test failures to understand...
   4 files                                     ✓   2.1s
  The test expects a return value of 42 but...
   src/main.go (+2/-2)                        ✓   0.3s
   go test ./...                              ✗   4.5s
  Tests still failing — different error now…
   src/main.go (85 lines)                     ✓   0.8s
```

See [tool-call-groups.md](tool-call-groups.md) for expanded group examples in both view modes.

### Relative Line Numbers

The right pane includes a gutter showing relative line numbers for vim-style navigation. See [line-numbers.md](line-numbers.md) for display format, `{count}j`/`{count}k` jump motions, and configuration.

### Cursor

The right pane has a visible cursor that highlights the current item (text block or tool call row) with the theme's `Highlight` background, **padded to the full width of the right pane**. Move with `j`/`k`/`↑`/`↓` when the right pane is focused. Supports `{count}j`/`{count}k` for multi-item jumps — see [line-numbers.md](line-numbers.md).

### Auto-follow

When showing the **currently running iteration**: auto-scroll follows new messages as they arrive (cursor stays at bottom). If the user manually scrolls up, auto-follow pauses. Moving the cursor back to the bottom (including via `G` / `End`) re-enables it.

When showing a **completed iteration**: cursor starts at the top, no auto-scroll.
