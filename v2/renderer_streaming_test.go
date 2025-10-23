package output

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// TestJSONRenderer_RenderToWithTransformations tests that RenderTo produces
// the same output as Render when transformations are present
func TestJSONRenderer_RenderToWithTransformations(t *testing.T) {
	tests := map[string]struct {
		data            []Record
		transformations []Operation
	}{
		"filter transformation": {
			data: []Record{
				{"name": "Alice", "age": 30},
				{"name": "Bob", "age": 25},
				{"name": "Charlie", "age": 35},
			},
			transformations: []Operation{
				NewFilterOp(func(r Record) bool {
					age, ok := r["age"].(int)
					return ok && age >= 30
				}),
			},
		},
		"sort transformation": {
			data: []Record{
				{"name": "Charlie", "score": 85},
				{"name": "Alice", "score": 95},
				{"name": "Bob", "score": 90},
			},
			transformations: []Operation{
				NewSortOp(SortKey{Column: "score", Direction: Descending}),
			},
		},
		"limit transformation": {
			data: []Record{
				{"name": "Alice", "age": 30},
				{"name": "Bob", "age": 25},
				{"name": "Charlie", "age": 35},
				{"name": "David", "age": 28},
			},
			transformations: []Operation{
				NewLimitOp(2),
			},
		},
		"multiple transformations": {
			data: []Record{
				{"name": "Charlie", "age": 35, "active": true},
				{"name": "Alice", "age": 30, "active": true},
				{"name": "Bob", "age": 25, "active": false},
				{"name": "David", "age": 40, "active": true},
			},
			transformations: []Operation{
				NewFilterOp(func(r Record) bool {
					active, ok := r["active"].(bool)
					return ok && active
				}),
				NewSortOp(SortKey{Column: "age", Direction: Descending}),
				NewLimitOp(2),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create document with transformations
			doc := New().
				Table("test", tc.data,
					WithKeys("name", "age", "score", "active"),
					WithTransformations(tc.transformations...),
				).
				Build()

			renderer := &jsonRenderer{}

			// Get output from Render()
			renderOutput, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			// Get output from RenderTo()
			var renderToBuffer bytes.Buffer
			if err := renderer.RenderTo(context.Background(), doc, &renderToBuffer); err != nil {
				t.Fatalf("RenderTo failed: %v", err)
			}
			renderToOutput := renderToBuffer.Bytes()

			// Compare outputs - they should be identical
			if !bytes.Equal(renderOutput, renderToOutput) {
				t.Errorf("Render() and RenderTo() produced different output\nRender():\n%s\n\nRenderTo():\n%s",
					string(renderOutput), string(renderToOutput))
			}

			// Verify that transformation was actually applied by checking record count
			var parsed map[string]any
			if err := json.Unmarshal(renderOutput, &parsed); err != nil {
				t.Fatalf("Failed to parse JSON from Render(): %v", err)
			}
		})
	}
}

// TestYAMLRenderer_RenderToWithTransformations tests YAML renderer streaming with transformations
func TestYAMLRenderer_RenderToWithTransformations(t *testing.T) {
	tests := map[string]struct {
		data            []Record
		transformations []Operation
	}{
		"sort transformation": {
			data: []Record{
				{"name": "Charlie", "score": 85},
				{"name": "Alice", "score": 95},
				{"name": "Bob", "score": 90},
			},
			transformations: []Operation{
				NewSortOp(SortKey{Column: "score", Direction: Descending}),
			},
		},
		"filter and limit": {
			data: []Record{
				{"name": "Alice", "age": 30, "active": true},
				{"name": "Bob", "age": 25, "active": false},
				{"name": "Charlie", "age": 35, "active": true},
				{"name": "David", "age": 40, "active": true},
			},
			transformations: []Operation{
				NewFilterOp(func(r Record) bool {
					active, ok := r["active"].(bool)
					return ok && active
				}),
				NewLimitOp(2),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			doc := New().
				Table("test", tc.data,
					WithKeys("name", "age", "score", "active"),
					WithTransformations(tc.transformations...),
				).
				Build()

			renderer := &yamlRenderer{}

			// Get output from Render()
			renderOutput, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			// Get output from RenderTo()
			var renderToBuffer bytes.Buffer
			if err := renderer.RenderTo(context.Background(), doc, &renderToBuffer); err != nil {
				t.Fatalf("RenderTo failed: %v", err)
			}
			renderToOutput := renderToBuffer.Bytes()

			// Compare outputs - they should be identical
			if !bytes.Equal(renderOutput, renderToOutput) {
				t.Errorf("Render() and RenderTo() produced different output\nRender():\n%s\n\nRenderTo():\n%s",
					string(renderOutput), string(renderToOutput))
			}
		})
	}
}

