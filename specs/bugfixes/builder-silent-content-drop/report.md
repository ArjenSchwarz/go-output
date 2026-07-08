# Bugfix Report: Builder Silently Drops Failed and Post-Build Content

**Date:** 2026-07-08
**Status:** Fixed
**Ticket:** T-1689

## Description of the Issue

The v2 fluent `Builder` can lose content with no error signal in three related ways:

1. **Post-Build mutations vanish silently.** `Build()` sets `b.doc = nil` and every mutator guards `if b.doc != nil`, so content or metadata added after `Build()` disappears with `HasErrors() == false`. This is internally inconsistent: the `Section`/`CollapsibleSection` paths *do* record errors for the analogous finalized-sub-builder misuse.
2. **Constructor failures are recorded but never surfaced by the promoted style.** `Builder.Table`/`Raw` swallow constructor errors into `b.errors`; `Build()` always returns `*Document` and rendering succeeds with the failed content silently missing. The fluent chain `output.New().Table(...).Build()` gives no point to call `HasErrors()`, and nothing documents that the caller must break the chain to check.
3. **Discarded constructor errors in renderers.** `NewGraphContentFromTable` returns `(*GraphContent, error)`, but both `extractGraphFromTable` implementations discard the error with `graph, _ :=` (graph_renderers.go:171 and :482).

**Reproduction steps:**
1. `builder := output.New()`; `doc := builder.Build()`
2. `builder.Text("hello")` — or any other content/metadata mutation
3. `builder.HasErrors()` returns `false` and `doc` (and any later render) contains no trace of the content

**Impact:** High severity (multi-agent code audit, 2026-07-07; claims verified by execution). Any consumer that accidentally reuses a builder after `Build()`, or whose table/raw data fails construction, produces silently incomplete output — the worst failure mode for an output library.

## Investigation Summary

Systematic (Fagan-style) inspection of `v2/document.go`, `v2/graph_content.go`, and `v2/graph_renderers.go`, confirming all three audit claims in code.

- **Symptoms examined:** post-Build mutations no-op with no recorded error; fluent chain offers no error observation point; `graph, _ :=` discards at graph_renderers.go:171/:482.
- **Code inspected:** `document.go:55-63` (`Build`), `:94-123` (`SetMetadata`/`AddContent` guards), `:126-158` (`Table`/`Raw` error accumulation), `:161-291` (Section paths that *do* record misuse errors); `graph_content.go:58-98` (`NewGraphContentFromTable`); `graph_renderers.go:137-176`, `:448-487` (both `extractGraphFromTable` implementations) and their callers at `:83` and `:275`.
- **Hypotheses tested:** whether the discarded graph error is reachable — it is not today (it only fires for a nil table, and both call sites dereference the table before calling), so it is fragile rather than currently harmful. Checked `TestBuilder_NilSafety` (document_test.go:346): it asserts post-Build mutations don't panic and don't modify the document, but does *not* assert errors stay absent, so recording errors is compatible with existing tests.

## Discovered Root Cause

**Defect type:** Silent failure / missing error propagation.

**Why it occurred:** `Build()` enforces document immutability by clearing `b.doc`; the `if b.doc != nil` guards were written purely to prevent nil-pointer panics. Error accumulation (`b.errors`) was introduced later for constructor failures and, later still, for finalized-sub-builder misuse in the Section paths — but the post-Build no-op branches were never wired into it. Because the fluent API has no per-call error channel and `Build()` returns only `*Document`, silent degradation is the structural default failure mode.

**Contributing factors:** the error-observation contract (`HasErrors()`/`Errors()` after building) is undocumented on `Builder`/`Build`; constructor signatures are split between `(*T, error)` and bare `*T`, inviting `_` discards.

## Resolution for the Issue

