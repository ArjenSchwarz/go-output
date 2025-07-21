# Implementation Tasks: Go-Output v2.0

This document provides a series of discrete, manageable coding tasks to implement the v2.0 redesign. Each task builds incrementally on previous tasks and focuses on test-driven development.

## Implementation Tasks

### 1. Core Architecture Setup
- [x] 1.1. Create v2 module structure and basic interfaces
  - Create new v2 module with proper go.mod (Requirements 8.3, 14.1)
  - Define Content interface with encoding.TextAppender and encoding.BinaryAppender (Requirements 8.3, 14.1)
  - Define ContentType enum (Table, Text, Raw, Section) (Requirements 4.1-4.8)
  - Implement basic ID generation for content uniqueness (Requirements 4.4)
  - Write unit tests for interface contracts and ID generation (Requirements 15.1, 15.2)

- [x] 1.2. Implement Document and Builder core structure
  - Create Document struct to hold content collection and metadata (Requirements 2.1, 2.3)
  - Implement Builder struct with fluent API pattern (Requirements 2.1, 2.4)
  - Add New() constructor and Build() finalizer methods (Requirements 2.1)
  - Write unit tests for document creation and builder pattern (Requirements 15.1, 15.2)
  - Implement thread-safety tests for document operations (Requirements 1.2, 1.3)

### 2. Schema System with Key Ordering
- [x] 2.1. Implement Schema and Field structures
  - Create Schema struct with Fields slice and keyOrder preservation (Requirements 5.1, 5.2, 5.6)
  - Implement Field struct with Name, Type, Formatter, and Hidden properties (Requirements 5.3, 5.4)
  - Add extractKeyOrder function to preserve exact field order (Requirements 5.2, 5.3)
  - Write comprehensive tests for key order preservation across different input patterns (Requirements 5.1-5.7, 15.2)

- [x] 2.2. Create functional options for schema configuration
  - Implement WithSchema() option for explicit field definitions (Requirements 2.2, 5.1)
  - Implement WithKeys() option for simple key ordering (Requirements 2.2, 5.2)
  - Add auto-schema detection that preserves source data key order (Requirements 5.6)
  - Write tests verifying key order consistency across all scenarios (Requirements 5.4, 5.5, 15.2)

### 3. Content System Implementation
- [x] 3.1. Implement TableContent with key order preservation
  - Create TableContent struct with schema and records (Requirements 4.1, 4.6, 5.1)
  - Implement encoding.TextAppender for TableContent preserving key order (Requirements 5.2, 5.3, 8.3)
  - Add findField helper method for field lookups (Requirements 4.1)
  - Write tests verifying exact key order preservation in output (Requirements 5.2, 5.3, 15.2)
  - Test mixed table scenarios with different key sets (Requirements 4.6, 4.7)

- [x] 3.2. Implement TextContent for unstructured content
  - Create TextContent struct with text and styling (Requirements 4.2, 4.4)
  - Implement TextStyle struct for formatting options (Requirements 7.2, 7.7)
  - Add encoding.TextAppender implementation (Requirements 8.3)
  - Write tests for text content preservation and styling (Requirements 4.2, 4.4, 15.2)

- [x] 3.3. Implement RawContent for format-specific content
  - Create RawContent struct for format-specific data (Requirements 4.3)
  - Implement encoding interfaces for raw content (Requirements 8.3)
  - Add format validation and content preservation (Requirements 4.3, 9.1)
  - Write tests for raw content handling across formats (Requirements 4.3, 15.2)

- [x] 3.4. Implement SectionContent for content grouping
  - Create SectionContent struct with nested content support (Requirements 4.8)
  - Implement hierarchical content rendering (Requirements 4.4, 4.8)
  - Add encoding interfaces for section content (Requirements 8.3)
  - Write tests for nested content order preservation (Requirements 4.4, 4.8, 15.2)

### 4. Builder Pattern Methods
- [x] 4.1. Implement Table() builder method
  - Add Table() method with title, data, and options (Requirements 2.1, 4.1)
  - Implement TableOption functional options pattern (Requirements 2.2)
  - Add data type conversion and validation (Requirements 9.1, 9.2)
  - Write tests for table creation with various data types (Requirements 9.1, 15.2)
  - Test key order preservation through builder (Requirements 5.1, 5.2)

