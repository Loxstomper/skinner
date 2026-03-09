package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDisplayName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"PROMPT_BUILD.md", "BUILD"},
		{"PROMPT_PLAN.md", "PLAN"},
		{"PROMPT_foo_bar.md", "foo_bar"},
		{"PROMPT_.md", ""},
	}
	for _, tt := range tests {
		got := DisplayName(tt.input)
		if got != tt.want {
			t.Errorf("DisplayName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestPromptList_ScanFiles(t *testing.T) {
	dir := t.TempDir()

	// Create some prompt files
	for _, name := range []string{"PROMPT_BUILD.md", "PROMPT_PLAN.md", "PROMPT_TEST.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("# "+name), 0644); err != nil {
			t.Fatal(err)
		}
	}
	// Create a non-prompt file (should be ignored)
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# README"), 0644); err != nil {
		t.Fatal(err)
	}

	pl := NewPromptList(dir)
	if len(pl.Files) != 3 {
		t.Errorf("expected 3 prompt files, got %d: %v", len(pl.Files), pl.Files)
	}

	// Should be sorted alphabetically
	expected := []string{"PROMPT_BUILD.md", "PROMPT_PLAN.md", "PROMPT_TEST.md"}
	for i, want := range expected {
		if pl.Files[i] != want {
			t.Errorf("Files[%d] = %q, want %q", i, pl.Files[i], want)
		}
	}
}

func TestPromptList_ScanFiles_Empty(t *testing.T) {
	dir := t.TempDir()
	pl := NewPromptList(dir)
	if len(pl.Files) != 0 {
		t.Errorf("expected 0 prompt files, got %d", len(pl.Files))
	}
}

func TestPromptList_ScanFiles_CursorClamp(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"PROMPT_A.md", "PROMPT_B.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	pl := NewPromptList(dir)
	pl.Cursor = 5 // way beyond range
	pl.ScanFiles(dir)
	if pl.Cursor != 1 { // clamped to last index
		t.Errorf("expected cursor=1 after clamp, got %d", pl.Cursor)
	}
}

func TestPromptList_SelectedFile(t *testing.T) {
	pl := PromptList{
		Files:  []string{"PROMPT_BUILD.md", "PROMPT_PLAN.md"},
		Cursor: 1,
	}
	if got := pl.SelectedFile(); got != "PROMPT_PLAN.md" {
		t.Errorf("SelectedFile() = %q, want %q", got, "PROMPT_PLAN.md")
	}
}

func TestPromptList_SelectedFile_Empty(t *testing.T) {
	pl := PromptList{}
	if got := pl.SelectedFile(); got != "" {
		t.Errorf("SelectedFile() = %q, want empty", got)
	}
}

func TestPromptList_HandleAction_MoveDown(t *testing.T) {
	pl := PromptList{
		Files: []string{"PROMPT_A.md", "PROMPT_B.md", "PROMPT_C.md"},
	}
	props := PromptListProps{Width: 32, Height: 5, Focused: true, Theme: testTheme()}

	pl.HandleAction("move_down", props)
	if pl.Cursor != 1 {
		t.Errorf("expected cursor=1, got %d", pl.Cursor)
	}

	pl.HandleAction("move_down", props)
	if pl.Cursor != 2 {
		t.Errorf("expected cursor=2, got %d", pl.Cursor)
	}

	// At bottom, shouldn't go further
	pl.HandleAction("move_down", props)
	if pl.Cursor != 2 {
		t.Errorf("expected cursor=2 at bottom, got %d", pl.Cursor)
	}
}

func TestPromptList_HandleAction_MoveUp(t *testing.T) {
	pl := PromptList{
		Files:  []string{"PROMPT_A.md", "PROMPT_B.md", "PROMPT_C.md"},
		Cursor: 2,
	}
	props := PromptListProps{Width: 32, Height: 5, Focused: true, Theme: testTheme()}

	pl.HandleAction("move_up", props)
	if pl.Cursor != 1 {
		t.Errorf("expected cursor=1, got %d", pl.Cursor)
	}

	pl.HandleAction("move_up", props)
	if pl.Cursor != 0 {
		t.Errorf("expected cursor=0, got %d", pl.Cursor)
	}

	// At top, shouldn't go further
	pl.HandleAction("move_up", props)
	if pl.Cursor != 0 {
		t.Errorf("expected cursor=0 at top, got %d", pl.Cursor)
	}
}

