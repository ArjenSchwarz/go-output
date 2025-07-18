package format

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"
)

// BenchmarkErrorCreation benchmarks error creation performance
func BenchmarkErrorCreation(b *testing.B) {
	b.Run("BaseError", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewOutputError(ErrInvalidFormat, SeverityError, "test error message")
		}
	})

	b.Run("ValidationError", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewValidationError(ErrMissingColumn, "test validation error")
		}
	})

	b.Run("ProcessingError", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewProcessingError(ErrFileWrite, "test processing error", true)
		}
	})
}

// BenchmarkLazyErrorMessage benchmarks lazy error message generation
func BenchmarkLazyErrorMessage(b *testing.B) {
	// Create error with expensive message generation
	expensiveMessageFunc := func() string {
		var builder strings.Builder
		for i := 0; i < 100; i++ {
			builder.WriteString(fmt.Sprintf("expensive operation %d ", i))
		}
		return builder.String()
	}

	b.Run("EagerMessage", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := NewErrorBuilder(ErrInvalidFormat, expensiveMessageFunc()).Build()
			_ = err // Don't call Error() to avoid triggering message generation
		}
	})

	b.Run("LazyMessage", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := NewErrorBuilder(ErrInvalidFormat, "base message").
				WithLazyMessage(expensiveMessageFunc).
				Build()
			_ = err // Don't call Error() to avoid triggering message generation
		}
	})

	b.Run("LazyMessageWithAccess", func(b *testing.B) {
		err := NewErrorBuilder(ErrInvalidFormat, "base message").
			WithLazyMessage(expensiveMessageFunc).
			Build()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = err.Error() // This should use cached message after first call
		}
	})
}

// BenchmarkValidatorExecution benchmarks validator execution performance
func BenchmarkValidatorExecution(b *testing.B) {
	// Create test data
	output := createPerformanceTestOutputArray(1000) // 1000 rows

	validators := []struct {
		name      string
		validator Validator
	}{
		{"RequiredColumns", NewRequiredColumnsValidator("Name", "Value")},
		{"DataType", NewDataTypeValidator().WithStringColumn("Name").WithIntColumn("Value")},
		{"EmptyDataset", NewEmptyDatasetValidator(false)},
		{"MalformedData", NewMalformedDataValidator(false)},
	}

	for _, v := range validators {
		b.Run(v.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = v.validator.Validate(output)
			}
		})
	}
}

// BenchmarkValidatorOrdering benchmarks optimized vs unoptimized validator execution
func BenchmarkValidatorOrdering(b *testing.B) {
	output := createPerformanceTestOutputArray(100)

	// Create validators with different performance characteristics
	validators := []Validator{
		NewConstraintValidator().AddConstraint(PositiveNumberConstraint("Value")), // Expensive
		NewDataTypeValidator().WithStringColumn("Name").WithIntColumn("Value"),    // Medium cost
		NewMalformedDataValidator(true),                                           // Medium cost, strict mode
		NewRequiredColumnsValidator("Name", "Value"),                              // Fast, fail-fast
		NewEmptyDatasetValidator(false),                                           // Fast, fail-fast
	}

	b.Run("UnoptimizedOrder", func(b *testing.B) {
		runner := NewValidationRunner(ValidationModeFailFast)
		for _, v := range validators {
			runner.AddValidator(v)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = runner.Validate(output)
		}
	})

	b.Run("OptimizedOrder", func(b *testing.B) {
		runner := NewOptimizedValidationRunner(ValidationModeFailFast)
		for _, v := range validators {
			runner.AddValidator(v)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = runner.Validate(output)
		}
	})
}

// BenchmarkMemoryAllocation benchmarks memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("ErrorCreation", func(b *testing.B) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := NewErrorBuilder(ErrInvalidFormat, "test message").
				WithField("testField").
				WithOperation("testOperation").
				WithSuggestions("suggestion1", "suggestion2").
				Build()
			_ = err
		}
		b.StopTimer()

		runtime.GC()
		runtime.ReadMemStats(&m2)
		b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc)/float64(b.N), "bytes/op")
	})

	b.Run("ValidationErrorCreation", func(b *testing.B) {
		var m1, m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := NewValidationErrorBuilder(ErrMissingColumn, "test validation").
				WithViolation("field1", "constraint1", "message1", "value1").
				WithViolation("field2", "constraint2", "message2", "value2").
				Build()
			_ = err
		}
		b.StopTimer()

		runtime.GC()
		runtime.ReadMemStats(&m2)
		b.ReportMetric(float64(m2.TotalAlloc-m1.TotalAlloc)/float64(b.N), "bytes/op")
	})
}

// BenchmarkContextGathering benchmarks error context gathering performance
func BenchmarkContextGathering(b *testing.B) {
	testData := map[string]interface{}{
		"field1": "value1",
		"field2": 42,
		"field3": true,
		"field4": []string{"a", "b", "c"},
		"field5": map[string]interface{}{"nested": "value"},
	}

	b.Run("MinimalContext", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := NewErrorBuilder(ErrInvalidDataType, "test error").
				WithField("field1").
				Build()
			_ = err
		}
	})

	b.Run("RichContext", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			context := ErrorContext{
				Operation: "validation",
				Field:     "field1",
				Value:     testData["field1"],
				Index:     i,
				Metadata:  testData,
			}
			err := NewErrorBuilder(ErrInvalidDataType, "test error").
				WithContext(context).
				Build()
			_ = err
		}
	})

	b.Run("LazyContext", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := NewErrorBuilder(ErrInvalidDataType, "test error").
				WithLazyMessage(func() string {
					// Context gathering happens only when message is accessed
					return fmt.Sprintf("Error in field %s with value %v at index %d",
						"field1", testData["field1"], i)
				}).
				Build()
			_ = err
		}
	})
}

