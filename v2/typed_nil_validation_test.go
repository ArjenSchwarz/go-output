package output

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
)

// Regression tests for T-1649: typed-nil interface values (a non-nil interface
// wrapping a nil concrete pointer) must not bypass Output validation. Each
// option type — transformer, progress, renderer, writer — previously passed
// the `== nil` checks with a typed nil and panicked during render instead of
// returning the documented validation error.

// typedNilTestRenderer implements Renderer with pointer receivers that touch
// fields, so any call on a typed-nil value panics.
type typedNilTestRenderer struct {
	format string
}

func (r *typedNilTestRenderer) Format() string { return r.format }

func (r *typedNilTestRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
	return []byte(r.format), nil
}

func (r *typedNilTestRenderer) RenderTo(ctx context.Context, doc *Document, w io.Writer) error {
	_, err := w.Write([]byte(r.format))
	return err
}

func (r *typedNilTestRenderer) SupportsStreaming() bool { return r.format != "" }

// typedNilTestWriter implements Writer with a pointer receiver that touches a
// field, so Write on a typed-nil value panics.
type typedNilTestWriter struct {
	lastFormat string
}

func (w *typedNilTestWriter) Write(ctx context.Context, format string, data []byte) error {
	w.lastFormat = format
	return nil
}

// typedNilTestProgress implements Progress with pointer receivers that touch
// fields, so any call on a typed-nil value panics.
type typedNilTestProgress struct {
	total int
}

func (p *typedNilTestProgress) SetTotal(total int)             { p.total = total }
func (p *typedNilTestProgress) SetCurrent(current int)         { p.total = current }
func (p *typedNilTestProgress) Increment(delta int)            { p.total += delta }
func (p *typedNilTestProgress) SetStatus(status string)        { p.total = len(status) }
func (p *typedNilTestProgress) Complete()                      { p.total = 0 }
func (p *typedNilTestProgress) Fail(err error)                 { p.total = 0 }
func (p *typedNilTestProgress) SetColor(color ProgressColor)   { p.total = int(color) }
func (p *typedNilTestProgress) IsActive() bool                 { return p.total > 0 }
func (p *typedNilTestProgress) SetContext(ctx context.Context) { p.total = 0 }
func (p *typedNilTestProgress) Close() error                   { p.total = 0; return nil }

// discardWriter returns a Writer that records nothing and always succeeds.
func discardWriter() Writer {
	return WriterFunc(func(ctx context.Context, format string, data []byte) error {
		return nil
	})
}

func textDocument() *Document {
	return New().Text("x").Build()
}

// requireValidationError asserts err is (or wraps) a *ValidationError whose
// message mentions the given field.
func requireValidationError(t *testing.T, err error, field string) {
	t.Helper()
	if err == nil {
		t.Fatalf("Render() error = nil, want validation error for %q", field)
	}
	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("Render() error = %v (%T), want *ValidationError for %q", err, err, field)
	}
	if !strings.Contains(err.Error(), field) {
		t.Errorf("Render() error = %q, want it to mention field %q", err.Error(), field)
	}
}

// TestRenderRejectsTypedNilTransformer verifies that a typed-nil Transformer
// (the ticket reproduction: NewFormatAwareTransformer(nil) returns a nil
// *FormatAwareTransformer that becomes a non-nil Transformer interface) is
// rejected with a validation error instead of panicking at Priority() during
// render.
func TestRenderRejectsTypedNilTransformer(t *testing.T) {
	tests := map[string]struct {
		option OutputOption
	}{
		"WithTransformer": {
			option: WithTransformer(NewFormatAwareTransformer(nil)),
		},
		"WithTransformers": {
			option: WithTransformers(NewFormatAwareTransformer(nil)),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			out := NewOutput(
				WithFormat(JSON()),
				WithWriter(discardWriter()),
				tt.option,
			)

			err := out.Render(context.Background(), textDocument())
			requireValidationError(t, err, "transformers[0]")
		})
	}
}

// TestTransformPipelineAddIgnoresTypedNil verifies TransformPipeline.Add
// ignores typed-nil transformers the same way it ignores untyped nil, so the
// pipeline never stores a value that panics during sorting or transformation.
func TestTransformPipelineAddIgnoresTypedNil(t *testing.T) {
	tp := NewTransformPipeline()
	tp.Add(NewFormatAwareTransformer(nil))

	if got := tp.Count(); got != 0 {
		t.Errorf("Count() after adding typed-nil transformer = %d, want 0", got)
	}

	got, err := tp.Transform(context.Background(), []byte("data"), FormatJSON)
	if err != nil {
		t.Fatalf("Transform() error = %v, want nil", err)
	}
	if string(got) != "data" {
		t.Errorf("Transform() = %q, want %q", got, "data")
	}
}

// TestNewFormatAwareTransformerRejectsTypedNil verifies that wrapping a
// typed-nil Transformer returns nil instead of a wrapper that panics on
// Name()/Priority().
func TestNewFormatAwareTransformerRejectsTypedNil(t *testing.T) {
	var typedNil Transformer = (*FormatAwareTransformer)(nil)
	if got := NewFormatAwareTransformer(typedNil); got != nil {
		t.Errorf("NewFormatAwareTransformer(typed nil) = %v, want nil", got)
	}
}