- [x] 4.2. Implement Text() and Header() builder methods
  - Add Text() method with styling options (Requirements 2.1, 4.2)
  - Implement Header() method for v1 compatibility (Requirements 7.1, 13.1)
  - Add TextOption functional options (Requirements 2.2)
  - Write tests for text content creation and styling (Requirements 4.2, 15.2)

- [x] 4.3. Implement Raw() and Section() builder methods
  - Add Raw() method for format-specific content (Requirements 2.1, 4.3)
  - Implement Section() method with nested builders (Requirements 2.1, 4.8)
  - Add proper content ordering in sections (Requirements 4.4)
  - Write tests for raw content and section nesting (Requirements 4.3, 4.8, 15.2)

### 5. Rendering Pipeline Foundation
- [x] 5.1. Define Renderer interface and format system
  - Create Renderer interface with Format(), Render(), and RenderTo() methods (Requirements 8.1, 8.2, 14.1)
  - Define Format struct with Name, Renderer, and Options (Requirements 7.1)
  - Add SupportsStreaming() method for streaming capability detection (Requirements 8.1)
  - Write interface tests and mock implementations (Requirements 15.2)

- [x] 5.2. Implement base renderer functionality
  - Create base renderer struct with common functionality (Requirements 8.5, 8.6)
  - Add context cancellation support for all renderers (Requirements 8.6)
  - Implement memory-efficient rendering patterns (Requirements 8.4, 11.1)
  - Write tests for context cancellation and memory usage (Requirements 8.6, 15.2)

### 6. Core Format Renderers
- [x] 6.1. Implement JSON and YAML renderers
  - Create JSONRenderer with proper data type preservation (Requirements 7.1, 9.1)
  - Implement YAMLRenderer with key order preservation (Requirements 7.1, 5.4)
  - Add streaming support for large datasets (Requirements 8.1, 11.3)
  - Write tests for data integrity and key ordering (Requirements 9.1, 5.4, 15.2)

- [x] 6.2. Implement CSV renderer with key order preservation
  - Create CSVRenderer that respects schema key order (Requirements 7.1, 5.2)
  - Add proper CSV escaping and encoding (Requirements 9.5, 12.3)
  - Implement streaming output for large tables (Requirements 8.1)
  - Write tests for CSV output correctness and key ordering (Requirements 5.2, 9.5, 15.2)

- [x] 6.3. Implement Table and HTML renderers
  - Create TableRenderer with styling support (Requirements 7.1, 7.7)
  - Implement HTMLRenderer with proper escaping (Requirements 7.1, 12.3)
  - Add table styling options from v1 (Requirements 7.7)
  - Write tests for table formatting and HTML safety (Requirements 7.7, 12.3, 15.2)

- [x] 6.4. Implement Markdown renderer with v1 features
  - Create MarkdownRenderer with table of contents support (Requirements 7.1, 7.8)
  - Add front matter support for metadata (Requirements 7.9)
  - Implement proper markdown escaping (Requirements 12.3)
  - Write tests for markdown structure and metadata (Requirements 7.8, 7.9, 15.2)

- [x] 6.5. Complete TOC generation implementation
  - Implement generateTableOfContents() method for automatic TOC creation (Requirements 7.8)
  - Add addSubsectionsToToC() for nested section support (Requirements 7.8)
  - Support TOC generation for SectionContent and TextContent with headers (Requirements 7.8)
  - Add proper anchor link generation and markdown escaping for TOC links (Requirements 7.8)
  - Write comprehensive tests for TOC generation with various content structures (Requirements 7.8, 15.2)

### 7. Graph Format Renderers
- [x] 7.1. Implement DOT renderer for graph visualization
  - Create DOTRenderer for Graphviz output (Requirements 7.1, 7.10)
  - Add support for from/to relationship extraction (Requirements 7.10)
  - Implement proper DOT syntax generation (Requirements 7.1)
  - Write tests for graph generation and DOT syntax (Requirements 7.10, 15.2)

- [x] 7.2. Implement Mermaid renderer
  - Create MermaidRenderer for diagram generation (Requirements 7.1, 7.10)
  - Add relationship mapping from table data (Requirements 7.10)
  - Implement Mermaid syntax generation (Requirements 7.1)
  - Write tests for Mermaid diagram generation (Requirements 7.10, 15.2)

