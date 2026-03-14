# Tasks View

## Overview

A read-only tasks viewer accessible via `t`. Displays beads (bd) issues in a two-pane layout with tab filtering, tree navigation, and fuzzy search. Designed for tracking project work without leaving the TUI.

## Activation

| Key      | Action                                          |
|----------|-------------------------------------------------|
| `t`      | Enter tasks view (configurable via keybindings) |
| `escape` | Leave tasks view (returns to previous view state) |

Entering tasks view preserves the current run view state (scroll position, selection, focus). Leaving restores it exactly. Tasks view is a modal overlay — it pushes onto a view stack and pops off on escape (same pattern as git view and file explorer).

## Layout

Tasks view uses the existing two-pane layout:

- **Left pane**: navigable issue list (tree or flat), fixed 32-character width
- **Right pane**: issue detail

The same layout rules apply — side layout, bottom layout, and auto switching at the 80-column threshold. The `[` toggle hides/shows the left pane or bottom bar as usual.

### Header Bar

A full-width tab bar sits above the two-pane area, replacing the normal header while tasks view is active:

```
 Ready (3)    All (12)    Blocked (2)    In Progress (4)
─────────────────────────────────────────────────────────
```

- Active tab is rendered in `Foreground` with bold text and an underline.
- Inactive tabs are rendered in `ForegroundDim`.
- Counts in parentheses reflect the current filter (including search).
- `H` / `L` switch between tabs (wrapping at edges).
- A horizontal rule (`─` in `ForegroundDim`) separates the tab bar from the panes.

## Data Model

### Fetching

A single `bd list --json --limit 0` call loads all issues on entry. Per-tab views are derived in-memory:

| Tab         | Source                                          |
|-------------|-------------------------------------------------|
| Ready       | `bd ready --json`                               |
| All         | `bd list --json --limit 0`                      |
| Blocked     | `bd list --json --status blocked`               |
| In Progress | `bd list --json --status in_progress`           |

On entry, all four commands run concurrently. Results are cached in-memory until manual refresh (`r`).

### In-Memory Graph

Issues are indexed by ID in a `map[string]*Issue` for O(1) lookup. Parent-child relationships are derived from each issue's `parent` field. The graph supports:

- **Parent-child tree**: used for the tree list in the left pane.
- **Transitive dependency walker**: follows `dependencies` (type `blocks`) up to depth 3, with cycle detection. Used for the blocking dependency tree in the right pane detail.
- **Fuzzy search**: matches against title, ID, and description fields.

### Issue Struct

Matches the bd JSON schema:

| Field          | Type            | JSON key         |
|----------------|-----------------|------------------|
| ID             | string          | `id`             |
| Title          | string          | `title`          |
| Description    | string          | `description`    |
| Status         | string          | `status`         |
| Priority       | int             | `priority`       |
| IssueType      | string          | `issue_type`     |
| Assignee       | string          | `assignee`       |
| Owner          | string          | `owner`          |
| CreatedBy      | string          | `created_by`     |
| Labels         | []string        | `labels`         |
| Parent         | string          | `parent`         |
| Dependencies   | []Dependency    | `dependencies`   |
| Dependents     | []Dependency    | `dependents`     |
| CreatedAt      | time.Time       | `created_at`     |
| UpdatedAt      | time.Time       | `updated_at`     |
| ClosedAt       | time.Time       | `closed_at`      |
| CloseReason    | string          | `close_reason`   |
| Gates          | []Gate          | `gates`          |
| RelatesTo      | []string        | `relates_to`     |
| Metadata       | map[string]any  | `metadata`       |
| ExternalRef    | string          | `external_ref`   |

## Left Pane — Issue List

### Tree Mode (Default)

Issues are displayed as a tree rooted at top-level issues (those with no parent). Child issues are indented 2 spaces per depth level. All nodes are expanded by default.

```
● 1 skinner-dfe  Tasks View TUI
  ◐ 1 skinner-pne  Write tasks-view spec
  ◌ 2 skinner-zz3  Define Issue struct
◇ 1 skinner-5mq  bd CLI integration
```

### Flat Mode

Pressing `f` toggles flat mode — all issues are shown in a flat list without indentation, sorted in bd default sort order. Pressing `f` again returns to tree mode.

### Display Format

Each row shows:

```
{status} {priority} {id}  {title}
```

| Element    | Description                                       |
|------------|---------------------------------------------------|
| Status icon | Single character reflecting issue status          |
| Priority   | Single digit (0–4)                                |
| ID         | Issue ID (e.g. `skinner-dfe`)                     |
| Title      | Truncated to fit available width                  |

### Status Icons

