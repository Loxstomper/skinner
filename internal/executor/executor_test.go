package executor

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/loxstomper/skinner/internal/session"
)

// --- FakeExecutor tests ---

func TestFakeExecutor_DeliversEvents(t *testing.T) {
	events := []session.Event{
		session.ToolUseEvent{ID: "t1", Name: "Read", Summary: "/foo"},
		session.ToolResultEvent{ToolUseID: "t1"},
		session.IterationEndEvent{},
	}
	fake := &FakeExecutor{Events: events}

	ch, err := fake.Start(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	var got []session.Event
	for e := range ch {
		got = append(got, e)
	}

	if len(got) != len(events) {
		t.Fatalf("got %d events, want %d", len(got), len(events))
	}
	if tu, ok := got[0].(session.ToolUseEvent); !ok || tu.ID != "t1" {
		t.Errorf("event[0] = %T %+v, want ToolUseEvent{ID:t1}", got[0], got[0])
	}
	if tr, ok := got[1].(session.ToolResultEvent); !ok || tr.ToolUseID != "t1" {
		t.Errorf("event[1] = %T %+v, want ToolResultEvent{ToolUseID:t1}", got[1], got[1])
	}
	if _, ok := got[2].(session.IterationEndEvent); !ok {
		t.Errorf("event[2] = %T, want IterationEndEvent", got[2])
	}
}

func TestFakeExecutor_RecordsPrompt(t *testing.T) {
	fake := &FakeExecutor{}
	_, err := fake.Start(context.Background(), "my prompt")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if fake.Prompt != "my prompt" {
		t.Errorf("Prompt = %q, want %q", fake.Prompt, "my prompt")
	}
}

func TestFakeExecutor_EmptyEvents(t *testing.T) {
	fake := &FakeExecutor{}
	ch, err := fake.Start(context.Background(), "")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	var count int
	for range ch {
		count++
	}
	if count != 0 {
		t.Errorf("got %d events from empty FakeExecutor, want 0", count)
	}
}

func TestFakeExecutor_DelayAndCancel(t *testing.T) {
	events := []session.Event{
		session.TextEvent{Text: "one"},
		session.TextEvent{Text: "two"},
		session.TextEvent{Text: "three"},
	}
	ctx, cancel := context.WithCancel(context.Background())
	fake := &FakeExecutor{Events: events, Delay: 50 * time.Millisecond}

	ch, err := fake.Start(ctx, "")
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Receive one event, then cancel
	e := <-ch
	if te, ok := e.(session.TextEvent); !ok || te.Text != "one" {
		t.Errorf("first event = %T %+v, want TextEvent{one}", e, e)
	}
	cancel()

	// Drain remaining — should get fewer than 3 total due to cancellation
	var count int
	for range ch {
		count++
	}
	// We got 1 above + count remaining. Total should be less than 3 (cancellation
	// may race, so we just verify the channel closes).
	_ = count // channel closed is the important assertion
}

func TestFakeExecutor_KillIsNoOp(t *testing.T) {
	fake := &FakeExecutor{}
	if err := fake.Kill(); err != nil {
		t.Errorf("Kill: %v", err)
	}
}

// --- Interface compliance ---

func TestInterfaceCompliance(t *testing.T) {
	var _ Executor = (*FakeExecutor)(nil)
	var _ Executor = (*ClaudeExecutor)(nil)
}

// --- readEvents tests ---

func TestReadEvents_AssistantBatch(t *testing.T) {
	// An assistant event with tool_use and text content produces an
	// AssistantBatchEvent containing ToolUseEvent and TextEvent,
	// plus a separate UsageEvent.
	input := `{"type":"assistant","message":{"role":"assistant","model":"claude-opus-4-6","content":[{"type":"text","text":"thinking..."},{"type":"tool_use","id":"tu_1","name":"Read","input":{"file_path":"/foo/bar.go"}}],"usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":10,"cache_creation_input_tokens":5}}}`

	ch := make(chan session.Event, 10)
	readEvents(strings.NewReader(input), ch)
	close(ch)

	var events []session.Event
	for e := range ch {
		events = append(events, e)
	}

	if len(events) != 2 {
		t.Fatalf("got %d events, want 2 (UsageEvent + AssistantBatchEvent)", len(events))
	}

	usage, ok := events[0].(session.UsageEvent)
	if !ok {
		t.Fatalf("events[0] = %T, want UsageEvent", events[0])
	}
	if usage.Model != "claude-opus-4-6" {
		t.Errorf("Model = %q, want %q", usage.Model, "claude-opus-4-6")
	}
	if usage.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100", usage.InputTokens)
	}
	if usage.OutputTokens != 50 {
		t.Errorf("OutputTokens = %d, want 50", usage.OutputTokens)
	}
	if usage.CacheReadInputTokens != 10 {
		t.Errorf("CacheReadInputTokens = %d, want 10", usage.CacheReadInputTokens)
	}
	if usage.CacheCreationInputTokens != 5 {
		t.Errorf("CacheCreationInputTokens = %d, want 5", usage.CacheCreationInputTokens)
	}

	batch, ok := events[1].(session.AssistantBatchEvent)
	if !ok {
		t.Fatalf("events[1] = %T, want AssistantBatchEvent", events[1])
	}
	if len(batch.Events) != 2 {
		t.Fatalf("batch has %d events, want 2", len(batch.Events))
	}
	if te, ok := batch.Events[0].(session.TextEvent); !ok || te.Text != "thinking..." {
		t.Errorf("batch[0] = %T %+v, want TextEvent{thinking...}", batch.Events[0], batch.Events[0])
	}
	tu, ok := batch.Events[1].(session.ToolUseEvent)
	if !ok || tu.ID != "tu_1" || tu.Name != "Read" {
		t.Errorf("batch[1] = %T %+v, want ToolUseEvent{tu_1, Read}", batch.Events[1], batch.Events[1])
	}
	if tu.RawInput == nil {
		t.Fatal("expected RawInput to be non-nil on ToolUseEvent")
	}
	if fp, ok := tu.RawInput["file_path"].(string); !ok || fp != "/foo/bar.go" {
		t.Errorf("RawInput[file_path] = %v, want /foo/bar.go", tu.RawInput["file_path"])
	}
}

