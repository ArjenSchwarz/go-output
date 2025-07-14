
## [Unreleased]

### Added
- Core error handling system with structured error types and interfaces
- Comprehensive error type hierarchy (OutputError, ValidationError, ProcessingError)
- Error codes organized by category (1xxx configuration, 2xxx validation, 3xxx processing, 4xxx runtime)
- Error severity levels (Fatal, Error, Warning, Info) with string representation
- Detailed error context with operation, field, value, and metadata information
- Fluent error builder pattern for constructing complex errors with suggestions
- Validation error support with violation tracking and composite error handling
- Processing error support with retry capability and partial result preservation
- JSON marshaling support for structured error output and monitoring integration
- Comprehensive test suite with 100% interface compliance verification
- Error handling and validation design documentation
- Comprehensive requirements document for error management system
- Architecture design for replacing log.Fatal() with structured error handling
- Validation framework specifications with built-in validators
- Recovery strategies and error handling modes documentation

1.4.0 / 2023-11-14
==================

  * Update Go version and support file as output and CLI at the same time

1.3.0 / 2023-06-17
==================

  * Add convenience function AddContents
  * Add license that was somehow missing
  * Update go version
  * Add support for YAML output format
