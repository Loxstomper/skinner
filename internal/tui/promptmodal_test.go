package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/loxstomper/skinner/internal/session"
)

func TestRenderPromptReadModal_ContainsFilename(t *testing.T) {
	props := PromptModalProps{
		Filename: "PROMPT_BUILD.md",
		Content:  "line one\nline two\nline three\n",
		Scroll:   0,
		Width:    120,
		Height:   40,
		Theme:    testTheme(),
	}

	result := RenderPromptReadModal(props)

	if !strings.Contains(result, "PROMPT_BUILD.md") {
		t.Error("expected filename in modal title")
	}
}

func TestRenderPromptReadModal_ContainsLineNumbers(t *testing.T) {
	content := "alpha\nbeta\ngamma\n"
	props := PromptModalProps{
		Filename: "PROMPT_TEST.md",
		Content:  content,
		Scroll:   0,
		Width:    120,
		Height:   40,
		Theme:    testTheme(),
	}

	result := RenderPromptReadModal(props)

	// Should contain line numbers 1, 2, 3
	if !strings.Contains(result, " 1 ") {
		t.Error("expected line number 1 in modal")
	}
	if !strings.Contains(result, " 2 ") {
		t.Error("expected line number 2 in modal")
	}
	if !strings.Contains(result, " 3 ") {
		t.Error("expected line number 3 in modal")
	}
}

func TestRenderPromptReadModal_ContainsContent(t *testing.T) {
	content := "Hello World\nSecond Line\n"
	props := PromptModalProps{
		Filename: "PROMPT_HELLO.md",
		Content:  content,
		Scroll:   0,
		Width:    120,
		Height:   40,
		Theme:    testTheme(),
	}

	result := RenderPromptReadModal(props)

	if !strings.Contains(result, "Hello World") {
		t.Error("expected content text in modal")
	}
	if !strings.Contains(result, "Second Line") {
		t.Error("expected second line content in modal")
	}
}

func TestRenderPromptReadModal_ContainsFooter(t *testing.T) {
	props := PromptModalProps{
		Filename: "PROMPT_FOO.md",
		Content:  "content\n",
		Scroll:   0,
		Width:    120,
		Height:   40,
		Theme:    testTheme(),
	}

	result := RenderPromptReadModal(props)

	if !strings.Contains(result, "e to edit") {
		t.Error("expected 'e to edit' in footer")
	}
	if !strings.Contains(result, "esc to close") {
		t.Error("expected 'esc to close' in footer")
	}
}

func TestRenderPromptReadModal_FooterShowsRunWhenNotRunning(t *testing.T) {
	props := PromptModalProps{
		Filename: "PROMPT_FOO.md",
		Content:  "content\n",
		Scroll:   0,
		Width:    120,
		Height:   40,
		Theme:    testTheme(),
		Running:  false,
	}

	result := RenderPromptReadModal(props)

	if !strings.Contains(result, "r to run") {
		t.Error("expected 'r to run' in footer when not running")
	}
	if !strings.Contains(result, "e to edit") {
		t.Error("expected 'e to edit' in footer")
	}
	if !strings.Contains(result, "esc to close") {
		t.Error("expected 'esc to close' in footer")
	}
}

func TestRenderPromptReadModal_FooterHidesRunWhenRunning(t *testing.T) {
	props := PromptModalProps{
		Filename: "PROMPT_FOO.md",
		Content:  "content\n",
		Scroll:   0,
		Width:    120,
		Height:   40,
		Theme:    testTheme(),
		Running:  true,
	}

	result := RenderPromptReadModal(props)

	if strings.Contains(result, "r to run") {
		t.Error("expected 'r to run' to be hidden in footer when running")
	}
	if !strings.Contains(result, "e to edit") {
		t.Error("expected 'e to edit' still present when running")
	}
	if !strings.Contains(result, "esc to close") {
		t.Error("expected 'esc to close' still present when running")
	}
}

func TestRenderPromptReadModal_ScrollClamped(t *testing.T) {
	// Short content that fits in one screen shouldn't scroll
	content := "line 1\nline 2\n"
	props := PromptModalProps{
		Filename: "PROMPT_X.md",
		Content:  content,
		Scroll:   100, // way beyond content
		Width:    120,
		Height:   40,
		Theme:    testTheme(),
	}

	// Should not panic and should still show content
	result := RenderPromptReadModal(props)
	if !strings.Contains(result, "line 1") {
		t.Error("expected content visible even with over-scrolled offset")
	}
}

func TestRenderPromptReadModal_EmptyContent(t *testing.T) {
	props := PromptModalProps{
		Filename: "PROMPT_EMPTY.md",
		Content:  "",
		Scroll:   0,
		Width:    120,
		Height:   40,
		Theme:    testTheme(),
	}

	result := RenderPromptReadModal(props)
	if !strings.Contains(result, "PROMPT_EMPTY.md") {
		t.Error("expected filename even with empty content")
	}
}

func TestPromptModalMaxScroll_ShortContent(t *testing.T) {
	// Content shorter than viewport: max scroll should be 0
	content := "line 1\nline 2\n"
	maxScroll := PromptModalMaxScroll(content, 40)
	if maxScroll != 0 {
		t.Errorf("expected maxScroll=0 for short content, got %d", maxScroll)
	}
}

func TestPromptModalMaxScroll_LongContent(t *testing.T) {
	// Generate content longer than a viewport
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, "line content here")
	}
	content := strings.Join(lines, "\n") + "\n"

	maxScroll := PromptModalMaxScroll(content, 40)
	if maxScroll <= 0 {
		t.Error("expected positive maxScroll for long content")
	}
}