- [x] 7.3. Implement Draw.io renderer (Basic XML - needs CSV update)
  - Create DrawIORenderer for XML diagram output (Requirements 7.1, 7.10)
  - Add Draw.io XML format generation (Requirements 7.1)
  - Implement relationship visualization (Requirements 7.10)
  - Write tests for Draw.io XML structure (Requirements 7.10, 15.2)

- [x] 7.4. Extend Mermaid renderer for additional chart types
  - Add Mermaid Gantt chart support for project timelines (Requirements 7.10, 7.11)
  - Implement Mermaid pie chart generation for data proportions (Requirements 7.10, 7.11)
  - Add flowchart improvements and enhanced syntax support (Requirements 7.10)
  - Create ChartContent type for specialized chart data structures (Requirements 7.10, 7.11)
  - Write comprehensive tests for all Mermaid chart types (Requirements 7.10, 15.2)

- [x] 7.5. Update Draw.io renderer to CSV format (v1 compatibility)
  - Replace XML output with Draw.io CSV format for better import compatibility (Requirements 7.12)
  - Implement DrawIOHeader configuration with placeholders (%Name%, %Image%) (Requirements 7.12)
  - Add layout options (auto, horizontalflow, verticalflow, horizontaltree, etc.) (Requirements 7.12)
  - Support connection definitions with from/to mappings and styling (Requirements 7.12)
  - Add hierarchical diagram support with parent-child relationships (Requirements 7.12)
  - Implement node and edge spacing control (Requirements 7.12)
  - Integrate AWS service shape definitions from v1 (Requirements 7.12)
  - Support manual positioning via coordinate columns (Requirements 7.12)
  - Write comprehensive tests for CSV format and all configuration options (Requirements 7.12, 15.2)

### 8. Transform System Implementation
- [x] 8.1. Create Transformer interface and pipeline
  - Define Transformer interface with Name(), Transform(), CanTransform(), Priority() (Requirements 6.1, 6.4, 14.1)
  - Implement TransformPipeline for managing multiple transformers (Requirements 6.2, 6.4)
  - Add priority-based execution ordering (Requirements 6.6)
  - Write tests for transformer pipeline execution order (Requirements 6.4, 6.6, 15.2)

- [x] 8.2. Implement v1 compatibility transformers
  - Create EmojiTransformer for emoji conversion (Requirements 6.2, 7.2)
  - Implement ColorTransformer with color scheme support (Requirements 6.2, 7.2)
  - Add SortTransformer for data sorting (Requirements 6.2, 7.2)
  - Create LineSplitTransformer for line splitting (Requirements 6.2, 7.2)
  - Write tests for each transformer's functionality (Requirements 7.2, 15.2)

- [x] 8.3. Add format-aware transformation
  - Implement format detection in transformers (Requirements 6.5)
  - Add conditional transformation based on output format (Requirements 6.5)
  - Ensure transformers don't modify original document data (Requirements 6.7)
  - Write tests for format-specific transformation behavior (Requirements 6.5, 6.7, 15.2)

### 9. Writer System Implementation
- [x] 9.1. Implement Writer interface and base functionality
  - Define Writer interface with Write() method (Requirements 3.1, 14.1)
  - Create base writer functionality with error handling (Requirements 10.1, 10.2)
  - Add context support for cancellation (Requirements 8.6)
  - Write tests for writer interface compliance (Requirements 15.2)

- [x] 9.2. Implement FileWriter with security
  - Create FileWriter using os.Root for directory confinement (Requirements 12.1, 12.2)
  - Add file path validation and pattern support (Requirements 12.2, 7.3)
  - Implement concurrent file writing for multiple formats (Requirements 3.3, 11.2)
  - Write tests for file security and concurrent access (Requirements 12.1, 12.2, 15.2)

- [x] 9.3. Implement StdoutWriter and MultiWriter
  - Create StdoutWriter for console output (Requirements 3.2)
  - Implement MultiWriter for multiple destinations (Requirements 3.1, 3.3)
  - Add streaming support for stdout output (Requirements 8.1, 3.2)
  - Write tests for multi-destination output (Requirements 3.1, 3.3, 15.2)

