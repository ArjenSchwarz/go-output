# draw.io CSV (v2 writer/reader)

Spec: `specs/drawio-csv-reader/` (T-1539). Writer-changes phase (tasks 1-4) and parser phase (tasks 5-8, `ParseDrawIOCSV`/`ParseDrawIOFile` in `v2/drawio_reader.go`) implemented; round-trip/property tests (tasks 9-10) remain.

## Parser (`v2/drawio_reader.go`)

- `ParseDrawIOCSV(io.Reader) (*ParsedDrawIO, error)`; `ParsedDrawIO{Header DrawIOHeader, Columns []string, Records []Record}`. `ParseDrawIOFile` is `os.Open` + parse, open error wrapped with `%w`.
- Pipeline: read-all → strip UTF-8 BOM → line split (`\r\n` stripped, lone `\r` kept — matches encoding/csv so directives/data normalize identically) → directive pre-pass → quote-parity scan → `csv.Reader` over header+data → duplicate-column check → record fill.
- Directive grammar: exact prefix `"# "` + case-sensitive key + `": "`, value verbatim untrimmed to EOL (`matchDrawIODirective`). 18 keys. Anything else `#`-leading before the column header is silently ignored; after the header it is **data** (no `Comment` mode on csv.Reader, Decision 8).
- Scalars last-wins; `connect` append-all (order preserved), unmarshaled via `drawioConnectionJSON` (unknown JSON keys ignored). Numeric keys (`nodespacing`/`levelspacing`/`edgespacing`/`padding`) `strconv.Atoi`-validated on **every** occurrence — a malformed superseded value still errors (intentional asymmetry, req 2.5/4.2).
- Absent directives → zero values, never `DefaultDrawIOHeader()` (Decision 11; absent-vs-0 for ints is accepted ambiguity).
- Quote-parity scan: seeded at the **header line start** (so multi-line quoted column names work — continuation lines aren't record boundaries); toggles on odd `"` count per physical line; only boundary lines after the header are tested for trailing directives. Runs over the whole section before csv, so `ErrDrawIOTrailingDirective` beats field-count errors anywhere (multi-block detection, Decision 9).
- Parity-vs-csv quote-model disagreement (bare quote mid-field, e.g. `x"q,b`): parity scan thinks following lines are quoted → trailing directive not flagged → csv bare-quote `ParseError` fires instead. Pinned by `TestParseDrawIOCSV_ParityVsCSVQuoteDisagreement`.
- Sentinels: `ErrDrawIONoColumnHeader` (4.4), `ErrDrawIOTrailingDirective` (4.5), `ErrDrawIODuplicateColumn` (4.6), `ErrDrawIODirective` (4.1/4.2 — connect JSON + non-integer numerics). All wrapped via `fmt.Errorf("%w: line %d: ...")`. CSV structural errors pass through as `*csv.ParseError` with `Line`/`StartLine` shifted by `headerIdx` (`adjustDrawIOCSVError`) so they're file-relative, not section-relative.
- `TrimLeadingSpace` stays false; `FieldsPerRecord` default enforces field counts; blank lines skipped by csv (and the pre-pass); empty cells become `""`, every column present in every record; zero data rows → `Records` is empty non-nil.
- goconst gotcha: directive keys used in both the key set and switch need consts (`drawioKeyLabel` etc.) or lint fails.

## Column-order plumbing

- `DrawIOContent` has a private `columns []string` field (`v2/graph_content.go`). Nil/empty means the renderer falls back to `extractColumnNames` (alphabetized from record keys) — the pre-feature behavior.
- Set via `WithDrawIOColumns(...)` (a `DrawIOOption`); `NewDrawIOContent`, `NewDrawIOContentFromTable`, and `Builder.DrawIO` (`v2/document.go`) all take variadic `opts ...DrawIOOption`. Backward compatible: no pre-existing caller passed options.
- `NewDrawIOContentFromTable` captures `table.schema.GetFieldNames()` order (Decision 13) — behavior change: table-sourced drawio CSV columns went from alphabetical to schema order. Nil table leaves columns nil. `GetKeyOrder` already returns a clone, so no aliasing.
- `GetColumns()` returns a defensive copy; `Clone()` does `slices.Clone(d.columns)`.
- Silent-drop contract: record keys outside the explicit column list are not rendered; columns missing from a record render as `""`. With explicit columns, the header row is written even with zero records.

## Connect JSON wire format (Decisions 10 + 14)

- `# connect:` values are encoded via the private `drawioConnectionJSON` mirror struct (`v2/drawio_reader.go`) with json tags `from/to/invert/label/style` — `Invert` has **no omitempty** (`"invert":false` is load-bearing for byte identity).
- `writeDrawIOHeader` uses `json.Encoder` with `SetEscapeHTML(false)` (so `&<>` stay verbatim); `Encode`'s trailing `\n` is the directive line terminator. Output is byte-identical to the old Sprintf for escape-free values.
- Do NOT add json tags to the public `DrawIOConnection`: `renderDrawIOContentJSON` marshals `GetHeader()`, so tags would flip JSON-document key casing. Guarded by `TestJSONRenderer_DrawIOConnectionKeyCasing`.
- Conversion `drawioConnectionJSON(conn)` works because field order/types match; staticcheck S1016 enforces it.

## CSV quoting (requirement 3.6)

`writeCSVRow` (`v2/graph_renderers.go`) quotes a field when it contains `,"\n\r`, **starts with `#`** (would look like a comment/directive), or is the **single empty sole field of a row** (would render as a blank line CSV readers skip). Empty fields in multi-field rows stay unquoted.

## Gotchas

- Tests are internal (package `output`); renderers are instantiated directly (`&drawioRenderer{}`, `&jsonRenderer{}`).
- The Makefile lives at the repo root, not in v2/ (targets operate on v2).
- `rune` CLI arg order is `rune complete <file> <task-id>`.
