# Decision Log: draw.io CSV Reader

## Decision 1: Use the full spec workflow

**Date**: 2026-06-11
**Status**: accepted

### Context

T-1539 asks for a v2 equivalent of v1's `GetHeaderAndContentsFromFile`. Scope assessment estimated 200-300 LOC across 3+ files, with open API-shape questions and round-trip edge cases (unescaped connect JSON, zero-value vs absent directives).

### Decision

Run the full spec workflow (requirements, design, tasks) rather than a smolspec.

### Rationale

The estimate exceeds the 80 LOC smolspec threshold by a wide margin, and the round-trip fidelity questions warrant explicit design decisions.

### Alternatives Considered

- **Smolspec**: Lightweight single document - Rejected because the implementation size and design questions exceed smolspec criteria.

### Consequences

**Positive:**
- Round-trip edge cases get explicit requirements and design coverage.

**Negative:**
- More process overhead for a moderately sized feature.

---

## Decision 2: API returns header and records, not a content object

**Date**: 2026-06-11
**Status**: superseded by Decision 6

### Context

The parser could return `(DrawIOHeader, []Record, error)`, a `*DrawIOContent`, or both. The driving use case (awstools `tgw overview --append`) reads an existing file, merges in new records, and re-renders.

### Decision

The public API returns the parsed `DrawIOHeader` and `[]Record` directly (user-approved choice).

### Rationale

The merge workflow needs the raw header and records anyway; a `*DrawIOContent` wrapper would immediately be unpacked via `GetHeader()`/`GetRecords()`. Callers that want a content object can call `NewDrawIOContent` with the returned values.

### Alternatives Considered

- ***DrawIOContent return**: Ready-made content object - Rejected because the merge flow still has to unpack it, adding a step without value.
- **Both forms**: Core function plus convenience constructor - Rejected to keep the API surface minimal; `NewDrawIOContent` already serves as the constructor.

### Consequences

**Positive:**
- Direct fit for the read-merge-rewrite workflow.
- Smaller API surface.

**Negative:**
- Callers wanting a `*DrawIOContent` must compose two calls.

---

## Decision 3: Tolerant parsing of unknown directives

**Date**: 2026-06-11
**Status**: accepted

### Context

draw.io supports more header directives than `DrawIOHeader` represents (`# stylename:`, `# vars:`, `# labels:`, etc.), and files may be hand-edited or written by v1.

### Decision

The parser guarantees round-trip fidelity only for v2 renderer output; unrecognized but well-formed directives are silently ignored (user-approved choice).

### Rationale

Tolerance keeps hand-edited and v1-written files usable on a best-effort basis without expanding `DrawIOHeader` to the full draw.io directive set, which the driving use case does not need.

### Alternatives Considered

- **Strict v2-only parsing**: Error on unknown directives - Rejected because it breaks on hand-edited and v1-written files for no benefit.
- **Full draw.io directive set**: Parse everything drawio.com supports - Rejected as scope expansion requiring `DrawIOHeader` struct changes without a current need.

### Consequences

**Positive:**
- Robust against hand-edits and forward-compatible with new directives.

**Negative:**
- Ignored directives are dropped on re-render; a hand-added `# vars:` line would be lost in the append cycle.

---

## Decision 4: Error on malformed input

**Date**: 2026-06-11
**Status**: accepted

### Context

Malformed connect JSON, bad numeric directive values, or inconsistent CSV rows could be skipped (best-effort) or reported as errors. In the append workflow, silently dropped rows mean silent diagram data loss on the next write.

### Decision

Return errors with line/directive context for malformed connect JSON, non-integer numeric directives, malformed CSV, and missing column header row (user-approved choice). Unlike v1, never panic or `log.Fatal`.

### Rationale

The append workflow rewrites the file from parsed data; anything unparseable that is skipped would be permanently lost. Failing loudly protects the data.

### Alternatives Considered

- **Best-effort parsing**: Skip unparseable parts - Rejected because silent data loss in a read-modify-write cycle is destructive.

### Consequences

**Positive:**
- No silent data loss; v2 error-handling conventions (returned errors) are followed.

**Negative:**
- A corrupted line blocks the whole file until fixed manually.

---

## Decision 5: Fix the renderer's connect JSON escaping as part of this feature

