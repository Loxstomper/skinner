# Implementation Plan

All spec features are implemented. This file tracks remaining gaps and learnings.

## Completed

### --exit flag quit confirmation bypass (2026-03-13)
- Fixed: `q` and single `ctrl+c` now quit immediately when `--exit` is active, bypassing the quit modal
- Files changed: `internal/tui/root.go` (3 code paths), `internal/tui/modal_test.go` (2 new tests)
- Spec refs: quit-confirmation.md §--exit Flag Bypass, iteration-loop.md §--exit behavior

## Deferred (per spec)

### Rate limit window utilization
- token-usage.md marks this as "Status: Placeholder"
- Header shows `5h: --` and `wk: --` placeholders
- TODO in model.go line 204: implement rate limit data fetching

## Learnings

- Integration tests use `executeBatchCmd` which synchronously executes `tea.Cmd`. Adding a separate `tea.Tick` for stats would block tests. Solution: piggyback on existing 1-second tick with a counter.
- `executeBatchCmd` now uses a 50ms timeout to skip blocking commands (ticks, channel reads), reducing test suite from ~92s to ~5s.
