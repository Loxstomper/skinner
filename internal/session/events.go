package session

// Event is the interface for typed events flowing from the executor to the
// session controller. These mirror the parser event types but live here so
// that the TUI never imports the parser package directly.
type Event interface{ event() }

// ToolUseEvent represents an assistant requesting a tool call.
type ToolUseEvent struct {
	ID       string
	Name     string
	Summary  string
	LineInfo string
	RawInput map[string]interface{}
}

func (ToolUseEvent) event() {}

// ToolResultEvent represents the result of a tool call execution.
type ToolResultEvent struct {
	ToolUseID string
	IsError   bool
	LineInfo  string
	Content   string
}

func (ToolResultEvent) event() {}

// TextEvent represents assistant text output.
type TextEvent struct {
	Text string
}

func (TextEvent) event() {}

// UsageEvent carries token usage from an assistant response.
type UsageEvent struct {
	Model                    string
	InputTokens              int64
	OutputTokens             int64
	CacheReadInputTokens     int64
	CacheCreationInputTokens int64
}

func (UsageEvent) event() {}

// AssistantBatchEvent wraps ToolUseEvent and TextEvent values from a single
// assistant message. The controller's ProcessAssistantBatch expects these
// grouped together to detect consecutive same-name tool call runs.
type AssistantBatchEvent struct {
	Events []Event
}

func (AssistantBatchEvent) event() {}

// IterationEndEvent signals that the current iteration's result has arrived.
type IterationEndEvent struct{}

func (IterationEndEvent) event() {}

// SubprocessExitEvent signals that the Claude subprocess has exited.
// A nil Err indicates clean exit.
type SubprocessExitEvent struct {
	Err error
}

func (SubprocessExitEvent) event() {}
