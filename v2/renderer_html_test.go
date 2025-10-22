package output

import (
	"context"
	"strings"
	"testing"
)

func TestHTMLRenderer_TableEscaping(t *testing.T) {
	// Test HTML escaping with dangerous content
	data := []map[string]any{
		{"name": "<script>alert('xss')</script>", "html": "<b>Bold</b>", "safe": "Normal Text"},
		{"name": "John & Jane", "html": "A > B", "safe": "Test"},
	}

	doc := New().
		Table("Test Escaping", data, WithKeys("name", "html", "safe")).
		Build()

	renderer := &htmlRenderer{}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render HTML: %v", err)
	}

	resultStr := string(result)

	// Should escape dangerous script tags
	if strings.Contains(resultStr, "<script>") {
		t.Errorf("HTML output contains unescaped script tag")
	}

	// Should contain escaped version
	if !strings.Contains(resultStr, "&lt;script&gt;") {
		t.Errorf("HTML output doesn't contain properly escaped script tag")
	}

	// Should escape HTML entities
	if !strings.Contains(resultStr, "&amp;") {
		t.Errorf("HTML output doesn't escape & character")
	}

	if !strings.Contains(resultStr, "&gt;") {
		t.Errorf("HTML output doesn't escape > character")
	}

	// Should have proper HTML structure
	if !strings.Contains(resultStr, "<table class=\"data-table\">") {
		t.Errorf("HTML output missing table structure")
	}

	if !strings.Contains(resultStr, "<thead>") || !strings.Contains(resultStr, "<tbody>") {
		t.Errorf("HTML output missing thead/tbody structure")
	}
}

func TestHTMLRenderer_KeyOrderPreservation(t *testing.T) {
	data := []map[string]any{
		{"c": "gamma", "a": "alpha", "b": "beta"},
		{"b": "bravo", "c": "charlie", "a": "alpha"},
	}

	doc := New().
		Table("Order Test", data, WithKeys("c", "a", "b")).
		Build()

	renderer := &htmlRenderer{}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render HTML: %v", err)
	}

	resultStr := string(result)

	// Find the header row
	lines := strings.Split(resultStr, "\n")
	var headerRowStart int
	for i, line := range lines {
		if strings.Contains(line, "<th>") {
			headerRowStart = i
			break
		}
	}

	if headerRowStart == 0 {
		t.Fatalf("Could not find header row in HTML output")
	}

	// Check that headers appear in correct order by examining their positions
	cPos := -1
	aPos := -1
	bPos := -1

	for i := headerRowStart; i < len(lines) && strings.Contains(lines[i], "<th>"); i++ {
		if strings.Contains(lines[i], "<th>c</th>") {
			cPos = i
		}
		if strings.Contains(lines[i], "<th>a</th>") {
			aPos = i
		}
		if strings.Contains(lines[i], "<th>b</th>") {
			bPos = i
		}
	}

	if cPos == -1 || aPos == -1 || bPos == -1 {
		t.Errorf("Could not find all headers in HTML output")
	}

	if !(cPos < aPos && aPos < bPos) {
		t.Errorf("Headers not in correct order: c=%d, a=%d, b=%d", cPos, aPos, bPos)
	}
}

func TestHTMLRenderer_TextContentStyling(t *testing.T) {
	doc := New().
		Text("Normal Text", WithTextStyle(TextStyle{})).
		Text("Bold Text", WithTextStyle(TextStyle{Bold: true})).
		Text("Header Text", WithTextStyle(TextStyle{Header: true})).
		Text("Colored Text", WithTextStyle(TextStyle{Color: "red", Size: 14})).
		Build()

	renderer := &htmlRenderer{}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render HTML text: %v", err)
	}

	resultStr := string(result)

	// Check for proper HTML elements
	if !strings.Contains(resultStr, "<p class=\"text-content\">Normal Text</p>") {
		t.Errorf("Normal text not rendered correctly")
	}

	if !strings.Contains(resultStr, "font-weight: bold") {
		t.Errorf("Bold style not applied")
	}

	if !strings.Contains(resultStr, "<h2 class=\"text-header\">Header Text</h2>") {
		t.Errorf("Header not rendered as h2")
	}

	if !strings.Contains(resultStr, "color: red") {
		t.Errorf("Color style not applied")
	}

	if !strings.Contains(resultStr, "font-size: 14px") {
		t.Errorf("Font size not applied")
	}
}

func TestHTMLRenderer_RawContent(t *testing.T) {
	doc := New().
		Raw(FormatHTML, []byte("<div>Safe HTML</div>")).
		Raw(FormatText, []byte("<script>alert('danger')</script>")).
		Build()

	renderer := &htmlRenderer{}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render HTML raw content: %v", err)
	}

	resultStr := string(result)

	// HTML format raw content should be included directly
	if !strings.Contains(resultStr, "<div>Safe HTML</div>") {
		t.Errorf("HTML raw content not included directly")
	}

	// Non-HTML raw content should be escaped
	if strings.Contains(resultStr, "<script>alert('danger')</script>") {
		t.Errorf("Non-HTML raw content not escaped")
	}

	if !strings.Contains(resultStr, "&lt;script&gt;") {
		t.Errorf("Non-HTML raw content not properly escaped")
	}

	if !strings.Contains(resultStr, "<pre class=\"raw-content\">") {
		t.Errorf("Non-HTML raw content not wrapped in pre tag")
	}
}

func TestHTMLRenderer_SectionContent(t *testing.T) {
	userData := []map[string]any{
		{"name": "Alice", "role": "Admin"},
	}

	doc := New().
		Section("User Management", func(b *Builder) {
			b.Text("Section description").
				Table("Users", userData, WithKeys("name", "role"))
		}).
		Build()

	renderer := &htmlRenderer{}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render HTML section: %v", err)
	}

	resultStr := string(result)

	// Should have section structure
	if !strings.Contains(resultStr, "<section class=\"content-section\">") {
		t.Errorf("Section not properly structured")
	}

	if !strings.Contains(resultStr, "<h1>User Management</h1>") {
		t.Errorf("Section title not rendered as h1")
	}

	// Should contain nested content
	if !strings.Contains(resultStr, "Section description") {
		t.Errorf("Section text content missing")
	}

	if !strings.Contains(resultStr, "<table class=\"data-table\">") {
		t.Errorf("Section table content missing")
	}

	if !strings.Contains(resultStr, "Alice") {
		t.Errorf("Table data in section missing")
	}
}

// TestHTMLRenderer_TransformationIntegration tests that HTMLRenderer applies transformations
func TestHTMLRenderer_TransformationIntegration(t *testing.T) {
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

	renderer := &htmlRenderer{}
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
