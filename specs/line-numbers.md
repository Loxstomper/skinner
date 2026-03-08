# Relative Line Numbers

## Overview

The right pane (message timeline) displays relative line numbers in a gutter, enabling vim-style `{count}j`/`{count}k` jump motions. Line numbers are on by default and can be toggled at runtime or configured in the config file.

## Display

A gutter column on the left side of the right pane shows line numbers:

```
  3  Looking at the test failures to understand
  2  what's going wrong with the parser module.
  1  The error suggests a type mismatch...
  0    Read   src/main.go (85 lines)              ✓   0.8s      ← cursor
  1  The test expects a return value of 42 but
  2  the function returns a string. I need to
  3  fix the return type.
  4    Edit   src/main.go (+2/-2)                 ✓   0.3s
```

- **Line 0**: The current cursor position. Rendered with the theme's `Highlight` color.
- **Other lines**: Relative distance from the cursor. Rendered in `ForegroundDim`.
- **Gutter width**: 3 characters plus a space separator (4 columns total). Numbers are right-aligned within the gutter.
- **Numbering unit**: Each timeline item (text block or tool call row) is one line number. Expanded content lines within a tool call share the same line number as their parent row.

## Vim Jump Motions

When the right pane is focused, the user can type a number prefix followed by `j` or `k` to jump by that many items:

| Input | Action |
|-------|--------|
| `5j` | Move cursor down 5 items |
| `12k` | Move cursor up 12 items |
| `j` | Move cursor down 1 item (existing behavior) |
| `k` | Move cursor up 1 item (existing behavior) |

### Number Accumulator

- Digit keys (`0`–`9`) accumulate into a count buffer when the right pane is focused.
- When `j` or `k` is pressed, the accumulated count is consumed as the jump distance, then the buffer is cleared.
- If no digits have been typed, `j`/`k` move by 1 (default behavior).
- The count buffer is cleared on any non-digit, non-`j`/`k` keypress.
- `0` as the first digit is not accumulated (to avoid confusion with vim's `0` go-to-beginning-of-line). A leading `0` is ignored; `05j` behaves the same as `5j`.

### Display of Pending Count

While digits are being accumulated, show the current count in the bottom-right corner of the right pane in `ForegroundDim`:

```
                                                          12
```

This gives visual feedback that a count is being entered.

## Configuration

### Config File

```toml
[view]
line_numbers = true  # true or false, default: true
```

### Runtime Toggle

| Key | Action |
|-----|--------|
| `#` | Toggle relative line numbers on/off |

The toggle does not persist to the config file.

See [config.md](config.md) for config file format and [keybindings.md](keybindings.md) for keybinding reference.
