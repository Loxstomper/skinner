# Plan Mode

## Overview

Plan mode launches Claude CLI interactively with a configurable seed prompt, giving the user full terminal control to discuss and create an implementation plan. Skinner suspends its TUI for the duration, then resumes when the user exits Claude.

This is complementary to the plan files pane ([plan-files.md](plan-files.md)) — plan mode is for *creating* plans, the plan pane is for *viewing* them.

## Trigger

| Key | Action |
|-----|--------|
| `p` | Enter plan mode (configurable via `[keybindings]`) |

Disabled while a run is in progress. The key is ignored and produces no effect.

## Behavior

1. User presses `p`.
2. Skinner suspends its TUI using `tea.Exec`, handing full terminal control to the child process.
3. The configured command is executed via `sh -c`.
4. The user interacts with Claude CLI — chatting, refining ideas, asking Claude to write an `IMPLEMENTATION_PLAN.md`, etc.
5. The user exits Claude CLI normally.
6. Skinner resumes its TUI in the same state it was in before plan mode.

## Command Execution

- The command is executed via `sh -c "<command>"`, allowing shell features (quoting, env vars, pipes).
- Default command: `claude "study specs/README.md"`
- Configurable via the `[plan]` section in the config file — see [config.md](config.md).

## Session Management

- Each invocation starts a **new** Claude session. There is no session resume or continuation.
- Skinner does not track plan mode sessions. It is entirely fire-and-forget.

## Error Handling

- If the command exits with a non-zero exit code, Skinner resumes and displays a transient status message in the header bar: `plan command failed (exit <code>)`.
- If the command cannot be started (e.g. `claude` not in PATH), the same status message is shown.
- The status message clears on the next keypress.

## Config

See [config.md](config.md) for the `[plan]` section.

```toml
[plan]
command = 'claude "study specs/README.md"'
```
