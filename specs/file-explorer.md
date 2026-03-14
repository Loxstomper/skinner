# File Explorer

## Overview

A read-only file explorer accessible via `f`. Shows a navigable file tree in the left pane with syntax-highlighted file previews in the right pane. Designed for browsing the project without leaving the TUI.

## Activation

| Key      | Action                                          |
|----------|-------------------------------------------------|
| `f`      | Enter file explorer (configurable via keybindings) |
| `escape` | Leave file explorer (returns to previous view state) |

Entering file explorer preserves the current run view state (scroll position, selection, focus). Leaving restores it exactly. File explorer is a modal overlay — it pushes onto a view stack and pops off on escape.

## Layout

File explorer reuses the existing two-pane layout:

- **Left pane**: navigable file tree
- **Right pane**: file preview

The same layout rules apply — side layout, bottom layout, and auto switching at the 80-column threshold. The `[` toggle hides/shows the left pane or bottom bar as usual.

## File Tree (Left Pane)

### Source

The file tree is built from a recursive filesystem walk starting at the current working directory. The `.git/` directory is always skipped. All other files and directories are included, including dotfiles.

### Refresh

The filesystem walk re-runs on a 5-second timer to pick up additions, removals, and renames. If the user is mid-search when a refresh triggers, the refresh is deferred until the search exits.

### Display Format

```
▼ internal/
  ▶ tui/
  ▶ model/
▼ cmd/
    main.go          ?
  go.mod
  go.sum             M
  CLAUDE.md
  config.yml → ../config.yml  🔗
```

| Element          | Display              | Color             |
|------------------|----------------------|-------------------|
| Collapsed dir    | `▶ dirname/`         | `Foreground`      |
| Expanded dir     | `▼ dirname/`         | `Foreground`      |
| File             | `filename`           | `Foreground`      |
| Symlink indicator | `🔗` after target    | `ForegroundDim`   |
| Symlink target   | `→ target`           | `ForegroundDim`   |
| Indent           | 2 spaces per depth level |               |

### Sorting

Within each directory level:
1. Directories first, sorted alphabetically (case-insensitive)
2. Files second, sorted alphabetically (case-insensitive)

### Initial State

On entry, the first level of the tree (CWD direct children) is expanded. All nested directories are collapsed.

### Git Status Indicators

Git status is displayed right-aligned on each file row:

| Indicator | Meaning     | Color         |
|-----------|-------------|---------------|
| `M`       | Modified    | `ForegroundDim` |
| `A`       | Added       | `DiffAdded`   |
| `D`       | Deleted     | `DiffRemoved` |
| `?`       | Untracked   | `DiffAdded`   |

Git status is sourced from `git status --porcelain` and refreshes on the same 5-second timer as the filesystem walk. Directories inherit a status indicator if any child has a status (showing the highest-priority status: `D` > `M` > `A` > `?`).

### Selection

- Same highlight style as other list panes: theme's `Highlight` background, padded to full width.
- Highlight only shown when the file tree is focused.

### Symlinks

Symlinks are followed for preview purposes. The display shows the symlink name with a `🔗` indicator and the target path as `→ target` in `ForegroundDim`.

### Empty State

When the CWD contains no files (excluding `.git/`):

```
  No files
```

"No files" is rendered in `ForegroundDim`.

## Drill-Down Navigation

```
File tree            File preview (read-only)    Scrollable preview
┌──────────┐        ┌──────────────────┐        ┌──────────────────┐
│ ▼ src/    │       │  1 package main  │        │  1 package main  │
│   main.go │  ←→   │  2               │  enter │  2               │
│   util.go │       │  3 import (      │   →    │  3 import (      │
└──────────┘        └──────────────────┘        └──────────────────┘
                                                   esc ←
```

Two depth levels, within the same two-pane area:

| Depth | Left pane  | Right pane                  | Enter action         | Escape action         |
|-------|------------|-----------------------------|----------------------|-----------------------|
| 1     | File tree  | Read-only preview of selected file | Enter scrollable mode | Leave file explorer |
| 2     | File tree  | Scrollable file content      | —                    | Back to depth 1       |

## Tree Navigation (Depth 1)

Standard navigation keys operate on the visible (expanded) rows:

| Key          | Action                                              |
|--------------|-----------------------------------------------------|
| `j` / `↓`   | Move cursor down                                    |
| `k` / `↑`   | Move cursor up                                      |
| `gg` / `Home`| Jump to top                                        |
| `G` / `End` | Jump to bottom                                      |
| `pgdn`      | Page scroll down                                    |
| `pgup`      | Page scroll up                                      |

Tree-specific keys:

| Key          | On file                        | On expanded dir              | On collapsed dir             |
|--------------|--------------------------------|------------------------------|------------------------------|
| `enter`      | Enter depth 2 (scrollable)     | Collapse directory           | Expand directory             |
| `l` / `→`   | Enter depth 2 (scrollable)     | Move cursor to first child   | Expand directory             |
| `h` / `←`   | Collapse parent, cursor to parent | Collapse directory         | Collapse parent, cursor to parent |
| `e`          | Open `$EDITOR`                 | —                            | —                            |

