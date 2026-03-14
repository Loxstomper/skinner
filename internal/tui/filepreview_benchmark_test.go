package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeSourceFile generates realistic Go source code at three scale points.
// Content includes package declarations, imports, struct definitions, methods,
// and error handling to exercise chroma's Go lexer across all token types.
//
//   - "small"  ~1KB   — short file, few functions
//   - "medium" ~10KB  — typical file with multiple types and methods
//   - "large"  ~100KB — large file with extensive logic
func makeSourceFile(size string) string {
	var target int
	switch size {
	case "small":
		target = 1024
	case "medium":
		target = 10 * 1024
	case "large":
		target = 100 * 1024
	default:
		panic("unknown size: " + size)
	}

	var b strings.Builder
	b.WriteString("package processor\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"context\"\n")
	b.WriteString("\t\"fmt\"\n")
	b.WriteString("\t\"sync\"\n")
	b.WriteString(")\n\n")

	section := 0
	for b.Len() < target {
		section++
		fmt.Fprintf(&b, "// Handler%d processes items in batch %d.\n", section, section)
		fmt.Fprintf(&b, "type Handler%d struct {\n", section)
		fmt.Fprintf(&b, "\tmu      sync.Mutex\n")
		fmt.Fprintf(&b, "\titems   []Item\n")
		fmt.Fprintf(&b, "\tresults map[string]Result\n")
		fmt.Fprintf(&b, "}\n\n")

		fmt.Fprintf(&b, "// Process runs the handler pipeline for batch %d.\n", section)
		fmt.Fprintf(&b, "func (h *Handler%d) Process(ctx context.Context) error {\n", section)
		fmt.Fprintf(&b, "\th.mu.Lock()\n")
		fmt.Fprintf(&b, "\tdefer h.mu.Unlock()\n\n")
		fmt.Fprintf(&b, "\tfor i, item := range h.items {\n")
		fmt.Fprintf(&b, "\t\tselect {\n")
		fmt.Fprintf(&b, "\t\tcase <-ctx.Done():\n")
		fmt.Fprintf(&b, "\t\t\treturn ctx.Err()\n")
		fmt.Fprintf(&b, "\t\tdefault:\n")
		fmt.Fprintf(&b, "\t\t}\n\n")
		fmt.Fprintf(&b, "\t\tif err := item.Validate(); err != nil {\n")
		fmt.Fprintf(&b, "\t\t\treturn fmt.Errorf(\"item %%d: %%w\", i, err)\n")
		fmt.Fprintf(&b, "\t\t}\n\n")
		fmt.Fprintf(&b, "\t\tresult, err := item.Transform()\n")
		fmt.Fprintf(&b, "\t\tif err != nil {\n")
		fmt.Fprintf(&b, "\t\t\treturn fmt.Errorf(\"transform %%d: %%w\", i, err)\n")
		fmt.Fprintf(&b, "\t\t}\n")
		fmt.Fprintf(&b, "\t\th.results[item.Key()] = result\n")
		fmt.Fprintf(&b, "\t}\n")
		fmt.Fprintf(&b, "\treturn nil\n")
		fmt.Fprintf(&b, "}\n\n")

		fmt.Fprintf(&b, "// Count returns the number of processed results in batch %d.\n", section)
		fmt.Fprintf(&b, "func (h *Handler%d) Count() int {\n", section)
		fmt.Fprintf(&b, "\th.mu.Lock()\n")
		fmt.Fprintf(&b, "\tdefer h.mu.Unlock()\n")
		fmt.Fprintf(&b, "\treturn len(h.results)\n")
		fmt.Fprintf(&b, "}\n\n")
	}
	return b.String()
}

