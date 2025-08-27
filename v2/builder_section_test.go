package output

import (
	"testing"
)

func TestBuilder_SectionMethod(t *testing.T) {
	tests := map[string]struct {
		title           string
		opts            []SectionOption
		contentCount    int
		expectedLevel   int
		expectedContent []string
	}{"empty section": {

		title:         "Empty Section",
		opts:          []SectionOption{},
		contentCount:  0,
		expectedLevel: 0,
	}, "section with level": {

		title:         "Nested Section",
		opts:          []SectionOption{WithLevel(2)},
		contentCount:  1,
		expectedLevel: 2,
	}, "simple section": {

		title:         "Test Section",
		opts:          []SectionOption{},
		contentCount:  2,
		expectedLevel: 0,
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			builder := New()

			var sectionBuilder *Builder
			result := builder.Section(tt.title, func(b *Builder) {
				sectionBuilder = b
				// Add test content based on contentCount
				for i := 0; i < tt.contentCount; i++ {
					b.Text("Content " + string(rune('A'+i)))
				}
			}, tt.opts...)

			// Verify builder returns itself for chaining
			if result != builder {
				t.Error("Section() should return the builder for chaining")
			}

			// Verify section builder is different from main builder
			if sectionBuilder == builder {
				t.Error("Section should create a new sub-builder")
			}

			// Build and check content
			doc := builder.Build()
			if len(doc.contents) != 1 {
				t.Fatalf("Expected 1 content, got %d", len(doc.contents))
			}

			// Verify it's a SectionContent
			section, ok := doc.contents[0].(*SectionContent)
			if !ok {
				t.Fatalf("Expected SectionContent, got %T", doc.contents[0])
			}

			if section.title != tt.title {
				t.Errorf("title = %q, want %q", section.title, tt.title)
			}

			if section.level != tt.expectedLevel {
				t.Errorf("level = %d, want %d", section.level, tt.expectedLevel)
			}

			if len(section.contents) != tt.contentCount {
				t.Errorf("content count = %d, want %d", len(section.contents), tt.contentCount)
			}
		})
	}
}

// TestBuilder_Graph tests the Graph method
func TestBuilder_AddCollapsibleSection(t *testing.T) {
	tests := map[string]struct {
		title            string
		content          []Content
		opts             []CollapsibleSectionOption
		expectedLevel    int
		expectedContent  int
		expectedExpanded bool
	}{"empty collapsible section": {

		title:            "No Details",
		content:          []Content{},
		opts:             []CollapsibleSectionOption{},
		expectedLevel:    0,
		expectedContent:  0,
		expectedExpanded: false,
	}, "expanded section with level": {

		title: "Important Information",
		content: []Content{
			NewTextContent("Critical details"),
		},
		opts: []CollapsibleSectionOption{
			WithSectionExpanded(true),
			WithSectionLevel(2),
		},
		expectedLevel:    2,
		expectedContent:  1,
		expectedExpanded: true,
	}, "simple collapsible section": {

		title: "Expandable Details",
		content: []Content{
			NewTextContent("Detail text 1"),
			NewTextContent("Detail text 2"),
		},
		opts:             []CollapsibleSectionOption{},
		expectedLevel:    0,
		expectedContent:  2,
		expectedExpanded: false,
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			builder := New()
			result := builder.AddCollapsibleSection(tt.title, tt.content, tt.opts...)

			// Verify builder returns itself for chaining
			if result != builder {
				t.Error("AddCollapsibleSection() should return the builder for chaining")
			}

			// Build and check content
			doc := builder.Build()
			if len(doc.contents) != 1 {
				t.Fatalf("Expected 1 content, got %d", len(doc.contents))
			}

			// Verify it's a DefaultCollapsibleSection
			section, ok := doc.contents[0].(*DefaultCollapsibleSection)
			if !ok {
				t.Fatalf("Expected DefaultCollapsibleSection, got %T", doc.contents[0])
			}

			if section.Title() != tt.title {
				t.Errorf("title = %q, want %q", section.Title(), tt.title)
			}

			if section.Level() != tt.expectedLevel {
				t.Errorf("level = %d, want %d", section.Level(), tt.expectedLevel)
			}

			if section.IsExpanded() != tt.expectedExpanded {
				t.Errorf("expanded = %t, want %t", section.IsExpanded(), tt.expectedExpanded)
			}

			if len(section.Content()) != tt.expectedContent {
				t.Errorf("content count = %d, want %d", len(section.Content()), tt.expectedContent)
			}
		})
	}
}

