package output

import (
	"context"
	"strings"
	"testing"
)

// These tests cover T-1516: the Output-level v1 compatibility options
// WithTableStyle, WithTOC, and WithFrontMatter must affect rendering.
// Before the fix they set fields on Output that nothing in the render
// path read, so a v1 migrator writing
// NewOutput(WithFormat(Markdown()), WithTOC(true)) silently got no ToC.

// renderCompat builds an Output from opts plus a capturing writer, renders
// doc, and returns the captured output.
func renderCompat(t *testing.T, doc *Document, opts ...OutputOption) string {
	t.Helper()
	w := &mockWriter{name: "capture"}
	out := NewOutput(append(opts, WithWriter(w))...)
	if err := out.Render(context.Background(), doc); err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}
	return w.String()
}

func TestWithTOCEnablesMarkdownToC(t *testing.T) {
	// Bug T-1516: expected the rendered markdown to contain a table of
	// contents; actual behaviour was that WithTOC(true) was silently ignored.
	doc := New().
		Header("Introduction").
		Table("Data", []map[string]any{{"Name": "Alice"}}, WithKeys("Name")).
		Build()

	got := renderCompat(t, doc,
		WithFormat(Markdown()),
		WithTOC(true),
	)
	if !strings.Contains(got, "## Table of Contents") {
		t.Errorf("WithTOC(true) output missing table of contents; got:\n%s", got)
	}
	if !strings.Contains(got, "[Introduction](#introduction)") {
		t.Errorf("WithTOC(true) output missing ToC entry for header; got:\n%s", got)
	}
}

func TestWithFrontMatterAddsMarkdownFrontMatter(t *testing.T) {
	// Bug T-1516: expected the rendered markdown to start with YAML front
	// matter; actual behaviour was that WithFrontMatter was silently ignored.
	doc := New().
		Text("body text").
		Build()

	got := renderCompat(t, doc,
		WithFormat(Markdown()),
		WithFrontMatter(map[string]string{"title": "Migration Test"}),
	)
	if !strings.HasPrefix(got, "---\n") {
		t.Errorf("WithFrontMatter output does not start with front matter delimiter; got:\n%s", got)
	}
	if !strings.Contains(got, "title: Migration Test") {
		t.Errorf("WithFrontMatter output missing front matter entry; got:\n%s", got)
	}
}

func TestWithTableStyleAppliesStyleToTableFormat(t *testing.T) {
	// Bug T-1516: expected the table to be drawn with the Bold style's
	// box-drawing characters; actual behaviour was that WithTableStyle was
	// silently ignored and the default style was used.
	doc := New().
		Table("Data", []map[string]any{{"Name": "Alice"}}, WithKeys("Name")).
		Build()

	got := renderCompat(t, doc,
		WithFormat(Table()),
		WithTableStyle("Bold"),
	)
	if !strings.Contains(got, "┏") {
		t.Errorf("WithTableStyle(\"Bold\") output missing Bold style characters; got:\n%s", got)
	}
}

func TestWithTableStylePreservesMaxColumnWidth(t *testing.T) {
	// WithTableStyle must override only the style: a max column width
	// configured through the format constructor must survive.
	longValue := strings.Repeat("x", 40)
	doc := New().
		Table("Data", []map[string]any{{"Name": longValue}}, WithKeys("Name")).
		Build()

	got := renderCompat(t, doc,
		WithFormat(TableWithMaxColumnWidth(10)),
		WithTableStyle("Bold"),
	)
	if !strings.Contains(got, "┏") {
		t.Errorf("WithTableStyle(\"Bold\") output missing Bold style characters; got:\n%s", got)
	}
	if strings.Contains(got, longValue) {
		t.Errorf("max column width lost: 40-char value rendered unwrapped; got:\n%s", got)
	}
}

func TestOutputCompatOptionsCombineWithFormatConstructors(t *testing.T) {
	// Output-level options are additive: front matter from WithFrontMatter
	// must combine with a ToC enabled through the format constructor.
	doc := New().
		Header("Overview").
		Build()

	got := renderCompat(t, doc,
		WithFormat(MarkdownWithToC(true)),
		WithFrontMatter(map[string]string{"author": "Test Suite"}),
	)
	if !strings.Contains(got, "author: Test Suite") {
		t.Errorf("combined output missing front matter; got:\n%s", got)
	}
	if !strings.Contains(got, "## Table of Contents") {
		t.Errorf("combined output lost constructor-enabled ToC; got:\n%s", got)
	}
}

func TestCompatOptionsLeaveOtherFormatsUntouched(t *testing.T) {
	// The v1 compatibility options only target their matching formats: JSON
	// output must be identical with and without them.
	doc := New().
		Table("Data", []map[string]any{{"Name": "Alice"}}, WithKeys("Name")).
		Build()

	plain := renderCompat(t, doc, WithFormat(JSON()))
	withOpts := renderCompat(t, doc,
		WithFormat(JSON()),
		WithTableStyle("Bold"),
		WithTOC(true),
		WithFrontMatter(map[string]string{"title": "Test"}),
	)

	if plain != withOpts {
		t.Errorf("JSON output changed by compat options:\nplain:\n%s\nwith options:\n%s", plain, withOpts)
	}
}
