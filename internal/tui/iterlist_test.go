package tui

import (
	"fmt"
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
	il.OnNewIteration(3, 20, nil)
	if il.Cursor != 2 {
		t.Errorf("expected cursor=2 after OnNewIteration(3), got %d", il.Cursor)
	}

	il.OnNewIteration(5, 20, nil)
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

	il.JumpToBottom(10, 20, nil)
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
	// Running iterations show live elapsed time without ... suffix
	if strings.Contains(result, "...") {
		t.Error("running iteration duration should not have '...' suffix")
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

	// Should show elapsed time without ... suffix
	if !strings.Contains(result, "2m14s") {
		t.Errorf("expected '2m14s' for running iteration, got: %s", result)
	}
	if strings.Contains(result, "2m14s...") {
		t.Errorf("running iteration should not have '...' suffix, got: %s", result)
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
		il.OnNewIteration(i, 5, nil)
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

	il.JumpToBottom(20, 5, nil)
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

	il.ScrollBy(3, len(iters), props.Height, nil)
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

	il.ScrollBy(-3, len(iters), props.Height, nil)
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

	il.ScrollBy(10, len(iters), props.Height, nil)
	// Max scroll = 15 - 10 = 5, but we're at 8 which is already beyond that
	// After adding 10, it should clamp to maxScroll = 5
	if il.Scroll != 5 {
		t.Errorf("expected scroll=5 (clamped), got %d", il.Scroll)
	}
}

func TestIterList_ScrollBy_ClampsAtTop(t *testing.T) {
	il := NewIterList()
	il.Scroll = 2

	il.ScrollBy(-10, 20, 10, nil)
	if il.Scroll != 0 {
		t.Errorf("expected scroll=0 (clamped at top), got %d", il.Scroll)
	}
}

func TestIterList_ClickRow_ValidRow(t *testing.T) {
	il := NewIterList()
	iters := makeIterations(10)

	changed := il.ClickRow(3, len(iters), 10, nil)
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

	changed := il.ClickRow(2, len(iters), 10, nil)
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

	changed := il.ClickRow(5, len(iters), 10, nil)
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
	il.ClickRow(1, len(iters), 10, nil)
	if il.AutoFollow.Following() {
		t.Error("expected auto-follow paused after clicking non-last row")
	}
}

func TestIterList_ClickRow_AtEnd_ContinuesAutoFollow(t *testing.T) {
	il := NewIterList()
	iters := makeIterations(5)

	// Click on last row
	il.ClickRow(4, len(iters), 10, nil)
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

// --- Run separator tests ---

func TestIterList_View_RunSeparator_SingleRun(t *testing.T) {
	il := NewIterList()
	iters := makeIterations(3)
	runs := []model.Run{{PromptName: "BUILD", StartIndex: 0}}
	props := IterListProps{
		Iterations: iters,
		Runs:       runs,
		Width:      40,
		Height:     10,
		Focused:    true,
		Theme:      testTheme(),
	}

	result := il.View(props)

	// No separator before the first (only) run
	if strings.Contains(result, "──") {
		t.Error("should not show separator for single run")
	}
	if !strings.Contains(result, "Iter 1") {
		t.Error("expected Iter 1")
	}
}

func TestIterList_View_RunSeparator_TwoRuns(t *testing.T) {
	il := NewIterList()
	iters := makeIterations(5)
	runs := []model.Run{
		{PromptName: "BUILD", StartIndex: 0},
		{PromptName: "PLAN", StartIndex: 3},
	}
	props := IterListProps{
		Iterations: iters,
		Runs:       runs,
		Width:      40,
		Height:     10,
		Focused:    true,
		Theme:      testTheme(),
	}

	result := il.View(props)

	// Separator with prompt name should appear between runs
	if !strings.Contains(result, "PLAN") {
		t.Error("expected separator with prompt name 'PLAN'")
	}
	if !strings.Contains(result, "──") {
		t.Error("expected separator dash characters")
	}
	// All iterations should be present
	for i := 1; i <= 5; i++ {
		expected := fmt.Sprintf("Iter %d", i)
		if !strings.Contains(result, expected) {
			t.Errorf("expected '%s' in output", expected)
		}
	}
}

func TestIterList_View_RunSeparator_ThreeRuns(t *testing.T) {
	il := NewIterList()
	iters := makeIterations(7)
	runs := []model.Run{
		{PromptName: "BUILD", StartIndex: 0},
		{PromptName: "PLAN", StartIndex: 3},
		{PromptName: "TEST", StartIndex: 5},
	}
	props := IterListProps{
		Iterations: iters,
		Runs:       runs,
		Width:      40,
		Height:     20,
		Focused:    true,
		Theme:      testTheme(),
	}

	result := il.View(props)

	// Both separators should appear
	if !strings.Contains(result, "PLAN") {
		t.Error("expected separator with 'PLAN'")
	}
	if !strings.Contains(result, "TEST") {
		t.Error("expected separator with 'TEST'")
	}
}

func TestIterList_View_RunSeparator_NoRuns(t *testing.T) {
	il := NewIterList()
	iters := makeIterations(3)
	props := IterListProps{
		Iterations: iters,
		Runs:       nil,
		Width:      40,
		Height:     10,
		Focused:    true,
		Theme:      testTheme(),
	}

	result := il.View(props)

	// Should render normally without separators
	if !strings.Contains(result, "Iter 1") {
		t.Error("expected Iter 1")
	}
}

func TestIterList_ClickRow_SkipsSeparator(t *testing.T) {
	il := NewIterList()
	iters := makeIterations(5)
	runs := []model.Run{
		{PromptName: "BUILD", StartIndex: 0},
		{PromptName: "PLAN", StartIndex: 3},
	}

	// Display rows: 0=Iter0, 1=Iter1, 2=Iter2, 3=separator, 4=Iter3, 5=Iter4
	// Click on separator row (display row 3)
	changed := il.ClickRow(3, len(iters), 10, runs)
	if changed {
		t.Error("clicking on separator should return false")
	}

	// Click on iteration after separator (display row 4 = Iter 3)
	changed = il.ClickRow(4, len(iters), 10, runs)
	if !changed {
		t.Error("clicking on Iter 3 (display row 4) should return true")
	}
	if il.Cursor != 3 {
		t.Errorf("expected cursor=3 after clicking display row 4, got %d", il.Cursor)
	}
}

func TestIterList_ScrollWithSeparators(t *testing.T) {
	il := NewIterList()
	iters := makeIterations(10)
	runs := []model.Run{
		{PromptName: "BUILD", StartIndex: 0},
		{PromptName: "PLAN", StartIndex: 5},
	}
	// Total display lines = 10 iterations + 1 separator = 11

	// Scroll clamping with separators: max scroll = 11 - 5 = 6
	il.ScrollBy(20, len(iters), 5, runs)
	if il.Scroll != 6 {
		t.Errorf("expected scroll clamped to 6, got %d", il.Scroll)
	}
}

func TestIterList_EnsureCursorVisible_WithSeparators(t *testing.T) {
	il := NewIterList()
	iters := makeIterations(10)
	runs := []model.Run{
		{PromptName: "BUILD", StartIndex: 0},
		{PromptName: "PLAN", StartIndex: 3},
	}
	props := IterListProps{
		Iterations: iters,
		Runs:       runs,
		Width:      40,
		Height:     5,
		Focused:    true,
		Theme:      testTheme(),
	}

	// Move cursor to iteration 5 (display row = 5 + 1 separator = 6)
	il.Cursor = 5
	il.ensureCursorVisible(props)
	// Display row 6 should be visible: scroll should be at least 6 - 4 = 2
	if il.Scroll > 6 || il.Scroll+5 <= 6 {
		t.Errorf("cursor display row 6 not visible with scroll=%d height=5", il.Scroll)
	}
}

func TestIterList_JumpToBottom_WithSeparators(t *testing.T) {
	il := NewIterList()
	runs := []model.Run{
		{PromptName: "BUILD", StartIndex: 0},
		{PromptName: "PLAN", StartIndex: 5},
	}
	// 10 iterations + 1 separator = 11 display lines
	il.JumpToBottom(10, 5, runs)
	if il.Cursor != 9 {
		t.Errorf("expected cursor=9, got %d", il.Cursor)
	}
	// Last iteration display row = 9 + 1 = 10
	// Scroll should be 10 - 5 + 1 = 6
	if il.Scroll != 6 {
		t.Errorf("expected scroll=6, got %d", il.Scroll)
	}
}

func TestIterList_OnNewIteration_WithSeparators(t *testing.T) {
	il := NewIterList()
	runs := []model.Run{
		{PromptName: "BUILD", StartIndex: 0},
		{PromptName: "PLAN", StartIndex: 5},
	}

	// Simulate adding iterations with auto-follow in a small viewport
	for i := 1; i <= 8; i++ {
		il.OnNewIteration(i, 5, runs)
	}

	if il.Cursor != 7 {
		t.Errorf("expected cursor=7, got %d", il.Cursor)
	}
	// Iteration 7 display row = 7 + 1 sep = 8
	// Total display = 8 + 1 = 9, max scroll = 9 - 5 = 4
	// Scroll to show cursor: 8 - 5 + 1 = 4
	if il.Scroll != 4 {
		t.Errorf("expected scroll=4, got %d", il.Scroll)
	}
}

// --- Separator helper function tests ---

func TestSeparatorsBefore(t *testing.T) {
	runs := []model.Run{
		{PromptName: "BUILD", StartIndex: 0},
		{PromptName: "PLAN", StartIndex: 3},
		{PromptName: "TEST", StartIndex: 5},
	}

	tests := []struct {
		iterIndex int
		want      int
	}{
		{0, 0}, // before first run's start
		{2, 0}, // still in first run
		{3, 1}, // at second run's start
		{4, 1}, // in second run
		{5, 2}, // at third run's start
		{6, 2}, // in third run
	}

	for _, tt := range tests {
		got := separatorsBefore(tt.iterIndex, runs)
		if got != tt.want {
			t.Errorf("separatorsBefore(%d) = %d, want %d", tt.iterIndex, got, tt.want)
		}
	}
}

func TestDisplayRowForIter(t *testing.T) {
	runs := []model.Run{
		{PromptName: "BUILD", StartIndex: 0},
		{PromptName: "PLAN", StartIndex: 3},
	}

	tests := []struct {
		iterIndex int
		want      int
	}{
		{0, 0}, // row 0
		{1, 1}, // row 1
		{2, 2}, // row 2
		{3, 4}, // row 4 (separator at 3)
		{4, 5}, // row 5
	}

	for _, tt := range tests {
		got := displayRowForIter(tt.iterIndex, runs)
		if got != tt.want {
			t.Errorf("displayRowForIter(%d) = %d, want %d", tt.iterIndex, got, tt.want)
		}
	}
}

func TestDisplayRowToIterIndex(t *testing.T) {
	runs := []model.Run{
		{PromptName: "BUILD", StartIndex: 0},
		{PromptName: "PLAN", StartIndex: 3},
		{PromptName: "TEST", StartIndex: 5},
	}

	tests := []struct {
		displayRow  int
		wantSep     bool
		wantIterIdx int
	}{
		{0, false, 0},
		{1, false, 1},
		{2, false, 2},
		{3, true, -1}, // separator for PLAN
		{4, false, 3},
		{5, false, 4},
		{6, true, -1}, // separator for TEST
		{7, false, 5},
		{8, false, 6},
	}

	for _, tt := range tests {
		isSep, idx := displayRowToIterIndex(tt.displayRow, 7, runs)
		if isSep != tt.wantSep {
			t.Errorf("displayRowToIterIndex(%d): isSep=%v, want %v", tt.displayRow, isSep, tt.wantSep)
		}
		if idx != tt.wantIterIdx {
			t.Errorf("displayRowToIterIndex(%d): idx=%d, want %d", tt.displayRow, idx, tt.wantIterIdx)
		}
	}
}

func TestTotalDisplayLines(t *testing.T) {
	tests := []struct {
		iterCount int
		runs      []model.Run
		want      int
	}{
		{5, nil, 5},
		{5, []model.Run{{StartIndex: 0}}, 5},
		{5, []model.Run{{StartIndex: 0}, {StartIndex: 3}}, 6},
		{7, []model.Run{{StartIndex: 0}, {StartIndex: 3}, {StartIndex: 5}}, 9},
	}

	for _, tt := range tests {
		got := totalDisplayLines(tt.iterCount, tt.runs)
		if got != tt.want {
			t.Errorf("totalDisplayLines(%d, %d runs) = %d, want %d",
				tt.iterCount, len(tt.runs), got, tt.want)
		}
	}
}

func TestRenderRunSeparator(t *testing.T) {
	th := testTheme()
	result := renderRunSeparator("PLAN", 30, th)

	if !strings.Contains(result, "PLAN") {
		t.Error("separator should contain the prompt name")
	}
	if !strings.Contains(result, "──") {
		t.Error("separator should contain dash characters")
	}
}
