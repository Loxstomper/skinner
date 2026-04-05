# Skinner

Go TUI that wraps Claude CLI and displays tool call activity in real time.

## Build & Test

- `make build` — compile the `skinner` binary
- `make test` — run all tests
- `make check` — run vet + lint + tests
- `make fmt` — format code
- `make lint` — run golangci-lint
- `make run` — build and run

## Task Management

This project uses **bd** (beads) for issue tracking.

### Quick Reference

```bash
bd show <id>          # View issue details
```

### Issue Tracking with bd (beads)

**IMPORTANT**: This project uses **bd (beads)** for ALL issue tracking. Do NOT use markdown TODOs, task lists, or other tracking methods.

#### Issue Types

- `bug` - Something broken
- `feature` - New functionality
- `task` - Work item (tests, docs, refactoring)
- `epic` - Large feature with subtasks
- `chore` - Maintenance (dependencies, tooling)

#### Priorities

- `0` - Critical (security, data loss, broken builds)
- `1` - High (major features, important bugs)
- `2` - Medium (default, nice-to-have)
- `3` - Low (polish, optimization)
- `4` - Backlog (future ideas)

#### Important Rules

- ❌ Do NOT create markdown TODO lists
- ❌ Do NOT use external issue trackers
- ❌ Do NOT duplicate tracking systems

