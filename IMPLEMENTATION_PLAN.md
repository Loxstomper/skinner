# Implementation Plan

## Completed

- **Git View Commit Stats Redesign** (tasks 1–9): 3-char hash, async total stats header, selected-row stats, all tests passing.
- **Bottom Layout Last-Focused Tracking**: h/← from main area now recalls last-focused bottom bar section instead of always defaulting to iterations.

## Deferred (per spec)

- **Rate Limit Window Display** (`specs/token-usage.md`): Header area reserved with `5h: --  wk: --` placeholder. Data source (Claude CLI `/usage` or API) to be determined. Per-tool-call token attribution is fully implemented.

## Known Gaps

- **Help Modal Scrolling** (`specs/help-modal.md`): Modal content does not scroll vertically when terminal is too small. pgdn/pgup are not defined as configurable actions in KeyMap. "Edit plan file" entry is hardcoded rather than driven by the keybinding system.

## Status

All specs fully implemented except minor help modal gaps above. `make check` passes (vet, lint, tests). No TODOs/FIXMEs in codebase.
