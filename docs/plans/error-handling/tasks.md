# Error Handling Implementation Tasks - TDD Approach

This document contains a series of incremental, test-driven development prompts for implementing the error handling and validation system in the go-output library. Each task builds on the previous ones and focuses on specific, testable functionality.

## Phase 1: Core Error Types and Interfaces

### Task 1: Basic Error Interface and Types
- [x] **Objective**: Create the foundational error interface hierarchy and basic error types.

**Test-driven prompt**:
"Write tests first, then implement a basic error type system for a Go library. Create:

1. Write tests for an `OutputError` interface with methods: `Code() ErrorCode`, `Severity() ErrorSeverity`, `Context() ErrorContext`, `Suggestions() []string`, `Wrap(error) OutputError`
2. Write tests for `ErrorCode` as a string type with constants like `ErrInvalidFormat = "OUT-1001"`, `ErrMissingRequired = "OUT-1002"`
3. Write tests for `ErrorSeverity` as an int type with constants: `SeverityFatal`, `SeverityError`, `SeverityWarning`, `SeverityInfo`
4. Write tests for `ErrorContext` struct with fields: `Operation string`, `Field string`, `Value interface{}`, `Index int`, `Metadata map[string]interface{}`
5. After writing comprehensive tests, implement a `baseError` struct that satisfies the `OutputError` interface

Focus on:
- Error message formatting that includes code, context, suggestions, and wrapped errors
- JSON marshaling capability for structured logging
- Immutable error creation with builder pattern methods like `WithContext()`, `WithSuggestions()`

The implementation should be in package `errors` and all types should be exported. Write thorough unit tests covering all methods and edge cases."

### Task 2: Validation Error Types
- [x] **Objective**: Extend the error system with validation-specific error types.

**Test-driven prompt**:
"Building on the basic error types from Task 1, write tests first then implement validation-specific error types:

1. Write tests for a `ValidationError` interface that extends `OutputError` with methods: `Violations() []Violation`, `IsComposite() bool`
2. Write tests for a `Violation` struct with fields: `Field string`, `Value interface{}`, `Constraint string`, `Message string`
3. Write tests for a `validationError` struct that embeds `baseError` and implements `ValidationError`
4. Write tests for a `CompositeError` type that can collect multiple validation errors
5. Write tests for constructor functions like `NewValidationError()`, `NewCompositeError()`

Focus on:
- Detailed error messages showing all violations in a readable format
- Ability to accumulate multiple validation errors before returning
- Methods to add violations and combine multiple validation errors
- JSON serialization that includes all violation details

Ensure the validation errors integrate seamlessly with the base error system and maintain all functionality from Task 1."

### Task 3: Processing Error Types
- [x] **Objective**: Add processing-specific error types for runtime failures.

**Test-driven prompt**:
"Building on Tasks 1-2, write tests first then implement processing error types:

1. Write tests for a `ProcessingError` interface extending `OutputError` with methods: `Retryable() bool`, `PartialResult() interface{}`
2. Write tests for a `processingError` struct implementing `ProcessingError`
3. Write tests for error codes like `ErrFileWrite = "OUT-3001"`, `ErrS3Upload = "OUT-3002"`, `ErrTemplateRender = "OUT-3003"`
4. Write tests for constructor functions with retry and partial result capabilities
5. Write tests for a `RetryableError` wrapper that marks errors as retryable

Focus on:
- Clear distinction between retryable and non-retryable processing errors
- Ability to store partial results when processing fails partway through
- Integration with the existing error hierarchy
- Error messages that help debugging runtime issues

All error types should work together seamlessly and support the builder pattern for adding context and suggestions."

## Phase 2: Validation Framework

### Task 4: Core Validator Interface
- [x] **Objective**: Create the validation framework foundation.

**Test-driven prompt**:
"Write tests first, then implement a flexible validation framework:

1. Write tests for a `Validator` interface with methods: `Validate(subject interface{}) error`, `Name() string`
2. Write tests for a `ValidatorFunc` type that implements `Validator` for function-based validators
3. Write tests for a `ValidatorChain` that can run multiple validators in sequence
4. Write tests for a `ValidationContext` that provides additional context during validation
5. Write tests for early termination vs. collecting all validation errors

Focus on:
- Generic validation that works with any Go type using interface{}
- Ability to chain validators and control execution flow
- Clear error reporting when validation fails
- Support for both fail-fast and collect-all validation modes

The framework should integrate with the error types from Phase 1 and be extensible for custom validators."

### Task 5: Built-in Data Validators
- [x] **Objective**: Implement common validators for data validation.

**Test-driven prompt**:
"Building on Task 4, write tests first then implement built-in validators for common data validation scenarios:

1. Write tests for `RequiredColumnsValidator` that checks if specified columns exist in data
2. Write tests for `DataTypeValidator` that validates column data types match expectations
3. Write tests for `NotEmptyValidator` that ensures datasets aren't empty
4. Write tests for `ConstraintValidator` that applies custom business rules to data rows
5. Write tests for a `Constraint` interface with `Check(row map[string]interface{}) error` method