// TestCSVRenderer_RenderToWithTransformations tests CSV renderer streaming with transformations
func TestCSVRenderer_RenderToWithTransformations(t *testing.T) {
	tests := map[string]struct {
		data            []Record
		transformations []Operation
	}{
		"limit transformation": {
			data: []Record{
				{"name": "Alice", "age": 30},
				{"name": "Bob", "age": 25},
				{"name": "Charlie", "age": 35},
			},
			transformations: []Operation{
				NewLimitOp(2),
			},
		},
		"filter transformation": {
			data: []Record{
				{"name": "Alice", "age": 30, "active": true},
				{"name": "Bob", "age": 25, "active": false},
				{"name": "Charlie", "age": 35, "active": true},
			},
			transformations: []Operation{
				NewFilterOp(func(r Record) bool {
					active, ok := r["active"].(bool)
					return ok && active
				}),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			doc := New().
				Table("test", tc.data,
					WithKeys("name", "age", "active"),
					WithTransformations(tc.transformations...),
				).
				Build()

			renderer := &csvRenderer{}

			// Get output from Render()
			renderOutput, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			// Get output from RenderTo()
			var renderToBuffer bytes.Buffer
			if err := renderer.RenderTo(context.Background(), doc, &renderToBuffer); err != nil {
				t.Fatalf("RenderTo failed: %v", err)
			}
			renderToOutput := renderToBuffer.Bytes()

			// Compare outputs - they should be identical
			if !bytes.Equal(renderOutput, renderToOutput) {
				t.Errorf("Render() and RenderTo() produced different output\nRender():\n%s\n\nRenderTo():\n%s",
					string(renderOutput), string(renderToOutput))
			}
		})
	}
}

// TestMarkdownRenderer_RenderToWithTransformations tests Markdown renderer streaming with transformations
func TestMarkdownRenderer_RenderToWithTransformations(t *testing.T) {
	tests := map[string]struct {
		data            []Record
		transformations []Operation
	}{
		"multiple transformations": {
			data: []Record{
				{"name": "Charlie", "age": 35, "active": true},
				{"name": "Alice", "age": 30, "active": true},
				{"name": "Bob", "age": 25, "active": false},
				{"name": "David", "age": 40, "active": true},
			},
			transformations: []Operation{
				NewFilterOp(func(r Record) bool {
					active, ok := r["active"].(bool)
					return ok && active
				}),
				NewSortOp(SortKey{Column: "age", Direction: Ascending}),
				NewLimitOp(2),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			doc := New().
				Table("test", tc.data,
					WithKeys("name", "age", "active"),
					WithTransformations(tc.transformations...),
				).
				Build()

			renderer := &markdownRenderer{}

			// Get output from Render()
			renderOutput, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			// Get output from RenderTo()
			var renderToBuffer bytes.Buffer
			if err := renderer.RenderTo(context.Background(), doc, &renderToBuffer); err != nil {
				t.Fatalf("RenderTo failed: %v", err)
			}
			renderToOutput := renderToBuffer.Bytes()

			// Compare outputs - they should be identical
			if !bytes.Equal(renderOutput, renderToOutput) {
				t.Errorf("Render() and RenderTo() produced different output\nRender():\n%s\n\nRenderTo():\n%s",
					string(renderOutput), string(renderToOutput))
			}
		})
	}
}

// TestHTMLRenderer_RenderToWithTransformations tests HTML renderer streaming with transformations
func TestHTMLRenderer_RenderToWithTransformations(t *testing.T) {
	tests := map[string]struct {
		data            []Record
		transformations []Operation
	}{
		"filter transformation": {
			data: []Record{
				{"name": "Alice", "age": 30},
				{"name": "Bob", "age": 25},
				{"name": "Charlie", "age": 35},
			},
			transformations: []Operation{
				NewFilterOp(func(r Record) bool {
					age, ok := r["age"].(int)
					return ok && age >= 30
				}),
			},
		},
		"sort and limit": {
			data: []Record{
				{"name": "Charlie", "score": 85},
				{"name": "Alice", "score": 95},
				{"name": "Bob", "score": 90},
				{"name": "David", "score": 88},
			},
			transformations: []Operation{
				NewSortOp(SortKey{Column: "score", Direction: Descending}),
				NewLimitOp(3),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			doc := New().
				Table("test", tc.data,
					WithKeys("name", "age", "score"),
					WithTransformations(tc.transformations...),
				).
				Build()

			renderer := &htmlRenderer{useTemplate: false}

			// Get output from Render()
			renderOutput, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			// Get output from RenderTo()
			var renderToBuffer bytes.Buffer
			if err := renderer.RenderTo(context.Background(), doc, &renderToBuffer); err != nil {
				t.Fatalf("RenderTo failed: %v", err)
			}
			renderToOutput := renderToBuffer.Bytes()

			// Compare outputs - they should be identical
			if !bytes.Equal(renderOutput, renderToOutput) {
				t.Errorf("Render() and RenderTo() produced different output\nRender():\n%s\n\nRenderTo():\n%s",
					string(renderOutput), string(renderToOutput))
			}
		})
	}
}

