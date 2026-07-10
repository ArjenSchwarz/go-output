package output

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// These tests are the regression tests for T-1524: multi-format rendering
// wrote to shared writers in nondeterministic order. Each format rendered AND
// wrote inside its own goroutine, so with two formats and one shared writer
// the format whose goroutine finished first appeared first in the output.
//
// Expected behaviour: writes to writers happen in declared format order
// (the order formats were passed to WithFormat/WithFormats), regardless of
// how long each format takes to render.
//
// Actual behaviour before the fix: write order followed goroutine completion
// order, so a slow first-declared format reliably lost the race to a fast
// second-declared format.

// orderTrackingWriter records the format name of every Write call and
// accumulates the written bytes, so tests can assert cross-format ordering.
type orderTrackingWriter struct {
	mu    sync.Mutex
	calls []string
	buf   bytes.Buffer
}

func (w *orderTrackingWriter) Write(_ context.Context, format string, data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.calls = append(w.calls, format)
	w.buf.Write(data)
	return nil
}

func (w *orderTrackingWriter) Calls() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	calls := make([]string, len(w.calls))
	copy(calls, w.calls)
	return calls
}

func (w *orderTrackingWriter) Output() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

// delayedRenderer renders fixed bytes after an optional delay, letting tests
// control which format finishes rendering first.
type delayedRenderer struct {
	name   string
	output []byte
	delay  time.Duration
}

func (r *delayedRenderer) Format() string { return r.name }

func (r *delayedRenderer) Render(ctx context.Context, _ *Document) ([]byte, error) {
	if r.delay > 0 {
		select {
		case <-time.After(r.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return r.output, nil
}

func (r *delayedRenderer) RenderTo(ctx context.Context, doc *Document, w io.Writer) error {
	data, err := r.Render(ctx, doc)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (r *delayedRenderer) SupportsStreaming() bool { return false }

// failingDelayedRenderer always fails after an optional delay.
type failingDelayedRenderer struct {
	name  string
	delay time.Duration
}

func (r *failingDelayedRenderer) Format() string { return r.name }

func (r *failingDelayedRenderer) Render(_ context.Context, _ *Document) ([]byte, error) {
	if r.delay > 0 {
		time.Sleep(r.delay)
	}
	return nil, fmt.Errorf("render failed for %s", r.name)
}

func (r *failingDelayedRenderer) RenderTo(_ context.Context, _ *Document, _ io.Writer) error {
	return fmt.Errorf("render failed for %s", r.name)
}

func (r *failingDelayedRenderer) SupportsStreaming() bool { return false }

// TestOutput_Render_WriteOrderMatchesFormatOrder verifies that when multiple
// formats share a single writer, their output is written in declared format
// order even when the first-declared format is the slowest to render.
func TestOutput_Render_WriteOrderMatchesFormatOrder(t *testing.T) {
	doc := New().Text("irrelevant").Build()

	// The first-declared format is deliberately the slowest so that, under
	// the pre-fix goroutine-per-format write model, it reliably writes last.
	formats := []Format{
		{Name: "first", Renderer: &delayedRenderer{name: "first", output: []byte("AAA\n"), delay: 30 * time.Millisecond}},
		{Name: "second", Renderer: &delayedRenderer{name: "second", output: []byte("BBB\n"), delay: 10 * time.Millisecond}},
		{Name: "third", Renderer: &delayedRenderer{name: "third", output: []byte("CCC\n")}},
	}

	want := []string{"first", "second", "third"}

	// Repeat to guard against a lucky scheduling order masking the bug.
	for i := range 5 {
		writer := &orderTrackingWriter{}
		out := NewOutput(WithFormats(formats...), WithWriter(writer))

		if err := out.Render(context.Background(), doc); err != nil {
			t.Fatalf("iteration %d: Render() failed: %v", i, err)
		}

		got := writer.Calls()
		if len(got) != len(want) {
			t.Fatalf("iteration %d: got %d writes, want %d: %v", i, len(got), len(want), got)
		}
		for j := range want {
			if got[j] != want[j] {
				t.Fatalf("iteration %d: writes in wrong order: got %v, want %v", i, got, want)
			}
		}

		// The concatenated bytes must also be in declared order.
		if output := writer.Output(); output != "AAA\nBBB\nCCC\n" {
			t.Fatalf("iteration %d: output bytes in wrong order: got %q", i, output)
		}
	}
}

// TestOutput_Render_WriteOrderWithMultipleWriters verifies that every writer
// receives the formats in declared order, and that within one format the
// writers are used in declared writer order.
func TestOutput_Render_WriteOrderWithMultipleWriters(t *testing.T) {
	doc := New().Text("irrelevant").Build()

	writer1 := &orderTrackingWriter{}
	writer2 := &orderTrackingWriter{}

	out := NewOutput(
		WithFormats(
			Format{Name: "first", Renderer: &delayedRenderer{name: "first", output: []byte("AAA\n"), delay: 20 * time.Millisecond}},
			Format{Name: "second", Renderer: &delayedRenderer{name: "second", output: []byte("BBB\n")}},
		),
		WithWriters(writer1, writer2),
	)

	if err := out.Render(context.Background(), doc); err != nil {
		t.Fatalf("Render() failed: %v", err)
	}

	want := []string{"first", "second"}
	for name, w := range map[string]*orderTrackingWriter{"writer1": writer1, "writer2": writer2} {
		got := w.Calls()
		if len(got) != len(want) {
			t.Fatalf("%s: got %d writes, want %d: %v", name, len(got), len(want), got)
		}
		for j := range want {
			if got[j] != want[j] {
				t.Fatalf("%s: writes in wrong order: got %v, want %v", name, got, want)
			}
		}
	}
}

// TestOutput_Render_WriteOrderSkipsFailedFormats verifies the error
// aggregation semantics around ordered writes: a format whose render fails
// produces an error but does not prevent the remaining formats from being
// written, and the successful formats are still written in declared order.
func TestOutput_Render_WriteOrderSkipsFailedFormats(t *testing.T) {
	doc := New().Text("irrelevant").Build()

	writer := &orderTrackingWriter{}
	out := NewOutput(
		WithFormats(
			Format{Name: "first", Renderer: &delayedRenderer{name: "first", output: []byte("AAA\n"), delay: 20 * time.Millisecond}},
			Format{Name: "broken", Renderer: &failingDelayedRenderer{name: "broken"}},
			Format{Name: "third", Renderer: &delayedRenderer{name: "third", output: []byte("CCC\n")}},
		),
		WithWriter(writer),
	)

	err := out.Render(context.Background(), doc)
	if err == nil {
		t.Fatal("Render() should return an error when a format fails to render")
	}
	if !strings.Contains(err.Error(), "broken") {
		t.Errorf("error should mention the failing format, got: %v", err)
	}

	want := []string{"first", "third"}
	got := writer.Calls()
	if len(got) != len(want) {
		t.Fatalf("got %d writes, want %d: %v", len(got), len(want), got)
	}
	for j := range want {
		if got[j] != want[j] {
			t.Fatalf("writes in wrong order: got %v, want %v", got, want)
		}
	}
}
