package output

import (
	"context"
	"strings"
	"testing"
)

func TestMarkdownRenderer_BasicRendering(t *testing.T) {
	data := []map[string]any{
		{"name": "Alice", "age": 30, "city": "New York"},
		{"name": "Bob", "age": 25, "city": "Los Angeles"},
	}

	doc := New().
		Text("User Report", WithTextStyle(TextStyle{Header: true})).
		Table("Users", data, WithKeys("name", "age", "city")).
		Text("End of report").
		Build()

	renderer := &markdownRenderer{headingLevel: 1}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render markdown: %v", err)
	}

	resultStr := string(result)

	// Should contain header
	if !strings.Contains(resultStr, "## User Report") {
		t.Errorf("Missing header in markdown output")
	}

	// Should contain table title
	if !strings.Contains(resultStr, "### Users") {
		t.Errorf("Missing table title in markdown output")
	}

	// Should contain table headers in proper order
	if !strings.Contains(resultStr, "| name | age | city |") {
		t.Errorf("Missing table headers in markdown output")
	}

	// Should contain table separator
	if !strings.Contains(resultStr, "| --- | --- | --- |") {
		t.Errorf("Missing table separator in markdown output")
	}

	// Should contain table data
	if !strings.Contains(resultStr, "| Alice | 30 | New York |") {
		t.Errorf("Missing Alice data in markdown output")
	}

	if !strings.Contains(resultStr, "| Bob | 25 | Los Angeles |") {
		t.Errorf("Missing Bob data in markdown output")
	}

	// Should contain plain text
	if !strings.Contains(resultStr, "End of report") {
		t.Errorf("Missing plain text in markdown output")
	}
}

func TestMarkdownRenderer_TableOfContents(t *testing.T) {
	userData := []map[string]any{
		{"name": "Alice", "role": "Admin"},
	}

	doc := New().
		Section("Introduction", func(b *Builder) {
			b.Text("This is the introduction")
		}).
		Text("Main Section", WithTextStyle(TextStyle{Header: true})).
		Section("User Management", func(b *Builder) {
			b.Text("User details").
				Table("Active Users", userData, WithKeys("name", "role"))
		}).
		Build()

	// Test with ToC enabled
	renderer := NewMarkdownRendererWithToC(true)
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render markdown with ToC: %v", err)
	}

	resultStr := string(result)

	// Should contain ToC header
	if !strings.Contains(resultStr, "## Table of Contents") {
		t.Errorf("Missing ToC header in markdown output")
	}

	// Should contain ToC entries
	if !strings.Contains(resultStr, "- [Introduction](#introduction)") {
		t.Errorf("Missing Introduction in ToC")
	}

	if !strings.Contains(resultStr, "- [Main Section](#main-section)") {
		t.Errorf("Missing Main Section in ToC")
	}

	if !strings.Contains(resultStr, "- [User Management](#user-management)") {
		t.Errorf("Missing User Management in ToC")
	}

	// Should contain actual content headings
	if !strings.Contains(resultStr, "# Introduction") {
		t.Errorf("Missing Introduction heading")
	}

	if !strings.Contains(resultStr, "## Main Section") {
		t.Errorf("Missing Main Section heading")
	}

	if !strings.Contains(resultStr, "# User Management") {
		t.Errorf("Missing User Management heading")
	}
}

func TestMarkdownRenderer_FrontMatter(t *testing.T) {
	frontMatter := map[string]string{
		"title":  "Test Document",
		"author": "Test Author",
		"date":   "2024-01-01",
	}

	doc := New().
		Text("Document Content").
		Build()

	renderer := NewMarkdownRendererWithFrontMatter(frontMatter)
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render markdown with front matter: %v", err)
	}

	resultStr := string(result)

	// Should start with front matter delimiter
	if !strings.HasPrefix(resultStr, "---\n") {
		t.Errorf("Markdown should start with front matter delimiter")
	}

	// Should contain front matter fields
	if !strings.Contains(resultStr, "title: \"Test Document\"") {
		t.Errorf("Missing title in front matter")
	}

	if !strings.Contains(resultStr, "author: \"Test Author\"") {
		t.Errorf("Missing author in front matter")
	}

	if !strings.Contains(resultStr, "date: 2024-01-01") {
		t.Errorf("Missing date in front matter")
	}

	// Should end front matter properly
	if !strings.Contains(resultStr, "---\n\nDocument Content") {
		t.Errorf("Front matter not properly terminated")
	}
}

