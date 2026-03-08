package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/loxstomper/skinner/internal/model"
)

func makeIterations(n int) []model.Iteration {
	iters := make([]model.Iteration, n)
	for i := range iters {
		iters[i] = model.Iteration{
			Index:     i,
			Status:    model.IterationCompleted,
			Duration:  time.Duration(i+1) * time.Minute,
			StartTime: time.Now().Add(-time.Duration(i+1) * time.Minute),
		}
	}
	return iters
}

func iterListProps(iters []model.Iteration) IterListProps {
	return IterListProps{
		Iterations: iters,
		Width:      32,
		Height:     20,
		Focused:    true,
		Theme:      testTheme(),
	}
}

func TestIterList_CursorDown(t *testing.T) {
	il := NewIterList()
	iters := makeIterations(5)
	props := iterListProps(iters)

	il.HandleAction("move_down", props)
	if il.Cursor != 1 {
		t.Errorf("expected cursor=1 after j, got %d", il.Cursor)
	}

	il.HandleAction("move_down", props)
	if il.Cursor != 2 {
		t.Errorf("expected cursor=2 after second j, got %d", il.Cursor)
	}
}

func TestIterList_CursorUp(t *testing.T) {
	il := NewIterList()
	il.Cursor = 3
	iters := makeIterations(5)
	props := iterListProps(iters)

	il.HandleAction("move_up", props)
	if il.Cursor != 2 {
		t.Errorf("expected cursor=2 after k, got %d", il.Cursor)
	}
}

func TestIterList_CursorBounds(t *testing.T) {
	t.Run("cannot go below 0", func(t *testing.T) {
		il := NewIterList()
		iters := makeIterations(3)
		props := iterListProps(iters)

		il.HandleAction("move_up", props)
		if il.Cursor != 0 {
			t.Errorf("expected cursor=0 at top, got %d", il.Cursor)
		}
	})

	t.Run("cannot exceed count-1", func(t *testing.T) {
		il := NewIterList()
		il.Cursor = 2
		iters := makeIterations(3)
		props := iterListProps(iters)

		il.HandleAction("move_down", props)
		if il.Cursor != 2 {
			t.Errorf("expected cursor=2 at bottom, got %d", il.Cursor)
		}
	})
}

func TestIterList_AutoFollow(t *testing.T) {
	il := NewIterList()

	// Initially following
	if !il.AutoFollow.Following() {
		t.Error("expected auto-follow to start as true")
	}

	// Moving up pauses auto-follow
	iters := makeIterations(5)
	props := iterListProps(iters)
	il.Cursor = 3
	il.HandleAction("move_up", props)
	if il.AutoFollow.Following() {
		t.Error("expected auto-follow to pause after moving up")
	}
}

func TestIterList_OnNewIteration(t *testing.T) {
	il := NewIterList()

	// Auto-follow moves cursor to last
	il.OnNewIteration(3, 20)
	if il.Cursor != 2 {
		t.Errorf("expected cursor=2 after OnNewIteration(3), got %d", il.Cursor)
	}

	il.OnNewIteration(5, 20)
	if il.Cursor != 4 {
		t.Errorf("expected cursor=4 after OnNewIteration(5), got %d", il.Cursor)
	}
}

func TestIterList_JumpToTop(t *testing.T) {
	il := NewIterList()
	il.Cursor = 5

	il.JumpToTop()
	if il.Cursor != 0 {
		t.Errorf("expected cursor=0 after JumpToTop, got %d", il.Cursor)
	}
}

func TestIterList_JumpToBottom(t *testing.T) {
	il := NewIterList()

	il.JumpToBottom(10, 20)
	if il.Cursor != 9 {
		t.Errorf("expected cursor=9 after JumpToBottom(10), got %d", il.Cursor)
	}
	if !il.AutoFollow.Following() {
		t.Error("expected auto-follow to resume after JumpToBottom")
	}
}

func TestIterList_View_Empty(t *testing.T) {
	il := NewIterList()
	props := IterListProps{
		Iterations: nil,
		Width:      32,
		Height:     10,
		Focused:    true,
		Theme:      testTheme(),
	}

	result := il.View(props)
	// Should render without panicking, empty content
	if result == "" {
		t.Error("expected non-empty view output (at least styled empty content)")
	}
}

