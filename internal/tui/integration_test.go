package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/loxstomper/skinner/internal/config"
	"github.com/loxstomper/skinner/internal/executor"
	"github.com/loxstomper/skinner/internal/model"
	"github.com/loxstomper/skinner/internal/session"
)

// newTestModel creates a Model wired to a FakeExecutor for integration testing.
// It sets a default window size so View() produces meaningful output.
func newTestModel(events []session.Event, maxIterations int) *Model {
	fake := &executor.FakeExecutor{Events: events}
	sess := model.Session{
		Mode:          "build",
		PromptFile:    "test-prompt.md",
		MaxIterations: maxIterations,
		StartTime:     time.Now(),
	}
	cfg := config.DefaultConfig()
	th := testTheme()
	m := NewModel(sess, cfg, "test prompt content", th, false, false, fake)
	m.width = 120
	m.height = 30
	return &m
}

// drainEvents runs Init() and then pumps all messages from the eventCh through
// Update() until the channel is drained and no more commands produce messages.
// This simulates a full Bubble Tea event loop for the given canned events.
func drainEvents(t *testing.T, m *Model) {
	t.Helper()

	// Run Init to start the first iteration and event forwarding.
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil cmd")
	}

	// Execute all batched commands and collect messages.
	msgs := executeBatchCmd(cmd)

	// Process messages until we've drained everything.
	for len(msgs) > 0 {
		var nextMsgs []tea.Msg
		for _, msg := range msgs {
			// Skip tick messages — they cause infinite loops.
			if _, ok := msg.(tickMsg); ok {
				continue
			}
			_, cmd = m.Update(msg)
			if cmd != nil {
				nextMsgs = append(nextMsgs, executeBatchCmd(cmd)...)
			}
		}
		msgs = nextMsgs
	}
}

// executeBatchCmd executes a tea.Cmd and collects all resulting messages.
// It handles tea.BatchMsg by recursively executing sub-commands.
// For commands that block on channels, it uses a short timeout.
func executeBatchCmd(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}

	msg := cmd()
	if msg == nil {
		return nil
	}

	// BatchMsg contains multiple commands to execute.
	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, subCmd := range batch {
			msgs = append(msgs, executeBatchCmd(subCmd)...)
		}
		return msgs
	}

	return []tea.Msg{msg}
}

// --- Single iteration with tool calls and text ---

func TestIntegration_SingleIteration(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.TextEvent{Text: "Let me read the code"},
			session.ToolUseEvent{ID: "tu1", Name: "Read", Summary: "main.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tu1", IsError: false, LineInfo: "(85 lines)"},
		session.AssistantBatchEvent{Events: []session.Event{
			session.TextEvent{Text: "Now I'll edit the file"},
			session.ToolUseEvent{ID: "tu2", Name: "Edit", Summary: "main.go (+3/-1)"},
		}},
		session.ToolResultEvent{ToolUseID: "tu2", IsError: false},
		session.UsageEvent{
			Model:        "claude-sonnet-4-5",
			InputTokens:  1000,
			OutputTokens: 500,
		},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	sess := m.Session()

	// Should have exactly 1 iteration, completed.
	if len(sess.Iterations) != 1 {
		t.Fatalf("expected 1 iteration, got %d", len(sess.Iterations))
	}
	iter := sess.Iterations[0]
	if iter.Status != model.IterationCompleted {
		t.Errorf("expected iteration completed, got %d", iter.Status)
	}

	// Should have 4 timeline items: text, tool call, text, tool call.
	if len(iter.Items) != 4 {
		t.Fatalf("expected 4 timeline items, got %d", len(iter.Items))
	}

	// First item: text block.
	if tb, ok := iter.Items[0].(*model.TextBlock); !ok {
		t.Error("expected first item to be TextBlock")
	} else if tb.Text != "Let me read the code" {
		t.Errorf("unexpected text: %q", tb.Text)
	}

	// Second item: Read tool call, completed.
	if tc, ok := iter.Items[1].(*model.ToolCall); !ok {
		t.Error("expected second item to be ToolCall")
	} else {
		if tc.Name != "Read" {
			t.Errorf("expected Read, got %q", tc.Name)
		}
		if tc.Status != model.ToolCallDone {
			t.Errorf("expected ToolCallDone, got %d", tc.Status)
		}
		if tc.LineInfo != "(85 lines)" {
			t.Errorf("expected line info '(85 lines)', got %q", tc.LineInfo)
		}
	}

	// Token accumulation.
	if sess.InputTokens != 1000 {
		t.Errorf("expected InputTokens=1000, got %d", sess.InputTokens)
	}
	if sess.OutputTokens != 500 {
		t.Errorf("expected OutputTokens=500, got %d", sess.OutputTokens)
	}

	// View should contain rendered content.
	view := m.View()
	if !strings.Contains(view, "Iter 1") {
		t.Error("expected 'Iter 1' in view")
	}
	if !strings.Contains(view, "✓") {
		t.Error("expected completed icon ✓ in view")
	}
}

