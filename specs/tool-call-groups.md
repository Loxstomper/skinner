# Tool Call Groups

## Overview

When an `assistant` event contains multiple consecutive `tool_use` blocks of the same tool type, they are displayed as a collapsible group in the message timeline instead of individual rows. This reduces visual noise, especially during batch operations like reading many files.

## Grouping Rules

Groups are formed from consecutive same-type `tool_use` blocks within a single `assistant` event's `message.content` array.

- Only consecutive blocks of the **same tool name** are grouped.
- Text blocks (`"type": "text"`) break groups.
- Different tool types break groups.
- Minimum group size is **2**. A single tool call of a type is rendered as a normal row.

### Example

Given an `assistant` event with content:

```
[text, Read, Read, Read, text, Edit, Edit, Bash]
```

This produces:

```
text block
 Read group (3)
text block
 Edit group (2)
 Bash (standalone row)
```

## Group Header Row

The group header row follows the same layout as a normal tool call row (see [tui-layout.md](tui-layout.md)), with these differences:

- **Icon** — the tool's Nerd Font icon, colored per state.
- **Tool name** — shown in full view, hidden in compact view (same as normal rows).
- **Summary** — shows a count instead of individual args:

| Tool  | Group summary  |
|-------|----------------|
| Read  | `N files`      |
| Edit  | `N edits`      |
| Write | `N files`      |
| Bash  | `N commands`   |
| Grep  | `N searches`   |
| Glob  | `N globs`      |
| Task  | `N tasks`      |

- **Result indicator** — `✓` if all children succeeded, `✗` if any child failed. Blank while any child is in progress.
- **Duration** — wallclock span from the first child's start time to the last child's result time. Shows `...` while any child is in progress. Colored per state.

### In-progress header

While results are still arriving, the header shows progress:

```
 Read   3/8 files                                  ...
```

The denominator is known immediately (from the `assistant` event's content array). The numerator increments as `tool_result` events arrive.

## Child Rows

All child rows are rendered immediately when the `assistant` event arrives. They are displayed in **request order** (their position in the `message.content` array), not result arrival order.

Child rows follow the standard tool call row layout but are indented by 2 extra spaces.

### Pending children (no result yet)

- **Arg summary** — shown (extracted from `tool_use` input as normal).
- **Line count metadata** — blank (not yet available or not shown for visual consistency).
- **Result indicator** — blank.
- **Duration** — `...`, colored with `DurationRunning`.
- **Icon / tool name** — colored with `ToolNameRunning`.

### Completed children

- Styled identically to standalone tool call rows (success or error colors), just indented.

## Expand / Collapse Behavior

### During a live run (in-progress group)

Groups are **expanded by default** while any child is still in progress. The user sees individual files ticking in as results arrive.

When the group completes (all children have results):

- If the **cursor is not on** the group header or any child row: the group **collapses automatically**.
- If the **cursor is on** the group header or any child row: the group **stays expanded** until the cursor moves away, then collapses.

### Reviewing a completed iteration

All groups start **collapsed**. Press `enter` on a group header to expand.

### Manual toggle

Pressing `enter` on a group header toggles expand/collapse at any time (both during a live run and when reviewing). A manually expanded group stays expanded until the user presses `enter` again to collapse it (auto-collapse does not override a manual expand).

A manually collapsed group during a live run stays collapsed (auto-expand does not override a manual collapse).

### Enter on child rows

`enter` on a child tool call row toggles the **expanded detail view** for that individual tool call (see [tui-layout.md](tui-layout.md#expandable-tool-call-detail)). This is the same expand/collapse behavior as standalone tool calls — it shows the command, output, diff, or other detail content below the child row.

## Cursor Navigation

When a group is **collapsed**, it occupies a single cursor position (the header row).

When a group is **expanded**, `j`/`k` moves through the header and each child row individually. For a group of 4, that is 5 cursor positions (1 header + 4 children).

## View Modes

Grouping behavior is **identical in both full and compact view modes**. The difference is only in how each row is rendered (tool name shown/hidden, etc.), not in the grouping logic.

### Full view — collapsed

```
 Read   4 files                                    ✓   2.1s
```

### Compact view — collapsed

```
 4 files                                           ✓   2.1s
```

### Full view — expanded (in-progress)

```
 Read   2/4 files                                  ...
    Read   src/main.go (142 lines)                 ✓   0.4s
    Read   src/parser.go (85 lines)                ✓   0.3s
    Read   src/parser_test.go                           ...
    Read   src/util.go                                  ...
```

### Compact view — expanded (in-progress)

```
 2/4 files                                         ...
    src/main.go (142 lines)                        ✓   0.4s
    src/parser.go (85 lines)                       ✓   0.3s
    src/parser_test.go                                  ...
    src/util.go                                         ...
```

### Full view — expanded (complete)

```
 Read   4 files                                    ✓   2.1s
    Read   src/main.go (142 lines)                 ✓   0.4s
    Read   src/parser.go (85 lines)                ✓   0.3s
    Read   src/parser_test.go (210 lines)          ✓   0.5s
    Read   src/util.go (64 lines)                  ✓   0.9s
```

### Compact view — expanded (complete)

```
 4 files                                           ✓   2.1s
    src/main.go (142 lines)                        ✓   0.4s
    src/parser.go (85 lines)                       ✓   0.3s
    src/parser_test.go (210 lines)                 ✓   0.5s
    src/util.go (64 lines)                         ✓   0.9s
```
