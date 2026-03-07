package tui

import (
	"testing"

	"github.com/loxstomper/skinner/internal/model"
)

// helper to build a standalone ToolCall item
func tc(id string) *model.ToolCall {
	return &model.ToolCall{ID: id, Name: "Read", Status: model.ToolCallRunning}
}

// helper to build a TextBlock item
func tb(text string) *model.TextBlock {
	return &model.TextBlock{Text: text}
}

// helper to build an expanded ToolCallGroup with n children
func expandedGroup(n int) *model.ToolCallGroup {
	g := &model.ToolCallGroup{ToolName: "Read", Expanded: true}
	for i := 0; i < n; i++ {
		g.Children = append(g.Children, &model.ToolCall{Name: "Read", Status: model.ToolCallRunning})
	}
	return g
}

// helper to build a collapsed ToolCallGroup with n children
func collapsedGroup(n int) *model.ToolCallGroup {
	g := expandedGroup(n)
	g.Expanded = false
	return g
}

func TestFlatCursorCount_Empty(t *testing.T) {
	count := FlatCursorCount(nil)
	if count != 0 {
		t.Errorf("FlatCursorCount(nil) = %d, want 0", count)
	}
	count = FlatCursorCount([]model.TimelineItem{})
	if count != 0 {
		t.Errorf("FlatCursorCount([]) = %d, want 0", count)
	}
}

func TestFlatCursorCount_StandaloneItems(t *testing.T) {
	items := []model.TimelineItem{tc("1"), tc("2"), tb("hello")}
	count := FlatCursorCount(items)
	if count != 3 {
		t.Errorf("FlatCursorCount = %d, want 3", count)
	}
}

func TestFlatCursorCount_ExpandedGroup(t *testing.T) {
	items := []model.TimelineItem{tc("1"), expandedGroup(3), tc("2")}
	// tc + (header + 3 children) + tc = 1 + 4 + 1 = 6
	count := FlatCursorCount(items)
	if count != 6 {
		t.Errorf("FlatCursorCount = %d, want 6", count)
	}
}

func TestFlatCursorCount_CollapsedGroup(t *testing.T) {
	items := []model.TimelineItem{tc("1"), collapsedGroup(3), tc("2")}
	// tc + header + tc = 3
	count := FlatCursorCount(items)
	if count != 3 {
		t.Errorf("FlatCursorCount = %d, want 3", count)
	}
}

func TestFlatToItem_StandaloneItems(t *testing.T) {
	items := []model.TimelineItem{tc("a"), tb("b"), tc("c")}

	tests := []struct {
		flatIdx   int
		wantItem  int
		wantChild int
	}{
		{0, 0, -1},
		{1, 1, -1},
		{2, 2, -1},
	}
	for _, tt := range tests {
		itemIdx, childIdx := FlatToItem(items, tt.flatIdx)
		if itemIdx != tt.wantItem || childIdx != tt.wantChild {
			t.Errorf("FlatToItem(items, %d) = (%d, %d), want (%d, %d)",
				tt.flatIdx, itemIdx, childIdx, tt.wantItem, tt.wantChild)
		}
	}
}

func TestFlatToItem_ExpandedGroup(t *testing.T) {
	// items: tc, expandedGroup(2), tc
	// flat positions: 0=tc, 1=group header, 2=child0, 3=child1, 4=tc
	items := []model.TimelineItem{tc("a"), expandedGroup(2), tc("c")}

	tests := []struct {
		flatIdx   int
		wantItem  int
		wantChild int
	}{
		{0, 0, -1}, // standalone tc
		{1, 1, -1}, // group header
		{2, 1, 0},  // group child 0
		{3, 1, 1},  // group child 1
		{4, 2, -1}, // standalone tc after group
	}
	for _, tt := range tests {
		itemIdx, childIdx := FlatToItem(items, tt.flatIdx)
		if itemIdx != tt.wantItem || childIdx != tt.wantChild {
			t.Errorf("FlatToItem(items, %d) = (%d, %d), want (%d, %d)",
				tt.flatIdx, itemIdx, childIdx, tt.wantItem, tt.wantChild)
		}
	}
}

func TestFlatToItem_CollapsedGroup(t *testing.T) {
	// items: tc, collapsedGroup(2), tc
	// flat positions: 0=tc, 1=group header (children hidden), 2=tc
	items := []model.TimelineItem{tc("a"), collapsedGroup(2), tc("c")}

	tests := []struct {
		flatIdx   int
		wantItem  int
		wantChild int
	}{
		{0, 0, -1}, // standalone tc
		{1, 1, -1}, // group header
		{2, 2, -1}, // standalone tc after group
	}
	for _, tt := range tests {
		itemIdx, childIdx := FlatToItem(items, tt.flatIdx)
		if itemIdx != tt.wantItem || childIdx != tt.wantChild {
			t.Errorf("FlatToItem(items, %d) = (%d, %d), want (%d, %d)",
				tt.flatIdx, itemIdx, childIdx, tt.wantItem, tt.wantChild)
		}
	}
}

