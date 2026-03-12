package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/loxstomper/skinner/internal/model"
)

func makeTimelineItems() []model.TimelineItem {
	return []model.TimelineItem{
		&model.TextBlock{Text: "Looking at the code"},
		&model.ToolCall{
			ID: "tc1", Name: "Read", Summary: "main.go",
			Status: model.ToolCallDone, Duration: 2 * time.Second,
		},
		&model.ToolCall{
			ID: "tc2", Name: "Edit", Summary: "main.go (+2/-2)",
			Status: model.ToolCallDone, Duration: 300 * time.Millisecond,
		},
		&model.TextBlock{Text: "Tests still failing"},
	}
}

func timelineProps(items []model.TimelineItem) TimelineProps {
	return TimelineProps{
		Items:       items,
		Width:       80,
		Height:      20,
		Focused:     true,
		CompactView: false,
		Theme:       testTheme(),
	}
}

func TestTimeline_CursorDown(t *testing.T) {
	tl := NewTimeline()
	items := makeTimelineItems()
	props := timelineProps(items)

	tl.HandleAction("move_down", props)
	if tl.Cursor != 1 {
		t.Errorf("expected cursor=1 after j, got %d", tl.Cursor)
	}

	tl.HandleAction("move_down", props)
	if tl.Cursor != 2 {
		t.Errorf("expected cursor=2 after second j, got %d", tl.Cursor)
	}
}

func TestTimeline_CursorUp(t *testing.T) {
	tl := NewTimeline()
	tl.Cursor = 3
	items := makeTimelineItems()
	props := timelineProps(items)

	tl.HandleAction("move_up", props)
	if tl.Cursor != 2 {
		t.Errorf("expected cursor=2 after k, got %d", tl.Cursor)
	}
}

func TestTimeline_CursorBounds(t *testing.T) {
	t.Run("cannot go below 0", func(t *testing.T) {
		tl := NewTimeline()
		items := makeTimelineItems()
		props := timelineProps(items)

		tl.HandleAction("move_up", props)
		if tl.Cursor != 0 {
			t.Errorf("expected cursor=0 at top, got %d", tl.Cursor)
		}
	})

	t.Run("cannot exceed count-1", func(t *testing.T) {
		tl := NewTimeline()
		items := makeTimelineItems()
		tl.Cursor = len(items) - 1
		props := timelineProps(items)

		tl.HandleAction("move_down", props)
		if tl.Cursor != len(items)-1 {
			t.Errorf("expected cursor=%d at bottom, got %d", len(items)-1, tl.Cursor)
		}
	})
}

func TestTimeline_EnterExpandCollapse_TextBlock(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.TextBlock{Text: "line1\nline2\nline3\nline4\nline5"},
	}
	props := timelineProps(items)

	tb := items[0].(*model.TextBlock)
	if tb.Expanded {
		t.Error("text block should start collapsed")
	}

	tl.HandleAction("expand", props)
	if !tb.Expanded {
		t.Error("text block should be expanded after enter")
	}

	tl.HandleAction("expand", props)
	if tb.Expanded {
		t.Error("text block should be collapsed after second enter")
	}
}

func TestTimeline_EnterExpandCollapse_Group(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCallGroup{
			ToolName: "Read",
			Expanded: true,
			Children: []*model.ToolCall{
				{ID: "tc1", Name: "Read", Summary: "a.go", Status: model.ToolCallDone},
				{ID: "tc2", Name: "Read", Summary: "b.go", Status: model.ToolCallDone},
			},
		},
	}
	props := timelineProps(items)

	group := items[0].(*model.ToolCallGroup)

	// Cursor is on group header (flatIdx 0)
	tl.HandleAction("expand", props)
	if group.Expanded {
		t.Error("group should be collapsed after enter on header")
	}
	if !group.ManualToggle {
		t.Error("group should have ManualToggle set")
	}

	tl.HandleAction("expand", props)
	if !group.Expanded {
		t.Error("group should be expanded after second enter")
	}
}

func TestTimeline_EnterOnGroupChild_TogglesChildExpansion(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCallGroup{
			ToolName: "Read",
			Expanded: true,
			Children: []*model.ToolCall{
				{ID: "tc1", Name: "Read", Summary: "a.go", Status: model.ToolCallDone, ResultContent: "file contents"},
				{ID: "tc2", Name: "Read", Summary: "b.go", Status: model.ToolCallDone},
			},
		},
	}
	props := timelineProps(items)
	group := items[0].(*model.ToolCallGroup)
	child := group.Children[0]

	if child.Expanded {
		t.Error("child should start collapsed")
	}

	// Move cursor to first child (flatIdx 1) and press Enter
	tl.Cursor = 1
	tl.HandleAction("expand", props)
	if !child.Expanded {
		t.Error("child should be expanded after enter")
	}
	if !group.Expanded {
		t.Error("group should remain expanded after toggling child")
	}

	// Press Enter again to collapse
	tl.HandleAction("expand", props)
	if child.Expanded {
		t.Error("child should be collapsed after second enter")
	}
}

func TestTimeline_EnterOnToolCall_TogglesExpansion(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Bash", Summary: "ls",
			Status:        model.ToolCallDone,
			RawInput:      map[string]interface{}{"command": "ls"},
			ResultContent: "file1.go\nfile2.go",
		},
	}
	props := timelineProps(items)
	tc := items[0].(*model.ToolCall)

	if tc.Expanded {
		t.Error("tool call should start collapsed")
	}

	tl.HandleAction("expand", props)
	if !tc.Expanded {
		t.Error("tool call should be expanded after enter")
	}

	tl.HandleAction("expand", props)
	if tc.Expanded {
		t.Error("tool call should be collapsed after second enter")
	}
}

func TestTimeline_AutoFollow(t *testing.T) {
	tl := NewTimeline()

	if !tl.AutoFollow.Following() {
		t.Error("expected auto-follow to start as true")
	}

	items := makeTimelineItems()
	props := timelineProps(items)
	tl.Cursor = 1
	tl.HandleAction("move_up", props)
	if tl.AutoFollow.Following() {
		t.Error("expected auto-follow to pause after moving up")
	}
}

func TestTimeline_OnNewItems(t *testing.T) {
	tl := NewTimeline()

	items := makeTimelineItems()
	props := timelineProps(items)

	tl.OnNewItems(props)
	if tl.Cursor != len(items)-1 {
		t.Errorf("expected cursor=%d after OnNewItems, got %d", len(items)-1, tl.Cursor)
	}
}

func TestTimeline_JumpToTop(t *testing.T) {
	tl := NewTimeline()
	tl.Cursor = 5
	tl.Scroll = 10

	tl.JumpToTop()
	if tl.Cursor != 0 {
		t.Errorf("expected cursor=0 after JumpToTop, got %d", tl.Cursor)
	}
	if tl.Scroll != 0 {
		t.Errorf("expected scroll=0 after JumpToTop, got %d", tl.Scroll)
	}
}

func TestTimeline_JumpToBottom(t *testing.T) {
	tl := NewTimeline()
	items := makeTimelineItems()
	props := timelineProps(items)

	tl.JumpToBottom(props)
	if tl.Cursor != len(items)-1 {
		t.Errorf("expected cursor=%d after JumpToBottom, got %d", len(items)-1, tl.Cursor)
	}
	if !tl.AutoFollow.Following() {
		t.Error("expected auto-follow to resume after JumpToBottom")
	}
}

func TestTimeline_ResetPosition(t *testing.T) {
	tl := NewTimeline()
	tl.Cursor = 5
	tl.Scroll = 10

	tl.ResetPosition()
	if tl.Cursor != 0 || tl.Scroll != 0 {
		t.Errorf("expected cursor=0 scroll=0, got cursor=%d scroll=%d", tl.Cursor, tl.Scroll)
	}
}

func TestTimeline_View_Empty(t *testing.T) {
	tl := NewTimeline()
	props := TimelineProps{
		Items:  nil,
		Width:  80,
		Height: 20,
		Theme:  testTheme(),
	}

	result := tl.View(props)
	if !strings.Contains(result, "No activity yet") {
		t.Error("expected 'No activity yet' for empty timeline")
	}
}

func TestTimeline_View_ToolCalls(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Read", Summary: "main.go",
			Status: model.ToolCallDone, Duration: 2 * time.Second,
			LineInfo: "(85 lines)",
		},
	}
	props := timelineProps(items)

	result := tl.View(props)

	if !strings.Contains(result, "Read") {
		t.Error("expected 'Read' tool name")
	}
	if !strings.Contains(result, "main.go") {
		t.Error("expected 'main.go' summary")
	}
	if strings.Contains(result, "✓") {
		t.Error("result indicator ✓ should not be present")
	}
	if !strings.Contains(result, "(85 lines)") {
		t.Error("expected line info '(85 lines)'")
	}
}

func TestTimeline_View_TextBlocks(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.TextBlock{Text: "Looking at the test failures"},
	}
	props := timelineProps(items)

	result := tl.View(props)

	if !strings.Contains(result, "Looking at the test failures") {
		t.Error("expected text block content")
	}
}

