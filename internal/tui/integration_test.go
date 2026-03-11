package tui

import (
	"fmt"
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
	return newTestModelWithExit(events, maxIterations, false)
}

// newTestModelWithExit creates a test Model with configurable exitOnComplete.
func newTestModelWithExit(events []session.Event, maxIterations int, exitOnComplete bool) *Model {
	fake := &executor.FakeExecutor{Events: events}
	sess := model.Session{
		Mode:          "build",
		PromptFile:    "test-prompt.md",
		MaxIterations: maxIterations,
		StartTime:     time.Now(),
	}
	cfg := config.DefaultConfig()
	th := testTheme()
	m := NewModel(sess, cfg, "test prompt content", th, false, exitOnComplete, fake)
	m.width = 120
	m.height = 40
	return &m
}

// drainEvents runs Init() and then pumps all messages from the eventCh through
// Update() until the channel is drained and no more commands produce messages.
// This simulates a full Bubble Tea event loop for the given canned events.
func drainEvents(t *testing.T, m *Model) {
	t.Helper()
	drainEventsTrackQuit(t, m)
}

// drainEventsTrackQuit is like drainEvents but returns true if any Update()
// call produced a tea.QuitMsg. This verifies that the model actually produces
// tea.Quit (not just sets m.quitting=true).
func drainEventsTrackQuit(t *testing.T, m *Model) bool {
	t.Helper()

	// Run Init to start the first iteration and event forwarding.
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil cmd")
	}

	// Execute all batched commands and collect messages.
	msgs := executeBatchCmd(cmd)
	quitReturned := false

	// Process messages until we've drained everything.
	for len(msgs) > 0 {
		var nextMsgs []tea.Msg
		for _, msg := range msgs {
			// Skip tick messages — they cause infinite loops.
			if _, ok := msg.(tickMsg); ok {
				continue
			}
			// Check if we received a QuitMsg (produced by tea.Quit).
			if _, ok := msg.(tea.QuitMsg); ok {
				quitReturned = true
				continue
			}
			_, cmd = m.Update(msg)
			if cmd != nil {
				nextMsgs = append(nextMsgs, executeBatchCmd(cmd)...)
			}
		}
		msgs = nextMsgs
	}

	return quitReturned
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
			session.ToolUseEvent{ID: "tu2", Name: "Edit", Summary: "main.go (+2/-2)"},
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

	// Starts focused on iterations pane.
	if m.focusedPane != iterationsPane {
		t.Error("expected initial focus on iterations pane")
	}

	// First Tab switches to prompts pane.
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focusedPane != promptsPane {
		t.Error("expected prompts pane focus after first tab")
	}

	// Second Tab switches to right pane.
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focusedPane != rightPane {
		t.Error("expected right pane focus after second tab")
	}

	// Third Tab cycles to plans pane.
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focusedPane != plansPane {
		t.Error("expected plans pane focus after third tab")
	}

	// Fourth Tab cycles back to iterations pane.
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focusedPane != iterationsPane {
		t.Error("expected iterations pane focus after fourth tab")
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
	if m.focusedPane != iterationsPane {
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
	if m.focusedPane != iterationsPane {
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

	// Focus right pane (Tab twice: iterations → prompts → right).
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
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

	// gg: first g sets pending action, second g jumps to top.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.pendingAction == "" {
		t.Error("expected pendingAction set after first g")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.pendingAction != "" {
		t.Error("expected pendingAction cleared after second g")
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
	if m.focusedPane != iterationsPane {
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

// --- Mouse click switches pane focus ---

func TestIntegration_MouseClickSwitchesFocus(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "main.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Starts on left pane.
	if m.focusedPane != iterationsPane {
		t.Fatal("expected initial focus on left pane")
	}

	// Click on right pane (X >= 32) switches focus.
	m.Update(tea.MouseMsg{
		X:      50,
		Y:      2, // pane row 1 (subtract 1 for header)
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	if m.focusedPane != rightPane {
		t.Error("expected right pane focus after clicking right pane")
	}

	// Click on left pane iteration section (X < 32, Y=7 to skip header + plan section + divider).
	m.Update(tea.MouseMsg{
		X:      10,
		Y:      7, // header=0, plan section=1-5, divider=6, iterations start at 7
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	if m.focusedPane != iterationsPane {
		t.Error("expected iterations pane focus after clicking iteration section")
	}
}

// --- Mouse click on header is ignored ---

func TestIntegration_MouseClickOnHeaderIgnored(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "main.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Click on header row (Y=0) should be ignored, focus stays on left.
	m.Update(tea.MouseMsg{
		X:      50,
		Y:      0,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	if m.focusedPane != iterationsPane {
		t.Error("expected focus unchanged after clicking header")
	}
}

// --- Mouse scroll switches focus and scrolls ---

func TestIntegration_MouseScrollSwitchesFocus(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "main.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Scroll wheel on right pane should switch focus.
	m.Update(tea.MouseMsg{
		X:      50,
		Y:      5,
		Button: tea.MouseButtonWheelDown,
	})
	if m.focusedPane != rightPane {
		t.Error("expected right pane focus after scrolling on right pane")
	}
}

// --- Mouse click on left pane selects iteration and resets timeline ---

func TestIntegration_MouseClickSelectsIteration(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Bash", Summary: "cmd"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 3)
	drainEvents(t, m)

	// Cursor should be at 2 (last iteration, auto-follow).
	if m.iterList.Cursor != 2 {
		t.Fatalf("expected cursor=2, got %d", m.iterList.Cursor)
	}

	// Move timeline cursor to verify it gets reset.
	m.focusedPane = rightPane
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.timeline.Cursor == 0 {
		// Move further if needed
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	}

	// Click on first iteration (Y=7: header=0, plan section=1-5, divider=6, first iter=7).
	m.Update(tea.MouseMsg{
		X:      10,
		Y:      7, // header + plan section + divider, then first iteration row
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})

	if m.focusedPane != iterationsPane {
		t.Error("expected iterations pane focus after clicking iteration section")
	}
	if m.iterList.Cursor != 0 {
		t.Errorf("expected cursor=0 after clicking first iteration, got %d", m.iterList.Cursor)
	}
	// Timeline should be reset when changing iteration.
	if m.timeline.Cursor != 0 {
		t.Errorf("expected timeline cursor=0 after clicking different iteration, got %d", m.timeline.Cursor)
	}
}

// --- Expanding a standalone tool call via Enter shows expanded content ---

func TestIntegration_ExpandToolCallShowsContent(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{
				ID:       "tc1",
				Name:     "Bash",
				Summary:  "go test ./...",
				RawInput: map[string]interface{}{"command": "go test ./..."},
			},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false, Content: "ok  mypackage 0.5s"},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Focus right pane where the tool call is (Tab twice: iterations → prompts → right).
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focusedPane != rightPane {
		t.Fatal("expected right pane focus")
	}

	// Tool call should start collapsed.
	iter := &m.Session().Iterations[0]
	tc := iter.Items[0].(*model.ToolCall)
	if tc.Expanded {
		t.Error("expected tool call to start collapsed")
	}

	// View should not show expanded content.
	view := m.View()
	if strings.Contains(view, "$ go test") {
		t.Error("expected no expanded content before Enter")
	}

	// Press Enter to expand.
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !tc.Expanded {
		t.Error("expected tool call to be expanded after Enter")
	}

	// View should now show expanded content (Bash shows "$ command" header).
	view = m.View()
	if !strings.Contains(view, "$ go test") {
		t.Error("expected '$ go test' in expanded view")
	}
	if !strings.Contains(view, "ok  mypackage") {
		t.Error("expected command output in expanded view")
	}

	// Press Enter again to collapse.
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if tc.Expanded {
		t.Error("expected tool call to be collapsed after second Enter")
	}

	// View should no longer show expanded content.
	view = m.View()
	if strings.Contains(view, "$ go test") {
		t.Error("expected expanded content hidden after collapse")
	}
}

// --- Left pane auto-hides below 80 columns ---

func TestIntegration_LeftPaneHiddenBelow80Cols(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "main.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// At 120 columns, left pane should be visible.
	if !m.leftPaneVisible {
		t.Error("expected left pane visible at 120 columns")
	}

	// Resize to 79 columns — left pane should auto-hide.
	m.Update(tea.WindowSizeMsg{Width: 79, Height: 30})
	if m.leftPaneVisible {
		t.Error("expected left pane hidden at 79 columns")
	}

	// View should not contain the separator.
	view := m.View()
	if strings.Contains(view, "│") {
		t.Error("expected no separator when left pane is hidden")
	}

	// Resize back to 80 columns — left pane should reappear.
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	if !m.leftPaneVisible {
		t.Error("expected left pane visible at 80 columns")
	}
}

// --- Toggle left pane with [ key ---

func TestIntegration_ToggleLeftPaneWithBracket(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	if !m.leftPaneVisible {
		t.Fatal("expected left pane visible initially")
	}

	// [ hides the left pane.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	if m.leftPaneVisible {
		t.Error("expected left pane hidden after [")
	}

	// [ again shows the left pane.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	if !m.leftPaneVisible {
		t.Error("expected left pane visible after second [")
	}
}

// --- Focus auto-switches to right pane when left pane hides ---

func TestIntegration_FocusPreservedWhenLayoutSwitches(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Start on iterations pane.
	if m.focusedPane != iterationsPane {
		t.Fatal("expected iterations pane focus")
	}

	// Resize to narrow — auto layout switches to bottom.
	// Focus is preserved: iterations pane still exists in the bottom bar.
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 30})
	if m.focusedPane != iterationsPane {
		t.Error("expected focus preserved on iterations pane after layout switch to bottom")
	}
	if m.leftPaneVisible {
		t.Error("expected leftPaneVisible=false in bottom layout")
	}
}

// --- Focus auto-switches when toggling left pane off while focused ---

func TestIntegration_FocusSwitchesWhenTogglingLeftPaneOff(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Start on left pane.
	if m.focusedPane != iterationsPane {
		t.Fatal("expected left pane focus")
	}

	// Toggle left pane off — focus should switch to right.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	if m.focusedPane != rightPane {
		t.Error("expected right pane focus after toggling left pane off")
	}
}

// --- Tab doesn't toggle to hidden left pane ---

func TestIntegration_TabSkipsHiddenLeftPane(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Hide left pane and switch to right pane.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	if m.focusedPane != rightPane {
		t.Fatal("expected right pane focus")
	}

	// Tab should not switch to hidden left pane.
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focusedPane != rightPane {
		t.Error("expected right pane focus after tab with hidden left pane")
	}
}

// --- h key doesn't focus hidden left pane ---

func TestIntegration_HKeySkipsHiddenLeftPane(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Hide left pane and switch to right.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	m.focusedPane = rightPane

	// h should not switch to hidden left pane.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if m.focusedPane != rightPane {
		t.Error("expected right pane focus after h with hidden left pane")
	}
}

// --- Mouse click targets right pane when left pane hidden ---

func TestIntegration_MouseClickRightPaneWhenLeftHidden(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "main.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Hide left pane.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})

	// Click at X=5 — should target right pane since left is hidden.
	m.Update(tea.MouseMsg{
		X:      5,
		Y:      2,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	if m.focusedPane != rightPane {
		t.Error("expected right pane focus after clicking when left pane is hidden")
	}
}

// --- --exit flag: single iteration produces tea.Quit ---

func TestIntegration_ExitFlag_SingleIteration(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Bash", Summary: "make test"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModelWithExit(events, 1, true)
	quitReturned := drainEventsTrackQuit(t, m)

	if !m.quitting {
		t.Error("expected quitting=true with --exit flag")
	}
	if !quitReturned {
		t.Error("expected tea.Quit cmd to be returned with --exit flag")
	}
	if m.activeModal != modalNone {
		t.Error("expected no modal with --exit flag")
	}
	if len(m.Session().Iterations) != 1 {
		t.Errorf("expected 1 iteration, got %d", len(m.Session().Iterations))
	}
	if m.Session().Iterations[0].Status != model.IterationCompleted {
		t.Errorf("expected iteration completed, got %d", m.Session().Iterations[0].Status)
	}
}

// --- --exit flag: multi-iteration completes all iterations then produces tea.Quit ---

func TestIntegration_ExitFlag_MultiIteration(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Bash", Summary: "make test"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModelWithExit(events, 3, true)
	quitReturned := drainEventsTrackQuit(t, m)

	if !m.quitting {
		t.Error("expected quitting=true after all iterations with --exit flag")
	}
	if !quitReturned {
		t.Error("expected tea.Quit cmd after all iterations complete")
	}

	// All 3 iterations should have completed before exiting.
	sess := m.Session()
	if len(sess.Iterations) != 3 {
		t.Fatalf("expected 3 iterations, got %d", len(sess.Iterations))
	}
	for i, iter := range sess.Iterations {
		if iter.Status != model.IterationCompleted {
			t.Errorf("iteration %d: expected completed, got %d", i, iter.Status)
		}
	}
}

// --- --exit flag: exits after last iteration fails ---

func TestIntegration_ExitFlag_LastIterationFails(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Bash", Summary: "make test"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: true},
		session.SubprocessExitEvent{Err: &testError{"process exited with code 1"}},
	}

	m := newTestModelWithExit(events, 1, true)
	quitReturned := drainEventsTrackQuit(t, m)

	if !m.quitting {
		t.Error("expected quitting=true with --exit flag after failure")
	}
	if !quitReturned {
		t.Error("expected tea.Quit cmd even when last iteration fails")
	}

	iter := m.Session().Iterations[0]
	if iter.Status != model.IterationFailed {
		t.Errorf("expected IterationFailed, got %d", iter.Status)
	}
}

// --- --exit flag: without flag, model stays alive after iterations complete ---

func TestIntegration_NoExitFlag_StaysAlive(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	quitReturned := drainEventsTrackQuit(t, m)

	if m.quitting {
		t.Error("expected quitting=false without --exit flag")
	}
	if quitReturned {
		t.Error("expected no tea.Quit cmd without --exit flag")
	}
}

// --- # toggles line numbers ---

func TestIntegration_HashTogglesLineNumbers(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Default config has line_numbers = true
	if !m.lineNumbers {
		t.Error("expected line numbers on by default")
	}

	// # toggles off
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'#'}})
	if m.lineNumbers {
		t.Error("expected line numbers off after #")
	}

	// # toggles back on
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'#'}})
	if !m.lineNumbers {
		t.Error("expected line numbers on after second #")
	}
}

