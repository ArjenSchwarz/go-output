# Add-to-File Feature Requirements

## Introduction

The add-to-file feature enables users to append new rendered output to existing files instead of replacing them. This feature replicates the v1 `ShouldAppend` functionality but adapts it to v2's Document-Builder pattern and writer architecture. This is essential for scenarios where users want to incrementally build reports or logs by adding new data to existing files without losing previous content.

## Requirements

### 1. FileWriter Append Mode Support

**User Story:** As a developer, I want to configure a FileWriter to append to existing files instead of replacing them, so that I can incrementally build output files without losing previous content.

**Acceptance Criteria:**

1. <a name="1.1"></a>The FileWriter SHALL provide a `WithAppendMode()` functional option that enables append mode
2. <a name="1.2"></a>When append mode is enabled, the FileWriter SHALL use `os.O_APPEND` flag when opening files for formats that support simple appending
3. <a name="1.3"></a>When append mode is disabled (default), the FileWriter SHALL use `os.O_TRUNC` flag to replace existing file contents
4. <a name="1.4"></a>The append mode configuration SHALL be set at FileWriter creation time and remain immutable thereafter
5. <a name="1.5"></a>The FileWriter SHALL be thread-safe when multiple goroutines write to the same file in append mode

### 2. Format-Specific Append Behavior

**User Story:** As a developer, I want different output formats to handle appending appropriately based on their structure, so that appended content remains valid and properly formatted.

**Acceptance Criteria:**

1. <a name="2.1"></a>The FileWriter SHALL support pure byte-level appending for text-based formats (table, markdown, CSV, YAML, JSON)
2. <a name="2.2"></a>For HTML format in append mode, the FileWriter SHALL read the existing file, locate the `<!-- go-output-append -->` comment marker, and replace it with new content followed by the marker
3. <a name="2.3"></a>If HTML append mode is used and the existing file does not contain the `<!-- go-output-append -->` comment marker, the FileWriter SHALL return an error
4. <a name="2.4"></a>The FileWriter SHALL NOT parse or merge structured data for JSON/YAML/CSV formats - it SHALL perform pure byte-level appending
5. <a name="2.5"></a>The documentation SHALL explain that JSON/YAML byte-level appending produces concatenated output (useful for NDJSON-style logging), not merged data structures
6. <a name="2.6"></a>The FileWriter MAY provide a `WithDisallowUnsafeAppend()` option to prevent appending to formats where concatenation may not be desired
7. <a name="2.7"></a>For CSV format, the FileWriter SHALL NOT write headers when appending to existing files with content

### 3. File Creation and Validation

**User Story:** As a developer, I want the FileWriter to handle missing files and format mismatches appropriately, so that I can avoid data corruption and receive clear error messages.

**Acceptance Criteria:**

1. <a name="3.1"></a>When append mode is enabled and the target file does not exist, the FileWriter SHALL create a new file and write content normally, including format-specific structures (e.g., HTML comment marker for HTML files)
2. <a name="3.2"></a>The FileWriter SHALL validate that the file format matches the output format when appending to existing files
3. <a name="3.3"></a>If the file extension does not match the expected extension for the output format, the FileWriter SHALL return a format mismatch error
4. <a name="3.4"></a>If the file has no extension, format validation SHALL be skipped and append SHALL proceed
5. <a name="3.5"></a>The FileWriter SHALL return an error with clear context if it cannot write to the file due to permissions or other I/O errors
6. <a name="3.6"></a>All validation SHALL occur before any file modifications are made
7. <a name="3.7"></a>The FileWriter SHALL assume all files are UTF-8 encoded
8. <a name="3.8"></a>Newly created files SHALL use 0644 permissions by default

### 4. HTML Comment Marker System

**User Story:** As a developer, I want HTML files to use a comment marker-based append system, so that new content can be inserted at the correct location within the HTML structure without affecting the DOM or styling.

**Acceptance Criteria:**