Focus on:
- Validators that work with typical data structures (slices of maps, structs, etc.)
- Detailed violation reporting showing which fields/rows failed validation
- Reusable constraint system for business rules
- Performance considerations for large datasets

Each validator should provide clear, actionable error messages and integrate with the validation error types from previous tasks."

### Task 6: Configuration Validators
- [x] **Objective**: Add validators for configuration and settings validation.

**Test-driven prompt**:
"Building on Tasks 4-5, write tests first then implement configuration validators:

1. Write tests for `FormatValidator` that validates output format strings against allowed values
2. Write tests for `FilePathValidator` that validates file paths and permissions
3. Write tests for `S3ConfigValidator` that validates S3 bucket and key configurations
4. Write tests for `CompatibilityValidator` that checks for incompatible setting combinations
5. Write tests for format-specific validators (e.g., `MermaidValidator`, `DotValidator`)

Focus on:
- Pre-flight validation to catch configuration errors early
- Helpful suggestions for fixing configuration issues
- Integration with external services (file system, S3) for validation
- Format-specific requirements and dependencies

All validators should use the error types from Phase 1 and provide actionable feedback for configuration issues."

## Phase 3: Error Handling Integration

### Task 7: Error Handler Interface and Basic Implementation
- [x] **Objective**: Create error handling system with different modes.

**Test-driven prompt**:
"Write tests first, then implement an error handling system:

1. Write tests for an `ErrorHandler` interface with methods: `HandleError(err error) error`, `SetMode(mode ErrorMode)`
2. Write tests for `ErrorMode` constants: `ErrorModeStrict`, `ErrorModeLenient`, `ErrorModeInteractive`
3. Write tests for a `DefaultErrorHandler` that implements different behaviors for each mode
4. Write tests for error collection and reporting in lenient mode
5. Write tests for a `LegacyErrorHandler` that mimics the old `log.Fatal` behavior for backward compatibility

Focus on:
- Strict mode: fail fast on any error
- Lenient mode: collect errors and continue where possible
- Legacy mode: maintain backward compatibility
- Clear behavior differences between modes with comprehensive test coverage

The error handler should work with all error types from Phases 1-2 and provide a foundation for integration with the main library."

### Task 8: Recovery Strategy Framework
- [x] **Objective**: Implement error recovery and fallback mechanisms.

**Test-driven prompt**:
"Building on Task 7, write tests first then implement error recovery strategies:

1. Write tests for a `RecoveryHandler` interface with methods: `Recover(err OutputError) error`, `CanRecover(err OutputError) bool`
2. Write tests for a `RecoveryStrategy` interface with methods: `Apply(err OutputError, context interface{}) (interface{}, error)`, `ApplicableFor(err OutputError) bool`
3. Write tests for `FormatFallbackStrategy` that falls back to simpler output formats
4. Write tests for `DefaultValueStrategy` that provides default values for missing data
5. Write tests for `RetryStrategy` with exponential backoff for transient errors

Focus on:
- Automatic recovery from common error conditions
- Configurable fallback chains (e.g., table → CSV → JSON)
- Integration with error handlers from Task 7
- Metrics and reporting on recovery attempts

Each strategy should be testable in isolation and composable with others."

### Task 9: OutputArray Error Integration
- [x] **Objective**: Integrate error handling into the main OutputArray type.

**Test-driven prompt**:
"Building on Tasks 7-8, write tests first then integrate error handling into the main library:

1. Write tests for adding `validators []Validator` and `errorHandler ErrorHandler` fields to `OutputArray`
2. Write tests for a new `Validate() error` method that runs all validators
3. Write tests for updating the `Write() error` method to use error handling instead of `log.Fatal`
4. Write tests for `AddValidator(v Validator)` and `WithErrorHandler(h ErrorHandler)` methods
5. Write tests for backward compatibility with a `WriteCompat()` method that maintains old behavior

Focus on:
- Gradual validation before expensive operations
- Seamless integration with existing OutputArray functionality
- Backward compatibility for existing users
- Clear migration path from old to new error handling

The integration should not break existing functionality while enabling the new error handling features."

## Phase 4: Settings and Format Integration

### Task 10: OutputSettings Validation
- [x] **Objective**: Add comprehensive validation to OutputSettings.

**Test-driven prompt**:
"Write tests first, then implement validation for OutputSettings:

1. Write tests for a `Validate() error` method on `OutputSettings` that validates all configuration
2. Write tests for format-specific validation (mermaid requires FromToColumns, etc.)
3. Write tests for file output validation (path exists, permissions, etc.)
4. Write tests for S3 configuration validation (credentials, bucket access, etc.)
5. Write tests for cross-field validation (incompatible option combinations)

Focus on:
- Early detection of configuration problems
- Format-specific requirement validation
- External dependency validation (files, S3)
- Clear error messages with suggestions for fixes

