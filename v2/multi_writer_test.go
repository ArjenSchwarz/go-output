package output

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// mockWriter is a test writer that records writes
type mockWriter struct {
	name    string
	buf     bytes.Buffer
	failErr error
	mu      sync.Mutex
}

func (m *mockWriter) Write(ctx context.Context, format string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failErr != nil {
		return m.failErr
	}

	_, err := m.buf.Write(data)
	return err
}

func (m *mockWriter) String() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.buf.String()
}

func TestNewMultiWriter(t *testing.T) {
	w1 := &mockWriter{name: "w1"}
	w2 := &mockWriter{name: "w2"}

	mw := NewMultiWriter(w1, w2)

	if mw == nil {
		t.Fatal("NewMultiWriter returned nil")
	}

	if mw.WriterCount() != 2 {
		t.Errorf("WriterCount() = %d, want 2", mw.WriterCount())
	}
}

func TestMultiWriterWrite(t *testing.T) {
	ctx := context.Background()
	testData := []byte("test data")

	tests := map[string]struct {
		writers  []Writer
		format   string
		data     []byte
		wantErr  bool
		errCount int
	}{"empty format": {

		writers: []Writer{
			&mockWriter{name: "w1"},
		},
		format:  "",
		data:    testData,
		wantErr: true,
	}, "multiple writers fail": {

		writers: []Writer{
			&mockWriter{name: "w1", failErr: errors.New("fail 1")},
			&mockWriter{name: "w2"},
			&mockWriter{name: "w3", failErr: errors.New("fail 2")},
		},
		format:   FormatText,
		data:     testData,
		wantErr:  true,
		errCount: 2,
	}, "nil data": {

		writers: []Writer{
			&mockWriter{name: "w1"},
		},
		format:  FormatText,
		data:    nil,
		wantErr: true,
	}, "no writers": {

		writers: []Writer{},
		format:  FormatText,
		data:    testData,
		wantErr: true,
	}, "one writer fails": {

		writers: []Writer{
			&mockWriter{name: "w1"},
			&mockWriter{name: "w2", failErr: errors.New("write failed")},
			&mockWriter{name: "w3"},
		},
		format:   FormatText,
		data:     testData,
		wantErr:  true,
		errCount: 1,
	}, "write to multiple writers": {

		writers: []Writer{
			&mockWriter{name: "w1"},
			&mockWriter{name: "w2"},
			&mockWriter{name: "w3"},
		},
		format:  FormatText,
		data:    testData,
		wantErr: false,
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mw := NewMultiWriter(tt.writers...)
			err := mw.Write(ctx, tt.format, tt.data)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}

				// Check error count for MultiWriteError
				if tt.errCount > 0 {
					var mwErr *MultiWriteError
					if errors.As(err, &mwErr) {
						if len(mwErr.Errors) != tt.errCount {
							t.Errorf("error count = %d, want %d", len(mwErr.Errors), tt.errCount)
						}
					}
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify all successful writers received the data
			for _, w := range tt.writers {
				if mw, ok := w.(*mockWriter); ok && mw.failErr == nil {
					if got := mw.String(); got != string(tt.data) {
						t.Errorf("writer %s: got %q, want %q", mw.name, got, string(tt.data))
					}
				}
			}
		})
	}
}