// --- Count+jump motions ---

func TestIntegration_CountJump_5jMovesDown5(t *testing.T) {
	// Create enough items for count+jump testing.
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.TextEvent{Text: "item 1"},
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "a.go"},
			session.ToolUseEvent{ID: "tc2", Name: "Read", Summary: "b.go"},
			session.ToolUseEvent{ID: "tc3", Name: "Edit", Summary: "c.go"},
			session.ToolUseEvent{ID: "tc4", Name: "Bash", Summary: "test"},
			session.TextEvent{Text: "item 6"},
			session.ToolUseEvent{ID: "tc5", Name: "Read", Summary: "d.go"},
			session.ToolUseEvent{ID: "tc6", Name: "Read", Summary: "e.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.ToolResultEvent{ToolUseID: "tc2", IsError: false},
		session.ToolResultEvent{ToolUseID: "tc3", IsError: false},
		session.ToolResultEvent{ToolUseID: "tc4", IsError: false},
		session.ToolResultEvent{ToolUseID: "tc5", IsError: false},
		session.ToolResultEvent{ToolUseID: "tc6", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Focus right pane (Tab twice: iterations → prompts → right).
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focusedPane != rightPane {
		t.Fatal("expected right pane focus")
	}

	// Move to top first.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.timeline.Cursor != 0 {
		t.Fatalf("expected cursor=0 at top, got %d", m.timeline.Cursor)
	}

	// Type "5j" — should move down 5 items.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	if m.timeline.CountBuffer != "5" {
		t.Errorf("expected CountBuffer='5', got %q", m.timeline.CountBuffer)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.timeline.Cursor != 5 {
		t.Errorf("expected cursor=5 after 5j, got %d", m.timeline.Cursor)
	}
	if m.timeline.CountBuffer != "" {
		t.Errorf("expected count buffer cleared after j, got %q", m.timeline.CountBuffer)
	}
}