Use the validation framework from Phase 2 and error types from Phase 1 for consistent error handling."

### Task 11: Format-Specific Error Handling
- [x] **Objective**: Add error handling to individual output formats.

**Test-driven prompt**:
"Building on previous tasks, write tests first then add error handling to output format implementations:

1. Write tests for error handling in table format generation (column width calculation, alignment issues)
2. Write tests for error handling in CSV format (escaping, encoding issues)
3. Write tests for error handling in JSON format (serialization, type conversion issues)
4. Write tests for error handling in mermaid format (graph validation, syntax issues)
5. Write tests for error handling in dot format (GraphViz syntax, node/edge validation)

Focus on:
- Format-specific error conditions and recovery
- Detailed error context showing which part of generation failed
- Integration with recovery strategies for fallback formats
- Performance considerations for error checking in generation loops

Each format should use the established error types and provide meaningful error messages for format-specific issues."

## Phase 5: Advanced Features and Integration

### Task 12: Error Reporting and Metrics
- [ ] **Objective**: Add comprehensive error reporting and metrics.

**Test-driven prompt**:
"Write tests first, then implement error reporting and metrics:

1. Write tests for an `ErrorReporter` interface with methods: `Report(err OutputError)`, `Summary() ErrorSummary`
2. Write tests for `ErrorSummary` struct with fields: `TotalErrors int`, `ByCategory map[ErrorCode]int`, `BySeverity map[ErrorSeverity]int`, etc.
3. Write tests for structured error output (JSON, formatted text)
4. Write tests for error metrics collection and reporting
5. Write tests for integration with observability tools (structured logging)

Focus on:
- Comprehensive error statistics and categorization
- Multiple output formats for different use cases
- Integration with monitoring and alerting systems
- Performance impact of error reporting

The reporting system should work with all error types and provide actionable insights for debugging and monitoring."

### Task 13: Interactive Error Handling
- [ ] **Objective**: Implement interactive error resolution.

**Test-driven prompt**:
"Building on previous tasks, write tests first then implement interactive error handling:

1. Write tests for user prompts when errors occur in interactive mode
2. Write tests for automatic fix suggestions and application
3. Write tests for retry mechanisms with user input
4. Write tests for configuration correction workflows
5. Write tests for graceful fallback when interaction isn't possible

Focus on:
- User-friendly error messages and prompts
- Automatic fix detection and application
- Graceful degradation to non-interactive mode
- Integration with existing error handling modes

The interactive system should enhance the user experience while maintaining all existing functionality."

## Phase 6: Final Integration and Polish

### Task 14: Complete Integration and Migration Support
- [ ] **Objective**: Wire all components together and provide migration tools.

**Test-driven prompt**:
"Write comprehensive integration tests and migration support:

1. Write integration tests that use all error handling components together
2. Write tests for migration helpers that ease transition from log.Fatal
3. Write tests for performance impact of error handling on large datasets
4. Write tests for memory usage and error object lifecycle
5. Write tests for complex scenarios combining validation, recovery, and reporting

Focus on:
- End-to-end scenarios with realistic data and configurations
- Performance benchmarks comparing old vs new error handling
- Memory efficiency and garbage collection impact
- Migration documentation and examples

This final task should demonstrate that all components work together seamlessly and provide a clear path for users to adopt the new error handling system."

### Task 15: Documentation and Examples
- [ ] **Objective**: Create comprehensive documentation and usage examples.

**Test-driven prompt**:
"Write comprehensive documentation and examples for the error handling system:

1. Create usage examples showing basic error handling patterns
2. Create migration examples showing how to move from log.Fatal to new error handling
3. Create advanced examples showing custom validators and recovery strategies
4. Create troubleshooting guides for common error scenarios
5. Create performance tuning guides for large-scale usage

Focus on:
- Clear, copy-paste examples for common use cases
- Step-by-step migration instructions
- Best practices for error handling in production
- Performance optimization techniques

The documentation should make it easy for users to understand and adopt the new error handling features."

## Implementation Notes

### Order Dependency
- Tasks must be completed in order as each builds on previous tasks
- Phase 1 provides the foundation for all subsequent phases
- Phase 2 depends on Phase 1 error types
- Phase 3 depends on Phases 1-2
- Phases 4-6 integrate everything together

### Testing Strategy
- Every task starts with writing comprehensive tests
- Tests should cover both success and failure scenarios
- Include edge cases and error conditions
- Maintain high test coverage throughout

### Integration Points
- Each task should integrate cleanly with previous tasks
- No orphaned code - everything must be wired together
- Backward compatibility maintained throughout
- Clear migration path for existing users

### Performance Considerations
- Validation overhead should be minimal
- Error creation should not allocate excessively
- Large dataset performance must be maintained
- Memory usage should be reasonable

This task list provides a complete, incremental approach to implementing the error handling system with test-driven development, ensuring high quality and maintainable code.