- [x] 9.4. Implement S3Writer for cloud storage
  - Create S3Writer with AWS SDK integration (Requirements 7.4)
  - Add proper error handling and retry logic (Requirements 10.1)
  - Implement streaming upload for large files (Requirements 8.1)
  - Write integration tests with S3 mock (Requirements 7.4, 15.3)

### 10. Progress System Implementation
- [x] 10.1. Implement Progress interface
  - Define Progress interface with SetTotal(), SetCurrent(), Increment() methods (Requirements 7.6)
  - Create progress implementations for different output formats (Requirements 7.6)
  - Add status message support (Requirements 7.6)
  - Write tests for progress tracking accuracy (Requirements 7.6, 15.2)

- [x] 10.2. Integrate progress with rendering pipeline
  - Add progress tracking to long-running operations (Requirements 7.6)
  - Implement progress updates during multi-format rendering (Requirements 3.3, 7.6)
  - Add failure handling and error reporting (Requirements 10.1)
  - Write tests for progress integration (Requirements 7.6, 15.2)

- [x] 10.3. Enhance Progress interface for v1 feature parity
  - Add ProgressColor enum with Default, Green, Red, Yellow, Blue colors (Requirements 7.6)
  - Extend Progress interface with SetColor(), IsActive(), SetContext() methods (Requirements 7.6)
  - Update ProgressOptions struct to match v1 configuration options (Requirements 7.6)
  - Write tests for color functionality and v1 compatibility methods (Requirements 7.6, 15.2)

- [x] 10.4. Implement PrettyProgress with go-pretty library
  - Add go-pretty/v6/progress dependency for professional progress bars (Requirements 7.6)
  - Create PrettyProgress struct with writer, tracker, and lifecycle management (Requirements 7.6)
  - Implement TTY detection using go-isatty for terminal-only progress display (Requirements 7.6)
  - Add signal handling for SIGWINCH (terminal resize) and graceful shutdown (Requirements 7.6)
  - Implement automatic cleanup with finalizers and active progress tracking (Requirements 7.6)
  - Write comprehensive tests for PrettyProgress functionality (Requirements 7.6, 15.2)

- [x] 10.5. Implement format-aware progress creation
  - Create NewProgress() function that auto-selects implementation based on output format (Requirements 7.6)
  - Implement logic to use NoOpProgress for JSON/CSV/YAML/DOT formats (Requirements 7.6)
  - Use PrettyProgress for Table/HTML/Markdown formats when TTY is detected (Requirements 7.6)
  - Add NewProgressForFormats() for intelligent selection with multiple output formats (Requirements 7.6)
  - Write tests for format-aware progress selection logic (Requirements 7.6, 15.2)

- [x] 10.6. Update existing progress implementations
  - Enhance TextProgress to implement new v1 compatibility methods (Requirements 7.6)
  - Update NoOpProgress to match v1 NoOpProgress behavior exactly (Requirements 7.6)
  - Add color support to TextProgress as fallback when go-pretty isn't available (Requirements 7.6)
  - Ensure all progress implementations are thread-safe with proper mutex usage (Requirements 7.6)
  - Write migration tests comparing v1 and v2 progress behavior (Requirements 7.6, 15.2)

- [x] 10.7. Integration with Output system
  - Update Output.Render() to use format-aware progress creation (Requirements 7.6)
  - Add automatic progress cleanup and lifecycle management to Output (Requirements 7.6)
  - Implement progress context cancellation integration with render context (Requirements 7.6)
  - Add progress color updates based on render success/failure states (Requirements 7.6)
  - Write integration tests for Output system with enhanced progress (Requirements 7.6, 15.2)

### 11. Output Configuration System
- [x] 11.1. Implement Output struct and options
  - Create Output struct with formats, pipeline, writers (Requirements 2.2, 2.3)
  - Implement functional options pattern for configuration (Requirements 2.2)
  - Add v1 compatibility options (TableStyle, TOC, FrontMatter) (Requirements 7.7, 7.8, 7.9)
  - Write tests for option composition (Requirements 2.2, 15.2)

- [x] 11.2. Implement main rendering orchestration
  - Create Render() method that coordinates all components (Requirements 2.3, 8.5)
  - Add concurrent rendering for multiple formats (Requirements 3.3, 11.2)
  - Implement proper error handling and context cancellation (Requirements 10.1, 8.6)
  - Write integration tests for end-to-end rendering (Requirements 15.3)

