# Keybindings

All keybindings listed here are defaults. Users can remap any action via the `[keybindings]` section in the config file — see [config.md](config.md). The help modal (`?`) always reflects the active keybinding configuration — see [help-modal.md](help-modal.md).

## Focus

| Key              | Action                                      |
|------------------|---------------------------------------------|
| `tab`            | Cycle focus: Iterations → Prompts → Timeline |
| `h` / `←`       | Focus iterations pane (from timeline)       |
| `l` / `→`       | Focus timeline pane                         |

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

In the **prompts pane**, cursor movement selects a prompt file. See [prompt-files.md](prompt-files.md).

In the **timeline pane**, cursor movement highlights individual items (text blocks or tool call rows). Digit keys (`1`–`9`) accumulate a count prefix for `j`/`k` jump motions — see [line-numbers.md](line-numbers.md).

## Actions

| Key              | Action                                      |
|------------------|---------------------------------------------|
| `enter`          | Focus timeline (iterations/prompts pane); expand/collapse selected text block, tool call, or tool call group (timeline); enter sub-scroll mode (on already-expanded tool call) |
| `r`              | Start a run from selected prompt file (prompt picker or read modal); opens iterations input modal. Disabled while a run is in progress. |
| `escape`         | Exit sub-scroll mode (returns to timeline); dismiss modal |

## View

| Key              | Action                                      |
|------------------|---------------------------------------------|
| `v`              | Toggle between full and compact view mode   |
| `#`              | Toggle relative line numbers on/off         |
| `[`              | Toggle left pane visibility                 |

## Global

| Key              | Action                                      |
|------------------|---------------------------------------------|
| `q`              | Show quit confirmation modal                |
| `ctrl+c`         | Show quit confirmation modal (double within 500ms to force quit) |
| `?`              | Show help modal (all keybindings)           |

See [quit-confirmation.md](quit-confirmation.md) for quit behavior and [help-modal.md](help-modal.md) for the help overlay.
