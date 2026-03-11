package tui

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/loxstomper/skinner/internal/theme"
)

// DiffViewProps contains the data needed to render a diff view.
type DiffViewProps struct {
	Hunks     []Hunk
	FilePath  string // for language detection via chroma
	Theme     theme.Theme
	ThemeName string // for chroma style selection
	Width     int    // pane width
	HScroll   int    // horizontal scroll offset for code content
}

// emphasisInfo holds pre-computed intra-line emphasis ranges for a paired row.
type emphasisInfo struct {
	oldRanges []CharRange
	newRanges []CharRange
}

// tokenSpan is a span of text with an optional foreground color from chroma.
type tokenSpan struct {
	Text string
	Fg   string // hex color, empty for default
}

// RenderDiff renders parsed hunks as styled output.
// Side-by-side when width >= 80, unified when < 80.
func RenderDiff(props DiffViewProps) string {
	if len(props.Hunks) == 0 {
		return ""
	}

	pairs := PairLines(props.Hunks)
	if len(pairs) == 0 {
		return ""
	}

	// Pre-compute intra-line emphasis for paired changed lines.
	emphasis := make([]emphasisInfo, len(pairs))
	for i, pair := range pairs {
		if pair.Left != nil && pair.Right != nil &&
			pair.Left.Type == DiffLineRemoved && pair.Right.Type == DiffLineAdded {
			oldR, newR := IntraLineChanges(pair.Left.Content, pair.Right.Content)
			emphasis[i] = emphasisInfo{oldRanges: oldR, newRanges: newR}
		}
	}

	// Create chroma lexer and style once for the whole render.
	lexer := getLexer(props.FilePath)
	style := getChromaStyle(props.ThemeName)

	if props.Width >= 80 {
		return renderSideBySide(props, pairs, emphasis, lexer, style)
	}
	return renderUnified(props, pairs, emphasis, lexer, style)
}

// renderSideBySide renders paired lines in a two-column layout.
func renderSideBySide(props DiffViewProps, pairs []SideBySideLine, emphasis []emphasisInfo,
	lexer chroma.Lexer, style *chroma.Style) string {

	maxLineNum := maxLineNumber(pairs)
	numWidth := digitCount(maxLineNum)
	if numWidth < 3 {
		numWidth = 3
	}
	// Gutter: right-aligned number + " │ "
	gutterWidth := numWidth + 3
	// Column separator: "│" (1 char)
	sepWidth := 1
	halfWidth := (props.Width - sepWidth) / 2
	rightHalf := props.Width - sepWidth - halfWidth
	leftContent := halfWidth - gutterWidth
	rightContent := rightHalf - gutterWidth
	if leftContent < 1 {
		leftContent = 1
	}
	if rightContent < 1 {
		rightContent = 1
	}

	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(props.Theme.DiffLineNumber))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(props.Theme.ForegroundDim))
	sep := sepStyle.Render("│")

	var lines []string
	for i, pair := range pairs {
		var leftGutter, rightGutter string
		var leftRendered, rightRendered string

		// Left side
		if pair.Left != nil {
			num := pair.Left.OldNum
			if pair.Left.Type == DiffLineAdded {
				num = pair.Left.NewNum
			}
			leftGutter = lineNumStyle.Render(fmt.Sprintf("%*d", numWidth, num)) + sepStyle.Render(" │ ")
			leftRendered = renderStyledLine(pair.Left.Content, pair.Left.Type,
				emphasis[i].oldRanges, props.Theme, leftContent, props.HScroll, lexer, style)
		} else {
			leftGutter = strings.Repeat(" ", gutterWidth)
			leftRendered = strings.Repeat(" ", leftContent)
		}

		// Right side
		if pair.Right != nil {
			num := pair.Right.NewNum
			if pair.Right.Type == DiffLineRemoved {
				num = pair.Right.OldNum
			}
			rightGutter = lineNumStyle.Render(fmt.Sprintf("%*d", numWidth, num)) + sepStyle.Render(" │ ")
			rightRendered = renderStyledLine(pair.Right.Content, pair.Right.Type,
				emphasis[i].newRanges, props.Theme, rightContent, props.HScroll, lexer, style)
		} else {
			rightGutter = strings.Repeat(" ", gutterWidth)
			rightRendered = strings.Repeat(" ", rightContent)
		}

		lines = append(lines, leftGutter+leftRendered+sep+rightGutter+rightRendered)
	}

	return strings.Join(lines, "\n")
}

