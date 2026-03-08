# Quit Confirmation

## Overview

A confirmation modal prevents accidental exits. Pressing `q` or `ctrl+c` shows a centered overlay asking the user to confirm before quitting.

## Trigger

| Input | Behavior |
|-------|----------|
| `q` | Show quit confirmation modal |
| `ctrl+c` (single) | Show quit confirmation modal |
| `ctrl+c` (double within 500ms) | Force quit immediately — no modal |

## Modal

A centered overlay displayed on top of the TUI content:

```
  ┌─────────────────────────────┐
  │                             │
  │   Are you sure you want     │
  │   to quit?                  │
  │                             │
  │   y - yes    n - cancel     │
  │                             │
  └─────────────────────────────┘
```

- **Border**: rendered with the theme's `ForegroundDim` color.
- **Text**: rendered with the theme's `Foreground` color.
- **Key hints**: `y`/`n` rendered with the theme's `Highlight` color.

## Keys While Modal Is Open

| Key | Action |
|-----|--------|
| `y` | Quit the application |
| `n` | Dismiss modal, return to TUI |
| `escape` | Dismiss modal, return to TUI |
| Any other key | Ignored |

While the modal is open, all other keybindings are disabled. Navigation, focus changes, and view toggles are blocked until the modal is dismissed.

## Double Ctrl+C

If the user presses `ctrl+c` twice within 500ms, the application force-quits immediately without showing the modal. This is an escape hatch for when the TUI is unresponsive.

Implementation: track the timestamp of the last `ctrl+c`. If a second `ctrl+c` arrives within 500ms of the first, exit immediately. Otherwise, show the confirmation modal.

## --exit Flag Bypass

When skinner is launched with the `--exit` flag, the quit confirmation modal is bypassed entirely. The `--exit` flag indicates automatic operation — the TUI exits cleanly after all iterations complete without any user interaction. See [iteration-loop.md](iteration-loop.md).

## Subprocess Handling

When the user confirms quit (`y`):
1. Kill the current subprocess (if running) via SIGTERM.
2. Exit the TUI with code 0.

The subprocess is **not** killed when the modal is shown — only when the user confirms.
