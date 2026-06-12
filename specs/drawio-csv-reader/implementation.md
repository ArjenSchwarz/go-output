# Implementation Explanation: draw.io CSV Reader

Three-level explanation of the drawio-csv-reader feature as implemented on branch
`T-1539/drawio-csv-reader` (spec: this directory). Generated as part of the
pre-push review.

## Beginner Level

### What Changed / What This Does

The go-output library could already *write* draw.io CSV files — a special text
format that the diagrams.net (draw.io) application can import to draw
architecture diagrams. A draw.io CSV file has two parts: a block of
configuration lines at the top that all start with `#` (called directives, e.g.
`# label: %Name%` tells draw.io what text to put on each box), followed by a
normal CSV table whose rows become the boxes in the diagram.

What the library could *not* do was read such a file back in. This change adds
that: `ParseDrawIOFile("diagram.csv")` (or `ParseDrawIOCSV` for any input
stream) reads a file and gives you back three things — the configuration
(header), the column names in their original order, and the data rows.

A few small changes were also made to the *writing* side so that a file you
read in and write out again comes back exactly the same, byte for byte.

### Why It Matters

Tools built on this library (like awstools) want to merge new data into an
existing diagram file: read the file, add rows, write it back. Without a
parser, that workflow was impossible — you could only ever generate a diagram
from scratch. With it, you can append to a diagram across multiple runs (for
example, collecting AWS network data from several accounts into one picture).

The "comes back exactly the same" property matters because it means appending
rows never silently corrupts or reorders the rest of the file.

### Key Concepts

- **CSV (comma-separated values)**: a plain-text table; each line is a row,
  values separated by commas. Values containing commas or quotes are wrapped in
  double quotes, like `"Production VPC, primary"`.
- **Directive**: a configuration line at the top of the file, format
  `# key: value`. Think of it as the diagram's settings panel saved as text.
- **Round-trip**: read a file in and write it back out. A *lossless* round-trip
  reproduces the original exactly — like photocopying a photocopy and getting
  an identical page.
- **Sentinel error**: a named, exported error value (like
  `ErrDrawIODuplicateColumn`) that calling code can check for with
  `errors.Is`, instead of having to match error message text.

## Intermediate Level

### Changes Overview

- **`v2/drawio_reader.go` (new)**: `ParseDrawIOCSV(io.Reader)` /
  `ParseDrawIOFile(path)` returning `*ParsedDrawIO{Header, Columns, Records}`;
  four sentinel errors; the 18 directive-key constants; the private
  `drawioConnectionJSON` mirror struct shared with the writer; the
  `writeDrawIODirective` grammar helper.
- **`v2/graph_content.go`**: `DrawIOContent` gains a `columns []string` field,
  a `DrawIOOption` functional-options type with `WithDrawIOColumns(...)`, a
  `GetColumns()` defensive-copy accessor, and column cloning in `Clone()`.
  `NewDrawIOContentFromTable` now captures the table schema's field order.
- **`v2/graph_renderers.go`**: the renderer prefers the explicit column order
  over alphabetized auto-detection; `# connect:` lines are emitted with
  `json.Encoder` (`SetEscapeHTML(false)`) instead of `fmt.Sprintf`; CSV fields
  starting with `#` and single-empty-field rows are now quoted.
- **`v2/document.go`**: `Builder.DrawIO` gains variadic `opts ...DrawIOOption`
  (backward compatible).
- **Tests**: ~77 parser table-test cases, exact-output writer tests, four
  property-based tests (`pgregory.net/rapid`, test-only dependency), and a
  golden awstools-style round-trip file.

### Implementation Approach

The parser is a staged pipeline over the whole input:

1. **BOM strip** — tolerate a UTF-8 byte order mark.
2. **Directive pre-pass** — consume `#`-prefixed and blank lines until the
   first data line (the column header row). Directives use an exact
   case-sensitive `# key: ` grammar with verbatim untrimmed values; unknown
   keys and free-form comments are ignored; scalar keys are last-wins while
   `# connect:` appends; numeric values are validated on every occurrence.
3. **Quote-parity scan** — before CSV parsing, scan the data section for
   directive lines appearing *after* the header row (a second diagram block),
   tracking quote parity so that lines inside multi-line quoted fields are
   never misclassified. Running this before `csv.Reader` makes the
   trailing-directive error take precedence over field-count errors.
4. **`csv.Reader`** — with `Comment` mode *off* (so data rows starting with
   `#` survive; comment handling already happened), `FieldsPerRecord`
   enforcement, and `TrimLeadingSpace` false (values round-trip verbatim).
5. **Materialization** — duplicate-column check, then one `Record` per row
   with every column present (`""` for empty cells).

On the writer side, three changes close the round-trip gaps: explicit column
order (parsed files re-render in file order instead of alphabetically),
connect JSON pinned to exact key order and escaping via a shared mirror
struct, and quoting rules that prevent data from being mistaken for comments
or blank lines.

### Trade-offs

- **Sentinel errors over a structured error type**: callers only need
  `errors.Is` discrimination; nothing reads context fields programmatically,
  so sentinels wrapped with `fmt.Errorf("%w: ...")` are the simplest adequate
  shape (Decision in design.md Error Handling).
- **Private mirror struct over tagging `DrawIOConnection`**: adding json tags
  to the public struct would have changed the JSON/YAML document renderers'
  output (key casing). The mirror confines the wire format to the CSV
  writer/parser (Decision 14), guarded by a key-casing regression test.
