# Keybindings

All keybindings listed here are defaults. Users can remap any action via the `[keybindings]` section in the config file — see [config.md](config.md). The help modal (`?`) always reflects the active keybinding configuration — see [help-modal.md](help-modal.md).

## Focus

| Key              | Action                                      |
|------------------|---------------------------------------------|
| `tab`            | Cycle focus: Plans → Iterations → Prompts → Timeline (side layout); Timeline → Plans → Iterations → Prompts (bottom layout) |
| `h` / `←`       | Focus left pane / bottom bar (Plans from plan content view; Iterations from timeline) |
| `l` / `→`       | Focus right pane / main area (plan content view or timeline) |

## Navigation (operates on focused pane)

| Key              | Action                                      |
|------------------|---------------------------------------------|
| `j` / `↓`       | Move cursor down (accepts `{count}` prefix) |
| `k` / `↑`       | Move cursor up (accepts `{count}` prefix)   |
| `g g` / `Home`  | Jump to top                                 |
| `G` / `End`     | Jump to bottom                              |
| `pgdn`           | Page scroll down; cursor moves into visible viewport if needed |
| `pgup`           | Page scroll up; cursor moves into visible viewport if needed  |

In the **iterations pane**, cursor movement selects which iteration is displayed in the timeline.

In the **plans pane**, cursor movement selects a plan file and live-updates the right pane with the rendered plan content. See [plan-files.md](plan-files.md).

In the **prompts pane**, cursor movement selects a prompt file. See [prompt-files.md](prompt-files.md).

In the **timeline pane**, cursor movement highlights individual items (text blocks or tool call rows). Digit keys (`1`–`9`) accumulate a count prefix for `j`/`k` jump motions — see [line-numbers.md](line-numbers.md).

## Actions

| Key              | Action                                      |
|------------------|---------------------------------------------|
| `enter`          | Focus timeline (iterations/prompts pane); expand/collapse selected text block, tool call, or tool call group (timeline); enter sub-scroll mode (on already-expanded tool call) |
| `e`              | Open `$EDITOR` for the selected plan file (plan list or plan content view) or prompt file (prompt read modal) |
| `p`              | Enter plan mode — launch Claude CLI interactively. Disabled while a run is in progress. See [plan-mode.md](plan-mode.md). |
| `r`              | Start a run from selected prompt file (prompt picker or read modal); opens iterations input modal. Disabled while a run is in progress. |
| `escape`         | Exit sub-scroll mode (returns to timeline); dismiss modal |

## View

| Key              | Action                                      |
|------------------|---------------------------------------------|
| `v`              | Toggle between full and compact view mode   |
| `#`              | Toggle relative line numbers on/off         |
| `[`              | Toggle left pane (side layout) or bottom bar (bottom layout) visibility |

## Global

| Key              | Action                                      |
|------------------|---------------------------------------------|
| `q`              | Show quit confirmation modal                |
| `ctrl+c`         | Show quit confirmation modal (double within 500ms to force quit) |
| `?`              | Show help modal (all keybindings)           |
| `ctrl+g`         | Enter git view (read-only commit history and diffs) |

See [quit-confirmation.md](quit-confirmation.md) for quit behavior and [help-modal.md](help-modal.md) for the help overlay.
