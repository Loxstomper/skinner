# Configuration

## Config File

Path: `~/.config/skinner/config.toml`

The config file is optional. If missing or incomplete, defaults apply for all values.

### Format

```toml
[view]
mode = "full"            # "full" or "compact"
layout = "auto"          # "side", "bottom", "auto"
line_numbers = true      # show relative line numbers in right pane

[theme]
name = "solarized-dark"

# Keybinding overrides. Only include keys you want to change.
# Values are key strings: single keys ("q"), modifiers ("ctrl+c"), or sequences ("g g").
[keybindings]
# quit = "q"
# plan_mode = "p"
# help = "?"
# toggle_left_pane = "["
# toggle_line_numbers = "#"
# toggle_view = "v"
# focus_left = "h"
# focus_right = "l"
# focus_toggle = "tab"
# move_down = "j"
# move_up = "k"
# jump_top = "g g"
# jump_bottom = "G"
# expand = "enter"
# escape = "escape"

[plan]
command = 'claude "study specs/README.md"'

# Per-model pricing (cost per token in USD) and context window size.
# Prices sourced from https://docs.anthropic.com/en/docs/about-claude/models
# Update these when pricing changes.
[pricing.claude-opus-4-6]
input          = 0.000005
output         = 0.000025
cache_read     = 0.0000005
cache_create   = 0.00000625
context_window = 200000

[pricing.claude-sonnet-4-5]
input          = 0.000003
output         = 0.000015
cache_read     = 0.0000003
cache_create   = 0.00000375
context_window = 200000

[pricing.claude-haiku-4-5]
input          = 0.000001
output         = 0.000005
cache_read     = 0.0000001
cache_create   = 0.00000125
context_window = 200000
```

### Fields

| Section | Key    | Values                  | Default            |
|---------|--------|-------------------------|---------------------|
| `view`  | `mode` | `"full"`, `"compact"`   | `"full"`            |
| `view`  | `layout` | `"side"`, `"bottom"`, `"auto"` | `"auto"`    |
| `view`  | `line_numbers` | `true`, `false`  | `true`             |
| `theme` | `name` | Any built-in theme name | `"solarized-dark"`  |

### Keybindings

The `[keybindings]` section allows remapping any action to a different key. Only include entries you want to override — omitted entries use the hardcoded defaults from [keybindings.md](keybindings.md).

| Key | Action | Default |
|-----|--------|---------|
| `quit` | Show quit confirmation | `"q"` |
| `help` | Show help modal | `"?"` |
| `toggle_left_pane` | Toggle left pane visibility | `"["` |
| `toggle_line_numbers` | Toggle relative line numbers | `"#"` |
| `toggle_view` | Toggle full/compact view | `"v"` |
| `focus_left` | Focus left pane | `"h"` |
| `focus_right` | Focus right pane | `"l"` |
| `focus_toggle` | Toggle focus between panes | `"tab"` |
| `move_down` | Move cursor down | `"j"` |
| `move_up` | Move cursor up | `"k"` |
| `jump_top` | Jump to top | `"g g"` |
| `jump_bottom` | Jump to bottom | `"G"` |
| `expand` | Expand/collapse item | `"enter"` |
| `plan_mode` | Enter plan mode | `"p"` |
| `escape` | Exit sub-scroll / dismiss modal | `"escape"` |

Key string format:
- Single keys: `"q"`, `"v"`, `"#"`, `"["`, `"?"`, `"G"`
- Modifier keys: `"ctrl+c"`, `"ctrl+k"`, `"alt+j"`
- Key sequences: `"g g"` (press `g` twice)
- Special keys: `"enter"`, `"escape"`, `"tab"`, `"pgup"`, `"pgdn"`, `"home"`, `"end"`

Note: `ctrl+c` quit behavior (single = modal, double within 500ms = force quit) is not configurable. Arrow key alternatives (`←`/`→`/`↑`/`↓`) are always active alongside their letter equivalents and are not independently configurable.

### Pricing

The `[pricing.<model>]` sections define per-token costs in USD. Each model entry has four keys:

| Key              | Description                                   |
|------------------|-----------------------------------------------|
| `input`          | Cost per input token                          |
| `output`         | Cost per output token                         |
| `cache_read`     | Cost per cache-read input token               |
| `cache_create`   | Cost per cache-creation input token           |
| `context_window` | Max context window size in tokens (e.g. 200000) |

The model key must match the `message.model` value from the stream-json output (e.g. `claude-opus-4-6`). The built-in defaults cover the current Claude model family. Users can add or update entries when pricing or context window sizes change.

If the model from a stream event is not found in the pricing table, tokens are tracked but cost and context window percentage are not calculated — both displays are omitted from the header.

### Plan Mode

| Section | Key | Values | Default |
|---------|-----|--------|---------|
| `plan` | `command` | Any shell command string | `'claude "study specs/README.md"'` |

The command is executed via `sh -c`, so shell quoting, environment variables, and pipes are supported. See [plan-mode.md](plan-mode.md).

## Defaults

- `view.mode` = `"full"` — show icon + tool name + summary; text blocks up to 3 lines.
- `view.layout` = `"auto"` — bottom layout when width < 80, side layout when ≥ 80. See [bottom-layout.md](bottom-layout.md).
- `view.line_numbers` = `true` — show relative line numbers in the right pane gutter. See [line-numbers.md](line-numbers.md).
- `theme.name` = `"solarized-dark"` — see [theme.md](theme.md) for available themes.
- `plan.command` = `'claude "study specs/README.md"'` — see [plan-mode.md](plan-mode.md).
- `keybindings` — all actions use hardcoded defaults. See [keybindings.md](keybindings.md).
- `pricing` — see below for defaults.

## CLI Overrides

The `--theme` CLI flag overrides `theme.name` from the config file. See [iteration-loop.md](iteration-loop.md) for CLI usage.

## Runtime Overrides

- `v` toggles view mode between full and compact at runtime. This does not persist to the config file.
- `#` toggles relative line numbers on/off at runtime. This does not persist to the config file.
- `[` toggles left pane visibility at runtime. This does not persist to the config file.

See [keybindings.md](keybindings.md) for all runtime controls.