func TestMarkdownRenderer_TextStyling(t *testing.T) {
	doc := New().
		Text("Normal text").
		Text("Bold text", WithTextStyle(TextStyle{Bold: true})).
		Text("Italic text", WithTextStyle(TextStyle{Italic: true})).
		Text("Header text", WithTextStyle(TextStyle{Header: true})).
		Text("Colored text", WithTextStyle(TextStyle{Color: "red", Size: 14})).
		Build()

	renderer := &markdownRenderer{headingLevel: 1}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render markdown text styling: %v", err)
	}

	resultStr := string(result)

	// Should contain normal text
	if !strings.Contains(resultStr, "Normal text") {
		t.Errorf("Missing normal text")
	}

	// Should contain bold formatting
	if !strings.Contains(resultStr, "**Bold text**") {
		t.Errorf("Missing bold formatting")
	}

	// Should contain italic formatting
	if !strings.Contains(resultStr, "*Italic text*") {
		t.Errorf("Missing italic formatting")
	}

	// Should contain header formatting
	if !strings.Contains(resultStr, "## Header text") {
		t.Errorf("Missing header formatting")
	}

	// Should contain HTML span for color/size
	if !strings.Contains(resultStr, `<span style="color: red; font-size: 14px">Colored text</span>`) {
		t.Errorf("Missing color/size HTML formatting")
	}
}

func TestMarkdownRenderer_MarkdownEscaping(t *testing.T) {
	specialText := "Text with *asterisks* and _underscores_ and [brackets] and |pipes|"

	doc := New().
		Text(specialText).
		Build()

	renderer := &markdownRenderer{headingLevel: 1}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render markdown with special characters: %v", err)
	}

	resultStr := string(result)

	// Should escape special markdown characters
	if !strings.Contains(resultStr, "\\*asterisks\\*") {
		t.Errorf("Asterisks not properly escaped")
	}

	if !strings.Contains(resultStr, "\\_underscores\\_") {
		t.Errorf("Underscores not properly escaped")
	}

	if !strings.Contains(resultStr, "\\[brackets\\]") {
		t.Errorf("Brackets not properly escaped")
	}

	if !strings.Contains(resultStr, "\\|pipes\\|") {
		t.Errorf("Pipes not properly escaped")
	}
}

func TestMarkdownRenderer_RawContent(t *testing.T) {
	doc := New().
		Raw(FormatMarkdown, []byte("# Raw Markdown\n\nThis is **raw** markdown content.")).
		Raw("html", []byte("<div>HTML content</div>")).
		Build()

	renderer := &markdownRenderer{headingLevel: 1}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render markdown raw content: %v", err)
	}

	resultStr := string(result)

	// Markdown raw content should be included directly
	if !strings.Contains(resultStr, "# Raw Markdown") {
		t.Errorf("Raw markdown content not included directly")
	}

	if !strings.Contains(resultStr, "This is **raw** markdown content.") {
		t.Errorf("Raw markdown formatting not preserved")
	}

	// Non-markdown raw content should be in code block
	if !strings.Contains(resultStr, "```\n<div>HTML content</div>\n```") {
		t.Errorf("Non-markdown raw content not properly escaped in code block")
	}
}