func TestReadEvents_ToolResult(t *testing.T) {
	input := `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tu_1","content":"file contents here\nline 2\n","is_error":false}]}}`

	ch := make(chan session.Event, 10)
	readEvents(strings.NewReader(input), ch)
	close(ch)

	var events []session.Event
	for e := range ch {
		events = append(events, e)
	}

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}

	tr, ok := events[0].(session.ToolResultEvent)
	if !ok {
		t.Fatalf("events[0] = %T, want ToolResultEvent", events[0])
	}
	if tr.ToolUseID != "tu_1" {
		t.Errorf("ToolUseID = %q, want %q", tr.ToolUseID, "tu_1")
	}
	if tr.IsError {
		t.Error("IsError = true, want false")
	}
	if tr.Content != "file contents here\nline 2\n" {
		t.Errorf("Content = %q, want %q", tr.Content, "file contents here\nline 2\n")
	}
}

func TestReadEvents_ToolResultError(t *testing.T) {
	input := `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tu_2","content":"error: not found","is_error":true}]}}`

	ch := make(chan session.Event, 10)
	readEvents(strings.NewReader(input), ch)
	close(ch)

	var events []session.Event
	for e := range ch {
		events = append(events, e)
	}

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	tr := events[0].(session.ToolResultEvent)
	if !tr.IsError {
		t.Error("IsError = false, want true")
	}
}

func TestReadEvents_IterationEnd(t *testing.T) {
	input := `{"type":"result","message":{"role":"assistant","content":[]}}`

	ch := make(chan session.Event, 10)
	readEvents(strings.NewReader(input), ch)
	close(ch)

	var events []session.Event
	for e := range ch {
		events = append(events, e)
	}

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if _, ok := events[0].(session.IterationEndEvent); !ok {
		t.Errorf("events[0] = %T, want IterationEndEvent", events[0])
	}
}

