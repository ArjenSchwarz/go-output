package output

import (
	"context"
	"strings"
	"testing"
)

// Tests for HTML format constants and constructors

func TestHTML_Format_UsesTemplateByDefault(t *testing.T) {

	// Get the HTML format and render a document
	format := HTML
	doc := New().Text("Test content").Build()

	output, err := format.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Should produce full HTML document with template
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("HTML format should use template by default (should start with DOCTYPE)")
	}

	// Should contain html tag
	if !strings.Contains(html, "<html") {
		t.Error("HTML format should produce full HTML document")
	}

	// Should contain default template title
	if !strings.Contains(html, "Output Report") {
		t.Error("HTML format should use default template title")
	}
}

func TestHTMLFragment_Format_SkipsTemplateWrapping(t *testing.T) {

	// Get the HTMLFragment format and render a document
	format := HTMLFragment
	doc := New().Text("Test content").Build()

	output, err := format.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Should NOT produce full HTML document
	if strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("HTMLFragment format should NOT start with DOCTYPE")
	}

	// Should NOT contain html tag
	if strings.Contains(html, "<html") {
		t.Error("HTMLFragment format should not produce full HTML document")
	}

	// Should still contain content
	if !strings.Contains(html, "Test content") {
		t.Error("HTMLFragment format should still render content")
	}
}

func TestHTMLWithTemplate_Nil_EnablesFragmentMode(t *testing.T) {

	// Create a format with nil template (should enable fragment mode)
	format := HTMLWithTemplate(nil)
	doc := New().Text("Test content").Build()

	output, err := format.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Should produce fragment output (no DOCTYPE)
	if strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("HTMLWithTemplate(nil) should produce fragment output")
	}

	// Should contain content
	if !strings.Contains(html, "Test content") {
		t.Error("Output should contain content")
	}
}

func TestHTMLWithTemplate_Custom_UsesCustomTemplate(t *testing.T) {

	// Create a custom template
	customTemplate := &HTMLTemplate{
		Title:       "My Custom Title",
		Description: "A custom description",
		Charset:     "UTF-8",
		BodyClass:   "custom-class",
	}

	// Create a format with custom template
	format := HTMLWithTemplate(customTemplate)
	doc := New().Text("Test content").Build()

	output, err := format.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Should produce full HTML document
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("HTMLWithTemplate(custom) should produce full HTML document")
	}

	// Should use custom template values
	if !strings.Contains(html, "My Custom Title") {
		t.Error("Should use custom template title")
	}

	if !strings.Contains(html, "custom-class") {
		t.Error("Should use custom body class")
	}

	if !strings.Contains(html, "A custom description") {
		t.Error("Should use custom description")
	}
}

func TestFormatConstructor_CreatesNewRendererInstance(t *testing.T) {

	// Create two format instances with custom templates
	template1 := &HTMLTemplate{Title: "Title1", Charset: "UTF-8"}
	template2 := &HTMLTemplate{Title: "Title2", Charset: "UTF-8"}

	format1 := HTMLWithTemplate(template1)
	format2 := HTMLWithTemplate(template2)

	// Render with both formats
	doc := New().Text("Test content").Build()

	output1, err1 := format1.Renderer.Render(context.Background(), doc)
	output2, err2 := format2.Renderer.Render(context.Background(), doc)

	if err1 != nil || err2 != nil {
		t.Fatalf("Render failed: %v, %v", err1, err2)
	}

	html1 := string(output1)
	html2 := string(output2)

	// Each should use its own template
	if !strings.Contains(html1, "Title1") {
		t.Error("Format 1 should use Title1")
	}

	if !strings.Contains(html2, "Title2") {
		t.Error("Format 2 should use Title2")
	}

	// They should have different content
	if html1 == html2 {
		t.Error("Different templates should produce different output")
	}
}

func TestHTML_Format_RendersFullDocument(t *testing.T) {

	// Test that HTML format produces a complete, valid-looking HTML document
	doc := New().
		Text("Heading").
		Table("Data", []map[string]any{
			{"Name": "Alice", "Age": 30},
		}, WithKeys("Name", "Age")).
		Build()

	output, err := HTML.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Verify structure
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("Missing DOCTYPE")
	}

	if !strings.Contains(html, "<html") {
		t.Error("Missing html tag")
	}

	if !strings.Contains(html, "<head>") {
		t.Error("Missing head tag")
	}

	if !strings.Contains(html, "<body") {
		t.Error("Missing body tag")
	}

	if !strings.HasSuffix(strings.TrimSpace(html), "</html>") {
		t.Error("Missing closing html tag")
	}

	// Verify content
	if !strings.Contains(html, "Heading") {
		t.Error("Missing heading content")
	}

	if !strings.Contains(html, "Alice") {
		t.Error("Missing table content")
	}
}