func TestIntegration_CountJump_12kMovesUp12(t *testing.T) {
	// Create enough items.
	var evts []session.Event
	var toolUses []session.Event
	var toolResults []session.Event
	for i := 0; i < 15; i++ {
		id := fmt.Sprintf("tc%d", i)
		toolUses = append(toolUses, session.ToolUseEvent{
			ID: id, Name: "Read", Summary: fmt.Sprintf("file%d.go", i),
		})
		toolResults = append(toolResults, session.ToolResultEvent{
			ToolUseID: id, IsError: false,
		})
	}
	evts = append(evts, session.AssistantBatchEvent{Events: toolUses})
	evts = append(evts, toolResults...)
	evts = append(evts, session.SubprocessExitEvent{Err: nil})

	m := newTestModel(evts, 1)
	drainEvents(t, m)

	// Focus right pane (Tab twice: iterations → prompts → right).
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m.Update(tea.KeyMsg{Type: tea.KeyTab})

	// Jump to bottom.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})

	// The items are grouped (15 consecutive Reads), so there's 1 group.
	// We need individual items for count testing. Let's use the cursor position.
	cursorAtBottom := m.timeline.Cursor

	// Type "12k" — should move up 12 items (or clamp).
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	if m.timeline.CountBuffer != "12" {
		t.Errorf("expected CountBuffer='12', got %q", m.timeline.CountBuffer)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	expected := cursorAtBottom - 12
	if expected < 0 {
		expected = 0
	}
	if m.timeline.Cursor != expected {
		t.Errorf("expected cursor=%d after 12k from %d, got %d", expected, cursorAtBottom, m.timeline.Cursor)
	}
}

func TestIntegration_CountJump_NoPrefix_MovesOne(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.TextEvent{Text: "first"},
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "a.go"},
			session.TextEvent{Text: "third"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Focus right pane (Tab twice: iterations → prompts → right), jump to top.
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})

	// j without count moves 1.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.timeline.Cursor != 1 {
		t.Errorf("expected cursor=1 after j without count, got %d", m.timeline.Cursor)
	}
}

