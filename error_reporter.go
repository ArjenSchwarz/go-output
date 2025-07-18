package format

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"
)

// ErrorReporter defines the interface for error reporting and monitoring integration
type ErrorReporter interface {
	// Report reports a single error to the monitoring system
	Report(err OutputError)
	// Summary returns aggregated error statistics
	Summary() ExtendedErrorSummary
	// Reset clears all collected error data
	Reset()
	// SetLogger sets a custom logger for structured logging
	SetLogger(logger Logger)
}

// Logger defines the interface for structured logging
type Logger interface {
	// Info logs an informational message with structured data
	Info(msg string, fields map[string]interface{})
	// Warn logs a warning message with structured data
	Warn(msg string, fields map[string]interface{})
	// Error logs an error message with structured data
	Error(msg string, fields map[string]interface{})
	// Fatal logs a fatal error message with structured data
	Fatal(msg string, fields map[string]interface{})
}

// ExtendedErrorSummary provides comprehensive error statistics and categorization
// This extends the base ErrorSummary with additional monitoring fields
type ExtendedErrorSummary struct {
	ErrorSummary                     // Embed the base ErrorSummary
	FirstOccurrence time.Time        `json:"first_occurrence"` // Timestamp of first error
	LastOccurrence  time.Time        `json:"last_occurrence"`  // Timestamp of most recent error
	TopErrors       []ErrorFrequency `json:"top_errors"`       // Most frequent errors
	ErrorRate       float64          `json:"error_rate"`       // Errors per minute
	Duration        time.Duration    `json:"duration"`         // Time span of error collection
}

// ErrorFrequency represents the frequency of a specific error type
type ErrorFrequency struct {
	Code     ErrorCode `json:"code"`      // Error code
	Count    int       `json:"count"`     // Number of occurrences
	LastSeen time.Time `json:"last_seen"` // Last occurrence timestamp
	Severity string    `json:"severity"`  // Error severity level
	Message  string    `json:"message"`   // Sample error message
}

// ErrorMetrics contains detailed metrics for monitoring integration
type ErrorMetrics struct {
	ErrorCount         map[ErrorCode]int     `json:"error_count"`         // Count by error code
	SeverityCount      map[ErrorSeverity]int `json:"severity_count"`      // Count by severity
	HourlyDistribution map[int]int           `json:"hourly_distribution"` // Errors by hour of day
	ErrorTrends        []ErrorTrend          `json:"error_trends"`        // Error trends over time
}

// ErrorTrend represents error frequency over time
type ErrorTrend struct {
	Timestamp time.Time `json:"timestamp"` // Time bucket
	Count     int       `json:"count"`     // Error count in this bucket
}

// DefaultErrorReporter provides the default implementation of ErrorReporter
type DefaultErrorReporter struct {
	mu              sync.RWMutex
	errors          []reportedError
	startTime       time.Time
	logger          Logger
	enableMetrics   bool
	metricsInterval time.Duration
}

// reportedError represents an error with reporting metadata
type reportedError struct {
	Error     OutputError `json:"error"`
	Timestamp time.Time   `json:"timestamp"`
	Context   string      `json:"context,omitempty"`
}

// NewErrorReporter creates a new DefaultErrorReporter
func NewErrorReporter() *DefaultErrorReporter {
	return &DefaultErrorReporter{
		errors:          make([]reportedError, 0),
		startTime:       time.Now(),
		logger:          &defaultLogger{},
		enableMetrics:   true,
		metricsInterval: time.Minute,
	}
}

// NewErrorReporterWithOptions creates a new DefaultErrorReporter with custom options
func NewErrorReporterWithOptions(logger Logger, enableMetrics bool) *DefaultErrorReporter {
	return &DefaultErrorReporter{
		errors:          make([]reportedError, 0),
		startTime:       time.Now(),
		logger:          logger,
		enableMetrics:   enableMetrics,
		metricsInterval: time.Minute,
	}
}