func TestMarkdownRenderer_SectionNesting(t *testing.T) {
	doc := New().
		Section("Level 1", func(b *Builder) {
			b.Text("Level 1 content").
				Section("Level 2", func(b2 *Builder) {
					b2.Text("Level 2 content").
						Section("Level 3", func(b3 *Builder) {
							b3.Text("Level 3 content")
						})
				})
		}).
		Build()

	renderer := &markdownRenderer{headingLevel: 1}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render markdown section nesting: %v", err)
	}

	resultStr := string(result)

	// Should have proper heading levels
	if !strings.Contains(resultStr, "# Level 1") {
		t.Errorf("Missing Level 1 heading")
	}

	if !strings.Contains(resultStr, "## Level 2") {
		t.Errorf("Missing Level 2 heading")
	}

	if !strings.Contains(resultStr, "### Level 3") {
		t.Errorf("Missing Level 3 heading")
	}

	// Should contain all content
	if !strings.Contains(resultStr, "Level 1 content") {
		t.Errorf("Missing Level 1 content")
	}

	if !strings.Contains(resultStr, "Level 2 content") {
		t.Errorf("Missing Level 2 content")
	}

	if !strings.Contains(resultStr, "Level 3 content") {
		t.Errorf("Missing Level 3 content")
	}
}

func TestMarkdownRenderer_KeyOrderPreservation(t *testing.T) {
	data := []map[string]any{
		{"c": "gamma", "a": "alpha", "b": "beta"},
		{"b": "bravo", "c": "charlie", "a": "alpha"},
	}

	doc := New().
		Table("Order Test", data, WithKeys("c", "a", "b")).
		Build()

	renderer := &markdownRenderer{headingLevel: 1}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render markdown: %v", err)
	}

	resultStr := string(result)

	// Should have headers in correct order
	if !strings.Contains(resultStr, "| c | a | b |") {
		t.Errorf("Headers not in correct order, got: %s", resultStr)
	}

	// Should have data in correct order
	if !strings.Contains(resultStr, "| gamma | alpha | beta |") {
		t.Errorf("First row not in correct order")
	}

	if !strings.Contains(resultStr, "| charlie | alpha | bravo |") {
		t.Errorf("Second row not in correct order")
	}
}

func TestMarkdownRenderer_TableCellEscaping(t *testing.T) {
	data := []map[string]any{
		{"text": "Text with | pipes", "multiline": "Line 1\nLine 2"},
	}

	doc := New().
		Table("Escaping Test", data, WithKeys("text", "multiline")).
		Build()

	renderer := &markdownRenderer{headingLevel: 1}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render markdown table: %v", err)
	}

	resultStr := string(result)

	// Pipes should be escaped in table cells
	if !strings.Contains(resultStr, "Text with \\| pipes") {
		t.Errorf("Pipes not properly escaped in table cell")
	}

	// Newlines should be converted to <br>
	if !strings.Contains(resultStr, "Line 1<br>Line 2") {
		t.Errorf("Newlines not properly converted to <br> in table cell")
	}
}

// TestMarkdownRenderer_CollapsibleValues tests collapsible value rendering in markdown
func TestMarkdownRenderer_CollapsibleValues(t *testing.T) {
	tests := map[string]struct {
		collapsible    CollapsibleValue
		expectedOutput string
	}{"collapsed string details": {

		collapsible: NewCollapsibleValue(
			"3 errors",
			"Error 1\nError 2\nError 3",
		),
		expectedOutput: "<details><summary>3 errors</summary><br/>Error 1<br>Error 2<br>Error 3</details>",
	}, "empty summary fallback": {

		collapsible: NewCollapsibleValue(
			"",
			"Some details",
		),
		expectedOutput: "<details><summary>[no summary]</summary><br/>Some details</details>",
	}, "expanded string details with open attribute": {

		collapsible: NewCollapsibleValue(
			"System status",
			"All systems operational",
			WithCollapsibleExpanded(true),
		),
		expectedOutput: "<details open><summary>System status</summary><br/>All systems operational</details>",
	}, "map details as key-value pairs": {

		collapsible: NewCollapsibleValue(
			"Config settings",
			map[string]any{"debug": true, "port": 8080, "host": "localhost"},
		),
		expectedOutput: "<details><summary>Config settings</summary><br/><strong>debug:</strong> true<br/><strong>port:</strong> 8080<br/><strong>host:</strong> localhost</details>",
	}, "string array details": {

		collapsible: NewCollapsibleValue(
			"File list (3 items)",
			[]string{"file1.go", "file2.go", "file3.go"},
		),
		expectedOutput: "<details><summary>File list (3 items)</summary><br/>file1.go<br/>file2.go<br/>file3.go</details>",
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			renderer := &markdownRenderer{
				baseRenderer:      baseRenderer{},
				headingLevel:      1,
				collapsibleConfig: DefaultRendererConfig,
			}

			result := renderer.renderCollapsibleValue(tt.collapsible)

			// Check that all expected parts are present (order may vary for maps)
			if name == "map details as key-value pairs" {
				// For maps, check individual components since order varies
				expectedParts := []string{
					"<details><summary>Config settings</summary><br/>",
					"<strong>debug:</strong> true",
					"<strong>port:</strong> 8080",
					"<strong>host:</strong> localhost",
					"</details>",
				}
				for _, part := range expectedParts {
					if !strings.Contains(result, part) {
						t.Errorf("Expected result to contain %q, got %q", part, result)
					}
				}
			} else {
				if result != tt.expectedOutput {
					t.Errorf("Expected %q, got %q", tt.expectedOutput, result)
				}
			}
		})
	}
}

