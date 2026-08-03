package output

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

func TestNewStderrWriter(t *testing.T) {
	sw := NewStderrWriter()

	if sw == nil {
		t.Fatal("NewStderrWriter returned nil")
	}

	if sw.name != "stderr" {
		t.Errorf("name = %q, want %q", sw.name, "stderr")
	}

	if sw.writer == nil {
		t.Error("writer is nil")
	}
}

func TestStderrWriterWrite(t *testing.T) {
	tests := map[string]struct {
		format     string
		data       []byte
		wantOutput string
		wantErr    bool
	}{"empty data with newline": {

		format:     FormatText,
		data:       []byte{},
		wantOutput: "",
		wantErr:    false,
	}, "empty format": {

		format:  "",
		data:    []byte("test"),
		wantErr: true,
	}, "multiple lines": {

		format:     FormatText,
		data:       []byte("line1\nline2\nline3"),
		wantOutput: "line1\nline2\nline3\n",
		wantErr:    false,
	}, "nil data": {

		format:  FormatText,
		data:    nil,
		wantErr: true,
	}, "write with newline": {

		format:     FormatText,
		data:       []byte("hello world\n"),
		wantOutput: "hello world\n",
		wantErr:    false,
	}, "write without newline": {

		format:     FormatText,
		data:       []byte("hello world"),
		wantOutput: "hello world\n",
		wantErr:    false,
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			sw := NewStderrWriter()
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

func TestStderrWriterConcurrency(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStderrWriter()
	sw.SetWriter(&buf)

	ctx := context.Background()
	var wg sync.WaitGroup

	// Write concurrently
	for i := range 10 {
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

func TestStderrWriterContextCancellation(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStderrWriter()
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

// TestStderrWriterSetWriterNil is a regression test for T-1387.
// SetWriter(nil) used to store the nil writer without validation, so a later
// Write passed input validation and then panicked on sw.writer.Write(data).
// Expected: a nil writer is ignored so the existing writer is kept, matching
// the nil handling in MultiWriter.AddWriter and WithProgressWriter.
func TestStderrWriterSetWriterNil(t *testing.T) {
	t.Run("nil writer is ignored and existing writer kept", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Write panicked after SetWriter(nil): %v", r)
			}
		}()

		var buf bytes.Buffer
		sw := NewStderrWriter()
		sw.SetWriter(&buf)
		sw.SetWriter(nil)

		if err := sw.Write(context.Background(), FormatText, []byte("test")); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got, want := buf.String(), "test\n"; got != want {
			t.Errorf("output = %q, want %q (existing writer should be kept)", got, want)
		}
	})

	t.Run("zero-value writer returns error instead of panicking", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Write panicked on nil internal writer: %v", r)
			}
		}()

		// A zero-value StderrWriter never went through NewStderrWriter, so
		// its internal writer is nil. Write must fail with a WriteError
		// rather than panic.
		sw := &StderrWriter{baseWriter: baseWriter{name: "stderr"}}

		err := sw.Write(context.Background(), FormatText, []byte("test"))
		if err == nil {
			t.Fatal("expected error for nil internal writer, got nil")
		}

		var writeErr *WriteError
		if !errors.As(err, &writeErr) {
			t.Errorf("error type = %T, want *WriteError", err)
		}
	})
}

func TestStderrWriterWriteError(t *testing.T) {
	sw := NewStderrWriter()
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

	if writeErr.Writer != "stderr" {
		t.Errorf("Writer = %q, want %q", writeErr.Writer, "stderr")
	}

	if writeErr.Format != FormatText {
		t.Errorf("Format = %q, want %q", writeErr.Format, FormatText)
	}
}