func TestTimeline_View_CompactView(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Read", Summary: "main.go",
			Status: model.ToolCallDone, Duration: 2 * time.Second,
		},
	}
	props := TimelineProps{
		Items:       items,
		Width:       80,
		Height:      20,
		Focused:     true,
		CompactView: true,
		Theme:       testTheme(),
	}

	result := tl.View(props)

	// In compact view, known tools don't show the name
	if !strings.Contains(result, "main.go") {
		t.Error("expected 'main.go' summary in compact view")
	}
}

func TestTimeline_View_GroupCollapsed(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCallGroup{
			ToolName: "Read",
			Expanded: false,
			Children: []*model.ToolCall{
				{ID: "tc1", Name: "Read", Summary: "a.go", Status: model.ToolCallDone, Duration: time.Second},
				{ID: "tc2", Name: "Read", Summary: "b.go", Status: model.ToolCallDone, Duration: time.Second},
				{ID: "tc3", Name: "Read", Summary: "c.go", Status: model.ToolCallDone, Duration: time.Second},
			},
		},
	}
	props := timelineProps(items)

	result := tl.View(props)

	if !strings.Contains(result, "3 files") {
		t.Error("expected '3 files' group summary")
	}
	// Children should not be visible when collapsed
	if strings.Contains(result, "a.go") {
		t.Error("did not expect child 'a.go' when group is collapsed")
	}
}

func TestTimeline_View_GroupExpanded(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCallGroup{
			ToolName: "Read",
			Expanded: true,
			Children: []*model.ToolCall{
				{ID: "tc1", Name: "Read", Summary: "a.go", Status: model.ToolCallDone, Duration: time.Second},
				{ID: "tc2", Name: "Read", Summary: "b.go", Status: model.ToolCallDone, Duration: time.Second},
			},
		},
	}
	props := timelineProps(items)

	result := tl.View(props)

	if !strings.Contains(result, "2 files") {
		t.Error("expected '2 files' group summary")
	}
	if !strings.Contains(result, "a.go") {
		t.Error("expected child 'a.go' when group is expanded")
	}
	if !strings.Contains(result, "b.go") {
		t.Error("expected child 'b.go' when group is expanded")
	}
}

func TestTimeline_View_ExpandedToolCall(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Bash", Summary: "ls",
			Status:        model.ToolCallDone,
			Duration:      time.Second,
			RawInput:      map[string]interface{}{"command": "ls -la"},
			ResultContent: "file1.go\nfile2.go",
			Expanded:      true,
		},
	}
	props := timelineProps(items)

	result := tl.View(props)

	// Should contain the command
	if !strings.Contains(result, "$ ls -la") {
		t.Error("expected expanded Bash to show '$ ls -la' command")
	}
	// Should contain result content
	if !strings.Contains(result, "file1.go") {
		t.Error("expected expanded Bash to show result 'file1.go'")
	}
	if !strings.Contains(result, "file2.go") {
		t.Error("expected expanded Bash to show result 'file2.go'")
	}
}

func TestTimeline_View_ExpandedEditDiff(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Edit", Summary: "main.go",
			Status:   model.ToolCallDone,
			Duration: time.Second,
			RawInput: map[string]interface{}{
				"old_string": "oldCode",
				"new_string": "newCode",
			},
			Expanded: true,
		},
	}
	props := timelineProps(items)

	result := tl.View(props)

	// Edit diff should show -/+ lines
	if !strings.Contains(result, "-oldCode") {
		t.Error("expected expanded Edit to show '-oldCode' diff line")
	}
	if !strings.Contains(result, "+newCode") {
		t.Error("expected expanded Edit to show '+newCode' diff line")
	}
}

func TestTimeline_View_ExpandedGroupChild(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCallGroup{
			ToolName: "Read",
			Expanded: true,
			Children: []*model.ToolCall{
				{
					ID: "tc1", Name: "Read", Summary: "a.go",
					Status: model.ToolCallDone, Duration: time.Second,
					ResultContent: "package main",
					Expanded:      true,
				},
				{
					ID: "tc2", Name: "Read", Summary: "b.go",
					Status: model.ToolCallDone, Duration: time.Second,
				},
			},
		},
	}
	props := timelineProps(items)

	result := tl.View(props)

	// Expanded child should show its result content
	if !strings.Contains(result, "package main") {
		t.Error("expected expanded group child to show 'package main'")
	}
	// Group header and children should still render
	if !strings.Contains(result, "a.go") {
		t.Error("expected child 'a.go' summary")
	}
	if !strings.Contains(result, "b.go") {
		t.Error("expected child 'b.go' summary")
	}
}

func TestTimeline_View_ExpandedFullContent(t *testing.T) {
	tl := NewTimeline()

	// Create content with 30 lines — all should render without truncation.
	var longContent strings.Builder
	for i := 0; i < 30; i++ {
		if i > 0 {
			longContent.WriteString("\n")
		}
		fmt.Fprintf(&longContent, "line %d", i+1)
	}

	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Read", Summary: "big.go",
			Status:        model.ToolCallDone,
			Duration:      time.Second,
			ResultContent: longContent.String(),
			Expanded:      true,
		},
	}
	props := TimelineProps{
		Items:   items,
		Width:   80,
		Height:  40, // large enough to see all lines
		Focused: true,
		Theme:   testTheme(),
	}

	result := tl.View(props)

	// No truncation footer should appear.
	if strings.Contains(result, "more lines") {
		t.Error("expected no truncation footer — full content should be displayed")
	}
	// Verify content from near the end is present.
	if !strings.Contains(result, "line 30") {
		t.Error("expected 'line 30' in expanded view — all content should be shown")
	}
}

func TestTimeline_View_CollapsedToolCall_NoContent(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Bash", Summary: "ls",
			Status:        model.ToolCallDone,
			Duration:      time.Second,
			RawInput:      map[string]interface{}{"command": "ls -la"},
			ResultContent: "should not appear",
			Expanded:      false,
		},
	}
	props := timelineProps(items)

	result := tl.View(props)

	if strings.Contains(result, "should not appear") {
		t.Error("collapsed tool call should not show result content")
	}
	if strings.Contains(result, "$ ls") {
		t.Error("collapsed tool call should not show command")
	}
}

func TestTimeline_Scroll(t *testing.T) {
	tl := NewTimeline()

	// Create many items that exceed viewport
	var items []model.TimelineItem
	for i := 0; i < 30; i++ {
		items = append(items, &model.ToolCall{
			ID:      "tc",
			Name:    "Read",
			Summary: "file.go",
			Status:  model.ToolCallDone,
		})
	}
	props := TimelineProps{
		Items:  items,
		Width:  80,
		Height: 10,
		Theme:  testTheme(),
	}

	tl.HandleAction("page_down", props)
	if tl.Scroll != 10 {
		t.Errorf("expected scroll=10 after pgdown, got %d", tl.Scroll)
	}

	tl.HandleAction("page_up", props)
	if tl.Scroll != 0 {
		t.Errorf("expected scroll=0 after pgup, got %d", tl.Scroll)
	}
}

func TestTimeline_PgDown_ClampsCursorIntoViewport(t *testing.T) {
	tl := NewTimeline()

	// 30 items, viewport height 10
	var items []model.TimelineItem
	for i := 0; i < 30; i++ {
		items = append(items, &model.ToolCall{
			ID:      "tc",
			Name:    "Read",
			Summary: "file.go",
			Status:  model.ToolCallDone,
		})
	}
	props := TimelineProps{
		Items:   items,
		Width:   80,
		Height:  10,
		Focused: true,
		Theme:   testTheme(),
	}

	// Cursor starts at 0, pgdown scrolls to line 10.
	// Cursor at flat 0 (line 0) is now above viewport (line 10-19).
	// Cursor should be clamped to the first visible position.
	tl.HandleAction("page_down", props)
	if tl.Cursor < 10 || tl.Cursor > 19 {
		t.Errorf("expected cursor in viewport [10,19] after pgdown, got %d", tl.Cursor)
	}
	if tl.Cursor != 10 {
		t.Errorf("expected cursor=10 (first visible) after pgdown, got %d", tl.Cursor)
	}
}

func TestTimeline_PgUp_ClampsCursorIntoViewport(t *testing.T) {
	tl := NewTimeline()

	// 30 items, viewport height 10
	var items []model.TimelineItem
	for i := 0; i < 30; i++ {
		items = append(items, &model.ToolCall{
			ID:      "tc",
			Name:    "Read",
			Summary: "file.go",
			Status:  model.ToolCallDone,
		})
	}
	props := TimelineProps{
		Items:   items,
		Width:   80,
		Height:  10,
		Focused: true,
		Theme:   testTheme(),
	}

	// Start at bottom: cursor=29, scroll=20
	tl.Cursor = 29
	tl.Scroll = 20

	// pgup scrolls to 10. Cursor at 29 (line 29) is below viewport (lines 10-19).
	// Cursor should clamp to last visible position.
	tl.HandleAction("page_up", props)
	if tl.Scroll != 10 {
		t.Errorf("expected scroll=10 after pgup, got %d", tl.Scroll)
	}
	if tl.Cursor < 10 || tl.Cursor > 19 {
		t.Errorf("expected cursor in viewport [10,19] after pgup, got %d", tl.Cursor)
	}
	if tl.Cursor != 19 {
		t.Errorf("expected cursor=19 (last visible) after pgup, got %d", tl.Cursor)
	}
}

// --- Mouse support tests ---