// Report reports a single error to the monitoring system
func (r *DefaultErrorReporter) Report(err OutputError) {
	if err == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Store the error with timestamp
	reportedErr := reportedError{
		Error:     err,
		Timestamp: time.Now(),
		Context:   err.Context().Operation,
	}
	r.errors = append(r.errors, reportedErr)

	// Log the error with structured data
	r.logError(err)
}

// logError logs the error with structured data for monitoring integration
func (r *DefaultErrorReporter) logError(err OutputError) {
	fields := map[string]interface{}{
		"error_code":  string(err.Code()),
		"severity":    err.Severity().String(),
		"operation":   err.Context().Operation,
		"field":       err.Context().Field,
		"value":       err.Context().Value,
		"suggestions": err.Suggestions(),
		"timestamp":   time.Now().UTC(),
	}

	// Add metadata if present
	if err.Context().Metadata != nil {
		fields["metadata"] = err.Context().Metadata
	}

	message := fmt.Sprintf("Error occurred: %s", err.Error())

	switch err.Severity() {
	case SeverityFatal:
		r.logger.Fatal(message, fields)
	case SeverityError:
		r.logger.Error(message, fields)
	case SeverityWarning:
		r.logger.Warn(message, fields)
	case SeverityInfo:
		r.logger.Info(message, fields)
	}
}

// Summary returns aggregated error statistics
func (r *DefaultErrorReporter) Summary() ExtendedErrorSummary {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.errors) == 0 {
		return ExtendedErrorSummary{
			ErrorSummary: ErrorSummary{
				ByCategory: make(map[ErrorCode]int),
				BySeverity: make(map[ErrorSeverity]int),
			},
			TopErrors: make([]ErrorFrequency, 0),
		}
	}

	summary := ExtendedErrorSummary{
		ErrorSummary: ErrorSummary{
			TotalErrors: len(r.errors),
			ByCategory:  make(map[ErrorCode]int),
			BySeverity:  make(map[ErrorSeverity]int),
			Suggestions: make([]string, 0),
		},
		FirstOccurrence: r.errors[0].Timestamp,
		LastOccurrence:  r.errors[len(r.errors)-1].Timestamp,
		TopErrors:       make([]ErrorFrequency, 0),
	}

	// Calculate duration
	summary.Duration = summary.LastOccurrence.Sub(summary.FirstOccurrence)

	// Calculate error rate (errors per minute)
	if summary.Duration.Minutes() > 0 {
		summary.ErrorRate = float64(summary.TotalErrors) / summary.Duration.Minutes()
	}

	// Track unique suggestions and error frequencies
	suggestionSet := make(map[string]bool)
	errorFreq := make(map[ErrorCode]*ErrorFrequency)

	for _, reportedErr := range r.errors {
		err := reportedErr.Error

		// Count by category
		summary.ByCategory[err.Code()]++

		// Count by severity
		summary.BySeverity[err.Severity()]++

		// Count fixable errors (warnings and info are considered fixable)
		if err.Severity() <= SeverityWarning {
			summary.FixableErrors++
		}

		// Collect unique suggestions
		for _, suggestion := range err.Suggestions() {
			if !suggestionSet[suggestion] {
				summary.Suggestions = append(summary.Suggestions, suggestion)
				suggestionSet[suggestion] = true
			}
		}

		// Track error frequencies
		if freq, exists := errorFreq[err.Code()]; exists {
			freq.Count++
			if reportedErr.Timestamp.After(freq.LastSeen) {
				freq.LastSeen = reportedErr.Timestamp
			}
		} else {
			errorFreq[err.Code()] = &ErrorFrequency{
				Code:     err.Code(),
				Count:    1,
				LastSeen: reportedErr.Timestamp,
				Severity: err.Severity().String(),
				Message:  err.Error(),
			}
		}
	}

	// Convert error frequencies to sorted slice
	for _, freq := range errorFreq {
		summary.TopErrors = append(summary.TopErrors, *freq)
	}

	// Sort by frequency (descending)
	sort.Slice(summary.TopErrors, func(i, j int) bool {
		return summary.TopErrors[i].Count > summary.TopErrors[j].Count
	})

	return summary
}