func TestPromptList_HandleAction_JumpTopBottom(t *testing.T) {
	pl := PromptList{
		Files: []string{"PROMPT_A.md", "PROMPT_B.md", "PROMPT_C.md", "PROMPT_D.md", "PROMPT_E.md"},
	}
	props := PromptListProps{Width: 32, Height: 5, Focused: true, Theme: testTheme()}

	pl.HandleAction("jump_bottom", props)
	if pl.Cursor != 4 {
		t.Errorf("expected cursor=4, got %d", pl.Cursor)
	}

	pl.HandleAction("jump_top", props)
	if pl.Cursor != 0 {
		t.Errorf("expected cursor=0, got %d", pl.Cursor)
	}
}

func TestPromptList_HandleAction_Empty(t *testing.T) {
	pl := PromptList{}
	props := PromptListProps{Width: 32, Height: 5, Focused: true, Theme: testTheme()}

	// Should not panic on empty list
	pl.HandleAction("move_down", props)
	pl.HandleAction("move_up", props)
	pl.HandleAction("jump_top", props)
	pl.HandleAction("jump_bottom", props)
}

func TestPromptList_View_WithFiles(t *testing.T) {
	pl := PromptList{
		Files: []string{"PROMPT_BUILD.md", "PROMPT_PLAN.md"},
	}
	props := PromptListProps{Width: 32, Height: 5, Focused: true, Theme: testTheme()}

	result := pl.View(props)

	if !strings.Contains(result, "Prompts") {
		t.Error("expected title 'Prompts' in output")
	}
	if !strings.Contains(result, "BUILD") {
		t.Error("expected 'BUILD' in output")
	}
	if !strings.Contains(result, "PLAN") {
		t.Error("expected 'PLAN' in output")
	}
}

func TestPromptList_View_Empty(t *testing.T) {
	pl := PromptList{}
	props := PromptListProps{Width: 32, Height: 5, Focused: true, Theme: testTheme()}

	result := pl.View(props)

	if !strings.Contains(result, "Prompts") {
		t.Error("expected title 'Prompts' in output")
	}
	if !strings.Contains(result, "No prompt files") {
		t.Error("expected 'No prompt files' in output")
	}
}

func TestPromptList_View_Scrolling(t *testing.T) {
	pl := PromptList{
		Files:  []string{"PROMPT_A.md", "PROMPT_B.md", "PROMPT_C.md", "PROMPT_D.md", "PROMPT_E.md", "PROMPT_F.md"},
		Cursor: 5,
		Scroll: 2,
	}
	// Height 5 = 1 title + 4 content rows
	props := PromptListProps{Width: 32, Height: 5, Focused: true, Theme: testTheme()}

	result := pl.View(props)

	// With scroll=2, content rows should show files C, D, E, F (indices 2-5)
	if !strings.Contains(result, "C") {
		t.Error("expected 'C' in scrolled view")
	}
	if !strings.Contains(result, "F") {
		t.Error("expected 'F' in scrolled view")
	}
	// A and B should be scrolled past
	if strings.Contains(result, "  A") {
		t.Error("did not expect 'A' in scrolled view")
	}
}

