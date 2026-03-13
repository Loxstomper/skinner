package tui

import (
	"testing"

	"github.com/loxstomper/skinner/internal/model"
)

// makeCollapsedToolCalls creates n collapsed tool calls, each contributing 1 line.
func makeCollapsedToolCalls(n int) []model.TimelineItem {
	items := make([]model.TimelineItem, n)
	for i := range n {
		items[i] = &model.ToolCall{Name: "Read", Summary: "file.go", Status: model.ToolCallDone}
		_ = i
	}
	return items
}

func TestVisibleRangeEmpty(t *testing.T) {
	w := visibleRange(nil, 0, 50, 0, 80, false)
	if w.StartItem != -1 {
		t.Errorf("StartItem = %d, want -1", w.StartItem)
	}
	if w.CursorItemIndex != -1 {
		t.Errorf("CursorItemIndex = %d, want -1", w.CursorItemIndex)
	}
}

func TestVisibleRangeZeroViewport(t *testing.T) {
	items := makeCollapsedToolCalls(5)
	w := visibleRange(items, 0, 0, 0, 80, false)
	if w.StartItem != -1 {
		t.Errorf("StartItem = %d, want -1", w.StartItem)
	}
}

func TestVisibleRangeAllVisible(t *testing.T) {
	// 5 collapsed items = 5 lines total. Viewport = 10 lines.
	items := makeCollapsedToolCalls(5)
	w := visibleRange(items, 0, 10, 0, 80, false)

	if w.StartItem != 0 {
		t.Errorf("StartItem = %d, want 0", w.StartItem)
	}
	if w.StartLineOffset != 0 {
		t.Errorf("StartLineOffset = %d, want 0", w.StartLineOffset)
	}
	if w.EndItem != 4 {
		t.Errorf("EndItem = %d, want 4", w.EndItem)
	}
	if w.EndLineOffset != 1 {
		t.Errorf("EndLineOffset = %d, want 1", w.EndLineOffset)
	}
	if w.AbsLineNumber != 0 {
		t.Errorf("AbsLineNumber = %d, want 0", w.AbsLineNumber)
	}
	if w.CursorItemIndex != 0 {
		t.Errorf("CursorItemIndex = %d, want 0", w.CursorItemIndex)
	}
}

func TestVisibleRangeScrollMiddle(t *testing.T) {
	// 10 collapsed items, viewport=3, scroll=4 → visible items 4,5,6
	items := makeCollapsedToolCalls(10)
	w := visibleRange(items, 4, 3, 5, 80, false)

	if w.StartItem != 4 {
		t.Errorf("StartItem = %d, want 4", w.StartItem)
	}
	if w.StartLineOffset != 0 {
		t.Errorf("StartLineOffset = %d, want 0", w.StartLineOffset)
	}
	if w.EndItem != 6 {
		t.Errorf("EndItem = %d, want 6", w.EndItem)
	}
	if w.EndLineOffset != 1 {
		t.Errorf("EndLineOffset = %d, want 1", w.EndLineOffset)
	}
	if w.AbsLineNumber != 4 {
		t.Errorf("AbsLineNumber = %d, want 4", w.AbsLineNumber)
	}
	// Cursor at flat pos 5 = item 5, which is in [4,6]
	if w.CursorItemIndex != 5 {
		t.Errorf("CursorItemIndex = %d, want 5", w.CursorItemIndex)
	}
}

func TestVisibleRangeScrollBottom(t *testing.T) {
	// 10 collapsed items, viewport=3, scroll=7 → visible items 7,8,9
	items := makeCollapsedToolCalls(10)
	w := visibleRange(items, 7, 3, 9, 80, false)

	if w.StartItem != 7 {
		t.Errorf("StartItem = %d, want 7", w.StartItem)
	}
	if w.EndItem != 9 {
		t.Errorf("EndItem = %d, want 9", w.EndItem)
	}
	if w.CursorItemIndex != 9 {
		t.Errorf("CursorItemIndex = %d, want 9", w.CursorItemIndex)
	}
}

func TestVisibleRangeCursorOffScreen(t *testing.T) {
	// 10 collapsed items, viewport=3, scroll=4, cursor=0 (above viewport)
	items := makeCollapsedToolCalls(10)
	w := visibleRange(items, 4, 3, 0, 80, false)

	if w.CursorItemIndex != -1 {
		t.Errorf("CursorItemIndex = %d, want -1 (off-screen)", w.CursorItemIndex)
	}
}

