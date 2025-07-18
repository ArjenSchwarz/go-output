package format

import (
	"fmt"
	"sync"
	"time"
)

// ErrorInjector provides utilities for injecting errors during testing
type ErrorInjector struct {
	mu                sync.RWMutex
	operationErrors   map[string]error           // Errors to inject for specific operations
	validationErrors  map[string]ValidationError // Validation errors to inject
	processingErrors  map[string]ProcessingError // Processing errors to inject
	errorCounts       map[string]int             // Count of how many times each error was injected
	enabled           bool                       // Whether error injection is enabled
	triggerConditions map[string]func() bool     // Conditions that must be met to trigger errors
}

// NewErrorInjector creates a new ErrorInjector for testing
func NewErrorInjector() *ErrorInjector {
	return &ErrorInjector{
		operationErrors:   make(map[string]error),
		validationErrors:  make(map[string]ValidationError),
		processingErrors:  make(map[string]ProcessingError),
		errorCounts:       make(map[string]int),
		enabled:           true,
		triggerConditions: make(map[string]func() bool),
	}
}

// Enable enables error injection
func (e *ErrorInjector) Enable() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = true
}

// Disable disables error injection
func (e *ErrorInjector) Disable() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = false
}

// IsEnabled returns whether error injection is enabled
func (e *ErrorInjector) IsEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.enabled
}

// InjectError injects a generic error for a specific operation
func (e *ErrorInjector) InjectError(operation string, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.operationErrors[operation] = err
}

// InjectValidationError injects a validation error for a specific operation
func (e *ErrorInjector) InjectValidationError(operation string, err ValidationError) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.validationErrors[operation] = err
}

// InjectProcessingError injects a processing error for a specific operation
func (e *ErrorInjector) InjectProcessingError(operation string, err ProcessingError) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.processingErrors[operation] = err
}

// InjectErrorWithCondition injects an error that only triggers when the condition is met
func (e *ErrorInjector) InjectErrorWithCondition(operation string, err error, condition func() bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.operationErrors[operation] = err
	e.triggerConditions[operation] = condition
}

// GetError retrieves an injected error for an operation
func (e *ErrorInjector) GetError(operation string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.enabled {
		return nil
	}

	// Check if there's a trigger condition
	if condition, exists := e.triggerConditions[operation]; exists {
		if !condition() {
			return nil
		}
	}

	// Increment error count
	e.errorCounts[operation]++

	// Check for specific error types first
	if err, exists := e.validationErrors[operation]; exists {
		return err
	}

	if err, exists := e.processingErrors[operation]; exists {
		return err
	}

	// Return generic error
	if err, exists := e.operationErrors[operation]; exists {
		return err
	}

	return nil
}

// GetErrorCount returns how many times an error was injected for an operation
func (e *ErrorInjector) GetErrorCount(operation string) int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.errorCounts[operation]
}

// Clear clears all injected errors
func (e *ErrorInjector) Clear() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.operationErrors = make(map[string]error)
	e.validationErrors = make(map[string]ValidationError)
	e.processingErrors = make(map[string]ProcessingError)
	e.errorCounts = make(map[string]int)
	e.triggerConditions = make(map[string]func() bool)
}

// GetAllOperations returns all operations that have injected errors
func (e *ErrorInjector) GetAllOperations() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	operations := make(map[string]bool)
	for op := range e.operationErrors {
		operations[op] = true
	}
	for op := range e.validationErrors {
		operations[op] = true
	}
	for op := range e.processingErrors {
		operations[op] = true
	}

	result := make([]string, 0, len(operations))
	for op := range operations {
		result = append(result, op)
	}
	return result
}

// CreateConfigurationError creates a configuration error for testing
func (e *ErrorInjector) CreateConfigurationError(code ErrorCode, message string) OutputError {
	return NewErrorBuilder(code, message).
		WithSeverity(SeverityError).
		WithOperation("test_configuration").
		WithSuggestions("this is a test error for configuration validation").
		Build()
}

