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
