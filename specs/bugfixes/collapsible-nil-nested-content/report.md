# Bugfix Report: Collapsible Sections Panic on Nil Nested Content

**Ticket:** T-1472
**Date:** 2026-08-17
**Status:** Fixed

## Description of the Issue

`NewCollapsibleSection` defensively copies the caller-provided `[]Content`
slice (T-1317, aliasing protection) but preserves nil entries. Every
downstream consumer assumes nested content is non-nil, so a single nil entry
turns into a nil pointer dereference panic far from the call that introduced
it:

- `DefaultCollapsibleSection.AppendText` calls `content.AppendText(nil)` on
  each nested item.
- `DefaultCollapsibleSection.Clone` calls `content.Clone()`.
- The collapsible-section paths of the Markdown, HTML, JSON, YAML, CSV, and
  table renderers iterate `section.Content()` and call
  `applyContentTransformations(ctx, content)` (or type-switch and call
  methods) without nil guards, so public `Render` calls panic.

The table helpers make it worse: `NewCollapsibleTable(title, nil)` and
`NewCollapsibleMultiTable` with a nil table wrap a nil `*TableContent`
pointer into a non-nil interface value (a typed nil), which even an
interface-level `content == nil` check cannot detect. (Renderer-side scope
merged from T-1570.)

**Reproduction steps:**
1. `section := output.NewCollapsibleSection("bad", []output.Content{nil})`
2. `doc := output.New().AddContent(section).Build()`
3. `output.Markdown().Renderer.Render(context.Background(), doc)` — panics
   with `runtime error: invalid memory address or nil pointer dereference`
4. `section.Clone()` — same panic. `output.NewCollapsibleTable("t", nil)`
   followed by `Clone()` panics via the typed-nil route.

**Impact:** Any caller passing nil nested content — directly or via a nil
table variable — crashes the process during rendering or cloning instead of
receiving normal output or an error. All six format renderers are affected
through their public `Render` paths.

## Investigation Summary

- **Symptoms examined:** Nil pointer dereference panics in `Clone`,
  `AppendText`, and all renderer collapsible paths, reproduced by the new
  regression tests before the fix (`(*TableContent).Clone(0x0)` in the
  typed-nil case).
- **Code inspected:** `v2/collapsible_section.go` (constructor, helpers,
  `AppendText`, `Clone`), `v2/document.go` (`Builder.AddContent`,
  `AddCollapsibleSection`, `freezeSectionContents`), `v2/content.go`
  (`SectionContent.AddContent`/`Clone` conventions), `v2/transform_data.go`
  (`sealContents` typed-nil guards from T-1677), `v2/renderer.go`
  (`applyContentTransformations`), and the collapsible paths of
  `markdown_renderer.go`, `html_renderer.go`, `json_yaml_renderer.go`,
  `csv_renderer.go`, `table_renderer.go`.
- **Hypotheses tested:** Confirmed nothing between construction and the
  dereference can catch the nil: the constructor stores it, `Content()`
  copies it out, and neither the section's own methods nor any renderer loop
  guards entries. Confirmed the freeze/seal walkers (T-1543/T-1677) already
  tolerate nil and typed-nil entries, so `Build()` succeeds and the panic
  surfaces only at render/clone time.

Root cause chain (Five Whys):
1. Why does rendering/cloning panic? Nested content entries are nil and
   consumers call interface methods on them unconditionally.
2. Why are nil entries present? `NewCollapsibleSection` copies the caller's
   slice verbatim; nils survive the copy.
3. Why does the constructor keep them? The T-1317 defensive copy addressed
   slice aliasing only; element validity was never checked.
4. Why do consumers assume non-nil? The builder pipeline enforces the
   invariant (`Builder.AddContent` rejects nil, `SectionContent.AddContent`
   drops nil), so downstream code trusted it — but the collapsible
   constructor is a public entry point that bypasses that enforcement.
5. Why do the table helpers dodge even an interface-nil check? They convert
   concrete nil pointers into typed-nil interface values — the same gap
   T-1677 documented in `sealContents`.

## Discovered Root Cause

