package tui

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/loxstomper/skinner/internal/theme"
)

// PlanList is the top section of the left pane that displays *_PLAN.md files.
// It owns its cursor position, scroll offset, and the cached list of plan files.
type PlanList struct {
	Cursor int
	Scroll int
	Files  []string // full filenames (e.g. "IMPLEMENTATION_PLAN.md")
}

// NewPlanList creates a new PlanList and scans the given directory for plan files.
func NewPlanList(dir string) PlanList {
	pl := PlanList{}
	pl.ScanFiles(dir)
	return pl
}

// PlanListProps contains the data needed to render the plan list.
type PlanListProps struct {
	Width   int
	Height  int // total height including title row
	Focused bool
	Theme   theme.Theme
}

// planListContentHeight is the fixed number of rows for file entries (excluding title).
const planListContentHeight = 4

// PlanListTotalHeight returns the total height of the plan list section
// including the title row.
func PlanListTotalHeight() int {
	return planListContentHeight + 1 // 4 content rows + 1 title row
}

// ScanFiles scans the given directory for *_PLAN.md files and updates the file list.
// It preserves the cursor position if possible.
func (pl *PlanList) ScanFiles(dir string) {
	pattern := filepath.Join(dir, "*_PLAN.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		pl.Files = nil
		return
	}

	// Extract just the filenames and sort alphabetically
	files := make([]string, 0, len(matches))
	for _, m := range matches {
		files = append(files, filepath.Base(m))
	}
	sort.Strings(files)
	pl.Files = files

	// Clamp cursor to valid range
	if pl.Cursor >= len(pl.Files) {
		pl.Cursor = len(pl.Files) - 1
	}
	if pl.Cursor < 0 {
		pl.Cursor = 0
	}
}

// PlanDisplayName strips the _PLAN.md suffix from a filename.
func PlanDisplayName(filename string) string {
	name := strings.TrimSuffix(filename, "_PLAN.md")
	return name
}

// SelectedFile returns the currently selected filename, or "" if no files exist.
func (pl *PlanList) SelectedFile() string {
	if len(pl.Files) == 0 || pl.Cursor >= len(pl.Files) {
		return ""
	}
	return pl.Files[pl.Cursor]
}

// HandleAction processes a resolved action for the plan list.
func (pl *PlanList) HandleAction(action string, props PlanListProps) {
	count := len(pl.Files)
	if count == 0 {
		return
	}

	switch action {
	case "move_down":
		if pl.Cursor < count-1 {
			pl.Cursor++
		}
		pl.ensureCursorVisible()

	case "move_up":
		if pl.Cursor > 0 {
			pl.Cursor--
		}
		pl.ensureCursorVisible()

	case "jump_bottom":
		pl.Cursor = count - 1
		pl.ensureCursorVisible()

	case "jump_top":
		pl.Cursor = 0
		pl.Scroll = 0

	case "page_down":
		pl.Cursor += planListContentHeight
		if pl.Cursor >= count {
			pl.Cursor = count - 1
		}
		pl.ensureCursorVisible()

	case "page_up":
		pl.Cursor -= planListContentHeight
		if pl.Cursor < 0 {
			pl.Cursor = 0
		}
		pl.ensureCursorVisible()
	}
}

// View renders the plan list with a title line and scrollable file list.
func (pl *PlanList) View(props PlanListProps) string {
	totalHeight := props.Height
	if totalHeight < 1 {
		return ""
	}

	// Title line
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(props.Theme.Foreground)).
		Bold(true)
	title := titleStyle.Render("  📋 Plans")

	contentHeight := totalHeight - 1 // subtract title row
	if contentHeight < 0 {
		contentHeight = 0
	}

	highlight := lipgloss.NewStyle().Background(lipgloss.Color(props.Theme.Highlight))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(props.Theme.ForegroundDim))

	var contentLines []string

	if len(pl.Files) == 0 {
		contentLines = append(contentLines, dimStyle.Render("  No plan files"))
	} else {
		for i, f := range pl.Files {
			displayName := PlanDisplayName(f)
			line := "  " + displayName

			if props.Focused && i == pl.Cursor {
				displayWidth := lipgloss.Width(line)
				if displayWidth < props.Width {
					line += strings.Repeat(" ", props.Width-displayWidth)
				}
				line = highlight.Render(line)
			} else {
				line = lipgloss.NewStyle().
					Foreground(lipgloss.Color(props.Theme.Foreground)).
					Render(line)
			}
			contentLines = append(contentLines, line)
		}
	}

	// Apply scroll: render only the visible slice
	start := pl.Scroll
	if start >= len(contentLines) {
		start = len(contentLines) - 1
	}
	if start < 0 {
		start = 0
	}
	end := start + contentHeight
	if end > len(contentLines) {
		end = len(contentLines)
	}

	visible := contentLines[start:end]

	// Build the output with title + content, constrained to total height
	var lines []string
	lines = append(lines, title)
	lines = append(lines, visible...)

	// Pad to fill the total height
	for len(lines) < totalHeight {
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().Width(props.Width).Height(totalHeight).Render(content)
}

