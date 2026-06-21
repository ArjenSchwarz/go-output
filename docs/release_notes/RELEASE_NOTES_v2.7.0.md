# Release Notes: go-output v2.7.0

**Release Date:** June 21, 2026

## Overview

Version 2.7.0 adds a draw.io CSV reader: output produced by the draw.io renderer can now be **parsed back into structured data**, giving a full write → read round-trip. It also includes a set of robustness and correctness fixes across rendering, transformations, progress reporting, and concurrent use.

## ⚠️ Breaking Changes

**No API breaking changes** — every new API is additive and existing behavior is unchanged except where it was previously incorrect (see Bug Fixes).

**One build requirement change:** the **minimum Go version is now 1.25** (raised from 1.24). This was necessary to pull in the fix for [GO-2026-5024](https://pkg.go.dev/vuln/GO-2026-5024) in the indirect `golang.org/x/sys` dependency — a Windows-only issue in code go-output does not call, addressed here as a hygiene measure. Upgrade your toolchain to Go 1.25 or newer before updating.

---

## ✨ New Features

### Draw.io CSV Round-Trip

**What's New:**
Two new functions read draw.io CSV (as written by the draw.io renderer) back into a structured `ParsedDrawIO` value:

```go
type ParsedDrawIO struct {
    Header  DrawIOHeader // zero-valued fields for absent directives
    Columns []string     // column header row, in file order
    Records []Record     // data rows, in file order
}

func ParseDrawIOCSV(r io.Reader) (*ParsedDrawIO, error)
func ParseDrawIOFile(path string) (*ParsedDrawIO, error)
```

**Why This Matters:**
Previously draw.io CSV was write-only. You can now load a generated diagram file, inspect or modify its header directives, columns, and records, and re-render it — useful for tooling that edits diagrams, validates generated output, or migrates data between formats.

**Example:**

```go
parsed, err := output.ParseDrawIOFile("infrastructure.csv")
if err != nil {
    log.Fatal(err)
}

fmt.Println("Columns:", parsed.Columns)
for _, rec := range parsed.Records {
    fmt.Printf("%v (%v)\n", rec["Name"], rec["Type"])
}
```

**Robust parsing:**
- Recognizes all 18 draw.io header directives and maps them onto `DrawIOHeader`.
- Handles UTF-8 BOM, CRLF line endings, blank lines, quoted fields (commas, quotes, embedded newlines), and data rows starting with `#`.
- Unknown directives and comments are ignored; absent directives leave the corresponding `DrawIOHeader` field zero-valued (defaults are never substituted).

**Discriminable errors:**
The parser returns sentinel errors that work with `errors.Is` and carry line/directive context:

```go
if _, err := output.ParseDrawIOFile("diagram.csv"); err != nil {
    switch {
    case errors.Is(err, output.ErrDrawIONoColumnHeader):
        // input had no column header row
    case errors.Is(err, output.ErrDrawIODuplicateColumn):
        // column header row repeats a name
    case errors.Is(err, output.ErrDrawIOTrailingDirective):
        // a directive appeared after the column header row
    case errors.Is(err, output.ErrDrawIODirective):
        // a directive had an invalid value (e.g. bad connect JSON)
    }
}
```

### Stable, Round-Trippable Draw.io Output

To make round-trips deterministic, the draw.io writer gained explicit column ordering:

```go
doc := output.New().
    DrawIO("infrastructure", records, output.DefaultDrawIOHeader(),
        output.WithDrawIOColumns("Name", "Type", "Region")).
    Build()
```

- New `DrawIOOption` type and `WithDrawIOColumns()` option, accepted by `NewDrawIOContent`, `NewDrawIOContentFromTable`, and `Builder.DrawIO`.
- New `GetColumns()` accessor on `DrawIOContent`.
- `NewDrawIOContentFromTable` now preserves the table's schema field order instead of alphabetizing columns at render time.
- `# connect:` directives emit deterministic JSON.

All of these are backward compatible — existing calls without the new option keep working.

---

## 🛠️ Bug Fixes

This release resolves a broad set of defects. They are grouped by theme below.

### Robustness — panics replaced with errors or safe no-ops
- Renderers no longer panic on nil documents or nil writers (Markdown, graph/DOT/Mermaid renderers, every `RenderTo` implementation, and `MultiWriter`).
- Builder methods reject or safely ignore nil inputs: `AddContent` (on both `Builder` and `SectionContent`), `Section` and nested-section callbacks, and table operations on nil content.
- Transformation handling no longer panics on nil operations or transformers; `GroupByOp.Validate` rejects nil aggregate functions; `Output.Render` validates nil configuration entries.
- `NewDrawIOContentFromTable` guards against a nil table; S3 output returns a clear error instead of panicking when a bucket is set without a client.
- Progress: fixed a startup context panic, a nil-writer panic on draw, and a progress-bar overflow panic.

### Deterministic output
- Stable row order in graph renderers, deterministic GroupBy aggregate column order, and deterministic Markdown front-matter order.
- `GenerateID` no longer returns a constant fallback ID.
- GroupBy no longer merges distinct rows whose composite keys collide.
- Transformers now run in priority order; `SortTransformer` keeps the Markdown separator row directly under the header.

### Data integrity
Caller-owned data is now defensively copied so later caller mutations cannot corrupt rendered output. This covers table records, schema key order, chart data, graph/Draw.io data, collapsible values and sections, and `CollapsibleValue` format hints. A race in `DefaultCollapsibleValue` lazy caching during concurrent rendering was also fixed.

### Rendering correctness
- CSV: headers are retained for later tables and for schema changes inside collapsible/nested sections; append no longer merges rows when the existing file lacks a trailing newline; flush errors are surfaced; tabular transformers parse quoted CSV correctly; draw.io CSV output is flushed.
- Graph renderers now apply table transformations; DOT and Mermaid labels are escaped; HTML charts render Mermaid correctly instead of emitting raw text; Mermaid pie charts accept all numeric types.
- The emoji transformer no longer corrupts words that contain emoji-indicator substrings.
- Caller context is threaded through nested renderers; `prettyProgress.SetContext` no longer marks a healthy progress as failed when the context is replaced.

### Other
- `AddColumn` rejects duplicate column names.
- The Builder surfaces previously dropped nested-section errors.
- The S3 append size limit now accounts for the size of the new data.

---

## 📦 Installation

```bash
go get github.com/ArjenSchwarz/go-output/v2@v2.7.0
```

---

## ⬆️ Upgrade Notes

- **Requires Go 1.25 or newer.** Update your toolchain before upgrading (see Breaking Changes).
- **No code changes required.** Aside from the toolchain bump, this is a drop-in upgrade from any v2.x release.
- Applications relying on the previous (incorrect) behaviors fixed above — for example, code that depended on alphabetized draw.io columns, or that passed nil values expecting a panic — should review the Bug Fixes section.
- Output that was previously nondeterministic (graph row order, GroupBy aggregate columns, Markdown front matter) is now stable; update any golden-file fixtures accordingly.

---

## 📖 Additional Resources

- [CHANGELOG.md](../../CHANGELOG.md) - Complete version history
- [README.md](../../README.md) - Quick start and feature overview
- [v2/docs/](../../v2/docs/) - API reference and migration guides
- [v2/examples/](../../v2/examples/) - Working code examples