func TestFlatToItem_OutOfRange(t *testing.T) {
	items := []model.TimelineItem{tc("a")}
	itemIdx, childIdx := FlatToItem(items, 99)
	if itemIdx != 0 || childIdx != -1 {
		t.Errorf("FlatToItem(items, 99) = (%d, %d), want (0, -1)", itemIdx, childIdx)
	}
}

func TestFlatToItem_Empty(t *testing.T) {
	itemIdx, childIdx := FlatToItem(nil, 0)
	if itemIdx != 0 || childIdx != -1 {
		t.Errorf("FlatToItem(nil, 0) = (%d, %d), want (0, -1)", itemIdx, childIdx)
	}
}

func TestItemToFlat_StandaloneItems(t *testing.T) {
	items := []model.TimelineItem{tc("a"), tb("b"), tc("c")}
	for i := range items {
		flat := ItemToFlat(items, i)
		if flat != i {
			t.Errorf("ItemToFlat(items, %d) = %d, want %d", i, flat, i)
		}
	}
}

func TestItemToFlat_ExpandedGroup(t *testing.T) {
	// items: tc, expandedGroup(2), tc
	// item 0 -> flat 0, item 1 -> flat 1, item 2 -> flat 4
	items := []model.TimelineItem{tc("a"), expandedGroup(2), tc("c")}

	tests := []struct {
		itemIdx  int
		wantFlat int
	}{
		{0, 0},
		{1, 1},
		{2, 4}, // after header(1) + 2 children
	}
	for _, tt := range tests {
		flat := ItemToFlat(items, tt.itemIdx)
		if flat != tt.wantFlat {
			t.Errorf("ItemToFlat(items, %d) = %d, want %d", tt.itemIdx, flat, tt.wantFlat)
		}
	}
}

func TestItemToFlat_CollapsedGroup(t *testing.T) {
	items := []model.TimelineItem{tc("a"), collapsedGroup(2), tc("c")}
	tests := []struct {
		itemIdx  int
		wantFlat int
	}{
		{0, 0},
		{1, 1},
		{2, 2}, // collapsed group only occupies 1 flat position
	}
	for _, tt := range tests {
		flat := ItemToFlat(items, tt.itemIdx)
		if flat != tt.wantFlat {
			t.Errorf("ItemToFlat(items, %d) = %d, want %d", tt.itemIdx, flat, tt.wantFlat)
		}
	}
}

func TestFlatToItem_ItemToFlat_Roundtrip(t *testing.T) {
	// Verify that FlatToItem and ItemToFlat are consistent for non-group items.
	items := []model.TimelineItem{tc("a"), expandedGroup(3), tb("text"), collapsedGroup(2), tc("z")}

	// For each flat position, FlatToItem gives (itemIdx, childIdx).
	// For non-child positions (childIdx == -1), ItemToFlat(itemIdx) should return flatIdx.
	count := FlatCursorCount(items)
	for f := 0; f < count; f++ {
		itemIdx, childIdx := FlatToItem(items, f)
		if childIdx == -1 {
			roundtrip := ItemToFlat(items, itemIdx)
			if roundtrip != f {
				t.Errorf("Roundtrip failed: FlatToItem(%d) = (%d, -1), ItemToFlat(%d) = %d, want %d",
					f, itemIdx, itemIdx, roundtrip, f)
			}
		}
	}
}

func TestItemLineCount_ToolCall(t *testing.T) {
	lc := ItemLineCount(tc("a"), false)
	if lc != 1 {
		t.Errorf("ItemLineCount(ToolCall, false) = %d, want 1", lc)
	}
	lc = ItemLineCount(tc("a"), true)
	if lc != 1 {
		t.Errorf("ItemLineCount(ToolCall, true) = %d, want 1", lc)
	}
}

