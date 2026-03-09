package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderPlanView_TitleCentered(t *testing.T) {
	dir := t.TempDir()
	filename := "TEST_PLAN.md"
	if err := os.WriteFile(filepath.Join(dir, filename), []byte("# Hello"), 0644); err != nil {
		t.Fatal(err)
	}

	result, _ := RenderPlanView(PlanViewProps{
		Filename: filename,
		Dir:      dir,
		Width:    40,
		Height:   10,
		Theme:    testTheme(),
	})

	// Title should contain the filename
	lines := strings.Split(result, "\n")
	if len(lines) == 0 {
		t.Fatal("expected at least one line")
	}
	if !strings.Contains(lines[0], filename) {
		t.Errorf("expected title to contain %q, got %q", filename, lines[0])
	}
}

func TestRenderPlanView_GlamourRendersOutput(t *testing.T) {
	dir := t.TempDir()
	filename := "TEST_PLAN.md"
	content := "# Heading\n\nSome paragraph text.\n\n- Item one\n- Item two\n"
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, totalLines := RenderPlanView(PlanViewProps{
		Filename: filename,
		Dir:      dir,
		Width:    60,
		Height:   20,
		Theme:    testTheme(),
	})

	// Glamour should produce some output with the heading content
	if !strings.Contains(result, "Heading") {
		t.Error("expected glamour output to contain 'Heading'")
	}
	if totalLines < 1 {
		t.Errorf("expected totalLines > 0, got %d", totalLines)
	}
}

func TestRenderPlanView_FileNotFound(t *testing.T) {
	dir := t.TempDir()

	result, totalLines := RenderPlanView(PlanViewProps{
		Filename: "MISSING_PLAN.md",
		Dir:      dir,
		Width:    40,
		Height:   10,
		Theme:    testTheme(),
	})

	if !strings.Contains(result, "File not found") {
		t.Error("expected 'File not found' message for missing file")
	}
	if totalLines != 0 {
		t.Errorf("expected totalLines=0 for missing file, got %d", totalLines)
	}
}

func TestRenderPlanView_EmptyFilename(t *testing.T) {
	result, totalLines := RenderPlanView(PlanViewProps{
		Filename: "",
		Dir:      t.TempDir(),
		Width:    40,
		Height:   10,
		Theme:    testTheme(),
	})

	if !strings.Contains(result, "No plan selected") {
		t.Error("expected 'No plan selected' message for empty filename")
	}
	if totalLines != 0 {
		t.Errorf("expected totalLines=0 for empty filename, got %d", totalLines)
	}
}

func TestRenderPlanView_ScrollClamping(t *testing.T) {
	dir := t.TempDir()
	filename := "TEST_PLAN.md"
	// Create content with many lines
	var lines []string
	for i := range 50 {
		lines = append(lines, "Line "+string(rune('A'+i%26)))
	}
	content := strings.Join(lines, "\n\n")
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Scroll beyond content — should not panic and should return valid output
	result, totalLines := RenderPlanView(PlanViewProps{
		Filename: filename,
		Dir:      dir,
		Width:    40,
		Height:   10,
		Scroll:   9999,
		Theme:    testTheme(),
	})

	if result == "" {
		t.Error("expected non-empty result even with large scroll")
	}
	if totalLines < 1 {
		t.Errorf("expected totalLines > 0, got %d", totalLines)
	}
}

func TestRenderPlanView_WordWrap(t *testing.T) {
	dir := t.TempDir()
	filename := "TEST_PLAN.md"
	// Long line that should be wrapped
	longLine := strings.Repeat("word ", 30)
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(longLine), 0644); err != nil {
		t.Fatal(err)
	}

	_, totalLines := RenderPlanView(PlanViewProps{
		Filename: filename,
		Dir:      dir,
		Width:    30,
		Height:   20,
		Theme:    testTheme(),
	})

	// With a 30-char width, a 150-char line should wrap into multiple lines
	if totalLines < 2 {
		t.Errorf("expected word wrap to produce multiple lines, got %d", totalLines)
	}
}

func TestRenderPlanView_ZeroSize(t *testing.T) {
	result, _ := RenderPlanView(PlanViewProps{
		Filename: "TEST_PLAN.md",
		Dir:      t.TempDir(),
		Width:    0,
		Height:   0,
		Theme:    testTheme(),
	})

	if result != "" {
		t.Errorf("expected empty string for zero size, got %q", result)
	}
}

func TestRenderPlanView_HeightOne(t *testing.T) {
	dir := t.TempDir()
	filename := "TEST_PLAN.md"
	if err := os.WriteFile(filepath.Join(dir, filename), []byte("# Test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Height of 1 means only title fits, no content
	result, totalLines := RenderPlanView(PlanViewProps{
		Filename: filename,
		Dir:      dir,
		Width:    40,
		Height:   1,
		Theme:    testTheme(),
	})

	if !strings.Contains(result, filename) {
		t.Error("expected title even with height=1")
	}
	if totalLines != 0 {
		t.Errorf("expected totalLines=0 with height=1, got %d", totalLines)
	}
}

func TestClampPlanScroll(t *testing.T) {
	tests := []struct {
		name       string
		scroll     int
		totalLines int
		viewHeight int
		want       int
	}{
		{"within bounds", 5, 20, 11, 5},
		{"at max", 10, 20, 11, 10},
		{"beyond max", 15, 20, 11, 10},
		{"negative", -5, 20, 11, 0},
		{"zero total", 0, 0, 11, 0},
		{"small view", 0, 5, 2, 0},  // contentHeight=1, scroll=0 already valid
		{"exact fit", 0, 10, 11, 0}, // contentHeight=10, maxScroll=0
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClampPlanScroll(tt.scroll, tt.totalLines, tt.viewHeight)
			if got != tt.want {
				t.Errorf("ClampPlanScroll(%d, %d, %d) = %d, want %d",
					tt.scroll, tt.totalLines, tt.viewHeight, got, tt.want)
			}
		})
	}
}

func TestRenderMarkdown(t *testing.T) {
	result := renderMarkdown("# Hello World\n\nSome **bold** text.", 60)
	if !strings.Contains(result, "Hello World") {
		t.Errorf("expected rendered markdown to contain 'Hello World', got %q", result)
	}
	if !strings.Contains(result, "bold") {
		t.Errorf("expected rendered markdown to contain 'bold', got %q", result)
	}
	if result == "" {
		t.Error("expected non-empty rendered output")
	}
}
