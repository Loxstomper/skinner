package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/loxstomper/skinner/internal/theme"
)

// PromptList is the bottom section of the left pane that displays PROMPT_*.md files.
// It owns its cursor position, scroll offset, and the cached list of prompt files.
type PromptList struct {
	Cursor int
	Scroll int
	Files  []string // full filenames (e.g. "PROMPT_BUILD.md")
}

// NewPromptList creates a new PromptList and scans the given directory for prompt files.
func NewPromptList(dir string) PromptList {
	pl := PromptList{}
	pl.ScanFiles(dir)
	return pl
}

// PromptListProps contains the data needed to render the prompt list.
type PromptListProps struct {
	Width   int
	Height  int // total height including title row
	Focused bool
	Theme   theme.Theme
}

// promptListContentHeight is the fixed number of rows for file entries (excluding title).
const promptListContentHeight = 4

// PromptListTotalHeight returns the total height of the prompt list section
// including the title row.
func PromptListTotalHeight() int {
	return promptListContentHeight + 1 // 4 content rows + 1 title row
}

// ScanFiles scans the given directory for PROMPT_*.md files and updates the file list.
// It preserves the cursor position if possible.
func (pl *PromptList) ScanFiles(dir string) {
	pattern := filepath.Join(dir, "PROMPT_*.md")
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

// DisplayName strips the PROMPT_ prefix and .md suffix from a filename.
func DisplayName(filename string) string {
	name := strings.TrimPrefix(filename, "PROMPT_")
	name = strings.TrimSuffix(name, ".md")
	return name
}

// SelectedFile returns the currently selected filename, or "" if no files exist.
func (pl *PromptList) SelectedFile() string {
	if len(pl.Files) == 0 || pl.Cursor >= len(pl.Files) {
		return ""
	}
	return pl.Files[pl.Cursor]
}

// HandleAction processes a resolved action for the prompt list.
func (pl *PromptList) HandleAction(action string, props PromptListProps) {
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
		pl.Cursor += promptListContentHeight
		if pl.Cursor >= count {
			pl.Cursor = count - 1
		}
		pl.ensureCursorVisible()

	case "page_up":
		pl.Cursor -= promptListContentHeight
		if pl.Cursor < 0 {
			pl.Cursor = 0
		}
		pl.ensureCursorVisible()
	}
}

// View renders the prompt list with a title line and scrollable file list.
func (pl *PromptList) View(props PromptListProps) string {
	totalHeight := props.Height
	if totalHeight < 1 {
		return ""
	}

	// Title line
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(props.Theme.Foreground)).
		Bold(true)
	title := titleStyle.Render("  📄 Prompts")

	contentHeight := totalHeight - 1 // subtract title row
	if contentHeight < 0 {
		contentHeight = 0
	}

	highlight := lipgloss.NewStyle().Background(lipgloss.Color(props.Theme.Highlight))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(props.Theme.ForegroundDim))

	var contentLines []string

	if len(pl.Files) == 0 {
		contentLines = append(contentLines, dimStyle.Render("  No prompt files"))
	} else {
		for i, f := range pl.Files {
			displayName := DisplayName(f)
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

// ensureCursorVisible adjusts scroll so the cursor row is within the viewport.
func (pl *PromptList) ensureCursorVisible() {
	if pl.Cursor < pl.Scroll {
		pl.Scroll = pl.Cursor
	}
	if pl.Cursor >= pl.Scroll+promptListContentHeight {
		pl.Scroll = pl.Cursor - promptListContentHeight + 1
	}
	pl.clampScroll(promptListContentHeight)
}

// clampScroll ensures scroll doesn't exceed the maximum valid offset.
func (pl *PromptList) clampScroll(contentHeight int) {
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
func (pl *PromptList) ScrollBy(delta int) {
	pl.Scroll += delta
	pl.clampScroll(promptListContentHeight)
}

// ClickRow handles a mouse click on the given row relative to the prompt list section.
// Row 0 is the title row (ignored). Returns true if the cursor changed.
func (pl *PromptList) ClickRow(row int) bool {
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

// IsInPromptSection returns true if the given pane-relative row falls within
// the prompt list section at the bottom of the left pane.
func IsInPromptSection(row int, paneHeight int) bool {
	promptStart := paneHeight - PromptListTotalHeight()
	return row >= promptStart
}

// PromptSectionRow converts a pane-relative row to a prompt-section-relative row.
func PromptSectionRow(row int, paneHeight int) int {
	promptStart := paneHeight - PromptListTotalHeight()
	return row - promptStart
}

// FileExists checks if the file list is non-empty (used to determine if
// fsnotify-style rescan is needed).
func (pl *PromptList) FileExists(name string) bool {
	for _, f := range pl.Files {
		if f == name {
			return true
		}
	}
	return false
}

// ReadFileContent reads the content of a prompt file from the given directory.
func ReadFileContent(dir, filename string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		return "", err
	}
	return string(data), nil
}
