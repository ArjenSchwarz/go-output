package output

import (
	"context"
	"strings"
	"testing"
)

// Integration tests for complete end-to-end HTML document generation

func TestIntegration_DocumentWithTableContent_ProducesValidHTML5(t *testing.T) {
	t.Parallel()

	data := []map[string]any{
		{"Name": "Alice", "Age": 30},
		{"Name": "Bob", "Age": 25},
	}

	doc := New().
		Text("Employee Data").
		Table("employees", data, WithKeys("Name", "Age")).
		Build()

	output, err := HTML.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Verify HTML5 structure
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("Missing DOCTYPE declaration")
	}

	if !strings.Contains(html, "<html") || !strings.Contains(html, "</html>") {
		t.Error("Missing html tags")
	}

	if !strings.Contains(html, "<head>") || !strings.Contains(html, "</head>") {
		t.Error("Missing head tags")
	}

	if !strings.Contains(html, "<body") || !strings.Contains(html, "</body>") {
		t.Error("Missing body tags")
	}

	// Verify content
	if !strings.Contains(html, "Alice") || !strings.Contains(html, "Bob") {
		t.Error("Missing table content")
	}

	if !strings.Contains(html, "<table") || !strings.Contains(html, "</table>") {
		t.Error("Missing table tags")
	}
}

func TestIntegration_DocumentWithMultipleContentTypes(t *testing.T) {
	t.Parallel()

	data := []map[string]any{
		{"Product": "Widget A", "Price": "$10"},
		{"Product": "Widget B", "Price": "$20"},
	}

	doc := New().
		Text("Product Catalog").
		Table("products", data, WithKeys("Product", "Price")).
		Text("Thank you for visiting").
		Build()

	output, err := HTML.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Verify all content
	if !strings.Contains(html, "Product Catalog") {
		t.Error("Missing text content")
	}

	if !strings.Contains(html, "Widget A") || !strings.Contains(html, "Widget B") {
		t.Error("Missing table data")
	}

	if !strings.Contains(html, "Thank you for visiting") {
		t.Error("Missing last text")
	}

	// Verify structure
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("Missing DOCTYPE")
	}

	if !strings.HasSuffix(strings.TrimSpace(html), "</html>") {
		t.Error("Missing closing html tag")
	}
}

func TestIntegration_DOCTYPEAndMetaTags(t *testing.T) {
	t.Parallel()

	doc := New().Text("Test Content").Build()
	output, err := HTML.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Verify DOCTYPE
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("DOCTYPE should be first")
	}

	// Verify meta tags
	if !strings.Contains(html, `<meta charset=`) {
		t.Error("Missing charset meta tag")
	}

	if !strings.Contains(html, `<meta name="viewport"`) {
		t.Error("Missing viewport meta tag")
	}

	if !strings.Contains(html, "<title>") {
		t.Error("Missing title tag")
	}
}

func TestIntegration_HTMLStructureOrder(t *testing.T) {
	t.Parallel()

	data := []map[string]any{{"X": "1"}}
	doc := New().
		Text("Header").
		Table("data", data, WithKeys("X")).
		Text("Footer").
		Build()

	output, err := HTML.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Find key positions
	docType := strings.Index(html, "<!DOCTYPE")
	htmlTag := strings.Index(html, "<html")
	headStart := strings.Index(html, "<head>")
	bodyStart := strings.Index(html, "<body")
	headerText := strings.Index(html, "Header")
	bodyEnd := strings.LastIndex(html, "</body>")
	htmlEnd := strings.LastIndex(html, "</html>")

	if docType == -1 || htmlTag == -1 || headStart == -1 || bodyStart == -1 ||
		headerText == -1 || bodyEnd == -1 || htmlEnd == -1 {
		t.Fatal("Missing expected HTML elements")
	}

	// Verify order
	if !(docType < htmlTag && htmlTag < headStart && headStart < bodyStart &&
		bodyStart < headerText && headerText < bodyEnd && bodyEnd < htmlEnd) {
		t.Error("HTML elements not in expected order")
	}
}

func TestIntegration_EndToEndRenderingPipeline(t *testing.T) {
	t.Parallel()

	data := []map[string]any{
		{"Metric": "CPU", "Value": "45%"},
		{"Metric": "Memory", "Value": "72%"},
	}

	doc := New().
		Text("System Status Report").
		Table("metrics", data, WithKeys("Metric", "Value")).
		Build()

	output, err := HTML.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Verify complete structure
	checks := []struct {
		name     string
		contains string
	}{
		{"DOCTYPE", "<!DOCTYPE html>"},
		{"html tag", "<html"},
		{"head section", "<head>"},
		{"body tag", "<body"},
		{"main content", "System Status Report"},
		{"table tag", "<table"},
		{"table data", "CPU"},
		{"closing body", "</body>"},
		{"closing html", "</html>"},
	}

	for _, check := range checks {
		if !strings.Contains(html, check.contains) {
			t.Errorf("Missing %s in output", check.name)
		}
	}

	// Verify ends properly
	if !strings.HasSuffix(strings.TrimSpace(html), "</html>") {
		t.Error("HTML should end with </html>")
	}
}

func TestIntegration_TemplateFeatures(t *testing.T) {
	t.Parallel()

	customTemplate := &HTMLTemplate{
		Title:     "Custom Report",
		Author:    "Test Team",
		Charset:   "UTF-8",
		Language:  "en",
		BodyClass: "report",
	}

	doc := New().Text("Report Content").Build()

	format := HTMLWithTemplate(customTemplate)
	output, err := format.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Verify template features
	if !strings.Contains(html, "Custom Report") {
		t.Error("Missing custom title")
	}

	if !strings.Contains(html, "Test Team") {
		t.Error("Missing custom author")
	}

	if !strings.Contains(html, "report") {
		t.Error("Missing custom body class")
	}

	if !strings.Contains(html, "Report Content") {
		t.Error("Missing content")
	}
}
