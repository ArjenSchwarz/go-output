package output

import (
	"context"
	"strings"
	"testing"
)

// TestHTMLRenderer_CollapsibleValue tests the HTML renderer's handling of CollapsibleValue interface
func TestHTMLRenderer_CollapsibleValue(t *testing.T) {
	tests := []struct {
		name           string
		collapsible    CollapsibleValue
		config         RendererConfig
		expectedOutput string
		checkOpen      bool
		checkClasses   bool
	}{
		{
			name:           "collapsed by default",
			collapsible:    NewCollapsibleValue("2 errors", []string{"syntax error", "missing import"}),
			config:         DefaultRendererConfig,
			expectedOutput: `<details class="collapsible-cell"><summary class="collapsible-summary">2 errors</summary><br/><div class="collapsible-details">syntax error<br/>missing import</div></details>`,
			checkOpen:      false,
			checkClasses:   true,
		},
		{
			name:           "expanded by default",
			collapsible:    NewCollapsibleValue("File path", "/very/long/path/to/file.txt", WithExpanded(true)),
			config:         DefaultRendererConfig,
			expectedOutput: `<details open class="collapsible-cell"><summary class="collapsible-summary">File path</summary><br/><div class="collapsible-details">/very/long/path/to/file.txt</div></details>`,
			checkOpen:      true,
			checkClasses:   true,
		},
		{
			name:        "force expansion override",
			collapsible: NewCollapsibleValue("Settings", "configuration data"),
			config: RendererConfig{
				ForceExpansion: true,
				HTMLCSSClasses: DefaultRendererConfig.HTMLCSSClasses,
			},
			expectedOutput: `<details open class="collapsible-cell"><summary class="collapsible-summary">Settings</summary><br/><div class="collapsible-details">configuration data</div></details>`,
			checkOpen:      true,
			checkClasses:   true,
		},
		{
			name:        "custom CSS classes",
			collapsible: NewCollapsibleValue("Data", "Some content"),
			config: RendererConfig{
				HTMLCSSClasses: map[string]string{
					"details": "custom-details",
					"summary": "custom-summary",
					"content": "custom-content",
				},
			},
			expectedOutput: `<details class="custom-details"><summary class="custom-summary">Data</summary><br/><div class="custom-content">Some content</div></details>`,
			checkOpen:      false,
			checkClasses:   true,
		},
		{
			name:           "empty array details",
			collapsible:    NewCollapsibleValue("Empty list", []string{}),
			config:         DefaultRendererConfig,
			expectedOutput: `<details class="collapsible-cell"><summary class="collapsible-summary">Empty list</summary><br/><div class="collapsible-details"></div></details>`,
			checkOpen:      false,
			checkClasses:   true,
		},
		{
			name:           "nil details fallback",
			collapsible:    NewCollapsibleValue("Test", nil),
			config:         DefaultRendererConfig,
			expectedOutput: `<details class="collapsible-cell"><summary class="collapsible-summary">Test</summary><br/><div class="collapsible-details">Test</div></details>`,
			checkOpen:      false,
			checkClasses:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := NewHTMLRendererWithCollapsible(tt.config)
			htmlRenderer := renderer.(*htmlRenderer)

			result := htmlRenderer.renderCollapsibleValue(tt.collapsible)

			if result != tt.expectedOutput {
				t.Errorf("Expected:\n%s\nGot:\n%s", tt.expectedOutput, result)
			}

			if tt.checkOpen {
				if !strings.Contains(result, " open") {
					t.Error("Expected open attribute in details element")
				}
			} else {
				// Should not contain open attribute when collapsed
				if strings.Contains(result, " open") && !tt.config.ForceExpansion {
					t.Error("Unexpected open attribute in details element")
				}
			}

			if tt.checkClasses {
				classes := tt.config.HTMLCSSClasses
				if classes == nil {
					classes = DefaultRendererConfig.HTMLCSSClasses
				}

				for _, className := range classes {
					if !strings.Contains(result, className) {
						t.Errorf("Expected CSS class %s in output", className)
					}
				}
			}
		})
	}
}

