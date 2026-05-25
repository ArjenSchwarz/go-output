# Bugfix Report: FileWriter CSV Append Can Merge Rows

**Bug ID:** T-1109
**Date:** 2026-05-25
**Status:** Fixed
**Severity:** Medium
**Component:** FileWriter CSV Append Functionality

## Problem Statement

When appending CSV data to an existing file using the FileWriter's append mode, if the existing CSV content does not end with a newline character, the first row of the new data is concatenated onto the last row of the existing data, producing invalid CSV output.

Repro shape:
1. Existing file content: `a,b\n1,2` (no trailing newline)
2. Append CSV payload: `a,b\n3,4\n`
3. Result became `a,b\n1,23,4\n` instead of `a,b\n1,2\n3,4\n`

This is the same defect previously fixed for the S3Writer (see `specs/bugfixes/csv-append-can-merge-rows-without-newline/report.md`, T-80).

## Root Cause Analysis

### Five Whys Analysis

1. **Why do CSV rows get merged?** -> The stripped data rows are appended directly to the file without ensuring line separation from existing content.
2. **Why isn't line separation ensured?** -> `appendCSVWithoutHeaders` normalizes the payload, strips the header, and passes `dataWithoutHeader` straight to `appendByteLevel` (which opens the file `O_APPEND`).
3. **Why doesn't it check for a trailing newline?** -> The original implementation assumed existing CSV files always end with a trailing newline.
4. **Why was this assumption made?** -> Most CSV generators add a trailing newline, but this is not guaranteed for manually created or truncated files.
5. **Why wasn't this caught earlier?** -> The equivalent S3Writer fix was applied to `combineCSVData`, but the FileWriter append path was not updated at the same time.

### Technical Details

The issue was in `appendCSVWithoutHeaders` in `v2/file_writer.go`. After stripping the header, `dataWithoutHeader` was passed directly to `appendByteLevel` with no check on whether the existing file ended with a newline.

Unlike the S3Writer (which reads the whole object into memory and concatenates in `combineCSVData`), the FileWriter appends via `os.OpenFile(..., O_APPEND)` without reading the existing content, so the S3 byte-slice check could not be applied verbatim.

## Solution

### Implementation

`appendCSVWithoutHeaders` now inspects the existing file before appending. A new helper `fileNeedsRowSeparator` stats the file, returns `false` for missing or empty files, and otherwise reads the final byte. If that byte is not `\n`, a single `\n` is prepended to the stripped data rows before they are appended:

```go
if needsSeparator, err := fw.fileNeedsRowSeparator(fullPath); err != nil {
    return fw.wrapError(FormatCSV, err)
} else if needsSeparator {
    dataWithoutHeader = append([]byte("\n"), dataWithoutHeader...)
}
return fw.appendByteLevel(ctx, fullPath, dataWithoutHeader)
```

The helper reads only the last byte (`ReadAt` at `size-1`) rather than loading the entire file, keeping the append path efficient.

### Why This Resolves the Issue

This mirrors the established S3Writer fix: it validates the trailing-newline assumption and corrects it when false, ensuring CSV row integrity while preserving existing behavior when files are already well-formed.

## Testing

### Regression Tests

Added two cases to `TestFileWriterCSVHeaderSkipping` in `v2/file_writer_append_test.go`:

- `existing CSV without trailing newline`: `a,b\n1,2` + `a,b\n3,4\n` -> `a,b\n1,2\n3,4\n`
- `existing CSV without trailing newline multiple rows`: `a,b\n1,2` + `a,b\n3,4\n5,6\n` -> `a,b\n1,2\n3,4\n5,6\n`

Both failed before the fix (producing the merged `a,b\n1,23,4\n` shape) and pass after.

### Test Results

- New regression cases pass.
- Full test suite passes (`make test-all`, unit + integration).
- Linter clean (`make lint`, 0 issues).
- Existing CSV append behavior (header stripping, CRLF normalization, header-only, empty data) unchanged.

## Files Modified

1. **`v2/file_writer.go`** - Added trailing-newline separation in `appendCSVWithoutHeaders` and the `fileNeedsRowSeparator` helper.
2. **`v2/file_writer_append_test.go`** - Added regression cases to `TestFileWriterCSVHeaderSkipping`.

## Related Issues

Mirrors the S3Writer fix for the same class of defect (T-80, `specs/bugfixes/csv-append-can-merge-rows-without-newline/report.md`).
