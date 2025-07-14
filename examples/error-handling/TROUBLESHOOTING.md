# Error Handling Troubleshooting Guide

This guide helps you diagnose and fix common error handling issues in the go-output library.

## Common Error Scenarios

### 1. Configuration Errors

#### Invalid Output Format
**Error**: `[OUT-1001] Invalid output format: xml`

**Cause**: Attempting to use an unsupported output format.

**Solution**:
```go
// Check supported formats
validFormats := []string{"json", "yaml", "csv", "html", "table", "markdown", "mermaid", "drawio", "dot"}

// Use a valid format
settings.SetOutputFormat("json")
```

**Prevention**:
- Always validate format strings before setting
- Use constants for format names
- Check documentation for supported formats

#### Missing Required Configuration
**Error**: `[OUT-1002] mermaid format requires FromToColumns or MermaidSettings`

**Cause**: Format-specific configuration is missing.

**Solution**:
```go
// For mermaid format, provide required configuration
settings.SetOutputFormat("mermaid")

// Option 1: Use FromToColumns for flowcharts
settings.AddFromToColumns("Source", "Target")

// Option 2: Use MermaidSettings for other chart types
settings.MermaidSettings = &mermaid.Settings{
    ChartType: "piechart",
}
```

**Prevention**:
- Read format-specific documentation
- Use `settings.Validate()` before processing
- Set up configuration validation in tests

#### Invalid File Path
**Error**: `[OUT-1004] Output directory does not exist: /invalid/path`

**Cause**: Output directory doesn't exist or is not writable.

**Solution**:
```go
// Check if directory exists
if _, err := os.Stat(filepath.Dir(outputPath)); os.IsNotExist(err) {
    // Create directory
    if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
        log.Fatal(err)
    }
}

settings.OutputFile = outputPath
```

**Prevention**:
- Validate file paths before use
- Create directories if they don't exist
- Use absolute paths when possible

### 2. Data Validation Errors

#### Missing Required Columns
**Error**: `[OUT-2001] Missing required columns: [Email, Department]`

**Cause**: Required data columns are missing from the dataset.

**Solution**:
```go
// Check data structure before processing
requiredColumns := []string{"Email", "Department"}
for _, col := range requiredColumns {
    found := false
    for _, key := range output.Keys {
        if key == col {
            found = true
            break
        }
    }
    if !found {
        return fmt.Errorf("missing required column: %s", col)
    }
}

// Or use a validator
output.AddValidator(validators.NewRequiredColumnsValidator(requiredColumns))
```

**Prevention**:
- Validate data structure early
- Use validators for consistent checking
- Document required data format

#### Invalid Data Type
**Error**: `[OUT-2002] Invalid data type for field 'Age': expected number, got string`

**Cause**: Data type doesn't match expected format.

**Solution**:
```go
// Type conversion
if ageStr, ok := data["Age"].(string); ok {
    if age, err := strconv.Atoi(ageStr); err == nil {
        data["Age"] = age
    }
}

// Or use a validator
output.AddValidator(validators.NewDataTypeValidator(map[string]reflect.Type{
    "Age": reflect.TypeOf(0),
}))
```

**Prevention**:
- Validate data types before processing
- Use type conversion where appropriate
- Define clear data schemas

#### Empty Dataset
**Error**: `[OUT-2004] Cannot process empty dataset`

**Cause**: No data provided for processing.

**Solution**:
```go
// Check for empty data
if len(output.Contents) == 0 {
    return errors.NewValidationError(
        errors.ErrEmptyDataset,
        "no data to process",
    ).WithSuggestions(
        "Add data using AddContents() or AddHolder()",
        "Check data source for available records",
    )
}
```

**Prevention**:
- Always check for empty datasets
- Provide meaningful error messages
- Consider default values for empty data

### 3. Processing Errors

#### File Write Failures
**Error**: `[OUT-3001] Failed to write output file: permission denied`

**Cause**: Insufficient permissions or disk full.

**Solution**:
```go
// Check permissions before writing
if err := checkWritePermissions(outputPath); err != nil {
    return fmt.Errorf("cannot write to %s: %w", outputPath, err)
}

// Handle disk space issues
if err := checkDiskSpace(outputPath); err != nil {
    return fmt.Errorf("insufficient disk space: %w", err)
}
```

**Prevention**:
- Check file permissions early
- Monitor disk space
- Use temporary files for large outputs

#### S3 Upload Failures
**Error**: `[OUT-3002] S3 upload failed: access denied`

