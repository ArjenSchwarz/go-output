## Unreleased

### Fixed
- **S3 Output Panic Prevention** - Fixed panic in `PrintByteSlice` when S3Output has Bucket set but S3Client is nil
  - Added validation guard clause to return clear error message instead of panicking
  - Error message guides users to use `SetS3Bucket()` for proper S3 configuration
  - Added regression test to prevent future occurrences
- **S3Writer Append Size Validation** - Fixed bug where append operations could exceed maxAppendSize limit
  - Now validates new data size before making API calls
  - Validates combined size (existing + new data) to prevent memory exhaustion
  - Provides specific error messages for different size limit violations
  - Prevents operations that would exceed the intended size guard
- **Draw.io CSV Output Buffer Flushing** - Fixed issue where `drawio.CreateCSV` function did not flush the underlying `bufio.Writer`, potentially causing data loss when writing to files with small datasets
## 2.6.0 / 2025-11-07

### Added
- **StderrWriter** - New writer for outputting to standard error stream
  - Follows same architecture as StdoutWriter for API consistency
  - Thread-safe concurrent operations with mutex protection
  - Context cancellation support for graceful termination

## 2.5.0 / 2025-10-30

### Breaking Changes
- **Format Variables Converted to Functions** - All format variables (`JSON`, `YAML`, `HTML`, etc.) are now functions that return fresh `Format` instances:
  - **Required Change**: Add parentheses `()` to all format references: `output.JSON` → `output.JSON()`
  - **Affected APIs**: `WithFormat()`, `WithFormats()`, all format variable assignments
  - **Why**: Prevents race conditions by ensuring each usage gets an independent renderer instance
  - **Benefit**: Enables parallel test execution with `t.Parallel()` without data races
  - See v2/docs/MIGRATION.md section "Migration from v2.4.x to v2.5.0" for detailed migration steps

### Changed
- Converted global Format variables to constructor functions for thread safety:
  - Core formats: `JSON()`, `YAML()`, `CSV()`, `HTML()`, `HTMLFragment()`, `Table()`, `Markdown()`, `DOT()`, `Mermaid()`, `DrawIO()`
  - Table styles: `TableDefault()`, `TableBold()`, `TableColoredBright()`, `TableLight()`, `TableRounded()`
  - Each function call returns a fresh Format with its own renderer instance
- Updated all internal code, tests, examples, and documentation to use new format functions

### Fixed
- **Race Conditions in Parallel Tests** - Applications using go-output can now safely run tests in parallel without encountering "concurrent map write" errors
- **Shared Mutable State** - Eliminated the last remnant of global mutable state by ensuring renderer instances are never shared between goroutines

## 2.4.0 / 2025-10-26

### Breaking Changes
- **Pipeline API Removed** - Replaced with more flexible per-content transformations system:
  - Use `WithTransformations()` on individual content items instead of `doc.Pipeline()`
  - Removed Pipeline struct and all related methods
  - See v2/docs/PIPELINE_MIGRATION.md for migration guidance

### Added
- **Per-Content Transformations** - Apply transformations to individual content items at creation time:
  - `WithTransformations()` for tables, `WithTextTransformations()` for text, `WithRawTransformations()` for raw content, `WithSectionTransformations()` for sections
  - Filter, sort, limit, and group operations can now differ between tables in the same document
  - Complete integration across all renderers with thread-safe operation
- **File Append Mode** - Append content to existing files instead of replacing:
  - `WithAppendMode()` option for FileWriter supporting HTML, CSV, and byte-level formats
  - HTML uses marker-based insertion (`<!-- go-output-append -->`) with atomic writes for crash safety
  - CSV automatically strips duplicate headers when appending
  - `WithPermissions()` for custom file permissions (default 0644)
  - `WithDisallowUnsafeAppend()` to prevent JSON/YAML appends
- **S3 Append Support** - Extend append functionality to S3 objects:
  - `WithS3AppendMode()` option using download-modify-upload pattern
  - `WithMaxAppendSize()` to limit append operations (default 100MB)
  - ETag-based optimistic locking for concurrent modification detection
  - Format-aware data combining (HTML markers, CSV header stripping)
- **HTML Template System** - Generate complete HTML documents with responsive styling:
  - Three built-in templates: `DefaultHTMLTemplate` (responsive), `MinimalHTMLTemplate` (no styling), `MermaidHTMLTemplate` (diagram-optimized)
  - Mobile-first responsive CSS with WCAG AA compliant colors
  - Customizable via `HTMLTemplate` struct (title, CSS, meta tags, head/body injection)
  - Automatic template wrapping with fragment mode for append operations