// GetMetrics returns detailed metrics for monitoring integration
func (r *DefaultErrorReporter) GetMetrics() ErrorMetrics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics := ErrorMetrics{
		ErrorCount:         make(map[ErrorCode]int),
		SeverityCount:      make(map[ErrorSeverity]int),
		HourlyDistribution: make(map[int]int),
		ErrorTrends:        make([]ErrorTrend, 0),
	}

	// Calculate hourly distribution and basic counts
	for _, reportedErr := range r.errors {
		err := reportedErr.Error

		// Count by error code
		metrics.ErrorCount[err.Code()]++

		// Count by severity
		metrics.SeverityCount[err.Severity()]++

		// Count by hour of day
		hour := reportedErr.Timestamp.Hour()
		metrics.HourlyDistribution[hour]++
	}

	// Calculate error trends (bucketed by time interval)
	if len(r.errors) > 0 {
		metrics.ErrorTrends = r.calculateErrorTrends()
	}

	return metrics
}

// calculateErrorTrends calculates error trends over time
func (r *DefaultErrorReporter) calculateErrorTrends() []ErrorTrend {
	if len(r.errors) == 0 {
		return []ErrorTrend{}
	}

	// Use 5-minute buckets for trends
	bucketSize := 5 * time.Minute
	startTime := r.errors[0].Timestamp.Truncate(bucketSize)
	endTime := r.errors[len(r.errors)-1].Timestamp.Truncate(bucketSize).Add(bucketSize)

	buckets := make(map[time.Time]int)

	// Initialize all buckets
	for t := startTime; t.Before(endTime); t = t.Add(bucketSize) {
		buckets[t] = 0
	}

	// Count errors in each bucket
	for _, reportedErr := range r.errors {
		bucket := reportedErr.Timestamp.Truncate(bucketSize)
		buckets[bucket]++
	}

	// Convert to sorted slice
	trends := make([]ErrorTrend, 0, len(buckets))
	for timestamp, count := range buckets {
		trends = append(trends, ErrorTrend{
			Timestamp: timestamp,
			Count:     count,
		})
	}

	// Sort by timestamp
	sort.Slice(trends, func(i, j int) bool {
		return trends[i].Timestamp.Before(trends[j].Timestamp)
	})

	return trends
}

// Reset clears all collected error data
func (r *DefaultErrorReporter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.errors = make([]reportedError, 0)
	r.startTime = time.Now()
}

// SetLogger sets a custom logger for structured logging
func (r *DefaultErrorReporter) SetLogger(logger Logger) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logger = logger
}

// GetErrorHistory returns the complete error history
func (r *DefaultErrorReporter) GetErrorHistory() []reportedError {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	history := make([]reportedError, len(r.errors))
	copy(history, r.errors)
	return history
}

// GetErrorsByTimeRange returns errors within a specific time range
func (r *DefaultErrorReporter) GetErrorsByTimeRange(start, end time.Time) []reportedError {
	r.mu.RLock()
	defer r.mu.RUnlock()

	filtered := make([]reportedError, 0)
	for _, err := range r.errors {
		if err.Timestamp.After(start) && err.Timestamp.Before(end) {
			filtered = append(filtered, err)
		}
	}

	return filtered
}

// GetErrorsByCode returns all errors with a specific error code
func (r *DefaultErrorReporter) GetErrorsByCode(code ErrorCode) []reportedError {
	r.mu.RLock()
	defer r.mu.RUnlock()

	filtered := make([]reportedError, 0)
	for _, err := range r.errors {
		if err.Error.Code() == code {
			filtered = append(filtered, err)
		}
	}

	return filtered
}

// GetErrorsBySeverity returns all errors with a specific severity level
func (r *DefaultErrorReporter) GetErrorsBySeverity(severity ErrorSeverity) []reportedError {
	r.mu.RLock()
	defer r.mu.RUnlock()

	filtered := make([]reportedError, 0)
	for _, err := range r.errors {
		if err.Error.Severity() == severity {
			filtered = append(filtered, err)
		}
	}

	return filtered
}

