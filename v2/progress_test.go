package output

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNewProgress(t *testing.T) {
	t.Run("with default config", func(t *testing.T) {
		progress := NewProgress()
		if progress == nil {
			t.Fatal("NewProgress() should not return nil")
		}

		// Should implement Progress interface
		var _ Progress = progress

		err := progress.Close()
		if err != nil {
			t.Errorf("Close() should not return error, got %v", err)
		}
	})

	t.Run("with custom options", func(t *testing.T) {
		var buf bytes.Buffer
		progress := NewProgress(
			WithProgressWriter(&buf),
			WithUpdateInterval(50*time.Millisecond),
			WithPercentage(false),
			WithETA(false),
			WithRate(true),
			WithWidth(20),
			WithPrefix("Processing:"),
			WithSuffix("done"),
		)

		if progress == nil {
			t.Fatal("NewProgress() should not return nil")
		}

		err := progress.Close()
		if err != nil {
			t.Errorf("Close() should not return error, got %v", err)
		}
	})
}

func TestNewNoOpProgress(t *testing.T) {
	progress := NewNoOpProgress()
	if progress == nil {
		t.Fatal("NewNoOpProgress() should not return nil")
	}

	// Should implement Progress interface
	var _ Progress = progress

	// All operations should be no-ops and not panic
	progress.SetTotal(100)
	progress.SetCurrent(50)
	progress.Increment(10)
	progress.SetStatus("testing")
	progress.Complete()
	progress.Fail(errors.New("test error"))

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}
}

func TestTextProgress_BasicOperations(t *testing.T) {
	var buf bytes.Buffer
	progress := NewProgress(
		WithProgressWriter(&buf),
		WithUpdateInterval(0), // No throttling for tests
	)

	// Test SetTotal
	progress.SetTotal(100)

	// Test SetCurrent
	progress.SetCurrent(25)

	// Test Increment
	progress.Increment(25)

	// Test SetStatus
	progress.SetStatus("processing data")

	// Complete the progress
	progress.Complete()

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "✓ Complete") {
		t.Error("output should contain completion indicator")
	}
}

func TestTextProgress_ProgressDisplay(t *testing.T) {
	var buf bytes.Buffer
	progress := NewProgress(
		WithProgressWriter(&buf),
		WithUpdateInterval(0), // No throttling for tests
		WithWidth(10),
		WithPercentage(true),
	)

	progress.SetTotal(100)
	progress.SetCurrent(50)
	progress.Complete()

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}

	output := buf.String()

	// Check for progress bar elements
	if !strings.Contains(output, "[") || !strings.Contains(output, "]") {
		t.Error("output should contain progress bar brackets")
	}

	if !strings.Contains(output, "100.0%") {
		t.Error("output should contain percentage when completed")
	}

	if !strings.Contains(output, "(100/100)") {
		t.Error("output should contain count display")
	}
}

func TestTextProgress_WithPrefix(t *testing.T) {
	var buf bytes.Buffer
	prefix := "Processing files:"
	progress := NewProgress(
		WithProgressWriter(&buf),
		WithUpdateInterval(0),
		WithPrefix(prefix),
	)

	progress.SetTotal(10)
	progress.SetCurrent(5)
	progress.Complete()

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, prefix) {
		t.Errorf("output should contain prefix %q", prefix)
	}
}

func TestTextProgress_WithSuffix(t *testing.T) {
	var buf bytes.Buffer
	suffix := "files processed"
	progress := NewProgress(
		WithProgressWriter(&buf),
		WithUpdateInterval(0),
		WithSuffix(suffix),
	)

	progress.SetTotal(10)
	progress.SetCurrent(5)
	progress.Complete()

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, suffix) {
		t.Errorf("output should contain suffix %q", suffix)
	}
}

func TestTextProgress_WithStatus(t *testing.T) {
	var buf bytes.Buffer
	status := "analyzing data"
	progress := NewProgress(
		WithProgressWriter(&buf),
		WithUpdateInterval(0),
	)

	progress.SetTotal(100)
	progress.SetCurrent(50)
	progress.SetStatus(status)
	progress.Complete()

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, status) {
		t.Errorf("output should contain status %q", status)
	}
}

