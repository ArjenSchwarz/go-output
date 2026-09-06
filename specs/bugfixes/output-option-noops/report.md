# Bugfix Report: WithTableStyle/WithTOC/WithFrontMatter Output Options Are Silent No-Ops

**Ticket:** T-1516
**Date:** 2026-08-22
**Status:** Fixed

## Description of the Issue

The Output-level options `WithTableStyle`, `WithTOC`, and `WithFrontMatter` in
`v2/output.go` set fields on the `Output` struct (`tableStyle`, `hasTOC`,
`frontMatter`) that nothing in the render path read. The options are documented
as "for v1 compatibility", but using them had no effect and produced no error.
The actual behaviour only existed on the parallel Format constructors
(`TableWithStyle`, `MarkdownWithToC`, `MarkdownWithFrontMatter`,
`MarkdownWithOptions`).

**Reproduction steps:**
1. Build a document with a header and a table.
2. Render it with `NewOutput(WithFormat(Markdown()), WithTOC(true), WithWriter(w)).Render(ctx, doc)`.
3. Observe: the markdown output contains no table of contents, and no error is returned. Same pattern for `WithFrontMatter` (no front matter emitted) and `WithTableStyle` (default table style used).

**Impact:** Silent misbehaviour targeted at exactly the audience these options
exist for — v1 migrators. A migrated call site compiles, runs, and quietly
drops the requested table style, ToC, or front matter.

The same class of dead API surface: `Format.Options map[string]any`
(`v2/renderer.go`) is never written or read by any library code.

## Investigation Summary

Followed the Fagan inspection phases (overview, inspection, five-whys, solution).

- **Symptoms examined:** Options accepted at construction with no rendering effect and no error.
- **Code inspected:** `v2/output.go` (options, `Render`, `renderWithConfig`), `v2/renderer.go` (`Format`, format constructors), `v2/markdown_renderer.go` and `v2/table_renderer.go` (renderer structs and constructors), `v2/base_renderer.go` (embedded `baseRenderer` with `sync.RWMutex`).
- **Hypotheses tested:** Grep for reads of `o.tableStyle`/`o.hasTOC`/`o.frontMatter` confirmed the fields are write-only. Grep for `Format.Options` confirmed it has no readers or writers anywhere in the library. Existing tests (`output_test.go`, `validation_test.go`) only assert that `NewOutput` succeeds — no behavioural assertions, which is why the no-op persisted.

## Discovered Root Cause

`Render` snapshots formats/writers/transformers/progress under the read lock
but never consults the v1-compat fields. Rendering behaviour lives entirely
inside `Renderer` instances constructed by the Format constructors, and the
Output-level fields were never plumbed into renderer construction.

**Defect type:** Data-flow defect / incomplete feature wiring (dead stores).

**Why it occurred:** During the v2 redesign, two parallel configuration
surfaces were created for the same v1-compat features — Output options and
Format constructors — but only the Format-constructor surface was wired to the
render path.

**Contributing factors:** Construction-only test coverage for these options;
no compile-time or runtime signal that the fields were unused.

## Resolution for the Issue

**Changes made:**
- `v2/output.go` - `Render` now snapshots `tableStyle`, `hasTOC`, and `frontMatter` under the read lock and derives effective formats via a new `applyV1CompatOptions` helper, placed after `validateConfigEntries` (so typed-nil renderers are rejected before the type assertions) and before rendering.
- `v2/output.go` - `applyV1CompatOptions` reconfigures matching built-in renderers: table formats backed by `*tableRenderer` get the configured style (preserving max column width and collapsible config); markdown formats backed by `*markdownRenderer` get ToC enabled and/or front matter merged (Output-level keys win on conflict), preserving heading level, collapsible config, and base renderer config. Renderers of other formats, and custom `Renderer` implementations, are left untouched.
- `v2/table_renderer.go` / `v2/markdown_renderer.go` - The per-renderer copy logic lives in unexported helpers next to the struct definitions (`(*tableRenderer).withStyle`, `(*markdownRenderer).withCompatOptions`), so a field added to either renderer is added to its copy in the same file rather than silently reverting to its zero value in the derived renderer.
- `v2/output.go` - Option doc comments updated to describe the wired behaviour and its additive semantics (`WithTOC(false)` is the zero value and does not disable a ToC enabled via `MarkdownWithToC(true)`).
- `v2/renderer.go` - `Format.Options` documented as informational only: it is not consumed by the render pipeline (see decision below).