- **Whole-input read over streaming**: the quote-parity pre-scan requires the
  data section before CSV parsing. Draw.io CSV files are small (KBs), so
  simplicity wins.
- **Tolerant directive parsing over strict**: unknown directives are ignored
  rather than errors, so files from newer draw.io versions or hand edits
  still parse (requirement 2.4); malformed values of *recognized* keys fail
  loudly (4.1/4.2).

## Expert Level

### Technical Deep Dive

The subtle correctness work is concentrated in three places:

- **Quote-parity scan semantics.** Quoted CSV fields may contain newlines, so
  "is this physical line a record boundary?" requires tracking whether the
  line starts inside an open quoted field. The scan counts quote characters
  per line (escaped `""` contributes even parity, so it cannot flip state) and
  seeds its state at the *header* line, making multi-line quoted column names
  safe. The scan's parity model and encoding/csv's quote model (quotes special
  only at field start) can disagree on malformed input; the design accepts
  that the *choice* of error shifts — both paths fail loudly — and
  `TestParseDrawIOCSV_ParityVsCSVQuoteDisagreement` pins which one fires.
- **Byte-identity of connect JSON.** The old `fmt.Sprintf` emission could not
  escape quotes/backslashes (producing invalid JSON), but its output for
  escape-free values is the compatibility target. `json.Encoder` with
  `SetEscapeHTML(false)` reproduces it byte-for-byte: field order follows the
  mirror struct declaration, `invert` carries no `omitempty` (the
  `"invert":false` token is load-bearing for requirement 3.5), and `Encode`'s
  trailing newline is the line terminator. Known deviation: invalid UTF-8 in
  values becomes U+FFFD where Sprintf passed it through (documented in
  design.md).
- **CRLF normalization.** The pre-pass line splitter strips `\r` only before
  `\n`, matching encoding/csv, so directives and data normalize identically.
  Consequence: two column names differing only by `\r` vs nothing collide
  after normalization — the round-trip property generators key uniqueness on
  the normalized name. `csv.ParseError` line numbers are shifted from
  data-section to whole-file coordinates before being returned.

The round-trip guarantees are property-tested (rapid, 3 properties + a
no-panic fuzz with a directive-biased generator reaching the
`json.Unmarshal`/`strconv.Atoi` paths) with an example-based golden file as an
anchor against generator blind spots. Idempotency (3.1) holds universally;
byte-identity (3.2) holds for CR-free renderer-produced files, with two
documented carve-outs (CR values; leading U+FEFF column name in a
directive-less file, which BOM tolerance consumes).

### Architecture Impact

- The parser lives in package `output` (not a subpackage) because it needs
  `DrawIOHeader`/`Record`; a subpackage would invert the dependency direction.
- The directive grammar now has a single definition: the `drawioKey*`
  constants and `writeDrawIODirective` are shared by `writeDrawIOHeader` and
  `matchDrawIODirective`, so adding a directive cannot silently desynchronize
  writer and parser (and the byte-identity property would catch any residual
  drift).
- `NewDrawIOContentFromTable` capturing schema order is a deliberate behavior
  change (Decision 13): table-sourced draw.io content now renders in schema
  order rather than alphabetized — aligned with the library's core key-order-
  preservation principle.
- Sentinel errors introduce the `var Err...` convention to v2, which
  otherwise uses structured error types; acceptable because parser callers
  need kind discrimination, not field access.

### Potential Issues

- **Pre-feature files**: output written by the *old* renderer may contain
  unquoted leading-`#` data fields or bare blank lines for single-empty-cell
  rows; these mis-parse (consumed as comment / skipped as blank). Documented
  as out of scope — old files are best-effort.
- **Multiple diagram blocks** are detected only if the second block emits at
  least one directive line; a second block with a fully zero-value header is
  indistinguishable from data and will be parsed as rows (out of scope, by
  design).
- **`extractColumnNames` fallback alphabetizes**: content without explicit
  columns still renders alphabetized (pre-existing behavior, kept for
  backward compatibility). Callers wanting order stability must pass
  `WithDrawIOColumns`.
- **`renderTableAsDrawIO` duplication**: the table-content render arm still
  re-implements row emission; it could now delegate through
  `NewDrawIOContentFromTable` + `renderDrawIOContent`. Left for a future
  cleanup (noted in the pre-push review).

## Completeness Assessment

**Fully implemented**: all 24 acceptance criteria across requirements
sections 1–4, each with a behavioral test (verified requirement-by-requirement
in the pre-push review); all 10 spec tasks complete; all 3 writer changes from
the design's parity audit; the four sentinel errors; property-based round-trip
verification with golden anchor; public API documented in v2/docs/API.md and
the v1 migration mapping in v2/docs/MIGRATION.md.

**Partially implemented**: none.

**Missing / deferred (deliberate)**: directives `DrawIOHeader` cannot
represent (`# stylename:`, `# vars:`, ...) are ignored, not parsed;
multi-block files are rejected, not supported; v1 `drawio` package untouched.
All recorded in the spec's Out of Scope list.

**Divergences discovered during explanation**: none beyond the documented
carve-outs; the U+FEFF byte-identity edge was promoted from agent notes into
the spec's Out of Scope list during the pre-push review.