| Icon | Status       | Color           |
|------|-------------|-----------------|
| `●`  | open        | `ForegroundDim` |
| `◐`  | in_progress | `StatusRunning` |
| `◌`  | ready       | `Foreground`    |
| `◇`  | blocked     | `StatusError`   |
| `✓`  | closed      | `StatusSuccess` |

### Type Coloring

The issue title is colored by issue type:

| Type     | Color                |
|----------|----------------------|
| bug      | `StatusError` (red)  |
| feature  | `StatusSuccess` (green) |
| task     | `Foreground` (default) |
| epic     | magenta (`#d33682` solarized) |
| chore    | yellow (`#b58900` solarized) |
| decision | cyan (`#2aa198` solarized) |

For themes other than solarized, the closest semantic color in the palette is used (e.g. nord14 for feature, nord11 for bug).

### Selection

- Same highlight style as other list panes: theme's `Highlight` background, padded to full width.
- Highlight only shown when the left pane is focused.

### Expand/Collapse (Tree Mode)

| Key     | Action                                          |
|---------|-------------------------------------------------|
| `space` | Toggle expand/collapse on the selected node     |

Collapsed nodes show a `▶` prefix. Expanded nodes show no prefix (children are visible below). Collapsing a node hides all descendants. The cursor skips over hidden children.

### Empty State

When no issues match the current tab and search:

```
  No issues
```

"No issues" is rendered in `ForegroundDim`.

## Right Pane — Issue Detail

When an issue is selected in the left pane, the right pane shows its full detail. Sections with no data are hidden entirely.

### Title + Priority

```
[P1] Write tasks-view spec
────────────────────────────────────────
```

- Priority badge `[P{n}]` in bold, colored by priority level:
  - P0: `StatusError` (critical)
  - P1: `StatusRunning` (high)
  - P2: `Foreground` (medium)
  - P3: `ForegroundDim` (low)
  - P4: `ForegroundDim` (backlog)
- Title in bold `Foreground`.
- Horizontal rule (`─` in `ForegroundDim`) below.

### Meta Line

```
task  ◐ in_progress  assigned: Lochie Ashcroft  parent: skinner-dfe
```

- Issue type, status icon + status text, assignee (if set), parent ID (if set).
- All in `ForegroundDim`.

### Labels

```
Labels: backend, v2
```

Comma-separated, in `ForegroundDim`. Hidden if no labels.

### Description

Full description text rendered in `Foreground`. Markdown is rendered via glamour with `auto` style.

### Blocking Dependencies

A tree of blocking dependencies (type `blocks`), walked transitively up to depth 3 with cycle detection:

```
Blocked by
├── ◐ skinner-pne  Write tasks-view spec
│   └── ● skinner-abc  Some upstream task
└── ◇ skinner-xyz  Another blocker
```

- Tree drawing uses `├──`, `└──`, `│` characters in `ForegroundDim`.
- Each node shows status icon + ID + title.
- Cycle detected: append `(cycle)` in `ForegroundDim` and stop recursion.
- Hidden if no blocking dependencies.

### Related

Non-blocking dependencies (`relates_to`, `discovered-from`, `parent-child`) listed flat:

```
Related
  parent: skinner-dfe  Tasks View TUI
  discovered-from: skinner-abc  Some other task
```

Hidden if no non-blocking relationships.

### Gates

```
Gates
  ☐ code-review
  ☑ tests-pass
```

- `☐` for incomplete gates, `☑` for complete.
- Hidden if no gates.

### Timestamps

```
Created  2026-03-14 12:49
Updated  2026-03-14 12:57
Closed   2026-03-14 13:00  "All tasks completed"
```

- Timestamps formatted as `YYYY-MM-DD HH:MM`.
- Close reason shown in quotes after the closed timestamp (if present).
- `ForegroundDim` for labels, `Foreground` for values.

## Depth Navigation

| Depth | Left pane          | Right pane             | Enter action           | Escape action        |
|-------|--------------------|------------------------|------------------------|----------------------|
| 0     | Issue list (focused) | Issue detail (static) | Focus right pane (depth 1) | Leave tasks view |
| 1     | Issue list (unfocused) | Detail (scrollable) | —                      | Back to depth 0     |

At depth 0, the left pane is focused and the right pane updates as the cursor moves but is not scrollable. At depth 1, the right pane becomes scrollable with `j`/`k` and the left pane cursor is frozen.

## Tab Filtering

| Tab         | Filter                               |
|-------------|--------------------------------------|
| Ready       | Issues with status `open` and no unresolved blocking dependencies |
| All         | All issues regardless of status      |
| Blocked     | Issues with status `blocked`         |
| In Progress | Issues with status `in_progress`     |

| Key | Action                |
|-----|-----------------------|
| `H` | Switch to previous tab |
| `L` | Switch to next tab     |