### Changed
- **Code Quality** - Modernized to Go 1.24+ patterns using `maps.Copy()` and standard library utilities
- **Clone Implementation** - Consolidated `TableContent.Clone()` with complete field coverage
- **Testing** - All tests use map-based table-driven patterns per Go 2025 best practices

## 2.3.3 / 2025-10-20

### Added
- **Mermaid Chart Rendering** - HTML and Markdown renderers now support ChartContent with proper format-specific rendering (HTML uses `<pre class="mermaid">` with CDN injection, Markdown uses code fences for GitHub/GitLab compatibility)

## 2.3.2 / 2025-10-17

### Fixed
- **S3Writer AWS SDK v2 Type Compatibility** - Achieved true type compatibility with AWS SDK v2 by directly importing and using actual SDK types (`s3.PutObjectInput`, `s3.PutObjectOutput`, `s3.Options`) instead of custom mirror types. Users can now pass `*s3.Client` directly to `NewS3Writer()` without any adapter code or type conversion.

## 2.3.1 / 2025-10-17

### Fixed
- **S3Writer Interface Alignment** - Updated S3Writer interface signatures to match AWS SDK v2 patterns (pointer fields and functional options).

## 2.3.0 / 2025-10-17

### Added
- **AWS Icons Package (v2/icons)**
  - Core AWS shape functionality with embedded aws.json data for 600+ AWS services
  - `GetAWSShape()` function with proper error handling for Draw.io style retrieval
  - `AllAWSGroups()` for discovering available AWS service categories
  - `AWSShapesInGroup()` for listing shapes in specific groups
  - `HasAWSShape()` convenience function for shape existence checking
  - Thread-safe concurrent access with O(1) lookup performance
  - Migration compatibility with v1 drawio.GetAWSShape() function
  - Package-level documentation with Draw.io integration examples
  - Performance testing validating acceptable memory footprint (~750KB-1MB)
- **Inline Styling Functions**
  - Stateless inline styling functions for ANSI terminal colors: `StyleWarning()`, `StylePositive()`, `StyleNegative()`, `StyleInfo()`, `StyleBold()`
  - Conditional styling variants with `*If` suffix for conditional formatting
  - Thread-safe functions using fatih/color library with automatic color enablement
- **Table Max Column Width Support**
  - `TableWithMaxColumnWidth()` and `TableWithStyleAndMaxColumnWidth()` format constructors
  - Automatic text wrapping within cells for terminal output with limited horizontal space
- **Format-Aware Array Handling**
  - Automatic array rendering in table cells as newline-separated values
  - Markdown format renders arrays with `<br/>` tags for GitHub/GitLab compatibility
  - JSON/YAML preserve native array structure

### Changed
- **API Naming Consistency**
  - Renamed `WithExpanded()` to `WithCollapsibleExpanded()` for consistency (backward compatible via deprecated wrapper)
- **Code Quality Improvements**
  - Eliminated ~400 lines of duplicated logic in JSON/YAML renderers through shared helper extraction
  - Merged pipeline execution methods into unified implementation, reducing codebase by 129 lines

## 2.2.0 / 2025-08-27

### Added
- **Data Transformation Pipeline System**
  - Complete Pipeline API with Filter, Sort, Limit, GroupBy, and AddColumn operations
  - Format-aware transformations that adapt behavior based on output format
  - Dual transformer system supporting both data and byte-level transformations
  - Transformation statistics collection with operation-level metrics
  - Comprehensive error handling with detailed operation context
- **Development Tooling & Automation**
  - Comprehensive Makefile with testing, linting, and code quality targets
  - Integration test separation with `INTEGRATION=1` environment variable support
  - Test coverage reporting with HTML output generation
  - Automated code modernization support with `modernize` tool integration

### Changed
- **Testing Infrastructure**
  - Modernized 47 test files to use map-based table tests and Go 1.24+ features
  - Split large test files into focused modules under 800 lines each
  - Updated all benchmarks to use new `b.Loop()` pattern

### Improved
- **Code Quality & Maintenance**
  - Enhanced v2 documentation with testing guidelines and development workflows
  - Applied modern Go idioms throughout codebase


