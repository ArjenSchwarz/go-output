# Error Handling Examples

This directory contains comprehensive examples and documentation for the go-output library's error handling system.

## Files Overview

### Examples
- **`basic_error_handling.go`** - Basic error handling patterns and usage
- **`migration_examples.go`** - Migration from log.Fatal to structured error handling
- **`advanced_examples.go`** - Advanced features including custom validators and recovery strategies

### Documentation
- **`TROUBLESHOOTING.md`** - Common error scenarios and solutions
- **`PERFORMANCE_GUIDE.md`** - Performance optimization techniques

## Running the Examples

### Prerequisites
```bash
go mod download
```

### Basic Error Handling
```bash
cd examples/error-handling
go run basic_error_handling.go
```

This example demonstrates:
- Basic error handling with validation
- Different error modes (strict, lenient, interactive)
- Error collection and reporting
- Error context and suggestions

### Migration Examples
```bash
go run migration_examples.go
```

This example shows:
- Migrating from log.Fatal to error returns
- Adding proper error handling to callers
- Implementing validation before operations
- Using recovery strategies
- Migration helper utilities

### Advanced Examples
```bash
go run advanced_examples.go
```

This example covers:
- Custom validators for business rules
- Custom recovery strategies
- Interactive error handling
- Error reporting and metrics
- Complex validation chains

## Key Concepts

### Error Modes
- **Strict Mode**: Fail fast on any error (default)
- **Lenient Mode**: Collect errors and continue where possible
- **Interactive Mode**: Prompt user for error resolution

### Error Types
- **OutputError**: Base error interface with code, severity, context, and suggestions
- **ValidationError**: Extends OutputError with validation violations
- **ProcessingError**: Extends OutputError with retry and partial result capabilities

### Error Codes
- **1xxx**: Configuration errors (invalid format, missing settings)
- **2xxx**: Validation errors (missing columns, invalid data)
- **3xxx**: Processing errors (file write, S3 upload, template rendering)

### Recovery Strategies
- **Format Fallback**: Automatically fall back to simpler formats
- **Default Values**: Provide default values for missing data
- **Retry**: Retry operations with exponential backoff

## Common Patterns

### Basic Error Handling
```go
if err := output.WriteWithValidation(); err != nil {
    if outputErr, ok := err.(errors.OutputError); ok {
        fmt.Printf("Error: %s\n", outputErr.Code())
        for _, suggestion := range outputErr.Suggestions() {
            fmt.Printf("Try: %s\n", suggestion)
        }
    }
}
```

### Validation
```go
output.AddValidator(validators.NewRequiredColumnsValidator([]string{"Name", "Email"}))
output.AddValidator(validators.NewDataTypeValidator(typeMap))

if err := output.Validate(); err != nil {
    return err
}
```

### Recovery
```go
recovery := errors.NewDefaultRecoveryHandler()
recovery.AddStrategy(errors.NewFormatFallbackStrategy([]string{"table", "csv", "json"}))
output.WithRecoveryHandler(recovery)
```

### Migration
```go
// Old way
output.Write() // Would call log.Fatal on errors

// New way
if err := output.WriteWithValidation(); err != nil {
    log.Printf("Error: %v", err)
    return err
}
```

## Best Practices

1. **Validate Early**: Use `output.Validate()` before expensive operations
2. **Choose Appropriate Mode**: Strict for development, lenient for production
3. **Add Context**: Use `WithContext()` and `WithSuggestions()` for detailed errors
4. **Use Recovery**: Implement recovery strategies for resilient applications
5. **Monitor Errors**: Collect error metrics for monitoring and alerting

## Migration Guide

### Step 1: Replace log.Fatal
```go
// Before
if err != nil {
    log.Fatal(err)
}

// After
if err != nil {
    return err
}
```

### Step 2: Add Error Handling
```go
// Before
processData(data)

// After
if err := processData(data); err != nil {
    handleError(err)
}
```

### Step 3: Add Validation
```go
// Before
output.Write()

// After
if err := output.Validate(); err != nil {
    return err
}
return output.WriteWithValidation()
```

### Step 4: Use New Features
```go
output.SetErrorMode(errors.ErrorModeLenient)
output.AddValidator(customValidator)
output.WithRecoveryHandler(recoveryHandler)
```

## Troubleshooting

Common issues and solutions:

1. **Invalid Format Error**: Check supported formats in documentation
2. **Missing Configuration**: Validate settings before use
3. **Validation Failures**: Check data structure and types
4. **Performance Issues**: Use performance profiling and optimization techniques

See `TROUBLESHOOTING.md` for detailed solutions.

## Performance Optimization

Key optimization techniques:

1. **Reuse Validators**: Cache validator instances
2. **Minimize Error Creation**: Reuse error instances where possible
3. **Choose Right Mode**: Use appropriate error mode for use case
4. **Limit Error Context**: Keep error context minimal
5. **Monitor Performance**: Use profiling to identify bottlenecks

See `PERFORMANCE_GUIDE.md` for detailed optimization strategies.

## Getting Help

- Check the troubleshooting guide for common issues
- Review the performance guide for optimization techniques
- Create an issue with detailed error information
- Include error codes, context, and reproduction steps