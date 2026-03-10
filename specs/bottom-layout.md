# Bottom Layout

## Overview

An alternative layout that stacks the sidebar sections at the bottom of the screen instead of to the left. Designed for narrow, tall terminals (e.g. phone screens) where horizontal space is limited but vertical space is abundant.

## Configuration

```toml
[view]
layout = "auto"  # "side", "bottom", "auto"
```

| Value    | Behavior                                              |
|----------|-------------------------------------------------------|
| `side`   | Traditional left sidebar layout                       |
| `bottom` | Bottom bar layout                                     |
| `auto`   | Bottom when terminal width < 80 columns, side when ≥ 80 |

Default: `"auto"`.

In `auto` mode, resizing the terminal past the threshold switches layout live. Focus is preserved across layout switches — if Iterations was focused in bottom mode, it stays focused in side mode.

## Structure

The bottom bar replaces the left pane entirely. When bottom layout is active, no left pane is rendered. The timeline (or plan content view) occupies the full width above the bottom bar.

```
┌──────────────────────────────┐
│ ⏱ 14m32s  ↑42.1k ↓8.3k   ⟳ │  ← header (1 line)
├──────────────────────────────┤
│                              │
│  Timeline / Plan content     │  ← remaining height
│                              │
│                              │
├── Plans ─────────────────────┤
│  PLAN_A.md                   │  ← 2 lines, scrollable
│ [PLAN_B.md]                  │
├── Iterations ────────────────┤
│  Iter 2 ✓  (1m48s)          │  ← 2 lines, scrollable
│  Iter 3 ⟳  (0m32s)          │
├── Prompts ───────────────────┤
│  PROMPT_A.md                 │  ← 2 lines, scrollable
│ [PROMPT_B.md]                │
└──────────────────────────────┘
```

Total bottom bar height: 9 rows (3 divider lines + 6 content lines).

## Sections

Each section displays a 2-line scrollable vertical list. Items are rendered in the same format as the side layout — one item per row, full width.

### Dividers

Each section has a labeled divider line above it using `─` in `ForegroundDim` with the section name in bold `Foreground` color:

```
── Plans ──────────────────────
── Iterations ─────────────────
── Prompts ────────────────────
```

Dividers are always visible, even when a section is empty. Empty sections show 2 blank rows below their divider.

### Plans Section

Lists `*_PLAN.md` files. Same content and behavior as the plan file picker in the side layout (see [plan-files.md](plan-files.md)). Selected item highlighted with `Highlight` background. When focused, the main area shows rendered plan content.

### Iterations Section

Lists iterations. Same format as the side layout iteration list: status icon, iteration number, duration. Selected item highlighted with `Highlight` background.

**Run separators are not shown** in bottom layout — only the flat iteration list is displayed. The 2-line window is too small for separator lines to be useful.

**Auto-follow**: same rules as side layout — follows latest iteration during a run, pauses on manual selection.

### Prompts Section

Lists `PROMPT_*.md` files. Same content and behavior as the prompt file picker in the side layout (see [prompt-files.md](prompt-files.md)). Selected item highlighted with `Highlight` background.

## Focus

Tab cycles in visual top-to-bottom order: **Timeline → Plans → Iterations → Prompts → Timeline**.

This differs from side layout order (Plans → Iterations → Prompts → Timeline) to match the visual position of panes on screen.

`h`/`←` and `l`/`→` navigate between the main area and bottom bar:
- From the main area, `h`/`←` is a no-op (no pane to the left). `j`/`↓` at the bottom of the timeline does not move into the bottom bar — use Tab.
- From the bottom bar, `l`/`→` focuses the main area (timeline or plan content).
- From the main area, `h`/`←` focuses the last-focused bottom bar section.

## Toggle

`[` toggles the bottom bar on/off. When hidden, the main area gets full terminal height. Same key as the side layout left pane toggle.

## Mouse

- **Target section**: determined by Y coordinate. Clicks/scrolls in the main area target the timeline or plan content view. Clicks/scrolls in the bottom bar region target the section under the pointer, determined by Y offset relative to the bottom bar start.
- **Scroll**: 3 lines per wheel tick within the targeted section. Scrolling a section switches focus to it.
- **Click**: selects the item at the clicked row within the targeted section. Click-to-expand rules are unchanged for the timeline (see [mouse.md](mouse.md)).
- **Section boundaries**: clicks on divider lines are ignored.
