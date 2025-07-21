package output

import (
	"context"
	"fmt"
	"strings"
)

// RenderError indicates rendering failure with detailed context
type RenderError struct {
	Format  string
	Content Content
	Cause   error
}

// Error returns the error message
func (e *RenderError) Error() string {
	contentType := "unknown"
	contentID := "unknown"
	if e.Content != nil {
		contentType = e.Content.Type().String()
		contentID = e.Content.ID()
	}
	return fmt.Sprintf("render %s for %s content %s: %v", e.Format, contentType, contentID, e.Cause)
}

// Unwrap returns the underlying error
func (e *RenderError) Unwrap() error {
	return e.Cause
}

// NewRenderError creates a new render error with context
func NewRenderError(format string, content Content, cause error) *RenderError {
	return &RenderError{
		Format:  format,
		Content: content,
		Cause:   cause,
	}
}

// ValidationError indicates invalid input with detailed field information
type ValidationError struct {
	Field   string // The field that failed validation
	Value   any    // The value that was invalid
	Message string // Human-readable error message
	Cause   error  // Optional underlying error
}

// Error returns the error message
func (e *ValidationError) Error() string {
	var parts []string

	if e.Field != "" {
		parts = append(parts, fmt.Sprintf("field %q", e.Field))
	}

	// Always include value part, even for nil
	parts = append(parts, fmt.Sprintf("value %v", e.Value))

	if e.Message != "" {
		parts = append(parts, e.Message)
	} else {
		parts = append(parts, "validation failed")
	}

	result := strings.Join(parts, ": ")

	if e.Cause != nil {
		result += fmt.Sprintf(": %v", e.Cause)
	}

	return result
}

// Unwrap returns the underlying error
func (e *ValidationError) Unwrap() error {
	return e.Cause
}

// NewValidationError creates a new validation error
func NewValidationError(field string, value any, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// NewValidationErrorWithCause creates a new validation error with an underlying cause
func NewValidationErrorWithCause(field string, value any, message string, cause error) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
		Cause:   cause,
	}
}

// ContextError represents an error with additional context information
type ContextError struct {
	Operation string         // The operation that failed (e.g., "render", "transform", "write")
	Context   map[string]any // Additional context information
	Cause     error          // The underlying error
}

// Error returns the error message with context
func (e *ContextError) Error() string {
	var parts []string

	if e.Operation != "" {
		parts = append(parts, fmt.Sprintf("operation %q failed", e.Operation))
	}

	if len(e.Context) > 0 {
		var contextParts []string
		for key, value := range e.Context {
			contextParts = append(contextParts, fmt.Sprintf("%s=%v", key, value))
		}
		parts = append(parts, fmt.Sprintf("context: %s", strings.Join(contextParts, ", ")))
	}

	if e.Cause != nil {
		parts = append(parts, fmt.Sprintf("cause: %v", e.Cause))
	}

	return strings.Join(parts, "; ")
}

// Unwrap returns the underlying error
func (e *ContextError) Unwrap() error {
	return e.Cause
}

// AddContext adds context information to the error
func (e *ContextError) AddContext(key string, value any) *ContextError {
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	e.Context[key] = value
	return e
}

// NewContextError creates a new context error
func NewContextError(operation string, cause error) *ContextError {
	return &ContextError{
		Operation: operation,
		Context:   make(map[string]any),
		Cause:     cause,
	}
}

// ErrorWithContext wraps an error with context information
func ErrorWithContext(operation string, cause error, contextPairs ...any) error {
	if cause == nil {
		return nil
	}

	ctxErr := NewContextError(operation, cause)

	// Add context pairs (key, value, key, value, ...)
	for i := 0; i < len(contextPairs)-1; i += 2 {
		if key, ok := contextPairs[i].(string); ok {
			ctxErr.AddContext(key, contextPairs[i+1])
		}
	}

	return ctxErr
}

// MultiError represents multiple errors that occurred during an operation
type MultiError struct {
	Operation string  // The operation that failed
	Errors    []error // The list of errors that occurred
}

// Error returns a formatted message for all errors
func (e *MultiError) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}

	if len(e.Errors) == 1 {
		if e.Operation != "" {
			return fmt.Sprintf("%s: %v", e.Operation, e.Errors[0])
		}
		return e.Errors[0].Error()
	}

	var parts []string
	if e.Operation != "" {
		parts = append(parts, fmt.Sprintf("%s failed with %d errors:", e.Operation, len(e.Errors)))
	} else {
		parts = append(parts, fmt.Sprintf("%d errors occurred:", len(e.Errors)))
	}

	for i, err := range e.Errors {
		parts = append(parts, fmt.Sprintf("  %d. %v", i+1, err))
	}

	return strings.Join(parts, "\n")
}

