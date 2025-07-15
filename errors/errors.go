// Package errors provides a comprehensive error handling system for the go-output library.
// It replaces the usage of log.Fatal() with structured, recoverable errors that provide
// detailed context, suggestions, and support for various error handling modes.
package errors

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ErrorCode represents a unique identifier for different types of errors
type ErrorCode string

// Error codes for different categories of errors
const (
	// Configuration errors (1xxx)
	ErrInvalidFormat        ErrorCode = "OUT-1001"
	ErrMissingRequired      ErrorCode = "OUT-1002"
	ErrIncompatibleConfig   ErrorCode = "OUT-1003"
	ErrInvalidFilePath      ErrorCode = "OUT-1004"
	ErrInvalidConfiguration ErrorCode = "OUT-1005"

	// Validation errors (2xxx)
	ErrMissingColumn       ErrorCode = "OUT-2001"
	ErrInvalidDataType     ErrorCode = "OUT-2002"
	ErrConstraintViolation ErrorCode = "OUT-2003"
	ErrEmptyDataset        ErrorCode = "OUT-2004"
	ErrCompositeValidation ErrorCode = "OUT-2005"

	// Processing errors (3xxx)
	ErrFileWrite       ErrorCode = "OUT-3001"
	ErrS3Upload        ErrorCode = "OUT-3002"
	ErrTemplateRender  ErrorCode = "OUT-3003"
	ErrMemoryExhausted ErrorCode = "OUT-3004"
	ErrRetryable       ErrorCode = "OUT-3005"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity int

// Severity levels from most to least severe
const (
	SeverityFatal   ErrorSeverity = iota // Unrecoverable errors that must stop execution
	SeverityError                        // Recoverable errors that prevent normal operation
	SeverityWarning                      // Issues that don't prevent operation but should be noted
	SeverityInfo                         // Informational messages and suggestions
)

// String returns the string representation of ErrorSeverity
func (s ErrorSeverity) String() string {
	switch s {
	case SeverityFatal:
		return "Fatal"
	case SeverityError:
		return "Error"
	case SeverityWarning:
		return "Warning"
	case SeverityInfo:
		return "Info"
	default:
		return "Unknown"
	}
}

// ErrorContext provides additional context about where and why an error occurred
type ErrorContext struct {
	Operation string                 // The operation being performed when the error occurred
	Field     string                 // The specific field that caused the error
	Value     interface{}            // The value that caused the error
	Index     int                    // The index/position where the error occurred (for arrays/slices)
	Metadata  map[string]interface{} // Additional context-specific metadata
}

// OutputError is the base interface for all errors in the go-output library.
// It extends the standard error interface with additional context and functionality.
type OutputError interface {
	error
	Code() ErrorCode                        // Unique error code for programmatic handling
	Severity() ErrorSeverity                // Severity level of the error
	Context() ErrorContext                  // Additional context about the error
	Suggestions() []string                  // Suggested fixes or next steps
	Wrap(error) OutputError                 // Wrap another error as the cause
	WithContext(ErrorContext) OutputError   // Builder method to add context
	WithSuggestions(...string) OutputError  // Builder method to add suggestions
	WithSeverity(ErrorSeverity) OutputError // Builder method to set severity
}

// baseError is the concrete implementation of OutputError
type baseError struct {
	code        ErrorCode
	severity    ErrorSeverity
	message     string
	context     ErrorContext
	suggestions []string
	cause       error
}

// NewError creates a new OutputError with the specified code and message.
// Default severity is SeverityError.
func NewError(code ErrorCode, message string) OutputError {
	return &baseError{
		code:        code,
		severity:    SeverityError,
		message:     message,
		context:     ErrorContext{},
		suggestions: make([]string, 0),
	}
}

// Error returns the formatted error message including code, context, suggestions, and cause
func (e *baseError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "[%s] %s", e.code, e.message)

	// Add context information
	if e.context.Field != "" {
		fmt.Fprintf(&b, " (field: %s)", e.context.Field)
	}

	// Add suggestions
	if len(e.suggestions) > 0 {
		fmt.Fprintf(&b, "\nSuggestions:\n")
		for _, s := range e.suggestions {
			fmt.Fprintf(&b, "  - %s\n", s)
		}
	}

	// Add wrapped error
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

// Suggestions returns the list of suggested fixes
func (e *baseError) Suggestions() []string {
	return e.suggestions
}

// Wrap creates a new error that wraps the given error as the cause.
// This follows the builder pattern and returns a new instance.
func (e *baseError) Wrap(cause error) OutputError {
	newErr := e.clone()
	newErr.cause = cause
	return newErr
}

// WithContext creates a new error with the specified context.
// This follows the builder pattern and returns a new instance.
func (e *baseError) WithContext(context ErrorContext) OutputError {
	newErr := e.clone()
	newErr.context = context
	return newErr
}

// WithSuggestions creates a new error with the specified suggestions.
// This follows the builder pattern and returns a new instance.
func (e *baseError) WithSuggestions(suggestions ...string) OutputError {
	newErr := e.clone()
	newErr.suggestions = make([]string, len(suggestions))
	copy(newErr.suggestions, suggestions)
	return newErr
}

// WithSeverity creates a new error with the specified severity.
// This follows the builder pattern and returns a new instance.
func (e *baseError) WithSeverity(severity ErrorSeverity) OutputError {
	newErr := e.clone()
	newErr.severity = severity
	return newErr
}

// clone creates a deep copy of the error for immutable builder pattern
func (e *baseError) clone() *baseError {
	newErr := &baseError{
		code:        e.code,
		severity:    e.severity,
		message:     e.message,
		context:     e.context,
		suggestions: make([]string, len(e.suggestions)),
		cause:       e.cause,
	}

	copy(newErr.suggestions, e.suggestions)

	// Deep copy metadata if it exists
	if e.context.Metadata != nil {
		newErr.context.Metadata = make(map[string]interface{})
		for k, v := range e.context.Metadata {
			newErr.context.Metadata[k] = v
		}
	}

	return newErr
}

// MarshalJSON implements json.Marshaler interface for structured logging
func (e *baseError) MarshalJSON() ([]byte, error) {
	var causeStr string
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
