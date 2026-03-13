package tui

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/loxstomper/skinner/internal/model"
	"github.com/loxstomper/skinner/internal/theme"
)

// makeTestItems produces a deterministic mix of timeline items for benchmarks.
// Distribution: 60% standalone ToolCall, 25% ToolCallGroup, 15% TextBlock.
func makeTestItems(n int) []model.TimelineItem {
	rng := rand.New(rand.NewSource(42))
	toolNames := []string{"Bash", "Read", "Edit", "Grep", "Glob"}
	items := make([]model.TimelineItem, 0, n)

	for i := 0; i < n; i++ {
		roll := rng.Float64()
		switch {
		case roll < 0.60:
			// Standalone ToolCall
			tc := makeToolCall(rng, toolNames[rng.Intn(len(toolNames))], i)
			if rng.Float64() < 0.10 {
				tc.Expanded = true
			}
			items = append(items, tc)

		case roll < 0.85:
			// ToolCallGroup
			toolName := toolNames[rng.Intn(len(toolNames))]
			childCount := 3 + rng.Intn(6) // 3-8
			children := make([]*model.ToolCall, childCount)
			for j := 0; j < childCount; j++ {
				children[j] = makeToolCall(rng, toolName, i*100+j)
			}
			group := &model.ToolCallGroup{
				ToolName: toolName,
				Children: children,
				Expanded: rng.Float64() < 0.50,
			}
			items = append(items, group)

		default:
			// TextBlock
			lineCount := 1 + rng.Intn(5) // 1-5
			lines := make([]string, lineCount)
			for j := 0; j < lineCount; j++ {
				lines[j] = fmt.Sprintf("Text block %d line %d with some content for realism.", i, j)
			}
			items = append(items, &model.TextBlock{
				Text:     strings.Join(lines, "\n"),
				Expanded: false,
			})
		}
	}
	return items
}

// makeToolCall creates a single completed tool call with realistic content.
func makeToolCall(rng *rand.Rand, toolName string, idx int) *model.ToolCall {
	tc := &model.ToolCall{
		ID:        fmt.Sprintf("tc_%d", idx),
		Name:      toolName,
		Summary:   fmt.Sprintf("%s operation %d", toolName, idx),
		Status:    model.ToolCallDone,
		StartTime: time.Now().Add(-time.Duration(idx) * time.Second),
		Duration:  time.Duration(100+rng.Intn(900)) * time.Millisecond,
	}

	switch toolName {
	case "Bash":
		lineCount := 5 + rng.Intn(16) // 5-20
		lines := make([]string, lineCount)
		for i := 0; i < lineCount; i++ {
			lines[i] = fmt.Sprintf("$ output line %d: some bash result content here", i)
		}
		tc.ResultContent = strings.Join(lines, "\n")
		tc.RawInput = map[string]interface{}{
			"command": fmt.Sprintf("echo 'command %d'", idx),
		}

	case "Edit":
		oldLines := 10 + rng.Intn(21) // 10-30
		newLines := 10 + rng.Intn(21)
		old := make([]string, oldLines)
		new := make([]string, newLines)
		for i := 0; i < oldLines; i++ {
			old[i] = fmt.Sprintf("  old line %d: original content here", i)
		}
		for i := 0; i < newLines; i++ {
			new[i] = fmt.Sprintf("  new line %d: replacement content here", i)
		}
		tc.RawInput = map[string]interface{}{
			"file_path":  fmt.Sprintf("internal/pkg/file_%d.go", idx),
			"old_string": strings.Join(old, "\n"),
			"new_string": strings.Join(new, "\n"),
		}
		tc.ResultContent = "Edit applied successfully"

	case "Read":
		tc.RawInput = map[string]interface{}{
			"file_path": fmt.Sprintf("internal/pkg/file_%d.go", idx),
		}
		tc.ResultContent = fmt.Sprintf("Contents of file_%d.go (50 lines)", idx)

	case "Grep":
		tc.RawInput = map[string]interface{}{
			"pattern": fmt.Sprintf("pattern_%d", idx),
		}
		tc.ResultContent = fmt.Sprintf("Found 5 matches for pattern_%d", idx)

	case "Glob":
		tc.RawInput = map[string]interface{}{
			"pattern": fmt.Sprintf("**/*_%d.go", idx),
		}
		tc.ResultContent = fmt.Sprintf("Matched 3 files for pattern_%d", idx)
	}

	return tc
}