// CreateValidationError creates a validation error for testing
func (e *ErrorInjector) CreateValidationError(code ErrorCode, message string, violations ...Violation) ValidationError {
	builder := NewValidationErrorBuilder(code, message).
		WithSeverity(SeverityError).
		WithSuggestions("this is a test validation error")

	for _, violation := range violations {
		builder.WithViolations(violation)
	}

	return builder.Build()
}

// CreateProcessingError creates a processing error for testing
func (e *ErrorInjector) CreateProcessingError(code ErrorCode, message string, retryable bool) ProcessingError {
	return NewErrorBuilder(code, message).
		WithSeverity(SeverityError).
		WithOperation("test_processing").
		WithSuggestions("this is a test processing error").
		BuildProcessing(retryable)
}

// MockValidator provides a configurable validator for testing scenarios
type MockValidator struct {
	name          string
	shouldFail    bool
	errorToReturn error
	validateFunc  func(any) error
	callCount     int
	lastSubject   any
	estimatedCost int
	isFailFast    bool
	mu            sync.RWMutex
}

// NewMockValidator creates a new MockValidator
func NewMockValidator(name string) *MockValidator {
	return &MockValidator{
		name:          name,
		shouldFail:    false,
		estimatedCost: 5, // Default moderate cost
		isFailFast:    false,
	}
}

// WithFailure configures the validator to fail with the specified error
func (m *MockValidator) WithFailure(err error) *MockValidator {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = true
	m.errorToReturn = err
	return m
}

// WithSuccess configures the validator to succeed
func (m *MockValidator) WithSuccess() *MockValidator {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = false
	m.errorToReturn = nil
	return m
}

// WithCustomValidation configures the validator to use a custom validation function
func (m *MockValidator) WithCustomValidation(validateFunc func(any) error) *MockValidator {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.validateFunc = validateFunc
	return m
}

// WithCost sets the estimated cost for performance testing
func (m *MockValidator) WithCost(cost int) *MockValidator {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.estimatedCost = cost
	return m
}

// WithFailFast sets whether this validator should run early
func (m *MockValidator) WithFailFast(failFast bool) *MockValidator {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isFailFast = failFast
	return m
}

// Validate implements the Validator interface
func (m *MockValidator) Validate(subject any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++
	m.lastSubject = subject

	// Use custom validation function if provided
	if m.validateFunc != nil {
		return m.validateFunc(subject)
	}

	// Return configured error if should fail
	if m.shouldFail {
		return m.errorToReturn
	}

	return nil
}

// Name implements the Validator interface
func (m *MockValidator) Name() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.name
}

// EstimatedCost implements the PerformanceAwareValidator interface
func (m *MockValidator) EstimatedCost() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.estimatedCost
}

// IsFailFast implements the PerformanceAwareValidator interface
func (m *MockValidator) IsFailFast() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isFailFast
}

// GetCallCount returns how many times Validate was called
func (m *MockValidator) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// GetLastSubject returns the last subject passed to Validate
func (m *MockValidator) GetLastSubject() any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastSubject
}

// Reset resets the validator's state
func (m *MockValidator) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount = 0
	m.lastSubject = nil
}

// MockErrorHandler provides a configurable error handler for testing
type MockErrorHandler struct {
	mode              ErrorMode
	collectedErrors   []error
	handleErrorFunc   func(error) error
	callCount         int
	lastError         error
	shouldReturnError bool
	errorToReturn     error
	mu                sync.RWMutex
}

// NewMockErrorHandler creates a new MockErrorHandler
func NewMockErrorHandler() *MockErrorHandler {
	return &MockErrorHandler{
		mode:            ErrorModeStrict,
		collectedErrors: make([]error, 0),
	}
}

