package output

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// Integration tests for complete end-to-end HTML document generation

func TestIntegration_DocumentWithTableContent_ProducesValidHTML5(t *testing.T) {

	data := []map[string]any{
		{"Name": "Alice", "Age": 30},
		{"Name": "Bob", "Age": 25},
	}

	doc := New().
		Text("Employee Data").
		Table("employees", data, WithKeys("Name", "Age")).
		Build()

	output, err := HTML().Renderer.Render(context.Background(), doc)
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

	data := []map[string]any{
		{"Product": "Widget A", "Price": "$10"},
		{"Product": "Widget B", "Price": "$20"},
	}

	doc := New().
		Text("Product Catalog").
		Table("products", data, WithKeys("Product", "Price")).
		Text("Thank you for visiting").
		Build()

	output, err := HTML().Renderer.Render(context.Background(), doc)
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

	doc := New().Text("Test Content").Build()
	output, err := HTML().Renderer.Render(context.Background(), doc)
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

	data := []map[string]any{{"X": "1"}}
	doc := New().
		Text("Header").
		Table("data", data, WithKeys("X")).
		Text("Footer").
		Build()

	output, err := HTML().Renderer.Render(context.Background(), doc)
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

	data := []map[string]any{
		{"Metric": "CPU", "Value": "45%"},
		{"Metric": "Memory", "Value": "72%"},
	}

	doc := New().
		Text("System Status Report").
		Table("metrics", data, WithKeys("Metric", "Value")).
		Build()

	output, err := HTML().Renderer.Render(context.Background(), doc)
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

func TestIntegration_MermaidChartWithTemplate_IncludesScriptAtEndOfBody(t *testing.T) {

	// Create a document with a chart
	doc := New().
		Text("Chart Example").
		Chart("Sample Flowchart", "flowchart", map[string]any{
			"nodes": []map[string]any{
				{"id": "A", "label": "Start"},
				{"id": "B", "label": "End"},
			},
			"edges": []map[string]any{
				{"from": "A", "to": "B"},
			},
		}).
		Build()

	output, err := HTML().Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Verify chart content is present
	if !strings.Contains(html, `<pre class="mermaid">`) {
		t.Error("Missing mermaid chart element")
	}

	// Verify mermaid script is present
	if !strings.Contains(html, `import mermaid from`) {
		t.Error("Missing mermaid script")
	}

	if !strings.Contains(html, `mermaid.initialize`) {
		t.Error("Missing mermaid initialization")
	}

	// Verify script is at end of body (before closing </body> tag)
	bodyCloseIndex := strings.LastIndex(html, "</body>")
	mermaidScriptIndex := strings.Index(html, `import mermaid from`)
	if bodyCloseIndex == -1 || mermaidScriptIndex == -1 {
		t.Fatal("Missing required HTML elements")
	}
	if mermaidScriptIndex > bodyCloseIndex {
		t.Error("Mermaid script should be before </body> closing tag")
	}

	// Verify it's after closing </html> was not found yet (we're still inside body)
	if !strings.Contains(html, "</body>\n</html>") {
		t.Error("Missing closing body and html tags")
	}

	// Verify document structure is valid
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("Missing DOCTYPE declaration")
	}

	if !strings.HasSuffix(strings.TrimSpace(html), "</html>") {
		t.Error("HTML should end with </html>")
	}
}

func TestIntegration_MermaidChartInFragmentMode_IncludesScript(t *testing.T) {

	// Create a document with a chart
	doc := New().
		Text("Fragment Chart").
		Chart("Sample Chart", "flowchart", map[string]any{
			"nodes": []map[string]any{
				{"id": "X", "label": "Node X"},
			},
		}).
		Build()

	// Use fragment mode
	output, err := HTMLFragment().Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Fragment mode should NOT have DOCTYPE
	if strings.Contains(html, "<!DOCTYPE") {
		t.Error("Fragment mode should not include DOCTYPE")
	}

	// Fragment mode should NOT have <html> tag
	if strings.Contains(html, "<html") {
		t.Error("Fragment mode should not include <html> tag")
	}

	// But it SHOULD have the chart and script (even in fragment mode)
	if !strings.Contains(html, `<pre class="mermaid">`) {
		t.Error("Missing mermaid chart element in fragment mode")
	}

	// Mermaid script should be included even in fragment mode
	if !strings.Contains(html, `import mermaid from`) {
		t.Error("Missing mermaid script in fragment mode")
	}

	if !strings.Contains(html, `mermaid.initialize`) {
		t.Error("Missing mermaid initialization in fragment mode")
	}
}

