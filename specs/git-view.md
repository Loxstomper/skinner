# Git View

## Overview

A read-only git history viewer accessible via `ctrl+g`. Shows commit history with side-by-side diffs, syntax highlighted via chroma using the active theme. Designed for reviewing what Claude changed during a session without leaving the TUI.

## Activation

| Key       | Action                                           |
|-----------|--------------------------------------------------|
| `ctrl+g`  | Enter git view                                   |
| `escape`  | Leave git view (returns to previous view state)  |

Entering git view preserves the current run view state (scroll position, selection, focus). Leaving restores it exactly. Git view is a modal overlay — it pushes onto a view stack and pops off on escape.

## Layout

Git view reuses the existing two-pane layout:

- **Left pane**: navigable list (commit list or file list)
- **Right pane**: preview (commit summary or diff)

The same layout rules apply — side layout, bottom layout, and auto switching at the 80-column threshold. The `[` toggle hides/shows the left pane or bottom bar as usual.

## Commit List

The left pane initially shows a list of all commits on the current branch via `git log`.

### Header Line

The first line of the commit list is a centered divider showing total additions and deletions across all commits in the repository:

```
────── +12.3K -4.2K ──────
```

- **Additions**: displayed in `DiffAdded` color
- **Deletions**: displayed in `DiffRemoved` color
- **Divider**: `─` characters in `ForegroundDim`, filling remaining width
- **Loading state**: shows `────── ... ──────` while the async count is in progress

The totals are computed asynchronously via `git log --shortstat --no-merges` over the entire history. A background goroutine runs on git view entry, accumulates totals incrementally, and sends Bubble Tea messages to update the header. The goroutine is cancelled if the user exits git view before it completes. The result is not cached — re-entering git view re-runs the count.

#### Number Formatting

| Value          | Format    | Example  |
|----------------|-----------|----------|
| < 1,000        | raw       | `+42`    |
| 1,000–9,999    | 1 decimal K | `+1.2K`  |
| 10,000–999,999 | whole K   | `+15K`   |
| 1,000,000+     | 1 decimal M | `+1.2M`  |
| 10,000,000+    | whole M   | `+15M`   |

### Display Format

Each commit row shows:

```
a3f  Fix parser edge case       3m ago
```

| Field          | Source                       | Color                |
|----------------|------------------------------|----------------------|
| Short hash     | `git log --format=%h`        | `ForegroundDim`      |
| Subject line   | `git log --format=%s`        | `Foreground`         |
| Relative time  | `git log --format=%cr`       | `ForegroundDim`      |

The hash is truncated to **3 characters**.

### Session Highlighting

Commits made during the current Skinner session are highlighted with a `DiffSessionCommit` foreground color for the subject line. Session start time is recorded when Skinner launches; any commit with an author date after that time is considered a session commit.

### Selected Row

The selected commit row uses `Highlight` background, consistent with other list panes. On the selected row, the relative time is **replaced** with the commit's addition and deletion counts:

```
a3f  Fix parser edge case       +42 -7
```

- **Additions**: `DiffAdded` color
- **Deletions**: `DiffRemoved` color
- Merge commits or commits with no changes show `+0 -0`

## Drill-Down Navigation

```
Commit list          File list           Side-by-side diff
┌──────────┐        ┌──────────┐       ┌─────────────────┐
│ ● a3f2 .. │──→    │ main.go  │──→    │ old  │ new      │
│   b1c4 .. │ enter │ model.go │ enter │ ...  │ ...      │
│   d5e6 .. │       │ tui.go   │       │      │          │
└──────────┘        └──────────┘       └─────────────────┘
    esc ←──────────── esc ←──────────── esc
```

Three depth levels, all within the same two-pane area:

| Depth | Left pane    | Right pane                    | Enter action       | Esc action           |
|-------|-------------|-------------------------------|--------------------|-----------------------|
| 1     | Commit list | Commit message + stats        | Drill into files   | Leave git view        |
| 2     | File list   | Side-by-side diff of selected | Enter sub-scroll   | Back to commit list   |
| 3     | File list   | Diff in sub-scroll mode       | —                  | Exit sub-scroll       |

## Commit Summary (Right Pane, Depth 1)

When a commit is selected but not drilled into, the right pane shows:

```
Fix parser edge case for nested JSON
────────────────────────────────────────
commit a3f2c1b
Author: Claude <noreply@anthropic.com>
Date:   3 minutes ago

    The stream parser was not handling nested braces correctly
    when tool call arguments contained JSON strings.

 main.go    | 12 ++++++------
 parser.go  |  8 ++++----
 2 files changed, 10 insertions(+), 10 deletions(-)
```