// TestHTMLRenderer_FormatDetailsAsHTML tests the formatting of different detail types
func TestHTMLRenderer_FormatDetailsAsHTML(t *testing.T) {
	renderer := NewHTMLRendererWithCollapsible(DefaultRendererConfig)
	htmlRenderer := renderer.(*htmlRenderer)

	tests := []struct {
		name     string
		details  any
		expected string
	}{
		{
			name:     "string details",
			details:  "Simple text content",
			expected: "Simple text content",
		},
		{
			name:     "string array as br-separated text",
			details:  []string{"item 1", "item 2", "item 3"},
			expected: "item 1<br/>item 2<br/>item 3",
		},
		{
			name:     "empty string array",
			details:  []string{},
			expected: "",
		},
		{
			name:     "map as key-value pairs",
			details:  map[string]any{"key1": "value1", "key2": "value2"},
			expected: "<strong>key1:</strong> value1<br/><strong>key2:</strong> value2",
		},
		{
			name:     "empty map",
			details:  map[string]any{},
			expected: "",
		},
		{
			name:     "HTML escaping in strings",
			details:  "Content with <script>alert('xss')</script> tags",
			expected: "Content with &lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt; tags",
		},
		{
			name:     "HTML escaping in arrays",
			details:  []string{"<b>bold</b>", "normal text"},
			expected: "&lt;b&gt;bold&lt;/b&gt;<br/>normal text",
		},
		{
			name:     "HTML escaping in maps",
			details:  map[string]any{"<key>": "<value>"},
			expected: "<strong>&lt;key&gt;:</strong> &lt;value&gt;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := htmlRenderer.formatDetailsAsHTML(tt.details)

			if tt.name == "map as key-value pairs" || tt.name == "HTML escaping in maps" {
				// For maps, order is not guaranteed, so check that all expected parts are present
				// Check for each key-value pair in the new format
				if strings.Contains(tt.expected, "key1") {
					if !strings.Contains(result, "<strong>key1:</strong> value1") {
						t.Errorf("Expected key1-value1 pair in result: %s", result)
					}
					if !strings.Contains(result, "<strong>key2:</strong> value2") {
						t.Errorf("Expected key2-value2 pair in result: %s", result)
					}
					if !strings.Contains(result, "<br/>") {
						t.Errorf("Expected <br/> separator in result: %s", result)
					}
				}
				if strings.Contains(tt.expected, "<key>") {
					if !strings.Contains(result, "<strong>&lt;key&gt;:</strong> &lt;value&gt;") {
						t.Errorf("Expected escaped key-value pair in result: %s", result)
					}
				}
			} else {
				if result != tt.expected {
					t.Errorf("Expected: %s, Got: %s", tt.expected, result)
				}
			}
		})
	}
}

// TestHTMLRenderer_CollapsibleInTable tests CollapsibleValue in table rendering
func TestHTMLRenderer_CollapsibleInTable(t *testing.T) {
	data := []map[string]any{
		{
			"file":   "main.go",
			"errors": []string{"syntax error", "missing import"},
			"status": "failed",
		},
		{
			"file":   "utils.go",
			"errors": []string{},
			"status": "passed",
		},
	}

	// Use ErrorListFormatter for the errors field
	table, err := NewTableContent("Test Results", data,
		WithSchema(
			Field{Name: "file", Type: "string"},
			Field{Name: "errors", Type: "array", Formatter: ErrorListFormatter()},
			Field{Name: "status", Type: "string"},
		))
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	renderer := NewHTMLRendererWithCollapsible(DefaultRendererConfig)
	doc := New().AddContent(table).Build()

	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := string(result)

	// Should contain HTML table structure
	if !strings.Contains(output, "<table class=\"data-table\">") {
		t.Error("Expected HTML table structure")
	}

	// Should contain collapsible details for errors field in first row
	if !strings.Contains(output, "<details") {
		t.Error("Expected collapsible details elements in table")
	}

	// Should contain error list in details
	if !strings.Contains(output, "syntax error") {
		t.Error("Expected error content in details")
	}

	// Should properly escape HTML in regular cells
	if !strings.Contains(output, "main.go") {
		t.Error("Expected file names in table cells")
	}
}

