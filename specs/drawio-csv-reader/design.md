# Design: draw.io CSV Reader

## Overview

Add `ParseDrawIOCSV`/`ParseDrawIOFile` to the v2 `output` package, parsing draw.io CSV (as written by `drawioRenderer`) into header directives, column order, and records. Three small writer changes make round-trip idempotency ([3.1](requirements.md#3.1)) and re-render byte-identity ([3.2](requirements.md#3.2)) hold: explicit column order, pinned connect JSON, and two extra quoting rules.

## Architecture

New file `v2/drawio_reader.go` (tests in `v2/drawio_reader_test.go`), package `output` — the parser needs `DrawIOHeader`/`Record`, so a subpackage would invert the dependency direction for no gain.

Writer changes are confined to `v2/graph_renderers.go` (`renderDrawIOContent`, `writeDrawIOHeader`, `writeCSVRow`) and `v2/graph_content.go` (`DrawIOContent` columns field, `DrawIOConnection` json tags, constructors). Builder integration is the existing `Builder.DrawIO` call site at `v2/document.go:232`.

### Parsing pipeline

```
input bytes
  → strip optional UTF-8 BOM                                  [1.6]
  → line scan: consume directive/comment/blank lines until
    first non-comment line = column header row                [1.5, 2.x, 4.4]
  → quote-parity scan of remaining data section: at each
    logical record boundary, an unquoted line matching
    "# <recognized-key>: " → trailing-directive error         [4.5]
  → csv.Reader (no Comment mode, FieldsPerRecord enforced)
    over column header row + data section                     [1.5, 3.4, 4.3]
  → duplicate-column check, record materialization (""-fill)  [1.3, 4.6]
```

Non-obvious points:

- **No `Comment='#'` on `csv.Reader`** (Decision 8): comment stripping happens only in the pre-pass, so data rows starting with `#` survive ([1.5](requirements.md#1.5)).
- **Quote-parity scan**: quoted CSV fields may contain newlines, so the trailing-directive check cannot naively split on `\n`. The scan tracks whether each physical line starts inside an open quoted field; only lines at a record boundary are tested against the directive pattern. This pass runs before `csv.Reader` so [4.5](requirements.md#4.5) takes precedence over field-count errors on the same line.
- **Directive grammar** ([2.1](requirements.md#2.1)): exact prefix `# ` + key + `: `, key case-sensitive, value = rest of line verbatim (no trimming). Lines not matching this with a recognized key are ignored in the pre-pass ([2.4](requirements.md#2.4)).
- **Line splitting in the pre-pass**: `\r\n` is stripped, a lone `\r` is kept — matching `encoding/csv` semantics so directives and data normalize identically on CRLF files.
- **Multi-line column header row**: a quoted column name may contain `\n`, so the header *record* can span physical lines; the quote-parity scan seeds its state at the start of the header line, not the line after it, so header continuation lines are never tested as record boundaries.
- **Whole-section scan precedence (deliberate)**: the parity scan runs over the entire data section before `csv.Reader`, so a trailing directive anywhere is reported before any field-count error — stronger than the per-line precedence [4.5](requirements.md#4.5) requires. The scan's quote model (parity counting) and csv's (quotes special only at field start) can disagree on malformed hand-written input, shifting *which* error is reported; both paths fail loudly.
- **Blank lines**: `encoding/csv` skips them unconditionally; this is the documented [1.6](requirements.md#1.6) behavior, made loss-free for new files by the writer's single-empty-field quoting rule.
- **`csv.Reader` defaults stay as-is**: `TrimLeadingSpace` remains false (values round-trip verbatim, [2.1](requirements.md#2.1) no-trimming applies to data too); only `FieldsPerRecord` enforcement is relied upon.
- **Known best-effort mis-parse**: a pre-feature file whose column header row starts with an unquoted `#` (column named `#x`) is consumed by the pre-pass as a comment, promoting the first data row to column header. Accepted: the modified writer quotes such fields ([3.6](requirements.md#3.6)), and old files are best-effort per Out of Scope.

### Directive mapping

Recognized keys map 1:1 onto `DrawIOHeader` fields (the 18 keys from [2.1](requirements.md#2.1)). String directives assign verbatim; `nodespacing`/`levelspacing`/`edgespacing`/`padding` go through `strconv.Atoi` (error → [4.2](requirements.md#4.2), checked on every occurrence even when superseded — intentional asymmetry with last-wins [2.5](requirements.md#2.5)); each `# connect:` value is unmarshaled via the private mirror struct (unknown keys ignored by default unmarshal semantics per [2.2](requirements.md#2.2); any unmarshal failure → [4.1](requirements.md#4.1)). Last-wins applies to scalar directives only: `connect` is append-all, exempt from [2.5](requirements.md#2.5), with every occurrence validated. No `DefaultDrawIOHeader()` seeding (Decision 11).

### Writer changes

| Change | Where | Behavior |
|---|---|---|
| Explicit column order | `DrawIOContent.columns` (new field); `renderDrawIOContent` uses it when non-empty, falls back to `extractColumnNames` | Parsed files re-render in file order incl. zero-record column header row ([3.3](requirements.md#3.3)) |
| Connect JSON | `writeDrawIOHeader`: replace Sprintf with `json.Encoder` + `SetEscapeHTML(false)` encoding a private mirror struct | Valid JSON, keys `from,to,invert,label,style`, `invert` always present, `&<>` verbatim ([3.5](requirements.md#3.5)). Byte-identical to current output for values without `"`/`\`/control chars/invalid UTF-8 (json replaces invalid sequences with U+FFFD; Sprintf passed them through) |
| Quoting rules | `writeCSVRow`: also quote when field starts with `#`, or row is a single empty field | No comment-mistakable or blank-line-mistakable data rows ([3.6](requirements.md#3.6)) |

`json.Encoder.Encode` appends `\n`, which serves as the directive line terminator (`# connect: ` prefix written first).

### Parity audit (column-order pattern extension)

| Call site | Needs explicit columns? | Rationale |
|---|---|---|
| `renderDrawIOContent` (graph_renderers.go:639) | Yes | Core change; only caller of `extractColumnNames` |
| `renderTableAsDrawIO` (graph_renderers.go:698) | No | Already uses `table.schema.GetFieldNames()` order |
| `renderGraphAsDrawIO` (graph_renderers.go:665) | No | Fixed 4-column layout |
| `NewDrawIOContent` (graph_content.go:436) | Yes | Gains `opts ...DrawIOOption`; `WithDrawIOColumns` sets the field |
| `NewDrawIOContentFromTable` (graph_content.go:448) | Yes | Sets columns from `table.schema.GetFieldNames()` (user-approved; fixes alphabetization for table-sourced content). Nil-table path leaves columns nil |
| `Builder.DrawIO` (document.go:232) | Yes | Gains `opts ...DrawIOOption`, forwards to `NewDrawIOContent` |
| `DrawIOContent.Clone` (graph_content.go:518) | Yes | Must `slices.Clone` the columns slice |
| `renderDrawIOContentJSON`/`...YAML` (json_yaml_renderer.go) | No (deliberate) | They marshal `GetHeader()`/`GetRecords()` only; columns stay a CSV-format concern. The private mirror struct (below) keeps their `DrawIOConnection` key casing unchanged |

Both variadic additions are backward compatible (no existing caller passes options).

## Components and Interfaces

```go
// drawio_reader.go

// ParsedDrawIO holds the result of parsing a draw.io CSV file.
type ParsedDrawIO struct {
    Header  DrawIOHeader // zero-valued fields for absent directives
    Columns []string     // column header row, file order
    Records []Record     // file order; every column present, "" for empty cells
}

func ParseDrawIOCSV(r io.Reader) (*ParsedDrawIO, error)
func ParseDrawIOFile(path string) (*ParsedDrawIO, error) // os.Open + ParseDrawIOCSV, wrapped errors

// graph_content.go additions
type DrawIOOption func(*DrawIOContent)
func WithDrawIOColumns(columns ...string) DrawIOOption
func NewDrawIOContent(title string, records []Record, header DrawIOHeader, opts ...DrawIOOption) *DrawIOContent
func (d *DrawIOContent) GetColumns() []string // copy, may be nil

// document.go
func (b *Builder) DrawIO(title string, records []Record, header DrawIOHeader, opts ...DrawIOOption) *Builder
```

Behavioral contracts:

- `ParseDrawIOCSV` reads the entire input; `Records` values are always `string` (typed as `any` via `Record`).
- Round-trip contract: for `f' = render(parse(f))`, `render(parse(f')) == f'` byte-for-byte; `f' == f` when `f` was produced by the modified renderer and contains no CR in field values.
- `WithDrawIOColumns` does not validate against record keys; keys absent from columns are simply not rendered (matches existing renderer behavior of skipping unknown keys), keys missing in a record render as `""`.
- Parser and renderer remain stateless; no locking added.

## Data Models

`DrawIOConnection` stays untagged: adding json tags would change `renderDrawIOContentJSON` output (json_yaml_renderer.go marshals the header, so connection keys would flip from `From/To/...` to lowercase in JSON-format documents). The CSV wire format (Decision 10) is pinned instead via a private mirror struct used by both the writer (marshal) and parser (unmarshal):

```go
// drawio_reader.go (unexported)
type drawioConnectionJSON struct {
    From   string `json:"from"`
    To     string `json:"to"`
    Invert bool   `json:"invert"` // no omitempty: "invert":false is load-bearing for byte identity (3.5)
    Label  string `json:"label"`
    Style  string `json:"style"`
}
```

`DrawIOContent` gains `columns []string`, cloned in `Clone()` and exposed via `GetColumns()`.

## Error Handling

New sentinel errors in `drawio_reader.go`, all wrapped with `fmt.Errorf("%w: ...")` to carry line/directive context while staying `errors.Is`-matchable:

| Sentinel | Requirement | Trigger |
|---|---|---|
| `ErrDrawIONoColumnHeader` | [4.4](requirements.md#4.4) | Empty input or comments/blank lines only |
| `ErrDrawIOTrailingDirective` | [4.5](requirements.md#4.5) | Recognized directive pattern after column header row |
| `ErrDrawIODuplicateColumn` | [4.6](requirements.md#4.6) | Duplicate name in column header row |
| `ErrDrawIODirective` | [4.1](requirements.md#4.1), [4.2](requirements.md#4.2) | Connect unmarshal failure or non-integer numeric directive (message names the directive) |

CSV structural errors ([4.3](requirements.md#4.3)) pass through as `*csv.ParseError`, which already carries line numbers; `ParseDrawIOFile` wraps `os.Open` errors. Sentinels (not one struct type with a kind field) because callers like awstools only need `errors.Is` discrimination; nothing programmatic reads the context fields. No panics anywhere on the parse path ([4.7](requirements.md#4.7)).

## Testing Strategy

`drawio_reader_test.go`, map-based table tests per project convention:

- **Directive parsing**: each of the 18 keys; unknown directives/non-directive comments ignored; duplicate last-wins; malformed-superseded-numeric still errors; connect order preservation; connect with unknown JSON keys; zero-value header for directive-less file.
- **Data section**: leading-`#` data rows parse as data; quoted fields with commas/quotes/newlines; BOM; blank lines; empty-cell `""`-fill; zero data rows.
- **Errors**: one test per sentinel, asserting `errors.Is` and message content (line/directive); trailing-directive precedence over field-count mismatch on the same line; an error-returning `io.Reader` (mid-stream, non-EOF) surfaces as a returned error ([4.7](requirements.md#4.7)); a parity-scan-vs-csv quote-model disagreement case pinning which error fires.
- **API surface**: `ParseDrawIOFile` happy path and wrapped `os.Open` error; `GetColumns` returns a defensive copy; multi-line quoted column name in the header row.
- **Writer changes** (in existing drawio test files): connect JSON byte-compatibility with the previous Sprintf output for escape-free values; a connect value containing `&<>` rendered verbatim (proves `SetEscapeHTML(false)` is wired, not just defaulted); new quoting rules; explicit column order incl. zero-record header row; record keys outside the column list are not rendered (pins the documented silent-drop contract of `WithDrawIOColumns`); `NewDrawIOContentFromTable` schema-order regression; `Clone` copies columns; JSON-renderer regression asserting `DrawIOConnection` key casing in JSON-format output is unchanged (guards the mirror-struct decision).

Property-based tests (`pgregory.net/rapid`, new test-only dependency) — the round-trip requirements are universal guarantees and example tests cannot cover the quoting/escaping interaction space:

- **Idempotency** ([3.1](requirements.md#3.1)): arbitrary `DrawIOHeader` (printable strings incl. `"`,`\`,`,`,`#`, newlines in record values), columns, records → render → parse → render → parse → render; second and third renders byte-equal.
- **Byte-identity** ([3.2](requirements.md#3.2)): same generators restricted to CR-free values; first re-render equals first render.
- **No-panic property** ([4.7](requirements.md#4.7)): arbitrary byte slices into `ParseDrawIOCSV`, plus a directive-biased generator emitting `# connect: <arbitrary-json-ish>` and `# padding: <arbitrary-token>` lines so the unmarshal/Atoi paths are actually reached; asserts error-or-success, never panic.
- Generator constraints (these encode documented carve-outs, not bugs): no `\n`/`\r` in header string fields (Out of Scope), unique column names (else `ErrDrawIODuplicateColumn` correctly fires), record keys ⊆ columns (keys outside the column list are intentionally not rendered).

One example-based golden round-trip with a realistic awstools-style file (connections, identity, parent, layout none + left/top) as an anchor against generator blind spots.
