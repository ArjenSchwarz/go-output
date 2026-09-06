# Implementation Explanation: Output-Level v1 Compatibility Options (T-1516)

Branch: `T-1516/bugfix-output-option-noops`
Scope: `WithTableStyle`, `WithTOC`, and `WithFrontMatter` now take effect during `Render`; `Format.Options` documented as not consumed by the render pipeline.

---

## Beginner Level

### What Changed

The go-output library lets you print data in different formats (tables, Markdown, JSON, ...). When you set up the printer — called an `Output` — you can pass options. Three of those options existed purely to help people migrating from version 1 of the library: `WithTableStyle` (which border characters a table uses), `WithTOC` (add a table of contents to Markdown), and `WithFrontMatter` (add a metadata block at the top of Markdown).

The bug: those three options were accepted but then ignored. The `Output` stored the values in fields that nothing ever read. You asked for a table of contents, the library said nothing, and you got no table of contents. This change connects the stored values to the rendering step, so the options now do what their names promise.

### Why It Matters

These options exist for exactly one audience: people converting v1 code to v2. Their converted program compiled and ran without errors but silently dropped formatting they asked for — the worst kind of bug, because there is no signal anything is wrong. Now the same code produces the styled table, the table of contents, and the front matter.

### Key Concepts

- **Renderer**: the object that turns a document into bytes for one format (one for tables, one for Markdown, ...). Think of it as a printing press configured for one kind of page.
- **Option function**: `WithTOC(true)` is a small function that tweaks the `Output` while it is being built — like ticking a box on an order form. The bug was that the box got ticked but nobody in the workshop ever looked at the form.
- **Additive semantics**: these options only ever add settings. `WithTOC(false)` means "I am not asking for anything", not "remove the table of contents someone else asked for".
- **Copy instead of modify**: at render time the library reconfigures a *copy* of the printing press rather than the original, so other users of the same press are unaffected.

---

## Intermediate Level

### Changes Overview

- `v2/output.go`: `Render` now snapshots `tableStyle`, `hasTOC`, `frontMatter` under the existing read lock, then calls a new pure helper `applyV1CompatOptions(formats, tableStyle, hasTOC, frontMatter) []Format` after `validateConfigEntries` and before `renderWithConfig`.
- `v2/table_renderer.go`: new unexported `(*tableRenderer).withStyle(name)` — shallow struct copy with `styleName` overridden.
- `v2/markdown_renderer.go`: new unexported `(*markdownRenderer).withCompatOptions(enableToC, frontMatter)` — field-by-field rebuild that ORs `includeToC` and merges front matter (Output-level keys win).
- `v2/renderer.go`: `Format.Options` documented as caller-owned metadata the render pipeline does not read.
- `v2/output_v1_compat_options_test.go`: ten behavioural tests asserting rendered bytes.
- `v2/docs/MIGRATION.md`, `CHANGELOG.md`, bugfix report: documentation of the wired behaviour and its additive semantics.

### Implementation Approach

`applyV1CompatOptions` copies the formats slice and, per entry, switches on `Format.Name` (`FormatTable` / `FormatMarkdown`) then type-asserts the renderer (`*tableRenderer` / `*markdownRenderer`). Both gates must pass: a custom `Renderer` registered under the name `"markdown"` fails the assertion and is skipped; a built-in renderer under a custom name is also skipped, matching the documented contract ("every configured markdown format"). Zero values short-circuit — if no compat option is set, the original slice is returned untouched, so unconfigured Outputs pay nothing.

The clone logic lives beside the struct definitions rather than in `output.go`. For `tableRenderer` (no locks, three fields) a shallow copy suffices. For `markdownRenderer` the embedded `baseRenderer` carries a `sync.RWMutex`, so copying the struct would copy a lock (`go vet` copylocks); instead the helper reads `config` under `RLock` and builds a fresh struct listing every field explicitly.

### Trade-offs