1. <a name="4.1"></a>The HTML renderer SHALL support two output modes: full HTML page (default) and HTML fragment
2. <a name="4.2"></a>When rendering a full HTML page, the HTML renderer SHALL include a `<!-- go-output-append -->` comment marker at the end before the closing body/html tags
3. <a name="4.3"></a>When the FileWriter appends to a new (non-existent) HTML file, the renderer SHALL generate a complete HTML page with the comment marker
4. <a name="4.4"></a>When the FileWriter appends to an existing HTML file, the renderer SHALL generate only HTML fragments (without full page structure)
5. <a name="4.5"></a>The FileWriter SHALL detect whether the target file exists to determine which rendering mode to request
6. <a name="4.6"></a>When appending to existing HTML files, the FileWriter SHALL preserve all content before the comment marker
7. <a name="4.7"></a>The FileWriter SHALL insert new HTML fragments immediately before the marker, then re-add the marker
8. <a name="4.8"></a>If the existing HTML file is missing the comment marker, the FileWriter SHALL return an error describing the issue
9. <a name="4.9"></a>The comment marker replacement SHALL preserve the HTML document structure and validity

### 5. Writer Configuration API

**User Story:** As a developer, I want a clean and explicit API to enable append mode, so that my code clearly expresses the intent to append rather than replace.

**Acceptance Criteria:**

1. <a name="5.1"></a>The system SHALL provide a `WithAppendMode()` function that returns a `FileWriterOption`
2. <a name="5.2"></a>The `WithAppendMode()` option SHALL be usable with `NewFileWriterWithOptions()` constructor
3. <a name="5.3"></a>The append mode SHALL default to `false` (replace mode) if not explicitly enabled
4. <a name="5.4"></a>The system SHALL provide a `WithPermissions(os.FileMode)` option to override default file permissions
5. <a name="5.5"></a>The API SHALL follow v2's functional options pattern consistently

### 6. Error Handling and Safety

**User Story:** As a developer, I want comprehensive error handling for append operations, so that I can debug issues and prevent data corruption.

**Acceptance Criteria:**

1. <a name="6.1"></a>The FileWriter SHALL wrap all errors with `WriteError` that includes writer name, format, and underlying cause
2. <a name="6.2"></a>Format mismatch errors SHALL include both the expected format and the actual file extension
3. <a name="6.3"></a>HTML comment marker errors SHALL include the file path and describe what was expected
4. <a name="6.4"></a>File I/O errors SHALL include the full file path and the operation that failed
5. <a name="6.5"></a>The FileWriter SHALL use `sync.Mutex` to serialize write operations within a single FileWriter instance
6. <a name="6.6"></a>Thread-safety guarantees SHALL apply only to concurrent goroutines using the same FileWriter instance, not across separate processes or FileWriter instances
7. <a name="6.7"></a>HTML comment marker replacement SHALL use write-to-temp-and-rename pattern to ensure atomicity
8. <a name="6.8"></a>Temporary files for atomic updates SHALL be created in the same directory as the target file with pattern `{filename}.tmp.{random}`
9. <a name="6.9"></a>On successful write, the temporary file SHALL be renamed to the target using `os.Rename()` for atomic replacement
10. <a name="6.10"></a>If any error occurs during HTML marker replacement, the original file SHALL remain unchanged

### 7. Multi-Section Document Support

**User Story:** As a developer, I want to append multi-section documents to existing files, so that I can build complex reports incrementally using v2's builder pattern.

**Acceptance Criteria:**

1. <a name="7.1"></a>The FileWriter SHALL support appending documents that contain multiple sections (created via builder pattern)
2. <a name="7.2"></a>When appending a multi-section document, the FileWriter SHALL append all sections in order
3. <a name="7.3"></a>For HTML format with multiple sections, the FileWriter SHALL insert all new sections before the comment marker
4. <a name="7.4"></a>The FileWriter SHALL NOT merge or deduplicate sections from the existing file with new sections
5. <a name="7.5"></a>Section boundaries SHALL be preserved using format-appropriate separators (newlines for text formats, marker positioning for HTML)

### 8. Documentation and Examples

**User Story:** As a developer, I want clear documentation and examples for the append feature, so that I can understand when and how to use it effectively.

**Acceptance Criteria:**

1. <a name="8.1"></a>The package documentation SHALL include examples demonstrating append mode for at least three formats (table, HTML, JSON)
2. <a name="8.2"></a>The documentation SHALL clearly explain which formats support clean appending and which produce concatenated output
3. <a name="8.3"></a>The documentation SHALL explain that JSON/YAML byte-level appending produces concatenated output (e.g., NDJSON-style logging), not merged data structures
4. <a name="8.4"></a>The documentation SHALL explain the HTML comment marker system and its requirements
5. <a name="8.5"></a>Code examples SHALL demonstrate error handling for format mismatches and missing files
6. <a name="8.6"></a>The documentation SHALL state the UTF-8 encoding requirement for all files

