package format

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ArjenSchwarz/go-output/errors"
)

// TestErrorReporter tests the ErrorReporter interface and implementations
func TestErrorReporter(t *testing.T) {
	tests := []struct {
		name             string
		setupErrors      func() []errors.OutputError
		expectTotal      int
		expectCategories map[errors.ErrorCode]int
		expectSeverities map[errors.ErrorSeverity]int
	}{
		{
			name: "single_error_reporting",
			setupErrors: func() []errors.OutputError {
				return []errors.OutputError{
					errors.NewError(errors.ErrInvalidFormat, "Test error"),
				}
			},
			expectTotal: 1,
			expectCategories: map[errors.ErrorCode]int{
				errors.ErrInvalidFormat: 1,
			},
			expectSeverities: map[errors.ErrorSeverity]int{
				errors.SeverityError: 1,
			},
		},
		{
			name: "multiple_error_reporting",
			setupErrors: func() []errors.OutputError {
				return []errors.OutputError{
					errors.NewError(errors.ErrInvalidFormat, "Error 1"),
					errors.NewError(errors.ErrMissingRequired, "Error 2"),
					errors.NewError(errors.ErrInvalidFormat, "Error 3"), // Duplicate category
					errors.NewValidationError(errors.ErrMissingColumn, "Error 4"),
				}
			},
			expectTotal: 4,
			expectCategories: map[errors.ErrorCode]int{
				errors.ErrInvalidFormat:   2,
				errors.ErrMissingRequired: 1,
				errors.ErrMissingColumn:   1,
			},
			expectSeverities: map[errors.ErrorSeverity]int{
				errors.SeverityError: 4,
			},
		},
		{
			name: "mixed_severity_reporting",
			setupErrors: func() []errors.OutputError {
				return []errors.OutputError{
					errors.NewError(errors.ErrInvalidFormat, "Error").WithSeverity(errors.SeverityFatal),
					errors.NewError(errors.ErrMissingRequired, "Warning").WithSeverity(errors.SeverityWarning),
					errors.NewError(errors.ErrInvalidDataType, "Info").WithSeverity(errors.SeverityInfo),
				}
			},
			expectTotal: 3,
			expectCategories: map[errors.ErrorCode]int{
				errors.ErrInvalidFormat:   1,
				errors.ErrMissingRequired: 1,
				errors.ErrInvalidDataType: 1,
			},
			expectSeverities: map[errors.ErrorSeverity]int{
				errors.SeverityFatal:   1,
				errors.SeverityWarning: 1,
				errors.SeverityInfo:    1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := NewDefaultErrorReporter()
			errorList := tt.setupErrors()

			// Report all errors
			for _, err := range errorList {
				reporter.Report(err)
			}

			// Get summary
			summary := reporter.Summary()

			// Check total count
			if summary.TotalErrors != tt.expectTotal {
				t.Errorf("Expected %d total errors, got %d", tt.expectTotal, summary.TotalErrors)
			}

			// Check category counts
			for expectedCode, expectedCount := range tt.expectCategories {
				if actualCount, exists := summary.ByCategory[expectedCode]; !exists {
					t.Errorf("Expected category %s not found in summary", expectedCode)
				} else if actualCount != expectedCount {
					t.Errorf("Expected %d errors for category %s, got %d", expectedCount, expectedCode, actualCount)
				}
			}

			// Check severity counts
			for expectedSeverity, expectedCount := range tt.expectSeverities {
				if actualCount, exists := summary.BySeverity[expectedSeverity]; !exists {
					t.Errorf("Expected severity %s not found in summary", expectedSeverity.String())
				} else if actualCount != expectedCount {
					t.Errorf("Expected %d errors for severity %s, got %d", expectedCount, expectedSeverity.String(), actualCount)
				}
			}
		})
	}
}