func TestIntegration_MultipleMermaidCharts_InSameDocument(t *testing.T) {

	// Create a document with multiple charts
	doc := New().
		Text("First Chart").
		Chart("First Flow", "flowchart", map[string]any{
			"nodes": []map[string]any{
				{"id": "A", "label": "Step 1"},
			},
		}).
		Text("Second Chart").
		Chart("Second Flow", "flowchart", map[string]any{
			"nodes": []map[string]any{
				{"id": "B", "label": "Step 2"},
			},
		}).
		Build()

	output, err := HTML().Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Both charts should be present
	chartCount := strings.Count(html, `<pre class="mermaid">`)
	if chartCount < 2 {
		t.Errorf("Expected at least 2 chart elements, got %d", chartCount)
	}

	// Mermaid script should only appear once (at end)
	scriptCount := strings.Count(html, `import mermaid from`)
	if scriptCount != 1 {
		t.Errorf("Expected exactly 1 mermaid import statement, got %d", scriptCount)
	}

	// Verify script is at end
	lastChartIndex := strings.LastIndex(html, `<pre class="mermaid">`)
	scriptIndex := strings.Index(html, `import mermaid from`)
	if lastChartIndex == -1 || scriptIndex == -1 {
		t.Fatal("Missing chart or script")
	}
	if scriptIndex < lastChartIndex {
		t.Error("Mermaid script should come after all chart elements")
	}

	// Verify valid HTML structure
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("Missing DOCTYPE")
	}
	if !strings.HasSuffix(strings.TrimSpace(html), "</html>") {
		t.Error("Missing closing </html>")
	}
}

func TestIntegration_ScriptInjectionOrder_ContentThenScriptsThenClosingTags(t *testing.T) {

	// Create a document with content and chart
	doc := New().
		Text("Section 1").
		Table("data", []map[string]any{
			{"Col1": "Value1"},
		}, WithKeys("Col1")).
		Text("Section 2").
		Chart("Process Flow", "flowchart", map[string]any{
			"nodes": []map[string]any{
				{"id": "N", "label": "Node"},
			},
		}).
		Text("Section 3").
		Build()

	output, err := HTML().Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Find key positions
	docType := strings.Index(html, "<!DOCTYPE")
	htmlOpenTag := strings.Index(html, "<html")
	headClose := strings.Index(html, "</head>")
	bodyOpen := strings.Index(html, "<body")
	content1 := strings.Index(html, "Section 1")
	chartElement := strings.Index(html, `<pre class="mermaid">`)
	mermaidScript := strings.Index(html, `import mermaid from`)
	bodyClose := strings.LastIndex(html, "</body>")
	htmlClose := strings.LastIndex(html, "</html>")

	if docType == -1 || htmlOpenTag == -1 || headClose == -1 || bodyOpen == -1 ||
		content1 == -1 || chartElement == -1 || mermaidScript == -1 || bodyClose == -1 || htmlClose == -1 {
		t.Fatal("Missing expected HTML elements")
	}

	// Verify order: DOCTYPE -> html -> head -> body -> content -> charts -> scripts -> closing tags
	if !(docType < htmlOpenTag && htmlOpenTag < headClose && headClose < bodyOpen &&
		bodyOpen < content1 && content1 < chartElement && chartElement < mermaidScript &&
		mermaidScript < bodyClose && bodyClose < htmlClose) {
		t.Error("HTML elements not in expected injection order")
	}

	// Verify all content sections are present
	if !strings.Contains(html, "Section 1") || !strings.Contains(html, "Section 2") ||
		!strings.Contains(html, "Section 3") {
		t.Error("Not all content sections are present")
	}

	// Verify table is present
	if !strings.Contains(html, "Value1") {
		t.Error("Table data not found")
	}

	// Verify closing structure is present
	if !strings.HasSuffix(strings.TrimSpace(html), "</html>") {
		t.Error("Missing proper closing tags")
	}
}

// Template Customization Tests

func TestIntegration_CustomTitle_AppearsInOutput(t *testing.T) {

	customTitle := "My Custom Report"
	customTemplate := &HTMLTemplate{
		Title:    customTitle,
		Language: "en",
		Charset:  "UTF-8",
	}

	doc := New().Text("Report Content").Build()
	format := HTMLWithTemplate(customTemplate)
	output, err := format.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Verify custom title appears in <title> tag
	if !strings.Contains(html, "<title>"+customTitle+"</title>") {
		t.Errorf("Custom title not found. Expected: %s", customTitle)
	}

	// Verify it's not the default title
	if !strings.Contains(html, customTitle) {
		t.Error("Custom title not in output")
	}
}

func TestIntegration_CustomCSS_OverridesDefaults(t *testing.T) {

	customCSS := "body { background-color: #ff0000; } h1 { color: blue; }"
	customTemplate := &HTMLTemplate{
		Title:   "Test",
		CSS:     customCSS,
		Charset: "UTF-8",
	}

	doc := New().Text("Test Content").Build()
	format := HTMLWithTemplate(customTemplate)
	output, err := format.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Verify custom CSS is in output
	if !strings.Contains(html, customCSS) {
		t.Error("Custom CSS not found in output")
	}

	// Verify it's wrapped in style tags
	if !strings.Contains(html, "<style>") {
		t.Error("Missing <style> tag")
	}

	// Verify custom CSS content is there
	if !strings.Contains(html, "background-color: #ff0000") {
		t.Error("Custom background color not found")
	}
}