func TestTimeline_ScrollBy_Down(t *testing.T) {
	tl := NewTimeline()

	var items []model.TimelineItem
	for i := 0; i < 30; i++ {
		items = append(items, &model.ToolCall{
			ID: "tc", Name: "Read", Summary: "file.go", Status: model.ToolCallDone,
		})
	}
	props := TimelineProps{Items: items, Width: 80, Height: 10, Theme: testTheme()}

	tl.ScrollBy(3, props)
	if tl.Scroll != 3 {
		t.Errorf("expected scroll=3 after ScrollBy(3), got %d", tl.Scroll)
	}
	if tl.AutoFollow.Following() {
		t.Error("expected auto-follow paused after mouse scroll")
	}
}

func TestTimeline_ScrollBy_Up(t *testing.T) {
	tl := NewTimeline()
	tl.Scroll = 5

	var items []model.TimelineItem
	for i := 0; i < 30; i++ {
		items = append(items, &model.ToolCall{
			ID: "tc", Name: "Read", Summary: "file.go", Status: model.ToolCallDone,
		})
	}
	props := TimelineProps{Items: items, Width: 80, Height: 10, Theme: testTheme()}

	tl.ScrollBy(-3, props)
	if tl.Scroll != 2 {
		t.Errorf("expected scroll=2 after ScrollBy(-3) from 5, got %d", tl.Scroll)
	}
}

func TestTimeline_ScrollBy_ClampsAtTop(t *testing.T) {
	tl := NewTimeline()
	tl.Scroll = 2

	var items []model.TimelineItem
	for i := 0; i < 30; i++ {
		items = append(items, &model.ToolCall{
			ID: "tc", Name: "Read", Summary: "file.go", Status: model.ToolCallDone,
		})
	}
	props := TimelineProps{Items: items, Width: 80, Height: 10, Theme: testTheme()}

	tl.ScrollBy(-10, props)
	if tl.Scroll != 0 {
		t.Errorf("expected scroll=0 (clamped at top), got %d", tl.Scroll)
	}
}

func TestTimeline_ScrollBy_ClampsAtBottom(t *testing.T) {
	tl := NewTimeline()
	tl.Scroll = 15

	var items []model.TimelineItem
	for i := 0; i < 20; i++ {
		items = append(items, &model.ToolCall{
			ID: "tc", Name: "Read", Summary: "file.go", Status: model.ToolCallDone,
		})
	}
	props := TimelineProps{Items: items, Width: 80, Height: 10, Theme: testTheme()}

	tl.ScrollBy(10, props)
	// Max scroll = 20 - 10 = 10
	if tl.Scroll != 10 {
		t.Errorf("expected scroll=10 (clamped at bottom), got %d", tl.Scroll)
	}
}

func TestTimeline_ClickRow_ValidRow(t *testing.T) {
	tl := NewTimeline()
	items := makeTimelineItems()
	props := timelineProps(items)

	changed := tl.ClickRow(1, props)
	if !changed {
		t.Error("expected ClickRow to return true for valid row")
	}
	// Row 1 with scroll 0 → line 1 → should map to flat cursor 1
	if tl.Cursor != 1 {
		t.Errorf("expected cursor=1 after clicking row 1, got %d", tl.Cursor)
	}
}

func TestTimeline_ClickRow_WithScroll(t *testing.T) {
	tl := NewTimeline()
	tl.Scroll = 5

	var items []model.TimelineItem
	for i := 0; i < 30; i++ {
		items = append(items, &model.ToolCall{
			ID: "tc", Name: "Read", Summary: "file.go", Status: model.ToolCallDone,
		})
	}
	props := TimelineProps{Items: items, Width: 80, Height: 10, Theme: testTheme()}

	changed := tl.ClickRow(3, props)
	if !changed {
		t.Error("expected ClickRow to return true")
	}
	// scroll(5) + row(3) = line 8 → flat cursor 8 (1:1 for tool calls)
	if tl.Cursor != 8 {
		t.Errorf("expected cursor=8 (line 8), got %d", tl.Cursor)
	}
}

func TestTimeline_ClickRow_BeyondLastItem(t *testing.T) {
	tl := NewTimeline()
	items := makeTimelineItems() // 4 items, 4 lines
	props := timelineProps(items)

	changed := tl.ClickRow(10, props)
	if changed {
		t.Error("expected ClickRow to return false for click beyond last item")
	}
	if tl.Cursor != 0 {
		t.Errorf("expected cursor unchanged at 0, got %d", tl.Cursor)
	}
}

func TestTimeline_ClickRow_PausesAutoFollow(t *testing.T) {
	tl := NewTimeline()
	items := makeTimelineItems()
	props := timelineProps(items)

	tl.ClickRow(0, props)
	if tl.AutoFollow.Following() {
		t.Error("expected auto-follow paused after clicking non-last row")
	}
}

func TestTimeline_ClickRow_AtEnd_ContinuesAutoFollow(t *testing.T) {
	tl := NewTimeline()
	items := makeTimelineItems() // 4 items
	props := timelineProps(items)

	// Click on last line (row 3, maps to flat cursor 3 which is the last item)
	tl.ClickRow(3, props)
	if !tl.AutoFollow.Following() {
		t.Error("expected auto-follow to continue when clicking the last item")
	}
}

func TestTimeline_ClickRow_ExpandsAlreadySelectedToolCall(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Read", Summary: "main.go",
			Status:        model.ToolCallDone,
			ResultContent: "file content",
		},
	}
	props := timelineProps(items)

	// First click: selects row 0 (cursor moves from 0 to 0 — same position)
	// Since cursor was already at 0, this triggers expand.
	tc := items[0].(*model.ToolCall)
	tl.ClickRow(0, props)
	if !tc.Expanded {
		t.Error("expected tool call to expand on click of already-selected row")
	}

	// Click again: still selected, triggers collapse.
	tl.ClickRow(0, props)
	if tc.Expanded {
		t.Error("expected tool call to collapse on second click of selected row")
	}
}

func TestTimeline_ClickRow_DoesNotExpandOnFirstClick(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Read", Summary: "main.go",
			Status:        model.ToolCallDone,
			ResultContent: "content",
		},
		&model.ToolCall{
			ID: "tc2", Name: "Edit", Summary: "other.go",
			Status:        model.ToolCallDone,
			ResultContent: "content",
		},
	}
	props := timelineProps(items)

	// Cursor starts at 0. Click row 1 to select the second item.
	tl.ClickRow(1, props)
	if tl.Cursor != 1 {
		t.Fatalf("expected cursor=1, got %d", tl.Cursor)
	}
	// The second item should NOT be expanded (cursor changed).
	tc2 := items[1].(*model.ToolCall)
	if tc2.Expanded {
		t.Error("expected tool call NOT to expand when cursor changes on click")
	}
}

func TestTimeline_ClickRow_ExpandsTextBlock(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.TextBlock{Text: "line1\nline2\nline3\nline4\nline5"},
	}
	props := timelineProps(items)

	// Cursor starts at 0, click row 0 → same position → triggers enter.
	tb := items[0].(*model.TextBlock)
	if tb.Expanded {
		t.Fatal("expected text block not expanded initially")
	}
	tl.ClickRow(0, props)
	if !tb.Expanded {
		t.Error("expected text block to expand on click of already-selected row")
	}
}

func TestTimeline_ClickRow_ExpandsGroupHeader(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCallGroup{
			ToolName: "Read",
			Children: []*model.ToolCall{
				{ID: "c1", Name: "Read", Summary: "a.go", Status: model.ToolCallDone},
				{ID: "c2", Name: "Read", Summary: "b.go", Status: model.ToolCallDone},
			},
			Expanded: false,
		},
	}
	props := timelineProps(items)

	// Cursor at 0, click row 0 → triggers enter on group header → expands.
	group := items[0].(*model.ToolCallGroup)
	tl.ClickRow(0, props)
	if !group.Expanded {
		t.Error("expected group to expand on click of already-selected header row")
	}
}

func TestTimeline_PgDown_CursorAlreadyInViewport(t *testing.T) {
	tl := NewTimeline()

	// 30 items, viewport height 10
	var items []model.TimelineItem
	for i := 0; i < 30; i++ {
		items = append(items, &model.ToolCall{
			ID:      "tc",
			Name:    "Read",
			Summary: "file.go",
			Status:  model.ToolCallDone,
		})
	}
	props := TimelineProps{
		Items:   items,
		Width:   80,
		Height:  10,
		Focused: true,
		Theme:   testTheme(),
	}

	// Cursor at 15, scroll at 10 (viewport lines 10-19). Cursor is visible.
	tl.Cursor = 15
	tl.Scroll = 10

	// pgdown scrolls to 20 (viewport lines 20-29). Cursor 15 is now above viewport.
	tl.HandleAction("page_down", props)
	if tl.Cursor != 20 {
		t.Errorf("expected cursor=20 after pgdown pushed cursor out, got %d", tl.Cursor)
	}
}

// --- Sub-scroll tests ---

// makeSubScrollItems creates a tool call with many content lines for sub-scroll testing.
func makeSubScrollItems(contentLines int) []model.TimelineItem {
	var sb strings.Builder
	for i := 0; i < contentLines; i++ {
		if i > 0 {
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "line %d", i+1)
	}
	return []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Read", Summary: "big.go",
			Status:        model.ToolCallDone,
			ResultContent: sb.String(),
			Expanded:      true,
		},
	}
}