// TestHTMLRenderer_CollapsibleConfig tests different configurations
func TestHTMLRenderer_CollapsibleConfig(t *testing.T) {
	testValue := NewCollapsibleValue("Summary", "Details content")

	configs := []struct {
		name   string
		config RendererConfig
		checks []func(string) bool
	}{
		{
			name:   "default config",
			config: DefaultRendererConfig,
			checks: []func(string) bool{
				func(s string) bool { return strings.Contains(s, "collapsible-cell") },
				func(s string) bool { return strings.Contains(s, "collapsible-summary") },
				func(s string) bool { return strings.Contains(s, "collapsible-details") },
				func(s string) bool { return !strings.Contains(s, " open") },
			},
		},
		{
			name: "force expansion",
			config: RendererConfig{
				ForceExpansion: true,
				HTMLCSSClasses: DefaultRendererConfig.HTMLCSSClasses,
			},
			checks: []func(string) bool{
				func(s string) bool { return strings.Contains(s, " open") },
			},
		},
		{
			name: "custom CSS classes",
			config: RendererConfig{
				HTMLCSSClasses: map[string]string{
					"details": "my-details",
					"summary": "my-summary",
					"content": "my-content",
				},
			},
			checks: []func(string) bool{
				func(s string) bool { return strings.Contains(s, "my-details") },
				func(s string) bool { return strings.Contains(s, "my-summary") },
				func(s string) bool { return strings.Contains(s, "my-content") },
			},
		},
	}

	for _, config := range configs {
		t.Run(config.name, func(t *testing.T) {
			renderer := NewHTMLRendererWithCollapsible(config.config)
			htmlRenderer := renderer.(*htmlRenderer)

			result := htmlRenderer.renderCollapsibleValue(testValue)

			for i, check := range config.checks {
				if !check(result) {
					t.Errorf("Check %d failed for config %s. Result: %s", i, config.name, result)
				}
			}
		})
	}
}

// TestHTMLRenderer_CollapsibleErrorHandling tests error handling and edge cases
func TestHTMLRenderer_CollapsibleErrorHandling(t *testing.T) {
	renderer := NewHTMLRendererWithCollapsible(DefaultRendererConfig)
	htmlRenderer := renderer.(*htmlRenderer)

	tests := []struct {
		name        string
		collapsible CollapsibleValue
		shouldPanic bool
	}{
		{
			name:        "empty summary fallback",
			collapsible: NewCollapsibleValue("", "details"),
			shouldPanic: false,
		},
		{
			name:        "nil details",
			collapsible: NewCollapsibleValue("summary", nil),
			shouldPanic: false,
		},
		{
			name:        "complex nested data",
			collapsible: NewCollapsibleValue("nested", map[string]any{"level1": map[string]any{"level2": "value"}}),
			shouldPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil && !tt.shouldPanic {
					t.Errorf("Unexpected panic: %v", r)
				}
			}()

			result := htmlRenderer.renderCollapsibleValue(tt.collapsible)

			if !tt.shouldPanic {
				// Should always produce valid HTML structure
				if !strings.Contains(result, "<details") || !strings.Contains(result, "</details>") {
					t.Errorf("Expected valid details structure, got: %s", result)
				}

				// Should always have summary and content divs
				if !strings.Contains(result, "<summary") || !strings.Contains(result, "<div class=") {
					t.Errorf("Expected summary and content structure, got: %s", result)
				}
			}
		})
	}
}
