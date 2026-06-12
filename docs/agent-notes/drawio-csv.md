# draw.io CSV (v2 writer/reader)

Spec: `specs/drawio-csv-reader/` (T-1539). Writer-changes phase (tasks 1-4) implemented; parser (`ParseDrawIOCSV`/`ParseDrawIOFile` in `v2/drawio_reader.go`) comes in later tasks.

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
