package output

import (
	"context"
	"strings"
	"testing"
)

// TestCollapsibleValue_CodeFences tests the code fence wrapping functionality
func TestCollapsibleValue_CodeFences(t *testing.T) {
	tests := map[string]struct {
		summary        string
		details        any
		options        []CollapsibleOption
		expectCode     bool
		expectedLang   string
		expectedOutput string
	}{"array details with Go code fence": {

		summary: "Error Stack",
		details: []string{
			"func main() {",
			"    fmt.Println(\"Hello\")",
			"}",
		},
		options:        []CollapsibleOption{WithCodeFences("go")},
		expectCode:     true,
		expectedLang:   "go",
		expectedOutput: "func main() {\n    fmt.Println(\"Hello\")\n}",
	}, "code fence without language":

	// Map order is not guaranteed, so we'll check individual parts

	{

		summary:        "Generic Code",
		details:        "SELECT * FROM users;",
		options:        []CollapsibleOption{WithCodeFences("")},
		expectCode:     true,
		expectedLang:   "",
		expectedOutput: "SELECT * FROM users;",
	}, "explicitly disable code fences": {

		summary:        "No Code",
		details:        "func test() {}",
		options:        []CollapsibleOption{WithoutCodeFences()},
		expectCode:     false,
		expectedOutput: "func test() {}",
	}, "map details with YAML code fence": {

		summary: "Settings",
		details: map[string]any{
			"name":    "test-app",
			"version": "1.0.0",
			"debug":   true,
		},
		options:      []CollapsibleOption{WithCodeFences("yaml")},
		expectCode:   true,
		expectedLang: "yaml",
	}, "no code fence when not specified": {

		summary:        "Plain Text",
		details:        "This is plain text without code fences",
		options:        []CollapsibleOption{},
		expectCode:     false,
		expectedOutput: "This is plain text without code fences",
	}, "string details with JSON code fence": {

		summary:        "Configuration",
		details:        `{"server": "localhost", "port": 8080}`,
		options:        []CollapsibleOption{WithCodeFences("json")},
		expectCode:     true,
		expectedLang:   "json",
		expectedOutput: `{"server": "localhost", "port": 8080}`,
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cv := NewCollapsibleValue(tt.summary, tt.details, tt.options...)

			// Test getter methods
			if cv.UseCodeFences() != tt.expectCode {
				t.Errorf("UseCodeFences() = %v, want %v", cv.UseCodeFences(), tt.expectCode)
			}

			if cv.CodeLanguage() != tt.expectedLang {
				t.Errorf("CodeLanguage() = %q, want %q", cv.CodeLanguage(), tt.expectedLang)
			}

			// Test HTML rendering
			t.Run("HTML", func(t *testing.T) {
				renderer := NewHTMLRendererWithCollapsible(DefaultRendererConfig)
				htmlRenderer := renderer.(*htmlRenderer)
				result := htmlRenderer.renderCollapsibleValue(cv)

				// Check for code blocks in HTML
				if tt.expectCode {
					if !strings.Contains(result, "<pre><code") {
						t.Error("Expected <pre><code> in HTML output")
					}
					if tt.expectedLang != "" && !strings.Contains(result, `class="language-`+tt.expectedLang) {
						t.Errorf("Expected language class %q in HTML output", tt.expectedLang)
					}
				} else {
					if strings.Contains(result, "<pre><code") {
						t.Error("Unexpected <pre><code> in HTML output")
					}
				}

				// Check content (for non-map types)
				if _, isMap := tt.details.(map[string]any); !isMap && tt.expectedOutput != "" {
					// For HTML, the content will be HTML-escaped
					// html.EscapeString uses &#34; for quotes
					// Need to escape & first, then other characters
					escapedOutput := tt.expectedOutput
					escapedOutput = strings.ReplaceAll(escapedOutput, "&", "&amp;")
					escapedOutput = strings.ReplaceAll(escapedOutput, "<", "&lt;")
					escapedOutput = strings.ReplaceAll(escapedOutput, ">", "&gt;")
					escapedOutput = strings.ReplaceAll(escapedOutput, "\"", "&#34;")
					escapedOutput = strings.ReplaceAll(escapedOutput, "'", "&#39;")
					if !strings.Contains(result, escapedOutput) {
						t.Errorf("Expected content to contain escaped version of %q\nActual result: %s", tt.expectedOutput, result)
					}
				}
			})

			// Test Markdown rendering
			t.Run("Markdown", func(t *testing.T) {
				renderer := NewMarkdownRendererWithCollapsible(DefaultRendererConfig)
				mdRenderer := renderer.(*markdownRenderer)
				result := mdRenderer.renderCollapsibleValue(cv)

				// Check for code fences in Markdown
				if tt.expectCode {
					if !strings.Contains(result, "```") {
						t.Error("Expected ``` code fence in Markdown output")
					}
					if tt.expectedLang != "" && !strings.Contains(result, "```"+tt.expectedLang) {
						t.Errorf("Expected language %q in code fence", tt.expectedLang)
					}
				} else {
					// Check that code fences are not present in the details section
					// Note: The result includes HTML details tags, so we need to be careful
					detailsStart := strings.Index(result, "</summary>")
					if detailsStart > 0 {
						detailsSection := result[detailsStart:]
						if strings.Contains(detailsSection, "```") {
							t.Error("Unexpected ``` code fence in Markdown output")
						}
					}
				}

				// Check content (for non-map types)
				if _, isMap := tt.details.(map[string]any); !isMap && tt.expectedOutput != "" {
					if !strings.Contains(result, tt.expectedOutput) {
						t.Errorf("Expected content %q in result", tt.expectedOutput)
					}
				}
			})
		})
	}
}

