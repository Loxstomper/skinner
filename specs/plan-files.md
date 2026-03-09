# Plan Files

## Overview

The left pane includes a fixed-height plan file picker at the top, above the iteration list. It lists `*_PLAN.md` files from the current working directory and allows users to browse, view, and edit them. When the plans pane is focused, the right pane replaces the message timeline with a glamour-rendered markdown view of the selected plan file.

## Left Pane Layout

The left pane is split vertically into three sections:

```
┌────────────────────────────┐
│  📋 Plans                  │  ← Title row
│  IMPLEMENTATION            │  ← Plan file entries (4 rows max)
│  RELEASE                   │
│                            │
│────────────────────────────│  ← Horizontal divider (─)
│   Iter 1  ✓  (2m14s)      │  ← Iteration list (flexible height)
│   Iter 2  ✓  (1m48s)      │
│   Iter 3  ⟳  (0m32s)      │
│                            │
│────────────────────────────│  ← Horizontal divider (─)
│  📄 Prompts                │  ← Title row
│  BUILD                     │
│  TEST                      │
│                            │
└────────────────────────────┘
```

- **Plan section**: fixed 5 rows total (1 title + 4 content rows), at the top of the left pane.
- **Divider**: a single line of `─` characters in `ForegroundDim`.
- **Iteration list**: takes remaining height after subtracting the plan section, prompt section, and two dividers.
- **Divider**: a single line of `─` characters in `ForegroundDim`.
- **Prompt section**: fixed 5 rows total (1 title + 4 content rows), at the bottom.

## Plan List

### Title

`📋 Plans` — bold, theme `Foreground` color.

### File Discovery

- Scans the current working directory for files matching `*_PLAN.md`.
- Files are sorted alphabetically.
- Display names strip the `_PLAN.md` suffix (e.g. `IMPLEMENTATION_PLAN.md` → `IMPLEMENTATION`, `RELEASE_PLAN.md` → `RELEASE`).
- The list rescans on each tick (1-second interval) to pick up file additions/removals. **Future improvement**: replace tick-based rescanning with `fsnotify` for both plan and prompt file discovery.

### Empty State

When no `*_PLAN.md` files exist, the content area shows:

```
  📋 Plans
  No plan files
```

"No plan files" is rendered in `ForegroundDim`.

### Selection

- Same highlight style as the iteration list: theme's `Highlight` background, padded to full width.
- Highlight only shown when the plan list is focused.

### Scrolling

When files exceed the 4-row content area, the list scrolls to keep the cursor visible. Standard navigation keys work: `j`/`k`, `gg`/`G`, `pgup`/`pgdn`.

## Focus Model

Four focus targets cycle via `Tab`:

```
Plans → Iterations → Prompts → Timeline → Plans
```

- **`Tab`**: cycles through all four panes in order.
- **`h` / `←`**: from Timeline or plan content view, focuses the Plans pane.
- **`l` / `→`**: from any left pane, focuses the right pane (plan content view or timeline, depending on context).

Both Plans, Iterations, and Prompts are visually in the left column. When the left pane is hidden (terminal < 80 columns), none can receive focus.

## Right Pane Behavior

The right pane content is driven by which left-pane section has focus (or last had focus, when the right pane itself is focused):

- **Plans focused** (or right pane focused after plans): right pane shows the plan content view.
- **Iterations focused** (or right pane focused after iterations): right pane shows the message timeline.
- **Prompts focused**: right pane shows the message timeline (prompt read modal is a separate overlay).

When focus moves between left-pane sections, the right pane swaps content accordingly.

## Plan Content View (Right Pane)

When the plans pane drives the right pane, the selected plan file is rendered as follows:

### Title Bar

A title bar at the top of the right pane showing the full filename (e.g. `IMPLEMENTATION_PLAN.md`) centered. Styled with bold text in theme `Foreground` color.

### Content

The plan file content is rendered using [glamour](https://github.com/charmbracelet/glamour) with the `auto` style (adapts to terminal background). Content is word-wrapped to the right pane width.

### Navigation

When the right pane is focused and showing plan content:

- **`j` / `↓`**: scroll down one line.
- **`k` / `↑`**: scroll up one line.
- **`gg` / `Home`**: scroll to top.
- **`G` / `End`**: scroll to bottom.
- **`pgdn`**: page scroll down.
- **`pgup`**: page scroll up.

### Scroll Position

- When tabbing away from the plans pane and back, the scroll position of the current plan is preserved.
- When moving the cursor between different plans in the list, the scroll position resets to the top.

### Editor Integration

- **`e`**: suspends the TUI and opens `$EDITOR` (fallback: `vi`) with the plan file path. Active when either the plan list or the plan content view is focused.
- On editor exit: TUI resumes, plan content is re-rendered with the updated file contents. Focus returns to the plan content view.

## File Deletion Handling

If a plan file is deleted externally while it is being viewed in the right pane:

- The right pane shows a brief dimmed message (e.g. "File not found") in `ForegroundDim`.
- The file is removed from the plan list on the next rescan tick.

## Mouse Support

- **Target detection**: clicks/scrolls in the left column above the first divider (between plans and iterations) target the plan list.
- **Scroll**: 3 lines per wheel tick, same as other panes.
- **Click**: single click selects a plan file. Clicking the title row is ignored.
- **Focus**: any mouse interaction switches focus to the targeted pane.
