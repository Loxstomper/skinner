package parser

import (
	"encoding/json"
	"testing"
)

// helper to build a stream-json line
func makeStreamJSON(t *testing.T, eventType string, message interface{}) string {
	t.Helper()
	msgBytes, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("failed to marshal message: %v", err)
	}
	evt := map[string]interface{}{
		"type":    eventType,
		"message": json.RawMessage(msgBytes),
	}
	line, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}
	return string(line)
}

func TestParseStreamEvent_EmptyAndInvalid(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"invalid json", "not json at all"},
		{"malformed json", `{"type": `},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events, err := ParseStreamEvent(tt.line)
			if err != nil {
				t.Errorf("expected nil error, got %v", err)
			}
			if events != nil {
				t.Errorf("expected nil events, got %v", events)
			}
		})
	}
}

func TestParseStreamEvent_UnknownType(t *testing.T) {
	line := `{"type": "unknown", "message": {}}`
	events, err := ParseStreamEvent(line)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if events != nil {
		t.Errorf("expected nil events for unknown type, got %v", events)
	}
}

func TestParseStreamEvent_ResultType(t *testing.T) {
	line := `{"type": "result", "message": {}}`
	events, err := ParseStreamEvent(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if _, ok := events[0].(IterationEndEvent); !ok {
		t.Errorf("expected IterationEndEvent, got %T", events[0])
	}
}

func TestParseStreamEvent_AssistantToolUse(t *testing.T) {
	msg := map[string]interface{}{
		"role":  "assistant",
		"model": "claude-sonnet-4-5-20250929",
		"content": []map[string]interface{}{
			{
				"type":  "tool_use",
				"id":    "toolu_abc123",
				"name":  "Read",
				"input": map[string]interface{}{"file_path": "/tmp/test.go"},
			},
		},
		"usage": map[string]interface{}{
			"input_tokens":                1000,
			"output_tokens":               200,
			"cache_read_input_tokens":     500,
			"cache_creation_input_tokens": 100,
		},
	}
	line := makeStreamJSON(t, "assistant", msg)

	events, err := ParseStreamEvent(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should produce UsageEvent + ToolUseEvent
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d: %v", len(events), events)
	}

	// First event: UsageEvent
	usage, ok := events[0].(UsageEvent)
	if !ok {
		t.Fatalf("expected UsageEvent, got %T", events[0])
	}
	if usage.Model != "claude-sonnet-4-5-20250929" {
		t.Errorf("usage.Model = %q, want %q", usage.Model, "claude-sonnet-4-5-20250929")
	}
	if usage.InputTokens != 1000 {
		t.Errorf("usage.InputTokens = %d, want 1000", usage.InputTokens)
	}
	if usage.OutputTokens != 200 {
		t.Errorf("usage.OutputTokens = %d, want 200", usage.OutputTokens)
	}
	if usage.CacheReadInputTokens != 500 {
		t.Errorf("usage.CacheReadInputTokens = %d, want 500", usage.CacheReadInputTokens)
	}
	if usage.CacheCreationInputTokens != 100 {
		t.Errorf("usage.CacheCreationInputTokens = %d, want 100", usage.CacheCreationInputTokens)
	}

	// Second event: ToolUseEvent
	toolUse, ok := events[1].(ToolUseEvent)
	if !ok {
		t.Fatalf("expected ToolUseEvent, got %T", events[1])
	}
	if toolUse.ID != "toolu_abc123" {
		t.Errorf("toolUse.ID = %q, want %q", toolUse.ID, "toolu_abc123")
	}
	if toolUse.Name != "Read" {
		t.Errorf("toolUse.Name = %q, want %q", toolUse.Name, "Read")
	}
	if toolUse.Summary != "/tmp/test.go" {
		t.Errorf("toolUse.Summary = %q, want %q", toolUse.Summary, "/tmp/test.go")
	}
}

func TestParseStreamEvent_AssistantText(t *testing.T) {
	msg := map[string]interface{}{
		"role": "assistant",
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": "Here is some output",
			},
		},
	}
	line := makeStreamJSON(t, "assistant", msg)

	events, err := ParseStreamEvent(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	textEvt, ok := events[0].(TextEvent)
	if !ok {
		t.Fatalf("expected TextEvent, got %T", events[0])
	}
	if textEvt.Text != "Here is some output" {
		t.Errorf("text = %q, want %q", textEvt.Text, "Here is some output")
	}
}

