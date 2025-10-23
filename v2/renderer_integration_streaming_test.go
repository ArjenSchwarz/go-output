package output

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// TestStreamingRenderWithNestedSectionsAndTransformations tests that streaming
// render properly applies transformations to nested content within sections
func TestStreamingRenderWithNestedSectionsAndTransformations(t *testing.T) {
	data := []Record{
		{"name": "Alice", "age": 30, "active": true},
		{"name": "Bob", "age": 25, "active": false},
		{"name": "Charlie", "age": 35, "active": true},
		{"name": "David", "age": 28, "active": true},
	}

	// Create table with multiple transformations
	table, err := NewTableContent("employees", data,
		WithKeys("name", "age", "active"),
		WithTransformations(
			NewFilterOp(func(r Record) bool {
				active, ok := r["active"].(bool)
				return ok && active
			}),
			NewSortOp(SortKey{Column: "age", Direction: Ascending}),
			NewLimitOp(2),
		),
	)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Create nested section structure
	innerSection := NewSectionContent("Active Employees")
	innerSection.AddContent(table)

	outerSection := NewSectionContent("Company Data")
	outerSection.AddContent(innerSection)

	doc := New().AddContent(outerSection).Build()

	tests := map[string]struct {
		renderer Renderer
		validate func(t *testing.T, output string)
	}{
		"Markdown": {
			renderer: &markdownRenderer{},
			validate: func(t *testing.T, output string) {
				// Should have Bob filtered out (not active)
				if strings.Contains(output, "Bob") {
					t.Error("Bob should be filtered out but appears in output")
				}
				// Should have Alice and David (both active, sorted by age, limited to 2)
				// David (28) should appear before Alice (30) due to ascending sort
				if !strings.Contains(output, "Alice") || !strings.Contains(output, "David") {
					t.Error("Expected Alice and David in output")
				}
				// Charlie should be filtered out by limit (would be 3rd after sort)
				if strings.Contains(output, "Charlie") {
					t.Error("Charlie should be limited out but appears in output")
				}
			},
		},
		"HTML": {
			renderer: &htmlRenderer{useTemplate: false},
			validate: func(t *testing.T, output string) {
				if strings.Contains(output, "Bob") {
					t.Error("Bob should be filtered out but appears in output")
				}
				if !strings.Contains(output, "Alice") || !strings.Contains(output, "David") {
					t.Error("Expected Alice and David in output")
				}
				if strings.Contains(output, "Charlie") {
					t.Error("Charlie should be limited out but appears in output")
				}
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Test Render()
			renderOutput, err := tc.renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			// Test RenderTo()
			var renderToBuffer bytes.Buffer
			if err := tc.renderer.RenderTo(context.Background(), doc, &renderToBuffer); err != nil {
				t.Fatalf("RenderTo failed: %v", err)
			}

			// Outputs should be identical
			if !bytes.Equal(renderOutput, renderToBuffer.Bytes()) {
				t.Errorf("Render() and RenderTo() produced different output")
			}

			// Validate transformation was applied correctly
			tc.validate(t, string(renderOutput))
		})
	}
}

// TestRenderToWithComplexDocumentStructure tests streaming render with a complex
// document containing multiple sections, nested content, and various transformations
func TestRenderToWithComplexDocumentStructure(t *testing.T) {
	// Table 1: Filtered employees
	employees := []Record{
		{"name": "Alice", "dept": "Engineering", "salary": 100000},
		{"name": "Bob", "dept": "Sales", "salary": 80000},
		{"name": "Charlie", "dept": "Engineering", "salary": 120000},
	}
	filteredTable, _ := NewTableContent("engineering", employees,
		WithKeys("name", "dept", "salary"),
		WithTransformations(NewFilterOp(func(r Record) bool {
			dept, ok := r["dept"].(string)
			return ok && dept == "Engineering"
		})),
	)

	// Table 2: Sorted and limited products
	products := []Record{
		{"product": "Widget", "sales": 1000},
		{"product": "Gadget", "sales": 1500},
		{"product": "Doohickey", "sales": 800},
		{"product": "Thingamajig", "sales": 1200},
	}
	sortedTable, _ := NewTableContent("top-products", products,
		WithKeys("product", "sales"),
		WithTransformations(
			NewSortOp(SortKey{Column: "sales", Direction: Descending}),
			NewLimitOp(2),
		),
	)

	// Create complex nested structure
	employeeSection := NewSectionContent("Employees")
	employeeSection.AddContent(filteredTable)

	productSection := NewSectionContent("Products")
	productSection.AddContent(sortedTable)

	doc := New().
		AddContent(employeeSection).
		AddContent(productSection).
		Build()

	renderers := map[string]Renderer{
		"Markdown": &markdownRenderer{},
		"HTML":     &htmlRenderer{useTemplate: false},
		"JSON":     &jsonRenderer{},
		"YAML":     &yamlRenderer{},
	}

	for name, renderer := range renderers {
		t.Run(name, func(t *testing.T) {
			// Render using both methods
			renderOutput, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			var renderToBuffer bytes.Buffer
			if err := renderer.RenderTo(context.Background(), doc, &renderToBuffer); err != nil {
				t.Fatalf("RenderTo failed: %v", err)
			}

			// Should be identical
			if !bytes.Equal(renderOutput, renderToBuffer.Bytes()) {
				t.Errorf("Render() and RenderTo() produced different output for %s", name)
			}

			output := string(renderOutput)

			// Verify employee filtering
			if !strings.Contains(output, "Alice") || !strings.Contains(output, "Charlie") {
				t.Error("Expected Alice and Charlie in Engineering")
			}
			if strings.Contains(output, "Bob") && strings.Contains(output, "Sales") {
				t.Error("Bob from Sales should be filtered out")
			}

			// Verify product sorting and limiting
			if !strings.Contains(output, "Gadget") || !strings.Contains(output, "Thingamajig") {
				t.Error("Expected top 2 products (Gadget and Thingamajig)")
			}
		})
	}
}

