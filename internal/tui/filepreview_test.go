package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/loxstomper/skinner/internal/theme"
)

func previewTestTheme() theme.Theme {
	return theme.Theme{
		Foreground:    "#839496",
		ForegroundDim: "#586e75",
		Highlight:     "#073642",
		DiffAdded:     "#859900",
		DiffRemoved:   "#dc322f",
	}
}

func previewProps(dir, path string, w, h int) FilePreviewProps {
	return FilePreviewProps{
		Path:      path,
		Dir:       dir,
		Width:     w,
		Height:    h,
		ThemeName: "solarized-dark",
		Theme:     previewTestTheme(),
	}
}

func TestRenderFilePreview_SourceCode(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n"), 0o644)

	props := previewProps(dir, "main.go", 60, 10)
	result := RenderFilePreview(props)

	if result.TotalLines != 5 {
		t.Errorf("expected 5 total lines, got %d", result.TotalLines)
	}
	if !strings.Contains(result.Content, "main.go") {
		t.Error("title bar should contain filename")
	}
	if !strings.Contains(result.Content, "package") {
		t.Error("content should contain 'package'")
	}
	if !strings.Contains(result.Content, "hello") {
		t.Error("content should contain 'hello'")
	}
}

func TestRenderFilePreview_SourceCodeWithChroma(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "test.go"), []byte("package main\n\nimport \"fmt\"\n"), 0o644)

	props := previewProps(dir, "test.go", 60, 10)
	result := RenderFilePreview(props)

	// Chroma should tokenize the Go source — output should not be empty
	if result.Content == "" {
		t.Error("source code preview should not be empty")
	}
	if result.TotalLines != 3 {
		t.Errorf("expected 3 total lines, got %d", result.TotalLines)
	}
}

func TestRenderFilePreview_Markdown(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Hello World\n\nThis is a test.\n"), 0o644)

	props := previewProps(dir, "README.md", 60, 10)
	result := RenderFilePreview(props)

	if !strings.Contains(result.Content, "README.md") {
		t.Error("title bar should contain filename")
	}
	// Glamour renders "# Hello World" as a styled heading
	if !strings.Contains(result.Content, "Hello World") {
		t.Error("content should contain rendered heading text")
	}
}

func TestRenderFilePreview_Binary(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "image.png"), []byte{0x89, 0x50, 0x4E, 0x47, 0x00, 0x00, 0x01}, 0o644)

	props := previewProps(dir, "image.png", 60, 10)
	result := RenderFilePreview(props)

	if !strings.Contains(result.Content, "Binary file") {
		t.Error("binary file should show binary message")
	}
	if !strings.Contains(result.Content, "preview not available") {
		t.Error("binary file should indicate preview not available")
	}
}

func TestRenderFilePreview_FileNotFound(t *testing.T) {
	dir := t.TempDir()

	props := previewProps(dir, "nonexistent.go", 60, 10)
	result := RenderFilePreview(props)

	if !strings.Contains(result.Content, "File not found") {
		t.Error("missing file should show 'File not found'")
	}
}

func TestRenderFilePreview_EmptyPath(t *testing.T) {
	dir := t.TempDir()

	props := previewProps(dir, "", 60, 10)
	result := RenderFilePreview(props)

	if result.Content == "" {
		t.Error("empty path should still render title bar")
	}
}

func TestRenderFilePreview_LineNumbers(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "test.go"), []byte("line1\nline2\nline3\n"), 0o644)

	props := previewProps(dir, "test.go", 60, 10)
	props.ShowLineNumbers = true
	result := RenderFilePreview(props)

	if !strings.Contains(result.Content, "1") {
		t.Error("should contain line number 1")
	}
	if !strings.Contains(result.Content, "2") {
		t.Error("should contain line number 2")
	}
}

func TestRenderFilePreview_LineNumbersNotShownForMarkdown(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Title\n\nParagraph.\n"), 0o644)

	props := previewProps(dir, "README.md", 60, 10)
	props.ShowLineNumbers = true
	result := RenderFilePreview(props)

	// Markdown is rendered via glamour, not with line numbers
	// The result should still contain content (glamour rendering)
	if !strings.Contains(result.Content, "Title") {
		t.Error("markdown should still render content")
	}
}