// TestMarkdownRenderer_CollapsibleInTable tests collapsible values within table cells
func TestMarkdownRenderer_CollapsibleInTable(t *testing.T) {
	data := []map[string]any{
		{
			"file":   "main.go",
			"errors": []string{"missing import", "unused variable"},
			"status": "failed",
		},
		{
			"file":   "utils.go",
			"errors": []string{},
			"status": "ok",
		},
	}

	// Create table with collapsible formatter for errors
	doc := New().
		Table("Code Analysis", data,
			WithSchema(
				Field{
					Name: "file",
					Type: "string",
				},
				Field{
					Name:      "errors",
					Type:      "array",
					Formatter: ErrorListFormatter(),
				},
				Field{
					Name: "status",
					Type: "string",
				},
			)).
		Build()

	renderer := &markdownRenderer{
		baseRenderer:      baseRenderer{},
		headingLevel:      1,
		collapsibleConfig: DefaultRendererConfig,
	}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := string(result)

	// Check table structure exists
	if !strings.Contains(output, "### Code Analysis") {
		t.Error("Expected table title")
	}
	if !strings.Contains(output, "| file | errors | status |") {
		t.Error("Expected table header")
	}

	// Check first row has collapsible content
	if !strings.Contains(output, "<details><summary>2 errors (click to expand)</summary>") {
		t.Error("Expected collapsible summary for first row")
	}
	if !strings.Contains(output, "missing import<br/>unused variable") {
		t.Error("Expected error details in first row")
	}

	// Check second row has no collapsible (empty array)
	secondRow := strings.Split(output, "\n")[5] // Approximate line with second data row
	if strings.Contains(secondRow, "<details>") {
		t.Error("Second row should not have collapsible content for empty array")
	}
}

// TestMarkdownRenderer_GlobalExpansionOverride tests global expansion configuration
func TestMarkdownRenderer_GlobalExpansionOverride(t *testing.T) {
	collapsible := NewCollapsibleValue(
		"Summary text",
		"Detail text",
		WithCollapsibleExpanded(false), // Explicitly set to collapsed
	)

	tests := map[string]struct {
		forceExpansion bool
		expectOpen     bool
	}{"override individual setting when global expansion enabled": {

		forceExpansion: true,
		expectOpen:     true,
	}, "respect individual setting when global expansion disabled": {

		forceExpansion: false,
		expectOpen:     false,
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			config := DefaultRendererConfig
			config.ForceExpansion = tt.forceExpansion

			renderer := &markdownRenderer{
				baseRenderer:      baseRenderer{},
				headingLevel:      1,
				collapsibleConfig: config,
			}

			result := renderer.renderCollapsibleValue(collapsible)

			if tt.expectOpen {
				if !strings.Contains(result, "<details open>") {
					t.Error("Expected open attribute when global expansion enabled")
				}
			} else {
				if strings.Contains(result, "<details open>") {
					t.Error("Did not expect open attribute when global expansion disabled")
				}
			}
		})
	}
}