func TestParseStreamEvent_AssistantEmptyText(t *testing.T) {
	msg := map[string]interface{}{
		"role": "assistant",
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": "",
			},
		},
	}
	line := makeStreamJSON(t, "assistant", msg)

	events, err := ParseStreamEvent(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty text blocks should be skipped
	if len(events) != 0 {
		t.Errorf("expected 0 events for empty text, got %d", len(events))
	}
}

func TestParseStreamEvent_AssistantMixedContent(t *testing.T) {
	msg := map[string]interface{}{
		"role":  "assistant",
		"model": "claude-opus-4-6",
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": "Let me read that file.",
			},
			{
				"type":  "tool_use",
				"id":    "toolu_001",
				"name":  "Read",
				"input": map[string]interface{}{"file_path": "/src/main.go"},
			},
		},
		"usage": map[string]interface{}{
			"input_tokens":  500,
			"output_tokens": 100,
		},
	}
	line := makeStreamJSON(t, "assistant", msg)

	events, err := ParseStreamEvent(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// UsageEvent + TextEvent + ToolUseEvent = 3
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d: %v", len(events), events)
	}

	if _, ok := events[0].(UsageEvent); !ok {
		t.Errorf("events[0]: expected UsageEvent, got %T", events[0])
	}
	if _, ok := events[1].(TextEvent); !ok {
		t.Errorf("events[1]: expected TextEvent, got %T", events[1])
	}
	if _, ok := events[2].(ToolUseEvent); !ok {
		t.Errorf("events[2]: expected ToolUseEvent, got %T", events[2])
	}
}

func TestParseStreamEvent_UserToolResult(t *testing.T) {
	msg := map[string]interface{}{
		"role": "user",
		"content": []map[string]interface{}{
			{
				"type":        "tool_result",
				"tool_use_id": "toolu_abc123",
				"is_error":    false,
				"content":     "line1\nline2\nline3",
			},
		},
	}
	line := makeStreamJSON(t, "user", msg)

	events, err := ParseStreamEvent(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	result, ok := events[0].(ToolResultEvent)
	if !ok {
		t.Fatalf("expected ToolResultEvent, got %T", events[0])
	}
	if result.ToolUseID != "toolu_abc123" {
		t.Errorf("ToolUseID = %q, want %q", result.ToolUseID, "toolu_abc123")
	}
	if result.IsError {
		t.Error("IsError = true, want false")
	}
	if result.LineInfo != "(3 lines)" {
		t.Errorf("LineInfo = %q, want %q", result.LineInfo, "(3 lines)")
	}
}

func TestParseStreamEvent_UserToolResultError(t *testing.T) {
	msg := map[string]interface{}{
		"role": "user",
		"content": []map[string]interface{}{
			{
				"type":        "tool_result",
				"tool_use_id": "toolu_err",
				"is_error":    true,
			},
		},
	}
	line := makeStreamJSON(t, "user", msg)

	events, err := ParseStreamEvent(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	result := events[0].(ToolResultEvent)
	if !result.IsError {
		t.Error("IsError = false, want true")
	}
}

func TestExtractToolSummary(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		input    map[string]interface{}
		want     string
	}{
		{
			name:     "Read file_path",
			toolName: "Read",
			input:    map[string]interface{}{"file_path": "/src/main.go"},
			want:     "/src/main.go",
		},
		{
			name:     "Edit file_path",
			toolName: "Edit",
			input:    map[string]interface{}{"file_path": "/src/utils.go", "old_string": "foo", "new_string": "bar"},
			want:     "/src/utils.go",
		},
		{
			name:     "Write file_path",
			toolName: "Write",
			input:    map[string]interface{}{"file_path": "/src/new.go", "content": "package main"},
			want:     "/src/new.go",
		},
		{
			name:     "Bash with description",
			toolName: "Bash",
			input:    map[string]interface{}{"command": "ls -la", "description": "List files"},
			want:     "List files",
		},
		{
			name:     "Bash without description falls back to command",
			toolName: "Bash",
			input:    map[string]interface{}{"command": "ls -la"},
			want:     "ls -la",
		},
		{
			name:     "Bash truncates long command",
			toolName: "Bash",
			input:    map[string]interface{}{"command": "echo " + string(make([]byte, 200))},
			want:     "echo " + string(make([]byte, 72)) + "...",
		},
		{
			name:     "Grep with pattern and path",
			toolName: "Grep",
			input:    map[string]interface{}{"pattern": "func main", "path": "/src"},
			want:     `"func main" in /src`,
		},
		{
			name:     "Grep with pattern only",
			toolName: "Grep",
			input:    map[string]interface{}{"pattern": "TODO"},
			want:     `"TODO"`,
		},
		{
			name:     "Glob pattern",
			toolName: "Glob",
			input:    map[string]interface{}{"pattern": "**/*.go"},
			want:     "**/*.go",
		},
		{
			name:     "Task description",
			toolName: "Task",
			input:    map[string]interface{}{"description": "Run tests"},
			want:     "Run tests",
		},
		{
			name:     "unknown tool returns empty",
			toolName: "UnknownTool",
			input:    map[string]interface{}{"foo": "bar"},
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputBytes, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("failed to marshal input: %v", err)
			}
			got := extractToolSummary(tt.toolName, inputBytes)
			if got != tt.want {
				t.Errorf("extractToolSummary(%q) = %q, want %q", tt.toolName, got, tt.want)
			}
		})
	}
}