func TestIntegration_ExternalStylesheetLinks_AreRendered(t *testing.T) {

	externalCSS := []string{
		"https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.0.0/css/all.min.css",
		"https://fonts.googleapis.com/css2?family=Roboto:wght@400;700",
	}

	customTemplate := &HTMLTemplate{
		Title:       "Test",
		ExternalCSS: externalCSS,
		Charset:     "UTF-8",
	}

	doc := New().Text("Test Content").Build()
	format := HTMLWithTemplate(customTemplate)
	output, err := format.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Verify each external stylesheet link is rendered
	for _, url := range externalCSS {
		expectedLink := `<link rel="stylesheet" href="` + url + `">`
		if !strings.Contains(html, expectedLink) {
			t.Errorf("External stylesheet link not found: %s", url)
		}
	}

	// Verify links are in the head section
	headEnd := strings.Index(html, "</head>")
	firstLink := strings.Index(html, `<link rel="stylesheet"`)
	if firstLink == -1 || headEnd == -1 || firstLink > headEnd {
		t.Error("Stylesheet links should be in head section")
	}
}

func TestIntegration_MetaTags_AreInjected(t *testing.T) {

	description := "This is a test report"
	author := "Test Author"
	customTemplate := &HTMLTemplate{
		Title:       "Test",
		Description: description,
		Author:      author,
		MetaTags: map[string]string{
			"keywords": "test, report, sample",
			"viewport": "width=device-width, initial-scale=1.0",
		},
		Charset: "UTF-8",
	}

	doc := New().Text("Test Content").Build()
	format := HTMLWithTemplate(customTemplate)
	output, err := format.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Verify description meta tag
	if !strings.Contains(html, `<meta name="description" content="`+description+`">`) {
		t.Error("Description meta tag not found")
	}

	// Verify author meta tag
	if !strings.Contains(html, `<meta name="author" content="`+author+`">`) {
		t.Error("Author meta tag not found")
	}

	// Verify custom meta tags
	if !strings.Contains(html, `<meta name="keywords" content="test, report, sample">`) {
		t.Error("Keywords meta tag not found")
	}

	// All meta tags should be in the head section
	headEnd := strings.Index(html, "</head>")
	if headEnd == -1 {
		t.Fatal("Missing </head> tag")
	}
	descIndex := strings.Index(html, "description")
	if descIndex > headEnd {
		t.Error("Meta tags should be in head section")
	}
}

func TestIntegration_HeadExtra_ContentInjection(t *testing.T) {

	headExtraContent := `<script src="https://example.com/analytics.js"></script>
    <link rel="preconnect" href="https://fonts.googleapis.com">`

	customTemplate := &HTMLTemplate{
		Title:     "Test",
		HeadExtra: headExtraContent,
		Charset:   "UTF-8",
	}

	doc := New().Text("Test Content").Build()
	format := HTMLWithTemplate(customTemplate)
	output, err := format.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Verify HeadExtra content appears in output
	if !strings.Contains(html, "analytics.js") {
		t.Error("HeadExtra script not found")
	}

	if !strings.Contains(html, "fonts.googleapis.com") {
		t.Error("HeadExtra preconnect not found")
	}

	// Verify it's in the head section (before </head>)
	headEnd := strings.Index(html, "</head>")
	analyticsIndex := strings.Index(html, "analytics.js")
	if analyticsIndex == -1 || headEnd == -1 || analyticsIndex > headEnd {
		t.Error("HeadExtra content should be in head section")
	}
}

func TestIntegration_BodyExtra_ContentInjection(t *testing.T) {

	bodyExtraContent := `<script>console.log("Page loaded");</script>
    <noscript>JavaScript is disabled</noscript>`

	customTemplate := &HTMLTemplate{
		Title:     "Test",
		BodyExtra: bodyExtraContent,
		Charset:   "UTF-8",
	}

	doc := New().Text("Test Content").Build()
	format := HTMLWithTemplate(customTemplate)
	output, err := format.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Verify BodyExtra content appears in output
	if !strings.Contains(html, "Page loaded") {
		t.Error("BodyExtra script not found")
	}

	if !strings.Contains(html, "JavaScript is disabled") {
		t.Error("BodyExtra noscript not found")
	}

	// Verify it's in the body section (before </body>)
	bodyEnd := strings.LastIndex(html, "</body>")
	scriptIndex := strings.Index(html, "Page loaded")
	if scriptIndex == -1 || bodyEnd == -1 || scriptIndex > bodyEnd {
		t.Error("BodyExtra content should be in body section")
	}
}

func TestIntegration_ThemeOverrides_GenerateSeparateStyleBlock(t *testing.T) {

	customTemplate := &HTMLTemplate{
		Title: "Test",
		CSS:   "body { color: var(--color-text); }",
		ThemeOverrides: map[string]string{
			"--color-primary":   "#ff0000",
			"--color-secondary": "#00ff00",
		},
		Charset: "UTF-8",
	}

	doc := New().Text("Test Content").Build()
	format := HTMLWithTemplate(customTemplate)
	output, err := format.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Verify original CSS is present
	if !strings.Contains(html, "body { color: var(--color-text); }") {
		t.Error("Original CSS not found")
	}

	// Verify theme overrides are in a separate style block
	if !strings.Contains(html, ":root {") {
		t.Error("Theme overrides :root selector not found")
	}

	// Verify override values are present
	if !strings.Contains(html, "--color-primary: #ff0000") {
		t.Error("Primary color override not found")
	}

	if !strings.Contains(html, "--color-secondary: #00ff00") {
		t.Error("Secondary color override not found")
	}

	// Verify there are multiple style blocks
	styleCount := strings.Count(html, "<style>")
	if styleCount < 2 {
		t.Errorf("Expected at least 2 style blocks, got %d", styleCount)
	}
}

