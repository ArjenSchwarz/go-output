## [Unreleased]

### Changed
- **Code Modernization with Go 1.24+ Features**
  - Modernized all benchmark tests to use Go 1.24's new `b.Loop()` pattern replacing `for i := 0; i < b.N; i++` loops
  - Updated all test loops to use modern `range` over integers syntax (e.g., `for i := range n` instead of `for i := 0; i < n; i++`)
  - Replaced manual slice searching with `slices.Contains` for improved readability
  - Simplified error handling and reduced code complexity across test suites
  - Removed unnecessary `b.ResetTimer()` calls now handled automatically by `b.Loop()`
  - Applied modernization consistently across 30+ test files including benchmarks, integration tests, and unit tests
  - Improved code maintainability and leveraged Go 1.24 performance optimizations

### Added
- **Code Cleanup and Tooling Planning**
  - Comprehensive feature planning documentation for code modernization using Go 1.24+ features
  - Requirements specification for applying modernize tool suggestions to leverage performance optimizations and modern idioms
  - Design documentation outlining phased approach for automated modernization across 118 identified improvements
  - Test organization strategy following Go 2025 best practices including map-based table tests and environment-based integration test separation
  - Makefile architecture design with comprehensive targets for test, lint, format, and development workflows
  - Task breakdown for implementing modernization including benchmarks conversion to new b.Loop() pattern
  - Decision log documenting scope limitation to v2 directory and modernization priorities
  - Research documentation on Go unit testing best practices for 2025 including b.Loop() benchmarks and integration test patterns

### Fixed
- **Progress Tracking Implementation**
  - Simplified noOpProgress to be a true no-op implementation by removing unused state fields
  - Eliminated theoretical race condition concerns by removing unnecessary field storage
  - Improved code clarity by making the no-op intent explicit

### Changed
- **Sort Operation Error Messages**
  - Enhanced sort validation error messages to include list of available columns
  - Added clarifying comment that missing columns in records are handled as nil values during comparison
  - Improved developer experience with more informative validation errors

### Fixed
- **Sort Operation Validation Enhancement**
  - Added validation to check that sort columns exist in the table data before attempting to sort
  - Sort operations now return a descriptive ValidationError when attempting to sort by non-existent columns
  - Prevents potential panics or undefined behavior when sorting with invalid column names
  - Re-enabled previously skipped integration test for invalid sort column error handling

### Changed
- **Code Quality Improvements**
  - Modernized test loops to use `range` over int in pipeline benchmark tests
  - Replaced manual slice searching with `slices.Contains` for improved readability in requirements validation tests
  - Added performance test constant for improved maintainability

### Added
- **Comprehensive Integration Testing and Requirements Validation**
  - End-to-end integration tests for complete transformation pipeline with multiple operations chained together (Filter → Sort → AddColumn → Limit)
  - Integration tests for complex aggregation pipeline with GroupBy, Count, Average, Min, Max operations
  - All output format integration testing (JSON, YAML, CSV, HTML, Table, Markdown) ensuring pipeline results render correctly
  - Concurrent pipeline execution tests validating thread-safe operations with multiple goroutines
  - Large dataset performance testing with 10,000 records demonstrating scalable pipeline processing
  - Schema evolution tests validating schema changes through multiple AddColumn operations
  - Requirements validation test suite ensuring all transformation pipeline requirements from design documentation are met
  - Tests confirm DataTransformer interface compliance, Pipeline API fluent interface, and all core operations functionality
  - Validated key order preservation, immutability guarantees, and context cancellation support
  - Error scenario testing covering invalid predicates, context cancellation, and operation timeout handling
  - Memory benchmark testing for allocation tracking and performance profiling
  - Marked integration testing and validation tasks (18.1, 18.2) as completed in transformation-pipeline feature

### Added
- **Data Transformation Pipeline Documentation and Examples**
  - Comprehensive API documentation for Pipeline system with detailed interface reference for all transformation operations
  - Complete pipeline documentation covering Filter, Sort, Limit, GroupBy, AddColumn operations with usage examples
  - Migration guide section for transitioning from byte transformers to data pipeline system with best practices
  - Format-aware transformation documentation explaining when to use data pipeline vs byte transformers
  - Performance comparison tables and optimization guidance for pipeline operations
  - Real-world pipeline transformation example demonstrating sales data analysis with filtering, calculations, sorting, and limiting
  - Complete pipeline example application with comprehensive transformation statistics and error handling
  - Migration patterns and checklist for systematic transition from manual data operations to pipeline API
  - Marked documentation and examples tasks (17.1, 17.2) as completed in transformation-pipeline feature

