package output

import (
	"testing"
)

func TestBuilder_AddCollapsibleTable(t *testing.T) {
	tests := map[string]struct {
		title     string
		tableData []map[string]any
		opts      []CollapsibleSectionOption
	}{"expanded collapsible table with level": {

		title: "Critical Issues",
		tableData: []map[string]any{
			{"severity": "high", "count": 5},
			{"severity": "medium", "count": 10},
		},
		opts: []CollapsibleSectionOption{
			WithSectionExpanded(true),
			WithSectionLevel(1),
		},
	}, "simple collapsible table": {

		title: "User Data",
		tableData: []map[string]any{
			{"name": "Alice", "age": 30},
			{"name": "Bob", "age": 25},
		},
		opts: []CollapsibleSectionOption{},
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Create table content first
			table, err := NewTableContent("Test Table", tt.tableData)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			builder := New()
			result := builder.AddCollapsibleTable(tt.title, table, tt.opts...)

			// Verify builder returns itself for chaining
			if result != builder {
				t.Error("AddCollapsibleTable() should return the builder for chaining")
			}

			// Build and check content
			doc := builder.Build()
			if len(doc.contents) != 1 {
				t.Fatalf("Expected 1 content, got %d", len(doc.contents))
			}

			// Verify it's a DefaultCollapsibleSection containing the table
			section, ok := doc.contents[0].(*DefaultCollapsibleSection)
			if !ok {
				t.Fatalf("Expected DefaultCollapsibleSection, got %T", doc.contents[0])
			}

			if section.Title() != tt.title {
				t.Errorf("title = %q, want %q", section.Title(), tt.title)
			}

			// Verify it contains exactly one content item (the table)
			sectionContent := section.Content()
			if len(sectionContent) != 1 {
				t.Fatalf("Expected 1 content in section, got %d", len(sectionContent))
			}

			// Verify the content is a table
			_, ok = sectionContent[0].(*TableContent)
			if !ok {
				t.Fatalf("Expected TableContent in section, got %T", sectionContent[0])
			}
		})
	}
}
