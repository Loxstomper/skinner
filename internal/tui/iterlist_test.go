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
	il.OnNewIteration(3)
	if il.Cursor != 2 {
		t.Errorf("expected cursor=2 after OnNewIteration(3), got %d", il.Cursor)
	}

	il.OnNewIteration(5)
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

	il.JumpToBottom(10)
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