### 12. Error Handling System
- [x] 12.1. Implement error types and handling
  - Create RenderError, ValidationError, TransformError types (Requirements 10.1, 10.3)
  - Add error wrapping and context information (Requirements 10.3)
  - Implement early validation with helpful messages (Requirements 10.2)
  - Write tests for error conditions and messages (Requirements 10.1, 15.2)

- [x] 12.2. Add debug tracing and panic recovery
  - Implement optional debug tracing for pipeline (Requirements 10.4)
  - Add panic recovery and conversion to errors (Requirements 10.5)
  - Create exported error types for programmatic handling (Requirements 10.6)
  - Write tests for debug features and panic recovery (Requirements 10.4, 10.5, 15.2)

- [x] 12.3. Enhance error context and reporting
  - Add more detailed error context including content type, format, and transformer information (Requirements 10.3)
  - Implement error aggregation for multiple format rendering failures (Requirements 10.1)
  - Add error source tracking through the rendering pipeline (Requirements 10.3)
  - Create structured error responses for programmatic error analysis (Requirements 10.6)
  - Write tests for enhanced error reporting and context preservation (Requirements 10.1, 10.3, 15.2)

### 13. Migration Tool Development
- [x] 13.1. Create AST-based migration tool
  - Implement Go AST parser for v1 code analysis (Requirements 13.2, 13.3)
  - Add pattern recognition for common v1 usage patterns (Requirements 13.3)
  - Create code transformation rules for v1 to v2 conversion (Requirements 13.2, 13.6)
  - Write tests for migration tool accuracy (Requirements 13.3, 15.2)

- [x] 13.2. Implement migration patterns and documentation
  - Add migration patterns for all v1 features (Requirements 13.5, 13.7)
  - Create before/after examples for all breaking changes (Requirements 13.7)
  - Generate comprehensive migration guide (Requirements 13.1, 13.5)
  - Write integration tests for migration tool with real v1 code (Requirements 13.3, 15.3)

### 14. Comprehensive Testing Suite
- [ ] 14.1. Create unit test suite
  - Implement unit tests for all public APIs (Requirements 15.2)
  - Add tests for key order preservation across all scenarios (Requirements 5.1-5.7)
  - Create tests for thread safety and concurrent operations (Requirements 1.2, 1.3)
  - Achieve minimum 80% test coverage (Requirements 15.1)

- [ ] 14.2. Implement integration and performance tests
  - Create end-to-end integration tests (Requirements 15.3)
  - Add benchmark tests using Go 1.24's testing.B.Loop (Requirements 15.4, 15.6)
  - Implement fuzz tests for input handling (Requirements 15.5)
  - Write performance comparison tests against v1 (Requirements 11.1-11.5)

- [ ] 14.3. Create comprehensive validation tests
  - Add tests for all v1 feature parity requirements (Requirements 7.1-7.10)
  - Implement security validation tests (Requirements 12.1-12.5)
  - Create data integrity tests across all formats (Requirements 9.1-9.5)
  - Add memory and resource usage validation (Requirements 11.1, 11.4)

### 15. Final Integration and Validation
- [ ] 15.1. Complete API documentation and examples
  - Generate comprehensive API documentation
  - Create working examples for all major use cases
  - Add migration examples for common v1 patterns
  - Validate all examples compile and run correctly

- [ ] 15.2. Final system integration testing
  - Run complete test suite and achieve target coverage
  - Perform end-to-end validation of all requirements
  - Execute performance benchmarks and memory analysis
  - Validate migration tool on real-world v1 codebases

## Task Dependencies

Tasks build incrementally with the following key dependencies:
- Core architecture (1.x) must be completed before content implementation (3.x)
- Schema system (2.x) is required for TableContent (3.1) and Builder methods (4.x)
- Renderer interface (5.1) is prerequisite for all format renderers (6.x, 7.x)
- Transform interface (8.1) must precede transformer implementations (8.2, 8.3)
- Writer interface (9.1) is required for all writer implementations (9.2-9.4)
- All core components must be implemented before integration tasks (11.x, 14.x, 15.x)

Each task includes specific requirement references and focuses on code implementation that can be executed by a coding agent within the development environment.