func TestVisibleRangeExpandedItem(t *testing.T) {
	// Item 0: collapsed (1 line)
	// Item 1: expanded with 5 content lines (1 + 5 = 6 lines)
	// Item 2: collapsed (1 line)
	// Total = 8 lines
	items := []model.TimelineItem{
		&model.ToolCall{Name: "Read", Status: model.ToolCallDone},
		&model.ToolCall{
			Name:          "Bash",
			Status:        model.ToolCallDone,
			Expanded:      true,
			RawInput:      map[string]interface{}{"command": "echo hello"},
			ResultContent: "line1\nline2\nline3\nline4\nline5",
		},
		&model.ToolCall{Name: "Read", Status: model.ToolCallDone},
	}

	// Viewport=4, scroll=0: see item 0 (1 line) + first 3 lines of item 1
	w := visibleRange(items, 0, 4, 0, 80, false)

	if w.StartItem != 0 {
		t.Errorf("StartItem = %d, want 0", w.StartItem)
	}
	if w.EndItem != 1 {
		t.Errorf("EndItem = %d, want 1", w.EndItem)
	}
	if w.EndLineOffset != 3 {
		t.Errorf("EndLineOffset = %d, want 3 (3 of 6 lines visible)", w.EndLineOffset)
	}
}

func TestVisibleRangePartialItemAtTop(t *testing.T) {
	// Item 0: expanded bash with 5 result lines → 1 header + 6 content = 7 lines
	// Item 1: collapsed (1 line)
	items := []model.TimelineItem{
		&model.ToolCall{
			Name:          "Bash",
			Status:        model.ToolCallDone,
			Expanded:      true,
			RawInput:      map[string]interface{}{"command": "echo hello"},
			ResultContent: "a\nb\nc\nd\ne",
		},
		&model.ToolCall{Name: "Read", Status: model.ToolCallDone},
	}

	lc0 := ItemLineCount(items[0], false, 80)

	// Scroll into the middle of item 0
	scroll := 3
	w := visibleRange(items, scroll, 5, 0, 80, false)

	if w.StartItem != 0 {
		t.Errorf("StartItem = %d, want 0", w.StartItem)
	}
	if w.StartLineOffset != 3 {
		t.Errorf("StartLineOffset = %d, want 3", w.StartLineOffset)
	}
	// Viewport covers scroll=3 to scroll=8. Item 0 ends at lc0.
	// If lc0 <= 8, item 1 is also visible.
	if lc0 <= scroll+5 {
		if w.EndItem != 1 {
			t.Errorf("EndItem = %d, want 1 (lc0=%d)", w.EndItem, lc0)
		}
	}
}

func TestVisibleRangeExpandedGroup(t *testing.T) {
	// An expanded group with 3 children (all collapsed):
	// 1 header + 3 children = 4 lines
	group := &model.ToolCallGroup{
		ToolName: "Read",
		Expanded: true,
		Children: []*model.ToolCall{
			{Name: "Read", Summary: "a.go", Status: model.ToolCallDone},
			{Name: "Read", Summary: "b.go", Status: model.ToolCallDone},
			{Name: "Read", Summary: "c.go", Status: model.ToolCallDone},
		},
	}
	items := []model.TimelineItem{
		&model.ToolCall{Name: "Bash", Status: model.ToolCallDone}, // flat 0
		group, // flat 1 (header), flat 2,3,4 (children)
		&model.ToolCall{Name: "Bash", Status: model.ToolCallDone}, // flat 5
	}

	// Viewport covers everything (scroll=0, height=10)
	w := visibleRange(items, 0, 10, 3, 80, false)

	if w.StartItem != 0 {
		t.Errorf("StartItem = %d, want 0", w.StartItem)
	}
	if w.EndItem != 2 {
		t.Errorf("EndItem = %d, want 2", w.EndItem)
	}
	// Cursor at flat pos 3 = child[1] of the group → CursorItemIndex should be group index (1)
	if w.CursorItemIndex != 1 {
		t.Errorf("CursorItemIndex = %d, want 1 (group index)", w.CursorItemIndex)
	}
}

func TestVisibleRangeCollapsedGroup(t *testing.T) {
	group := &model.ToolCallGroup{
		ToolName: "Read",
		Expanded: false,
		Children: []*model.ToolCall{
			{Name: "Read", Summary: "a.go", Status: model.ToolCallDone},
			{Name: "Read", Summary: "b.go", Status: model.ToolCallDone},
		},
	}
	items := []model.TimelineItem{
		group, // flat 0, 1 line
		&model.ToolCall{Name: "Bash", Status: model.ToolCallDone}, // flat 1
	}

	w := visibleRange(items, 0, 10, 0, 80, false)

	if w.StartItem != 0 {
		t.Errorf("StartItem = %d, want 0", w.StartItem)
	}
	if w.EndItem != 1 {
		t.Errorf("EndItem = %d, want 1", w.EndItem)
	}
	if w.CursorItemIndex != 0 {
		t.Errorf("CursorItemIndex = %d, want 0", w.CursorItemIndex)
	}
}

func TestVisibleRangeGroupCursorOnHeader(t *testing.T) {
	group := &model.ToolCallGroup{
		ToolName: "Read",
		Expanded: true,
		Children: []*model.ToolCall{
			{Name: "Read", Summary: "a.go", Status: model.ToolCallDone},
		},
	}
	items := []model.TimelineItem{group}

	// Cursor at flat 0 = group header
	w := visibleRange(items, 0, 10, 0, 80, false)
	if w.CursorItemIndex != 0 {
		t.Errorf("CursorItemIndex = %d, want 0", w.CursorItemIndex)
	}
}