**Cause**: Invalid AWS credentials or bucket permissions.

**Solution**:
```go
// Validate S3 configuration
if err := validateS3Config(s3Config); err != nil {
    return fmt.Errorf("S3 configuration invalid: %w", err)
}

// Test connectivity
if err := testS3Connection(s3Config); err != nil {
    return fmt.Errorf("S3 connection failed: %w", err)
}
```

**Prevention**:
- Validate AWS credentials
- Test S3 connectivity before upload
- Use IAM roles with minimal permissions

#### Template Rendering Errors
**Error**: `[OUT-3003] Template rendering failed: unknown field 'InvalidField'`

**Cause**: Template references non-existent data fields.

**Solution**:
```go
// Validate template fields
if err := validateTemplateFields(template, data); err != nil {
    return fmt.Errorf("template validation failed: %w", err)
}

// Provide safe fallbacks
templateData := map[string]interface{}{
    "Title":   getOrDefault(data, "Title", "Untitled"),
    "Content": getOrDefault(data, "Content", "No content available"),
}
```

**Prevention**:
- Validate template fields
- Use safe fallbacks
- Test templates with sample data

### 4. Memory and Performance Issues

#### Memory Exhaustion
**Error**: `[OUT-3004] Memory exhausted while processing large dataset`

**Cause**: Processing dataset too large for available memory.

**Solution**:
```go
// Process data in chunks
chunkSize := 1000
for i := 0; i < len(largeDataset); i += chunkSize {
    end := i + chunkSize
    if end > len(largeDataset) {
        end = len(largeDataset)
    }
    
    chunk := largeDataset[i:end]
    if err := processChunk(chunk); err != nil {
        return err
    }
}
```

**Prevention**:
- Monitor memory usage
- Process data in chunks
- Use streaming for large datasets

#### Slow Processing
**Issue**: Processing takes too long for large datasets.

**Solution**:
```go
// Use performance profiling
profiler := errors.NewPerformanceProfiler()
profiler.Enable()

err := profiler.ProfileOperation("data_processing", func() error {
    return processLargeDataset(data)
})

// Check performance metrics
report := profiler.PerformanceReport()
fmt.Printf("Processing took: %v\n", report.Operations["data_processing"].AverageTime)
```

**Prevention**:
- Use performance profiling
- Optimize data structures
- Consider parallel processing

## Error Handling Best Practices

### 1. Error Mode Selection

```go
// Development: Use strict mode for immediate feedback
output.SetErrorMode(errors.ErrorModeStrict)

// Production: Use lenient mode for better resilience
output.SetErrorMode(errors.ErrorModeLenient)

// Interactive tools: Use interactive mode for user guidance
output.SetErrorMode(errors.ErrorModeInteractive)
```

### 2. Validation Strategy

```go
// Validate early and often
func processData(data []map[string]interface{}) error {
    // 1. Validate input structure
    if len(data) == 0 {
        return errors.NewValidationError(errors.ErrEmptyDataset, "no data provided")
    }
    
    // 2. Validate configuration
    if err := settings.Validate(); err != nil {
        return fmt.Errorf("configuration invalid: %w", err)
    }
    
    // 3. Validate data before processing
    if err := output.Validate(); err != nil {
        return fmt.Errorf("data validation failed: %w", err)
    }
    
    // 4. Process with error handling
    return output.WriteWithValidation()
}
```

### 3. Recovery Strategies

```go
// Set up recovery for common failures
recovery := errors.NewDefaultRecoveryHandler()

// Format fallback: table → csv → json
recovery.AddStrategy(errors.NewFormatFallbackStrategy([]string{"table", "csv", "json"}))

// Default values for missing data
recovery.AddStrategy(errors.NewDefaultValueStrategy(map[string]interface{}{
    "Status": "Unknown",
    "Value":  0,
}))

// Retry for transient errors
recovery.AddStrategy(errors.NewRetryStrategy(3, time.Second))

output.WithRecoveryHandler(recovery)
```

### 4. Error Reporting

```go
// Collect error metrics
reporter := format.NewDefaultErrorReporter()

// Process with error collection
if err := output.WriteWithValidation(); err != nil {
    reporter.Report(err)
}

// Generate comprehensive report
summary := reporter.Summary()
fmt.Printf("Error Summary: %d total, %d fixable\n", 
    summary.TotalErrors, summary.FixableErrors)

// Export for monitoring
if err := exportErrorMetrics(summary); err != nil {
    log.Printf("Failed to export metrics: %v", err)
}
```