### Added
- **Pipeline Performance Benchmarking Suite**
  - Comprehensive benchmark suite for pipeline operations with varying data sizes (10, 100, 1000, 10000 records)
  - Filter operation benchmarks demonstrating performance across different dataset sizes
  - Sort operation benchmarks with 1, 2, and 3 sort keys showing multi-key sorting overhead
  - Aggregation benchmarks testing GroupBy performance with 10, 50, and 100 distinct groups
  - Complex pipeline chain benchmarks measuring combined filter, sort, and limit operations
  - AddColumn calculated field benchmarks for performance analysis of field transformations
  - Manual vs pipeline comparison benchmarks showing pipeline efficiency relative to manual data manipulation
  - Operation optimization benchmarks demonstrating effectiveness of filter-before-sort optimization
  - Memory allocation benchmarks with allocation reporting for performance profiling
  - Timeout and context cancellation benchmarks measuring overhead of resource management
  - Concurrent pipeline execution benchmarks validating thread-safe performance characteristics
  - GroupBy with multiple aggregate functions benchmarks comparing single vs multiple aggregates
  - Performance tests demonstrate acceptable overhead and validate optimization strategies
  - Marked performance benchmarking tasks (16.1, 16.2) as completed in transformation-pipeline feature

### Added
- **Format-Aware Transformation Support**
  - Implemented FormatAwareOperation interface extending Operation with format context support
  - Added ApplyWithFormat() method to all operation types (FilterOp, SortOp, LimitOp, GroupByOp, AddColumnOp) for format-aware transformation
  - Added CanTransform() method to check if operations apply to specific content and format combinations
  - Enhanced Pipeline with ExecuteWithFormat() and ExecuteWithFormatContext() methods for format-aware pipeline execution
  - Added format validation ensuring only valid formats (JSON, YAML, CSV, HTML, Table, Markdown, DOT, Mermaid, DrawIO) are processed
  - Operations now maintain backward compatibility through delegation to existing Apply() methods when format awareness isn't needed
  - Enhanced error handling with format context tracking in PipelineError for debugging transformation issues
  - Comprehensive test suite with 250+ lines covering format parameter passing, operation filtering, and backward compatibility
  - Marked format-aware transformation tasks (14.1, 14.2) as completed in transformation-pipeline feature

### Added
- **Comprehensive Backward Compatibility Test Suite**
  - Added backward compatibility integration tests verifying existing transformers work without modification
  - Added backward compatibility performance benchmarks to prevent performance degradation
  - Added comprehensive regression test suite covering existing functionality preservation
  - Tests ensure transformer interface methods remain unchanged and transformation configuration methods are preserved
  - Verified immutability guarantees are maintained and thread-safety is preserved across concurrent operations
  - Added error handling preservation tests including TransformError handling and context cancellation
  - Added API stability tests verifying all transformer and pipeline methods remain present
  - Comprehensive test coverage includes 1100+ lines of test code covering integration, performance, and regression scenarios
  - Marked backward compatibility tasks (13.1, 13.2) as completed in transformation-pipeline feature

### Added
- **Transformation Statistics Collection System**
  - Implemented comprehensive TransformStats collection during pipeline execution with input/output record counts, filtered count tracking, and duration measurements
  - Added GetTransformStats() method to Document for retrieving transformation statistics from document metadata
  - Enhanced metadata handling to properly preserve transformation statistics alongside original document metadata
  - Added comprehensive test suite for statistics collection covering basic metrics, operation timing, record processing counts, multiple table contents, and edge cases
  - Statistics persist across document operations and provide detailed operation-level metrics including duration and records processed
  - Added graceful handling of empty results and corrupted metadata scenarios
  - Marked performance monitoring and stats tasks (12.1, 12.2) as completed in transformation-pipeline feature

### Added
- **Pipeline Error Handling System**
  - Implemented comprehensive PipelineError type with detailed operation context including stage, input samples, and pipeline metadata
  - Added pipeline-specific error handling with support for operation context, stage tracking, and fail-fast behavior
  - Integrated PipelineError into ToStructuredError() converter for consistent error reporting across the system
  - Enhanced operation validation with detailed ValidationError messages for better debugging experience
  - Added comprehensive test coverage for pipeline error scenarios including type mismatches and validation failures
  - Marked error handling and validation tasks (11.1, 11.2) as completed in transformation-pipeline feature
