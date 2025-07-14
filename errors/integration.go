package errors

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// MigrationHelper provides utilities to ease transition from log.Fatal to new error handling
type MigrationHelper struct {
	legacyMode     bool
	migrationSteps []MigrationStep
}

// MigrationStep represents a single step in the migration process
type MigrationStep struct {
	Name        string
	Description string
	Check       func() (bool, string)
	Fix         func() error
}

// NewMigrationHelper creates a new migration helper
func NewMigrationHelper() *MigrationHelper {
	return &MigrationHelper{
		legacyMode:     false,
		migrationSteps: make([]MigrationStep, 0),
	}
}

// EnableLegacyMode enables legacy mode for gradual migration
func (m *MigrationHelper) EnableLegacyMode() {
	m.legacyMode = true
}

// DisableLegacyMode disables legacy mode
func (m *MigrationHelper) DisableLegacyMode() {
	m.legacyMode = false
}

// IsLegacyMode returns true if legacy mode is enabled
func (m *MigrationHelper) IsLegacyMode() bool {
	return m.legacyMode
}

// AddMigrationStep adds a migration step to the helper
func (m *MigrationHelper) AddMigrationStep(step MigrationStep) {
	m.migrationSteps = append(m.migrationSteps, step)
}

// CheckMigrationStatus checks the status of all migration steps
func (m *MigrationHelper) CheckMigrationStatus() MigrationStatus {
	status := MigrationStatus{
		TotalSteps:     len(m.migrationSteps),
		CompletedSteps: 0,
		FailedSteps:    0,
		Results:        make([]MigrationStepResult, 0),
		StartTime:      time.Now(),
	}

	for _, step := range m.migrationSteps {
		result := MigrationStepResult{
			Step: step,
		}

		passed, message := step.Check()
		result.Passed = passed
		result.Message = message

		if passed {
			status.CompletedSteps++
		} else {
			status.FailedSteps++
		}

		status.Results = append(status.Results, result)
	}

	status.EndTime = time.Now()
	status.Duration = status.EndTime.Sub(status.StartTime)

	return status
}