**Approach rationale:** The ticket prescribes deriving effective renderers in
`Render`; doing it at render time keeps `NewOutput` order-independent
(`WithTableStyle` before or after `WithFormat` behaves the same) and keeps the
change strictly additive — configurations that do not use these options render
exactly as before. Cloning the existing renderer rather than calling the
public constructors preserves orthogonal settings (max column width,
collapsible config, heading level). `markdownRenderer` embeds `baseRenderer`
which contains a `sync.RWMutex`, so the clone copies fields explicitly (config
read under its lock) instead of copying the struct, keeping `go vet`'s
copylocks clean.

**Alternatives considered:**
- Applying the options inside `NewOutput` by rewriting `o.formats` — rejected: option application order would matter (`WithTableStyle` before `WithFormat` would see no formats yet).
- Reconstructing renderers via the public constructors (`NewTableRendererWithStyle` etc.) — rejected: silently discards orthogonal renderer settings such as max column width.
- Returning a validation error when the options are set without a matching format — rejected as out of scope: the options are format-targeted hints, and erroring would break existing (currently harmless) usage.

**Decision on `Format.Options` (per ticket):** documented, not wired. Nothing
in the library or its examples ever writes or reads the field, there are no
defined key semantics to honour, and inventing them would add a third
configuration surface for behaviour the Format constructors and Output options
already cover. Removing the exported field would be an API break, which the
ticket rules out. The field now carries a doc comment stating it is
informational metadata not consumed by the render pipeline.

## Regression Test

**Test file:** `v2/output_v1_compat_options_test.go`
**Test names:**
- `TestWithTOCEnablesMarkdownToC`
- `TestWithFrontMatterAddsMarkdownFrontMatter`
- `TestWithTableStyleAppliesStyleToTableFormat`
- `TestWithTableStylePreservesMaxColumnWidth`
- `TestOutputCompatOptionsCombineWithFormatConstructors`
- `TestCompatOptionsSkipCustomRenderers`
- `TestWithTOCFalseLeavesConstructorToCEnabled`
- `TestWithFrontMatterMergesConstructorEntries`
- `TestCompatOptionsDoNotMutateStoredFormats`
- `TestCompatOptionsLeaveOtherFormatsUntouched`

**What it verifies:** Each Output-level option affects the rendered bytes of
its matching format; style override preserves constructor-configured max
column width; Output options combine with Format-constructor settings;
`WithTOC(false)` does not disable a constructor-enabled ToC (additive
semantics); front matter merges with constructor entries and Output-level
keys win on conflict; custom `Renderer` implementations registered under the
table or markdown format names are skipped entirely; stored formats are not
mutated (a `Format` shared between two Outputs does not leak compat options);
and non-matching formats (JSON) are byte-identical with and without the
options.

**Run command:** `cd v2 && go test -run 'TestWithTOC|TestWithFrontMatter|TestWithTableStyle|TestOutputCompatOptions|TestCompatOptions' ./`

The core behavioural tests (`TestWithTOCEnablesMarkdownToC`,
`TestWithFrontMatterAddsMarkdownFrontMatter`,
`TestWithTableStyleAppliesStyleToTableFormat`,
`TestWithTableStylePreservesMaxColumnWidth`,
`TestOutputCompatOptionsCombineWithFormatConstructors`) failed before the fix
(red) and pass after (green); the JSON guard passed in both states. The
remaining tests were added during review to lock in the documented invariants
(custom-renderer skip, additive zero values, merge conflict rule, no mutation
of stored formats).

## Affected Files

| File | Change |
|------|--------|
| `v2/output.go` | Snapshot compat fields in `Render`; add `applyV1CompatOptions`; update option docs |
| `v2/table_renderer.go` | Add unexported `withStyle` copy helper next to the struct |
| `v2/markdown_renderer.go` | Add unexported `withCompatOptions` copy helper next to the struct |
| `v2/renderer.go` | Document `Format.Options` as not consumed by the render pipeline |
| `v2/output_v1_compat_options_test.go` | New regression tests |
| `v2/docs/MIGRATION.md` | Note on additive semantics of the compat options |
| `CHANGELOG.md` | Entry under Unreleased / Fixed |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass (`make check`)

**Manual verification:**
- Inspected rendered output in test failures/successes to confirm ToC, front matter, and Bold box-drawing characters appear only when requested.

## Prevention

**Recommendations to avoid similar bugs:**
- When adding an option, add a behavioural test that asserts its effect on output, not just that construction succeeds.
- Avoid parallel configuration surfaces for the same feature; when unavoidable, wire both to the same implementation in the same change.
- Periodically grep for write-only struct fields (dead stores) in exported configuration paths.

## Related

- Transit ticket T-1516
- Render path restructure: T-1524 (two-phase transform/write), T-1649 (typed-nil validation)
