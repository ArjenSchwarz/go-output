# Performance Optimization Guide for Error Handling

This guide provides best practices and techniques for optimizing error handling performance in the go-output library.

## Performance Principles

### 1. Error Handling Overhead
- Error creation: ~100ns per error
- Validation: ~500ns per validator per record
- Recovery: ~1-5ms per recovery attempt
- Reporting: ~50ns per error reported

### 2. Memory Usage
- Base error: ~200 bytes
- Validation error: ~400 bytes
- Error with context: ~600 bytes
- Composite error: ~1KB + 200 bytes per sub-error

## Optimization Strategies

### 1. Minimize Error Creation

#### Reuse Error Instances
```go
// Bad: Creating new errors in hot paths
func validateItem(item interface{}) error {
    if item == nil {
        return errors.NewValidationError(errors.ErrNilValue, "item cannot be nil")
    }
    return nil
}

// Good: Reuse error instances
var (
    ErrNilItem = errors.NewValidationError(errors.ErrNilValue, "item cannot be nil")
)

func validateItem(item interface{}) error {
    if item == nil {
        return ErrNilItem
    }
    return nil
}
```

#### Use Error Pools for Frequent Operations
```go
import "sync"

var errorPool = sync.Pool{
    New: func() interface{} {
        return &errors.BaseError{}
    },
}

func createOptimizedError(code errors.ErrorCode, message string) error {
    err := errorPool.Get().(*errors.BaseError)
    err.Reset(code, message)
    return err
}

func releaseError(err error) {
    if baseErr, ok := err.(*errors.BaseError); ok {
        errorPool.Put(baseErr)
    }
}
```

### 2. Optimize Validation

#### Validator Caching
```go
// Bad: Creating validators repeatedly
func processData(data []map[string]interface{}) error {
    for _, item := range data {
        validator := validators.NewEmailValidator() // Expensive!
        if err := validator.Validate(item); err != nil {
            return err
        }
    }
    return nil
}

// Good: Cache validators
var (
    emailValidator = validators.NewEmailValidator()
    rangeValidator = validators.NewRangeValidator(0, 100)
)

func processData(data []map[string]interface{}) error {
    for _, item := range data {
        if err := emailValidator.Validate(item); err != nil {
            return err
        }
    }
    return nil
}
```

#### Early Validation
```go
// Bad: Validate during processing
func processLargeDataset(data []map[string]interface{}) error {
    output := createOutput()
    
    for _, item := range data {
        // Expensive validation for each item
        if err := validateItem(item); err != nil {
            return err
        }
        output.AddContents(item)
    }
    
    return output.WriteWithValidation()
}

// Good: Validate once upfront
func processLargeDataset(data []map[string]interface{}) error {
    // Quick validation first
    if err := validateDataStructure(data); err != nil {
        return err
    }
    
    output := createOutput()
    
    // Fast processing without per-item validation
    for _, item := range data {
        output.AddContents(item)
    }
    
    return output.WriteWithValidation()
}
```

#### Conditional Validation
```go
// Bad: Always run expensive validation
func processItem(item map[string]interface{}) error {
    // Always runs expensive validation
    if err := expensiveValidation(item); err != nil {
        return err
    }
    return processItemFast(item)
}

// Good: Conditional validation
func processItem(item map[string]interface{}) error {
    // Quick checks first
    if err := quickValidation(item); err != nil {
        return err
    }
    
    // Expensive validation only when needed
    if needsDetailedValidation(item) {
        if err := expensiveValidation(item); err != nil {
            return err
        }
    }
    
    return processItemFast(item)
}
```

### 3. Optimize Error Modes

#### Choose the Right Mode
```go
// Development: Strict mode for immediate feedback
func developmentMode() {
    output.SetErrorMode(errors.ErrorModeStrict)
    // Fails fast, minimal overhead
}

// Production: Lenient mode for resilience
func productionMode() {
    output.SetErrorMode(errors.ErrorModeLenient)
    // Collects errors, higher throughput
}

// Batch processing: Collect all errors
func batchProcessing() {
    output.SetErrorMode(errors.ErrorModeLenient)
    // Process all data, handle errors at end
}
```