func TestIterList_View_SingleIteration(t *testing.T) {
	il := NewIterList()
	iters := []model.Iteration{
		{
			Index:     0,
			Status:    model.IterationRunning,
			StartTime: time.Now().Add(-30 * time.Second),
		},
	}
	props := IterListProps{
		Iterations: iters,
		Width:      40,
		Height:     10,
		Focused:    true,
		Theme:      testTheme(),
	}

	result := il.View(props)

	if !strings.Contains(result, "Iter 1") {
		t.Error("expected 'Iter 1' in output")
	}
	if !strings.Contains(result, "⟳") {
		t.Error("expected running icon ⟳")
	}
	// Running iterations show live elapsed time with ... suffix
	if !strings.Contains(result, "...") {
		t.Error("expected '...' suffix for running iteration duration")
	}
	if !strings.Contains(result, "s...") {
		t.Error("expected elapsed time with '...' suffix (e.g. '30.0s...')")
	}
}

func TestIterList_View_MultipleIterations(t *testing.T) {
	il := NewIterList()
	iters := []model.Iteration{
		{Index: 0, Status: model.IterationCompleted, Duration: 2 * time.Minute},
		{Index: 1, Status: model.IterationCompleted, Duration: time.Minute},
		{Index: 2, Status: model.IterationRunning, StartTime: time.Now().Add(-30 * time.Second)},
	}
	props := IterListProps{
		Iterations: iters,
		Width:      40,
		Height:     10,
		Focused:    true,
		Theme:      testTheme(),
	}

	result := il.View(props)

	if !strings.Contains(result, "Iter 1") {
		t.Error("expected 'Iter 1'")
	}
	if !strings.Contains(result, "Iter 2") {
		t.Error("expected 'Iter 2'")
	}
	if !strings.Contains(result, "Iter 3") {
		t.Error("expected 'Iter 3'")
	}
	if !strings.Contains(result, "✓") {
		t.Error("expected completed icon ✓")
	}
	if !strings.Contains(result, "⟳") {
		t.Error("expected running icon ⟳")
	}
}

func TestIterList_View_RunningDurationShowsElapsed(t *testing.T) {
	il := NewIterList()
	// Iteration started 2 minutes and 14 seconds ago
	iters := []model.Iteration{
		{
			Index:     0,
			Status:    model.IterationRunning,
			StartTime: time.Now().Add(-2*time.Minute - 14*time.Second),
		},
	}
	props := IterListProps{
		Iterations: iters,
		Width:      50,
		Height:     10,
		Focused:    true,
		Theme:      testTheme(),
	}

	result := il.View(props)

	// Should show elapsed time with ... suffix, e.g. "2m14s..."
	if !strings.Contains(result, "2m14s...") {
		t.Errorf("expected '2m14s...' for running iteration, got: %s", result)
	}
}

func TestIterList_View_CompletedDurationNoSuffix(t *testing.T) {
	il := NewIterList()
	iters := []model.Iteration{
		{
			Index:    0,
			Status:   model.IterationCompleted,
			Duration: 2*time.Minute + 14*time.Second,
		},
	}
	props := IterListProps{
		Iterations: iters,
		Width:      50,
		Height:     10,
		Focused:    true,
		Theme:      testTheme(),
	}

	result := il.View(props)

	// Completed iterations show final duration without ... suffix
	if !strings.Contains(result, "2m14s") {
		t.Errorf("expected '2m14s' for completed iteration, got: %s", result)
	}
	// Should NOT have ... suffix
	if strings.Contains(result, "2m14s...") {
		t.Errorf("completed iteration should not have '...' suffix, got: %s", result)
	}
}

func TestIterList_PageDown(t *testing.T) {
	il := NewIterList()
	iters := makeIterations(30)
	props := iterListProps(iters)
	props.Height = 10

	il.HandleAction("page_down", props)
	if il.Cursor != 10 {
		t.Errorf("expected cursor=10 after pgdown with height=10, got %d", il.Cursor)
	}
}

func TestIterList_PageUp(t *testing.T) {
	il := NewIterList()
	il.Cursor = 15
	iters := makeIterations(30)
	props := iterListProps(iters)
	props.Height = 10

	il.HandleAction("page_up", props)
	if il.Cursor != 5 {
		t.Errorf("expected cursor=5 after pgup from 15 with height=10, got %d", il.Cursor)
	}
}

// Scroll behavior tests — cursor beyond viewport triggers scroll adjustment

func TestIterList_ScrollDown_CursorBelowViewport(t *testing.T) {
	il := NewIterList()
	iters := makeIterations(20)
	props := iterListProps(iters)
	props.Height = 5

	// Move cursor beyond the viewport (height=5, so visible rows 0-4)
	for i := 0; i < 6; i++ {
		il.HandleAction("move_down", props)
	}

	if il.Cursor != 6 {
		t.Errorf("expected cursor=6, got %d", il.Cursor)
	}
	// Scroll should have adjusted so cursor is visible (scroll = cursor - height + 1 = 2)
	if il.Scroll != 2 {
		t.Errorf("expected scroll=2, got %d", il.Scroll)
	}
}

