# Implementation Plan

All tasks from the "Centred Header & Context Window Percentage" feature have been implemented.

## Completed

- **Context window percentage in header** — `ContextWindow` field added to `ModelPricing`, latest usage tracked per assistant event (`LastInputTokens`, `LastCacheReadTokens`), header centred with `ctx N%` display color-coded by threshold (0–69% dim, 70–89% warning, 90%+ critical).
- **Lint fixes** — golangci-lint v2 config migration (`version: "2"`, removed deprecated `gosimple`), fixed `errcheck`, `gocritic`, and `nilerr` warnings across config, tui, and parser packages.
- **Config tests** — `TestDefaultPricing`, `TestLoadConfig_ContextWindowFromTOML`, `TestLoadConfig_NoConfigFile` added.

## Notes

- No test files exist yet for `model`, `parser`, `theme`, or `tui` packages. Future work should add coverage.
- The `context_window` TOML parsing uses `strconv.Atoi` (integer) while pricing fields use `strconv.ParseFloat`.
