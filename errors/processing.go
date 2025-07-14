package errors

import (
	"encoding/json"
	"strings"
)

// ProcessingError extends OutputError with processing-specific functionality for runtime failures
type ProcessingError interface {
	OutputError
	Retryable() bool                               // Whether this error is suitable for retry
	PartialResult() interface{}                    // Any partial result that was produced before the error
	WithRetryable(bool) ProcessingError            // Builder method to set retryable flag
	WithPartialResult(interface{}) ProcessingError // Builder method to set partial result
}

// processingError is the concrete implementation of ProcessingError
type processingError struct {
	*baseError
	retryable     bool
	partialResult interface{}
}

// NewProcessingError creates a new ProcessingError with the specified code and message
func NewProcessingError(code ErrorCode, message string) ProcessingError {
	severity := SeverityError
	// Memory exhaustion is typically fatal
	if code == ErrMemoryExhausted {
		severity = SeverityFatal
	}

	return &processingError{
		baseError:     newBaseError(code, message).WithSeverity(severity).(*baseError),
		retryable:     false,
		partialResult: nil,
	}
}

// NewProcessingErrorWithOptions creates a new ProcessingError with initial options
func NewProcessingErrorWithOptions(code ErrorCode, message string, retryable bool, partialResult interface{}) ProcessingError {
	err := NewProcessingError(code, message)
	return err.WithRetryable(retryable).WithPartialResult(partialResult)
}

// Error returns the formatted error message including retry and partial result information
func (e *processingError) Error() string {
	var b strings.Builder

	// Start with base error message
	b.WriteString(e.baseError.Error())

	// Add processing-specific information
	if e.retryable {
		b.WriteString("\nThis error is retryable.")
	}

	if e.partialResult != nil {
		b.WriteString("\nPartial result available.")
	}

	return b.String()
}

// Retryable returns whether this error is suitable for retry
func (e *processingError) Retryable() bool {
	return e.retryable
}

// PartialResult returns any partial result that was produced before the error
func (e *processingError) PartialResult() interface{} {
	return e.partialResult
}

// WithRetryable creates a new ProcessingError with the specified retryable flag
func (e *processingError) WithRetryable(retryable bool) ProcessingError {
	newErr := &processingError{
		baseError:     e.baseError.clone(),
		retryable:     retryable,
		partialResult: e.partialResult,
	}
	return newErr
}

// WithPartialResult creates a new ProcessingError with the specified partial result
func (e *processingError) WithPartialResult(result interface{}) ProcessingError {
	newErr := &processingError{
		baseError:     e.baseError.clone(),
		retryable:     e.retryable,
		partialResult: result,
	}
	return newErr
}

// WithContext creates a new ProcessingError with the specified context
func (e *processingError) WithContext(context ErrorContext) OutputError {
	newErr := &processingError{
		baseError:     e.baseError.clone().WithContext(context).(*baseError),
		retryable:     e.retryable,
		partialResult: e.partialResult,
	}
	return newErr
}

// WithSuggestions creates a new ProcessingError with the specified suggestions
func (e *processingError) WithSuggestions(suggestions ...string) OutputError {
	newErr := &processingError{
		baseError:     e.baseError.clone().WithSuggestions(suggestions...).(*baseError),
		retryable:     e.retryable,
		partialResult: e.partialResult,
	}
	return newErr
}

// WithSeverity creates a new ProcessingError with the specified severity
func (e *processingError) WithSeverity(severity ErrorSeverity) OutputError {
	newErr := &processingError{
		baseError:     e.baseError.clone().WithSeverity(severity).(*baseError),
		retryable:     e.retryable,
		partialResult: e.partialResult,
	}
	return newErr
}

// Wrap creates a new ProcessingError that wraps the given error as the cause
func (e *processingError) Wrap(cause error) OutputError {
	newErr := &processingError{
		baseError:     e.baseError.clone().Wrap(cause).(*baseError),
		retryable:     e.retryable,
		partialResult: e.partialResult,
	}
	return newErr
}

// MarshalJSON implements json.Marshaler interface for structured logging
func (e *processingError) MarshalJSON() ([]byte, error) {
	var causeStr string
	if e.baseError.cause != nil {
		causeStr = e.baseError.cause.Error()
	}

	return json.Marshal(struct {
		Code          ErrorCode    `json:"code"`
		Severity      string       `json:"severity"`
		Message       string       `json:"message"`
		Context       ErrorContext `json:"context,omitempty"`
		Suggestions   []string     `json:"suggestions,omitempty"`
		Cause         string       `json:"cause,omitempty"`
		Retryable     bool         `json:"retryable"`
		PartialResult interface{}  `json:"partial_result,omitempty"`
	}{
		Code:          e.baseError.code,
		Severity:      e.baseError.severity.String(),
		Message:       e.baseError.message,
		Context:       e.baseError.context,
		Suggestions:   e.baseError.suggestions,
		Cause:         causeStr,
		Retryable:     e.retryable,
		PartialResult: e.partialResult,
	})
}

// retryableError is a special processing error that wraps other errors as retryable
type retryableError struct {
	*processingError
	maxAttempts    int // Maximum retry attempts
	initialDelayMs int // Initial delay in milliseconds for exponential backoff
}

// NewRetryableError creates a new retryable error that wraps another error
func NewRetryableError(cause error, message string) ProcessingError {
	err := NewProcessingError(ErrRetryable, message).
		WithRetryable(true).
		Wrap(cause)

	if processingErr, ok := err.(*processingError); ok {
		return &retryableError{
			processingError: processingErr,
			maxAttempts:     3,
			initialDelayMs:  1000,
		}
	}

	return err.(ProcessingError)
}

// NewRetryableErrorWithBackoff creates a new retryable error with specific retry configuration
func NewRetryableErrorWithBackoff(cause error, message string, maxAttempts int, initialDelayMs int) ProcessingError {
	err := NewProcessingError(ErrRetryable, message).
		WithRetryable(true).
		Wrap(cause)

	if processingErr, ok := err.(*processingError); ok {
		return &retryableError{
			processingError: processingErr,
			maxAttempts:     maxAttempts,
			initialDelayMs:  initialDelayMs,
		}
	}

	return err.(ProcessingError)
}

// WrapAsRetryable wraps an existing error as retryable
func WrapAsRetryable(err error) ProcessingError {
	if processingErr, ok := err.(ProcessingError); ok {
		// If it's already a ProcessingError, make it retryable
		return processingErr.WithRetryable(true)
	}

	// Otherwise, wrap it as a new retryable error
	return NewRetryableError(err, "Operation failed but can be retried")
}

// IsTransient determines if an error represents a transient condition that might succeed on retry
func IsTransient(err error) bool {
	if processingErr, ok := err.(ProcessingError); ok {
		if processingErr.Retryable() {
			return true
		}

		// Some error codes are inherently transient
		switch processingErr.Code() {
		case ErrS3Upload:
			return true // Network operations are typically transient
		case ErrFileWrite, ErrTemplateRender, ErrMemoryExhausted:
			return false // These are typically not transient
		}
	}

	return false
}