`NewCollapsibleSection` — the single choke point through which all
collapsible-section content flows — lacks the nil-filtering invariant the
rest of the content pipeline enforces, and downstream consumers
(`AppendText`, `Clone`, and six renderer paths) rely on that invariant
without defensive guards.

**Defect type:** Missing validation

**Why it occurred:** The constructor predates the pipeline-wide nil
hardening; later fixes (nil content in `Builder.AddContent`,
`SectionContent.AddContent`, the freeze/seal walkers) never audited the
collapsible constructor or the renderer collapsible loops.

**Contributing factors:** Nested content is only reachable through the
constructor in normal use, so the invariant held whenever callers passed
valid content and the gap never surfaced. Typed nils from the table helpers
are invisible to the conventional `content == nil` check.

## Resolution for the Issue

_To be completed after the fix is implemented._

## Regression Test

**Test file:** `v2/collapsible_nil_content_test.go`
**Test names:** `TestNewCollapsibleSectionDropsNilContent`,
`TestCollapsibleTableHelpersDropNilTables`,
`TestCollapsibleSectionNilContentMethodsNoPanic`,
`TestRenderCollapsibleSectionNilContent`,
`TestRenderersTolerateMalformedCollapsibleSection`

**What it verifies:**
- The constructor drops nil entries (alone, and mixed with valid content).
- The table helpers drop nil `*TableContent` values instead of storing typed
  nils; `Clone` no longer panics.
- `AppendText` and `Clone` tolerate a malformed section built via struct
  literal (defence-in-depth guards).
- The ticket's reproduction renders through all six public format paths and
  clones without panicking.
- Every renderer skips a nil entry injected past the constructor and still
  renders the valid sibling content.

**Run command:** `go test -run 'TestNewCollapsibleSectionDropsNilContent|TestCollapsibleTableHelpersDropNilTables|TestCollapsibleSectionNilContentMethodsNoPanic|TestRenderCollapsibleSectionNilContent|TestRenderersTolerateMalformedCollapsibleSection' ./v2/...`

## Affected Files

| File | Change |
|------|--------|
| `v2/collapsible_section.go` | Constructor filters nil entries; helpers skip nil tables; `AppendText`/`Clone` defensive guards |
| `v2/markdown_renderer.go` | Skip nil entries in collapsible section loop |
| `v2/html_renderer.go` | Skip nil entries in collapsible section loop |
| `v2/json_yaml_renderer.go` | Skip nil entries in JSON and YAML collapsible section loops |
| `v2/csv_renderer.go` | Skip nil entries in collapsible section loop |
| `v2/table_renderer.go` | Skip nil entries in collapsible section loop |
| `v2/collapsible_nil_content_test.go` | Regression tests |
| `CHANGELOG.md` | Changelog entry |

## Verification

**Automated:**
- [ ] Regression test passes
- [ ] Full test suite passes (`make test`)
- [ ] Linters/validators pass (`make lint`)

**Manual verification:**
- Confirmed the regression tests fail with the original panics before the
  fix (constructor keeps nils; `(*TableContent).Clone` on typed nil
  segfaults).

## Prevention

**Recommendations to avoid similar bugs:**
- Any constructor that stores caller-provided `Content` values should
  enforce the same non-nil invariant as `Builder.AddContent` /
  `SectionContent.AddContent`; the content pipeline's consumers assume it.
- Helpers that wrap concrete pointers into interface slices should nil-check
  the concrete pointer before wrapping — after wrapping, the nil is
  undetectable without reflection.
- When T-1649 (consolidated typed-nil validation) lands, the collapsible
  constructor should adopt the shared helper to also catch typed nils passed
  directly inside `[]Content`.

## Related

- T-1472 (this fix; T-1570 renderer-side scope merged in)
- T-1543 — section freeze semantics; `freezeSectionContents` nil guards
- T-1677 — seal semantics; `sealContents` typed-nil guards and the
  documented typed-nil gap in `Builder.AddContent`
- T-1649 — consolidated typed-nil validation (open)
- specs/bugfixes/stdio-writer-nil-setwriter — precedent for deferring
  one-off reflection helpers to T-1649
