package output

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestMarkdownRenderer_SectionWithTableTransformations tests that tables
// with transformations inside sections are properly transformed
func TestMarkdownRenderer_SectionWithTableTransformations(t *testing.T) {
	tests := map[string]struct {
		tableData       []Record
		transformations []Operation
		expectRecords   int
	}{
		"filter transformation on nested table": {
			tableData: []Record{
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
			expectRecords: 2, // Alice and Charlie
		},
		"limit transformation on nested table": {
			tableData: []Record{
				{"name": "Alice", "age": 30},
				{"name": "Bob", "age": 25},
				{"name": "Charlie", "age": 35},
			},
			transformations: []Operation{
				NewLimitOp(2),
			},
			expectRecords: 2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create table with transformations
			table, err := NewTableContent("test-table", tc.tableData,
				WithKeys("name", "age"),
				WithTransformations(tc.transformations...),
			)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			// Create section containing the table
			section := NewSectionContent("Test Section")
			section.AddContent(table)

			// Create document with the section
			doc := New().AddContent(section).Build()

			renderer := &markdownRenderer{}
			result, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			resultStr := string(result)

			// Count the number of data rows in the markdown table
			// Each data row should be present in the output
			lines := strings.Split(resultStr, "\n")
			dataRows := 0
			for _, line := range lines {
				// Data rows contain pipe characters and are not header or separator lines
				if strings.Contains(line, "|") && !strings.Contains(line, "---") && !strings.Contains(line, "name") {
					dataRows++
				}
			}

			if dataRows != tc.expectRecords {
				t.Errorf("Expected %d data rows after transformation, got %d\nOutput:\n%s",
					tc.expectRecords, dataRows, resultStr)
			}
		})
	}
}

// TestMarkdownRenderer_NestedSectionsWithTransformations tests multi-level nested sections
func TestMarkdownRenderer_NestedSectionsWithTransformations(t *testing.T) {
	data := []Record{
		{"name": "Alice", "age": 30},
		{"name": "Bob", "age": 25},
		{"name": "Charlie", "age": 35},
	}

	// Create table with filter transformation
	table, err := NewTableContent("test-table", data,
		WithKeys("name", "age"),
		WithTransformations(NewFilterOp(func(r Record) bool {
			age, ok := r["age"].(int)
			return ok && age >= 30
		})),
	)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Create nested section structure: Section1 > Section2 > Table
	innerSection := NewSectionContent("Inner Section")
	innerSection.AddContent(table)

	outerSection := NewSectionContent("Outer Section")
	outerSection.AddContent(innerSection)

	doc := New().AddContent(outerSection).Build()

	renderer := &markdownRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	resultStr := string(result)

	// The filtered table should only show Alice and Charlie (age >= 30), not Bob
	if strings.Contains(resultStr, "Bob") {
		t.Error("Transformation not applied: Bob should be filtered out but appears in output")
	}
	if !strings.Contains(resultStr, "Alice") || !strings.Contains(resultStr, "Charlie") {
		t.Error("Transformation incorrectly filtered expected records")
	}
}

// TestMarkdownRenderer_SectionWithMixedTransformations tests section with some
// content having transformations and some without
func TestMarkdownRenderer_SectionWithMixedTransformations(t *testing.T) {
	data := []Record{
		{"name": "Alice", "age": 30},
		{"name": "Bob", "age": 25},
		{"name": "Charlie", "age": 35},
	}

	// Table with transformation
	transformedTable, err := NewTableContent("transformed", data,
		WithKeys("name", "age"),
		WithTransformations(NewLimitOp(1)),
	)
	if err != nil {
		t.Fatalf("Failed to create transformed table: %v", err)
	}

	// Table without transformation
	untransformedTable, err := NewTableContent("untransformed", data,
		WithKeys("name", "age"),
	)
	if err != nil {
		t.Fatalf("Failed to create untransformed table: %v", err)
	}

	section := NewSectionContent("Test Section")
	section.AddContent(transformedTable)
	section.AddContent(untransformedTable)

	doc := New().AddContent(section).Build()

	renderer := &markdownRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	resultStr := string(result)

	// Both tables should be present but with different row counts
	// This is a basic sanity check - the transformed table should have fewer rows
	if !strings.Contains(resultStr, "transformed") || !strings.Contains(resultStr, "untransformed") {
		t.Error("Both tables should be present in output")
	}
}

// TestJSONRenderer_SectionWithTableTransformations tests JSON rendering of sections with nested transformations
func TestJSONRenderer_SectionWithTableTransformations(t *testing.T) {
	data := []Record{
		{"name": "Alice", "age": 30},
		{"name": "Bob", "age": 25},
		{"name": "Charlie", "age": 35},
	}

	// Create table with filter transformation
	table, err := NewTableContent("test-table", data,
		WithKeys("name", "age"),
		WithTransformations(NewFilterOp(func(r Record) bool {
			age, ok := r["age"].(int)
			return ok && age >= 30
		})),
	)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Create section containing the table
	section := NewSectionContent("Test Section")
	section.AddContent(table)

	doc := New().AddContent(section).Build()

	renderer := &jsonRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Parse JSON to verify transformation was applied
	var parsed map[string]any
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Navigate to the section contents
	contents, ok := parsed["contents"].([]any)
	if !ok || len(contents) == 0 {
		t.Fatalf("Expected contents array in JSON output, got: %v", parsed)
	}

	// Get the table from the section contents
	tableItem, ok := contents[0].(map[string]any)
	if !ok {
		t.Fatal("Expected first content to be a map")
	}

	tableData, ok := tableItem["data"].([]any)
	if !ok {
		t.Fatal("Expected data array in table")
	}

	// Should only have 2 records after filter (Alice and Charlie)
	if len(tableData) != 2 {
		t.Errorf("Expected 2 records after transformation, got %d\nOutput:\n%s", len(tableData), string(result))
	}
}