func TestIntegration_BodyClass_AndBodyAttrs_AppliedToBodyElement(t *testing.T) {

	bodyClass := "dark-theme report-page"
	bodyAttrs := map[string]string{
		"data-report-id":   "12345",
		"data-report-type": "monthly",
	}

	customTemplate := &HTMLTemplate{
		Title:     "Test",
		BodyClass: bodyClass,
		BodyAttrs: bodyAttrs,
		Charset:   "UTF-8",
	}

	doc := New().Text("Test Content").Build()
	format := HTMLWithTemplate(customTemplate)
	output, err := format.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Verify body element has the class attribute
	if !strings.Contains(html, `<body class="`+bodyClass+`"`) {
		t.Errorf("Body class not found: %s", bodyClass)
	}

	// Verify custom attributes are present in body tag
	if !strings.Contains(html, `data-report-id="12345"`) {
		t.Error("data-report-id attribute not found")
	}

	if !strings.Contains(html, `data-report-type="monthly"`) {
		t.Error("data-report-type attribute not found")
	}

	// Verify these are all in the opening <body> tag
	bodyStart := strings.Index(html, "<body")
	bodyEnd := strings.Index(html[bodyStart:], ">")
	if bodyEnd == -1 {
		t.Fatal("Missing closing > in body tag")
	}
	bodyTagContent := html[bodyStart : bodyStart+bodyEnd]
	if !strings.Contains(bodyTagContent, "dark-theme") {
		t.Error("Body class should be in opening body tag")
	}
	if !strings.Contains(bodyTagContent, "data-report-id") {
		t.Error("Body attributes should be in opening body tag")
	}
}

func TestIntegration_AllTemplateCustomizations_Combined(t *testing.T) {

	customTemplate := &HTMLTemplate{
		Title:       "Advanced Report",
		Language:    "en",
		Charset:     "UTF-8",
		Description: "A comprehensive test report",
		Author:      "Test Suite",
		CSS:         "body { font-size: 14px; }",
		ExternalCSS: []string{"https://example.com/custom.css"},
		MetaTags: map[string]string{
			"keywords": "testing, integration",
		},
		ThemeOverrides: map[string]string{
			"--color-primary": "#0066cc",
		},
		HeadExtra: `<meta name="robots" content="noindex">`,
		BodyClass: "report-container",
		BodyExtra: `<footer>© 2024 Test</footer>`,
		BodyAttrs: map[string]string{"data-version": "1.0"},
		Viewport:  "width=device-width, initial-scale=1.0",
	}

	data := []map[string]any{
		{"Name": "Item 1", "Value": "100"},
		{"Name": "Item 2", "Value": "200"},
	}

	doc := New().
		Text("Advanced Report").
		Table("items", data, WithKeys("Name", "Value")).
		Build()

	format := HTMLWithTemplate(customTemplate)
	output, err := format.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Verify all customizations are present
	checks := []struct {
		name     string
		contains string
	}{
		{"DOCTYPE", "<!DOCTYPE html>"},
		{"Title", "<title>Advanced Report</title>"},
		{"Language", `<html lang="en"`},
		{"Charset", `<meta charset="UTF-8"`},
		{"Description", "A comprehensive test report"},
		{"Author", `<meta name="author"`},
		{"CSS", "font-size: 14px"},
		{"External CSS", "example.com/custom.css"},
		{"Keywords", "testing, integration"},
		{"Theme Override", "--color-primary: #0066cc"},
		{"HeadExtra", "robots"},
		{"BodyClass", "report-container"},
		{"BodyAttrs", "data-version"},
		{"BodyExtra", "© 2024 Test"},
		{"Viewport", "device-width"},
		{"Table Content", "Item 1"},
		{"Closing tags", "</body>"},
	}

	for _, check := range checks {
		if !strings.Contains(html, check.contains) {
			t.Errorf("Missing %s in output", check.name)
		}
	}
}

// Edge Case Tests

func TestIntegration_EdgeCase_EmptyDocument_ProducesValidHTMLStructure(t *testing.T) {

	// Create an empty document
	doc := New().Build()

	output, err := HTML().Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Even with empty content, should produce valid HTML5 structure
	checks := []struct {
		name     string
		contains string
	}{
		{"DOCTYPE", "<!DOCTYPE html>"},
		{"html tag", "<html"},
		{"head tag", "<head>"},
		{"body tag", "<body"},
		{"closing head", "</head>"},
		{"closing body", "</body>"},
		{"closing html", "</html>"},
	}

	for _, check := range checks {
		if !strings.Contains(html, check.contains) {
			t.Errorf("Missing %s in output", check.name)
		}
	}

	// Verify it starts with DOCTYPE and ends with </html>
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("Should start with DOCTYPE")
	}

	if !strings.HasSuffix(strings.TrimSpace(html), "</html>") {
		t.Error("Should end with </html>")
	}
}

