package tui

import (
	"strings"

	"github.com/loxstomper/skinner/internal/model"
)

// FlatCursorCount returns the total number of navigable positions across all
// items, accounting for expanded groups (header + children).
func FlatCursorCount(items []model.TimelineItem) int {
	count := 0
	for _, item := range items {
		switch it := item.(type) {
		case *model.ToolCallGroup:
			count++ // header
			if it.Expanded {
				count += len(it.Children)
			}
		default:
			count++
		}
	}
	return count
}

// FlatToItem maps a flat cursor index to (item index, child index).
// childIdx == -1 means non-group item or group header.
// childIdx >= 0 means a group child.
func FlatToItem(items []model.TimelineItem, flatIdx int) (itemIdx int, childIdx int) {
	pos := 0
	for i, item := range items {
		if group, ok := item.(*model.ToolCallGroup); ok {
			if pos == flatIdx {
				return i, -1 // group header
			}
			pos++
			if group.Expanded {
				for ci := range group.Children {
					if pos == flatIdx {
						return i, ci
					}
					pos++
				}
			}
		} else {
			if pos == flatIdx {
				return i, -1
			}
			pos++
		}
	}
	return 0, -1
}

// ItemToFlat maps an item index to its flat cursor position (the header
// position for groups).
func ItemToFlat(items []model.TimelineItem, itemIdx int) int {
	pos := 0
	for i, item := range items {
		if i == itemIdx {
			return pos
		}
		if group, ok := item.(*model.ToolCallGroup); ok {
			pos++ // header
			if group.Expanded {
				pos += len(group.Children)
			}
		} else {
			pos++
		}
	}
	return pos
}

// ItemLineCount returns the number of rendered lines for a single timeline item.
func ItemLineCount(item model.TimelineItem, compactView bool) int {
	switch it := item.(type) {
	case *model.TextBlock:
		lines := strings.Count(it.Text, "\n") + 1
		maxLines := 3
		if compactView {
			maxLines = 1
		}
		if !it.Expanded && lines > maxLines {
			return maxLines
		}
		return lines
	case *model.ToolCall:
		return 1
	case *model.ToolCallGroup:
		if it.Expanded {
			return 1 + len(it.Children) // header + children
		}
		return 1 // collapsed: header only
	}
	return 1
}

// TotalLines returns the total number of rendered lines across all items.
func TotalLines(items []model.TimelineItem, compactView bool) int {
	total := 0
	for _, item := range items {
		total += ItemLineCount(item, compactView)
	}
	return total
}

// FlatCursorLineRange returns the start line and line count for the given flat
// cursor position.
func FlatCursorLineRange(items []model.TimelineItem, flatIdx int, compactView bool) (lineStart int, lineCount int) {
	line := 0
	pos := 0
	for _, item := range items {
		switch it := item.(type) {
		case *model.TextBlock:
			lc := ItemLineCount(it, compactView)
			if pos == flatIdx {
				return line, lc
			}
			line += lc
			pos++
		case *model.ToolCall:
			if pos == flatIdx {
				return line, 1
			}
			line++
			pos++
		case *model.ToolCallGroup:
			// Header
			if pos == flatIdx {
				return line, 1
			}
			line++
			pos++
			if it.Expanded {
				for range it.Children {
					if pos == flatIdx {
						return line, 1
					}
					line++
					pos++
				}
			}
		}
	}
	return line, 1
}
