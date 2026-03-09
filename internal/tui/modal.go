package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/loxstomper/skinner/internal/config"
	"github.com/loxstomper/skinner/internal/theme"
)

// modalType identifies which modal is currently displayed.
type modalType int

const (
	modalNone modalType = iota
	modalQuitConfirm
	modalHelp
	modalPromptRead
	modalRunConfig
)

// RenderQuitConfirmModal renders a centered quit confirmation overlay.
func RenderQuitConfirmModal(width, height int, th theme.Theme) string {
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(th.ForegroundDim)).
		Padding(1, 3)

	textStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.Foreground))

	highlightStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.Highlight))

	body := textStyle.Render("Are you sure you want") + "\n" +
		textStyle.Render("to quit?") + "\n" +
		"\n" +
		highlightStyle.Render("y") + textStyle.Render(" - yes    ") +
		highlightStyle.Render("n") + textStyle.Render(" - cancel")

	modal := borderStyle.Render(body)

	return centerOverlay(modal, width, height)
}

// helpSection groups actions under a section header for the help modal.
type helpSection struct {
	Title   string
	Entries []helpEntry
}

// helpEntry is a single action+key row in the help modal.
type helpEntry struct {
	Label string // e.g. "Move down"
	Key   string // e.g. "j / ↓"
}

// actionDisplayName returns the human-readable label for an action.
func actionDisplayName(action string) string {
	switch action {
	case config.ActionMoveDown:
		return "Move down"
	case config.ActionMoveUp:
		return "Move up"
	case config.ActionJumpTop:
		return "Jump to top"
	case config.ActionJumpBottom:
		return "Jump to bottom"
	case config.ActionFocusToggle:
		return "Toggle pane"
	case config.ActionFocusLeft:
		return "Focus left"
	case config.ActionFocusRight:
		return "Focus right"
	case config.ActionExpand:
		return "Expand / collapse"
	case config.ActionToggleView:
		return "Toggle view mode"
	case config.ActionToggleLineNumbers:
		return "Toggle line numbers"
	case config.ActionToggleLeftPane:
		return "Toggle left pane"
	case config.ActionQuit:
		return "Quit"
	case config.ActionHelp:
		return "Help"
	case config.ActionRun:
		return "Run prompt"
	case config.ActionEscape:
		return "Escape"
	default:
		return action
	}
}

// arrowAlternate returns the arrow symbol alternate for letter-based nav keys.
func arrowAlternate(action string) string {
	switch action {
	case config.ActionMoveDown:
		return "↓"
	case config.ActionMoveUp:
		return "↑"
	case config.ActionFocusLeft:
		return "←"
	case config.ActionFocusRight:
		return "→"
	default:
		return ""
	}
}

// buildHelpSections constructs the sections for the help modal from the KeyMap.
func buildHelpSections(km *config.KeyMap) []helpSection {
	entryFor := func(action string) helpEntry {
		label := actionDisplayName(action)
		binding, ok := km.Bindings[action]
		keyStr := ""
		if ok {
			keyStr = binding.DisplayString()
		}
		if alt := arrowAlternate(action); alt != "" {
			keyStr = keyStr + " / " + alt
		}
		return helpEntry{Label: label, Key: keyStr}
	}

	return []helpSection{
		{
			Title: "Navigation",
			Entries: []helpEntry{
				entryFor(config.ActionMoveDown),
				entryFor(config.ActionMoveUp),
				entryFor(config.ActionJumpTop),
				entryFor(config.ActionJumpBottom),
			},
		},
		{
			Title: "Focus",
			Entries: []helpEntry{
				entryFor(config.ActionFocusToggle),
				entryFor(config.ActionFocusLeft),
				entryFor(config.ActionFocusRight),
			},
		},
		{
			Title: "Actions",
			Entries: []helpEntry{
				entryFor(config.ActionExpand),
				entryFor(config.ActionRun),
				{Label: "Edit plan file", Key: "e"},
				entryFor(config.ActionToggleView),
				entryFor(config.ActionToggleLineNumbers),
				entryFor(config.ActionToggleLeftPane),
			},
		},
		{
			Title: "Global",
			Entries: []helpEntry{
				entryFor(config.ActionQuit),
				{Label: "Force quit", Key: "ctrl+c ×2"},
				entryFor(config.ActionHelp),
			},
		},
	}
}