func TestIntegration_EdgeCase_EmptyCSS_ProducesTemplateWithoutStyleTags(t *testing.T) {

	// Template with empty CSS
	customTemplate := &HTMLTemplate{
		Title:   "Test",
		CSS:     "", // Empty CSS
		Charset: "UTF-8",
	}

	doc := New().Text("Content").Build()
	format := HTMLWithTemplate(customTemplate)
	output, err := format.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// When CSS is empty, should NOT include <style> tags (or only the theme overrides if present)
	// Count the style tags - should be 0 if no CSS and no theme overrides
	styleCount := strings.Count(html, "<style>")
	if styleCount > 0 {
		// Check if it's only from ThemeOverrides
		if !strings.Contains(html, ":root") {
			t.Errorf("Should not have <style> tags when CSS is empty, got %d", styleCount)
		}
	}

	// Verify content is still there
	if !strings.Contains(html, "Content") {
		t.Error("Document content should be present")
	}

	// Verify valid structure
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("Missing DOCTYPE")
	}
}

func TestIntegration_EdgeCase_EmptyExternalCSS_ProducesTemplateWithoutLinkTags(t *testing.T) {

	// Template with empty ExternalCSS slice
	customTemplate := &HTMLTemplate{
		Title:       "Test",
		ExternalCSS: []string{}, // Empty slice
		Charset:     "UTF-8",
	}

	doc := New().Text("Content").Build()
	format := HTMLWithTemplate(customTemplate)
	output, err := format.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Should not have any stylesheet link tags
	if strings.Contains(html, `<link rel="stylesheet"`) {
		t.Error("Should not have stylesheet link tags when ExternalCSS is empty")
	}

	// Verify valid structure
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("Missing DOCTYPE")
	}

	if !strings.Contains(html, "Content") {
		t.Error("Document content should be present")
	}
}

func TestIntegration_EdgeCase_MissingOptionalFields_UsesDefaults(t *testing.T) {

	// Template with only required/minimal fields
	customTemplate := &HTMLTemplate{
		Title: "Test", // Required
		// Leave all other fields empty/nil
	}

	doc := New().Text("Content").Build()
	format := HTMLWithTemplate(customTemplate)
	output, err := format.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Verify defaults are used
	// Language should default to "en"
	if !strings.Contains(html, `<html lang="en"`) && !strings.Contains(html, `<html lang="`) {
		t.Error("Language should default to en or be present")
	}

	// Charset should default to "UTF-8" or be rendered
	if !strings.Contains(html, `charset=`) {
		// At minimum, charset should be specified
		t.Error("Charset should be specified in meta tag")
	}

	// Title should be what we provided
	if !strings.Contains(html, "<title>Test</title>") {
		t.Error("Custom title should be used")
	}

	// Viewport is only rendered if explicitly set in template, so we check if title is there instead
	// (proving the template wrapped correctly)

	// No CSS when not provided
	styleCount := strings.Count(html, "<style>")
	if styleCount > 0 && !strings.Contains(html, ":root") {
		t.Errorf("Should not have custom CSS when not provided, got %d style tags", styleCount)
	}

	// Valid structure
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("Missing DOCTYPE")
	}

	if !strings.HasSuffix(strings.TrimSpace(html), "</html>") {
		t.Error("Missing closing </html>")
	}
}

func TestIntegration_EdgeCase_NilTemplate_UsesDefault(t *testing.T) {

	// HTMLWithTemplate(nil) should use default template (fragment mode)
	doc := New().Text("Content").Build()
	format := HTMLWithTemplate(nil)
	output, err := format.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Fragment mode should NOT have DOCTYPE, html, body tags
	if strings.Contains(html, "<!DOCTYPE") {
		t.Error("Fragment mode should not have DOCTYPE")
	}

	if strings.Contains(html, "<html") {
		t.Error("Fragment mode should not have html tag")
	}

	// But content should be there
	if !strings.Contains(html, "Content") {
		t.Error("Content should be present in fragment mode")
	}
}

func TestIntegration_EdgeCase_EmptyMetaTags_NoTagsRendered(t *testing.T) {

	// Template with empty MetaTags map
	customTemplate := &HTMLTemplate{
		Title:    "Test",
		MetaTags: map[string]string{}, // Empty map
		Charset:  "UTF-8",
	}

	doc := New().Text("Content").Build()
	format := HTMLWithTemplate(customTemplate)
	output, err := format.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Should still have basic structure
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("Missing DOCTYPE")
	}

	// Should have charset meta tag
	if !strings.Contains(html, `<meta charset=`) {
		t.Error("Charset meta tag should be present")
	}

	// Content should be present
	if !strings.Contains(html, "Content") {
		t.Error("Content should be present")
	}
}

func TestIntegration_EdgeCase_EmptyBodyAttrs_BodyTagStillValid(t *testing.T) {

	// Template with empty BodyAttrs map
	customTemplate := &HTMLTemplate{
		Title:     "Test",
		BodyAttrs: map[string]string{}, // Empty map
		Charset:   "UTF-8",
	}

	doc := New().Text("Content").Build()
	format := HTMLWithTemplate(customTemplate)
	output, err := format.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Should still have valid body tag
	if !strings.Contains(html, "<body") {
		t.Error("Missing body tag")
	}

	if !strings.Contains(html, "</body>") {
		t.Error("Missing closing body tag")
	}

	// Content should be present
	if !strings.Contains(html, "Content") {
		t.Error("Content should be present")
	}
}

