# Token Usage

## Overview

Token usage is displayed in two places:

1. **Header bar** — API rate limit window utilization (5-hour and weekly).
2. **Tool call rows** — per-tool-call token counts (cached vs real), shown inline.

## Header — Rate Limit Windows

> **Status: Placeholder.** The header area is reserved but the data source and fetching mechanism are deferred to a future implementation pass.

The header bar includes a token usage area showing the current utilization of Anthropic's rate limit windows:

```
  ⏱ 14m32s   ↑42.1k ↓8.3k tokens   ctx 62%   ~$1.24   5h: 34%  wk: 12%   Iter 3/10 ⟳
```

- **5h: N%** — utilization of the 5-hour rolling token window.
- **wk: N%** — utilization of the weekly rolling token window.

Both values are displayed in `ForegroundDim` by default. Color thresholds match the context window display:
- Normal (`ForegroundDim`) — 0–69%
- Warning (`StatusRunning`) — 70–89%
- Critical (`StatusError`) — 90%+

### Data Source (Deferred)

The Claude CLI exposes current usage via the `/usage` command. The implementation should:

- Fetch usage data lazily at the start of each iteration (not real-time).
- Parse the response to extract 5-hour and weekly window utilization percentages.
- Display placeholder text (`5h: --  wk: --`) until the first fetch completes.
- If the fetch fails, continue displaying `--` without blocking the iteration.

The exact mechanism (shelling out to `claude /usage`, direct API call, or other) is to be determined during implementation.

## Tool Call Rows — Per-Call Token Counts

Each tool call row displays an approximate token count showing cached vs real (non-cached) input tokens. This appears inline after the existing metadata:

```
   Read   src/main.go (85 lines)  [↑1.2k ⚡812]       ✓   0.8s
   Bash   go test ./...           [↑340 ⚡28.1k]       ✗   4.5s
```

- **↑N** — real (non-cached) input tokens attributed to this tool call.
- **⚡N** — cached input tokens attributed to this tool call.
- Displayed in `ForegroundDim`, placed between the line count metadata and the result indicator.
- Token counts use the same formatting as the header: `k` suffix for thousands, `M` for millions, no suffix under 1000.

### Attribution

Tokens are reported per assistant turn in the stream-json `assistant` event's `message.usage`, not per individual tool call. When a turn contains multiple tool calls, tokens are attributed approximately:

- Divide the turn's token counts equally across all tool calls in that turn.
- Round to the nearest integer.

This is an approximation — the actual token cost of each tool call varies — but provides useful directional information.

### Fields Used

From `message.usage` in `assistant` events:

| Field | Attribution |
|-------|------------|
| `input_tokens` | Real input tokens (split across tool calls) |
| `cache_read_input_tokens` | Cached tokens (split across tool calls) |

`output_tokens` and `cache_creation_input_tokens` are not shown per-tool-call (they are already tracked in the header totals).

### Timing

Token counts for a tool call are available immediately when the `assistant` event is parsed (since `message.usage` is on the assistant event, not the user/result event). They can be displayed as soon as the tool call row appears.