func TestIntegration_CountJump_BufferClearsOnOtherKeys(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.TextEvent{Text: "item"},
		}},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Focus right pane (Tab twice: iterations → prompts → right).
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m.Update(tea.KeyMsg{Type: tea.KeyTab})

	// Type "5" to start count buffer.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	if m.timeline.CountBuffer != "5" {
		t.Errorf("expected CountBuffer='5', got %q", m.timeline.CountBuffer)
	}

	// Press v (toggle view) — should clear the count buffer.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if m.timeline.CountBuffer != "" {
		t.Errorf("expected count buffer cleared after v, got %q", m.timeline.CountBuffer)
	}
}

func TestIntegration_CountJump_LeadingZeroIgnored(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.TextEvent{Text: "item 1"},
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "a.go"},
			session.TextEvent{Text: "item 3"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Focus right pane.
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})

	// Type "05j" — leading 0 is ignored, so it's "5j".
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	if m.timeline.CountBuffer != "" {
		t.Errorf("expected empty CountBuffer after leading 0, got %q", m.timeline.CountBuffer)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	// Should move down by 5 (clamped to max 2 since only 3 items)
	if m.timeline.Cursor != 2 {
		t.Errorf("expected cursor=2 after 05j (clamped), got %d", m.timeline.Cursor)
	}
}

func TestIntegration_CountJump_DigitsOnlyOnRightPane(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "a.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Focus left pane (default).
	if m.focusedPane != iterationsPane {
		t.Fatal("expected left pane focus")
	}

	// Type "5" on left pane — should NOT accumulate count buffer.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	if m.timeline.CountBuffer != "" {
		t.Errorf("expected no count buffer on left pane, got %q", m.timeline.CountBuffer)
	}
}

func TestIntegration_CountJump_PendingCountInView(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.TextEvent{Text: "some text"},
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "a.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Focus right pane (Tab twice: iterations → prompts → right) and type a count.
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'7'}})

	view := m.View()
	if !strings.Contains(view, "7") {
		t.Error("expected pending count '7' visible in view")
	}
}

// --- Help modal open/close ---

func TestIntegration_HelpModal_OpenClose(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "main.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// No modal initially.
	if m.activeModal != modalNone {
		t.Fatal("expected no modal initially")
	}

	// ? opens the help modal.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.activeModal != modalHelp {
		t.Error("expected help modal after pressing ?")
	}

	// View should contain help modal content.
	view := m.View()
	if !strings.Contains(view, "Keybindings") {
		t.Error("expected 'Keybindings' title in help modal view")
	}
	if !strings.Contains(view, "Navigation") {
		t.Error("expected 'Navigation' section in help modal view")
	}
	if !strings.Contains(view, "Move down") {
		t.Error("expected 'Move down' entry in help modal view")
	}
	if !strings.Contains(view, "Press any key to close") {
		t.Error("expected dismiss instructions in help modal view")
	}

	// Any key dismisses the help modal.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if m.activeModal != modalNone {
		t.Error("expected help modal dismissed after pressing any key")
	}
}

