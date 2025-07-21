# Requirements Document: Go-Output Library v2.0 (Final)

## Introduction

This document outlines the requirements for go-output v2.0, a complete redesign of the library that addresses current implementation issues while providing a cleaner, more maintainable API. As a major version release, v2.0 makes a clean break from v1 to solve fundamental architectural problems and provide a modern, idiomatic Go API.

**Minimum Go Version**: 1.24
**Breaking Change**: This is a major version bump (v2.0.0) with no backward compatibility with v1.

## Requirements

### 1. State Management Requirements

**User Story**: As a library maintainer, I want encapsulated state management, so that the library is thread-safe and avoids race conditions.

**Acceptance Criteria**:
1.1. The system SHALL eliminate all global variables completely.
1.2. The system SHALL implement thread-safe operations for all state modifications.
1.3. The system SHALL ensure that multiple output instances can operate independently without interfering with each other.
1.4. The system SHALL provide a clear state lifecycle that is properly initialized and cleaned up.
1.5. The system SHALL use instance-based state management throughout.

### 2. API Design Requirements

**User Story**: As a developer, I want a clean, intuitive API, so that I can create outputs without understanding internal complexity.

**Acceptance Criteria**:
2.1. The system SHALL provide a fluent builder pattern for document construction.
2.2. The system SHALL use functional options for configuration instead of settings structs.
2.3. The system SHALL separate document building from rendering/output.
2.4. The system SHALL provide type-safe APIs where possible using Go generics.
2.5. The system SHALL NOT require users to understand internal implementation details.
2.6. The system SHALL make a clean break from v1 API with no compatibility aliases.

### 3. Multiple Output Format Requirements

**User Story**: As a developer, I want to output data in multiple formats simultaneously, so that I can generate reports for different audiences efficiently.

**Acceptance Criteria**:
3.1. The system SHALL support writing to multiple destinations in different formats from a single document.
3.2. The system SHALL support streaming to stdout while buffering for file output.
3.3. The system SHALL process multiple outputs concurrently where possible.
3.4. The system SHALL handle format-specific transformations independently for each output destination.
3.5. The system SHALL support adding new output formats without modifying existing code.

### 4. Content Type Requirements

**User Story**: As a user, I want to mix different types of content in my output, so that I can create comprehensive reports.

**Acceptance Criteria**:
4.1. The system SHALL support multiple tables with completely different structures in the same document.
4.2. The system SHALL support adding unstructured text content between tables.
4.3. The system SHALL support adding format-specific raw content (e.g., HTML snippets in HTML output).
4.4. The system SHALL preserve the order of mixed content types as they were added.
4.5. WHEN rendering different content types, THEN each format renderer SHALL handle them appropriately.
4.6. Each piece of structured content (table) SHALL maintain its own independent set of keys.
4.7. The system SHALL NOT require any key consistency or overlap between different pieces of structured content.
4.8. The system SHALL support nested content through sections or groups.

### 5. Key Ordering Requirements

**User Story**: As a user, I want consistent and predictable key ordering in my output, so that my reports are professional and reproducible.

**Acceptance Criteria**:
5.1. Each table SHALL maintain its own key order as defined when the table was created.
5.2. The system SHALL preserve the exact order of keys as specified by the user for each table.
5.3. WHEN a table is rendered, THEN the columns SHALL appear in the exact order specified in the schema.
5.4. The key order SHALL be consistent across all output formats for the same table.
5.5. The system SHALL NOT reorder keys alphabetically or by any other criteria unless explicitly requested.
5.6. WHEN using auto-schema, THEN the system SHALL preserve the order keys appear in the source data.
5.7. Tables within the same document MAY have completely different key orders.

### 6. Transform System Requirements

**User Story**: As a developer, I want composable transformers for output modification, so that I can customize output without modifying core logic.

**Acceptance Criteria**:
6.1. The system SHALL implement transformers as separate, composable components.
6.2. Built-in features (emoji conversion, colors, sorting, line splitting) SHALL be implemented as transformers.
6.3. The system SHALL allow adding custom transformers to the pipeline.
6.4. WHEN multiple transformers are configured, THEN they SHALL be applied in a deterministic order.
6.5. Transformers SHALL be format-aware and only apply where appropriate.
6.6. The system SHALL support transformer priorities to control execution order.
6.7. Transformers SHALL NOT modify the original document data.

### 7. Feature Parity Requirements

**User Story**: As a v1 user, I want all v1 functionality available in v2, so that I can migrate without losing capabilities.

