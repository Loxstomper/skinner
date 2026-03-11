package tui

import (
	"strconv"
	"strings"
)

// DiffLineType indicates whether a diff line is context, added, or removed.
type DiffLineType int

const (
	DiffLineContext DiffLineType = iota
	DiffLineAdded
	DiffLineRemoved
)

// DiffLine represents a single line in a parsed diff hunk.
type DiffLine struct {
	Type    DiffLineType
	OldNum  int    // Line number in old file (0 for added lines)
	NewNum  int    // Line number in new file (0 for removed lines)
	Content string // Line content without the +/- prefix
}

// Hunk represents a parsed diff hunk with its header range and lines.
type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Lines    []DiffLine
}

// SideBySideLine represents a paired row for side-by-side diff rendering.
type SideBySideLine struct {
	Left  *DiffLine // nil means blank/padding on the left side
	Right *DiffLine // nil means blank/padding on the right side
}

// ParseUnifiedDiff parses a unified diff string into structured hunks.
// It handles the standard unified diff format with @@ headers and +/-/space prefixed lines.
func ParseUnifiedDiff(diff string) []Hunk {
	lines := strings.Split(diff, "\n")
	var hunks []Hunk
	var current *Hunk
	var oldNum, newNum int

	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			h := parseHunkHeader(line)
			if h != nil {
				hunks = append(hunks, *h)
				current = &hunks[len(hunks)-1]
				oldNum = current.OldStart
				newNum = current.NewStart
			}
			continue
		}

		if current == nil {
			continue
		}

		switch {
		case strings.HasPrefix(line, "-"):
			current.Lines = append(current.Lines, DiffLine{
				Type:    DiffLineRemoved,
				OldNum:  oldNum,
				Content: line[1:],
			})
			oldNum++
		case strings.HasPrefix(line, "+"):
			current.Lines = append(current.Lines, DiffLine{
				Type:    DiffLineAdded,
				NewNum:  newNum,
				Content: line[1:],
			})
			newNum++
		case strings.HasPrefix(line, " "):
			current.Lines = append(current.Lines, DiffLine{
				Type:    DiffLineContext,
				OldNum:  oldNum,
				NewNum:  newNum,
				Content: line[1:],
			})
			oldNum++
			newNum++
		case line == `\ No newline at end of file`:
			continue
		}
	}

	return hunks
}

// parseHunkHeader parses a @@ -old,count +new,count @@ line.
func parseHunkHeader(line string) *Hunk {
	// Find the range between @@ markers
	if !strings.HasPrefix(line, "@@") {
		return nil
	}
	end := strings.Index(line[2:], "@@")
	if end < 0 {
		return nil
	}
	rangeStr := strings.TrimSpace(line[2 : end+2])

	parts := strings.Fields(rangeStr)
	if len(parts) < 2 {
		return nil
	}

	oldStart, oldCount := parseRange(parts[0])
	newStart, newCount := parseRange(parts[1])

	return &Hunk{
		OldStart: oldStart,
		OldCount: oldCount,
		NewStart: newStart,
		NewCount: newCount,
	}
}

// parseRange parses "-start,count" or "+start,count" returning start and count.
func parseRange(s string) (int, int) {
	// Strip leading -/+
	if len(s) > 0 && (s[0] == '-' || s[0] == '+') {
		s = s[1:]
	}
	parts := strings.SplitN(s, ",", 2)
	start, _ := strconv.Atoi(parts[0])
	count := 1
	if len(parts) == 2 {
		count, _ = strconv.Atoi(parts[1])
	}
	return start, count
}

// PairLines converts hunk lines into side-by-side paired rows.
// Context lines appear on both sides. Removed+added blocks are paired row-by-row,
// with the shorter side padded with nil entries for blank rows.
func PairLines(hunks []Hunk) []SideBySideLine {
	var result []SideBySideLine

	for _, hunk := range hunks {
		var removed []DiffLine
		var added []DiffLine

		flushBlock := func() {
			maxLen := len(removed)
			if len(added) > maxLen {
				maxLen = len(added)
			}
			for i := 0; i < maxLen; i++ {
				pair := SideBySideLine{}
				if i < len(removed) {
					r := removed[i]
					pair.Left = &r
				}
				if i < len(added) {
					a := added[i]
					pair.Right = &a
				}
				result = append(result, pair)
			}
			removed = nil
			added = nil
		}

		for _, line := range hunk.Lines {
			switch line.Type {
			case DiffLineContext:
				flushBlock()
				l := line
				result = append(result, SideBySideLine{Left: &l, Right: &l})
			case DiffLineRemoved:
				removed = append(removed, line)
			case DiffLineAdded:
				added = append(added, line)
			}
		}
		flushBlock()
	}

	return result
}
