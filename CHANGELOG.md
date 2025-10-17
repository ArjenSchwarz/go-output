## [Unreleased]

### Added
- **Inline Styling Functions (v2.2.1)**
  - Stateless inline styling functions for ANSI terminal colors: `StyleWarning()`, `StylePositive()`, `StyleNegative()`, `StyleInfo()`, `StyleBold()`
  - Conditional styling variants with `*If` suffix: `StyleWarningIf()`, `StylePositiveIf()`, etc.
  - Thread-safe functions using fatih/color library with automatic color enablement
  - Can be used directly in table data and text content
  - Comprehensive test coverage with ANSI code validation
- **Table Max Column Width Support (v2.2.1)**
  - `TableWithMaxColumnWidth(width int)` format constructor for limiting column widths
  - `TableWithStyleAndMaxColumnWidth(style, width)` combining style and width configuration
  - `NewTableRendererWithStyleAndWidth()` renderer constructor
  - Automatic text wrapping within cells using go-pretty's `WidthMax` configuration
  - Particularly useful for terminal output with limited horizontal space
  - Comprehensive test suite validating width configuration and key order preservation
- **Format-Aware Array Handling (v2.2.1)**
  - Automatic array rendering in table cells as newline-separated values
  - Markdown format renders arrays with `<br/>` tags for GitHub/GitLab compatibility
  - JSON/YAML preserve native array structure
  - Support for `[]string` and `[]any` slice types
  - Empty arrays render as empty strings in table/markdown formats
  - Automatic markdown escaping for array elements

### Improved
- **Documentation (v2.2.1)**
  - Added inline styling functions section to API.md with usage examples and notes
  - Added table max column width documentation with format constructors and configuration
  - Added format-specific behavior section documenting array handling across formats
  - Updated MIGRATION.md with migration examples for inline styling, max column width, and array handling
  - Updated version to v2.2.1 in API documentation
  - Added comprehensive migration guidance showing v1 vs v2 approaches

### Changed
- **API Naming Consistency (v2)**
  - Renamed `WithExpanded()` to `WithCollapsibleExpanded()` for consistency with section options
  - Old `WithExpanded()` remains available as deprecated wrapper for backward compatibility
  - Affects collapsible value formatters and table field configuration

### Improved
- **Code Quality & Maintenance (v2)**
  - Eliminated ~400 lines of duplicated logic in JSON/YAML renderers through shared helper extraction
  - Merged pipeline execution methods (`ExecuteContext`/`ExecuteWithFormatContext`) into unified implementation
  - Reduced codebase by 129 net lines while improving maintainability
  - Added `.gitignore` to prevent binary file commits in examples

### Added
- **AWS Icons Package Performance Testing (v2/icons)**
  - Memory usage test validating package data stays within expected ~750KB-1MB range
  - Startup time test verifying fast data access (< 100ms)
  - Performance validation tests confirming acceptable memory footprint for embedded JSON data
- **Development Configuration**
  - Added `coverage.out` and `mem.out` to .gitignore for cleaner repository

### Added
- **AWS Icons Package Documentation and Examples (v2/icons)**
  - Comprehensive package-level documentation with Draw.io integration examples
  - Example functions demonstrating GetAWSShape basic usage and error handling
  - Examples for discovering icons using AllAWSGroups and AWSShapesInGroup helpers
  - Draw.io integration example showing placeholder-based dynamic icon assignment
  - Usage examples with proper godoc formatting and testable output

### Changed
- **Development Configuration**
  - Added golangci-lint to allowed Bash commands in Claude settings for automated code quality checks

### Added
- **AWS Icons Package Testing Suite (v2/icons)**
  - Integration tests for Draw.io workflow compatibility with AWS icons
  - Migration compatibility tests verifying v1/v2 consistency
  - Concurrent access tests for thread safety validation (race detector enabled)
  - Benchmark suite for performance validation (O(1) lookup verification)
  - Test coverage for placeholder patterns and multi-icon diagrams
  - Duplicate key handling tests ensuring last-value-wins behavior
  - Case sensitivity tests verifying exact matching requirements
- **AWS Icons Package Implementation (v2/icons)**
  - Core AWS shape functionality with embedded aws.json data
  - GetAWSShape() function with error handling for Draw.io style retrieval
  - AllAWSGroups() for discovering available service categories
  - AWSShapesInGroup() for listing shapes in specific groups
  - HasAWSShape() convenience function for shape existence checking
  - Comprehensive test suite with 100% coverage for all functions
  - Memory-efficient design with one-time JSON parsing at package initialization
- **Draw.io AWS Icons Support Specification**
  - Complete requirements specification for AWS icon integration in v2
  - Design documentation for icons package architecture and implementation strategy
  - Task breakdown with implementation steps for AWS shapes functionality
  - Decision log documenting design choices and rationale for icon support approach

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
