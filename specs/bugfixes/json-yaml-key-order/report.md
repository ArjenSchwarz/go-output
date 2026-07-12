# Bugfix Report: JSON/YAML Record Objects Do Not Preserve Key Order

**Date:** 2026-07-12
**Status:** Fixed
**Ticket:** T-1520

## Description of the Issue

Key order preservation is the library's flagship feature: "Key ordering is
**never** alphabetized or reordered - it preserves the exact order specified
by users." The JSON renderer violated this for every emitted record object,
and the YAML renderer violated it on all paths except the buffered
single-table path. User-specified order survived only in the out-of-band
`schema.keys` array; the record objects themselves were alphabetized.

**Reproduction steps:**
1. Build a document: `New().Table("t", data, WithKeys("zebra", "alpha", "mike")).Build()`
2. Render with the JSON renderer (or the YAML renderer with two or more contents, a section, or the stream path).
3. Observe each record object serialized as `{"alpha": ..., "mike": ..., "zebra": ...}` — alphabetical, not the requested `zebra, alpha, mike`.

**Impact:** High severity (from code audit) — violates the library's #1
stated requirement. Affects every consumer of JSON output, and YAML output
for multi-content documents, sections, collapsible sections, and the stream
path. Any downstream tool that displays record objects in serialized order
shows the wrong column order.

## Investigation Summary

Focused inspection of `v2/json_yaml_renderer.go` (root cause was already
identified precisely by the code audit in T-1520; investigation confirmed it
and mapped the full set of affected paths).

- **Symptoms examined:** serialized key order of record objects in rendered
  JSON/YAML bytes, across single-content, multi-content, section,
  collapsible-section, and stream render paths.
- **Code inspected:** `renderTableContentJSON`, `renderDocumentGeneric`,
  `renderSectionContentJSON`, `renderCollapsibleSectionJSON`,
  `renderTableContentJSONStream`, `renderTextContentJSONStream`,
  `renderRawContentJSONStream`, `renderSectionContentJSONStream`, and the
  YAML equivalents.
- **Hypotheses tested:** regression tests asserting byte-level key order
  (`v2/json_yaml_key_order_test.go`) confirmed 9 of 10 paths emit
  alphabetized records; only the buffered single-table YAML path (already
  built on `yaml.Node`) preserves order.

Affected paths confirmed by failing tests (all emitted `[alpha mike zebra]`
for requested order `[zebra alpha mike]`):

| Path | JSON | YAML |
|------|------|------|
| Buffered single table | broken | OK (yaml.Node) |
| Multi-content document | broken | broken |
| Section-nested table | broken | broken |
| Collapsible-section-nested table | broken | broken |
| Stream table (`renderContentTo`) | broken | broken |

## Discovered Root Cause

`renderTableContentJSON` builds each record as
`orderedRecord := make(map[string]any)` with the comment "Create ordered map
that preserves key order for JSON marshaling" — but Go maps have no order and
`encoding/json` sorts map keys alphabetically when marshaling. The insertion
order loop is a no-op for output purposes.

**Defect type:** Logic error (incorrect assumption that Go map insertion
order survives marshaling).

**Why it occurred:** the code was written as if inserting keys in order into
a map produced ordered output. The existing test
(`TestJSONRenderer_TableKeyOrderPreservation`) only asserted the
`schema.keys` array order and key *presence* in records — it unmarshaled
records into maps, which cannot detect serialized key order — so the defect
was invisible to the suite.

**Contributing factors:**
- The multi-content path (`renderDocumentGeneric`) compounds the problem:
  correctly rendered content bytes are unmarshaled into `any`
  (map — order destroyed again) and re-marshaled. The same round trip exists
  in `renderSectionContentJSON`/`renderCollapsibleSectionJSON` and their YAML
  equivalents, so even the order-preserving YAML table node was destroyed
  when nested.
- `renderTableContentJSONStream` was ~125 lines of hand-rolled byte-level
  JSON with dead setup (an unused `json.Encoder` and an unused `result` map)
  that still marshaled each record through a map, so it had the same defect.
- The YAML stream path used plain maps rather than the `yaml.Node`
  construction the buffered path already had.

## Resolution for the Issue

**Changes made** (all in `v2/json_yaml_renderer.go`; no public API signature
changes):

- Introduced `orderedJSONObject` (a `[]jsonMember` implementing
  `json.Marshaler`) that emits object members in slice order. Table envelopes
  and record objects are built with it in a new `buildTableContentJSON`,
  shared by the buffered path (`renderTableContentJSON`) and the stream path.
  The JSON table envelope is now `title, schema, data` with `schema.keys`
  before `schema.fields` (previously alphabetical), matching YAML.
- `renderDocumentGeneric`'s unmarshal parameter became a `wrapContentFunc`:
  the multi-content array now embeds already-rendered bytes —
  `json.RawMessage` for JSON, parsed `*yaml.Node` (new `yamlContentNode`
  helper) for YAML — instead of the order-destroying
  unmarshal→re-marshal round trip. The same replacement was applied to
  `renderSectionContentJSON`, `renderCollapsibleSectionJSON`,
  `renderSectionContentYAML`, `renderCollapsibleSectionYAML`, and
  `renderSectionContentYAMLStream`. The JSON section envelope is now ordered
  (`type, title, level, contents`).