- **Enhanced Operation Validation**
  - Improved validation error messages across all operations (Filter, Sort, Limit, GroupBy, AddColumn)
  - Added detailed field-level validation with specific error context for troubleshooting
  - Enhanced sort operation validation with column name and direction validation
  - Improved group by operation validation with aggregate function validation
  - Added position validation for AddColumn operations with non-negative constraints

### Added
- **Development Environment Enhancement**
  - Added comprehensive Serena memory system for project documentation and development guidance
  - Created project overview, code style conventions, and task completion requirements documentation
  - Added suggested development commands reference for improved developer experience

### Changed
- **Code Quality Improvements**
  - Replaced hardcoded "unknown" string literals with consistent `unknownValue` constant across content.go, operations.go, and output.go
  - Enhanced Claude settings to support additional Serena MCP tools for improved development workflow
- **Transformation Pipeline Task Completion**
  - Marked dual transformer system integration tasks (10.1, 10.2) as completed in transformation-pipeline feature

### Added
- **Dual Transformer System Implementation**
  - Implemented complete dual transformer architecture with data and byte transformation capabilities
  - Added TransformerAdapter for unified handling of DataTransformer and ByteTransformer interfaces
  - Integrated transformer system into base renderer with format-aware transformation pipeline
  - Data transformers now process document content before rendering with priority-based ordering
  - Byte transformers process rendered output after format conversion with pipeline execution
  - Added comprehensive integration tests covering end-to-end transformer pipeline with HTML, format filtering, and priority ordering
  - Enhanced renderer integration tests with mock transformers for thorough transformer system validation
  - Added renderDocumentWithFormat method to baseRenderer for dual transformer system support
  - Updated graph renderers (Mermaid, Draw.io) to support format-aware transformer pipeline integration
- **Pipeline Execution Engine Implementation**
  - Implemented Execute() and ExecuteContext() methods with full operation processing
  - Added operation optimization logic with intelligent reordering (filters before sorts for performance)
  - Comprehensive test suite for pipeline execution including reordering, lazy evaluation, error propagation, and context cancellation
  - Support for optimized execution patterns with filters applied before expensive operations
- **AddColumn Operation for Calculated Fields**
  - Implemented AddColumnOp struct with support for calculated fields in transformation pipeline
  - Support for calculated fields accessing all record data through function-based transformations
  - Schema evolution system with field position specification for flexible column ordering
  - Type inference and validation for calculated field values with comprehensive error handling
  - Added Pipeline.AddColumn() method for fluent API integration with transformation pipeline
  - Comprehensive test suite covering different data types, field positioning, and schema updates
  - Context cancellation support for long-running AddColumn operations
  - Immutability preservation through content cloning during field addition operations
- **GroupBy and Aggregation Operations Implementation**
  - Implemented GroupByOp struct with support for grouping by single or multiple columns
  - Added standard aggregate functions: Count, Sum, Average, Min, Max
  - Support for custom aggregate functions through AggregateFunc interface
  - Preserved key order in aggregated results schema maintaining v2 design principles
  - Added Pipeline.GroupBy() method for fluent API integration
  - Comprehensive test coverage including validation, different numeric types, and custom aggregates
  - Context cancellation support for long-running aggregation operations
  - Immutability preservation through content cloning
- **Pipeline API Foundation Implementation**
  - Created Pipeline struct with document reference, operations slice, and configurable options
  - Implemented Document.Pipeline() method to create transformation pipelines from documents
  - Added Operation interface with Name(), Apply(), CanOptimize(), and Validate() methods
  - Created PipelineOptions struct with MaxOperations and MaxExecutionTime configuration
  - Implemented Pipeline validation to ensure table content presence and operation compatibility
  - Added Execute() and ExecuteContext() methods with timeout support and context cancellation
  - Comprehensive error handling with detailed context tracking through ContextError
  - Transformation statistics tracking with per-operation metrics and duration measurement
  - Deep cloning of table content to preserve document immutability during transformations
  - Support for mixed content documents (preserves non-table content unchanged)
  - Added GetTransformStats() method to retrieve transformation statistics from document metadata
  - Created placeholder operation types (FilterOp, SortOp, LimitOp) for testing pipeline behavior
  - Comprehensive test suite covering initialization, chaining, immutability, validation, and timeout handling
