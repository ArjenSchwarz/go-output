package format

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ArjenSchwarz/go-output/errors"
)

// ErrorReporter interface defines methods for error reporting and metrics collection
type ErrorReporter interface {
	Report(err errors.OutputError)
	Summary() ErrorSummary
	Clear()
}

// ErrorSummary contains comprehensive statistics about collected errors
type ErrorSummary struct {
	TotalErrors    int                               `json:"total_errors"`
	ByCategory     map[errors.ErrorCode]int          `json:"by_category"`
	BySeverity     map[errors.ErrorSeverity]int      `json:"by_severity"`
	Suggestions    []string                          `json:"suggestions"`
	FixableErrors  int                               `json:"fixable_errors"`
	Timestamp      time.Time                         `json:"timestamp"`
	TimeRange      TimeRange                         `json:"time_range,omitempty"`
	TopErrors      []ErrorFrequency                  `json:"top_errors,omitempty"`
	ContextSummary map[string]map[string]interface{} `json:"context_summary,omitempty"`
}

// TimeRange represents a time range for error collection
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ErrorFrequency represents the frequency of a specific error
type ErrorFrequency struct {
	Code      errors.ErrorCode `json:"code"`
	Count     int              `json:"count"`
	Message   string           `json:"message"`
	LastSeen  time.Time        `json:"last_seen"`
	Frequency float64          `json:"frequency"`
}

// DefaultErrorReporter implements ErrorReporter with basic functionality
type DefaultErrorReporter struct {
	mu                sync.RWMutex
	errors            []ErrorEntry
	categoryCount     map[errors.ErrorCode]int
	severityCount     map[errors.ErrorSeverity]int
	allSuggestions    map[string]bool
	startTime         time.Time
	lastReportTime    time.Time
	contextAggregator *ContextAggregator
}

// ErrorEntry represents a single error entry with metadata
type ErrorEntry struct {
	Error     errors.OutputError  `json:"error"`
	Timestamp time.Time           `json:"timestamp"`
	Context   errors.ErrorContext `json:"context"`
}

// ContextAggregator aggregates error context for analysis
type ContextAggregator struct {
	mu         sync.RWMutex
	operations map[string]int
	fields     map[string]int
	values     map[string]int
}

// NewDefaultErrorReporter creates a new default error reporter
func NewDefaultErrorReporter() *DefaultErrorReporter {
	return &DefaultErrorReporter{
		errors:            make([]ErrorEntry, 0),
		categoryCount:     make(map[errors.ErrorCode]int),
		severityCount:     make(map[errors.ErrorSeverity]int),
		allSuggestions:    make(map[string]bool),
		startTime:         time.Now(),
		contextAggregator: NewContextAggregator(),
	}
}

// NewContextAggregator creates a new context aggregator
func NewContextAggregator() *ContextAggregator {
	return &ContextAggregator{
		operations: make(map[string]int),
		fields:     make(map[string]int),
		values:     make(map[string]int),
	}
}

// Report records an error for analysis and metrics
func (r *DefaultErrorReporter) Report(err errors.OutputError) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	entry := ErrorEntry{
		Error:     err,
		Timestamp: now,
		Context:   err.Context(),
	}

	r.errors = append(r.errors, entry)
	r.categoryCount[err.Code()]++
	r.severityCount[err.Severity()]++
	r.lastReportTime = now

	// Collect suggestions
	for _, suggestion := range err.Suggestions() {
		r.allSuggestions[suggestion] = true
	}

	// Aggregate context information
	r.contextAggregator.AddContext(err.Context())
}

// Summary returns a comprehensive summary of all reported errors
func (r *DefaultErrorReporter) Summary() ErrorSummary {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Convert suggestions map to slice
	suggestions := make([]string, 0, len(r.allSuggestions))
	for suggestion := range r.allSuggestions {
		suggestions = append(suggestions, suggestion)
	}
	sort.Strings(suggestions)

	// Calculate fixable errors (warnings and info that have suggestions)
	fixableCount := 0
	for _, entry := range r.errors {
		if (entry.Error.Severity() == errors.SeverityWarning || entry.Error.Severity() == errors.SeverityInfo) &&
			len(entry.Error.Suggestions()) > 0 {
			fixableCount++
		}
	}

	// Generate top errors
	topErrors := r.generateTopErrors(5)

	// Get time range
	var timeRange TimeRange
	if len(r.errors) > 0 {
		timeRange = TimeRange{
			Start: r.startTime,
			End:   r.lastReportTime,
		}
	}

	return ErrorSummary{
		TotalErrors:    len(r.errors),
		ByCategory:     r.copyMap(r.categoryCount),
		BySeverity:     r.copySeverityMap(r.severityCount),
		Suggestions:    suggestions,
		FixableErrors:  fixableCount,
		Timestamp:      time.Now(),
		TimeRange:      timeRange,
		TopErrors:      topErrors,
		ContextSummary: r.contextAggregator.GetSummary(),
	}
}

