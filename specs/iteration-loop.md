# Iteration Loop

## Overview

The TUI manages a loop of Claude CLI invocations, similar to the shell script it replaces. Each invocation is called an "iteration". Iterations are grouped into "runs" — each run uses a specific prompt file and iteration count.

## CLI Arguments

```
skinner [--theme=<name>] [--exit] [plan] [max_iterations]
```

| Arguments         | Mode  | Prompt file      | Max iterations |
|-------------------|-------|------------------|----------------|
| (none)            | idle  | —                | —              |
| `build`           | build | PROMPT_BUILD.md  | unlimited      |
| `build 20`        | build | PROMPT_BUILD.md  | 20             |
| `plan`            | plan  | PROMPT_PLAN.md   | unlimited      |
| `plan 5`          | plan  | PROMPT_PLAN.md   | 5              |

When launched with no positional arguments, the TUI starts in **idle mode** — the session timer is not running, the iteration list is empty, and no subprocess is spawned. The user starts a run interactively by selecting a prompt file and pressing `r` (see [prompt-files.md](prompt-files.md) and [run-modal.md](run-modal.md)).

When launched with positional arguments, the first run begins immediately as before.

The `--theme` flag selects a color theme (default: `solarized-dark`). See [theme.md](theme.md).

The `--exit` flag causes the TUI to quit automatically after all iterations complete (or the last iteration fails), rather than remaining open for browsing. `--exit` requires both a prompt mode and an iteration count — it is invalid without them. When used without positional arguments or without a count, print an error and exit:

```
--exit requires a prompt mode and iteration count
Usage: skinner [--theme=<name>] [--exit] <plan|build> <max_iterations>
```

When `--exit` is active:
- The TUI exits with code 0 after the final iteration completes.
- The quit confirmation modal is bypassed entirely (see [quit-confirmation.md](quit-confirmation.md)).
- No user interaction is required — the process exits cleanly on its own.

## Session Phases

The session has three phases:

| Phase | Description |
|-------|-------------|
| **Idle** | TUI loaded, no run in progress. Session timer not ticking. User can browse prompt files and start a run. |
| **Running** | Iterations executing. Session timer ticking. `r` key is disabled. |
| **Finished** | All iterations in the current run completed (or last iteration failed). Session timer paused. User can browse results, select a new prompt, and start another run. |

Launching with CLI args skips the Idle phase and enters Running immediately. After a run completes, the session enters Finished. From Finished, pressing `r` on a prompt file transitions back to Running — the session timer resumes and new iterations append to the existing list.

## Runs

A run is a sequence of iterations using a single prompt file. The session tracks all runs:

```go
type Run struct {
    PromptName string  // display name, e.g. "BUILD"
    PromptFile string  // full path, e.g. "PROMPT_BUILD.md"
    StartIndex int     // first iteration index in session
}
```

Multiple runs can occur within a single session. Each run appends its iterations to the session's iteration list. Run boundaries are displayed as separators in the iteration list (see [tui-layout.md](tui-layout.md)).

## Iteration Lifecycle

1. Read the prompt file content fresh from disk (in case it was edited).
2. Spawn `claude -p --dangerously-skip-permissions --output-format=stream-json --verbose` with the prompt file content piped to stdin.
3. Parse stdout line-by-line as newline-delimited JSON (see [stream-json-format.md](stream-json-format.md)).
4. Track tool calls and their results in real time, updating the TUI.
5. When the subprocess exits:
   - Mark the iteration as completed (exit code 0) or failed (non-zero).
   - Record final duration and tool call count.
6. If max iterations not reached, start the next iteration (go to step 2).
7. If max iterations reached, stop looping. Session enters Finished phase. TUI remains open for browsing.

## Subprocess Management

- Each iteration runs one `claude` subprocess at a time (sequential, not parallel).
- On `ctrl+c` or `q`, show the quit confirmation modal (see [quit-confirmation.md](quit-confirmation.md)). Double `ctrl+c` within 500ms force-quits immediately. Quitting exits the entire TUI.
- The current working directory for the subprocess should be the directory where `skinner` was invoked.

## Session Scope

All iterations across all runs within a single `skinner` invocation form one "session". There is no persistence across separate invocations. Iteration history is kept in memory for the lifetime of the process.
