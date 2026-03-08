# Implementation Plan

No outstanding tasks.

## Completed

### Fix tool call row highlight (per-segment background)
Resolved in commit. The highlight background is now baked into each styled
segment during rendering rather than applied as a post-hoc wrapper. This
prevents inner ANSI resets from interrupting the highlight background.

## Known Issues

### Integration test timeouts
`TestIntegration_HelpModal_EnterDismisses` and `TestIntegration_ExitFlag_SingleIteration`
occasionally hang on `bubbletea.Tick` channel receives. They do eventually pass
within the 90s Go test timeout but slow down the test suite. The `make check`
target completes successfully despite the delay.
