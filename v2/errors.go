package output

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// RenderError indicates rendering failure with detailed context
type RenderError struct {
	Format    string
	Content   Content
	Renderer  string         // Type information about the renderer
	Operation string         // Specific operation that failed (e.g., "encode", "format", "stream")
	Context   map[string]any // Additional context information
	Cause     error
}

// Error returns the error message
func (e *RenderError) Error() string {
	var parts []string

	// Operation and format
	if e.Operation != "" {
		parts = append(parts, fmt.Sprintf("operation %q failed", e.Operation))
	} else {
		parts = append(parts, "render failed")
	}

	// Format information
	if e.Format != "" {
		parts = append(parts, fmt.Sprintf("format=%s", e.Format))
	}

	// Renderer information
	if e.Renderer != "" {
		parts = append(parts, fmt.Sprintf("renderer=%s", e.Renderer))
	}

	// Content information
	if e.Content != nil {
		parts = append(parts, fmt.Sprintf("content_type=%s", e.Content.Type().String()))
		parts = append(parts, fmt.Sprintf("content_id=%s", e.Content.ID()))
	}

	// Additional context
	if len(e.Context) > 0 {
		var contextParts []string
		for key, value := range e.Context {
			contextParts = append(contextParts, fmt.Sprintf("%s=%v", key, value))
		}
		parts = append(parts, fmt.Sprintf("context=[%s]", strings.Join(contextParts, ", ")))
	}

	// Cause
	if e.Cause != nil {
		parts = append(parts, fmt.Sprintf("cause: %v", e.Cause))
	}

	return strings.Join(parts, "; ")
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
		Context: make(map[string]any),
		Cause:   cause,
	}
}

// NewRenderErrorWithDetails creates a new render error with detailed context
func NewRenderErrorWithDetails(format, renderer, operation string, content Content, cause error) *RenderError {
	return &RenderError{
		Format:    format,
		Content:   content,
		Renderer:  renderer,
		Operation: operation,
		Context:   make(map[string]any),
		Cause:     cause,
	}
}

// AddContext adds context information to the render error
func (e *RenderError) AddContext(key string, value any) *RenderError {
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	e.Context[key] = value
	return e
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
	Operation string                // The operation that failed
	Errors    []error               // The list of errors that occurred
	SourceMap map[error]ErrorSource // Maps errors to their sources
	Context   map[string]any        // Additional context for the operation
}

// ErrorSource provides information about where an error originated
type ErrorSource struct {
	Component string         // Component that generated the error (e.g., "renderer", "transformer", "writer")
	Details   map[string]any // Additional source details
}