func TestIntegration_PromptModal_EnterOpensModal(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}
	m := newTestModel(events, 1)

	// Create a prompt file in a temp directory
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "PROMPT_BUILD.md"), []byte("build instructions\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	m.workDir = tmpDir
	m.promptList.ScanFiles(tmpDir)

	drainEvents(t, m)

	// Focus on prompts pane
	m.focusedPane = promptsPane

	// Press Enter (expand action)
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.activeModal != modalPromptRead {
		t.Errorf("expected modalPromptRead after Enter, got %d", m.activeModal)
	}
	if m.promptModalFile != "PROMPT_BUILD.md" {
		t.Errorf("expected promptModalFile=PROMPT_BUILD.md, got %q", m.promptModalFile)
	}
	if !strings.Contains(m.promptModalContent, "build instructions") {
		t.Error("expected file content in promptModalContent")
	}
}

func TestIntegration_PromptModal_EscDismisses(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}
	m := newTestModel(events, 1)

	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "PROMPT_TEST.md"), []byte("test\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	m.workDir = tmpDir
	m.promptList.ScanFiles(tmpDir)

	drainEvents(t, m)

	// Open the modal
	m.focusedPane = promptsPane
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.activeModal != modalPromptRead {
		t.Fatal("expected modal to be open")
	}

	// Esc dismisses
	m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.activeModal != modalNone {
		t.Error("expected modal dismissed after esc")
	}
}

func TestIntegration_PromptModal_ScrollKeys(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}
	m := newTestModel(events, 1)

	// Create a file with lots of content
	tmpDir := t.TempDir()
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, "line content")
	}
	content := strings.Join(lines, "\n") + "\n"
	err := os.WriteFile(filepath.Join(tmpDir, "PROMPT_LONG.md"), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}
	m.workDir = tmpDir
	m.promptList.ScanFiles(tmpDir)

	drainEvents(t, m)

	// Open modal
	m.focusedPane = promptsPane
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.activeModal != modalPromptRead {
		t.Fatal("expected modal open")
	}
	if m.promptModalScroll != 0 {
		t.Fatal("expected initial scroll=0")
	}

	// j scrolls down
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.promptModalScroll != 1 {
		t.Errorf("expected scroll=1 after j, got %d", m.promptModalScroll)
	}

	// k scrolls back up
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.promptModalScroll != 0 {
		t.Errorf("expected scroll=0 after k, got %d", m.promptModalScroll)
	}

	// k doesn't go below 0
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.promptModalScroll != 0 {
		t.Error("expected scroll clamped at 0")
	}
}

func TestIntegration_PromptModal_BlocksNavigation(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}
	m := newTestModel(events, 1)

	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "PROMPT_NAV.md"), []byte("nav test\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	m.workDir = tmpDir
	m.promptList.ScanFiles(tmpDir)

	drainEvents(t, m)

	// Open modal
	m.focusedPane = promptsPane
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.activeModal != modalPromptRead {
		t.Fatal("expected modal open")
	}

	// Tab should not change focus while modal is open
	savedPane := m.focusedPane
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focusedPane != savedPane {
		t.Error("expected focus unchanged while prompt modal is active")
	}
	if m.activeModal != modalPromptRead {
		t.Error("expected modal to remain open after Tab")
	}
}

func TestIntegration_PromptModal_ViewShowsContent(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}
	m := newTestModel(events, 1)

	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "PROMPT_VIEW.md"), []byte("visible content\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	m.workDir = tmpDir
	m.promptList.ScanFiles(tmpDir)

	drainEvents(t, m)

	// Open modal
	m.focusedPane = promptsPane
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	view := m.View()
	if !strings.Contains(view, "PROMPT_VIEW.md") {
		t.Error("expected filename in view")
	}
	if !strings.Contains(view, "visible content") {
		t.Error("expected file content in view")
	}
	if !strings.Contains(view, "e to edit") {
		t.Error("expected footer hint in view")
	}
}

func TestIntegration_PromptModal_NoFileNoModal(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}
	m := newTestModel(events, 1)

	// Empty directory - no prompt files
	tmpDir := t.TempDir()
	m.workDir = tmpDir
	m.promptList.ScanFiles(tmpDir)

	drainEvents(t, m)

	// Focus prompts and press Enter - should not open modal
	m.focusedPane = promptsPane
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.activeModal != modalNone {
		t.Error("expected no modal when no files selected")
	}
}

func TestIntegration_PromptModal_PgDownPgUp(t *testing.T) {
	events := []session.Event{
		session.SubprocessExitEvent{Err: nil},
	}
	m := newTestModel(events, 1)

	tmpDir := t.TempDir()
	var lines []string
	for i := 0; i < 200; i++ {
		lines = append(lines, "long content line")
	}
	content := strings.Join(lines, "\n") + "\n"
	err := os.WriteFile(filepath.Join(tmpDir, "PROMPT_PG.md"), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}
	m.workDir = tmpDir
	m.promptList.ScanFiles(tmpDir)

	drainEvents(t, m)

	m.focusedPane = promptsPane
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.activeModal != modalPromptRead {
		t.Fatal("expected modal open")
	}

	// pgdown scrolls by 10
	m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	if m.promptModalScroll != 10 {
		t.Errorf("expected scroll=10 after pgdown, got %d", m.promptModalScroll)
	}

	// pgup scrolls back
	m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	if m.promptModalScroll != 0 {
		t.Errorf("expected scroll=0 after pgup, got %d", m.promptModalScroll)
	}
}