func TestVisibleRangeTextBlock(t *testing.T) {
	items := []model.TimelineItem{
		&model.TextBlock{Text: "line1\nline2\nline3"}, // 3 lines (collapsed = max 3)
		&model.ToolCall{Name: "Bash", Status: model.ToolCallDone},
	}

	w := visibleRange(items, 0, 10, 0, 80, false)

	if w.StartItem != 0 {
		t.Errorf("StartItem = %d, want 0", w.StartItem)
	}
	if w.EndItem != 1 {
		t.Errorf("EndItem = %d, want 1", w.EndItem)
	}
}

func TestVisibleRangeCompactView(t *testing.T) {
	// In compact view, text blocks are capped at 1 line when collapsed.
	items := []model.TimelineItem{
		&model.TextBlock{Text: "line1\nline2\nline3"}, // compact: 1 line
		&model.ToolCall{Name: "Bash", Status: model.ToolCallDone},
	}

	w := visibleRange(items, 0, 1, 0, 80, true)

	// Only 1 line visible, should show just item 0
	if w.StartItem != 0 {
		t.Errorf("StartItem = %d, want 0", w.StartItem)
	}
	if w.EndItem != 0 {
		t.Errorf("EndItem = %d, want 0", w.EndItem)
	}
}

func TestVisibleRangeConsistentWithTotalLines(t *testing.T) {
	// For a fully visible timeline, EndItem should be the last item,
	// and sum of all line counts should match TotalLines.
	items := []model.TimelineItem{
		&model.ToolCall{Name: "Read", Status: model.ToolCallDone},
		&model.TextBlock{Text: "hello"},
		&model.ToolCall{Name: "Bash", Status: model.ToolCallDone, Expanded: true,
			RawInput: map[string]interface{}{"command": "ls"}, ResultContent: "a\nb"},
		&model.ToolCall{Name: "Read", Status: model.ToolCallDone},
	}

	total := TotalLines(items, false, 80)
	w := visibleRange(items, 0, total, 0, 80, false)

	if w.StartItem != 0 {
		t.Errorf("StartItem = %d, want 0", w.StartItem)
	}
	if w.EndItem != len(items)-1 {
		t.Errorf("EndItem = %d, want %d", w.EndItem, len(items)-1)
	}
	if w.StartLineOffset != 0 {
		t.Errorf("StartLineOffset = %d, want 0", w.StartLineOffset)
	}
	// EndLineOffset should be the full line count of the last item
	lastLC := ItemLineCount(items[len(items)-1], false, 80)
	if w.EndLineOffset != lastLC {
		t.Errorf("EndLineOffset = %d, want %d", w.EndLineOffset, lastLC)
	}
}

func TestVisibleRangeEarlyExit(t *testing.T) {
	// With 1000 collapsed items, visibleRange should stop early.
	// We can't directly test "early exit" but we can verify correctness.
	items := makeCollapsedToolCalls(1000)
	w := visibleRange(items, 500, 10, 505, 80, false)

	if w.StartItem != 500 {
		t.Errorf("StartItem = %d, want 500", w.StartItem)
	}
	if w.EndItem != 509 {
		t.Errorf("EndItem = %d, want 509", w.EndItem)
	}
	if w.CursorItemIndex != 505 {
		t.Errorf("CursorItemIndex = %d, want 505", w.CursorItemIndex)
	}
}

func TestVisibleRangeCursorBelowViewport(t *testing.T) {
	items := makeCollapsedToolCalls(10)
	// Viewport shows items 0-2, cursor at item 8
	w := visibleRange(items, 0, 3, 8, 80, false)

	if w.CursorItemIndex != -1 {
		t.Errorf("CursorItemIndex = %d, want -1 (cursor below viewport)", w.CursorItemIndex)
	}
}

func TestVisibleRangeWidthAffectsEditLayout(t *testing.T) {
	// Edit diffs switch from unified to side-by-side at width >= 120.
	// Verify visibleRange produces correct results at different widths.
	tc := &model.ToolCall{
		Name:     "Edit",
		Status:   model.ToolCallDone,
		Expanded: true,
		RawInput: map[string]interface{}{
			"old_string": "aaa\nbbb\nccc",
			"new_string": "xxx\nyyy",
		},
	}
	items := []model.TimelineItem{tc}

	narrowLC := ItemLineCount(tc, false, 80)
	wideLC := ItemLineCount(tc, false, 140)

	wNarrow := visibleRange(items, 0, 100, 0, 80, false)
	wWide := visibleRange(items, 0, 100, 0, 140, false)

	if wNarrow.EndLineOffset != narrowLC {
		t.Errorf("narrow EndLineOffset = %d, want %d", wNarrow.EndLineOffset, narrowLC)
	}
	if wWide.EndLineOffset != wideLC {
		t.Errorf("wide EndLineOffset = %d, want %d", wWide.EndLineOffset, wideLC)
	}
}
