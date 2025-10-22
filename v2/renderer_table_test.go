package output

import (
	"context"
	"strings"
	"testing"
)

func TestTableRenderer_KeyOrderPreservation(t *testing.T) {
	tests := map[string]struct {
		keys     []string
		data     []map[string]any
		expected []string
	}{"preserve explicit key order": {

		keys: []string{"c", "a", "b"},
		data: []map[string]any{
			{"a": "alpha", "b": "beta", "c": "gamma"},
			{"c": "charlie", "b": "bravo", "a": "alpha"},
		},
		expected: []string{"c", "a", "b"},
	}, "preserve numeric and string keys": {

		keys: []string{"id", "name", "score", "active"},
		data: []map[string]any{
			{"name": "Alice", "id": 1, "active": true, "score": 95.5},
			{"score": 87.2, "id": 2, "name": "Bob", "active": false},
		},
		expected: []string{"id", "name", "score", "active"},
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create table with explicit key order
			doc := New().
				Table("Test Table", tt.data, WithKeys(tt.keys...)).
				Build()

			// Test with table renderer
			renderer := &tableRenderer{}
			ctx := context.Background()

			result, err := renderer.Render(ctx, doc)
			if err != nil {
				t.Fatalf("Failed to render table: %v", err)
			}

			resultStr := string(result)

			// Verify that the output contains the table title
			if !strings.Contains(resultStr, "Test Table") {
				t.Errorf("Output does not contain table title")
			}

			// Split into lines to check header order
			lines := strings.Split(resultStr, "\n")
			var headerLine string

			// Find the header line (should contain all our keys)
			// Look for uppercase versions since go-pretty converts headers to uppercase
			upperKeys := make([]string, len(tt.keys))
			for i, key := range tt.keys {
				upperKeys[i] = strings.ToUpper(key)
			}

			for _, line := range lines {
				if strings.Contains(line, upperKeys[0]) && strings.Contains(line, upperKeys[1]) {
					headerLine = line
					break
				}
			}

			if headerLine == "" {
				t.Errorf("Could not find header line in output. Full output:\n%s", resultStr)
				return
			}

			// Verify that keys appear in the correct order in the header
			// We check that each key appears before the next one
			// Use uppercase versions for comparison
			for i := 0; i < len(tt.expected)-1; i++ {
				key1 := strings.ToUpper(tt.expected[i])
				key2 := strings.ToUpper(tt.expected[i+1])

				pos1 := strings.Index(headerLine, key1)
				pos2 := strings.Index(headerLine, key2)

				if pos1 == -1 {
					t.Errorf("Key %s not found in header", key1)
				}
				if pos2 == -1 {
					t.Errorf("Key %s not found in header", key2)
				}
				if pos1 >= pos2 {
					t.Errorf("Key %s appears after %s in header, expected before",
						key1, key2)
				}
			}
		})
	}
}

func TestTableRenderer_MixedContent(t *testing.T) {
	// Create a document with mixed content types
	data := []map[string]any{
		{"name": "Alice", "age": 30, "city": "New York"},
		{"name": "Bob", "age": 25, "city": "Los Angeles"},
	}

	doc := New().
		Text("User Report").
		Table("Users", data, WithKeys("name", "age", "city")).
		Text("End of report").
		Build()

	renderer := &tableRenderer{}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render mixed content: %v", err)
	}

	resultStr := string(result)

	// Should contain text content
	if !strings.Contains(resultStr, "User Report") {
		t.Errorf("Output missing text content 'User Report'")
	}

	if !strings.Contains(resultStr, "End of report") {
		t.Errorf("Output missing text content 'End of report'")
	}

	// Should contain table data
	if !strings.Contains(resultStr, "Alice") {
		t.Errorf("Output missing table data 'Alice'")
	}

	if !strings.Contains(resultStr, "Bob") {
		t.Errorf("Output missing table data 'Bob'")
	}

	// Should contain table title
	if !strings.Contains(resultStr, "Users") {
		t.Errorf("Output missing table title 'Users'")
	}
}

func TestTableRenderer_SectionContent(t *testing.T) {
	// Create a document with section content
	userData := []map[string]any{
		{"name": "Alice", "role": "Admin"},
		{"name": "Bob", "role": "User"},
	}

	doc := New().
		Section("User Management", func(b *Builder) {
			b.Text("This section contains user information").
				Table("Active Users", userData, WithKeys("name", "role"))
		}).
		Build()

	renderer := &tableRenderer{}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render section content: %v", err)
	}

	resultStr := string(result)

	// Should contain section marker
	if !strings.Contains(resultStr, "=== User Management ===") {
		t.Errorf("Output missing section header")
	}

	// Should contain section text and table data
	if !strings.Contains(resultStr, "This section contains user information") {
		t.Errorf("Output missing section text")
	}

	if !strings.Contains(resultStr, "Alice") || !strings.Contains(resultStr, "Admin") {
		t.Errorf("Output missing table data from section")
	}
}

