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

	// Idle state: only show stopped timer placeholder and "Idle" status
	if p.Phase == model.PhaseIdle {
		centreRendered := dim.Render("⏱ --")
		rightRendered := dim.Render("Idle ")

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
	rightRendered := dim.Render(iterText+" ") + styledStatusIcon + dim.Render(" ")

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
