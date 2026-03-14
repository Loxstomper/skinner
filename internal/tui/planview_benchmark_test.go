package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makePlanMarkdown generates realistic plan-style markdown content at three
// scale points. Content includes headings, prose, bullet lists, and fenced
// code blocks to exercise all glamour rendering paths.
//
//   - "small"  ~1KB   — short plan, few sections
//   - "medium" ~10KB  — typical plan with multiple sections and code blocks
//   - "large"  ~100KB — very large plan with extensive code examples
func makePlanMarkdown(size string) string {
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
	section := 0
	for b.Len() < target {
		section++
		fmt.Fprintf(&b, "## Section %d — Feature Implementation\n\n", section)
		fmt.Fprintf(&b, "This section covers the implementation of feature %d. ", section)
		fmt.Fprintf(&b, "The goal is to refactor the existing handler to support concurrent processing ")
		fmt.Fprintf(&b, "while maintaining backward compatibility with the v1 API surface.\n\n")

		fmt.Fprintf(&b, "### Requirements\n\n")
		for i := 1; i <= 5; i++ {
			fmt.Fprintf(&b, "- Requirement %d.%d: Ensure the pipeline handles edge cases for input validation\n", section, i)
		}
		b.WriteString("\n")

		fmt.Fprintf(&b, "### Implementation\n\n")
		fmt.Fprintf(&b, "```go\n")
		fmt.Fprintf(&b, "func Process%d(ctx context.Context, items []Item) error {\n", section)
		fmt.Fprintf(&b, "\tfor i, item := range items {\n")
		fmt.Fprintf(&b, "\t\tif err := validate(item); err != nil {\n")
		fmt.Fprintf(&b, "\t\t\treturn fmt.Errorf(\"item %%d: %%w\", i, err)\n")
		fmt.Fprintf(&b, "\t\t}\n")
		fmt.Fprintf(&b, "\t\tif err := store(ctx, item); err != nil {\n")
		fmt.Fprintf(&b, "\t\t\treturn err\n")
		fmt.Fprintf(&b, "\t\t}\n")
		fmt.Fprintf(&b, "\t}\n")
		fmt.Fprintf(&b, "\treturn nil\n")
		fmt.Fprintf(&b, "}\n")
		fmt.Fprintf(&b, "```\n\n")

		fmt.Fprintf(&b, "### Notes\n\n")
		fmt.Fprintf(&b, "The approach above is intentionally sequential. A future iteration may ")
		fmt.Fprintf(&b, "introduce a worker pool, but premature concurrency adds complexity without ")
		fmt.Fprintf(&b, "measurable benefit at current throughput levels.\n\n")
	}
	return b.String()
}

// BenchmarkPlanViewUncached measures the full render path on a cache miss:
// file read + glamour markdown rendering. Each iteration invalidates the cache
// by using a nil cache, forcing a fresh glamour render every time. This is the
// baseline showing why caching is needed — glamour is expensive for large files.
func BenchmarkPlanViewUncached(b *testing.B) {
	for _, size := range []string{"small", "medium", "large"} {
		b.Run(size, func(b *testing.B) {
			dir := b.TempDir()
			filename := "PLAN.md"
			content := makePlanMarkdown(size)
			if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644); err != nil {
				b.Fatal(err)
			}

			props := PlanViewProps{
				Filename: filename,
				Dir:      dir,
				Width:    100,
				Height:   50,
				Theme:    lookupTestTheme(),
				Cache:    nil, // nil cache forces full render every iteration
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				RenderPlanView(props)
			}
		})
	}
}

// BenchmarkPlanViewCached measures the render path on a cache hit: os.Stat
// check + scroll slicing only. The cache is primed once before the timer
// starts. Cached iterations should be near-zero cost regardless of file size,
// confirming the cache eliminates glamour as a per-frame bottleneck.
func BenchmarkPlanViewCached(b *testing.B) {
	for _, size := range []string{"small", "medium", "large"} {
		b.Run(size, func(b *testing.B) {
			dir := b.TempDir()
			filename := "PLAN.md"
			content := makePlanMarkdown(size)
			if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644); err != nil {
				b.Fatal(err)
			}

			cache := &RenderCache{}
			props := PlanViewProps{
				Filename: filename,
				Dir:      dir,
				Width:    100,
				Height:   50,
				Theme:    lookupTestTheme(),
				Cache:    cache,
			}

			// Prime the cache with one render
			RenderPlanView(props)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				RenderPlanView(props)
			}
		})
	}
}
