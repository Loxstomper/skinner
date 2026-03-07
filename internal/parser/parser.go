package parser

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Raw JSON event structures from Claude CLI stream-json output.

type streamEvent struct {
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message"`
}

type message struct {
	Role    string          `json:"role"`
	Model   string          `json:"model,omitempty"`
	Content json.RawMessage `json:"content"`
	Usage   *messageUsage   `json:"usage,omitempty"`
}

type messageUsage struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
}

type contentBlock struct {
	Type      string          `json:"type"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	Text      string          `json:"text,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   interface{}     `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

// Parsed event types emitted to the TUI.

type ToolUseEvent struct {
	ID       string
	Name     string
	Summary  string
	LineInfo string
}

type ToolResultEvent struct {
	ToolUseID string
	IsError   bool
	LineInfo  string
}

type TextEvent struct {
	Text string
}

type UsageEvent struct {
	Model                    string
	InputTokens              int64
	OutputTokens             int64
	CacheReadInputTokens     int64
	CacheCreationInputTokens int64
}

type IterationEndEvent struct{}

// ParseStreamEvent parses a single line of stream-json output and returns
// zero or more parsed events.
func ParseStreamEvent(line string) ([]interface{}, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}

	var event streamEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return nil, nil //nolint:nilerr // skip unparseable lines
	}

	switch event.Type {
	case "assistant":
		return parseAssistantEvent(event.Message)
	case "user":
		return parseUserEvent(event.Message)
	case "result":
		return []interface{}{IterationEndEvent{}}, nil
	default:
		return nil, nil
	}
}

func parseAssistantEvent(raw json.RawMessage) ([]interface{}, error) {
	var msg message
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, err
	}

	var blocks []contentBlock
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return nil, err
	}

	var events []interface{}

	// Emit usage event if present
	if msg.Usage != nil {
		events = append(events, UsageEvent{
			Model:                    msg.Model,
			InputTokens:              msg.Usage.InputTokens,
			OutputTokens:             msg.Usage.OutputTokens,
			CacheReadInputTokens:     msg.Usage.CacheReadInputTokens,
			CacheCreationInputTokens: msg.Usage.CacheCreationInputTokens,
		})
	}

	for _, block := range blocks {
		switch block.Type {
		case "text":
			if block.Text != "" {
				events = append(events, TextEvent{Text: block.Text})
			}
		case "tool_use":
			summary := extractToolSummary(block.Name, block.Input)
			lineInfo := extractToolUseLineInfo(block.Name, block.Input)
			events = append(events, ToolUseEvent{
				ID:       block.ID,
				Name:     block.Name,
				Summary:  summary,
				LineInfo: lineInfo,
			})
		}
	}
	return events, nil
}

func parseUserEvent(raw json.RawMessage) ([]interface{}, error) {
	var msg message
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, err
	}

	var blocks []contentBlock
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return nil, err
	}

	var events []interface{}
	for _, block := range blocks {
		if block.Type == "tool_result" {
			lineInfo := extractToolResultLineInfo(block.Content)
			events = append(events, ToolResultEvent{
				ToolUseID: block.ToolUseID,
				IsError:   block.IsError,
				LineInfo:  lineInfo,
			})
		}
	}
	return events, nil
}

func extractToolSummary(toolName string, inputRaw json.RawMessage) string {
	var input map[string]interface{}
	if err := json.Unmarshal(inputRaw, &input); err != nil {
		return ""
	}

	switch toolName {
	case "Read", "Edit", "Write":
		if fp, ok := input["file_path"].(string); ok {
			return fp
		}
	case "Bash":
		if desc, ok := input["description"].(string); ok && desc != "" {
			return desc
		}
		if cmd, ok := input["command"].(string); ok {
			return truncate(cmd, 80)
		}
	case "Grep":
		pattern, _ := input["pattern"].(string)
		path, _ := input["path"].(string)
		if path != "" {
			return fmt.Sprintf("%q in %s", pattern, path)
		}
		return fmt.Sprintf("%q", pattern)
	case "Glob":
		if pat, ok := input["pattern"].(string); ok {
			return pat
		}
	case "Task":
		if desc, ok := input["description"].(string); ok {
			return desc
		}
	}

	return ""
}

// extractToolUseLineInfo computes line count metadata for Edit and Write tools
// from the tool_use input in the assistant event.
func extractToolUseLineInfo(toolName string, inputRaw json.RawMessage) string {
	if toolName != "Edit" && toolName != "Write" {
		return ""
	}

	var input map[string]interface{}
	if err := json.Unmarshal(inputRaw, &input); err != nil {
		return ""
	}

	switch toolName {
	case "Edit":
		oldStr, _ := input["old_string"].(string)
		newStr, _ := input["new_string"].(string)
		oldLines := strings.Count(oldStr, "\n")
		newLines := strings.Count(newStr, "\n")
		added := newLines - oldLines
		removed := oldLines - newLines
		if added > 0 {
			return fmt.Sprintf("(+%d)", added)
		}
		if removed > 0 {
			return fmt.Sprintf("(-%d)", removed)
		}
		// Net zero: show both counts when there are actual lines
		if oldLines > 0 {
			return fmt.Sprintf("(+%d/-%d)", oldLines, oldLines)
		}
		return ""

	case "Write":
		content, _ := input["content"].(string)
		lines := strings.Count(content, "\n") + 1
		return fmt.Sprintf("(%d lines)", lines)
	}

	return ""
}

// extractToolResultLineInfo extracts line count from a Read tool result content.
// The content is the tool_result content field, which may be a string.
func extractToolResultLineInfo(content interface{}) string {
	str, ok := content.(string)
	if !ok || str == "" {
		return ""
	}
	lines := strings.Count(str, "\n") + 1
	return fmt.Sprintf("(%d lines)", lines)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