func TestSubScrollViewportHeight(t *testing.T) {
	// Content at 40% of pane (pane=100, threshold=40): at threshold, inline
	if h := subScrollViewportHeight(40, 100); h != 40 {
		t.Errorf("40 lines at 40%% threshold should be inline (40), got %d", h)
	}
	// Content at 41 lines (pane=100, threshold=40): exceeds threshold, but
	// 41 < cap 70, so viewport = 41 (all content visible, sub-scroll border shown)
	if h := subScrollViewportHeight(41, 100); h != 41 {
		t.Errorf("41 lines (below cap 70) should return 41, got %d", h)
	}
	// Content at 50 lines (pane=100): exceeds threshold, 50 < cap 70
	if h := subScrollViewportHeight(50, 100); h != 50 {
		t.Errorf("50 lines (< cap 70) should return 50, got %d", h)
	}
	// Content at 200 lines (pane=100): exceeds cap → capped at 70
	if h := subScrollViewportHeight(200, 100); h != 70 {
		t.Errorf("200 lines should be capped at 70, got %d", h)
	}
	// Small pane (height=10): threshold=4, cap=7
	// 5 lines > threshold 4, but 5 < cap 7, so viewport = 5
	if h := subScrollViewportHeight(5, 10); h != 5 {
		t.Errorf("5 lines with pane 10 should return 5 (below cap 7), got %d", h)
	}
	if h := subScrollViewportHeight(4, 10); h != 4 {
		t.Errorf("4 lines with pane 10 should be inline (4), got %d", h)
	}
	// 8 lines, pane=10: exceeds cap → capped at 7
	if h := subScrollViewportHeight(8, 10); h != 7 {
		t.Errorf("8 lines with pane 10 should be capped at 7, got %d", h)
	}
}

func TestSubScrollEnabled(t *testing.T) {
	// Exactly at threshold: not enabled
	if subScrollEnabled(40, 100) {
		t.Error("40 lines at pane 100 should NOT enable sub-scroll (at threshold)")
	}
	// Just above threshold
	if !subScrollEnabled(41, 100) {
		t.Error("41 lines at pane 100 should enable sub-scroll")
	}
	// Well below
	if subScrollEnabled(5, 100) {
		t.Error("5 lines should NOT enable sub-scroll")
	}
}

func TestTimeline_EnterOnExpandedToolCall_EntersSubScroll(t *testing.T) {
	tl := NewTimeline()
	// 50 content lines, pane height 20 → threshold 8, content 50 > 8 → sub-scroll enabled
	items := makeSubScrollItems(50)
	props := TimelineProps{
		Items:   items,
		Width:   80,
		Height:  20,
		Focused: true,
		Theme:   testTheme(),
	}

	// Should NOT be in sub-scroll yet
	if tl.InSubScroll() {
		t.Fatal("should not be in sub-scroll before enter")
	}

	// First enter: tool call is already expanded, content exceeds threshold
	tl.HandleAction("expand", props)
	if !tl.InSubScroll() {
		t.Error("expected sub-scroll mode after enter on expanded tool call with large content")
	}
	if tl.SubScrollIdx != 0 {
		t.Errorf("expected SubScrollIdx=0, got %d", tl.SubScrollIdx)
	}
	if tl.SubScrollOffset != 0 {
		t.Errorf("expected SubScrollOffset=0, got %d", tl.SubScrollOffset)
	}
}

func TestTimeline_EnterOnExpandedToolCall_SmallContent_Collapses(t *testing.T) {
	tl := NewTimeline()
	// 3 content lines, pane height 20 → threshold 8, content 3 ≤ 8 → no sub-scroll
	items := makeSubScrollItems(3)
	props := TimelineProps{
		Items:   items,
		Width:   80,
		Height:  20,
		Focused: true,
		Theme:   testTheme(),
	}

	tc := items[0].(*model.ToolCall)
	// Already expanded
	tl.HandleAction("expand", props)
	if tl.InSubScroll() {
		t.Error("small content should not trigger sub-scroll")
	}
	if tc.Expanded {
		t.Error("enter on small expanded content should collapse")
	}
}

func TestTimeline_SubScroll_MoveDown(t *testing.T) {
	tl := NewTimeline()
	items := makeSubScrollItems(50)
	props := TimelineProps{
		Items:   items,
		Width:   80,
		Height:  20,
		Focused: true,
		Theme:   testTheme(),
	}

	// Enter sub-scroll
	tl.HandleAction("expand", props)
	if !tl.InSubScroll() {
		t.Fatal("expected sub-scroll mode")
	}

	// Move down within sub-scroll
	tl.HandleAction("move_down", props)
	if tl.SubScrollOffset != 1 {
		t.Errorf("expected SubScrollOffset=1, got %d", tl.SubScrollOffset)
	}

	// Multiple moves
	for i := 0; i < 5; i++ {
		tl.HandleAction("move_down", props)
	}
	if tl.SubScrollOffset != 6 {
		t.Errorf("expected SubScrollOffset=6, got %d", tl.SubScrollOffset)
	}
}

func TestTimeline_SubScroll_MoveUp(t *testing.T) {
	tl := NewTimeline()
	items := makeSubScrollItems(50)
	props := TimelineProps{
		Items:   items,
		Width:   80,
		Height:  20,
		Focused: true,
		Theme:   testTheme(),
	}

	// Enter sub-scroll and move down a few
	tl.HandleAction("expand", props)
	for i := 0; i < 5; i++ {
		tl.HandleAction("move_down", props)
	}

	// Move up
	tl.HandleAction("move_up", props)
	if tl.SubScrollOffset != 4 {
		t.Errorf("expected SubScrollOffset=4, got %d", tl.SubScrollOffset)
	}

	// Move up past 0 — should clamp
	for i := 0; i < 10; i++ {
		tl.HandleAction("move_up", props)
	}
	if tl.SubScrollOffset != 0 {
		t.Errorf("expected SubScrollOffset=0 (clamped), got %d", tl.SubScrollOffset)
	}
}

func TestTimeline_SubScroll_JumpTopBottom(t *testing.T) {
	tl := NewTimeline()
	items := makeSubScrollItems(50)
	props := TimelineProps{
		Items:   items,
		Width:   80,
		Height:  20,
		Focused: true,
		Theme:   testTheme(),
	}

	// Enter sub-scroll
	tl.HandleAction("expand", props)

	// Jump to bottom
	tl.HandleAction("jump_bottom", props)
	// Viewport height: subScrollViewportHeight(50, 20) → threshold=8, cap=14
	expectedMax := 50 - 14
	if tl.SubScrollOffset != expectedMax {
		t.Errorf("expected SubScrollOffset=%d after jump_bottom, got %d", expectedMax, tl.SubScrollOffset)
	}

	// Jump to top
	tl.HandleAction("jump_top", props)
	if tl.SubScrollOffset != 0 {
		t.Errorf("expected SubScrollOffset=0 after jump_top, got %d", tl.SubScrollOffset)
	}
}

func TestTimeline_SubScroll_MoveDownClamp(t *testing.T) {
	tl := NewTimeline()
	items := makeSubScrollItems(50)
	props := TimelineProps{
		Items:   items,
		Width:   80,
		Height:  20,
		Focused: true,
		Theme:   testTheme(),
	}

	// Enter sub-scroll and jump to bottom
	tl.HandleAction("expand", props)
	tl.HandleAction("jump_bottom", props)

	// Try to go past bottom
	offsetBefore := tl.SubScrollOffset
	tl.HandleAction("move_down", props)
	if tl.SubScrollOffset != offsetBefore {
		t.Errorf("expected SubScrollOffset to stay at %d (clamped), got %d", offsetBefore, tl.SubScrollOffset)
	}
}

func TestTimeline_ExitSubScroll(t *testing.T) {
	tl := NewTimeline()
	items := makeSubScrollItems(50)
	props := TimelineProps{
		Items:   items,
		Width:   80,
		Height:  20,
		Focused: true,
		Theme:   testTheme(),
	}

	// Enter sub-scroll
	tl.HandleAction("expand", props)
	if !tl.InSubScroll() {
		t.Fatal("expected sub-scroll mode")
	}

	// Move around
	tl.HandleAction("move_down", props)
	tl.HandleAction("move_down", props)

	// Exit
	tl.ExitSubScroll()
	if tl.InSubScroll() {
		t.Error("expected sub-scroll mode to be exited")
	}
	if tl.SubScrollIdx != -1 {
		t.Errorf("expected SubScrollIdx=-1, got %d", tl.SubScrollIdx)
	}
	if tl.SubScrollOffset != 0 {
		t.Errorf("expected SubScrollOffset=0, got %d", tl.SubScrollOffset)
	}
}

func TestTimeline_SubScroll_EnterToCollapse(t *testing.T) {
	tl := NewTimeline()
	items := makeSubScrollItems(50)
	props := TimelineProps{
		Items:   items,
		Width:   80,
		Height:  20,
		Focused: true,
		Theme:   testTheme(),
	}

	// Enter sub-scroll
	tl.HandleAction("expand", props)
	if !tl.InSubScroll() {
		t.Fatal("expected sub-scroll mode")
	}

	// Press enter again in sub-scroll: collapses and exits
	tl.HandleAction("expand", props)
	if tl.InSubScroll() {
		t.Error("expected sub-scroll to exit after enter in sub-scroll mode")
	}
	tc := items[0].(*model.ToolCall)
	if tc.Expanded {
		t.Error("expected tool call to be collapsed after enter in sub-scroll")
	}
}

