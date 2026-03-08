package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/loxstomper/skinner/internal/model"
	"github.com/loxstomper/skinner/internal/theme"
)

func TestExpandedContentLines_Bash(t *testing.T) {
	tc := &model.ToolCall{
		Name:     "Bash",
		Expanded: true,
		RawInput: map[string]interface{}{
			"command": "go test ./...",
		},
		ResultContent: "ok  github.com/foo/bar\nFAIL github.com/foo/baz",
	}

	lines := expandedContentLines(tc)
	if lines == nil {
		t.Fatal("expected lines, got nil")
	}
	if lines[0] != "$ go test ./..." {
		t.Errorf("expected '$ go test ./...', got %q", lines[0])
	}
	if len(lines) != 3 {
		t.Errorf("expected 3 lines (command + 2 output), got %d", len(lines))
	}
	if lines[1] != "ok  github.com/foo/bar" {
		t.Errorf("expected output line 1, got %q", lines[1])
	}
	if lines[2] != "FAIL github.com/foo/baz" {
		t.Errorf("expected output line 2, got %q", lines[2])
	}
}

func TestExpandedContentLines_BashNoOutput(t *testing.T) {
	tc := &model.ToolCall{
		Name:     "Bash",
		Expanded: true,
		RawInput: map[string]interface{}{
			"command": "mkdir -p /tmp/test",
		},
	}

	lines := expandedContentLines(tc)
	if lines == nil {
		t.Fatal("expected lines, got nil")
	}
	if len(lines) != 1 {
		t.Errorf("expected 1 line (command only), got %d", len(lines))
	}
	if lines[0] != "$ mkdir -p /tmp/test" {
		t.Errorf("expected '$ mkdir -p /tmp/test', got %q", lines[0])
	}
}

func TestExpandedContentLines_Edit(t *testing.T) {
	tc := &model.ToolCall{
		Name:     "Edit",
		Expanded: true,
		RawInput: map[string]interface{}{
			"old_string": "return \"hello\"",
			"new_string": "name := \"world\"\nreturn fmt.Sprintf(\"hello, %s\", name)",
		},
	}

	lines := expandedContentLines(tc)
	if lines == nil {
		t.Fatal("expected lines, got nil")
	}
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "-return \"hello\"" {
		t.Errorf("expected old line with - prefix, got %q", lines[0])
	}
	if lines[1] != "+name := \"world\"" {
		t.Errorf("expected first new line with + prefix, got %q", lines[1])
	}
	if lines[2] != "+return fmt.Sprintf(\"hello, %s\", name)" {
		t.Errorf("expected second new line with + prefix, got %q", lines[2])
	}
}

