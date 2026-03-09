package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanDisplayName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"IMPLEMENTATION_PLAN.md", "IMPLEMENTATION"},
		{"RELEASE_PLAN.md", "RELEASE"},
		{"MY_COOL_PLAN.md", "MY_COOL"},
	}
	for _, tt := range tests {
		got := PlanDisplayName(tt.input)
		if got != tt.want {
			t.Errorf("PlanDisplayName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestPlanList_ScanFiles(t *testing.T) {
	dir := t.TempDir()

	// Create some plan files
	for _, name := range []string{"IMPLEMENTATION_PLAN.md", "RELEASE_PLAN.md", "TEST_PLAN.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("# "+name), 0644); err != nil {
			t.Fatal(err)
		}
	}
	// Create a non-plan file (should be ignored)
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# README"), 0644); err != nil {
		t.Fatal(err)
	}

	pl := NewPlanList(dir)
	if len(pl.Files) != 3 {
		t.Errorf("expected 3 plan files, got %d: %v", len(pl.Files), pl.Files)
	}

	// Should be sorted alphabetically
	expected := []string{"IMPLEMENTATION_PLAN.md", "RELEASE_PLAN.md", "TEST_PLAN.md"}
	for i, want := range expected {
		if pl.Files[i] != want {
			t.Errorf("Files[%d] = %q, want %q", i, pl.Files[i], want)
		}
	}
}

func TestPlanList_ScanFiles_Empty(t *testing.T) {
	dir := t.TempDir()
	pl := NewPlanList(dir)
	if len(pl.Files) != 0 {
		t.Errorf("expected 0 plan files, got %d", len(pl.Files))
	}
}

func TestPlanList_ScanFiles_CursorClamp(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"A_PLAN.md", "B_PLAN.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	pl := NewPlanList(dir)
	pl.Cursor = 5 // way beyond range
	pl.ScanFiles(dir)
	if pl.Cursor != 1 { // clamped to last index
		t.Errorf("expected cursor=1 after clamp, got %d", pl.Cursor)
	}
}

func TestPlanList_SelectedFile(t *testing.T) {
	pl := PlanList{
		Files:  []string{"IMPLEMENTATION_PLAN.md", "RELEASE_PLAN.md"},
		Cursor: 1,
	}
	if got := pl.SelectedFile(); got != "RELEASE_PLAN.md" {
		t.Errorf("SelectedFile() = %q, want %q", got, "RELEASE_PLAN.md")
	}
}

func TestPlanList_SelectedFile_Empty(t *testing.T) {
	pl := PlanList{}
	if got := pl.SelectedFile(); got != "" {
		t.Errorf("SelectedFile() = %q, want empty", got)
	}
}