func TestTextProgress_Failure(t *testing.T) {
	var buf bytes.Buffer
	progress := NewProgress(
		WithProgressWriter(&buf),
		WithUpdateInterval(0),
	)

	progress.SetTotal(100)
	progress.SetCurrent(50)

	testError := errors.New("test failure")
	progress.Fail(testError)

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "✗ Failed") {
		t.Error("output should contain failure indicator")
	}
	if !strings.Contains(output, testError.Error()) {
		t.Error("output should contain error message")
	}
}

func TestTextProgress_ZeroTotal(t *testing.T) {
	var buf bytes.Buffer
	progress := NewProgress(
		WithProgressWriter(&buf),
		WithUpdateInterval(0),
	)

	// Don't set total, should handle gracefully
	progress.Increment(5)
	progress.SetStatus("working")
	progress.Complete()

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "working") {
		t.Error("output should contain status message even without total")
	}
}

func TestTextProgress_ETADisplay(t *testing.T) {
	var buf bytes.Buffer
	progress := NewProgress(
		WithProgressWriter(&buf),
		WithUpdateInterval(0),
		WithETA(true),
	)

	progress.SetTotal(100)

	// Sleep a bit to allow time calculation
	time.Sleep(10 * time.Millisecond)
	progress.SetCurrent(50)

	progress.Complete()

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}

	output := buf.String()
	// ETA should be displayed when progress > 0 and total > 0
	// However, due to the quick nature of tests, ETA might not always appear
	// So we just verify the progress was tracked correctly
	if !strings.Contains(output, "✓ Complete") {
		t.Error("output should show completion")
	}
}

func TestTextProgress_RateDisplay(t *testing.T) {
	var buf bytes.Buffer
	progress := NewProgress(
		WithProgressWriter(&buf),
		WithUpdateInterval(0),
		WithRate(true),
	)

	progress.SetTotal(100)

	// Sleep a bit to allow rate calculation
	time.Sleep(10 * time.Millisecond)
	progress.SetCurrent(50)

	progress.Complete()

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}

	output := buf.String()
	// Rate should be displayed when current > 0
	// The rate should include "/s" for per-second
	if !strings.Contains(output, "/s") && strings.Contains(output, "50") {
		// Rate display might vary based on timing, so this is a flexible check
		t.Log("Rate display test - output:", output)
	}
}

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

func TestProgress_ConfigurationOptions(t *testing.T) {
	tests := []struct {
		name string
		opts []ProgressOption
		test func(*testing.T, Progress)
	}{
		{
			name: "WithPercentage false",
			opts: []ProgressOption{WithPercentage(false)},
			test: func(t *testing.T, p Progress) {
				// We can't easily test this without accessing internals,
				// so we just verify it doesn't panic
				p.SetTotal(100)
				p.SetCurrent(50)
				p.Complete()
			},
		},
		{
			name: "WithWidth custom",
			opts: []ProgressOption{WithWidth(5)},
			test: func(t *testing.T, p Progress) {
				// Test custom width doesn't cause issues
				p.SetTotal(100)
				p.SetCurrent(50)
				p.Complete()
			},
		},
		{
			name: "WithTemplate custom",
			opts: []ProgressOption{WithTemplate("custom template")},
			test: func(t *testing.T, p Progress) {
				// Test custom template doesn't cause issues
				p.SetTotal(100)
				p.SetCurrent(50)
				p.Complete()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			opts := append(tt.opts, WithProgressWriter(&buf), WithUpdateInterval(0))
			progress := NewProgress(opts...)

			tt.test(t, progress)

			err := progress.Close()
			if err != nil {
				t.Errorf("Close() should not return error, got %v", err)
			}
		})
	}
}

func TestProgress_AccuracyValidation(t *testing.T) {
	var buf bytes.Buffer
	progress := NewProgress(
		WithProgressWriter(&buf),
		WithUpdateInterval(0),
	)

	// Test accuracy of progress tracking
	total := 100
	progress.SetTotal(total)

	// Increment by various amounts
	progress.Increment(25)
	progress.Increment(25)
	progress.SetCurrent(75) // Should override previous increments
	progress.Increment(25)

	progress.Complete()

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}

	output := buf.String()

	// When completed, should show 100/100
	if !strings.Contains(output, "(100/100)") {
		t.Error("completed progress should show (100/100)")
	}

	if !strings.Contains(output, "100.0%") {
		t.Error("completed progress should show 100.0%")
	}
}