// --- Tool call grouping: consecutive same-name tool calls become a group ---

func TestIntegration_ToolCallGrouping(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "r1", Name: "Read", Summary: "a.go"},
			session.ToolUseEvent{ID: "r2", Name: "Read", Summary: "b.go"},
			session.ToolUseEvent{ID: "r3", Name: "Read", Summary: "c.go"},
		}},
		session.ToolResultEvent{ToolUseID: "r1", IsError: false},
		session.ToolResultEvent{ToolUseID: "r2", IsError: false},
		session.ToolResultEvent{ToolUseID: "r3", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	iter := m.Session().Iterations[0]

	// 3 consecutive Reads should be grouped into 1 ToolCallGroup.
	if len(iter.Items) != 1 {
		t.Fatalf("expected 1 timeline item (group), got %d", len(iter.Items))
	}

	group, ok := iter.Items[0].(*model.ToolCallGroup)
	if !ok {
		t.Fatal("expected ToolCallGroup")
	}
	if group.ToolName != "Read" {
		t.Errorf("expected group tool name 'Read', got %q", group.ToolName)
	}
	if len(group.Children) != 3 {
		t.Errorf("expected 3 children, got %d", len(group.Children))
	}
	if group.Status() != model.ToolCallDone {
		t.Errorf("expected group status Done, got %d", group.Status())
	}

	// Group stays expanded because the cursor is on it (auto-follow puts
	// cursor on the group in the currently-viewed iteration).
	if !group.Expanded {
		t.Error("expected group to remain expanded when cursor is on it")
	}

	view := m.View()
	if !strings.Contains(view, "3 files") {
		t.Error("expected '3 files' group summary in view")
	}
}

// --- Multi-iteration: iteration list grows, cursor follows ---

func TestIntegration_MultiIteration(t *testing.T) {
	// Each iteration gets the same events from the FakeExecutor.
	// With maxIterations=3, we need 3 sets of events.
	// But FakeExecutor sends all events from a single Start() call,
	// so for multi-iteration we need to set maxIterations and have the
	// subprocess exit cleanly each time.
	//
	// The root model calls spawnIteration() again on SubprocessExitEvent
	// if ShouldStartNext() returns true. But FakeExecutor is re-used across
	// calls — each Start() call returns the same events.

	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Bash", Summary: "make test"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 3)
	drainEvents(t, m)

	sess := m.Session()

	// Should have 3 iterations, all completed.
	if len(sess.Iterations) != 3 {
		t.Fatalf("expected 3 iterations, got %d", len(sess.Iterations))
	}
	for i, iter := range sess.Iterations {
		if iter.Status != model.IterationCompleted {
			t.Errorf("iteration %d: expected completed, got %d", i, iter.Status)
		}
		if iter.Index != i {
			t.Errorf("iteration %d: expected index %d, got %d", i, i, iter.Index)
		}
	}

	// IterList cursor should follow to the last iteration (auto-follow).
	if m.iterList.Cursor != 2 {
		t.Errorf("expected iterList cursor=2, got %d", m.iterList.Cursor)
	}

	// View should show all 3 iterations.
	view := m.View()
	if !strings.Contains(view, "Iter 1") {
		t.Error("expected 'Iter 1' in view")
	}
	if !strings.Contains(view, "Iter 2") {
		t.Error("expected 'Iter 2' in view")
	}
	if !strings.Contains(view, "Iter 3") {
		t.Error("expected 'Iter 3' in view")
	}
}

// --- Key navigation: tab between panes ---