// TestYAMLRenderer_SectionWithTableTransformations tests YAML rendering of sections with nested transformations
func TestYAMLRenderer_SectionWithTableTransformations(t *testing.T) {
	data := []Record{
		{"name": "Alice", "age": 30},
		{"name": "Bob", "age": 25},
		{"name": "Charlie", "age": 35},
	}

	// Create table with sort transformation
	table, err := NewTableContent("test-table", data,
		WithKeys("name", "age"),
		WithTransformations(NewSortOp(SortKey{Column: "age", Direction: Ascending})),
	)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	section := NewSectionContent("Test Section")
	section.AddContent(table)

	doc := New().AddContent(section).Build()

	renderer := &yamlRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Parse YAML to verify transformation was applied
	var parsed map[string]any
	if err := yaml.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	// The data should be sorted by age (Bob=25, Alice=30, Charlie=35)
	resultStr := string(result)
	bobIndex := strings.Index(resultStr, "Bob")
	aliceIndex := strings.Index(resultStr, "Alice")
	charlieIndex := strings.Index(resultStr, "Charlie")

	if bobIndex == -1 || aliceIndex == -1 || charlieIndex == -1 {
		t.Fatal("Expected all names in output")
	}

	// Bob should appear first (age 25), then Alice (30), then Charlie (35)
	if !(bobIndex < aliceIndex && aliceIndex < charlieIndex) {
		t.Error("Data not sorted correctly: expected Bob < Alice < Charlie by age")
	}
}

// TestHTMLRenderer_SectionWithTableTransformations tests HTML rendering of sections with nested transformations
func TestHTMLRenderer_SectionWithTableTransformations(t *testing.T) {
	data := []Record{
		{"name": "Alice", "age": 30},
		{"name": "Bob", "age": 25},
		{"name": "Charlie", "age": 35},
	}

	// Create table with filter transformation
	table, err := NewTableContent("test-table", data,
		WithKeys("name", "age"),
		WithTransformations(NewFilterOp(func(r Record) bool {
			age, ok := r["age"].(int)
			return ok && age >= 30
		})),
	)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	section := NewSectionContent("Test Section")
	section.AddContent(table)

	doc := New().AddContent(section).Build()

	renderer := &htmlRenderer{useTemplate: false}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	resultStr := string(result)

	// Bob (age 25) should not appear in the output
	if strings.Contains(resultStr, "Bob") {
		t.Error("Transformation not applied: Bob should be filtered out but appears in output\n" + resultStr)
	}

	// Alice and Charlie should be present
	if !strings.Contains(resultStr, "Alice") || !strings.Contains(resultStr, "Charlie") {
		t.Error("Expected records missing from output")
	}
}

// TestRenderer_NestedContentErrorPropagation tests that transformation errors in nested content propagate correctly
func TestRenderer_NestedContentErrorPropagation(t *testing.T) {
	data := []Record{
		{"name": "Alice", "age": 30},
	}

	// Create table with invalid transformation
	table, err := NewTableContent("test-table", data,
		WithKeys("name", "age"),
		WithTransformations(NewLimitOp(-1)), // Invalid: negative limit
	)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	section := NewSectionContent("Test Section")
	section.AddContent(table)

	doc := New().AddContent(section).Build()

	renderers := map[string]Renderer{
		"Markdown": &markdownRenderer{},
		"JSON":     &jsonRenderer{},
		"YAML":     &yamlRenderer{},
		"HTML":     &htmlRenderer{useTemplate: false},
	}

	for name, renderer := range renderers {
		t.Run(name, func(t *testing.T) {
			_, err := renderer.Render(context.Background(), doc)
			if err == nil {
				t.Error("Expected error for invalid transformation in nested content, got none")
			} else if !strings.Contains(err.Error(), "invalid") {
				t.Errorf("Expected error to mention 'invalid', got: %v", err)
			}
		})
	}
}

// TestRenderer_CollapsibleSectionWithTransformations tests that collapsible sections apply nested transformations
func TestRenderer_CollapsibleSectionWithTransformations(t *testing.T) {
	data := []Record{
		{"name": "Alice", "age": 30},
		{"name": "Bob", "age": 25},
		{"name": "Charlie", "age": 35},
	}

	// Create table with limit transformation
	table, err := NewTableContent("test-table", data,
		WithKeys("name", "age"),
		WithTransformations(NewLimitOp(2)),
	)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Create collapsible section with the table as content
	section := NewCollapsibleSection("Collapsible Section", []Content{table})

	doc := New().AddContent(section).Build()

	renderer := &markdownRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	resultStr := string(result)

	// Count data rows - should only have 2 after limit
	lines := strings.Split(resultStr, "\n")
	dataRows := 0
	for _, line := range lines {
		if strings.Contains(line, "|") && !strings.Contains(line, "---") && !strings.Contains(line, "name") {
			dataRows++
		}
	}

	if dataRows != 2 {
		t.Errorf("Expected 2 data rows after limit transformation, got %d\nOutput:\n%s", dataRows, resultStr)
	}
}
