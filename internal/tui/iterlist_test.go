package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

	il.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}, props)
	if il.Cursor != 1 {
		t.Errorf("expected cursor=1 after j, got %d", il.Cursor)
	}

	il.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}, props)
	if il.Cursor != 2 {
		t.Errorf("expected cursor=2 after second j, got %d", il.Cursor)
	}
}

func TestIterList_CursorUp(t *testing.T) {
	il := NewIterList()
	il.Cursor = 3
	iters := makeIterations(5)
	props := iterListProps(iters)

	il.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}, props)
	if il.Cursor != 2 {
		t.Errorf("expected cursor=2 after k, got %d", il.Cursor)
	}
}

func TestIterList_CursorBounds(t *testing.T) {
	t.Run("cannot go below 0", func(t *testing.T) {
		il := NewIterList()
		iters := makeIterations(3)
		props := iterListProps(iters)

		il.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}, props)
		if il.Cursor != 0 {
			t.Errorf("expected cursor=0 at top, got %d", il.Cursor)
		}
	})

	t.Run("cannot exceed count-1", func(t *testing.T) {
		il := NewIterList()
		il.Cursor = 2
		iters := makeIterations(3)
		props := iterListProps(iters)

		il.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}, props)
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
	il.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}, props)
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

	il.Update(tea.KeyMsg{Type: tea.KeyPgDown}, props)
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

	il.Update(tea.KeyMsg{Type: tea.KeyPgUp}, props)
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
		il.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}, props)
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
		il.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}, props)
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

	il.Update(tea.KeyMsg{Type: tea.KeyPgDown}, props)
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

	il.Update(tea.KeyMsg{Type: tea.KeyPgUp}, props)
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
