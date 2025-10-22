## Unreleased

### Added
- **Thread Safety & Performance Testing Suite (TDD)** - Complete testing infrastructure for concurrent operations and performance validation of per-content transformations:
  - Thread safety tests covering concurrent rendering of same document with multiple goroutines, concurrent rendering of different content with shared operations, cloned content independence verification, operation safety during concurrent execution, and concurrent cloning operations
  - All tests pass with `-race` detector enabled confirming zero data races
  - `ValidateStatelessOperation()` testing utility for detecting non-deterministic operations by applying operations twice to cloned content and comparing results with `reflect.DeepEqual()`
  - Comprehensive test suite for statelessness validation covering detection of stateful operations (call counters, mutable state), verification of deterministic operations (filter, sort, limit), and usage examples
  - Performance benchmarks establishing baseline metrics for 100 content items with 10 transformations each (~21ms per iteration), 1000-record tables with transformations (~1.3ms), transformation storage memory overhead (~348ns, 56B, 3 allocs), transformation execution time breakdown, cloning overhead (~19.7μs per 100-record table), and multiple clones in transformation chains
  - Memory allocation tracking with `b.ReportAllocs()` for identifying optimization opportunities
  - All benchmarks meet performance target: system handles 100 items × 10 transformations without degradation
  - Test files total 1,196 lines covering concurrent operations, stateless validation, and performance characteristics
- **Advanced Error Handling Tests (TDD)** - Comprehensive test suite for validation and context cancellation error handling in per-content transformations:
  - Validation error tests covering configuration errors (nil predicates, negative limits, empty column names, invalid groupby operations)
  - Data-dependent validation error tests for missing columns and empty operations
  - Error message context tests verifying content ID and operation index inclusion in all error messages
  - Fail-fast behavior tests confirming rendering stops immediately on validation errors
  - Context cancellation detection tests for pre-cancelled contexts and deadline exceeded scenarios
  - Context propagation tests verifying context.Canceled and context.DeadlineExceeded proper wrapping
  - Context cancellation error message tests ensuring proper error context with content ID
  - Rendering stop tests confirming no operations execute when context is cancelled
  - All tests follow TDD red-green pattern with proper test organization using map-based table tests
- **Renderer Integration for Per-Content Transformations (TDD)** - Integrated transformation execution across all renderers following Test-Driven Development:
  - Updated JSONRenderer and YAMLRenderer via shared `renderDocumentGeneric()` function to call `applyContentTransformations()` before rendering each content item
  - Updated HTMLRenderer via `baseRenderer.renderTransformedDocument()` to apply transformations in the base rendering pipeline
  - Updated CSVRenderer, TableRenderer, and MarkdownRenderer with direct transformation calls in their custom rendering loops
  - All renderers properly preserve document immutability by cloning content before applying transformations
  - Fail-fast error handling propagates transformation errors immediately with detailed context
  - Context cancellation properly flows through all renderer implementations for responsive cancellation
  - Comprehensive test suite with 850+ lines of tests covering all renderers:
    - JSONRenderer: 5 test functions covering transformation integration, fail-fast errors, context cancellation, immutability, and mixed content scenarios
    - YAMLRenderer: 4 test functions covering filter operations, sort/limit chains, fail-fast behavior, and context handling
    - CSVRenderer, TableRenderer, MarkdownRenderer, HTMLRenderer: Integration tests verifying transformation application
  - All tests verify transformed output correctness (filtered records, sorted data, limited results)
  - All renderers support mixing transformed and non-transformed content in the same document
