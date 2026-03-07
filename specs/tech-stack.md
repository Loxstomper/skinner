# Tech Stack

## Language

Go

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework (model-view-update architecture).
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — Styling and layout for the two-pane design.
- Standard library `os/exec` — Spawning and managing the `claude` subprocess (behind the executor interface).
- Standard library `encoding/json` — Parsing stream-json output.
- Standard library `bufio` — Line-by-line reading of subprocess stdout.

## Architecture

The codebase is layered for testability. Business logic, subprocess management, and TUI rendering are separated into independent packages with clear dependency boundaries. Side effects are isolated behind interfaces. See [architecture.md](architecture.md) for the full package map, interface definitions, and testing strategy.

### Package Overview

| Package | Role |
|---------|------|
| `internal/model` | Pure data types (Session, Iteration, ToolCall, etc.) |
| `internal/parser` | Stream-JSON line parsing (pure functions) |
| `internal/config` | TOML config loading, pricing defaults |
| `internal/theme` | Color theme definitions and lookup |
| `internal/session` | Business logic controller — event processing, grouping, cost tracking, iteration lifecycle |
| `internal/executor` | Subprocess abstraction — interface + real `os/exec` implementation |
| `internal/tui` | Bubble Tea components — root coordinator, header, iteration list, timeline, and pure helpers |

## Build

A `Makefile` provides the standard targets:

| Target    | Description                        |
|-----------|------------------------------------|
| `build`   | Compile the `skinner` binary       |
| `clean`   | Remove the compiled binary         |
| `test`    | Run all tests (`go test ./...`)    |
| `fmt`     | Format source with `gofmt`         |
| `lint`    | Run `golangci-lint`                |
| `vet`     | Run `go vet` on all packages       |
| `check`   | Run `vet` + `lint` + `test`        |
| `install` | Install binary to `GOPATH/bin`     |
| `run`     | Build and run `skinner`            |

## Testing

Testing is a first-class concern. Every package except `cmd/skinner` has unit tests. The layered architecture ensures most logic is testable without a terminal, subprocess, or wall-clock time:

- **Pure function packages** (`parser`, `model`, `theme`, `tui/format`, `tui/cursor`, `tui/autofollow`) are tested with table-driven unit tests.
- **Session controller** is tested by feeding typed events and asserting session state, using an injectable clock for deterministic timing.
- **TUI components** are tested by constructing components with props and asserting `View()` output or state after `Update()` calls.
- **Integration tests** use a fake executor to drive the full root model without a real subprocess.