- **Subject line header**: the first line of the commit message, displayed bold above a horizontal rule. Uses `DiffSessionCommit` color for commits made during the session, `Foreground` otherwise (matching the left pane commit list styling). The subject is not repeated in the indented message body below.
- **Horizontal rule**: `─` character repeated to pane width, in `ForegroundDim`
- Full commit details from `git show --stat` below the rule
- Stat summary at the bottom with `DiffAdded`/`DiffRemoved` colors for `+`/`-`

## File List (Left Pane, Depth 2)

When drilling into a commit, the left pane switches to the list of changed files.

### Display Format

```
M  main.go       +6 -2
A  newfile.go     +34
D  old.go              -28
```

| Field       | Color           |
|-------------|-----------------|
| Status (M/A/D/R) | `ForegroundDim` |
| Filename    | `Foreground`    |
| Additions   | `DiffAdded`     |
| Deletions   | `DiffRemoved`   |

### Lazy Loading

The file list is loaded immediately on drill-in via `git diff-tree --no-commit-id -r --numstat <sha>`. The actual diff content for each file is loaded only when that file is selected, via `git diff <sha>~1 <sha> -- <file>`.

## Side-by-Side Diff (Right Pane, Depth 2)

When a file is selected in the file list, the right pane shows the diff.

### Layout Switching

| Condition                          | Rendering      |
|------------------------------------|----------------|
| Right pane width >= 80 columns     | Side-by-side   |
| Right pane width < 80 columns      | Unified diff   |

### Side-by-Side Rendering

```
  12 │ func parse(s string) {      │   12 │ func parse(s string) {
  13 │ ░░░tab := strings.Split(s)░░│   13 │ ░░░tab := strings.Fields(s)░
     │                             │   14 │ ░░░if len(tab) == 0 {░░░░░░░
     │                             │   15 │ ░░░░░░░return nil░░░░░░░░░░░
     │                             │   16 │ ░░░}░░░░░░░░░░░░░░░░░░░░░░░
  14 │ for _, v := range tab {     │   17 │ for _, v := range tab {
```

- **Left column**: old file content with old line numbers
- **Right column**: new file content with new line numbers
- **Separator**: `│` character in `ForegroundDim`
- **Line numbers**: displayed in `DiffLineNumber` color, right-aligned in a gutter

### Line Pairing

Unified diff hunks are split into paired rows:

- **Context lines**: appear on both sides, identical
- **Removed then added block**: pair `-` and `+` lines row by row. If the counts are unequal, the shorter side gets blank/empty rows
- **Remove-only or add-only block**: content on one side, blank on the other

### Line Backgrounds

| Line type | Background color     |
|-----------|---------------------|
| Added     | `DiffAddedBg`       |
| Removed   | `DiffRemovedBg`     |
| Context   | none (terminal default) |

### Intra-Line Highlighting

For paired changed lines (a removed line matched with an added line), compute character-level differences using the Myers diff algorithm (`sergi/go-diff`). The changed characters within each line are highlighted with emphasis colors:

| Highlight      | Background color        |
|----------------|------------------------|
| Changed chars (removed side) | `DiffRemovedEmphasis`  |
| Changed chars (added side)   | `DiffAddedEmphasis`    |

Unchanged characters within the line keep the base `DiffRemovedBg`/`DiffAddedBg` background.

### Syntax Highlighting

File content is syntax highlighted using chroma's token iterator. The language is detected from the filename extension.

Color layering order:
1. **Background**: diff status (`DiffAddedBg`, `DiffRemovedBg`, or terminal default)
2. **Foreground**: chroma syntax colors, mapped through the active theme's chroma style
3. **Emphasis overlay**: intra-line change backgrounds (`DiffAddedEmphasis`, `DiffRemovedEmphasis`) replace the diff background for changed characters

The chroma style used is derived from the active theme:

| Theme           | Chroma style       |
|-----------------|--------------------|
| solarized-dark  | `solarized-dark`   |
| solarized-light | `solarized-light`  |
| monokai         | `monokai`          |
| nord            | `nord`             |

### Unified Diff Rendering (Narrow)

When the right pane is narrower than 80 columns, a standard unified diff is rendered:

```
@@ -12,3 +12,7 @@
  12 │ func parse(s string) {
- 13 │ ░░░tab := strings.Split(s)░░
+ 13 │ ░░░tab := strings.Fields(s)░
+ 14 │ ░░░if len(tab) == 0 {░░░░░░░
+ 15 │ ░░░░░░░return nil░░░░░░░░░░░
+ 16 │ ░░░}░░░░░░░░░░░░░░░░░░░░░░░
  17 │ for _, v := range tab {
```

Same coloring rules apply — line backgrounds, intra-line highlighting, and syntax highlighting.

## Scrolling

### Vertical Scroll

