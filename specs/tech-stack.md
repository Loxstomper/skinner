# Tech Stack

## Language

Go

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework (model-view-update architecture).
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — Styling and layout for the two-pane design.
- Standard library `os/exec` — Spawning and managing the `claude` subprocess.
- Standard library `encoding/json` — Parsing stream-json output.
- Standard library `bufio` — Line-by-line reading of subprocess stdout.

## Build

Standard `go build`. Single binary output named `skinner`.