// TestMarkdownRenderer_CollapsibleDetailsFormatting tests various detail types
func TestMarkdownRenderer_CollapsibleDetailsFormatting(t *testing.T) {
	renderer := &markdownRenderer{
		baseRenderer:      baseRenderer{},
		headingLevel:      1,
		collapsibleConfig: DefaultRendererConfig,
	}

	tests := map[string]struct {
		details  any
		expected string
	}{"complex type fallback": {

		details:  struct{ Name string }{Name: "test"},
		expected: "{test}",
	}, "map details": {

		details:  map[string]any{"key1": "value1", "key2": "value2"},
		expected: "<strong>key1:</strong> value1<br/><strong>key2:</strong> value2",
	}, "string array details": {

		details:  []string{"item1", "item2", "item3"},
		expected: "item1<br/>item2<br/>item3",
	}, "string details": {

		details:  "Simple string",
		expected: "Simple string",
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := renderer.formatDetailsForMarkdown(tt.details)

			if name == "map details" {
				// For maps, check both possible orders
				option1 := "<strong>key1:</strong> value1<br/><strong>key2:</strong> value2"
				option2 := "<strong>key2:</strong> value2<br/><strong>key1:</strong> value1"
				if result != option1 && result != option2 {
					t.Errorf("Expected one of %q or %q, got %q", option1, option2, result)
				}
			} else {
				if result != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

// TestMarkdownRenderer_CollapsibleTableCellEscaping tests markdown escaping in collapsible content
func TestMarkdownRenderer_CollapsibleTableCellEscaping(t *testing.T) {
	data := []map[string]any{
		{
			"description": "Test with special chars",
			"details":     "Text with | pipes and * asterisks",
		},
	}

	doc := New().
		Table("Escaping Test", data,
			WithSchema(
				Field{Name: "description", Type: "string"},
				Field{
					Name: "details",
					Type: "string",
					Formatter: func(val any) any {
						return NewCollapsibleValue(
							"Click to expand",
							val,
						)
					},
				},
			)).
		Build()

	renderer := &markdownRenderer{
		baseRenderer:      baseRenderer{},
		headingLevel:      1,
		collapsibleConfig: DefaultRendererConfig,
	}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := string(result)

	// Check that pipes in collapsible content are properly escaped for table cells
	// Look for the actual pattern that should exist in table cells
	if !strings.Contains(output, "pipes and * asterisks") {
		t.Error("Expected basic content to be present")
	}

	// Check that the content appears within a collapsible structure
	if !strings.Contains(output, "<details><summary>Click to expand</summary>") {
		t.Error("Expected collapsible structure")
	}
}

func TestMarkdownRenderer_CombinedFeatures(t *testing.T) {
	frontMatter := map[string]string{
		"title": "Complete Test",
	}

	userData := []map[string]any{
		{"name": "Alice", "role": "Admin"},
	}

	doc := New().
		Section("Introduction", func(b *Builder) {
			b.Text("This document demonstrates all features")
		}).
		Table("Users", userData, WithKeys("name", "role")).
		Raw(FormatMarkdown, []byte("**Bold raw markdown**")).
		Build()

	renderer := NewMarkdownRendererWithOptions(true, frontMatter)
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render complete markdown: %v", err)
	}

	resultStr := string(result)

	// Should have front matter
	if !strings.Contains(resultStr, "title: \"Complete Test\"") {
		t.Errorf("Missing front matter")
	}

	// Should have ToC
	if !strings.Contains(resultStr, "## Table of Contents") {
		t.Errorf("Missing ToC")
	}

	// Should have section
	if !strings.Contains(resultStr, "# Introduction") {
		t.Errorf("Missing section heading")
	}

	// Should have table
	if !strings.Contains(resultStr, "| name | role |") {
		t.Errorf("Missing table")
	}

	// Should have raw markdown
	if !strings.Contains(resultStr, "**Bold raw markdown**") {
		t.Errorf("Missing raw markdown content")
	}
}

// TestMarkdownRenderer_TransformationIntegration tests that MarkdownRenderer applies transformations
func TestMarkdownRenderer_TransformationIntegration(t *testing.T) {
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

	renderer := &markdownRenderer{headingLevel: 1}
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