#### Optimize Error Collection
```go
// Bad: Collect all error details
func processWithFullErrorCollection(data []map[string]interface{}) error {
    output := createOutput()
    output.SetErrorMode(errors.ErrorModeLenient)
    
    for _, item := range data {
        if err := validateItem(item); err != nil {
            // Expensive error context creation
            enrichedErr := err.WithContext(errors.ErrorContext{
                Operation: "item_validation",
                Field:     "all_fields",
                Value:     item,
                Metadata:  map[string]interface{}{"index": i},
            })
            output.ReportError(enrichedErr)
        }
    }
    
    return output.WriteWithValidation()
}

// Good: Minimal error collection
func processWithMinimalErrorCollection(data []map[string]interface{}) error {
    output := createOutput()
    output.SetErrorMode(errors.ErrorModeLenient)
    
    errorCount := 0
    for _, item := range data {
        if err := validateItem(item); err != nil {
            errorCount++
            // Only collect essential error info
            if errorCount <= 10 { // Limit error collection
                output.ReportError(err)
            }
        }
    }
    
    return output.WriteWithValidation()
}
```

### 4. Memory Optimization

#### Limit Error Context
```go
// Bad: Large error context
func createErrorWithLargeContext(item map[string]interface{}) error {
    return errors.NewValidationError(
        errors.ErrInvalidDataType,
        "validation failed",
    ).WithContext(errors.ErrorContext{
        Operation: "data_validation",
        Field:     "item",
        Value:     item, // Potentially large object
        Metadata:  item, // Duplicate large data
    })
}

// Good: Minimal error context
func createErrorWithMinimalContext(item map[string]interface{}) error {
    return errors.NewValidationError(
        errors.ErrInvalidDataType,
        "validation failed",
    ).WithContext(errors.ErrorContext{
        Operation: "data_validation",
        Field:     "item",
        Value:     fmt.Sprintf("%T", item), // Just type info
        Metadata:  map[string]interface{}{"size": len(item)},
    })
}
```

#### Error Cleanup
```go
// Periodically clear collected errors
func processInBatches(data []map[string]interface{}) error {
    output := createOutput()
    batchSize := 1000
    
    for i := 0; i < len(data); i += batchSize {
        end := i + batchSize
        if end > len(data) {
            end = len(data)
        }
        
        batch := data[i:end]
        if err := processBatch(batch); err != nil {
            return err
        }
        
        // Clear errors after each batch
        output.ClearErrors()
    }
    
    return nil
}
```

### 5. Recovery Optimization

#### Efficient Recovery Strategies
```go
// Bad: Expensive recovery attempts
func inefficientRecovery() {
    recovery := errors.NewDefaultRecoveryHandler()
    
    // Expensive recovery strategies
    recovery.AddStrategy(&ExpensiveRecoveryStrategy{})
    recovery.AddStrategy(&AnotherExpensiveStrategy{})
    
    output.WithRecoveryHandler(recovery)
}

// Good: Fast recovery strategies
func efficientRecovery() {
    recovery := errors.NewDefaultRecoveryHandler()
    
    // Fast format fallback
    recovery.AddStrategy(errors.NewFormatFallbackStrategy([]string{"json"}))
    
    // Quick default values
    recovery.AddStrategy(errors.NewDefaultValueStrategy(map[string]interface{}{
        "status": "unknown",
    }))
    
    output.WithRecoveryHandler(recovery)
}
```

#### Recovery Limits
```go
// Limit recovery attempts
type LimitedRecoveryHandler struct {
    handler    errors.RecoveryHandler
    maxRetries int
    retryCount int
}

func (h *LimitedRecoveryHandler) Recover(err errors.OutputError) error {
    if h.retryCount >= h.maxRetries {
        return err // Stop recovery after limit
    }
    
    h.retryCount++
    return h.handler.Recover(err)
}
```

## Benchmarking and Profiling

### 1. Performance Profiling
```go
// Profile error handling performance
func profileErrorHandling() {
    profiler := errors.NewPerformanceProfiler()
    profiler.Enable()
    
    // Measure validation performance
    profiler.ProfileOperation("validation", func() error {
        return validateLargeDataset(testData)
    })
    
    // Measure error creation performance
    profiler.ProfileOperation("error_creation", func() error {
        for i := 0; i < 1000; i++ {
            err := errors.NewValidationError(errors.ErrInvalidFormat, "test")
            _ = err
        }
        return nil
    })
    
    // Generate performance report
    report := profiler.PerformanceReport()
    fmt.Printf("Performance Report:\n%s", report.String())
}
```

