package format

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// mockLogger implements the Logger interface for testing
type mockLogger struct {
	logs []logEntry
}

type logEntry struct {
	level   string
	message string
	fields  map[string]interface{}
}

func (m *mockLogger) Info(msg string, fields map[string]interface{}) {
	m.logs = append(m.logs, logEntry{"INFO", msg, fields})
}

func (m *mockLogger) Warn(msg string, fields map[string]interface{}) {
	m.logs = append(m.logs, logEntry{"WARN", msg, fields})
}

func (m *mockLogger) Error(msg string, fields map[string]interface{}) {
	m.logs = append(m.logs, logEntry{"ERROR", msg, fields})
}

func (m *mockLogger) Fatal(msg string, fields map[string]interface{}) {
	m.logs = append(m.logs, logEntry{"FATAL", msg, fields})
}

func (m *mockLogger) getLogsByLevel(level string) []logEntry {
	var filtered []logEntry
	for _, log := range m.logs {
		if log.level == level {
			filtered = append(filtered, log)
		}
	}
	return filtered
}

func TestNewErrorReporter(t *testing.T) {
	reporter := NewErrorReporter()

	if reporter == nil {
		t.Fatal("NewErrorReporter() returned nil")
	}

	if reporter.logger == nil {
		t.Error("Expected logger to be initialized")
	}

	if !reporter.enableMetrics {
		t.Error("Expected metrics to be enabled by default")
	}

	if len(reporter.errors) != 0 {
		t.Error("Expected errors slice to be empty initially")
	}
}

func TestNewErrorReporterWithOptions(t *testing.T) {
	mockLogger := &mockLogger{}
	reporter := NewErrorReporterWithOptions(mockLogger, false)

	if reporter == nil {
		t.Fatal("NewErrorReporterWithOptions() returned nil")
	}

	if reporter.logger != mockLogger {
		t.Error("Expected custom logger to be set")
	}

	if reporter.enableMetrics {
		t.Error("Expected metrics to be disabled")
	}
}

func TestErrorReporter_Report(t *testing.T) {
	mockLogger := &mockLogger{}
	reporter := NewErrorReporterWithOptions(mockLogger, true)

	// Create test error
	err := NewErrorBuilder(ErrInvalidFormat, "test error").
		WithSeverity(SeverityError).
		WithField("test_field").
		WithSuggestions("try this", "or this").
		Build()

	// Report the error
	reporter.Report(err)

	// Check that error was stored
	if len(reporter.errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(reporter.errors))
	}

	reportedErr := reporter.errors[0]
	if reportedErr.Error.Code() != ErrInvalidFormat {
		t.Errorf("Expected error code %s, got %s", ErrInvalidFormat, reportedErr.Error.Code())
	}

	// Check that error was logged
	errorLogs := mockLogger.getLogsByLevel("ERROR")
	if len(errorLogs) != 1 {
		t.Fatalf("Expected 1 error log, got %d", len(errorLogs))
	}

	log := errorLogs[0]
	if !strings.Contains(log.message, "test error") {
		t.Error("Expected log message to contain error text")
	}

	if log.fields["error_code"] != string(ErrInvalidFormat) {
		t.Error("Expected error_code field in log")
	}

	if log.fields["severity"] != "error" {
		t.Error("Expected severity field in log")
	}

	if log.fields["field"] != "test_field" {
		t.Error("Expected field field in log")
	}
}

func TestErrorReporter_ReportNilError(t *testing.T) {
	reporter := NewErrorReporter()

	// Report nil error
	reporter.Report(nil)

	// Check that no error was stored
	if len(reporter.errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(reporter.errors))
	}
}

func TestErrorReporter_Summary_Empty(t *testing.T) {
	reporter := NewErrorReporter()

	summary := reporter.Summary()

	if summary.TotalErrors != 0 {
		t.Errorf("Expected 0 total errors, got %d", summary.TotalErrors)
	}

	if len(summary.ByCategory) != 0 {
		t.Error("Expected empty ByCategory map")
	}

	if len(summary.BySeverity) != 0 {
		t.Error("Expected empty BySeverity map")
	}

	if len(summary.TopErrors) != 0 {
		t.Error("Expected empty TopErrors slice")
	}
}