## 2.1.3 / 2025-08-04

### Fixed
- Enhanced markdown table cell escaping to prevent formatting issues
  - Now escapes pipes (|), asterisks (*), underscores (_), backticks (`), and square brackets ([])
  - Maintains table structure integrity while preventing unintended markdown formatting
  - Replaces newlines with `<br>` tags for proper table cell display

## 2.1.1 / 2025-08-01

### Added
- **Code Fence Support for Collapsible Fields**
  - New `WithCodeFences(language string)` option for wrapping collapsible details in syntax-highlighted code blocks
  - `WithoutCodeFences()` option to explicitly disable code fence wrapping
  - Support for language-specific syntax highlighting (e.g., "json", "yaml", "go", "bash")
  - Proper newline preservation in code fences without HTML escaping
  - Works in both HTML renderer (using `<pre><code class="language-{lang}">`) and Markdown renderer (using ``` code fences)
  - Comprehensive test suite covering string, array, and map content types with code fence wrapping
  - Example application demonstrating code review results, configuration files, API responses, and error logs with code highlighting

### Fixed
- Improved Markdown escaping logic to be more selective and produce more legible output
- Fixed overly aggressive escaping in markdown table cells - now only escapes pipes and handles newlines
- Added dedicated HTML content escaping for content inside `<details>` and `<summary>` tags where GitHub processes markdown
- Fixed escaping of default placeholders like `[no summary]` which are literal strings, not markdown syntax

## 2.1.0 / 2025-07-28

### Added
- **Expandable Sections Feature**
  - Complete collapsible content system with support across all output formats (JSON, YAML, CSV, HTML, Table, Markdown, DOT, Mermaid, Draw.io)
  - CollapsibleValue interface with Summary(), Details(), IsExpanded(), and FormatHint() methods
  - CollapsibleSection interface for hierarchical nesting support (up to 3 levels)
  - Pre-built formatter functions for common patterns (ErrorListFormatter, FilePathFormatter, JSONFormatter)
  - Global expansion control with configurable renderer settings and `--expand` flag support
  - Comprehensive integration tests covering real-world scenarios including GitHub PR comments, API responses, and terminal output
  - Production-ready example applications for GitHub PR comments, terminal analysis, and CSV export
- **Table Renderer Enhancements**
  - CollapsibleValue detection and handling with proper detail formatting and 2-space indentation
  - Configurable expansion indicators with graceful fallback handling
  - Support for multiple detail data types (strings, arrays, complex objects)
  - Table style lookup map replacing large switch statement for improved maintainability

### Changed
- Enhanced table renderer to integrate with collapsible infrastructure while maintaining backward compatibility
- Improved code quality with reduced cyclomatic complexity in table styling
- Updated development workflow with CLAUDE.md configuration for v2 project guidance

## 2.0.0 / 2025-07-22

### Added
- **FileWriter absolute path support**
  - New `WithAbsolutePaths()` option to allow writing to absolute file paths
  - Maintains security with directory traversal protection even when absolute paths are enabled
  - Conditional validation that respects the absolute path configuration
  - Comprehensive test coverage for absolute path functionality and security edge cases
- **Complete v2 API documentation and examples**
  - Comprehensive API documentation with detailed interface reference for all public types and methods
  - Working examples for basic usage, multiple formats, key ordering, error handling, and concurrent operations
  - Chart generation examples including Gantt charts, pie charts, and flow diagrams with Mermaid and Draw.io support
  - Migration examples demonstrating patterns for transitioning from v1 to v2 architecture
  - Mixed content document examples showing hierarchical content organization
  - Progress tracking examples with visual feedback and format-aware behavior
  - Performance examples for large datasets and streaming operations
- **AST-based migration tool** for automated code conversion from v1 to v2
  - Comprehensive pattern recognition for all v1 usage patterns
  - Advanced transformation engine with type-aware code generation
  - Support for complex scenarios including multiple outputs and custom settings
  - Extensive test coverage including real-world code examples
  - Command-line interface with dry-run and verbose modes
- **Migration documentation suite**
  - `BREAKING_CHANGES.md`: Detailed before/after examples for all breaking changes
  - `MIGRATION.md`: Comprehensive migration guide with patterns and examples
  - `MIGRATION_EXAMPLES.md`: Practical migration examples for common use cases
  - `MIGRATION_QUICK_REFERENCE.md`: Quick reference table for common replacements
