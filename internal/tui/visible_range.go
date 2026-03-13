package tui

import "github.com/loxstomper/skinner/internal/model"

// visibleWindow describes which items in the timeline overlap the visible
// viewport. Used by the two-phase rendering approach: phase 1 computes this
// cheaply (zero styling), phase 2 renders only these items.
type visibleWindow struct {
	StartItem       int // index of first item overlapping viewport
	StartLineOffset int // lines to skip within the first item
	EndItem         int // index of last item overlapping viewport (inclusive)
	EndLineOffset   int // lines to include from the last item
	AbsLineNumber   int // absolute line number of first visible line (for gutter)
	CursorItemIndex int // index of item containing cursor, or -1 if cursor is off-screen
}

// visibleRangeFromBottom computes the visible window when pinned to the
// bottom of the timeline (auto-follow mode). It walks items backward from the
// last item, accumulating line counts via ItemLineCount, and stops once
// viewportHeight lines are covered — making the backward walk O(visible)
// rather than O(n). This is the fast path for the common auto-follow case
// where the user is watching a live feed and all items are collapsed.
func visibleRangeFromBottom(items []model.TimelineItem, viewportHeight, cursorPos, width int, compactView bool) visibleWindow {
	w := visibleWindow{
		StartItem:       -1,
		CursorItemIndex: -1,
	}

	n := len(items)
	if n == 0 || viewportHeight <= 0 {
		return w
	}

	// Walk backwards from the last item, accumulating line counts.
	linesAccum := 0
	startIdx := 0

	for i := n - 1; i >= 0; i-- {
		lc := ItemLineCount(items[i], compactView, width)
		linesAccum += lc
		if linesAccum >= viewportHeight {
			startIdx = i
			w.StartLineOffset = linesAccum - viewportHeight
			break
		}
	}
	// If loop completed without break, all items fit: startIdx=0, StartLineOffset=0.

	w.StartItem = startIdx
	w.EndItem = n - 1
	w.EndLineOffset = ItemLineCount(items[n-1], compactView, width)

	// Compute AbsLineNumber and starting flatPos in one forward pass up to startIdx.
	linesBefore := 0
	flatPos := 0
	for i := 0; i < startIdx; i++ {
		linesBefore += ItemLineCount(items[i], compactView, width)
		if g, ok := items[i].(*model.ToolCallGroup); ok {
			flatPos++
			if g.Expanded {
				flatPos += len(g.Children)
			}
		} else {
			flatPos++
		}
	}
	w.AbsLineNumber = linesBefore + w.StartLineOffset

	// Walk visible items to find cursor ownership.
	for i := startIdx; i <= w.EndItem; i++ {
		switch it := items[i].(type) {
		case *model.ToolCallGroup:
			if flatPos == cursorPos {
				w.CursorItemIndex = i
			}
			flatPos++
			if it.Expanded {
				for range it.Children {
					if flatPos == cursorPos {
						w.CursorItemIndex = i
					}
					flatPos++
				}
			}
		default:
			if flatPos == cursorPos {
				w.CursorItemIndex = i
			}
			flatPos++
		}
	}

	return w
}

// visibleRange computes which items overlap the viewport defined by
// [scrollOffset, scrollOffset+viewportHeight). It walks items forward from
// index 0, accumulating line counts via ItemLineCount, and stops once past
// the viewport — making it O(firstVisible + visible) rather than O(n).
//
// The cursorPos parameter is the flat cursor position (same coordinate system
// as Timeline.Cursor). CursorItemIndex is set to the items-slice index of the
// item that owns the cursor, or -1 if the cursor's item doesn't overlap the
// viewport.
func visibleRange(items []model.TimelineItem, scrollOffset, viewportHeight, cursorPos, width int, compactView bool) visibleWindow {
	w := visibleWindow{
		StartItem:       -1,
		CursorItemIndex: -1,
		AbsLineNumber:   scrollOffset,
	}

	if len(items) == 0 || viewportHeight <= 0 {
		return w
	}

	viewEnd := scrollOffset + viewportHeight
	currentLine := 0
	flatPos := 0

	for i, item := range items {
		lc := ItemLineCount(item, compactView, width)
		itemEnd := currentLine + lc

		overlaps := itemEnd > scrollOffset && currentLine < viewEnd

		if overlaps {
			if w.StartItem == -1 {
				w.StartItem = i
				w.StartLineOffset = scrollOffset - currentLine
				if w.StartLineOffset < 0 {
					w.StartLineOffset = 0
				}
			}
			w.EndItem = i
			endInItem := viewEnd - currentLine
			if endInItem > lc {
				endInItem = lc
			}
			w.EndLineOffset = endInItem
		}

		// Advance flatPos and check cursor ownership.
		switch it := item.(type) {
		case *model.ToolCallGroup:
			if overlaps && flatPos == cursorPos {
				w.CursorItemIndex = i
			}
			flatPos++
			if it.Expanded {
				for range it.Children {
					if overlaps && flatPos == cursorPos {
						w.CursorItemIndex = i
					}
					flatPos++
				}
			}
		default:
			if overlaps && flatPos == cursorPos {
				w.CursorItemIndex = i
			}
			flatPos++
		}

		currentLine = itemEnd

		// Early exit once we've passed the viewport.
		if currentLine >= viewEnd {
			break
		}
	}

	return w
}