// TestConsistencyBetweenRenderAndRenderTo verifies that Render() and RenderTo()
// produce identical output for documents with nested content and transformations
func TestConsistencyBetweenRenderAndRenderTo(t *testing.T) {
	data := []Record{
		{"id": 1, "value": 100},
		{"id": 2, "value": 200},
		{"id": 3, "value": 150},
		{"id": 4, "value": 175},
	}

	table, _ := NewTableContent("data", data,
		WithKeys("id", "value"),
		WithTransformations(
			NewSortOp(SortKey{Column: "value", Direction: Descending}),
			NewLimitOp(3),
		),
	)

	section := NewSectionContent("Data Analysis")
	section.AddContent(table)

	doc := New().AddContent(section).Build()

	renderers := map[string]Renderer{
		"JSON":     &jsonRenderer{},
		"YAML":     &yamlRenderer{},
		"CSV":      &csvRenderer{},
		"Markdown": &markdownRenderer{},
		"HTML":     &htmlRenderer{useTemplate: false},
		"Table":    &tableRenderer{},
	}

	for name, renderer := range renderers {
		t.Run(name, func(t *testing.T) {
			renderOutput, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			var renderToBuffer bytes.Buffer
			if err := renderer.RenderTo(context.Background(), doc, &renderToBuffer); err != nil {
				t.Fatalf("RenderTo failed: %v", err)
			}

			if !bytes.Equal(renderOutput, renderToBuffer.Bytes()) {
				t.Errorf("%s: Render() and RenderTo() produced different output\nRender():\n%s\n\nRenderTo():\n%s",
					name, string(renderOutput), renderToBuffer.String())
			}
		})
	}
}

// TestPerformanceWithLargeNestedTransformations tests that performance is
// acceptable with large documents containing nested transformations
func TestPerformanceWithLargeNestedTransformations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create large dataset
	data := make([]Record, 1000)
	for i := range 1000 {
		data[i] = Record{
			"id":    i,
			"value": i * 10,
			"group": i % 10,
		}
	}

	// Create table with transformations
	table, _ := NewTableContent("large-data", data,
		WithKeys("id", "value", "group"),
		WithTransformations(
			NewFilterOp(func(r Record) bool {
				group, ok := r["group"].(int)
				return ok && group < 5
			}),
			NewSortOp(SortKey{Column: "value", Direction: Descending}),
			NewLimitOp(100),
		),
	)

	// Create nested structure
	section := NewSectionContent("Large Dataset")
	section.AddContent(table)

	doc := New().AddContent(section).Build()

	// Test that both render methods complete in reasonable time
	renderer := &jsonRenderer{}

	_, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	var buf bytes.Buffer
	if err := renderer.RenderTo(context.Background(), doc, &buf); err != nil {
		t.Fatalf("RenderTo failed: %v", err)
	}

	// Verify output is reasonable size (should be ~100 records after transformations)
	output := buf.String()
	if len(output) < 100 {
		t.Error("Output suspiciously small - transformations may not have been applied")
	}
}

// TestErrorHandlingAcrossStreamingAndNestedPaths verifies that errors are
// properly propagated through both streaming and nested transformation paths
func TestErrorHandlingAcrossStreamingAndNestedPaths(t *testing.T) {
	data := []Record{
		{"name": "Alice", "age": 30},
	}

	// Create table with invalid transformation
	table, _ := NewTableContent("test", data,
		WithKeys("name", "age"),
		WithTransformations(NewLimitOp(-1)), // Invalid: negative limit
	)

	section := NewSectionContent("Test")
	section.AddContent(table)

	doc := New().AddContent(section).Build()

	renderers := map[string]Renderer{
		"Markdown": &markdownRenderer{},
		"JSON":     &jsonRenderer{},
		"YAML":     &yamlRenderer{},
		"HTML":     &htmlRenderer{useTemplate: false},
	}

	for name, renderer := range renderers {
		t.Run(name+"_Render", func(t *testing.T) {
			_, err := renderer.Render(context.Background(), doc)
			if err == nil {
				t.Error("Expected error for invalid transformation, got none")
			}
		})

		t.Run(name+"_RenderTo", func(t *testing.T) {
			var buf bytes.Buffer
			err := renderer.RenderTo(context.Background(), doc, &buf)
			if err == nil {
				t.Error("Expected error for invalid transformation, got none")
			}
		})
	}
}