- Enhanced error handling system with detailed context and source tracking
- Extended RenderError with renderer type, operation, and context information
- New WriterError type for write operation failures with detailed context
- Enhanced MultiError with source mapping and error aggregation capabilities
- StructuredError type for machine-readable error analysis and programmatic handling
- Error source tracking throughout the rendering pipeline with component identification
- ToStructuredError() function for converting any error type to structured format
- Comprehensive error context preservation across all rendering operations
- Enhanced error message formatting with structured information display
- Complete test suite for enhanced error reporting and context preservation
- Transform system implementation with Transformer interface and priority-based pipeline
- Format-aware transformation capabilities with enhanced format detection
- FormatDetector for identifying format capabilities (text-based, structured, tabular, graph, color support, emoji support)
- FormatAwareTransformer wrapper providing enhanced format detection and data integrity preservation
- Enhanced transformer implementations:
  - EnhancedEmojiTransformer with format-specific emoji substitutions (HTML entities, conservative markdown)
  - EnhancedColorTransformer with color support detection
  - EnhancedSortTransformer with tabular format detection
- DataIntegrityValidator ensuring transformers don't modify original document data
- Comprehensive test coverage for all transform system components including data integrity and concurrent operations
- Initial v2.0 module structure with clean architecture and no global state
- Core Content interface with encoding.TextAppender and encoding.BinaryAppender support
- Document struct for holding content collections with thread-safe operations
- Builder pattern implementation with fluent API for document construction
- v2 schema system with key order preservation functionality
- Schema and Field structures with explicit key ordering
- Functional options pattern for table configuration (WithSchema, WithKeys, WithAutoSchema)
- Table options system with automatic schema detection
- TableContent implementation with preserved key ordering and encoding interface support
- TextContent and TextStyle for unstructured text with styling options (bold, italic, color, size, header)
- Text and Header builder methods with functional options pattern
- RawContent for format-specific content (HTML, CSS, JSON, XML, etc.) with format validation
- SectionContent for hierarchical document structure with nested content and indentation
- Section builder method with function-based content definition and level support
- Comprehensive functional options for all content types (text, raw, section)
- Builder error handling system with HasErrors() and Errors() methods for tracking build failures
- Thread-safe error collection during document construction
- golangci-lint configuration with interface{} to any conversion
- CLAUDE.md development guide for v2 architecture
- Comprehensive test coverage for all content types including thread-safety and concurrent operations
- Integration tests demonstrating mixed content scenarios and key order preservation
- Error handling tests for builder pattern validation and thread safety
- Design and requirements documentation for complete v2 redesign
- Task tracking system for incremental implementation
- Renderer interface with Format(), Render(), RenderTo(), and SupportsStreaming() methods
- Format struct for output format configuration with Name, Renderer, and Options fields
- Built-in format constants for all v1 formats: JSON, YAML, CSV, HTML, Table, Markdown, DOT, Mermaid, DrawIO
- BaseRenderer struct with common functionality for thread-safe, context-aware rendering
- Context cancellation support for all rendering operations
- Memory-efficient rendering patterns using bytes.Buffer and streaming approaches
- Comprehensive test suite covering interface compliance, context cancellation, error handling, and thread safety
- Streaming support categorization (JSON, YAML, CSV, HTML, Table, Markdown support streaming; DOT, Mermaid, DrawIO do not)
- Error handling with proper error wrapping and validation for nil inputs
- Thread-safe concurrent rendering operations using sync.RWMutex
- Complete renderer implementations for all supported formats:
  - JSON/YAML renderer with format-aware serialization and streaming support
  - CSV renderer with configurable headers and proper escaping
  - HTML renderer with semantic structure and table of contents support
  - Table renderer with multiple style options (Default, Bold, ColoredBright, Light, Rounded)
  - Markdown renderer with nested section support and table of contents generation
  - Graph renderers (DOT, Mermaid, DrawIO) with format-specific output