## Debugging Techniques

### 1. Enable Debug Logging

```go
// Enable detailed error context
err := output.WriteWithValidation()
if outputErr, ok := err.(errors.OutputError); ok {
    context := outputErr.Context()
    log.Printf("Error context: %+v", context)
}
```

### 2. Use Error Inspection

```go
// Inspect error details
func inspectError(err error) {
    if outputErr, ok := err.(errors.OutputError); ok {
        fmt.Printf("Code: %s\n", outputErr.Code())
        fmt.Printf("Severity: %s\n", outputErr.Severity())
        fmt.Printf("Context: %+v\n", outputErr.Context())
        fmt.Printf("Suggestions: %v\n", outputErr.Suggestions())
    }
}
```

### 3. Validation Testing

```go
// Test validation in isolation
func testValidation(t *testing.T) {
    output := createTestOutput()
    
    // Test with valid data
    validData := createValidTestData()
    output.AddContents(validData)
    
    if err := output.Validate(); err != nil {
        t.Errorf("Valid data failed validation: %v", err)
    }
    
    // Test with invalid data
    invalidData := createInvalidTestData()
    output.AddContents(invalidData)
    
    if err := output.Validate(); err == nil {
        t.Error("Invalid data passed validation")
    }
}
```

## Migration Troubleshooting

### 1. log.Fatal Replacement Issues

**Problem**: Code still using log.Fatal after migration.

**Solution**:
```go
// Use migration helper to find remaining issues
helper := errors.NewLegacyMigrationHelper()
helper.AnalyzeCode(sourceCode)

report := helper.GetMigrationReport()
fmt.Println(report)
```

### 2. Backward Compatibility

**Problem**: Existing code breaks with new error handling.

**Solution**:
```go
// Use compatibility methods during transition
output.WriteCompat() // Uses log.Fatal for backward compatibility

// Or enable legacy mode
output.EnableLegacyMode()
if err := output.WriteWithValidation(); err != nil {
    log.Fatal(err) // Temporary during migration
}
```

### 3. Testing Migration

**Problem**: Tests fail after migration.

**Solution**:
```go
// Update tests to expect errors
func TestOutput(t *testing.T) {
    output := createTestOutput()
    
    // Old test (with log.Fatal)
    // output.Write() // Would call log.Fatal
    
    // New test (with error handling)
    if err := output.WriteWithValidation(); err != nil {
        if !isExpectedError(err) {
            t.Errorf("Unexpected error: %v", err)
        }
    }
}
```

## Performance Optimization

### 1. Reduce Validation Overhead

```go
// Cache validators
var (
    emailValidator = validators.NewEmailValidator()
    rangeValidator = validators.NewRangeValidator(0, 100)
)

// Reuse validators
output.AddValidator(emailValidator)
output.AddValidator(rangeValidator)
```

### 2. Optimize Error Creation

```go
// Minimize error creation in hot paths
func processItem(item interface{}) error {
    // Quick validation first
    if item == nil {
        return errors.ErrNilValue // Reuse error instances
    }
    
    // Expensive validation only when needed
    if needsDetailedValidation(item) {
        return validateDetailed(item)
    }
    
    return nil
}
```

### 3. Memory Management

```go
// Clear errors periodically
output.ClearErrors()

// Use object pooling for frequent operations
var errorPool = sync.Pool{
    New: func() interface{} {
        return &errors.BaseError{}
    },
}
```

## Getting Help

### 1. Error Code Reference

All error codes follow the pattern `OUT-XXXX`:
- 1xxx: Configuration errors
- 2xxx: Validation errors  
- 3xxx: Processing errors
- 4xxx: System errors

### 2. Debug Information

When reporting issues, include:
- Error code and message
- Error context and suggestions
- Sample data that reproduces the issue
- Configuration settings used
- Go version and OS information

### 3. Common Solutions

| Error Pattern | Common Cause | Quick Fix |
|---------------|--------------|-----------|
| `OUT-1001` | Invalid format | Use supported format |
| `OUT-1002` | Missing config | Add required configuration |
| `OUT-2001` | Missing columns | Check data structure |
| `OUT-2002` | Type mismatch | Convert data types |
| `OUT-3001` | File write error | Check permissions |
| `OUT-3002` | S3 error | Validate AWS credentials |

For additional help, check the documentation or create an issue with detailed error information.