func TestExpandedContentLines_Read(t *testing.T) {
	tc := &model.ToolCall{
		Name:          "Read",
		Expanded:      true,
		ResultContent: "line1\nline2\nline3",
	}

	lines := expandedContentLines(tc)
	if lines == nil {
		t.Fatal("expected lines, got nil")
	}
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestExpandedContentLines_Grep(t *testing.T) {
	tc := &model.ToolCall{
		Name:          "Grep",
		Expanded:      true,
		ResultContent: "file1.go:10:match\nfile2.go:20:match",
	}

	lines := expandedContentLines(tc)
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestExpandedContentLines_Glob(t *testing.T) {
	tc := &model.ToolCall{
		Name:          "Glob",
		Expanded:      true,
		ResultContent: "src/main.go\nsrc/util.go",
	}

	lines := expandedContentLines(tc)
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

func TestExpandedContentLines_Task(t *testing.T) {
	tc := &model.ToolCall{
		Name:          "Task",
		Expanded:      true,
		ResultContent: "task output here",
	}

	lines := expandedContentLines(tc)
	if len(lines) != 1 {
		t.Errorf("expected 1 line, got %d", len(lines))
	}
}

func TestExpandedContentLines_Write(t *testing.T) {
	tc := &model.ToolCall{
		Name:     "Write",
		Expanded: true,
		RawInput: map[string]interface{}{
			"content": "package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}",
		},
	}

	lines := expandedContentLines(tc)
	if lines == nil {
		t.Fatal("expected lines, got nil")
	}
	if len(lines) != 5 {
		t.Errorf("expected 5 lines, got %d", len(lines))
	}
	if lines[0] != "package main" {
		t.Errorf("expected first line 'package main', got %q", lines[0])
	}
}

func TestExpandedContentLines_UnknownTool(t *testing.T) {
	tc := &model.ToolCall{
		Name:          "CustomTool",
		Expanded:      true,
		ResultContent: "some output",
	}

	lines := expandedContentLines(tc)
	if len(lines) != 1 {
		t.Errorf("expected 1 line, got %d", len(lines))
	}
	if lines[0] != "some output" {
		t.Errorf("expected 'some output', got %q", lines[0])
	}
}

func TestExpandedContentLines_NotExpanded(t *testing.T) {
	tc := &model.ToolCall{
		Name:          "Read",
		Expanded:      false,
		ResultContent: "some content",
	}

	lines := expandedContentLines(tc)
	if lines != nil {
		t.Errorf("expected nil for collapsed tool call, got %v", lines)
	}
}

func TestExpandedContentLines_NoContent(t *testing.T) {
	tc := &model.ToolCall{
		Name:     "Read",
		Expanded: true,
	}

	lines := expandedContentLines(tc)
	if lines != nil {
		t.Errorf("expected nil for empty content, got %v", lines)
	}
}

func TestExpandedContentLines_EmptyResultContent(t *testing.T) {
	tc := &model.ToolCall{
		Name:          "Read",
		Expanded:      true,
		ResultContent: "",
	}

	lines := expandedContentLines(tc)
	if lines != nil {
		t.Errorf("expected nil for empty result content, got %v", lines)
	}
}

func TestExpandedContentLines_FullContent(t *testing.T) {
	// Build content with 30 lines — all should be returned without truncation.
	// Sub-scroll (phase 3.2) handles viewport management for large content.
	var contentLines []string
	for i := 1; i <= 30; i++ {
		contentLines = append(contentLines, fmt.Sprintf("line %d", i))
	}
	tc := &model.ToolCall{
		Name:          "Read",
		Expanded:      true,
		ResultContent: strings.Join(contentLines, "\n"),
	}

	lines := expandedContentLines(tc)
	if len(lines) != 30 {
		t.Errorf("expected all 30 lines returned, got %d", len(lines))
	}
	if lines[0] != "line 1" {
		t.Errorf("expected first line 'line 1', got %q", lines[0])
	}
	if lines[29] != "line 30" {
		t.Errorf("expected last line 'line 30', got %q", lines[29])
	}
}

func TestRenderEditDiff_BasicReplacement(t *testing.T) {
	input := map[string]interface{}{
		"old_string": "hello",
		"new_string": "world",
	}

	lines := renderEditDiff(input)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0] != "-hello" {
		t.Errorf("expected '-hello', got %q", lines[0])
	}
	if lines[1] != "+world" {
		t.Errorf("expected '+world', got %q", lines[1])
	}
}

func TestRenderEditDiff_MultiLine(t *testing.T) {
	input := map[string]interface{}{
		"old_string": "line1\nline2",
		"new_string": "new1\nnew2\nnew3",
	}

	lines := renderEditDiff(input)
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(lines))
	}
	if lines[0] != "-line1" {
		t.Errorf("expected '-line1', got %q", lines[0])
	}
	if lines[1] != "-line2" {
		t.Errorf("expected '-line2', got %q", lines[1])
	}
	if lines[2] != "+new1" {
		t.Errorf("expected '+new1', got %q", lines[2])
	}
	if lines[3] != "+new2" {
		t.Errorf("expected '+new2', got %q", lines[3])
	}
	if lines[4] != "+new3" {
		t.Errorf("expected '+new3', got %q", lines[4])
	}
}

func TestRenderEditDiff_EmptyOldString(t *testing.T) {
	input := map[string]interface{}{
		"old_string": "",
		"new_string": "added line",
	}

	lines := renderEditDiff(input)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d: %v", len(lines), lines)
	}
	if lines[0] != "+added line" {
		t.Errorf("expected '+added line', got %q", lines[0])
	}
}

func TestRenderEditDiff_EmptyNewString(t *testing.T) {
	input := map[string]interface{}{
		"old_string": "removed line",
		"new_string": "",
	}

	lines := renderEditDiff(input)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d: %v", len(lines), lines)
	}
	if lines[0] != "-removed line" {
		t.Errorf("expected '-removed line', got %q", lines[0])
	}
}

func TestRenderEditDiff_BothEmpty(t *testing.T) {
	input := map[string]interface{}{
		"old_string": "",
		"new_string": "",
	}

	lines := renderEditDiff(input)
	if lines != nil {
		t.Errorf("expected nil for both empty, got %v", lines)
	}
}

func TestRenderEditDiff_NilInput(t *testing.T) {
	lines := renderEditDiff(nil)
	if lines != nil {
		t.Errorf("expected nil for nil input, got %v", lines)
	}
}

func TestRenderEditDiff_MissingFields(t *testing.T) {
	input := map[string]interface{}{
		"file_path": "src/main.go",
	}

	lines := renderEditDiff(input)
	if lines != nil {
		t.Errorf("expected nil when old/new missing, got %v", lines)
	}
}

func TestToolCallLineCount_Collapsed(t *testing.T) {
	tc := &model.ToolCall{
		Name:     "Read",
		Expanded: false,
	}

	count := toolCallLineCount(tc)
	if count != 1 {
		t.Errorf("expected 1 for collapsed, got %d", count)
	}
}

func TestToolCallLineCount_ExpandedWithContent(t *testing.T) {
	tc := &model.ToolCall{
		Name:          "Read",
		Expanded:      true,
		ResultContent: "line1\nline2\nline3",
	}

	count := toolCallLineCount(tc)
	// 1 header + 3 content lines = 4
	if count != 4 {
		t.Errorf("expected 4 (1 header + 3 content), got %d", count)
	}
}

