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

func TestRenderQuitConfirmModal(t *testing.T) {
	th := testTheme()
	result := RenderQuitConfirmModal(80, 24, th)

	if !strings.Contains(result, "Are you sure you want") {
		t.Error("expected 'Are you sure you want' in modal")
	}
	if !strings.Contains(result, "to quit?") {
		t.Error("expected 'to quit?' in modal")
	}
	if !strings.Contains(result, "y") {
		t.Error("expected 'y' hint in modal")
	}
	if !strings.Contains(result, "n") {
		t.Error("expected 'n' hint in modal")
	}
}

func TestIntegration_QuitShowsModal(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// q should show the quit confirmation modal.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if m.activeModal != modalQuitConfirm {
		t.Error("expected quit confirmation modal after pressing q")
	}

	// View should show the modal.
	view := m.View()
	if !strings.Contains(view, "Are you sure you want") {
		t.Error("expected modal content in view after pressing q")
	}
}

func TestIntegration_QuitModal_YConfirmsQuit(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Open quit modal.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if m.activeModal != modalQuitConfirm {
		t.Fatal("expected quit modal")
	}

	// y confirms quit.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if !m.quitting {
		t.Error("expected quitting=true after y in quit modal")
	}
	if cmd == nil {
		t.Error("expected tea.Quit cmd after y")
	}
}

func TestIntegration_QuitModal_NDismisses(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Open quit modal.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if m.activeModal != modalQuitConfirm {
		t.Fatal("expected quit modal")
	}

	// n dismisses.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.activeModal != modalNone {
		t.Error("expected modal dismissed after n")
	}
	if m.quitting {
		t.Error("expected quitting=false after n")
	}
}

func TestIntegration_QuitModal_EscapeDismisses(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Open quit modal.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if m.activeModal != modalQuitConfirm {
		t.Fatal("expected quit modal")
	}

	// escape dismisses.
	m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.activeModal != modalNone {
		t.Error("expected modal dismissed after escape")
	}
}

func TestIntegration_QuitModal_OtherKeysIgnored(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Open quit modal.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if m.activeModal != modalQuitConfirm {
		t.Fatal("expected quit modal")
	}

	// Random keys should be ignored (not dismiss or quit).
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.activeModal != modalQuitConfirm {
		t.Error("expected modal to remain open after j")
	}
	if m.quitting {
		t.Error("expected quitting=false after j")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if m.activeModal != modalQuitConfirm {
		t.Error("expected modal to remain open after x")
	}
}

func TestIntegration_QuitModal_BlocksNavigation(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "main.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 3)
	drainEvents(t, m)

	savedCursor := m.iterList.Cursor

	// Open quit modal.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// k should not move cursor while modal is active.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.iterList.Cursor != savedCursor {
		t.Error("expected cursor unchanged while modal is active")
	}
}

func TestIntegration_CtrlCShowsModal(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Single ctrl+c shows modal.
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if m.activeModal != modalQuitConfirm {
		t.Error("expected quit confirmation modal after single ctrl+c")
	}
	if m.quitting {
		t.Error("expected quitting=false after single ctrl+c (modal shown)")
	}
}

func TestIntegration_DoubleCtrlCForceQuits(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// First ctrl+c shows modal.
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if m.activeModal != modalQuitConfirm {
		t.Fatal("expected quit modal after first ctrl+c")
	}

	// Simulate the lastCtrlCAt being set recently (within 500ms).
	m.lastCtrlCAt = time.Now()

	// Second ctrl+c within window: force quit.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !m.quitting {
		t.Error("expected quitting=true after double ctrl+c")
	}
	if cmd == nil {
		t.Error("expected tea.Quit cmd after double ctrl+c")
	}
}

func TestIntegration_DoubleCtrlCExpired(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Set lastCtrlCAt to >500ms ago.
	m.lastCtrlCAt = time.Now().Add(-600 * time.Millisecond)

	// ctrl+c after expiry shows modal instead of force-quitting.
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if m.quitting {
		t.Error("expected quitting=false when ctrl+c window expired")
	}
	if m.activeModal != modalQuitConfirm {
		t.Error("expected quit modal after expired ctrl+c window")
	}
}