- **Per-Content Transformations Specification** - Complete requirements, design, and implementation specification for attaching transformations directly to individual content items (tables, text, sections) at creation time, enabling different operations to be applied to different content in the same document.
- **Content Interface Transformation Support** - Extended Content interface with `Clone()` and `GetTransformations()` methods to enable per-content transformation capabilities across all content types (TableContent, TextContent, RawContent, SectionContent, DefaultCollapsibleSection, GraphContent, ChartContent, DrawIOContent)
- **Transformation Execution Helper (TDD)** - Implemented core transformation execution logic following Test-Driven Development:
  - Added `applyContentTransformations()` helper function in v2/renderer.go for executing per-content transformations during rendering
  - Clones content once at start to preserve immutability of original document data
  - Applies transformations sequentially in user-specified order
  - Validates each operation configuration before execution via `Validate()` method
  - Checks context cancellation before each operation for responsive cancellation
  - Provides detailed error messages including content ID, operation index (zero-based), and operation name
  - Implements fail-fast error handling (stops immediately on first transformation error)
  - Comprehensive test suite with 9 test functions covering no-transformations, sequential execution, validation, context cancellation, error messages, immutability preservation, lazy execution, multiple transformations, and fail-fast behavior
  - All tests verify transformations execute during rendering only (not during Build())
- **TableContent Transformation Storage (TDD)** - Implemented per-content transformations for TableContent following Test-Driven Development:
  - Added `transformations []Operation` field to TableContent struct for storing operation references
  - Created `WithTransformations(ops ...Operation)` TableOption function supporting variadic arguments and method chaining
  - Updated `GetTransformations()` to return transformations slice (empty slice instead of nil when no transformations exist)
  - Enhanced `Clone()` method to preserve transformations with shallow copy of operation references (operations are shared, not cloned)
  - Comprehensive test suite covering single/multiple/zero transformations, order preservation, cloning behavior, and operation instance sharing
  - Updated tableConfig struct to include transformations field for functional options pattern
  - Modified NewTableContent to apply transformations from configuration
