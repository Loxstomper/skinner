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
	ID        string
	Name      string
	Summary   string
	LineInfo  string
	StartTime time.Time
	Duration  time.Duration
	Status    ToolCallStatus
	IsError   bool
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

type IterationStatus int

const (
	IterationRunning IterationStatus = iota
	IterationCompleted
	IterationFailed
)

type Iteration struct {
	Index     int
	Status    IterationStatus
	Items     []TimelineItem
	StartTime time.Time
	Duration  time.Duration
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

type Session struct {
	Iterations    []Iteration
	Mode          string // "build" or "plan"
	PromptFile    string
	MaxIterations int // 0 = unlimited

	// Token tracking (accumulated across all iterations)
	InputTokens         int64
	OutputTokens        int64
	CacheReadTokens     int64
	CacheCreationTokens int64
	TotalCost           float64

	// Latest usage from most recent assistant event (replaced, not accumulated)
	LastInputTokens     int64
	LastCacheReadTokens int64

	StartTime time.Time
}