// WithMode sets the error handling mode
func (m *MockErrorHandler) WithMode(mode ErrorMode) *MockErrorHandler {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mode = mode
	return m
}

// WithCustomHandler sets a custom error handling function
func (m *MockErrorHandler) WithCustomHandler(handler func(error) error) *MockErrorHandler {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handleErrorFunc = handler
	return m
}

// WithErrorReturn configures the handler to return a specific error
func (m *MockErrorHandler) WithErrorReturn(err error) *MockErrorHandler {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldReturnError = true
	m.errorToReturn = err
	return m
}

// HandleError implements the ErrorHandler interface
func (m *MockErrorHandler) HandleError(err error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++
	m.lastError = err
	m.collectedErrors = append(m.collectedErrors, err)

	// Use custom handler if provided
	if m.handleErrorFunc != nil {
		return m.handleErrorFunc(err)
	}

	// Return configured error if should return error
	if m.shouldReturnError {
		return m.errorToReturn
	}

	return nil
}

// SetMode implements the ErrorHandler interface
func (m *MockErrorHandler) SetMode(mode ErrorMode) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mode = mode
}

// GetMode implements the ErrorHandler interface
func (m *MockErrorHandler) GetMode() ErrorMode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mode
}

// GetCollectedErrors implements the ErrorHandler interface
func (m *MockErrorHandler) GetCollectedErrors() []error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]error, len(m.collectedErrors))
	copy(result, m.collectedErrors)
	return result
}

// Clear implements the ErrorHandler interface
func (m *MockErrorHandler) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.collectedErrors = make([]error, 0)
}

// GetCallCount returns how many times HandleError was called
func (m *MockErrorHandler) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// GetLastError returns the last error passed to HandleError
func (m *MockErrorHandler) GetLastError() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastError
}

// Reset resets the handler's state
func (m *MockErrorHandler) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount = 0
	m.lastError = nil
	m.collectedErrors = make([]error, 0)
}

// MockRecoveryStrategy provides a configurable recovery strategy for testing
type MockRecoveryStrategy struct {
	name              string
	priority          int
	shouldApply       bool
	applicableForFunc func(OutputError) bool
	applyFunc         func(OutputError, any) (any, error)
	callCount         int
	lastError         OutputError
	lastContext       any
	mu                sync.RWMutex
}

// NewMockRecoveryStrategy creates a new MockRecoveryStrategy
func NewMockRecoveryStrategy(name string) *MockRecoveryStrategy {
	return &MockRecoveryStrategy{
		name:        name,
		priority:    10,
		shouldApply: true,
	}
}

// WithPriority sets the strategy priority
func (m *MockRecoveryStrategy) WithPriority(priority int) *MockRecoveryStrategy {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.priority = priority
	return m
}

// WithApplicableFor sets a custom function to determine if the strategy applies
func (m *MockRecoveryStrategy) WithApplicableFor(fn func(OutputError) bool) *MockRecoveryStrategy {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.applicableForFunc = fn
	return m
}

// WithApply sets a custom apply function
func (m *MockRecoveryStrategy) WithApply(fn func(OutputError, any) (any, error)) *MockRecoveryStrategy {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.applyFunc = fn
	return m
}

// WithSuccess configures the strategy to always succeed
func (m *MockRecoveryStrategy) WithSuccess() *MockRecoveryStrategy {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldApply = true
	m.applyFunc = func(err OutputError, context any) (any, error) {
		return "recovery_successful", nil
	}
	return m
}

// WithFailure configures the strategy to always fail
func (m *MockRecoveryStrategy) WithFailure(err error) *MockRecoveryStrategy {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldApply = true
	m.applyFunc = func(OutputError, any) (any, error) {
		return nil, err
	}
	return m
}

// Apply implements the RecoveryStrategy interface
func (m *MockRecoveryStrategy) Apply(err OutputError, context any) (any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++
	m.lastError = err
	m.lastContext = context

	if m.applyFunc != nil {
		return m.applyFunc(err, context)
	}

	if m.shouldApply {
		return "mock_recovery_result", nil
	}

	return nil, fmt.Errorf("mock recovery strategy failed")
}