// TestCollapsibleValue_CodeFencesInTable tests code fence rendering in tables
func TestCollapsibleValue_CodeFencesInTable(t *testing.T) {
	// Create a custom formatter that returns collapsible values with code fences
	codeFormatter := func(val any) any {
		if code, ok := val.(string); ok {
			return NewCollapsibleValue("Code Sample", code, WithCodeFences("go"))
		}
		return val
	}

	data := []map[string]any{
		{
			"function": "main",
			"code":     "func main() {\n    fmt.Println(\"Hello, World!\")\n}",
		},
		{
			"function": "helper",
			"code":     "func helper() string {\n    return \"helper\"\n}",
		},
	}

	table, err := NewTableContent("Code Examples", data,
		WithSchema(
			Field{Name: "function", Type: "string"},
			Field{Name: "code", Type: "string", Formatter: codeFormatter},
		))
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Test HTML rendering
	t.Run("HTML Table", func(t *testing.T) {
		renderer := NewHTMLRendererWithCollapsible(DefaultRendererConfig)
		doc := New().AddContent(table).Build()

		result, err := renderer.Render(context.Background(), doc)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}

		output := string(result)

		// Should contain code blocks within details elements
		if !strings.Contains(output, "<pre><code") {
			t.Error("Expected <pre><code> blocks in table")
		}

		// Should have language class
		if !strings.Contains(output, `class="language-go"`) {
			t.Error("Expected language-go class in code blocks")
		}

		// Should contain the actual code
		if !strings.Contains(output, "fmt.Println") {
			t.Error("Expected code content in output")
		}
	})

	// Test Markdown rendering
	t.Run("Markdown Table", func(t *testing.T) {
		renderer := NewMarkdownRendererWithCollapsible(DefaultRendererConfig)
		doc := New().AddContent(table).Build()

		result, err := renderer.Render(context.Background(), doc)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}

		output := string(result)

		// Should contain code fences
		if !strings.Contains(output, "```go") {
			t.Errorf("Expected ```go code fences in table, got output:\n%s", output)
		}

		// Should contain the actual code
		if !strings.Contains(output, "fmt.Println") {
			t.Error("Expected code content in output")
		}
	})
}

