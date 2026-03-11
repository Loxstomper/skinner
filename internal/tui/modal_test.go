package tui

import (
	"os"
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
	result := RenderHelpModal(80, 40, th, &km, 0)

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
	if !strings.Contains(result, "File Explorer") {
		t.Error("expected 'File Explorer' section in help modal")
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
	// Check file explorer entries.
	if !strings.Contains(result, "Search files") {
		t.Error("expected 'Search files' in help modal")
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

	result := RenderHelpModal(80, 40, th, &km, 0)

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

func TestRenderHelpModal_ScrollsWhenTooSmall(t *testing.T) {
	th := testTheme()
	km := config.DefaultKeyMap()

	// Render with a very small height — content should be clipped.
	result := RenderHelpModal(80, 15, th, &km, 0)

	// Should show the top content (Navigation header) but not the footer.
	if !strings.Contains(result, "Navigation") {
		t.Error("expected 'Navigation' section visible at scroll offset 0")
	}
	if strings.Contains(result, "Press any key to close") {
		t.Error("expected footer to be clipped at small height")
	}

	// With a large scroll offset, the top content should be hidden.
	result2 := RenderHelpModal(80, 15, th, &km, 30)
	if strings.Contains(result2, "Navigation") {
		t.Error("expected 'Navigation' section scrolled away at offset 30")
	}
	// Footer should now be visible.
	if !strings.Contains(result2, "Press any key to close") {
		t.Error("expected footer visible after scrolling down")
	}
}

func TestIntegration_HelpModal_PgUpPgDnScroll(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}

	m := newTestModel(events, 1)
	m.height = 15 // Small terminal to trigger scrolling.
	drainEvents(t, m)

	// Open help modal.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if m.activeModal != modalHelp {
		t.Fatal("expected help modal open")
	}

	// pgdn should scroll, not dismiss.
	m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	if m.activeModal != modalHelp {
		t.Error("expected help modal still open after pgdn")
	}
	if m.helpModalScroll == 0 {
		t.Error("expected helpModalScroll to increase after pgdn")
	}

	// pgup should scroll back.
	saved := m.helpModalScroll
	m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	if m.activeModal != modalHelp {
		t.Error("expected help modal still open after pgup")
	}
	if m.helpModalScroll >= saved {
		t.Error("expected helpModalScroll to decrease after pgup")
	}

	// Any other key should dismiss.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if m.activeModal != modalNone {
		t.Error("expected help modal dismissed after pressing x")
	}
	if m.helpModalScroll != 0 {
		t.Error("expected helpModalScroll reset after dismissal")
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

func TestRenderRunModal_ContainsLabel(t *testing.T) {
	th := testTheme()
	result := RenderRunModal(80, 24, th, "10", false)

	if !strings.Contains(result, "Iterations:") {
		t.Error("expected 'Iterations:' label in run modal")
	}
}

func TestRenderRunModal_ContainsValue(t *testing.T) {
	th := testTheme()
	result := RenderRunModal(80, 24, th, "25", false)

	if !strings.Contains(result, "25") {
		t.Error("expected value '25' in run modal")
	}
}

func TestRenderRunModal_ContainsHints(t *testing.T) {
	th := testTheme()
	result := RenderRunModal(80, 24, th, "10", false)

	if !strings.Contains(result, "enter to start") {
		t.Error("expected 'enter to start' hint in run modal")
	}
	if !strings.Contains(result, "esc to cancel") {
		t.Error("expected 'esc to cancel' hint in run modal")
	}
}

func TestRenderRunModal_SelectedState(t *testing.T) {
	th := testTheme()
	// Selected state should still contain the value
	result := RenderRunModal(80, 24, th, "10", true)

	if !strings.Contains(result, "10") {
		t.Error("expected value '10' visible in selected state")
	}
	if !strings.Contains(result, "Iterations:") {
		t.Error("expected label in selected state")
	}
}

func TestRenderRunModal_EmptyValue(t *testing.T) {
	th := testTheme()
	result := RenderRunModal(80, 24, th, "", false)

	if !strings.Contains(result, "Iterations:") {
		t.Error("expected label with empty value")
	}
	if !strings.Contains(result, "enter to start") {
		t.Error("expected hints with empty value")
	}
}

func TestIntegration_RunModal_RKeyOpensFromPromptPicker(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}
	m := newTestModel(events, 1)

	tmpDir := t.TempDir()
	writeFile(t, tmpDir, "PROMPT_BUILD.md", "build instructions\n")
	m.workDir = tmpDir
	m.promptList.ScanFiles(tmpDir)

	drainEvents(t, m)

	// Focus prompts pane
	m.focusedPane = promptsPane

	// Press r to open run modal
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	if m.activeModal != modalRunConfig {
		t.Errorf("expected modalRunConfig after pressing r, got %d", m.activeModal)
	}
	if m.runModalValue != "10" {
		t.Errorf("expected default pre-fill value '10', got %q", m.runModalValue)
	}
	if !m.runModalSelected {
		t.Error("expected value to be selected on first open")
	}
}

func TestIntegration_RunModal_RKeyDisabledWhileRunning(t *testing.T) {
	events := []session.Event{
		// Don't send exit event — iteration stays running
	}
	m := newTestModel(events, 1)

	tmpDir := t.TempDir()
	writeFile(t, tmpDir, "PROMPT_BUILD.md", "build\n")
	m.workDir = tmpDir
	m.promptList.ScanFiles(tmpDir)

	// Init starts a run, so phase is Running
	m.Init()
	m.width = 120
	m.height = 30
	m.focusedPane = promptsPane

	// r should be ignored while running
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if m.activeModal != modalNone {
		t.Error("expected no modal while phase is Running")
	}
}

func TestIntegration_RunModal_EscDismisses(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}
	m := newTestModel(events, 1)

	tmpDir := t.TempDir()
	writeFile(t, tmpDir, "PROMPT_BUILD.md", "build\n")
	m.workDir = tmpDir
	m.promptList.ScanFiles(tmpDir)

	drainEvents(t, m)
	m.focusedPane = promptsPane

	// Open run modal
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if m.activeModal != modalRunConfig {
		t.Fatal("expected run modal open")
	}

	// Esc dismisses
	m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.activeModal != modalNone {
		t.Error("expected modal dismissed after esc")
	}
}

