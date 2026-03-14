# skinner

Keeping an eye on [Ralph](https://ghuntley.com/ralph/).

A Go TUI that wraps Claude CLI and displays tool call activity in real time. I've been ralphing every day and wanted a better experience from my phone to observe what Claude is up to while it works.

This is hyper-personalised software — I built it for myself. You're welcome to use it, steal ideas from it, or build your own version.

> **Heads up:** This is under heavy development and may look completely different tomorrow.

![screenshot](./screenshot.png)


## Install

```
go install github.com/loxstomper/skinner/cmd/skinner@latest
```

Or clone and build locally:

```
git clone https://github.com/loxstomper/skinner.git
cd skinner
make build    # builds ./skinner
make install  # installs to $GOPATH/bin
```

## Usage

```
skinner [--theme=<name>] [--exit] [build|plan] [max_iterations]
```

Skinner runs Claude CLI as a subprocess and renders a two-pane TUI — iterations on the left, tool call timeline on the right. Launch with `build` or `plan` to start immediately, or run without arguments to browse prompts and plans interactively. Press `?` for help.

## Features

- **Two-pane layout** — iterations on the left, tool call timeline on the right (auto-switches to bottom layout on narrow terminals)
- **Tool call groups** — consecutive same-type calls collapse into expandable groups
- **Vim navigation** — hjkl, count+motion, gg/G, configurable keybindings
- **Mouse support** — scroll and click in both panes
- **File explorer** — browse project files with syntax highlighting and git status
- **Git viewer** — commit history with side-by-side diffs
- **Plan & prompt pickers** — browse and select `*_PLAN.md` / `PROMPT_*.md` files
- **Themes** — solarized-dark (default), solarized-light, monokai, nord
- **System stats** — live CPU/memory in the header
- **Token tracking** — input/output/cache tokens, cost, rate limits, context window %
- **TOML config** — `~/.config/skinner/config.toml` for view mode, theme, keybindings, pricing
- **Thinking indicator** — shows elapsed time while waiting for Claude's response

## Details

See [`specs/`](specs/) for design notes.

## License

MIT
