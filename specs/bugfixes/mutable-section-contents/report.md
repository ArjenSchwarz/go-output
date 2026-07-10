# Bugfix Report: Built Documents Expose Mutable Section Contents

**Date:** 2026-07-10
**Status:** In progress
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

_To be filled in after the fix is implemented._

## Regression Test

**Test file:** `v2/section_content_immutability_test.go`

**Tests:**
- `TestSectionContent_PostBuildMutation_DoesNotChangeRenderedOutput` — rendering a built document yields identical output before and after an attempted post-build `AddContent`; the injected content never appears.
- `TestSectionContent_PostBuildMutation_NestedSectionAlsoFrozen` — nested sections obtained via `Contents()` cannot be mutated post-build.
- `TestSectionContent_PostBuildMutation_SectionInCollapsibleFrozen` — sections reachable through `DefaultCollapsibleSection.Content()` cannot be mutated post-build.
- `TestSectionContent_PreBuildAddContentStillWorks` — the legitimate building phase is unaffected.
- `TestSectionContent_CloneOfBuiltSectionIsMutable` — `Clone()` remains the escape hatch for a caller-owned mutable copy.
- `TestSectionContent_ConcurrentRenderAndPostBuildMutation` — concurrent render + attempted mutation is race-free (run with `-race`).

**Run command:** `cd v2 && go test -race -run 'TestSectionContent_PostBuildMutation|TestSectionContent_PreBuild|TestSectionContent_CloneOfBuilt|TestSectionContent_ConcurrentRenderAndPostBuildMutation' .`

**Pre-fix results (red):** the three PostBuildMutation tests fail (output changes, contents grow), and the concurrent test reports data races under `-race`.

## Affected Files

_To be finalized with the fix._

## Verification

**Automated:**
- [ ] Regression test passes
- [ ] Full test suite passes
- [ ] Linters/validators pass

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