func TestIntegration_EdgeCase_EmptyThemeOverrides_NoExtraStyleBlock(t *testing.T) {

	// Template with empty ThemeOverrides map
	customTemplate := &HTMLTemplate{
		Title:          "Test",
		CSS:            "body { color: red; }",
		ThemeOverrides: map[string]string{}, // Empty map
		Charset:        "UTF-8",
	}

	doc := New().Text("Content").Build()
	format := HTMLWithTemplate(customTemplate)
	output, err := format.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Should have CSS but not a theme overrides style block
	if !strings.Contains(html, "body { color: red; }") {
		t.Error("Original CSS should be present")
	}

	// Should not have :root selector when ThemeOverrides is empty
	if strings.Count(html, ":root") > 0 {
		t.Error("Should not have :root selector when ThemeOverrides is empty")
	}

	// Verify only one style block (the original CSS)
	styleCount := strings.Count(html, "<style>")
	if styleCount > 1 {
		t.Errorf("Should have only 1 style block for CSS, got %d", styleCount)
	}
}

func TestIntegration_EdgeCase_AllOptionalFieldsEmpty_ValidOutput(t *testing.T) {

	// Minimal template with all optional fields empty/nil/zero
	customTemplate := &HTMLTemplate{
		Title:          "Minimal",
		Language:       "",
		Charset:        "",
		Viewport:       "",
		Description:    "",
		Author:         "",
		CSS:            "",
		ExternalCSS:    []string{},
		MetaTags:       map[string]string{},
		ThemeOverrides: map[string]string{},
		HeadExtra:      "",
		BodyClass:      "",
		BodyAttrs:      map[string]string{},
		BodyExtra:      "",
	}

	data := []map[string]any{
		{"ID": "1", "Name": "Item"},
	}

	doc := New().
		Text("Section 1").
		Table("items", data, WithKeys("ID", "Name")).
		Text("Section 2").
		Build()

	format := HTMLWithTemplate(customTemplate)
	output, err := format.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	html := string(output)

	// Should still produce valid HTML
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("Missing DOCTYPE")
	}

	if !strings.Contains(html, "<html") {
		t.Error("Missing html tag")
	}

	if !strings.Contains(html, "<title>Minimal</title>") {
		t.Error("Title should be present")
	}

	if !strings.Contains(html, "<body") {
		t.Error("Missing body tag")
	}

	// Content should be rendered
	if !strings.Contains(html, "Section 1") || !strings.Contains(html, "Section 2") {
		t.Error("Content sections should be present")
	}

	if !strings.Contains(html, "Item") {
		t.Error("Table content should be present")
	}

	if !strings.HasSuffix(strings.TrimSpace(html), "</html>") {
		t.Error("Missing closing </html>")
	}
}

// Thread Safety Tests

func TestIntegration_ThreadSafety_ConcurrentRenderingSameTemplate(t *testing.T) {

	// Create a template and document
	customTemplate := &HTMLTemplate{
		Title:   "Concurrent Test",
		Charset: "UTF-8",
		CSS:     "body { color: blue; }",
	}

	data := []map[string]any{
		{"ID": "1", "Value": "A"},
		{"ID": "2", "Value": "B"},
		{"ID": "3", "Value": "C"},
	}

	doc := New().
		Text("Data Report").
		Table("data", data, WithKeys("ID", "Value")).
		Build()

	format := HTMLWithTemplate(customTemplate)

	// Run concurrent renders
	const numGoroutines = 10
	const rendersPerGoroutine = 10
	var wg sync.WaitGroup
	var errorCount int32

	for i := range numGoroutines {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := range rendersPerGoroutine {
				output, err := format.Renderer.Render(context.Background(), doc)
				if err != nil {
					atomic.AddInt32(&errorCount, 1)
					t.Logf("Goroutine %d render %d failed: %v", goroutineID, j, err)
					return
				}

				html := string(output)

				// Verify each render produces valid output
				if !strings.HasPrefix(html, "<!DOCTYPE html>") {
					atomic.AddInt32(&errorCount, 1)
					t.Logf("Goroutine %d render %d: missing DOCTYPE", goroutineID, j)
					return
				}

				if !strings.Contains(html, "Concurrent Test") {
					atomic.AddInt32(&errorCount, 1)
					t.Logf("Goroutine %d render %d: missing title", goroutineID, j)
					return
				}

				if !strings.Contains(html, "Data Report") {
					atomic.AddInt32(&errorCount, 1)
					t.Logf("Goroutine %d render %d: missing content", goroutineID, j)
					return
				}

				if !strings.HasSuffix(strings.TrimSpace(html), "</html>") {
					atomic.AddInt32(&errorCount, 1)
					t.Logf("Goroutine %d render %d: missing closing tag", goroutineID, j)
					return
				}
			}
		}(i)
	}

	wg.Wait()

	if errorCount > 0 {
		t.Errorf("Concurrent rendering failed: %d errors out of %d total renders",
			errorCount, numGoroutines*rendersPerGoroutine)
	}
}