func TestIntegration_HelpModal_ReflectsCustomKeybindings(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	// Create model with custom keybindings.
	fake := &executor.FakeExecutor{Events: events}
	sess := model.Session{
		Mode:          "build",
		PromptFile:    "test.md",
		MaxIterations: 1,
		StartTime:     time.Now(),
	}
	cfg := config.DefaultConfig()
	// Remap quit to "x" instead of "q".
	cfg.KeyMap.Bindings[config.ActionQuit] = config.KeyBinding{Keys: []string{"x"}}
	th := testTheme()
	m := NewModel(sess, cfg, "prompt", th, false, false, fake)
	m.width = 120
	m.height = 30
	drainEvents(t, &m)

	// Open help modal.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.activeModal != modalHelp {
		t.Fatal("expected help modal")
	}

	// Help modal should show "x" for quit, not "q".
	view := m.View()
	if !strings.Contains(view, "Quit") {
		t.Error("expected 'Quit' label in help modal")
	}
	// The custom binding "x" should appear in the view.
	// Since the view contains styled text, we check for the key character.
	if !strings.Contains(view, "x") {
		t.Error("expected custom quit key 'x' in help modal view")
	}
}

// --- Sub-scroll: enter, navigate, exit ---

func TestIntegration_SubScroll_EnterNavigateExit(t *testing.T) {
	// Create a tool call with large content that will trigger sub-scroll.
	// Sub-scroll activates when content > 40% of pane height.
	// With height=30, pane height ~29, 40% = ~11 lines. We need more.
	var largeOutput strings.Builder
	for i := 0; i < 50; i++ {
		fmt.Fprintf(&largeOutput, "output line %d\n", i+1)
	}

	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{
				ID:       "tc1",
				Name:     "Bash",
				Summary:  "long command",
				RawInput: map[string]interface{}{"command": "long-cmd"},
			},
		}},
		session.ToolResultEvent{
			ToolUseID: "tc1",
			IsError:   false,
			Content:   largeOutput.String(),
		},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Focus right pane (Tab twice: iterations → prompts → right).
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focusedPane != rightPane {
		t.Fatal("expected right pane focus")
	}

	// Tool call should start collapsed.
	iter := &m.Session().Iterations[0]
	tc := iter.Items[0].(*model.ToolCall)
	if tc.Expanded {
		t.Fatal("expected tool call collapsed initially")
	}
	if m.timeline.InSubScroll() {
		t.Fatal("expected not in sub-scroll initially")
	}

	// First Enter: expand the tool call.
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !tc.Expanded {
		t.Fatal("expected tool call expanded after first Enter")
	}
	if m.timeline.InSubScroll() {
		t.Fatal("expected not in sub-scroll after first Enter (just expanded)")
	}

	// Second Enter on expanded tool call with large content: enter sub-scroll.
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.timeline.InSubScroll() {
		t.Fatal("expected sub-scroll mode after second Enter on large expanded content")
	}
	if m.timeline.SubScrollOffset != 0 {
		t.Errorf("expected SubScrollOffset=0 at start, got %d", m.timeline.SubScrollOffset)
	}

	// j scrolls down in sub-scroll.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.timeline.SubScrollOffset != 1 {
		t.Errorf("expected SubScrollOffset=1 after j, got %d", m.timeline.SubScrollOffset)
	}

	// k scrolls back up in sub-scroll.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.timeline.SubScrollOffset != 0 {
		t.Errorf("expected SubScrollOffset=0 after k, got %d", m.timeline.SubScrollOffset)
	}

	// G jumps to bottom of sub-scroll.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if m.timeline.SubScrollOffset == 0 {
		t.Error("expected SubScrollOffset > 0 after G (jump to bottom)")
	}

	// gg jumps to top of sub-scroll.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.timeline.SubScrollOffset != 0 {
		t.Errorf("expected SubScrollOffset=0 after gg, got %d", m.timeline.SubScrollOffset)
	}

	// Escape exits sub-scroll.
	m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.timeline.InSubScroll() {
		t.Error("expected sub-scroll exited after Escape")
	}
	// Tool call should still be expanded after exiting sub-scroll.
	if !tc.Expanded {
		t.Error("expected tool call still expanded after Escape exits sub-scroll")
	}
}

func TestIntegration_SubScroll_EnterCollapses(t *testing.T) {
	// Sub-scroll: pressing Enter should collapse and exit.
	var largeOutput strings.Builder
	for i := 0; i < 50; i++ {
		fmt.Fprintf(&largeOutput, "output line %d\n", i+1)
	}

	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{
				ID:       "tc1",
				Name:     "Bash",
				Summary:  "long command",
				RawInput: map[string]interface{}{"command": "long-cmd"},
			},
		}},
		session.ToolResultEvent{
			ToolUseID: "tc1",
			IsError:   false,
			Content:   largeOutput.String(),
		},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Focus right pane (Tab twice: iterations → prompts → right) and expand.
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m.Update(tea.KeyMsg{Type: tea.KeyTab})

	iter := &m.Session().Iterations[0]
	tc := iter.Items[0].(*model.ToolCall)

	// Expand, then enter sub-scroll.
	m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // expand
	m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // enter sub-scroll
	if !m.timeline.InSubScroll() {
		t.Fatal("expected sub-scroll mode")
	}

	// Enter in sub-scroll collapses and exits.
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.timeline.InSubScroll() {
		t.Error("expected sub-scroll exited after Enter")
	}
	if tc.Expanded {
		t.Error("expected tool call collapsed after Enter in sub-scroll")
	}
}

// --- Configurable keybindings apply end-to-end ---