// ApplicableFor implements the RecoveryStrategy interface
func (m *MockRecoveryStrategy) ApplicableFor(err OutputError) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.applicableForFunc != nil {
		return m.applicableForFunc(err)
	}

	return true // Default to applicable for all errors
}

// Name implements the RecoveryStrategy interface
func (m *MockRecoveryStrategy) Name() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.name
}

// Priority implements the RecoveryStrategy interface
func (m *MockRecoveryStrategy) Priority() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.priority
}

// GetCallCount returns how many times Apply was called
func (m *MockRecoveryStrategy) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// GetLastError returns the last error passed to Apply
func (m *MockRecoveryStrategy) GetLastError() OutputError {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastError
}

// GetLastContext returns the last context passed to Apply
func (m *MockRecoveryStrategy) GetLastContext() any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastContext
}

// Reset resets the strategy's state
func (m *MockRecoveryStrategy) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount = 0
	m.lastError = nil
	m.lastContext = nil
}

// ErrorScenario represents a complete error testing scenario
type ErrorScenario struct {
	Name        string
	Description string
	Setup       func(*ErrorInjector)
	Validators  []Validator
	Handler     ErrorHandler
	Recovery    []RecoveryStrategy
	Expected    ExpectedResults
}

// ExpectedResults defines what results are expected from an error scenario
type ExpectedResults struct {
	ShouldFail          bool
	ExpectedErrorCode   ErrorCode
	ExpectedSeverity    ErrorSeverity
	ExpectedViolations  int
	ShouldRecover       bool
	RecoveryStrategy    string
	ValidationCallCount int
	HandlerCallCount    int
}

// ErrorScenarioRunner runs error testing scenarios
type ErrorScenarioRunner struct {
	scenarios []ErrorScenario
	results   map[string]ScenarioResult
}

// ScenarioResult contains the results of running a scenario
type ScenarioResult struct {
	Scenario     ErrorScenario
	Success      bool
	ActualError  error
	RecoveryUsed string
	CallCounts   map[string]int
	Duration     time.Duration
	ErrorMessage string
}

// NewErrorScenarioRunner creates a new scenario runner
func NewErrorScenarioRunner() *ErrorScenarioRunner {
	return &ErrorScenarioRunner{
		scenarios: make([]ErrorScenario, 0),
		results:   make(map[string]ScenarioResult),
	}
}

// AddScenario adds a scenario to the runner
func (r *ErrorScenarioRunner) AddScenario(scenario ErrorScenario) {
	r.scenarios = append(r.scenarios, scenario)
}

