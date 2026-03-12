package model

import "time"

// TimelineItem is the interface for items displayed in the right pane timeline.
type TimelineItem interface {
	timelineItem()
}

type ToolCallStatus int

const (
	ToolCallRunning ToolCallStatus = iota
	ToolCallDone
	ToolCallError
)

type ToolCall struct {
	ID            string
	Name          string
	Summary       string
	LineInfo      string
	StartTime     time.Time
	Duration      time.Duration
	Status        ToolCallStatus
	IsError       bool
	RawInput      map[string]interface{}
	ResultContent string
	Expanded      bool

	// Per-tool-call token attribution (divided equally from assistant turn usage)
	InputTokens     int64
	CacheReadTokens int64
}

func (*ToolCall) timelineItem() {}

type TextBlock struct {
	Text     string
	Expanded bool
}

func (*TextBlock) timelineItem() {}

type ToolCallGroup struct {
	ToolName     string
	Children     []*ToolCall
	Expanded     bool
	ManualToggle bool
}

func (*ToolCallGroup) timelineItem() {}

func (g *ToolCallGroup) Status() ToolCallStatus {
	hasError := false
	for _, c := range g.Children {
		if c.Status == ToolCallRunning {
			return ToolCallRunning
		}
		if c.Status == ToolCallError {
			hasError = true
		}
	}
	if hasError {
		return ToolCallError
	}
	return ToolCallDone
}

func (g *ToolCallGroup) GroupDuration() time.Duration {
	if g.Status() == ToolCallRunning {
		return 0
	}
	var earliest time.Time
	var latestEnd time.Time
	for i, c := range g.Children {
		if i == 0 || c.StartTime.Before(earliest) {
			earliest = c.StartTime
		}
		end := c.StartTime.Add(c.Duration)
		if i == 0 || end.After(latestEnd) {
			latestEnd = end
		}
	}
	return latestEnd.Sub(earliest)
}

func (g *ToolCallGroup) CompletedCount() int {
	count := 0
	for _, c := range g.Children {
		if c.Status != ToolCallRunning {
			count++
		}
	}
	return count
}

func (g *ToolCallGroup) ToolCallCount() int {
	return len(g.Children)
}

// SessionPhase represents the current phase of the session lifecycle.
type SessionPhase int

const (
	PhaseIdle     SessionPhase = iota // TUI loaded, no run in progress
	PhaseRunning                      // Iterations executing
	PhaseFinished                     // Current run completed
)

// Run represents a sequence of iterations using a single prompt file.
type Run struct {
	PromptName    string // display name, e.g. "BUILD"
	PromptFile    string // full path, e.g. "PROMPT_BUILD.md"
	StartIndex    int    // first iteration index in session
	MaxIterations int    // 0 = unlimited, per-run limit
}

type IterationStatus int

const (
	IterationRunning IterationStatus = iota
	IterationCompleted
	IterationFailed
)

type Iteration struct {
	Index             int
	Status            IterationStatus
	Items             []TimelineItem
	StartTime         time.Time
	Duration          time.Duration
	ThinkingStartTime time.Time // zero value = not thinking
}

// HasRunningToolCall returns true if any tool call in the iteration is still running.
func (iter *Iteration) HasRunningToolCall() bool {
	for _, item := range iter.Items {
		switch it := item.(type) {
		case *ToolCall:
			if it.Status == ToolCallRunning {
				return true
			}
		case *ToolCallGroup:
			if it.Status() == ToolCallRunning {
				return true
			}
		}
	}
	return false
}

// IsThinking returns true when the iteration is running, has a thinking start
// time set, and no tool calls are currently in progress.
func (iter *Iteration) IsThinking() bool {
	return iter.Status == IterationRunning && !iter.ThinkingStartTime.IsZero() && !iter.HasRunningToolCall()
}

func (iter *Iteration) ToolCallCount() int {
	count := 0
	for _, item := range iter.Items {
		switch it := item.(type) {
		case *ToolCall:
			count++
		case *ToolCallGroup:
			count += it.ToolCallCount()
		}
	}
	return count
}

// RateLimitInfo holds API rate limit window utilization percentages.
// Nil pointer fields indicate unknown/unfetched values.
type RateLimitInfo struct {
	FiveHourPercent *float64 // 5-hour rolling window utilization, nil = unknown
	WeeklyPercent   *float64 // weekly rolling window utilization, nil = unknown
}

type Session struct {
	Iterations    []Iteration
	Mode          string // "build", "plan", or "idle"
	PromptFile    string
	MaxIterations int // 0 = unlimited

	// Run tracking
	Runs  []Run
	Phase SessionPhase

	// Accumulated duration across runs (for pause/resume timer)
	AccumulatedDuration time.Duration

	// Token tracking (accumulated across all iterations)
	InputTokens         int64
	OutputTokens        int64
	CacheReadTokens     int64
	CacheCreationTokens int64
	TotalCost           float64

	// Latest usage from most recent assistant event (replaced, not accumulated)
	LastInputTokens     int64
	LastCacheReadTokens int64

	// Rate limit window utilization
	// TODO: implement rate limit data fetching at iteration start
	RateLimit RateLimitInfo

	StartTime time.Time
}
