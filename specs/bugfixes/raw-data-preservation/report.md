# Bugfix Report: WithDataPreservation(false) is ignored by RawContent

**Ticket:** T-1557
**Date:** 2026-08-17
**Status:** Fixed

## Description of the Issue

`WithDataPreservation(false)` records `rawConfig.preserveData = false` and its
godoc describes it as enabling or disabling data preservation (copying). Tests
describe the combination with `WithFormatValidation(false)` as a "performance
mode". In reality `NewRawContent` always copied the input byte slice regardless
of the flag, so the public option was a no-op. `TestRawContent_DataPreservation`
even asserted that data is copied when preservation is disabled, codifying the
contradiction of the option contract.

**Reproduction steps:**
1. `data := []byte("original data")`
2. `content, _ := output.NewRawContent(output.FormatText, data, output.WithDataPreservation(false))`
3. Mutate `data[0]` and observe that `content.Data()` still holds the original
   bytes — the "disable copying" opt-out did nothing.

**Impact:** Low severity, but a public API contract violation. Callers passing
`WithDataPreservation(false)` to avoid a copy of large raw payloads (HTML, SVG,
CSV blobs) silently paid the copy anyway, and the option's documented behavior,
the tests, and the implementation all disagreed with each other.

## Investigation Summary

- **Symptoms examined:** `WithDataPreservation(false)` has no observable effect
  on `NewRawContent`; `TestRawContent_DataPreservation` asserts copy-on-disable.
- **Code inspected:** `v2/raw_options.go` (option and defaults),
  `v2/content.go` (`NewRawContent`, `Data`, `Clone`, `AppendText`/
  `AppendBinary`), `v2/document.go` (`Builder.Raw` pass-through),
  `v2/raw_options_test.go`, `v2/raw_content_test.go`, and all renderer call
  sites of `RawContent.Data()`.
- **Hypotheses tested:** Checked whether `preserveData` was consumed anywhere
  outside tests (it was not — only written, never read in non-test code), and
  whether any internal code path mutates `RawContent.data` after construction
  (none does; renderers only read it).

## Discovered Root Cause

`rawConfig.preserveData` is set by `WithDataPreservation` and defaulted to
`true` in `ApplyRawOptions` (`v2/raw_options.go`), but `NewRawContent`
(`v2/content.go`) never read the flag — it unconditionally copied the input
slice.

**Defect type:** Dead configuration / unwired option (logic omission).

**Why it occurred (Five Whys):**
1. The option is a no-op because `NewRawContent` never consults
   `rc.preserveData`.
2. The option/config plumbing was written separately from the constructor,
   which was implemented with an unconditional defensive copy and never wired
   to the flag.
3. The behavioral test did not catch it because it was written to codify the
   observed behavior ("data should still be copied even when preservation is
   disabled") instead of the option's documented contract.
4. The contradiction was masked by a reassuring comment ("for safety") that
   made the wrong assertion look intentional.
5. No tooling flagged it: `preserveData` is a private field exercised by unit
   tests of the config struct itself, so it never registered as unused.

**Contributing factors:** The codebase has been moving strongly toward
defensive copying and immutability (T-1543, T-1677), which made "always copy"
look like the safe interpretation even though it contradicted the public
option's documentation.

## Resolution for the Issue

**Changes made:**
- `v2/content.go` — `NewRawContent` now copies the input only when
  `rc.preserveData` is true; with `WithDataPreservation(false)` it stores the
  caller's slice directly. Godoc updated to state the default and the opt-out.
- `v2/raw_options.go` — `WithDataPreservation` godoc now documents the
  contract precisely, including the aliasing hazard: after opting out, the
  caller must not modify the slice, because changes become visible in every
  subsequent render and violate the document immutability guarantees.
- `v2/raw_content_test.go` — the disable-case assertion that contradicted the
  contract was replaced by `TestRawContent_DataPreservationDisabled_AliasesInput`,
  which asserts the documented aliasing behavior; the copy assertions for the
  default and explicit `true` cases remain.
- `CHANGELOG.md` — entry under Unreleased/Fixed.

**Approach rationale:** The ticket allowed either implementing the option as
documented or removing it. Removing `WithDataPreservation` would be a breaking
API change for v2 consumers; keeping it as a permanent no-op would leave dead,
misleading "performance mode" API surface. Honoring `false` as an explicit
performance opt-out is consistent with the immutability direction of
T-1543/T-1677: those fixes closed *implicit* mutation channels that surprised
callers, whereas this is an explicit, opt-in ownership transfer in the style of
`bytes.NewBuffer` ("the caller should not use buf after this call"). The safe
behavior (copying) remains the default, and the read-side defenses are
untouched: `Data()` still returns a copy and `Clone()` still deep-copies, so
opting out at construction never hands mutable access to the internal slice to
other consumers.

**Alternatives considered:**
- Deprecate `WithDataPreservation` and document that data is always copied —
  rejected: keeps a dead option whose name promises behavior it does not have.
- Remove the option entirely — rejected: breaking public API change,
  disproportionate to the defect.

## Regression Test

**Test file:** `v2/raw_content_test.go`
**Test name:** `TestRawContent_DataPreservationDisabled_AliasesInput`

**What it verifies:** With `WithDataPreservation(false)`, `NewRawContent`
stores the caller's slice without copying, so a mutation of the original slice
is visible through the content. The sibling test
`TestRawContent_DataPreservation` verifies that the default and explicit
`WithDataPreservation(true)` still copy.

**Run command:** `go test -run 'TestRawContent_DataPreservation' ./v2/`

## Affected Files

| File | Change |
|------|--------|
| `v2/content.go` | `NewRawContent` honors `preserveData`; godoc updated |
| `v2/raw_options.go` | `WithDataPreservation` godoc documents contract and aliasing hazard |
| `v2/raw_content_test.go` | Regression test added; contradictory assertion removed |
| `CHANGELOG.md` | Unreleased/Fixed entry |
| `specs/bugfixes/raw-data-preservation/report.md` | This report |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed no non-test code reads `RawContent.data` mutably and no renderer
  retains the slice beyond a single append, so aliasing only affects callers
  who explicitly opted out and then mutate their own slice.

## Prevention

**Recommendations to avoid similar bugs:**
- When adding a functional option, wire it into the consuming constructor in
  the same change and add a behavioral test for both values of the flag —
  config-struct tests alone only prove the plumbing, not the behavior.
- Treat a test comment that explains away a contradiction with the public
  godoc ("should still be copied even when preservation is disabled") as a
  review red flag: tests must assert the documented contract, not the current
  implementation.

## Known Limitation

The opt-out removes only the construction-time copy. Renderers read raw
content through `RawContent.Data()`, which returns a defensive copy on every
call (and several call sites convert via `string(raw.Data())`, copying twice),
so each render still pays a copy regardless of `WithDataPreservation(false)`.
This is pre-existing behavior, out of scope for this fix; T-2233 tracks the
render-side accessor change.

## Related

- Transit ticket T-1557
- T-1543 / T-1677 — the immutability work this decision was weighed against
- T-2233 — follow-up: renderers copy RawContent via Data() on every render,
  undermining the opt-out's render-path benefit
