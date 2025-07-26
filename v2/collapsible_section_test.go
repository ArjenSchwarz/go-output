package output

import (
	"reflect"
	"testing"
)

func TestCollapsibleSectionInterface(t *testing.T) {
	tests := []struct {
		name     string
		section  CollapsibleSection
		expected struct {
			title      string
			isExpanded bool
			level      int
		}
	}{
		{
			name: "basic section with defaults",
			section: NewCollapsibleSection("Test Section", []Content{
				NewTextContent("Test content"),
			}),
			expected: struct {
				title      string
				isExpanded bool
				level      int
			}{
				title:      "Test Section",
				isExpanded: false,
				level:      0,
			},
		},
		{
			name: "section with expanded option",
			section: NewCollapsibleSection("Expanded Section", []Content{
				NewTextContent("Test content"),
			}, WithSectionExpanded(true)),
			expected: struct {
				title      string
				isExpanded bool
				level      int
			}{
				title:      "Expanded Section",
				isExpanded: true,
				level:      0,
			},
		},
		{
			name: "section with level option",
			section: NewCollapsibleSection("Nested Section", []Content{
				NewTextContent("Test content"),
			}, WithSectionLevel(2)),
			expected: struct {
				title      string
				isExpanded bool
				level      int
			}{
				title:      "Nested Section",
				isExpanded: false,
				level:      2,
			},
		},
		{
			name: "empty title fallback",
			section: NewCollapsibleSection("", []Content{
				NewTextContent("Test content"),
			}),
			expected: struct {
				title      string
				isExpanded bool
				level      int
			}{
				title:      "[untitled section]",
				isExpanded: false,
				level:      0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Title()
			if got := tt.section.Title(); got != tt.expected.title {
				t.Errorf("Title() = %v, want %v", got, tt.expected.title)
			}

			// Test IsExpanded()
			if got := tt.section.IsExpanded(); got != tt.expected.isExpanded {
				t.Errorf("IsExpanded() = %v, want %v", got, tt.expected.isExpanded)
			}

			// Test Level()
			if got := tt.section.Level(); got != tt.expected.level {
				t.Errorf("Level() = %v, want %v", got, tt.expected.level)
			}
		})
	}
}

func TestCollapsibleSectionContent(t *testing.T) {
	// Create test content
	text1 := NewTextContent("First content")
	text2 := NewTextContent("Second content")
	table1, _ := NewTableContent("Test Table",
		[]map[string]any{{"col1": "value1"}},
		WithKeys("col1"))

	tests := []struct {
		name            string
		content         []Content
		expectedCount   int
		validateContent func([]Content) error
	}{
		{
			name:          "single content item",
			content:       []Content{text1},
			expectedCount: 1,
			validateContent: func(content []Content) error {
				if content[0] != text1 {
					return errorf("expected text1, got %v", content[0])
				}
				return nil
			},
		},
		{
			name:          "multiple content items",
			content:       []Content{text1, text2, table1},
			expectedCount: 3,
			validateContent: func(content []Content) error {
				if content[0] != text1 || content[1] != text2 || content[2] != table1 {
					return errorf("content items not in expected order")
				}
				return nil
			},
		},
		{
			name:          "empty content",
			content:       []Content{},
			expectedCount: 0,
			validateContent: func(content []Content) error {
				if len(content) != 0 {
					return errorf("expected empty content, got %d items", len(content))
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			section := NewCollapsibleSection("Test Section", tt.content)

			// Get content
			content := section.Content()

			// Check count
			if len(content) != tt.expectedCount {
				t.Errorf("Content() returned %d items, want %d", len(content), tt.expectedCount)
			}

			// Validate content
			if err := tt.validateContent(content); err != nil {
				t.Errorf("Content validation failed: %v", err)
			}

			// Test that returned content is a copy (modification doesn't affect original)
			if len(content) > 0 {
				originalFirst := content[0]
				content[0] = nil
				sectionContent := section.Content()
				if sectionContent[0] == nil {
					t.Error("Content() should return a copy, but modification affected original")
				}
				if sectionContent[0] != originalFirst {
					t.Error("Content() copy should contain same references")
				}
			}
		})
	}
}

func TestCollapsibleSectionLevelLimits(t *testing.T) {
	tests := []struct {
		name          string
		inputLevel    int
		expectedLevel int
	}{
		{"valid level 0", 0, 0},
		{"valid level 1", 1, 1},
		{"valid level 2", 2, 2},
		{"valid level 3", 3, 3},
		{"level too high", 4, 0}, // Should be ignored, default to 0
		{"level too high 10", 10, 0},
		{"negative level", -1, 0}, // Should be ignored, default to 0
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			section := NewCollapsibleSection("Test", []Content{}, WithSectionLevel(tt.inputLevel))
			if got := section.Level(); got != tt.expectedLevel {
				t.Errorf("Level() = %v, want %v", got, tt.expectedLevel)
			}
		})
	}
}

