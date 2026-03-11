package tui

import (
	"strings"
	"testing"

	"github.com/loxstomper/skinner/internal/theme"
)

func diffTestTheme() theme.Theme {
	th, _ := theme.LookupTheme("solarized-dark")
	return th
}

func testDiffProps(width int, diff string) DiffViewProps {
	return DiffViewProps{
		Hunks:     ParseUnifiedDiff(diff),
		FilePath:  "example.go",
		Theme:     diffTestTheme(),
		ThemeName: "solarized-dark",
		Width:     width,
		HScroll:   0,
	}
}

// simpleDiff is a minimal unified diff for testing.
const simpleDiff = `@@ -1,3 +1,3 @@
 func main() {
-    fmt.Println("hello")
+    fmt.Println("world")
 }`

func TestRenderDiff_SideBySideAboveThreshold(t *testing.T) {
	props := testDiffProps(100, simpleDiff)
	result := RenderDiff(props)
	if result == "" {
		t.Fatal("expected non-empty output for side-by-side render")
	}
	// Side-by-side should have content on both sides of a "│" separator.
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		// Each line should contain the "│" column separator.
		if !strings.Contains(line, "│") {
			t.Errorf("side-by-side line missing │ separator: %q", line)
		}
	}
}

func TestRenderDiff_UnifiedBelowThreshold(t *testing.T) {
	props := testDiffProps(60, simpleDiff)
	result := RenderDiff(props)
	if result == "" {
		t.Fatal("expected non-empty output for unified render")
	}
	lines := strings.Split(result, "\n")
	// Unified should have 4 lines: context, removed, added, context
	if len(lines) != 4 {
		t.Fatalf("unified render: expected 4 lines, got %d", len(lines))
	}
}

func TestRenderDiff_WidthThresholdExact(t *testing.T) {
	// Width 80 should use side-by-side; 79 should use unified.
	diffText := `@@ -1,2 +1,2 @@
-old line
+new line`

	propsSBS := testDiffProps(80, diffText)
	resultSBS := RenderDiff(propsSBS)
	linesSBS := strings.Split(resultSBS, "\n")

	propsUni := testDiffProps(79, diffText)
	resultUni := RenderDiff(propsUni)
	linesUni := strings.Split(resultUni, "\n")

	// Side-by-side paired: 1 row (old paired with new).
	// Unified: 2 rows (removed then added).
	if len(linesSBS) >= len(linesUni) {
		t.Errorf("side-by-side should have fewer lines than unified for paired change: sbs=%d, uni=%d",
			len(linesSBS), len(linesUni))
	}
}

func TestRenderDiff_LineNumbersPresent(t *testing.T) {
	props := testDiffProps(100, simpleDiff)
	result := RenderDiff(props)
	// Line numbers 1, 2, 3 should appear in the output.
	for _, num := range []string{"1", "2", "3"} {
		if !strings.Contains(result, num) {
			t.Errorf("expected line number %s in output", num)
		}
	}
}

func TestRenderDiff_HorizontalScroll(t *testing.T) {
	longDiff := `@@ -1,1 +1,1 @@
-abcdefghijklmnopqrstuvwxyz
+ABCDEFGHIJKLMNOPQRSTUVWXYZ`

	// Without scroll — content starts from the beginning.
	props := testDiffProps(100, longDiff)
	noScroll := RenderDiff(props)

	// With scroll of 5 — first 5 characters of content should be clipped.
	props.HScroll = 5
	scrolled := RenderDiff(props)

	// The non-scrolled version should contain "abc" and the scrolled should not.
	if !strings.Contains(noScroll, "abc") {
		t.Error("expected 'abc' in non-scrolled output")
	}
	if strings.Contains(scrolled, "abc") {
		t.Error("expected 'abc' to be clipped in scrolled output")
	}
	// The scrolled version should contain content starting from offset.
	if !strings.Contains(scrolled, "fgh") {
		t.Error("expected 'fgh' in scrolled output")
	}
}

func TestRenderDiff_EmptyHunks(t *testing.T) {
	props := DiffViewProps{
		Hunks: nil,
		Width: 100,
	}
	result := RenderDiff(props)
	if result != "" {
		t.Errorf("expected empty output for nil hunks, got %q", result)
	}
}

func TestRenderDiff_UnifiedPrefixes(t *testing.T) {
	props := testDiffProps(60, simpleDiff)
	result := RenderDiff(props)
	lines := strings.Split(result, "\n")
	// Unified format: context lines have no +/- prefix,
	// removed lines have "-", added lines have "+".
	// Line 0: context (func main)
	// Line 1: removed
	// Line 2: added
	// Line 3: context
	if !strings.Contains(lines[1], "-") {
		t.Errorf("expected '-' prefix in removed line: %q", lines[1])
	}
	if !strings.Contains(lines[2], "+") {
		t.Errorf("expected '+' prefix in added line: %q", lines[2])
	}
}

func TestRenderDiff_SideBySidePadding(t *testing.T) {
	// A diff with only additions should have blank left side.
	addOnly := `@@ -1,0 +1,2 @@
+new line 1
+new line 2`

	props := testDiffProps(100, addOnly)
	result := RenderDiff(props)
	if result == "" {
		t.Fatal("expected non-empty output")
	}
	// Should still render (blank left, content right).
	lines := strings.Split(result, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
}

func TestDigitCount(t *testing.T) {
	tests := []struct {
		n    int
		want int
	}{
		{0, 1},
		{1, 1},
		{9, 1},
		{10, 2},
		{99, 2},
		{100, 3},
		{999, 3},
		{1000, 4},
	}
	for _, tt := range tests {
		got := digitCount(tt.n)
		if got != tt.want {
			t.Errorf("digitCount(%d) = %d, want %d", tt.n, got, tt.want)
		}
	}
}

func TestChromaStyleName(t *testing.T) {
	tests := []struct {
		theme string
		want  string
	}{
		{"solarized-dark", "solarized-dark"},
		{"solarized-light", "solarized-light"},
		{"monokai", "monokai"},
		{"nord", "nord"},
		{"unknown", "monokai"},
	}
	for _, tt := range tests {
		got := chromaStyleName(tt.theme)
		if got != tt.want {
			t.Errorf("chromaStyleName(%q) = %q, want %q", tt.theme, got, tt.want)
		}
	}
}

func TestRenderDiff_SyntaxHighlighting(t *testing.T) {
	// Go source should get syntax-highlighted with chroma.
	goDiff := `@@ -1,1 +1,1 @@
-func old() string { return "hello" }
+func new() string { return "world" }`

	props := testDiffProps(120, goDiff)
	result := RenderDiff(props)
	// The output should be non-empty and contain the function names.
	if !strings.Contains(result, "func") {
		t.Error("expected 'func' keyword in syntax-highlighted output")
	}
	if !strings.Contains(result, "return") {
		t.Error("expected 'return' keyword in syntax-highlighted output")
	}
}

func TestRenderDiff_IntraLineEmphasis(t *testing.T) {
	// When a line has a small change, intra-line emphasis should be applied.
	diff := `@@ -1,1 +1,1 @@
-tab := strings.Split(s)
+tab := strings.Fields(s)`

	props := testDiffProps(120, diff)
	result := RenderDiff(props)
	// The output should contain both the old and new content.
	if !strings.Contains(result, "Split") {
		t.Error("expected 'Split' in output")
	}
	if !strings.Contains(result, "Fields") {
		t.Error("expected 'Fields' in output")
	}
}