func TestIntegration_TabTogglesFocus(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "main.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Starts focused on left pane.
	if m.focusedPane != leftPane {
		t.Error("expected initial focus on left pane")
	}

	// Tab switches to right.
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focusedPane != rightPane {
		t.Error("expected right pane focus after tab")
	}

	// Tab switches back to left.
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focusedPane != leftPane {
		t.Error("expected left pane focus after second tab")
	}
}

// --- Key navigation: j/k in iteration list ---

func TestIntegration_JKNavigatesIterList(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Bash", Summary: "cmd"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 3)
	drainEvents(t, m)

	// Focus on left pane (default).
	if m.focusedPane != leftPane {
		t.Fatal("expected left pane focus")
	}

	// Cursor should be at 2 (last iteration, auto-follow).
	if m.iterList.Cursor != 2 {
		t.Fatalf("expected cursor=2, got %d", m.iterList.Cursor)
	}

	// k moves up.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.iterList.Cursor != 1 {
		t.Errorf("expected cursor=1 after k, got %d", m.iterList.Cursor)
	}

	// j moves down.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.iterList.Cursor != 2 {
		t.Errorf("expected cursor=2 after j, got %d", m.iterList.Cursor)
	}
}

// --- Enter on left pane switches to right pane ---

func TestIntegration_EnterSwitchesPaneFromLeft(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "main.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Start on left pane.
	if m.focusedPane != leftPane {
		t.Fatal("expected left pane focus")
	}

	// Enter switches to right pane.
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.focusedPane != rightPane {
		t.Error("expected right pane focus after enter on left")
	}
}

// --- Enter on right pane toggles expand/collapse ---

func TestIntegration_EnterTogglesExpandOnRight(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.TextEvent{Text: "line1\nline2\nline3\nline4\nline5"},
		}},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Focus right pane.
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focusedPane != rightPane {
		t.Fatal("expected right pane focus")
	}

	// Text block should start collapsed.
	iter := &m.Session().Iterations[0]
	tb := iter.Items[0].(*model.TextBlock)
	if tb.Expanded {
		t.Error("expected text block to start collapsed")
	}

	// Enter expands.
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !tb.Expanded {
		t.Error("expected text block to be expanded after enter")
	}

	// Enter again collapses.
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if tb.Expanded {
		t.Error("expected text block to be collapsed after second enter")
	}
}

// --- v toggles compact view ---

func TestIntegration_VTogglesCompactView(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	if m.compactView {
		t.Error("expected compact view off initially")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if !m.compactView {
		t.Error("expected compact view on after v")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if m.compactView {
		t.Error("expected compact view off after second v")
	}
}

// --- gg jumps to top ---

func TestIntegration_GGJumpsToTop(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Bash", Summary: "cmd"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 3)
	drainEvents(t, m)

	// Cursor should be at last iteration (auto-follow).
	if m.iterList.Cursor != 2 {
		t.Fatalf("expected cursor=2, got %d", m.iterList.Cursor)
	}

	// gg: first g sets pending, second g jumps to top.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if !m.gPending {
		t.Error("expected gPending after first g")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.gPending {
		t.Error("expected gPending cleared after second g")
	}
	if m.iterList.Cursor != 0 {
		t.Errorf("expected cursor=0 after gg, got %d", m.iterList.Cursor)
	}
}

// --- G jumps to bottom ---

func TestIntegration_GJumpsToBottom(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Bash", Summary: "cmd"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 3)
	drainEvents(t, m)

	// Move cursor away from bottom first.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.iterList.Cursor != 0 {
		t.Fatalf("expected cursor=0, got %d", m.iterList.Cursor)
	}

	// G jumps to bottom.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if m.iterList.Cursor != 2 {
		t.Errorf("expected cursor=2 after G, got %d", m.iterList.Cursor)
	}
}

// --- Changing iteration selection resets timeline position ---

func TestIntegration_IterationChangeResetsTimeline(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "file.go"},
			session.ToolUseEvent{ID: "tc2", Name: "Read", Summary: "other.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.ToolResultEvent{ToolUseID: "tc2", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 2)
	drainEvents(t, m)

	// Focus right pane and move cursor.
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	savedCursor := m.timeline.Cursor

	// Switch back to left pane and move to a different iteration.
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})

	// Timeline should be reset when changing iteration.
	if m.timeline.Cursor == savedCursor && savedCursor != 0 {
		t.Error("expected timeline cursor to reset when changing iteration")
	}
}

// --- Error tool result ---