func TestIntegration_CustomKeybindings_EndToEnd(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.TextEvent{Text: "first item"},
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "a.go"},
			session.TextEvent{Text: "third item"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	// Create model with custom keybindings: remap move_down to "n",
	// move_up to "p", quit to "x", toggle_view to "m".
	fake := &executor.FakeExecutor{Events: events}
	sess := model.Session{
		Mode:          "build",
		PromptFile:    "test.md",
		MaxIterations: 1,
		StartTime:     time.Now(),
	}
	cfg := config.DefaultConfig()
	cfg.KeyMap.Bindings[config.ActionMoveDown] = config.KeyBinding{Keys: []string{"n"}}
	cfg.KeyMap.Bindings[config.ActionMoveUp] = config.KeyBinding{Keys: []string{"p"}}
	cfg.KeyMap.Bindings[config.ActionPlanMode] = config.KeyBinding{Keys: []string{"P"}} // avoid conflict with custom move_up
	cfg.KeyMap.Bindings[config.ActionQuit] = config.KeyBinding{Keys: []string{"x"}}
	cfg.KeyMap.Bindings[config.ActionToggleView] = config.KeyBinding{Keys: []string{"m"}}
	th := testTheme()
	m := NewModel(sess, cfg, "prompt", th, false, false, fake)
	m.width = 120
	m.height = 30
	drainEvents(t, &m)

	// Focus right pane (Tab twice: iterations → prompts → right).
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focusedPane != rightPane {
		t.Fatal("expected right pane focus")
	}

	// Jump to top first.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.timeline.Cursor != 0 {
		t.Fatalf("expected cursor=0 at top, got %d", m.timeline.Cursor)
	}

	// "n" (custom move_down) should move cursor down.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.timeline.Cursor != 1 {
		t.Errorf("expected cursor=1 after custom move_down 'n', got %d", m.timeline.Cursor)
	}

	// "p" (custom move_up) should move cursor up.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if m.timeline.Cursor != 0 {
		t.Errorf("expected cursor=0 after custom move_up 'p', got %d", m.timeline.Cursor)
	}

	// "j" (old default move_down) should NOT move cursor — it's no longer bound.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.timeline.Cursor != 0 {
		t.Errorf("expected cursor=0 after unbound 'j', got %d", m.timeline.Cursor)
	}

	// "m" (custom toggle_view) should toggle compact view.
	if m.compactView {
		t.Fatal("expected compact view off initially")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if !m.compactView {
		t.Error("expected compact view on after custom toggle 'm'")
	}

	// "v" (old default toggle_view) should NOT toggle — it's no longer bound.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if !m.compactView {
		t.Error("expected compact view still on after unbound 'v'")
	}

	// "x" (custom quit) should show quit modal.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if m.activeModal != modalQuitConfirm {
		t.Error("expected quit modal after custom quit key 'x'")
	}

	// Dismiss quit modal.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.activeModal != modalNone {
		t.Error("expected modal dismissed")
	}

	// "q" (old default quit) should NOT show quit modal — it's no longer bound.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if m.activeModal != modalNone {
		t.Error("expected no modal after unbound 'q'")
	}
}

func TestIntegration_CustomKeybindings_CustomSequence(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Bash", Summary: "cmd"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	// Remap jump_top from "g g" to "z z".
	fake := &executor.FakeExecutor{Events: events}
	sess := model.Session{
		Mode:          "build",
		PromptFile:    "test.md",
		MaxIterations: 3,
		StartTime:     time.Now(),
	}
	cfg := config.DefaultConfig()
	cfg.KeyMap.Bindings[config.ActionJumpTop] = config.KeyBinding{Keys: []string{"z", "z"}}
	th := testTheme()
	m := NewModel(sess, cfg, "prompt", th, false, false, fake)
	m.width = 120
	m.height = 30
	drainEvents(t, &m)

	// Cursor at last iteration (auto-follow).
	if m.iterList.Cursor != 2 {
		t.Fatalf("expected cursor=2, got %d", m.iterList.Cursor)
	}

	// "z z" (custom jump_top) should jump to top.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	if m.pendingAction == "" {
		t.Error("expected pendingAction set after first 'z'")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	if m.iterList.Cursor != 0 {
		t.Errorf("expected cursor=0 after custom zz, got %d", m.iterList.Cursor)
	}

	// "g g" (old default) should NOT jump to top.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}) // move down first
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.iterList.Cursor == 0 {
		t.Error("expected 'gg' to NOT jump to top since it's remapped to 'zz'")
	}
}

// --- Plan pane integration tests ---

func TestIntegration_PlanSelectionSwapsRightPane(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "main.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Initially in timeline mode on iterations pane
	if m.rightPaneMode != timelineMode {
		t.Error("expected initial rightPaneMode=timelineMode")
	}

	// Tab to prompts, then right pane, then plans
	m.Update(tea.KeyMsg{Type: tea.KeyTab}) // → prompts
	m.Update(tea.KeyMsg{Type: tea.KeyTab}) // → right pane
	m.Update(tea.KeyMsg{Type: tea.KeyTab}) // → plans

	if m.focusedPane != plansPane {
		t.Fatal("expected plans pane focus")
	}
	if m.rightPaneMode != planMode {
		t.Error("expected rightPaneMode=planMode after focusing plans")
	}

	// Tab back to iterations
	m.Update(tea.KeyMsg{Type: tea.KeyTab}) // → iterations

	if m.focusedPane != iterationsPane {
		t.Fatal("expected iterations pane focus")
	}
	if m.rightPaneMode != timelineMode {
		t.Error("expected rightPaneMode=timelineMode after focusing iterations")
	}
}