func TestCollapsibleSectionFormatHints(t *testing.T) {
	markdownHints := map[string]any{
		"class": "expandable-section",
		"style": "border: 1px solid #ccc",
	}

	jsonHints := map[string]any{
		"includeMetadata": true,
		"compact":         false,
	}

	section := NewCollapsibleSection("Test Section", []Content{},
		WithSectionFormatHint("markdown", markdownHints),
		WithSectionFormatHint("json", jsonHints),
	)

	tests := []struct {
		name     string
		format   string
		expected map[string]any
	}{
		{"markdown hints", "markdown", markdownHints},
		{"json hints", "json", jsonHints},
		{"non-existent format", "xml", nil},
		{"empty format", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := section.FormatHint(tt.format)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("FormatHint(%q) = %v, want %v", tt.format, got, tt.expected)
			}
		})
	}
}

func TestCollapsibleSectionHelperFunctions(t *testing.T) {
	// Create test data
	table1, _ := NewTableContent("Table 1",
		[]map[string]any{{"col1": "value1"}},
		WithKeys("col1"))

	table2, _ := NewTableContent("Table 2",
		[]map[string]any{{"col2": "value2"}},
		WithKeys("col2"))

	text1 := NewTextContent("Summary text")

	t.Run("NewCollapsibleTable", func(t *testing.T) {
		section := NewCollapsibleTable("Single Table Section", table1, WithSectionExpanded(true))

		if section.Title() != "Single Table Section" {
			t.Errorf("Title() = %v, want %v", section.Title(), "Single Table Section")
		}

		if !section.IsExpanded() {
			t.Error("IsExpanded() = false, want true")
		}

		content := section.Content()
		if len(content) != 1 {
			t.Fatalf("Content() returned %d items, want 1", len(content))
		}

		if content[0] != table1 {
			t.Error("Content does not contain expected table")
		}
	})

	t.Run("NewCollapsibleMultiTable", func(t *testing.T) {
		tables := []*TableContent{table1, table2}
		section := NewCollapsibleMultiTable("Multi Table Section", tables, WithSectionLevel(1))

		if section.Title() != "Multi Table Section" {
			t.Errorf("Title() = %v, want %v", section.Title(), "Multi Table Section")
		}

		if section.Level() != 1 {
			t.Errorf("Level() = %v, want %v", section.Level(), 1)
		}

		content := section.Content()
		if len(content) != 2 {
			t.Fatalf("Content() returned %d items, want 2", len(content))
		}

		if content[0] != table1 || content[1] != table2 {
			t.Error("Content does not contain expected tables in correct order")
		}
	})

	t.Run("NewCollapsibleReport", func(t *testing.T) {
		mixedContent := []Content{text1, table1, table2}
		section := NewCollapsibleReport("Analysis Report", mixedContent,
			WithSectionExpanded(false),
			WithSectionLevel(2),
		)

		if section.Title() != "Analysis Report" {
			t.Errorf("Title() = %v, want %v", section.Title(), "Analysis Report")
		}

		if section.IsExpanded() {
			t.Error("IsExpanded() = true, want false")
		}

		if section.Level() != 2 {
			t.Errorf("Level() = %v, want %v", section.Level(), 2)
		}

		content := section.Content()
		if len(content) != 3 {
			t.Fatalf("Content() returned %d items, want 3", len(content))
		}

		// Verify content order
		if content[0] != text1 || content[1] != table1 || content[2] != table2 {
			t.Error("Content does not contain expected items in correct order")
		}
	})
}

func TestCollapsibleSectionNestedScenario(t *testing.T) {
	// Create nested sections to test hierarchical structure
	innerText := NewTextContent("Inner content")
	innerSection := NewCollapsibleSection("Inner Section", []Content{innerText}, WithSectionLevel(3))

	middleTable, _ := NewTableContent("Middle Table",
		[]map[string]any{{"data": "value"}},
		WithKeys("data"))
	middleSection := NewCollapsibleSection("Middle Section", []Content{middleTable}, WithSectionLevel(2))

	// Note: In real implementation, sections might contain other sections as Content
	// This tests the level hierarchy concept
	outerSection := NewCollapsibleSection("Outer Section", []Content{
		NewTextContent("Outer content"),
	}, WithSectionLevel(1))

	// Verify levels are maintained correctly
	if innerSection.Level() != 3 {
		t.Errorf("Inner section level = %v, want 3", innerSection.Level())
	}
	if middleSection.Level() != 2 {
		t.Errorf("Middle section level = %v, want 2", middleSection.Level())
	}
	if outerSection.Level() != 1 {
		t.Errorf("Outer section level = %v, want 1", outerSection.Level())
	}
}

// Helper function for error creation in tests
func errorf(format string, args ...any) error {
	return &testError{message: sprintf(format, args...)}
}

type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}

func sprintf(format string, args ...any) string {
	// Simple sprintf implementation for tests
	// In real code, use fmt.Sprintf
	return format
}
