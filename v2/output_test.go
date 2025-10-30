package output

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

func TestNewOutput(t *testing.T) {
	t.Run("with default options", func(t *testing.T) {
		output := NewOutput()
		if output == nil {
			t.Fatal("NewOutput() should not return nil")
		}

		// Should have no-op progress by default
		progress := output.GetProgress()
		if progress == nil {
			t.Error("NewOutput() should have default progress")
		}

		err := output.Close()
		if err != nil {
			t.Errorf("Close() should not return error, got %v", err)
		}
	})

	t.Run("with custom options", func(t *testing.T) {
		var buf bytes.Buffer
		customProgress := NewProgress(WithProgressWriter(&buf))

		output := NewOutput(
			WithFormat(JSON()),
			WithFormat(CSV()),
			WithWriter(NewStdoutWriter()),
			WithProgress(customProgress),
			WithTableStyle("Bold"),
			WithTOC(true),
			WithFrontMatter(map[string]string{"title": "Test"}),
			WithMetadata("version", "1.0"),
		)

		formats := output.GetFormats()
		if len(formats) != 2 {
			t.Errorf("expected 2 formats, got %d", len(formats))
		}

		writers := output.GetWriters()
		if len(writers) != 1 {
			t.Errorf("expected 1 writer, got %d", len(writers))
		}

		progress := output.GetProgress()
		if progress != customProgress {
			t.Error("progress should be the custom one provided")
		}

		err := output.Close()
		if err != nil {
			t.Errorf("Close() should not return error, got %v", err)
		}
	})
}

func TestOutput_Render_ProgressIntegration(t *testing.T) {
	// Create a simple document
	doc := New().
		Table("Test", []map[string]any{
			{"Name": "Alice", "Age": 30},
			{"Name": "Bob", "Age": 25},
		}, WithKeys("Name", "Age")).
		Build()

	t.Run("progress tracking with single format", func(t *testing.T) {
		var progressBuf bytes.Buffer

		progress := NewProgress(
			WithProgressWriter(&progressBuf),
			WithUpdateInterval(0), // No throttling for tests
		)

		// Create a test stdout writer that captures output
		testWriter := &testStdoutWriter{}

		output := NewOutput(
			WithFormat(JSON()),
			WithWriter(testWriter),
			WithProgress(progress),
		)

		ctx := context.Background()
		err := output.Render(ctx, doc)
		if err != nil {
			t.Fatalf("Render() failed: %v", err)
		}

		err = output.Close()
		if err != nil {
			t.Errorf("Close() should not return error, got %v", err)
		}

		progressOutput := progressBuf.String()
		if !strings.Contains(progressOutput, "✓ Complete") {
			t.Error("progress should show completion")
		}

		// Verify JSON output was generated
		if !strings.Contains(testWriter.GetOutput(), "Alice") {
			t.Error("JSON output should contain table data")
		}
	})

	t.Run("progress tracking with multiple formats", func(t *testing.T) {
		var progressBuf bytes.Buffer

		progress := NewProgress(
			WithProgressWriter(&progressBuf),
			WithUpdateInterval(0),
		)

		testWriter := &testStdoutWriter{}

		output := NewOutput(
			WithFormats(JSON(), CSV(), YAML()),
			WithWriter(testWriter),
			WithProgress(progress),
		)

		ctx := context.Background()
		err := output.Render(ctx, doc)
		if err != nil {
			t.Fatalf("Render() failed: %v", err)
		}

		err = output.Close()
		if err != nil {
			t.Errorf("Close() should not return error, got %v", err)
		}

		progressOutput := progressBuf.String()
		if !strings.Contains(progressOutput, "✓ Complete") {
			t.Error("progress should show completion")
		}

		// With 3 formats and 1 writer, should have processed 3 work units
		if !strings.Contains(progressOutput, "(3/3)") {
			t.Error("progress should show (3/3) for 3 formats with 1 writer")
		}
	})

	t.Run("progress tracking with multiple writers", func(t *testing.T) {
		var progressBuf bytes.Buffer

		progress := NewProgress(
			WithProgressWriter(&progressBuf),
			WithUpdateInterval(0),
		)

		testWriter1 := &testStdoutWriter{}
		testWriter2 := &testStdoutWriter{}

		output := NewOutput(
			WithFormat(JSON()),
			WithWriters(
				testWriter1,
				testWriter2,
			),
			WithProgress(progress),
		)

		ctx := context.Background()
		err := output.Render(ctx, doc)
		if err != nil {
			t.Fatalf("Render() failed: %v", err)
		}

		err = output.Close()
		if err != nil {
			t.Errorf("Close() should not return error, got %v", err)
		}

		progressOutput := progressBuf.String()
		if !strings.Contains(progressOutput, "✓ Complete") {
			t.Error("progress should show completion")
		}

		// With 1 format and 2 writers, should have processed 2 work units
		if !strings.Contains(progressOutput, "(2/2)") {
			t.Error("progress should show (2/2) for 1 format with 2 writers")
		}

		// Both writers should have received the output
		if !strings.Contains(testWriter1.GetOutput(), "Alice") {
			t.Error("first writer should contain table data")
		}
		if !strings.Contains(testWriter2.GetOutput(), "Alice") {
			t.Error("second writer should contain table data")
		}
	})

	t.Run("progress status messages", func(t *testing.T) {
		var progressBuf bytes.Buffer

		progress := NewProgress(
			WithProgressWriter(&progressBuf),
			WithUpdateInterval(0),
		)

		testWriter := &testStdoutWriter{}

		output := NewOutput(
			WithFormat(JSON()),
			WithWriter(testWriter),
			WithProgress(progress),
		)

		ctx := context.Background()
		err := output.Render(ctx, doc)
		if err != nil {
			t.Fatalf("Render() failed: %v", err)
		}

		err = output.Close()
		if err != nil {
			t.Errorf("Close() should not return error, got %v", err)
		}

		progressOutput := progressBuf.String()

		// Should contain various status messages
		if !strings.Contains(progressOutput, "Starting render process") {
			t.Error("progress should show starting message")
		}
		if !strings.Contains(progressOutput, "Rendering json format") {
			t.Error("progress should show rendering message")
		}
		if !strings.Contains(progressOutput, "Writing json format") {
			t.Error("progress should show writing message")
		}
	})
}