// RenderHelpModal renders a centered help overlay showing all keybindings.
func RenderHelpModal(width, height int, th theme.Theme, km *config.KeyMap) string {
	sections := buildHelpSections(km)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.Foreground)).
		Bold(true)

	sectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.Foreground)).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.Foreground))

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.Highlight))

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.ForegroundDim))

	// Determine column widths from content.
	maxLabel := 0
	maxKey := 0
	for _, sec := range sections {
		for _, e := range sec.Entries {
			if len(e.Label) > maxLabel {
				maxLabel = len(e.Label)
			}
			if len(e.Key) > maxKey {
				maxKey = len(e.Key)
			}
		}
	}

	// Inner content width: indent(4) + label + gap(4) + key
	innerWidth := 4 + maxLabel + 4 + maxKey
	if innerWidth < 20 {
		innerWidth = 20
	}

	// Build content lines.
	var lines []string
	for i, sec := range sections {
		if i > 0 {
			lines = append(lines, "") // blank line between sections
		}
		lines = append(lines, "  "+sectionStyle.Render(sec.Title))
		for _, e := range sec.Entries {
			gap := innerWidth - 4 - len(e.Label) - len(e.Key)
			if gap < 2 {
				gap = 2
			}
			line := "    " + labelStyle.Render(e.Label) + strings.Repeat(" ", gap) + keyStyle.Render(e.Key)
			lines = append(lines, line)
		}
	}
	lines = append(lines, "")
	footer := dimStyle.Render("Press any key to close")
	// Center footer within inner width.
	footerPad := (innerWidth - len("Press any key to close")) / 2
	if footerPad < 0 {
		footerPad = 0
	}
	lines = append(lines, strings.Repeat(" ", footerPad)+footer)

	body := strings.Join(lines, "\n")

	// Build the title bar centered in the border.
	title := titleStyle.Render(" Keybindings ")

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(th.ForegroundDim)).
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderRight(true).
		Padding(1, 2)

	modal := borderStyle.Render(body)

	// Inject title into the top border line.
	modalLines := strings.Split(modal, "\n")
	if len(modalLines) > 0 {
		topBorder := modalLines[0]
		borderWidth := lipgloss.Width(topBorder)
		titleWidth := lipgloss.Width(title)
		insertPos := (borderWidth - titleWidth) / 2
		if insertPos > 1 {
			// Replace characters in the top border with the title.
			modalLines[0] = replaceInLine(topBorder, title, insertPos)
		}
	}
	modal = strings.Join(modalLines, "\n")

	return centerOverlay(modal, width, height)
}

// replaceInLine replaces a portion of a rendered line with new content at a visual position.
func replaceInLine(line string, replacement string, pos int) string {
	// Work with runes to handle the border characters properly.
	runes := []rune(line)
	replRunes := []rune(replacement)
	if pos+len(replRunes) > len(runes) {
		return line
	}
	result := make([]rune, len(runes))
	copy(result, runes)
	copy(result[pos:], replRunes)
	return string(result)
}

// RenderRunModal renders a centered modal for entering the iteration count before starting a run.
// When selected is true, the value is rendered with a selection highlight (pre-filled state).
// When selected is false, a cursor block is shown after the value.
func RenderRunModal(width, height int, th theme.Theme, value string, selected bool) string {
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(th.ForegroundDim)).
		Padding(1, 3)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.Foreground))

	dimStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(th.ForegroundDim))

	// Build the input field display
	var inputDisplay string
	if selected {
		// Selected state: render value with highlight background (selection)
		selectedStyle := lipgloss.NewStyle().
			Background(lipgloss.Color(th.Highlight))
		inputDisplay = selectedStyle.Render(value)
	} else {
		// Normal state: render value followed by a cursor block
		valueStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(th.Foreground))
		cursorStyle := lipgloss.NewStyle().
			Background(lipgloss.Color(th.Foreground))
		inputDisplay = valueStyle.Render(value) + cursorStyle.Render(" ")
	}

	// "Iterations: [value]"
	line := labelStyle.Render("Iterations: ") + inputDisplay

	// Hints
	hint1 := dimStyle.Render("enter to start")
	hint2 := dimStyle.Render("esc to cancel")

	body := line + "\n" +
		"\n" +
		hint1 + "\n" +
		hint2

	modal := borderStyle.Render(body)

	return centerOverlay(modal, width, height)
}

// centerOverlay places a rendered block in the center of the terminal.
func centerOverlay(content string, termWidth, termHeight int) string {
	lines := strings.Split(content, "\n")
	contentHeight := len(lines)

	// Determine the max line width (visible width).
	contentWidth := 0
	for _, line := range lines {
		w := lipgloss.Width(line)
		if w > contentWidth {
			contentWidth = w
		}
	}

	// Vertical centering: pad above with empty lines.
	topPad := (termHeight - contentHeight) / 2
	if topPad < 0 {
		topPad = 0
	}

	// Horizontal centering: pad each line with spaces.
	leftPad := (termWidth - contentWidth) / 2
	if leftPad < 0 {
		leftPad = 0
	}

	var sb strings.Builder
	padding := strings.Repeat(" ", leftPad)

	for i := 0; i < topPad; i++ {
		sb.WriteString(strings.Repeat(" ", termWidth))
		sb.WriteByte('\n')
	}

	for i, line := range lines {
		sb.WriteString(padding)
		sb.WriteString(line)
		// Pad right to fill the full width.
		lineW := lipgloss.Width(line)
		if remaining := termWidth - leftPad - lineW; remaining > 0 {
			sb.WriteString(strings.Repeat(" ", remaining))
		}
		if i < len(lines)-1 {
			sb.WriteByte('\n')
		}
	}

	// Fill remaining rows below the modal.
	bottomRows := termHeight - topPad - contentHeight
	for i := 0; i < bottomRows; i++ {
		sb.WriteByte('\n')
		sb.WriteString(strings.Repeat(" ", termWidth))
	}

	return sb.String()
}