Switching tabs:
- Resets the cursor to position 0.
- Clears any active search.
- Preserves tree/flat mode.
- Uses bd default sort order within each tab.

## Fuzzy Search

### Activation

Pressing `/` while at depth 0 opens a search input at the top of the left pane.

### Input Bar

```
┌────────────────────────────────┐
│ / tasks-view█                  │
│ ◐ 1 skinner-pne  Write tasks  │
│ ● 2 skinner-zz3  Define Issue │
│                                │
└────────────────────────────────┘
```

The input bar shows `/` as a prefix followed by the query text. Styled with `Foreground` for the prefix and input text.

### Matching

Uses in-process fuzzy matching against issue title, ID, and description. Results are ranked by match quality. The matching function is provided by the `internal/bd` package's graph.

### Behavior

- The tree view is replaced by a flat ranked result list while searching.
- The detail pane live-updates as the top match (or selected match) changes.
- `j` / `k` or `↓` / `↑` move through matches.
- `enter` confirms: dismisses search, tree restores with the selected issue highlighted.
- `escape` cancels: dismisses search, restores pre-search state.
- Typing updates the filter in real time.
- Tab counts update to reflect the intersection of search results with each tab filter.

## Keybindings

### Navigation (Depth 0 — List Focused)

| Key              | Action                                      |
|------------------|---------------------------------------------|
| `j` / `↓`       | Move cursor down                            |
| `k` / `↑`       | Move cursor up                              |
| `gg` / `Home`   | Jump to top                                 |
| `G` / `End`     | Jump to bottom                              |
| `enter`          | Enter depth 1 (detail scrollable)           |
| `escape`         | Leave tasks view                            |
| `space`          | Toggle expand/collapse (tree mode)          |
| `H`              | Switch to previous tab                      |
| `L`              | Switch to next tab                          |
| `/`              | Activate fuzzy search                       |
| `f`              | Toggle flat/tree mode                       |
| `r`              | Manual refresh (re-fetch from bd)           |

### Navigation (Depth 1 — Detail Scrollable)

| Key              | Action                                      |
|------------------|---------------------------------------------|
| `j` / `↓`       | Scroll detail down                          |
| `k` / `↑`       | Scroll detail up                            |
| `gg` / `Home`   | Scroll to top                               |
| `G` / `End`     | Scroll to bottom                            |
| `escape`         | Back to depth 0                             |

## Refresh

Pressing `r` triggers a full re-fetch:
- All four bd commands re-run concurrently.
- A loading indicator replaces the left pane content while fetching.
- On completion, the graph is rebuilt and the current tab filter is reapplied.
- The cursor resets to position 0.

## States

### Loading

On entry and on refresh, the left pane shows:

```
  Loading...
```

"Loading..." is rendered in `ForegroundDim`, centered vertically and horizontally.

### Error

If `bd` is not installed, the `.beads` directory is missing, or the dolt server is not running:

```
  Error: bd not found
  Press r to retry
```

Error message in `StatusError`, retry hint in `ForegroundDim`.

### Empty

When the current tab has no issues (after filtering):

```
  No issues
```

"No issues" is rendered in `ForegroundDim`.

## v1 Scope Exclusions

The following are explicitly out of scope for the initial implementation:

- **No write operations** — tasks view is read-only. No claiming, updating, or closing issues from the TUI.
- **No auto-refresh** — data is fetched on entry and on manual `r` only. No polling timer.
- **No gate checking** — gates are displayed but not validated or toggled.
- **No inline editing** — no text input fields for issue fields.

## Theme Colors

No new theme colors are required. Tasks view reuses:

| Existing color    | Used for                                          |
|-------------------|---------------------------------------------------|
| `Foreground`      | Titles, active tab, priority P2, default type color |
| `ForegroundDim`   | Meta text, inactive tabs, timestamps, tree lines, status icons (open) |
| `StatusRunning`   | In-progress icon, priority P1                     |
| `StatusSuccess`   | Closed icon, feature type color                   |
| `StatusError`     | Blocked icon, bug type color, priority P0, error messages |
| `Highlight`       | Selected row background                           |

Epic, chore, and decision type colors use direct hex values from the active theme's palette (see Type Coloring table above).

## Mouse Support

### Issue List (Left Pane)

- **Scroll**: mouse wheel scrolls the list, 3 lines per tick.
- **Click**: single click selects an issue and updates detail. Single click on an expanded parent toggles collapse.
- **Focus**: any mouse interaction switches focus to the issue list.

### Issue Detail (Right Pane)

- **Scroll**: mouse wheel scrolls the detail (enters depth 1 if not already). 3 lines per tick, clamped to content bounds.
- **Focus**: scrolling the detail switches focus to the right pane.
