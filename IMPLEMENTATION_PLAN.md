# Implementation Plan: System Stats

All tasks complete. The system stats feature is fully implemented.

## Summary

- `internal/stats/stats.go` — CPU/memory parsing from `/proc/stat` and `/proc/meminfo`
- `internal/stats/stats_test.go` — 13 tests for parsing edge cases
- `internal/model/model.go` — `CPUPercent`, `MemPercent`, `PrevCPUActive`, `PrevCPUTotal` on Session
- `internal/tui/root.go` — Stats read piggybacked on existing 1-second tick (every 2 ticks = 2s interval)
- `internal/tui/header.go` — `⚙ N% ◼ N%` rendered far-right with color thresholds; hidden when `/proc` unavailable
- `internal/tui/header_test.go` — 5 tests: present, nil placeholder, both-nil hidden, color thresholds, idle state

## Learnings

- Integration tests use `executeBatchCmd` which synchronously executes `tea.Cmd`. Adding a separate `tea.Tick` for stats would block tests. Solution: piggyback on existing 1-second tick with a counter.
- `executeBatchCmd` now uses a 50ms timeout to skip blocking commands (ticks, channel reads), reducing test suite from ~92s to ~5s.
