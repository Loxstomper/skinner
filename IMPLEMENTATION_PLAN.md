# Implementation Plan

All benchmark tasks (specs/benchmarks.md) completed. No remaining tasks.

## Completed

- Benchmarks (tasks 1-8): `internal/tui/benchmark_test.go` — makeTestItems helper, BenchmarkTimelineView, BenchmarkFlatCursorLineRange, BenchmarkTotalLines, BenchmarkFlatCursorCount, BenchmarkExpandedContentLines, BenchmarkNewItemArrival. All pass with `make check` clean.

## Learnings

- Module path is `github.com/loxstomper/skinner` (not `github.com/lox/skinner`)
- No `theme.DefaultTheme()` exists; use `theme.LookupTheme("solarized-dark")` in tests
- `expandedContentLines` is unexported — accessible from `_test.go` in the same package