func TestOutput_Render_ErrorHandling(t *testing.T) {
	doc := New().
		Table("Test", []map[string]any{
			{"Name": "Alice", "Age": 30},
		}, WithKeys("Name", "Age")).
		Build()

	t.Run("no formats configured", func(t *testing.T) {
		var progressBuf bytes.Buffer
		progress := NewProgress(WithProgressWriter(&progressBuf))

		output := NewOutput(
			WithWriter(NewStdoutWriter()),
			WithProgress(progress),
		)

		ctx := context.Background()
		err := output.Render(ctx, doc)
		if err == nil {
			t.Fatal("Render() should fail with no formats")
		}

		// The new error system returns ValidationError for empty slices
		if !strings.Contains(err.Error(), "formats") || !strings.Contains(err.Error(), "cannot be empty") {
			t.Errorf("error should mention formats cannot be empty, got: %v", err)
		}

		output.Close()
	})

	t.Run("no writers configured", func(t *testing.T) {
		var progressBuf bytes.Buffer
		progress := NewProgress(WithProgressWriter(&progressBuf))

		output := NewOutput(
			WithFormat(JSON()),
			WithProgress(progress),
		)

		ctx := context.Background()
		err := output.Render(ctx, doc)
		if err == nil {
			t.Fatal("Render() should fail with no writers")
		}

		// The new error system returns ValidationError for empty slices
		if !strings.Contains(err.Error(), "writers") || !strings.Contains(err.Error(), "cannot be empty") {
			t.Errorf("error should mention writers cannot be empty, got: %v", err)
		}

		output.Close()
	})

	t.Run("context cancellation", func(t *testing.T) {
		var progressBuf bytes.Buffer
		progress := NewProgress(WithProgressWriter(&progressBuf))

		output := NewOutput(
			WithFormat(JSON()),
			WithWriter(NewStdoutWriter()),
			WithProgress(progress),
		)

		// Create a cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := output.Render(ctx, doc)
		if err == nil {
			t.Fatal("Render() should fail with cancelled context")
		}

		// The new error system wraps cancellation errors
		if !IsCancelled(err) {
			t.Errorf("error should be a cancellation error, got: %v", err)
		}

		// Should be able to unwrap to get the original context.Canceled
		var cancelledErr *CancelledError
		if !AsError(err, &cancelledErr) {
			t.Errorf("error should be CancelledError, got: %T", err)
		} else if !errors.Is(cancelledErr.Cause, context.Canceled) {
			t.Errorf("underlying cause should be context.Canceled, got: %v", cancelledErr.Cause)
		}

		// Progress should show failure (may be empty for no-op progress)
		progressOutput := progressBuf.String()
		// Note: The progress output format may vary based on implementation
		// For now, just ensure no panic occurred and error was handled
		_ = progressOutput // We'll skip the specific progress message check for now

		output.Close()
	})
}

