
1.5.0 / In Development
======================

  * Implement basic error interface and types for comprehensive error handling
  * Add structured error system with error codes, severity levels, and context
  * Support for error suggestions and wrapped errors with builder pattern
  * JSON marshaling capability for structured logging
  * Add validation error types with violation tracking and composite error collection
  * Support for detailed field-level validation failures with context and constraints
  * Implement processing error types for runtime failures with retry and partial result support
  * Add retryable error wrapper with exponential backoff configuration and transient error detection
  * Create core validator interface with function-based and chain-based validation support
  * Add validation context and configurable fail-fast vs collect-all error handling modes
  * Implement built-in data validators: RequiredColumnsValidator, DataTypeValidator, NotEmptyValidator, and ConstraintValidator
  * Add constraint interface for custom business rules with common implementations for positive numbers and non-empty strings
  * Implement configuration validators: FormatValidator, FilePathValidator, S3ConfigValidator, and CompatibilityValidator  
  * Add format-specific validators for Mermaid and DOT with comprehensive validation rules
  * Support for AWS S3 bucket name validation and file path permission checking

1.4.0 / 2023-11-14
==================

  * Update Go version and support file as output and CLI at the same time

1.3.0 / 2023-06-17
==================

  * Add convenience function AddContents
  * Add license that was somehow missing
  * Update go version
  * Add support for YAML output format