func TestIntegration_PlanFocusRestoresTimeline(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "main.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Switch to plan mode
	m.focusedPane = plansPane
	m.rightPaneMode = planMode

	// Use 'l' to go to right pane (still in plan mode)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if m.focusedPane != rightPane {
		t.Fatal("expected right pane focus after 'l'")
	}
	if m.rightPaneMode != planMode {
		t.Error("expected rightPaneMode=planMode after 'l' from plans")
	}

	// Use 'h' to go back to plans
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if m.focusedPane != plansPane {
		t.Error("expected plans pane focus after 'h' from plan content")
	}

	// Now tab to iterations — right pane should switch to timeline
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.rightPaneMode != timelineMode {
		t.Error("expected rightPaneMode=timelineMode after tabbing to iterations")
	}
}

func TestIntegration_PlanScrollResetOnCursorChange(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "main.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Set up plan list with multiple files
	m.planList.Files = []string{"A_PLAN.md", "B_PLAN.md"}
	m.planList.Cursor = 0
	m.focusedPane = plansPane
	m.rightPaneMode = planMode
	m.planViewScroll = 10

	// Move cursor down — scroll should reset
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.planList.Cursor != 1 {
		t.Errorf("expected cursor=1, got %d", m.planList.Cursor)
	}
	if m.planViewScroll != 0 {
		t.Errorf("expected planViewScroll=0 after cursor change, got %d", m.planViewScroll)
	}
}

func TestIntegration_PlanEditorKey(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "main.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Set up plan list
	m.planList.Files = []string{"TEST_PLAN.md"}
	m.planList.Cursor = 0
	m.focusedPane = plansPane
	m.rightPaneMode = planMode

	// Press 'e' — should return a command (editor launch)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd == nil {
		t.Error("expected non-nil command from 'e' key (editor launch)")
	}
}

func TestIntegration_PlanEditorKeyFromRightPane(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "main.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Set up plan list and focus right pane in plan mode
	m.planList.Files = []string{"TEST_PLAN.md"}
	m.planList.Cursor = 0
	m.focusedPane = rightPane
	m.rightPaneMode = planMode

	// Press 'e' — should return a command (editor launch)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd == nil {
		t.Error("expected non-nil command from 'e' key in plan content view")
	}
}

func TestIntegration_PlanEditorKeyNotFromTimeline(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "main.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Focus right pane in timeline mode — 'e' should not launch editor
	m.focusedPane = rightPane
	m.rightPaneMode = timelineMode

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd != nil {
		t.Error("expected nil command from 'e' key in timeline mode")
	}
}

func TestIntegration_PlanViewRendersInRightPane(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "main.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Switch to plan mode — right pane should show plan content
	m.focusedPane = plansPane
	m.rightPaneMode = planMode

	view := m.View()
	// Plan list title should appear in the left pane
	if !strings.Contains(view, "Plans") {
		t.Error("expected 'Plans' title in view")
	}
}

func TestIntegration_PlanNavInRightPane(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "main.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Set up state for plan content scrolling
	m.focusedPane = rightPane
	m.rightPaneMode = planMode
	m.planViewTotalLines = 100
	m.planViewScroll = 0

	// j should scroll down
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.planViewScroll != 1 {
		t.Errorf("expected planViewScroll=1 after 'j', got %d", m.planViewScroll)
	}

	// k should scroll up
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.planViewScroll != 0 {
		t.Errorf("expected planViewScroll=0 after 'k', got %d", m.planViewScroll)
	}

	// gg should jump to top
	m.planViewScroll = 50
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.planViewScroll != 0 {
		t.Errorf("expected planViewScroll=0 after 'gg', got %d", m.planViewScroll)
	}
}

func TestIntegration_PlanMouseClickSelectsPlan(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "main.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Set up plan list
	m.planList.Files = []string{"A_PLAN.md", "B_PLAN.md", "C_PLAN.md"}
	m.planList.Cursor = 0

	// Click on second plan file (row 2 in plan section: title=0, first file=1, second file=2)
	// Y=3 (header=0, pane starts at 1, plan title=1, first file=2, second file=3)
	m.Update(tea.MouseMsg{
		X:      10,
		Y:      3, // header + plan title row + first file row = second file row
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})

	if m.focusedPane != plansPane {
		t.Error("expected plans pane focus after clicking plan section")
	}
	if m.planList.Cursor != 1 {
		t.Errorf("expected cursor=1 after clicking second plan, got %d", m.planList.Cursor)
	}
	if m.rightPaneMode != planMode {
		t.Error("expected rightPaneMode=planMode after clicking plan section")
	}
}