`h` on a root-level item is a no-op.

## File Preview (Right Pane)

### Title Bar

A title bar at the top of the right pane showing the relative file path (e.g. `internal/tui/root.go`) centered. Styled with bold text in theme `Foreground` color.

### Content

File preview is lazy-loaded — content is read only when the cursor moves to a new file. The preview updates as the cursor moves through the tree.

| File type       | Rendering                                          |
|-----------------|----------------------------------------------------|
| Markdown (`.md`) | Rendered via [glamour](https://github.com/charmbracelet/glamour) with `auto` style |
| Source code     | Syntax highlighted via chroma, language detected from file extension |
| Binary files    | "Binary file — preview not available" in `ForegroundDim` |
| Directories     | Empty (no preview)                                 |

Binary file detection: attempt to read the first 512 bytes and check for null bytes.

File content and rendered output are cached via [render-cache.md](render-cache.md) to avoid re-reading and re-rendering on every frame. For markdown files, this caches the glamour output. For source code files, this caches the raw lines (chroma tokenization of visible lines still runs per-frame as it is already O(visible)).

### Line Numbers

Line numbers are displayed in the preview gutter, styled in `ForegroundDim`. The `#` key toggles line numbers on/off (reusing the existing line number infrastructure from [line-numbers.md](line-numbers.md)).

Line numbers are not shown for glamour-rendered markdown.

### File Not Found

If a file is deleted externally while being previewed, the right pane shows "File not found" in `ForegroundDim`.

## Scrollable Preview (Depth 2)

When the user presses `enter` on a file at depth 1, the preview becomes scrollable:

| Key            | Action                         |
|----------------|--------------------------------|
| `j` / `↓`     | Scroll down one line           |
| `k` / `↑`     | Scroll up one line             |
| `h`           | Scroll left (long lines)        |
| `l`           | Scroll right (long lines)       |
| `gg` / `Home` | Scroll to top                  |
| `G` / `End`   | Scroll to bottom               |
| `pgdn`        | Page scroll down               |
| `pgup`        | Page scroll up                 |
| `#`           | Toggle line numbers             |
| `e`           | Open `$EDITOR` for this file   |
| `escape`      | Exit scrollable mode (depth 1) |

While in depth 2:
- Tree navigation in the left pane is disabled.
- The line number gutter does not scroll horizontally — only the code content.

## Fuzzy Search

### Activation

Pressing `/` while the file tree is focused opens a search input at the bottom of the left pane.

### Input Bar

```
┌────────────────────────────┐
│ ▼ internal/                │
│   ▶ tui/                   │
│   main.go                  │
│                            │
│ / root.go█                 │  ← search input
└────────────────────────────┘
```

The input bar shows `/` as a prefix followed by the query text. Styled with `Foreground` for the prefix and input text.

### Matching

Uses in-process fuzzy matching (`sahilm/fuzzy` or equivalent Go library). The full flat list of file paths (relative to CWD) is searched. Results are ranked by match quality.

### Behavior

- The tree view is replaced by the flat ranked result list while searching.
- The preview live-updates as the top match (or selected match) changes.
- `j` / `k` or `↓` / `↑` move through matches.
- `enter` confirms: dismisses search, tree restores with the selected file's parent directories expanded, cursor on the file.
- `escape` cancels: dismisses search, tree restores to pre-search state.
- Typing updates the filter in real time.

## Editor Integration

| Key | Context                    | Action                                    |
|-----|----------------------------|-------------------------------------------|
| `e` | File selected (depth 1)    | Suspend TUI, open `$EDITOR` (fallback: `vi`) with file path |
| `e` | Scrollable preview (depth 2) | Same — suspend TUI, open `$EDITOR`       |

On editor exit: TUI resumes, file preview is re-rendered with updated content. Scroll position resets to top if the file was modified.

## Scrolling

When the tree is taller than the left pane, standard scroll-to-cursor behavior keeps the cursor visible (same pattern as the iteration list).

## Mouse Support

### File Tree (Left Pane)

- **Scroll**: mouse wheel scrolls the tree, 3 lines per tick.
- **Click**: single click on a file selects it and updates preview. Single click on a directory toggles expand/collapse.
- **Focus**: any mouse interaction switches focus to the file tree.

### File Preview (Right Pane)

- **Scroll**: mouse wheel scrolls the preview (enters depth 2 if not already). 3 lines per tick, clamped to content bounds.
- **Focus**: scrolling the preview switches focus to the right pane.

## Theme Colors

No new theme colors are required. File explorer reuses:

| Existing color    | Used for                                          |
|-------------------|---------------------------------------------------|
| `Foreground`      | Filenames, directory names, tree arrows            |
| `ForegroundDim`   | Git `M` status, symlink indicators, line numbers, binary file message |
| `DiffAdded`       | Git `A` and `?` status indicators                  |
| `DiffRemoved`     | Git `D` status indicator                           |
| `Highlight`       | Selected row background                            |

Syntax highlighting uses the active theme's chroma style (same mapping as [git-view.md](git-view.md)).