- **Render-time derivation vs. rewriting `o.formats` in `NewOutput`**: doing it in `Render` keeps option order irrelevant (`WithTableStyle` before `WithFormat` works) and stored state untouched. Cost: the derivation re-runs on every `Render` call — negligible next to actual rendering.
- **Cloning the existing renderer vs. calling public constructors**: `NewTableRendererWithStyle` and friends would discard orthogonal settings (max column width, collapsible config, heading level). Cloning preserves them.
- **Additive-only semantics vs. full override**: `WithTOC(false)` cannot disable a constructor-enabled ToC because `false` is indistinguishable from "option not used" without extra state. Chosen deliberately and documented in godoc, MIGRATION.md, and the CHANGELOG.
- **`Format.Options` documented, not wired**: nothing reads or writes it, there are no key semantics to honour, and wiring it would create a third configuration surface. Removing it would break the API, which the ticket rules out.

---

## Expert Level

### Technical Deep Dive

Ordering matters twice. First, `applyV1CompatOptions` runs after `validateConfigEntries`, which uses reflection-based `isNilValue` to reject typed nils — so the type assertions can assume any `*tableRenderer`/`*markdownRenderer` match is a non-nil pointer, and the helper documents that precondition. Second, within the markdown merge, `maps.Copy(merged, mr.frontMatter)` precedes `maps.Copy(merged, frontMatter)`, which is what makes Output-level keys win.

Concurrency: all renderer fields are construction-only writes (verified by grep and an external `-race` test with 8 concurrent Renders); render paths only read. The `RLock` around `mr.config` is therefore defensive, but consistent with `baseRenderer`'s own access discipline. Clone-while-render is concurrent reads on the original. When only `hasTOC` is set, `merged` aliases `mr.frontMatter` — safe because front matter is only read (sorted iteration) during render. The shallow `tableRenderer` copy shares `collapsibleConfig`'s reference-typed internals (`HTMLCSSClasses`, `DataTransformers`, `ByteTransformers`) with the original; the table render path never mutates them, and `TransformPipeline` guards its own state with a mutex — semantically identical to the already-supported concurrent Render on one renderer instance.

The main maintenance hazard is the field-by-field rebuild: a future `markdownRenderer` field omitted from `withCompatOptions` silently reverts to its zero value in the derived renderer — the exact bug class T-1516 fixes. Mitigation: the helper sits directly under the struct definition with a comment stating the invariant, and `TestCompatOptionsDoNotMutateStoredFormats` plus the combination tests would catch drops of the currently-observable fields.

### Architecture Impact

Two parallel configuration surfaces (Format constructors and Output-level options) now converge on the same renderer state at render time instead of only one being live. The derivation is a pure function over snapshots, so `Render`'s existing concurrency contract is unchanged. The unexported clone helpers establish a per-renderer copy idiom (precedent: `Schema.clone`) that any future compat wiring should follow. `Format.Options` is now explicitly caller-owned metadata; if key semantics are ever defined, that is a deliberate API design task, not a wiring gap.

### Potential Issues

- Unknown style names (`WithTableStyle("bold")`, lowercase) silently fall back to the default style — inherited from `getTableStyle`, now documented on the option, but still a silent path a strict-validation follow-up could tighten.
- A built-in renderer registered under a non-standard format name is skipped by design; users doing that must use the constructors for configuration.
- `headingLevel` preservation is asserted structurally (field copied) but no test observes it through rendered bytes; a dropped field would clamp to 1 in `renderSectionContentMarkdownWithDepth` (`min(max(depth,1),6)`), degrading gracefully.
- The additive model means there is no Output-level way to *disable* a constructor-enabled feature. If that is ever needed, the options would need tri-state values — an API change.

---

## Completeness Assessment

**Fully implemented:**
- All three Output-level options wired into `Render` with additive semantics, order independence, and preservation of orthogonal constructor settings.
- Custom renderers and non-matching formats untouched (tested); stored formats never mutated (tested); typed-nil safety via validation ordering.
- `Format.Options` decision executed as documented (doc comment, no wiring).
- Documentation aligned: godoc, MIGRATION.md note, CHANGELOG entry, bugfix report.

**Partially implemented (accepted gaps):**
- `headingLevel` preservation not observed via rendered bytes (graceful degradation if ever dropped).
- Unknown-style fallback is documented rather than validated.

**Missing:** nothing required by the ticket. No divergences between the report and the code after the review round; the report's validation-ordering wording and test inventory were corrected during this review.