func TestTableRenderer_StyleConfiguration(t *testing.T) {
	data := []map[string]any{
		{"name": "Alice", "age": 30},
		{"name": "Bob", "age": 25},
	}

	tests := map[string]struct {
		renderer  *tableRenderer
		styleName string
	}{"bold style":

	// default

	{

		renderer:  NewTableRendererWithStyle("Bold").(*tableRenderer),
		styleName: "Bold",
	}, "default style": {

		renderer:  &tableRenderer{},
		styleName: "ColoredBright",
	}, "light style": {

		renderer:  NewTableRendererWithStyle("Light").(*tableRenderer),
		styleName: "Light",
	}, "rounded style": {

		renderer:  NewTableRendererWithStyle("Rounded").(*tableRenderer),
		styleName: "Rounded",
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			doc := New().
				Table("Styled Table", data, WithKeys("name", "age")).
				Build()

			ctx := context.Background()
			result, err := tt.renderer.Render(ctx, doc)
			if err != nil {
				t.Fatalf("Failed to render table with style: %v", err)
			}

			resultStr := string(result)

			// Should contain the data regardless of style
			if !strings.Contains(resultStr, "Alice") {
				t.Errorf("Table data missing from %s style", tt.styleName)
			}

			if !strings.Contains(resultStr, "Bob") {
				t.Errorf("Table data missing from %s style", tt.styleName)
			}

			// Should contain table title (may be split across lines with ANSI codes)
			if !strings.Contains(resultStr, "Styled") || !(strings.Contains(resultStr, "Table") || strings.Contains(resultStr, "Tabl") || strings.Contains(resultStr, "Tab")) {
				t.Errorf("Table title missing from %s style. Full output:\n%s", tt.styleName, resultStr)
			}

			// Different styles should produce different output (basic check)
			if tt.styleName != "ColoredBright" {
				// Create a default renderer for comparison
				defaultRenderer := &tableRenderer{}
				defaultResult, err := defaultRenderer.Render(ctx, doc)
				if err != nil {
					t.Fatalf("Failed to render with default style: %v", err)
				}

				// The styled output should be different from default
				// (This is a basic check - different styles will have different ANSI codes)
				if string(result) == string(defaultResult) {
					t.Errorf("Style %s produced identical output to default", tt.styleName)
				}
			}
		})
	}
}

func TestTableRenderer_PredefinedStyles(t *testing.T) {
	data := []map[string]any{
		{"id": 1, "name": "Test"},
	}

	doc := New().
		Table("Style Test", data, WithKeys("id", "name")).
		Build()

	// Test some of the predefined style formats
	styles := []Format{
		TableDefault,
		TableBold,
		TableColoredBright,
		TableLight,
		TableRounded,
	}

	ctx := context.Background()

	for _, style := range styles {
		t.Run(style.Name+"_"+style.Renderer.(*tableRenderer).styleName, func(t *testing.T) {
			result, err := style.Renderer.Render(ctx, doc)
			if err != nil {
				t.Fatalf("Failed to render with predefined style: %v", err)
			}

			resultStr := string(result)

			// Should contain the basic data
			if !strings.Contains(resultStr, "Test") {
				t.Errorf("Table data missing from predefined style")
			}

			if !strings.Contains(resultStr, "Style") || !strings.Contains(resultStr, "Test") {
				t.Errorf("Table title missing from predefined style")
			}
		})
	}
}

func TestTableWithStyle_Function(t *testing.T) {
	// Test the TableWithStyle function
	customStyle := TableWithStyle("Double")

	if customStyle.Name != FormatTable {
		t.Errorf("TableWithStyle should have name %q, got %s", FormatTable, customStyle.Name)
	}

	renderer, ok := customStyle.Renderer.(*tableRenderer)
	if !ok {
		t.Fatalf("TableWithStyle should return tableRenderer")
	}

	if renderer.styleName != "Double" {
		t.Errorf("TableWithStyle should set styleName to 'Double', got %s", renderer.styleName)
	}
}

// TestTableRenderer_TransformationIntegration tests that TableRenderer applies transformations
func TestTableRenderer_TransformationIntegration(t *testing.T) {
	data := []Record{
		{"name": "Alice", "age": 30},
		{"name": "Bob", "age": 25},
		{"name": "Charlie", "age": 35},
	}

	doc := New().
		Table("test", data,
			WithKeys("name", "age"),
			WithTransformations(
				NewFilterOp(func(r Record) bool {
					return r["age"].(int) >= 30
				}),
			),
		).
		Build()

	renderer := &tableRenderer{styleName: "Default"}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	resultStr := string(result)
	// Should contain Alice and Charlie but not Bob
	if !strings.Contains(resultStr, "Alice") {
		t.Error("Missing Alice after filter")
	}
	if !strings.Contains(resultStr, "Charlie") {
		t.Error("Missing Charlie after filter")
	}
	if strings.Contains(resultStr, "Bob") {
		t.Error("Bob should be filtered out")
	}
}
