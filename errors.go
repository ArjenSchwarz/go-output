package format

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ErrorCode represents a structured error code for the go-output library
type ErrorCode string

// Error codes organized by category (1xxx, 2xxx, 3xxx, 4xxx)
const (
	// Configuration errors (1xxx)
	ErrInvalidFormat      ErrorCode = "OUT-1001"
	ErrMissingRequired    ErrorCode = "OUT-1002"
	ErrIncompatibleConfig ErrorCode = "OUT-1003"
	ErrInvalidFilePath    ErrorCode = "OUT-1004"
	ErrInvalidS3Config    ErrorCode = "OUT-1005"

	// Validation errors (2xxx)
	ErrMissingColumn       ErrorCode = "OUT-2001"
	ErrInvalidDataType     ErrorCode = "OUT-2002"
	ErrConstraintViolation ErrorCode = "OUT-2003"
	ErrEmptyDataset        ErrorCode = "OUT-2004"
	ErrMalformedData       ErrorCode = "OUT-2005"

	// Processing errors (3xxx)
	ErrFileWrite        ErrorCode = "OUT-3001"
	ErrS3Upload         ErrorCode = "OUT-3002"
	ErrTemplateRender   ErrorCode = "OUT-3003"
	ErrMemoryExhausted  ErrorCode = "OUT-3004"
	ErrFormatGeneration ErrorCode = "OUT-3005"

	// Runtime errors (4xxx)
	ErrNetworkTimeout     ErrorCode = "OUT-4001"
	ErrPermissionDenied   ErrorCode = "OUT-4002"
	ErrResourceNotFound   ErrorCode = "OUT-4003"
	ErrServiceUnavailable ErrorCode = "OUT-4004"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity int

const (
	SeverityInfo ErrorSeverity = iota
	SeverityWarning
	SeverityError
	SeverityFatal
)

// String returns the string representation of ErrorSeverity
func (s ErrorSeverity) String() string {
	switch s {
	case SeverityFatal:
		return "fatal"
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	case SeverityInfo:
		return "info"
	default:
		return "unknown"
	}
}

// ErrorContext provides detailed context information about where and why an error occurred
type ErrorContext struct {
	Operation string                 `json:"operation,omitempty"` // The operation being performed
	Field     string                 `json:"field,omitempty"`     // The field that caused the error
	Value     interface{}            `json:"value,omitempty"`     // The problematic value
	Index     int                    `json:"index,omitempty"`     // Index in array/slice if applicable
	Metadata  map[string]interface{} `json:"metadata,omitempty"`  // Additional context information
}

// OutputError is the base error interface for all go-output errors
type OutputError interface {
	error
	Code() ErrorCode
	Severity() ErrorSeverity
	Context() ErrorContext
	Suggestions() []string
	Wrap(error) OutputError
}

// ValidationError represents errors that occur during validation
type ValidationError interface {
	OutputError
	Violations() []Violation
	IsComposite() bool
}

// ProcessingError represents errors that occur during processing/runtime
type ProcessingError interface {
	OutputError
	Retryable() bool
	PartialResult() interface{}
}

// Violation represents a single validation violation
type Violation struct {
	Field      string      `json:"field"`      // Field name that violated constraint
	Value      interface{} `json:"value"`      // The violating value
	Constraint string      `json:"constraint"` // Name of the violated constraint
	Message    string      `json:"message"`    // Human-readable violation message
}

// baseError provides the common implementation for all OutputError types
type baseError struct {
	code        ErrorCode
	severity    ErrorSeverity
	message     string
	context     ErrorContext
	suggestions []string
	cause       error
}

// Error implements the error interface
func (e *baseError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "[%s] %s", e.code, e.message)

	if e.context.Field != "" {
		fmt.Fprintf(&b, " (field: %s)", e.context.Field)
	}

	if e.context.Operation != "" {
		fmt.Fprintf(&b, " (operation: %s)", e.context.Operation)
	}

	if len(e.suggestions) > 0 {
		fmt.Fprintf(&b, "\nSuggestions:")
		for _, s := range e.suggestions {
			fmt.Fprintf(&b, "\n  - %s", s)
		}
	}

	if e.cause != nil {
		fmt.Fprintf(&b, "\nCaused by: %v", e.cause)
	}

	return b.String()
}

// Code returns the error code
func (e *baseError) Code() ErrorCode {
	return e.code
}

// Severity returns the error severity
func (e *baseError) Severity() ErrorSeverity {
	return e.severity
}

// Context returns the error context
func (e *baseError) Context() ErrorContext {
	return e.context
}

// Suggestions returns suggested fixes
func (e *baseError) Suggestions() []string {
	return e.suggestions
}

