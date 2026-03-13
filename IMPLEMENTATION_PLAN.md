# Viewport-Only Rendering — Implementation Plan

Spec: [specs/viewport-rendering.md](specs/viewport-rendering.md)

## Tasks

1. ~~**Add `expandedContentLineCount` function to `expand.go`**~~ ✅ DONE
   - Implemented `expandedContentLineCount(tc, width)` with zero-allocation helpers: `bashContentLineCount`, `editContentLineCount`, `writeContentLineCount`, `resultContentLineCount`
   - Edit diffs are width-aware: unified (old+new) when width < 120, side-by-side (max) when width >= 120
   - 18 unit tests verify counts match `len(expandedContentLines(...))` for all tool types, plus Edit layout-specific tests
   - Note: existing `toolCallLineCount` is left unchanged — it still uses `expandedContentLines` (allocating). Task 2 will address using the new function.

2. **Add `itemLineCount` helper to `expand.go` (or extend existing `ItemLineCount` in `cursor.go`)**
   - Returns line count for any `TimelineItem` using `expandedContentLineCount` for expanded tool calls
   - This is the building block for both `visibleRange` and `visibleRangeFromBottom`
   - Should match the existing `ItemLineCount` in `cursor.go` — consider reusing or replacing it

3. **Implement `visibleRange` function in `timeline.go`**
   - Walks items forward from index 0, accumulating line counts via `itemLineCount`
   - Stops once past `scrollOffset + viewportHeight`
   - Returns `visibleWindow` struct with: start/end item indices, line offsets within partial items, absolute line number of first visible line, cursor item index
   - Unit test: verify correct window at various scroll positions (top, middle, bottom, partial items)

4. **Implement `visibleRangeFromBottom` function in `timeline.go`**
   - Walks items backward from the last item, accumulating line counts
   - Stops once `viewportHeight` lines are accounted for
   - Returns the same `visibleWindow` struct
   - Unit test: verify matches `visibleRange` result when scroll is at bottom

5. **Refactor `View()` to use two-phase rendering**
   - Phase 1: call `visibleRange` (or `visibleRangeFromBottom` when auto-following) to get the visible window
   - Phase 2: iterate only items in `[StartItem..EndItem]`, render styled lines using existing `renderToolCallLine`, `renderTextBlockLines`, `expandedContentLines` etc.
   - Handle `StartLineOffset`/`EndLineOffset` for items partially above/below the viewport
   - Remove the old "render all, slice later" code path from `View()` and `renderWithLines()`
   - Line numbers: use `AbsLineNumber` from the visible window to compute gutter values
   - Cursor highlighting: use `CursorItemIndex` from the visible window

6. **Handle sub-scroll in viewport rendering**
   - When in sub-scroll mode, the visible window may be entirely within one expanded item's content
   - Phase 2 calls `expandedContentLines` for that item and slices to the sub-scroll offset
   - Verify existing sub-scroll enter/exit/navigation still works correctly

7. **Handle groups in viewport rendering**
   - Expanded groups: phase 1 counts 1 (header) + visible children line counts
   - Phase 2 renders group header + only children that fall in the visible range
   - Collapsed groups: 1 line, same as before

8. **Add `makeCollapsedTestItems` helper to `benchmark_test.go`**
   - All items collapsed, no expanded content — simulates the feed-watching case
   - Used by the new scaling benchmark

9. **Add `BenchmarkTimelineViewScaling` benchmark**
   - Runs View() at n=50, 200, 500, 1000 with all collapsed items and auto-follow active
   - Verifies roughly constant time across n values (O(visible) not O(n))

10. **Run full benchmark suite, verify targets**
    - `BenchmarkTimelineView` at n=500: < 0.5ms, < 4K allocs, < 150KB
    - `BenchmarkTimelineViewScaling`: constant across n values
    - `BenchmarkNewItemArrival` at n=500: significant improvement over current 4.7ms
    - Existing cursor benchmarks unchanged (no regressions)

11. **Manual smoke test**
    - Run skinner against a real Claude session with 100+ tool calls
    - Verify: smooth scrolling, correct line numbers, cursor highlighting, expand/collapse, sub-scroll, group rendering, auto-follow, compact view toggle
