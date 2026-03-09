package session

import (
	"testing"
	"time"

	"github.com/loxstomper/skinner/internal/config"
	"github.com/loxstomper/skinner/internal/model"
)

// fakeClock returns a clock function that returns the value of the pointed-to time.
func fakeClock(t *time.Time) func() time.Time {
	return func() time.Time { return *t }
}

func defaultTestConfig() config.Config {
	return config.Config{
		Pricing: map[string]config.ModelPricing{
			"claude-sonnet-4-5": {
				Input:         0.000003,
				Output:        0.000015,
				CacheRead:     0.0000003,
				CacheCreate:   0.00000375,
				ContextWindow: 200000,
			},
		},
	}
}

func TestProcessAssistantBatch_SingleToolCall(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartIteration()
	ctrl.ProcessAssistantBatch([]Event{
		ToolUseEvent{
			ID: "t1", Name: "Read", Summary: "/foo/bar.go", LineInfo: "",
			RawInput: map[string]interface{}{"file_path": "/foo/bar.go"},
		},
	})

	if len(sess.Iterations) != 1 {
		t.Fatalf("expected 1 iteration, got %d", len(sess.Iterations))
	}
	iter := sess.Iterations[0]
	if len(iter.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(iter.Items))
	}

	tc, ok := iter.Items[0].(*model.ToolCall)
	if !ok {
		t.Fatalf("expected *model.ToolCall, got %T", iter.Items[0])
	}
	if tc.ID != "t1" {
		t.Errorf("expected ID t1, got %s", tc.ID)
	}
	if tc.Name != "Read" {
		t.Errorf("expected Name Read, got %s", tc.Name)
	}
	if tc.Summary != "/foo/bar.go" {
		t.Errorf("expected Summary /foo/bar.go, got %s", tc.Summary)
	}
	if tc.Status != model.ToolCallRunning {
		t.Errorf("expected ToolCallRunning, got %d", tc.Status)
	}
	if !tc.StartTime.Equal(now) {
		t.Errorf("expected StartTime %v, got %v", now, tc.StartTime)
	}
	if tc.RawInput == nil {
		t.Fatal("expected RawInput to be non-nil")
	}
	if fp, ok := tc.RawInput["file_path"].(string); !ok || fp != "/foo/bar.go" {
		t.Errorf("expected RawInput[file_path] = /foo/bar.go, got %v", tc.RawInput["file_path"])
	}
}

func TestProcessAssistantBatch_GroupsConsecutiveSameNameTools(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartIteration()
	ctrl.ProcessAssistantBatch([]Event{
		ToolUseEvent{ID: "t1", Name: "Read", Summary: "a.go"},
		ToolUseEvent{ID: "t2", Name: "Read", Summary: "b.go"},
		ToolUseEvent{ID: "t3", Name: "Read", Summary: "c.go"},
	})

	iter := sess.Iterations[0]
	if len(iter.Items) != 1 {
		t.Fatalf("expected 1 group item, got %d items", len(iter.Items))
	}

	group, ok := iter.Items[0].(*model.ToolCallGroup)
	if !ok {
		t.Fatalf("expected *model.ToolCallGroup, got %T", iter.Items[0])
	}
	if group.ToolName != "Read" {
		t.Errorf("expected group ToolName Read, got %s", group.ToolName)
	}
	if len(group.Children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(group.Children))
	}
	if !group.Expanded {
		t.Error("expected group to start expanded")
	}
	if group.ManualToggle {
		t.Error("expected ManualToggle false")
	}

	for i, child := range group.Children {
		if child.Status != model.ToolCallRunning {
			t.Errorf("child %d: expected ToolCallRunning, got %d", i, child.Status)
		}
	}
}

func TestProcessAssistantBatch_MixedToolsNotGrouped(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartIteration()
	ctrl.ProcessAssistantBatch([]Event{
		ToolUseEvent{ID: "t1", Name: "Read", Summary: "a.go"},
		ToolUseEvent{ID: "t2", Name: "Write", Summary: "b.go"},
		ToolUseEvent{ID: "t3", Name: "Read", Summary: "c.go"},
	})

	iter := sess.Iterations[0]
	if len(iter.Items) != 3 {
		t.Fatalf("expected 3 standalone items, got %d", len(iter.Items))
	}
	for i, item := range iter.Items {
		if _, ok := item.(*model.ToolCall); !ok {
			t.Errorf("item %d: expected *model.ToolCall, got %T", i, item)
		}
	}
}