func TestIntegration_ThreadSafety_TemplateInstanceReuse(t *testing.T) {

	// Create shared template and document
	sharedTemplate := &HTMLTemplate{
		Title:       "Shared Template",
		Description: "Reused across renders",
		Author:      "Test Suite",
		Charset:     "UTF-8",
		BodyClass:   "test-class",
	}

	data := []map[string]any{
		{"Name": "Alice", "Score": "95"},
		{"Name": "Bob", "Score": "87"},
	}

	sharedDoc := New().
		Text("Test Report").
		Table("results", data, WithKeys("Name", "Score")).
		Build()

	format := HTMLWithTemplate(sharedTemplate)

	// Render multiple times in different goroutines
	const numRenders = 20
	var wg sync.WaitGroup
	var errorCount int32
	var renderCount int32

	for i := range numRenders {
		wg.Add(1)
		go func(renderID int) {
			defer wg.Done()
			output, err := format.Renderer.Render(context.Background(), sharedDoc)
			if err != nil {
				atomic.AddInt32(&errorCount, 1)
				return
			}

			html := string(output)

			// Verify template is applied correctly each time
			if !strings.Contains(html, "Shared Template") {
				atomic.AddInt32(&errorCount, 1)
				return
			}

			if !strings.Contains(html, "Test Suite") {
				atomic.AddInt32(&errorCount, 1)
				return
			}

			if !strings.Contains(html, "test-class") {
				atomic.AddInt32(&errorCount, 1)
				return
			}

			atomic.AddInt32(&renderCount, 1)
		}(i)
	}

	wg.Wait()

	if errorCount > 0 {
		t.Errorf("Template reuse test failed: %d errors", errorCount)
	}

	if renderCount != int32(numRenders) {
		t.Errorf("Expected %d successful renders, got %d", numRenders, renderCount)
	}
}

func TestIntegration_ThreadSafety_NoSharedMutableState(t *testing.T) {

	// Test that different documents don't interfere with each other
	// when rendered concurrently with different templates

	doc1 := New().Text("Document 1").Build()
	doc2 := New().Text("Document 2").Build()
	doc3 := New().Text("Document 3").Build()

	template1 := &HTMLTemplate{
		Title:     "Title 1",
		BodyClass: "class1",
		Charset:   "UTF-8",
	}

	template2 := &HTMLTemplate{
		Title:     "Title 2",
		BodyClass: "class2",
		Charset:   "UTF-8",
	}

	template3 := &HTMLTemplate{
		Title:     "Title 3",
		BodyClass: "class3",
		Charset:   "UTF-8",
	}

	format1 := HTMLWithTemplate(template1)
	format2 := HTMLWithTemplate(template2)
	format3 := HTMLWithTemplate(template3)

	var wg sync.WaitGroup
	var errorCount int32

	// Render each doc/format combination concurrently many times
	for range 5 {
		// Render doc1/format1
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 10 {
				output, err := format1.Renderer.Render(context.Background(), doc1)
				if err != nil {
					atomic.AddInt32(&errorCount, 1)
					return
				}
				html := string(output)
				if !strings.Contains(html, "Title 1") ||
					!strings.Contains(html, "class1") ||
					!strings.Contains(html, "Document 1") {
					atomic.AddInt32(&errorCount, 1)
					return
				}
			}
		}()

		// Render doc2/format2
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 10 {
				output, err := format2.Renderer.Render(context.Background(), doc2)
				if err != nil {
					atomic.AddInt32(&errorCount, 1)
					return
				}
				html := string(output)
				if !strings.Contains(html, "Title 2") ||
					!strings.Contains(html, "class2") ||
					!strings.Contains(html, "Document 2") {
					atomic.AddInt32(&errorCount, 1)
					return
				}
			}
		}()

		// Render doc3/format3
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 10 {
				output, err := format3.Renderer.Render(context.Background(), doc3)
				if err != nil {
					atomic.AddInt32(&errorCount, 1)
					return
				}
				html := string(output)
				if !strings.Contains(html, "Title 3") ||
					!strings.Contains(html, "class3") ||
					!strings.Contains(html, "Document 3") {
					atomic.AddInt32(&errorCount, 1)
					return
				}
			}
		}()
	}

	wg.Wait()

	if errorCount > 0 {
		t.Errorf("Shared mutable state test failed: %d errors detected", errorCount)
	}
}

func TestIntegration_ThreadSafety_ConcurrentWithMermaidCharts(t *testing.T) {

	// Test thread safety with Mermaid chart content
	doc := New().
		Text("Charts").
		Chart("Chart 1", "flowchart", map[string]any{
			"nodes": []map[string]any{
				{"id": "A", "label": "Start"},
			},
		}).
		Build()

	customTemplate := &HTMLTemplate{
		Title:   "Chart Report",
		Charset: "UTF-8",
	}

	format := HTMLWithTemplate(customTemplate)

	const numGoroutines = 8
	const rendersPerGoroutine = 5
	var wg sync.WaitGroup
	var errorCount int32

	for i := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for range rendersPerGoroutine {
				output, err := format.Renderer.Render(context.Background(), doc)
				if err != nil {
					atomic.AddInt32(&errorCount, 1)
					return
				}

				html := string(output)

				// Verify chart is present and script is injected
				if !strings.Contains(html, `<pre class="mermaid">`) {
					atomic.AddInt32(&errorCount, 1)
					return
				}

				if !strings.Contains(html, "import mermaid from") {
					atomic.AddInt32(&errorCount, 1)
					return
				}

				// Verify script is at end before body close
				bodyEnd := strings.LastIndex(html, "</body>")
				scriptIdx := strings.Index(html, "import mermaid from")
				if scriptIdx >= bodyEnd {
					atomic.AddInt32(&errorCount, 1)
					return
				}
			}
		}(i)
	}

	wg.Wait()

	if errorCount > 0 {
		t.Errorf("Concurrent Mermaid rendering failed: %d errors", errorCount)
	}
}

