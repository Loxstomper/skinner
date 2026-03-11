package tui

import (
	"github.com/sergi/go-diff/diffmatchpatch"
)

// CharRange represents a range of characters [Start, End) within a line.
type CharRange struct {
	Start int
	End   int
}

// IntraLineChanges computes character-level differences between two lines
// using the Myers diff algorithm. It returns ranges of changed characters
// for the old (removed) line and the new (added) line respectively.
// Identical lines return empty slices for both sides.
func IntraLineChanges(oldLine, newLine string) ([]CharRange, []CharRange) {
	if oldLine == newLine {
		return nil, nil
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldLine, newLine, false)
	diffs = dmp.DiffCleanupSemantic(diffs)

	var oldRanges, newRanges []CharRange
	var oldPos, newPos int

	for _, d := range diffs {
		runeLen := len([]rune(d.Text))
		switch d.Type {
		case diffmatchpatch.DiffEqual:
			oldPos += runeLen
			newPos += runeLen
		case diffmatchpatch.DiffDelete:
			oldRanges = append(oldRanges, CharRange{Start: oldPos, End: oldPos + runeLen})
			oldPos += runeLen
		case diffmatchpatch.DiffInsert:
			newRanges = append(newRanges, CharRange{Start: newPos, End: newPos + runeLen})
			newPos += runeLen
		}
	}

	return oldRanges, newRanges
}