// Wrap wraps another error as the cause
func (e *baseError) Wrap(err error) OutputError {
	e.cause = err
	return e
}

// MarshalJSON implements JSON marshaling for structured error output
func (e *baseError) MarshalJSON() ([]byte, error) {
	causeStr := ""
	if e.cause != nil {
		causeStr = e.cause.Error()
	}

	return json.Marshal(struct {
		Code        ErrorCode    `json:"code"`
		Severity    string       `json:"severity"`
		Message     string       `json:"message"`
		Context     ErrorContext `json:"context,omitempty"`
		Suggestions []string     `json:"suggestions,omitempty"`
		Cause       string       `json:"cause,omitempty"`
	}{
		Code:        e.code,
		Severity:    e.severity.String(),
		Message:     e.message,
		Context:     e.context,
		Suggestions: e.suggestions,
		Cause:       causeStr,
	})
}

// validationError implements ValidationError interface
type validationError struct {
	baseError
	violations []Violation
}

// Violations returns the validation violations
func (v *validationError) Violations() []Violation {
	return v.violations
}

// IsComposite returns true if this error contains multiple violations
func (v *validationError) IsComposite() bool {
	return len(v.violations) > 1
}

// Error returns a formatted error message including violations
func (v *validationError) Error() string {
	var b strings.Builder
	b.WriteString(v.baseError.Error())

	if len(v.violations) > 0 {
		b.WriteString("\nValidation violations:")
		for _, violation := range v.violations {
			fmt.Fprintf(&b, "\n  - %s: %s (value: %v)",
				violation.Field, violation.Message, violation.Value)
		}
	}

	return b.String()
}

// processingError implements ProcessingError interface
type processingError struct {
	baseError
	retryable     bool
	partialResult interface{}
}

// Retryable returns true if the error is retryable
func (p *processingError) Retryable() bool {
	return p.retryable
}

// PartialResult returns any partial result that was generated before the error
func (p *processingError) PartialResult() interface{} {
	return p.partialResult
}

// NewOutputError creates a new base OutputError
func NewOutputError(code ErrorCode, severity ErrorSeverity, message string) OutputError {
	return &baseError{
		code:        code,
		severity:    severity,
		message:     message,
		context:     ErrorContext{},
		suggestions: []string{},
	}
}

// NewConfigError creates a new configuration error
func NewConfigError(code ErrorCode, message string) OutputError {
	return NewOutputError(code, SeverityError, message)
}

// NewValidationError creates a new validation error
func NewValidationError(code ErrorCode, message string) ValidationError {
	return &validationError{
		baseError: baseError{
			code:        code,
			severity:    SeverityError,
			message:     message,
			context:     ErrorContext{},
			suggestions: []string{},
		},
		violations: []Violation{},
	}
}

// NewProcessingError creates a new processing error
func NewProcessingError(code ErrorCode, message string, retryable bool) ProcessingError {
	return &processingError{
		baseError: baseError{
			code:        code,
			severity:    SeverityError,
			message:     message,
			context:     ErrorContext{},
			suggestions: []string{},
		},
		retryable: retryable,
	}
}

// WrapError wraps a standard error as an OutputError
func WrapError(err error) OutputError {
	if err == nil {
		return nil
	}

	// If it's already an OutputError, return as-is
	if outputErr, ok := err.(OutputError); ok {
		return outputErr
	}

	// Wrap as a generic processing error
	return &baseError{
		code:     ErrFormatGeneration,
		severity: SeverityError,
		message:  "unexpected error occurred",
		cause:    err,
	}
}

// ErrorBuilder provides a fluent interface for building errors
type ErrorBuilder struct {
	err *baseError
}

// NewErrorBuilder creates a new error builder
func NewErrorBuilder(code ErrorCode, message string) *ErrorBuilder {
	return &ErrorBuilder{
		err: &baseError{
			code:        code,
			severity:    SeverityError,
			message:     message,
			context:     ErrorContext{},
			suggestions: []string{},
		},
	}
}

// WithSeverity sets the error severity
func (b *ErrorBuilder) WithSeverity(severity ErrorSeverity) *ErrorBuilder {
	b.err.severity = severity
	return b
}

// WithContext sets the error context
func (b *ErrorBuilder) WithContext(context ErrorContext) *ErrorBuilder {
	b.err.context = context
	return b
}

// WithField sets the field in the error context
func (b *ErrorBuilder) WithField(field string) *ErrorBuilder {
	b.err.context.Field = field
	return b
}

// WithOperation sets the operation in the error context
func (b *ErrorBuilder) WithOperation(operation string) *ErrorBuilder {
	b.err.context.Operation = operation
	return b
}