func TestExtractToolUseLineInfo(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		input    map[string]interface{}
		want     string
	}{
		{
			name:     "non-Edit/Write tool returns empty",
			toolName: "Read",
			input:    map[string]interface{}{},
			want:     "",
		},
		{
			name:     "Edit adds lines",
			toolName: "Edit",
			input: map[string]interface{}{
				"old_string": "line1",
				"new_string": "line1\nline2\nline3",
			},
			want: "(+2)",
		},
		{
			name:     "Edit removes lines",
			toolName: "Edit",
			input: map[string]interface{}{
				"old_string": "line1\nline2\nline3",
				"new_string": "line1",
			},
			want: "(-2)",
		},
		{
			name:     "Edit net zero with changes",
			toolName: "Edit",
			input: map[string]interface{}{
				"old_string": "line1\nline2",
				"new_string": "new1\nnew2",
			},
			// Each has 1 newline, so oldLines=1, newLines=1, net zero → (+1/-1)
			want: "(+1/-1)",
		},
		{
			name:     "Edit single line replacement no info",
			toolName: "Edit",
			input: map[string]interface{}{
				"old_string": "old",
				"new_string": "new",
			},
			want: "",
		},
		{
			name:     "Write shows line count",
			toolName: "Write",
			input: map[string]interface{}{
				"content": "line1\nline2\nline3",
			},
			want: "(3 lines)",
		},
		{
			name:     "Write single line",
			toolName: "Write",
			input: map[string]interface{}{
				"content": "single line",
			},
			want: "(1 lines)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputBytes, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("failed to marshal input: %v", err)
			}
			got := extractToolUseLineInfo(tt.toolName, inputBytes)
			if got != tt.want {
				t.Errorf("extractToolUseLineInfo(%q) = %q, want %q", tt.toolName, got, tt.want)
			}
		})
	}
}

func TestExtractToolResultLineInfo(t *testing.T) {
	tests := []struct {
		name    string
		content interface{}
		want    string
	}{
		{
			name:    "string content counts lines",
			content: "line1\nline2\nline3\nline4",
			want:    "(4 lines)",
		},
		{
			name:    "single line",
			content: "just one line",
			want:    "(1 lines)",
		},
		{
			name:    "empty string returns empty",
			content: "",
			want:    "",
		},
		{
			name:    "nil content returns empty",
			content: nil,
			want:    "",
		},
		{
			name:    "non-string content returns empty",
			content: 42,
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractToolResultLineInfo(tt.content)
			if got != tt.want {
				t.Errorf("extractToolResultLineInfo(%v) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short string unchanged",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exact length unchanged",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "long string truncated with ellipsis",
			input:  "hello world",
			maxLen: 8,
			want:   "hello...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}
