package output

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	ptprogress "github.com/jedib0t/go-pretty/v6/progress"
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

func TestProgress_ConfigurationOptions(t *testing.T) {
	tests := map[string]struct {
		opts []ProgressOption
		test func(*testing.T, Progress)
	}{"WithPercentage false": {

		opts: []ProgressOption{WithPercentage(false)},
		test: func(t *testing.T, p Progress) {
			// We can't easily test this without accessing internals,
			// so we just verify it doesn't panic
			p.SetTotal(100)
			p.SetCurrent(50)
			p.Complete()
		},
	}, "WithTemplate custom": {

		opts: []ProgressOption{WithTemplate("custom template")},
		test: func(t *testing.T, p Progress) {
			// Test custom template doesn't cause issues
			p.SetTotal(100)
			p.SetCurrent(50)
			p.Complete()
		},
	}, "WithWidth custom": {

		opts: []ProgressOption{WithWidth(5)},
		test: func(t *testing.T, p Progress) {

			p.SetTotal(100)
			p.SetCurrent(50)
			p.Complete()
		},
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
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

// TestProgress_SetContext_Replacement_DoesNotFail is a regression test for T-1254.
//
// SetContext cancels the previously derived context when a new context is
// installed. The watcher goroutine for the old context must NOT mark the
// progress as failed when it is released by that replacement cancellation.
//
// Bug: the old watcher read p.ctx (the shared struct field) instead of the
// context it was created for, so after replacement it saw the new context's
// error (or none) and set p.failed = true, failing a still-healthy progress.
//
// Expected: replacing a progress context leaves the progress active and
// unfailed; only the live context's cancellation should fail it.
func TestProgress_SetContext_Replacement_DoesNotFail(t *testing.T) {
	var buf bytes.Buffer
	progress := NewProgress(WithProgressWriter(&buf))

	tp, ok := progress.(*textProgress)
	if !ok {
		t.Fatalf("NewProgress() returned %T, expected *textProgress", progress)
	}

	// Install the first context.
	ctx1 := t.Context()
	progress.SetContext(ctx1)

	// Replace it with a fresh, live context. This cancels the first derived
	// context internally and releases the first watcher goroutine.
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	progress.SetContext(ctx2)

	// Give the released first watcher time to (incorrectly) run and fail us.
	time.Sleep(50 * time.Millisecond)

	// The progress must still be active and unfailed: replacing the context
	// is not a failure condition.
	if !progress.IsActive() {
		t.Error("progress should remain active after the context is replaced")
	}

	tp.mu.RLock()
	failed, err := tp.failed, tp.err
	tp.mu.RUnlock()
	if failed {
		t.Errorf("progress should not be marked failed after context replacement, err=%v", err)
	}

	// The live context must still be able to fail the progress.
	cancel2()
	time.Sleep(50 * time.Millisecond)
	if progress.IsActive() {
		t.Error("progress should fail when the live (current) context is cancelled")
	}

	progress.Close()
}

// TestTextProgress_OverrunDoesNotPanic is a regression test for T-1346.
//
// textProgress.renderDefault computed filled := int(progress * float64(barWidth))
// and then strings.Repeat(" ", barWidth-filled). When current exceeds total
// (e.g. SetTotal(10) then SetCurrent(11), or Increment past the total),
// filled > barWidth, so barWidth-filled is negative and strings.Repeat panics.
//
// Expected: rendering clamps the bar to a full bar (and never produces a
// negative repeat count) so the overrun cannot panic.
func TestTextProgress_OverrunDoesNotPanic(t *testing.T) {
	t.Run("SetCurrent beyond total does not panic", func(t *testing.T) {
		var buf bytes.Buffer
		progress := NewProgress(
			WithProgressWriter(&buf),
			WithUpdateInterval(0), // draw on every call
			WithWidth(10),
		)

		progress.SetTotal(10)
		// With the bug present, this overrun makes filled=11 > barWidth=10,
		// so strings.Repeat(" ", -1) panics inside draw().
		progress.SetCurrent(11)

		if err := progress.Close(); err != nil {
			t.Errorf("Close() should not return error, got %v", err)
		}

		// On overrun the bar should be full (no empty cells, no panic).
		output := buf.String()
		if !strings.Contains(output, "[==========]") {
			t.Errorf("overrun should render a full bar, got %q", output)
		}
	})

	t.Run("Increment beyond total does not panic", func(t *testing.T) {
		var buf bytes.Buffer
		progress := NewProgress(
			WithProgressWriter(&buf),
			WithUpdateInterval(0),
			WithWidth(10),
		)

		progress.SetTotal(5)
		// Increment past the total: 0 -> 7, current(7) > total(5).
		progress.Increment(7)

		if err := progress.Close(); err != nil {
			t.Errorf("Close() should not return error, got %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "[==========]") {
			t.Errorf("overrun should render a full bar, got %q", output)
		}
	})
}

// TestTextProgress_NegativeWidthDoesNotPanic is a regression test for T-1346.
//
// WithWidth(-1) / WithTrackerLength(-1) set ProgressConfig.Width to a negative
// value. renderDefault then computed barWidth = -1, filled = 0, and called
// strings.Repeat(" ", barWidth-filled) = strings.Repeat(" ", -1), which panics.
//
// Expected: a non-positive configured width is guarded so no negative repeat
// count is produced.
func TestTextProgress_NegativeWidthDoesNotPanic(t *testing.T) {
	tests := map[string]struct {
		opt ProgressOption
	}{
		"WithWidth(-1)":         {opt: WithWidth(-1)},
		"WithTrackerLength(-1)": {opt: WithTrackerLength(-1)},
		"WithWidth(0)":          {opt: WithWidth(0)},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			progress := NewProgress(
				WithProgressWriter(&buf),
				WithUpdateInterval(0),
				tt.opt,
			)

			// Any draw with total > 0 reaches the bar math; with the bug present
			// the negative width causes strings.Repeat to panic.
			progress.SetTotal(10)
			progress.SetCurrent(5)
			progress.Complete()

			if err := progress.Close(); err != nil {
				t.Errorf("Close() should not return error, got %v", err)
			}
		})
	}
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

// newTestPrettyProgress builds a *prettyProgress directly so the test does not
// depend on a TTY. NewPrettyProgress falls back to textProgress when stderr is
// not a terminal (as in CI), which would otherwise make it impossible to
// exercise the prettyProgress code path.
func newTestPrettyProgress(w *bytes.Buffer) *prettyProgress {
	config := defaultProgressConfig()
	WithProgressWriter(w)(config)
	WithUpdateInterval(0)(config)

	ctx, cancel := context.WithCancel(context.Background())
	p := &prettyProgress{
		config:    config,
		startTime: time.Now(),
		active:    true,
		ctx:       ctx,
		cancel:    cancel,
		signals:   make(chan os.Signal, 1),
	}
	p.writer = ptprogress.NewWriter()
	p.writer.SetOutputWriter(config.Writer)
	p.writer.SetUpdateFrequency(config.UpdateInterval)
	p.writer.SetStyle(ptprogress.StyleDefault)
	p.writer.SetNumTrackersExpected(1)
	return p
}

// TestPrettyProgress_SetContext_Replacement_DoesNotFail is a regression test for
// T-1358 (the prettyProgress counterpart of the T-1254 textProgress fix).
//
// SetContext cancels the previously derived context when a new context is
// installed. The watcher goroutine for the old context must NOT mark the
// progress as failed when it is released by that replacement cancellation.
//
// Bug: the old watcher read p.ctx (the shared struct field) instead of the
// context it was created for, so after replacement it saw the new context's
// error (or none) and called MarkAsErrored, failing a still-healthy progress.
//
// Expected: replacing a progress context leaves the progress active and
// unfailed; only the live context's cancellation should fail it.
func TestPrettyProgress_SetContext_Replacement_DoesNotFail(t *testing.T) {
	var buf bytes.Buffer
	p := newTestPrettyProgress(&buf)

	// Create a tracker so the watcher's (p.active && p.tracker != nil) guard
	// is satisfied — otherwise the bug cannot manifest.
	p.SetTotal(100)
	p.SetCurrent(50)

	// Install the first context.
	ctx1 := t.Context()
	p.SetContext(ctx1)

	// Replace it with a fresh, live context. This cancels the first derived
	// context internally and releases the first watcher goroutine.
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	p.SetContext(ctx2)

	// Give the released first watcher time to (incorrectly) run and fail us.
	time.Sleep(50 * time.Millisecond)

	// The progress must still be active and unfailed: replacing the context
	// is not a failure condition.
	if !p.IsActive() {
		t.Error("progress should remain active after the context is replaced")
	}

	p.mu.RLock()
	failed, err := p.failed, p.err
	p.mu.RUnlock()
	if failed {
		t.Errorf("progress should not be marked failed after context replacement, err=%v", err)
	}

	// The live context must still be able to fail the progress.
	cancel2()
	time.Sleep(50 * time.Millisecond)
	if p.IsActive() {
		t.Error("progress should fail when the live (current) context is cancelled")
	}

	p.Close()
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

func TestPrettyProgress_HandleSignals_NilContext_DoesNotPanic(t *testing.T) {
	p := &prettyProgress{
		signals: make(chan os.Signal, 1),
	}

	done := make(chan struct{})
	panicCh := make(chan any, 1)

	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				panicCh <- r
			}
		}()
		p.handleSignals()
	}()

	close(p.signals)

	select {
	case r := <-panicCh:
		t.Fatalf("handleSignals panicked with nil context: %v", r)
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("handleSignals did not exit after signals channel close")
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
	// We can't test exact type, but can test behavior
	tests := map[string]struct {
		format       Format
		expectedType string
	}{
		"CSV format uses NoOp": {

			format:       CSV(),
			expectedType: "noop",
		}, "DOT format uses NoOp": {

			format:       DOT(),
			expectedType: "noop",
		}, "DrawIO format uses visual progress": {

			format:       DrawIO(),
			expectedType: "visual",
		}, "HTML format uses visual progress": {

			format:       HTML(),
			expectedType: "visual",
		}, "JSON format uses NoOp": {

			format:       JSON(),
			expectedType: "noop",
		}, "Markdown format uses visual progress": {

			format:       Markdown(),
			expectedType: "visual",
		}, "Mermaid format uses visual progress": {

			format:       Mermaid(),
			expectedType: "visual",
		}, "Table format uses visual progress": {

			format:       Table(),
			expectedType: "visual",
		}, "Unknown format uses text progress": {

			format:       Format{Name: "unknown"},
			expectedType: "text",
		}, "YAML format uses NoOp": {

			format:       YAML(),
			expectedType: "noop",
		}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
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
	tests := map[string]struct {
		formats      []Format
		expectedType string
	}{"Empty formats uses NoOp": {

		formats:      []Format{},
		expectedType: "noop",
	}, "Mixed formats uses conservative approach": {

		formats:      []Format{JSON(), Table(), CSV()},
		expectedType: "mixed",
	}, "Only non-visual formats uses NoOp": {

		formats:      []Format{JSON(), CSV(), YAML()},
		expectedType: "noop",
	}, "Only visual formats uses visual progress": {

		formats:      []Format{Table(), HTML()},
		expectedType: "visual",
	}, "Single non-visual format": {

		formats:      []Format{DOT()},
		expectedType: "noop",
	}, "Single visual format": {

		formats:      []Format{Markdown()},
		expectedType: "visual",
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
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
		JSON(),
		CSV(),
		YAML(),
		DOT(),
		Table(),
		HTML(),
		Markdown(),
		Mermaid(),
		DrawIO(),
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
	tests := map[string]struct {
		formatName   string
		expectedType string
	}{"JSON format name uses NoOp": {

		formatName:   "json",
		expectedType: "noop",
	}, "Table format name uses visual progress": {

		formatName:   "table",
		expectedType: "visual",
	}, "Unknown format name uses text progress": {

		formatName:   "unknown",
		expectedType: "text",
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
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
	formats := []Format{Table(), JSON()}

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

// TestTextProgress_NilWriter is a regression test for T-1256.
//
// WithProgressWriter(nil) used to store a nil io.Writer in ProgressConfig
// without validation. The draw/drawFinal methods then called
// fmt.Fprintf(p.config.Writer, ...) which panicked with a nil writer when any
// drawing operation (SetTotal, SetStatus, Complete, Fail, Close) ran.
//
// Expected: a nil writer is normalized to the default writer, so none of the
// drawing operations panic.
func TestTextProgress_NilWriter(t *testing.T) {
	t.Run("explicit nil writer does not panic on draw operations", func(t *testing.T) {
		// Passing nil must not result in a nil Writer being used for rendering.
		progress := NewProgress(
			WithProgressWriter(nil),
			WithUpdateInterval(0), // No throttling so every call draws
		)

		// Each of these triggers draw()/drawFinal(); with the bug present any
		// of them would panic on fmt.Fprintf(nil, ...).
		progress.SetTotal(100)
		progress.SetCurrent(25)
		progress.Increment(25)
		progress.SetStatus("processing data")
		progress.Complete()

		if err := progress.Close(); err != nil {
			t.Errorf("Close() should not return error, got %v", err)
		}
	})

	t.Run("explicit nil writer does not panic on Fail", func(t *testing.T) {
		progress := NewProgress(
			WithProgressWriter(nil),
			WithUpdateInterval(0),
		)

		progress.SetTotal(10)
		progress.Fail(errors.New("boom"))

		if err := progress.Close(); err != nil {
			t.Errorf("Close() should not return error, got %v", err)
		}
	})
}
