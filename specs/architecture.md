# Architecture

## Overview

The codebase is structured in layers with clear dependency boundaries. Each layer is independently testable. Side effects (subprocess spawning, file I/O, wall-clock time) are isolated behind interfaces so that business logic and rendering can be tested with pure functions and deterministic inputs.

## Package Map

```
cmd/skinner/main.go           CLI entry point, wiring
internal/
  config/                      TOML config loading, pricing defaults
  model/                       Pure data types (Session, Iteration, ToolCall, etc.)
  parser/                      Stream-JSON line parsing (pure functions)
  theme/                       Color theme definitions and lookup
  session/                     Business logic controller (no I/O, no TUI)
  executor/                    Subprocess abstraction (interface + real impl)
  tui/
    root.go                    Root Bubble Tea model (thin coordinator)
    header.go                  Header bar component
    iterlist.go                Iteration list component (left pane)
    timeline.go                Message timeline component (right pane)
    autofollow.go              Auto-follow state machine
    cursor.go                  Flat cursor addressing over timeline items
    format.go                  Duration/token formatting, tool icons
```

### Dependency Direction

```
main.go
  ├── config
  ├── theme
  ├── model
  ├── executor (real impl)
  └── tui/root
        ├── session (controller)
        │     └── model
        ├── executor (interface only)
        ├── tui/header
        │     ├── model
        │     └── theme
        ├── tui/iterlist
        │     ├── model
        │     └── theme
        ├── tui/timeline
        │     ├── model
        │     ├── theme
        │     ├── tui/cursor
        │     ├── tui/autofollow
        │     └── tui/format
        └── tui/format
```

No package imports `tui/root`. The `session` package never imports `tui` or `executor`. The `executor` package imports `parser` internally but exposes only typed events.

## Session Controller

**Package:** `internal/session`

The session controller owns all non-UI business logic. It operates on `model.Session` via methods that take typed events and mutate session state. It has no I/O, no Bubble Tea dependency, and no rendering.

### Responsibilities

- **Event processing** — Accept typed events (from executor), update session state.
- **Tool result matching** — Find the pending tool call by ID, apply result status and duration.
- **Tool call grouping** — Group consecutive same-type tool calls from a single assistant batch into `ToolCallGroup` or standalone `ToolCall` items.
- **Token accumulation** — Sum token counts across assistant events; track latest input/cache values for context window %.
- **Cost calculation** — Compute per-event cost using pricing config; accumulate session total.
- **Iteration lifecycle** — Create new iterations, mark completed/failed, decide whether to start next.

### Interface

```go
type Controller struct {
    Session *model.Session
    Config  config.Config
    Clock   func() time.Time  // injectable, defaults to time.Now
}

// ProcessAssistantBatch handles a batch of tool use and text events from one
// assistant message. It creates timeline items (grouping consecutive same-type
// tool calls) and appends them to the running iteration.
func (c *Controller) ProcessAssistantBatch(events []Event)

// ProcessToolResult finds the matching tool call and applies result status,
// duration, and line info. Returns the affected ToolCallGroup (if any) so
// the caller can handle expand/collapse UI concerns.
func (c *Controller) ProcessToolResult(result ToolResultEvent) *model.ToolCallGroup

// ProcessUsage accumulates token counts and cost.
func (c *Controller) ProcessUsage(usage UsageEvent)

// StartIteration creates a new running iteration and appends it to the session.
func (c *Controller) StartIteration()

// CompleteIteration marks the running iteration as completed or failed.
func (c *Controller) CompleteIteration(err error)

// ShouldStartNext returns true if another iteration should begin.
func (c *Controller) ShouldStartNext() bool

// RunningIterationIdx returns the index of the running iteration, or -1.
func (c *Controller) RunningIterationIdx() int

// HasKnownModel returns true if pricing info is available for the current model.
func (c *Controller) HasKnownModel() bool
```

### Event Types

The session controller defines its own event types (or re-exports from parser). These are the typed events that flow from the executor:

```go
type Event interface{ event() }

type ToolUseEvent struct { ID, Name, Summary, LineInfo string }
type ToolResultEvent struct { ToolUseID string; IsError bool; LineInfo string }
type TextEvent struct { Text string }
type UsageEvent struct { Model string; InputTokens, OutputTokens, CacheReadInputTokens, CacheCreationInputTokens int64 }
type IterationEndEvent struct{}
```