func TestProcessAssistantBatch_TextEventsBreakRuns(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartIteration()
	ctrl.ProcessAssistantBatch([]Event{
		ToolUseEvent{ID: "t1", Name: "Read", Summary: "a.go"},
		ToolUseEvent{ID: "t2", Name: "Read", Summary: "b.go"},
		TextEvent{Text: "thinking..."},
		ToolUseEvent{ID: "t3", Name: "Read", Summary: "c.go"},
	})

	iter := sess.Iterations[0]
	if len(iter.Items) != 3 {
		t.Fatalf("expected 3 items (group, text, standalone), got %d", len(iter.Items))
	}

	// First item: group of 2 Reads
	group, ok := iter.Items[0].(*model.ToolCallGroup)
	if !ok {
		t.Fatalf("item 0: expected *model.ToolCallGroup, got %T", iter.Items[0])
	}
	if len(group.Children) != 2 {
		t.Errorf("expected 2 children in group, got %d", len(group.Children))
	}

	// Second item: text block
	tb, ok := iter.Items[1].(*model.TextBlock)
	if !ok {
		t.Fatalf("item 1: expected *model.TextBlock, got %T", iter.Items[1])
	}
	if tb.Text != "thinking..." {
		t.Errorf("expected text 'thinking...', got %q", tb.Text)
	}

	// Third item: standalone Read
	tc, ok := iter.Items[2].(*model.ToolCall)
	if !ok {
		t.Fatalf("item 2: expected *model.ToolCall, got %T", iter.Items[2])
	}
	if tc.ID != "t3" {
		t.Errorf("expected ID t3, got %s", tc.ID)
	}
}

func TestProcessAssistantBatch_TextOnly(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartIteration()
	ctrl.ProcessAssistantBatch([]Event{
		TextEvent{Text: "hello"},
		TextEvent{Text: "world"},
	})

	iter := sess.Iterations[0]
	if len(iter.Items) != 2 {
		t.Fatalf("expected 2 text items, got %d", len(iter.Items))
	}
	for i, item := range iter.Items {
		tb, ok := item.(*model.TextBlock)
		if !ok {
			t.Fatalf("item %d: expected *model.TextBlock, got %T", i, item)
		}
		if tb.Expanded {
			t.Errorf("item %d: expected not expanded", i)
		}
	}
}

func TestProcessAssistantBatch_EmptyBatch(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartIteration()
	ctrl.ProcessAssistantBatch([]Event{})

	iter := sess.Iterations[0]
	if len(iter.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(iter.Items))
	}
}

func TestProcessAssistantBatch_NoRunningIteration(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	// Don't start an iteration — should not panic
	ctrl.ProcessAssistantBatch([]Event{
		ToolUseEvent{ID: "t1", Name: "Read", Summary: "a.go"},
	})

	if len(sess.Iterations) != 0 {
		t.Errorf("expected 0 iterations, got %d", len(sess.Iterations))
	}
}

func TestProcessToolResult_Standalone(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartIteration()
	ctrl.ProcessAssistantBatch([]Event{
		ToolUseEvent{ID: "t1", Name: "Read", Summary: "a.go"},
	})

	// Advance clock before result
	now = now.Add(500 * time.Millisecond)

	group := ctrl.ProcessToolResult(ToolResultEvent{
		ToolUseID: "t1",
		IsError:   false,
		LineInfo:  "(42 lines)",
		Content:   "file contents here\nline 2",
	})

	if group != nil {
		t.Error("expected nil group for standalone tool call")
	}

	tc := sess.Iterations[0].Items[0].(*model.ToolCall)
	if tc.Status != model.ToolCallDone {
		t.Errorf("expected ToolCallDone, got %d", tc.Status)
	}
	if tc.Duration != 500*time.Millisecond {
		t.Errorf("expected 500ms duration, got %v", tc.Duration)
	}
	// Read tool gets LineInfo from result
	if tc.LineInfo != "(42 lines)" {
		t.Errorf("expected LineInfo '(42 lines)', got %q", tc.LineInfo)
	}
	if tc.ResultContent != "file contents here\nline 2" {
		t.Errorf("expected ResultContent to be set, got %q", tc.ResultContent)
	}
}

func TestProcessToolResult_GroupChild(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartIteration()
	ctrl.ProcessAssistantBatch([]Event{
		ToolUseEvent{ID: "t1", Name: "Read", Summary: "a.go"},
		ToolUseEvent{ID: "t2", Name: "Read", Summary: "b.go"},
	})

	now = now.Add(200 * time.Millisecond)
	group := ctrl.ProcessToolResult(ToolResultEvent{ToolUseID: "t1"})

	if group == nil {
		t.Fatal("expected non-nil group")
	}
	if group.ToolName != "Read" {
		t.Errorf("expected group ToolName Read, got %s", group.ToolName)
	}
	if group.Children[0].Status != model.ToolCallDone {
		t.Error("expected first child done")
	}
	if group.Children[1].Status != model.ToolCallRunning {
		t.Error("expected second child still running")
	}
	if group.Status() != model.ToolCallRunning {
		t.Error("expected group still running (one child pending)")
	}
}