// RunScenario runs a single scenario
func (r *ErrorScenarioRunner) RunScenario(scenario ErrorScenario) ScenarioResult {
	start := time.Now()
	result := ScenarioResult{
		Scenario:   scenario,
		CallCounts: make(map[string]int),
	}

	// Setup error injection
	injector := NewErrorInjector()
	if scenario.Setup != nil {
		scenario.Setup(injector)
	}

	// Create test data
	output := &OutputArray{
		Settings: &OutputSettings{OutputFormat: "json"},
		Keys:     []string{"Name", "Value"},
		Contents: []OutputHolder{
			{Contents: map[string]any{"Name": "test", "Value": 123}},
		},
	}

	// Run validation if validators are provided
	if len(scenario.Validators) > 0 {
		for _, validator := range scenario.Validators {
			if err := validator.Validate(output); err != nil {
				result.ActualError = err
				break
			}
		}
	}

	// Test error handling if handler is provided
	if scenario.Handler != nil && result.ActualError != nil {
		handledErr := scenario.Handler.HandleError(result.ActualError)
		if handledErr != nil {
			result.ActualError = handledErr
		}
	}

	// Test recovery if strategies are provided
	if len(scenario.Recovery) > 0 && result.ActualError != nil {
		if outputErr, ok := result.ActualError.(OutputError); ok {
			for _, strategy := range scenario.Recovery {
				if strategy.ApplicableFor(outputErr) {
					if _, err := strategy.Apply(outputErr, output.Settings); err == nil {
						result.RecoveryUsed = strategy.Name()
						result.ActualError = nil // Recovery successful
						break
					}
				}
			}
		}
	}

	// Collect call counts from mock objects
	for _, validator := range scenario.Validators {
		if mockValidator, ok := validator.(*MockValidator); ok {
			result.CallCounts[mockValidator.Name()] = mockValidator.GetCallCount()
		}
	}

	if mockHandler, ok := scenario.Handler.(*MockErrorHandler); ok {
		result.CallCounts["handler"] = mockHandler.GetCallCount()
	}

	for _, strategy := range scenario.Recovery {
		if mockStrategy, ok := strategy.(*MockRecoveryStrategy); ok {
			result.CallCounts[strategy.Name()] = mockStrategy.GetCallCount()
		}
	}

	// Evaluate success
	result.Success = r.evaluateScenario(scenario, result)
	result.Duration = time.Since(start)

	r.results[scenario.Name] = result
	return result
}

// RunAllScenarios runs all configured scenarios
func (r *ErrorScenarioRunner) RunAllScenarios() map[string]ScenarioResult {
	for _, scenario := range r.scenarios {
		r.RunScenario(scenario)
	}
	return r.results
}

// evaluateScenario evaluates whether a scenario met its expected results
func (r *ErrorScenarioRunner) evaluateScenario(scenario ErrorScenario, result ScenarioResult) bool {
	expected := scenario.Expected

	// Check if failure expectation matches
	hasFailed := result.ActualError != nil
	if hasFailed != expected.ShouldFail {
		result.ErrorMessage = fmt.Sprintf("expected failure: %v, got failure: %v", expected.ShouldFail, hasFailed)
		return false
	}

	// If we expected failure, check error details
	if expected.ShouldFail && result.ActualError != nil {
		if outputErr, ok := result.ActualError.(OutputError); ok {
			if expected.ExpectedErrorCode != "" && outputErr.Code() != expected.ExpectedErrorCode {
				result.ErrorMessage = fmt.Sprintf("expected error code %s, got %s", expected.ExpectedErrorCode, outputErr.Code())
				return false
			}

			if outputErr.Severity() != expected.ExpectedSeverity {
				result.ErrorMessage = fmt.Sprintf("expected severity %v, got %v", expected.ExpectedSeverity, outputErr.Severity())
				return false
			}

			if validationErr, ok := outputErr.(ValidationError); ok {
				if len(validationErr.Violations()) != expected.ExpectedViolations {
					result.ErrorMessage = fmt.Sprintf("expected %d violations, got %d", expected.ExpectedViolations, len(validationErr.Violations()))
					return false
				}
			}
		}
	}

	// Check recovery expectations
	recoveryUsed := result.RecoveryUsed != ""
	if recoveryUsed != expected.ShouldRecover {
		result.ErrorMessage = fmt.Sprintf("expected recovery: %v, got recovery: %v", expected.ShouldRecover, recoveryUsed)
		return false
	}

	if expected.ShouldRecover && expected.RecoveryStrategy != "" && result.RecoveryUsed != expected.RecoveryStrategy {
		result.ErrorMessage = fmt.Sprintf("expected recovery strategy %s, got %s", expected.RecoveryStrategy, result.RecoveryUsed)
		return false
	}

	return true
}

// GetResults returns all scenario results
func (r *ErrorScenarioRunner) GetResults() map[string]ScenarioResult {
	return r.results
}

