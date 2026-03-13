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