// renderUnified renders paired lines in a single-column unified diff format.
func renderUnified(props DiffViewProps, pairs []SideBySideLine, emphasis []emphasisInfo,
	lexer chroma.Lexer, style *chroma.Style) string {

	maxLineNum := maxLineNumber(pairs)
	numWidth := digitCount(maxLineNum)
	if numWidth < 3 {
		numWidth = 3
	}
	// Gutter: prefix (1) + space + right-aligned number + " │ "
	// e.g. "- 13 │ " or "+ 14 │ " or "  12 │ "
	gutterWidth := 1 + 1 + numWidth + 3
	contentWidth := props.Width - gutterWidth
	if contentWidth < 1 {
		contentWidth = 1
	}

	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(props.Theme.DiffLineNumber))
	sepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(props.Theme.ForegroundDim))
	addedPrefixStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(props.Theme.DiffAdded))
	removedPrefixStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(props.Theme.DiffRemoved))

	var lines []string
	for i, pair := range pairs {
		// Unified rendering: emit left (removed) lines first, then right (added) lines.
		// Context lines appear once (left == right pointer in PairLines).
		if pair.Left != nil && pair.Right != nil && pair.Left.Type == DiffLineContext {
			// Context line
			gutter := " " + " " + lineNumStyle.Render(fmt.Sprintf("%*d", numWidth, pair.Left.OldNum)) + sepStyle.Render(" │ ")
			content := renderStyledLine(pair.Left.Content, DiffLineContext,
				nil, props.Theme, contentWidth, props.HScroll, lexer, style)
			lines = append(lines, gutter+content)
			continue
		}

		// Removed line (left side)
		if pair.Left != nil {
			num := pair.Left.OldNum
			gutter := removedPrefixStyle.Render("-") + " " + lineNumStyle.Render(fmt.Sprintf("%*d", numWidth, num)) + sepStyle.Render(" │ ")
			content := renderStyledLine(pair.Left.Content, DiffLineRemoved,
				emphasis[i].oldRanges, props.Theme, contentWidth, props.HScroll, lexer, style)
			lines = append(lines, gutter+content)
		}

		// Added line (right side)
		if pair.Right != nil {
			num := pair.Right.NewNum
			gutter := addedPrefixStyle.Render("+") + " " + lineNumStyle.Render(fmt.Sprintf("%*d", numWidth, num)) + sepStyle.Render(" │ ")
			content := renderStyledLine(pair.Right.Content, DiffLineAdded,
				emphasis[i].newRanges, props.Theme, contentWidth, props.HScroll, lexer, style)
			lines = append(lines, gutter+content)
		}
	}

	return strings.Join(lines, "\n")
}

