# Viewport-Only Rendering — Implementation Plan

Spec: [specs/viewport-rendering.md](specs/viewport-rendering.md)

## Tasks

1. ~~**Add `expandedContentLineCount` function to `expand.go`**~~ ✅ DONE
   - Implemented `expandedContentLineCount(tc, width)` with zero-allocation helpers: `bashContentLineCount`, `editContentLineCount`, `writeContentLineCount`, `resultContentLineCount`
   - Edit diffs are width-aware: unified (old+new) when width < 120, side-by-side (max) when width >= 120
   - 18 unit tests verify counts match `len(expandedContentLines(...))` for all tool types, plus Edit layout-specific tests

2. ~~**Add width parameter to `ItemLineCount` and make line counting zero-allocation**~~ ✅ DONE
   - Updated `toolCallLineCount(tc, width)` and `toolCallLineCountCapped(tc, width, paneHeight)` to use `expandedContentLineCount` — zero allocation for line counting
   - Added `width int` parameter to `ItemLineCount`, `TotalLines`, `FlatCursorLineRange`, `LineToFlatCursor`
   - Updated all callers in `timeline.go` to pass `props.Width`
   - Updated all tests to pass width parameter (80 for non-Edit tests, matching unified layout)
   - This also fixes a pre-existing bug where Edit tool calls at width >= 120 had mismatched line counts between cursor functions (which always counted unified) and rendering (which used side-by-side)

3. ~~**Implement `visibleRange` function in `visible_range.go`**~~ ✅ DONE
   - Implemented `visibleRange(items, scrollOffset, viewportHeight, cursorPos, width, compactView)` returning `visibleWindow` struct
   - Walks items forward accumulating line counts via `ItemLineCount`, stops once past viewport (early exit)
   - Returns: StartItem/EndItem (inclusive), StartLineOffset/EndLineOffset for partial items, AbsLineNumber, CursorItemIndex (-1 if off-screen)
   - Groups: CursorItemIndex maps to group's item index for both header and children
   - 17 unit tests: empty, zero viewport, all visible, scroll middle/bottom, cursor off-screen (above/below), expanded items, partial items at top, expanded/collapsed groups, cursor on group header/child, text blocks, compact view, width-dependent Edit layout, consistency with TotalLines, early exit at n=1000

4. ~~**Implement `visibleRangeFromBottom` function in `visible_range.go`**~~ ✅ DONE
   - Walks items backward from last item, accumulating line counts via `ItemLineCount`, stops once `viewportHeight` lines covered — O(visible) backward walk
   - Added `width` and `cursorPos` parameters (spec signature was incomplete — these are needed for `ItemLineCount` and `CursorItemIndex`)
   - Computes `AbsLineNumber` and flat cursor position via single forward pass to `startIdx`
   - 11 tests: 9-subtest table test verifying exact match with `visibleRange` at scroll-bottom (all collapsed, all fit, expanded items, expanded groups, large set, single item, expanded larger than viewport, compact view, width-dependent Edit layout), plus empty and zero-viewport edge cases

5. ~~**Refactor `View()` to use two-phase rendering**~~ ✅ DONE
   - Phase 1: `visibleRange(items, tl.Scroll, Height, Cursor, Width, CompactView)` computes visible window
   - Phase 2: iterate only items in `[StartItem..EndItem]`, render styled lines
   - `StartLineOffset` trimming for partial items at viewport top; `Height` cap for bottom
   - Added `renderVisibleLines()` — gutter + highlight padding + count buffer, no scroll slicing
   - Sub-scroll mode falls back to full render + `renderWithLines()` (to be optimized in task 6)
   - Groups fully handled: header + expanded children rendered only when in visible range
   - Thinking indicator appended only when last item is in visible range
   - Used `ItemToFlat()` to compute `flatPos` at `StartItem` for cursor highlighting
   - Benchmark results at n=500: ~360μs (was 4.7ms), 2,201 allocs (was 38,537), ~200KB (was 1.4MB)
   - Constant time across n=50/200/500 confirming O(visible) behavior

6. **Handle sub-scroll in viewport rendering**
   - Currently sub-scroll falls back to full-render path (all items rendered, sliced by `renderWithLines`)
   - Need to integrate sub-scroll with two-phase rendering: `visibleRange` with capped line counts
   - Verify existing sub-scroll enter/exit/navigation still works correctly (currently passing all tests)

7. ~~**Handle groups in viewport rendering**~~ ✅ DONE (completed as part of task 5)
   - Groups are rendered in phase 2 with the same logic as before, but only when in the visible range

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
