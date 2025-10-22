package output

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestFileWriterConcurrentAppends(t *testing.T) {
	t.Parallel()
	skipIfNotIntegration(t)

	tempDir := t.TempDir()
	fw, err := NewFileWriterWithOptions(tempDir, "test-{format}.{ext}", WithAppendMode())
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	ctx := context.Background()
	testFile := filepath.Join(fw.dir, "concurrent.txt")

	// Create initial file
	if err := os.WriteFile(testFile, []byte("initial\n"), 0644); err != nil {
		t.Fatalf("failed to create initial file: %v", err)
	}

	// Launch multiple goroutines to append concurrently
	numGoroutines := 10
	var wg sync.WaitGroup
	var successCount atomic.Int32
	var errorCount atomic.Int32

	for i := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			data := fmt.Appendf(nil, "append-%d\n", id)
			if err := fw.appendByteLevel(ctx, testFile, data); err != nil {
				errorCount.Add(1)
				t.Logf("error appending from goroutine %d: %v", id, err)
				return
			}
			successCount.Add(1)
		}(i)
	}

	wg.Wait()

	// Verify all appends succeeded
	if errorCount.Load() != 0 {
		t.Errorf("expected no errors, but got %d errors", errorCount.Load())
	}

	if successCount.Load() != int32(numGoroutines) {
		t.Errorf("expected %d successful appends, got %d", numGoroutines, successCount.Load())
	}

	// Verify file contains all appended data
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	contentStr := string(content)

	// Check that initial data is present
	if !strings.Contains(contentStr, "initial") {
		t.Error("initial data not found in file")
	}

	// Check that all appended data is present
	for i := 0; i < numGoroutines; i++ {
		expectedLine := fmt.Sprintf("append-%d", i)
		if !strings.Contains(contentStr, expectedLine) {
			t.Errorf("expected append data %q not found in file", expectedLine)
		}
	}

	// Verify we have the expected number of lines (initial + 10 appends)
	lines := strings.Split(strings.TrimSpace(contentStr), "\n")
	expectedLines := numGoroutines + 1
	if len(lines) != expectedLines {
		t.Errorf("expected %d lines, got %d", expectedLines, len(lines))
	}
}

func TestFileWriterConcurrentWriteAndAppend(t *testing.T) {
	t.Parallel()
	skipIfNotIntegration(t)

	tempDir := t.TempDir()
	fw, err := NewFileWriterWithOptions(tempDir, "test-{format}.{ext}", WithAppendMode())
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	ctx := context.Background()

	tests := map[string]struct {
		numConcurrentOps int
		numAppendsPerOp  int
	}{
		"light concurrent load": {
			numConcurrentOps: 5,
			numAppendsPerOp:  2,
		},
		"heavy concurrent load": {
			numConcurrentOps: 20,
			numAppendsPerOp:  5,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			filename := fmt.Sprintf("concurrent-%s.txt", strings.ReplaceAll(name, " ", "-"))
			filepath := filepath.Join(fw.dir, filename)

			// Create initial file
			if err := os.WriteFile(filepath, []byte("start\n"), 0644); err != nil {
				t.Fatalf("failed to create initial file: %v", err)
			}

			var wg sync.WaitGroup
			var errorCount atomic.Int32

			// Launch concurrent operations
			for op := range tc.numConcurrentOps {
				wg.Add(1)
				go func(opID int) {
					defer wg.Done()

					for appID := range tc.numAppendsPerOp {
						data := fmt.Appendf(nil, "op-%d-append-%d\n", opID, appID)
						if err := fw.appendByteLevel(ctx, filepath, data); err != nil {
							errorCount.Add(1)
							return
						}
					}
				}(op)
			}

			wg.Wait()

			if errorCount.Load() != 0 {
				t.Errorf("expected no errors, got %d", errorCount.Load())
			}

			// Verify file exists and has expected content
			content, err := os.ReadFile(filepath)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			contentStr := string(content)
			lines := strings.Split(strings.TrimSpace(contentStr), "\n")
			expectedLineCount := 1 + tc.numConcurrentOps*tc.numAppendsPerOp

			if len(lines) != expectedLineCount {
				t.Errorf("expected %d lines, got %d", expectedLineCount, len(lines))
			}

			// Verify start line is present
			if !strings.HasPrefix(contentStr, "start") {
				t.Error("start line not found")
			}
		})
	}
}