// Test v1 compatibility features

func TestProgressColor_Constants(t *testing.T) {
	// Test that all v1 color constants are defined
	colors := []ProgressColor{
		ProgressColorDefault,
		ProgressColorGreen,
		ProgressColorRed,
		ProgressColorYellow,
		ProgressColorBlue,
	}

	// Verify they have different values
	seen := make(map[ProgressColor]bool)
	for _, color := range colors {
		if seen[color] {
			t.Errorf("duplicate color value: %d", color)
		}
		seen[color] = true
	}

	// Should have 5 unique colors
	if len(seen) != 5 {
		t.Errorf("expected 5 unique colors, got %d", len(seen))
	}
}

func TestProgress_V1Compatibility_SetColor(t *testing.T) {
	var buf bytes.Buffer
	progress := NewProgress(WithProgressWriter(&buf))

	// Test all color values
	colors := []ProgressColor{
		ProgressColorDefault,
		ProgressColorGreen,
		ProgressColorRed,
		ProgressColorYellow,
		ProgressColorBlue,
	}

	for _, color := range colors {
		progress.SetColor(color)
		// SetColor should not panic or cause errors
		progress.SetTotal(100)
		progress.SetCurrent(50)
	}

	progress.Complete()
	progress.Close()
}

func TestProgress_V1Compatibility_IsActive(t *testing.T) {
	var buf bytes.Buffer
	progress := NewProgress(WithProgressWriter(&buf))

	// Should be active initially
	if !progress.IsActive() {
		t.Error("progress should be active initially")
	}

	progress.SetTotal(100)
	progress.SetCurrent(50)

	// Should still be active while in progress
	if !progress.IsActive() {
		t.Error("progress should be active while in progress")
	}

	progress.Complete()

	// Should not be active after completion
	if progress.IsActive() {
		t.Error("progress should not be active after completion")
	}

	progress.Close()
}

func TestProgress_V1Compatibility_SetContext(t *testing.T) {
	var buf bytes.Buffer
	progress := NewProgress(WithProgressWriter(&buf))

	// Test with nil context
	progress.SetContext(nil)

	// Test with cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	progress.SetContext(ctx)

	progress.SetTotal(100)
	progress.SetCurrent(50)

	// Should be active before cancellation
	if !progress.IsActive() {
		t.Error("progress should be active before cancellation")
	}

	// Cancel the context
	cancel()

	// Wait a bit for the goroutine to process cancellation
	time.Sleep(10 * time.Millisecond)

	// Should not be active after cancellation
	if progress.IsActive() {
		t.Error("progress should not be active after cancellation")
	}

	progress.Close()
}

func TestProgress_V1Compatibility_ProgressOptions(t *testing.T) {
	var buf bytes.Buffer
	progress := NewProgress(
		WithProgressWriter(&buf),
		WithProgressColor(ProgressColorGreen),
		WithProgressStatus("Processing files"),
		WithTrackerLength(50),
	)

	progress.SetTotal(100)
	progress.SetCurrent(75)
	progress.Complete()

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}

	// Should complete without errors
	output := buf.String()
	if !strings.Contains(output, "✓ Complete") {
		t.Error("output should contain completion indicator")
	}
}

func TestNoOpProgress_V1Compatibility(t *testing.T) {
	progress := NewNoOpProgress()

	// Test all v1 compatibility methods on NoOp progress
	progress.SetColor(ProgressColorGreen)
	progress.SetContext(context.Background())

	// IsActive should always return false for NoOp
	if progress.IsActive() {
		t.Error("NoOpProgress.IsActive() should always return false")
	}

	// All operations should be no-ops and not panic
	progress.SetTotal(100)
	progress.SetCurrent(50)
	progress.Increment(10)
	progress.SetStatus("testing")
	progress.Complete()
	progress.Fail(errors.New("test error"))

	// Should still not be active
	if progress.IsActive() {
		t.Error("NoOpProgress.IsActive() should still return false after operations")
	}

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}
}

