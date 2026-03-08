# Iteration Loop

## Overview

The TUI manages a loop of Claude CLI invocations, similar to the shell script it replaces. Each invocation is called an "iteration".

## CLI Arguments

```
skinner [--theme=<name>] [--exit] [plan] [max_iterations]
```

| Arguments         | Mode  | Prompt file      | Max iterations |
|-------------------|-------|------------------|----------------|
| (none)            | build | PROMPT_BUILD.md  | unlimited      |
| `20`              | build | PROMPT_BUILD.md  | 20             |
| `plan`            | plan  | PROMPT_PLAN.md   | unlimited      |
| `plan 5`          | plan  | PROMPT_PLAN.md   | 5              |

The `--theme` flag selects a color theme (default: `solarized-dark`). See [theme.md](theme.md).

The `--exit` flag causes the TUI to quit automatically after all iterations complete (or the last iteration fails), rather than remaining open for browsing. When `--exit` is active:
- The TUI exits with code 0 after the final iteration completes.
- The quit confirmation modal is bypassed entirely (see [quit-confirmation.md](quit-confirmation.md)).
- No user interaction is required — the process exits cleanly on its own.

## Iteration Lifecycle

1. Read the prompt file content.
2. Spawn `claude -p --dangerously-skip-permissions --output-format=stream-json --verbose` with the prompt file content piped to stdin.
3. Parse stdout line-by-line as newline-delimited JSON (see [stream-json-format.md](stream-json-format.md)).
4. Track tool calls and their results in real time, updating the TUI.
5. When the subprocess exits:
   - Mark the iteration as completed (exit code 0) or failed (non-zero).
   - Record final duration and tool call count.
6. If max iterations not reached, start the next iteration (go to step 2).
7. If max iterations reached, stop looping. TUI remains open for browsing.

## Subprocess Management

- Each iteration runs one `claude` subprocess at a time (sequential, not parallel).
- On `ctrl+c` or `q`, show the quit confirmation modal (see [quit-confirmation.md](quit-confirmation.md)). Double `ctrl+c` within 500ms force-quits immediately.
- The current working directory for the subprocess should be the directory where `skinner` was invoked.

## Session Scope

All iterations within a single `skinner` invocation form one "session". There is no persistence across separate runs. Iteration history is kept in memory for the lifetime of the process.
