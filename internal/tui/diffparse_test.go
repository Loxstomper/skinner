package tui

import (
	"strings"
	"testing"
)

func TestParseUnifiedDiff_SingleHunk(t *testing.T) {
	diff := strings.Join([]string{
		"diff --git a/main.go b/main.go",
		"index abc1234..def5678 100644",
		"--- a/main.go",
		"+++ b/main.go",
		"@@ -10,4 +10,4 @@ func main() {",
		" func parse(s string) {",
		"-\ttab := strings.Split(s)",
		"+\ttab := strings.Fields(s)",
		" \tfor _, v := range tab {",
	}, "\n")

	hunks := ParseUnifiedDiff(diff)
	if len(hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(hunks))
	}

	h := hunks[0]
	if h.OldStart != 10 || h.OldCount != 4 || h.NewStart != 10 || h.NewCount != 4 {
		t.Errorf("wrong hunk range: old=%d,%d new=%d,%d", h.OldStart, h.OldCount, h.NewStart, h.NewCount)
	}
	if len(h.Lines) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(h.Lines))
	}

	// Context line
	if h.Lines[0].Type != DiffLineContext || h.Lines[0].OldNum != 10 || h.Lines[0].NewNum != 10 {
		t.Errorf("line 0: expected context at 10/10, got type=%d old=%d new=%d",
			h.Lines[0].Type, h.Lines[0].OldNum, h.Lines[0].NewNum)
	}
	if h.Lines[0].Content != "func parse(s string) {" {
		t.Errorf("line 0 content: %q", h.Lines[0].Content)
	}

	// Removed line
	if h.Lines[1].Type != DiffLineRemoved || h.Lines[1].OldNum != 11 {
		t.Errorf("line 1: expected removed at old=11, got type=%d old=%d", h.Lines[1].Type, h.Lines[1].OldNum)
	}
	if h.Lines[1].Content != "\ttab := strings.Split(s)" {
		t.Errorf("line 1 content: %q", h.Lines[1].Content)
	}

	// Added line
	if h.Lines[2].Type != DiffLineAdded || h.Lines[2].NewNum != 11 {
		t.Errorf("line 2: expected added at new=11, got type=%d new=%d", h.Lines[2].Type, h.Lines[2].NewNum)
	}
	if h.Lines[2].Content != "\ttab := strings.Fields(s)" {
		t.Errorf("line 2 content: %q", h.Lines[2].Content)
	}

	// Trailing context
	if h.Lines[3].Type != DiffLineContext || h.Lines[3].OldNum != 12 || h.Lines[3].NewNum != 12 {
		t.Errorf("line 3: expected context at 12/12, got type=%d old=%d new=%d",
			h.Lines[3].Type, h.Lines[3].OldNum, h.Lines[3].NewNum)
	}
}

func TestParseUnifiedDiff_MultipleHunks(t *testing.T) {
	diff := strings.Join([]string{
		"@@ -1,3 +1,3 @@",
		" line1",
		"-old",
		"+new",
		" line3",
		"@@ -20,2 +20,3 @@",
		" existing",
		"+inserted",
		" end",
	}, "\n")

	hunks := ParseUnifiedDiff(diff)
	if len(hunks) != 2 {
		t.Fatalf("expected 2 hunks, got %d", len(hunks))
	}
	if hunks[0].OldStart != 1 || hunks[1].OldStart != 20 {
		t.Errorf("wrong hunk starts: %d, %d", hunks[0].OldStart, hunks[1].OldStart)
	}
	if len(hunks[0].Lines) != 4 {
		t.Errorf("hunk 0: expected 4 lines, got %d", len(hunks[0].Lines))
	}
	if len(hunks[1].Lines) != 3 {
		t.Errorf("hunk 1: expected 3 lines, got %d", len(hunks[1].Lines))
	}
}

func TestParseUnifiedDiff_AddOnly(t *testing.T) {
	diff := strings.Join([]string{
		"@@ -5,2 +5,5 @@",
		" before",
		"+new1",
		"+new2",
		"+new3",
		" after",
	}, "\n")

	hunks := ParseUnifiedDiff(diff)
	if len(hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(hunks))
	}

	// Count line types
	var ctx, add, rem int
	for _, l := range hunks[0].Lines {
		switch l.Type {
		case DiffLineContext:
			ctx++
		case DiffLineAdded:
			add++
		case DiffLineRemoved:
			rem++
		}
	}
	if ctx != 2 || add != 3 || rem != 0 {
		t.Errorf("expected 2 context, 3 added, 0 removed; got %d, %d, %d", ctx, add, rem)
	}
}

