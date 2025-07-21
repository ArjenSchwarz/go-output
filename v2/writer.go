package output

import (
	"context"
	"fmt"
)

// Writer outputs rendered data to various destinations
type Writer interface {
	// Write outputs the rendered data for the specified format
	Write(ctx context.Context, format string, data []byte) error
}

// WriterFunc is an adapter to allow the use of ordinary functions as Writers
type WriterFunc func(ctx context.Context, format string, data []byte) error

// Write calls f(ctx, format, data)
func (f WriterFunc) Write(ctx context.Context, format string, data []byte) error {
	return f(ctx, format, data)
}

// WriteError represents an error that occurred during writing
type WriteError struct {
	Writer string
	Format string
	Cause  error
}

// Error returns the error message
func (e *WriteError) Error() string {
	return fmt.Sprintf("write error for %s writer (format: %s): %v", e.Writer, e.Format, e.Cause)
}

// Unwrap returns the underlying error
func (e *WriteError) Unwrap() error {
	return e.Cause
}

// baseWriter provides common functionality for writers
type baseWriter struct {
	name string
}

// wrapError wraps an error with writer context
func (b *baseWriter) wrapError(format string, err error) error {
	if err == nil {
		return nil
	}
	return &WriteError{
		Writer: b.name,
		Format: format,
		Cause:  err,
	}
}

// validateInput validates the input data
func (b *baseWriter) validateInput(format string, data []byte) error {
	if format == "" {
		return b.wrapError(format, fmt.Errorf("format cannot be empty"))
	}
	if data == nil {
		return b.wrapError(format, fmt.Errorf("data cannot be nil"))
	}
	return nil
}
