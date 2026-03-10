# Path Trimming

## Overview

File paths in tool call summaries and expanded views are automatically trimmed to save horizontal space. This is always enabled — no configuration needed.

## Trimming Rules

Paths are trimmed using two rules, applied in order:

1. **CWD prefix** — if the path starts with the current working directory followed by `/`, strip that prefix. The CWD is the directory where skinner launched the Claude CLI subprocess.
2. **Home directory prefix** — if the path starts with `$HOME/`, replace that prefix with `~/`.

If neither rule matches, the path is shown in full.

### Examples

Given CWD `/home/lox/Development/skinner`:

| Raw path                                          | Trimmed                    |
|---------------------------------------------------|----------------------------|
| `/home/lox/Development/skinner/internal/tui/view.go` | `internal/tui/view.go`     |
| `/home/lox/.config/skinner/config.toml`           | `~/.config/skinner/config.toml` |
| `/etc/hosts`                                      | `/etc/hosts`               |

## Where It Applies

Path trimming applies everywhere a file path is displayed:

- **Tool call summary rows** — the one-line summary for Read, Edit, Write, Grep, Glob.
- **Tool call group headers** — when groups are collapsed, any path in the summary is trimmed.
- **Expanded detail headers** — the path shown at the top of expanded tool call content.

It does **not** apply to:
- File contents within expanded views (the actual file text is unchanged).
- The header bar (no paths are displayed there).

## Applicable Tools

| Tool  | Path field(s) trimmed                |
|-------|--------------------------------------|
| Read  | `input.file_path`                    |
| Edit  | `input.file_path`                    |
| Write | `input.file_path`                    |
| Grep  | `input.path` (if present)            |
| Glob  | `input.pattern` (if it contains an absolute path prefix) |
| Bash  | Not trimmed (command string, not a path) |
| Task  | Not trimmed (description, not a path) |