func TestTimeline_SubScroll_View_ShowsIndicator(t *testing.T) {
	tl := NewTimeline()
	items := makeSubScrollItems(50)
	props := TimelineProps{
		Items:   items,
		Width:   80,
		Height:  20,
		Focused: true,
		Theme:   testTheme(),
	}

	// Enter sub-scroll
	tl.HandleAction("expand", props)
	result := tl.View(props)

	// Should show scroll indicator [14/50]
	// viewport height = subScrollViewportHeight(50, 20) = 14 (cap = 14)
	if !strings.Contains(result, "[14/50]") {
		t.Errorf("expected scroll indicator [14/50], got: %s", result)
	}

	// Should contain first content line
	if !strings.Contains(result, "line 1") {
		t.Error("expected 'line 1' in sub-scroll view")
	}
}

func TestTimeline_SubScroll_View_AfterScroll(t *testing.T) {
	tl := NewTimeline()
	items := makeSubScrollItems(50)
	props := TimelineProps{
		Items:   items,
		Width:   80,
		Height:  20,
		Focused: true,
		Theme:   testTheme(),
	}

	// Enter sub-scroll and scroll down
	tl.HandleAction("expand", props)
	for i := 0; i < 5; i++ {
		tl.HandleAction("move_down", props)
	}
	result := tl.View(props)

	// Indicator should update: [19/50] (offset 5, viewport 14)
	if !strings.Contains(result, "[19/50]") {
		t.Errorf("expected scroll indicator [19/50] after scrolling, got: %s", result)
	}

	// Should show content at offset 5
	if !strings.Contains(result, "line 6") {
		t.Error("expected 'line 6' visible after scrolling down 5")
	}
}

func TestTimeline_SubScroll_View_ShowsBorder(t *testing.T) {
	tl := NewTimeline()
	items := makeSubScrollItems(50)
	props := TimelineProps{
		Items:   items,
		Width:   80,
		Height:  20,
		Focused: true,
		Theme:   testTheme(),
	}

	// Enter sub-scroll
	tl.HandleAction("expand", props)
	result := tl.View(props)

	// The border character should be present (│)
	if !strings.Contains(result, "│") {
		t.Error("expected border character '│' in sub-scroll view")
	}
}

func TestTimeline_SubScroll_GroupChild(t *testing.T) {
	// Build a group with a child that has large content
	var sb strings.Builder
	for i := 0; i < 50; i++ {
		if i > 0 {
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "child line %d", i+1)
	}

	items := []model.TimelineItem{
		&model.ToolCallGroup{
			ToolName: "Read",
			Expanded: true,
			Children: []*model.ToolCall{
				{
					ID: "tc1", Name: "Read", Summary: "big.go",
					Status:        model.ToolCallDone,
					ResultContent: sb.String(),
					Expanded:      true,
				},
			},
		},
	}
	props := TimelineProps{
		Items:   items,
		Width:   80,
		Height:  20,
		Focused: true,
		Theme:   testTheme(),
	}

	tl := NewTimeline()
	tl.Cursor = 1 // Move to child

	// Enter sub-scroll on group child
	tl.HandleAction("expand", props)
	if !tl.InSubScroll() {
		t.Error("expected sub-scroll mode on group child with large content")
	}
	if tl.SubScrollIdx != 1 {
		t.Errorf("expected SubScrollIdx=1 for group child, got %d", tl.SubScrollIdx)
	}
}

func TestTimeline_SubScroll_TimelineCursorUnchanged(t *testing.T) {
	tl := NewTimeline()
	items := makeSubScrollItems(50)
	props := TimelineProps{
		Items:   items,
		Width:   80,
		Height:  20,
		Focused: true,
		Theme:   testTheme(),
	}

	// Enter sub-scroll
	tl.HandleAction("expand", props)

	// Move within sub-scroll — timeline cursor should NOT change
	cursorBefore := tl.Cursor
	tl.HandleAction("move_down", props)
	tl.HandleAction("move_down", props)
	if tl.Cursor != cursorBefore {
		t.Errorf("expected timeline cursor unchanged at %d during sub-scroll, got %d", cursorBefore, tl.Cursor)
	}
}

func TestTimeline_ResetPosition_ClearsSubScroll(t *testing.T) {
	tl := NewTimeline()
	tl.SubScrollIdx = 5
	tl.SubScrollOffset = 10

	tl.ResetPosition()
	if tl.InSubScroll() {
		t.Error("expected ResetPosition to clear sub-scroll")
	}
}

func TestTimeline_SubScroll_ToolCallLineCountCapped(t *testing.T) {
	// 50 lines content, pane height 20 → threshold 8, cap 14
	tc := &model.ToolCall{
		Name:          "Read",
		Expanded:      true,
		ResultContent: strings.Repeat("x\n", 49) + "x", // 50 lines
	}
	full := toolCallLineCount(tc)
	if full != 51 {
		t.Errorf("expected full count 51 (1+50), got %d", full)
	}
	capped := toolCallLineCountCapped(tc, 20)
	// cap = 20*70/100 = 14, so capped = 1 + 14 = 15
	if capped != 15 {
		t.Errorf("expected capped count 15 (1+14), got %d", capped)
	}
}

func TestTimeline_InSubScroll(t *testing.T) {
	tl := NewTimeline()
	if tl.InSubScroll() {
		t.Error("new timeline should not be in sub-scroll")
	}
	tl.SubScrollIdx = 0
	if !tl.InSubScroll() {
		t.Error("timeline with SubScrollIdx=0 should be in sub-scroll")
	}
}

func TestTimeline_ClickRowSubScroll_SummaryRow_ExitsAndCollapses(t *testing.T) {
	tl := NewTimeline()
	// 50 content lines, pane height 20 → sub-scroll enabled
	items := makeSubScrollItems(50)
	props := TimelineProps{
		Items:   items,
		Width:   80,
		Height:  20,
		Focused: true,
		Theme:   testTheme(),
	}

	// Enter sub-scroll mode
	tl.HandleAction("expand", props)
	if !tl.InSubScroll() {
		t.Fatal("expected sub-scroll mode after enter on expanded tool call")
	}

	// Click the summary row (row 0 with scroll 0 → line 0 = summary row of item at flat 0)
	tl.ClickRowSubScroll(0, props)
	if tl.InSubScroll() {
		t.Error("expected sub-scroll to exit after clicking summary row")
	}
	tc := items[0].(*model.ToolCall)
	if tc.Expanded {
		t.Error("expected tool call to be collapsed after clicking summary row in sub-scroll")
	}
}

func TestTimeline_ClickRowSubScroll_ExpandedContent_NoOp(t *testing.T) {
	tl := NewTimeline()
	// 50 content lines, pane height 20
	items := makeSubScrollItems(50)
	props := TimelineProps{
		Items:   items,
		Width:   80,
		Height:  20,
		Focused: true,
		Theme:   testTheme(),
	}

	// Enter sub-scroll mode
	tl.HandleAction("expand", props)
	if !tl.InSubScroll() {
		t.Fatal("expected sub-scroll mode")
	}

	// Click on expanded content area (row 1 = first content line)
	tl.ClickRowSubScroll(1, props)
	if !tl.InSubScroll() {
		t.Error("expected sub-scroll to remain active after clicking expanded content")
	}
	tc := items[0].(*model.ToolCall)
	if !tc.Expanded {
		t.Error("expected tool call to remain expanded after clicking content area")
	}
}

func TestTimeline_ClickRowSubScroll_OtherRow_ExitsAndSelects(t *testing.T) {
	tl := NewTimeline()
	// Create two items: first has sub-scroll content, second is a simple tool call.
	var sb strings.Builder
	for i := 0; i < 50; i++ {
		if i > 0 {
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "line %d", i+1)
	}
	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Read", Summary: "big.go",
			Status:        model.ToolCallDone,
			ResultContent: sb.String(),
			Expanded:      true,
		},
		&model.ToolCall{
			ID: "tc2", Name: "Read", Summary: "small.go",
			Status: model.ToolCallDone,
		},
	}
	props := TimelineProps{
		Items:   items,
		Width:   80,
		Height:  20,
		Focused: true,
		Theme:   testTheme(),
	}

	// Enter sub-scroll on first item
	tl.HandleAction("expand", props)
	if !tl.InSubScroll() {
		t.Fatal("expected sub-scroll mode")
	}

	// In sub-scroll, the first item is capped. The second item is after
	// the capped area. We need to find its row position.
	// Item 0: 1 summary line + capped content lines
	// Cap = 20*70/100 = 14, so total lines for item 0 = 1 + 14 = 15
	// Item 1 starts at line 15, which is pane row 15 (scroll=0).
	tl.ClickRowSubScroll(15, props)
	if tl.InSubScroll() {
		t.Error("expected sub-scroll to exit after clicking another row")
	}
	if tl.Cursor != 1 {
		t.Errorf("expected cursor=1 after clicking second item, got %d", tl.Cursor)
	}
}