func TestPromptList_ClickRow(t *testing.T) {
	pl := PromptList{
		Files: []string{"PROMPT_A.md", "PROMPT_B.md", "PROMPT_C.md"},
	}

	// Row 0 is title — should be ignored
	if pl.ClickRow(0) {
		t.Error("expected ClickRow(0) to return false for title row")
	}

	// Row 1 = first file
	if !pl.ClickRow(1) {
		t.Error("expected ClickRow(1) to return true")
	}
	if pl.Cursor != 0 {
		t.Errorf("expected cursor=0, got %d", pl.Cursor)
	}

	// Row 2 = second file
	if !pl.ClickRow(2) {
		t.Error("expected ClickRow(2) to return true")
	}
	if pl.Cursor != 1 {
		t.Errorf("expected cursor=1, got %d", pl.Cursor)
	}

	// Row beyond files — should be ignored
	if pl.ClickRow(5) {
		t.Error("expected ClickRow(5) to return false for out-of-range")
	}
}

func TestPromptList_ClickRow_WithScroll(t *testing.T) {
	pl := PromptList{
		Files:  []string{"PROMPT_A.md", "PROMPT_B.md", "PROMPT_C.md", "PROMPT_D.md", "PROMPT_E.md"},
		Scroll: 2,
	}

	// Row 1 with scroll 2 = file index 2 (C)
	if !pl.ClickRow(1) {
		t.Error("expected ClickRow(1) to return true")
	}
	if pl.Cursor != 2 {
		t.Errorf("expected cursor=2 (scroll 2 + row 0), got %d", pl.Cursor)
	}
}

func TestIsInPromptSection(t *testing.T) {
	paneHeight := 30
	// Prompt section starts at row 25 (30 - 5 = 25)

	if IsInPromptSection(24, paneHeight) {
		t.Error("row 24 should not be in prompt section")
	}
	if !IsInPromptSection(25, paneHeight) {
		t.Error("row 25 should be in prompt section")
	}
	if !IsInPromptSection(29, paneHeight) {
		t.Error("row 29 should be in prompt section")
	}
}

func TestPromptSectionRow(t *testing.T) {
	paneHeight := 30
	// Prompt section starts at row 25
	got := PromptSectionRow(26, paneHeight)
	if got != 1 {
		t.Errorf("PromptSectionRow(26, 30) = %d, want 1", got)
	}
}

func TestPromptList_ScrollBy(t *testing.T) {
	pl := PromptList{
		Files: []string{"PROMPT_A.md", "PROMPT_B.md", "PROMPT_C.md", "PROMPT_D.md", "PROMPT_E.md", "PROMPT_F.md", "PROMPT_G.md"},
	}

	pl.ScrollBy(2)
	if pl.Scroll != 2 {
		t.Errorf("expected scroll=2, got %d", pl.Scroll)
	}

	// Should clamp at max
	pl.ScrollBy(100)
	// Max scroll = 7 - 4 = 3
	if pl.Scroll != 3 {
		t.Errorf("expected scroll=3 (clamped), got %d", pl.Scroll)
	}

	// Should clamp at 0
	pl.ScrollBy(-100)
	if pl.Scroll != 0 {
		t.Errorf("expected scroll=0 (clamped), got %d", pl.Scroll)
	}
}

func TestPromptList_FileExists(t *testing.T) {
	pl := PromptList{
		Files: []string{"PROMPT_BUILD.md", "PROMPT_PLAN.md"},
	}

	if !pl.FileExists("PROMPT_BUILD.md") {
		t.Error("expected FileExists to return true for existing file")
	}
	if pl.FileExists("PROMPT_OTHER.md") {
		t.Error("expected FileExists to return false for non-existing file")
	}
}

func TestReadFileContent(t *testing.T) {
	dir := t.TempDir()
	content := "# Test Prompt\n\nThis is a test."
	if err := os.WriteFile(filepath.Join(dir, "PROMPT_TEST.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := ReadFileContent(dir, "PROMPT_TEST.md")
	if err != nil {
		t.Fatalf("ReadFileContent failed: %v", err)
	}
	if got != content {
		t.Errorf("ReadFileContent = %q, want %q", got, content)
	}
}

func TestReadFileContent_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadFileContent(dir, "PROMPT_MISSING.md")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestPromptListTotalHeight(t *testing.T) {
	h := PromptListTotalHeight()
	if h != 5 {
		t.Errorf("PromptListTotalHeight() = %d, want 5 (4 content + 1 title)", h)
	}
}
