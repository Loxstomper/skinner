# Stream JSON Format

The Claude CLI `--output-format=stream-json` emits newline-delimited JSON. Each line is a JSON object with a `type` field.

## Event Types

### `assistant` — Tool Call

Emitted when Claude invokes a tool.

```json
{
  "type": "assistant",
  "message": {
    "model": "claude-sonnet-4-6",
    "id": "msg_...",
    "role": "assistant",
    "content": [
      {
        "type": "tool_use",
        "id": "toolu_...",
        "name": "Bash",
        "input": {
          "command": "ls *.md",
          "description": "List markdown files"
        }
      }
    ],
    "usage": {
      "input_tokens": 1234,
      "output_tokens": 567,
      "cache_read_input_tokens": 8901,
      "cache_creation_input_tokens": 2345
    }
  },
  "session_id": "...",
  "uuid": "..."
}
```

- `message.content` is an array. Elements can be:
  - `"type": "text"` — Claude's reasoning or response text. Display as a text block in the timeline.
  - `"type": "tool_use"` — a tool invocation.
- `message.usage` contains token counts for this API call:
  - `input_tokens` — input tokens (excluding cache).
  - `output_tokens` — output tokens.
  - `cache_read_input_tokens` — tokens read from prompt cache (optional, may be 0 or absent).
  - `cache_creation_input_tokens` — tokens written to prompt cache (optional, may be 0 or absent).
- `message.model` — the model ID (e.g. `claude-sonnet-4-6`, `claude-opus-4-6`). Used for cost calculation.
- For `tool_use` elements:
  - `id` is the unique tool use ID used to correlate with the result.
  - `name` is the tool name (e.g. `Read`, `Edit`, `Bash`, `Grep`, `Glob`, `Write`, `Task`).
  - `input` is a tool-specific arguments object.

### `user` — Tool Result

Emitted when a tool call completes and returns its result.

```json
{
  "type": "user",
  "message": {
    "role": "user",
    "content": [
      {
        "tool_use_id": "toolu_...",
        "type": "tool_result",
        "content": "file contents here...",
        "is_error": false
      }
    ]
  },
  "session_id": "...",
  "uuid": "..."
}
```

- `tool_use_id` matches the `id` from the corresponding `tool_use` event.
- `content` is the tool output (string). Not displayed in the timeline.
- `is_error` indicates whether the tool call failed. Displayed as `✓` (false) or `✗` (true) on the tool call row.

### `result` — End of Session

Emitted at the end of a Claude invocation. Used to mark an iteration as complete.

## Token and Cost Tracking

Accumulate token counts from `message.usage` across all `assistant` events in the session. Track four totals:

- `input_tokens` — total input tokens (excluding cache)
- `output_tokens` — total output tokens
- `cache_read_input_tokens` — total cache read tokens
- `cache_creation_input_tokens` — total cache creation tokens

### Context Window Usage

In addition to accumulated totals, track the **latest** `input_tokens` and `cache_read_input_tokens` values from the most recent `assistant` event (not accumulated — replaced each time). These represent the current context window consumption and are used to calculate the context window percentage displayed in the header (see [tui-layout.md](tui-layout.md)). The percentage is: `(latest_input_tokens + latest_cache_read_input_tokens) / context_window * 100`, where `context_window` comes from the model's pricing config (see [config.md](config.md)).

### Cost Calculation

Cost is computed per `assistant` event using the model's pricing rates (see [config.md](config.md) for the pricing table):

```
cost = (input_tokens * input_rate)
     + (output_tokens * output_rate)
     + (cache_read_input_tokens * cache_read_rate)
     + (cache_creation_input_tokens * cache_create_rate)
```

Accumulate cost across all events in the session. If a model ID is not found in the pricing table, track tokens but skip cost calculation — display tokens without a cost estimate.

## Tool Input Summaries

Each tool type has different args. The following fields should be used to generate a short summary. Each tool also has a Nerd Font icon for display in the timeline (see [tui-layout.md](tui-layout.md)).

| Tool  | Icon | Codepoint | Summary field(s)                                      |
|-------|------|-----------|-------------------------------------------------------|
| Read  | ``  | `f02d`    | `input.file_path`                                     |
| Edit  | ``  | `f044`    | `input.file_path`                                     |
| Write | ``  | `f0c7`    | `input.file_path`                                     |
| Bash  | ``  | `f120`    | `input.description`, fallback to truncated `input.command` |
| Grep  | ``  | `f002`    | `input.pattern` in `input.path`                       |
| Glob  | ``  | `f07b`    | `input.pattern`                                       |
| Task  | ``  | `f085`    | `input.description`                                   |

For unknown tools not in this table, use the fallback icon `` (`f059`, question-circle) and always display the raw tool name alongside it.

## Line Count Metadata

Read, Edit, and Write tool calls display line count metadata after the summary text. This gives quick visibility into the scope of each file operation.

### Read — lines read

Extracted from the `tool_result` content in the corresponding `user` event. The result contains `cat -n` style numbered output. Count the number of lines in the result content string.

Displayed as `(N lines)`.

### Edit — lines added/removed

Computed from the `tool_use` input fields in the `assistant` event:
- Count newlines in `input.old_string` → `old_lines`
- Count newlines in `input.new_string` → `new_lines`
- Added = `new_lines - old_lines` (if positive)
- Removed = `old_lines - new_lines` (if positive)

Displayed as `(+A/-R)`, e.g. `(+3/-1)`. If only additions: `(+3)`. If only removals: `(-2)`. If net zero: `(+2/-2)`.

### Write — lines written

Count the number of lines in `input.content` from the `tool_use` input in the `assistant` event.

Displayed as `(N lines)`.