func TestPlanList_HandleAction_MoveDown(t *testing.T) {
	pl := PlanList{
		Files: []string{"A_PLAN.md", "B_PLAN.md", "C_PLAN.md"},
	}
	props := PlanListProps{Width: 32, Height: 5, Focused: true, Theme: testTheme()}

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

func TestPlanList_HandleAction_MoveUp(t *testing.T) {
	pl := PlanList{
		Files:  []string{"A_PLAN.md", "B_PLAN.md", "C_PLAN.md"},
		Cursor: 2,
	}
	props := PlanListProps{Width: 32, Height: 5, Focused: true, Theme: testTheme()}

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

func TestPlanList_HandleAction_JumpTopBottom(t *testing.T) {
	pl := PlanList{
		Files: []string{"A_PLAN.md", "B_PLAN.md", "C_PLAN.md", "D_PLAN.md", "E_PLAN.md"},
	}
	props := PlanListProps{Width: 32, Height: 5, Focused: true, Theme: testTheme()}

	pl.HandleAction("jump_bottom", props)
	if pl.Cursor != 4 {
		t.Errorf("expected cursor=4, got %d", pl.Cursor)
	}

	pl.HandleAction("jump_top", props)
	if pl.Cursor != 0 {
		t.Errorf("expected cursor=0, got %d", pl.Cursor)
	}
}

func TestPlanList_HandleAction_PageUpDown(t *testing.T) {
	pl := PlanList{
		Files: []string{"A_PLAN.md", "B_PLAN.md", "C_PLAN.md", "D_PLAN.md",
			"E_PLAN.md", "F_PLAN.md", "G_PLAN.md", "H_PLAN.md"},
	}
	props := PlanListProps{Width: 32, Height: 5, Focused: true, Theme: testTheme()}

	pl.HandleAction("page_down", props)
	if pl.Cursor != 4 {
		t.Errorf("expected cursor=4 after page_down, got %d", pl.Cursor)
	}

	pl.HandleAction("page_up", props)
	if pl.Cursor != 0 {
		t.Errorf("expected cursor=0 after page_up, got %d", pl.Cursor)
	}
}

func TestPlanList_HandleAction_Empty(t *testing.T) {
	pl := PlanList{}
	props := PlanListProps{Width: 32, Height: 5, Focused: true, Theme: testTheme()}

	// Should not panic on empty list
	pl.HandleAction("move_down", props)
	pl.HandleAction("move_up", props)
	pl.HandleAction("jump_top", props)
	pl.HandleAction("jump_bottom", props)
	pl.HandleAction("page_down", props)
	pl.HandleAction("page_up", props)
}

func TestPlanList_View_WithFiles(t *testing.T) {
	pl := PlanList{
		Files: []string{"IMPLEMENTATION_PLAN.md", "RELEASE_PLAN.md"},
	}
	props := PlanListProps{Width: 32, Height: 5, Focused: true, Theme: testTheme()}

	result := pl.View(props)

	if !strings.Contains(result, "Plans") {
		t.Error("expected title 'Plans' in output")
	}
	if !strings.Contains(result, "IMPLEMENTATION") {
		t.Error("expected 'IMPLEMENTATION' in output")
	}
	if !strings.Contains(result, "RELEASE") {
		t.Error("expected 'RELEASE' in output")
	}
}

func TestPlanList_View_Empty(t *testing.T) {
	pl := PlanList{}
	props := PlanListProps{Width: 32, Height: 5, Focused: true, Theme: testTheme()}

	result := pl.View(props)

	if !strings.Contains(result, "Plans") {
		t.Error("expected title 'Plans' in output")
	}
	if !strings.Contains(result, "No plan files") {
		t.Error("expected 'No plan files' in output")
	}
}

func TestPlanList_View_Scrolling(t *testing.T) {
	pl := PlanList{
		Files:  []string{"A_PLAN.md", "B_PLAN.md", "C_PLAN.md", "D_PLAN.md", "E_PLAN.md", "F_PLAN.md"},
		Cursor: 5,
		Scroll: 2,
	}
	// Height 5 = 1 title + 4 content rows
	props := PlanListProps{Width: 32, Height: 5, Focused: true, Theme: testTheme()}

	result := pl.View(props)

	// With scroll=2, content rows should show files C, D, E, F (indices 2-5)
	if !strings.Contains(result, "C") {
		t.Error("expected 'C' in scrolled view")
	}
	if !strings.Contains(result, "F") {
		t.Error("expected 'F' in scrolled view")
	}
}

func TestPlanList_View_ZeroHeight(t *testing.T) {
	pl := PlanList{Files: []string{"A_PLAN.md"}}
	props := PlanListProps{Width: 32, Height: 0, Focused: true, Theme: testTheme()}

	result := pl.View(props)
	if result != "" {
		t.Errorf("expected empty string for zero height, got %q", result)
	}
}

func TestPlanList_ClickRow(t *testing.T) {
	pl := PlanList{
		Files: []string{"A_PLAN.md", "B_PLAN.md", "C_PLAN.md"},
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

func TestPlanList_ClickRow_WithScroll(t *testing.T) {
	pl := PlanList{
		Files:  []string{"A_PLAN.md", "B_PLAN.md", "C_PLAN.md", "D_PLAN.md", "E_PLAN.md"},
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

func TestPlanList_ScrollBy(t *testing.T) {
	pl := PlanList{
		Files: []string{"A_PLAN.md", "B_PLAN.md", "C_PLAN.md", "D_PLAN.md",
			"E_PLAN.md", "F_PLAN.md", "G_PLAN.md"},
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

func TestIsInPlanSection(t *testing.T) {
	// Plan section is rows 0-4 (5 rows total)
	if !IsInPlanSection(0) {
		t.Error("row 0 should be in plan section")
	}
	if !IsInPlanSection(4) {
		t.Error("row 4 should be in plan section")
	}
	if IsInPlanSection(5) {
		t.Error("row 5 should not be in plan section")
	}
}

func TestPlanSectionRow(t *testing.T) {
	// Plan section starts at row 0, so pane-relative == section-relative
	got := PlanSectionRow(3)
	if got != 3 {
		t.Errorf("PlanSectionRow(3) = %d, want 3", got)
	}
}

func TestPlanListTotalHeight(t *testing.T) {
	h := PlanListTotalHeight()
	if h != 5 {
		t.Errorf("PlanListTotalHeight() = %d, want 5 (4 content + 1 title)", h)
	}
}