func TestProcessToolResult_Error(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartIteration()
	ctrl.ProcessAssistantBatch([]Event{
		ToolUseEvent{ID: "t1", Name: "Bash", Summary: "run tests"},
	})

	now = now.Add(time.Second)
	ctrl.ProcessToolResult(ToolResultEvent{ToolUseID: "t1", IsError: true})

	tc := sess.Iterations[0].Items[0].(*model.ToolCall)
	if tc.Status != model.ToolCallError {
		t.Errorf("expected ToolCallError, got %d", tc.Status)
	}
	if !tc.IsError {
		t.Error("expected IsError true")
	}
}

func TestProcessToolResult_ReadLineInfoFromResult(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartIteration()
	ctrl.ProcessAssistantBatch([]Event{
		ToolUseEvent{ID: "t1", Name: "Read", Summary: "/foo.go", LineInfo: ""},
	})

	now = now.Add(100 * time.Millisecond)
	ctrl.ProcessToolResult(ToolResultEvent{ToolUseID: "t1", LineInfo: "(100 lines)"})

	tc := sess.Iterations[0].Items[0].(*model.ToolCall)
	if tc.LineInfo != "(100 lines)" {
		t.Errorf("expected LineInfo from result, got %q", tc.LineInfo)
	}
}

func TestProcessToolResult_EditKeepsExistingLineInfo(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartIteration()
	ctrl.ProcessAssistantBatch([]Event{
		ToolUseEvent{ID: "t1", Name: "Edit", Summary: "/foo.go", LineInfo: "(+5)"},
	})

	now = now.Add(100 * time.Millisecond)
	ctrl.ProcessToolResult(ToolResultEvent{ToolUseID: "t1", LineInfo: "(100 lines)"})

	tc := sess.Iterations[0].Items[0].(*model.ToolCall)
	// Edit already has LineInfo from tool_use, so result LineInfo is not applied
	if tc.LineInfo != "(+5)" {
		t.Errorf("expected LineInfo from tool_use input '(+5)', got %q", tc.LineInfo)
	}
}

func TestProcessToolResult_NotFound(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartIteration()
	ctrl.ProcessAssistantBatch([]Event{
		ToolUseEvent{ID: "t1", Name: "Read", Summary: "a.go"},
	})

	// Non-existent ID — should not panic
	group := ctrl.ProcessToolResult(ToolResultEvent{ToolUseID: "t999"})
	if group != nil {
		t.Error("expected nil group for unknown tool use ID")
	}
}

func TestProcessUsage_KnownModel(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.ProcessUsage(UsageEvent{
		Model:                    "claude-sonnet-4-5",
		InputTokens:              1000,
		OutputTokens:             500,
		CacheReadInputTokens:     200,
		CacheCreationInputTokens: 100,
	})

	if sess.InputTokens != 1000 {
		t.Errorf("expected InputTokens 1000, got %d", sess.InputTokens)
	}
	if sess.OutputTokens != 500 {
		t.Errorf("expected OutputTokens 500, got %d", sess.OutputTokens)
	}
	if sess.CacheReadTokens != 200 {
		t.Errorf("expected CacheReadTokens 200, got %d", sess.CacheReadTokens)
	}
	if sess.CacheCreationTokens != 100 {
		t.Errorf("expected CacheCreationTokens 100, got %d", sess.CacheCreationTokens)
	}
	if sess.LastInputTokens != 1000 {
		t.Errorf("expected LastInputTokens 1000, got %d", sess.LastInputTokens)
	}
	if sess.LastCacheReadTokens != 200 {
		t.Errorf("expected LastCacheReadTokens 200, got %d", sess.LastCacheReadTokens)
	}

	if !ctrl.HasKnownModel() {
		t.Error("expected HasKnownModel true")
	}
	if ctrl.LastModel() != "claude-sonnet-4-5" {
		t.Errorf("expected LastModel claude-sonnet-4-5, got %s", ctrl.LastModel())
	}

	// Verify cost: 1000*0.000003 + 500*0.000015 + 200*0.0000003 + 100*0.00000375
	// = 0.003 + 0.0075 + 0.00006 + 0.000375 = 0.010935
	expectedCost := 0.010935
	if diff := sess.TotalCost - expectedCost; diff > 0.000001 || diff < -0.000001 {
		t.Errorf("expected TotalCost ~%.6f, got %.6f", expectedCost, sess.TotalCost)
	}
}

