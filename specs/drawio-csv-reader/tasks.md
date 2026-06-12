---
references:
    - specs/drawio-csv-reader/requirements.md
    - specs/drawio-csv-reader/design.md
    - specs/drawio-csv-reader/decision_log.md
---
# draw.io CSV Reader

## Writer changes

- [x] 1. Write failing tests for DrawIOContent column-order plumbing <!-- id:ih3bs6v -->
  - Test WithDrawIOColumns option sets column order on NewDrawIOContent and Builder.DrawIO (forwarding)
  - Test GetColumns returns a defensive copy (mutating the result does not affect content)
  - Test Clone deep-copies the columns slice
  - Test NewDrawIOContentFromTable captures table.schema.GetFieldNames() order; nil table leaves columns nil (Decision 13)
  - Define DrawIOOption type / WithDrawIOColumns / GetColumns stubs so the package compiles (types exempt from TDD); assertions must fail before task 2
  - Stream: 1
  - Requirements: [3.3](requirements.md#3.3)

- [x] 2. Implement columns field, DrawIOOption, and constructor/builder plumbing <!-- id:ih3bs6w -->
  - v2/graph_content.go: columns []string on DrawIOContent; variadic opts on NewDrawIOContent and NewDrawIOContentFromTable; slices.Clone in Clone(); GetColumns accessor
  - v2/document.go:232: Builder.DrawIO gains opts ...DrawIOOption and forwards them
  - Backward compatible: no existing caller passes options (verified in design parity audit)
  - Blocked-by: ih3bs6v (Write failing tests for DrawIOContent column-order plumbing)
  - Stream: 1
  - Requirements: [3.3](requirements.md#3.3)

- [x] 3. Write failing tests for drawio renderer output changes <!-- id:ih3bs6x -->
  - Explicit column order used by renderDrawIOContent; column header row written even with zero records
  - Record keys outside the column list are not rendered (pins WithDrawIOColumns silent-drop contract)
  - Connect JSON: byte-identical to old Sprintf output for escape-free values; quotes/backslashes escaped per JSON; ampersand/angle-brackets verbatim (proves SetEscapeHTML(false)); invert:false always present
  - Quoting: leading-# field quoted; single-empty-field row rendered as a quoted empty string not a blank line
  - JSON-renderer regression: DrawIOConnection key casing in JSON document output unchanged (guards Decision 14 mirror struct)
  - Blocked-by: ih3bs6w (Implement columns field, DrawIOOption, and constructor/builder plumbing)
  - Stream: 1
  - Requirements: [3.3](requirements.md#3.3), [3.5](requirements.md#3.5), [3.6](requirements.md#3.6)

- [x] 4. Implement renderer changes: explicit columns, connect JSON encoder, quoting rules <!-- id:ih3bs6y -->
  - v2/graph_renderers.go renderDrawIOContent: use GetColumns() when non-empty; fall back to extractColumnNames
  - writeDrawIOHeader: emit connect via drawioConnectionJSON mirror struct (json tags from/to/invert/label/style; no omitempty on Invert) with json.Encoder + SetEscapeHTML(false); Encode trailing newline is the line terminator
  - writeCSVRow: also quote fields starting with # and a single empty sole field
  - Place drawioConnectionJSON in the new v2/drawio_reader.go per design (created here; parser added later)
  - Blocked-by: ih3bs6x (Write failing tests for drawio renderer output changes)
  - Stream: 1
  - Requirements: [3.3](requirements.md#3.3), [3.5](requirements.md#3.5), [3.6](requirements.md#3.6)

## Parser

- [x] 5. Define parser API surface and write failing core parser tests <!-- id:ih3bs6z -->
  - v2/drawio_reader.go: ParsedDrawIO struct; ParseDrawIOCSV/ParseDrawIOFile stubs; sentinel error vars (types/wiring exempt from TDD)
  - v2/drawio_reader_test.go map-based table tests: all 18 directive keys map to DrawIOHeader fields; grammar is case-sensitive exact-prefix with verbatim untrimmed values; CRLF handling (\r\n stripped / lone \r kept)
  - Tolerance: unknown directives and non-directive comments ignored; scalar duplicates last-wins; connect append-all preserving order with unknown JSON keys ignored
  - Zero-value header when directives absent (never DefaultDrawIOHeader); BOM tolerated; blank lines skipped; empty cells materialize as empty string for every column; records and columns in file order; zero-data-rows returns columns + empty records
  - Stream: 1
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2), [1.3](requirements.md#1.3), [1.4](requirements.md#1.4), [1.6](requirements.md#1.6), [2.1](requirements.md#2.1), [2.2](requirements.md#2.2), [2.3](requirements.md#2.3), [2.4](requirements.md#2.4), [2.5](requirements.md#2.5)

- [x] 6. Implement ParseDrawIOCSV/ParseDrawIOFile core pipeline <!-- id:ih3bs70 -->
  - Pipeline per design: BOM strip, directive pre-pass, quote-parity scan seeded at the column header line, csv.Reader with no Comment mode and FieldsPerRecord enforcement, empty-string fill
  - Connect unmarshal via drawioConnectionJSON (from task 4)
  - TrimLeadingSpace stays false
  - Blocked-by: ih3bs6y (Implement renderer changes: explicit columns, connect JSON encoder, quoting rules), ih3bs6z (Define parser API surface and write failing core parser tests)
  - Stream: 1
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2), [1.3](requirements.md#1.3), [1.4](requirements.md#1.4), [1.6](requirements.md#1.6), [2.1](requirements.md#2.1), [2.2](requirements.md#2.2), [2.3](requirements.md#2.3), [2.4](requirements.md#2.4), [2.5](requirements.md#2.5)

- [x] 7. Write failing error-handling and edge-case parser tests <!-- id:ih3bs71 -->
  - Per-sentinel tests asserting errors.Is plus directive/line context: connect unparseable (4.1); non-integer numeric incl. superseded occurrence (4.2); field-count mismatch with line context (4.3); empty/comment-only input (4.4); trailing directive after data incl. precedence over field-count on the same line (4.5); duplicate column names (4.6)
  - Edge cases: leading-# data row parses as data (1.5); quoted fields with commas/quotes/newlines (3.4); multi-line quoted column header name; parity-scan-vs-csv quote-model disagreement case pinning which error fires
  - 4.7: error-returning io.Reader (mid-stream non-EOF) surfaces as error; ParseDrawIOFile wraps os.Open errors; nothing panics
  - Blocked-by: ih3bs70 (Implement ParseDrawIOCSV/ParseDrawIOFile core pipeline)
  - Stream: 1
  - Requirements: [1.5](requirements.md#1.5), [3.4](requirements.md#3.4), [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.3](requirements.md#4.3), [4.4](requirements.md#4.4), [4.5](requirements.md#4.5), [4.6](requirements.md#4.6), [4.7](requirements.md#4.7)

- [x] 8. Implement parser error handling and edge cases <!-- id:ih3bs72 -->
  - Sentinels: ErrDrawIONoColumnHeader, ErrDrawIOTrailingDirective, ErrDrawIODuplicateColumn, ErrDrawIODirective (per design Error Handling table)
  - Whole-section parity scan runs before csv parsing so 4.5 wins over 4.3
  - Blocked-by: ih3bs71 (Write failing error-handling and edge-case parser tests)
  - Stream: 1
  - Requirements: [1.5](requirements.md#1.5), [3.4](requirements.md#3.4), [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.3](requirements.md#4.3), [4.4](requirements.md#4.4), [4.5](requirements.md#4.5), [4.6](requirements.md#4.6), [4.7](requirements.md#4.7)

## Round-trip and validation

- [x] 9. Add property-based round-trip tests and golden example <!-- id:ih3bs73 -->
  - Add pgregory.net/rapid as test-only dependency (go.mod via make mod-tidy)
  - Idempotency property (3.1): render-parse-render-parse-render; second and third renders byte-equal
  - Byte-identity property (3.2): CR-free generated content; first re-render equals first render
  - No-panic property (4.7): arbitrary bytes plus directive-biased generator emitting connect/padding lines so unmarshal/Atoi paths are reached
  - Generator constraints encode documented carve-outs: no newline/CR in header string fields; unique column names; record keys subset of columns
  - Golden example: awstools-style file with connections, identity, parent, layout none + left/top
  - Fix any implementation bugs the properties surface
  - Blocked-by: ih3bs72 (Implement parser error handling and edge cases)
  - Stream: 1
  - Requirements: [3.1](requirements.md#3.1), [3.2](requirements.md#3.2), [4.7](requirements.md#4.7)

- [x] 10. Run full validation pipeline and fix findings <!-- id:ih3bs74 -->
  - make check (fmt + lint + tests) and make modernize; fix all findings
  - Verify existing drawio and JSON/YAML renderer tests still pass unchanged (byte-compat and key-casing guards)
  - Blocked-by: ih3bs73 (Add property-based round-trip tests and golden example)
  - Stream: 1
