# Benchmarks

## Overview

Performance benchmarks for the TUI rendering and cursor calculation hot paths. These benchmarks live in `internal/tui/benchmark_test.go` and are designed to track performance over time as the codebase evolves. They use Go's `testing.B` framework with sub-benchmarks at multiple scale points.

## Scale Points

Each benchmark runs at three item counts to reveal scaling behavior:

| Label | Items | Represents |
|-------|-------|------------|
| `n=50` | 50 | Typical short session |
| `n=200` | 200 | Moderate session with many tool calls |
| `n=500` | 500 | Heavy session, stress test |

## Test Data Helper

A shared helper function `makeTestItems(n int)` constructs a realistic mix of timeline items:

- **60% standalone `ToolCall`** — mix of Bash, Read, Edit, Grep, Glob. All completed with result content. ~10% of these are expanded.
- **25% `ToolCallGroup`** — groups of 3–8 children of the same tool type. ~50% expanded (simulating in-progress groups). Each child has result content.
- **15% `TextBlock`** — 1–5 lines of text. All collapsed.

Edit tool calls include `RawInput` with `old_string`/`new_string` fields (10–30 lines each) to exercise the diff rendering path.

Bash tool calls include `ResultContent` with 5–20 lines of output.

The helper is deterministic (seeded RNG) so results are reproducible across runs.

## Benchmarks

### `BenchmarkTimelineView`

Measures the full `Timeline.View()` render path. This is the primary benchmark — it exercises item iteration, expanded content rendering, diff styling, line number gutters, and string joining.

```go
func BenchmarkTimelineView(b *testing.B) {
    for _, n := range []int{50, 200, 500} {
        b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
            items := makeTestItems(n)
            tl := NewTimeline()
            props := TimelineProps{
                Items:       items,
                Width:       140,
                Height:      50,
                Focused:     true,
                CompactView: false,
                LineNumbers: true,
                Theme:       theme.DefaultTheme(),
            }
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                tl.View(props)
            }
        })
    }
}
```

### `BenchmarkFlatCursorLineRange`

Measures the O(n) cursor-to-line mapping used on every keystroke.

```go
func BenchmarkFlatCursorLineRange(b *testing.B) {
    for _, n := range []int{50, 200, 500} {
        b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
            items := makeTestItems(n)
            maxPos := FlatCursorCount(items) - 1
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                FlatCursorLineRange(items, maxPos, false)
            }
        })
    }
}
```

### `BenchmarkTotalLines`

Measures the O(n) total line count used for scroll clamping and auto-follow.

```go
func BenchmarkTotalLines(b *testing.B) {
    for _, n := range []int{50, 200, 500} {
        b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
            items := makeTestItems(n)
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                TotalLines(items, false)
            }
        })
    }
}
```

### `BenchmarkFlatCursorCount`

Measures the O(n) navigable position count used on every cursor move.

```go
func BenchmarkFlatCursorCount(b *testing.B) {
    for _, n := range []int{50, 200, 500} {
        b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
            items := makeTestItems(n)
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                FlatCursorCount(items)
            }
        })
    }
}
```

### `BenchmarkExpandedContentLines`

Measures the cost of computing expanded content for a single tool call. Tests both Bash (string split) and Edit (diff rendering) paths.

```go
func BenchmarkExpandedContentLines(b *testing.B) {
    b.Run("Bash", func(b *testing.B) { ... })
    b.Run("Edit", func(b *testing.B) { ... })
}
```

### `BenchmarkNewItemArrival`

Measures the combined cost of a new item arriving: `OnNewItems()` (which calls `FlatCursorCount` and `scrollToBottom` which calls `TotalLines`) followed by `View()`. This simulates the hot path during rapid tool call arrival.

```go
func BenchmarkNewItemArrival(b *testing.B) {
    for _, n := range []int{50, 200, 500} {
        b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
            items := makeTestItems(n)
            tl := NewTimeline()
            props := TimelineProps{
                Items:       items,
                Width:       140,
                Height:      50,
                Focused:     true,
                CompactView: false,
                LineNumbers: true,
                Theme:       theme.DefaultTheme(),
            }
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                tl.OnNewItems(props)
                tl.View(props)
            }
        })
    }
}
```

### `BenchmarkTimelineViewScaling`

Verifies that viewport-only rendering (see [viewport-rendering.md](viewport-rendering.md)) scales with viewport height, not total item count. Runs `View()` at n=50, 200, 500, 1000 with all items collapsed and auto-follow active (pinned to bottom). Times should be roughly constant across all n values.

```go
func BenchmarkTimelineViewScaling(b *testing.B) {
    for _, n := range []int{50, 200, 500, 1000} {
        b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
            items := makeCollapsedTestItems(n)
            tl := NewTimeline()
            props := TimelineProps{
                Items:       items,
                Width:       140,
                Height:      50,
                Focused:     true,
                CompactView: false,
                LineNumbers: true,
                Theme:       theme.DefaultTheme(),
            }
            tl.scrollToBottom(props)
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                tl.View(props)
            }
        })
    }
}
```

## Running

```bash
# Run all benchmarks
go test -bench=. -benchmem ./internal/tui/

# Run a specific benchmark
go test -bench=BenchmarkTimelineView -benchmem ./internal/tui/

# Compare before/after a change (using benchstat)
go test -bench=. -benchmem -count=10 ./internal/tui/ > old.txt
# ... make changes ...
go test -bench=. -benchmem -count=10 ./internal/tui/ > new.txt
benchstat old.txt new.txt
```