// GetSuccessfulScenarios returns scenarios that passed
func (r *ErrorScenarioRunner) GetSuccessfulScenarios() []string {
	var successful []string
	for name, result := range r.results {
		if result.Success {
			successful = append(successful, name)
		}
	}
	return successful
}

// GetFailedScenarios returns scenarios that failed
func (r *ErrorScenarioRunner) GetFailedScenarios() []string {
	var failed []string
	for name, result := range r.results {
		if !result.Success {
			failed = append(failed, name)
		}
	}
	return failed
}

// TestDataBuilder helps build test data for error scenarios
type TestDataBuilder struct {
	output *OutputArray
}

// NewTestDataBuilder creates a new test data builder
func NewTestDataBuilder() *TestDataBuilder {
	return &TestDataBuilder{
		output: &OutputArray{
			Settings: &OutputSettings{OutputFormat: "json"},
			Keys:     []string{},
			Contents: []OutputHolder{},
		},
	}
}

// WithKeys sets the column keys
func (b *TestDataBuilder) WithKeys(keys ...string) *TestDataBuilder {
	b.output.Keys = keys
	return b
}

// WithRow adds a data row
func (b *TestDataBuilder) WithRow(data map[string]any) *TestDataBuilder {
	b.output.Contents = append(b.output.Contents, OutputHolder{Contents: data})
	return b
}

// WithEmptyRow adds an empty row
func (b *TestDataBuilder) WithEmptyRow() *TestDataBuilder {
	b.output.Contents = append(b.output.Contents, OutputHolder{Contents: make(map[string]any)})
	return b
}

// WithNilRow adds a row with nil contents
func (b *TestDataBuilder) WithNilRow() *TestDataBuilder {
	b.output.Contents = append(b.output.Contents, OutputHolder{Contents: nil})
	return b
}

// WithFormat sets the output format
func (b *TestDataBuilder) WithFormat(format string) *TestDataBuilder {
	b.output.Settings.OutputFormat = format
	return b
}

// Build returns the constructed OutputArray
func (b *TestDataBuilder) Build() *OutputArray {
	return b.output
}

// ErrorTypeHelper provides utilities for creating different types of errors
type ErrorTypeHelper struct{}

// NewErrorTypeHelper creates a new error type helper
func NewErrorTypeHelper() *ErrorTypeHelper {
	return &ErrorTypeHelper{}
}

// CreateConfigurationErrors creates various configuration errors for testing
func (h *ErrorTypeHelper) CreateConfigurationErrors() []OutputError {
	return []OutputError{
		NewConfigError(ErrInvalidFormat, "invalid output format specified"),
		NewConfigError(ErrMissingRequired, "required configuration missing"),
		NewConfigError(ErrIncompatibleConfig, "incompatible configuration options"),
		NewConfigError(ErrInvalidFilePath, "invalid file path specified"),
		NewConfigError(ErrInvalidS3Config, "invalid S3 configuration"),
	}
}

// CreateValidationErrors creates various validation errors for testing
func (h *ErrorTypeHelper) CreateValidationErrors() []ValidationError {
	return []ValidationError{
		NewValidationErrorBuilder(ErrMissingColumn, "missing required columns").
			WithViolation("Name", "required", "column is required", nil).
			Build(),
		NewValidationErrorBuilder(ErrInvalidDataType, "invalid data types").
			WithViolation("Age", "type", "expected number, got string", "not_a_number").
			Build(),
		NewValidationErrorBuilder(ErrConstraintViolation, "constraint violations").
			WithViolation("Price", "positive", "value must be positive", -10).
			Build(),
		NewValidationErrorBuilder(ErrEmptyDataset, "empty dataset").Build(),
		NewValidationErrorBuilder(ErrMalformedData, "malformed data detected").
			WithViolation("Data", "well_formed", "data appears corrupted", "\x00invalid").
			Build(),
	}
}