// TestErrorSummaryJSONMarshaling tests JSON serialization of error summaries
func TestErrorSummaryJSONMarshaling(t *testing.T) {
	reporter := NewDefaultErrorReporter()

	// Add various errors
	reporter.Report(errors.NewError(errors.ErrInvalidFormat, "Test error 1"))
	reporter.Report(errors.NewError(errors.ErrMissingRequired, "Test error 2").WithSeverity(errors.SeverityWarning))
	reporter.Report(errors.NewValidationError(errors.ErrMissingColumn, "Test validation error"))

	summary := reporter.Summary()

	// Marshal to JSON
	jsonData, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("Failed to marshal error summary to JSON: %v", err)
	}

	// Unmarshal and verify structure
	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Check required fields
	requiredFields := []string{"total_errors", "by_category", "by_severity", "timestamp", "suggestions"}
	for _, field := range requiredFields {
		if _, exists := unmarshaled[field]; !exists {
			t.Errorf("Required field '%s' not found in JSON output", field)
		}
	}

	// Verify total errors count
	if totalErrors, ok := unmarshaled["total_errors"].(float64); !ok || int(totalErrors) != 3 {
		t.Errorf("Expected total_errors to be 3, got %v", unmarshaled["total_errors"])
	}
}

// TestErrorSummaryTextFormatting tests text formatting of error summaries
func TestErrorSummaryTextFormatting(t *testing.T) {
	reporter := NewDefaultErrorReporter()

	// Add errors with different characteristics
	reporter.Report(errors.NewError(errors.ErrInvalidFormat, "Invalid JSON format").
		WithSuggestions("Check JSON syntax", "Validate data types"))
	reporter.Report(errors.NewError(errors.ErrMissingRequired, "Missing field").
		WithSeverity(errors.SeverityWarning))

	summary := reporter.Summary()
	textOutput := summary.FormatText()

	// Check that text output contains expected information
	expectedContent := []string{
		"Error Summary",
		"Total Errors: 2",
		"By Category:",
		"OUT-1001", // ErrInvalidFormat
		"OUT-1002", // ErrMissingRequired
		"By Severity:",
		"ERROR: 1",
		"WARNING: 1",
		"Suggestions:",
		"Check JSON syntax",
		"Validate data types",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(textOutput, expected) {
			t.Errorf("Expected text output to contain '%s', but it was missing. Output:\n%s", expected, textOutput)
		}
	}
}

// TestErrorMetricsCollection tests metrics collection functionality
func TestErrorMetricsCollection(t *testing.T) {
	metrics := NewErrorMetrics()

	// Simulate error collection over time
	startTime := time.Now()

	// Add errors at different times
	metrics.RecordError(errors.NewError(errors.ErrInvalidFormat, "Error 1"))
	time.Sleep(1 * time.Millisecond) // Small delay to ensure different timestamps
	metrics.RecordError(errors.NewError(errors.ErrMissingRequired, "Error 2"))
	time.Sleep(1 * time.Millisecond)
	metrics.RecordError(errors.NewError(errors.ErrInvalidFormat, "Error 3"))

	// Test metrics retrieval
	totalCount := metrics.GetTotalErrorCount()
	if totalCount != 3 {
		t.Errorf("Expected total error count to be 3, got %d", totalCount)
	}

	// Test time-based metrics
	endTime := time.Now()
	countInTimeRange := metrics.GetErrorCountInTimeRange(startTime, endTime)
	if countInTimeRange != 3 {
		t.Errorf("Expected 3 errors in time range, got %d", countInTimeRange)
	}

	// Test error rate calculation
	rate := metrics.GetErrorRate(1 * time.Second)
	if rate < 0 {
		t.Errorf("Error rate should be non-negative, got %f", rate)
	}

	// Test most frequent error types
	frequentErrors := metrics.GetMostFrequentErrors(2)
	if len(frequentErrors) == 0 {
		t.Error("Expected frequent errors list to be non-empty")
	}

	// Verify most frequent error is ErrInvalidFormat (appears twice)
	if len(frequentErrors) > 0 && frequentErrors[0].Code != errors.ErrInvalidFormat {
		t.Errorf("Expected most frequent error to be %s, got %s", errors.ErrInvalidFormat, frequentErrors[0].Code)
	}
}

