package output

import (
	"context"
	"fmt"
	"sync"
)

// MultiWriter writes rendered output to multiple destinations
type MultiWriter struct {
	baseWriter
	writers []Writer
	mu      sync.RWMutex // For concurrent access to writers slice
}

// NewMultiWriter creates a new MultiWriter with the specified writers
func NewMultiWriter(writers ...Writer) *MultiWriter {
	return &MultiWriter{
		baseWriter: baseWriter{name: "multi"},
		writers:    writers,
	}
}

// Write implements the Writer interface, writing to all configured writers
func (mw *MultiWriter) Write(ctx context.Context, format string, data []byte) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return mw.wrapError(format, ctx.Err())
	default:
	}

	// Validate input
	if err := mw.validateInput(format, data); err != nil {
		return err
	}

	// Get writers snapshot for concurrent access
	mw.mu.RLock()
	writers := make([]Writer, len(mw.writers))
	copy(writers, mw.writers)
	mw.mu.RUnlock()

	if len(writers) == 0 {
		return mw.wrapError(format, fmt.Errorf("no writers configured"))
	}

	// Write to all writers concurrently
	var wg sync.WaitGroup
	errChan := make(chan error, len(writers))

	for _, writer := range writers {
		wg.Add(1)
		go func(w Writer) {
			defer wg.Done()
			if err := w.Write(ctx, format, data); err != nil {
				errChan <- err
			}
		}(writer)
	}

	// Wait for all writers to complete
	wg.Wait()
	close(errChan)

	// Collect all errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	// If any writers failed, return a combined error
	if len(errors) > 0 {
		return mw.wrapError(format, &MultiWriteError{Errors: errors})
	}

	return nil
}

// AddWriter adds a writer to the multi-writer
func (mw *MultiWriter) AddWriter(w Writer) {
	mw.mu.Lock()
	defer mw.mu.Unlock()
	mw.writers = append(mw.writers, w)
}

// RemoveWriter removes a writer from the multi-writer
func (mw *MultiWriter) RemoveWriter(w Writer) {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	// Find and remove the writer
	for i, writer := range mw.writers {
		if writer == w {
			mw.writers = append(mw.writers[:i], mw.writers[i+1:]...)
			break
		}
	}
}

// WriterCount returns the number of configured writers
func (mw *MultiWriter) WriterCount() int {
	mw.mu.RLock()
	defer mw.mu.RUnlock()
	return len(mw.writers)
}

// MultiWriteError represents errors from multiple writers
type MultiWriteError struct {
	Errors []error
}

// Error returns a combined error message
func (e *MultiWriteError) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("multiple write errors (%d errors)", len(e.Errors))
}

// Unwrap returns the underlying errors for errors.Is and errors.As
func (e *MultiWriteError) Unwrap() []error {
	return e.Errors
}
