package output

import (
	"strings"
	"sync"
	"testing"
)

// TestCollapsibleValue_ConcurrentDetails verifies that Details() is safe to call
// concurrently on a shared value. Regression test for T-1233: Details() lazily
// wrote memoryProcessor, processedDetails, and detailsProcessed without
// synchronization, racing when a shared document/value is rendered concurrently.
func TestCollapsibleValue_ConcurrentDetails(t *testing.T) {
	// Use details large enough to trigger the memory-optimized processing path
	// (>1KB string), exercising the memoryProcessor lazy init as well.
	largeDetails := strings.Repeat("detail-content ", 200)
	cv := NewCollapsibleValue("summary", largeDetails, WithMaxLength(0))

	const goroutines = 64
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			_ = cv.Details()
		}()
	}
	wg.Wait()

	if got, ok := cv.Details().(string); !ok || got != largeDetails {
		t.Errorf("Details() = %v, want stable %q", cv.Details(), largeDetails)
	}
}

// TestCollapsibleValue_ConcurrentFormatHint verifies that FormatHint() is safe to
// call concurrently on a shared value. Regression test for T-1233: FormatHint()
// lazily allocated and wrote hintsAccessed without synchronization.
func TestCollapsibleValue_ConcurrentFormatHint(t *testing.T) {
	cv := NewCollapsibleValue("summary", "details",
		WithFormatHint(FormatJSON, map[string]any{"key": "value"}))

	const goroutines = 64
	formats := []string{FormatJSON, FormatYAML, FormatTable, FormatHTML, FormatCSV, FormatMarkdown}
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range goroutines {
		format := formats[i%len(formats)]
		go func() {
			defer wg.Done()
			_ = cv.FormatHint(format)
		}()
	}
	wg.Wait()
}

// TestCollapsibleValue_ConcurrentRender verifies a shared value rendered through the
// render path (Details() + FormatHint()) concurrently does not race. Regression test
// for T-1233 reproducing the JSON formatValueForJSON access pattern.
func TestCollapsibleValue_ConcurrentRender(t *testing.T) {
	cv := NewCollapsibleValue("summary", strings.Repeat("x", 2000),
		WithFormatHint(FormatJSON, map[string]any{"key": "value"}))

	const goroutines = 64
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			_ = cv.Summary()
			_ = cv.Details()
			_ = cv.IsExpanded()
			_ = cv.FormatHint(FormatJSON)
		}()
	}
	wg.Wait()
}
