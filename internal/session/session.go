package session

import (
	"time"

	"github.com/loxstomper/skinner/internal/config"
	"github.com/loxstomper/skinner/internal/model"
)

// Controller owns all non-UI business logic: event processing, tool call
// grouping, token accumulation, cost calculation, and iteration lifecycle.
// It operates on a model.Session and has no I/O or Bubble Tea dependency.
type Controller struct {
	Session *model.Session
	Config  config.Config
	Clock   func() time.Time // injectable, defaults to time.Now

	// Internal tracking for model pricing
	hasKnownModel bool
	lastModel     string

	// Pending token attribution: set by ProcessUsage, consumed by the next
	// ProcessAssistantBatch call. The executor sends UsageEvent before
	// AssistantBatchEvent for the same assistant turn.
	pendingInputTokens     int64
	pendingCacheReadTokens int64
}

// NewController creates a Controller with the given session and config.
// Clock defaults to time.Now if nil.
func NewController(session *model.Session, cfg config.Config, clock func() time.Time) *Controller {
	if clock == nil {
		clock = time.Now
	}
	return &Controller{
		Session: session,
		Config:  cfg,
		Clock:   clock,
	}
}

// toolRun tracks a consecutive run of same-name tool use events during
// batch processing.
type toolRun struct {
	name   string
	events []ToolUseEvent
}

// ProcessAssistantBatch handles a batch of tool use and text events from one
// assistant message. It groups consecutive same-name tool calls: runs of 1
// become standalone ToolCalls, runs of 2+ become ToolCallGroups. Items are
// appended to the running iteration.
func (c *Controller) ProcessAssistantBatch(events []Event) {
	idx := c.RunningIterationIdx()
	if idx < 0 {
		return
	}
	iter := &c.Session.Iterations[idx]

	// Collect runs of consecutive same-name ToolUseEvents into groups.
	var pending []interface{} // *model.TextBlock or *toolRun
	var currentRun *toolRun

	flushRun := func() {
		if currentRun != nil {
			pending = append(pending, currentRun)
			currentRun = nil
		}
	}

	for _, evt := range events {
		switch e := evt.(type) {
		case ToolUseEvent:
			if currentRun != nil && currentRun.name == e.Name {
				currentRun.events = append(currentRun.events, e)
			} else {
				flushRun()
				currentRun = &toolRun{name: e.Name, events: []ToolUseEvent{e}}
			}
		case TextEvent:
			flushRun()
			pending = append(pending, &model.TextBlock{Text: e.Text})
		}
	}
	flushRun()

	// Count total tool calls for token attribution.
	var totalToolCalls int
	for _, p := range pending {
		if run, ok := p.(*toolRun); ok {
			totalToolCalls += len(run.events)
		}
	}

	// Distribute pending tokens equally across tool calls, then clear.
	var perCallInput, perCallCacheRead int64
	if totalToolCalls > 0 {
		perCallInput = (c.pendingInputTokens + int64(totalToolCalls)/2) / int64(totalToolCalls)
		perCallCacheRead = (c.pendingCacheReadTokens + int64(totalToolCalls)/2) / int64(totalToolCalls)
	}
	c.pendingInputTokens = 0
	c.pendingCacheReadTokens = 0

	// Convert pending items to timeline items
	now := c.Clock()
	for _, p := range pending {
		switch v := p.(type) {
		case *model.TextBlock:
			iter.Items = append(iter.Items, v)
		case *toolRun:
			if len(v.events) == 1 {
				e := v.events[0]
				iter.Items = append(iter.Items, &model.ToolCall{
					ID:              e.ID,
					Name:            e.Name,
					Summary:         e.Summary,
					LineInfo:        e.LineInfo,
					StartTime:       now,
					Status:          model.ToolCallRunning,
					RawInput:        e.RawInput,
					InputTokens:     perCallInput,
					CacheReadTokens: perCallCacheRead,
				})
			} else {
				group := &model.ToolCallGroup{
					ToolName:     v.name,
					Expanded:     true,
					ManualToggle: false,
				}
				for _, e := range v.events {
					group.Children = append(group.Children, &model.ToolCall{
						ID:              e.ID,
						Name:            e.Name,
						Summary:         e.Summary,
						LineInfo:        e.LineInfo,
						StartTime:       now,
						Status:          model.ToolCallRunning,
						RawInput:        e.RawInput,
						InputTokens:     perCallInput,
						CacheReadTokens: perCallCacheRead,
					})
				}
				iter.Items = append(iter.Items, group)
			}
		}
	}
}

