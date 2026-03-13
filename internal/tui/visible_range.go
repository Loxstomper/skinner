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

// itemLineCountForSubScroll returns the line count for an item, using capped
// line counts when the item (or a group child) is the sub-scrolled item.
// flatPos is the starting flat cursor position for this item.
// subScrollIdx is the flat position of the sub-scrolled item (-1 if not active).
// paneHeight is the viewport height used for sub-scroll capping calculations.
func itemLineCountForSubScroll(item model.TimelineItem, compactView bool, width, flatPos, subScrollIdx, paneHeight int) int {
	if subScrollIdx < 0 {
		return ItemLineCount(item, compactView, width)
	}

	switch it := item.(type) {
	case *model.ToolCall:
		if flatPos == subScrollIdx {
			return toolCallLineCountCapped(it, width, paneHeight)
		}
	case *model.ToolCallGroup:
		if it.Expanded {
			childStart := flatPos + 1
			childEnd := childStart + len(it.Children)
			if subScrollIdx >= childStart && subScrollIdx < childEnd {
				// Sub-scrolled child is in this group — compute with capped child.
				lc := 1 // header
				cf := childStart
				for _, c := range it.Children {
					if cf == subScrollIdx {
						lc += toolCallLineCountCapped(c, width, paneHeight)
					} else {
						lc += toolCallLineCount(c, width)
					}
					cf++
				}
				return lc
			}
		}
	}

	return ItemLineCount(item, compactView, width)
}

// flatAdvance returns the number of flat cursor positions consumed by an item.
func flatAdvance(item model.TimelineItem) int {
	if g, ok := item.(*model.ToolCallGroup); ok {
		n := 1 // header
		if g.Expanded {
			n += len(g.Children)
		}
		return n
	}
	return 1
}

// visibleRangeFromBottom computes the visible window when pinned to the
// bottom of the timeline (auto-follow mode). It walks items backward from the
// last item, accumulating line counts via ItemLineCount, and stops once
// viewportHeight lines are covered — making the backward walk O(visible)
// rather than O(n). This is the fast path for the common auto-follow case
// where the user is watching a live feed and all items are collapsed.
//
// subScrollIdx is the flat cursor position of the sub-scrolled item (-1 if not active).
func visibleRangeFromBottom(items []model.TimelineItem, viewportHeight, cursorPos, width int, compactView bool, subScrollIdx int) visibleWindow {
	w := visibleWindow{
		StartItem:       -1,
		CursorItemIndex: -1,
	}

	n := len(items)
	if n == 0 || viewportHeight <= 0 {
		return w
	}

	// Walk backwards from the last item, accumulating line counts.
	// We need flatPos for sub-scroll capping, so precompute ending flatPos
	// and walk backward.
	linesAccum := 0
	startIdx := 0

	// Compute per-item flatPos start positions for sub-scroll lookup.
	flatStarts := make([]int, n)
	fp := 0
	for i := 0; i < n; i++ {
		flatStarts[i] = fp
		fp += flatAdvance(items[i])
	}

	for i := n - 1; i >= 0; i-- {
		lc := itemLineCountForSubScroll(items[i], compactView, width, flatStarts[i], subScrollIdx, viewportHeight)
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
	w.EndLineOffset = itemLineCountForSubScroll(items[n-1], compactView, width, flatStarts[n-1], subScrollIdx, viewportHeight)

	// Compute AbsLineNumber in one forward pass up to startIdx.
	linesBefore := 0
	for i := 0; i < startIdx; i++ {
		linesBefore += itemLineCountForSubScroll(items[i], compactView, width, flatStarts[i], subScrollIdx, viewportHeight)
	}
	flatPos := flatStarts[startIdx]
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
//
// subScrollIdx is the flat cursor position of the sub-scrolled item (-1 if not active).
// When active, the sub-scrolled item's line count is capped per sub-scroll rules.
func visibleRange(items []model.TimelineItem, scrollOffset, viewportHeight, cursorPos, width int, compactView bool, subScrollIdx int) visibleWindow {
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
		lc := itemLineCountForSubScroll(item, compactView, width, flatPos, subScrollIdx, viewportHeight)
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
