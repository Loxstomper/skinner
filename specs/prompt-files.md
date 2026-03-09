# Prompt Files

## Overview

The left pane includes a fixed-height prompt file picker at the bottom, below the iteration list. It lists `PROMPT_*.md` files from the current working directory and allows users to browse and edit them.

## Left Pane Layout

The left pane is split vertically:

```
┌────────────────────────────┐
│   Iter 1  ✓  (2m14s)      │  ← Iteration list (flexible height)
│   Iter 2  ✓  (1m48s)      │
│   Iter 3  ⟳  (0m32s)      │
│                            │
│                            │
│────────────────────────────│  ← Horizontal divider (─)
│  📄 Prompts                │  ← Title row
│  BUILD                     │  ← Prompt file entries (4 rows max)
│  PLAN                      │
│  TEST                      │
│                            │
└────────────────────────────┘
```

- **Iteration list**: takes remaining height after subtracting the prompt section and divider.
- **Divider**: a single line of `─` characters in `ForegroundDim`.
- **Prompt section**: fixed 5 rows total (1 title + 4 content rows).

## Prompt List

### Title

`📄 Prompts` — bold, theme `Foreground` color.

### File Discovery

- Scans the current working directory for files matching `PROMPT_*.md`.
- Files are sorted alphabetically.
- Display names strip the `PROMPT_` prefix and `.md` suffix (e.g. `PROMPT_BUILD.md` → `BUILD`).
- The list rescans on each tick (1-second interval) to pick up file additions/removals.

### Empty State

When no `PROMPT_*.md` files exist, the content area shows:

```
  📄 Prompts
  No prompt files
```

"No prompt files" is rendered in `ForegroundDim`.

### Selection

- Same highlight style as the iteration list: theme's `Highlight` background, padded to full width.
- Highlight only shown when the prompt list is focused.

### Scrolling

When files exceed the 4-row content area, the list scrolls to keep the cursor visible. Standard navigation keys work: `j`/`k`, `gg`/`G`, `pgup`/`pgdn`.

## Focus Model

Three focus targets cycle via `Tab`:

```
Iterations → Prompts → Timeline → Iterations
```

- **`Tab`**: cycles through all three panes in order.
- **`h` / `←`**: from Timeline, focuses Iterations pane.
- **`l` / `→`**: from any left pane, focuses Timeline.
- **`Enter`** on the prompt list: currently focuses the Timeline (will open read modal in future).

Both Iterations and Prompts are visually in the left column. When the left pane is hidden (terminal < 80 columns), neither can receive focus.

## Mouse Support

- **Target detection**: clicks/scrolls in the left column below the divider target the prompt list; above target the iteration list.
- **Scroll**: 3 lines per wheel tick, same as iteration list.
- **Click**: single click selects a prompt file. Clicking the title row is ignored.
- **Focus**: any mouse interaction switches focus to the targeted pane.