func TestIntegration_ToolCallError(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Bash", Summary: "make test"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: true},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	iter := m.Session().Iterations[0]
	tc := iter.Items[0].(*model.ToolCall)
	if tc.Status != model.ToolCallError {
		t.Errorf("expected ToolCallError, got %d", tc.Status)
	}
	if !tc.IsError {
		t.Error("expected IsError=true")
	}
}

// --- Usage accumulates across multiple events ---

func TestIntegration_UsageAccumulation(t *testing.T) {
	events := []session.Event{
		session.UsageEvent{Model: "claude-sonnet-4-5", InputTokens: 1000, OutputTokens: 200},
		session.UsageEvent{Model: "claude-sonnet-4-5", InputTokens: 500, OutputTokens: 100},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	sess := m.Session()
	if sess.InputTokens != 1500 {
		t.Errorf("expected InputTokens=1500, got %d", sess.InputTokens)
	}
	if sess.OutputTokens != 300 {
		t.Errorf("expected OutputTokens=300, got %d", sess.OutputTokens)
	}
	if sess.TotalCost == 0 {
		t.Error("expected non-zero cost with known model pricing")
	}
}

// --- View renders properly at each stage ---

func TestIntegration_ViewRendering(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "main.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false, LineInfo: "(42 lines)"},
		session.AssistantBatchEvent{Events: []session.Event{
			session.TextEvent{Text: "The code looks good"},
		}},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	view := m.View()

	// Header should be present.
	if !strings.Contains(view, "Iter 1") {
		t.Error("expected 'Iter 1' in header")
	}

	// Iteration list should show completed.
	if !strings.Contains(view, "✓") {
		t.Error("expected completed icon ✓")
	}

	// Timeline should show tool call details.
	if !strings.Contains(view, "Read") {
		t.Error("expected 'Read' tool name in view")
	}
	if !strings.Contains(view, "main.go") {
		t.Error("expected 'main.go' summary in view")
	}

	// Text block content should be visible.
	if !strings.Contains(view, "The code looks good") {
		t.Error("expected text block content in view")
	}

	// Separator should be present.
	if !strings.Contains(view, "│") {
		t.Error("expected separator │ in view")
	}
}

// --- View before window size shows "Starting..." ---

func TestIntegration_ViewBeforeWindowSize(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	fake := &executor.FakeExecutor{Events: events}
	sess := model.Session{
		Mode:          "build",
		PromptFile:    "test.md",
		MaxIterations: 1,
		StartTime:     time.Now(),
	}
	cfg := config.DefaultConfig()
	th := testTheme()
	m := NewModel(sess, cfg, "prompt", th, false, false, fake)
	// Don't set width/height — default is 0.

	view := m.View()
	if view != "Starting..." {
		t.Errorf("expected 'Starting...' before window size, got %q", view)
	}
}

// --- h/l arrow keys switch panes ---

func TestIntegration_HLSwitchesPanes(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// l switches to right pane.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if m.focusedPane != rightPane {
		t.Error("expected right pane after l")
	}

	// h switches to left pane.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if m.focusedPane != leftPane {
		t.Error("expected left pane after h")
	}
}

// --- Mixed tool types don't group ---

func TestIntegration_MixedToolsNoGrouping(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "a.go"},
			session.ToolUseEvent{ID: "tc2", Name: "Edit", Summary: "a.go (+1/-1)"},
			session.ToolUseEvent{ID: "tc3", Name: "Read", Summary: "b.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.ToolResultEvent{ToolUseID: "tc2", IsError: false},
		session.ToolResultEvent{ToolUseID: "tc3", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	iter := m.Session().Iterations[0]

	// Read, Edit, Read → 3 standalone tool calls (no grouping since
	// the same-name runs are only 1 each).
	if len(iter.Items) != 3 {
		t.Fatalf("expected 3 items (no grouping), got %d", len(iter.Items))
	}

	for i, item := range iter.Items {
		if _, ok := item.(*model.ToolCall); !ok {
			t.Errorf("item %d: expected ToolCall, got %T", i, item)
		}
	}
}

// --- Subprocess failure marks iteration as failed ---

func TestIntegration_SubprocessFailure(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Bash", Summary: "cmd"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: &testError{"process killed"}},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	iter := m.Session().Iterations[0]
	if iter.Status != model.IterationFailed {
		t.Errorf("expected IterationFailed, got %d", iter.Status)
	}
}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
