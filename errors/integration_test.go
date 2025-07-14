package errors

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

// Test comprehensive integration of all error handling components
func TestIntegratedErrorSystem_EndToEnd(t *testing.T) {
	system := NewIntegratedErrorSystem()
	system.EnableProfiling()

	// Set up error handler in lenient mode
	system.handler.SetMode(ErrorModeLenient)

	// Add recovery strategies
	formatStrategy := NewFormatFallbackStrategy([]string{"table", "csv", "json"})
	defaultStrategy := NewDefaultValueStrategy(map[string]interface{}{
		"missing_field": "default_value",
	})
	
	system.recoveryHandler.AddStrategy(formatStrategy)
	system.recoveryHandler.AddStrategy(defaultStrategy)

	// Test various error scenarios
	testErrors := []error{
		NewError(ErrInvalidFormat, "format error"),
		NewValidationError(ErrMissingRequired, "missing field"),
		NewProcessingError(ErrFileWrite, "file write error"),
		NewError(ErrInvalidDataType, "type error").WithSeverity(SeverityWarning),
	}

	// Process all errors
	results := system.ProcessErrors(testErrors)

	// In lenient mode, only fatal errors should be returned
	if len(results) > 0 {
		t.Errorf("Expected no unrecoverable errors, got %d", len(results))
	}

	// Check error summary
	summary := system.GetErrorSummary()
	if summary.TotalErrors != len(testErrors) {
		t.Errorf("Error summary count = %d, want %d", summary.TotalErrors, len(testErrors))
	}

	// Check performance report
	report := system.GetPerformanceReport()
	if report.TotalOperations == 0 {
		t.Error("Expected performance metrics to be collected")
	}

	// Verify specific operations were profiled
	if _, exists := report.Operations["process_error"]; !exists {
		t.Error("Expected 'process_error' operation to be profiled")
	}

	// Check system health
	health := system.SystemHealthCheck()
	if !health.Healthy {
		t.Errorf("System health check failed: %v", health.Issues)
	}
}

func TestIntegratedErrorSystem_PerformanceWithLargeDataset(t *testing.T) {
	system := NewIntegratedErrorSystem()
	system.EnableProfiling()
	system.handler.SetMode(ErrorModeLenient)

	// Generate large number of errors to test performance
	errorCount := 1000
	errors := make([]error, errorCount)
	for i := 0; i < errorCount; i++ {
		if i%4 == 0 {
			errors[i] = NewError(ErrInvalidFormat, fmt.Sprintf("format error %d", i))
		} else if i%4 == 1 {
			errors[i] = NewValidationError(ErrMissingRequired, fmt.Sprintf("validation error %d", i))
		} else if i%4 == 2 {
			errors[i] = NewProcessingError(ErrFileWrite, fmt.Sprintf("processing error %d", i))
		} else {
			errors[i] = fmt.Errorf("regular error %d", i)
		}
	}

	// Measure processing time
	startTime := time.Now()
	results := system.ProcessErrors(errors)
	duration := time.Since(startTime)

	// Performance assertions
	if duration > time.Second {
		t.Errorf("Processing %d errors took too long: %v", errorCount, duration)
	}

	avgTimePerError := duration / time.Duration(errorCount)
	if avgTimePerError > time.Millisecond {
		t.Errorf("Average time per error too high: %v", avgTimePerError)
	}

	// Check that all errors were processed
	summary := system.GetErrorSummary()
	if summary.TotalErrors != errorCount {
		t.Errorf("Expected %d errors in summary, got %d", errorCount, summary.TotalErrors)
	}

	// In lenient mode, non-fatal errors should not be returned
	expectedResults := 0 // All test errors are non-fatal
	if len(results) != expectedResults {
		t.Errorf("Expected %d unprocessed errors, got %d", expectedResults, len(results))
	}

	t.Logf("Processed %d errors in %v (avg: %v per error)", 
		errorCount, duration, avgTimePerError)
}

