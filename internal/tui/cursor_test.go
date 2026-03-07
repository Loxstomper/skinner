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

// helper to build an expanded ToolCall with result content that produces
// contentLines lines of output. Total display lines = 1 (header) + contentLines.
func expandedTC(id string, contentLines int) *model.ToolCall {
	content := "line1"
	for i := 2; i <= contentLines; i++ {
		content += "\nline" + string(rune('0'+i))
	}
	return &model.ToolCall{
		ID:            id,
		Name:          "Read",
		Status:        model.ToolCallDone,
		ResultContent: content,
		Expanded:      true,
	}
}

// helper to build an expanded ToolCallGroup where one child is expanded
func expandedGroupWithExpandedChild(nChildren, expandedChildIdx, contentLines int) *model.ToolCallGroup {
	g := &model.ToolCallGroup{ToolName: "Read", Expanded: true}
	for i := 0; i < nChildren; i++ {
		if i == expandedChildIdx {
			g.Children = append(g.Children, expandedTC("child", contentLines))
		} else {
			g.Children = append(g.Children, &model.ToolCall{Name: "Read", Status: model.ToolCallRunning})
		}
	}
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

func TestLineToFlatCursor_Empty(t *testing.T) {
	got := LineToFlatCursor(nil, 0, false)
	if got != 0 {
		t.Errorf("LineToFlatCursor(nil, 0) = %d, want 0", got)
	}
}

func TestLineToFlatCursor_StandaloneItems(t *testing.T) {
	// 3 tool calls = 3 lines, one per flat position
	items := []model.TimelineItem{tc("a"), tc("b"), tc("c")}
	tests := []struct {
		line int
		want int
	}{
		{0, 0},
		{1, 1},
		{2, 2},
	}
	for _, tt := range tests {
		got := LineToFlatCursor(items, tt.line, false)
		if got != tt.want {
			t.Errorf("LineToFlatCursor(items, %d) = %d, want %d", tt.line, got, tt.want)
		}
	}
}

func TestLineToFlatCursor_TextBlock(t *testing.T) {
	// tc(a) at line 0 (1 line), tb("x\ny\nz\nw") at lines 1-3 (collapsed to 3 lines), tc(b) at line 4
	items := []model.TimelineItem{tc("a"), tb("x\ny\nz\nw"), tc("b")}
	tests := []struct {
		line int
		want int
	}{
		{0, 0}, // tc("a")
		{1, 1}, // first line of text block
		{2, 1}, // second line of text block — still flat pos 1
		{3, 1}, // third line of text block — still flat pos 1
		{4, 2}, // tc("b")
	}
	for _, tt := range tests {
		got := LineToFlatCursor(items, tt.line, false)
		if got != tt.want {
			t.Errorf("LineToFlatCursor(items, %d) = %d, want %d", tt.line, got, tt.want)
		}
	}
}

func TestLineToFlatCursor_ExpandedGroup(t *testing.T) {
	// tc, expandedGroup(2), tc
	// line 0: tc (flat 0)
	// line 1: group header (flat 1)
	// line 2: child 0 (flat 2)
	// line 3: child 1 (flat 3)
	// line 4: tc (flat 4)
	items := []model.TimelineItem{tc("a"), expandedGroup(2), tc("c")}
	tests := []struct {
		line int
		want int
	}{
		{0, 0},
		{1, 1},
		{2, 2},
		{3, 3},
		{4, 4},
	}
	for _, tt := range tests {
		got := LineToFlatCursor(items, tt.line, false)
		if got != tt.want {
			t.Errorf("LineToFlatCursor(items, %d) = %d, want %d", tt.line, got, tt.want)
		}
	}
}

func TestLineToFlatCursor_CollapsedGroup(t *testing.T) {
	// tc, collapsedGroup(2), tc
	// line 0: tc (flat 0)
	// line 1: group header (flat 1)
	// line 2: tc (flat 2)
	items := []model.TimelineItem{tc("a"), collapsedGroup(2), tc("c")}
	tests := []struct {
		line int
		want int
	}{
		{0, 0},
		{1, 1},
		{2, 2},
	}
	for _, tt := range tests {
		got := LineToFlatCursor(items, tt.line, false)
		if got != tt.want {
			t.Errorf("LineToFlatCursor(items, %d) = %d, want %d", tt.line, got, tt.want)
		}
	}
}

func TestLineToFlatCursor_BeyondEnd(t *testing.T) {
	items := []model.TimelineItem{tc("a"), tc("b")}
	// Line 99 is way beyond — should return last flat position (1)
	got := LineToFlatCursor(items, 99, false)
	if got != 1 {
		t.Errorf("LineToFlatCursor(items, 99) = %d, want 1", got)
	}
}

func TestLineToFlatCursor_RoundtripWithFlatCursorLineRange(t *testing.T) {
	// For every flat cursor position, FlatCursorLineRange gives us the start line.
	// LineToFlatCursor(startLine) should return the same flat cursor position.
	items := []model.TimelineItem{
		tc("a"),
		expandedGroup(3),
		tb("line1\nline2\nline3\nline4"), // collapsed to 3 lines
		collapsedGroup(2),
		tc("z"),
	}
	count := FlatCursorCount(items)
	for f := 0; f < count; f++ {
		lineStart, _ := FlatCursorLineRange(items, f, false)
		roundtrip := LineToFlatCursor(items, lineStart, false)
		if roundtrip != f {
			t.Errorf("Roundtrip failed: flat %d -> line %d -> flat %d", f, lineStart, roundtrip)
		}
	}
}

func TestLineToFlatCursor_CompactView(t *testing.T) {
	// In compact view, text blocks take only 1 line
	items := []model.TimelineItem{tc("a"), tb("x\ny\nz\nw"), tc("b")}
	// line 0: tc("a") (flat 0)
	// line 1: tb (flat 1, compact = 1 line)
	// line 2: tc("b") (flat 2)
	tests := []struct {
		line int
		want int
	}{
		{0, 0},
		{1, 1},
		{2, 2},
	}
	for _, tt := range tests {
		got := LineToFlatCursor(items, tt.line, true)
		if got != tt.want {
			t.Errorf("LineToFlatCursor(items, %d, compact) = %d, want %d", tt.line, got, tt.want)
		}
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

// --- Expanded ToolCall tests (multi-line tool calls) ---

func TestItemLineCount_ExpandedToolCall(t *testing.T) {
	// An expanded ToolCall with 3 content lines = 1 header + 3 content = 4 lines
	etc := expandedTC("a", 3)
	lc := ItemLineCount(etc, false)
	if lc != 4 {
		t.Errorf("ItemLineCount(expanded ToolCall, 3 content lines) = %d, want 4", lc)
	}

	// A collapsed ToolCall still returns 1
	collapsed := tc("b")
	lc = ItemLineCount(collapsed, false)
	if lc != 1 {
		t.Errorf("ItemLineCount(collapsed ToolCall) = %d, want 1", lc)
	}
}

func TestItemLineCount_GroupWithExpandedChild(t *testing.T) {
	// Group with 3 children, child 1 expanded with 2 content lines.
	// header(1) + child0(1) + child1(1+2=3) + child2(1) = 6
	g := expandedGroupWithExpandedChild(3, 1, 2)
	lc := ItemLineCount(g, false)
	if lc != 6 {
		t.Errorf("ItemLineCount(group with expanded child) = %d, want 6", lc)
	}
}

func TestTotalLines_WithExpandedToolCall(t *testing.T) {
	// expandedTC with 3 content lines = 4 display lines, tc = 1 line
	items := []model.TimelineItem{
		expandedTC("a", 3), // 4 lines
		tc("b"),            // 1 line
	}
	total := TotalLines(items, false)
	if total != 5 {
		t.Errorf("TotalLines with expanded ToolCall = %d, want 5", total)
	}
}

func TestFlatCursorLineRange_ExpandedToolCall(t *testing.T) {
	// tc(a), expandedTC(b, 3 content lines), tc(c)
	// flat 0 = tc(a) -> line 0, count 1
	// flat 1 = expandedTC(b) -> line 1, count 4 (1 header + 3 content)
	// flat 2 = tc(c) -> line 5, count 1
	items := []model.TimelineItem{tc("a"), expandedTC("b", 3), tc("c")}
	tests := []struct {
		flatIdx   int
		wantStart int
		wantCount int
	}{
		{0, 0, 1},
		{1, 1, 4},
		{2, 5, 1},
	}
	for _, tt := range tests {
		start, count := FlatCursorLineRange(items, tt.flatIdx, false)
		if start != tt.wantStart || count != tt.wantCount {
			t.Errorf("FlatCursorLineRange(items, %d) = (%d, %d), want (%d, %d)",
				tt.flatIdx, start, count, tt.wantStart, tt.wantCount)
		}
	}
}

func TestFlatCursorLineRange_GroupWithExpandedChild(t *testing.T) {
	// expandedGroupWithExpandedChild(3, 1, 2):
	//   header(1), child0(1), child1(1+2=3), child2(1)
	// Placed after tc(a):
	// flat 0 = tc(a) -> line 0, count 1
	// flat 1 = group header -> line 1, count 1
	// flat 2 = child 0 -> line 2, count 1
	// flat 3 = child 1 (expanded) -> line 3, count 3
	// flat 4 = child 2 -> line 6, count 1
	// flat 5 = tc(c) -> line 7, count 1
	items := []model.TimelineItem{tc("a"), expandedGroupWithExpandedChild(3, 1, 2), tc("c")}
	tests := []struct {
		flatIdx   int
		wantStart int
		wantCount int
	}{
		{0, 0, 1},
		{1, 1, 1},
		{2, 2, 1},
		{3, 3, 3},
		{4, 6, 1},
		{5, 7, 1},
	}
	for _, tt := range tests {
		start, count := FlatCursorLineRange(items, tt.flatIdx, false)
		if start != tt.wantStart || count != tt.wantCount {
			t.Errorf("FlatCursorLineRange(items, %d) = (%d, %d), want (%d, %d)",
				tt.flatIdx, start, count, tt.wantStart, tt.wantCount)
		}
	}
}

func TestLineToFlatCursor_ExpandedToolCall(t *testing.T) {
	// tc(a), expandedTC(b, 3 content lines), tc(c)
	// line 0: tc(a) -> flat 0
	// line 1: expandedTC header -> flat 1
	// line 2: expandedTC content line 1 -> flat 1 (same cursor)
	// line 3: expandedTC content line 2 -> flat 1
	// line 4: expandedTC content line 3 -> flat 1
	// line 5: tc(c) -> flat 2
	items := []model.TimelineItem{tc("a"), expandedTC("b", 3), tc("c")}
	tests := []struct {
		line int
		want int
	}{
		{0, 0},
		{1, 1},
		{2, 1}, // content line maps to same flat cursor
		{3, 1},
		{4, 1},
		{5, 2},
	}
	for _, tt := range tests {
		got := LineToFlatCursor(items, tt.line, false)
		if got != tt.want {
			t.Errorf("LineToFlatCursor(items, %d) = %d, want %d", tt.line, got, tt.want)
		}
	}
}

func TestLineToFlatCursor_GroupWithExpandedChild(t *testing.T) {
	// expandedGroupWithExpandedChild(3, 1, 2):
	// line 0: tc(a) -> flat 0
	// line 1: group header -> flat 1
	// line 2: child 0 -> flat 2
	// line 3: child 1 header -> flat 3
	// line 4: child 1 content 1 -> flat 3
	// line 5: child 1 content 2 -> flat 3
	// line 6: child 2 -> flat 4
	// line 7: tc(c) -> flat 5
	items := []model.TimelineItem{tc("a"), expandedGroupWithExpandedChild(3, 1, 2), tc("c")}
	tests := []struct {
		line int
		want int
	}{
		{0, 0},
		{1, 1},
		{2, 2},
		{3, 3},
		{4, 3}, // content line maps to child 1's flat cursor
		{5, 3},
		{6, 4},
		{7, 5},
	}
	for _, tt := range tests {
		got := LineToFlatCursor(items, tt.line, false)
		if got != tt.want {
			t.Errorf("LineToFlatCursor(items, %d) = %d, want %d", tt.line, got, tt.want)
		}
	}
}

func TestLineToFlatCursor_RoundtripWithExpandedToolCalls(t *testing.T) {
	// Comprehensive roundtrip test including expanded tool calls
	items := []model.TimelineItem{
		tc("a"),
		expandedTC("b", 2),                      // 3 display lines
		expandedGroupWithExpandedChild(2, 0, 3), // header + child0(4 lines) + child1(1 line)
		tb("line1\nline2\nline3\nline4"),        // collapsed to 3 lines
		tc("z"),
	}
	count := FlatCursorCount(items)
	for f := 0; f < count; f++ {
		lineStart, _ := FlatCursorLineRange(items, f, false)
		roundtrip := LineToFlatCursor(items, lineStart, false)
		if roundtrip != f {
			t.Errorf("Roundtrip failed: flat %d -> line %d -> flat %d", f, lineStart, roundtrip)
		}
	}
}
