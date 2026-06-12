# Requirements: draw.io CSV Reader

## Introduction

The v2 library can write draw.io CSV output (via `DrawIOContent` and the drawio renderer) but cannot read such a file back in. This feature adds a parser that reads a draw.io CSV file produced by the v2 drawio renderer and returns the `DrawIOHeader` configuration, the column names, and the data records. It replaces v1's `GetHeaderAndContentsFromFile` and unblocks read-merge-rewrite workflows such as the awstools `tgw overview --append` multi-account merge (Transit ticket T-1539).

## Out of Scope

- Parsing draw.io header directives that `DrawIOHeader` does not represent (e.g. `# stylename:`, `# styles:`, `# vars:`, `# labels:`); such directives are ignored, not errors.
- Files containing multiple diagram blocks (multiple header + table sections concatenated); the parser detects and rejects them rather than supporting them. Detection relies on the second block emitting at least one directive line; a block with a fully zero-value header is not detectable.
- Fidelity guarantees for files not produced by the v2 drawio renderer (hand-written or v1-written files parse on a best-effort basis under the tolerance rules below).
- Recovering records that pre-feature renderer output stored as bare blank lines (a single-column table row with an empty cell); CSV parsing cannot distinguish these from blank separator lines, so they are lost on read.
- Header directive values containing line breaks; the draw.io CSV header format cannot represent them, and the writer's behavior for such values is unchanged by this feature.
- Parsing draw.io XML (`.drawio`) files.
- Semantic validation of parsed values (e.g. whether `%Name%` placeholders match actual columns).
- Changes to the v1 `drawio` package.

## Requirements

### 1. Parse draw.io CSV into header, columns, and records

**User Story:** As a developer migrating from v1, I want to read an existing draw.io CSV file into its header configuration, column order, and data records, so that I can merge new data and re-render the diagram.

**Acceptance Criteria:**

1. <a name="1.1"></a>WHEN given input produced by the v2 drawio renderer, the system SHALL return the `DrawIOHeader` represented by the file's header directives, the column names in file order, and the data rows as `[]Record` in file order.
2. <a name="1.2"></a>The system SHALL accept input from an `io.Reader` and SHALL provide a convenience function that reads from a file path.
3. <a name="1.3"></a>Each returned record SHALL contain an entry for every column name; WHEN a CSV cell is empty, the record value SHALL be the empty string `""`.
4. <a name="1.4"></a>WHEN the input contains a column header row but no data rows, the system SHALL return the column names and zero records without error.
5. <a name="1.5"></a>WHEN a row after the column header row begins with `#` but does not match a recognized directive pattern (see [4.5](#4.5)), the system SHALL parse it as data, not as a comment.
6. <a name="1.6"></a>The system SHALL skip blank lines and SHALL tolerate a leading UTF-8 byte order mark.

### 2. Header directive parsing

**User Story:** As a developer, I want header directives parsed back into `DrawIOHeader` fields, so that re-rendering preserves the diagram configuration.

**Acceptance Criteria:**

1. <a name="2.1"></a>The system SHALL recognize directive lines of the exact form `# key: value` (key matched case-sensitively, value taken verbatim to the end of the line) and SHALL populate the corresponding `DrawIOHeader` field for every directive the v2 renderer emits: `label`, `style`, `identity`, `parent`, `parentstyle`, `namespace`, `connect`, `height`, `width`, `ignore`, `nodespacing`, `levelspacing`, `edgespacing`, `padding`, `link`, `left`, `top`, `layout`.
2. <a name="2.2"></a>WHEN one or more `# connect:` directives are present, the system SHALL parse each value into a `DrawIOConnection`, ignoring unknown JSON keys, and SHALL preserve their file order in `DrawIOHeader.Connections`.
3. <a name="2.3"></a>WHEN a directive is absent from the input, the corresponding `DrawIOHeader` field SHALL be its zero value; the system SHALL NOT substitute `DefaultDrawIOHeader()` values.
4. <a name="2.4"></a>WHEN a comment line contains an unrecognized directive key or no `key: value` structure, the system SHALL ignore that line.
5. <a name="2.5"></a>WHEN a recognized directive other than `connect` appears more than once, the last occurrence SHALL win; malformed occurrences still error per [4.2](#4.2) (intentional asymmetry: valid superseded values are dropped, malformed ones fail loudly).

### 3. Round-trip stability

**User Story:** As a developer using the append workflow, I want write→read→write to be stable, so that repeated append runs do not alter or corrupt the diagram file.

**Acceptance Criteria:**

1. <a name="3.1"></a>WHEN a file produced by the v2 drawio renderer is parsed and the returned header, columns, and records are rendered again, parsing and re-rendering that output SHALL reproduce it byte-for-byte (parse→render is idempotent from the first re-render onward).
2. <a name="3.2"></a>WHEN a file produced by the v2 drawio renderer (as modified by this feature) contains no carriage-return characters in field values, the first re-render SHALL already be byte-identical to the original file; files written by the renderer before this feature are covered by [3.1](#3.1) only.
3. <a name="3.3"></a>WHEN re-rendering parsed content, the system SHALL write columns in the parsed file order, including the column header row when there are zero records.
4. <a name="3.4"></a>The system SHALL correctly parse quoted CSV fields containing commas, double quotes, and newlines as written by the renderer.
5. <a name="3.5"></a>The renderer SHALL emit each `# connect:` value as valid JSON with exactly the keys `from`, `to`, `invert`, `label`, `style` in that order, all keys always present, string values escaped per JSON rules (including double quotes and backslashes), and no HTML escaping (`&`, `<`, `>` written verbatim).
6. <a name="3.6"></a>WHEN a data field value begins with `#`, or a row consists of a single empty field, the renderer SHALL quote the field so the line cannot be mistaken for a comment, directive, or blank line.

### 4. Error handling

**User Story:** As a developer, I want malformed input to fail with a clear error, so that the append workflow stops loudly instead of silently losing diagram data.

**Acceptance Criteria:**

1. <a name="4.1"></a>WHEN a `# connect:` directive value cannot be parsed into a `DrawIOConnection`, the system SHALL return an error identifying the offending directive.
2. <a name="4.2"></a>WHEN a recognized numeric directive (`nodespacing`, `levelspacing`, `edgespacing`, `padding`) has a non-integer value in any occurrence — including one superseded per [2.5](#2.5) — the system SHALL return an error identifying the directive.
3. <a name="4.3"></a>WHEN the CSV data section is malformed (e.g. a row's field count differs from the column header row), the system SHALL return an error including line context.
4. <a name="4.4"></a>WHEN the input contains no column header row (empty input or comments only), the system SHALL return an error.
5. <a name="4.5"></a>WHEN an unquoted line after the column header row matches a recognized directive pattern (`# key: value` with a key from [2.1](#2.1), e.g. the start of a second diagram block), the system SHALL return an error distinguishable from a malformed-CSV error; this check SHALL take precedence over [4.3](#4.3) for the same line.
6. <a name="4.6"></a>WHEN the column header row contains duplicate column names, the system SHALL return an error.
7. <a name="4.7"></a>The system SHALL report all failures via returned errors and SHALL NOT panic or terminate the process on malformed input or I/O failure.