// WithValue sets the value in the error context
func (b *ErrorBuilder) WithValue(value interface{}) *ErrorBuilder {
	b.err.context.Value = value
	return b
}

// WithSuggestions adds suggestions to the error
func (b *ErrorBuilder) WithSuggestions(suggestions ...string) *ErrorBuilder {
	b.err.suggestions = append(b.err.suggestions, suggestions...)
	return b
}

// WithCause sets the underlying cause
func (b *ErrorBuilder) WithCause(cause error) *ErrorBuilder {
	b.err.cause = cause
	return b
}

// Build returns the constructed error
func (b *ErrorBuilder) Build() OutputError {
	return b.err
}

// BuildValidation returns the constructed error as a ValidationError
func (b *ErrorBuilder) BuildValidation() ValidationError {
	return &validationError{
		baseError:  *b.err,
		violations: []Violation{},
	}
}

// BuildProcessing returns the constructed error as a ProcessingError
func (b *ErrorBuilder) BuildProcessing(retryable bool) ProcessingError {
	return &processingError{
		baseError: *b.err,
		retryable: retryable,
	}
}

// ErrorMode represents different error handling modes
type ErrorMode int

const (
	// ErrorModeStrict fails fast on any error with severity >= Error
	ErrorModeStrict ErrorMode = iota
	// ErrorModeLenient continues processing on recoverable errors and collects all issues
	ErrorModeLenient
	// ErrorModeInteractive prompts user for error resolution
	ErrorModeInteractive
)

// String returns the string representation of ErrorMode
func (m ErrorMode) String() string {
	switch m {
	case ErrorModeStrict:
		return "strict"
	case ErrorModeLenient:
		return "lenient"
	case ErrorModeInteractive:
		return "interactive"
	default:
		return "unknown"
	}
}

// ErrorHandler defines the interface for handling errors in different modes
type ErrorHandler interface {
	// HandleError processes an error according to the configured mode
	HandleError(err error) error
	// SetMode changes the error handling mode
	SetMode(mode ErrorMode)
	// GetMode returns the current error handling mode
	GetMode() ErrorMode
	// GetCollectedErrors returns all collected errors (for lenient mode)
	GetCollectedErrors() []error
	// Clear clears all collected errors
	Clear()
}

// ErrorSummary provides aggregated information about collected errors
type ErrorSummary struct {
	TotalErrors   int                   `json:"total_errors"`   // Total number of errors
	ByCategory    map[ErrorCode]int     `json:"by_category"`    // Errors grouped by error code
	BySeverity    map[ErrorSeverity]int `json:"by_severity"`    // Errors grouped by severity
	Suggestions   []string              `json:"suggestions"`    // Aggregated suggestions
	FixableErrors int                   `json:"fixable_errors"` // Number of automatically fixable errors
}

// DefaultErrorHandler provides the default implementation of ErrorHandler
type DefaultErrorHandler struct {
	mode            ErrorMode
	collectedErrors []error
	warningHandler  func(error)
}

// NewDefaultErrorHandler creates a new DefaultErrorHandler with strict mode
func NewDefaultErrorHandler() *DefaultErrorHandler {
	return &DefaultErrorHandler{
		mode:            ErrorModeStrict,
		collectedErrors: make([]error, 0),
	}
}

// NewErrorHandlerWithMode creates a new DefaultErrorHandler with the specified mode
func NewErrorHandlerWithMode(mode ErrorMode) *DefaultErrorHandler {
	return &DefaultErrorHandler{
		mode:            mode,
		collectedErrors: make([]error, 0),
	}
}

// HandleError processes an error according to the configured mode
func (h *DefaultErrorHandler) HandleError(err error) error {
	if err == nil {
		return nil
	}

	// Convert to OutputError if needed
	outputErr, ok := err.(OutputError)
	if !ok {
		outputErr = WrapError(err)
	}

	switch h.mode {
	case ErrorModeStrict:
		return h.handleStrict(outputErr)
	case ErrorModeLenient:
		return h.handleLenient(outputErr)
	case ErrorModeInteractive:
		return h.handleInteractive(outputErr)
	default:
		return h.handleStrict(outputErr)
	}
}

// handleStrict implements strict mode error handling
func (h *DefaultErrorHandler) handleStrict(err OutputError) error {
	// Handle warnings through warning handler if configured
	if err.Severity() == SeverityWarning && h.warningHandler != nil {
		h.warningHandler(err)
	}

	// In strict mode, fail fast on errors and fatal issues
	if err.Severity() >= SeverityError {
		return err
	}

	// Info and warnings don't cause failures in strict mode
	return nil
}