// ViewBottom renders a compact 2-row plan list for the bottom bar (no title line).
func (pl *PlanList) ViewBottom(props PlanListProps) string {
	height := props.Height
	style := lipgloss.NewStyle().Width(props.Width).Height(height)
	highlight := lipgloss.NewStyle().Background(lipgloss.Color(props.Theme.Highlight))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(props.Theme.ForegroundDim))

	var contentLines []string

	if len(pl.Files) == 0 {
		contentLines = append(contentLines, dimStyle.Render("  No plan files"))
	} else {
		for i, f := range pl.Files {
			displayName := PlanDisplayName(f)
			line := "  " + displayName

			if props.Focused && i == pl.Cursor {
				displayWidth := lipgloss.Width(line)
				if displayWidth < props.Width {
					line += strings.Repeat(" ", props.Width-displayWidth)
				}
				line = highlight.Render(line)
			} else {
				line = lipgloss.NewStyle().
					Foreground(lipgloss.Color(props.Theme.Foreground)).
					Render(line)
			}
			contentLines = append(contentLines, line)
		}
	}

	// Apply scroll: show only the visible slice
	start := pl.Scroll
	if start >= len(contentLines) {
		start = len(contentLines) - 1
	}
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > len(contentLines) {
		end = len(contentLines)
	}

	visible := contentLines[start:end]

	// Pad to fill height
	for len(visible) < height {
		visible = append(visible, "")
	}

	return style.Render(strings.Join(visible, "\n"))
}

// ensureCursorVisible adjusts scroll so the cursor row is within the viewport.
func (pl *PlanList) ensureCursorVisible() {
	if pl.Cursor < pl.Scroll {
		pl.Scroll = pl.Cursor
	}
	if pl.Cursor >= pl.Scroll+planListContentHeight {
		pl.Scroll = pl.Cursor - planListContentHeight + 1
	}
	pl.clampScroll(planListContentHeight)
}

// clampScroll ensures scroll doesn't exceed the maximum valid offset.
func (pl *PlanList) clampScroll(contentHeight int) {
	count := len(pl.Files)
	maxScroll := count - contentHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if pl.Scroll > maxScroll {
		pl.Scroll = maxScroll
	}
	if pl.Scroll < 0 {
		pl.Scroll = 0
	}
}

// ScrollBy adjusts the scroll offset by delta lines.
func (pl *PlanList) ScrollBy(delta int) {
	pl.Scroll += delta
	pl.clampScroll(planListContentHeight)
}

// ClickRow handles a mouse click on the given row relative to the plan list section.
// Row 0 is the title row (ignored). Returns true if the cursor changed.
func (pl *PlanList) ClickRow(row int) bool {
	// Row 0 is the title, content starts at row 1
	if row < 1 {
		return false
	}
	target := pl.Scroll + (row - 1) // subtract title row
	if target < 0 || target >= len(pl.Files) {
		return false
	}
	pl.Cursor = target
	return true
}

// IsInPlanSection returns true if the given pane-relative row falls within
// the plan list section at the top of the left pane.
func IsInPlanSection(row int) bool {
	return row < PlanListTotalHeight()
}

// PlanSectionRow converts a pane-relative row to a plan-section-relative row.
func PlanSectionRow(row int) int {
	return row
}