func TestRenderFilePreview_Scroll(t *testing.T) {
	dir := t.TempDir()
	var lines []string
	for i := 1; i <= 50; i++ {
		lines = append(lines, "// line "+strings.Repeat("x", i))
	}
	_ = os.WriteFile(filepath.Join(dir, "big.go"), []byte(strings.Join(lines, "\n")+"\n"), 0o644)

	props := previewProps(dir, "big.go", 60, 10)
	props.Scroll = 10
	result := RenderFilePreview(props)

	if result.TotalLines != 50 {
		t.Errorf("expected 50 total lines, got %d", result.TotalLines)
	}
	// The visible content should start from line 11 (scroll=10)
	if !strings.Contains(result.Content, "xxxxxxxxxxx") {
		t.Error("scrolled content should show lines from scroll offset")
	}
}

func TestRenderFilePreview_HScroll(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "wide.go"), []byte("// ABCDEFGHIJKLMNOP\n"), 0o644)

	props := previewProps(dir, "wide.go", 60, 10)
	props.HScroll = 5
	result := RenderFilePreview(props)

	// After hscroll=5, "// AB" should be trimmed, leaving "CDEFGHIJKLMNOP"
	if strings.Contains(result.Content, "// AB") {
		t.Error("horizontal scroll should trim left characters")
	}
}

func TestRenderFilePreview_Directory(t *testing.T) {
	dir := t.TempDir()
	_ = os.Mkdir(filepath.Join(dir, "subdir"), 0o755)

	props := previewProps(dir, "subdir", 60, 10)
	result := RenderFilePreview(props)

	// Directory should show empty preview (just title)
	if result.TotalLines != 0 {
		t.Errorf("directory preview should have 0 content lines, got %d", result.TotalLines)
	}
}

func TestRenderFilePreview_SmallDimensions(t *testing.T) {
	result := RenderFilePreview(FilePreviewProps{Width: 0, Height: 0})
	if result.Content != "" {
		t.Error("zero dimensions should return empty content")
	}

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "tiny.go"), []byte("package main\n"), 0o644)
	props := previewProps(dir, "tiny.go", 30, 5)
	result = RenderFilePreview(props)
	if result.Content == "" {
		t.Error("narrow preview should still render content")
	}

	result = RenderFilePreview(FilePreviewProps{Width: 10, Height: 1})
	// Height=1 means only title bar, no content area
	if result.Content == "" {
		t.Error("height=1 should still render title bar")
	}
}

func TestRenderFilePreview_MarkdownCachePopulatedAndReused(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "cached.md"), []byte("# Cached\n\nMarkdown content.\n"), 0o644)

	cache := &RenderCache{}
	props := previewProps(dir, "cached.md", 60, 10)
	props.Cache = cache

	// First call — cache miss, populates cache
	result1 := RenderFilePreview(props)
	if !strings.Contains(result1.Content, "Cached") {
		t.Error("first render should contain 'Cached'")
	}

	// Verify cache was populated
	fullPath := filepath.Join(dir, "cached.md")
	cachedLines, hit := cache.Get(fullPath, 60)
	if !hit {
		t.Fatal("cache should be populated after first markdown render")
	}
	if len(cachedLines) == 0 {
		t.Fatal("cached lines should not be empty")
	}

	// Second call — cache hit, identical output
	result2 := RenderFilePreview(props)
	if result1.Content != result2.Content {
		t.Error("cached markdown render should produce identical output")
	}
	if result1.TotalLines != result2.TotalLines {
		t.Errorf("totalLines mismatch: first=%d, second=%d", result1.TotalLines, result2.TotalLines)
	}
}

func TestRenderFilePreview_MarkdownCacheInvalidatedOnChange(t *testing.T) {
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "changing.md")
	_ = os.WriteFile(mdPath, []byte("# Original\n\nFirst version.\n"), 0o644)

	cache := &RenderCache{}
	props := previewProps(dir, "changing.md", 60, 10)
	props.Cache = cache

	// First render
	result1 := RenderFilePreview(props)
	if !strings.Contains(result1.Content, "Original") {
		t.Error("first render should contain 'Original'")
	}

	// Modify file with new modtime
	newTime := time.Now().Add(2 * time.Second)
	_ = os.WriteFile(mdPath, []byte("# Updated\n\nSecond version.\n"), 0o644)
	_ = os.Chtimes(mdPath, newTime, newTime)

	// Second render — should show updated content
	result2 := RenderFilePreview(props)
	if !strings.Contains(result2.Content, "Updated") {
		t.Error("second render should contain 'Updated' after file modification")
	}
}

