# Help Modal

## Overview

A centered overlay modal displaying all current keybindings. Opened with `?`, dismissed by pressing any key.

## Trigger

| Key | Action |
|-----|--------|
| `?` | Open help modal |

## Modal Layout

A centered overlay with a two-column layout showing action names and their bound keys:

```
  ┌──────────── Keybindings ────────────┐
  │                                     │
  │  Navigation                         │
  │    Move down             j / ↓      │
  │    Move up               k / ↑      │
  │    Jump to top           g g        │
  │    Jump to bottom        G          │
  │    Page down             pgdn       │
  │    Page up               pgup       │
  │                                     │
  │  Focus                              │
  │    Toggle pane           tab        │
  │    Focus left            h / ←      │
  │    Focus right           l / →      │
  │                                     │
  │  Actions                            │
  │    Expand / collapse     enter      │
  │    Plan mode             p          │
  │    Toggle view mode      v          │
  │    Toggle line numbers   #          │
  │    Toggle left pane      [          │
  │                                     │
  │  File Explorer                      │
  │    Search files          /          │
  │    Open in editor        e          │
  │    Expand / collapse     enter      │
  │    Back / exit           escape     │
  │                                     │
  │  Global                             │
  │    Quit                  q          │
  │    Force quit            ctrl+c ×2  │
  │    Help                  ?          │
  │                                     │
  │          Press any key to close     │
  └─────────────────────────────────────┘
```

## Styling

- **Border**: theme's `ForegroundDim` color.
- **Title** ("Keybindings"): theme's `Foreground` color, bold.
- **Section headers** ("Navigation", "Focus", etc.): theme's `Foreground` color, bold.
- **Action names**: theme's `Foreground` color.
- **Key names**: theme's `Highlight` color.
- **Footer** ("Press any key to close"): theme's `ForegroundDim` color.

## Behavior

- While the modal is open, all other keybindings are disabled.
- Pressing **any** key (letter, number, arrow, escape, enter, etc.) dismisses the modal.
- The modal reflects the user's configured keybindings (from the `[keybindings]` section of the config file), not hardcoded defaults. If a user has remapped `q` to `x`, the modal shows `x` for Quit.

See [keybindings.md](keybindings.md) for the default keybindings and [config.md](config.md) for configurable keymapping.

## Sizing

- The modal width is fixed at 40 characters (or adapts to the longest action+key pair plus padding).
- The modal is horizontally and vertically centered in the terminal.
- If the terminal is too small to display the full modal, the content scrolls vertically within the modal bounds.
