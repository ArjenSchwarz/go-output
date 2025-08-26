package output

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestTextProgress_ThreadSafety(t *testing.T) {
	var buf bytes.Buffer
	progress := NewProgress(
		WithProgressWriter(&buf),
		WithUpdateInterval(0),
	)

	progress.SetTotal(100)

	// Test concurrent operations
	done := make(chan bool, 3)

	go func() {
		for range 10 {
			progress.Increment(1)
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	go func() {
		for range 5 {
			progress.SetStatus("working")
			time.Sleep(2 * time.Millisecond)
		}
		done <- true
	}()

	go func() {
		for i := range 3 {
			progress.SetCurrent(i * 10)
			time.Sleep(3 * time.Millisecond)
		}
		done <- true
	}()

	// Wait for all goroutines
	<-done
	<-done
	<-done

	progress.Complete()

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}

	// Should not panic and should complete successfully
	output := buf.String()
	if !strings.Contains(output, "✓ Complete") {
		t.Error("concurrent operations should still result in completion")
	}
}
