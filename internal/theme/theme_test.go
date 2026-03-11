package theme

import (
	"sort"
	"testing"
)

func TestLookupTheme(t *testing.T) {
	t.Run("existing themes return valid theme", func(t *testing.T) {
		for _, name := range []string{"solarized-dark", "solarized-light", "monokai", "nord"} {
			th, ok := LookupTheme(name)
			if !ok {
				t.Errorf("LookupTheme(%q) returned ok=false, want true", name)
				continue
			}
			// Every theme must have non-empty core colors
			if th.Foreground == "" {
				t.Errorf("theme %q has empty Foreground", name)
			}
			if th.StatusRunning == "" {
				t.Errorf("theme %q has empty StatusRunning", name)
			}
			if th.StatusSuccess == "" {
				t.Errorf("theme %q has empty StatusSuccess", name)
			}
			if th.StatusError == "" {
				t.Errorf("theme %q has empty StatusError", name)
			}
			// Every theme must have non-empty diff colors
			diffFields := map[string]string{
				"DiffAdded":           th.DiffAdded,
				"DiffRemoved":         th.DiffRemoved,
				"DiffAddedBg":         th.DiffAddedBg,
				"DiffRemovedBg":       th.DiffRemovedBg,
				"DiffAddedEmphasis":   th.DiffAddedEmphasis,
				"DiffRemovedEmphasis": th.DiffRemovedEmphasis,
				"DiffLineNumber":      th.DiffLineNumber,
				"DiffSessionCommit":   th.DiffSessionCommit,
			}
			for field, value := range diffFields {
				if value == "" {
					t.Errorf("theme %q has empty %s", name, field)
				}
			}
		}
	})

	t.Run("missing theme returns false", func(t *testing.T) {
		_, ok := LookupTheme("nonexistent")
		if ok {
			t.Error("LookupTheme(nonexistent) returned ok=true, want false")
		}
	})

	t.Run("empty name returns false", func(t *testing.T) {
		_, ok := LookupTheme("")
		if ok {
			t.Error("LookupTheme('') returned ok=true, want false")
		}
	})
}

func TestThemeNames(t *testing.T) {
	names := ThemeNames()

	t.Run("contains all four themes", func(t *testing.T) {
		expected := []string{"monokai", "nord", "solarized-dark", "solarized-light"}
		if len(names) != len(expected) {
			t.Fatalf("ThemeNames() returned %d names, want %d: %v", len(names), len(expected), names)
		}
		for i, name := range expected {
			if names[i] != name {
				t.Errorf("ThemeNames()[%d] = %q, want %q", i, names[i], name)
			}
		}
	})

	t.Run("is sorted", func(t *testing.T) {
		if !sort.StringsAreSorted(names) {
			t.Errorf("ThemeNames() is not sorted: %v", names)
		}
	})

	t.Run("all names are valid lookups", func(t *testing.T) {
		for _, name := range names {
			if _, ok := LookupTheme(name); !ok {
				t.Errorf("ThemeNames() contains %q but LookupTheme returns false", name)
			}
		}
	})
}