func TestParseUnifiedDiff_RemoveOnly(t *testing.T) {
	diff := strings.Join([]string{
		"@@ -5,4 +5,2 @@",
		" before",
		"-old1",
		"-old2",
		" after",
	}, "\n")

	hunks := ParseUnifiedDiff(diff)
	if len(hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(hunks))
	}

	var rem int
	for _, l := range hunks[0].Lines {
		if l.Type == DiffLineRemoved {
			rem++
		}
	}
	if rem != 2 {
		t.Errorf("expected 2 removed, got %d", rem)
	}
}

func TestParseUnifiedDiff_NoNewlineMarker(t *testing.T) {
	diff := strings.Join([]string{
		"@@ -1,2 +1,2 @@",
		"-old",
		`\ No newline at end of file`,
		"+new",
		`\ No newline at end of file`,
	}, "\n")

	hunks := ParseUnifiedDiff(diff)
	if len(hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(hunks))
	}
	if len(hunks[0].Lines) != 2 {
		t.Errorf("expected 2 lines (no newline markers excluded), got %d", len(hunks[0].Lines))
	}
}

func TestParseUnifiedDiff_EmptyDiff(t *testing.T) {
	hunks := ParseUnifiedDiff("")
	if len(hunks) != 0 {
		t.Errorf("expected 0 hunks for empty diff, got %d", len(hunks))
	}
}

func TestParseUnifiedDiff_HunkHeaderNoCount(t *testing.T) {
	// When count is 1, git omits the ",count" part
	diff := strings.Join([]string{
		"@@ -1 +1 @@",
		"-old",
		"+new",
	}, "\n")

	hunks := ParseUnifiedDiff(diff)
	if len(hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(hunks))
	}
	if hunks[0].OldCount != 1 || hunks[0].NewCount != 1 {
		t.Errorf("expected count 1,1 got %d,%d", hunks[0].OldCount, hunks[0].NewCount)
	}
}

func TestPairLines_ContextOnly(t *testing.T) {
	hunks := []Hunk{{
		OldStart: 1, OldCount: 3, NewStart: 1, NewCount: 3,
		Lines: []DiffLine{
			{Type: DiffLineContext, OldNum: 1, NewNum: 1, Content: "a"},
			{Type: DiffLineContext, OldNum: 2, NewNum: 2, Content: "b"},
			{Type: DiffLineContext, OldNum: 3, NewNum: 3, Content: "c"},
		},
	}}

	pairs := PairLines(hunks)
	if len(pairs) != 3 {
		t.Fatalf("expected 3 pairs, got %d", len(pairs))
	}
	for i, p := range pairs {
		if p.Left == nil || p.Right == nil {
			t.Errorf("pair %d: expected both sides non-nil", i)
		}
	}
}

func TestPairLines_EqualBlock(t *testing.T) {
	hunks := []Hunk{{
		OldStart: 1, OldCount: 3, NewStart: 1, NewCount: 3,
		Lines: []DiffLine{
			{Type: DiffLineContext, OldNum: 1, NewNum: 1, Content: "before"},
			{Type: DiffLineRemoved, OldNum: 2, Content: "old"},
			{Type: DiffLineAdded, NewNum: 2, Content: "new"},
			{Type: DiffLineContext, OldNum: 3, NewNum: 3, Content: "after"},
		},
	}}

	pairs := PairLines(hunks)
	if len(pairs) != 3 {
		t.Fatalf("expected 3 pairs, got %d", len(pairs))
	}

	// Context
	if pairs[0].Left.Content != "before" || pairs[0].Right.Content != "before" {
		t.Error("pair 0: expected context 'before' on both sides")
	}

	// Changed pair
	if pairs[1].Left == nil || pairs[1].Left.Content != "old" {
		t.Error("pair 1: expected left='old'")
	}
	if pairs[1].Right == nil || pairs[1].Right.Content != "new" {
		t.Error("pair 1: expected right='new'")
	}

	// Context
	if pairs[2].Left.Content != "after" || pairs[2].Right.Content != "after" {
		t.Error("pair 2: expected context 'after' on both sides")
	}
}

