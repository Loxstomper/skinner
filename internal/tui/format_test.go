package tui

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name    string
		d       time.Duration
		running bool
		want    string
	}{
		{"running returns ellipsis", 5 * time.Second, true, "..."},
		{"zero duration running", 0, true, "..."},
		{"completed short", 3500 * time.Millisecond, false, "3.5s"},
		{"completed exact minute", 60 * time.Second, false, "1m00s"},
		{"completed over minute", 90 * time.Second, false, "1m30s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDuration(tt.d, tt.running)
			if got != tt.want {
				t.Errorf("FormatDuration(%v, %v) = %q, want %q", tt.d, tt.running, got, tt.want)
			}
		})
	}
}

func TestFormatDurationValue(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"zero", 0, "0.0s"},
		{"sub-second", 500 * time.Millisecond, "0.5s"},
		{"few seconds", 12300 * time.Millisecond, "12.3s"},
		{"just under a minute", 59900 * time.Millisecond, "59.9s"},
		{"exactly one minute", 60 * time.Second, "1m00s"},
		{"one minute thirty", 90 * time.Second, "1m30s"},
		{"five minutes five seconds", 305 * time.Second, "5m05s"},
		{"over ten minutes", 625 * time.Second, "10m25s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDurationValue(tt.d)
			if got != tt.want {
				t.Errorf("FormatDurationValue(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		name   string
		tokens int64
		want   string
	}{
		{"zero", 0, "0"},
		{"small", 500, "500"},
		{"just under 1k", 999, "999"},
		{"exactly 1k", 1000, "1.0k"},
		{"1.5k", 1500, "1.5k"},
		{"large", 12345, "12.3k"},
		{"very large", 150000, "150.0k"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTokens(tt.tokens)
			if got != tt.want {
				t.Errorf("FormatTokens(%d) = %q, want %q", tt.tokens, got, tt.want)
			}
		})
	}
}

func TestToolIcon(t *testing.T) {
	knownTools := []struct {
		name string
		icon string
	}{
		{"Read", "\uf02d"},
		{"Edit", "\uf044"},
		{"Write", "\uf0c7"},
		{"Bash", "\uf120"},
		{"Grep", "\uf002"},
		{"Glob", "\uf07b"},
		{"Task", "\uf085"},
	}
	for _, tt := range knownTools {
		t.Run(tt.name, func(t *testing.T) {
			got := ToolIcon(tt.name)
			if got != tt.icon {
				t.Errorf("ToolIcon(%q) = %q, want %q", tt.name, got, tt.icon)
			}
		})
	}

	t.Run("unknown tool gets fallback icon", func(t *testing.T) {
		got := ToolIcon("UnknownTool")
		want := "\uf059"
		if got != want {
			t.Errorf("ToolIcon(\"UnknownTool\") = %q, want %q", got, want)
		}
	})
}

func TestGroupSummaryUnit(t *testing.T) {
	tests := []struct {
		toolName string
		want     string
	}{
		{"Read", "files"},
		{"Write", "files"},
		{"Edit", "edits"},
		{"Bash", "commands"},
		{"Grep", "searches"},
		{"Glob", "globs"},
		{"Task", "tasks"},
		{"Custom", "calls"},
		{"", "calls"},
	}
	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			got := GroupSummaryUnit(tt.toolName)
			if got != tt.want {
				t.Errorf("GroupSummaryUnit(%q) = %q, want %q", tt.toolName, got, tt.want)
			}
		})
	}
}

func TestIsKnownTool(t *testing.T) {
	known := []string{"Read", "Edit", "Write", "Bash", "Grep", "Glob", "Task"}
	for _, name := range known {
		t.Run(name+"_known", func(t *testing.T) {
			if !IsKnownTool(name) {
				t.Errorf("IsKnownTool(%q) = false, want true", name)
			}
		})
	}

	unknown := []string{"Custom", "WebFetch", "read", "", "BASH"}
	for _, name := range unknown {
		t.Run(name+"_unknown", func(t *testing.T) {
			if IsKnownTool(name) {
				t.Errorf("IsKnownTool(%q) = true, want false", name)
			}
		})
	}
}