func TestItemLineCount_TextBlock(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		expanded    bool
		compactView bool
		want        int
	}{
		{"single line normal", "hello", false, false, 1},
		{"single line compact", "hello", false, true, 1},
		{"multi line within limit", "a\nb\nc", false, false, 3},
		{"multi line exceeds limit collapsed", "a\nb\nc\nd\ne", false, false, 3},
		{"multi line exceeds limit expanded", "a\nb\nc\nd\ne", true, false, 5},
		{"compact single line", "a\nb\nc\nd\ne", false, true, 1},
		{"compact expanded", "a\nb\nc\nd\ne", true, true, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &model.TextBlock{Text: tt.text, Expanded: tt.expanded}
			got := ItemLineCount(item, tt.compactView)
			if got != tt.want {
				t.Errorf("ItemLineCount = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestItemLineCount_Group(t *testing.T) {
	expanded := expandedGroup(3)
	lc := ItemLineCount(expanded, false)
	if lc != 4 { // header + 3 children
		t.Errorf("ItemLineCount(expanded group of 3) = %d, want 4", lc)
	}

	collapsed := collapsedGroup(3)
	lc = ItemLineCount(collapsed, false)
	if lc != 1 { // header only
		t.Errorf("ItemLineCount(collapsed group) = %d, want 1", lc)
	}
}

func TestTotalLines_Empty(t *testing.T) {
	total := TotalLines(nil, false)
	if total != 0 {
		t.Errorf("TotalLines(nil) = %d, want 0", total)
	}
}

func TestTotalLines_Mixed(t *testing.T) {
	items := []model.TimelineItem{
		tc("a"),          // 1 line
		expandedGroup(2), // 3 lines (header + 2)
		tb("x\ny\nz\nw"), // 3 lines (4 > maxLines=3, collapsed)
	}
	total := TotalLines(items, false)
	if total != 7 { // 1 + 3 + 3
		t.Errorf("TotalLines = %d, want 7", total)
	}
}

func TestTotalLines_CompactView(t *testing.T) {
	items := []model.TimelineItem{
		tc("a"),          // 1 line
		tb("x\ny\nz\nw"), // 1 line in compact (4 > maxLines=1)
	}
	total := TotalLines(items, true)
	if total != 2 { // 1 + 1
		t.Errorf("TotalLines(compact) = %d, want 2", total)
	}
}

func TestFlatCursorLineRange_StandaloneItems(t *testing.T) {
	items := []model.TimelineItem{tc("a"), tc("b"), tc("c")}

	tests := []struct {
		flatIdx   int
		wantStart int
		wantCount int
	}{
		{0, 0, 1},
		{1, 1, 1},
		{2, 2, 1},
	}
	for _, tt := range tests {
		start, count := FlatCursorLineRange(items, tt.flatIdx, false)
		if start != tt.wantStart || count != tt.wantCount {
			t.Errorf("FlatCursorLineRange(items, %d) = (%d, %d), want (%d, %d)",
				tt.flatIdx, start, count, tt.wantStart, tt.wantCount)
		}
	}
}

func TestFlatCursorLineRange_TextBlock(t *testing.T) {
	// Multi-line text block (4 lines, collapsed to 3)
	items := []model.TimelineItem{tc("a"), tb("x\ny\nz\nw"), tc("b")}
	// flat 0 = tc -> line 0, count 1
	// flat 1 = tb -> line 1, count 3 (collapsed from 4 to 3)
	// flat 2 = tc -> line 4, count 1
	tests := []struct {
		flatIdx   int
		wantStart int
		wantCount int
	}{
		{0, 0, 1},
		{1, 1, 3},
		{2, 4, 1},
	}
	for _, tt := range tests {
		start, count := FlatCursorLineRange(items, tt.flatIdx, false)
		if start != tt.wantStart || count != tt.wantCount {
			t.Errorf("FlatCursorLineRange(items, %d) = (%d, %d), want (%d, %d)",
				tt.flatIdx, start, count, tt.wantStart, tt.wantCount)
		}
	}
}

func TestFlatCursorLineRange_ExpandedGroup(t *testing.T) {
	// tc, expandedGroup(2), tc
	// flat 0 = tc -> line 0
	// flat 1 = group header -> line 1
	// flat 2 = child 0 -> line 2
	// flat 3 = child 1 -> line 3
	// flat 4 = tc -> line 4
	items := []model.TimelineItem{tc("a"), expandedGroup(2), tc("c")}
	tests := []struct {
		flatIdx   int
		wantStart int
		wantCount int
	}{
		{0, 0, 1},
		{1, 1, 1},
		{2, 2, 1},
		{3, 3, 1},
		{4, 4, 1},
	}
	for _, tt := range tests {
		start, count := FlatCursorLineRange(items, tt.flatIdx, false)
		if start != tt.wantStart || count != tt.wantCount {
			t.Errorf("FlatCursorLineRange(items, %d) = (%d, %d), want (%d, %d)",
				tt.flatIdx, start, count, tt.wantStart, tt.wantCount)
		}
	}
}

func TestFlatCursorLineRange_OutOfRange(t *testing.T) {
	items := []model.TimelineItem{tc("a")}
	start, count := FlatCursorLineRange(items, 99, false)
	// Out-of-range returns (total lines, 1) — the fallthrough at end
	if count != 1 {
		t.Errorf("FlatCursorLineRange(items, 99) count = %d, want 1", count)
	}
	_ = start // start value is the total line count
}

func TestFlatCursorLineRange_Empty(t *testing.T) {
	start, count := FlatCursorLineRange(nil, 0, false)
	if start != 0 || count != 1 {
		t.Errorf("FlatCursorLineRange(nil, 0) = (%d, %d), want (0, 1)", start, count)
	}
}

func TestFlatCursorCount_MixedItems(t *testing.T) {
	items := []model.TimelineItem{
		tc("a"),
		expandedGroup(3),
		tb("text"),
		collapsedGroup(2),
		tc("z"),
	}
	// tc=1, expanded(header+3)=4, tb=1, collapsed(header)=1, tc=1 = 8
	count := FlatCursorCount(items)
	if count != 8 {
		t.Errorf("FlatCursorCount = %d, want 8", count)
	}
}