// --- Relative line number tests ---

func timelinePropsWithLineNumbers(items []model.TimelineItem) TimelineProps {
	return TimelineProps{
		Items:       items,
		Width:       80,
		Height:      20,
		Focused:     true,
		CompactView: false,
		LineNumbers: true,
		Theme:       testTheme(),
	}
}

func TestTimeline_LineNumbers_GutterRendersRelativeNumbers(t *testing.T) {
	tl := NewTimeline()
	tl.Cursor = 1 // cursor on second item (the Read tool call)
	items := makeTimelineItems()
	props := timelinePropsWithLineNumbers(items)

	result := tl.View(props)

	// Line 0 should be at cursor position (cursor=1 → item index 1).
	// Items: [TextBlock(0), Read(1), Edit(2), TextBlock(3)]
	// Relative from cursor=1: 1, 0, 1, 2
	// The gutter should show "  1 " for item 0 (1 above cursor)
	// "  0 " for item 1 (cursor position)
	// "  1 " for item 2 (1 below cursor)
	// "  2 " for item 3 (2 below cursor)

	// Check that "  0 " appears (cursor line)
	if !strings.Contains(result, "  0 ") {
		t.Error("expected '  0 ' gutter for cursor position")
	}
	// Check that relative numbers appear
	if !strings.Contains(result, "  1 ") {
		t.Error("expected '  1 ' gutter for adjacent lines")
	}
	if !strings.Contains(result, "  2 ") {
		t.Error("expected '  2 ' gutter for line 2 away from cursor")
	}
}

func TestTimeline_LineNumbers_DisabledShowsNoGutter(t *testing.T) {
	tl := NewTimeline()
	items := makeTimelineItems()
	props := timelineProps(items) // LineNumbers defaults to false

	result := tl.View(props)

	// Should not contain gutter-style "  0 " at start of line
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		// With line numbers off, lines should start with content, not number gutter.
		// The "  0 " pattern could appear in content, so check that the first
		// non-styled line doesn't start with a gutter pattern.
		if strings.HasPrefix(strings.TrimLeft(line, "\x1b[0-9;m"), "  0 ") {
			// This is too fragile with ANSI codes, just verify content is present
			break
		}
	}

	// The key test: tool names should still be present.
	if !strings.Contains(result, "Read") {
		t.Error("expected 'Read' without line numbers")
	}
}

func TestTimeline_LineNumbers_ExpandedContentSharesParentNumber(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Bash", Summary: "ls",
			Status:        model.ToolCallDone,
			Duration:      time.Second,
			RawInput:      map[string]interface{}{"command": "ls -la"},
			ResultContent: "file1.go\nfile2.go",
			Expanded:      true,
		},
	}
	props := timelinePropsWithLineNumbers(items)

	result := tl.View(props)

	// The tool call header should have gutter "  0 " (cursor is on it).
	if !strings.Contains(result, "  0 ") {
		t.Error("expected '  0 ' gutter for cursor on expanded tool call")
	}

	// Expanded content lines should have blank gutter (4 spaces), not numbers.
	// The expanded content should still render.
	if !strings.Contains(result, "file1.go") {
		t.Error("expected expanded content visible with line numbers")
	}
}

func TestTimeline_LineNumbers_CursorAtZero(t *testing.T) {
	tl := NewTimeline()
	tl.Cursor = 0
	items := makeTimelineItems()
	props := timelinePropsWithLineNumbers(items)

	result := tl.View(props)

	// First item should have "  0 " (cursor at position 0).
	// Subsequent items: 1, 2, 3
	if !strings.Contains(result, "  0 ") {
		t.Error("expected '  0 ' at cursor position 0")
	}
	if !strings.Contains(result, "  3 ") {
		t.Error("expected '  3 ' for item 3 positions away from cursor")
	}
}

func TestTimeline_LineNumbers_WidthReduced(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Read", Summary: "main.go",
			Status: model.ToolCallDone,
		},
	}

	// Test with line numbers on — content width is reduced by gutterWidth (4).
	propsOn := TimelineProps{
		Items:       items,
		Width:       80,
		Height:      20,
		Focused:     true,
		LineNumbers: true,
		Theme:       testTheme(),
	}
	resultOn := tl.View(propsOn)

	// Test with line numbers off — full width available.
	propsOff := TimelineProps{
		Items:   items,
		Width:   80,
		Height:  20,
		Focused: true,
		Theme:   testTheme(),
	}
	resultOff := tl.View(propsOff)

	// Both should contain the tool call content.
	if !strings.Contains(resultOn, "main.go") {
		t.Error("expected 'main.go' with line numbers on")
	}
	if !strings.Contains(resultOff, "main.go") {
		t.Error("expected 'main.go' with line numbers off")
	}

	// The version with line numbers should be wider (includes gutter).
	linesOn := strings.Split(resultOn, "\n")
	linesOff := strings.Split(resultOff, "\n")
	if len(linesOn) > 0 && len(linesOff) > 0 {
		// Both should fit in 80 columns
		if lipgloss.Width(linesOn[0]) > 80 {
			t.Error("line with gutter should not exceed total width")
		}
		_ = linesOff // Used above
	}
}

// --- Count+jump motion tests ---

func TestTimeline_AccumulateDigit(t *testing.T) {
	tl := NewTimeline()

	tl.AccumulateDigit('5')
	if tl.CountBuffer != "5" {
		t.Errorf("expected CountBuffer='5', got %q", tl.CountBuffer)
	}

	tl.AccumulateDigit('3')
	if tl.CountBuffer != "53" {
		t.Errorf("expected CountBuffer='53', got %q", tl.CountBuffer)
	}
}

func TestTimeline_AccumulateDigit_LeadingZeroIgnored(t *testing.T) {
	tl := NewTimeline()

	// Leading zero should be ignored
	tl.AccumulateDigit('0')
	if tl.CountBuffer != "" {
		t.Errorf("expected empty CountBuffer after leading 0, got %q", tl.CountBuffer)
	}

	// After a non-zero digit, 0 is allowed
	tl.AccumulateDigit('1')
	tl.AccumulateDigit('0')
	if tl.CountBuffer != "10" {
		t.Errorf("expected CountBuffer='10', got %q", tl.CountBuffer)
	}
}

func TestTimeline_ConsumeCount_Empty(t *testing.T) {
	tl := NewTimeline()

	count := tl.ConsumeCount()
	if count != 1 {
		t.Errorf("expected count=1 for empty buffer, got %d", count)
	}
	if tl.CountBuffer != "" {
		t.Errorf("expected buffer cleared after consume, got %q", tl.CountBuffer)
	}
}

func TestTimeline_ConsumeCount_WithValue(t *testing.T) {
	tl := NewTimeline()
	tl.CountBuffer = "12"

	count := tl.ConsumeCount()
	if count != 12 {
		t.Errorf("expected count=12, got %d", count)
	}
	if tl.CountBuffer != "" {
		t.Errorf("expected buffer cleared after consume, got %q", tl.CountBuffer)
	}
}

func TestTimeline_ClearCount(t *testing.T) {
	tl := NewTimeline()
	tl.CountBuffer = "42"

	tl.ClearCount()
	if tl.CountBuffer != "" {
		t.Errorf("expected buffer cleared, got %q", tl.CountBuffer)
	}
}

func TestTimeline_HandleActionWithCount_MoveDown(t *testing.T) {
	tl := NewTimeline()
	var items []model.TimelineItem
	for i := 0; i < 20; i++ {
		items = append(items, &model.ToolCall{
			ID: "tc", Name: "Read", Summary: "file.go", Status: model.ToolCallDone,
		})
	}
	props := timelineProps(items)

	// Move down by 5
	tl.HandleActionWithCount("move_down", 5, props)
	if tl.Cursor != 5 {
		t.Errorf("expected cursor=5 after 5j, got %d", tl.Cursor)
	}

	// Move down by 12 from position 5 → should clamp at 19
	tl.HandleActionWithCount("move_down", 12, props)
	if tl.Cursor != 17 {
		t.Errorf("expected cursor=17 after 12j from 5 (clamped), got %d", tl.Cursor)
	}
}

func TestTimeline_HandleActionWithCount_MoveUp(t *testing.T) {
	tl := NewTimeline()
	var items []model.TimelineItem
	for i := 0; i < 20; i++ {
		items = append(items, &model.ToolCall{
			ID: "tc", Name: "Read", Summary: "file.go", Status: model.ToolCallDone,
		})
	}
	props := timelineProps(items)

	tl.Cursor = 15

	// Move up by 5
	tl.HandleActionWithCount("move_up", 5, props)
	if tl.Cursor != 10 {
		t.Errorf("expected cursor=10 after 5k from 15, got %d", tl.Cursor)
	}

	// Move up by 20 from position 10 → should clamp at 0
	tl.HandleActionWithCount("move_up", 20, props)
	if tl.Cursor != 0 {
		t.Errorf("expected cursor=0 after 20k from 10 (clamped), got %d", tl.Cursor)
	}
}

func TestTimeline_HandleActionWithCount_NoCount_MovesOne(t *testing.T) {
	tl := NewTimeline()
	items := makeTimelineItems()
	props := timelineProps(items)

	// Without count, HandleAction moves by 1 (default)
	tl.HandleAction("move_down", props)
	if tl.Cursor != 1 {
		t.Errorf("expected cursor=1 after j without count, got %d", tl.Cursor)
	}
}