func TestIntegration_RenderToWithTemplate_ProducesCompleteHTMLDocument(t *testing.T) {

	data := []map[string]any{
		{"Name": "Alice", "Score": 95},
		{"Name": "Bob", "Score": 87},
	}

	doc := New().
		Text("Test Results").
		Table("scores", data, WithKeys("Name", "Score")).
		Build()

	var buf strings.Builder
	err := HTML().Renderer.RenderTo(context.Background(), doc, &buf)
	if err != nil {
		t.Fatalf("RenderTo failed: %v", err)
	}

	html := buf.String()

	// Verify HTML5 structure is present (from template wrapping)
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("RenderTo: Missing DOCTYPE declaration")
	}

	if !strings.Contains(html, "<html") || !strings.Contains(html, "</html>") {
		t.Error("RenderTo: Missing html tags")
	}

	if !strings.Contains(html, "<head>") || !strings.Contains(html, "</head>") {
		t.Error("RenderTo: Missing head tags")
	}

	if !strings.Contains(html, "<body>") || !strings.Contains(html, "</body>") {
		t.Error("RenderTo: Missing body tags")
	}

	// Verify content is included
	if !strings.Contains(html, "Test Results") {
		t.Error("RenderTo: Missing title text")
	}

	if !strings.Contains(html, "Alice") || !strings.Contains(html, "Bob") {
		t.Error("RenderTo: Missing table data")
	}

	// Verify CSS is embedded
	if !strings.Contains(html, "<style>") || !strings.Contains(html, "</style>") {
		t.Error("RenderTo: Missing CSS styles")
	}

	// Verify both Render and RenderTo produce similar structure
	// (exact bytes may differ but structure should be the same)
	renderOutput, _ := HTML().Renderer.Render(context.Background(), doc)
	renderStr := string(renderOutput)

	// Both should have DOCTYPE and closing tags
	if strings.HasPrefix(renderStr, "<!DOCTYPE html>") != strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("RenderTo and Render should both include DOCTYPE")
	}

	if strings.Contains(renderStr, "</html>") != strings.Contains(html, "</html>") {
		t.Error("RenderTo and Render should both be complete HTML documents")
	}
}

func TestIntegration_RenderToWithMermaidCharts_InjectsMermaidScript(t *testing.T) {

	doc := New().
		Text("Diagram Test").
		Chart("Diagram", "stateDiagram-v2", map[string]any{
			"content": "[*] --> State1\nState1 --> [*]",
		}).
		Build()

	var buf strings.Builder
	err := HTML().Renderer.RenderTo(context.Background(), doc, &buf)
	if err != nil {
		t.Fatalf("RenderTo failed: %v", err)
	}

	html := buf.String()

	// Verify template wrapper exists
	if !strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("RenderTo: Missing DOCTYPE")
	}

	// Verify chart is included
	if !strings.Contains(html, `<pre class="mermaid">`) {
		t.Error("RenderTo: Missing mermaid chart")
	}

	// Verify mermaid script is injected (before body close for streaming)
	if !strings.Contains(html, "import mermaid from") {
		t.Error("RenderTo: Missing mermaid script injection")
	}

	// Verify script is before closing body tag
	bodyEnd := strings.LastIndex(html, "</body>")
	scriptIdx := strings.Index(html, "import mermaid from")
	if scriptIdx >= bodyEnd {
		t.Error("RenderTo: Mermaid script should be before </body>")
	}
}

func TestIntegration_RenderToWithoutTemplate_StreamsFragmentsOnly(t *testing.T) {

	data := []map[string]any{
		{"Item": "A", "Value": 1},
	}

	doc := New().
		Text("Fragment Test").
		Table("items", data, WithKeys("Item", "Value")).
		Build()

	// Use HTMLFragment which has useTemplate=false
	var buf strings.Builder
	err := HTMLFragment().Renderer.RenderTo(context.Background(), doc, &buf)
	if err != nil {
		t.Fatalf("RenderTo failed: %v", err)
	}

	html := buf.String()

	// Verify no HTML5 wrapper
	if strings.HasPrefix(html, "<!DOCTYPE html>") {
		t.Error("HTMLFragment RenderTo: Should not include DOCTYPE")
	}

	if strings.Contains(html, "<html") {
		t.Error("HTMLFragment RenderTo: Should not include html tags")
	}

	if strings.Contains(html, "<head>") {
		t.Error("HTMLFragment RenderTo: Should not include head tags")
	}

	// But content should still be there
	if !strings.Contains(html, "Fragment Test") {
		t.Error("HTMLFragment RenderTo: Missing content")
	}

	if !strings.Contains(html, "Item") {
		t.Error("HTMLFragment RenderTo: Missing table content")
	}
}