// BenchmarkFilePreviewMarkdownUncached measures the full file preview render
// path for markdown files on a cache miss: file read + glamour rendering.
// Each iteration uses a nil cache, forcing a fresh glamour render every time.
// This is the baseline showing why caching is needed for markdown preview.
func BenchmarkFilePreviewMarkdownUncached(b *testing.B) {
	for _, size := range []string{"small", "medium", "large"} {
		b.Run(size, func(b *testing.B) {
			dir := b.TempDir()
			path := "preview.md"
			content := makePlanMarkdown(size)
			if err := os.WriteFile(filepath.Join(dir, path), []byte(content), 0644); err != nil {
				b.Fatal(err)
			}

			props := FilePreviewProps{
				Path:      path,
				Dir:       dir,
				Width:     100,
				Height:    50,
				Theme:     lookupTestTheme(),
				ThemeName: "dark",
				Cache:     nil, // nil cache forces full render every iteration
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				RenderFilePreview(props)
			}
		})
	}
}

// BenchmarkFilePreviewMarkdownCached measures the file preview render path
// for markdown files on a cache hit: os.Stat check + scroll slicing only.
// The cache is primed once before the timer starts. Cached iterations should
// be near-zero cost regardless of file size, confirming the cache eliminates
// glamour as a per-frame bottleneck in file preview.
func BenchmarkFilePreviewMarkdownCached(b *testing.B) {
	for _, size := range []string{"small", "medium", "large"} {
		b.Run(size, func(b *testing.B) {
			dir := b.TempDir()
			path := "preview.md"
			content := makePlanMarkdown(size)
			if err := os.WriteFile(filepath.Join(dir, path), []byte(content), 0644); err != nil {
				b.Fatal(err)
			}

			cache := &RenderCache{}
			props := FilePreviewProps{
				Path:      path,
				Dir:       dir,
				Width:     100,
				Height:    50,
				Theme:     lookupTestTheme(),
				ThemeName: "dark",
				Cache:     cache,
			}

			// Prime the cache with one render
			RenderFilePreview(props)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				RenderFilePreview(props)
			}
		})
	}
}

// BenchmarkFilePreviewSourceUncached measures the full file preview render
// path for source code on a cache miss: file read + line split + chroma
// highlighting of visible lines. Each iteration uses a nil cache, forcing
// a fresh file read and split every time.
func BenchmarkFilePreviewSourceUncached(b *testing.B) {
	for _, size := range []string{"small", "medium", "large"} {
		b.Run(size, func(b *testing.B) {
			dir := b.TempDir()
			path := "handler.go"
			content := makeSourceFile(size)
			if err := os.WriteFile(filepath.Join(dir, path), []byte(content), 0644); err != nil {
				b.Fatal(err)
			}

			props := FilePreviewProps{
				Path:            path,
				Dir:             dir,
				Width:           100,
				Height:          50,
				ShowLineNumbers: true,
				Theme:           lookupTestTheme(),
				ThemeName:       "dark",
				Cache:           nil, // nil cache forces full render every iteration
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				RenderFilePreview(props)
			}
		})
	}
}

// BenchmarkFilePreviewSourceCached measures the file preview render path for
// source code on a cache hit: os.Stat check + chroma highlighting of visible
// lines. The file read and line split are skipped. Chroma highlighting is
// O(visible) and still runs per-frame — only the I/O is cached.
func BenchmarkFilePreviewSourceCached(b *testing.B) {
	for _, size := range []string{"small", "medium", "large"} {
		b.Run(size, func(b *testing.B) {
			dir := b.TempDir()
			path := "handler.go"
			content := makeSourceFile(size)
			if err := os.WriteFile(filepath.Join(dir, path), []byte(content), 0644); err != nil {
				b.Fatal(err)
			}

			cache := &RenderCache{}
			props := FilePreviewProps{
				Path:            path,
				Dir:             dir,
				Width:           100,
				Height:          50,
				ShowLineNumbers: true,
				Theme:           lookupTestTheme(),
				ThemeName:       "dark",
				Cache:           cache,
			}

			// Prime the cache with one render
			RenderFilePreview(props)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				RenderFilePreview(props)
			}
		})
	}
}
