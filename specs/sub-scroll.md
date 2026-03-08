# Sub-Scroll for Expanded Content

## Overview

When a tool call is expanded, its full content is displayed without truncation. For large content, the expanded area becomes a scrollable sub-viewport with its own scroll state, entered and exited independently of the main timeline scroll.

This replaces the previous 20-line truncation limit for expanded content.

## Adaptive Sizing

The expanded content area uses adaptive sizing based on the content length relative to the right pane height:

| Content height | Behavior |
|----------------|----------|
| Fits within 40% of pane | Inline display — no sub-scroll, content shown in full |
| Exceeds 40% of pane | Capped at 70% of pane height, sub-scroll enabled |

When sub-scroll is enabled, a scroll position indicator is shown at the bottom-right of the expanded area:

```
   Read   src/main.go (85 lines)  [↑1.2k ⚡812]       ✓   0.8s
      1  package main
      2
      3  import (
      4      "fmt"
      ...
      28     return result
                                                    [28/85]
```

The indicator `[current/total]` is rendered in `ForegroundDim`.

## Sub-Scroll Mode

When an expanded tool call's content exceeds the inline threshold, the user can enter sub-scroll mode:

1. **Entering**: Press `enter` on an already-expanded tool call to enter sub-scroll mode. The expanded area receives a subtle border (using `ForegroundDim`) to indicate it is the active scroll target.
2. **Navigating**: While in sub-scroll mode, `j`/`k`/`↑`/`↓` scroll within the expanded content. `g g` and `G` jump to the top and bottom of the expanded content.
3. **Exiting**: Press `escape` to exit sub-scroll mode and return focus to the main timeline. The cursor remains on the tool call row.

While in sub-scroll mode:
- Main timeline navigation is disabled.
- The scroll position indicator updates in real time.
- The tool call's summary row remains visible (pinned above the sub-viewport).

## Interaction with Other Keys

| Key | Behavior in sub-scroll |
|-----|----------------------|
| `j`/`k`/`↑`/`↓` | Scroll within expanded content |
| `g g` | Jump to top of expanded content |
| `G` | Jump to bottom of expanded content |
| `escape` | Exit sub-scroll, return to timeline |
| `q` | Show quit confirmation (see [quit-confirmation.md](quit-confirmation.md)) |
| `?` | Show help modal (see [help-modal.md](help-modal.md)) |
| All other keys | Ignored while in sub-scroll |

## No Truncation

Expanded content is **never** truncated. The full content of the tool call input or result is displayed:

| Tool | Content shown in full |
|------|----------------------|
| Bash | `$ command` header + full command output |
| Edit | Full diff (see [tui-layout.md](tui-layout.md) for diff format) |
| Read | Full file contents from the tool result |
| Write | Full `content` from the tool input |
| Grep | Full search results |
| Glob | Full matched file list |
| Task | Full task output |
| Other | Full tool result content |