func TestErrorReporter_Summary_WithErrors(t *testing.T) {
	reporter := NewErrorReporter()

	// Create and report multiple errors
	err1 := NewErrorBuilder(ErrInvalidFormat, "error 1").
		WithSeverity(SeverityError).
		WithSuggestions("suggestion 1").
		Build()

	err2 := NewErrorBuilder(ErrInvalidFormat, "error 2").
		WithSeverity(SeverityWarning).
		WithSuggestions("suggestion 2").
		Build()

	err3 := NewErrorBuilder(ErrMissingColumn, "error 3").
		WithSeverity(SeverityError).
		WithSuggestions("suggestion 1", "suggestion 3").
		Build()

	reporter.Report(err1)
	time.Sleep(time.Millisecond) // Ensure different timestamps
	reporter.Report(err2)
	time.Sleep(time.Millisecond)
	reporter.Report(err3)

	summary := reporter.Summary()

	// Check total errors
	if summary.TotalErrors != 3 {
		t.Errorf("Expected 3 total errors, got %d", summary.TotalErrors)
	}

	// Check category counts
	if summary.ByCategory[ErrInvalidFormat] != 2 {
		t.Errorf("Expected 2 ErrInvalidFormat errors, got %d", summary.ByCategory[ErrInvalidFormat])
	}

	if summary.ByCategory[ErrMissingColumn] != 1 {
		t.Errorf("Expected 1 ErrMissingColumn error, got %d", summary.ByCategory[ErrMissingColumn])
	}

	// Check severity counts
	if summary.BySeverity[SeverityError] != 2 {
		t.Errorf("Expected 2 error severity, got %d", summary.BySeverity[SeverityError])
	}

	if summary.BySeverity[SeverityWarning] != 1 {
		t.Errorf("Expected 1 warning severity, got %d", summary.BySeverity[SeverityWarning])
	}

	// Check fixable errors (warnings and info are fixable)
	if summary.FixableErrors != 1 {
		t.Errorf("Expected 1 fixable error, got %d", summary.FixableErrors)
	}

	// Check unique suggestions
	expectedSuggestions := []string{"suggestion 1", "suggestion 2", "suggestion 3"}
	if len(summary.Suggestions) != len(expectedSuggestions) {
		t.Errorf("Expected %d suggestions, got %d", len(expectedSuggestions), len(summary.Suggestions))
	}

	// Check top errors
	if len(summary.TopErrors) != 2 {
		t.Errorf("Expected 2 top errors, got %d", len(summary.TopErrors))
	}

	// Top error should be ErrInvalidFormat (appears twice)
	if summary.TopErrors[0].Code != ErrInvalidFormat {
		t.Errorf("Expected top error to be %s, got %s", ErrInvalidFormat, summary.TopErrors[0].Code)
	}

	if summary.TopErrors[0].Count != 2 {
		t.Errorf("Expected top error count to be 2, got %d", summary.TopErrors[0].Count)
	}

	// Check timestamps
	if summary.FirstOccurrence.IsZero() {
		t.Error("Expected FirstOccurrence to be set")
	}

	if summary.LastOccurrence.IsZero() {
		t.Error("Expected LastOccurrence to be set")
	}

	if !summary.LastOccurrence.After(summary.FirstOccurrence) {
		t.Error("Expected LastOccurrence to be after FirstOccurrence")
	}

	// Check duration and error rate
	if summary.Duration <= 0 {
		t.Error("Expected positive duration")
	}

	if summary.ErrorRate <= 0 {
		t.Error("Expected positive error rate")
	}
}