func TestPairLines_UnequalBlock_MoreAdded(t *testing.T) {
	hunks := []Hunk{{
		OldStart: 1, OldCount: 2, NewStart: 1, NewCount: 4,
		Lines: []DiffLine{
			{Type: DiffLineRemoved, OldNum: 1, Content: "old"},
			{Type: DiffLineAdded, NewNum: 1, Content: "new1"},
			{Type: DiffLineAdded, NewNum: 2, Content: "new2"},
			{Type: DiffLineAdded, NewNum: 3, Content: "new3"},
			{Type: DiffLineContext, OldNum: 2, NewNum: 4, Content: "end"},
		},
	}}

	pairs := PairLines(hunks)
	if len(pairs) != 4 { // 3 change rows + 1 context
		t.Fatalf("expected 4 pairs, got %d", len(pairs))
	}

	// First change row: both sides present
	if pairs[0].Left == nil || pairs[0].Left.Content != "old" {
		t.Error("pair 0: expected left='old'")
	}
	if pairs[0].Right == nil || pairs[0].Right.Content != "new1" {
		t.Error("pair 0: expected right='new1'")
	}

	// Second and third rows: left is nil (padding)
	if pairs[1].Left != nil {
		t.Error("pair 1: expected left=nil (padding)")
	}
	if pairs[1].Right == nil || pairs[1].Right.Content != "new2" {
		t.Error("pair 1: expected right='new2'")
	}

	if pairs[2].Left != nil {
		t.Error("pair 2: expected left=nil (padding)")
	}
	if pairs[2].Right == nil || pairs[2].Right.Content != "new3" {
		t.Error("pair 2: expected right='new3'")
	}
}

func TestPairLines_UnequalBlock_MoreRemoved(t *testing.T) {
	hunks := []Hunk{{
		OldStart: 1, OldCount: 4, NewStart: 1, NewCount: 2,
		Lines: []DiffLine{
			{Type: DiffLineRemoved, OldNum: 1, Content: "old1"},
			{Type: DiffLineRemoved, OldNum: 2, Content: "old2"},
			{Type: DiffLineRemoved, OldNum: 3, Content: "old3"},
			{Type: DiffLineAdded, NewNum: 1, Content: "new"},
			{Type: DiffLineContext, OldNum: 4, NewNum: 2, Content: "end"},
		},
	}}

	pairs := PairLines(hunks)
	if len(pairs) != 4 { // 3 change rows + 1 context
		t.Fatalf("expected 4 pairs, got %d", len(pairs))
	}

	// First row: both sides
	if pairs[0].Left == nil || pairs[0].Left.Content != "old1" {
		t.Error("pair 0: expected left='old1'")
	}
	if pairs[0].Right == nil || pairs[0].Right.Content != "new" {
		t.Error("pair 0: expected right='new'")
	}

	// Remaining rows: right is nil (padding)
	if pairs[1].Right != nil {
		t.Error("pair 1: expected right=nil (padding)")
	}
	if pairs[1].Left == nil || pairs[1].Left.Content != "old2" {
		t.Error("pair 1: expected left='old2'")
	}

	if pairs[2].Right != nil {
		t.Error("pair 2: expected right=nil (padding)")
	}
	if pairs[2].Left == nil || pairs[2].Left.Content != "old3" {
		t.Error("pair 2: expected left='old3'")
	}
}

func TestPairLines_AddOnlyBlock(t *testing.T) {
	hunks := []Hunk{{
		OldStart: 1, OldCount: 2, NewStart: 1, NewCount: 4,
		Lines: []DiffLine{
			{Type: DiffLineContext, OldNum: 1, NewNum: 1, Content: "before"},
			{Type: DiffLineAdded, NewNum: 2, Content: "ins1"},
			{Type: DiffLineAdded, NewNum: 3, Content: "ins2"},
			{Type: DiffLineContext, OldNum: 2, NewNum: 4, Content: "after"},
		},
	}}

	pairs := PairLines(hunks)
	if len(pairs) != 4 {
		t.Fatalf("expected 4 pairs, got %d", len(pairs))
	}

	// Add-only rows: left is nil
	if pairs[1].Left != nil {
		t.Error("pair 1: expected left=nil for add-only")
	}
	if pairs[1].Right == nil || pairs[1].Right.Content != "ins1" {
		t.Error("pair 1: expected right='ins1'")
	}
}

