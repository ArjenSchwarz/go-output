package output

import (
	"context"
	"errors"
	"testing"
)

func TestWriterFunc(t *testing.T) {
	var called bool
	var receivedFormat string
	var receivedData []byte

	writer := WriterFunc(func(ctx context.Context, format string, data []byte) error {
		called = true
		receivedFormat = format
		receivedData = data
		return nil
	})

	ctx := context.Background()
	testData := []byte("test data")

	err := writer.Write(ctx, FormatJSON, testData)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !called {
		t.Error("WriterFunc was not called")
	}

	if receivedFormat != FormatJSON {
		t.Errorf("format = %q, want %q", receivedFormat, FormatJSON)
	}

	if string(receivedData) != string(testData) {
		t.Errorf("data = %q, want %q", receivedData, testData)
	}
}

func TestWriterFuncWithError(t *testing.T) {
	expectedErr := errors.New("write failed")
	writer := WriterFunc(func(ctx context.Context, format string, data []byte) error {
		return expectedErr
	})

	ctx := context.Background()
	err := writer.Write(ctx, FormatJSON, []byte("test"))

	if err != expectedErr {
		t.Errorf("error = %v, want %v", err, expectedErr)
	}
}

func TestWriterFuncWithCancellation(t *testing.T) {
	writer := WriterFunc(func(ctx context.Context, format string, data []byte) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := writer.Write(ctx, FormatJSON, []byte("test"))
	if err != context.Canceled {
		t.Errorf("error = %v, want %v", err, context.Canceled)
	}
}

func TestWriteError(t *testing.T) {
	cause := errors.New("underlying error")
	err := &WriteError{
		Writer: "TestWriter",
		Format: FormatJSON,
		Cause:  cause,
	}

	expectedMsg := "write error for TestWriter writer (format: json): underlying error"
	if err.Error() != expectedMsg {
		t.Errorf("Error() = %q, want %q", err.Error(), expectedMsg)
	}

	if err.Unwrap() != cause {
		t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), cause)
	}
}

func TestBaseWriterValidation(t *testing.T) {
	bw := &baseWriter{name: "test"}

	tests := []struct {
		name    string
		format  string
		data    []byte
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid input",
			format:  FormatJSON,
			data:    []byte("test"),
			wantErr: false,
		},
		{
			name:    "empty format",
			format:  "",
			data:    []byte("test"),
			wantErr: true,
			errMsg:  "format cannot be empty",
		},
		{
			name:    "nil data",
			format:  FormatJSON,
			data:    nil,
			wantErr: true,
			errMsg:  "data cannot be nil",
		},
		{
			name:    "empty data is valid",
			format:  FormatJSON,
			data:    []byte{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bw.validateInput(tt.format, tt.data)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
					return
				}

				var writeErr *WriteError
				if !errors.As(err, &writeErr) {
					t.Errorf("error type = %T, want *WriteError", err)
				}

				if writeErr.Writer != "test" {
					t.Errorf("writer = %q, want %q", writeErr.Writer, "test")
				}

				if writeErr.Format != tt.format {
					t.Errorf("format = %q, want %q", writeErr.Format, tt.format)
				}

				if !errors.Is(err, writeErr.Cause) {
					t.Errorf("cause not properly wrapped")
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestBaseWriterWrapError(t *testing.T) {
	bw := &baseWriter{name: "TestWriter"}

	// Test nil error
	if err := bw.wrapError(FormatJSON, nil); err != nil {
		t.Errorf("wrapError(nil) = %v, want nil", err)
	}

	// Test non-nil error
	cause := errors.New("test error")
	err := bw.wrapError(FormatYAML, cause)

	var writeErr *WriteError
	if !errors.As(err, &writeErr) {
		t.Fatalf("error type = %T, want *WriteError", err)
	}

	if writeErr.Writer != "TestWriter" {
		t.Errorf("Writer = %q, want %q", writeErr.Writer, "TestWriter")
	}

	if writeErr.Format != FormatYAML {
		t.Errorf("Format = %q, want %q", writeErr.Format, FormatYAML)
	}

	if !errors.Is(err, cause) {
		t.Error("wrapped error should be retrievable with errors.Is")
	}
}