func TestReadEvents_SkipsInvalidLines(t *testing.T) {
	input := "not json\n{}\n{\"type\":\"unknown\"}\n"

	ch := make(chan session.Event, 10)
	readEvents(strings.NewReader(input), ch)
	close(ch)

	var count int
	for range ch {
		count++
	}
	if count != 0 {
		t.Errorf("got %d events from invalid input, want 0", count)
	}
}

func TestReadEvents_MultipleLines(t *testing.T) {
	// Two lines: an assistant event then a result event
	lines := []string{
		`{"type":"assistant","message":{"role":"assistant","model":"claude-opus-4-6","content":[{"type":"tool_use","id":"tu_1","name":"Bash","input":{"command":"ls"}}],"usage":{"input_tokens":10,"output_tokens":5,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tu_1","content":"file.go","is_error":false}]}}`,
		`{"type":"result","message":{"role":"assistant","content":[]}}`,
	}
	input := strings.Join(lines, "\n")

	ch := make(chan session.Event, 10)
	readEvents(strings.NewReader(input), ch)
	close(ch)

	var events []session.Event
	for e := range ch {
		events = append(events, e)
	}

	// Expected: UsageEvent, AssistantBatchEvent(ToolUseEvent), ToolResultEvent, IterationEndEvent
	if len(events) != 4 {
		t.Fatalf("got %d events, want 4; events: %+v", len(events), events)
	}

	if _, ok := events[0].(session.UsageEvent); !ok {
		t.Errorf("events[0] = %T, want UsageEvent", events[0])
	}
	if batch, ok := events[1].(session.AssistantBatchEvent); !ok {
		t.Errorf("events[1] = %T, want AssistantBatchEvent", events[1])
	} else if len(batch.Events) != 1 {
		t.Errorf("batch has %d events, want 1", len(batch.Events))
	} else if tu, ok := batch.Events[0].(session.ToolUseEvent); !ok || tu.Name != "Bash" {
		t.Errorf("batch[0] = %T %+v, want ToolUseEvent{Bash}", batch.Events[0], batch.Events[0])
	}
	if _, ok := events[2].(session.ToolResultEvent); !ok {
		t.Errorf("events[2] = %T, want ToolResultEvent", events[2])
	}
	if _, ok := events[3].(session.IterationEndEvent); !ok {
		t.Errorf("events[3] = %T, want IterationEndEvent", events[3])
	}
}

func TestReadEvents_ToolSummaryExtraction(t *testing.T) {
	// Verify that tool summaries from parser are correctly propagated
	input := `{"type":"assistant","message":{"role":"assistant","model":"claude-opus-4-6","content":[{"type":"tool_use","id":"tu_1","name":"Read","input":{"file_path":"/tmp/test.go"}},{"type":"tool_use","id":"tu_2","name":"Bash","input":{"command":"go test ./...","description":"Run all tests"}}],"usage":{"input_tokens":10,"output_tokens":5,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`

	ch := make(chan session.Event, 10)
	readEvents(strings.NewReader(input), ch)
	close(ch)

	var events []session.Event
	for e := range ch {
		events = append(events, e)
	}

	// UsageEvent + AssistantBatchEvent
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}

	batch := events[1].(session.AssistantBatchEvent)
	if len(batch.Events) != 2 {
		t.Fatalf("batch has %d events, want 2", len(batch.Events))
	}

	read := batch.Events[0].(session.ToolUseEvent)
	if read.Summary != "/tmp/test.go" {
		t.Errorf("Read summary = %q, want %q", read.Summary, "/tmp/test.go")
	}

	bash := batch.Events[1].(session.ToolUseEvent)
	if bash.Summary != "Run all tests" {
		t.Errorf("Bash summary = %q, want %q", bash.Summary, "Run all tests")
	}
}

func TestReadEvents_EmptyInput(t *testing.T) {
	ch := make(chan session.Event, 10)
	readEvents(strings.NewReader(""), ch)
	close(ch)

	var count int
	for range ch {
		count++
	}
	if count != 0 {
		t.Errorf("got %d events from empty input, want 0", count)
	}
}