func TestErrorReporter_GetMetrics(t *testing.T) {
	reporter := NewErrorReporter()

	// Create errors at different times
	now := time.Now()
	err1 := NewErrorBuilder(ErrInvalidFormat, "error 1").WithSeverity(SeverityError).Build()
	err2 := NewErrorBuilder(ErrMissingColumn, "error 2").WithSeverity(SeverityWarning).Build()

	reporter.Report(err1)
	reporter.Report(err2)

	metrics := reporter.GetMetrics()

	// Check error counts
	if metrics.ErrorCount[ErrInvalidFormat] != 1 {
		t.Errorf("Expected 1 ErrInvalidFormat, got %d", metrics.ErrorCount[ErrInvalidFormat])
	}

	if metrics.ErrorCount[ErrMissingColumn] != 1 {
		t.Errorf("Expected 1 ErrMissingColumn, got %d", metrics.ErrorCount[ErrMissingColumn])
	}

	// Check severity counts
	if metrics.SeverityCount[SeverityError] != 1 {
		t.Errorf("Expected 1 error severity, got %d", metrics.SeverityCount[SeverityError])
	}

	if metrics.SeverityCount[SeverityWarning] != 1 {
		t.Errorf("Expected 1 warning severity, got %d", metrics.SeverityCount[SeverityWarning])
	}

	// Check hourly distribution
	hour := now.Hour()
	if metrics.HourlyDistribution[hour] != 2 {
		t.Errorf("Expected 2 errors in hour %d, got %d", hour, metrics.HourlyDistribution[hour])
	}

	// Check error trends
	if len(metrics.ErrorTrends) == 0 {
		t.Error("Expected error trends to be calculated")
	}
}

func TestErrorReporter_Reset(t *testing.T) {
	reporter := NewErrorReporter()

	// Add some errors
	err := NewErrorBuilder(ErrInvalidFormat, "test error").Build()
	reporter.Report(err)

	if len(reporter.errors) != 1 {
		t.Fatal("Expected 1 error before reset")
	}

	// Reset
	reporter.Reset()

	if len(reporter.errors) != 0 {
		t.Errorf("Expected 0 errors after reset, got %d", len(reporter.errors))
	}

	// Check that start time was updated
	if reporter.startTime.IsZero() {
		t.Error("Expected start time to be updated after reset")
	}
}

func TestErrorReporter_SetLogger(t *testing.T) {
	reporter := NewErrorReporter()
	mockLogger := &mockLogger{}

	reporter.SetLogger(mockLogger)

	// Report an error to test the new logger
	err := NewErrorBuilder(ErrInvalidFormat, "test error").WithSeverity(SeverityError).Build()
	reporter.Report(err)

	// Check that the mock logger received the log
	if len(mockLogger.logs) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(mockLogger.logs))
	}
}

func TestErrorReporter_GetErrorHistory(t *testing.T) {
	reporter := NewErrorReporter()

	// Add some errors
	err1 := NewErrorBuilder(ErrInvalidFormat, "error 1").Build()
	err2 := NewErrorBuilder(ErrMissingColumn, "error 2").Build()

	reporter.Report(err1)
	reporter.Report(err2)

	history := reporter.GetErrorHistory()

	if len(history) != 2 {
		t.Errorf("Expected 2 errors in history, got %d", len(history))
	}

	// Verify it's a copy (modifying shouldn't affect original)
	history[0].Context = "modified"
	originalHistory := reporter.GetErrorHistory()
	if originalHistory[0].Context == "modified" {
		t.Error("Expected history to be a copy, but original was modified")
	}
}

func TestErrorReporter_GetErrorsByTimeRange(t *testing.T) {
	reporter := NewErrorReporter()

	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	// Manually add errors with specific timestamps
	reporter.mu.Lock()
	reporter.errors = []reportedError{
		{Error: NewErrorBuilder(ErrInvalidFormat, "old error").Build(), Timestamp: past},
		{Error: NewErrorBuilder(ErrMissingColumn, "current error").Build(), Timestamp: now},
		{Error: NewErrorBuilder(ErrInvalidDataType, "future error").Build(), Timestamp: future},
	}
	reporter.mu.Unlock()

	// Get errors in a specific range
	start := past.Add(time.Minute)
	end := future.Add(-time.Minute)
	filtered := reporter.GetErrorsByTimeRange(start, end)

	if len(filtered) != 1 {
		t.Errorf("Expected 1 error in time range, got %d", len(filtered))
	}

	if filtered[0].Error.Code() != ErrMissingColumn {
		t.Errorf("Expected ErrMissingColumn, got %s", filtered[0].Error.Code())
	}
}

func TestErrorReporter_GetErrorsByCode(t *testing.T) {
	reporter := NewErrorReporter()

	err1 := NewErrorBuilder(ErrInvalidFormat, "error 1").Build()
	err2 := NewErrorBuilder(ErrMissingColumn, "error 2").Build()
	err3 := NewErrorBuilder(ErrInvalidFormat, "error 3").Build()

	reporter.Report(err1)
	reporter.Report(err2)
	reporter.Report(err3)

	filtered := reporter.GetErrorsByCode(ErrInvalidFormat)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 errors with ErrInvalidFormat, got %d", len(filtered))
	}

	for _, err := range filtered {
		if err.Error.Code() != ErrInvalidFormat {
			t.Errorf("Expected ErrInvalidFormat, got %s", err.Error.Code())
		}
	}
}

