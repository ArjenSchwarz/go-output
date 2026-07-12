package output

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

// TestFileWriterSetExtensionConcurrentWithWrite is a regression test for T-1629
// (FileWriter extension map races with writes).
//
// Bug: FileWriter.Write called generateFilename, which reads fw.extensions,
// before acquiring fw.mu, while SetExtension mutates the same map under fw.mu.
// A goroutine changing extensions while another goroutine writes could hit a
// Go map read/write data race and crash with concurrent map access.
//
// Expected: SetExtension and Write can run concurrently on the same FileWriter
// without a data race.
// Actual (before fix): the race detector reports a data race on fw.extensions,
// and the runtime could throw "concurrent map read and map write".
//
// Run with: go test -race -run TestFileWriterSetExtensionConcurrentWithWrite ./...
func TestFileWriterSetExtensionConcurrentWithWrite(t *testing.T) {
	fw, err := NewFileWriter(t.TempDir(), "race-{format}.{ext}")
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	ctx := context.Background()
	const writeIterations = 100

	var wg sync.WaitGroup
	wg.Add(2)
	done := make(chan struct{})

	// Mutate the extension map for the whole duration of the writer goroutine
	// to guarantee the map writes overlap with the map reads in Write.
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-done:
				return
			default:
				fw.SetExtension(FormatJSON, fmt.Sprintf("json%d", i%3))
			}
		}
	}()

	go func() {
		defer wg.Done()
		defer close(done)
		for range writeIterations {
			if err := fw.Write(ctx, FormatJSON, []byte(`{"ok":true}`)); err != nil {
				t.Errorf("Write failed: %v", err)
				return
			}
		}
	}()

	wg.Wait()
}
