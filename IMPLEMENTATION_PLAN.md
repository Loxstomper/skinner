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

## Tasks

6. **Add integration tests for cached plan view and file preview rendering**
   - Test `RenderPlanView` with cache: first call populates cache, second call uses it, verify identical output
   - Test `RenderPlanView` cache invalidation: modify temp file between calls, verify re-render
   - Test `RenderFilePreview` markdown path with cache: same hit/miss pattern
   - Test `RenderFilePreview` source path with cache: verify cached raw lines, chroma still applied

7. **Rename `benchmark_test.go` to `timeline_benchmark_test.go`**
   - Rename the file; no code changes needed
   - Update `specs/benchmarks.md` if any references to the old filename remain (already done in spec update)

8. **Create `planview_benchmark_test.go`**
   - Add `makePlanMarkdown(size string)` helper generating realistic markdown (headings, prose, code blocks) at ~1KB/10KB/100KB
   - Add `BenchmarkPlanViewUncached` — parameterized by small/medium/large, invalidates cache each iteration, measures full glamour render cost
   - Add `BenchmarkPlanViewCached` — parameterized by small/medium/large, primes cache once, measures cached render cost

9. **Create `filepreview_benchmark_test.go`**
   - Add `makeSourceFile(size string)` helper generating realistic Go source code at ~1KB/10KB/100KB
   - Add `BenchmarkFilePreviewMarkdownUncached` — markdown file preview without cache
   - Add `BenchmarkFilePreviewMarkdownCached` — markdown file preview with warm cache
   - Add `BenchmarkFilePreviewSourceUncached` — source code preview without cache
   - Add `BenchmarkFilePreviewSourceCached` — source code preview with warm cache

10. **Run full benchmark suite and verify improvements**
    - Run `go test -bench=. -benchmem ./internal/tui/` to confirm all benchmarks pass
    - Verify cached benchmarks show near-zero cost regardless of file size
    - Verify timeline benchmarks have no regressions from the rename
    - Run `make check` to confirm no lint/test failures