// TestRenderTypedNilProgressFallsBackToNoOp verifies that a typed-nil Progress
// passed to WithProgress is treated as absent (no-op fallback) instead of
// panicking at SetTotal during render.
func TestRenderTypedNilProgressFallsBackToNoOp(t *testing.T) {
	var progress *typedNilTestProgress

	var mu sync.Mutex
	var written []byte
	out := NewOutput(
		WithFormat(JSON()),
		WithWriter(WriterFunc(func(ctx context.Context, format string, data []byte) error {
			mu.Lock()
			defer mu.Unlock()
			written = append(written, data...)
			return nil
		})),
		WithProgress(progress),
	)

	if err := out.Render(context.Background(), textDocument()); err != nil {
		t.Fatalf("Render() with typed-nil progress error = %v, want nil", err)
	}
	if len(written) == 0 {
		t.Error("Render() with typed-nil progress wrote no data, want rendered output")
	}
}

// TestRenderRejectsTypedNilRenderer verifies that a Format carrying a
// typed-nil Renderer is rejected with a validation error instead of panicking
// at Render() time.
func TestRenderRejectsTypedNilRenderer(t *testing.T) {
	var renderer *typedNilTestRenderer
	out := NewOutput(
		WithFormat(Format{Name: "typednil", Renderer: renderer}),
		WithWriter(discardWriter()),
	)

	err := out.Render(context.Background(), textDocument())
	requireValidationError(t, err, "formats[0].renderer")
}

// TestRenderRejectsTypedNilWriter verifies that a typed-nil Writer is rejected
// with a validation error instead of panicking at Write() time.
func TestRenderRejectsTypedNilWriter(t *testing.T) {
	var writer *typedNilTestWriter
	out := NewOutput(
		WithFormat(JSON()),
		WithWriter(writer),
	)

	err := out.Render(context.Background(), textDocument())
	requireValidationError(t, err, "writers[0]")
}

// TestRenderRejectsTypedNilDocument verifies that a nil *Document (boxed into
// a typed-nil interface by the validation helper) is rejected with a
// validation error instead of panicking at doc.GetContents().
func TestRenderRejectsTypedNilDocument(t *testing.T) {
	out := NewOutput(
		WithFormat(JSON()),
		WithWriter(discardWriter()),
	)

	err := out.Render(context.Background(), nil)
	requireValidationError(t, err, "document")
}

// TestStdioSetWriterIgnoresTypedNil verifies that SetWriter ignores a
// typed-nil io.Writer (deferred from T-1387 to this ticket), keeping the
// previously configured writer instead of storing a value that panics on the
// next Write.
func TestStdioSetWriterIgnoresTypedNil(t *testing.T) {
	type stdioWriter interface {
		Writer
		SetWriter(w io.Writer)
	}

	tests := map[string]struct {
		newWriter func() stdioWriter
	}{
		"stdout": {newWriter: func() stdioWriter { return NewStdoutWriter() }},
		"stderr": {newWriter: func() stdioWriter { return NewStderrWriter() }},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			sw := tt.newWriter()

			var buf bytes.Buffer
			sw.SetWriter(&buf)

			var typedNil *bytes.Buffer
			sw.SetWriter(typedNil)

			if err := sw.Write(context.Background(), FormatJSON, []byte("data")); err != nil {
				t.Fatalf("Write() after typed-nil SetWriter error = %v, want nil", err)
			}
			if got := buf.String(); !strings.Contains(got, "data") {
				t.Errorf("Write() after typed-nil SetWriter wrote %q to original writer, want it to contain %q", got, "data")
			}
		})
	}
}

// TestMultiWriterIgnoresTypedNilWriters verifies that NewMultiWriter and
// AddWriter drop typed-nil writers the same way they drop untyped nil, so a
// nil concrete pointer boxed into the Writer interface never reaches the
// write goroutine where calling Write on it would panic.
func TestMultiWriterIgnoresTypedNilWriters(t *testing.T) {
	var typedNil *typedNilTestWriter

	t.Run("NewMultiWriter drops typed nil", func(t *testing.T) {
		valid := &typedNilTestWriter{}
		mw := NewMultiWriter(valid, typedNil)

		if got := mw.WriterCount(); got != 1 {
			t.Fatalf("WriterCount() = %d, want 1 (typed-nil writer should be dropped)", got)
		}
		if err := mw.Write(context.Background(), FormatText, []byte("data")); err != nil {
			t.Fatalf("Write() error = %v, want nil", err)
		}
		if valid.lastFormat != FormatText {
			t.Errorf("valid writer lastFormat = %q, want %q", valid.lastFormat, FormatText)
		}
	})

	t.Run("AddWriter ignores typed nil", func(t *testing.T) {
		mw := NewMultiWriter()
		mw.AddWriter(typedNil)

		if got := mw.WriterCount(); got != 0 {
			t.Fatalf("after AddWriter(typed nil): WriterCount() = %d, want 0", got)
		}
	})
}
