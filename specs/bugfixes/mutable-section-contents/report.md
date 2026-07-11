# Bugfix Report: Built Documents Expose Mutable Section Contents

**Date:** 2026-07-10
**Status:** Fixed
**Ticket:** T-1543

## Description of the Issue

`Document.GetContents` copies only the top-level slice, but the elements are the same `Content` pointers stored in the built document. A caller can type-assert a returned value to `*SectionContent` and call its exported `AddContent` method after `Builder.Build()`, mutating the supposedly finalized document. This breaks the v2 immutability promise in two ways:

1. Rendered output changes after build — content injected post-build appears in subsequent renders.
2. `SectionContent.AddContent` appends to `s.contents` with no synchronization while renderers recursively read `section.Contents()` during rendering, a data race detectable with `go test -race`.

The same exposure exists one level deeper: `SectionContent.Contents()` and `DefaultCollapsibleSection.Content()` copy their slices but share the nested pointers, so nested sections are equally mutable.

**Reproduction steps:**
1. Build a document containing a section: `doc := New().Section("Report", func(b *Builder) { b.Text("original") }).Build()`
2. Render it (e.g. markdown) and note the output.
3. `section := doc.GetContents()[0].(*SectionContent); section.AddContent(NewTextContent("injected"))`
4. Render again — the output now contains "injected".
5. Run the mutation concurrently with rendering under `-race` — data race reported.

**Impact:** Any consumer relying on built-document immutability (the core v2 design promise). Output can silently change between renders of the same document, and concurrent render + mutation is a data race with undefined behaviour.

## Investigation Summary

Followed the systematic debugging methodology (Fagan inspection phases).

- **Symptoms examined:** Post-build `AddContent` succeeds and alters rendered output; `go test -race` reports races between `AddContent` writes and renderer reads of `section.Contents()`.
- **Code inspected:** `v2/document.go` (`GetContents`, `Build`, `Builder.Section`), `v2/content.go` (`SectionContent`), `v2/collapsible_section.go`, renderer recursion sites (`html_renderer.go:280`, `markdown_renderer.go:307`, `json_yaml_renderer.go:287/798`, `csv_renderer.go:173`, `table_renderer.go:121`), and the transform pipeline (`renderer.go:179`, `base_renderer.go:127`).
- **Hypotheses tested:**
  - Internal code might append to sections post-build (would rule out freezing) — ruled out: the only production caller of `SectionContent.AddContent` is `Builder.Section` during building, and the transform pipeline clones content before applying operations and builds new `Document` values rather than mutating sections in place.
  - The bug might be limited to top-level sections — ruled out: nested sections and sections inside collapsible sections are reachable through the same shared-pointer pattern.

## Discovered Root Cause

The immutability boundary is enforced on the **Builder**, not on the **content graph the Document exposes**. `Build()` clears the builder's document reference (and T-1689 added post-build guards to builder methods), but the content objects inside the document remain mutable through their own exported APIs. `SectionContent.AddContent` has no notion of a finalized state and appends unconditionally, without synchronization.

**Defect type:** Missing state guard / data race (state management).

**Why it occurred:** `AddContent` was designed for the building phase (`Builder.Section` uses it to populate sections). Immutability work focused on the builder surface, assuming content objects were private after build — but `GetContents`/`Contents()`/`Content()` hand out the real pointers and `AddContent` is exported.

**Contributing factors:** Slice-copy-only accessors give a false sense of protection; there is no freeze/finalize concept on content types.

## Resolution for the Issue

Sections are now frozen when the document is built.

**Changes made:**
- `v2/content.go` — Added a `frozen atomic.Bool` field to `SectionContent`. `AddContent` is a no-op once frozen (documented; consistent with the existing silent nil-drop, since `SectionContent` has no error channel). An exported `Frozen()` predicate lets callers guard against the no-op up front. `Clone` is documented as the escape hatch: clones are never frozen, so callers can still obtain a mutable copy of a built section.
- `v2/document.go` — `Builder.Build()` now calls the new `freezeSectionContents` helper on every top-level content value. The helper recursively freezes every `*SectionContent` reachable through nested sections and through `*DefaultCollapsibleSection` contents. `Build` and `GetContents` godoc updated to state the guarantee. Freezing happens only in the user-facing `Build()`: `Section()` and `CollapsibleSection()` harvest their sub-builders through an internal non-freezing `build()` method, so a caller-held section added inside a callback stays mutable until the outer document is actually built (found in PR review; the original fix routed the harvest through the public `Build()`, freezing such sections prematurely).