func TestToolCallLineCount_ExpandedNoContent(t *testing.T) {
	tc := &model.ToolCall{
		Name:     "Read",
		Expanded: true,
	}

	count := toolCallLineCount(tc)
	// Expanded but no content: still just 1 (header only)
	if count != 1 {
		t.Errorf("expected 1 for expanded with no content, got %d", count)
	}
}

func TestToolCallLineCount_ExpandedLargeContent(t *testing.T) {
	// Build content with 30 lines — all lines counted without truncation.
	var contentLines []string
	for i := 1; i <= 30; i++ {
		contentLines = append(contentLines, fmt.Sprintf("line %d", i))
	}
	tc := &model.ToolCall{
		Name:          "Read",
		Expanded:      true,
		ResultContent: strings.Join(contentLines, "\n"),
	}

	count := toolCallLineCount(tc)
	// 1 header + 30 content lines = 31
	if count != 31 {
		t.Errorf("expected 31 (1 header + 30 content), got %d", count)
	}
}

func TestToolCallLineCount_ExpandedBash(t *testing.T) {
	tc := &model.ToolCall{
		Name:     "Bash",
		Expanded: true,
		RawInput: map[string]interface{}{
			"command": "echo hello",
		},
		ResultContent: "hello",
	}

	count := toolCallLineCount(tc)
	// 1 header + 2 content lines ($ command + output) = 3
	if count != 3 {
		t.Errorf("expected 3 (1 header + $ cmd + output), got %d", count)
	}
}

func TestToolCallLineCount_ExpandedEdit(t *testing.T) {
	tc := &model.ToolCall{
		Name:     "Edit",
		Expanded: true,
		RawInput: map[string]interface{}{
			"old_string": "old",
			"new_string": "new1\nnew2",
		},
	}

	count := toolCallLineCount(tc)
	// 1 header + 3 diff lines (-old, +new1, +new2) = 4
	if count != 4 {
		t.Errorf("expected 4 (1 header + 3 diff lines), got %d", count)
	}
}

func TestRenderExpandedContentLine_DimColor(t *testing.T) {
	th, _ := theme.LookupTheme("solarized-dark")
	line := renderExpandedContentLine("some content", "Read", 80, th)
	// Should contain the indented text
	if !strings.Contains(line, "some content") {
		t.Errorf("expected line to contain 'some content', got %q", line)
	}
}

func TestRenderExpandedContentLine_EditRemoved(t *testing.T) {
	th, _ := theme.LookupTheme("solarized-dark")
	line := renderExpandedContentLine("-old line", "Edit", 80, th)
	if !strings.Contains(line, "-old line") {
		t.Errorf("expected line to contain '-old line', got %q", line)
	}
}

func TestRenderExpandedContentLine_EditAdded(t *testing.T) {
	th, _ := theme.LookupTheme("solarized-dark")
	line := renderExpandedContentLine("+new line", "Edit", 80, th)
	if !strings.Contains(line, "+new line") {
		t.Errorf("expected line to contain '+new line', got %q", line)
	}
}

func TestRenderExpandedContentLine_Truncation(t *testing.T) {
	th, _ := theme.LookupTheme("solarized-dark")
	longLine := strings.Repeat("x", 100)
	line := renderExpandedContentLine(longLine, "Read", 30, th)
	// The rendered line should not contain the full 100 chars + 4 indent
	if strings.Contains(line, strings.Repeat("x", 100)) {
		t.Errorf("expected line to be truncated")
	}
}

func TestRenderExpandedContentLine_NonEditDashNotRed(t *testing.T) {
	th, _ := theme.LookupTheme("solarized-dark")
	// A line starting with - in a non-Edit tool should use dim color, not red
	line := renderExpandedContentLine("-not a diff", "Bash", 80, th)
	if !strings.Contains(line, "-not a diff") {
		t.Errorf("expected line to contain '-not a diff', got %q", line)
	}
}

func TestExpandedContentLines_BashNoCommand(t *testing.T) {
	tc := &model.ToolCall{
		Name:          "Bash",
		Expanded:      true,
		RawInput:      map[string]interface{}{},
		ResultContent: "output only",
	}

	lines := expandedContentLines(tc)
	if lines == nil {
		t.Fatal("expected lines, got nil")
	}
	// Should just have the output, no $ command line
	if len(lines) != 1 {
		t.Errorf("expected 1 line (output only), got %d: %v", len(lines), lines)
	}
	if lines[0] != "output only" {
		t.Errorf("expected 'output only', got %q", lines[0])
	}
}

func TestExpandedContentLines_WriteNoContent(t *testing.T) {
	tc := &model.ToolCall{
		Name:     "Write",
		Expanded: true,
		RawInput: map[string]interface{}{
			"file_path": "src/main.go",
		},
	}

	lines := expandedContentLines(tc)
	if lines != nil {
		t.Errorf("expected nil for Write with no content, got %v", lines)
	}
}
