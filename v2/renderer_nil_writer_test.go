package output

import (
	"context"
	"strings"
	"testing"
)

// TestRenderTo_NilWriter is a regression test for T-1120.
//
// Every built-in renderer's RenderTo wrote to the io.Writer without first
// checking whether it was nil. After a successful render this caused a nil
// pointer panic on w.Write(...) rather than returning an error. This was
// inconsistent with baseRenderer.renderDocumentTo and the CSV renderer, which
// already returned a "writer cannot be nil" error.
//
// Expected behaviour: RenderTo returns an error containing "writer cannot be
// nil" and does not panic. Actual behaviour before the fix: a nil pointer
// panic from w.Write.
func TestRenderTo_NilWriter(t *testing.T) {
	doc := New().
		Table("people", []map[string]any{
			{"name": "Alice", "age": 30},
		}, WithKeys("name", "age")).
		Text("some text").
		Build()

	renderers := map[string]Renderer{
		"json":        &jsonRenderer{},
		"yaml":        &yamlRenderer{},
		"markdown":    &markdownRenderer{headingLevel: 1},
		"table":       &tableRenderer{},
		"csv":         &csvRenderer{},
		"html":        &htmlRenderer{useTemplate: true, template: DefaultHTMLTemplate},
		"html-noTmpl": &htmlRenderer{useTemplate: false},
		"dot":         &dotRenderer{},
		"mermaid":     &mermaidRenderer{},
		"drawio":      &drawioRenderer{},
	}

	for name, renderer := range renderers {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("RenderTo panicked on nil writer: %v", r)
				}
			}()

			err := renderer.RenderTo(context.Background(), doc, nil)
			if err == nil {
				t.Fatal("expected error for nil writer, got nil")
			}
			if !strings.Contains(err.Error(), "writer cannot be nil") {
				t.Errorf("expected error containing %q, got %q", "writer cannot be nil", err.Error())
			}
		})
	}
}

// TestHTMLRenderer_NilWriterWithTemplate is a regression test for T-1120.
//
// htmlRenderer.RenderTo writes the template header to w before delegating to
// baseRenderer.renderDocumentTo (which has its own nil-writer guard). With
// template output enabled this earlier write panicked on a nil writer. The
// guard must run before any write so the error is returned instead.
func TestHTMLRenderer_NilWriterWithTemplate(t *testing.T) {
	doc := New().Text("hello").Build()
	renderer := &htmlRenderer{useTemplate: true, template: DefaultHTMLTemplate}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RenderTo panicked on nil writer: %v", r)
		}
	}()

	err := renderer.RenderTo(context.Background(), doc, nil)
	if err == nil {
		t.Fatal("expected error for nil writer, got nil")
	}
	if !strings.Contains(err.Error(), "writer cannot be nil") {
		t.Errorf("expected error containing %q, got %q", "writer cannot be nil", err.Error())
	}
}