func TestRenderHelpModal(t *testing.T) {
	th := testTheme()
	km := config.DefaultKeyMap()
	result := RenderHelpModal(80, 24, th, &km)

	// Check that section headers are present.
	if !strings.Contains(result, "Navigation") {
		t.Error("expected 'Navigation' section in help modal")
	}
	if !strings.Contains(result, "Focus") {
		t.Error("expected 'Focus' section in help modal")
	}
	if !strings.Contains(result, "Actions") {
		t.Error("expected 'Actions' section in help modal")
	}
	if !strings.Contains(result, "Global") {
		t.Error("expected 'Global' section in help modal")
	}

	// Check some keybinding entries are present.
	if !strings.Contains(result, "Move down") {
		t.Error("expected 'Move down' in help modal")
	}
	if !strings.Contains(result, "Quit") {
		t.Error("expected 'Quit' in help modal")
	}
	if !strings.Contains(result, "Press any key to close") {
		t.Error("expected footer text in help modal")
	}
	if !strings.Contains(result, "Keybindings") {
		t.Error("expected 'Keybindings' title in help modal")
	}
}

func TestRenderHelpModal_ReflectsCustomBindings(t *testing.T) {
	th := testTheme()
	km := config.DefaultKeyMap()
	// Remap quit from q to x.
	km.Bindings[config.ActionQuit] = config.ParseKeyBinding("x")

	result := RenderHelpModal(80, 24, th, &km)

	if !strings.Contains(result, "x") {
		t.Error("expected remapped key 'x' in help modal")
	}
}

func TestIntegration_HelpModal_QuestionMarkOpens(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// ? should open help modal.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.activeModal != modalHelp {
		t.Error("expected help modal after pressing ?")
	}

	// View should show the help modal content.
	view := m.View()
	if !strings.Contains(view, "Keybindings") {
		t.Error("expected help modal content in view after pressing ?")
	}
	if !strings.Contains(view, "Navigation") {
		t.Error("expected Navigation section in help modal view")
	}
}

func TestIntegration_HelpModal_AnyKeyDismisses(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Open help modal.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.activeModal != modalHelp {
		t.Fatal("expected help modal")
	}

	// Any key dismisses.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.activeModal != modalNone {
		t.Error("expected modal dismissed after pressing j")
	}
}

func TestIntegration_HelpModal_EscapeDismisses(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Open help modal.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.activeModal != modalHelp {
		t.Fatal("expected help modal")
	}

	// Escape dismisses.
	m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.activeModal != modalNone {
		t.Error("expected modal dismissed after escape")
	}
}

func TestIntegration_HelpModal_EnterDismisses(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	drainEvents(t, m)

	// Open help modal.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.activeModal != modalHelp {
		t.Fatal("expected help modal")
	}

	// Enter dismisses.
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.activeModal != modalNone {
		t.Error("expected modal dismissed after enter")
	}
}

func TestIntegration_HelpModal_BlocksNavigation(t *testing.T) {
	events := []session.Event{
		session.AssistantBatchEvent{Events: []session.Event{
			session.ToolUseEvent{ID: "tc1", Name: "Read", Summary: "main.go"},
		}},
		session.ToolResultEvent{ToolUseID: "tc1", IsError: false},
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 3)
	drainEvents(t, m)

	savedCursor := m.iterList.Cursor

	// Open help modal.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})

	// Note: pressing j will dismiss the help modal (any key dismisses),
	// but should NOT move the cursor during the modal-handling step.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.iterList.Cursor != savedCursor {
		t.Error("expected cursor unchanged when key dismissed help modal")
	}
}

func TestIntegration_ExitFlagBypassesModal(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	// Use exitOnComplete=true.
	fake := &executor.FakeExecutor{Events: events}
	sess := model.Session{
		Mode:          "build",
		PromptFile:    "test-prompt.md",
		MaxIterations: 1,
		StartTime:     time.Now(),
	}
	cfg := config.DefaultConfig()
	th := testTheme()
	m := NewModel(sess, cfg, "test prompt", th, false, true, fake)
	m.width = 120
	m.height = 30
	drainEvents(t, &m)

	// With exitOnComplete, the model should have quit directly
	// without showing a modal.
	if !m.quitting {
		t.Error("expected quitting=true with --exit flag")
	}
	if m.activeModal != modalNone {
		t.Error("expected no modal with --exit flag")
	}
}