- Replaced the ~125-line hand-rolled `renderTableContentJSONStream` (with
  its dead `json.Encoder` and unused `result` map) with a three-line
  `json.Encoder` encode of `buildTableContentJSON`;
  `renderSectionContentJSONStream`'s manual byte emission now reuses the
  buffered section rendering plus a trailing newline.
  `renderTextContentJSONStream`/`renderRawContentJSONStream` (and their YAML
  counterparts) now reuse `buildTextContentData`/`buildRawContentData`
  instead of re-inlining the same maps.
- The buffered YAML table construction moved into
  `buildTableContentYAMLNode`, now shared with
  `renderTableContentYAMLStream` (previously map-based, so alphabetized).

Net effect on `v2/json_yaml_renderer.go`: 334 lines removed against
131 added.

**Approach rationale:** follows the remediation prescribed in the ticket:
order-preserving encoding (custom ordered marshaler for JSON, `yaml.Node`
for YAML — the mechanism the buffered YAML table path already used
correctly), composition from rendered bytes rather than lossy round trips,
and replacement of the fragile hand-rolled stream emitter with the same
ordered construction through a `json.Encoder`. `json.MarshalIndent` and
`json.Encoder` re-indent custom-marshaler output, so the compact bytes from
`orderedJSONObject.MarshalJSON` and embedded `json.RawMessage` values come
out properly indented.

**Alternatives considered:**
- Emitting records as arrays of `[key, value]` pairs - Rejected: changes the
  output shape consumers rely on; record objects must stay objects.
- A third-party ordered-map JSON library - Rejected: a ~25-line
  `json.Marshaler` suffices; not worth a dependency.
- `encoding/json/v2` - Rejected: still experimental behind `GOEXPERIMENT`
  in Go 1.25.
- Leaving nested-section/collapsible paths for a follow-up - Rejected: they
  share the identical defect (round trip through unordered maps), and the
  flagship guarantee would still fail for any table inside a section.

Known remaining limitation (pre-existing, unchanged): the YAML
section/collapsible *envelope* keys (`type`, `title`, `level`, ...) are
still emitted from maps and thus sorted; record data — the subject of the
guarantee — is ordered everywhere.

## Regression Test

**Test file:** `v2/json_yaml_key_order_test.go`
**Test names:** `TestJSONRenderer_RecordObjectsSerializeInKeyOrder`,
`TestJSONRenderer_MultiContentRecordObjectsSerializeInKeyOrder`,
`TestJSONRenderer_SectionTableRecordObjectsSerializeInKeyOrder`,
`TestJSONRenderer_CollapsibleSectionTableRecordObjectsSerializeInKeyOrder`,
`TestJSONRenderer_StreamTableRecordObjectsSerializeInKeyOrder`, and the five
YAML equivalents (`TestYAMLRenderer_*SerializeInKeyOrder`).

**What it verifies:** the keys of each emitted record object/mapping appear
in the emitted bytes in the user-specified order (`zebra, alpha, mike` — a
deliberately non-alphabetical order), across single-content, multi-content,
section, collapsible-section, and stream render paths. Unlike the
pre-existing key-order test, these decode the raw bytes token-by-token
(JSON) or via `yaml.Node` (YAML) so alphabetization cannot hide behind an
unordered unmarshal.

**Run command:** `go test -run 'SerializeInKeyOrder' ./...` (from `v2/`)

## Affected Files

| File | Change |
|------|--------|
| `v2/json_yaml_renderer.go` | Ordered JSON marshaling, round-trip removal, stream emitter replacement, shared YAML node builder |
| `v2/json_yaml_key_order_test.go` | New byte-level key-order regression tests (10 tests) |
| `CHANGELOG.md` | Unreleased → Fixed entry; removed a stray pre-existing merge-conflict marker |
| `specs/bugfixes/json-yaml-key-order/report.md` | This report |

## Verification

**Automated:**
- [x] Regression test passes (`go test -run 'SerializeInKeyOrder' ./...` — 10/10)
- [x] Full test suite passes (`go test ./...` and `INTEGRATION=1 make test-integration`)
- [x] Linters/validators pass (`golangci-lint run` — 0 issues; `gofmt -l` clean; `go vet` clean)

**Manual verification:**
- Rendered a document with a multi-table body and a section-nested table
  through the public `Output.Render` API to both JSON and YAML with keys
  `zebra, alpha, mike`: every record object/mapping serializes in exactly
  that order on every path.

## Prevention

**Recommendations to avoid similar bugs:**
- Never rely on Go map insertion order for serialized output; if order
  matters, marshal through an order-aware structure.
- When asserting output ordering in tests, decode the raw bytes
  (token streams / node trees), not unmarshal-into-map — maps erase the very
  property under test.
- Avoid render→unmarshal→re-marshal round trips when composing documents;
  embed already-rendered bytes (`json.RawMessage`, parsed `yaml.Node`).

## Related

- Transit ticket T-1520 (code audit finding, architecture/high)
- v2/CLAUDE.md "Key Order Preservation System" — the documented contract