// TestStructuredLogging tests integration with structured logging
func TestStructuredLogging(t *testing.T) {
	logger := NewStructuredLogger()

	// Create an error with rich context
	err := errors.NewError(errors.ErrInvalidDataType, "Invalid data type in field").
		WithContext(errors.ErrorContext{
			Operation: "data_validation",
			Field:     "user_age",
			Value:     "not_a_number",
			Index:     5,
			Metadata: map[string]interface{}{
				"expected_type": "integer",
				"received_type": "string",
			},
		}).
		WithSuggestions("Convert string to integer", "Validate input data")

	// Log the error
	logEntry := logger.LogError(err)

	// Verify log entry structure
	if logEntry.Level != "ERROR" {
		t.Errorf("Expected log level to be ERROR, got %s", logEntry.Level)
	}

	if logEntry.ErrorCode != string(errors.ErrInvalidDataType) {
		t.Errorf("Expected error code %s, got %s", errors.ErrInvalidDataType, logEntry.ErrorCode)
	}

	if logEntry.Context.Operation != "data_validation" {
		t.Errorf("Expected operation to be 'data_validation', got %s", logEntry.Context.Operation)
	}

	if len(logEntry.Suggestions) != 2 {
		t.Errorf("Expected 2 suggestions, got %d", len(logEntry.Suggestions))
	}

	// Test JSON serialization of log entry
	jsonData, jsonErr := json.Marshal(logEntry)
	if jsonErr != nil {
		t.Fatalf("Failed to marshal log entry to JSON: %v", jsonErr)
	}

	// Verify JSON contains expected fields
	var logMap map[string]interface{}
	if unmarshalErr := json.Unmarshal(jsonData, &logMap); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal log JSON: %v", unmarshalErr)
	}

	requiredFields := []string{"timestamp", "level", "message", "error_code", "context", "suggestions"}
	for _, field := range requiredFields {
		if _, exists := logMap[field]; !exists {
			t.Errorf("Required field '%s' not found in log JSON", field)
		}
	}
}

// TestErrorReportingPerformance tests performance impact of error reporting
func TestErrorReportingPerformance(t *testing.T) {
	reporter := NewDefaultErrorReporter()

	// Generate a large number of errors
	numErrors := 10000
	testErrors := make([]errors.OutputError, numErrors)
	for i := 0; i < numErrors; i++ {
		testErrors[i] = errors.NewError(errors.ErrInvalidDataType, "Performance test error")
	}

	// Measure reporting time
	startTime := time.Now()
	for _, err := range testErrors {
		reporter.Report(err)
	}
	reportingTime := time.Since(startTime)

	// Measure summary generation time
	summaryStartTime := time.Now()
	summary := reporter.Summary()
	summaryTime := time.Since(summaryStartTime)

	// Verify correctness
	if summary.TotalErrors != numErrors {
		t.Errorf("Expected %d total errors, got %d", numErrors, summary.TotalErrors)
	}

	// Performance assertions (these are reasonable expectations, adjust if needed)
	maxReportingTime := 100 * time.Millisecond
	maxSummaryTime := 50 * time.Millisecond

	if reportingTime > maxReportingTime {
		t.Errorf("Error reporting took too long: %v (max: %v)", reportingTime, maxReportingTime)
	}

	if summaryTime > maxSummaryTime {
		t.Errorf("Summary generation took too long: %v (max: %v)", summaryTime, maxSummaryTime)
	}

	t.Logf("Performance metrics: Reporting %d errors took %v, Summary generation took %v",
		numErrors, reportingTime, summaryTime)
}

// TestObservabilityIntegration tests integration with observability tools
func TestObservabilityIntegration(t *testing.T) {
	// Test Prometheus-style metrics
	metricsExporter := NewPrometheusMetricsExporter()

	// Add various errors
	metricsExporter.RecordError(errors.NewError(errors.ErrInvalidFormat, "Error 1"))
	metricsExporter.RecordError(errors.NewError(errors.ErrMissingRequired, "Error 2"))
	metricsExporter.RecordError(errors.NewError(errors.ErrInvalidFormat, "Error 3"))

	// Export metrics
	metricsOutput := metricsExporter.Export()

	// Check Prometheus format
	expectedMetrics := []string{
		"# HELP go_output_errors_total Total number of errors",
		"# TYPE go_output_errors_total counter",
		"go_output_errors_total{code=\"OUT-1001\"} 2",
		"go_output_errors_total{code=\"OUT-1002\"} 1",
	}

	for _, expected := range expectedMetrics {
		if !strings.Contains(metricsOutput, expected) {
			t.Errorf("Expected metrics output to contain '%s', but it was missing. Output:\n%s", expected, metricsOutput)
		}
	}
}

