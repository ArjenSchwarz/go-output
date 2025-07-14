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
	SeverityFatal ErrorSeverity = iota
	SeverityError
	SeverityWarning
	SeverityInfo
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