### 2. Memory Profiling
```go
import (
    "runtime"
    "testing"
)

func BenchmarkErrorHandling(b *testing.B) {
    var m1, m2 runtime.MemStats
    
    runtime.GC()
    runtime.ReadMemStats(&m1)
    
    for i := 0; i < b.N; i++ {
        err := errors.NewValidationError(errors.ErrInvalidFormat, "test")
        _ = err
    }
    
    runtime.GC()
    runtime.ReadMemStats(&m2)
    
    bytesPerOp := (m2.Alloc - m1.Alloc) / uint64(b.N)
    b.ReportMetric(float64(bytesPerOp), "bytes/op")
}
```

### 3. Benchmark Results
```go
// Example benchmark results
func ExampleBenchmarkResults() {
    fmt.Println(`
Benchmark Results:
BenchmarkErrorCreation-8           1000000    1000 ns/op    400 B/op    5 allocs/op
BenchmarkValidation-8              500000     2000 ns/op    800 B/op    8 allocs/op
BenchmarkErrorHandling-8           100000     10000 ns/op   2000 B/op   15 allocs/op
BenchmarkRecovery-8                10000      100000 ns/op  5000 B/op   25 allocs/op
`)
}
```

## Performance Monitoring

### 1. Runtime Monitoring
```go
// Monitor error handling performance in production
func monitorPerformance() {
    profiler := errors.NewPerformanceProfiler()
    profiler.Enable()
    
    // Set up periodic reporting
    go func() {
        ticker := time.NewTicker(1 * time.Minute)
        defer ticker.Stop()
        
        for range ticker.C {
            report := profiler.PerformanceReport()
            
            // Log performance metrics
            for operation, stats := range report.Operations {
                if stats.AverageTime > 100*time.Millisecond {
                    log.Printf("SLOW: %s took %v on average", operation, stats.AverageTime)
                }
            }
            
            // Clear metrics after reporting
            profiler.Clear()
        }
    }()
}
```

### 2. Error Rate Monitoring
```go
// Monitor error rates
func monitorErrorRates() {
    reporter := format.NewDefaultErrorReporter()
    
    // Track error rates over time
    go func() {
        ticker := time.NewTicker(5 * time.Minute)
        defer ticker.Stop()
        
        for range ticker.C {
            summary := reporter.Summary()
            
            // Alert on high error rates
            if summary.TotalErrors > 1000 {
                log.Printf("HIGH ERROR RATE: %d errors in last 5 minutes", summary.TotalErrors)
            }
            
            // Export metrics to monitoring system
            exportMetrics(summary)
            
            // Clear old errors
            reporter.Clear()
        }
    }()
}
```

## Best Practices Summary

### 1. Error Creation
- Reuse error instances for common errors
- Use error pools for high-frequency operations
- Minimize error context size
- Avoid creating errors in hot paths

### 2. Validation
- Cache validators instead of creating new ones
- Validate early and once
- Use conditional validation
- Implement fast-fail strategies

### 3. Error Modes
- Choose appropriate mode for use case
- Limit error collection in production
- Use strict mode for development
- Use lenient mode for batch processing

### 4. Recovery
- Implement fast recovery strategies
- Limit recovery attempts
- Use format fallbacks efficiently
- Avoid expensive recovery operations

### 5. Memory Management
- Clear errors periodically
- Limit error context size
- Use object pooling
- Monitor memory usage

### 6. Monitoring
- Profile error handling performance
- Monitor error rates
- Set up alerts for performance degradation
- Export metrics to monitoring systems

## Performance Targets

### Recommended Limits
- Error creation: < 1μs per error
- Validation: < 10μs per validator per record
- Error collection: < 100 errors per second
- Recovery: < 10ms per recovery attempt
- Memory usage: < 1KB per error with context

### Scale Testing
```go
// Test with different data sizes
func testPerformanceAtScale() {
    sizes := []int{100, 1000, 10000, 100000}
    
    for _, size := range sizes {
        data := generateTestData(size)
        
        start := time.Now()
        err := processDataWithErrorHandling(data)
        duration := time.Since(start)
        
        fmt.Printf("Size: %d, Time: %v, Errors: %v\n", size, duration, err != nil)
    }
}
```

By following these optimization techniques, you can maintain high performance while providing comprehensive error handling in your go-output applications.