func TestProcessUsage_UnknownModel(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.ProcessUsage(UsageEvent{
		Model:        "claude-unknown-9",
		InputTokens:  1000,
		OutputTokens: 500,
	})

	// Tokens still accumulated
	if sess.InputTokens != 1000 {
		t.Errorf("expected InputTokens 1000, got %d", sess.InputTokens)
	}
	if sess.OutputTokens != 500 {
		t.Errorf("expected OutputTokens 500, got %d", sess.OutputTokens)
	}

	// Cost should be 0 for unknown model
	if sess.TotalCost != 0 {
		t.Errorf("expected TotalCost 0, got %f", sess.TotalCost)
	}
	if ctrl.HasKnownModel() {
		t.Error("expected HasKnownModel false for unknown model")
	}
}

func TestProcessUsage_AccumulatesAcrossCalls(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.ProcessUsage(UsageEvent{
		Model:        "claude-sonnet-4-5",
		InputTokens:  1000,
		OutputTokens: 500,
	})
	ctrl.ProcessUsage(UsageEvent{
		Model:        "claude-sonnet-4-5",
		InputTokens:  2000,
		OutputTokens: 1000,
	})

	if sess.InputTokens != 3000 {
		t.Errorf("expected accumulated InputTokens 3000, got %d", sess.InputTokens)
	}
	if sess.OutputTokens != 1500 {
		t.Errorf("expected accumulated OutputTokens 1500, got %d", sess.OutputTokens)
	}
	// LastInputTokens should be from the most recent call
	if sess.LastInputTokens != 2000 {
		t.Errorf("expected LastInputTokens 2000 (latest), got %d", sess.LastInputTokens)
	}
}

func TestStartIteration(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartIteration()

	if len(sess.Iterations) != 1 {
		t.Fatalf("expected 1 iteration, got %d", len(sess.Iterations))
	}
	iter := sess.Iterations[0]
	if iter.Index != 0 {
		t.Errorf("expected Index 0, got %d", iter.Index)
	}
	if iter.Status != model.IterationRunning {
		t.Errorf("expected IterationRunning, got %d", iter.Status)
	}
	if !iter.StartTime.Equal(now) {
		t.Errorf("expected StartTime %v, got %v", now, iter.StartTime)
	}

	// Second iteration
	now = now.Add(5 * time.Minute)
	ctrl.StartIteration()

	if len(sess.Iterations) != 2 {
		t.Fatalf("expected 2 iterations, got %d", len(sess.Iterations))
	}
	if sess.Iterations[1].Index != 1 {
		t.Errorf("expected Index 1, got %d", sess.Iterations[1].Index)
	}
}

func TestCompleteIteration_Success(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartIteration()

	now = now.Add(30 * time.Second)
	ctrl.CompleteIteration(nil)

	iter := sess.Iterations[0]
	if iter.Status != model.IterationCompleted {
		t.Errorf("expected IterationCompleted, got %d", iter.Status)
	}
	if iter.Duration != 30*time.Second {
		t.Errorf("expected 30s duration, got %v", iter.Duration)
	}
}

func TestCompleteIteration_Failed(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartIteration()

	now = now.Add(10 * time.Second)
	ctrl.CompleteIteration(errTest)

	iter := sess.Iterations[0]
	if iter.Status != model.IterationFailed {
		t.Errorf("expected IterationFailed, got %d", iter.Status)
	}
	if iter.Duration != 10*time.Second {
		t.Errorf("expected 10s duration, got %v", iter.Duration)
	}
}

func TestCompleteIteration_NoRunning(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	// Should not panic when there's no running iteration
	ctrl.CompleteIteration(nil)

	if len(sess.Iterations) != 0 {
		t.Errorf("expected 0 iterations, got %d", len(sess.Iterations))
	}
}

