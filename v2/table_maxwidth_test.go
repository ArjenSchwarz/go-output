package output

import (
	"context"
	"strings"
	"testing"
)

func TestTableMaxColumnWidth(t *testing.T) {
	tests := map[string]struct {
		maxWidth   int
		data       []Record
		keys       []string
		checkWidth bool
	}{
		"with max width 20": {
			maxWidth: 20,
			data: []Record{
				{
					"Name":        "Alice",
					"Description": "This is a very long description that should be truncated",
				},
			},
			keys:       []string{"Name", "Description"},
			checkWidth: true,
		},
		"with max width 50": {
			maxWidth: 50,
			data: []Record{
				{
					"ID":      "12345",
					"Details": "Short",
				},
			},
			keys:       []string{"ID", "Details"},
			checkWidth: true,
		},
		"with no max width": {
			maxWidth: 0,
			data: []Record{
				{
					"Name": "Bob",
					"Info": "Some information here",
				},
			},
			keys:       []string{"Name", "Info"},
			checkWidth: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create document
			builder := New()
			builder.Table("", tc.data, WithKeys(tc.keys...))
			doc := builder.Build()

			// Create renderer with max column width
			var renderer Renderer
			if tc.maxWidth > 0 {
				renderer = NewTableRendererWithStyleAndWidth("Default", tc.maxWidth)
			} else {
				renderer = NewTableRendererWithStyle("Default")
			}

			// Render
			output, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			outputStr := string(output)

			// Basic validation: output should contain the keys (case-insensitive as table may uppercase headers)
			outputLower := strings.ToLower(outputStr)
			for _, key := range tc.keys {
				keyLower := strings.ToLower(key)
				if !strings.Contains(outputLower, keyLower) {
					t.Errorf("Output should contain key %q (found in output: %v)", key, strings.Contains(outputStr, key))
				}
			}

			// If we set a max width, verify the renderer was configured
			// (We can't easily test the actual width without parsing the table,
			// but we can verify it renders without error)
			if tc.checkWidth && len(outputStr) == 0 {
				t.Error("Expected non-empty output with max width configured")
			}
		})
	}
}

func TestTableWithMaxColumnWidthHelper(t *testing.T) {
	data := []Record{
		{"Name": "Alice", "Email": "alice@example.com"},
		{"Name": "Bob", "Email": "bob@example.com"},
	}

	builder := New()
	builder.Table("Users", data, WithKeys("Name", "Email"))
	doc := builder.Build()

	// Test TableWithMaxColumnWidth helper
	format := TableWithMaxColumnWidth(30)
	if format.Name != FormatTable {
		t.Errorf("Expected format name %q, got %q", FormatTable, format.Name)
	}

	output, err := format.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	outputStr := string(output)
	outputLower := strings.ToLower(outputStr)
	if !strings.Contains(outputLower, "name") || !strings.Contains(outputLower, "email") {
		t.Error("Output should contain column headers")
	}
}

func TestTableWithStyleAndMaxColumnWidthHelper(t *testing.T) {
	data := []Record{
		{"ID": "1", "Description": "A very long description that exceeds normal width"},
	}

	builder := New()
	builder.Table("", data, WithKeys("ID", "Description"))
	doc := builder.Build()

	// Test TableWithStyleAndMaxColumnWidth helper
	format := TableWithStyleAndMaxColumnWidth("Bold", 25)
	if format.Name != FormatTable {
		t.Errorf("Expected format name %q, got %q", FormatTable, format.Name)
	}

	output, err := format.Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	outputStr := string(output)
	if len(outputStr) == 0 {
		t.Error("Expected non-empty output")
	}
}

func TestMaxColumnWidthWithLongContent(t *testing.T) {
	longText := strings.Repeat("This is a long text. ", 20)

	data := []Record{
		{"Column1": "Short", "Column2": longText},
	}

	builder := New()
	builder.Table("", data, WithKeys("Column1", "Column2"))
	doc := builder.Build()

	// Render with max width
	renderer := NewTableRendererWithStyleAndWidth("Default", 40)
	output, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	outputStr := string(output)

	// Verify output contains table structure (case-insensitive)
	outputLower := strings.ToLower(outputStr)
	if !strings.Contains(outputLower, "column1") || !strings.Contains(outputLower, "column2") {
		t.Error("Output should contain column headers")
	}

	// The long text should be present (possibly wrapped or truncated by go-pretty)
	// We just verify the table renders successfully
	if len(outputStr) == 0 {
		t.Error("Expected non-empty rendered table")
	}
}

func TestMaxColumnWidthPreservesKeyOrder(t *testing.T) {
	data := []Record{
		{"Z_Field": "1", "A_Field": "2", "M_Field": "3"},
	}

	keys := []string{"Z_Field", "A_Field", "M_Field"}

	builder := New()
	builder.Table("", data, WithKeys(keys...))
	doc := builder.Build()

	renderer := NewTableRendererWithStyleAndWidth("Default", 30)
	output, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	outputStr := string(output)

	// Find positions of headers in output (case-insensitive)
	outputLower := strings.ToLower(outputStr)
	zPos := strings.Index(outputLower, "z_field")
	aPos := strings.Index(outputLower, "a_field")
	mPos := strings.Index(outputLower, "m_field")

	// All headers should be present
	if zPos == -1 || aPos == -1 || mPos == -1 {
		t.Fatal("All headers should be present in output")
	}

	// Key order should be preserved: Z before A before M
	if !(zPos < aPos && aPos < mPos) {
		t.Errorf("Key order not preserved: Z_Field at %d, A_Field at %d, M_Field at %d", zPos, aPos, mPos)
	}
}