func TestIntegration_RunModal_DigitInput(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}
	m := newTestModel(events, 1)

	tmpDir := t.TempDir()
	writeFile(t, tmpDir, "PROMPT_BUILD.md", "build\n")
	m.workDir = tmpDir
	m.promptList.ScanFiles(tmpDir)

	drainEvents(t, m)
	m.focusedPane = promptsPane

	// Open run modal
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	// First digit replaces selected value
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	if m.runModalValue != "5" {
		t.Errorf("expected value '5' after digit replaces selection, got %q", m.runModalValue)
	}
	if m.runModalSelected {
		t.Error("expected selected=false after typing digit")
	}

	// Second digit appends
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	if m.runModalValue != "53" {
		t.Errorf("expected value '53' after appending digit, got %q", m.runModalValue)
	}
}

func TestIntegration_RunModal_BackspaceDeletesLast(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}
	m := newTestModel(events, 1)

	tmpDir := t.TempDir()
	writeFile(t, tmpDir, "PROMPT_BUILD.md", "build\n")
	m.workDir = tmpDir
	m.promptList.ScanFiles(tmpDir)

	drainEvents(t, m)
	m.focusedPane = promptsPane

	// Open run modal, type a value
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}}) // replaces "10" with "2"
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}}) // "25"

	// Backspace deletes last digit
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.runModalValue != "2" {
		t.Errorf("expected value '2' after backspace, got %q", m.runModalValue)
	}
}

func TestIntegration_RunModal_BackspaceClearsSelected(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}
	m := newTestModel(events, 1)

	tmpDir := t.TempDir()
	writeFile(t, tmpDir, "PROMPT_BUILD.md", "build\n")
	m.workDir = tmpDir
	m.promptList.ScanFiles(tmpDir)

	drainEvents(t, m)
	m.focusedPane = promptsPane

	// Open run modal (value is "10", selected)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if !m.runModalSelected {
		t.Fatal("expected selected=true")
	}

	// Backspace while selected clears the value
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.runModalValue != "" {
		t.Errorf("expected empty value after backspace on selected, got %q", m.runModalValue)
	}
	if m.runModalSelected {
		t.Error("expected selected=false after backspace")
	}
}