**Changes made:**
- `v2/document.go` (`SetMetadata`, `AddContent`) — the `if b.doc != nil` silent no-op guards now record a builder error when called after `Build()`: `"SetMetadata %q: cannot set metadata after Build() has been called"` / `"AddContent: cannot add content after Build() has been called"`. All fluent content helpers (`Table`, `Text`, `Section`, …) route through `AddContent`, so every post-Build mutation is now detectable via `HasErrors()`/`Errors()`. The built document remains unchanged, preserving immutability.
- `v2/document.go` (godoc on `Builder`, `Build`, `Table`, `Raw`) — documents the fluent-chain error contract: fluent methods never return errors mid-chain; failures are recorded on the builder and the offending content is omitted from the rendered output; `Build` is one-shot; check `HasErrors()`/`Errors()` after building.
- `v2/graph_renderers.go` (both `extractGraphFromTable` implementations and their `Render` callers) — the methods now return `(*GraphContent, error)` and the DOT/Mermaid `Render` methods propagate the error instead of discarding it with `graph, _ :=`. A `(nil, nil)` return still means "not a graph table, render normally".

**Approach rationale:** mirrors the error-accumulation pattern the builder already uses for constructor failures and finalized-sub-builder misuse (Section/CollapsibleSection), removing the internal inconsistency without any breaking API change. The graph fix propagates rather than re-hides the error so any future error condition in `NewGraphContentFromTable` surfaces through `Render`.

**Alternatives considered:**
- Panic on post-Build mutation — rejected: the fluent API is deliberately non-panicking, and existing (buggy but tolerated) callers would start crashing.
- Change `Build()` to return `(*Document, error)` now — rejected: breaking change; queued for v3 on T-1697 (Build is chain-terminal, so the fluent style survives that change).
- Keep `extractGraphFromTable`'s single-return signature and just drop the error explicitly — rejected: the error would remain unobservable; propagation costs only two small caller changes.

## Regression Test

**Test file:** `v2/document_test.go`
**Test name:** `TestBuilder_PostBuildMutationsRecordErrors`

**What it verifies:** `AddContent`, `SetMetadata`, `Table`, and `Text` called after `Build()` each record exactly one builder error (detectable via `HasErrors()`/`Errors()`) while leaving the already-built document unchanged.

**Run command:** `cd v2 && go test -run TestBuilder_PostBuildMutationsRecordErrors -v`

The discarded graph error has no failing test because the error path is unreachable today (nil-table check precedes the only error condition); the fix propagates the error so future error conditions surface, covered by compile-time checking and the existing renderer tests.

## Affected Files

| File | Change |
|------|--------|
| `v2/document.go` | `SetMetadata`/`AddContent` record errors post-Build; godoc for the error contract on `Builder`, `Build`, `Table`, `Raw` |
| `v2/graph_renderers.go` | Both `extractGraphFromTable` methods return `(*GraphContent, error)`; DOT/Mermaid `Render` propagate the error |
| `v2/document_test.go` | Regression test `TestBuilder_PostBuildMutationsRecordErrors` |

## Verification

**Automated:**
- [x] Regression test passes (`TestBuilder_PostBuildMutationsRecordErrors`, 4 subtests)
- [x] Full test suite passes (`make test` and `make test-integration`)
- [x] Linters/validators pass (`golangci-lint run`: 0 issues; `gofmt` clean)

**Manual verification:**
- Confirmed `TestBuilder_NilSafety` still passes: post-Build mutations remain non-panicking and the built document remains unchanged; only the error recording is new.

## Prevention

**Recommendations to avoid similar bugs:**
- Route all builder misuse through the existing `b.errors` accumulation channel instead of silent no-op guards.
- Never discard `error` returns with `_`; propagate or handle explicitly with a comment stating why.
- v3 (queued on T-1697): make `Build()` return `(*Document, error)` — Build is chain-terminal so the fluent style survives — and unify content constructors on `(*T, error)` routed through builder error accumulation.

## Related

- T-1689 (this bug), found in multi-agent code audit 2026-07-07
- T-1697 — Compile the v3 breaking-changes inventory (v3 portion of the fix queued there)
- T-1516 — no-op Output options; T-1527 — error taxonomy
- T-1353 — nil nested content in SectionContent (introduced the Section-path misuse errors this fix mirrors)