// ProcessToolResult finds the matching tool call by ID and applies the result
// status, duration, and line info. Returns the affected ToolCallGroup (if any)
// so the caller can handle expand/collapse UI concerns. Returns nil if the
// tool call was standalone or not found.
func (c *Controller) ProcessToolResult(result ToolResultEvent) *model.ToolCallGroup {
	idx := c.RunningIterationIdx()
	if idx < 0 {
		return nil
	}
	iter := &c.Session.Iterations[idx]

	for _, item := range iter.Items {
		if tc, ok := item.(*model.ToolCall); ok && tc.ID == result.ToolUseID {
			c.applyToolResult(tc, result)
			return nil
		}
		if group, ok := item.(*model.ToolCallGroup); ok {
			for _, child := range group.Children {
				if child.ID == result.ToolUseID {
					c.applyToolResult(child, result)
					return group
				}
			}
		}
	}
	return nil
}

// applyToolResult updates a single ToolCall with result data.
func (c *Controller) applyToolResult(tc *model.ToolCall, result ToolResultEvent) {
	tc.Duration = c.Clock().Sub(tc.StartTime)
	tc.IsError = result.IsError
	tc.ResultContent = result.Content
	if result.IsError {
		tc.Status = model.ToolCallError
	} else {
		tc.Status = model.ToolCallDone
	}
	// Read tool gets line info from the result (not the tool_use input)
	if result.LineInfo != "" && tc.LineInfo == "" && tc.Name == "Read" {
		tc.LineInfo = result.LineInfo
	}
}

// ProcessUsage accumulates token counts and computes cost using pricing config.
// It also stores pending per-tool-call attribution tokens that will be
// distributed by the next ProcessAssistantBatch call.
func (c *Controller) ProcessUsage(usage UsageEvent) {
	c.Session.InputTokens += usage.InputTokens
	c.Session.OutputTokens += usage.OutputTokens
	c.Session.CacheReadTokens += usage.CacheReadInputTokens
	c.Session.CacheCreationTokens += usage.CacheCreationInputTokens
	c.Session.LastInputTokens = usage.InputTokens
	c.Session.LastCacheReadTokens = usage.CacheReadInputTokens

	// Store pending tokens for per-tool-call attribution. The executor
	// sends UsageEvent before AssistantBatchEvent for the same turn.
	c.pendingInputTokens = usage.InputTokens
	c.pendingCacheReadTokens = usage.CacheReadInputTokens

	if pricing, ok := c.Config.Pricing[usage.Model]; ok {
		c.hasKnownModel = true
		c.lastModel = usage.Model
		c.Session.TotalCost += float64(usage.InputTokens) * pricing.Input
		c.Session.TotalCost += float64(usage.OutputTokens) * pricing.Output
		c.Session.TotalCost += float64(usage.CacheReadInputTokens) * pricing.CacheRead
		c.Session.TotalCost += float64(usage.CacheCreationInputTokens) * pricing.CacheCreate
	}
}

// StartIteration creates a new running iteration and appends it to the session.
func (c *Controller) StartIteration() {
	iter := model.Iteration{
		Index:     len(c.Session.Iterations),
		Status:    model.IterationRunning,
		StartTime: c.Clock(),
	}
	c.Session.Iterations = append(c.Session.Iterations, iter)
}

// CompleteIteration marks the running iteration as completed or failed and
// records its duration.
func (c *Controller) CompleteIteration(err error) {
	idx := c.RunningIterationIdx()
	if idx < 0 {
		return
	}
	iter := &c.Session.Iterations[idx]
	iter.Duration = c.Clock().Sub(iter.StartTime)
	if err != nil {
		iter.Status = model.IterationFailed
	} else {
		iter.Status = model.IterationCompleted
	}
}

// ShouldStartNext returns true if another iteration should begin.
// The caller must check quitting state separately.
func (c *Controller) ShouldStartNext() bool {
	count := len(c.Session.Iterations)
	if c.Session.MaxIterations > 0 && count >= c.Session.MaxIterations {
		return false
	}
	return true
}

// RunningIterationIdx returns the index of the running iteration, or -1.
func (c *Controller) RunningIterationIdx() int {
	for i, iter := range c.Session.Iterations {
		if iter.Status == model.IterationRunning {
			return i
		}
	}
	return -1
}

// HasKnownModel returns true if pricing info is available for the current model.
func (c *Controller) HasKnownModel() bool {
	return c.hasKnownModel
}

// LastModel returns the model name from the most recent usage event with
// known pricing.
func (c *Controller) LastModel() string {
	return c.lastModel
}
