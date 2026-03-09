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

### Run Key

Pressing `r` while the prompt picker is focused and a file is selected opens the iterations input modal (see [run-modal.md](run-modal.md)). The selected prompt file is passed to the modal. `r` is disabled while a run is in progress.

### Scrolling

When files exceed the 4-row content area, the list scrolls to keep the cursor visible. Standard navigation keys work: `j`/`k`, `gg`/`G`, `pgup`/`pgdn`.

## Focus Model

Four focus targets cycle via `Tab`:

```
Plans → Iterations → Prompts → Timeline → Plans
```

- **`Tab`**: cycles through all four panes in order.
- **`h` / `←`**: from Timeline focuses Iterations pane; from plan content view focuses Plans pane.
- **`l` / `→`**: from any left pane, focuses the right pane (timeline or plan content view).
- **`Enter`** on the prompt list: opens the prompt read modal for the selected file.

Plans, Iterations, and Prompts are visually in the left column. When the left pane is hidden (terminal < 80 columns), none can receive focus. See [plan-files.md](plan-files.md) for the plan file picker.

## Mouse Support

- **Target detection**: clicks/scrolls in the left column below the divider target the prompt list; above target the iteration list.
- **Scroll**: 3 lines per wheel tick, same as iteration list.
- **Click**: single click selects a prompt file. Clicking the title row is ignored.
- **Focus**: any mouse interaction switches focus to the targeted pane.

## Prompt Read Modal

Full-screen centered overlay (same pattern as help/quit modals) for viewing prompt file content.

### Appearance

- **Size**: ~80% of terminal width and height.
- **Title bar**: full filename (e.g. `PROMPT_BUILD.md`) injected into the top border, centered.
- **Content**: plain text with absolute line numbers in a dimmed gutter (right-aligned, `ForegroundDim`).
- **Footer**: `e to edit · r to run · esc to close` centered at the bottom, rendered in `ForegroundDim`. The `r to run` hint is hidden while a run is in progress.

### Navigation

- **`j` / `↓`**: scroll down one line.
- **`k` / `↑`**: scroll up one line.
- **`pgdn`**: scroll down 10 lines.
- **`pgup`**: scroll up 10 lines.
- **`esc`**: dismiss modal.
- All other keys are blocked while the modal is open.

### Run Key

- **`r`**: opens the iterations input modal (see [run-modal.md](run-modal.md)) with the currently viewed prompt file. Disabled while a run is in progress (the hint is hidden from the footer).

### Editor Integration

- **`e`**: suspends the TUI and opens `$EDITOR` (fallback: `vi`) with the prompt file path.
- On editor exit: modal dismisses, TUI resumes, prompt file list is rescanned.