func TestErrorReporter_GetErrorsBySeverity(t *testing.T) {
	reporter := NewErrorReporter()

	err1 := NewErrorBuilder(ErrInvalidFormat, "error 1").WithSeverity(SeverityError).Build()
	err2 := NewErrorBuilder(ErrMissingColumn, "error 2").WithSeverity(SeverityWarning).Build()
	err3 := NewErrorBuilder(ErrInvalidDataType, "error 3").WithSeverity(SeverityError).Build()

	reporter.Report(err1)
	reporter.Report(err2)
	reporter.Report(err3)

	filtered := reporter.GetErrorsBySeverity(SeverityError)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 errors with SeverityError, got %d", len(filtered))
	}

	for _, err := range filtered {
		if err.Error.Severity() != SeverityError {
			t.Errorf("Expected SeverityError, got %s", err.Error.Severity().String())
		}
	}
}

func TestErrorReporter_ExportJSON(t *testing.T) {
	reporter := NewErrorReporter()

	testErr := NewErrorBuilder(ErrInvalidFormat, "test error").
		WithSeverity(SeverityError).
		WithSuggestions("test suggestion").
		Build()

	reporter.Report(testErr)

	jsonData, err := reporter.ExportJSON()
	if err != nil {
		t.Fatalf("ExportJSON() failed: %v", err)
	}

	// Verify it's valid JSON
	var summary ExtendedErrorSummary
	if err := json.Unmarshal(jsonData, &summary); err != nil {
		t.Fatalf("Failed to unmarshal exported JSON: %v", err)
	}

	if summary.TotalErrors != 1 {
		t.Errorf("Expected 1 total error in exported JSON, got %d", summary.TotalErrors)
	}
}

func TestErrorReporter_ExportMetricsJSON(t *testing.T) {
	reporter := NewErrorReporter()

	testErr := NewErrorBuilder(ErrInvalidFormat, "test error").WithSeverity(SeverityError).Build()
	reporter.Report(testErr)

	jsonData, err := reporter.ExportMetricsJSON()
	if err != nil {
		t.Fatalf("ExportMetricsJSON() failed: %v", err)
	}

	// Verify it's valid JSON
	var metrics ErrorMetrics
	if err := json.Unmarshal(jsonData, &metrics); err != nil {
		t.Fatalf("Failed to unmarshal exported metrics JSON: %v", err)
	}

	if metrics.ErrorCount[ErrInvalidFormat] != 1 {
		t.Errorf("Expected 1 ErrInvalidFormat in exported metrics, got %d", metrics.ErrorCount[ErrInvalidFormat])
	}
}

func TestDefaultLogger(t *testing.T) {
	logger := &defaultLogger{}

	// Test that logging methods don't panic
	fields := map[string]interface{}{
		"test_field": "test_value",
		"number":     42,
	}

	logger.Info("test info", fields)
	logger.Warn("test warning", fields)
	logger.Error("test error", fields)
	logger.Fatal("test fatal", fields)

	// If we reach here, no panics occurred
}

func TestMonitoringIntegration(t *testing.T) {
	reporter := NewErrorReporter()
	integration := NewMonitoringIntegration(reporter, "test-service")

	if integration.serviceName != "test-service" {
		t.Errorf("Expected service name 'test-service', got '%s'", integration.serviceName)
	}

	if integration.reporter != reporter {
		t.Error("Expected reporter to be set correctly")
	}

	// Test webhook configuration
	integration.SetWebhook("http://example.com/webhook", "test-api-key")

	if integration.webhookURL != "http://example.com/webhook" {
		t.Errorf("Expected webhook URL to be set correctly")
	}

	if integration.apiKey != "test-api-key" {
		t.Errorf("Expected API key to be set correctly")
	}
}