func TestPairLines_RemoveOnlyBlock(t *testing.T) {
	hunks := []Hunk{{
		OldStart: 1, OldCount: 4, NewStart: 1, NewCount: 2,
		Lines: []DiffLine{
			{Type: DiffLineContext, OldNum: 1, NewNum: 1, Content: "before"},
			{Type: DiffLineRemoved, OldNum: 2, Content: "del1"},
			{Type: DiffLineRemoved, OldNum: 3, Content: "del2"},
			{Type: DiffLineContext, OldNum: 4, NewNum: 2, Content: "after"},
		},
	}}

	pairs := PairLines(hunks)
	if len(pairs) != 4 {
		t.Fatalf("expected 4 pairs, got %d", len(pairs))
	}

	// Remove-only rows: right is nil
	if pairs[1].Right != nil {
		t.Error("pair 1: expected right=nil for remove-only")
	}
	if pairs[1].Left == nil || pairs[1].Left.Content != "del1" {
		t.Error("pair 1: expected left='del1'")
	}
}

func TestParseUnifiedDiff_LineNumberProgression(t *testing.T) {
	diff := strings.Join([]string{
		"@@ -5,5 +5,6 @@",
		" context1",
		"-removed1",
		"-removed2",
		"+added1",
		"+added2",
		"+added3",
		" context2",
	}, "\n")

	hunks := ParseUnifiedDiff(diff)
	if len(hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(hunks))
	}

	lines := hunks[0].Lines
	// context1: old=5, new=5
	if lines[0].OldNum != 5 || lines[0].NewNum != 5 {
		t.Errorf("context1: expected 5/5, got %d/%d", lines[0].OldNum, lines[0].NewNum)
	}
	// removed1: old=6
	if lines[1].OldNum != 6 {
		t.Errorf("removed1: expected old=6, got %d", lines[1].OldNum)
	}
	// removed2: old=7
	if lines[2].OldNum != 7 {
		t.Errorf("removed2: expected old=7, got %d", lines[2].OldNum)
	}
	// added1: new=6
	if lines[3].NewNum != 6 {
		t.Errorf("added1: expected new=6, got %d", lines[3].NewNum)
	}
	// added2: new=7
	if lines[4].NewNum != 7 {
		t.Errorf("added2: expected new=7, got %d", lines[4].NewNum)
	}
	// added3: new=8
	if lines[5].NewNum != 8 {
		t.Errorf("added3: expected new=8, got %d", lines[5].NewNum)
	}
	// context2: old=8, new=9
	if lines[6].OldNum != 8 || lines[6].NewNum != 9 {
		t.Errorf("context2: expected 8/9, got %d/%d", lines[6].OldNum, lines[6].NewNum)
	}
}

func TestPairLines_MultipleHunks(t *testing.T) {
	hunks := []Hunk{
		{
			OldStart: 1, OldCount: 2, NewStart: 1, NewCount: 2,
			Lines: []DiffLine{
				{Type: DiffLineRemoved, OldNum: 1, Content: "old1"},
				{Type: DiffLineAdded, NewNum: 1, Content: "new1"},
				{Type: DiffLineContext, OldNum: 2, NewNum: 2, Content: "ctx"},
			},
		},
		{
			OldStart: 10, OldCount: 1, NewStart: 10, NewCount: 1,
			Lines: []DiffLine{
				{Type: DiffLineRemoved, OldNum: 10, Content: "old10"},
				{Type: DiffLineAdded, NewNum: 10, Content: "new10"},
			},
		},
	}

	pairs := PairLines(hunks)
	if len(pairs) != 3 { // hunk1: 1 change pair + 1 context = 2; hunk2: 1 change pair = 1; total = 3
		t.Fatalf("expected 3 pairs, got %d", len(pairs))
	}
}
