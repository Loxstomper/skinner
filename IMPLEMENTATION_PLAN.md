# Implementation Plan

## 1. Fix edit line spec example

The spec example `(+3/-1)` is impossible with the described algorithm. The algorithm computes `added = new_lines - old_lines` and `removed = old_lines - new_lines` — these are mutually exclusive, so you can never have both positive at the same time.

### Tasks

- [ ] Update `specs/stream-json-format.md` line 146: change the example from `(+3/-1)` to `(+3)` and adjust the surrounding text to clarify that additions and removals are mutually exclusive except in the net-zero case
- [ ] Update `specs/tui-layout.md` lines 95 and 107: change `(+3/-1)` to `(+2/-2)` (a valid net-zero example) in both Full view and Compact view examples

## 2. Document `--exit` flag

The `--exit` flag exists in `cmd/skinner/main.go` but is not documented.

### Tasks

- [ ] Add `--exit` to the CLI arguments table in `specs/iteration-loop.md`
- [ ] Document its behavior: when set, the TUI quits automatically after all iterations complete (or the last iteration fails), rather than remaining open for browsing