func TestMonitoringIntegration_SendSummaryToMonitoring(t *testing.T) {
	reporter := NewErrorReporter()
	integration := NewMonitoringIntegration(reporter, "test-service")

	// Add some errors
	err := NewErrorBuilder(ErrInvalidFormat, "test error").Build()
	reporter.Report(err)

	// This should not panic or return an error
	if err := integration.SendSummaryToMonitoring(); err != nil {
		t.Errorf("SendSummaryToMonitoring() failed: %v", err)
	}
}

func TestMonitoringIntegration_CheckThresholds(t *testing.T) {
	reporter := NewErrorReporter()
	integration := NewMonitoringIntegration(reporter, "test-service")

	threshold := AlertThreshold{
		ErrorRate:     1.0, // 1 error per minute
		TotalErrors:   2,
		SeverityLevel: SeverityError,
		TimeWindow:    time.Minute,
	}

	// No errors - should not exceed threshold
	if integration.CheckThresholds(threshold) {
		t.Error("Expected threshold not to be exceeded with no errors")
	}

	// Add one error - should not exceed threshold
	err1 := NewErrorBuilder(ErrInvalidFormat, "error 1").WithSeverity(SeverityWarning).Build()
	reporter.Report(err1)

	if integration.CheckThresholds(threshold) {
		t.Error("Expected threshold not to be exceeded with one warning")
	}

	// Add error with sufficient severity - should exceed threshold
	err2 := NewErrorBuilder(ErrMissingColumn, "error 2").WithSeverity(SeverityError).Build()
	reporter.Report(err2)

	if !integration.CheckThresholds(threshold) {
		t.Error("Expected threshold to be exceeded with error severity")
	}
}

func TestErrorFrequency_Sorting(t *testing.T) {
	reporter := NewErrorReporter()

	// Add errors with different frequencies
	for i := 0; i < 3; i++ {
		err := NewErrorBuilder(ErrInvalidFormat, "frequent error").Build()
		reporter.Report(err)
	}

	for i := 0; i < 1; i++ {
		err := NewErrorBuilder(ErrMissingColumn, "rare error").Build()
		reporter.Report(err)
	}

	summary := reporter.Summary()

	if len(summary.TopErrors) != 2 {
		t.Fatalf("Expected 2 error types, got %d", len(summary.TopErrors))
	}

	// Most frequent should be first
	if summary.TopErrors[0].Code != ErrInvalidFormat {
		t.Errorf("Expected most frequent error to be ErrInvalidFormat, got %s", summary.TopErrors[0].Code)
	}

	if summary.TopErrors[0].Count != 3 {
		t.Errorf("Expected most frequent error count to be 3, got %d", summary.TopErrors[0].Count)
	}

	if summary.TopErrors[1].Code != ErrMissingColumn {
		t.Errorf("Expected second error to be ErrMissingColumn, got %s", summary.TopErrors[1].Code)
	}

	if summary.TopErrors[1].Count != 1 {
		t.Errorf("Expected second error count to be 1, got %d", summary.TopErrors[1].Count)
	}
}

func TestErrorTrends_Calculation(t *testing.T) {
	reporter := NewErrorReporter()

	// Add errors with specific timestamps
	now := time.Now()
	reporter.mu.Lock()
	reporter.errors = []reportedError{
		{Error: NewErrorBuilder(ErrInvalidFormat, "error 1").Build(), Timestamp: now},
		{Error: NewErrorBuilder(ErrMissingColumn, "error 2").Build(), Timestamp: now.Add(time.Minute)},
		{Error: NewErrorBuilder(ErrInvalidDataType, "error 3").Build(), Timestamp: now.Add(6 * time.Minute)},
	}
	reporter.mu.Unlock()

	trends := reporter.calculateErrorTrends()

	if len(trends) == 0 {
		t.Fatal("Expected error trends to be calculated")
	}

	// Trends should be sorted by timestamp
	for i := 1; i < len(trends); i++ {
		if trends[i].Timestamp.Before(trends[i-1].Timestamp) {
			t.Error("Expected trends to be sorted by timestamp")
		}
	}

	// Should have at least 2 buckets with errors
	bucketsWithErrors := 0
	for _, trend := range trends {
		if trend.Count > 0 {
			bucketsWithErrors++
		}
	}

	if bucketsWithErrors < 2 {
		t.Errorf("Expected at least 2 buckets with errors, got %d", bucketsWithErrors)
	}
}