// TestErrorReportingConfiguration tests different reporting configurations
func TestErrorReportingConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		config ErrorReportingConfig
		errors []errors.OutputError
		expect func(t *testing.T, summary ErrorSummary)
	}{
		{
			name: "severity_filter",
			config: ErrorReportingConfig{
				MinSeverity:    errors.SeverityError,
				IncludeContext: true,
			},
			errors: []errors.OutputError{
				errors.NewError(errors.ErrInvalidFormat, "Error").WithSeverity(errors.SeverityError),
				errors.NewError(errors.ErrMissingRequired, "Warning").WithSeverity(errors.SeverityWarning),
				errors.NewError(errors.ErrInvalidDataType, "Info").WithSeverity(errors.SeverityInfo),
			},
			expect: func(t *testing.T, summary ErrorSummary) {
				if summary.TotalErrors != 1 {
					t.Errorf("Expected 1 error after severity filtering, got %d", summary.TotalErrors)
				}
			},
		},
		{
			name: "category_inclusion",
			config: ErrorReportingConfig{
				MinSeverity:       errors.SeverityInfo, // Allow all severity levels
				IncludeCategories: []errors.ErrorCode{errors.ErrInvalidFormat, errors.ErrMissingRequired},
				ExcludeCategories: []errors.ErrorCode{},
			},
			errors: []errors.OutputError{
				errors.NewError(errors.ErrInvalidFormat, "Error 1"),
				errors.NewError(errors.ErrMissingRequired, "Error 2"),
				errors.NewError(errors.ErrInvalidDataType, "Error 3"),
			},
			expect: func(t *testing.T, summary ErrorSummary) {
				if summary.TotalErrors != 2 {
					t.Errorf("Expected 2 errors after category filtering, got %d", summary.TotalErrors)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := NewConfigurableErrorReporter(tt.config)

			for _, err := range tt.errors {
				reporter.Report(err)
			}

			summary := reporter.Summary()
			tt.expect(t, summary)
		})
	}
}

// TestErrorSummaryAggregation tests aggregation of multiple error summaries
func TestErrorSummaryAggregation(t *testing.T) {
	// Create multiple reporters (simulating different components/modules)
	reporter1 := NewDefaultErrorReporter()
	reporter2 := NewDefaultErrorReporter()
	reporter3 := NewDefaultErrorReporter()

	// Add errors to different reporters
	reporter1.Report(errors.NewError(errors.ErrInvalidFormat, "Error from component 1"))
	reporter1.Report(errors.NewError(errors.ErrMissingRequired, "Another error from component 1"))

	reporter2.Report(errors.NewError(errors.ErrInvalidFormat, "Error from component 2"))
	reporter2.Report(errors.NewError(errors.ErrInvalidDataType, "Error from component 2"))

	reporter3.Report(errors.NewError(errors.ErrMissingColumn, "Error from component 3"))

	// Get individual summaries
	summary1 := reporter1.Summary()
	summary2 := reporter2.Summary()
	summary3 := reporter3.Summary()

	// Aggregate summaries
	aggregatedSummary := AggregateSummaries(summary1, summary2, summary3)

	// Verify aggregation
	expectedTotal := 5
	if aggregatedSummary.TotalErrors != expectedTotal {
		t.Errorf("Expected %d total errors in aggregated summary, got %d", expectedTotal, aggregatedSummary.TotalErrors)
	}

	// Check category aggregation
	expectedInvalidFormat := 2
	if count, exists := aggregatedSummary.ByCategory[errors.ErrInvalidFormat]; !exists || count != expectedInvalidFormat {
		t.Errorf("Expected %d ErrInvalidFormat errors, got %d", expectedInvalidFormat, count)
	}

	// Verify that all unique categories are present
	expectedCategories := 4 // ErrInvalidFormat, ErrMissingRequired, ErrInvalidDataType, ErrMissingColumn
	if len(aggregatedSummary.ByCategory) != expectedCategories {
		t.Errorf("Expected %d unique categories, got %d", expectedCategories, len(aggregatedSummary.ByCategory))
	}
}