// Error returns a formatted message for all errors
func (e *MultiError) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}

	if len(e.Errors) == 1 {
		var contextInfo string
		if len(e.Context) > 0 {
			var contextParts []string
			for key, value := range e.Context {
				contextParts = append(contextParts, fmt.Sprintf("%s=%v", key, value))
			}
			contextInfo = fmt.Sprintf(" [%s]", strings.Join(contextParts, ", "))
		}

		if e.Operation != "" {
			return fmt.Sprintf("%s%s: %v", e.Operation, contextInfo, e.Errors[0])
		}
		return fmt.Sprintf("%v%s", e.Errors[0], contextInfo)
	}

	var parts []string
	if e.Operation != "" {
		parts = append(parts, fmt.Sprintf("%s failed with %d errors:", e.Operation, len(e.Errors)))
	} else {
		parts = append(parts, fmt.Sprintf("%d errors occurred:", len(e.Errors)))
	}

	// Add context information
	if len(e.Context) > 0 {
		var contextParts []string
		for key, value := range e.Context {
			contextParts = append(contextParts, fmt.Sprintf("%s=%v", key, value))
		}
		parts = append(parts, fmt.Sprintf("Context: %s", strings.Join(contextParts, ", ")))
	}

	for i, err := range e.Errors {
		errorInfo := fmt.Sprintf("  %d. %v", i+1, err)

		// Add source information if available
		if source, exists := e.SourceMap[err]; exists {
			var sourceDetails []string
			if source.Component != "" {
				sourceDetails = append(sourceDetails, fmt.Sprintf("component=%s", source.Component))
			}
			for key, value := range source.Details {
				sourceDetails = append(sourceDetails, fmt.Sprintf("%s=%v", key, value))
			}
			if len(sourceDetails) > 0 {
				errorInfo += fmt.Sprintf(" [%s]", strings.Join(sourceDetails, ", "))
			}
		}

		parts = append(parts, errorInfo)
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

// AddWithSource adds an error with source information to the multi-error
func (e *MultiError) AddWithSource(err error, component string, details map[string]any) {
	if err != nil {
		e.Errors = append(e.Errors, err)
		if e.SourceMap == nil {
			e.SourceMap = make(map[error]ErrorSource)
		}
		e.SourceMap[err] = ErrorSource{
			Component: component,
			Details:   details,
		}
	}
}

// AddContext adds context information to the multi-error
func (e *MultiError) AddContext(key string, value any) *MultiError {
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	e.Context[key] = value
	return e
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
		SourceMap: make(map[error]ErrorSource),
		Context:   make(map[string]any),
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

// StructuredError provides machine-readable error information for programmatic analysis
type StructuredError struct {
	Code      string         `json:"code"`      // Error code for programmatic identification
	Message   string         `json:"message"`   // Human-readable error message
	Component string         `json:"component"` // Component that generated the error
	Operation string         `json:"operation"` // Operation that failed
	Context   map[string]any `json:"context"`   // Additional context information
	Details   map[string]any `json:"details"`   // Detailed error information
	Timestamp string         `json:"timestamp"` // When the error occurred
	Cause     error          `json:"-"`         // Underlying error (not serialized)
}

// Error implements the error interface
func (e *StructuredError) Error() string {
	var parts []string

	if e.Code != "" {
		parts = append(parts, fmt.Sprintf("code=%s", e.Code))
	}

	if e.Component != "" {
		parts = append(parts, fmt.Sprintf("component=%s", e.Component))
	}

	if e.Operation != "" {
		parts = append(parts, fmt.Sprintf("operation=%s", e.Operation))
	}

	if e.Message != "" {
		parts = append(parts, fmt.Sprintf("message=%s", e.Message))
	}

	// Add context information
	if len(e.Context) > 0 {
		var contextParts []string
		for key, value := range e.Context {
			contextParts = append(contextParts, fmt.Sprintf("%s=%v", key, value))
		}
		parts = append(parts, fmt.Sprintf("context=[%s]", strings.Join(contextParts, ", ")))
	}

	if e.Cause != nil {
		parts = append(parts, fmt.Sprintf("cause: %v", e.Cause))
	}

	return strings.Join(parts, "; ")
}

// Unwrap returns the underlying error
func (e *StructuredError) Unwrap() error {
	return e.Cause
}

// NewStructuredError creates a new structured error
func NewStructuredError(code, component, operation, message string) *StructuredError {
	return &StructuredError{
		Code:      code,
		Component: component,
		Operation: operation,
		Message:   message,
		Context:   make(map[string]any),
		Details:   make(map[string]any),
		Timestamp: fmt.Sprintf("%d", timeNow().Unix()),
	}
}

// NewStructuredErrorWithCause creates a new structured error with an underlying cause
func NewStructuredErrorWithCause(code, component, operation, message string, cause error) *StructuredError {
	return &StructuredError{
		Code:      code,
		Component: component,
		Operation: operation,
		Message:   message,
		Context:   make(map[string]any),
		Details:   make(map[string]any),
		Timestamp: fmt.Sprintf("%d", timeNow().Unix()),
		Cause:     cause,
	}
}

// AddContext adds context information to the structured error
func (e *StructuredError) AddContext(key string, value any) *StructuredError {
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	e.Context[key] = value
	return e
}

// AddDetail adds detailed information to the structured error
func (e *StructuredError) AddDetail(key string, value any) *StructuredError {
	if e.Details == nil {
		e.Details = make(map[string]any)
	}
	e.Details[key] = value
	return e
}

// ToStructuredError converts any error to a StructuredError for analysis
func ToStructuredError(err error, defaultCode, defaultComponent, defaultOperation string) *StructuredError {
	if err == nil {
		return nil
	}

	// Check if it's already a StructuredError
	var structErr *StructuredError
	if AsError(err, &structErr) {
		return structErr
	}

	// Check for specific error types and extract information
	var renderErr *RenderError
	if AsError(err, &renderErr) {
		structured := NewStructuredErrorWithCause("RENDER_ERROR", "renderer", renderErr.Operation, renderErr.Error(), renderErr)
		structured.AddContext("format", renderErr.Format)
		structured.AddContext("renderer", renderErr.Renderer)
		if renderErr.Content != nil {
			structured.AddContext("content_type", renderErr.Content.Type().String())
			structured.AddContext("content_id", renderErr.Content.ID())
		}
		for k, v := range renderErr.Context {
			structured.AddContext(k, v)
		}
		return structured
	}

	var transformErr *TransformError
	if AsError(err, &transformErr) {
		structured := NewStructuredErrorWithCause("TRANSFORM_ERROR", "transformer", "transform", transformErr.Error(), transformErr)
		structured.AddContext("transformer", transformErr.Transformer)
		structured.AddContext("format", transformErr.Format)
		structured.AddContext("input_size", len(transformErr.Input))
		return structured
	}

	var validationErr *ValidationError
	if AsError(err, &validationErr) {
		structured := NewStructuredErrorWithCause("VALIDATION_ERROR", "validator", "validate", validationErr.Error(), validationErr)
		structured.AddContext("field", validationErr.Field)
		structured.AddDetail("value", validationErr.Value)
		structured.AddDetail("validation_message", validationErr.Message)
		return structured
	}

	var cancelledErr *CancelledError
	if AsError(err, &cancelledErr) {
		structured := NewStructuredErrorWithCause("CANCELLED_ERROR", "system", cancelledErr.Operation, cancelledErr.Error(), cancelledErr)
		return structured
	}

	var writerErr *WriterError
	if AsError(err, &writerErr) {
		structured := NewStructuredErrorWithCause("WRITER_ERROR", "writer", writerErr.Operation, writerErr.Error(), writerErr)
		structured.AddContext("writer", writerErr.Writer)
		structured.AddContext("format", writerErr.Format)
		for k, v := range writerErr.Context {
			structured.AddContext(k, v)
		}
		return structured
	}

	// Default structured error for unknown types
	return NewStructuredErrorWithCause(defaultCode, defaultComponent, defaultOperation, err.Error(), err)
}

// WriterError represents an error that occurred during writing
type WriterError struct {
	Writer    string         // Type information about the writer
	Format    string         // Format being written
	Operation string         // Specific operation that failed (e.g., "write", "open", "close")
	Context   map[string]any // Additional context information
	Cause     error          // The underlying error
}

// Error returns the error message
func (e *WriterError) Error() string {
	var parts []string

	// Operation and format
	if e.Operation != "" {
		parts = append(parts, fmt.Sprintf("operation %q failed", e.Operation))
	} else {
		parts = append(parts, "write failed")
	}

	// Format information
	if e.Format != "" {
		parts = append(parts, fmt.Sprintf("format=%s", e.Format))
	}

	// Writer information
	if e.Writer != "" {
		parts = append(parts, fmt.Sprintf("writer=%s", e.Writer))
	}

	// Additional context
	if len(e.Context) > 0 {
		var contextParts []string
		for key, value := range e.Context {
			contextParts = append(contextParts, fmt.Sprintf("%s=%v", key, value))
		}
		parts = append(parts, fmt.Sprintf("context=[%s]", strings.Join(contextParts, ", ")))
	}

	// Cause
	if e.Cause != nil {
		parts = append(parts, fmt.Sprintf("cause: %v", e.Cause))
	}

	return strings.Join(parts, "; ")
}

// Unwrap returns the underlying error
func (e *WriterError) Unwrap() error {
	return e.Cause
}

// NewWriterError creates a new writer error with context
func NewWriterError(writer, format string, cause error) *WriterError {
	return &WriterError{
		Writer:  writer,
		Format:  format,
		Context: make(map[string]any),
		Cause:   cause,
	}
}

// NewWriterErrorWithDetails creates a new writer error with detailed context
func NewWriterErrorWithDetails(writer, format, operation string, cause error) *WriterError {
	return &WriterError{
		Writer:    writer,
		Format:    format,
		Operation: operation,
		Context:   make(map[string]any),
		Cause:     cause,
	}
}

// AddContext adds context information to the writer error
func (e *WriterError) AddContext(key string, value any) *WriterError {
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	e.Context[key] = value
	return e
}

// timeNow returns the current time - this can be mocked in tests
var timeNow = time.Now