The atomic flag makes the fix race-free: `Build()` completes (and thus the `Store(true)`) before the document is shared, so any post-build `AddContent` reliably observes `frozen == true` and returns without touching the slice — no write ever races with renderer reads.

**Approach rationale:** Freezing at `Build()` enforces immutability on the object graph itself, matching the existing builder-surface guards from T-1689, at near-zero cost (one atomic load per `AddContent`, one tree walk per `Build`). It was verified safe: the only production caller of `SectionContent.AddContent` is `Builder.Section` (runs before freeze), and the render/transform pipeline clones content before applying operations and constructs new `Document` values rather than mutating sections in place.

**Alternatives considered:**
- **Deep-clone in `GetContents`** — correct but expensive: `baseRenderer` calls `GetContents` several times per render, so every render of every format would deep-copy the whole document (including large tables); it also changes pointer-identity semantics for existing callers.
- **Mutex on `SectionContent`** — fixes the data race but not the immutability violation: rendered output could still change after build, which is the primary bug.
- **Recording an error or panicking on post-build `AddContent`** — `SectionContent` has no error channel, and the v2 fluent API is deliberately non-panicking; a documented no-op matches the established convention.

## Regression Test

**Test file:** `v2/section_content_immutability_test.go`

**Tests:**
- `TestSectionContent_PostBuildMutation_DoesNotChangeRenderedOutput` — rendering a built document yields identical output before and after an attempted post-build `AddContent`; the injected content never appears.
- `TestSectionContent_PostBuildMutation_NestedSectionAlsoFrozen` — nested sections obtained via `Contents()` cannot be mutated post-build.
- `TestSectionContent_PostBuildMutation_SectionInCollapsibleFrozen` — sections reachable through `DefaultCollapsibleSection.Content()` cannot be mutated post-build.
- `TestSectionContent_PreBuildAddContentStillWorks` — the legitimate building phase is unaffected.
- `TestSectionContent_SectionCallbackDoesNotFreezeCallerHeldSection` — a caller-held section added via `AddContent` inside a `Section()` callback stays mutable until the outer `Build()`, then freezes (regression test for the premature-freeze issue found in PR review).
- `TestSectionContent_CollapsibleSectionCallbackDoesNotFreezeCallerHeldSection` — same pattern through `CollapsibleSection()`.
- `TestSectionContent_FrozenPredicate` — `Frozen()` is false while building, true after `Build()`, and false on a `Clone()` of a frozen section.
- `TestSectionContent_CloneOfBuiltSectionIsMutable` — `Clone()` remains the escape hatch for a caller-owned mutable copy.
- `TestSectionContent_ConcurrentRenderAndPostBuildMutation` — concurrent render + attempted mutation is race-free (run with `-race`).

**Run command:** `cd v2 && go test -race -run 'TestSectionContent_' .`

**Pre-fix results (red):** the three PostBuildMutation tests fail (output changes, contents grow), the two callback tests fail (sections freeze before the outer `Build()`), and the concurrent test reports data races under `-race`.

## Affected Files

| File | Change |
|------|--------|
| `v2/content.go` | `frozen` flag on `SectionContent`; `AddContent` no-ops when frozen; exported `Frozen()` predicate; `Clone`/`AddContent` godoc |
| `v2/document.go` | `Build()` freezes reachable sections via new `freezeSectionContents`; internal non-freezing `build()` harvest for `Section`/`CollapsibleSection`; `Build`/`GetContents` godoc |
| `v2/section_content_immutability_test.go` | New regression and race coverage (9 tests) |

## Verification

**Automated:**
- [x] Regression test passes — all 6 tests in `section_content_immutability_test.go` pass, including `TestSectionContent_ConcurrentRenderAndPostBuildMutation` under `-race`
- [x] Full test suite passes — `make test` and `make test-integration` green; full `go test -race` on `v2` and `v2/icons` green
- [x] Linters/validators pass — `golangci-lint run`: 0 issues; `gofmt -l`: clean

## Prevention

**Recommendations to avoid similar bugs:**
- When a type promises immutability after a lifecycle point, enforce it on the object graph (freeze/clone), not just on the factory/builder that produced it.
- Treat every exported mutator on content types as a post-build mutation vector and guard or document it explicitly.
- Prefer race tests (`-race`) for any accessor/mutator pair on shared content.

## Related

- Transit ticket T-1543 (this bug)
- T-1689 — post-build guards on Builder methods (same immutability theme, builder surface only)
- T-1677 — `TableContent.Transform` mutates built documents (sibling bug, fixed separately)
- T-1317 — defensive copies in `DefaultCollapsibleSection` (slice-copy-only pattern)