**Acceptance Criteria**:
7.1. The system SHALL support all output formats from v1: JSON, YAML, CSV, HTML, Table, Markdown, DOT, Mermaid, Draw.io.
7.2. The system SHALL support all v1 transformations: emoji conversion, colors, sorting, line splitting.
7.3. The system SHALL support file output with configurable paths.
7.4. The system SHALL support S3 output for cloud storage.
7.5. The system SHALL support all v1 data types and conversions.
7.6. The system SHALL support progress indicators for long-running operations with full v1 feature parity including:
  - Professional progress bar rendering using go-pretty library or equivalent
  - Color support (Default, Green, Red, Yellow, Blue) for visual feedback  
  - TTY detection to only display progress bars in terminal environments
  - Format-aware progress creation (no-op for JSON/CSV/YAML/DOT, visual for Table/HTML/Markdown)
  - Signal handling for terminal resize (SIGWINCH) and graceful shutdown
  - IsActive() status reporting for lifecycle management
  - Automatic cleanup and finalizer support
  - Thread-safe concurrent access with proper mutex protection
  - Context cancellation support for early termination
7.7. The system SHALL support all v1 table styling options.
7.8. The system SHALL support table of contents generation for applicable formats.
7.9. The system SHALL support front matter for markdown output.
7.10. The system SHALL support all v1 chart and diagram features including:
  - DOT format for Graphviz diagrams with node relationships
  - Mermaid flowcharts for process visualization
  - Mermaid Gantt charts for project timeline visualization  
  - Mermaid pie charts for data proportion visualization
  - Draw.io CSV format for diagram import with layout and styling configuration
7.11. The system SHALL automatically detect appropriate chart types based on data structure and content.
7.12. The Draw.io CSV renderer SHALL support all v1 Draw.io features including:
  - Header configuration with placeholders (%Name%, %Image%) for dynamic content
  - Layout options (auto, horizontalflow, verticalflow, horizontaltree, etc.)
  - Connection definitions with from/to mappings and styling
  - Hierarchical diagrams with parent-child relationships
  - Node and edge spacing control
  - AWS service shape integration with pre-defined shapes
  - Manual positioning via coordinate columns

### 8. Rendering Requirements

**User Story**: As a user, I want efficient and flexible rendering options, so that I can handle both small and large datasets effectively.

**Acceptance Criteria**:
8.1. The system SHALL support streaming output for large datasets.
8.2. The system SHALL support buffered output for smaller datasets.
8.3. The system SHALL use the new encoding.TextAppender and encoding.BinaryAppender interfaces.
8.4. The system SHALL handle memory efficiently when dealing with large datasets.
8.5. WHEN rendering to multiple formats, THEN the system SHALL reuse document traversal where possible.
8.6. The system SHALL support context cancellation during rendering.

### 9. Data Integrity Requirements

**User Story**: As a user, I want my data to be accurately preserved across all transformations, so that I can trust the output.

**Acceptance Criteria**:
9.1. The system SHALL preserve all data types (strings, integers, floats, booleans, nil values) correctly.
9.2. The system SHALL handle missing keys gracefully without losing other data.
9.3. The system SHALL maintain data integrity when converting between different formats.
9.4. The system SHALL preserve the original data without modification during rendering.
9.5. The system SHALL handle special characters and encoding correctly in all formats.

### 10. Error Handling and Debugging Requirements

**User Story**: As a developer, I want clear error messages and debugging options, so that I can diagnose issues effectively.

**Acceptance Criteria**:
10.1. The system SHALL provide clear, actionable error messages for all error conditions.
10.2. The system SHALL validate inputs early and fail fast with helpful messages.
10.3. The system SHALL include context in errors (e.g., which content, which format, which transformer).
10.4. The system SHALL support optional debug tracing of the rendering pipeline.
10.5. The system SHALL handle panics gracefully and convert them to errors where possible.
10.6. Error types SHALL be exported for programmatic error handling.

### 11. Performance Requirements

**User Story**: As a user processing large datasets, I want efficient output generation, so that my applications remain responsive.

**Acceptance Criteria**:
11.1. The system SHALL minimize memory allocations using append interfaces.
11.2. The system SHALL process outputs concurrently when multiple destinations are configured.
11.3. The system SHALL support streaming for datasets too large to fit in memory.
11.4. The system SHALL use weak pointers for caching where appropriate.
11.5. The system SHALL avoid redundant processing when generating multiple formats.

### 12. Security Requirements

**User Story**: As a system administrator, I want secure file operations, so that the library cannot be exploited.

