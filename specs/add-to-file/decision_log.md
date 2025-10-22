# Decision Log - Add-to-File Feature

## Overview
This document tracks all design decisions made during the requirements gathering phase for the add-to-file feature in v2.

---

## Decision 1: Use Writer-Level Configuration (Option A)
**Date:** 2025-10-21
**Status:** Accepted
**Context:** Need to decide where append mode configuration should live - Writer, Document, or format-specific
**Decision:** Use `WithAppendMode()` functional option at FileWriter creation time
**Rationale:**
- Appending is a writer concern, not a document or format concern
- Makes intent explicit - users must deliberately choose append vs replace
- Consistent with v2's functional options pattern
- Allows different writers to have different append behavior for the same document

**Alternatives Considered:**
- Document-level option (rejected - documents shouldn't know about I/O details)
- Format-specific behavior (rejected - makes behavior implicit and surprising)

---

## Decision 2: Use HTML Comment Marker Instead of Div Element
**Date:** 2025-10-21
**Status:** Accepted
**Context:** v1 used `<div id='end'></div>` as HTML append placeholder
**Decision:** Change to `<!-- go-output-append -->` HTML comment marker
**Rationale:**
- Comment is invisible to users and browsers, won't affect rendering
- Avoids DOM pollution (no unnecessary div element)
- Survives HTML minification and tidying tools
- No risk of CSS/JavaScript interference (ID collisions, styling)
- Less likely to be accidentally deleted by users

**Alternatives Considered:**
- Keep v1's `<div id='end'></div>` (rejected - too fragile)
- Use `</body>` tag detection (rejected - assumes full HTML documents, not fragments)
- Use data attributes (rejected - still creates DOM node)

**Breaking Change:** Yes - v1 HTML files with div placeholders won't work in v2

---

## Decision 3: Support JSON/YAML Byte-Level Appending by Default
**Date:** 2025-10-21
**Status:** Accepted
**Context:** Byte-level appending to JSON/YAML produces invalid output (e.g., `{"a":1}{"b":2}`)
**Decision:** Allow byte-level appending by default, provide optional `WithDisallowUnsafeAppend()` to prevent it
**Rationale:**
- Primary use case is NDJSON-style logging (newline-delimited JSON objects)
- Users explicitly enable append mode, so they understand the implications
- Disallowing by default would break legitimate use cases
- Documentation will clearly explain the concatenation behavior

**Alternatives Considered:**
- Error by default, require `WithUnsafeAppend()` opt-in (rejected - breaks valid use cases)
- Add separate JSONL/NDJSON format (considered for future, not blocking this feature)
- Smart merging (parse, merge, serialize) (rejected - expensive, complex, changes semantics)

---

## Decision 4: Skip CSV Headers When Appending
**Date:** 2025-10-21
**Status:** Accepted
**Context:** CSV files have headers in first row - appending creates duplicate headers
**Decision:** FileWriter SHALL NOT write headers when appending to existing CSV files with content
**Rationale:**
- Duplicate headers break CSV parsers and tools
- Headers only needed once at file creation
- Simple to implement - check if file exists and has content

**Implementation Note:** Renderer needs to track whether headers have been written

---

## Decision 5: Use sync.Mutex for Thread Safety
**Date:** 2025-10-21
**Status:** Accepted
**Context:** Need thread-safe append operations
**Decision:** Use `sync.Mutex` to serialize write operations within a FileWriter instance
**Rationale:**
- Standard Go pattern for protecting shared resources
- Provides safety for concurrent goroutines using same FileWriter
- Simple and well-understood behavior
- Properly protects both simple append and HTML read-modify-write operations

**Limitations:**
- Only protects within single process
- Does not protect across separate FileWriter instances pointing to same file
- Does not provide cross-process locking

**Alternatives Considered:**
- `syscall.Flock` for cross-process locking (rejected - adds complexity, platform differences)
- No locking, rely on `os.O_APPEND` (rejected - doesn't protect HTML read-modify-write)

---

## Decision 6: Use Write-to-Temp-and-Rename for HTML Atomic Updates
**Date:** 2025-10-21
**Status:** Accepted
**Context:** HTML append requires read-modify-write which could leave file corrupted on crash
**Decision:** Use industry-standard atomic file update pattern:
1. Read existing file content
2. Transform in memory (insert new content before marker)
3. Write to temporary file `{filename}.tmp.{random}`
4. Use `os.Rename()` to atomically replace original

**Rationale:**
- `os.Rename()` is atomic on most filesystems
- Ensures original file remains intact if process crashes during write
- Protects against partial writes, disk full, permission errors
- Industry-standard pattern for safe file updates

**Alternatives Considered:**
- Direct in-place modification (rejected - not crash-safe)
- Backup-and-restore on error (rejected - more complex, same result)

---

## Decision 7: Validate File Extensions by Default, Skip for No-Extension Files
**Date:** 2025-10-21
**Status:** Accepted
**Context:** Need to prevent appending wrong format to existing files
**Decision:**
- Validate file extension matches expected format for the output
- If file has no extension, skip validation and proceed
- Error if extension doesn't match

**Rationale:**
- Extension validation is cheap and catches common user errors
- No-extension files are legitimate (e.g., `/dev/stdout`, log files)
- Content validation would be expensive and complex
- Users control file paths, so extension validation is pragmatic

**Alternatives Considered:**
- Content validation (parse file to verify format) (rejected - too expensive)
- Optional strict mode with content validation (deferred to future)

---

## Decision 8: UTF-8 Encoding Assumption
**Date:** 2025-10-21
**Status:** Accepted
**Context:** Need to specify text encoding for files
**Decision:** FileWriter SHALL assume all files are UTF-8 encoded
**Rationale:**
- UTF-8 is the de facto standard for modern text files
- Go strings are UTF-8 by default
- Simplifies implementation - no encoding detection or conversion
- Users needing other encodings can handle conversion themselves

**Documentation Requirement:** Clearly state UTF-8 requirement in docs

---

## Decision 9: Default File Permissions 0644 with Override Option
**Date:** 2025-10-21
**Status:** Accepted
**Context:** Need to specify permissions for newly created files
**Decision:**
- Default to 0644 (rw-r--r--)
- Provide `WithPermissions(os.FileMode)` option to override

**Rationale:**
- 0644 is standard for user-created files (owner write, all read)
- Matches common Unix/Linux conventions
- Option provides flexibility for security-sensitive scenarios

---

## Decision 10: Provide HTML Template Helper Function
**Date:** 2025-10-21
**Status:** Accepted
**Context:** Users need way to create initial HTML files with comment marker
**Decision:** Provide helper function to generate HTML template with `<!-- go-output-append -->` marker
**Rationale:**
- Makes it easy to start using append mode for HTML
- Ensures marker is correctly placed
- Reduces user errors
- Low implementation cost

**API:** Details to be determined in design phase

---

## Decision 11: Immutable Append Mode Configuration
**Date:** 2025-10-21
**Status:** Accepted
**Context:** Should append mode be changeable after FileWriter creation?
**Decision:** Append mode SHALL be set at creation time and remain immutable
**Rationale:**
- Simpler mental model - FileWriter behavior is consistent
- Avoids bugs from mode switching mid-operation
- If users need both modes, they can create two FileWriter instances
- Consistent with other FileWriter configuration (directory, pattern)

---

## Decision 12: No IsAppendMode() Query Method
**Date:** 2025-10-21
**Status:** Accepted (Removed from Requirements)
**Context:** Original requirements included `IsAppendMode() bool` method
**Decision:** Remove this method from requirements
**Rationale:**
- Violates encapsulation - FileWriter is behavior, not queryable state
- No clear use case for querying mode after configuration
- Caller already knows what they configured
- Consistent with Go's io.Writer pattern (doesn't expose internal state)

---

## Decision 13: Multi-Section Format-Appropriate Separators
**Date:** 2025-10-21
**Status:** Accepted
**Context:** How to handle boundaries between sections when appending
**Decision:** Use format-appropriate separators:
- Text formats: newlines
- HTML: marker positioning (all sections before final marker)

**Rationale:**
- Each format has natural section separation mechanisms
- Renderer already handles section formatting
- Writer just preserves what renderer produces

---

## Decision 14: S3Writer Append Support via Download-Modify-Upload
**Date:** 2025-10-21 (Revised)
**Status:** Accepted
**Context:** Should S3Writer support append mode?
**Decision:** S3Writer SHALL support append mode using download-modify-upload pattern with safeguards
**Rationale:**
- User has specific use case: infrequent NDJSON logging to S3
- Updates are sporadic (monthly to a few times daily), so race condition risk is low
- Files are small, so download/upload cost is minimal
- ETag-based conditional updates can detect conflicts
- Size limits prevent memory/bandwidth issues

**Safeguards:**
- ETag-based conflict detection (return error if concurrent modification detected)
- Configurable size limit (default 100MB)
- Clear documentation about non-atomic behavior
- Recommendation to enable S3 versioning

**Alternatives Considered:**
- No S3 append support (rejected - breaks legitimate use case)
- S3 multipart upload for appends (rejected - overcomplicated for small objects)

**Revised:** Initially decided against S3 append, but user provided valid use case that justifies the implementation with appropriate safeguards

---

## Decision 15: HTML Fragment Mode for Append Operations
**Date:** 2025-10-21
**Status:** Accepted
**Context:** New HTML templating engine supports two modes: full HTML page and HTML fragment
**Decision:** FileWriter SHALL use full HTML page mode for new files and HTML fragment mode for appending to existing files
**Rationale:**
- Cleaner separation of concerns - renderer doesn't need to know about append vs create
- Full page structure only needed once (when file is created)
- Fragments avoid duplicating html/head/body tags when appending
- FileWriter can determine mode by checking if file exists
- Simpler than having renderer generate full pages and FileWriter stripping structure

**Implementation Flow:**
1. New file: FileWriter detects file doesn't exist → requests full HTML page from renderer (includes marker)
2. Existing file: FileWriter detects file exists → requests HTML fragment from renderer → inserts before marker

**Benefits:**
- No duplicate HTML structure tags
- Cleaner generated HTML
- Renderer API is simpler (just two modes, not append-aware)
- FileWriter owns the append logic completely

---

## Future Considerations

### Items Deferred to Future Releases
1. **JSONL/NDJSON Format Support** - Dedicated format for newline-delimited JSON
2. **Strict Content Validation Mode** - Optional content parsing to verify format
3. **File Rotation** - Automatic rotation based on size/age
4. **Compression Support** - Appending to gzip/bzip2 files
5. **S3 Append Mode** - If there's demand and a good solution for conflicts

### Open Questions
None at this time.

---

## Version History
- 2025-10-21: Initial decision log created during requirements gathering
