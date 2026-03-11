package tui

import (
	"testing"
)

func TestIntraLineChanges_IdenticalLines(t *testing.T) {
	old, new := IntraLineChanges("hello world", "hello world")
	if len(old) != 0 {
		t.Errorf("expected no old ranges for identical lines, got %v", old)
	}
	if len(new) != 0 {
		t.Errorf("expected no new ranges for identical lines, got %v", new)
	}
}

func TestIntraLineChanges_EmptyLines(t *testing.T) {
	old, new := IntraLineChanges("", "")
	if len(old) != 0 {
		t.Errorf("expected no old ranges for empty lines, got %v", old)
	}
	if len(new) != 0 {
		t.Errorf("expected no new ranges for empty lines, got %v", new)
	}
}

func TestIntraLineChanges_SingleWordChange(t *testing.T) {
	oldRanges, newRanges := IntraLineChanges("hello world", "hello earth")

	if len(oldRanges) == 0 {
		t.Fatal("expected old ranges for changed word")
	}
	if len(newRanges) == 0 {
		t.Fatal("expected new ranges for changed word")
	}

	// "world" changed to "earth" — the differing region should cover those characters
	// The exact ranges depend on the diff algorithm, but the changed portion
	// should be in the second half of the string
	oldChanged := extractChanged("hello world", oldRanges)
	newChanged := extractChanged("hello earth", newRanges)

	if oldChanged == "" {
		t.Error("expected non-empty old changed text")
	}
	if newChanged == "" {
		t.Error("expected non-empty new changed text")
	}

	// The changed text should contain the differing words
	if !containsSubstring(oldChanged, "world") {
		t.Errorf("expected old changed text to contain 'world', got %q", oldChanged)
	}
	if !containsSubstring(newChanged, "earth") {
		t.Errorf("expected new changed text to contain 'earth', got %q", newChanged)
	}
}

func TestIntraLineChanges_MultipleChanges(t *testing.T) {
	oldRanges, newRanges := IntraLineChanges("foo bar baz", "fox bat baz")

	if len(oldRanges) == 0 {
		t.Fatal("expected old ranges for multiple changes")
	}
	if len(newRanges) == 0 {
		t.Fatal("expected new ranges for multiple changes")
	}

	// "baz" is unchanged on both sides, so it shouldn't be in any range
	oldChanged := extractChanged("foo bar baz", oldRanges)
	newChanged := extractChanged("fox bat baz", newRanges)

	if containsSubstring(oldChanged, "baz") {
		t.Errorf("unchanged 'baz' should not be in old ranges, changed text: %q", oldChanged)
	}
	if containsSubstring(newChanged, "baz") {
		t.Errorf("unchanged 'baz' should not be in new ranges, changed text: %q", newChanged)
	}
}

func TestIntraLineChanges_EntireLineChanged(t *testing.T) {
	oldRanges, newRanges := IntraLineChanges("abcdef", "xyz")

	if len(oldRanges) == 0 {
		t.Fatal("expected old ranges when entire line changed")
	}
	if len(newRanges) == 0 {
		t.Fatal("expected new ranges when entire line changed")
	}

	// Most/all characters should be marked as changed
	oldTotal := totalChanged(oldRanges)
	newTotal := totalChanged(newRanges)

	if oldTotal < 4 {
		t.Errorf("expected most of old line to be marked changed, got %d chars", oldTotal)
	}
	if newTotal < 2 {
		t.Errorf("expected most of new line to be marked changed, got %d chars", newTotal)
	}
}

func TestIntraLineChanges_InsertionOnly(t *testing.T) {
	oldRanges, newRanges := IntraLineChanges("hello", "hello world")

	if len(oldRanges) != 0 {
		t.Errorf("expected no old ranges for pure insertion, got %v", oldRanges)
	}
	if len(newRanges) == 0 {
		t.Fatal("expected new ranges for insertion")
	}

	newChanged := extractChanged("hello world", newRanges)
	if !containsSubstring(newChanged, " world") {
		t.Errorf("expected inserted text to contain ' world', got %q", newChanged)
	}
}

func TestIntraLineChanges_DeletionOnly(t *testing.T) {
	oldRanges, newRanges := IntraLineChanges("hello world", "hello")

	if len(oldRanges) == 0 {
		t.Fatal("expected old ranges for deletion")
	}
	if len(newRanges) != 0 {
		t.Errorf("expected no new ranges for pure deletion, got %v", newRanges)
	}

	oldChanged := extractChanged("hello world", oldRanges)
	if !containsSubstring(oldChanged, " world") {
		t.Errorf("expected deleted text to contain ' world', got %q", oldChanged)
	}
}

func TestIntraLineChanges_RangesAreValid(t *testing.T) {
	// Verify that ranges are non-overlapping and within bounds
	cases := []struct {
		old string
		new string
	}{
		{"func foo() {", "func bar() {"},
		{"  x := 1", "  x := 2"},
		{"import \"fmt\"", "import \"os\""},
		{"", "new content"},
		{"old content", ""},
	}

	for _, tc := range cases {
		oldRanges, newRanges := IntraLineChanges(tc.old, tc.new)

		oldRunes := []rune(tc.old)
		newRunes := []rune(tc.new)

		for i, r := range oldRanges {
			if r.Start < 0 || r.End > len(oldRunes) || r.Start >= r.End {
				t.Errorf("invalid old range %v for line %q", r, tc.old)
			}
			if i > 0 && r.Start < oldRanges[i-1].End {
				t.Errorf("overlapping old ranges: %v and %v", oldRanges[i-1], r)
			}
		}
		for i, r := range newRanges {
			if r.Start < 0 || r.End > len(newRunes) || r.Start >= r.End {
				t.Errorf("invalid new range %v for line %q", r, tc.new)
			}
			if i > 0 && r.Start < newRanges[i-1].End {
				t.Errorf("overlapping new ranges: %v and %v", newRanges[i-1], r)
			}
		}
	}
}

// Helper: extract changed text from a string given character ranges
func extractChanged(s string, ranges []CharRange) string {
	runes := []rune(s)
	var result []rune
	for _, r := range ranges {
		end := r.End
		if end > len(runes) {
			end = len(runes)
		}
		result = append(result, runes[r.Start:end]...)
	}
	return string(result)
}

// Helper: check if s contains substr
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Helper: total number of changed characters across ranges
func totalChanged(ranges []CharRange) int {
	total := 0
	for _, r := range ranges {
		total += r.End - r.Start
	}
	return total
}
