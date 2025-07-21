package output

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

func TestNewStdoutWriter(t *testing.T) {
	sw := NewStdoutWriter()

	if sw == nil {
		t.Fatal("NewStdoutWriter returned nil")
	}

	if sw.name != "stdout" {
		t.Errorf("name = %q, want %q", sw.name, "stdout")
	}

	if sw.writer == nil {
		t.Error("writer is nil")
	}
}

func TestStdoutWriterWrite(t *testing.T) {
	tests := []struct {
		name       string
		format     string
		data       []byte
		wantOutput string
		wantErr    bool
	}{
		{
			name:       "write with newline",
			format:     FormatText,
			data:       []byte("hello world\n"),
			wantOutput: "hello world\n",
			wantErr:    false,
		},
		{
			name:       "write without newline",
			format:     FormatText,
			data:       []byte("hello world"),
			wantOutput: "hello world\n",
			wantErr:    false,
		},
		{
			name:       "empty data with newline",
			format:     FormatText,
			data:       []byte{},
			wantOutput: "",
			wantErr:    false,
		},
		{
			name:       "multiple lines",
			format:     FormatText,
			data:       []byte("line1\nline2\nline3"),
			wantOutput: "line1\nline2\nline3\n",
			wantErr:    false,
		},
		{
			name:    "empty format",
			format:  "",
			data:    []byte("test"),
			wantErr: true,
		},
		{
			name:    "nil data",
			format:  FormatText,
			data:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			sw := NewStdoutWriter()
			sw.SetWriter(&buf)

			ctx := context.Background()
			err := sw.Write(ctx, tt.format, tt.data)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if got := buf.String(); got != tt.wantOutput {
				t.Errorf("output = %q, want %q", got, tt.wantOutput)
			}
		})
	}
}

func TestStdoutWriterConcurrency(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStdoutWriter()
	sw.SetWriter(&buf)

	ctx := context.Background()
	var wg sync.WaitGroup

	// Write concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			data := []byte(strings.Repeat("X", n+1))
			if err := sw.Write(ctx, FormatText, data); err != nil {
				t.Errorf("concurrent write %d failed: %v", n, err)
			}
		}(i)
	}

	wg.Wait()

	// Check that all writes succeeded
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 10 {
		t.Errorf("expected 10 lines, got %d", len(lines))
	}

	// Verify each line has the correct length
	seen := make(map[int]bool)
	for _, line := range lines {
		length := len(line)
		if length < 1 || length > 10 {
			t.Errorf("unexpected line length: %d", length)
		}
		seen[length] = true
	}

	// All lengths from 1-10 should be present
	for i := 1; i <= 10; i++ {
		if !seen[i] {
			t.Errorf("missing line with length %d", i)
		}
	}
}

func TestStdoutWriterContextCancellation(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStdoutWriter()
	sw.SetWriter(&buf)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := sw.Write(ctx, FormatText, []byte("test"))
	if err == nil {
		t.Error("expected context cancellation error")
	}

	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("error should mention context cancellation, got: %v", err)
	}

	// Buffer should be empty
	if buf.Len() > 0 {
		t.Error("no data should be written after cancellation")
	}
}

// mockFailWriter is a writer that always fails
type mockFailWriter struct{}

func (m *mockFailWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("write failed")
}

func TestStdoutWriterWriteError(t *testing.T) {
	sw := NewStdoutWriter()
	sw.SetWriter(&mockFailWriter{})

	ctx := context.Background()
	err := sw.Write(ctx, FormatText, []byte("test"))

	if err == nil {
		t.Error("expected write error")
	}

	var writeErr *WriteError
	if !errors.As(err, &writeErr) {
		t.Errorf("error type = %T, want *WriteError", err)
	}

	if writeErr.Writer != "stdout" {
		t.Errorf("Writer = %q, want %q", writeErr.Writer, "stdout")
	}

	if writeErr.Format != FormatText {
		t.Errorf("Format = %q, want %q", writeErr.Format, FormatText)
	}
}