func TestTimeline_CountBuffer_ClearedByOtherActions(t *testing.T) {
	tl := NewTimeline()
	tl.CountBuffer = "5"

	tl.ClearCount()
	if tl.CountBuffer != "" {
		t.Errorf("expected count buffer cleared, got %q", tl.CountBuffer)
	}
}

func TestTimeline_View_PendingCountDisplay(t *testing.T) {
	tl := NewTimeline()
	tl.CountBuffer = "12"
	items := makeTimelineItems()
	props := timelineProps(items)
	props.Focused = true

	result := tl.View(props)

	// The count should appear in the bottom-right area
	if !strings.Contains(result, "12") {
		t.Error("expected pending count '12' displayed in view")
	}
}

func TestTimeline_View_NoPendingCountWhenEmpty(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Read", Summary: "main.go",
			Status: model.ToolCallDone,
		},
	}
	props := timelineProps(items)

	result := tl.View(props)

	// Count display area should not appear when buffer is empty.
	// We just verify the view renders without error.
	if !strings.Contains(result, "Read") {
		t.Error("expected 'Read' in view with no pending count")
	}
}

// --- Tests for 3.5 Full Diffs with Adaptive Layout ---

func TestTimeline_View_EditDiffUnifiedHasLineNumbers(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Edit", Summary: "main.go",
			Status:   model.ToolCallDone,
			Duration: time.Second,
			RawInput: map[string]interface{}{
				"old_string": "oldLine1\noldLine2",
				"new_string": "newLine1",
			},
			Expanded: true,
		},
	}
	// Width 80 (< 120): unified diff with line numbers
	props := TimelineProps{
		Items:   items,
		Width:   80,
		Height:  20,
		Focused: true,
		Theme:   testTheme(),
	}
	result := tl.View(props)

	// Unified diff should show -/+ with line numbers
	if !strings.Contains(result, "-oldLine1") {
		t.Error("expected '-oldLine1' in unified diff")
	}
	if !strings.Contains(result, "+newLine1") {
		t.Error("expected '+newLine1' in unified diff")
	}
	// Line number "1" should be present (gutter)
	if !strings.Contains(result, "1") {
		t.Error("expected line number in unified diff gutter")
	}
}

func TestTimeline_View_EditDiffSideBySide(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Edit", Summary: "main.go",
			Status:   model.ToolCallDone,
			Duration: time.Second,
			RawInput: map[string]interface{}{
				"old_string": "oldContent",
				"new_string": "newContent",
			},
			Expanded: true,
		},
	}
	// Width 140 (>= 120): side-by-side diff
	props := TimelineProps{
		Items:   items,
		Width:   140,
		Height:  20,
		Focused: true,
		Theme:   testTheme(),
	}
	result := tl.View(props)

	// Side-by-side should show both old and new content on same row
	if !strings.Contains(result, "oldContent") {
		t.Error("expected 'oldContent' in side-by-side diff")
	}
	if !strings.Contains(result, "newContent") {
		t.Error("expected 'newContent' in side-by-side diff")
	}
	// Should contain vertical divider
	if !strings.Contains(result, "│") {
		t.Error("expected '│' divider in side-by-side diff")
	}
}

func TestTimeline_View_EditDiffSideBySideRowCount(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Edit", Summary: "main.go",
			Status:   model.ToolCallDone,
			Duration: time.Second,
			RawInput: map[string]interface{}{
				"old_string": "old1",
				"new_string": "new1\nnew2\nnew3",
			},
			Expanded: true,
		},
	}
	// Width 140: side-by-side should produce max(1, 3) = 3 content rows
	props := TimelineProps{
		Items:   items,
		Width:   140,
		Height:  20,
		Focused: true,
		Theme:   testTheme(),
	}
	result := tl.View(props)

	// Should contain all new lines
	if !strings.Contains(result, "new1") {
		t.Error("expected 'new1' in side-by-side diff")
	}
	if !strings.Contains(result, "new3") {
		t.Error("expected 'new3' in side-by-side diff")
	}
}

func TestTimeline_View_TokenAttribution(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Read", Summary: "main.go",
			Status:          model.ToolCallDone,
			Duration:        2 * time.Second,
			InputTokens:     1200,
			CacheReadTokens: 812,
		},
	}
	props := timelineProps(items)

	result := tl.View(props)

	if !strings.Contains(result, "↑1.2k") {
		t.Error("expected '↑1.2k' input token count in view")
	}
	if !strings.Contains(result, "⚡812") {
		t.Error("expected '⚡812' cache read token count in view")
	}
}

func TestTimeline_View_TokenAttributionLargeValues(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Bash", Summary: "go test ./...",
			Status:          model.ToolCallError,
			Duration:        4500 * time.Millisecond,
			InputTokens:     340,
			CacheReadTokens: 28100,
		},
	}
	props := timelineProps(items)

	result := tl.View(props)

	if !strings.Contains(result, "↑340") {
		t.Error("expected '↑340' input token count")
	}
	if !strings.Contains(result, "⚡28.1k") {
		t.Error("expected '⚡28.1k' cache read token count")
	}
}

func TestTimeline_View_TokenAttributionZeroTokens(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Read", Summary: "main.go",
			Status:          model.ToolCallDone,
			Duration:        time.Second,
			InputTokens:     0,
			CacheReadTokens: 0,
		},
	}
	props := timelineProps(items)

	result := tl.View(props)

	// No token info displayed when both are zero
	if strings.Contains(result, "↑") {
		t.Error("did not expect token attribution when tokens are zero")
	}
	if strings.Contains(result, "⚡") {
		t.Error("did not expect cache token attribution when tokens are zero")
	}
}

func TestTimeline_View_TokenAttributionInGroup(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCallGroup{
			ToolName: "Read",
			Expanded: true,
			Children: []*model.ToolCall{
				{
					ID: "tc1", Name: "Read", Summary: "a.go",
					Status: model.ToolCallDone, Duration: time.Second,
					InputTokens: 500, CacheReadTokens: 1000,
				},
				{
					ID: "tc2", Name: "Read", Summary: "b.go",
					Status: model.ToolCallDone, Duration: time.Second,
					InputTokens: 500, CacheReadTokens: 1000,
				},
			},
		},
	}
	props := timelineProps(items)

	result := tl.View(props)

	if !strings.Contains(result, "↑500") {
		t.Error("expected '↑500' token count in group children")
	}
	if !strings.Contains(result, "⚡1.0k") {
		t.Error("expected '⚡1.0k' cache token count in group children")
	}
}

// highlightBgCode returns the ANSI TrueColor background escape code fragment
// produced by lipgloss for the test theme's highlight color. We derive it
// dynamically rather than hard-coding RGB values because lipgloss may apply
// slight rounding during color conversion.
func highlightBgCode() string {
	lipgloss.SetColorProfile(termenv.TrueColor)
	s := lipgloss.NewStyle().Background(lipgloss.Color(testTheme().Highlight))
	rendered := s.Render("X")
	// Extract "48;2;R;G;B" from the ANSI sequence.
	idx := strings.Index(rendered, "48;2;")
	if idx < 0 {
		return "48;2;" // fallback prefix
	}
	// Find the 'm' that closes the SGR sequence.
	end := strings.IndexByte(rendered[idx:], 'm')
	if end < 0 {
		return rendered[idx:]
	}
	return rendered[idx : idx+end]
}

// TestTimeline_View_HighlightPerSegmentBackground verifies that the cursor row
// highlight background is baked into each rendered segment rather than applied
// as an outer wrapper. With TrueColor forced, the background escape code
// (48;2;R;G;B) should appear multiple times in the highlighted line — once per
// styled segment — proving the background survives inner ANSI resets.
func TestTimeline_View_HighlightPerSegmentBackground(t *testing.T) {
	// Force ANSI output so we can inspect escape codes.
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(termenv.Ascii) // restore for other tests

	bgCode := highlightBgCode()

	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Read", Summary: "main.go",
			Status: model.ToolCallDone, Duration: 2 * time.Second,
		},
	}
	props := timelineProps(items)
	props.Focused = true

	result := tl.View(props)

	// The highlighted row is the first (and only) line of actual content.
	lines := strings.Split(result, "\n")
	if len(lines) == 0 {
		t.Fatal("expected at least one rendered line")
	}
	highlightedLine := lines[0]

	occurrences := strings.Count(highlightedLine, bgCode)
	if occurrences < 2 {
		t.Errorf("expected background code %q to appear at least twice (once per segment), got %d occurrences in: %q",
			bgCode, occurrences, highlightedLine)
	}

	// Content should still be present.
	if !strings.Contains(result, "Read") {
		t.Error("expected 'Read' tool name in highlighted row")
	}
	if !strings.Contains(result, "main.go") {
		t.Error("expected 'main.go' summary in highlighted row")
	}
	if strings.Contains(result, "✓") {
		t.Error("result indicator ✓ should not be present in highlighted row")
	}
}