- **Data Transformation System Core Implementation**
  - Implemented DataTransformer interface with Name(), TransformData(), CanTransform(), Priority(), and Describe() methods for structured data operations
  - Created TransformContext struct for carrying metadata through transformation pipeline with Format, Document, Metadata, and Stats fields
  - Added TransformStats tracking system with InputRecords, OutputRecords, FilteredCount, Duration, and detailed OperationStat metrics
  - Implemented TransformerAdapter for unified handling of both data transformers and byte transformers
  - Added TransformableContent interface with Clone() and Transform() methods for immutable transformation support
  - Implemented Clone() and Transform() methods on TableContent to support deep copying and functional transformations
  - Created PipelineError type with detailed error context including operation, stage, input, and contextual metadata
  - Comprehensive test suite with 100% coverage for DataTransformer interface, TransformContext, TransformerAdapter, and TransformableContent
- **Data Transformation Operations Implementation**
  - Implemented core data transformation operations system with Operation interface
  - Added FilterOp operation for filtering table records based on predicate functions with type-safe validation
  - Added SortOp operation for sorting table records by columns or custom comparators with multi-key support
  - Added LimitOp operation for limiting the number of table records with boundary validation
  - Added comprehensive test suite for all operation types with 800+ lines of test coverage
  - Added operation validation logic and error handling with detailed error messages
  - Added support for multiple sort keys and sort directions (ascending/descending)
  - Added context cancellation support for long-running operations to prevent blocking
  - Added immutability preservation for transformed content ensuring original data remains unchanged
- **Filter Operation Enhancement**
  - Implemented Pipeline.Filter() method with fluent API for chaining operations
  - Added comprehensive predicate support for different data types (string, integer, boolean, float)
  - Added extensive test coverage for filter functionality including complex predicates and chained operations
  - Added schema and key order preservation during filtering operations
  - Added support for filtering with missing fields and type assertions within predicate functions
  - Added validation for nil predicates with detailed error messages
- **Sort Operation Implementation**
  - Added Pipeline.Sort() method with support for multiple sort keys and flexible sorting configurations
  - Added Pipeline.SortBy() convenience method for single-column sorting with direction specification
  - Added Pipeline.SortWith() method for custom comparator-based sorting with flexible comparison logic
  - Implemented comprehensive test suite with 320+ lines of test coverage for sort functionality
  - Added support for sorting different data types (string, integer, boolean, time) with type-aware comparison
  - Added multi-column sorting with mixed sort directions (ascending/descending per column)
  - Added immutability preservation ensuring original documents remain unchanged during sort operations
  - Added validation for sort operations including empty keys detection and error handling
  - Added fluent API integration allowing chaining of sort operations with other pipeline operations
- **Limit Operation Implementation**
  - Added Pipeline.Limit() method for limiting the number of records returned from transformation pipeline
  - Implemented comprehensive test suite with 295+ lines of test coverage for limit functionality
  - Added support for chaining limit operations with filter and sort operations
  - Added validation for negative limit values with proper error handling
  - Added immutability preservation ensuring original documents remain unchanged during limit operations
  - Added boundary condition handling for limits larger than data size and zero limits
- **Feature Planning Documentation**
  - Added comprehensive requirements for Pipeline Visualization feature with 10 detailed user stories covering visualization modes, data capture, performance metrics, and interactive debugging
  - Added comprehensive requirements for Transformation Pipeline Enhancement with 11 detailed user stories covering data-level transformations, format-aware operations, and pipeline API for complex operations
  - Created decision logs and idea documentation for both features to support future implementation

### Changed
- **Code Modernization**
  - Updated test loops to use modern Go idioms (slices.Contains and range over int)
  - Replaced manual slice searching with slices.Contains for improved readability
- **AddColumn Implementation Task Completion**
  - Updated transformation pipeline tasks to mark AddColumn implementation as completed
  - Marked tasks 8.1 and 8.2 as completed in transformation-pipeline/tasks.md
- **Transformation Pipeline Task Completion**
  - Updated transformation pipeline tasks to mark operation interface and base operations as completed
  - Marked tasks 3.1 and 3.2 as completed in transformation-pipeline/tasks.md
- **Documentation Restructuring**
  - Moved v1 documentation files to have `v1-` prefix (v1-DOCUMENTATION.md, v1-GETTING_STARTED.md, v1-README.md)
  - Reorganized v2 documentation into dedicated `v2/docs/` directory for better organization
  - Updated all README links to point to new documentation locations
  - Consolidated v2 documentation files (API, MIGRATION, BREAKING_CHANGES, etc.) under single docs directory

### Removed
- **Debug Test File Cleanup**
  - Removed v2/debug_test.go that was not part of the core functionality
- Removed unused `.golangci.yaml-off` configuration file

### Fixed
- Updated local Claude settings to include additional MCP tools

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
