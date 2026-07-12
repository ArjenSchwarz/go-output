# Implementation Explanation: Freeze Section Contents at Build (T-1543)

Branch: `T-1543/bugfix-mutable-section-contents` — explains the fix for built documents exposing mutable section contents.

## Beginner Level

### What Changed

The go-output library lets you build a "document" out of pieces of content (tables, text, sections) and then render it to formats like Markdown or HTML. The rule is: once you call `Build()`, the document is finished and can never change.

Before this fix, that rule had a hole. Asking a built document for its contents (`GetContents()`) handed back the *actual* section objects inside it — not copies. Anyone holding one of those sections could call its `AddContent` method and quietly stuff new content into the "finished" document. Rendering it again would show the injected content.

Now, `Build()` walks through the document and marks every section as **frozen**. Calling `AddContent` on a frozen section simply does nothing. If you genuinely need to modify a section from a built document, you call `Clone()` to get your own unfrozen copy.

### Why It Matters

Programs rely on the promise that a built document never changes: they may render it several times, or render it from multiple threads at once. Before the fix, a post-build modification could change what gets rendered — or worse, corrupt memory when a modification happened at the same moment as a render (a "data race"). Now the document really is read-only after `Build()`.

### Key Concepts

- **Immutability**: an object that cannot change after creation. Like a printed report — you can photocopy it (Clone) but not edit the original.
- **Freezing**: flipping a one-way switch on an object that disables its modification methods.
- **Data race**: two threads touching the same memory at the same time, at least one writing. Outcomes are unpredictable; Go's `-race` detector finds them.

---

## Intermediate Level

### Changes Overview

Two production files changed:

- `v2/content.go` — `SectionContent` gains a `frozen atomic.Bool`. `AddContent` returns early when frozen (alongside the existing nil-drop). New exported `Frozen()` predicate. `Clone()` documented as the escape hatch: clones are never frozen.
- `v2/document.go` — `Builder.Build()` now calls `freezeSectionContents` on each top-level content value after finalizing the document. The helper recursively freezes every `*SectionContent` reachable through nested sections and through `*DefaultCollapsibleSection` contents. `Section()` and `CollapsibleSection()` harvest their sub-builders through a new unexported, non-freezing `build()`.

Plus `v2/section_content_immutability_test.go` (9 regression tests) and a CHANGELOG entry.

### Implementation Approach

The immutability boundary moves from the Builder surface (T-1689 guarded builder methods post-build) onto the content graph itself. `GetContents()`, `SectionContent.Contents()`, and `DefaultCollapsibleSection.Content()` all copy their slices but share element pointers — by design, to avoid deep copies on every render. So the objects themselves must refuse mutation.

The `Build()`/`build()` split exists because `Section(title, func(b *Builder))` composes sections by running the callback against a sub-builder and harvesting its document. Round-1 review found that harvesting via the public `Build()` froze caller-held sections the moment the callback returned — before the outer document was built. The unexported `build()` finalizes (detaches the document from the builder) without freezing; only the user-facing `Build()` freezes.

`AddContent` is the only exported mutator on `SectionContent`, so guarding it closes the whole mutation surface.

### Trade-offs

- **Deep-clone in `GetContents()`** — correct but expensive: `baseRenderer` calls `GetContents()` several times per render, so every render would deep-copy the document; pointer-identity semantics would also change for existing callers. Rejected.
- **Mutex on `SectionContent`** — fixes the race but not the bug: output could still change after build. Rejected.
- **Error or panic on frozen `AddContent`** — `SectionContent` has no error channel and the v2 fluent API is deliberately non-panicking; the silent no-op matches `AddContent`'s existing silent nil-drop. The `Frozen()` predicate gives callers an up-front check.

Cost of the chosen design: one atomic load per `AddContent`, one tree walk per `Build()`. Zero per-render cost.

---

## Expert Level

### Technical Deep Dive

The race-freedom argument: `Build()` completes the freeze walk before the `*Document` is returned, so the handoff of the document to any other goroutine establishes happens-before for `frozen.Store(true)`. Any post-build `AddContent` therefore observes `frozen == true` and returns without touching `s.contents` — no write ever overlaps a renderer's recursive `Contents()` reads. The atomic (rather than a plain bool) additionally keeps the *misuse* path defined: a goroutine that retained a section pointer from a callback and calls `AddContent` with no synchronization against `Build()` gets a defined read of the flag rather than a racy one.

The freeze walk lives in `Build()` outside the builder mutex, which is safe: `build()` detaches the document (sets `b.doc = nil`) under the mutex first, so no builder method can reach the detached document, and `frozen` is atomic. A second `Build()` call returns nil (existing behavior); the walk guards `doc != nil`.

`freezeSectionContents` recurses through exactly the two content types that hold nested `Content`: `*SectionContent` and `*DefaultCollapsibleSection`. All other content types (`TableContent`, `TextContent`, `RawContent`, graph/chart/draw-io content) are leaves. The walker carries a maintenance note cross-referencing T-1677, which adds a second walker over the same graph — any future content type holding nested `Content` must be added to both.

`Clone()` returns an unfrozen copy at every level: it constructs new structs (fresh zero-value `atomic.Bool`) and deep-copies nested contents via their own `Clone()`, so a clone of a frozen graph is mutable throughout.

### Architecture Impact

- Completes the immutability story started by T-1689 (builder-surface guards): the guarantee now holds on the object graph, not just the factory.
- The transform pipeline is unaffected: it clones content before applying operations and assembles internal `Document` values directly (`base_renderer.go`), which never escape to callers — unfrozen internals there are fine.
- New public API: `SectionContent.Frozen()`. Small, orthogonal, and hard to misuse.
- Sibling fix T-1677 (seal-on-AddContent for `TableContent`) lands in the same files but different functions; the two walkers are documented as needing joint maintenance.

### Potential Issues

- **Silent no-op semantics**: a caller unaware of freezing gets no error. Mitigated by godoc on `AddContent`/`Build`/`GetContents`, the `Frozen()` predicate, and consistency with the existing nil-drop convention — but it remains a discoverability trade-off inherited from the fluent, non-panicking API design.
- **Freeze is one-way and per-object**: a frozen section added into a *new* builder stays frozen (its `AddContent` is dead). `Clone()` is the documented path; misuse shows up immediately in tests.
- **Walker/type drift**: a future content type with nested `Content` must be added to `freezeSectionContents` (and T-1677's walker). The maintenance note in the code is the guard.

## Completeness Assessment

- **Fully implemented**: freeze-at-Build for top-level, nested, and collapsible-wrapped sections; no-op `AddContent` with race-free atomic guard; `Frozen()` predicate; unfrozen `Clone()` escape hatch; non-freezing `build()` harvest for both `Section()` and `CollapsibleSection()`; godoc on all touched exported symbols; CHANGELOG entry; 9 regression tests including a `-race` concurrency test.
- **Partially implemented**: nothing identified.
- **Missing**: nothing identified against the bugfix report. Table sealing is explicitly out of scope (T-1677, parallel PR #101).