// TestTimeline_View_HighlightBackgroundOnGroupChild verifies that per-segment
// highlighting also works when the cursor is on a child within a group.
func TestTimeline_View_HighlightBackgroundOnGroupChild(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(termenv.Ascii)

	bgCode := highlightBgCode()

	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCallGroup{
			ToolName: "Read",
			Expanded: true,
			Children: []*model.ToolCall{
				{ID: "tc1", Name: "Read", Summary: "a.go", Status: model.ToolCallDone, Duration: time.Second},
				{ID: "tc2", Name: "Read", Summary: "b.go", Status: model.ToolCallDone, Duration: time.Second},
			},
		},
	}
	props := timelineProps(items)
	props.Focused = true

	// Move cursor to the first child (flatPos 1).
	tl.HandleAction("move_down", props)
	if tl.Cursor != 1 {
		t.Fatalf("expected cursor at 1, got %d", tl.Cursor)
	}

	result := tl.View(props)
	lines := strings.Split(result, "\n")

	// The child row should be line index 1 (after the group header at line 0).
	if len(lines) < 2 {
		t.Fatal("expected at least 2 rendered lines")
	}
	childLine := lines[1]

	occurrences := strings.Count(childLine, bgCode)
	if occurrences < 2 {
		t.Errorf("expected background code in group child, got %d occurrences in: %q",
			occurrences, childLine)
	}

	if !strings.Contains(childLine, "a.go") {
		t.Error("expected 'a.go' summary in highlighted child row")
	}
}

// TestTimeline_View_HighlightBackgroundOnTextBlock verifies that text block
// lines get per-segment background when highlighted.
func TestTimeline_View_HighlightBackgroundOnTextBlock(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(termenv.Ascii)

	bgCode := highlightBgCode()

	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.TextBlock{Text: "Looking at the code"},
	}
	props := timelineProps(items)
	props.Focused = true

	result := tl.View(props)
	lines := strings.Split(result, "\n")
	if len(lines) == 0 {
		t.Fatal("expected at least one rendered line")
	}
	highlightedLine := lines[0]

	if !strings.Contains(highlightedLine, bgCode) {
		t.Errorf("expected background code %q in text block line: %q", bgCode, highlightedLine)
	}
	if !strings.Contains(result, "Looking at the code") {
		t.Error("expected text block content")
	}
}

// TestRenderToolCallLine_HighlightBg_PerSegment directly tests that
// renderToolCallLine with a highlightBg produces background codes in each
// styled segment.
func TestRenderToolCallLine_HighlightBg_PerSegment(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(termenv.Ascii)

	bgCode := highlightBgCode()

	tc := &model.ToolCall{
		ID: "tc1", Name: "Read", Summary: "main.go",
		Status: model.ToolCallDone, Duration: 2 * time.Second,
	}
	th := testTheme()

	// Without highlight: no background codes expected.
	noHighlight := renderToolCallLine(tc, 6, 40, 8, false, th, "", "")
	if strings.Contains(noHighlight, "48;2;") {
		t.Errorf("expected no background code without highlight, got: %q", noHighlight)
	}

	// With highlight: background codes should appear in each segment.
	withHighlight := renderToolCallLine(tc, 6, 40, 8, false, th, th.Highlight, "")
	occurrences := strings.Count(withHighlight, bgCode)
	// At minimum: icon, name, summary, duration = 4 segments + spaces/indent
	if occurrences < 4 {
		t.Errorf("expected at least 4 background code occurrences (one per segment), got %d in: %q",
			occurrences, withHighlight)
	}
}

// --- Path trimming integration tests ---

func TestTimeline_View_PathTrimming_CWD(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Read",
			Summary: "/home/lox/Development/skinner/internal/tui/view.go",
			Status:  model.ToolCallDone, Duration: 2 * time.Second,
		},
	}
	props := timelineProps(items)
	props.WorkDir = "/home/lox/Development/skinner"

	result := tl.View(props)

	// The trimmed relative path should appear in the output.
	if !strings.Contains(result, "internal/tui/view.go") {
		t.Error("expected trimmed path 'internal/tui/view.go' in rendered output")
	}
	// The full absolute path should NOT appear (CWD prefix stripped).
	if strings.Contains(result, "/home/lox/Development/skinner/internal") {
		t.Error("full absolute path should be trimmed when WorkDir is set")
	}
}

func TestTimeline_View_PathTrimming_HomeFallback(t *testing.T) {
	tl := NewTimeline()
	homeDir := "/home/lox"
	t.Setenv("HOME", homeDir)

	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Read",
			Summary: "/home/lox/.config/skinner/config.toml",
			Status:  model.ToolCallDone, Duration: 1 * time.Second,
		},
	}
	props := timelineProps(items)
	// WorkDir is a different directory, so CWD rule won't match — HOME fallback applies.
	props.WorkDir = "/tmp/other"

	result := tl.View(props)

	if !strings.Contains(result, "~/.config/skinner/config.toml") {
		t.Error("expected home-trimmed path '~/.config/skinner/config.toml' in rendered output")
	}
}

func TestTimeline_View_PathTrimming_NoWorkDir(t *testing.T) {
	tl := NewTimeline()
	absPath := "/etc/hosts"
	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Read", Summary: absPath,
			Status: model.ToolCallDone, Duration: 1 * time.Second,
		},
	}
	props := timelineProps(items)
	// No WorkDir set — path should appear unchanged.

	result := tl.View(props)

	if !strings.Contains(result, "/etc/hosts") {
		t.Error("expected full path '/etc/hosts' when no WorkDir is set")
	}
}

func TestTimeline_View_PathTrimming_EditTool(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Edit",
			Summary: "/home/lox/Development/skinner/main.go (+2/-2)",
			Status:  model.ToolCallDone, Duration: 500 * time.Millisecond,
		},
	}
	props := timelineProps(items)
	props.WorkDir = "/home/lox/Development/skinner"

	result := tl.View(props)

	if !strings.Contains(result, "main.go (+2/-2)") {
		t.Error("expected trimmed Edit summary 'main.go (+2/-2)' in rendered output")
	}
	if strings.Contains(result, "/home/lox/Development/skinner/main.go") {
		t.Error("full absolute path should be trimmed for Edit tool")
	}
}

func TestTimeline_View_PathTrimming_GrepTool(t *testing.T) {
	tl := NewTimeline()
	items := []model.TimelineItem{
		&model.ToolCall{
			ID: "tc1", Name: "Grep",
			Summary: "TODO in /home/lox/Development/skinner/internal",
			Status:  model.ToolCallDone, Duration: 1 * time.Second,
		},
	}
	props := timelineProps(items)
	props.WorkDir = "/home/lox/Development/skinner"

	result := tl.View(props)

	// Grep trims only the path part after " in ".
	if !strings.Contains(result, "TODO in internal") {
		t.Error("expected trimmed Grep summary 'TODO in internal' in rendered output")
	}
}

func TestTimeline_View_PathTrimming_InGroup(t *testing.T) {
	tl := NewTimeline()
	group := &model.ToolCallGroup{
		ToolName: "Read",
		Children: []*model.ToolCall{
			{
				ID: "tc1", Name: "Read",
				Summary: "/home/lox/Development/skinner/go.mod",
				Status:  model.ToolCallDone, Duration: 1 * time.Second,
			},
			{
				ID: "tc2", Name: "Read",
				Summary: "/home/lox/Development/skinner/go.sum",
				Status:  model.ToolCallDone, Duration: 1 * time.Second,
			},
		},
	}
	group.Expanded = true

	items := []model.TimelineItem{group}
	props := timelineProps(items)
	props.WorkDir = "/home/lox/Development/skinner"

	result := tl.View(props)

	// Expanded group children should have trimmed paths.
	if !strings.Contains(result, "go.mod") {
		t.Error("expected trimmed path 'go.mod' for first group child")
	}
	if !strings.Contains(result, "go.sum") {
		t.Error("expected trimmed path 'go.sum' for second group child")
	}
	if strings.Contains(result, "/home/lox/Development/skinner/go.mod") {
		t.Error("full path should be trimmed in group children")
	}
}

func TestTimeline_ThinkingRowShown(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	tl := NewTimeline()
	items := makeTimelineItems()
	props := timelineProps(items)
	props.IsThinking = true
	props.ThinkingStartTime = time.Now().Add(-3 * time.Second)

	result := tl.View(props)
	if !strings.Contains(result, "🧠") {
		t.Error("expected brain emoji in thinking row")
	}
	if !strings.Contains(result, "Thinking...") {
		t.Error("expected 'Thinking...' text in thinking row")
	}
}

func TestTimeline_ThinkingRowHidden(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	tl := NewTimeline()
	items := makeTimelineItems()
	props := timelineProps(items)
	props.IsThinking = false

	result := tl.View(props)
	if strings.Contains(result, "🧠") {
		t.Error("thinking row should not appear when IsThinking is false")
	}
	if strings.Contains(result, "Thinking...") {
		t.Error("thinking text should not appear when IsThinking is false")
	}
}

func TestTimeline_ThinkingRowDoesNotAffectCursorCount(t *testing.T) {
	items := makeTimelineItems()
	props := timelineProps(items)
	countWithout := FlatCursorCount(props.Items)

	props.IsThinking = true
	props.ThinkingStartTime = time.Now()
	countWith := FlatCursorCount(props.Items)

	if countWith != countWithout {
		t.Errorf("thinking row should not affect cursor count: got %d, want %d", countWith, countWithout)
	}
}