func TestMultiWriterAddRemove(t *testing.T) {
	mw := NewMultiWriter()

	if mw.WriterCount() != 0 {
		t.Errorf("initial WriterCount() = %d, want 0", mw.WriterCount())
	}

	w1 := &mockWriter{name: "w1"}
	w2 := &mockWriter{name: "w2"}
	w3 := &mockWriter{name: "w3"}

	// Add writers
	mw.AddWriter(w1)
	if mw.WriterCount() != 1 {
		t.Errorf("after adding w1: WriterCount() = %d, want 1", mw.WriterCount())
	}

	mw.AddWriter(w2)
	mw.AddWriter(w3)
	if mw.WriterCount() != 3 {
		t.Errorf("after adding all: WriterCount() = %d, want 3", mw.WriterCount())
	}

	// Remove writer
	mw.RemoveWriter(w2)
	if mw.WriterCount() != 2 {
		t.Errorf("after removing w2: WriterCount() = %d, want 2", mw.WriterCount())
	}

	// Remove non-existent writer (should be no-op)
	mw.RemoveWriter(w2)
	if mw.WriterCount() != 2 {
		t.Errorf("after removing w2 again: WriterCount() = %d, want 2", mw.WriterCount())
	}

	// Write and verify only w1 and w3 receive data
	ctx := context.Background()
	testData := []byte("test")
	err := mw.Write(ctx, FormatText, testData)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if w1.String() != "test" {
		t.Error("w1 should have received data")
	}
	if w2.String() != "" {
		t.Error("w2 should not have received data")
	}
	if w3.String() != "test" {
		t.Error("w3 should have received data")
	}
}

func TestMultiWriterConcurrency(t *testing.T) {
	// Create writers that count writes
	var count1, count2, count3 int32

	w1 := WriterFunc(func(ctx context.Context, format string, data []byte) error {
		atomic.AddInt32(&count1, 1)
		return nil
	})

	w2 := WriterFunc(func(ctx context.Context, format string, data []byte) error {
		atomic.AddInt32(&count2, 1)
		return nil
	})

	w3 := WriterFunc(func(ctx context.Context, format string, data []byte) error {
		atomic.AddInt32(&count3, 1)
		return nil
	})

	mw := NewMultiWriter(w1, w2, w3)
	ctx := context.Background()

	// Write concurrently
	var wg sync.WaitGroup
	numGoroutines := 10
	numWrites := 10

	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range numWrites {
				if err := mw.Write(ctx, FormatText, []byte("data")); err != nil {
					t.Errorf("concurrent write failed: %v", err)
				}
			}
		}()
	}

	wg.Wait()

	// Verify all writers received all writes
	expectedCount := int32(numGoroutines * numWrites)
	if atomic.LoadInt32(&count1) != expectedCount {
		t.Errorf("w1 count = %d, want %d", count1, expectedCount)
	}
	if atomic.LoadInt32(&count2) != expectedCount {
		t.Errorf("w2 count = %d, want %d", count2, expectedCount)
	}
	if atomic.LoadInt32(&count3) != expectedCount {
		t.Errorf("w3 count = %d, want %d", count3, expectedCount)
	}
}

func TestMultiWriterContextCancellation(t *testing.T) {
	w1 := &mockWriter{name: "w1"}
	w2 := &mockWriter{name: "w2"}

	mw := NewMultiWriter(w1, w2)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := mw.Write(ctx, FormatText, []byte("test"))
	if err == nil {
		t.Error("expected context cancellation error")
	}

	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("error should mention context cancellation, got: %v", err)
	}

	// Writers should not have received any data
	if w1.String() != "" || w2.String() != "" {
		t.Error("no data should be written after cancellation")
	}
}

func TestMultiWriteError(t *testing.T) {
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")

	// Single error
	mwErr := &MultiWriteError{Errors: []error{err1}}
	if mwErr.Error() != "error 1" {
		t.Errorf("single error message = %q, want %q", mwErr.Error(), "error 1")
	}

	// Multiple errors
	mwErr = &MultiWriteError{Errors: []error{err1, err2}}
	expected := "multiple write errors (2 errors)"
	if mwErr.Error() != expected {
		t.Errorf("multiple errors message = %q, want %q", mwErr.Error(), expected)
	}

	// Test Unwrap
	unwrapped := mwErr.Unwrap()
	if len(unwrapped) != 2 {
		t.Errorf("Unwrap() returned %d errors, want 2", len(unwrapped))
	}

	// Test errors.Is with wrapped errors
	if !errors.Is(mwErr, err1) {
		t.Error("errors.Is should find err1")
	}
	if !errors.Is(mwErr, err2) {
		t.Error("errors.Is should find err2")
	}
}