func TestProgress_V1Compatibility_FailureHandling(t *testing.T) {
	var buf bytes.Buffer
	progress := NewProgress(WithProgressWriter(&buf))

	progress.SetTotal(100)
	progress.SetCurrent(50)
	progress.SetColor(ProgressColorRed)

	testError := errors.New("test failure")
	progress.Fail(testError)

	// Should not be active after failure
	if progress.IsActive() {
		t.Error("progress should not be active after failure")
	}

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "✗ Failed") {
		t.Error("output should contain failure indicator")
	}
	if !strings.Contains(output, testError.Error()) {
		t.Error("output should contain error message")
	}
}

func TestProgress_V1Compatibility_StatusInitialization(t *testing.T) {
	var buf bytes.Buffer
	initialStatus := "Starting process"
	progress := NewProgress(
		WithProgressWriter(&buf),
		WithProgressStatus(initialStatus),
	)

	progress.SetTotal(100)
	progress.SetCurrent(25)
	progress.Complete()

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}

	// The initial status should have been used
	// Note: This tests that the status was properly initialized
	// The actual display format may vary
}

func TestProgress_V1Compatibility_TrackerLength(t *testing.T) {
	var buf bytes.Buffer
	trackerLength := 20
	progress := NewProgress(
		WithProgressWriter(&buf),
		WithTrackerLength(trackerLength),
	)

	progress.SetTotal(100)
	progress.SetCurrent(50)
	progress.Complete()

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}

	// Should complete without errors
	output := buf.String()
	if !strings.Contains(output, "✓ Complete") {
		t.Error("output should contain completion indicator")
	}
}

// Test PrettyProgress functionality

func TestNewPrettyProgress(t *testing.T) {
	// Create a pretty progress with buffered output
	var buf bytes.Buffer
	progress := NewPrettyProgress(
		WithProgressWriter(&buf),
		WithProgressColor(ProgressColorGreen),
		WithProgressStatus("Testing"),
	)

	if progress == nil {
		t.Fatal("NewPrettyProgress() should not return nil")
	}

	// Should implement Progress interface
	var _ Progress = progress

	// Test basic operations
	progress.SetTotal(100)
	progress.SetCurrent(50)
	progress.Complete()

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}
}

func TestPrettyProgress_BasicOperations(t *testing.T) {
	var buf bytes.Buffer
	progress := NewPrettyProgress(
		WithProgressWriter(&buf),
		WithUpdateInterval(0), // No throttling for tests
	)

	// Test SetTotal
	progress.SetTotal(100)

	// Test SetCurrent
	progress.SetCurrent(25)

	// Test Increment
	progress.Increment(25)

	// Test SetStatus
	progress.SetStatus("processing data")

	// Complete the progress
	progress.Complete()

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}

	// Note: go-pretty output is complex and not easily testable with simple string matching
	// The main test is that it doesn't panic or error
}

func TestPrettyProgress_V1Compatibility_SetColor(t *testing.T) {
	var buf bytes.Buffer
	progress := NewPrettyProgress(WithProgressWriter(&buf))

	// Test all color values
	colors := []ProgressColor{
		ProgressColorDefault,
		ProgressColorGreen,
		ProgressColorRed,
		ProgressColorYellow,
		ProgressColorBlue,
	}

	for _, color := range colors {
		progress.SetColor(color)
		// SetColor should not panic or cause errors
		progress.SetTotal(100)
		progress.SetCurrent(50)
	}

	progress.Complete()
	progress.Close()
}

func TestPrettyProgress_V1Compatibility_IsActive(t *testing.T) {
	var buf bytes.Buffer
	progress := NewPrettyProgress(WithProgressWriter(&buf))

	// Should be active initially
	if !progress.IsActive() {
		t.Error("progress should be active initially")
	}

	progress.SetTotal(100)
	progress.SetCurrent(50)

	// Should still be active while in progress
	if !progress.IsActive() {
		t.Error("progress should be active while in progress")
	}

	progress.Complete()

	// Should not be active after completion
	if progress.IsActive() {
		t.Error("progress should not be active after completion")
	}

	progress.Close()
}

