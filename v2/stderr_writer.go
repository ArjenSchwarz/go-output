package output

import (
	"context"
	"io"
	"os"
	"sync"
)

// StderrWriter writes rendered output to standard error
type StderrWriter struct {
	baseWriter
	mu     sync.Mutex // For concurrent access protection
	writer io.Writer  // Allows injection for testing
}

// NewStderrWriter creates a new StderrWriter
func NewStderrWriter() *StderrWriter {
	return &StderrWriter{
		baseWriter: baseWriter{name: "stderr"},
		writer:     os.Stderr,
	}
}

// Write implements the Writer interface
func (sw *StderrWriter) Write(ctx context.Context, format string, data []byte) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return sw.wrapError(format, ctx.Err())
	default:
	}

	// Validate input
	if err := sw.validateInput(format, data); err != nil {
		return err
	}

	// Write with proper locking for concurrent access
	sw.mu.Lock()
	defer sw.mu.Unlock()

	// Write all data at once
	_, err := sw.writer.Write(data)
	if err != nil {
		return sw.wrapError(format, err)
	}

	// Add newline if data doesn't end with one
	if len(data) > 0 && data[len(data)-1] != '\n' {
		_, err = sw.writer.Write([]byte{'\n'})
		if err != nil {
			return sw.wrapError(format, err)
		}
	}

	return nil
}

// SetWriter sets a custom writer (useful for testing)
func (sw *StderrWriter) SetWriter(w io.Writer) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.writer = w
}