// handleLenient implements lenient mode error handling
func (h *DefaultErrorHandler) handleLenient(err OutputError) error {
	// Collect all errors for batch reporting
	h.collectedErrors = append(h.collectedErrors, err)

	// Only fail immediately on fatal errors
	if err.Severity() == SeverityFatal {
		return err
	}

	// Continue processing for all other error types
	return nil
}

// handleInteractive implements interactive mode error handling
func (h *DefaultErrorHandler) handleInteractive(err OutputError) error {
	// For now, interactive mode behaves like strict mode
	// In a full implementation, this would prompt the user for resolution
	// TODO: Implement user prompting and guided error resolution
	return h.handleStrict(err)
}

// SetMode changes the error handling mode
func (h *DefaultErrorHandler) SetMode(mode ErrorMode) {
	h.mode = mode
}

// GetMode returns the current error handling mode
func (h *DefaultErrorHandler) GetMode() ErrorMode {
	return h.mode
}

// GetCollectedErrors returns all collected errors
func (h *DefaultErrorHandler) GetCollectedErrors() []error {
	return h.collectedErrors
}

// Clear clears all collected errors
func (h *DefaultErrorHandler) Clear() {
	h.collectedErrors = make([]error, 0)
}

// SetWarningHandler sets a custom warning handler function
func (h *DefaultErrorHandler) SetWarningHandler(handler func(error)) {
	h.warningHandler = handler
}

// Summary generates an ErrorSummary from collected errors
func (h *DefaultErrorHandler) Summary() ErrorSummary {
	summary := ErrorSummary{
		TotalErrors:   len(h.collectedErrors),
		ByCategory:    make(map[ErrorCode]int),
		BySeverity:    make(map[ErrorSeverity]int),
		Suggestions:   make([]string, 0),
		FixableErrors: 0,
	}

	suggestionSet := make(map[string]bool) // To avoid duplicate suggestions

	for _, err := range h.collectedErrors {
		if outputErr, ok := err.(OutputError); ok {
			// Count by category
			summary.ByCategory[outputErr.Code()]++

			// Count by severity
			summary.BySeverity[outputErr.Severity()]++

			// Collect unique suggestions
			for _, suggestion := range outputErr.Suggestions() {
				if !suggestionSet[suggestion] {
					summary.Suggestions = append(summary.Suggestions, suggestion)
					suggestionSet[suggestion] = true
				}
			}

			// Count fixable errors (warnings and info are considered fixable)
			if outputErr.Severity() <= SeverityWarning {
				summary.FixableErrors++
			}
		}
	}

	return summary
}

// HasErrors returns true if there are any collected errors
func (h *DefaultErrorHandler) HasErrors() bool {
	return len(h.collectedErrors) > 0
}

// HasErrorsWithSeverity returns true if there are errors with the specified severity or higher
func (h *DefaultErrorHandler) HasErrorsWithSeverity(severity ErrorSeverity) bool {
	for _, err := range h.collectedErrors {
		if outputErr, ok := err.(OutputError); ok {
			if outputErr.Severity() >= severity {
				return true
			}
		}
	}
	return false
}

// ValidationErrorBuilder provides a fluent interface for building validation errors
type ValidationErrorBuilder struct {
	err        *validationError
	violations []Violation
}

// NewValidationErrorBuilder creates a new validation error builder
func NewValidationErrorBuilder(code ErrorCode, message string) *ValidationErrorBuilder {
	return &ValidationErrorBuilder{
		err: &validationError{
			baseError: baseError{
				code:        code,
				severity:    SeverityError,
				message:     message,
				context:     ErrorContext{},
				suggestions: []string{},
			},
			violations: []Violation{},
		},
	}
}

// WithViolation adds a violation to the validation error
func (b *ValidationErrorBuilder) WithViolation(field, constraint, message string, value interface{}) *ValidationErrorBuilder {
	b.violations = append(b.violations, Violation{
		Field:      field,
		Value:      value,
		Constraint: constraint,
		Message:    message,
	})
	return b
}

// WithViolations adds multiple violations
func (b *ValidationErrorBuilder) WithViolations(violations ...Violation) *ValidationErrorBuilder {
	b.violations = append(b.violations, violations...)
	return b
}

// WithSeverity sets the error severity
func (b *ValidationErrorBuilder) WithSeverity(severity ErrorSeverity) *ValidationErrorBuilder {
	b.err.severity = severity
	return b
}

// WithSuggestions adds suggestions to the error
func (b *ValidationErrorBuilder) WithSuggestions(suggestions ...string) *ValidationErrorBuilder {
	b.err.suggestions = append(b.err.suggestions, suggestions...)
	return b
}

// Build returns the constructed validation error
func (b *ValidationErrorBuilder) Build() ValidationError {
	b.err.violations = b.violations
	return b.err
}