func TestIterList_ScrollUp_CursorAboveViewport(t *testing.T) {
	il := NewIterList()
	il.Cursor = 10
	il.Scroll = 10
	iters := makeIterations(20)
	props := iterListProps(iters)
	props.Height = 5

	// Move cursor up above the viewport
	for i := 0; i < 3; i++ {
		il.HandleAction("move_up", props)
	}

	if il.Cursor != 7 {
		t.Errorf("expected cursor=7, got %d", il.Cursor)
	}
	// Scroll should have adjusted so cursor is visible
	if il.Scroll != 7 {
		t.Errorf("expected scroll=7, got %d", il.Scroll)
	}
}

func TestIterList_ScrollClamp_AtBoundaries(t *testing.T) {
	il := NewIterList()
	iters := makeIterations(3)
	props := iterListProps(iters)
	props.Height = 10

	// With 3 items and height 10, scroll should be clamped to 0
	il.Scroll = 5
	il.ensureCursorVisible(props)
	if il.Scroll != 0 {
		t.Errorf("expected scroll=0 (clamped), got %d", il.Scroll)
	}
}

func TestIterList_PageDown_ScrollsViewport(t *testing.T) {
	il := NewIterList()
	iters := makeIterations(30)
	props := iterListProps(iters)
	props.Height = 10

	il.HandleAction("page_down", props)
	// Cursor moved to 10, scroll should keep cursor visible
	if il.Cursor != 10 {
		t.Errorf("expected cursor=10, got %d", il.Cursor)
	}
	if il.Scroll > il.Cursor || il.Scroll+props.Height <= il.Cursor {
		t.Errorf("cursor %d not visible with scroll=%d height=%d", il.Cursor, il.Scroll, props.Height)
	}
}

func TestIterList_PageUp_ScrollsViewport(t *testing.T) {
	il := NewIterList()
	il.Cursor = 20
	il.Scroll = 15
	iters := makeIterations(30)
	props := iterListProps(iters)
	props.Height = 10

	il.HandleAction("page_up", props)
	// Cursor moved to 10
	if il.Cursor != 10 {
		t.Errorf("expected cursor=10, got %d", il.Cursor)
	}
	// Scroll should keep cursor visible
	if il.Scroll > il.Cursor || il.Scroll+props.Height <= il.Cursor {
		t.Errorf("cursor %d not visible with scroll=%d height=%d", il.Cursor, il.Scroll, props.Height)
	}
}

func TestIterList_OnNewIteration_ScrollFollows(t *testing.T) {
	il := NewIterList()
	// Simulate auto-follow with a small viewport
	for i := 1; i <= 15; i++ {
		il.OnNewIteration(i, 5)
	}

	if il.Cursor != 14 {
		t.Errorf("expected cursor=14, got %d", il.Cursor)
	}
	// Scroll should keep cursor visible in a height-5 viewport
	if il.Scroll != 10 {
		t.Errorf("expected scroll=10, got %d", il.Scroll)
	}
}

func TestIterList_JumpToBottom_ScrollsToEnd(t *testing.T) {
	il := NewIterList()

	il.JumpToBottom(20, 5)
	if il.Cursor != 19 {
		t.Errorf("expected cursor=19, got %d", il.Cursor)
	}
	if il.Scroll != 15 {
		t.Errorf("expected scroll=15, got %d", il.Scroll)
	}
}

func TestIterList_JumpToTop_ScrollsToTop(t *testing.T) {
	il := NewIterList()
	il.Cursor = 15
	il.Scroll = 10

	il.JumpToTop()
	if il.Cursor != 0 {
		t.Errorf("expected cursor=0, got %d", il.Cursor)
	}
	if il.Scroll != 0 {
		t.Errorf("expected scroll=0, got %d", il.Scroll)
	}
}

// --- Mouse support tests ---

func TestIterList_ScrollBy_Down(t *testing.T) {
	il := NewIterList()
	iters := makeIterations(20)
	props := iterListProps(iters)
	props.Height = 10

	il.ScrollBy(3, len(iters), props.Height)
	if il.Scroll != 3 {
		t.Errorf("expected scroll=3 after ScrollBy(3), got %d", il.Scroll)
	}
	if il.AutoFollow.Following() {
		t.Error("expected auto-follow paused after mouse scroll")
	}
}

func TestIterList_ScrollBy_Up(t *testing.T) {
	il := NewIterList()
	il.Scroll = 5
	iters := makeIterations(20)
	props := iterListProps(iters)
	props.Height = 10

	il.ScrollBy(-3, len(iters), props.Height)
	if il.Scroll != 2 {
		t.Errorf("expected scroll=2 after ScrollBy(-3) from 5, got %d", il.Scroll)
	}
}