// Clear removes all collected errors and resets counters
func (r *DefaultErrorReporter) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.errors = make([]ErrorEntry, 0)
	r.categoryCount = make(map[errors.ErrorCode]int)
	r.severityCount = make(map[errors.ErrorSeverity]int)
	r.allSuggestions = make(map[string]bool)
	r.startTime = time.Now()
	r.contextAggregator = NewContextAggregator()
}

// generateTopErrors generates a list of most frequent errors
func (r *DefaultErrorReporter) generateTopErrors(limit int) []ErrorFrequency {
	if len(r.errors) == 0 {
		return nil
	}

	// Count error frequency by code and message
	errorFreq := make(map[string]*ErrorFrequency)
	totalTime := r.lastReportTime.Sub(r.startTime).Seconds()
	if totalTime <= 0 {
		totalTime = 1 // Avoid division by zero
	}

	for _, entry := range r.errors {
		key := fmt.Sprintf("%s:%s", entry.Error.Code(), entry.Error.Error())
		if freq, exists := errorFreq[key]; exists {
			freq.Count++
			if entry.Timestamp.After(freq.LastSeen) {
				freq.LastSeen = entry.Timestamp
			}
		} else {
			errorFreq[key] = &ErrorFrequency{
				Code:     entry.Error.Code(),
				Count:    1,
				Message:  entry.Error.Error(),
				LastSeen: entry.Timestamp,
			}
		}
	}

	// Calculate frequency per second and sort
	freqList := make([]ErrorFrequency, 0, len(errorFreq))
	for _, freq := range errorFreq {
		freq.Frequency = float64(freq.Count) / totalTime
		freqList = append(freqList, *freq)
	}

	sort.Slice(freqList, func(i, j int) bool {
		return freqList[i].Count > freqList[j].Count
	})

	if len(freqList) > limit {
		freqList = freqList[:limit]
	}

	return freqList
}

// copyMap creates a copy of the error code map
func (r *DefaultErrorReporter) copyMap(original map[errors.ErrorCode]int) map[errors.ErrorCode]int {
	copy := make(map[errors.ErrorCode]int)
	for k, v := range original {
		copy[k] = v
	}
	return copy
}

// copySeverityMap creates a copy of the severity map
func (r *DefaultErrorReporter) copySeverityMap(original map[errors.ErrorSeverity]int) map[errors.ErrorSeverity]int {
	copy := make(map[errors.ErrorSeverity]int)
	for k, v := range original {
		copy[k] = v
	}
	return copy
}

// AddContext adds context information to the aggregator
func (ca *ContextAggregator) AddContext(ctx errors.ErrorContext) {
	ca.mu.Lock()
	defer ca.mu.Unlock()

	if ctx.Operation != "" {
		ca.operations[ctx.Operation]++
	}
	if ctx.Field != "" {
		ca.fields[ctx.Field]++
	}
	if ctx.Value != nil {
		valueStr := fmt.Sprintf("%v", ctx.Value)
		if len(valueStr) < 100 { // Avoid very long values
			ca.values[valueStr]++
		}
	}
}

// GetSummary returns a summary of aggregated context information
func (ca *ContextAggregator) GetSummary() map[string]map[string]interface{} {
	ca.mu.RLock()
	defer ca.mu.RUnlock()

	summary := make(map[string]map[string]interface{})

	if len(ca.operations) > 0 {
		summary["operations"] = make(map[string]interface{})
		for op, count := range ca.operations {
			summary["operations"][op] = count
		}
	}

	if len(ca.fields) > 0 {
		summary["fields"] = make(map[string]interface{})
		for field, count := range ca.fields {
			summary["fields"][field] = count
		}
	}

	if len(ca.values) > 0 {
		summary["common_values"] = make(map[string]interface{})
		for value, count := range ca.values {
			summary["common_values"][value] = count
		}
	}

	return summary
}

