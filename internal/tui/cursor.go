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
// The width parameter determines Edit diff layout (side-by-side when >= 120).
func ItemLineCount(item model.TimelineItem, compactView bool, width int) int {
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
		return toolCallLineCount(it, width)
	case *model.ToolCallGroup:
		if it.Expanded {
			lines := 1 // header
			for _, child := range it.Children {
				lines += toolCallLineCount(child, width)
			}
			return lines
		}
		return 1 // collapsed: header only
	}
	return 1
}

// TotalLines returns the total number of rendered lines across all items.
// The width parameter determines Edit diff layout (side-by-side when >= 120).
func TotalLines(items []model.TimelineItem, compactView bool, width int) int {
	total := 0
	for _, item := range items {
		total += ItemLineCount(item, compactView, width)
	}
	return total
}

// LineToFlatCursor maps a rendered line number to the flat cursor position
// that owns that line (inverse of FlatCursorLineRange). If the line is beyond
// all items, it returns the last valid flat position.
// The width parameter determines Edit diff layout (side-by-side when >= 120).
func LineToFlatCursor(items []model.TimelineItem, line int, compactView bool, width int) int {
	if len(items) == 0 {
		return 0
	}
	currentLine := 0
	flatPos := 0
	for _, item := range items {
		switch it := item.(type) {
		case *model.TextBlock:
			lc := ItemLineCount(it, compactView, width)
			if line < currentLine+lc {
				return flatPos
			}
			currentLine += lc
			flatPos++
		case *model.ToolCall:
			lc := toolCallLineCount(it, width)
			if line < currentLine+lc {
				return flatPos
			}
			currentLine += lc
			flatPos++
		case *model.ToolCallGroup:
			// Header line
			if line < currentLine+1 {
				return flatPos
			}
			currentLine++
			flatPos++
			if it.Expanded {
				for _, child := range it.Children {
					clc := toolCallLineCount(child, width)
					if line < currentLine+clc {
						return flatPos
					}
					currentLine += clc
					flatPos++
				}
			}
		}
	}
	// Beyond all items: return last valid position
	total := FlatCursorCount(items)
	if total > 0 {
		return total - 1
	}
	return 0
}

// FlatCursorLineRange returns the start line and line count for the given flat
// cursor position.
// The width parameter determines Edit diff layout (side-by-side when >= 120).
func FlatCursorLineRange(items []model.TimelineItem, flatIdx int, compactView bool, width int) (lineStart int, lineCount int) {
	line := 0
	pos := 0
	for _, item := range items {
		switch it := item.(type) {
		case *model.TextBlock:
			lc := ItemLineCount(it, compactView, width)
			if pos == flatIdx {
				return line, lc
			}
			line += lc
			pos++
		case *model.ToolCall:
			lc := toolCallLineCount(it, width)
			if pos == flatIdx {
				return line, lc
			}
			line += lc
			pos++
		case *model.ToolCallGroup:
			// Header
			if pos == flatIdx {
				return line, 1
			}
			line++
			pos++
			if it.Expanded {
				for _, child := range it.Children {
					clc := toolCallLineCount(child, width)
					if pos == flatIdx {
						return line, clc
					}
					line += clc
					pos++
				}
			}
		}
	}
	return line, 1
}
