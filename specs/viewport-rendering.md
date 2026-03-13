# Viewport-Only Rendering

## Overview

`Timeline.View()` only renders items that fall within the visible viewport, skipping all off-screen items. This eliminates O(n) styling and allocation costs, replacing them with O(visible) work per frame regardless of total item count.

## Problem

The current `View()` implementation builds a styled `renderedLine` for every item in the iteration, then slices to the visible window. At n=500 items this produces ~39K allocations and ~1.4MB of garbage per frame. During rapid tool call arrival (10+ events/sec), this creates ~14MB/sec of short-lived allocations, causing GC pressure that manifests as UI freezes.

Benchmark data (n=500):

| Metric | Current |
|--------|---------|
| Time per View() | 4.7ms |
| Allocations per View() | 38,537 |
| Bytes per View() | 1.4MB |

The cursor helper functions (`FlatCursorCount`, `TotalLines`, `FlatCursorLineRange`) are cheap (microseconds, zero allocs for collapsed items) and are not a bottleneck.

## Design

### Two-Phase Render

Replace the current single-pass "render everything, slice visible" with two phases:

**Phase 1 — Line counting walk (cheap, zero allocs)**

Walk the items list, accumulating a running line count per item without any styling or string allocation:

| Item type | Line count |
|-----------|-----------|
| Collapsed tool call | 1 |
| Expanded tool call | 1 + len(content lines) |
| Collapsed text block | 1 |
| Expanded text block | number of visible text lines |
| Collapsed group | 1 |
| Expanded group | 1 + number of visible children |

Stop walking once we've passed `scrollOffset + viewportHeight`. Record the index of the first item that overlaps the viewport and the line offset within that item.

**Phase 2 — Render visible items only (expensive, but bounded)**

Starting from the first visible item, render styled lines using the existing rendering functions (`renderToolCallLine`, `renderTextBlockLines`, `expandedContentLines`, etc.) until we've filled the viewport height. This produces at most `viewportHeight` styled lines.

### `visibleRange` Function

A new function computes the visible item range:

```go
type visibleWindow struct {
    StartItem       int // index of first item overlapping viewport
    StartLineOffset int // lines to skip within the first item
    EndItem         int // index of last item overlapping viewport (inclusive)
    EndLineOffset   int // lines to include from the last item
    AbsLineNumber   int // absolute line number of first visible line (for gutter)
    CursorItemIndex int // which visible item the cursor is on (-1 if off-screen)
}

func visibleRange(items []model.TimelineItem, scrollOffset, viewportHeight, cursorPos int, compactView bool) visibleWindow
```

This function does the same item walk as `TotalLines` but stops early. For the common case (all collapsed), it's a simple integer comparison per item.

### Auto-Follow Optimization

When auto-follow is active (pinned to bottom), walk the item list **backwards** from the last item. Count lines until `viewportHeight` is reached. This is O(visible) in the common case where the user is watching a live feed and all items are collapsed.

```go
func visibleRangeFromBottom(items []model.TimelineItem, viewportHeight int, compactView bool) visibleWindow
```

### Content Line Count Without Styling

Expanded items need their content line count during the counting walk (phase 1). Currently `expandedContentLines()` both splits content and styles it. Split this into:

```go
// Cheap — returns line count only, no styling or allocation
func expandedContentLineCount(tc *model.ToolCall, width int) int

// Expensive — returns styled lines, called only for visible items
func expandedContentLines(tc *model.ToolCall, width int, theme theme.Theme) []string
```

For most tool types, `expandedContentLineCount` is `strings.Count(content, "\n") + 1` — a single scan with no allocation. For Edit diffs, it's `max(len(oldLines), len(newLines))` which requires a count of newlines in both strings.

## Interaction With Existing Features

### Line Numbers

The line counting walk tracks the absolute line number as it goes. When phase 2 begins rendering, it knows the absolute line number of the first visible line. Relative line numbers are computed as `abs(visibleLineNumber - cursorLineNumber)`, same as today but only for rendered lines.

### Sub-Scroll

When in sub-scroll mode, a single expanded item may be taller than the viewport. The `StartLineOffset` and `EndLineOffset` fields in `visibleWindow` handle partial rendering of the expanded content. Phase 2 slices into the content lines array for that single item, rendering only the visible sub-range.

### Cursor Highlighting

The `CursorItemIndex` field in `visibleWindow` indicates which visible item (if any) is under the cursor. Phase 2 applies the highlight background only to that item's lines. If the cursor is off-screen, no highlighting is needed.

### Groups

Expanded groups contain child items. During the line counting walk, an expanded group's line count is 1 (header) + number of visible children. Each visible child contributes 1 line if collapsed, or 1 + content lines if expanded. Phase 2 renders the group header and iterates only visible children.

### Compact View

Compact view changes the rendering of each item (shorter tool call lines, 1-line text blocks) but doesn't change the line counting logic — each collapsed item is still 1 line. The `compactView` flag is passed through to both phases.

## Performance Target

At n=500 with a 50-line viewport:

| Metric | Current | Target |
|--------|---------|--------|
| Time per View() | 4.7ms | < 0.5ms |
| Allocations per View() | 38,537 | < 4,000 |
| Bytes per View() | 1.4MB | < 150KB |

The counting walk (phase 1) should be < 20us at n=500 (comparable to existing `TotalLines`). Phase 2 should scale with viewport height, not item count.

## Benchmark Updates

Add a new benchmark to verify viewport-only rendering scales with viewport size, not item count:

### `BenchmarkTimelineViewScaling`

Runs `View()` at n=50, 200, 500, 1000 with a fixed viewport height of 50. All items collapsed. The benchmark should show roughly constant time across all n values, confirming O(visible) behavior.

```go
func BenchmarkTimelineViewScaling(b *testing.B) {
    for _, n := range []int{50, 200, 500, 1000} {
        b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
            items := makeCollapsedTestItems(n) // all collapsed, no expanded content
            tl := NewTimeline()
            props := TimelineProps{
                Items:  items,
                Width:  140,
                Height: 50,
                // ... standard props
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

### `BenchmarkNewItemArrival` (updated)

The existing benchmark should show significant improvement since `View()` is now O(visible). At n=500 with auto-follow, the backward walk makes this particularly fast.

## Implementation Notes

- The existing `TotalLines`, `FlatCursorCount`, and `FlatCursorLineRange` functions remain unchanged — they're already fast enough and are needed for scroll bar calculations and cursor bounds.
- `scrollOffset` management is unchanged. The viewport rendering is a pure optimization of `View()` — the scroll model stays the same.
- The `renderedLine` struct and per-line rendering functions are reused as-is. Only the loop in `View()` changes from "all items" to "visible items".
