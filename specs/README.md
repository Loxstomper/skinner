# Skinner — Specifications

Skinner is a Go TUI that runs Claude CLI in a loop and displays tool call activity in real time.

## Specs

| Spec | Description |
|------|-------------|
| [iteration-loop.md](iteration-loop.md) | CLI arguments, iteration lifecycle, subprocess management |
| [tui-layout.md](tui-layout.md) | Header bar, two-pane layout: iteration list (left) and message timeline (right) |
| [tool-call-groups.md](tool-call-groups.md) | Collapsible groups for consecutive same-type tool calls |
| [stream-json-format.md](stream-json-format.md) | Claude CLI stream-json event types and how to parse them |
| [duration-tracking.md](duration-tracking.md) | Wallclock timing for tool calls and iterations |
| [keybindings.md](keybindings.md) | Keyboard navigation and controls |
| [theme.md](theme.md) | Color theme system, built-in themes, `--theme` flag |
| [mouse.md](mouse.md) | Mouse scroll and click support for both panes |
| [config.md](config.md) | TOML config file, view mode, theme, CLI overrides |
| [tech-stack.md](tech-stack.md) | Go, Bubble Tea, Lip Gloss, standard library |
| [architecture.md](architecture.md) | Package structure, layered design, interfaces, testing strategy |
