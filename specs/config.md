# Configuration

## Config File

Path: `~/.config/skinner/config.toml`

The config file is optional. If missing or incomplete, defaults apply for all values.

### Format

```toml
[view]
mode = "full"  # "full" or "compact"

[theme]
name = "solarized-dark"

# Per-model pricing (cost per token in USD).
# Prices sourced from https://docs.anthropic.com/en/docs/about-claude/models
# Update these when pricing changes.
[pricing.claude-opus-4-6]
input    = 0.000005
output   = 0.000025
cache_read   = 0.0000005
cache_create = 0.00000625

[pricing.claude-sonnet-4-5]
input    = 0.000003
output   = 0.000015
cache_read   = 0.0000003
cache_create = 0.00000375

[pricing.claude-haiku-4-5]
input    = 0.000001
output   = 0.000005
cache_read   = 0.0000001
cache_create = 0.00000125
```

### Fields

| Section | Key    | Values                  | Default            |
|---------|--------|-------------------------|---------------------|
| `view`  | `mode` | `"full"`, `"compact"`   | `"full"`            |
| `theme` | `name` | Any built-in theme name | `"solarized-dark"`  |

### Pricing

The `[pricing.<model>]` sections define per-token costs in USD. Each model entry has four keys:

| Key            | Description                         |
|----------------|-------------------------------------|
| `input`        | Cost per input token                |
| `output`       | Cost per output token               |
| `cache_read`   | Cost per cache-read input token     |
| `cache_create` | Cost per cache-creation input token |

The model key must match the `message.model` value from the stream-json output (e.g. `claude-opus-4-6`). The built-in defaults cover the current Claude model family. Users can add or update entries when pricing changes.

If the model from a stream event is not found in the pricing table, tokens are tracked but cost is not calculated — the cost display is omitted from the header.

## Defaults

- `view.mode` = `"full"` — show icon + tool name + summary; text blocks up to 3 lines.
- `theme.name` = `"solarized-dark"` — see [theme.md](theme.md) for available themes.
- `pricing` — see below for defaults.

## CLI Overrides

The `--theme` CLI flag overrides `theme.name` from the config file. See [iteration-loop.md](iteration-loop.md) for CLI usage.

## Runtime Overrides

- `v` toggles view mode between full and compact at runtime. This does not persist to the config file. See [keybindings.md](keybindings.md).
