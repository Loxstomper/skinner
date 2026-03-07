package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
			ID: "tc2", Name: "Edit", Summary: "main.go (+3/-1)",
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

	tl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}, props)
	if tl.Cursor != 1 {
		t.Errorf("expected cursor=1 after j, got %d", tl.Cursor)
	}

	tl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}, props)
	if tl.Cursor != 2 {
		t.Errorf("expected cursor=2 after second j, got %d", tl.Cursor)
	}
}

func TestTimeline_CursorUp(t *testing.T) {
	tl := NewTimeline()
	tl.Cursor = 3
	items := makeTimelineItems()
	props := timelineProps(items)

	tl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}, props)
	if tl.Cursor != 2 {
		t.Errorf("expected cursor=2 after k, got %d", tl.Cursor)
	}
}

func TestTimeline_CursorBounds(t *testing.T) {
	t.Run("cannot go below 0", func(t *testing.T) {
		tl := NewTimeline()
		items := makeTimelineItems()
		props := timelineProps(items)

		tl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}, props)
		if tl.Cursor != 0 {
			t.Errorf("expected cursor=0 at top, got %d", tl.Cursor)
		}
	})

	t.Run("cannot exceed count-1", func(t *testing.T) {
		tl := NewTimeline()
		items := makeTimelineItems()
		tl.Cursor = len(items) - 1
		props := timelineProps(items)

		tl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}, props)
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

	tl.Update(tea.KeyMsg{Type: tea.KeyEnter}, props)
	if !tb.Expanded {
		t.Error("text block should be expanded after enter")
	}

	tl.Update(tea.KeyMsg{Type: tea.KeyEnter}, props)
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
	tl.Update(tea.KeyMsg{Type: tea.KeyEnter}, props)
	if group.Expanded {
		t.Error("group should be collapsed after enter on header")
	}
	if !group.ManualToggle {
		t.Error("group should have ManualToggle set")
	}

	tl.Update(tea.KeyMsg{Type: tea.KeyEnter}, props)
	if !group.Expanded {
		t.Error("group should be expanded after second enter")
	}
}

func TestTimeline_EnterOnGroupChild_NoOp(t *testing.T) {
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

	// Move cursor to first child (flatIdx 1)
	tl.Cursor = 1
	tl.Update(tea.KeyMsg{Type: tea.KeyEnter}, props)
	if !group.Expanded {
		t.Error("group should remain expanded after enter on child")
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
	tl.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}, props)
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
	if !strings.Contains(result, "✓") {
		t.Error("expected success indicator ✓")
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

	tl.Update(tea.KeyMsg{Type: tea.KeyPgDown}, props)
	if tl.Scroll != 10 {
		t.Errorf("expected scroll=10 after pgdown, got %d", tl.Scroll)
	}

	tl.Update(tea.KeyMsg{Type: tea.KeyPgUp}, props)
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
	tl.Update(tea.KeyMsg{Type: tea.KeyPgDown}, props)
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
	tl.Update(tea.KeyMsg{Type: tea.KeyPgUp}, props)
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
	tl.Update(tea.KeyMsg{Type: tea.KeyPgDown}, props)
	if tl.Cursor != 20 {
		t.Errorf("expected cursor=20 after pgdown pushed cursor out, got %d", tl.Cursor)
	}
}
