# Render Cache

## Overview

A shared `RenderCache` provides single-slot caching for expensive file rendering operations. It eliminates per-frame re-rendering of markdown (glamour) and source code (chroma) content that hasn't changed, fixing UI lockups when viewing large files.

## Problem

Both `RenderPlanView` and `RenderFilePreview` re-render file content on every `View()` call. Since Bubble Tea calls `View()` on every event (keypress, mouse, tick), this means:

- **Glamour markdown rendering**: full goldmark parse + chroma code block highlighting on every frame. For large plan files with code fences, this can take hundreds of milliseconds — blocking the event loop and making the TUI unresponsive.
- **File I/O**: `os.ReadFile()` on every frame, even when the file hasn't changed.
- **Source code**: `os.ReadFile()` + `strings.Split()` on every frame. Chroma tokenization is already O(visible) since the preview slices to visible lines before highlighting, but the file read and split are wasteful.

The timeline pane solved a similar problem with two-phase viewport-only rendering (see [viewport-rendering.md](viewport-rendering.md)). The render cache solves it for content views where partial rendering isn't possible (glamour requires the full document).

## Design

### `RenderCache` Struct

A single-slot cache that maps a file path, modification time, and render width to pre-rendered lines:

```go
type RenderCache struct {
    path    string
    modTime time.Time
    width   int
    lines   []string
}
```

The cache stores one entry at a time. Since only one file preview or plan view is visible at a time, a single slot is sufficient.

### Cache Key

The cache is valid when all three fields match:

| Field | Source | Invalidated by |
|-------|--------|----------------|
| `path` | File being rendered | Selecting a different file |
| `modTime` | `os.Stat()` result | File modified on disk (editor save, external change) |
| `width` | Terminal/pane width | Terminal resize |

### `Get` Method

```go
func (c *RenderCache) Get(path string, width int) ([]string, bool)
```

1. Compare `path` and `width` against cached values. If either differs → miss.
2. Call `os.Stat(path)` to get current modification time. If stat fails → miss.
3. Compare mod time against cached value. If different → miss.
4. Return cached lines → hit.

On a cache hit, the only syscall is a single `os.Stat()`. No file read, no rendering.

### `Set` Method

```go
func (c *RenderCache) Set(path string, modTime time.Time, width int, lines []string)
```

Stores the rendered lines along with the cache key fields. Called by the rendering functions after a cache miss produces new content.

### File Location

`internal/tui/rendercache.go` with unit tests in `internal/tui/rendercache_test.go`.

## Integration

### Plan View (`planview.go`)

`RenderPlanView` receives a `*RenderCache` via its props. On each call:

1. Call `cache.Get(path, width)`.
2. On hit: use cached lines directly, apply scroll slicing.
3. On miss: read file, call `renderMarkdown()`, call `cache.Set()`, then apply scroll slicing.

Scrolling never invalidates the cache — it only changes which slice of the cached lines is displayed.

### File Preview (`filepreview.go`)

`RenderFilePreview` receives the same `*RenderCache` via its props. On each call:

1. Call `cache.Get(path, width)`.
2. On hit: use cached lines, apply scroll/hscroll and line numbers.
3. On miss:
   - **Markdown files**: read file, call `renderMarkdown()`, store rendered lines.
   - **Source code files**: read file, split into lines, store raw lines. Chroma tokenization still runs per-frame on the visible slice (it's already O(visible) and the cost is low).

For source code, the cache avoids repeated `os.ReadFile()` + `strings.Split()` but not the chroma highlighting of visible lines. This is intentional — chroma per-line cost is small, and caching styled output would require the cache key to include theme and horizontal scroll position.

### Cache Ownership

The root model owns a single `*RenderCache` instance and passes it through props to both `RenderPlanView` and `RenderFilePreview`. Since only one of these views is active at a time, they naturally share the single cache slot — switching views simply causes a cache miss on the first frame.

## Performance Characteristics

### Cache Hit (Steady State)

| Operation | Cost |
|-----------|------|
| Field comparison (path, width) | Negligible |
| `os.Stat()` | ~1μs (cached by OS) |
| Mod time comparison | Negligible |
| Scroll slicing | Negligible |
| **Total** | **~1μs** |

### Cache Miss (File Changed / First Render)

Same as current behavior — full file read + glamour/split. This is acceptable because it only happens when the file actually changes, not on every frame.

## What This Does Not Solve

The first render of a very large markdown file will still block the event loop for the duration of the glamour render. Fixing that would require async rendering in a goroutine with a loading placeholder — a separate enhancement beyond the scope of this spec.
