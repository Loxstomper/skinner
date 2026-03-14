# Render Cache — Implementation Plan

Spec: [specs/render-cache.md](specs/render-cache.md)

## Completed

1. ~~Create `RenderCache` struct in `internal/tui/rendercache.go`~~ — Done
2. ~~Add unit tests for `RenderCache` in `internal/tui/rendercache_test.go`~~ — Done (7 tests: empty miss, hit after set, path change, width change, modtime change, deleted file, nil safety)
3. ~~Add `*RenderCache` to `PlanViewProps` and integrate in `RenderPlanView`~~ — Done
   - Added `Cache *RenderCache` field to `PlanViewProps`
   - `RenderPlanView` calls `cache.Get` before file read; on hit skips `os.ReadFile` + `renderMarkdown`; on miss renders then calls `cache.Set` with modtime from `os.Stat`
   - Nil cache handled via `RenderCache.Get`/`Set` nil-receiver checks — existing tests pass unchanged
4. ~~Add `*RenderCache` to `FilePreviewProps` and integrate in `RenderFilePreview`~~ — Done
   - Added `Cache *RenderCache` field to `FilePreviewProps`
   - Cache check happens in `RenderFilePreview` before file read; on hit routes to `renderMarkdownPreviewFromLines` or `renderSourcePreviewFromLines`
   - Markdown: caches glamour-rendered lines (same as plan view)
   - Source: caches raw split lines; chroma tokenization still runs per-frame on visible slice (intentional — per-line cost is low, and caching styled output would require theme + hscroll in cache key)
   - Nil cache handled via existing nil-receiver checks — all 13 existing tests pass unchanged
5. ~~Wire `RenderCache` into root model~~ — Done
   - Plan view: passed through `PlanViewProps.Cache`
   - File preview: passed through `FilePreviewProps.Cache` in `fileExplorerView`
6. ~~Add integration tests for cached plan view and file preview rendering~~ — Done
7. ~~Rename `benchmark_test.go` to `timeline_benchmark_test.go`~~ — Done
8. ~~Create `planview_benchmark_test.go`~~ — Done
   - `makePlanMarkdown(size string)` helper generates realistic markdown (headings, prose, bullet lists, fenced code blocks) at ~1KB/10KB/100KB
   - `BenchmarkPlanViewUncached` — nil cache forces full glamour render each iteration; scales with file size (353μs→21.6ms for small→large)
   - `BenchmarkPlanViewCached` — primes cache once; near-constant ~70-96μs regardless of file size, confirming cache eliminates glamour bottleneck

9. ~~Create `filepreview_benchmark_test.go`~~ — Done
   - `makeSourceFile(size string)` helper generates realistic Go source (structs, methods, error handling) at ~1KB/10KB/100KB
   - `BenchmarkFilePreviewMarkdownUncached` — nil cache forces full glamour render; scales with file size (348μs→21ms for small→large)
   - `BenchmarkFilePreviewMarkdownCached` — primes cache once; near-constant ~74-106μs regardless of file size
   - `BenchmarkFilePreviewSourceUncached` — full file read + split + chroma highlight of visible lines (~3ms)
   - `BenchmarkFilePreviewSourceCached` — cache hit skips I/O; chroma still runs per-frame (~2.9ms) — confirms cache saves only I/O, not highlighting (by design)

## Tasks

10. **Run full benchmark suite and verify improvements**
    - Run `go test -bench=. -benchmem ./internal/tui/` to confirm all benchmarks pass
    - Verify cached benchmarks show near-zero cost regardless of file size
    - Verify timeline benchmarks have no regressions from the rename
    - Run `make check` to confirm no lint/test failures
