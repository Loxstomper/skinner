# Color Themes

## Overview

All TUI colors are driven by a theme. The theme does **not** set a global background color — it respects the terminal's own background to avoid conflicts with transparency and terminal themes.

## CLI Flag

```
skinner --theme=<name> [plan] [max_iterations]
```

Default: `solarized-dark`.

If an invalid theme name is provided, print available themes and exit:

```
Unknown theme "foo". Available themes:
  solarized-dark (default)
  solarized-light
  monokai
  nord
```

## Theme Structure

Each theme defines hex color values for the following semantic roles:

| Role              | Used for                                           |
|-------------------|----------------------------------------------------|
| `Foreground`      | Default text                                       |
| `ForegroundDim`   | Muted/secondary text, pane separator               |
| `Highlight`       | Selected row background (both panes)               |
| `StatusRunning`   | `⟳` icon                                          |
| `StatusSuccess`   | `✓` icon                                          |
| `StatusError`     | `✗` icon                                          |
| `ToolNameRunning` | Tool name column while call is in progress         |
| `ToolNameSuccess` | Tool name column on success                        |
| `ToolNameError`   | Tool name column on error                          |
| `ToolSummary`     | Arg summary text in tool call rows (always neutral)|
| `DurationRunning` | Duration column while call is in progress          |
| `DurationSuccess` | Duration column on success                         |
| `DurationError`   | Duration column on error                           |
| `IterRunning`     | Running iteration text in left pane                |
| `IterSuccess`     | Completed iteration text in left pane              |
| `IterError`       | Failed iteration text in left pane                 |
| `TextBlock`       | Claude's reasoning/response text blocks            |

## Rendering Rules

- **Status icons** (`✓`, `✗`, `⟳`): colored per state.
- **Tool icon**: colored per state (running/success/error), same `ToolName*` roles.
- **Tool name**: colored per state (running/success/error).
- **Arg summary**: always dim (`ToolSummary`).
- **Duration**: colored per state.
- **Iteration list entries**: colored per state.
- **Text blocks**: `TextBlock` color.
- **Pane separator**: `ForegroundDim`.
- **Selected row background**: `Highlight`.
- **Global background**: not set (terminal default).

## Built-in Themes

### Solarized Dark (default)

Based on Ethan Schoonover's Solarized palette.

| Role              | Hex       | Solarized name |
|-------------------|-----------|----------------|
| `Foreground`      | `#839496` | base0          |
| `ForegroundDim`   | `#586e75` | base01         |
| `Highlight`       | `#073642` | base02         |
| `StatusRunning`   | `#b58900` | yellow         |
| `StatusSuccess`   | `#859900` | green          |
| `StatusError`     | `#dc322f` | red            |
| `ToolNameRunning` | `#b58900` | yellow         |
| `ToolNameSuccess` | `#859900` | green          |
| `ToolNameError`   | `#dc322f` | red            |
| `ToolSummary`     | `#586e75` | base01         |
| `DurationRunning` | `#b58900` | yellow         |
| `DurationSuccess` | `#2aa198` | cyan           |
| `DurationError`   | `#dc322f` | red            |
| `IterRunning`     | `#b58900` | yellow         |
| `IterSuccess`     | `#839496` | base0          |
| `IterError`       | `#dc322f` | red            |
| `TextBlock`       | `#839496` | base0          |

### Solarized Light

Inverted variant of Solarized.

| Role              | Hex       | Solarized name |
|-------------------|-----------|----------------|
| `Foreground`      | `#657b83` | base00         |
| `ForegroundDim`   | `#93a1a1` | base1          |
| `Highlight`       | `#eee8d5` | base2          |
| `StatusRunning`   | `#b58900` | yellow         |
| `StatusSuccess`   | `#859900` | green          |
| `StatusError`     | `#dc322f` | red            |
| `ToolNameRunning` | `#b58900` | yellow         |
| `ToolNameSuccess` | `#859900` | green          |
| `ToolNameError`   | `#dc322f` | red            |
| `ToolSummary`     | `#93a1a1` | base1          |
| `DurationRunning` | `#b58900` | yellow         |
| `DurationSuccess` | `#2aa198` | cyan           |
| `DurationError`   | `#dc322f` | red            |
| `IterRunning`     | `#b58900` | yellow         |
| `IterSuccess`     | `#657b83` | base00         |
| `IterError`       | `#dc322f` | red            |
| `TextBlock`       | `#657b83` | base00         |

### Monokai

Warm, vibrant palette popular in editors.

| Role              | Hex       |
|-------------------|-----------|
| `Foreground`      | `#f8f8f2` |
| `ForegroundDim`   | `#75715e` |
| `Highlight`       | `#49483e` |
| `StatusRunning`   | `#e6db74` |
| `StatusSuccess`   | `#a6e22e` |
| `StatusError`     | `#f92672` |
| `ToolNameRunning` | `#e6db74` |
| `ToolNameSuccess` | `#a6e22e` |
| `ToolNameError`   | `#f92672` |
| `ToolSummary`     | `#75715e` |
| `DurationRunning` | `#e6db74` |
| `DurationSuccess` | `#66d9ef` |
| `DurationError`   | `#f92672` |
| `IterRunning`     | `#e6db74` |
| `IterSuccess`     | `#f8f8f2` |
| `IterError`       | `#f92672` |
| `TextBlock`       | `#f8f8f2` |

### Nord

Cool blue-grey, minimal contrast palette.

| Role              | Hex       | Nord name    |
|-------------------|-----------|--------------|
| `Foreground`      | `#d8dee9` | nord4        |
| `ForegroundDim`   | `#4c566a` | nord3        |
| `Highlight`       | `#3b4252` | nord1        |
| `StatusRunning`   | `#ebcb8b` | nord13       |
| `StatusSuccess`   | `#a3be8c` | nord14       |
| `StatusError`     | `#bf616a` | nord11       |
| `ToolNameRunning` | `#ebcb8b` | nord13       |
| `ToolNameSuccess` | `#a3be8c` | nord14       |
| `ToolNameError`   | `#bf616a` | nord11       |
| `ToolSummary`     | `#4c566a` | nord3        |
| `DurationRunning` | `#ebcb8b` | nord13       |
| `DurationSuccess` | `#88c0d0` | nord8        |
| `DurationError`   | `#bf616a` | nord11       |
| `IterRunning`     | `#ebcb8b` | nord13       |
| `IterSuccess`     | `#d8dee9` | nord4        |
| `IterError`       | `#bf616a` | nord11       |
| `TextBlock`       | `#d8dee9` | nord4        |
