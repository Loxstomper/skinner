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
| [quit-confirmation.md](quit-confirmation.md) | Quit confirmation modal, double ctrl+c force quit |
| [token-usage.md](token-usage.md) | Rate limit window display (header), per-tool-call token counts |
| [sub-scroll.md](sub-scroll.md) | Adaptive sizing and sub-scroll for expanded tool call content |
| [line-numbers.md](line-numbers.md) | Relative line numbers, vim-style count+j/k jump motions |
| [help-modal.md](help-modal.md) | Keybinding help overlay, configurable keymapping display |