func TestShouldStartNext(t *testing.T) {
	tests := []struct {
		name     string
		max      int
		count    int
		expected bool
	}{
		{"unlimited with 0", 0, 0, true},
		{"unlimited with 5", 0, 5, true},
		{"max 3 with 1", 3, 1, true},
		{"max 3 with 2", 3, 2, true},
		{"max 3 at limit", 3, 3, false},
		{"max 3 over limit", 3, 4, false},
		{"max 1 at limit", 1, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			sess := &model.Session{MaxIterations: tt.max}
			ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

			for range tt.count {
				sess.Iterations = append(sess.Iterations, model.Iteration{
					Index:  len(sess.Iterations),
					Status: model.IterationCompleted,
				})
			}

			got := ctrl.ShouldStartNext()
			if got != tt.expected {
				t.Errorf("ShouldStartNext() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRunningIterationIdx(t *testing.T) {
	tests := []struct {
		name     string
		statuses []model.IterationStatus
		expected int
	}{
		{"empty", nil, -1},
		{"all completed", []model.IterationStatus{model.IterationCompleted, model.IterationCompleted}, -1},
		{"first running", []model.IterationStatus{model.IterationRunning}, 0},
		{"second running", []model.IterationStatus{model.IterationCompleted, model.IterationRunning}, 1},
		{"failed + running", []model.IterationStatus{model.IterationFailed, model.IterationRunning}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			sess := &model.Session{}
			ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

			for i, status := range tt.statuses {
				sess.Iterations = append(sess.Iterations, model.Iteration{
					Index:  i,
					Status: status,
				})
			}

			got := ctrl.RunningIterationIdx()
			if got != tt.expected {
				t.Errorf("RunningIterationIdx() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestNewController_DefaultClock(t *testing.T) {
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), nil)

	if ctrl.Clock == nil {
		t.Fatal("expected Clock to be non-nil when passed nil")
	}

	// Verify it returns approximately now
	before := time.Now()
	got := ctrl.Clock()
	after := time.Now()
	if got.Before(before) || got.After(after) {
		t.Errorf("default Clock returned %v, expected between %v and %v", got, before, after)
	}
}

func TestFullLifecycle(t *testing.T) {
	// End-to-end test: start iteration, process batch with grouping,
	// process results, process usage, complete iteration, verify state.
	now := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	// Start iteration
	ctrl.StartIteration()
	if ctrl.RunningIterationIdx() != 0 {
		t.Fatal("expected running iteration at index 0")
	}

	// Process a batch with mixed content: 3 Reads (grouped) + text + 1 Write
	ctrl.ProcessAssistantBatch([]Event{
		ToolUseEvent{ID: "r1", Name: "Read", Summary: "a.go"},
		ToolUseEvent{ID: "r2", Name: "Read", Summary: "b.go"},
		ToolUseEvent{ID: "r3", Name: "Read", Summary: "c.go"},
		TextEvent{Text: "I'll now edit the file"},
		ToolUseEvent{ID: "w1", Name: "Write", Summary: "output.go", LineInfo: "(50 lines)"},
	})

	iter := &sess.Iterations[0]
	if len(iter.Items) != 3 {
		t.Fatalf("expected 3 items (group + text + standalone), got %d", len(iter.Items))
	}

	// Process usage
	ctrl.ProcessUsage(UsageEvent{
		Model:        "claude-sonnet-4-5",
		InputTokens:  5000,
		OutputTokens: 2000,
	})

	// Complete read results
	now = now.Add(100 * time.Millisecond)
	for _, id := range []string{"r1", "r2", "r3"} {
		group := ctrl.ProcessToolResult(ToolResultEvent{ToolUseID: id})
		if group == nil {
			t.Fatalf("expected group for child %s", id)
		}
	}

	// Check group status after all children complete
	readGroup := iter.Items[0].(*model.ToolCallGroup)
	if readGroup.Status() != model.ToolCallDone {
		t.Error("expected read group to be done")
	}

	// Complete write
	now = now.Add(200 * time.Millisecond)
	ctrl.ProcessToolResult(ToolResultEvent{ToolUseID: "w1"})

	writeTc := iter.Items[2].(*model.ToolCall)
	if writeTc.Status != model.ToolCallDone {
		t.Error("expected write to be done")
	}

	// Complete iteration
	now = now.Add(time.Second)
	ctrl.CompleteIteration(nil)

	if iter.Status != model.IterationCompleted {
		t.Error("expected iteration completed")
	}
	if ctrl.RunningIterationIdx() != -1 {
		t.Error("expected no running iteration")
	}
	if !ctrl.HasKnownModel() {
		t.Error("expected HasKnownModel true")
	}
	if sess.TotalCost == 0 {
		t.Error("expected non-zero cost")
	}
}

func TestTokenAttribution_SingleToolCall(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartIteration()

	// Usage arrives before the batch (as per executor ordering).
	ctrl.ProcessUsage(UsageEvent{
		Model:                "claude-sonnet-4-5",
		InputTokens:          1000,
		CacheReadInputTokens: 500,
	})

	ctrl.ProcessAssistantBatch([]Event{
		ToolUseEvent{ID: "t1", Name: "Read", Summary: "a.go"},
	})

	tc := sess.Iterations[0].Items[0].(*model.ToolCall)
	if tc.InputTokens != 1000 {
		t.Errorf("expected InputTokens 1000, got %d", tc.InputTokens)
	}
	if tc.CacheReadTokens != 500 {
		t.Errorf("expected CacheReadTokens 500, got %d", tc.CacheReadTokens)
	}
}

func TestTokenAttribution_DividesEvenly(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartIteration()

	ctrl.ProcessUsage(UsageEvent{
		Model:                "claude-sonnet-4-5",
		InputTokens:          900,
		CacheReadInputTokens: 600,
	})

	ctrl.ProcessAssistantBatch([]Event{
		ToolUseEvent{ID: "t1", Name: "Read", Summary: "a.go"},
		ToolUseEvent{ID: "t2", Name: "Write", Summary: "b.go"},
		ToolUseEvent{ID: "t3", Name: "Bash", Summary: "run tests"},
	})

	// 900 / 3 = 300, 600 / 3 = 200
	for i, item := range sess.Iterations[0].Items {
		tc := item.(*model.ToolCall)
		if tc.InputTokens != 300 {
			t.Errorf("tool call %d: expected InputTokens 300, got %d", i, tc.InputTokens)
		}
		if tc.CacheReadTokens != 200 {
			t.Errorf("tool call %d: expected CacheReadTokens 200, got %d", i, tc.CacheReadTokens)
		}
	}
}

func TestTokenAttribution_DividesWithRounding(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartIteration()

	ctrl.ProcessUsage(UsageEvent{
		Model:                "claude-sonnet-4-5",
		InputTokens:          1000,
		CacheReadInputTokens: 100,
	})

	ctrl.ProcessAssistantBatch([]Event{
		ToolUseEvent{ID: "t1", Name: "Read", Summary: "a.go"},
		ToolUseEvent{ID: "t2", Name: "Read", Summary: "b.go"},
		ToolUseEvent{ID: "t3", Name: "Read", Summary: "c.go"},
	})

	// 1000 / 3 = 333 (rounded: (1000+1)/3 = 333), 100 / 3 = 33 (rounded: (100+1)/3 = 33)
	group := sess.Iterations[0].Items[0].(*model.ToolCallGroup)
	for i, child := range group.Children {
		if child.InputTokens != 333 {
			t.Errorf("child %d: expected InputTokens 333, got %d", i, child.InputTokens)
		}
		if child.CacheReadTokens != 33 {
			t.Errorf("child %d: expected CacheReadTokens 33, got %d", i, child.CacheReadTokens)
		}
	}
}

func TestTokenAttribution_GroupChildren(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartIteration()

	ctrl.ProcessUsage(UsageEvent{
		Model:                "claude-sonnet-4-5",
		InputTokens:          400,
		CacheReadInputTokens: 200,
	})

	// 2 consecutive Reads → grouped, total 2 tool calls
	ctrl.ProcessAssistantBatch([]Event{
		ToolUseEvent{ID: "t1", Name: "Read", Summary: "a.go"},
		ToolUseEvent{ID: "t2", Name: "Read", Summary: "b.go"},
	})

	group := sess.Iterations[0].Items[0].(*model.ToolCallGroup)
	for i, child := range group.Children {
		if child.InputTokens != 200 {
			t.Errorf("child %d: expected InputTokens 200, got %d", i, child.InputTokens)
		}
		if child.CacheReadTokens != 100 {
			t.Errorf("child %d: expected CacheReadTokens 100, got %d", i, child.CacheReadTokens)
		}
	}
}

func TestTokenAttribution_PendingClearedAfterBatch(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartIteration()

	// First turn: usage → batch
	ctrl.ProcessUsage(UsageEvent{
		Model:                "claude-sonnet-4-5",
		InputTokens:          1000,
		CacheReadInputTokens: 500,
	})
	ctrl.ProcessAssistantBatch([]Event{
		ToolUseEvent{ID: "t1", Name: "Read", Summary: "a.go"},
	})

	// Second turn: no usage event before batch → tokens should be 0
	ctrl.ProcessAssistantBatch([]Event{
		ToolUseEvent{ID: "t2", Name: "Write", Summary: "b.go"},
	})

	tc2 := sess.Iterations[0].Items[1].(*model.ToolCall)
	if tc2.InputTokens != 0 {
		t.Errorf("expected InputTokens 0 for second batch (no usage), got %d", tc2.InputTokens)
	}
	if tc2.CacheReadTokens != 0 {
		t.Errorf("expected CacheReadTokens 0 for second batch (no usage), got %d", tc2.CacheReadTokens)
	}
}

func TestTokenAttribution_TextOnlyBatchPreservesTokens(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartIteration()

	ctrl.ProcessUsage(UsageEvent{
		Model:                "claude-sonnet-4-5",
		InputTokens:          500,
		CacheReadInputTokens: 200,
	})

	// Text-only batch has no tool calls; pending tokens are cleared
	ctrl.ProcessAssistantBatch([]Event{
		TextEvent{Text: "thinking..."},
	})

	// Verify pending tokens were still cleared (no tool calls to attribute to)
	// A subsequent batch without usage should have 0 tokens
	ctrl.ProcessAssistantBatch([]Event{
		ToolUseEvent{ID: "t1", Name: "Read", Summary: "a.go"},
	})

	tc := sess.Iterations[0].Items[1].(*model.ToolCall)
	if tc.InputTokens != 0 {
		t.Errorf("expected InputTokens 0 after text-only batch cleared pending, got %d", tc.InputTokens)
	}
}

func TestPhase(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	// Default phase is Idle
	if ctrl.Phase() != model.PhaseIdle {
		t.Errorf("expected PhaseIdle, got %d", ctrl.Phase())
	}

	// Start a run → Running
	ctrl.StartRun("BUILD", "PROMPT_BUILD.md", 3)
	if ctrl.Phase() != model.PhaseRunning {
		t.Errorf("expected PhaseRunning after StartRun, got %d", ctrl.Phase())
	}
}

func TestStartRun(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartRun("BUILD", "PROMPT_BUILD.md", 5)

	if len(sess.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(sess.Runs))
	}
	run := sess.Runs[0]
	if run.PromptName != "BUILD" {
		t.Errorf("expected PromptName BUILD, got %s", run.PromptName)
	}
	if run.PromptFile != "PROMPT_BUILD.md" {
		t.Errorf("expected PromptFile PROMPT_BUILD.md, got %s", run.PromptFile)
	}
	if run.StartIndex != 0 {
		t.Errorf("expected StartIndex 0, got %d", run.StartIndex)
	}
	if run.MaxIterations != 5 {
		t.Errorf("expected MaxIterations 5, got %d", run.MaxIterations)
	}
	if sess.Phase != model.PhaseRunning {
		t.Errorf("expected PhaseRunning, got %d", sess.Phase)
	}
	if sess.PromptFile != "PROMPT_BUILD.md" {
		t.Errorf("expected session PromptFile updated, got %s", sess.PromptFile)
	}
	if !sess.StartTime.Equal(now) {
		t.Errorf("expected StartTime %v, got %v", now, sess.StartTime)
	}
}

func TestStartRun_MultipleRuns(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	// First run with 2 iterations
	ctrl.StartRun("BUILD", "PROMPT_BUILD.md", 2)
	ctrl.StartIteration()
	now = now.Add(time.Minute)
	ctrl.CompleteIteration(nil)
	ctrl.StartIteration()
	now = now.Add(time.Minute)
	ctrl.CompleteIteration(nil)

	// Phase should be Finished after run completes
	if sess.Phase != model.PhaseFinished {
		t.Errorf("expected PhaseFinished after first run, got %d", sess.Phase)
	}

	// Second run starts at iteration index 2
	now = now.Add(time.Minute)
	ctrl.StartRun("PLAN", "PROMPT_PLAN.md", 3)

	if len(sess.Runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(sess.Runs))
	}
	if sess.Runs[1].StartIndex != 2 {
		t.Errorf("expected second run StartIndex 2, got %d", sess.Runs[1].StartIndex)
	}
	if sess.Runs[1].PromptName != "PLAN" {
		t.Errorf("expected second run PromptName PLAN, got %s", sess.Runs[1].PromptName)
	}
	if sess.Phase != model.PhaseRunning {
		t.Errorf("expected PhaseRunning after second StartRun, got %d", sess.Phase)
	}
}

func TestShouldStartNext_PerRunLimits(t *testing.T) {
	tests := []struct {
		name          string
		maxIterations int
		runIterations int
		expected      bool
	}{
		{"unlimited run with 0", 0, 0, true},
		{"unlimited run with 5", 0, 5, true},
		{"max 3 run with 1", 3, 1, true},
		{"max 3 run at limit", 3, 3, false},
		{"max 1 run at limit", 1, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			sess := &model.Session{}
			ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

			// Start a run
			ctrl.StartRun("BUILD", "PROMPT_BUILD.md", tt.maxIterations)

			// Add iterations to the run
			for range tt.runIterations {
				sess.Iterations = append(sess.Iterations, model.Iteration{
					Index:  len(sess.Iterations),
					Status: model.IterationCompleted,
				})
			}

			got := ctrl.ShouldStartNext()
			if got != tt.expected {
				t.Errorf("ShouldStartNext() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestShouldStartNext_MultiRunOffset(t *testing.T) {
	// Second run should count from its own start index
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	// First run: 3 iterations
	ctrl.StartRun("BUILD", "PROMPT_BUILD.md", 3)
	for range 3 {
		ctrl.StartIteration()
		now = now.Add(time.Minute)
		ctrl.CompleteIteration(nil)
	}

	// Second run: max 2 iterations, starting at index 3
	now = now.Add(time.Minute)
	ctrl.StartRun("PLAN", "PROMPT_PLAN.md", 2)

	// No iterations yet in second run
	if !ctrl.ShouldStartNext() {
		t.Error("expected ShouldStartNext true with 0 iterations in second run")
	}

	// Add 1 iteration to second run
	ctrl.StartIteration()
	now = now.Add(time.Minute)
	ctrl.CompleteIteration(nil)

	// Should still allow one more
	// Phase was set to Finished by CompleteIteration, need to re-enter Running
	sess.Phase = model.PhaseRunning
	if !ctrl.ShouldStartNext() {
		t.Error("expected ShouldStartNext true with 1 iteration in second run (max 2)")
	}

	// Add second iteration
	ctrl.StartIteration()
	now = now.Add(time.Minute)
	ctrl.CompleteIteration(nil)

	// Now should be done (phase already Finished from CompleteIteration)
	if ctrl.ShouldStartNext() {
		t.Error("expected ShouldStartNext false with 2 iterations in second run (max 2)")
	}
}

func TestCompleteIteration_TransitionsToFinished(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartRun("BUILD", "PROMPT_BUILD.md", 1)
	ctrl.StartIteration()
	now = now.Add(30 * time.Second)
	ctrl.CompleteIteration(nil)

	if sess.Phase != model.PhaseFinished {
		t.Errorf("expected PhaseFinished when run complete, got %d", sess.Phase)
	}
	if sess.AccumulatedDuration != 30*time.Second {
		t.Errorf("expected AccumulatedDuration 30s, got %v", sess.AccumulatedDuration)
	}
}

func TestCompleteIteration_FailureTransitionsToFinished(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartRun("BUILD", "PROMPT_BUILD.md", 5)
	ctrl.StartIteration()
	now = now.Add(10 * time.Second)
	ctrl.CompleteIteration(errTest)

	// Even with iterations remaining, failure transitions to Finished
	if sess.Phase != model.PhaseFinished {
		t.Errorf("expected PhaseFinished on failure, got %d", sess.Phase)
	}
}

func TestCompleteIteration_StaysRunningWhenMoreIterations(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	ctrl.StartRun("BUILD", "PROMPT_BUILD.md", 3)
	ctrl.StartIteration()
	now = now.Add(10 * time.Second)
	ctrl.CompleteIteration(nil)

	// Still has 2 more iterations to go, should stay Running
	if sess.Phase != model.PhaseRunning {
		t.Errorf("expected PhaseRunning when more iterations remain, got %d", sess.Phase)
	}
}

func TestAccumulatedDuration_PauseResume(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sess := &model.Session{}
	ctrl := NewController(sess, defaultTestConfig(), fakeClock(&now))

	// First run: 1 iteration, 30s
	ctrl.StartRun("BUILD", "PROMPT_BUILD.md", 1)
	ctrl.StartIteration()
	now = now.Add(30 * time.Second)
	ctrl.CompleteIteration(nil)

	if sess.AccumulatedDuration != 30*time.Second {
		t.Errorf("expected 30s accumulated, got %v", sess.AccumulatedDuration)
	}

	// Pause for a while (user browsing results)
	now = now.Add(5 * time.Minute)

	// Second run: 1 iteration, 20s
	ctrl.StartRun("PLAN", "PROMPT_PLAN.md", 1)
	ctrl.StartIteration()
	now = now.Add(20 * time.Second)
	ctrl.CompleteIteration(nil)

	// Should be 30s + 20s = 50s (not including the 5 min pause)
	if sess.AccumulatedDuration != 50*time.Second {
		t.Errorf("expected 50s accumulated, got %v", sess.AccumulatedDuration)
	}
}

// errTest is a sentinel error for testing.
var errTest = &testError{}

type testError struct{}

func (e *testError) Error() string { return "test error" }
