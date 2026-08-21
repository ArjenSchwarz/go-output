# Implementation Explanation: WithDataPreservation(false) honored by RawContent (T-1557)

Branch: `T-1557/bugfix-raw-data-preservation` — diff against `origin/main`, commits `ff7e82b`, `ec1e437`, `9c4770a`.

## Beginner Level

### What Changed

The go-output library lets you add "raw content" (a chunk of bytes, like an HTML snippet) to a document. When you hand it your bytes, the library normally makes its own private copy, so if you later change your original bytes, the document is unaffected.

There is an option called `WithDataPreservation(false)` whose documentation says: "skip that copy". Skipping the copy is useful when the byte slice is large (a big CSV or SVG blob) and you don't want to pay for a duplicate in memory. The bug: the option did nothing. The library copied your bytes no matter what you asked for. Worse, a unit test asserted the wrong behavior ("data should still be copied even when preservation is disabled"), so the contradiction was locked in.

The fix makes the option work as documented: pass `WithDataPreservation(false)` and the library now keeps a reference to *your* slice instead of copying it. Nothing changes if you don't use the option — copying is still the default.

### Why It Matters

- Users who opted out of copying to save memory were silently paying for the copy anyway.
- The documentation, the tests, and the code all disagreed with each other — a contract violation even if nothing "crashed".
- The trade-off is now real and documented: if you opt out, you must not modify your slice afterwards, because the document is looking at the same bytes you are.

### Key Concepts

- **Byte slice**: Go's `[]byte` is a view onto an underlying array. Two slices can point at the same bytes — changing one is visible through the other. That sharing is called **aliasing**.
- **Defensive copy**: making a private duplicate of input data so the caller can't change it under you.
- **Ownership transfer**: an API pattern ("the caller should not use buf after this call" — like `bytes.NewBuffer`) where you hand your data over and promise to stop touching it.
- **Functional option**: the `WithX(...)` configuration functions passed to a constructor.

## Intermediate Level

### Changes Overview

- `v2/content.go` — `NewRawContent` now reads `rawConfig.preserveData` (previously written by the option but never read anywhere in non-test code). When true (default), it copies as before; when false, it stores the caller's slice directly. Godoc updated.
- `v2/raw_options.go` — `WithDataPreservation` godoc rewritten to state the full contract: default copies; `false` aliases the caller's slice; after opting out the caller must not mutate the slice; `Data()` and `Clone()` still return copies regardless.
- `v2/raw_content_test.go` — the assertion that codified the bug (copy-on-disable) was replaced. `TestRawContent_DataPreservation` now covers the preserve side (explicit `true` and the default), with a doc comment noting the split; new regression test `TestRawContent_DataPreservationDisabled_AliasesInput` asserts that mutating the original slice is visible through the content.
- `CHANGELOG.md` — Unreleased/Fixed entry.
- `specs/bugfixes/raw-data-preservation/report.md` — full investigation report (root cause: dead configuration / unwired option).

### Implementation Approach

The change is deliberately minimal — one conditional in the constructor:

```go
contentData := data
if rc.preserveData {
    contentData = make([]byte, len(data))
    copy(contentData, data)
}
```

The read-side defenses are untouched: `RawContent.Data()` still returns a fresh copy on every call, and `Clone()` still deep-copies. So opting out of the construction copy never exposes the internal slice to *other* consumers — the only aliasing path is the caller's own retained slice.

### Trade-offs

- **Honor `false` vs. remove the option**: removing `WithDataPreservation` is a breaking v2 API change; leaving it a documented no-op keeps misleading "performance mode" surface. Honoring it matches the option's name and godoc. (From the report's rationale.)
- **Aliasing hazard vs. memory**: the opt-out reintroduces a mutation channel, but an *explicit, opt-in* one — consistent with the recent immutability work (T-1543, T-1677), which closed *implicit* channels. Default behavior is unchanged.
- **Test replacement**: the old disable-case assertion tested the implementation ("we copy anyway") rather than the contract. Replacing a wrong test is the sanctioned exception to "don't touch tests".

## Expert Level

### Technical Deep Dive

`rawConfig.preserveData` defaults to `true` in `ApplyRawOptions`, so every existing call site — including `Builder.Raw`, which forwards options verbatim — keeps copy semantics. The only behavioral delta is for callers who already passed `WithDataPreservation(false)` and were previously getting an undocumented copy; they now get the aliasing they asked for. That is technically an observable behavior change for existing binaries, justified as a bug fix because the previous behavior contradicted the documented contract.

Internal safety of the aliased slice rests on two invariants, both verified in review: no non-test code mutates `RawContent.data` after construction (`Data()`/`Clone()` copy unconditionally; `AppendText`/`AppendBinary` only append the bytes outward into the caller's buffer; renderers go through `Data()` or `AppendText`; `applyContentTransformations` clones before applying operations; `sealContents` is a no-op for `RawContent`), and no renderer retains the slice beyond a single append. `Data()`/`Clone()` copying means the aliased slice can never leak out of the content to third parties.

### Architecture Impact

- Establishes the ownership-transfer idiom (`bytes.NewBuffer`-style) as the library's shape for performance opt-outs, alongside the copy-by-default immutability direction of T-1543/T-1677. Future opt-outs (e.g., a table-data equivalent) should follow the same pattern: default safe, explicit opt-out, read-side copies intact.
- No API surface change; only semantics of an existing option.

### Potential Issues

- A caller who opts out and then mutates the slice concurrently with `Render` has a data race (unsynchronized read/write of shared bytes). This is inherent to aliasing and covered by the "must not modify" contract — the godoc's "must not modify" subsumes it.
- `Data()` still copies on every call; a caller who opted out at construction to avoid copies still pays per-call copies if they read data back via `Data()`. That is intentional (read-side defense) but worth knowing when reasoning about the performance benefit: the saving is on the construction/render path, not the read-back path.
- The aliasing test mutates its slice after `NewRawContent`; if future refactors make renderers cache raw bytes at build time, this test is the tripwire.

## Completeness Assessment

- **Fully implemented**: the option is wired into `NewRawContent`; both flag values have behavioral tests; godoc on both the constructor and the option documents the contract and hazard; CHANGELOG and bugfix report present; all tests and lint pass.
- **Partially implemented**: nothing identified.
- **Missing**: nothing required by the ticket. Optional hardening not done (acceptable): no explicit test that `Data()` returns a copy *when preservation is disabled* — it is implied by `Data()`'s unconditional copy and existing `Data()` tests, but a direct assertion would pin the read-side defense claim in the report.