// renderStyledLine renders a single line of diff content with syntax highlighting,
// diff background, and intra-line emphasis applied.
func renderStyledLine(content string, lineType DiffLineType, emphRanges []CharRange,
	th theme.Theme, width, hscroll int, lexer chroma.Lexer, chromaStyle *chroma.Style) string {

	// Determine background colors.
	baseBg := ""
	emphBg := ""
	switch lineType {
	case DiffLineAdded:
		baseBg = th.DiffAddedBg
		emphBg = th.DiffAddedEmphasis
	case DiffLineRemoved:
		baseBg = th.DiffRemovedBg
		emphBg = th.DiffRemovedEmphasis
	}

	// Tokenize with chroma for syntax foreground colors.
	tokens := tokenizeLine(content, lexer, chromaStyle)

	// Build character-level styling.
	runes := []rune(content)
	type charStyle struct {
		r  rune
		fg string
		bg string
	}
	chars := make([]charStyle, len(runes))

	// Apply chroma foreground colors.
	pos := 0
	for _, tok := range tokens {
		for _, r := range tok.Text {
			if pos < len(chars) {
				chars[pos] = charStyle{r: r, fg: tok.Fg, bg: baseBg}
				pos++
			}
		}
	}
	for i := pos; i < len(chars); i++ {
		chars[i] = charStyle{r: runes[i], bg: baseBg}
	}

	// Overlay emphasis backgrounds on changed character ranges.
	if emphBg != "" {
		for _, rng := range emphRanges {
			for j := rng.Start; j < rng.End && j < len(chars); j++ {
				chars[j].bg = emphBg
			}
		}
	}

	// Apply horizontal scroll.
	if hscroll > 0 {
		if hscroll >= len(chars) {
			chars = nil
		} else {
			chars = chars[hscroll:]
		}
	}

	// Truncate to visible width.
	if len(chars) > width {
		chars = chars[:width]
	}

	// Group consecutive characters with the same style and render.
	var b strings.Builder
	for i := 0; i < len(chars); {
		j := i + 1
		for j < len(chars) && chars[j].fg == chars[i].fg && chars[j].bg == chars[i].bg {
			j++
		}
		var text strings.Builder
		for k := i; k < j; k++ {
			text.WriteRune(chars[k].r)
		}
		s := lipgloss.NewStyle()
		if chars[i].fg != "" {
			s = s.Foreground(lipgloss.Color(chars[i].fg))
		}
		if chars[i].bg != "" {
			s = s.Background(lipgloss.Color(chars[i].bg))
		}
		b.WriteString(s.Render(text.String()))
		i = j
	}

	// Pad remaining width with base background.
	rendered := len(chars)
	if rendered < width {
		padding := strings.Repeat(" ", width-rendered)
		s := lipgloss.NewStyle()
		if baseBg != "" {
			s = s.Background(lipgloss.Color(baseBg))
		}
		b.WriteString(s.Render(padding))
	}

	return b.String()
}

// tokenizeLine tokenizes a single line using chroma and returns styled spans.
func tokenizeLine(content string, lexer chroma.Lexer, style *chroma.Style) []tokenSpan {
	if lexer == nil || style == nil {
		return []tokenSpan{{Text: content}}
	}

	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return []tokenSpan{{Text: content}}
	}

	var spans []tokenSpan
	for _, token := range iterator.Tokens() {
		// Skip final newline tokens that chroma may add.
		if token.Value == "\n" || token.Value == "" {
			continue
		}
		entry := style.Get(token.Type)
		fg := ""
		if entry.Colour.IsSet() {
			fg = entry.Colour.String()
		}
		spans = append(spans, tokenSpan{Text: token.Value, Fg: fg})
	}
	return spans
}

// getLexer returns a chroma lexer matched to the file path, or nil if none found.
func getLexer(filePath string) chroma.Lexer {
	if filePath == "" {
		return nil
	}
	lexer := lexers.Match(filePath)
	if lexer == nil {
		return nil
	}
	return chroma.Coalesce(lexer)
}

// getChromaStyle returns the chroma style for the given theme name.
func getChromaStyle(themeName string) *chroma.Style {
	name := chromaStyleName(themeName)
	s := styles.Get(name)
	return s
}

// chromaStyleName maps a skinner theme name to a chroma style name.
func chromaStyleName(themeName string) string {
	switch themeName {
	case "solarized-dark":
		return "solarized-dark"
	case "solarized-light":
		return "solarized-light"
	case "monokai":
		return "monokai"
	case "nord":
		return "nord"
	default:
		return "monokai"
	}
}

// maxLineNumber returns the highest line number across all paired rows.
func maxLineNumber(pairs []SideBySideLine) int {
	max := 0
	for _, p := range pairs {
		if p.Left != nil {
			if p.Left.OldNum > max {
				max = p.Left.OldNum
			}
			if p.Left.NewNum > max {
				max = p.Left.NewNum
			}
		}
		if p.Right != nil {
			if p.Right.OldNum > max {
				max = p.Right.OldNum
			}
			if p.Right.NewNum > max {
				max = p.Right.NewNum
			}
		}
	}
	return max
}

// digitCount returns the number of digits needed to display n.
func digitCount(n int) int {
	if n <= 0 {
		return 1
	}
	count := 0
	for n > 0 {
		n /= 10
		count++
	}
	return count
}
