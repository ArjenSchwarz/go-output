# Implementation Explanation: T-1677 — TableContent.Transform Mutates Built Documents

Branch: `T-1677/bugfix-table-transform-mutates-documents` (PR #101), over merge-base `2d4afe5`.

## Beginner Level

### What Changed

The go-output library lets you build a "document" out of tables and text, then render it as JSON, HTML, Markdown, and so on. The library promises that once you call `Build()`, the document is finished — nothing can change it afterwards.

There was a hole in that promise. Every table has a `Transform` method that rewrites its rows in place. If you fetched a table back out of a finished document and called `Transform` on it, the document's contents changed — later renders would show the new data, breaking the "finished means finished" rule.

The fix: when a table is attached to a document, it gets marked as **sealed**. Calling `Transform` on a sealed table now returns an error telling you to make a copy first (`Clone()`) and transform the copy. Tables that were never attached to a document still work exactly as before.

### Why It Matters

Programs rely on the immutability promise — for example, sharing one document between multiple threads that each render it. If the data can silently change underneath them, you get wrong output or crashes that are very hard to trace. This fix turns a silent corruption into a clear, immediate error message.

### Key Concepts

- **Immutability**: an object that cannot change after creation. Like a printed report — you can photocopy and edit the copy, but the original stays as printed.
- **Sealing**: a one-way flag flipped when a table joins a document. Think of it as laminating the page.
- **Clone**: making an independent copy. The copy is *not* sealed, so you may transform it freely without touching the document.

---

## Intermediate Level

### Changes Overview

- `v2/content.go` — `TableContent` gains an unexported `sealed atomic.Bool` and a `seal()` method. `Clone()` deliberately returns an unsealed copy and its godoc now says so.
- `v2/transform_data.go` — `Transform` (1) errors early when `sealed` is set, and (2) passes the transform function `tc.Records()` (a defensive copy: new slice, new record maps) instead of the live `tc.records`. New `sealContents(Content)` helper recursively seals tables nested in `SectionContent` and `DefaultCollapsibleSection`, with typed-nil guards.
- `v2/document.go` — `Builder.AddContent` calls `sealContents(content)` before appending. All thirteen content-adding builder methods funnel through `AddContent`, so every table reachable from a document is sealed.
- `v2/transform_data_immutability_test.go` — regression tests: sealed transform errors (top-level, in-section, in-collapsible), clone-remains-transformable, standalone-still-works, failed-transform-leaves-records-unchanged, and typed-nil AddContent does not panic.

### Implementation Approach

Seal-at-attach mirrors the enforcement pattern from T-1689 (builder records errors on post-`Build()` mutation): keep the fluent API non-panicking, surface violations as errors. The flag is an `atomic.Bool` because `Transform` may be called from a different goroutine than the builder; a plain bool would be a data race on the flag itself.

Passing `Records()` to the transform function fixes a second defect independently of sealing: previously a function could mutate the live record maps and then return an error, leaving the table half-mutated. Now the table is only assigned when the function succeeds.

### Trade-offs

- **Sealing vs deep-cloning in `GetContents()`**: cloning on read would preserve mutability but change pointer-identity semantics for all callers and add cost to every render. Rejected.
- **Sealing at `AddContent` vs at `Build()`**: `AddContent` is strictly earlier (the document shares the pointer from that moment) and avoids touching `Build()`, which the parallel T-1543 PR (#103) is restructuring for `SectionContent` freezing.
- **Error vs silent no-op vs panic**: an error matches `Transform`'s existing signature and the library's non-panicking contract.
- The records copy handed to `fn` is shallow per record (matching `Records()` semantics): new slice, new maps, nested reference values shared. Documented rather than deep-copied — a full deep copy of arbitrary `any` values would need reflection and still couldn't handle unexported fields reliably.

---

## Expert Level

### Technical Deep Dive

`sealContents` is a type switch over the only two container content types (`*SectionContent.contents`, `*DefaultCollapsibleSection.content`) plus `*TableContent`. Each case guards against typed-nil pointers: `Builder.AddContent`'s `content == nil` check only catches an untyped nil interface, so `(*SectionContent)(nil)` reaches the walk — round 1 of review found this panic, fixed in `f5c9dd6`, with regression coverage in `TestAddContentTypedNilDoesNotPanic` (pre-fix behaviour was to store the typed nil untouched, so ignoring it preserves main's semantics).

Sub-builder paths (`Section`, `CollapsibleSection`) seal twice — once when the inner builder's `AddContent` runs, once when the outer walk descends into the section. `seal()` is an idempotent atomic store, so this is harmless.

The sealed check runs before any read of `tc.records`, which is what closes the render race: a sealed `Transform` cannot write concurrently with renderer reads. A residual TOCTOU window exists — `Transform` concurrent with the *initial* `AddContent` could pass the check before the seal lands — but any program in that situation was already racing unsynchronized builder/content access on main; the guarantee is for happens-before-ordered attach-then-share usage.

`go vet` copylocks is clean: `atomic.Bool` embeds `noCopy`, but `Clone()` builds a fresh composite literal rather than copying `*t`, and no code passes `TableContent` by value.

### Architecture Impact

- `TransformableContent` interface unchanged — no breaking API change within v2. Its godoc now licenses implementations to refuse attached content.
- Establishes the seal/freeze precedent that T-1543 (PR #103) extends to `SectionContent` at `Build()` time. The two PRs touch the same files in different functions; merge order matters only for textual conflicts.
- The known remaining escape — exported `SectionContent.AddContent` can attach an unsealed table to an already-sealed section — is explicitly scoped to T-1543.

### Potential Issues

- The sealed error is a bare `fmt.Errorf` (no sentinel), so callers cannot `errors.Is`-match it to trigger the documented clone-and-retry remedy without string matching. Consistent with current package style; an `ErrContentSealed` sentinel is a possible follow-up.
- Nested reference values inside records remain shared with the transform function's copy; a function that mutates shared nested state before failing can still leak that mutation. Documented in the godoc, CHANGELOG, and report.
- CI does not run `-race`, so the concurrency aspect is enforced structurally (check-before-read) rather than by detector; the new tests pass under `-race` locally.

## Completeness Assessment

- **Fully implemented**: sealing on every builder attach path (verified: all 13 content-adding methods route through `AddContent`; the only `doc.contents` append is inside it); sealed `Transform` rejection; failed-transform record protection; typed-nil tolerance; unsealed clones; regression tests; CHANGELOG and bugfix report.
- **Partially implemented (deliberate scope)**: post-attach mutation via `SectionContent.AddContent` is left to T-1543; nested-value sharing in the records copy is documented, not eliminated.
- **Missing**: nothing required by the ticket. Every claim in the bugfix report was verified against the code during review.