// BenchmarkValidationOverhead measures validation overhead compared to processing
func BenchmarkValidationOverhead(b *testing.B) {
	output := createPerformanceTestOutputArray(1000)

	// Simulate processing work
	processData := func() {
		time.Sleep(100 * time.Microsecond) // Simulate 100μs of processing
	}

	b.Run("ProcessingOnly", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			processData()
		}
	})

	b.Run("ProcessingWithValidation", func(b *testing.B) {
		runner := NewOptimizedValidationRunner(ValidationModeFailFast)
		runner.AddValidator(NewRequiredColumnsValidator("Name", "Value"))
		runner.AddValidator(NewEmptyDatasetValidator(false))

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = runner.Validate(output)
			processData()
		}
	})
}

// TestValidationOverheadRequirement tests that validation overhead is reasonable
func TestValidationOverheadRequirement(t *testing.T) {
	output := createPerformanceTestOutputArray(100)

	// Measure validation time in isolation
	runner := NewOptimizedValidationRunner(ValidationModeFailFast)
	runner.AddValidator(NewRequiredColumnsValidator("Name", "Value"))
	runner.AddValidator(NewEmptyDatasetValidator(false))

	// Warm up the cache
	_ = runner.Validate(output)

	// Measure validation time
	start := time.Now()
	iterations := 1000
	for i := 0; i < iterations; i++ {
		if err := runner.Validate(output); err != nil {
			t.Fatalf("Validation failed: %v", err)
		}
	}
	validationTime := time.Since(start)

	avgValidationTime := validationTime / time.Duration(iterations)
	t.Logf("Average validation time per iteration: %v", avgValidationTime)
	t.Logf("Total validation time for %d iterations: %v", iterations, validationTime)

	// Performance requirement: validation should be very fast
	// For a 100-row dataset, validation should take less than 100μs per iteration
	maxAllowedTime := 100 * time.Microsecond
	if avgValidationTime > maxAllowedTime {
		t.Errorf("Average validation time %v exceeds maximum allowed time %v", avgValidationTime, maxAllowedTime)
	}

	// Test that caching is working - second run should be faster
	start = time.Now()
	for i := 0; i < iterations; i++ {
		if err := runner.Validate(output); err != nil {
			t.Fatalf("Validation failed: %v", err)
		}
	}
	cachedValidationTime := time.Since(start)

	avgCachedTime := cachedValidationTime / time.Duration(iterations)
	t.Logf("Average cached validation time per iteration: %v", avgCachedTime)

	// Cached validation should be significantly faster (at least 20% improvement)
	if avgCachedTime > avgValidationTime*8/10 {
		t.Logf("Warning: Caching may not be working effectively. Cached: %v, Original: %v", avgCachedTime, avgValidationTime)
	}
}

// createPerformanceTestOutputArray creates a test OutputArray with specified number of rows
func createPerformanceTestOutputArray(rows int) *OutputArray {
	output := &OutputArray{
		Settings: &OutputSettings{},
		Keys:     []string{"Name", "Value"},
		Contents: make([]OutputHolder, rows),
	}

	for i := 0; i < rows; i++ {
		output.Contents[i] = OutputHolder{
			Contents: map[string]interface{}{
				"Name":  fmt.Sprintf("Item%d", i),
				"Value": i,
			},
		}
	}

	return output
}

// BenchmarkErrorHandlerModes benchmarks different error handling modes
func BenchmarkErrorHandlerModes(b *testing.B) {
	errors := []error{
		NewOutputError(ErrInvalidFormat, SeverityWarning, "warning 1"),
		NewOutputError(ErrMissingColumn, SeverityError, "error 1"),
		NewOutputError(ErrInvalidDataType, SeverityWarning, "warning 2"),
		NewOutputError(ErrConstraintViolation, SeverityError, "error 2"),
	}

	b.Run("StrictMode", func(b *testing.B) {
		handler := NewDefaultErrorHandler()
		handler.SetMode(ErrorModeStrict)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			handler.Clear()
			for _, err := range errors {
				_ = handler.HandleError(err)
			}
		}
	})

	b.Run("LenientMode", func(b *testing.B) {
		handler := NewDefaultErrorHandler()
		handler.SetMode(ErrorModeLenient)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			handler.Clear()
			for _, err := range errors {
				_ = handler.HandleError(err)
			}
		}
	})
}

// BenchmarkCompositeError benchmarks composite error performance
func BenchmarkCompositeError(b *testing.B) {
	errors := make([]error, 100)
	for i := 0; i < 100; i++ {
		errors[i] = NewOutputError(ErrInvalidFormat, SeverityError, fmt.Sprintf("error %d", i))
	}

	b.Run("CompositeErrorCreation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			composite := NewCompositeError()
			for _, err := range errors {
				composite.Add(err)
			}
		}
	})

	b.Run("CompositeErrorFormatting", func(b *testing.B) {
		composite := NewCompositeError()
		for _, err := range errors {
			composite.Add(err)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = composite.Error()
		}
	})
}