// CreateProcessingErrors creates various processing errors for testing
func (h *ErrorTypeHelper) CreateProcessingErrors() []ProcessingError {
	return []ProcessingError{
		NewProcessingError(ErrFileWrite, "failed to write file", true),
		NewProcessingError(ErrS3Upload, "failed to upload to S3", true),
		NewProcessingError(ErrTemplateRender, "template rendering failed", false),
		NewProcessingError(ErrMemoryExhausted, "out of memory", false),
		NewProcessingError(ErrFormatGeneration, "format generation failed", false),
	}
}

// CreateRuntimeErrors creates various runtime errors for testing
func (h *ErrorTypeHelper) CreateRuntimeErrors() []OutputError {
	return []OutputError{
		NewErrorBuilder(ErrNetworkTimeout, "network timeout occurred").
			WithSeverity(SeverityError).
			BuildProcessing(true),
		NewErrorBuilder(ErrPermissionDenied, "permission denied").
			WithSeverity(SeverityError).
			BuildProcessing(false),
		NewErrorBuilder(ErrResourceNotFound, "resource not found").
			WithSeverity(SeverityError).
			BuildProcessing(false),
		NewErrorBuilder(ErrServiceUnavailable, "service unavailable").
			WithSeverity(SeverityError).
			BuildProcessing(true),
	}
}

// CreateErrorsWithSeverity creates errors with different severity levels
func (h *ErrorTypeHelper) CreateErrorsWithSeverity() map[ErrorSeverity][]OutputError {
	return map[ErrorSeverity][]OutputError{
		SeverityInfo: {
			NewErrorBuilder(ErrInvalidFormat, "info level error").
				WithSeverity(SeverityInfo).Build(),
		},
		SeverityWarning: {
			NewErrorBuilder(ErrMissingColumn, "warning level error").
				WithSeverity(SeverityWarning).Build(),
		},
		SeverityError: {
			NewErrorBuilder(ErrConstraintViolation, "error level error").
				WithSeverity(SeverityError).Build(),
		},
		SeverityFatal: {
			NewErrorBuilder(ErrMemoryExhausted, "fatal level error").
				WithSeverity(SeverityFatal).Build(),
		},
	}
}

// PerformanceTestHelper provides utilities for testing error handling performance
type PerformanceTestHelper struct {
	measurements map[string]time.Duration
	mu           sync.RWMutex
}

// NewPerformanceTestHelper creates a new performance test helper
func NewPerformanceTestHelper() *PerformanceTestHelper {
	return &PerformanceTestHelper{
		measurements: make(map[string]time.Duration),
	}
}

// MeasureOperation measures the time taken by an operation
func (h *PerformanceTestHelper) MeasureOperation(name string, operation func() error) error {
	start := time.Now()
	err := operation()
	duration := time.Since(start)

	h.mu.Lock()
	h.measurements[name] = duration
	h.mu.Unlock()

	return err
}

// GetMeasurement returns the measurement for an operation
func (h *PerformanceTestHelper) GetMeasurement(name string) time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.measurements[name]
}

// GetAllMeasurements returns all measurements
func (h *PerformanceTestHelper) GetAllMeasurements() map[string]time.Duration {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make(map[string]time.Duration)
	for k, v := range h.measurements {
		result[k] = v
	}
	return result
}

// VerifyPerformanceOverhead verifies that error handling overhead is within acceptable limits
func (h *PerformanceTestHelper) VerifyPerformanceOverhead(baselineOperation, errorOperation string, maxOverheadPercent float64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	baseline, baselineExists := h.measurements[baselineOperation]
	errorTime, errorExists := h.measurements[errorOperation]

	if !baselineExists || !errorExists {
		return false
	}

	if baseline == 0 {
		return errorTime == 0
	}

	overhead := float64(errorTime-baseline) / float64(baseline) * 100
	return overhead <= maxOverheadPercent
}

// Reset clears all measurements
func (h *PerformanceTestHelper) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.measurements = make(map[string]time.Duration)
}
