## [Unreleased]

### Added
- Initial v2.0 module structure with clean architecture and no global state
- Core Content interface with encoding.TextAppender and encoding.BinaryAppender support
- Document struct for holding content collections with thread-safe operations
- Builder pattern implementation with fluent API for document construction
- Comprehensive test coverage including thread-safety and concurrent operation tests
- Design and requirements documentation for complete v2 redesign
- Task tracking system for incremental implementation

### Changed
- Complete architectural redesign eliminating all global variables
- Minimum Go version requirement updated to 1.24

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