func TestIterList_ScrollBy_ClampsAtBottom(t *testing.T) {
	il := NewIterList()
	il.Scroll = 8
	iters := makeIterations(15)
	props := iterListProps(iters)
	props.Height = 10

	il.ScrollBy(10, len(iters), props.Height)
	// Max scroll = 15 - 10 = 5, but we're at 8 which is already beyond that
	// After adding 10, it should clamp to maxScroll = 5
	if il.Scroll != 5 {
		t.Errorf("expected scroll=5 (clamped), got %d", il.Scroll)
	}
}

func TestIterList_ScrollBy_ClampsAtTop(t *testing.T) {
	il := NewIterList()
	il.Scroll = 2

	il.ScrollBy(-10, 20, 10)
	if il.Scroll != 0 {
		t.Errorf("expected scroll=0 (clamped at top), got %d", il.Scroll)
	}
}

func TestIterList_ClickRow_ValidRow(t *testing.T) {
	il := NewIterList()
	iters := makeIterations(10)

	changed := il.ClickRow(3, len(iters), 10)
	if !changed {
		t.Error("expected ClickRow to return true for valid row")
	}
	if il.Cursor != 3 {
		t.Errorf("expected cursor=3 after clicking row 3, got %d", il.Cursor)
	}
}

func TestIterList_ClickRow_WithScroll(t *testing.T) {
	il := NewIterList()
	il.Scroll = 5
	iters := makeIterations(20)

	changed := il.ClickRow(2, len(iters), 10)
	if !changed {
		t.Error("expected ClickRow to return true")
	}
	// scroll(5) + row(2) = 7
	if il.Cursor != 7 {
		t.Errorf("expected cursor=7 (scroll 5 + row 2), got %d", il.Cursor)
	}
}

func TestIterList_ClickRow_BeyondLastItem(t *testing.T) {
	il := NewIterList()
	iters := makeIterations(3)

	changed := il.ClickRow(5, len(iters), 10)
	if changed {
		t.Error("expected ClickRow to return false for click beyond last item")
	}
	if il.Cursor != 0 {
		t.Errorf("expected cursor unchanged at 0, got %d", il.Cursor)
	}
}

func TestIterList_ClickRow_PausesAutoFollow(t *testing.T) {
	il := NewIterList()
	iters := makeIterations(5)

	// Click on non-last row
	il.ClickRow(1, len(iters), 10)
	if il.AutoFollow.Following() {
		t.Error("expected auto-follow paused after clicking non-last row")
	}
}

func TestIterList_ClickRow_AtEnd_ContinuesAutoFollow(t *testing.T) {
	il := NewIterList()
	iters := makeIterations(5)

	// Click on last row
	il.ClickRow(4, len(iters), 10)
	if !il.AutoFollow.Following() {
		t.Error("expected auto-follow to continue when clicking the last row")
	}
}

func TestIterList_View_FormatShowsDurationOnly(t *testing.T) {
	il := NewIterList()
	iters := []model.Iteration{
		{
			Index:    0,
			Status:   model.IterationCompleted,
			Duration: 2*time.Minute + 14*time.Second,
			Items: []model.TimelineItem{
				&model.ToolCall{Name: "Read"},
				&model.ToolCall{Name: "Edit"},
				&model.ToolCall{Name: "Bash"},
			},
		},
	}
	props := IterListProps{
		Iterations: iters,
		Width:      50,
		Height:     10,
		Focused:    true,
		Theme:      testTheme(),
	}

	result := il.View(props)

	// Should show duration in parentheses without call count
	if !strings.Contains(result, "(2m14s)") {
		t.Errorf("expected '(2m14s)' format, got: %s", result)
	}
	// Should NOT show call count
	if strings.Contains(result, "calls") {
		t.Errorf("should not contain 'calls' in output, got: %s", result)
	}
}

func TestIterList_View_OnlyRendersVisibleSlice(t *testing.T) {
	il := NewIterList()
	il.Cursor = 7
	il.Scroll = 5
	iters := makeIterations(20)
	props := iterListProps(iters)
	props.Height = 5

	result := il.View(props)

	// Should show iterations 6-10 (indices 5-9, 1-indexed display 6-10)
	if !strings.Contains(result, "Iter 6") {
		t.Error("expected 'Iter 6' in visible view")
	}
	if !strings.Contains(result, "Iter 10") {
		t.Error("expected 'Iter 10' in visible view")
	}
	// Should NOT show iterations outside the viewport
	if strings.Contains(result, "Iter 1 ") {
		t.Error("did not expect 'Iter 1' in visible view (scrolled past)")
	}
	if strings.Contains(result, "Iter 15") {
		t.Error("did not expect 'Iter 15' in visible view (below viewport)")
	}
}