// ExportJSON exports the error summary as JSON for external monitoring systems
func (r *DefaultErrorReporter) ExportJSON() ([]byte, error) {
	summary := r.Summary()
	return json.MarshalIndent(summary, "", "  ")
}

// ExportMetricsJSON exports detailed metrics as JSON
func (r *DefaultErrorReporter) ExportMetricsJSON() ([]byte, error) {
	metrics := r.GetMetrics()
	return json.MarshalIndent(metrics, "", "  ")
}

// defaultLogger provides a basic implementation of the Logger interface
type defaultLogger struct{}

// Info logs an informational message
func (l *defaultLogger) Info(msg string, fields map[string]interface{}) {
	l.logWithFields("INFO", msg, fields)
}

// Warn logs a warning message
func (l *defaultLogger) Warn(msg string, fields map[string]interface{}) {
	l.logWithFields("WARN", msg, fields)
}

// Error logs an error message
func (l *defaultLogger) Error(msg string, fields map[string]interface{}) {
	l.logWithFields("ERROR", msg, fields)
}

// Fatal logs a fatal error message
func (l *defaultLogger) Fatal(msg string, fields map[string]interface{}) {
	l.logWithFields("FATAL", msg, fields)
}

// logWithFields logs a message with structured fields
func (l *defaultLogger) logWithFields(level, msg string, fields map[string]interface{}) {
	// Create a structured log entry
	entry := map[string]interface{}{
		"level":     level,
		"message":   msg,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	// Add all fields
	for k, v := range fields {
		entry[k] = v
	}

	// Marshal to JSON for structured logging
	if jsonData, err := json.Marshal(entry); err == nil {
		log.Println(string(jsonData))
	} else {
		// Fallback to simple logging if JSON marshaling fails
		log.Printf("[%s] %s %v", level, msg, fields)
	}
}

// MonitoringIntegration provides integration with external monitoring systems
type MonitoringIntegration struct {
	reporter    ErrorReporter
	webhookURL  string
	apiKey      string
	serviceName string
}

// NewMonitoringIntegration creates a new monitoring integration
func NewMonitoringIntegration(reporter ErrorReporter, serviceName string) *MonitoringIntegration {
	return &MonitoringIntegration{
		reporter:    reporter,
		serviceName: serviceName,
	}
}

// SetWebhook configures webhook integration for external monitoring
func (m *MonitoringIntegration) SetWebhook(url, apiKey string) {
	m.webhookURL = url
	m.apiKey = apiKey
}

// SendSummaryToMonitoring sends error summary to external monitoring system
func (m *MonitoringIntegration) SendSummaryToMonitoring() error {
	summary := m.reporter.Summary()

	// Create monitoring payload
	payload := map[string]interface{}{
		"service":   m.serviceName,
		"timestamp": time.Now().UTC(),
		"summary":   summary,
		"metrics":   m.reporter.(*DefaultErrorReporter).GetMetrics(),
	}

	// In a real implementation, this would send the payload to the monitoring system
	// For now, we'll just log it as structured data
	if jsonData, err := json.MarshalIndent(payload, "", "  "); err == nil {
		log.Printf("Monitoring payload: %s", string(jsonData))
	}

	return nil
}

// AlertThreshold defines thresholds for triggering alerts
type AlertThreshold struct {
	ErrorRate     float64       // Errors per minute threshold
	TotalErrors   int           // Total error count threshold
	SeverityLevel ErrorSeverity // Minimum severity level for alerts
	TimeWindow    time.Duration // Time window for rate calculations
}

// CheckThresholds checks if any alert thresholds are exceeded
func (m *MonitoringIntegration) CheckThresholds(threshold AlertThreshold) bool {
	summary := m.reporter.Summary()

	// Check error rate threshold
	if summary.ErrorRate > threshold.ErrorRate {
		return true
	}

	// Check total error threshold
	if summary.TotalErrors > threshold.TotalErrors {
		return true
	}

	// Check severity threshold
	for severity, count := range summary.BySeverity {
		if severity >= threshold.SeverityLevel && count > 0 {
			return true
		}
	}

	return false
}
