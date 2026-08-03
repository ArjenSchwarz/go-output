# Bugfix Report: Nil Functional Options Panic Constructors

**Date:** 2026-08-03
**Status:** Fixed
**Ticket:** T-1444

## Description of the Issue

Every v2 public constructor and option applicator that accepts variadic functional options invoked each option without checking for nil. Passing a nil option value caused a runtime panic (`invalid memory address or nil pointer dereference`) instead of being ignored.

**Reproduction steps:**
1. Call any option-accepting constructor with a nil option, e.g. `output.NewTableContent("x", data, nil)` or `output.NewOutput(nil)`
2. Observe `panic: runtime error: invalid memory address or nil pointer dereference`

**Impact:** Any caller that builds option slices dynamically (e.g. `var opts []output.TableOption` with conditional appends) could crash the program with a non-obvious panic. Affected all eleven option families: TableOption, TextOption, RawOption, SectionOption, OutputOption, CollapsibleOption, CollapsibleSectionOption, ProgressOption, FileWriterOption, S3WriterOption, and DrawIOOption.

## Investigation Summary

Applied the systematic-debugger (Fagan inspection) methodology.

- **Symptoms examined:** Nil pointer dereference panics from the option application loops in constructors.
- **Code inspected:** All variadic option application sites found via `grep -rn "range opts"` across v2. Fourteen loops in eleven files, all following the pattern `for _, opt := range opts { opt(x) }` with no nil guard. All other variadic option functions (Builder methods, `NewCollapsibleTable`, `NewProgressForFormat`, collapsible formatters, etc.) forward to these fourteen sites.
- **Hypotheses tested:** Checked whether any site already guarded against nil (`grep "opt == nil"` — none did), confirming the panic surface is uniform across all option families.

## Discovered Root Cause

The option application loops call every element of the variadic slice unconditionally. Function types in Go are nilable, so a nil `TableOption` (etc.) is a valid value of the type, and calling it dereferences a nil pointer.

**Defect type:** Missing validation / boundary condition (nil input)

**Why it occurred:** The functional options pattern was implemented with the implicit assumption that all options come from the library's `With*` constructors and are never nil. That non-nil contract is invisible in the type system, so nothing enforced or documented it, and callers building option slices dynamically can easily introduce nils.

**Contributing factors:** The same loop pattern was copied across eleven files as new option families were added, replicating the missing guard each time.

## Resolution for the Issue

**Changes made:** Added a `if opt == nil { continue }` guard to all fourteen option application loops:

- `v2/table_options.go` - `ApplyTableOptions`
- `v2/text_options.go` - `ApplyTextOptions`
- `v2/raw_options.go` - `ApplyRawOptions`
- `v2/section_options.go` - `ApplySectionOptions`
- `v2/output.go` - `NewOutput`
- `v2/collapsible.go` - `NewCollapsibleValue`
- `v2/collapsible_section.go` - `NewCollapsibleSection`
- `v2/progress.go` - `NewProgress`, `NewAutoProgress`
- `v2/progress_pretty.go` - `NewPrettyProgress`
- `v2/file_writer.go` - `NewFileWriterWithOptions`
- `v2/s3_writer.go` - `NewS3WriterWithOptions`
- `v2/graph_content.go` - `NewDrawIOContent`, `NewDrawIOContentFromTable`

**Approach rationale:** Nil options are silently skipped rather than reported as errors. Options are configuration-only and additive, most of these constructors do not return errors (erroring would require breaking API changes), and a uniform skip rule across all families is the simplest contract for callers. No existing code could depend on the previous behaviour, since nil options always panicked.

**Alternatives considered:**
- Return a validation error from constructors that already return errors (`NewTableContent`, `NewRawContent`, `NewFileWriterWithOptions`) and skip elsewhere - Rejected: inconsistent behaviour between families makes the API harder to reason about, and a nil option is not a meaningful failure condition for configuration-only options.
- Panic with a descriptive message - Rejected: still crashes callers; the ticket asks for the crash surface to be removed.

## Regression Test

**Test file:** `v2/nil_options_test.go`
**Test names:** `TestNilFunctionalOptionsDoNotPanic`, `TestNilOptionsInterleavedWithRealOptions`

**What it verifies:** Every public option-accepting constructor and applicator accepts nil options without panicking, and real options surrounding a nil are still applied (table key order, text style).

**Run command:** `go test -run "TestNilFunctionalOptionsDoNotPanic|TestNilOptionsInterleavedWithRealOptions" ./v2`

## Affected Files

| File | Change |
|------|--------|
| `v2/table_options.go` | Nil guard in `ApplyTableOptions` |
| `v2/text_options.go` | Nil guard in `ApplyTextOptions` |
| `v2/raw_options.go` | Nil guard in `ApplyRawOptions` |
| `v2/section_options.go` | Nil guard in `ApplySectionOptions` |
| `v2/output.go` | Nil guard in `NewOutput` |
| `v2/collapsible.go` | Nil guard in `NewCollapsibleValue` |
| `v2/collapsible_section.go` | Nil guard in `NewCollapsibleSection` |
| `v2/progress.go` | Nil guards in `NewProgress` and `NewAutoProgress` |
| `v2/progress_pretty.go` | Nil guard in `NewPrettyProgress` |
| `v2/file_writer.go` | Nil guard in `NewFileWriterWithOptions` |
| `v2/s3_writer.go` | Nil guard in `NewS3WriterWithOptions` |
| `v2/graph_content.go` | Nil guards in `NewDrawIOContent` and `NewDrawIOContentFromTable` |
| `v2/nil_options_test.go` | New regression tests |
| `CHANGELOG.md` | Fixed entry |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`make test`; the pre-existing example go.mod staleness that breaks `make fmt` in `v2/examples/` also exists on main and is unrelated)
- [x] Linters/validators pass (`make lint`: 0 issues; `gofmt -l` clean)

**Manual verification:**
- Red/green cycle: all 20 regression subtests panicked before the fix and pass after it.
- Godoc comments on all fourteen application sites now state "Nil options are ignored."

## Prevention

**Recommendations to avoid similar bugs:**
- When adding a new option family, copy an existing option loop (they now all contain the nil guard) rather than rewriting the pattern.
- Regression tests in `v2/nil_options_test.go` cover every option family; add a case there when introducing a new option-accepting constructor.

## Related

- Transit ticket T-1444
