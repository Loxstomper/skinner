# Hooks

## Overview

Hooks are user-defined shell commands that run at specific points in the iteration lifecycle. They allow external scripts to control iteration behavior, perform cleanup, or react to events without modifying Skinner itself.

## Hook Types

There are two categories of hooks, distinguished by their prefix:

| Prefix | Behavior | Stdout | Failure |
|--------|----------|--------|---------|
| `pre-*` | **Blocking** — Skinner waits for the command to finish before proceeding. | Parsed as JSON (see contract below). | Stops the loop. The iteration is not started. Session enters Finished phase. |
| `on-*` | **Fire-and-forget** — Skinner spawns the command and does not wait for it. | Ignored. | Ignored. The loop continues normally. |

## Defined Hooks

| Hook | Type | When Fired |
|------|------|------------|
| `pre-iteration` | `pre-*` | Before each iteration starts, after the prompt file is read but before the Claude subprocess is spawned. |
| `on-iteration-end` | `on-*` | After each iteration's Claude subprocess exits (success or failure). |
| `on-error` | `on-*` | After an iteration exits with a non-zero exit code. Fired in addition to `on-iteration-end`. |
| `on-idle` | `on-*` | When the session enters Idle or Finished phase (no run in progress). |

### Firing Order

When an iteration completes successfully:
1. `on-iteration-end`
2. If max iterations reached: `on-idle`

When an iteration fails (non-zero exit):
1. `on-iteration-end`
2. `on-error`
3. `on-idle` (loop stops on failure)

When a `pre-iteration` hook fails or signals done:
1. `on-idle`

## pre-iteration JSON Contract

The `pre-iteration` hook's stdout is read in full after the process exits and parsed as a single JSON object. The hook communicates intent through the following keys:

| Key | Type | Description |
|-----|------|-------------|
| `prompt` | string | Replacement prompt for this iteration. |
| `title` | string | Header text displayed in the timeline pane. |
| `done` | bool | When `true`, stop the loop. |

### Provide a prompt

```json
{"prompt": "Fix the failing tests in auth_test.go"}
```

When `prompt` is present, its value replaces the prompt file content for this iteration only. The prompt file on disk is not modified. The next iteration re-reads the prompt file as usual.

### Set a title

```json
{"title": "Fixing auth tests"}
```

When `title` is present, its value appears as a header in the timeline pane for this iteration. The title is optional — if omitted, no header is shown. When `done` is `true`, `title` is ignored (the iteration never starts, so there is nothing to display).

`title` can be combined with `prompt` to label the work being done:

```json
{"title": "Auth test fixes", "prompt": "Fix the failing tests in auth_test.go"}
```

### Signal completion

```json
{"done": true}
```

When `done` is `true`, the loop stops. No more iterations are started. The session enters Finished phase.

### Default behavior

If stdout is empty or is not valid JSON, the iteration proceeds normally using the prompt file content. This means a hook that simply exits 0 with no output has no effect on the iteration.

### Precedence

If `done` is `true`, the loop stops — `prompt` and `title` are ignored.

## Execution Model

All hooks are executed via `sh -c <command>`, inheriting the working directory from the Skinner process (the directory where `skinner` was invoked).

### Environment Variables

The following environment variables are set for every hook invocation:

| Variable | Description | Example |
|----------|-------------|---------|
| `SKINNER_HOOK` | Name of the hook being fired. | `pre-iteration` |
| `SKINNER_ITERATION` | 1-based index of the current iteration within the session. For `pre-iteration`, this is the iteration about to start. For `on-*` hooks, this is the iteration that just finished. Not set for `on-idle` when no iterations have run. | `3` |
| `SKINNER_ITERATION_EXIT` | Exit code of the Claude subprocess. Only set for `on-iteration-end` and `on-error`. | `1` |
| `SKINNER_PROMPT_FILE` | Path to the prompt file for the current run. Not set during idle with no prior run. | `PROMPT_BUILD.md` |
| `SKINNER_MAX_ITERATIONS` | Max iteration count for the current run, or `unlimited` if no limit was set. | `20` |
| `SKINNER_RUN_INDEX` | 0-based index of the current run within the session. | `0` |

### Timeouts

Each hook type has a configurable timeout. If a hook exceeds its timeout, the process is killed (SIGKILL).

| Hook type | Default timeout |
|-----------|----------------|
| `pre-*` | 30s |
| `on-*` | 10s |

## TOML Configuration

Hooks are configured in `~/.config/skinner/config.toml` under the `[hooks]` section. Each key is a hook name, and its value is the shell command string.

```toml
[hooks]
pre-iteration = "./scripts/check-ready.sh"
on-iteration-end = "echo 'iteration done' >> /tmp/skinner.log"
on-error = "notify-send 'Skinner iteration failed'"
on-idle = "./scripts/cleanup.sh"

[hooks.timeout]
pre-iteration = "60s"    # override default 30s
on-error = "5s"          # override default 10s
```

### Timeout Format

Timeout values are duration strings: an integer followed by a unit suffix.

| Unit | Suffix |
|------|--------|
| Seconds | `s` |
| Minutes | `m` |

Examples: `"30s"`, `"2m"`, `"5s"`.

### Omitted Hooks

Hooks not defined in the config are simply not fired. There are no built-in default hooks.

## Error Handling

### pre-* hooks

- **Non-zero exit code**: The loop stops. The iteration is not started. The hook's stderr is logged. Session enters Finished phase.
- **Timeout exceeded**: Treated as a failure (same as non-zero exit).
- **Invalid JSON on stdout**: Ignored — iteration proceeds with the prompt file content. Only well-formed JSON with recognized keys has an effect.

### on-* hooks

- **Non-zero exit code**: Ignored. The loop continues.
- **Timeout exceeded**: Process is killed. No effect on the loop.
- **Any output**: Ignored.

## Interaction with --exit

When `--exit` is active (see [iteration-loop.md](iteration-loop.md)):

- All hooks fire normally during the run.
- When a `pre-iteration` hook signals `{"done": true}`, the TUI exits with code 0 (same as reaching max iterations).
- When a `pre-iteration` hook fails (non-zero exit), the TUI exits with code 1.
- `on-idle` fires before the TUI exits, giving cleanup hooks a chance to run.
- The TUI waits for any in-flight `on-*` hooks to finish (up to their timeout) before exiting.