These mirror the parser event types but live in the session package to avoid the TUI depending on the parser.

## Executor

**Package:** `internal/executor`

The executor abstracts subprocess spawning behind an interface. The real implementation wraps `os/exec` and the parser. Test code provides a fake implementation that emits canned events.

### Interface

```go
// Executor starts a Claude CLI subprocess and returns a stream of typed events.
type Executor interface {
    Start(ctx context.Context, prompt string) (<-chan session.Event, error)
    Kill() error
}
```

### Real Implementation

`ClaudeExecutor` in `internal/executor/claude.go`:

- Spawns `claude -p --dangerously-skip-permissions --output-format=stream-json --verbose`.
- Pipes prompt to stdin.
- Reads stdout line-by-line with `bufio.Scanner`.
- Parses each line with `parser.ParseStreamEvent()`.
- Sends typed events to the returned channel.
- Sends a sentinel (channel close or a `SubprocessExitEvent`) when the process exits.

The parser package is consumed entirely within the executor — no other package imports it.

### Fake Implementation (for tests)

`FakeExecutor` in `internal/executor/fake.go`:

```go
type FakeExecutor struct {
    Events []session.Event  // canned events to emit
    Delay  time.Duration    // optional delay between events
}

func (f *FakeExecutor) Start(ctx context.Context, prompt string) (<-chan session.Event, error) {
    ch := make(chan session.Event, len(f.Events))
    for _, e := range f.Events {
        ch <- e
    }
    close(ch)
    return ch, nil
}
```

## TUI Components

The TUI is decomposed into sub-components. Each component follows Bubble Tea's model-view-update pattern with its own `Update` and `View` methods. Components are **controlled** — they receive data from the root model and only own their own view state (cursor position, scroll offset, expanded/collapsed flags).

### Root Model (`tui/root.go`)

The root model is a thin coordinator. It owns:

- `session.Controller` — business logic
- `executor.Executor` — subprocess abstraction
- `Header`, `IterList`, `Timeline` — sub-components
- Focus state (which pane is active)
- Terminal dimensions

The root model's `Update` method:

1. Receives Bubble Tea messages.
2. Delegates event processing to `session.Controller` for business events.
3. Routes key events to the focused sub-component.
4. Passes updated state down to sub-components.

The root model's `View` method:

1. Calls `Header.View()`, `IterList.View()`, `Timeline.View()`.
2. Joins them with a separator.

### Header (`tui/header.go`)

Stateless renderer. Receives a `HeaderProps` struct and returns a rendered string.

```go
type HeaderProps struct {
    SessionDuration   time.Duration
    InputTokens       int64
    OutputTokens      int64
    ContextPercent    int      // -1 if unknown
    TotalCost         float64
    HasKnownModel     bool
    IterationCount    int
    MaxIterations     int
    SessionStatus     IterationStatus  // running/completed/failed
    Width             int
    Theme             theme.Theme
}

func RenderHeader(p HeaderProps) string
```

No `Update` method. No state. Pure function.

### Iteration List (`tui/iterlist.go`)

Left pane component. Owns:

- Cursor position (selected iteration index)
- `AutoFollow` state

```go
type IterList struct {
    Cursor     int
    AutoFollow AutoFollow
}

type IterListProps struct {
    Iterations []model.Iteration
    Width      int
    Height     int
    Focused    bool
    Theme      theme.Theme
}

func (il *IterList) Update(msg tea.KeyMsg, props IterListProps) tea.Cmd
func (il *IterList) View(props IterListProps) string
func (il *IterList) OnNewIteration(count int)  // auto-follow hook
func (il *IterList) SelectedIndex() int
```

### Timeline (`tui/timeline.go`)

Right pane component. Owns:

- Cursor position (flat index over visible items)
- Scroll offset
- `AutoFollow` state
- Expand/collapse state is stored on the `model.TextBlock` and `model.ToolCallGroup` structs themselves (since it's per-item, not per-view).

```go
type Timeline struct {
    Cursor     int
    Scroll     int
    AutoFollow AutoFollow
}

type TimelineProps struct {
    Items       []model.TimelineItem
    Width       int
    Height      int
    Focused     bool
    CompactView bool
    Theme       theme.Theme
}

func (tl *Timeline) Update(msg tea.KeyMsg, props TimelineProps) tea.Cmd
func (tl *Timeline) View(props TimelineProps) string
func (tl *Timeline) OnNewItems(props TimelineProps)  // auto-follow hook
```

The Timeline component uses the cursor and format helpers internally.

## Pure Helpers

### Auto-Follow (`tui/autofollow.go`)

A small state machine shared by IterList and Timeline.

```go
type AutoFollow struct {
    following bool
}

func NewAutoFollow() AutoFollow              // starts following
func (af *AutoFollow) OnManualMove(atEnd bool) // pause if not at end
func (af *AutoFollow) OnNewItem()              // no-op (doesn't resume)
func (af *AutoFollow) JumpToEnd()              // resume following
func (af AutoFollow) Following() bool          // query state
```

Rules:
- Starts in following mode.
- Any manual cursor movement pauses following, unless the cursor is at the end position.
- Moving to the end (via `G`/`End` or natural arrival) resumes following.
- New items arriving do not resume following.

### Flat Cursor (`tui/cursor.go`)

Pure functions that compute cursor positions over `[]model.TimelineItem`, accounting for expanded/collapsed groups.

```go
// FlatCursorCount returns total navigable positions.
func FlatCursorCount(items []model.TimelineItem) int

// FlatToItem maps a flat cursor index to (item index, child index).
// childIdx == -1 for non-group items or group headers.
func FlatToItem(items []model.TimelineItem, flatIdx int) (itemIdx, childIdx int)

// ItemToFlat maps an item index to its flat cursor position.
func ItemToFlat(items []model.TimelineItem, itemIdx int) int

// FlatCursorLineRange returns (start line, line count) for a flat position.
// Used for ensuring cursor visibility within scroll viewport.
func FlatCursorLineRange(items []model.TimelineItem, flatIdx int, compactView bool) (lineStart, lineCount int)
```

These operate on slices, not on a Model receiver. Fully testable with constructed item slices.

### Format (`tui/format.go`)

Pure formatting functions with no receiver or external state.

```go
func FormatDuration(d time.Duration, running bool) string
func FormatDurationValue(d time.Duration) string
func FormatTokens(tokens int64) string
func ToolIcon(name string) string
func GroupSummaryUnit(toolName string) string
func IsKnownTool(name string) bool
```

## Clock

Time-dependent operations (tool call start time, iteration start time, duration calculation) accept a clock function:

```go
type Clock func() time.Time
```

The session controller accepts this in its constructor. In production, pass `time.Now`. In tests, pass a controllable fake:

```go
func fakeClock(t *time.Time) func() time.Time {
    return func() time.Time { return *t }
}
```

## Testing Strategy

| Layer | What to test | How |
|-------|-------------|-----|
| `parser` | JSON line → typed events, summary extraction, line info extraction | Table-driven tests with JSON string fixtures |
| `model` | `ToolCallGroup.Status()`, `GroupDuration()`, `CompletedCount()`, `Iteration.ToolCallCount()` | Construct structs, call methods, assert |
| `theme` | `LookupTheme()` returns correct theme, `ThemeNames()` is sorted | Simple assertions |
| `config` | TOML parsing, defaults, missing file | Already tested; extend with edge cases |
| `session` | Event processing, grouping, tool result matching, token/cost accumulation, iteration lifecycle | Feed events to `Controller`, assert `Session` state. Use fake clock. |
| `autofollow` | State machine transitions | Call methods, assert `Following()` |
| `cursor` | Flat cursor math over various item configurations | Construct `[]TimelineItem` slices, assert positions |
| `format` | Duration formatting, token formatting, icon lookup | Table-driven pure function tests |
| `header` | Rendered output for various stats | Call `RenderHeader()` with props, assert substring content |
| `iterlist` | Cursor movement, selection, rendering | Construct component + props, call `Update`/`View`, assert |
| `timeline` | Cursor movement, scroll, expand/collapse, rendering | Same pattern |
| `executor` | Real executor: skip in CI. Fake executor: verify channel behavior. | Unit test the fake; integration test the real with a mock binary if needed |
| Integration | Full TUI loop with fake executor | Create root model with `FakeExecutor`, send messages via `Update()`, assert `View()` output |