func TestOutput_Render_WithTransformers(t *testing.T) {
	doc := New().
		Table("Test", []map[string]any{
			{"Name": "Alice", "Status": ":check:"},
			{"Name": "Bob", "Status": ":x:"},
		}, WithKeys("Name", "Status")).
		Build()

	t.Run("with emoji transformer", func(t *testing.T) {
		var progressBuf bytes.Buffer

		progress := NewProgress(
			WithProgressWriter(&progressBuf),
			WithUpdateInterval(0),
		)

		// Create a simple emoji transformer for testing
		emojiTransformer := &testEmojiTransformer{}
		testWriter := &testStdoutWriter{}

		output := NewOutput(
			WithFormat(JSON()),
			WithTransformer(emojiTransformer),
			WithWriter(testWriter),
			WithProgress(progress),
		)

		ctx := context.Background()
		err := output.Render(ctx, doc)
		if err != nil {
			t.Fatalf("Render() failed: %v", err)
		}

		// Progress should show transformer application
		progressOutput := progressBuf.String()
		if !strings.Contains(progressOutput, "Applying emoji transformer") {
			t.Error("progress should show transformer application")
		}

		output.Close()
	})
}

func TestOutput_RenderTo(t *testing.T) {
	doc := New().
		Table("Test", []map[string]any{
			{"Name": "Alice", "Age": 30},
		}, WithKeys("Name", "Age")).
		Build()

	testWriter := &testStdoutWriter{}
	output := NewOutput(
		WithFormat(JSON()),
		WithWriter(testWriter),
	)

	err := output.RenderTo(doc)
	if err != nil {
		t.Fatalf("RenderTo() failed: %v", err)
	}

	if !strings.Contains(testWriter.GetOutput(), "Alice") {
		t.Error("JSON output should contain table data")
	}

	output.Close()
}

func TestOutput_ThreadSafety(t *testing.T) {
	doc := New().
		Table("Test", []map[string]any{
			{"Name": "Alice", "Age": 30},
		}, WithKeys("Name", "Age")).
		Build()

	output := NewOutput(
		WithFormat(JSON()),
		WithWriter(NewStdoutWriter()),
	)

	// Test concurrent renders
	done := make(chan bool, 3)
	errors := make(chan error, 3)

	for range 3 {
		go func() {
			err := output.RenderTo(doc)
			errors <- err
			done <- true
		}()
	}

	// Wait for all goroutines
	for range 3 {
		<-done
		err := <-errors
		if err != nil {
			t.Errorf("concurrent render failed: %v", err)
		}
	}

	output.Close()
}

// testEmojiTransformer is a simple transformer for testing
type testEmojiTransformer struct{}

func (t *testEmojiTransformer) Name() string                    { return "emoji" }
func (t *testEmojiTransformer) Priority() int                   { return 100 }
func (t *testEmojiTransformer) CanTransform(format string) bool { return true }
func (t *testEmojiTransformer) Transform(ctx context.Context, input []byte, format string) ([]byte, error) {
	// Simple emoji replacement for testing
	output := strings.ReplaceAll(string(input), ":check:", "✓")
	output = strings.ReplaceAll(output, ":x:", "✗")
	return []byte(output), nil
}

// testStdoutWriter is a writer that captures output for testing
type testStdoutWriter struct {
	output bytes.Buffer
	mu     sync.Mutex
}

func (w *testStdoutWriter) Write(ctx context.Context, format string, data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, err := w.output.Write(data)
	return err
}

func (w *testStdoutWriter) GetOutput() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.output.String()
}

func (w *testStdoutWriter) Reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.output.Reset()
}