### 9. Testing Requirements

**User Story:** As a maintainer, I want comprehensive test coverage for append functionality, so that the feature remains reliable across formats and edge cases.

**Acceptance Criteria:**

1. <a name="9.1"></a>The test suite SHALL include tests for append mode with all supported formats
2. <a name="9.2"></a>The test suite SHALL verify that appending to non-existent files creates new files
3. <a name="9.3"></a>The test suite SHALL verify that format mismatch detection works correctly
4. <a name="9.4"></a>The test suite SHALL test HTML comment marker replacement with valid and invalid files
5. <a name="9.5"></a>The test suite SHALL verify thread-safety of append operations with concurrent goroutines using the same FileWriter instance
6. <a name="9.6"></a>The test suite SHALL test error handling for permission errors and I/O failures
7. <a name="9.7"></a>The test suite SHALL verify that multi-section documents append correctly
8. <a name="9.8"></a>The test suite SHALL verify CSV header handling when appending to existing files
9. <a name="9.9"></a>The test suite SHALL verify atomic write behavior for HTML marker replacement (crash safety)
10. <a name="9.10"></a>The test suite SHALL verify append behavior on Linux, macOS, and Windows platforms
11. <a name="9.11"></a>The test suite SHALL verify S3Writer append mode with both existing and non-existent objects
12. <a name="9.12"></a>The test suite SHALL verify S3Writer ETag-based conflict detection when concurrent modifications occur
13. <a name="9.13"></a>The test suite SHALL verify S3Writer size validation rejects objects exceeding the configured maximum

### 10. S3Writer Append Mode Support

**User Story:** As a developer, I want to append to S3 objects for infrequent logging scenarios, so that I can use the same append pattern for both local files and S3 storage.

**Acceptance Criteria:**

1. <a name="10.1"></a>The S3Writer SHALL support append mode using download-modify-upload pattern
2. <a name="10.2"></a>When append mode is enabled and the S3 object does not exist, the S3Writer SHALL create a new object with initial content
3. <a name="10.3"></a>When append mode is enabled and the S3 object exists, the S3Writer SHALL download the existing object, append new content, and upload the modified object
4. <a name="10.4"></a>The S3Writer SHALL use ETag-based conditional updates to detect concurrent modifications during upload
5. <a name="10.5"></a>If the ETag has changed between download and upload (indicating concurrent modification), the S3Writer SHALL return an error
6. <a name="10.6"></a>The S3Writer SHALL validate that object size does not exceed a configurable maximum (default 100MB) before attempting append
7. <a name="10.7"></a>The S3Writer SHALL provide a `WithMaxAppendSize(int64)` option to configure the maximum object size for append operations
8. <a name="10.8"></a>The documentation SHALL clearly warn that S3 append mode uses download-modify-upload and is not atomic
9. <a name="10.9"></a>The documentation SHALL explain that S3 append mode is designed for infrequent updates to small objects (e.g., sporadic NDJSON logging)
10. <a name="10.10"></a>The documentation SHALL recommend enabling S3 versioning for data protection when using append mode

### 11. Backward Compatibility Considerations

**User Story:** As a v1 user migrating to v2, I want to understand how append mode differs between versions, so that I can migrate my code successfully.

**Acceptance Criteria:**

1. <a name="11.1"></a>The migration guide SHALL document that v1's `ShouldAppend` boolean is replaced by v2's `WithAppendMode()` option
2. <a name="11.2"></a>The migration guide SHALL explain that append behavior is now configured on the Writer, not the Settings
3. <a name="11.3"></a>The migration guide SHALL note that v2 does not support v1's `OutputFileFormat` concept - format is determined by the renderer, not the writer
4. <a name="11.4"></a>The migration guide SHALL document that v2 uses HTML comment marker `<!-- go-output-append -->` instead of v1's `<div id='end'></div>` placeholder (breaking change)
5. <a name="11.5"></a>The migration guide SHALL provide code examples showing v1 and v2 equivalents side-by-side
6. <a name="11.6"></a>The documentation SHALL acknowledge that v2's append mode matches v1's behavior for simple byte-level appending
7. <a name="11.7"></a>The migration guide SHALL document that v2 adds S3Writer append mode support via download-modify-upload pattern