func TestIntegration_RunModal_EmptyEnterIgnored(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}
	m := newTestModel(events, 1)

	tmpDir := t.TempDir()
	writeFile(t, tmpDir, "PROMPT_BUILD.md", "build\n")
	m.workDir = tmpDir
	m.promptList.ScanFiles(tmpDir)

	drainEvents(t, m)
	m.focusedPane = promptsPane

	// Open run modal and clear the value
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace}) // clears selected "10"

	// Enter with empty value does nothing
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.activeModal != modalRunConfig {
		t.Error("expected modal to remain open when enter pressed with empty value")
	}
}

func TestIntegration_RunModal_EnterStartsRun(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}
	m := newTestModel(events, 1)

	tmpDir := t.TempDir()
	writeFile(t, tmpDir, "PROMPT_BUILD.md", "build instructions\n")
	m.workDir = tmpDir
	m.promptList.ScanFiles(tmpDir)

	drainEvents(t, m)
	m.focusedPane = promptsPane

	// Open run modal and press enter with default "10"
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.activeModal != modalNone {
		t.Error("expected modal closed after enter")
	}
	if m.runModalLastValue != "10" {
		t.Errorf("expected last value saved as '10', got %q", m.runModalLastValue)
	}
	if cmd == nil {
		t.Error("expected a command to be returned (spawnIteration)")
	}
	// Session should now have a new run
	if len(m.controller.Session.Runs) < 2 {
		t.Errorf("expected at least 2 runs (initial + new), got %d", len(m.controller.Session.Runs))
	}
}

func TestIntegration_RunModal_PreFillMemory(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}
	m := newTestModel(events, 1)

	tmpDir := t.TempDir()
	writeFile(t, tmpDir, "PROMPT_BUILD.md", "build\n")
	m.workDir = tmpDir
	m.promptList.ScanFiles(tmpDir)

	drainEvents(t, m)
	m.focusedPane = promptsPane

	// Open run modal, change value to "25", press enter
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Drain the new iteration events
	drainEvents(t, m)

	// Re-open: should pre-fill with "25"
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if m.runModalValue != "25" {
		t.Errorf("expected pre-fill '25', got %q", m.runModalValue)
	}
}

func TestIntegration_RunModal_RFromPromptReadModal(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}
	m := newTestModel(events, 1)

	tmpDir := t.TempDir()
	writeFile(t, tmpDir, "PROMPT_REVIEW.md", "review instructions\n")
	m.workDir = tmpDir
	m.promptList.ScanFiles(tmpDir)

	drainEvents(t, m)
	m.focusedPane = promptsPane

	// Open prompt read modal first
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.activeModal != modalPromptRead {
		t.Fatal("expected prompt read modal open")
	}

	// Press r from within prompt read modal
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if m.activeModal != modalRunConfig {
		t.Errorf("expected run config modal after r in prompt read modal, got %d", m.activeModal)
	}
}

func TestIntegration_RunModal_NonDigitIgnored(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}
	m := newTestModel(events, 1)

	tmpDir := t.TempDir()
	writeFile(t, tmpDir, "PROMPT_BUILD.md", "build\n")
	m.workDir = tmpDir
	m.promptList.ScanFiles(tmpDir)

	drainEvents(t, m)
	m.focusedPane = promptsPane

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}}) // replaces "10" -> "5"

	// Non-digit keys should be ignored
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if m.runModalValue != "5" {
		t.Errorf("expected value unchanged after non-digit, got %q", m.runModalValue)
	}
}

func TestIntegration_RunModal_ViewShowsModal(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}
	m := newTestModel(events, 1)

	tmpDir := t.TempDir()
	writeFile(t, tmpDir, "PROMPT_BUILD.md", "build\n")
	m.workDir = tmpDir
	m.promptList.ScanFiles(tmpDir)

	drainEvents(t, m)
	m.focusedPane = promptsPane

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	view := m.View()
	if !strings.Contains(view, "Iterations:") {
		t.Error("expected 'Iterations:' in view when run modal is open")
	}
	if !strings.Contains(view, "enter to start") {
		t.Error("expected 'enter to start' hint in view")
	}
}

func TestPromptNameFromFile(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"PROMPT_BUILD.md", "BUILD"},
		{"PROMPT_PLAN.md", "PLAN"},
		{"PROMPT_REVIEW.md", "REVIEW"},
		{"BUILD.md", "BUILD"},
		{"PROMPT_.md", ""},
	}
	for _, tt := range tests {
		got := promptNameFromFile(tt.input)
		if got != tt.expected {
			t.Errorf("promptNameFromFile(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

// writeFile is a test helper to create a file in a directory.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(dir+"/"+name, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}
}