- Complete chart content system with support for Gantt and pie charts
- Draw.io CSV renderer with full header configuration and layout options
- Graph content system with edge-based relationship modeling
- Builder methods for charts, graphs, and Draw.io diagrams
- Support for hierarchical diagrams with parent-child relationships
- Connection definitions with from/to mappings and styling for Draw.io
- AWS service shape integration and manual positioning support
- Comprehensive renderer test suite with format-specific validation and edge case coverage
- Comprehensive test coverage for chart, graph, and Draw.io functionality
- Writer interface system with Write() method for flexible output destinations
- FileWriter implementation with security features including directory confinement and path validation
- StdoutWriter for console output with streaming support
- MultiWriter for writing to multiple destinations simultaneously
- S3Writer for cloud storage integration with AWS SDK
- Comprehensive writer test suite with security validation, concurrency testing, and error handling
- Core Output system implementation with NewOutput() factory and fluent configuration API
- Progress system implementation with PrettyProgress, TextProgress, and NoOpProgress implementations
- Progress interface with v1 feature parity including color support, TTY detection, and context cancellation
- Output.Render() method with concurrent format processing and progress tracking integration
- OutputOption pattern for configurable Output instances (WithFormat, WithWriter, WithProgress, etc.)
- Comprehensive output system test suite covering progress integration, error handling, and thread safety
- Enhanced design documentation with complete progress system specifications and v1 compatibility details
- Debug tracing system with configurable levels (TRACE, INFO, WARN, ERROR, OFF)
- Panic recovery and error wrapping capabilities
- Comprehensive error handling with context and stack traces
- Safe execution wrappers for operations with panic recovery
- Global debug tracer functionality for cross-package debugging

### Changed
- Updated v2 redesign tasks to mark migration tool development as completed (tasks 13.1-13.2)
- Complete architectural redesign eliminating all global variables
- Minimum Go version requirement updated to 1.24
- Updated exported function names for better API consistency (generateID → GenerateID, addContent → AddContent)
- Replaced interface{} with any type throughout codebase per Go 1.24 best practices
- Enhanced Content interface documentation with proper comments
- Updated task tracking to mark content system implementation as completed (tasks 3.1-3.4)
- Refactored if-else chain to switch statement for better code quality and linting compliance
- Enhanced Builder pattern with improved error handling instead of silent failures
- Updated task tracking to mark builder pattern methods as completed (tasks 4.1-4.3)
- Updated task tracking to mark rendering pipeline foundation as completed (tasks 5.1-5.2)
- Updated task tracking to mark renderer implementations as completed (tasks 6.1-6.9)
- Enhanced schema system with improved field type detection
- Agent task documentation with completed v2 implementation milestones
- Output system with improved error handling and debug capabilities
- Progress tracking with enhanced debugging support
- Transformer system with better error management and tracing

### Added
- **Comprehensive testing suite for v2 architecture**
  - Benchmark tests for performance validation across all components
  - Builder methods test coverage for all convenience methods and chaining
  - Integration tests for complete workflow scenarios and concurrent operations
  - Raw options, renderer constructors, section options, text options, and validation tests
  - Memory allocation benchmarks and performance comparisons
  - Thread-safety testing for concurrent document building

### Fixed
- **Windows build compatibility**: Fixed progress system to work on Windows by using build constraints for Unix-only signals
  - Split signal handling into platform-specific files (`progress_pretty_unix.go` and `progress_pretty_windows.go`) in both v1 and v2
  - Removed `syscall.SIGWINCH` dependency on Windows where the signal doesn't exist
  - Fixed typo in v2 format detector ("detetcor" → "detector")
- Resolved all linting issues identified by golangci-lint including gocritic ifElseChain warnings
- Ensured proper code formatting and Go conventions compliance

1.5.1 / 2025-07-18

  * Add fix for bug where AddToBuffer and different output formats didn't work nice together.

1.5.0 / 2025-07-18
==================

  * Add comprehensive progress indicator system with visual progress bars for terminal output
  * Implement progress factory with format-aware behavior (visual for table/markdown/HTML, no-op for JSON/YAML/CSV/DOT)
  * Add progress context support for cancellation and proper cleanup
  * Fix issue where not all tables were shown when writing to file
  * Improve error handling for DrawIO shape tests and file operations
  * Refactor HTML generation for better performance (return bytes instead of strings)
  * Simplify number formatting logic for consistent numeric value handling
  * Add extensive test coverage for helper functions and progress indicators
  * Expand documentation with progress examples and enhanced Mermaid package README
  * Add gofmt checks and improve code quality across modules
  * Enhance GitHub Actions workflow efficiency

1.4.0 / 2023-11-14
==================

  * Update Go version and support file as output and CLI at the same time

1.3.0 / 2023-06-17
==================

  * Add convenience function AddContents
  * Add license that was somehow missing
  * Update go version
  * Add support for YAML output format