// RunMigration executes the migration process
func (m *MigrationHelper) RunMigration() MigrationResult {
	result := MigrationResult{
		Status:    m.CheckMigrationStatus(),
		FixedSteps: 0,
		Errors:    make([]error, 0),
		StartTime: time.Now(),
	}

	for i, stepResult := range result.Status.Results {
		if !stepResult.Passed && stepResult.Step.Fix != nil {
			if err := stepResult.Step.Fix(); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to fix step %s: %w", stepResult.Step.Name, err))
			} else {
				result.FixedSteps++
				// Re-check the step
				passed, message := stepResult.Step.Check()
				result.Status.Results[i].Passed = passed
				result.Status.Results[i].Message = message
			}
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result
}

// MigrationStatus represents the current migration status
type MigrationStatus struct {
	TotalSteps     int
	CompletedSteps int
	FailedSteps    int
	Results        []MigrationStepResult
	StartTime      time.Time
	EndTime        time.Time
	Duration       time.Duration
}

// MigrationStepResult represents the result of a single migration step
type MigrationStepResult struct {
	Step    MigrationStep
	Passed  bool
	Message string
}

// MigrationResult represents the result of running the migration
type MigrationResult struct {
	Status     MigrationStatus
	FixedSteps int
	Errors     []error
	StartTime  time.Time
	EndTime    time.Time
	Duration   time.Duration
}

// IsComplete returns true if all migration steps are completed
func (s MigrationStatus) IsComplete() bool {
	return s.FailedSteps == 0
}

// CompletionPercentage returns the percentage of completed steps
func (s MigrationStatus) CompletionPercentage() float64 {
	if s.TotalSteps == 0 {
		return 100.0
	}
	return float64(s.CompletedSteps) / float64(s.TotalSteps) * 100.0
}

// String returns a human-readable status summary
func (s MigrationStatus) String() string {
	return fmt.Sprintf("Migration Status: %d/%d steps completed (%.1f%%)",
		s.CompletedSteps, s.TotalSteps, s.CompletionPercentage())
}

// String returns a human-readable result summary
func (r MigrationResult) String() string {
	success := "SUCCESS"
	if len(r.Errors) > 0 {
		success = "PARTIAL"
	}
	if r.Status.FailedSteps > r.FixedSteps {
		success = "FAILED"
	}

	return fmt.Sprintf("Migration Result: %s - %d/%d steps completed, %d fixed, %d errors (took %v)",
		success, r.Status.CompletedSteps, r.Status.TotalSteps,
		r.FixedSteps, len(r.Errors), r.Duration)
}

// PerformanceProfiler profiles the performance of error handling operations
type PerformanceProfiler struct {
	enabled bool
	metrics []PerformanceMetric
}

// PerformanceMetric represents a single performance measurement
type PerformanceMetric struct {
	Operation   string
	Duration    time.Duration
	MemoryUsage int64
	ErrorCount  int
	Timestamp   time.Time
}

// NewPerformanceProfiler creates a new performance profiler
func NewPerformanceProfiler() *PerformanceProfiler {
	return &PerformanceProfiler{
		enabled: false,
		metrics: make([]PerformanceMetric, 0),
	}
}

// Enable enables performance profiling
func (p *PerformanceProfiler) Enable() {
	p.enabled = true
}

// Disable disables performance profiling
func (p *PerformanceProfiler) Disable() {
	p.enabled = false
}

// IsEnabled returns true if profiling is enabled
func (p *PerformanceProfiler) IsEnabled() bool {
	return p.enabled
}

// ProfileOperation profiles a specific operation
func (p *PerformanceProfiler) ProfileOperation(operation string, fn func() error) error {
	if !p.enabled {
		return fn()
	}

	startTime := time.Now()
	err := fn()
	duration := time.Since(startTime)

	metric := PerformanceMetric{
		Operation: operation,
		Duration:  duration,
		Timestamp: startTime,
	}

	if err != nil {
		metric.ErrorCount = 1
	}

	p.metrics = append(p.metrics, metric)
	return err
}

// GetMetrics returns all collected performance metrics
func (p *PerformanceProfiler) GetMetrics() []PerformanceMetric {
	return p.metrics
}

// Clear clears all collected metrics
func (p *PerformanceProfiler) Clear() {
	p.metrics = make([]PerformanceMetric, 0)
}

// GetAverageTime returns the average time for a specific operation
func (p *PerformanceProfiler) GetAverageTime(operation string) time.Duration {
	var total time.Duration
	count := 0

	for _, metric := range p.metrics {
		if metric.Operation == operation {
			total += metric.Duration
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return total / time.Duration(count)
}

// GetTotalErrors returns the total number of errors for a specific operation
func (p *PerformanceProfiler) GetTotalErrors(operation string) int {
	total := 0
	for _, metric := range p.metrics {
		if metric.Operation == operation {
			total += metric.ErrorCount
		}
	}
	return total
}

// PerformanceReport generates a performance report
func (p *PerformanceProfiler) PerformanceReport() PerformanceReport {
	report := PerformanceReport{
		TotalOperations: len(p.metrics),
		Operations:      make(map[string]OperationStats),
		StartTime:       time.Now(),
		EndTime:         time.Now(),
	}

	if len(p.metrics) > 0 {
		report.StartTime = p.metrics[0].Timestamp
		report.EndTime = p.metrics[len(p.metrics)-1].Timestamp
	}

	// Calculate statistics per operation
	for _, metric := range p.metrics {
		stats, exists := report.Operations[metric.Operation]
		if !exists {
			stats = OperationStats{
				Count:      0,
				TotalTime:  0,
				TotalErrors: 0,
				MinTime:    metric.Duration,
				MaxTime:    metric.Duration,
			}
		}

		stats.Count++
		stats.TotalTime += metric.Duration
		stats.TotalErrors += metric.ErrorCount

		if metric.Duration < stats.MinTime {
			stats.MinTime = metric.Duration
		}
		if metric.Duration > stats.MaxTime {
			stats.MaxTime = metric.Duration
		}

		report.Operations[metric.Operation] = stats
	}

	// Calculate average times
	for operation, stats := range report.Operations {
		if stats.Count > 0 {
			stats.AverageTime = stats.TotalTime / time.Duration(stats.Count)
			report.Operations[operation] = stats
		}
	}

	return report
}

// PerformanceReport contains performance statistics
type PerformanceReport struct {
	TotalOperations int
	Operations      map[string]OperationStats
	StartTime       time.Time
	EndTime         time.Time
}

// OperationStats contains statistics for a specific operation
type OperationStats struct {
	Count       int
	TotalTime   time.Duration
	AverageTime time.Duration
	MinTime     time.Duration
	MaxTime     time.Duration
	TotalErrors int
}

// String returns a human-readable performance report
func (r PerformanceReport) String() string {
	duration := r.EndTime.Sub(r.StartTime)
	result := fmt.Sprintf("Performance Report (%d operations over %v):\n", r.TotalOperations, duration)

	for operation, stats := range r.Operations {
		errorRate := float64(stats.TotalErrors) / float64(stats.Count) * 100
		result += fmt.Sprintf("  %s: %d ops, avg %v (min %v, max %v), %.1f%% errors\n",
			operation, stats.Count, stats.AverageTime, stats.MinTime, stats.MaxTime, errorRate)
	}

	return result
}

// IntegratedErrorSystem combines all error handling components for end-to-end scenarios
type IntegratedErrorSystem struct {
	handler         ErrorHandler
	reporter        interface{} // ErrorReporter from main package
	recoveryHandler RecoveryHandler
	profiler        *PerformanceProfiler
	migrationHelper *MigrationHelper
}

// NewIntegratedErrorSystem creates a new integrated error system
func NewIntegratedErrorSystem() *IntegratedErrorSystem {
	return &IntegratedErrorSystem{
		handler:         NewDefaultErrorHandler(),
		reporter:        nil, // Would be set by main package
		recoveryHandler: NewDefaultRecoveryHandler(),
		profiler:        NewPerformanceProfiler(),
		migrationHelper: NewMigrationHelper(),
	}
}

// SetErrorHandler sets the error handler
func (s *IntegratedErrorSystem) SetErrorHandler(handler ErrorHandler) {
	s.handler = handler
}

// SetErrorReporter sets the error reporter
func (s *IntegratedErrorSystem) SetErrorReporter(reporter interface{}) {
	s.reporter = reporter
}

// SetRecoveryHandler sets the recovery handler
func (s *IntegratedErrorSystem) SetRecoveryHandler(handler RecoveryHandler) {
	s.recoveryHandler = handler
}

// EnableProfiling enables performance profiling
func (s *IntegratedErrorSystem) EnableProfiling() {
	s.profiler.Enable()
}

// DisableProfiling disables performance profiling
func (s *IntegratedErrorSystem) DisableProfiling() {
	s.profiler.Disable()
}

// GetMigrationHelper returns the migration helper
func (s *IntegratedErrorSystem) GetMigrationHelper() *MigrationHelper {
	return s.migrationHelper
}

// ProcessError processes an error through the complete error handling pipeline
func (s *IntegratedErrorSystem) ProcessError(err error) error {
	return s.profiler.ProfileOperation("process_error", func() error {
		if err == nil {
			return nil
		}

		// Convert to OutputError if needed
		var outputErr OutputError
		if oe, ok := err.(OutputError); ok {
			outputErr = oe
		} else {
			outputErr = WrapError(err)
		}

		// Report the error (if reporter is available)
		if s.reporter != nil {
			// Note: In a real implementation, this would call the reporter
			// s.reporter.Report(outputErr)
		}

		// Try recovery first
		if s.recoveryHandler != nil && s.recoveryHandler.CanRecover(outputErr) {
			if recoveryErr := s.recoveryHandler.Recover(outputErr); recoveryErr == nil {
				// Recovery successful
				return nil
			}
		}

		// Handle the error through the error handler
		return s.handler.HandleError(outputErr)
	})
}

// ProcessErrors processes multiple errors as a batch
func (s *IntegratedErrorSystem) ProcessErrors(errors []error) []error {
	var results []error

	for _, err := range errors {
		if processedErr := s.ProcessError(err); processedErr != nil {
			results = append(results, processedErr)
		}
	}

	return results
}

// GetErrorSummary returns a summary of all errors
func (s *IntegratedErrorSystem) GetErrorSummary() ErrorSummary {
	return s.handler.GetSummary()
}

// GetPerformanceReport returns a performance report
func (s *IntegratedErrorSystem) GetPerformanceReport() PerformanceReport {
	return s.profiler.PerformanceReport()
}

// Clear clears all collected errors and metrics
func (s *IntegratedErrorSystem) Clear() {
	s.handler.Clear()
	if s.reporter != nil {
		// Note: In a real implementation, this would call the reporter's Clear method
		// s.reporter.Clear()
	}
	s.profiler.Clear()
}

// SystemHealthCheck performs a comprehensive health check of the error system
func (s *IntegratedErrorSystem) SystemHealthCheck() SystemHealthReport {
	report := SystemHealthReport{
		Timestamp: time.Now(),
		Healthy:   true,
		Issues:    make([]string, 0),
	}

	// Check error handler
	if s.handler == nil {
		report.Healthy = false
		report.Issues = append(report.Issues, "Error handler is nil")
	}

	// Check error reporter
	if s.reporter == nil {
		report.Healthy = false
		report.Issues = append(report.Issues, "Error reporter is nil")
	}

	// Check recovery handler
	if s.recoveryHandler == nil {
		report.Issues = append(report.Issues, "Recovery handler is nil (optional)")
	}

	// Check profiler
	if s.profiler == nil {
		report.Issues = append(report.Issues, "Performance profiler is nil")
	}

	// Test basic error processing
	testErr := NewError(ErrInvalidFormat, "health check test")
	if err := s.ProcessError(testErr); err != testErr {
		// In strict mode, the error should be returned as-is
		if s.handler.Mode() == ErrorModeStrict {
			// This is expected behavior
		} else {
			report.Issues = append(report.Issues, "Basic error processing failed")
		}
	}

	return report
}

// SystemHealthReport contains the health status of the error system
type SystemHealthReport struct {
	Timestamp time.Time
	Healthy   bool
	Issues    []string
}

// String returns a human-readable health report
func (r SystemHealthReport) String() string {
	status := "HEALTHY"
	if !r.Healthy {
		status = "UNHEALTHY"
	}

	result := fmt.Sprintf("System Health: %s at %v\n", status, r.Timestamp.Format(time.RFC3339))
	if len(r.Issues) > 0 {
		result += "Issues:\n"
		for _, issue := range r.Issues {
			result += fmt.Sprintf("  - %s\n", issue)
		}
	}
	return result
}

// LegacyMigrationHelper provides utilities for migrating from log.Fatal
type LegacyMigrationHelper struct {
	logFatalCount int
	panicCount    int
	errorAnalysis map[string]int
}

// NewLegacyMigrationHelper creates a new legacy migration helper
func NewLegacyMigrationHelper() *LegacyMigrationHelper {
	return &LegacyMigrationHelper{
		errorAnalysis: make(map[string]int),
	}
}

// AnalyzeCode analyzes code for migration opportunities
func (h *LegacyMigrationHelper) AnalyzeCode(code string) {
	h.logFatalCount = strings.Count(code, "log.Fatal")
	h.panicCount = strings.Count(code, "panic(")
	
	// Analyze different error patterns
	h.errorAnalysis["log.Fatal"] = strings.Count(code, "log.Fatal")
	h.errorAnalysis["log.Fatalf"] = strings.Count(code, "log.Fatalf")
	h.errorAnalysis["log.Fatalln"] = strings.Count(code, "log.Fatalln")
	h.errorAnalysis["panic"] = strings.Count(code, "panic(")
	h.errorAnalysis["os.Exit"] = strings.Count(code, "os.Exit")
}

// GetMigrationPlan returns a migration plan based on the analysis
func (h *LegacyMigrationHelper) GetMigrationPlan() []MigrationStep {
	var steps []MigrationStep
	
	// Step 1: Replace log.Fatal with error returns
	if h.logFatalCount > 0 {
		steps = append(steps, MigrationStep{
			Name:        "Replace log.Fatal calls",
			Description: fmt.Sprintf("Replace %d log.Fatal calls with error returns", h.logFatalCount),
			Check: func() (bool, string) {
				// This would need to be implemented to check actual code
				return false, "Manual code review required"
			},
			Fix: func() error {
				// This would need to be implemented with actual code transformation
				return fmt.Errorf("manual fix required")
			},
		})
	}
	
	// Step 2: Add error handling
	steps = append(steps, MigrationStep{
		Name:        "Add error handling",
		Description: "Add proper error handling to all functions",
		Check: func() (bool, string) {
			return false, "Error handling patterns need to be implemented"
		},
		Fix: func() error {
			return fmt.Errorf("requires code changes")
		},
	})
	
	// Step 3: Add validation
	steps = append(steps, MigrationStep{
		Name:        "Add validation",
		Description: "Add input validation before operations",
		Check: func() (bool, string) {
			return false, "Validation needs to be added"
		},
	})
	
	return steps
}

// GetMigrationReport returns a detailed migration report
func (h *LegacyMigrationHelper) GetMigrationReport() string {
	report := "Legacy Migration Analysis Report:\n"
	report += fmt.Sprintf("\nError Pattern Analysis:\n")
	
	for pattern, count := range h.errorAnalysis {
		if count > 0 {
			report += fmt.Sprintf("  %s: %d occurrences\n", pattern, count)
		}
	}
	
	report += fmt.Sprintf("\nRecommended Migration Steps:\n")
	steps := h.GetMigrationPlan()
	for i, step := range steps {
		report += fmt.Sprintf("  %d. %s - %s\n", i+1, step.Name, step.Description)
	}
	
	return report
}

// CustomLegacyErrorHandler provides backward compatibility with custom log output
type CustomLegacyErrorHandler struct {
	logOutput func(v ...interface{})
}

// NewCustomLegacyErrorHandler creates a new custom legacy error handler
func NewCustomLegacyErrorHandler() *CustomLegacyErrorHandler {
	return &CustomLegacyErrorHandler{
		logOutput: func(v ...interface{}) {
			log.Fatal(v...)
		},
	}
}

// HandleError handles errors using log.Fatal for backward compatibility
func (h *CustomLegacyErrorHandler) HandleError(err error) error {
	if err != nil {
		h.logOutput(err)
	}
	return nil
}

// SetMode does nothing in legacy mode
func (h *CustomLegacyErrorHandler) SetMode(mode ErrorMode) {
	// Legacy mode doesn't support different modes
}

// WithCustomLogOutput allows customizing the log output for testing
func (h *CustomLegacyErrorHandler) WithCustomLogOutput(logFn func(v ...interface{})) *CustomLegacyErrorHandler {
	h.logOutput = logFn
	return h
}

// CodeTransformer provides utilities for transforming code patterns
type CodeTransformer struct {
	transformations map[string]string
}

// NewCodeTransformer creates a new code transformer
func NewCodeTransformer() *CodeTransformer {
	return &CodeTransformer{
		transformations: make(map[string]string),
	}
}

// AddTransformation adds a code transformation rule
func (t *CodeTransformer) AddTransformation(from, to string) {
	t.transformations[from] = to
}

// GetStandardTransformations returns standard migration transformations
func (t *CodeTransformer) GetStandardTransformations() map[string]string {
	return map[string]string{
		"log.Fatal(err)":          "return err",
		"log.Fatalf(msg, args)":   "return fmt.Errorf(msg, args)",
		"log.Fatalln(msg)":       "return fmt.Errorf(msg)",
		"if err != nil { panic }":  "if err != nil { return err }",
		"panic(err)":             "return err",
	}
}

// TransformCode applies transformations to code (simplified example)
func (t *CodeTransformer) TransformCode(code string) string {
	result := code
	for from, to := range t.transformations {
		result = strings.ReplaceAll(result, from, to)
	}
	return result
}

// MigrationGuide provides step-by-step migration guidance
type MigrationGuide struct {
	steps []MigrationGuideStep
}

// MigrationGuideStep represents a single step in the migration guide
type MigrationGuideStep struct {
	Title       string
	Description string
	CodeBefore  string
	CodeAfter   string
	Notes       []string
}

// NewMigrationGuide creates a new migration guide
func NewMigrationGuide() *MigrationGuide {
	return &MigrationGuide{
		steps: []MigrationGuideStep{
			{
				Title:       "Replace log.Fatal with error returns",
				Description: "Convert functions that use log.Fatal to return errors instead",
				CodeBefore:  "func processData(data []byte) {\n    if len(data) == 0 {\n        log.Fatal(\"data cannot be empty\")\n    }\n    // process data...\n}",
				CodeAfter:   "func processData(data []byte) error {\n    if len(data) == 0 {\n        return fmt.Errorf(\"data cannot be empty\")\n    }\n    // process data...\n    return nil\n}",
				Notes: []string{
					"Change function signature to return error",
					"Replace log.Fatal calls with return statements",
					"Update all callers to handle returned errors",
				},
			},
		},
	}
}

// GetStep returns a specific migration step
func (g *MigrationGuide) GetStep(index int) *MigrationGuideStep {
	if index < 0 || index >= len(g.steps) {
		return nil
	}
	return &g.steps[index]
}

// GetAllSteps returns all migration steps
func (g *MigrationGuide) GetAllSteps() []MigrationGuideStep {
	return g.steps
}

// GenerateGuide generates a complete migration guide as a string
func (g *MigrationGuide) GenerateGuide() string {
	guide := "# Migration Guide: From log.Fatal to Structured Error Handling\n\n"
	
	for i, step := range g.steps {
		guide += fmt.Sprintf("## Step %d: %s\n\n", i+1, step.Title)
		guide += fmt.Sprintf("%s\n\n", step.Description)
		
		guide += "### Before:\n```go\n" + step.CodeBefore + "\n```\n\n"
		guide += "### After:\n```go\n" + step.CodeAfter + "\n```\n\n"
		
		if len(step.Notes) > 0 {
			guide += "### Notes:\n"
			for _, note := range step.Notes {
				guide += fmt.Sprintf("- %s\n", note)
			}
			guide += "\n"
		}
	}
	
	return guide
}