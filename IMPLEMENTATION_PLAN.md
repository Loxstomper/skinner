# Implementation Plan: System Stats

Add system-wide CPU and memory utilization to the header bar per `specs/system-stats.md`.

## Tasks

1. ~~Create `internal/stats/stats.go`~~ — DONE. Package provides `ParseCPUSample`, `CPUPercent`, `ParseMemPercent`, and file-reading wrappers (`ReadCPUSampleFrom`, `ReadMemPercentFrom`).
2. ~~Create `internal/stats/stats_test.go`~~ — DONE. 13 tests covering: normal parsing, minimal fields, too-few fields, no cpu line, CPU delta calc, first-sample nil, zero-prev, high utilization, mem normal/high/missing/empty/zero-total.
3. Add `CPUPercent *int` and `MemPercent *int` fields to the `Session` struct in `internal/model/model.go`. Nil means no data yet.
4. Add CPU previous-sample fields (`PrevCPUActive int64`, `PrevCPUTotal int64`) to Session for delta calculation.
5. Define `systemStatsTickMsg` message type and `systemStatsTickCmd()` (2-second interval) in `internal/tui/root.go`.
6. Handle `systemStatsTickMsg` in `Update()` — call the stats reader, compute percentages, store in Session, return next tick command.
7. Fire `systemStatsTickCmd()` in `Init()` via `tea.Batch` alongside existing tick commands.
8. Add `CPUPercent *int` and `MemPercent *int` to `HeaderProps` in `internal/tui/header.go`.
9. Populate the new `HeaderProps` fields from Session state in `headerProps()` in `internal/tui/root.go`.
10. Render `⚙ N% ◼ N%` in `RenderHeader()` on the far right, after the iteration indicator, with color thresholds (green <50%, yellow 50-79%, red 80%+). Show `⚙ --% ◼ --%` in `ForegroundDim` when nil.
11. If `/proc` files are unreadable, hide the stats section entirely (graceful non-Linux degradation).
12. Add header rendering tests in `internal/tui/header_test.go` covering: stats present, stats nil (placeholder), and color threshold boundaries.
13. Run `make check` and fix any lint/vet/test issues.