**Date**: 2026-06-11
**Status**: accepted

### Context

`writeDrawIOHeader` builds `# connect:` JSON with `fmt.Sprintf` and no string escaping. A connection style or label containing a double quote produces invalid JSON, which the new parser (and draw.io itself) cannot read, breaking round-trip requirement [3.1](requirements.md#3.1).

### Decision

Include a writer-side fix so `# connect:` values are always valid JSON (requirement [3.5](requirements.md#3.5)).

### Rationale

Round-trip stability is the core promise of this feature; it cannot hold if the writer can emit unparseable output. The fix is small and benefits draw.io import correctness independently of the parser.

### Alternatives Considered

- **Document as known limitation**: Leave writer untouched - Rejected because quotes in styles/labels are plausible and would corrupt the append cycle.
- **Lenient JSON-ish parsing in the reader**: Accept the broken output - Rejected as fragile and divergent from what draw.io itself accepts.

### Consequences

**Positive:**
- Round-trip holds for all representable header values; draw.io import compatibility improves.

**Negative:**
- Touches existing renderer code, so existing renderer tests may need byte-level expectations updated if formatting changes.

---

## Decision 6: Parser returns a result struct with explicit column order

**Date**: 2026-06-11
**Status**: accepted

### Context

Both the design-critic and peer-review-validator reviews found that the Decision 2 shape `(DrawIOHeader, []Record, error)` cannot carry column names: records are maps, so column order is lost, an empty-data file loses its column header row entirely, and the writer's `extractColumnNames` alphabetizes columns on every render. Round-trip requirement [3.1](requirements.md#3.1) is unachievable with that shape.

### Decision

The parser returns `(*ParsedDrawIO, error)` where `ParsedDrawIO` holds `Header DrawIOHeader`, `Columns []string` (file order), and `Records []Record` (file order). The writer gains support for an explicit column order that takes precedence over alphabetized auto-detection (user-approved choice, revising Decision 2).

### Rationale

Explicit columns are the only way to preserve order and round-trip empty-data files. A result struct keeps the signature narrow and extensible, and column-order preservation matches the library's headline key-order-preservation principle.

### Alternatives Considered

- **Four return values** `(DrawIOHeader, []string, []Record, error)`: Same information - Rejected as an awkwardly wide Go signature.
- **Keep 2-value shape with carve-outs**: Document the losses - Rejected because it permanently breaks round-trip for non-alphabetical files.

### Consequences

**Positive:**
- Round-trip holds including empty-data files and non-alphabetical column orders.

**Negative:**
- Requires a writer-side change (`DrawIOContent` explicit column order) in addition to the parser.

---

## Decision 7: Round-trip invariant is idempotency, not universal byte-identity

**Date**: 2026-06-11
**Status**: accepted

### Context

The peer review showed universal byte-identity is falsifiable: `encoding/csv` normalizes `\r\n` to `\n` inside quoted fields, and `fmt.Sprint` value formatting makes the first render of programmatically built content unstable. The append workflow only needs stability from the first re-render onward.

### Decision

Requirement [3.1](requirements.md#3.1) demands parse→render idempotency (`render(parse(f'))` == `f'`); byte-identity with the original file ([3.2](requirements.md#3.2)) is guaranteed only for renderer-produced files free of carriage returns in field values.

### Rationale

Idempotency is exactly what the read-merge-rewrite cycle needs and is achievable; absolute byte-identity is hostage to CSV normalization rules outside our control.

### Alternatives Considered

- **Universal byte-identity**: Stronger claim - Rejected because the first property-based test would falsify it (CRLF normalization).
- **No formal invariant**: Just "round-trips" informally - Rejected as untestable.

### Consequences

**Positive:**
- A precise, testable property that holds for all inputs.

**Negative:**
- Files containing CRLF in field values are normalized on the first rewrite.

---

## Decision 8: Comments recognized only before the column header row; writer quotes leading-# fields

**Date**: 2026-06-11
**Status**: accepted

### Context

The writer only quotes fields containing `,`, `"`, or newlines, so a data value starting with `#` is written unquoted and a CSV reader in comment mode (v1's approach) silently drops the row — silent data loss in the append cycle. Separately, rejecting trailing directive lines ([4.5](requirements.md#4.5)) requires recognizing directive patterns after the data section, which conflicts with treating every `#` line as data.

### Decision

The parser does not use CSV comment mode: it splits the directive block from the data section in a pre-pass, so leading-`#` data survives. After the column header row, only unquoted lines matching a recognized directive pattern are rejected (multi-block detection); all other `#` lines are data. The writer additionally quotes fields beginning with `#` so its own output is never ambiguous.

### Rationale

Reader-side handling fixes the data-loss hazard for existing files; writer-side quoting removes the residual ambiguity (a data value that happens to look like a directive) from all future output. The writer also quotes a row consisting of a single empty field, since an unquoted one renders as a blank line that CSV readers silently skip. Together these make requirement [3.2](requirements.md#3.2) hold for output of the modified renderer.

### Alternatives Considered

- **CSV comment mode (`Comment='#'`)**: v1's approach - Rejected because it silently drops data rows starting with `#`.
- **Writer-side quoting only**: Leaves existing files exposed - Rejected as incomplete; old files still parse wrong.
- **Reader-side only**: Validator's preference - Rejected because multi-block detection (Decision 9) needs directive-pattern recognition after the header row, and unquoted directive-lookalike data would then false-positive; quoting at the writer eliminates that for self-produced files.

### Consequences

**Positive:**
- No silent row loss; multi-block files are detectable.

**Negative:**
- A hand-written unquoted data row that exactly matches a directive pattern (e.g. first cell `# label: x`) errors instead of parsing as data. This fails loudly, not silently, and cannot occur in renderer-produced files.

---

## Decision 9: Reject trailing directive lines (multi-diagram files) with an error

**Date**: 2026-06-11
**Status**: accepted

### Context

A v2 document with multiple `DrawIOContent`s renders as concatenated header + table blocks. A parser that silently read only the first block would, in the append cycle, rewrite the file and delete the other diagrams.

### Decision

Directive-pattern lines after the column header row produce an error distinguishable from malformed-CSV errors ([4.5](requirements.md#4.5)). Multi-diagram parsing stays out of scope.

### Rationale

"Out of scope" must not mean "undefined": the failure mode is silent destruction of user data, which Decision 4 already rules out.

### Alternatives Considered

- **Parse first block, ignore rest**: Simple - Rejected as silent data loss.
- **Support multi-block parsing**: Return multiple results - Rejected as scope expansion without a current need.

### Consequences

**Positive:**
- The append workflow cannot silently destroy multi-diagram files whose later blocks emit directive lines (which every non-zero-value header does).

**Negative:**
- Legitimate multi-diagram files cannot be read at all until such support is added.
- A second block with a fully zero-value header emits no directives and is not detectable; this residual hole is documented in the requirements' Out of Scope section. Closing it would require the writer to always emit a directive, breaking byte compatibility with existing files.

---

## Decision 10: Pin the connect-JSON wire format

**Date**: 2026-06-11
**Status**: accepted

### Context

`DrawIOConnection` has no json struct tags. A naive `json.Marshal` in the Decision 5 writer fix would emit capitalized keys (`From`, `To`, ...), changing output bytes and breaking draw.io import, while `json.Unmarshal` would silently accept either casing.

### Decision

The wire format is pinned ([3.5](requirements.md#3.5)): keys `from`, `to`, `invert`, `label`, `style` in that order, all always present (`invert` never omitted), JSON string escaping applied. The parser unmarshals into the same tagged struct and ignores unknown keys ([2.2](requirements.md#2.2)).

### Rationale

The format must match what existing files contain and what draw.io accepts; leaving it to the implementation invites accidental byte and compatibility breaks.

### Alternatives Considered

- **Leave format to implementation**: Less spec text - Rejected; the default `json.Marshal` output would be wrong.
- **Keep hand-rolled Sprintf with manual escaping**: Minimal diff - Viable but rejected in favor of tagged-struct marshaling, which cannot miss an escape case.

### Consequences

**Positive:**
- Existing files, new output, and draw.io all agree on one format.

**Negative:**
- The pinned format must be implemented carefully; see Decision 14 for why json tags on the public struct were rejected.

---

## Decision 11: Absent directives map to zero values, not DefaultDrawIOHeader defaults

**Date**: 2026-06-11
**Status**: accepted

### Context

`DefaultDrawIOHeader()` sets Layout=auto, NodeSpacing=40, etc. A reader could seed those defaults for directives missing from the file. The writer also only emits numeric directives when > 0, so "absent" and "explicit 0" are indistinguishable in `DrawIOHeader`'s int fields.

### Decision

Absent directives yield the field's zero value ([2.3](requirements.md#2.3)). The absent-vs-zero ambiguity for numeric fields is accepted: draw.io treats 0 and absent identically, and re-rendering suppresses both.

### Rationale

Seeding defaults would make re-rendered output differ from the parsed file (e.g. a file without `# layout:` would gain one), breaking idempotency. Zero values reproduce exactly what was read.

### Alternatives Considered

- **Seed DefaultDrawIOHeader values**: Matches constructor expectations - Rejected because it breaks round-trip idempotency for files lacking those directives.
- **Pointer/sentinel fields for absent-vs-zero**: Precise - Rejected as an API change to `DrawIOHeader` without a practical need.

### Consequences

**Positive:**
- Round-trip idempotency holds; parser output reflects the file, nothing more.

**Negative:**
- Callers building new content from parsed headers of sparse files get zero values where they might expect draw.io defaults.

---

## Decision 12: Column order passed via variadic functional options

**Date**: 2026-06-12
**Status**: accepted

### Context

The writer needs an explicit column order on `DrawIOContent` (Decision 6). The order must flow through both `NewDrawIOContent` and `Builder.DrawIO` without breaking existing callers.

### Decision

Add `opts ...DrawIOOption` to `NewDrawIOContent` and `Builder.DrawIO`, with `WithDrawIOColumns(columns ...string)` as the first option (user-approved choice).

### Rationale

Variadic options are backward compatible (no existing caller passes options) and match the library's established functional-options idiom (`WithKeys`, `WithSchema`).

### Alternatives Considered

- **Separate constructors** (`NewDrawIOContentWithColumns`, `Builder.DrawIOWithColumns`): No signature changes - Rejected because it doubles the API surface for one parameter.

### Consequences

**Positive:**
- Zero migration for existing callers; extensible for future options.

**Negative:**
- One more option type in the public API.

---

## Decision 13: NewDrawIOContentFromTable preserves schema column order

**Date**: 2026-06-12
**Status**: accepted

### Context

`NewDrawIOContentFromTable` copies a table's records but not its schema order, so the renderer alphabetizes columns — at odds with the library's key-order-preservation principle. The new columns field makes preserving order trivial.

### Decision

`NewDrawIOContentFromTable` sets the columns field from `table.schema.GetFieldNames()` (user-approved choice).

### Rationale

Consistency with the library's headline guarantee; one line of code now that the field exists.

### Alternatives Considered

- **Leave alphabetized**: No behavior change for existing users - Rejected; the inconsistency is a latent bug, and the change only affects column order in draw.io CSV output, not data.

### Consequences

**Positive:**
- Table-sourced draw.io output respects user-specified key order.

**Negative:**
- Existing users of this constructor see column order change from alphabetical to schema order (behavior change, though aligned with documented library principles).

---

## Decision 14: Connect JSON via a private mirror struct, not json tags on DrawIOConnection

**Date**: 2026-06-12
**Status**: accepted

### Context

The design review found that tagging the public `DrawIOConnection` struct would change more than the CSV wire format: `renderDrawIOContentJSON` (json_yaml_renderer.go) marshals the full `DrawIOHeader`, so JSON-format document output would flip connection keys from `From/To/Invert/Label/Style` to lowercase — a visible behavior change in an unrelated renderer.

### Decision

`DrawIOConnection` stays untagged. The drawio CSV writer and parser marshal/unmarshal through a private `drawioConnectionJSON` mirror struct carrying the json tags.

### Rationale

Decision 10 pins the CSV wire format only; it does not require tags on the public type. The mirror struct confines the format to the one place it applies and leaves every other renderer's output untouched.

### Alternatives Considered

- **Tag the public struct**: Simpler, makes JSON renderer output consistent with YAML's lowercase keys - Rejected because silently changing JSON-format output is a side effect outside this feature's scope.

### Consequences

**Positive:**
- No behavior change outside the drawio CSV format; a JSON-renderer regression test guards this.

**Negative:**
- A five-field struct is duplicated privately and must be kept in sync with `DrawIOConnection`.

---