// TestCollapsibleValue_CodeFencesTruncation tests truncation with code fences
func TestCollapsibleValue_CodeFencesTruncation(t *testing.T) {
	longCode := strings.Repeat("// This is a very long line of code\n", 50)

	cv := NewCollapsibleValue("Long Code",
		longCode,
		WithCodeFences("go"),
		WithMaxLength(100),
		WithTruncateIndicator("...[truncated]"))

	// Test that truncation works with code fences
	config := RendererConfig{
		MaxDetailLength:   100,
		TruncateIndicator: "...[truncated]",
		HTMLCSSClasses:    DefaultRendererConfig.HTMLCSSClasses,
	}

	t.Run("HTML Truncation", func(t *testing.T) {
		renderer := NewHTMLRendererWithCollapsible(config)
		htmlRenderer := renderer.(*htmlRenderer)
		result := htmlRenderer.renderCollapsibleValue(cv)

		// Should contain truncation indicator
		if !strings.Contains(result, "[truncated]") {
			t.Error("Expected truncation indicator in HTML output")
		}

		// Should still have code formatting
		if !strings.Contains(result, "<pre><code") {
			t.Error("Expected code formatting even with truncation")
		}
	})

	t.Run("Markdown Truncation", func(t *testing.T) {
		renderer := NewMarkdownRendererWithCollapsible(config)
		mdRenderer := renderer.(*markdownRenderer)
		result := mdRenderer.renderCollapsibleValue(cv)

		// Should contain truncation indicator
		if !strings.Contains(result, "[truncated]") {
			t.Error("Expected truncation indicator in Markdown output")
		}

		// Should still have code fences
		if !strings.Contains(result, "```go") {
			t.Error("Expected code fences even with truncation")
		}
	})
}

// TestMarkdownCodeFenceEscaping verifies that content in code fences is not escaped
func TestMarkdownCodeFenceEscaping(t *testing.T) {
	// JSON content with characters that would normally be escaped
	jsonContent := `{"server": "localhost", "port": 8080, "enabled": true, "tags": ["api", "web"]}`

	// Test WITH code fences
	cvWithFences := NewCollapsibleValue("JSON Config", jsonContent, WithCodeFences("json"))
	renderer := NewMarkdownRendererWithCollapsible(DefaultRendererConfig)
	mdRenderer := renderer.(*markdownRenderer)
	resultWithFences := mdRenderer.renderCollapsibleValue(cvWithFences)

	// Should contain unescaped content within code fences
	if !strings.Contains(resultWithFences, `"server": "localhost"`) {
		t.Error("Expected unescaped content in code fences")
	}
	if !strings.Contains(resultWithFences, `["api", "web"]`) {
		t.Error("Expected unescaped array brackets in code fences")
	}
	if !strings.Contains(resultWithFences, "```json") {
		t.Error("Expected json code fence")
	}

	// Test WITHOUT code fences for comparison
	cvWithoutFences := NewCollapsibleValue("JSON Config", jsonContent)
	resultWithoutFences := mdRenderer.renderCollapsibleValue(cvWithoutFences)

	// Should still contain the content but might have some escaping in HTML context
	if !strings.Contains(resultWithoutFences, "localhost") {
		t.Error("Expected content in non-code-fence version")
	}

	// Verify code fences are only in the fenced version
	if strings.Contains(resultWithoutFences, "```") {
		t.Error("Unexpected code fences in non-fenced version")
	}

	t.Logf("With fences:\n%s", resultWithFences)
	t.Logf("Without fences:\n%s", resultWithoutFences)
}

// TestMarkdownCodeFenceNewlines verifies that newlines are preserved in code fences
func TestMarkdownCodeFenceNewlines(t *testing.T) {
	// Multi-line content with actual newlines
	codeContent := "func main() {\n    fmt.Println(\"Hello\")\n    fmt.Println(\"World\")\n}"

	// Test WITH code fences
	cvWithFences := NewCollapsibleValue("Go Code", codeContent, WithCodeFences("go"))
	renderer := NewMarkdownRendererWithCollapsible(DefaultRendererConfig)
	mdRenderer := renderer.(*markdownRenderer)
	resultWithFences := mdRenderer.renderCollapsibleValue(cvWithFences)

	// Should contain actual newlines, not <br> tags
	if strings.Contains(resultWithFences, "<br>") {
		t.Errorf("Code fences should not contain <br> tags. Found: %s", resultWithFences)
	}

	// Should contain the actual newlines
	if !strings.Contains(resultWithFences, "func main() {\n    fmt.Println") {
		t.Error("Code fences should preserve literal newlines")
	}

	// Test WITHOUT code fences for comparison
	cvWithoutFences := NewCollapsibleValue("Go Code", codeContent)
	resultWithoutFences := mdRenderer.renderCollapsibleValue(cvWithoutFences)

	// Without code fences, it might use <br> tags (which is acceptable)
	// but let's see what actually happens

	t.Logf("With fences:\n%s", resultWithFences)
	t.Logf("Without fences:\n%s", resultWithoutFences)
}
