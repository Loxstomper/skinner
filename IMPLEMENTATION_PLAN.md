# Implementation Plan

## Completed

- **Git View Commit Stats Redesign** (tasks 1–9): 3-char hash, async total stats header, selected-row stats, all tests passing.
- **Bottom Layout Last-Focused Tracking**: h/← from main area now recalls last-focused bottom bar section instead of always defaulting to iterations.
- **Help Modal Gaps Resolved**: Scroll support for small terminals (pgdn/pgup scroll modal content), pgdn/pgup entries added to Navigation section, "Edit plan file" driven by ActionEditPlan keybinding, GitView entry added to Actions section.

## Deferred (per spec)

- **Rate Limit Window Display** (`specs/token-usage.md`): Header area reserved with `5h: --  wk: --` placeholder. Data source (Claude CLI `/usage` or API) to be determined. Per-tool-call token attribution is fully implemented.

## File Explorer (`specs/file-explorer.md`)

### Phase 1: Foundation

1. ~~**Add `sahilm/fuzzy` dependency**~~ — Deferred to Phase 6 (task 13) when fuzzy search is implemented.

2. ~~**Add `ActionFileExplorer` keybinding**~~ ✅ `ActionFileExplorer = "file_explorer"` in `internal/config/keymap.go`, default binding `f`, added to `AllActions()`, help modal "Actions" section, and `actionDisplayName()`. Tests pass.

3. ~~**File tree data model**~~ ✅ Implemented in `internal/tui/filetree.go`: `FileNode`, `BuildFileTree`, `FlattenVisible`, `IsBinary`, `FindParent`. 17 tests covering sort order, .git skip, symlinks, expand/collapse, binary detection.

4. ~~**Git status integration**~~ ✅ Implemented in `internal/tui/filetree.go`: `ApplyGitStatus`, `parsePorcelain`, directory status inheritance (D>M>A>?). Tests: porcelain parsing, file status application, directory inheritance priority.

### Phase 2: Left Pane — File Tree Rendering

5. ~~**File tree component**~~ ✅ Implemented in `internal/tui/filetreeview.go`: `FileTreeView` struct (Cursor, Scroll, roots, cached rows), `FileTreeViewProps` (Width, Height, Focused, Theme), `HandleAction()` for navigation (move_down/up, jump_top/bottom, page_down/up), `View()` renders tree with indent (2×depth), `▶`/`▼` icons, `🔗` symlinks, right-aligned git status. Colors per spec. Scroll-to-cursor, ScrollBy, ClickRow. 30 tests.

6. ~~**Tree-specific `h`/`l` navigation**~~ ✅ Implemented in `FileTreeView.HandleAction()` via focus_left/focus_right/expand actions: h on file → collapse parent + cursor to parent, h on expanded dir → collapse, h on collapsed dir → collapse parent, l on collapsed dir → expand, l on expanded dir → cursor to first child, l on file → signal depth 2, enter on dir → toggle, enter on file → signal depth 2. 11 dedicated tests.

### Phase 3: Right Pane — File Preview

7. ~~**File preview renderer**~~ ✅ Implemented in `internal/tui/filepreview.go`: `RenderFilePreview()` pure render function with `FilePreviewProps` and `FilePreviewResult`. Markdown via glamour (reuses `renderMarkdown`), source code via chroma (reuses `getLexer`/`getChromaStyle`/`tokenizeLine`), binary detection, file-not-found handling, line number gutter in `ForegroundDim`, horizontal scroll, title bar centered+bold. `ClampFilePreviewScroll` for scroll bounds. 16 tests.

### Phase 4: Root Model Integration

8. **File explorer state in root model** — In `internal/tui/root.go`:
   - Add `fileExplorerActive bool`, `fileExplorerDepth int` (0=tree, 1=scrollable), `fileExplorerTree *FileTreeView`
   - Add `filePreviewScroll`, `filePreviewHScroll` int for depth 1 scroll state
   - `enterFileExplorer()` — save current view state, build tree with first level expanded, start refresh timer
   - `exitFileExplorer()` — restore view state, cancel refresh timer
   - Tests: enter/exit preserves run view state, depth transitions.

9. **Key routing for file explorer** — In `handleKey()`:
   - When `fileExplorerActive`, route keys to file explorer handlers instead of normal handlers
   - Depth 0: tree navigation, `enter`/`h`/`l` tree actions, `e` for editor, `/` for search, `escape` exits explorer
   - Depth 1: `j`/`k` vertical scroll, `h`/`l` horizontal scroll, `gg`/`G`/`pgup`/`pgdn`, `#` toggle line numbers, `e` for editor, `escape` returns to depth 0
   - `q` and `ctrl+c` still show quit confirmation; `?` still shows help
   - Tests: verify key routing at each depth, modal keys still work.

10. **File explorer View() rendering** — In root `View()`:
    - When `fileExplorerActive`, render file tree in left pane and file preview in right pane (bypass normal iteration/timeline rendering)
    - Reuse existing two-pane layout calculation (side/bottom/auto)
    - `[` toggle still works for hiding left pane
    - Tests: verify two-pane output, layout switching at 80-col threshold.

### Phase 5: Refresh Timer & Editor

11. **5-second refresh timer** — Following git view's `gitTickCmd` pattern:
    - `fileExplorerTickMsg` / `fileExplorerRefreshMsg` message types
    - `fileExplorerTickCmd()` fires every 5 seconds
    - `fileExplorerRefreshCmd()` re-walks filesystem + runs `git status --porcelain` in background
    - On refresh: merge new tree state preserving expand/collapse and cursor position
    - Defer refresh during active search
    - Tests: verify tree updates on refresh, cursor preserved, expand state preserved.

12. **Editor integration** — `e` on a file:
    - Suspend TUI via `tea.ExecProcess` with `$EDITOR` (fallback `vi`), same pattern as plan file editing
    - On resume: re-read file for preview, reset scroll if file modified
    - Tests: verify editor command is constructed correctly.

### Phase 6: Fuzzy Search

13. **Fuzzy search mode** — In `FileTreeView`:
    - `/` activates search: show input bar at bottom of left pane (`/ query█`)
    - Collect all file paths (flat list from tree), feed to `sahilm/fuzzy` on each keystroke
    - Replace tree view with ranked flat result list during search
    - `j`/`k` navigate results, preview live-updates to selected match
    - `enter` confirms: dismiss search, expand parents of selected file, cursor on file
    - `escape` cancels: restore pre-search tree state
    - Tests: fuzzy matching ranks correct results, enter expands path to file, escape restores state.

### Phase 7: Mouse Support

14. **Mouse events for file explorer** — In root model mouse handler:
    - Left pane clicks: select file/toggle dir in tree, switch focus to tree
    - Left pane scroll: 3 lines per tick in tree
    - Right pane scroll: scroll preview (enter depth 1 if not already), switch focus to right pane
    - Tests: verify click selects correct row, scroll direction, focus switching.

### Phase 8: Spec & Help Updates

15. **Update help modal** — Add file explorer keybindings to help modal sections:
    - Global section: `f` — Enter file explorer
    - File explorer section (new): tree navigation, search, editor, depth keys
    - Update `specs/help-modal.md` if needed.

16. **Final verification** — `make check` passes (vet, lint, tests). Review spec compliance against `specs/file-explorer.md`. No TODOs/FIXMEs.

## Status

All prior specs fully implemented. `make check` passes (vet, lint, tests). No TODOs/FIXMEs in codebase. File explorer Phases 1-3 (tasks 1-7) complete; Phase 4 (root model integration) pending.