// TestBuilder_AddCollapsibleTable tests the AddCollapsibleTable method
func TestBuilder_CollapsibleSection(t *testing.T) {
	tests := map[string]struct {
		title           string
		opts            []CollapsibleSectionOption
		contentCount    int
		expectedLevel   int
		expectedContent []string
	}{"empty collapsible section": {

		title:         "No Results",
		opts:          []CollapsibleSectionOption{},
		contentCount:  0,
		expectedLevel: 0,
	}, "expanded section with level and sub-builder": {

		title:         "Important Findings",
		opts:          []CollapsibleSectionOption{WithSectionLevel(1), WithSectionExpanded(true)},
		contentCount:  3,
		expectedLevel: 1,
	}, "simple collapsible section with sub-builder": {

		title:         "Analysis Results",
		opts:          []CollapsibleSectionOption{},
		contentCount:  2,
		expectedLevel: 0,
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			builder := New()

			var sectionBuilder *Builder
			result := builder.CollapsibleSection(tt.title, func(b *Builder) {
				sectionBuilder = b
				// Add test content based on contentCount
				for i := 0; i < tt.contentCount; i++ {
					b.Text("Content " + string(rune('A'+i)))
				}
			}, tt.opts...)

			// Verify builder returns itself for chaining
			if result != builder {
				t.Error("CollapsibleSection() should return the builder for chaining")
			}

			// Verify section builder is different from main builder
			if sectionBuilder == builder {
				t.Error("CollapsibleSection should create a new sub-builder")
			}

			// Build and check content
			doc := builder.Build()
			if len(doc.contents) != 1 {
				t.Fatalf("Expected 1 content, got %d", len(doc.contents))
			}

			// Verify it's a DefaultCollapsibleSection
			section, ok := doc.contents[0].(*DefaultCollapsibleSection)
			if !ok {
				t.Fatalf("Expected DefaultCollapsibleSection, got %T", doc.contents[0])
			}

			if section.Title() != tt.title {
				t.Errorf("title = %q, want %q", section.Title(), tt.title)
			}

			if section.Level() != tt.expectedLevel {
				t.Errorf("level = %d, want %d", section.Level(), tt.expectedLevel)
			}

			if len(section.Content()) != tt.contentCount {
				t.Errorf("content count = %d, want %d", len(section.Content()), tt.contentCount)
			}
		})
	}
}

// TestBuilder_CollapsibleSection_Mixed_Content tests collapsible sections with mixed content types
func TestBuilder_CollapsibleSection_Mixed_Content(t *testing.T) {
	builder := New()

	// Create a collapsible section with mixed content
	result := builder.CollapsibleSection("Analysis Report", func(b *Builder) {
		b.Text("Summary: Found multiple issues")
		b.Table("Issues", []map[string]any{
			{"type": "error", "count": 5},
			{"type": "warning", "count": 10},
		}, WithKeys("type", "count"))
		b.Text("See detailed breakdown above")
	}, WithSectionExpanded(false), WithSectionLevel(2))

	// Verify method chaining
	if result != builder {
		t.Error("CollapsibleSection() should return the builder for chaining")
	}

	// Build and verify
	doc := builder.Build()
	if len(doc.contents) != 1 {
		t.Fatalf("Expected 1 content, got %d", len(doc.contents))
	}

	section, ok := doc.contents[0].(*DefaultCollapsibleSection)
	if !ok {
		t.Fatalf("Expected DefaultCollapsibleSection, got %T", doc.contents[0])
	}

	if section.Title() != "Analysis Report" {
		t.Errorf("title = %q, want %q", section.Title(), "Analysis Report")
	}

	if section.Level() != 2 {
		t.Errorf("level = %d, want %d", section.Level(), 2)
	}

	if section.IsExpanded() != false {
		t.Errorf("expanded = %t, want %t", section.IsExpanded(), false)
	}

	// Verify content types
	content := section.Content()
	if len(content) != 3 {
		t.Fatalf("Expected 3 content items, got %d", len(content))
	}

	// Check content types
	_, ok1 := content[0].(*TextContent)
	_, ok2 := content[1].(*TableContent)
	_, ok3 := content[2].(*TextContent)

	if !ok1 || !ok2 || !ok3 {
		t.Error("Content types don't match expected: text, table, text")
	}
}
