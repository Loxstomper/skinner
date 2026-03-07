package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/loxstomper/skinner/internal/model"
	"github.com/loxstomper/skinner/internal/theme"
)

func testTheme() theme.Theme {
	th, _ := theme.LookupTheme("solarized-dark")
	return th
}

func TestRenderHeader_DurationAndTokens(t *testing.T) {
	p := HeaderProps{
		SessionDuration: 2*time.Minute + 30*time.Second,
		InputTokens:     42100,
		OutputTokens:    8300,
		ContextPercent:  -1,
		HasKnownModel:   false,
		IterationCount:  1,
		MaxIterations:   0,
		SessionStatus:   model.IterationRunning,
		Width:           120,
		Theme:           testTheme(),
	}

	result := RenderHeader(p)

	if !strings.Contains(result, "⏱") {
		t.Error("expected duration icon ⏱")
	}
	if !strings.Contains(result, "2m30s") {
		t.Error("expected formatted duration 2m30s")
	}
	if !strings.Contains(result, "42.1k") {
		t.Error("expected input tokens 42.1k")
	}
	if !strings.Contains(result, "8.3k") {
		t.Error("expected output tokens 8.3k")
	}
	if !strings.Contains(result, "tokens") {
		t.Error("expected 'tokens' label")
	}
}

func TestRenderHeader_ContextPercent(t *testing.T) {
	tests := []struct {
		name    string
		pct     int
		want    string
		notWant string
	}{
		{"normal", 50, "ctx 50%", ""},
		{"warning", 75, "ctx 75%", ""},
		{"critical", 95, "ctx 95%", ""},
		{"unknown", -1, "", "ctx"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := HeaderProps{
				InputTokens:    1000,
				OutputTokens:   500,
				ContextPercent: tt.pct,
				HasKnownModel:  tt.pct >= 0,
				IterationCount: 1,
				SessionStatus:  model.IterationRunning,
				Width:          120,
				Theme:          testTheme(),
			}

			result := RenderHeader(p)

			if tt.want != "" && !strings.Contains(result, tt.want) {
				t.Errorf("expected %q in output", tt.want)
			}
			if tt.notWant != "" && strings.Contains(result, tt.notWant) {
				t.Errorf("did not expect %q in output", tt.notWant)
			}
		})
	}
}

func TestRenderHeader_Cost(t *testing.T) {
	t.Run("shown when known model", func(t *testing.T) {
		p := HeaderProps{
			TotalCost:      1.24,
			HasKnownModel:  true,
			IterationCount: 1,
			SessionStatus:  model.IterationRunning,
			Width:          120,
			Theme:          testTheme(),
		}
		result := RenderHeader(p)
		if !strings.Contains(result, "~$1.24") {
			t.Error("expected cost ~$1.24")
		}
	})

	t.Run("hidden when unknown model", func(t *testing.T) {
		p := HeaderProps{
			TotalCost:      1.24,
			HasKnownModel:  false,
			IterationCount: 1,
			SessionStatus:  model.IterationRunning,
			Width:          120,
			Theme:          testTheme(),
		}
		result := RenderHeader(p)
		if strings.Contains(result, "~$") {
			t.Error("did not expect cost when model is unknown")
		}
	})
}

func TestRenderHeader_IterationProgress(t *testing.T) {
	tests := []struct {
		name          string
		iterCount     int
		maxIterations int
		want          string
	}{
		{"unlimited", 3, 0, "Iter 3"},
		{"limited", 3, 10, "Iter 3/10"},
		{"zero iterations shows 1", 0, 0, "Iter 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := HeaderProps{
				IterationCount: tt.iterCount,
				MaxIterations:  tt.maxIterations,
				SessionStatus:  model.IterationRunning,
				Width:          120,
				Theme:          testTheme(),
			}
			result := RenderHeader(p)
			if !strings.Contains(result, tt.want) {
				t.Errorf("expected %q in output, got: %s", tt.want, result)
			}
		})
	}
}

func TestRenderHeader_StatusIcons(t *testing.T) {
	tests := []struct {
		name   string
		status model.IterationStatus
		icon   string
	}{
		{"running", model.IterationRunning, "⟳"},
		{"completed", model.IterationCompleted, "✓"},
		{"failed", model.IterationFailed, "✗"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := HeaderProps{
				IterationCount: 1,
				SessionStatus:  tt.status,
				Width:          120,
				Theme:          testTheme(),
			}
			result := RenderHeader(p)
			if !strings.Contains(result, tt.icon) {
				t.Errorf("expected status icon %q", tt.icon)
			}
		})
	}
}
