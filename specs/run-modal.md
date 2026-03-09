# Run Modal

## Overview

A small centered modal that prompts the user for the number of iterations before starting a run. Opened by pressing `r` on the prompt file picker or within the prompt read modal.

## Trigger

| Context | Key | Action |
|---------|-----|--------|
| Prompt picker (focused, file selected) | `r` | Open run modal for selected prompt file |
| Prompt read modal (viewing a file) | `r` | Open run modal for viewed prompt file |

The `r` key is disabled while a run is in progress (session phase is Running). In the read modal, the `r to run` footer hint is hidden during a run.

## Modal Layout

A centered overlay displayed on top of the TUI content:

```
  ┌─────────────────────────┐
  │                         │
  │   Iterations: [10    ]  │
  │                         │
  │   enter to start        │
  │   esc to cancel         │
  │                         │
  └─────────────────────────┘
```

- **Border**: rendered with the theme's `ForegroundDim` color.
- **Label** ("Iterations:"): rendered with the theme's `Foreground` color.
- **Input field**: rendered with the theme's `Foreground` color, cursor visible.
- **Hints** ("enter to start", "esc to cancel"): rendered with the theme's `ForegroundDim` color.

## Input Field

- **Pre-fill**: `10` on first use. On subsequent uses within the same session, pre-filled with the last entered value.
- **Selection**: the pre-filled value is fully selected so typing immediately replaces it.
- **Validation**: only digit characters (`0`-`9`) are accepted. Non-numeric input is ignored.
- **Value of `0`**: means unlimited iterations (no max).
- **Empty field**: treated as invalid — `enter` does nothing until a value is entered.

## Keys While Modal Is Open

| Key | Action |
|-----|--------|
| `0`-`9` | Enter/replace digit in input field |
| `backspace` | Delete last digit |
| `enter` | Start the run with the entered iteration count |
| `esc` | Dismiss modal, return to previous context |
| Any other key | Ignored |

While the modal is open, all other keybindings are disabled.

## Behavior on Enter

When the user presses `enter` with a valid value:

1. The run modal closes.
2. If the prompt read modal was open, it also closes.
3. The prompt file content is read fresh from disk (in case it was edited via `e`).
4. A new `Run` is created and appended to the session (see [iteration-loop.md](iteration-loop.md)).
5. The session transitions to the Running phase.
6. The session timer starts (or resumes if this is not the first run).
7. Iterations begin executing.

## Behavior on Escape

The modal is dismissed. If opened from the read modal, the read modal remains open. If opened from the prompt picker, focus returns to the prompt picker. No run is started.