- **TextContent Transformation Storage (TDD)** - Implemented per-content transformations for TextContent following Test-Driven Development:
  - Added `transformations []Operation` field to TextContent struct
  - Created `WithTextTransformations(ops ...Operation)` TextOption function (type-specific naming required due to Go's no-overloading constraint)
  - Updated `GetTransformations()` to return transformations slice (returns empty slice when no transformations)
  - Enhanced `Clone()` method to preserve transformations with shallow copy of operation instances
  - Comprehensive test suite with 13 subtests covering transformation storage, retrieval, cloning, and order preservation
  - Updated textConfig struct to include transformations field
- **RawContent Transformation Storage (TDD)** - Implemented per-content transformations for RawContent following Test-Driven Development:
  - Added `transformations []Operation` field to RawContent struct
  - Created `WithRawTransformations(ops ...Operation)` RawOption function
  - Updated `GetTransformations()` to return transformations slice
  - Enhanced `Clone()` method to preserve transformations with shallow copy
  - Comprehensive test suite with 9 subtests covering all transformation scenarios
  - Updated rawConfig struct to include transformations field
- **SectionContent Transformation Storage (TDD)** - Implemented per-content transformations for SectionContent following Test-Driven Development:
  - Added `transformations []Operation` field to SectionContent struct
  - Created `WithSectionTransformations(ops ...Operation)` SectionOption function
  - Updated `GetTransformations()` to return transformations slice
  - Enhanced `Clone()` method to preserve transformations AND correctly handle nested content deep cloning
  - Comprehensive test suite with 13 subtests including nested content cloning verification
  - Updated sectionConfig struct to include transformations field

### Changed
- **Clone Implementation Consolidation** - Moved TableContent.Clone() from transform_data.go to content.go with complete field coverage (id, title, records, schema) fixing incomplete implementation that was missing id and title fields
- **Code Modernization** - Updated map copying to use maps.Copy() instead of manual loops per Go 1.24+ best practices

### Added
- **HTML Template System Integration Testing (Phase 6)** - Complete integration test suite for HTML document rendering and thread safety validation
  - 30 comprehensive integration tests covering full document generation workflow
  - Mermaid chart integration tests (4): script injection order, fragment mode, multiple charts, XSS prevention
  - Template customization tests (9): custom title, CSS overrides, external stylesheets, meta tags, head/body injection, theme overrides, body class/attributes, combined customizations
  - Edge case tests (9): empty document, empty CSS, empty external stylesheets, missing optional fields, nil template, empty collections, all empty fields
  - Thread safety tests (4): concurrent rendering same template, template instance reuse, no shared mutable state, concurrent Mermaid chart rendering
  - All tests pass with `-race` flag verifying zero data races
  - Code modernized to Go 1.24+ `range` syntax for all for loops
  - Full test coverage of HTML rendering pipeline with 72.1% overall statement coverage
  - All code passes golangci-lint, go fmt, and modernize validation
- **HTML Template System Testing (Phase 3)** - Comprehensive test suite for HTML template wrapping functionality
  - `wrapInTemplate()` function implementation in `html_renderer.go` for generating complete HTML5 documents
  - Added `useTemplate` and `template` fields to `htmlRenderer` struct for template configuration
  - Integrated template wrapping into rendering pipeline after mermaid script injection
  - 20 comprehensive test functions covering all template features:
    - DOCTYPE declaration and HTML element structure
    - Meta tags (charset, viewport, description, author, custom)
    - CSS styling (embedded, external, theme overrides)
    - Body customization (class, attributes, extra content)
    - Head content injection
    - Fragment content injection and placement
    - Nil template fallback to DefaultHTMLTemplate
    - Edge cases with empty fields and HTML structure ordering
  - Full HTML escaping validation for XSS prevention in metadata fields
  - Proper element ordering verification (DOCTYPE → html → head → body)
  - All tests pass with proper use of map-based table-driven test patterns
  - Code passes golangci-lint, go fmt, and modernize validation
- **HTML Template System Responsive CSS (Phase 2)** - Implemented modern responsive CSS styling system for HTML document templates
  - `html_css.go` with two CSS constant variables: `defaultResponsiveCSS` and `mermaidOptimizedCSS`
  - Mobile-first responsive CSS using CSS custom properties for theming (colors, spacing, typography, layout)
  - WCAG AA compliant color contrast values for accessibility
  - System font stack (-apple-system, BlinkMacSystemFont, Segoe UI, Roboto) for optimal performance
  - Responsive table styling with mobile stacking pattern (`@media max-width: 480px`)
  - Desktop table layout with hover effects (`@media min-width: 481px`)
  - Support for all content types: tables, sections, details/summary collapsibles, text content, mermaid diagrams
  - Comprehensive tests verifying CSS structure: custom properties, breakpoints, table stacking, font stack, color contrast, mermaid styles
  - All code passes golangci-lint, go fmt, and full test suite
- **HTML Template System Core Implementation (Phase 1)** - Implemented foundation for full HTML document generation with responsive CSS templates
  - `HTMLTemplate` struct with 14 fields for metadata, styling, and customization
  - Three built-in template variants: `DefaultHTMLTemplate` (modern responsive), `MinimalHTMLTemplate` (no styling), `MermaidHTMLTemplate` (diagram-optimized)
  - Comprehensive godoc documentation with security warnings for unescaped content fields (CSS, HeadExtra, BodyExtra)
  - Package-level template variables for zero-allocation usage
  - Full test coverage with map-based table-driven tests for field defaults and custom overrides
  - CSS custom property support via `ThemeOverrides` field for easy theme customization
- **HTML Template System Specification** - Created comprehensive specification documents for full HTML document generation with responsive CSS templates
  - Requirements documentation outlining template structure, responsive design system, and operational modes
  - Design documentation detailing architecture, component layers, template wrapping, and CSS theming
  - Task breakdown for implementation planning and progress tracking
  - Decision log documenting key architectural decisions including string-based templates, embedded CSS, mobile-first design, and CSS custom properties
  - Initial idea document outlining the feature concept and motivation

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