| Key            | Action                |
|----------------|-----------------------|
| `j` / `↓`     | Scroll down           |
| `k` / `↑`     | Scroll up             |
| Mouse wheel    | Scroll (3 lines/tick) |
| `gg` / `Home`  | Jump to top           |
| `G` / `End`    | Jump to bottom        |

In the diff view, both sides scroll in sync — there is a single scroll position applied to both columns.

### Horizontal Scroll

| Key    | Action                              |
|--------|-------------------------------------|
| `h`    | Scroll left (when diff is focused)  |
| `l`    | Scroll right (when diff is focused) |

Horizontal scroll applies to both sides simultaneously. Long lines are truncated at the pane edge; horizontal scroll reveals the rest. The line number gutter does not scroll — only the code content.

Note: `h`/`l` are repurposed for horizontal scroll only within the diff view at depth 2/3. At depth 1 (commit list), they retain their normal focus-switching behavior.

## Live Updating

The commit list refreshes automatically to pick up new commits made during the session. A `git log` poll runs every 5 seconds. New commits appear at the top of the list. If the user has not manually scrolled the commit list, it auto-follows to show the latest commit. Manual selection pauses auto-follow (same pattern as iteration auto-follow).

## Theme Colors

The following semantic color roles are added for the git view:

| Role                  | Used for                                              |
|-----------------------|-------------------------------------------------------|
| `DiffAdded`           | Addition count text (`+42`), stat `+` characters      |
| `DiffRemoved`         | Deletion count text (`-7`), stat `-` characters       |
| `DiffAddedBg`         | Background for added lines in diff                    |
| `DiffRemovedBg`       | Background for removed lines in diff                  |
| `DiffAddedEmphasis`   | Background for changed characters on added lines      |
| `DiffRemovedEmphasis` | Background for changed characters on removed lines    |
| `DiffLineNumber`      | Line number gutter in diff view                       |
| `DiffSessionCommit`   | Subject line color for commits made during session     |

### Built-in Theme Values

#### Solarized Dark

| Role                  | Hex       | Note                        |
|-----------------------|-----------|-----------------------------|
| `DiffAdded`           | `#859900` | solarized green             |
| `DiffRemoved`         | `#dc322f` | solarized red               |
| `DiffAddedBg`         | `#1a3a1a` | muted green background      |
| `DiffRemovedBg`       | `#3a1a1a` | muted red background        |
| `DiffAddedEmphasis`   | `#2d5a2d` | stronger green background   |
| `DiffRemovedEmphasis` | `#5a2d2d` | stronger red background     |
| `DiffLineNumber`      | `#586e75` | base01                      |
| `DiffSessionCommit`   | `#268bd2` | solarized blue              |

#### Solarized Light

| Role                  | Hex       | Note                        |
|-----------------------|-----------|-----------------------------|
| `DiffAdded`           | `#859900` | solarized green             |
| `DiffRemoved`         | `#dc322f` | solarized red               |
| `DiffAddedBg`         | `#e6f2e6` | light green background      |
| `DiffRemovedBg`       | `#f2e6e6` | light red background        |
| `DiffAddedEmphasis`   | `#c8e6c8` | stronger green background   |
| `DiffRemovedEmphasis` | `#e6c8c8` | stronger red background     |
| `DiffLineNumber`      | `#93a1a1` | base1                       |
| `DiffSessionCommit`   | `#268bd2` | solarized blue              |

#### Monokai

| Role                  | Hex       | Note                        |
|-----------------------|-----------|-----------------------------|
| `DiffAdded`           | `#a6e22e` | monokai green               |
| `DiffRemoved`         | `#f92672` | monokai red                 |
| `DiffAddedBg`         | `#2a3a1a` | muted green background      |
| `DiffRemovedBg`       | `#3a1a2a` | muted red background        |
| `DiffAddedEmphasis`   | `#3d5a2d` | stronger green background   |
| `DiffRemovedEmphasis` | `#5a2d3d` | stronger red background     |
| `DiffLineNumber`      | `#75715e` | monokai comment             |
| `DiffSessionCommit`   | `#66d9ef` | monokai cyan                |

#### Nord

| Role                  | Hex       | Note                        |
|-----------------------|-----------|-----------------------------|
| `DiffAdded`           | `#a3be8c` | nord14                      |
| `DiffRemoved`         | `#bf616a` | nord11                      |
| `DiffAddedBg`         | `#2e3440` | nord0 + green tint          |
| `DiffRemovedBg`       | `#3b2c2f` | nord0 + red tint            |
| `DiffAddedEmphasis`   | `#3a4a3a` | stronger green tint         |
| `DiffRemovedEmphasis` | `#4a3a3a` | stronger red tint           |
| `DiffLineNumber`      | `#4c566a` | nord3                       |
| `DiffSessionCommit`   | `#88c0d0` | nord8                       |