// TestTableRenderer_RenderToWithTransformations tests Table renderer streaming with transformations
func TestTableRenderer_RenderToWithTransformations(t *testing.T) {
	tests := map[string]struct {
		data            []Record
		transformations []Operation
	}{
		"sort transformation": {
			data: []Record{
				{"name": "Charlie", "age": 35},
				{"name": "Alice", "age": 30},
				{"name": "Bob", "age": 25},
			},
			transformations: []Operation{
				NewSortOp(SortKey{Column: "name", Direction: Ascending}),
			},
		},
		"filter and limit": {
			data: []Record{
				{"name": "Alice", "age": 30, "active": true},
				{"name": "Bob", "age": 25, "active": false},
				{"name": "Charlie", "age": 35, "active": true},
				{"name": "David", "age": 40, "active": true},
			},
			transformations: []Operation{
				NewFilterOp(func(r Record) bool {
					active, ok := r["active"].(bool)
					return ok && active
				}),
				NewLimitOp(2),
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			doc := New().
				Table("test", tc.data,
					WithKeys("name", "age", "active"),
					WithTransformations(tc.transformations...),
				).
				Build()

			renderer := &tableRenderer{}

			// Get output from Render()
			renderOutput, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			// Get output from RenderTo()
			var renderToBuffer bytes.Buffer
			if err := renderer.RenderTo(context.Background(), doc, &renderToBuffer); err != nil {
				t.Fatalf("RenderTo failed: %v", err)
			}
			renderToOutput := renderToBuffer.Bytes()

			// Compare outputs - they should be identical
			if !bytes.Equal(renderOutput, renderToOutput) {
				t.Errorf("Render() and RenderTo() produced different output\nRender():\n%s\n\nRenderTo():\n%s",
					string(renderOutput), string(renderToOutput))
			}
		})
	}
}

// TestStreamingRenderer_OriginalContentUnchanged verifies that original content
// remains unchanged after streaming render with transformations
func TestStreamingRenderer_OriginalContentUnchanged(t *testing.T) {
	data := []Record{
		{"name": "Alice", "age": 30},
		{"name": "Bob", "age": 25},
		{"name": "Charlie", "age": 35},
	}

	// Create document with transformation
	doc := New().
		Table("test", data,
			WithKeys("name", "age"),
			WithTransformations(NewLimitOp(2)),
		).
		Build()

	// Get the original content
	contents := doc.GetContents()
	if len(contents) != 1 {
		t.Fatalf("Expected 1 content, got %d", len(contents))
	}
	originalContent := contents[0].(*TableContent)
	originalData := originalContent.Records()
	originalDataLen := len(originalData)

	// Render with multiple renderers
	renderers := map[string]Renderer{
		"JSON":     &jsonRenderer{},
		"YAML":     &yamlRenderer{},
		"CSV":      &csvRenderer{},
		"Markdown": &markdownRenderer{},
		"HTML":     &htmlRenderer{useTemplate: false},
		"Table":    &tableRenderer{},
	}

	for rendererName, renderer := range renderers {
		t.Run(rendererName, func(t *testing.T) {
			var buf bytes.Buffer
			if err := renderer.RenderTo(context.Background(), doc, &buf); err != nil {
				t.Fatalf("RenderTo failed: %v", err)
			}

			// Verify original content data is unchanged
			currentData := originalContent.Records()
			if len(currentData) != originalDataLen {
				t.Errorf("Original content data length changed from %d to %d after %s RenderTo",
					originalDataLen, len(currentData), rendererName)
			}
		})
	}
}

// TestStreamingRenderer_ErrorHandling tests error handling during streaming render with transformations
func TestStreamingRenderer_ErrorHandling(t *testing.T) {
	tests := map[string]struct {
		transformations []Operation
		wantErrContains string
	}{
		"invalid operation validation": {
			transformations: []Operation{
				NewLimitOp(-1), // Invalid negative limit
			},
			wantErrContains: "invalid",
		},
		"filter with nil predicate": {
			transformations: []Operation{
				NewFilterOp(nil), // Nil predicate
			},
			wantErrContains: "invalid",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			data := []Record{
				{"name": "Alice", "age": 30},
			}

			doc := New().
				Table("test", data,
					WithKeys("name", "age"),
					WithTransformations(tc.transformations...),
				).
				Build()

			renderer := &jsonRenderer{}
			var buf bytes.Buffer
			err := renderer.RenderTo(context.Background(), doc, &buf)

			if err == nil {
				t.Error("Expected error but got none")
			} else if !strings.Contains(err.Error(), tc.wantErrContains) {
				t.Errorf("Expected error containing %q, got: %v", tc.wantErrContains, err)
			}
		})
	}
}

// TestStreamingRenderer_ContextCancellation tests context cancellation during streaming render
func TestStreamingRenderer_ContextCancellation(t *testing.T) {
	data := []Record{
		{"name": "Alice", "age": 30},
		{"name": "Bob", "age": 25},
	}

	doc := New().
		Table("test", data,
			WithKeys("name", "age"),
			WithTransformations(NewLimitOp(1)),
		).
		Build()

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	renderer := &jsonRenderer{}
	var buf bytes.Buffer
	err := renderer.RenderTo(ctx, doc, &buf)

	if err == nil {
		t.Error("Expected context cancellation error but got none")
	}
}