// Unwrap returns the first error for compatibility with errors.Unwrap
func (e *MultiError) Unwrap() error {
	if len(e.Errors) == 0 {
		return nil
	}
	return e.Errors[0]
}

// Add adds an error to the multi-error
func (e *MultiError) Add(err error) {
	if err != nil {
		e.Errors = append(e.Errors, err)
	}
}

// HasErrors returns true if there are any errors
func (e *MultiError) HasErrors() bool {
	return len(e.Errors) > 0
}

// ErrorOrNil returns the MultiError if it has errors, otherwise nil
func (e *MultiError) ErrorOrNil() error {
	if e.HasErrors() {
		return e
	}
	return nil
}

// NewMultiError creates a new multi-error
func NewMultiError(operation string) *MultiError {
	return &MultiError{
		Operation: operation,
		Errors:    make([]error, 0),
	}
}

// CancelledError represents an operation that was cancelled
type CancelledError struct {
	Operation string // The operation that was cancelled
	Cause     error  // The cancellation cause (usually context.Canceled or context.DeadlineExceeded)
}

// Error returns the error message
func (e *CancelledError) Error() string {
	if e.Operation != "" {
		return fmt.Sprintf("operation %q was cancelled: %v", e.Operation, e.Cause)
	}
	return fmt.Sprintf("operation was cancelled: %v", e.Cause)
}

// Unwrap returns the underlying error
func (e *CancelledError) Unwrap() error {
	return e.Cause
}

// NewCancelledError creates a new cancelled error
func NewCancelledError(operation string, cause error) *CancelledError {
	return &CancelledError{
		Operation: operation,
		Cause:     cause,
	}
}

// IsCancelled checks if an error represents a cancellation
func IsCancelled(err error) bool {
	if err == nil {
		return false
	}

	// Check for context cancellation
	if err == context.Canceled || err == context.DeadlineExceeded {
		return true
	}

	// Check for wrapped cancellation errors
	var cancelledErr *CancelledError
	return AsError(err, &cancelledErr)
}

// AsError is a wrapper around errors.As for better type safety
func AsError[T error](err error, target *T) bool {
	if err == nil {
		return false
	}

	// Try direct type assertion first
	if typedErr, ok := err.(T); ok {
		*target = typedErr
		return true
	}

	// For wrapped errors, we need to unwrap and check
	for err != nil {
		if typedErr, ok := err.(T); ok {
			*target = typedErr
			return true
		}

		// Try to unwrap the error
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrapper.Unwrap()
		} else {
			break
		}
	}

	return false
}

// ValidateInput performs early validation with helpful error messages
func ValidateInput(field string, value any, validator func(any) error) error {
	if validator == nil {
		return nil
	}

	if err := validator(value); err != nil {
		return NewValidationErrorWithCause(field, value, "validation failed", err)
	}

	return nil
}

// ValidateNonEmpty validates that a string field is not empty
func ValidateNonEmpty(field, value string) error {
	if value == "" {
		return NewValidationError(field, value, "cannot be empty")
	}
	return nil
}

// ValidateNonNil validates that a value is not nil
func ValidateNonNil(field string, value any) error {
	if value == nil {
		return NewValidationError(field, value, "cannot be nil")
	}
	return nil
}

// ValidateSliceNonEmpty validates that a slice is not empty
func ValidateSliceNonEmpty[T any](field string, value []T) error {
	if len(value) == 0 {
		return NewValidationError(field, value, "cannot be empty")
	}
	return nil
}

// ValidateMapNonEmpty validates that a map is not empty
func ValidateMapNonEmpty[K comparable, V any](field string, value map[K]V) error {
	if len(value) == 0 {
		return NewValidationError(field, value, "cannot be empty")
	}
	return nil
}

// FailFast returns the first non-nil error from a list of errors
func FailFast(errors ...error) error {
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}

// CollectErrors collects all non-nil errors into a MultiError
func CollectErrors(operation string, errors ...error) error {
	multiErr := NewMultiError(operation)
	for _, err := range errors {
		if err != nil {
			multiErr.Add(err)
		}
	}
	return multiErr.ErrorOrNil()
}
