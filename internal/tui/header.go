package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/loxstomper/skinner/internal/model"
	"github.com/loxstomper/skinner/internal/theme"
)

// HeaderProps contains the data needed to render the header bar.
type HeaderProps struct {
	Phase           model.SessionPhase
	SessionDuration time.Duration
	InputTokens     int64
	OutputTokens    int64
	ContextPercent  int // -1 if unknown
	TotalCost       float64
	HasKnownModel   bool
	RateLimit       model.RateLimitInfo
	IterationCount  int
	MaxIterations   int
	SessionStatus   model.IterationStatus
	StatusFlash     string
	CPUPercent      *int // nil = no data yet
	MemPercent      *int // nil = no data yet
	Width           int
	Theme           theme.Theme
}

// RenderHeader renders the header bar as a single-line string.
// It is a pure function with no side effects.
func RenderHeader(p HeaderProps) string {
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color(p.Theme.ForegroundDim))

	// Status flash: show error message centered in header bar
	if p.StatusFlash != "" {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(p.Theme.StatusError))
		flashRendered := errorStyle.Render(p.StatusFlash)
		flashWidth := lipgloss.Width(flashRendered)
		leftPad := (p.Width - flashWidth) / 2
		if leftPad < 1 {
			leftPad = 1
		}
		rightPad := p.Width - leftPad - flashWidth
		if rightPad < 0 {
			rightPad = 0
		}
		return strings.Repeat(" ", leftPad) + flashRendered + strings.Repeat(" ", rightPad)
	}

	// System stats section (far right, after iteration indicator)
	statsRendered := renderSystemStats(p.CPUPercent, p.MemPercent, p.Theme)

	// Idle state: only show stopped timer placeholder and "Idle" status
	if p.Phase == model.PhaseIdle {
		centreRendered := dim.Render("⏱ --")
		rightRendered := dim.Render("Idle") + statsRendered

		centreWidth := lipgloss.Width(centreRendered)
		rightWidth := lipgloss.Width(rightRendered)
		availableWidth := p.Width - rightWidth
		leftPad := (availableWidth - centreWidth) / 2
		if leftPad < 1 {
			leftPad = 1
		}
		gap := p.Width - leftPad - centreWidth - rightWidth
		if gap < 0 {
			gap = 0
		}

		return strings.Repeat(" ", leftPad) + centreRendered + strings.Repeat(" ", gap) + rightRendered
	}

	// Build centre content: duration, tokens, context %, cost
	dur := FormatDurationValue(p.SessionDuration)
	centreText := fmt.Sprintf("⏱ %s   ↑%s ↓%s tokens", dur, FormatTokens(p.InputTokens), FormatTokens(p.OutputTokens))
	centreRendered := dim.Render(centreText)

	// Context window percentage
	if p.ContextPercent >= 0 {
		ctxText := fmt.Sprintf("   ctx %d%%", p.ContextPercent)
		var ctxColor string
		switch {
		case p.ContextPercent >= 90:
			ctxColor = p.Theme.StatusError
		case p.ContextPercent >= 70:
			ctxColor = p.Theme.StatusRunning
		default:
			ctxColor = p.Theme.ForegroundDim
		}
		centreRendered += lipgloss.NewStyle().Foreground(lipgloss.Color(ctxColor)).Render(ctxText)
	}

	// Cost
	if p.HasKnownModel {
		centreRendered += dim.Render(fmt.Sprintf("   ~$%.2f", p.TotalCost))
	}

	// Rate limit windows
	centreRendered += renderRateLimitField("5h", p.RateLimit.FiveHourPercent, p.Theme)
	centreRendered += renderRateLimitField("wk", p.RateLimit.WeeklyPercent, p.Theme)

	// Right side: iteration progress + status icon
	iterCount := p.IterationCount
	if iterCount == 0 {
		iterCount = 1
	}
	var iterText string
	if p.MaxIterations > 0 {
		iterText = fmt.Sprintf("Iter %d/%d", iterCount, p.MaxIterations)
	} else {
		iterText = fmt.Sprintf("Iter %d", iterCount)
	}

	var statusIcon, statusColor string
	switch p.SessionStatus {
	case model.IterationRunning:
		statusIcon = "⟳"
		statusColor = p.Theme.StatusRunning
	case model.IterationFailed:
		statusIcon = "✗"
		statusColor = p.Theme.StatusError
	default:
		statusIcon = "✓"
		statusColor = p.Theme.StatusSuccess
	}

	styledStatusIcon := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render(statusIcon)
	rightRendered := dim.Render(iterText+" ") + styledStatusIcon + statsRendered

	// Centre the stats in the space to the left of the right-aligned iteration indicator
	centreWidth := lipgloss.Width(centreRendered)
	rightWidth := lipgloss.Width(rightRendered)
	availableWidth := p.Width - rightWidth
	leftPad := (availableWidth - centreWidth) / 2
	if leftPad < 1 {
		leftPad = 1
	}
	gap := p.Width - leftPad - centreWidth - rightWidth
	if gap < 0 {
		gap = 0
	}

	return strings.Repeat(" ", leftPad) + centreRendered + strings.Repeat(" ", gap) + rightRendered
}

// renderSystemStats renders the system CPU and memory stats section.
// Returns empty string if both values are nil (e.g. non-Linux platform where /proc is unavailable).
func renderSystemStats(cpuPct, memPct *int, th theme.Theme) string {
	// If both are nil and we never got data, hide entirely (non-Linux graceful degradation)
	if cpuPct == nil && memPct == nil {
		return ""
	}

	dim := lipgloss.NewStyle().Foreground(lipgloss.Color(th.ForegroundDim))

	cpuRendered := renderStatValue("⚙", cpuPct, th)
	memRendered := renderStatValue("◼", memPct, th)

	return dim.Render("   ") + cpuRendered + dim.Render(" ") + memRendered + dim.Render(" ")
}

// renderStatValue renders a single stat indicator like "⚙ 42%" with color thresholds.
func renderStatValue(icon string, pct *int, th theme.Theme) string {
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color(th.ForegroundDim))
	if pct == nil {
		return dim.Render(fmt.Sprintf("%s --%%", icon))
	}

	var color string
	switch {
	case *pct >= 80:
		color = th.StatusError
	case *pct >= 50:
		color = th.StatusRunning
	default:
		color = th.StatusSuccess
	}

	style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	return style.Render(fmt.Sprintf("%s %d%%", icon, *pct))
}

// renderRateLimitField renders a single rate limit field (e.g. "5h: 34%" or "5h: --").
func renderRateLimitField(label string, pct *float64, th theme.Theme) string {
	if pct == nil {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(th.ForegroundDim)).
			Render(fmt.Sprintf("   %s: --", label))
	}

	text := fmt.Sprintf("   %s: %.0f%%", label, *pct)
	var color string
	switch {
	case *pct >= 90:
		color = th.StatusError
	case *pct >= 70:
		color = th.StatusRunning
	default:
		color = th.ForegroundDim
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(text)
}
