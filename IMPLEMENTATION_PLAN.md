# Render Cache — Implementation Plan

Spec: [specs/render-cache.md](specs/render-cache.md)

## Completed

1. ~~Create `RenderCache` struct in `internal/tui/rendercache.go`~~ — Done
2. ~~Add unit tests for `RenderCache` in `internal/tui/rendercache_test.go`~~ — Done (7 tests: empty miss, hit after set, path change, width change, modtime change, deleted file, nil safety)

## Tasks

3. **Add `*RenderCache` to `PlanViewProps` and integrate in `RenderPlanView`**
   - Add `Cache *RenderCache` field to `PlanViewProps`
   - In `RenderPlanView`: call `cache.Get(path, width)` before file read; on hit, skip `os.ReadFile` and `renderMarkdown`, use cached lines; on miss, render as before then call `cache.Set`
   - Handle nil cache (no-op, render without caching) for backward compatibility with existing tests

4. **Add `*RenderCache` to `FilePreviewProps` and integrate in `RenderFilePreview`**
   - Add `Cache *RenderCache` field to `FilePreviewProps`
   - In `renderMarkdownPreview`: call `cache.Get` before rendering; on hit use cached lines; on miss render then `cache.Set`
   - In `renderSourcePreview`: call `cache.Get` to cache raw source lines (pre-split, pre-chroma); on hit skip `os.ReadFile` + `strings.Split`; chroma tokenization of visible lines still runs per-frame
   - Handle nil cache for backward compatibility

5. **Wire `RenderCache` into root model**
   - Add a `renderCache *RenderCache` field to the root model, initialized in the constructor
   - Pass it through `PlanViewProps.Cache` when calling `RenderPlanView`
   - Pass it through `FilePreviewProps.Cache` when calling `RenderFilePreview`

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