func TestIntegration_PlanMouseWheelScrollsPlanContent(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "main.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Set up plan mode with enough content to scroll
	m.focusedPane = rightPane
	m.rightPaneMode = planMode
	m.planViewTotalLines = 100
	m.planViewScroll = 0

	// Scroll down on right pane — should increase planViewScroll
	m.Update(tea.MouseMsg{
		X:      50,
		Y:      5,
		Button: tea.MouseButtonWheelDown,
	})

	if m.focusedPane != rightPane {
		t.Error("expected right pane focus after scrolling right pane")
	}
	if m.planViewScroll != 3 {
		t.Errorf("expected planViewScroll=3 after wheel down, got %d", m.planViewScroll)
	}

	// Scroll up — should decrease planViewScroll
	m.Update(tea.MouseMsg{
		X:      50,
		Y:      5,
		Button: tea.MouseButtonWheelUp,
	})

	if m.planViewScroll != 0 {
		t.Errorf("expected planViewScroll=0 after wheel up, got %d", m.planViewScroll)
	}

	// Verify clamping at 0 — scroll up beyond top
	m.Update(tea.MouseMsg{
		X:      50,
		Y:      5,
		Button: tea.MouseButtonWheelUp,
	})

	if m.planViewScroll != 0 {
		t.Errorf("expected planViewScroll clamped at 0, got %d", m.planViewScroll)
	}

	// Verify clamping at max — scroll to near the end then past it
	m.planViewScroll = 90
	m.Update(tea.MouseMsg{
		X:      50,
		Y:      5,
		Button: tea.MouseButtonWheelDown,
	})

	// Max scroll = totalLines - (viewHeight - 1) = 100 - (29 - 1) = 72
	// (window height 30 minus header 1 = 29 for right pane, minus 1 for title = 28 content rows)
	// So scroll of 93 should clamp to 72
	maxScroll := 100 - (m.rightPaneHeight() - 1)
	if m.planViewScroll != maxScroll {
		t.Errorf("expected planViewScroll clamped at %d, got %d", maxScroll, m.planViewScroll)
	}

	// Verify focus switches to right pane when scrolling from another pane
	m.focusedPane = plansPane
	m.rightPaneMode = planMode
	m.planViewScroll = 5
	m.Update(tea.MouseMsg{
		X:      50,
		Y:      5,
		Button: tea.MouseButtonWheelDown,
	})

	if m.focusedPane != rightPane {
		t.Error("expected focus switch to right pane on mouse scroll")
	}
	if m.planViewScroll != 8 {
		t.Errorf("expected planViewScroll=8 after scrolling from plans pane, got %d", m.planViewScroll)
	}
}

// TestIntegration_PlanModeDisabledDuringRun verifies that pressing 'p' while
// a run is in progress does nothing — plan mode must not interrupt active work.
func TestIntegration_PlanModeDisabledDuringRun(t *testing.T) {
	// Use a long-running event stream so the model stays in PhaseRunning.
	events := []session.Event{
		session.TextEvent{Text: "working..."},
	}
	m := newTestModel(events, 1)

	// Init starts the run — model enters PhaseRunning.
	_ = m.Init()

	// Press 'p' while running — should return nil cmd.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if cmd != nil {
		t.Error("expected nil cmd when pressing 'p' during PhaseRunning")
	}
}

// TestIntegration_PlanModeKeyReturnsExecCmd verifies that pressing 'p' while
// idle returns a non-nil tea.Cmd (the tea.ExecProcess command).
func TestIntegration_PlanModeKeyReturnsExecCmd(t *testing.T) {
	m := newTestModel(nil, 0)
	// Force idle mode so Init doesn't start a run.
	m.controller.Session.Mode = "idle"
	m.Init()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if cmd == nil {
		t.Error("expected non-nil cmd when pressing 'p' in idle mode")
	}
}

// TestIntegration_PlanModeErrorSetsFlash verifies that receiving a
// planModeDoneMsg with an error sets the statusFlash field.
func TestIntegration_PlanModeErrorSetsFlash(t *testing.T) {
	m := newTestModel(nil, 0)
	m.controller.Session.Mode = "idle"
	m.Init()

	_, _ = m.Update(planModeDoneMsg{err: fmt.Errorf("exit status 1")})

	if m.statusFlash == "" {
		t.Error("expected statusFlash to be set after planModeDoneMsg with error")
	}
	if !strings.Contains(m.statusFlash, "plan command failed") {
		t.Errorf("expected flash to contain 'plan command failed', got %q", m.statusFlash)
	}
}

// TestIntegration_PlanModeFlashClearsOnKeypress verifies that the status
// flash message clears on the next keypress, preventing stale error display.
func TestIntegration_PlanModeFlashClearsOnKeypress(t *testing.T) {
	m := newTestModel(nil, 0)
	m.controller.Session.Mode = "idle"
	m.Init()

	// Set flash via error message.
	_, _ = m.Update(planModeDoneMsg{err: fmt.Errorf("exit status 1")})
	if m.statusFlash == "" {
		t.Fatal("expected statusFlash to be set")
	}

	// Press any key — flash should clear.
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.statusFlash != "" {
		t.Errorf("expected statusFlash to be cleared after keypress, got %q", m.statusFlash)
	}
}

// TestIntegration_PlanModeSuccessNoFlash verifies that a successful plan mode
// exit (no error) does not set a flash message.
func TestIntegration_PlanModeSuccessNoFlash(t *testing.T) {
	m := newTestModel(nil, 0)
	m.controller.Session.Mode = "idle"
	m.Init()

	_, _ = m.Update(planModeDoneMsg{err: nil})

	if m.statusFlash != "" {
		t.Errorf("expected no flash on successful exit, got %q", m.statusFlash)
	}
}

// TestIntegration_PlanModeFlashRendersInHeader verifies that the status flash
// message appears in the rendered header output.
func TestIntegration_PlanModeFlashRendersInHeader(t *testing.T) {
	m := newTestModel(nil, 0)
	m.controller.Session.Mode = "idle"
	m.Init()

	_, _ = m.Update(planModeDoneMsg{err: fmt.Errorf("exit status 1")})

	view := m.View()
	if !strings.Contains(view, "plan command failed") {
		t.Error("expected 'plan command failed' in rendered view")
	}
}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
