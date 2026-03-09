# skinner

Keeping an eye on [Ralph](https://ghuntley.com/ralph/).

A Go TUI that wraps Claude CLI and displays tool call activity in real time. I've been ralphing every day and wanted a better experience from my phone to observe what Claude is up to while it works.

This is hyper-personalised software — I built it for myself. You're welcome to use it, steal ideas from it, or build your own version.

> **Heads up:** This is under heavy development and may look completely different tomorrow.


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
skinner [--theme=<name>] [--exit] [plan] [max_iterations]
```

Skinner runs Claude CLI as a subprocess and renders a two-pane TUI — iterations on the left, tool call timeline on the right. It reads from `PROMPT_BUILD.md` by default, or `PROMPT_PLAN.md` in plan mode. Navigate with vim-style keybindings, expand tool calls to see details, scroll around. Press `?` for help.

## Details

See [`specs/`](specs/) for design notes.

## License

MIT
