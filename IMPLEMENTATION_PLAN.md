# Implementation Plan

## Completed

- **Git View Commit Stats Redesign** (tasks 1–9): 3-char hash, async total stats header, selected-row stats, all tests passing.
- **Bottom Layout Last-Focused Tracking**: h/← from main area now recalls last-focused bottom bar section instead of always defaulting to iterations.
- **Help Modal Gaps Resolved**: Scroll support for small terminals (pgdn/pgup scroll modal content), pgdn/pgup entries added to Navigation section, "Edit plan file" driven by ActionEditPlan keybinding, GitView entry added to Actions section.

## Deferred (per spec)

- **Rate Limit Window Display** (`specs/token-usage.md`): Header area reserved with `5h: --  wk: --` placeholder. Data source (Claude CLI `/usage` or API) to be determined. Per-tool-call token attribution is fully implemented.

## File Explorer (`specs/file-explorer.md`)

### Phase 1–3: Foundation, Tree Rendering, File Preview ✅

Tasks 1–7 complete. `FileNode`, `BuildFileTree`, `ApplyGitStatus`, `FileTreeView`, `RenderFilePreview` all implemented with full test coverage.

### Phase 4: Root Model Integration ✅

8. ~~**File explorer state in root model**~~ ✅ Implemented in `internal/tui/fileexplorer.go`:
   - State fields: `fileExplorerActive`, `fileExplorerDepth` (0=tree, 1=scrollable), `fileExplorerTree`, `filePreviewScroll`, `filePreviewHScroll`
   - `enterFileExplorer()` builds tree from CWD with git status, starts 5s refresh timer
   - `exitFileExplorer()` clears state, preserves original run view state
   - 22 tests covering enter/exit, depth transitions, navigation, scroll, merge/refresh

9. ~~**Key routing for file explorer**~~ ✅ Implemented in `handleFileExplorerKey()`:
   - When `fileExplorerActive`, all keys routed to file explorer before git view or normal handlers
   - Depth 0: tree navigation (j/k/gg/G/pgdn/pgup), enter/h/l tree actions, e for editor, [ for left pane toggle, # for line numbers
   - Depth 1: j/k vertical scroll, h/l horizontal scroll, gg/G/pgdn/pgup, # toggle line numbers, e for editor, escape returns to depth 0
   - q and ctrl+c show quit confirmation; ? shows help
   - f key toggles file explorer on/off

10. ~~**File explorer View() rendering**~~ ✅ Implemented in `renderFileExplorer()`:
    - Two-pane layout: file tree left, file preview right
    - Reuses existing layout width/height calculations
    - [ toggle hides/shows left pane
    - Preview scroll clamped to actual content bounds

11. ~~**5-second refresh timer**~~ ✅ Implemented:
    - `fileExplorerTickMsg`/`fileExplorerRefreshMsg` message types
    - `fileExplorerTickCmd()` fires every 5 seconds
    - `fileExplorerRefreshCmd()` re-walks filesystem + runs `git status --porcelain`
    - `mergeFileExplorerTree()` preserves expand/collapse and cursor position

12. ~~**Editor integration**~~ ✅ Implemented in `launchFileExplorerEditor()`:
    - `e` on file → `tea.ExecProcess` with `$EDITOR` (fallback `vi`)
    - `fileExplorerEditorDoneMsg` handled in Update()

14. ~~**Mouse events for file explorer**~~ ✅ Implemented in `handleFileExplorerMouse()`:
    - Left pane clicks: select file/toggle dir, switch to depth 0
    - Left pane scroll: 3 lines per tick
    - Right pane scroll: scroll preview, enter depth 1 if not already

13. ~~**Fuzzy search mode**~~ ✅ Implemented in `FileTreeView`:
    - `/` activates search at depth 0: input bar at bottom of left pane (`/ query█`)
    - `sahilm/fuzzy` dependency added, matches against all file paths (flat list)
    - Tree view replaced by ranked flat result list during search
    - `j`/`k` (↓/↑) navigate results, `SearchSelectedNode()` updates preview
    - `enter` confirms: expands parent dirs, cursor on selected file
    - `escape` cancels: restores pre-search tree state (cursor, scroll, expand state)
    - Refresh deferred during search to avoid disrupting results
    - 18 new tests: matching, navigation, cancel/confirm, view rendering, integration

### Remaining Tasks

15. **Update help modal** — Add file explorer keybindings to help modal sections:
    - Global section: `f` — Enter file explorer
    - File explorer section (new): tree navigation, search, editor, depth keys
    - Update `specs/help-modal.md` if needed.

16. **Final verification** — `make check` passes (vet, lint, tests). Review spec compliance against `specs/file-explorer.md`. No TODOs/FIXMEs.

## Status

All prior specs fully implemented. `make check` passes (vet, lint, tests). File explorer Phase 4 (tasks 8-14) complete. Remaining: help modal updates (task 15), final verification (task 16).