// FormatText returns a human-readable text representation of the error summary
func (es *ErrorSummary) FormatText() string {
	var sb strings.Builder

	sb.WriteString("=== Error Summary ===\n")
	sb.WriteString(fmt.Sprintf("Generated at: %s\n", es.Timestamp.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Total Errors: %d\n", es.TotalErrors))

	if es.FixableErrors > 0 {
		sb.WriteString(fmt.Sprintf("Fixable Errors: %d\n", es.FixableErrors))
	}

	if !es.TimeRange.Start.IsZero() {
		duration := es.TimeRange.End.Sub(es.TimeRange.Start)
		sb.WriteString(fmt.Sprintf("Time Range: %s (Duration: %s)\n",
			es.TimeRange.Start.Format(time.RFC3339), duration.String()))
	}

	// Category breakdown
	if len(es.ByCategory) > 0 {
		sb.WriteString("\nBy Category:\n")
		// Sort categories by count (descending)
		type categoryCount struct {
			code  errors.ErrorCode
			count int
		}
		categories := make([]categoryCount, 0, len(es.ByCategory))
		for code, count := range es.ByCategory {
			categories = append(categories, categoryCount{code, count})
		}
		sort.Slice(categories, func(i, j int) bool {
			return categories[i].count > categories[j].count
		})

		for _, cat := range categories {
			sb.WriteString(fmt.Sprintf("  %s: %d\n", cat.code, cat.count))
		}
	}

	// Severity breakdown
	if len(es.BySeverity) > 0 {
		sb.WriteString("\nBy Severity:\n")
		severityOrder := []errors.ErrorSeverity{
			errors.SeverityFatal, errors.SeverityError,
			errors.SeverityWarning, errors.SeverityInfo,
		}
		for _, severity := range severityOrder {
			if count, exists := es.BySeverity[severity]; exists && count > 0 {
				sb.WriteString(fmt.Sprintf("  %s: %d\n", strings.ToUpper(severity.String()), count))
			}
		}
	}

	// Top errors
	if len(es.TopErrors) > 0 {
		sb.WriteString("\nMost Frequent Errors:\n")
		for i, topError := range es.TopErrors {
			sb.WriteString(fmt.Sprintf("  %d. %s (Count: %d, Rate: %.2f/sec)\n",
				i+1, topError.Code, topError.Count, topError.Frequency))
			if len(topError.Message) > 60 {
				sb.WriteString(fmt.Sprintf("     Message: %s...\n", topError.Message[:60]))
			} else {
				sb.WriteString(fmt.Sprintf("     Message: %s\n", topError.Message))
			}
		}
	}

	// Suggestions
	if len(es.Suggestions) > 0 {
		sb.WriteString("\nSuggestions:\n")
		for _, suggestion := range es.Suggestions {
			sb.WriteString(fmt.Sprintf("  • %s\n", suggestion))
		}
	}

	// Context summary
	if len(es.ContextSummary) > 0 {
		sb.WriteString("\nContext Analysis:\n")
		if operations, exists := es.ContextSummary["operations"]; exists {
			sb.WriteString("  Most Common Operations:\n")
			for op, count := range operations {
				sb.WriteString(fmt.Sprintf("    %s: %v\n", op, count))
			}
		}
		if fields, exists := es.ContextSummary["fields"]; exists {
			sb.WriteString("  Most Common Problem Fields:\n")
			for field, count := range fields {
				sb.WriteString(fmt.Sprintf("    %s: %v\n", field, count))
			}
		}
	}

	return sb.String()
}

// ErrorMetrics provides time-based metrics for error analysis
type ErrorMetrics struct {
	mu           sync.RWMutex
	errors       []TimestampedError
	totalCount   int
	categoryFreq map[errors.ErrorCode]int
}

// TimestampedError represents an error with its occurrence time
type TimestampedError struct {
	Error     errors.OutputError
	Timestamp time.Time
}

// NewErrorMetrics creates a new error metrics collector
func NewErrorMetrics() *ErrorMetrics {
	return &ErrorMetrics{
		errors:       make([]TimestampedError, 0),
		categoryFreq: make(map[errors.ErrorCode]int),
	}
}

// RecordError records an error with timestamp for metrics
func (em *ErrorMetrics) RecordError(err errors.OutputError) {
	em.mu.Lock()
	defer em.mu.Unlock()

	timestampedError := TimestampedError{
		Error:     err,
		Timestamp: time.Now(),
	}

	em.errors = append(em.errors, timestampedError)
	em.totalCount++
	em.categoryFreq[err.Code()]++
}

// GetTotalErrorCount returns the total number of recorded errors
func (em *ErrorMetrics) GetTotalErrorCount() int {
	em.mu.RLock()
	defer em.mu.RUnlock()
	return em.totalCount
}

// GetErrorCountInTimeRange returns the number of errors in a specific time range
func (em *ErrorMetrics) GetErrorCountInTimeRange(start, end time.Time) int {
	em.mu.RLock()
	defer em.mu.RUnlock()

	count := 0
	for _, err := range em.errors {
		if err.Timestamp.After(start) && err.Timestamp.Before(end) {
			count++
		}
	}
	return count
}

// GetErrorRate returns the error rate per given duration
func (em *ErrorMetrics) GetErrorRate(duration time.Duration) float64 {
	em.mu.RLock()
	defer em.mu.RUnlock()

	if len(em.errors) == 0 {
		return 0
	}

	now := time.Now()
	start := now.Add(-duration)

	count := em.GetErrorCountInTimeRange(start, now)
	return float64(count) / duration.Seconds()
}

// GetMostFrequentErrors returns the most frequent error types
func (em *ErrorMetrics) GetMostFrequentErrors(limit int) []ErrorFrequency {
	em.mu.RLock()
	defer em.mu.RUnlock()

	if len(em.categoryFreq) == 0 {
		return nil
	}

	freqList := make([]ErrorFrequency, 0, len(em.categoryFreq))
	for code, count := range em.categoryFreq {
		freqList = append(freqList, ErrorFrequency{
			Code:  code,
			Count: count,
		})
	}

	sort.Slice(freqList, func(i, j int) bool {
		return freqList[i].Count > freqList[j].Count
	})

	if len(freqList) > limit {
		freqList = freqList[:limit]
	}

	return freqList
}

// StructuredLogger provides structured logging for errors
type StructuredLogger struct {
	serviceName string
	version     string
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp   time.Time           `json:"timestamp"`
	Level       string              `json:"level"`
	Service     string              `json:"service,omitempty"`
	Version     string              `json:"version,omitempty"`
	Message     string              `json:"message"`
	ErrorCode   string              `json:"error_code"`
	Severity    string              `json:"severity"`
	Context     errors.ErrorContext `json:"context"`
	Suggestions []string            `json:"suggestions"`
	TraceID     string              `json:"trace_id,omitempty"`
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger() *StructuredLogger {
	return &StructuredLogger{
		serviceName: "go-output",
		version:     "1.0.0",
	}
}

// LogError logs an error as a structured log entry
func (sl *StructuredLogger) LogError(err errors.OutputError) LogEntry {
	entry := LogEntry{
		Timestamp:   time.Now(),
		Level:       sl.severityToLogLevel(err.Severity()),
		Service:     sl.serviceName,
		Version:     sl.version,
		Message:     err.Error(),
		ErrorCode:   string(err.Code()),
		Severity:    err.Severity().String(),
		Context:     err.Context(),
		Suggestions: err.Suggestions(),
	}

	return entry
}

// severityToLogLevel converts error severity to log level
func (sl *StructuredLogger) severityToLogLevel(severity errors.ErrorSeverity) string {
	switch severity {
	case errors.SeverityFatal:
		return "FATAL"
	case errors.SeverityError:
		return "ERROR"
	case errors.SeverityWarning:
		return "WARN"
	case errors.SeverityInfo:
		return "INFO"
	default:
		return "ERROR"
	}
}

// PrometheusMetricsExporter exports error metrics in Prometheus format
type PrometheusMetricsExporter struct {
	mu         sync.RWMutex
	errorCount map[errors.ErrorCode]int
}

// NewPrometheusMetricsExporter creates a new Prometheus metrics exporter
func NewPrometheusMetricsExporter() *PrometheusMetricsExporter {
	return &PrometheusMetricsExporter{
		errorCount: make(map[errors.ErrorCode]int),
	}
}

// RecordError records an error for Prometheus metrics
func (pe *PrometheusMetricsExporter) RecordError(err errors.OutputError) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.errorCount[err.Code()]++
}

// Export returns metrics in Prometheus format
func (pe *PrometheusMetricsExporter) Export() string {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	var sb strings.Builder

	sb.WriteString("# HELP go_output_errors_total Total number of errors\n")
	sb.WriteString("# TYPE go_output_errors_total counter\n")

	// Sort error codes for consistent output
	codes := make([]errors.ErrorCode, 0, len(pe.errorCount))
	for code := range pe.errorCount {
		codes = append(codes, code)
	}
	sort.Slice(codes, func(i, j int) bool {
		return string(codes[i]) < string(codes[j])
	})

	for _, code := range codes {
		count := pe.errorCount[code]
		sb.WriteString(fmt.Sprintf("go_output_errors_total{code=\"%s\"} %d\n", code, count))
	}

	return sb.String()
}

// ErrorReportingConfig configures error reporting behavior
type ErrorReportingConfig struct {
	MinSeverity       errors.ErrorSeverity
	IncludeCategories []errors.ErrorCode
	ExcludeCategories []errors.ErrorCode
	IncludeContext    bool
	MaxErrors         int
	TimeWindow        time.Duration
}

// ConfigurableErrorReporter implements ErrorReporter with configuration options
type ConfigurableErrorReporter struct {
	*DefaultErrorReporter
	config ErrorReportingConfig
}

// NewConfigurableErrorReporter creates a new configurable error reporter
func NewConfigurableErrorReporter(config ErrorReportingConfig) *ConfigurableErrorReporter {
	return &ConfigurableErrorReporter{
		DefaultErrorReporter: NewDefaultErrorReporter(),
		config:               config,
	}
}

// Report reports an error if it matches the configuration criteria
func (cr *ConfigurableErrorReporter) Report(err errors.OutputError) {
	// Check severity filter
	if err.Severity() > cr.config.MinSeverity {
		return
	}

	// Check category inclusion/exclusion
	if len(cr.config.IncludeCategories) > 0 {
		included := false
		for _, includeCode := range cr.config.IncludeCategories {
			if err.Code() == includeCode {
				included = true
				break
			}
		}
		if !included {
			return
		}
	}

	for _, excludeCode := range cr.config.ExcludeCategories {
		if err.Code() == excludeCode {
			return
		}
	}

	// Check max errors limit
	if cr.config.MaxErrors > 0 && len(cr.errors) >= cr.config.MaxErrors {
		return
	}

	// Report the error
	cr.DefaultErrorReporter.Report(err)
}

// AggregateSummaries aggregates multiple error summaries into one
func AggregateSummaries(summaries ...ErrorSummary) ErrorSummary {
	if len(summaries) == 0 {
		return ErrorSummary{
			ByCategory:     make(map[errors.ErrorCode]int),
			BySeverity:     make(map[errors.ErrorSeverity]int),
			Suggestions:    make([]string, 0),
			ContextSummary: make(map[string]map[string]interface{}),
			Timestamp:      time.Now(),
		}
	}

	aggregated := ErrorSummary{
		ByCategory:     make(map[errors.ErrorCode]int),
		BySeverity:     make(map[errors.ErrorSeverity]int),
		Suggestions:    make([]string, 0),
		ContextSummary: make(map[string]map[string]interface{}),
		Timestamp:      time.Now(),
	}

	suggestionSet := make(map[string]bool)
	var earliestStart, latestEnd time.Time

	for i, summary := range summaries {
		aggregated.TotalErrors += summary.TotalErrors
		aggregated.FixableErrors += summary.FixableErrors

		// Aggregate categories
		for code, count := range summary.ByCategory {
			aggregated.ByCategory[code] += count
		}

		// Aggregate severities
		for severity, count := range summary.BySeverity {
			aggregated.BySeverity[severity] += count
		}

		// Collect unique suggestions
		for _, suggestion := range summary.Suggestions {
			suggestionSet[suggestion] = true
		}

		// Track time range
		if i == 0 || summary.TimeRange.Start.Before(earliestStart) {
			earliestStart = summary.TimeRange.Start
		}
		if i == 0 || summary.TimeRange.End.After(latestEnd) {
			latestEnd = summary.TimeRange.End
		}
	}

	// Convert suggestions set to slice
	for suggestion := range suggestionSet {
		aggregated.Suggestions = append(aggregated.Suggestions, suggestion)
	}
	sort.Strings(aggregated.Suggestions)

	// Set time range
	if !earliestStart.IsZero() {
		aggregated.TimeRange = TimeRange{
			Start: earliestStart,
			End:   latestEnd,
		}
	}

	return aggregated
}
