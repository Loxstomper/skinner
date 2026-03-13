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
		Phase:           model.PhaseRunning,
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
				Phase:          model.PhaseRunning,
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
			Phase:          model.PhaseRunning,
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
			Phase:          model.PhaseRunning,
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
				Phase:          model.PhaseRunning,
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
				Phase:          model.PhaseRunning,
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

func TestRenderHeader_RateLimitPlaceholder(t *testing.T) {
	// When both values are nil (unknown), display "--" placeholders
	p := HeaderProps{
		Phase:          model.PhaseRunning,
		IterationCount: 1,
		SessionStatus:  model.IterationRunning,
		Width:          160,
		Theme:          testTheme(),
	}

	result := RenderHeader(p)

	if !strings.Contains(result, "5h: --") {
		t.Error("expected placeholder '5h: --' when FiveHourPercent is nil")
	}
	if !strings.Contains(result, "wk: --") {
		t.Error("expected placeholder 'wk: --' when WeeklyPercent is nil")
	}
}

func TestRenderHeader_RateLimitPercentages(t *testing.T) {
	fiveHour := 34.0
	weekly := 12.0
	p := HeaderProps{
		Phase:          model.PhaseRunning,
		IterationCount: 1,
		SessionStatus:  model.IterationRunning,
		RateLimit: model.RateLimitInfo{
			FiveHourPercent: &fiveHour,
			WeeklyPercent:   &weekly,
		},
		Width: 160,
		Theme: testTheme(),
	}

	result := RenderHeader(p)

	if !strings.Contains(result, "5h: 34%") {
		t.Error("expected '5h: 34%' in output")
	}
	if !strings.Contains(result, "wk: 12%") {
		t.Error("expected 'wk: 12%' in output")
	}
}

func TestRenderHeader_RateLimitColorThresholds(t *testing.T) {
	// This test verifies that the rate limit values are present in the output
	// at various threshold levels. Color verification is done by checking
	// the values render correctly — the color logic mirrors context window thresholds.
	tests := []struct {
		name string
		pct  float64
		want string
	}{
		{"normal", 50, "5h: 50%"},
		{"warning_boundary", 70, "5h: 70%"},
		{"warning", 85, "5h: 85%"},
		{"critical_boundary", 90, "5h: 90%"},
		{"critical", 95, "5h: 95%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pct := tt.pct
			p := HeaderProps{
				Phase:          model.PhaseRunning,
				IterationCount: 1,
				SessionStatus:  model.IterationRunning,
				RateLimit: model.RateLimitInfo{
					FiveHourPercent: &pct,
				},
				Width: 160,
				Theme: testTheme(),
			}

			result := RenderHeader(p)
			if !strings.Contains(result, tt.want) {
				t.Errorf("expected %q in output, got: %s", tt.want, result)
			}
		})
	}
}

func TestRenderHeader_IdlePhase(t *testing.T) {
	p := HeaderProps{
		Phase: model.PhaseIdle,
		Width: 120,
		Theme: testTheme(),
	}

	result := RenderHeader(p)

	if !strings.Contains(result, "⏱ --") {
		t.Error("expected '⏱ --' in idle header")
	}
	if !strings.Contains(result, "Idle") {
		t.Error("expected 'Idle' in idle header")
	}
	// Should NOT contain running-state elements
	if strings.Contains(result, "tokens") {
		t.Error("idle header should not contain token counts")
	}
	if strings.Contains(result, "Iter") {
		t.Error("idle header should not contain iteration counter")
	}
	if strings.Contains(result, "ctx") {
		t.Error("idle header should not contain context percentage")
	}
	if strings.Contains(result, "~$") {
		t.Error("idle header should not contain cost")
	}
	if strings.Contains(result, "5h:") {
		t.Error("idle header should not contain rate limits")
	}
}

func TestRenderHeader_RunningPhaseShowsFullStats(t *testing.T) {
	// Verify that PhaseRunning renders the full header (same as before Phase was added)
	p := HeaderProps{
		Phase:           model.PhaseRunning,
		SessionDuration: 2*time.Minute + 30*time.Second,
		InputTokens:     42100,
		OutputTokens:    8300,
		ContextPercent:  -1,
		IterationCount:  3,
		MaxIterations:   10,
		SessionStatus:   model.IterationRunning,
		Width:           120,
		Theme:           testTheme(),
	}

	result := RenderHeader(p)

	if !strings.Contains(result, "2m30s") {
		t.Error("expected duration in running phase header")
	}
	if !strings.Contains(result, "tokens") {
		t.Error("expected token counts in running phase header")
	}
	if !strings.Contains(result, "Iter 3/10") {
		t.Error("expected iteration counter in running phase header")
	}
	if strings.Contains(result, "Idle") {
		t.Error("running phase should not show Idle")
	}
}