**Acceptance Criteria**:
12.1. The system SHALL use os.Root for directory-confined file operations.
12.2. The system SHALL validate file paths to prevent directory traversal.
12.3. The system SHALL properly escape content for each output format.
12.4. The system SHALL implement resource limits to prevent DoS attacks.
12.5. The system SHALL not leak sensitive information in error messages.

### 13. Migration Requirements

**User Story**: As a v1 user, I want clear migration paths and tools, so that I can upgrade to v2 efficiently.

**Acceptance Criteria**:
13.1. The system SHALL provide a comprehensive migration guide.
13.2. The system SHALL provide an automated migration tool for common patterns.
13.3. The migration tool SHALL be able to convert 80% of typical v1 usage automatically.
13.4. The migration tool SHALL be included in the main module initially.
13.5. The migration guide SHALL include step-by-step instructions for manual conversion.
13.6. The migration guide SHALL be written such that an AI agent could follow it to perform migrations.
13.7. All breaking changes SHALL be clearly documented with before/after examples.
13.8. The system SHALL NOT provide a compatibility layer (clean break).

### 14. Extensibility Requirements

**User Story**: As a library maintainer, I want the library to be easily extensible, so that new features can be added without breaking existing code.

**Acceptance Criteria**:
14.1. The system SHALL use interfaces for all major components (renderers, transformers, writers).
14.2. The system SHALL allow users to implement custom renderers for new formats.
14.3. The system SHALL allow users to implement custom transformers.
14.4. The system SHALL allow users to implement custom content types.
14.5. The system SHALL NOT require modifying core code to add new functionality.

### 15. Testing Requirements

**User Story**: As a library maintainer, I want comprehensive test coverage, so that I can refactor with confidence.

**Acceptance Criteria**:
15.1. The system SHALL maintain at least 80% test coverage.
15.2. The system SHALL include unit tests for all public APIs.
15.3. The system SHALL include integration tests for end-to-end scenarios.
15.4. The system SHALL include benchmark tests for performance-critical paths.
15.5. The system SHALL include fuzz tests for input handling.
15.6. The system SHALL use Go 1.24's testing.B.Loop for accurate benchmarks.

## Success Criteria

The v2.0 redesign will be considered successful when:
- All architectural issues from v1 are resolved (global state, key ordering, mixed content)
- The new API is demonstrably cleaner and more intuitive than v1
- All v1 functionality is available in v2 (though with different APIs)
- Performance is equal to or better than v1 for common use cases
- The migration tool successfully converts typical v1 code
- The library can handle all v1 use cases plus new scenarios like mixed content
- Key ordering is preserved exactly as specified by users

## Technical Constraints

- Must use Go 1.24 or later
- Must be published as v2 module (github.com/ArjenSchwarz/go-output/v2)
- Must not depend on v1 code
- Should minimize external dependencies
- Must maintain reasonable binary size
- Must work on all platforms supported by Go
- v1 will remain available indefinitely but receive no updates

## Non-Requirements

The following are explicitly out of scope for v2.0:
- Backward compatibility with v1 API (clean break)
- Support for custom output formats beyond the provided extension points
- Automatic error recovery or retry mechanisms
- Real-time collaborative editing features
- Built-in data persistence or caching beyond rendering
- Maintaining v1 after v2 release

## Breaking Changes from v1

The following v1 features will be removed or significantly changed:
1. **OutputArray struct**: Replaced with Document and Builder pattern
2. **OutputSettings struct**: Replaced with functional options
3. **Keys field**: Replaced with per-table schemas that preserve order
4. **AddContents method**: Replaced with type-specific methods
5. **Write method**: Replaced with Render method that requires context
6. **Global state**: Completely eliminated
7. **String-based configuration**: Replaced with type-safe options
8. **Import path**: Changed to /v2

## Future Enhancements

The following features are planned for future versions but out of scope for v2.0:

1. **Plugin System**: Dynamic loading of formats/transformers
2. **Template System**: User-defined templates for custom output formatting
3. **Performance Metrics**: Built-in performance tracking and optimization
4. **Extended Graph Support**: Additional graph visualization options and formats
5. **Advanced Table Features**: Grouping, aggregation, and complex table operations
6. **Draw.io XML Format**: Native XML export for advanced diagram features (currently using CSV for v1 compatibility)
7. **Real-time Collaboration**: Live editing and sharing capabilities
8. **Data Persistence**: Built-in caching and storage beyond rendering
9. **Automatic Schema Detection**: Enhanced schema inference from complex data structures
10. **Advanced Transformers**: AI-powered content transformation and optimization