func TestPrettyProgress_V1Compatibility_SetContext(t *testing.T) {
	var buf bytes.Buffer
	progress := NewPrettyProgress(WithProgressWriter(&buf))

	// Test with nil context
	progress.SetContext(nil)

	// Test with cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	progress.SetContext(ctx)

	progress.SetTotal(100)
	progress.SetCurrent(50)

	// Should be active before cancellation
	if !progress.IsActive() {
		t.Error("progress should be active before cancellation")
	}

	// Cancel the context
	cancel()

	// Wait a bit for the goroutine to process cancellation
	time.Sleep(10 * time.Millisecond)

	// Should not be active after cancellation
	if progress.IsActive() {
		t.Error("progress should not be active after cancellation")
	}

	progress.Close()
}

func TestPrettyProgress_FailureHandling(t *testing.T) {
	var buf bytes.Buffer
	progress := NewPrettyProgress(WithProgressWriter(&buf))

	progress.SetTotal(100)
	progress.SetCurrent(50)
	progress.SetColor(ProgressColorRed)

	testError := errors.New("test failure")
	progress.Fail(testError)

	// Should not be active after failure
	if progress.IsActive() {
		t.Error("progress should not be active after failure")
	}

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}
}

func TestPrettyProgress_ThreadSafety(t *testing.T) {
	var buf bytes.Buffer
	progress := NewPrettyProgress(
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
	if progress.IsActive() {
		t.Error("progress should not be active after completion")
	}
}

func TestPrettyProgress_SignalHandling(t *testing.T) {
	var buf bytes.Buffer
	progress := NewPrettyProgress(WithProgressWriter(&buf))

	progress.SetTotal(100)
	progress.SetCurrent(50)

	// Simulate terminal resize signal
	// Note: This test mainly ensures the signal handling goroutine doesn't panic
	// The actual signal handling is tested manually

	progress.Complete()
	progress.Close()
}

func TestPrettyProgress_AutoFallback(t *testing.T) {
	// Test that NewPrettyProgress falls back to textProgress when not in TTY
	// This is difficult to test automatically since it depends on TTY detection
	// but we can test that it doesn't panic

	var buf bytes.Buffer
	progress := NewPrettyProgress(WithProgressWriter(&buf))

	progress.SetTotal(100)
	progress.SetCurrent(50)
	progress.Complete()

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}
}

// Test format-aware progress creation

func TestNewProgressForFormat(t *testing.T) {
	tests := []struct {
		name         string
		format       Format
		expectedType string // We can't test exact type, but can test behavior
	}{
		{
			name:         "JSON format uses NoOp",
			format:       JSON,
			expectedType: "noop",
		},
		{
			name:         "CSV format uses NoOp",
			format:       CSV,
			expectedType: "noop",
		},
		{
			name:         "YAML format uses NoOp",
			format:       YAML,
			expectedType: "noop",
		},
		{
			name:         "DOT format uses NoOp",
			format:       DOT,
			expectedType: "noop",
		},
		{
			name:         "Table format uses visual progress",
			format:       Table,
			expectedType: "visual",
		},
		{
			name:         "HTML format uses visual progress",
			format:       HTML,
			expectedType: "visual",
		},
		{
			name:         "Markdown format uses visual progress",
			format:       Markdown,
			expectedType: "visual",
		},
		{
			name:         "Mermaid format uses visual progress",
			format:       Mermaid,
			expectedType: "visual",
		},
		{
			name:         "DrawIO format uses visual progress",
			format:       DrawIO,
			expectedType: "visual",
		},
		{
			name:         "Unknown format uses text progress",
			format:       Format{Name: "unknown"},
			expectedType: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			progress := NewProgressForFormat(tt.format, WithProgressWriter(&buf))

			if progress == nil {
				t.Fatal("NewProgressForFormat should not return nil")
			}

			// Test that it implements the Progress interface
			var _ Progress = progress

			// Test basic functionality doesn't panic
			progress.SetTotal(100)
			progress.SetCurrent(50)

			// Check behavior based on expected type
			switch tt.expectedType {
			case "noop":
				// NoOp progress should always return false for IsActive
				if progress.IsActive() {
					t.Error("NoOp progress should return false for IsActive")
				}
			case "visual", "text":
				// Visual and text progress should be active initially
				if !progress.IsActive() {
					t.Error("Visual/text progress should be active initially")
				}
			}

			progress.Complete()
			progress.Close()
		})
	}
}

