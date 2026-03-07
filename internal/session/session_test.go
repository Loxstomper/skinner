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

// errTest is a sentinel error for testing.
var errTest = &testError{}

type testError struct{}

func (e *testError) Error() string { return "test error" }