func TestRenderFilePreview_SourceCachePopulatedAndReused(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "cached.go"), []byte("package main\n\nfunc main() {\n\tprintln(\"cached\")\n}\n"), 0o644)

	cache := &RenderCache{}
	props := previewProps(dir, "cached.go", 60, 10)
	props.Cache = cache

	// First call — cache miss, populates cache with raw lines
	result1 := RenderFilePreview(props)
	if result1.TotalLines != 5 {
		t.Errorf("expected 5 total lines, got %d", result1.TotalLines)
	}

	// Verify cache was populated with raw source lines
	fullPath := filepath.Join(dir, "cached.go")
	cachedLines, hit := cache.Get(fullPath, 60)
	if !hit {
		t.Fatal("cache should be populated after first source render")
	}
	// Cache stores raw lines (not chroma-styled), so first line should be "package main"
	if len(cachedLines) == 0 {
		t.Fatal("cached lines should not be empty")
	}
	if cachedLines[0] != "package main" {
		t.Errorf("cached source line should be raw text, got %q", cachedLines[0])
	}

	// Second call — cache hit, chroma still applies to visible lines
	result2 := RenderFilePreview(props)
	if result1.TotalLines != result2.TotalLines {
		t.Errorf("totalLines mismatch: first=%d, second=%d", result1.TotalLines, result2.TotalLines)
	}
	// Content should be identical since chroma runs on same raw lines
	if result1.Content != result2.Content {
		t.Error("cached source render should produce identical output")
	}
}

func TestApplyHScroll(t *testing.T) {
	tests := []struct {
		line    string
		hscroll int
		want    string
	}{
		{"hello world", 0, "hello world"},
		{"hello world", 5, " world"},
		{"hello", 10, ""},
		{"hello", -1, "hello"},
		{"", 5, ""},
	}
	for _, tt := range tests {
		got := applyHScroll(tt.line, tt.hscroll)
		if got != tt.want {
			t.Errorf("applyHScroll(%q, %d) = %q, want %q", tt.line, tt.hscroll, got, tt.want)
		}
	}
}

func TestIsMarkdown(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"README.md", true},
		{"PLAN.MD", true},
		{"doc.markdown", true},
		{"notes.mkd", true},
		{"main.go", false},
		{"style.css", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isMarkdown(tt.path)
		if got != tt.want {
			t.Errorf("isMarkdown(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestClampFilePreviewScroll(t *testing.T) {
	// viewHeight=10 means contentHeight=9 (subtract title)
	// 20 total lines, max scroll = 20-9 = 11
	got := ClampFilePreviewScroll(15, 20, 10)
	if got != 11 {
		t.Errorf("expected clamped scroll 11, got %d", got)
	}

	// Scroll within bounds
	got = ClampFilePreviewScroll(5, 20, 10)
	if got != 5 {
		t.Errorf("expected scroll 5, got %d", got)
	}

	// Negative scroll
	got = ClampFilePreviewScroll(-1, 20, 10)
	if got != 0 {
		t.Errorf("expected scroll 0, got %d", got)
	}

	// viewHeight=1 means no content area
	got = ClampFilePreviewScroll(5, 20, 1)
	if got != 0 {
		t.Errorf("expected scroll 0 for height=1, got %d", got)
	}
}

func TestClampPreviewScrollVal(t *testing.T) {
	// totalLines=5, viewHeight=10 → maxScroll=0
	got := clampPreviewScrollVal(3, 5, 10)
	if got != 0 {
		t.Errorf("expected 0 when content fits viewport, got %d", got)
	}

	// totalLines=20, viewHeight=5 → maxScroll=15
	got = clampPreviewScrollVal(20, 20, 5)
	if got != 15 {
		t.Errorf("expected 15, got %d", got)
	}
}
