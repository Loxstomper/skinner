# Implementation Plan

## Completed

- **Git View Commit Stats Redesign** (tasks 1–9): 3-char hash, async total stats header, selected-row stats, all tests passing.
- **Bottom Layout Last-Focused Tracking**: h/← from main area now recalls last-focused bottom bar section instead of always defaulting to iterations.
- **Help Modal Gaps Resolved**: Scroll support for small terminals (pgdn/pgup scroll modal content), pgdn/pgup entries added to Navigation section, "Edit plan file" driven by ActionEditPlan keybinding, GitView entry added to Actions section.

## Deferred (per spec)

- **Rate Limit Window Display** (`specs/token-usage.md`): Header area reserved with `5h: --  wk: --` placeholder. Data source (Claude CLI `/usage` or API) to be determined. Per-tool-call token attribution is fully implemented.

## Status

All specs fully implemented. `make check` passes (vet, lint, tests). No TODOs/FIXMEs in codebase.
