# Implementation Plan

## 1. Fix tool call row highlight (per-segment background)

**File**: `internal/tui/timeline.go` — `renderToolCallLine`, `renderGroupHeaderLine`

- Add an optional background color parameter (e.g. `highlightBg string`) to both functions
- When non-empty, apply `.Background(lipgloss.Color(highlightBg))` alongside each segment's `.Foreground()` style
- This ensures the highlight background survives across all ANSI segments instead of being reset by inner escape codes

**File**: `internal/tui/timeline.go` — `renderTextBlockLines`

- Add the same optional background color parameter for consistency (text blocks may work by accident today but should use the same approach)

**File**: `internal/tui/timeline.go` — `View` method

- When building `renderedLine` entries, pass the highlight background color for the row at the current cursor position
- Store the highlight color in `renderedLine` (e.g. a `highlighted bool` field) so `renderWithLines` can use it

**File**: `internal/tui/timeline.go` — `renderWithLines`

- Remove the post-hoc `highlight.Render(text)` wrapping for highlighted rows (lines 522-528)
- The highlight is now baked into each row during rendering, so padding to full width still needs to happen but no outer wrap is needed

**Tests**:

- Test that `renderToolCallLine` with a highlight background produces output where the background color spans the full row (verify ANSI sequences)