// makeCollapsedTestItems produces n collapsed tool calls — simulates the
// feed-watching case where no items are expanded. Used by the scaling benchmark
// to verify O(visible) rendering independent of total item count.
func makeCollapsedTestItems(n int) []model.TimelineItem {
	rng := rand.New(rand.NewSource(42))
	toolNames := []string{"Bash", "Read", "Edit", "Grep", "Glob"}
	items := make([]model.TimelineItem, 0, n)
	for i := 0; i < n; i++ {
		tc := makeToolCall(rng, toolNames[rng.Intn(len(toolNames))], i)
		tc.Expanded = false
		items = append(items, tc)
	}
	return items
}

func lookupTestTheme() theme.Theme {
	th, _ := theme.LookupTheme("solarized-dark")
	return th
}

func BenchmarkTimelineView(b *testing.B) {
	for _, n := range []int{50, 200, 500} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			items := makeTestItems(n)
			tl := NewTimeline()
			props := TimelineProps{
				Items:       items,
				Width:       140,
				Height:      50,
				Focused:     true,
				CompactView: false,
				LineNumbers: true,
				Theme:       lookupTestTheme(),
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tl.View(props)
			}
		})
	}
}

func BenchmarkFlatCursorLineRange(b *testing.B) {
	for _, n := range []int{50, 200, 500} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			items := makeTestItems(n)
			maxPos := FlatCursorCount(items) - 1
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				FlatCursorLineRange(items, maxPos, false, 140)
			}
		})
	}
}

func BenchmarkTotalLines(b *testing.B) {
	for _, n := range []int{50, 200, 500} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			items := makeTestItems(n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				TotalLines(items, false, 140)
			}
		})
	}
}

func BenchmarkFlatCursorCount(b *testing.B) {
	for _, n := range []int{50, 200, 500} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			items := makeTestItems(n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				FlatCursorCount(items)
			}
		})
	}
}

func BenchmarkExpandedContentLines(b *testing.B) {
	rng := rand.New(rand.NewSource(99))

	b.Run("Bash", func(b *testing.B) {
		tc := makeToolCall(rng, "Bash", 0)
		tc.Expanded = true
		// Ensure 20 lines of result content
		lines := make([]string, 20)
		for i := 0; i < 20; i++ {
			lines[i] = fmt.Sprintf("$ bash output line %d with realistic content", i)
		}
		tc.ResultContent = strings.Join(lines, "\n")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			expandedContentLines(tc)
		}
	})

	b.Run("Edit", func(b *testing.B) {
		tc := makeToolCall(rng, "Edit", 0)
		tc.Expanded = true
		// Ensure ~20 lines each for old/new
		old := make([]string, 20)
		new := make([]string, 20)
		for i := 0; i < 20; i++ {
			old[i] = fmt.Sprintf("  old line %d: original content for diff benchmark", i)
			new[i] = fmt.Sprintf("  new line %d: replacement content for diff benchmark", i)
		}
		tc.RawInput = map[string]interface{}{
			"file_path":  "internal/pkg/benchmark_file.go",
			"old_string": strings.Join(old, "\n"),
			"new_string": strings.Join(new, "\n"),
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			expandedContentLines(tc)
		}
	})
}

func BenchmarkNewItemArrival(b *testing.B) {
	for _, n := range []int{50, 200, 500} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			items := makeTestItems(n)
			tl := NewTimeline()
			props := TimelineProps{
				Items:       items,
				Width:       140,
				Height:      50,
				Focused:     true,
				CompactView: false,
				LineNumbers: true,
				Theme:       lookupTestTheme(),
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tl.OnNewItems(props)
				tl.View(props)
			}
		})
	}
}

// BenchmarkTimelineViewScaling verifies that viewport-only rendering scales with
// viewport height, not total item count. All items are collapsed with auto-follow
// active (pinned to bottom). Times should be roughly constant across n values,
// confirming O(visible) behavior.
func BenchmarkTimelineViewScaling(b *testing.B) {
	for _, n := range []int{50, 200, 500, 1000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			items := makeCollapsedTestItems(n)
			tl := NewTimeline()
			props := TimelineProps{
				Items:       items,
				Width:       140,
				Height:      50,
				Focused:     true,
				CompactView: false,
				LineNumbers: true,
				Theme:       lookupTestTheme(),
			}
			tl.scrollToBottom(props)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tl.View(props)
			}
		})
	}
}