func TestIntegratedErrorSystem_MemoryUsage(t *testing.T) {
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	system := NewIntegratedErrorSystem()
	system.handler.SetMode(ErrorModeLenient)

	// Create and process many errors to test memory usage
	errorCount := 5000
	for i := 0; i < errorCount; i++ {
		err := NewError(ErrInvalidFormat, fmt.Sprintf("error %d", i)).
			WithContext(ErrorContext{
				Operation: "test_operation",
				Field:     fmt.Sprintf("field_%d", i),
				Value:     i,
			}).
			WithSuggestions("suggestion 1", "suggestion 2")
		
		system.ProcessError(err)
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Check memory usage
	allocatedBytes := m2.Alloc - m1.Alloc
	bytesPerError := allocatedBytes / uint64(errorCount)

	// Memory usage should be reasonable (less than 1KB per error on average)
	maxBytesPerError := uint64(1024)
	if bytesPerError > maxBytesPerError {
		t.Errorf("Memory usage too high: %d bytes per error (max: %d)", 
			bytesPerError, maxBytesPerError)
	}

	t.Logf("Memory usage: %d bytes for %d errors (avg: %d bytes per error)", 
		allocatedBytes, errorCount, bytesPerError)

	// Test memory cleanup
	system.Clear()
	summary := system.GetErrorSummary()
	if summary.TotalErrors != 0 {
		t.Errorf("Expected 0 errors after clear, got %d", summary.TotalErrors)
	}
}

func TestMigrationHelper_CompleteWorkflow(t *testing.T) {
	helper := NewMigrationHelper()

	// Add migration steps
	step1 := MigrationStep{
		Name:        "Replace log.Fatal calls",
		Description: "Check if log.Fatal calls have been replaced with error returns",
		Check: func() (bool, string) {
			// Simulate checking for log.Fatal usage
			return false, "Found 3 log.Fatal calls in codebase"
		},
		Fix: func() error {
			// Simulate fixing log.Fatal calls
			return nil
		},
	}

	step2 := MigrationStep{
		Name:        "Add error handling",
		Description: "Check if proper error handling is implemented",
		Check: func() (bool, string) {
			return true, "Error handling is properly implemented"
		},
	}

	step3 := MigrationStep{
		Name:        "Update tests",
		Description: "Check if tests are updated for new error handling",
		Check: func() (bool, string) {
			return false, "Tests need to be updated for error handling"
		},
		Fix: func() error {
			// Simulate updating tests
			return nil
		},
	}

	helper.AddMigrationStep(step1)
	helper.AddMigrationStep(step2)
	helper.AddMigrationStep(step3)

	// Check initial status
	status := helper.CheckMigrationStatus()
	if status.TotalSteps != 3 {
		t.Errorf("Expected 3 total steps, got %d", status.TotalSteps)
	}
	if status.CompletedSteps != 1 {
		t.Errorf("Expected 1 completed step, got %d", status.CompletedSteps)
	}
	if status.FailedSteps != 2 {
		t.Errorf("Expected 2 failed steps, got %d", status.FailedSteps)
	}

	// Check completion percentage
	expectedPercentage := 33.3
	actualPercentage := status.CompletionPercentage()
	if actualPercentage < expectedPercentage-1 || actualPercentage > expectedPercentage+1 {
		t.Errorf("Expected completion percentage around %.1f%%, got %.1f%%", 
			expectedPercentage, actualPercentage)
	}

	// Run migration
	result := helper.RunMigration()
	if result.FixedSteps != 2 {
		t.Errorf("Expected 2 fixed steps, got %d", result.FixedSteps)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %d: %v", len(result.Errors), result.Errors)
	}

	// Check final status
	finalStatus := helper.CheckMigrationStatus()
	if !finalStatus.IsComplete() {
		t.Error("Expected migration to be complete after running fixes")
	}
}

func TestMigrationHelper_LegacyMode(t *testing.T) {
	helper := NewMigrationHelper()

	// Test legacy mode
	if helper.IsLegacyMode() {
		t.Error("Expected legacy mode to be disabled by default")
	}

	helper.EnableLegacyMode()
	if !helper.IsLegacyMode() {
		t.Error("Expected legacy mode to be enabled")
	}

	helper.DisableLegacyMode()
	if helper.IsLegacyMode() {
		t.Error("Expected legacy mode to be disabled")
	}
}

func TestPerformanceProfiler_Operations(t *testing.T) {
	profiler := NewPerformanceProfiler()

	// Test disabled profiling
	err := profiler.ProfileOperation("test_op", func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(profiler.GetMetrics()) != 0 {
		t.Error("Expected no metrics when profiling is disabled")
	}

	// Test enabled profiling
	profiler.Enable()

	err = profiler.ProfileOperation("test_op", func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	metrics := profiler.GetMetrics()
	if len(metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(metrics))
	}

	if metrics[0].Operation != "test_op" {
		t.Errorf("Expected operation 'test_op', got '%s'", metrics[0].Operation)
	}

	if metrics[0].Duration < 10*time.Millisecond {
		t.Errorf("Expected duration >= 10ms, got %v", metrics[0].Duration)
	}

	// Test error handling
	err = profiler.ProfileOperation("error_op", func() error {
		return fmt.Errorf("test error")
	})

	if err == nil {
		t.Error("Expected error to be returned")
	}

	errorMetrics := profiler.GetMetrics()
	errorCount := 0
	for _, metric := range errorMetrics {
		errorCount += metric.ErrorCount
	}

	if errorCount != 1 {
		t.Errorf("Expected 1 error in metrics, got %d", errorCount)
	}
}

func TestPerformanceProfiler_Statistics(t *testing.T) {
	profiler := NewPerformanceProfiler()
	profiler.Enable()

	// Run multiple operations
	for i := 0; i < 5; i++ {
		profiler.ProfileOperation("fast_op", func() error {
			time.Sleep(1 * time.Millisecond)
			return nil
		})
	}

	for i := 0; i < 3; i++ {
		profiler.ProfileOperation("slow_op", func() error {
			time.Sleep(5 * time.Millisecond)
			return nil
		})
	}

	// Test average time calculation
	fastAvg := profiler.GetAverageTime("fast_op")
	slowAvg := profiler.GetAverageTime("slow_op")

	if fastAvg >= slowAvg {
		t.Errorf("Expected fast_op (%v) to be faster than slow_op (%v)", fastAvg, slowAvg)
	}

	// Test performance report
	report := profiler.PerformanceReport()
	if report.TotalOperations != 8 {
		t.Errorf("Expected 8 total operations, got %d", report.TotalOperations)
	}

	if len(report.Operations) != 2 {
		t.Errorf("Expected 2 operation types, got %d", len(report.Operations))
	}

	// Check fast operation stats
	fastStats, exists := report.Operations["fast_op"]
	if !exists {
		t.Error("Expected fast_op in report")
	}
	if fastStats.Count != 5 {
		t.Errorf("Expected 5 fast_op operations, got %d", fastStats.Count)
	}

	// Check slow operation stats
	slowStats, exists := report.Operations["slow_op"]
	if !exists {
		t.Error("Expected slow_op in report")
	}
	if slowStats.Count != 3 {
		t.Errorf("Expected 3 slow_op operations, got %d", slowStats.Count)
	}
}

func TestIntegratedErrorSystem_ComplexScenarios(t *testing.T) {
	// Test complex scenario: validation + recovery + reporting
	system := NewIntegratedErrorSystem()
	system.EnableProfiling()
	system.handler.SetMode(ErrorModeLenient)

	// Add recovery strategies
	system.recoveryHandler.AddStrategy(NewFormatFallbackStrategy([]string{"invalid", "json"}))
	system.recoveryHandler.AddStrategy(NewDefaultValueStrategy(map[string]interface{}{
		"missing_field": "recovered_value",
	}))

	// Create a complex validation error
	validationErr := NewValidationError(ErrMissingRequired, "Complex validation failure").
		WithContext(ErrorContext{
			Operation: "data_validation",
			Field:     "required_field",
		}).
		// Note: WithViolations would need to be implemented for ValidationError
		WithSuggestions("Provide a value for the required field")

	// Process the error
	result := system.ProcessError(validationErr)

	// Error should be handled gracefully in lenient mode
	if result != nil {
		t.Errorf("Expected error to be handled, got: %v", result)
	}

	// Check that error was reported
	summary := system.GetErrorSummary()
	if summary.TotalErrors != 1 {
		t.Errorf("Expected 1 error in summary, got %d", summary.TotalErrors)
	}

	// Verify error categorization
	if summary.BySeverity[SeverityError] != 1 {
		t.Errorf("Expected 1 error with severity Error, got %d", summary.BySeverity[SeverityError])
	}

	if summary.ByCategory[ErrMissingRequired] != 1 {
		t.Errorf("Expected 1 error with code ErrMissingRequired, got %d", summary.ByCategory[ErrMissingRequired])
	}

	// Check suggestions
	if len(summary.Suggestions) == 0 {
		t.Error("Expected suggestions in summary")
	}

	// Verify performance was tracked
	report := system.GetPerformanceReport()
	if report.TotalOperations == 0 {
		t.Error("Expected performance operations to be tracked")
	}
}

func TestIntegratedErrorSystem_RecoveryScenarios(t *testing.T) {
	system := NewIntegratedErrorSystem()
	system.handler.SetMode(ErrorModeStrict) // Use strict mode to test recovery

	// Add format fallback strategy
	strategy := NewFormatFallbackStrategy([]string{"table", "csv", "json"})
	system.recoveryHandler.AddStrategy(strategy)

	// Create a format error that can be recovered
	formatErr := NewError(ErrInvalidFormat, "Invalid table format").
		WithContext(ErrorContext{
			Operation: "format_output",
		})

	// Mock a context that the strategy can work with
	// Note: In a real scenario, this would be the actual format configuration
	mockContext := &MockFormatContext{OutputFormat: "table"}

	// Test recovery by applying strategy directly
	recovered, err := strategy.Apply(formatErr, mockContext)
	if err != nil {
		t.Errorf("Recovery failed: %v", err)
	}

	if mockFormatCtx, ok := recovered.(*MockFormatContext); !ok {
		t.Error("Expected MockFormatContext to be returned")
	} else if mockFormatCtx.OutputFormat != "csv" {
		t.Errorf("Expected format to be changed to csv, got %s", mockFormatCtx.OutputFormat)
	}
}

// MockFormatContext simulates a format configuration for testing
type MockFormatContext struct {
	OutputFormat string
}

func TestSystemHealthCheck_Comprehensive(t *testing.T) {
	// Test healthy system
	system := NewIntegratedErrorSystem()
	health := system.SystemHealthCheck()

	if !health.Healthy {
		t.Errorf("Expected healthy system, got issues: %v", health.Issues)
	}

	// Test system with missing components
	brokenSystem := &IntegratedErrorSystem{}
	brokenHealth := brokenSystem.SystemHealthCheck()

	if brokenHealth.Healthy {
		t.Error("Expected unhealthy system with missing components")
	}

	if len(brokenHealth.Issues) == 0 {
		t.Error("Expected issues to be reported for broken system")
	}

	// Verify specific issues are detected
	hasHandlerIssue := false
	hasReporterIssue := false
	for _, issue := range brokenHealth.Issues {
		if issue == "Error handler is nil" {
			hasHandlerIssue = true
		}
		if issue == "Error reporter is nil" {
			hasReporterIssue = true
		}
	}

	if !hasHandlerIssue {
		t.Error("Expected 'Error handler is nil' issue to be detected")
	}
	if !hasReporterIssue {
		t.Error("Expected 'Error reporter is nil' issue to be detected")
	}
}

// Benchmark the integrated error system performance
func BenchmarkIntegratedErrorSystem_ProcessError(b *testing.B) {
	system := NewIntegratedErrorSystem()
	system.handler.SetMode(ErrorModeLenient)
	
	err := NewError(ErrInvalidFormat, "benchmark error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		system.ProcessError(err)
	}
}

func BenchmarkIntegratedErrorSystem_ProcessErrors(b *testing.B) {
	system := NewIntegratedErrorSystem()
	system.handler.SetMode(ErrorModeLenient)

	errors := make([]error, 100)
	for i := range errors {
		errors[i] = NewError(ErrInvalidFormat, fmt.Sprintf("error %d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		system.ProcessErrors(errors)
		system.Clear() // Clear between iterations for consistent benchmarking
	}
}

func BenchmarkPerformanceProfiler_ProfileOperation(b *testing.B) {
	profiler := NewPerformanceProfiler()
	profiler.Enable()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		profiler.ProfileOperation("benchmark_op", func() error {
			return nil
		})
	}
}

// Test migration helper with failing fixes
func TestMigrationHelper_FailingFixes(t *testing.T) {
	helper := NewMigrationHelper()

	failingStep := MigrationStep{
		Name:        "Failing step",
		Description: "A step that fails to fix",
		Check: func() (bool, string) {
			return false, "Step is not completed"
		},
		Fix: func() error {
			return fmt.Errorf("fix failed")
		},
	}

	passingStep := MigrationStep{
		Name:        "Passing step",
		Description: "A step that is already complete",
		Check: func() (bool, string) {
			return true, "Step is completed"
		},
	}

	helper.AddMigrationStep(failingStep)
	helper.AddMigrationStep(passingStep)

	result := helper.RunMigration()

	if result.FixedSteps != 0 {
		t.Errorf("Expected 0 fixed steps, got %d", result.FixedSteps)
	}

	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(result.Errors))
	}

	if result.Status.IsComplete() {
		t.Error("Expected migration to be incomplete due to failing fix")
	}
}

// Test error system with different handler modes
func TestIntegratedErrorSystem_DifferentModes(t *testing.T) {
	testErr := NewError(ErrInvalidFormat, "test error")

	modes := []ErrorMode{ErrorModeStrict, ErrorModeLenient, ErrorModeInteractive}
	for _, mode := range modes {
		t.Run(fmt.Sprintf("Mode_%s", mode.String()), func(t *testing.T) {
			system := NewIntegratedErrorSystem()
			system.handler.SetMode(mode)

			result := system.ProcessError(testErr)

			switch mode {
			case ErrorModeStrict:
				// Strict mode should return the error
				if result == nil {
					t.Error("Expected error to be returned in strict mode")
				}
			case ErrorModeLenient:
				// Lenient mode should collect the error
				if result != nil {
					t.Errorf("Expected error to be collected in lenient mode, got: %v", result)
				}
				summary := system.GetErrorSummary()
				if summary.TotalErrors != 1 {
					t.Errorf("Expected 1 error in summary, got %d", summary.TotalErrors)
				}
			case ErrorModeInteractive:
				// Interactive mode falls back to lenient in non-interactive environment
				if result != nil {
					t.Errorf("Expected error to be handled in interactive mode, got: %v", result)
				}
			}
		})
	}
}