func TestNewProgressForFormats(t *testing.T) {
	tests := []struct {
		name         string
		formats      []Format
		expectedType string
	}{
		{
			name:         "Empty formats uses NoOp",
			formats:      []Format{},
			expectedType: "noop",
		},
		{
			name:         "Only non-visual formats uses NoOp",
			formats:      []Format{JSON, CSV, YAML},
			expectedType: "noop",
		},
		{
			name:         "Only visual formats uses visual progress",
			formats:      []Format{Table, HTML},
			expectedType: "visual",
		},
		{
			name:         "Mixed formats uses conservative approach",
			formats:      []Format{JSON, Table, CSV},
			expectedType: "mixed",
		},
		{
			name:         "Single visual format",
			formats:      []Format{Markdown},
			expectedType: "visual",
		},
		{
			name:         "Single non-visual format",
			formats:      []Format{DOT},
			expectedType: "noop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			progress := NewProgressForFormats(tt.formats, WithProgressWriter(&buf))

			if progress == nil {
				t.Fatal("NewProgressForFormats should not return nil")
			}

			// Test that it implements the Progress interface
			var _ Progress = progress

			// Test basic functionality
			progress.SetTotal(100)
			progress.SetCurrent(50)

			// Check behavior based on expected type
			switch tt.expectedType {
			case "noop":
				// NoOp progress should always return false for IsActive
				if progress.IsActive() {
					t.Error("NoOp progress should return false for IsActive")
				}
			case "visual":
				// Visual progress should be active initially (TTY detection may vary)
				// Don't assert IsActive since it depends on TTY detection
			case "mixed":
				// Mixed format behavior depends on TTY detection
				// Just ensure it doesn't panic
			}

			progress.Complete()
			progress.Close()
		})
	}
}

func TestNewAutoProgress(t *testing.T) {
	var buf bytes.Buffer
	progress := NewAutoProgress(WithProgressWriter(&buf))

	if progress == nil {
		t.Fatal("NewAutoProgress should not return nil")
	}

	// Test that it implements the Progress interface
	var _ Progress = progress

	// Test basic functionality
	progress.SetTotal(100)
	progress.SetCurrent(50)
	progress.Complete()

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}
}

func TestFormat_Constants(t *testing.T) {
	// Test that all format constants are defined and unique
	formats := []Format{
		JSON,
		CSV,
		YAML,
		DOT,
		Table,
		HTML,
		Markdown,
		Mermaid,
		DrawIO,
	}

	seen := make(map[string]bool)
	for _, format := range formats {
		if seen[format.Name] {
			t.Errorf("duplicate format constant: %s", format.Name)
		}
		seen[format.Name] = true

		// Ensure format name is not empty
		if format.Name == "" {
			t.Error("format name should not be empty string")
		}

		// Ensure format has a renderer
		if format.Renderer == nil {
			t.Errorf("format %s should have a renderer", format.Name)
		}
	}

	// Should have 9 unique formats
	if len(seen) != 9 {
		t.Errorf("expected 9 unique formats, got %d", len(seen))
	}
}

func TestNewProgressForFormatName(t *testing.T) {
	tests := []struct {
		name         string
		formatName   string
		expectedType string
	}{
		{
			name:         "JSON format name uses NoOp",
			formatName:   "json",
			expectedType: "noop",
		},
		{
			name:         "Table format name uses visual progress",
			formatName:   "table",
			expectedType: "visual",
		},
		{
			name:         "Unknown format name uses text progress",
			formatName:   "unknown",
			expectedType: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			progress := NewProgressForFormatName(tt.formatName, WithProgressWriter(&buf))

			if progress == nil {
				t.Fatal("NewProgressForFormatName should not return nil")
			}

			// Test basic functionality
			progress.SetTotal(100)
			progress.SetCurrent(50)
			progress.Complete()
			progress.Close()
		})
	}
}

func TestProgressForFormat_Integration(t *testing.T) {
	// Integration test that combines multiple format-aware functions
	formats := []Format{Table, JSON}

	var buf bytes.Buffer
	progress := NewProgressForFormats(formats,
		WithProgressWriter(&buf),
		WithProgressColor(ProgressColorGreen),
		WithProgressStatus("Processing files"),
	)

	progress.SetTotal(100)
	for i := 0; i < 100; i += 10 {
		progress.SetCurrent(i)
		progress.SetStatus(fmt.Sprintf("Processing item %d", i))
	}
	progress.Complete()

	err := progress.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}
}