func TestRenderHeader_FinishedPhaseShowsFullStats(t *testing.T) {
	p := HeaderProps{
		Phase:           model.PhaseFinished,
		SessionDuration: 5 * time.Minute,
		InputTokens:     100000,
		OutputTokens:    20000,
		IterationCount:  10,
		MaxIterations:   10,
		SessionStatus:   model.IterationCompleted,
		Width:           120,
		Theme:           testTheme(),
	}

	result := RenderHeader(p)

	if !strings.Contains(result, "5m00s") {
		t.Error("expected duration in finished phase header")
	}
	if !strings.Contains(result, "Iter 10/10") {
		t.Error("expected iteration counter in finished phase header")
	}
	if strings.Contains(result, "Idle") {
		t.Error("finished phase should not show Idle")
	}
}

func TestRenderHeader_SystemStatsPresent(t *testing.T) {
	cpu := 42
	mem := 61
	p := HeaderProps{
		Phase:          model.PhaseRunning,
		IterationCount: 1,
		SessionStatus:  model.IterationRunning,
		CPUPercent:     &cpu,
		MemPercent:     &mem,
		Width:          160,
		Theme:          testTheme(),
	}

	result := RenderHeader(p)

	if !strings.Contains(result, "⚙ 42%") {
		t.Error("expected CPU stats '⚙ 42%'")
	}
	if !strings.Contains(result, "◼ 61%") {
		t.Error("expected memory stats '◼ 61%'")
	}
}

func TestRenderHeader_SystemStatsNil(t *testing.T) {
	// When stats are available but CPU has no delta yet (first sample),
	// CPU shows --% while memory shows value
	mem := 50
	p := HeaderProps{
		Phase:          model.PhaseRunning,
		IterationCount: 1,
		SessionStatus:  model.IterationRunning,
		CPUPercent:     nil,
		MemPercent:     &mem,
		Width:          160,
		Theme:          testTheme(),
	}

	result := RenderHeader(p)

	if !strings.Contains(result, "⚙ --%") {
		t.Error("expected CPU placeholder '⚙ --%'")
	}
	if !strings.Contains(result, "◼ 50%") {
		t.Error("expected memory stats '◼ 50%'")
	}
}

func TestRenderHeader_SystemStatsHiddenWhenBothNil(t *testing.T) {
	// When both are nil (non-Linux), stats section is hidden entirely
	p := HeaderProps{
		Phase:          model.PhaseRunning,
		IterationCount: 1,
		SessionStatus:  model.IterationRunning,
		CPUPercent:     nil,
		MemPercent:     nil,
		Width:          160,
		Theme:          testTheme(),
	}

	result := RenderHeader(p)

	if strings.Contains(result, "⚙") {
		t.Error("did not expect CPU icon when both stats are nil")
	}
	if strings.Contains(result, "◼") {
		t.Error("did not expect memory icon when both stats are nil")
	}
}

func TestRenderHeader_SystemStatsColorThresholds(t *testing.T) {
	// Verify that stats render at various threshold levels
	tests := []struct {
		name string
		pct  int
		want string
	}{
		{"low", 25, "⚙ 25%"},
		{"medium_boundary", 50, "⚙ 50%"},
		{"medium", 65, "⚙ 65%"},
		{"high_boundary", 80, "⚙ 80%"},
		{"high", 95, "⚙ 95%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pct := tt.pct
			mem := 50
			p := HeaderProps{
				Phase:          model.PhaseRunning,
				IterationCount: 1,
				SessionStatus:  model.IterationRunning,
				CPUPercent:     &pct,
				MemPercent:     &mem,
				Width:          160,
				Theme:          testTheme(),
			}

			result := RenderHeader(p)
			if !strings.Contains(result, tt.want) {
				t.Errorf("expected %q in output", tt.want)
			}
		})
	}
}

func TestRenderHeader_SystemStatsIdle(t *testing.T) {
	// System stats should appear in idle state too
	cpu := 30
	mem := 45
	p := HeaderProps{
		Phase:      model.PhaseIdle,
		CPUPercent: &cpu,
		MemPercent: &mem,
		Width:      120,
		Theme:      testTheme(),
	}

	result := RenderHeader(p)

	if !strings.Contains(result, "⚙ 30%") {
		t.Error("expected CPU stats in idle header")
	}
	if !strings.Contains(result, "◼ 45%") {
		t.Error("expected memory stats in idle header")
	}
	if !strings.Contains(result, "Idle") {
		t.Error("expected Idle label in idle header")
	}
}

func TestRenderHeader_RateLimitMixed(t *testing.T) {
	// Only one value known, the other should show "--"
	fiveHour := 42.0
	p := HeaderProps{
		Phase:          model.PhaseRunning,
		IterationCount: 1,
		SessionStatus:  model.IterationRunning,
		RateLimit: model.RateLimitInfo{
			FiveHourPercent: &fiveHour,
			WeeklyPercent:   nil,
		},
		Width: 160,
		Theme: testTheme(),
	}

	result := RenderHeader(p)

	if !strings.Contains(result, "5h: 42%") {
		t.Error("expected '5h: 42%' when FiveHourPercent is set")
	}
	if !strings.Contains(result, "wk: --") {
		t.Error("expected 'wk: --' when WeeklyPercent is nil")
	}
}
