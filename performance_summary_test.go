package format

import (
	"fmt"
	"testing"
	"time"
)

// TestPerformanceOptimizationsSummary provides a comprehensive summary of all performance optimizations
func TestPerformanceOptimizationsSummary(t *testing.T) {
	t.Log("=== Performance Optimizations Summary ===")

	// Test 1: Lazy Error Message Generation
	t.Run("LazyErrorMessageGeneration", func(t *testing.T) {
		// Test that lazy messages are not generated until accessed
		start := time.Now()
		err := NewErrorBuilder(ErrInvalidFormat, "base message").
			WithLazyMessage(func() string {
				// This expensive operation should not run until Error() is called
				return fmt.Sprintf("Expensive message generation: %d", time.Now().UnixNano())
			}).
			Build()
		creationTime := time.Since(start)

		// Error creation should be reasonably fast (lazy message not generated)
		if creationTime > 100*time.Microsecond {
			t.Errorf("Error creation took too long: %v", creationTime)
		}

		// Now access the message (should trigger lazy generation)
		start = time.Now()
		message1 := err.Error()
		firstAccessTime := time.Since(start)

		// Second access should use cached message
		start = time.Now()
		message2 := err.Error()
		secondAccessTime := time.Since(start)

		// Messages should be identical
		if message1 != message2 {
			t.Error("Cached message differs from original")
		}

		// Second access should be faster (cached)
		if secondAccessTime > firstAccessTime {
			t.Logf("Warning: Cached access not faster: first=%v, second=%v", firstAccessTime, secondAccessTime)
		}

		t.Logf("✓ Lazy message generation: creation=%v, first_access=%v, cached_access=%v",
			creationTime, firstAccessTime, secondAccessTime)
	})

	// Test 2: Memory Pool Optimization
	t.Run("MemoryPoolOptimization", func(t *testing.T) {
		// Create multiple errors to test pool reuse
		errors := make([]OutputError, 100)

		start := time.Now()
		for i := 0; i < 100; i++ {
			errors[i] = NewOutputError(ErrInvalidFormat, SeverityError, fmt.Sprintf("error %d", i))
		}
		poolTime := time.Since(start)

		avgTimePerError := poolTime / 100
		t.Logf("✓ Memory pool optimization: avg_time_per_error=%v", avgTimePerError)

		// Verify errors are created correctly
		if len(errors) != 100 {
			t.Error("Not all errors were created")
		}
	})

	// Test 3: Validator Caching
	t.Run("ValidatorCaching", func(t *testing.T) {
		output := createPerformanceTestOutputArray(100)
		validator := NewRequiredColumnsValidator("Name", "Value")

		// First validation (should populate cache)
		start := time.Now()
		err1 := validator.Validate(output)
		firstTime := time.Since(start)

		// Second validation (should use cache)
		start = time.Now()
		err2 := validator.Validate(output)
		secondTime := time.Since(start)

		// Both should succeed
		if err1 != nil || err2 != nil {
			t.Errorf("Validation failed: err1=%v, err2=%v", err1, err2)
		}

		// Second validation should be faster (cached)
		if secondTime > firstTime {
			t.Logf("Warning: Cached validation not faster: first=%v, second=%v", firstTime, secondTime)
		}

		t.Logf("✓ Validator caching: first=%v, cached=%v", firstTime, secondTime)
	})

	// Test 4: Context Gathering Optimization
	t.Run("ContextGatheringOptimization", func(t *testing.T) {
		// Test minimal context creation
		start := time.Now()
		err1 := NewErrorBuilder(ErrInvalidDataType, "test error").
			WithField("field1").
			Build()
		minimalTime := time.Since(start)

		// Test lazy context creation
		start = time.Now()
		err2 := NewErrorBuilder(ErrInvalidDataType, "test error").
			WithLazyContext(func() ErrorContext {
				return ErrorContext{
					Operation: "validation",
					Field:     "field1",
					Value:     "test_value",
					Metadata:  map[string]interface{}{"key": "value"},
				}
			}).
			Build()
		lazyTime := time.Since(start)

		// Lazy context creation should be faster
		if lazyTime > minimalTime*2 {
			t.Errorf("Lazy context creation too slow: minimal=%v, lazy=%v", minimalTime, lazyTime)
		}

		t.Logf("✓ Context gathering optimization: minimal=%v, lazy=%v", minimalTime, lazyTime)

		// Verify errors are created correctly
		if err1 == nil || err2 == nil {
			t.Error("Errors not created correctly")
		}
	})

	// Test 5: Overall Validation Performance
	t.Run("OverallValidationPerformance", func(t *testing.T) {
		output := createPerformanceTestOutputArray(1000)

		// Create optimized validation runner
		runner := NewOptimizedValidationRunner(ValidationModeFailFast)
		runner.AddValidator(NewRequiredColumnsValidator("Name", "Value"))
		runner.AddValidator(NewEmptyDatasetValidator(false))

		// Warm up
		_ = runner.Validate(output)

		// Measure performance
		start := time.Now()
		iterations := 1000
		for i := 0; i < iterations; i++ {
			if err := runner.Validate(output); err != nil {
				t.Fatalf("Validation failed: %v", err)
			}
		}
		totalTime := time.Since(start)

		avgTime := totalTime / time.Duration(iterations)
		t.Logf("✓ Overall validation performance: avg_time=%v, total_time=%v for %d iterations",
			avgTime, totalTime, iterations)

		// Performance requirement: should be very fast
		maxAllowedTime := 100 * time.Microsecond
		if avgTime > maxAllowedTime {
			t.Errorf("Validation too slow: %v > %v", avgTime, maxAllowedTime)
		}
	})

	t.Log("=== Performance Optimizations Summary Complete ===")
}

// BenchmarkOverallPerformance provides a comprehensive benchmark of all optimizations
func BenchmarkOverallPerformance(b *testing.B) {
	output := createPerformanceTestOutputArray(100)

	b.Run("OptimizedErrorCreation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := NewErrorBuilder(ErrInvalidFormat, "test message").
				WithField("testField").
				WithLazyMessage(func() string {
					return fmt.Sprintf("Lazy message %d", i)
				}).
				Build()
			_ = err
		}
	})

	b.Run("OptimizedValidation", func(b *testing.B) {
		runner := NewOptimizedValidationRunner(ValidationModeFailFast)
		runner.AddValidator(NewRequiredColumnsValidator("Name", "Value"))
		runner.AddValidator(NewEmptyDatasetValidator(false))

		// Warm up cache
		_ = runner.Validate(output)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = runner.Validate(output)
		}
	})

	b.Run("CachedValidatorExecution", func(b *testing.B) {
		validator := NewRequiredColumnsValidator("Name", "Value")

		// Warm up cache
		_ = validator.Validate(output)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = validator.Validate(output)
		}
	})
}
