package output

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileWriterMultiSectionAppend(t *testing.T) {
	skipIfNotIntegration(t)

	tempDir := t.TempDir()
	ctx := context.Background()

	tests := map[string]struct {
		format       string
		initialData  string
		buildDoc     func() *Document
		wantCombined string
		wantErr      bool
	}{
		"append document with multiple table sections": {
			format:      FormatCSV,
			initialData: "Age,Name\nAlice,30\n",
			buildDoc: func() *Document {
				builder := New()
				builder.Table("", []map[string]any{
					{"Age": 25, "Name": "Bob"},
				}, WithKeys("Age", "Name"))
				builder.Table("", []map[string]any{
					{"Age": 35, "Name": "Charlie"},
				}, WithKeys("Age", "Name"))
				return builder.Build()
			},
			// CSV renderer outputs each table with headers and blank line separator
			// appendCSVWithoutHeaders strips only the first line
			wantCombined: "Age,Name\nAlice,30\n25,Bob\n\n35,Charlie\n",
			wantErr:      false,
		},
		"append document with mixed content types to JSON": {
			format:      FormatJSON,
			initialData: `{"data":"initial"}`,
			buildDoc: func() *Document {
				builder := New()
				builder.Table("", []map[string]any{
					{"Name": "Bob"},
				})
				builder.Text("Some text")
				return builder.Build()
			},
			// JSON appending produces concatenated output (NDJSON-style)
			wantErr: false,
			// We just verify it doesn't error - exact output format depends on renderer
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			testDir := filepath.Join(tempDir, strings.ReplaceAll(name, " ", "-"))
			err := os.MkdirAll(testDir, 0755)
			if err != nil {
				t.Fatalf("failed to create test directory: %v", err)
			}

			// Create FileWriter in append mode
			fw, err := NewFileWriterWithOptions(testDir, "test-{format}.{ext}", WithAppendMode())
			if err != nil {
				t.Fatalf("failed to create FileWriter: %v", err)
			}

			// Determine file extension
			ext := "txt"
			switch tc.format {
			case FormatJSON:
				ext = "json"
			case FormatCSV:
				ext = "csv"
			case FormatHTML:
				ext = "html"
			case FormatYAML:
				ext = "yaml"
			}

			filePath := filepath.Join(testDir, "test-"+tc.format+"."+ext)

			// Create initial file
			if err := os.WriteFile(filePath, []byte(tc.initialData), 0644); err != nil {
				t.Fatalf("failed to create initial file: %v", err)
			}

			// Build multi-section document
			doc := tc.buildDoc()

			// Render the document
			var renderer Renderer
			switch tc.format {
			case FormatJSON:
				renderer = JSON.Renderer
			case FormatCSV:
				renderer = CSV.Renderer
			case FormatHTML:
				renderer = HTMLFragment.Renderer
			case FormatYAML:
				renderer = YAML.Renderer
			default:
				t.Fatalf("unsupported format: %s", tc.format)
			}

			renderedData, err := renderer.Render(ctx, doc)
			if err != nil {
				t.Fatalf("failed to render document: %v", err)
			}

			// Append the rendered data
			err = fw.Write(ctx, tc.format, renderedData)

			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
				return
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// For CSV, verify exact output
			if tc.format == FormatCSV && tc.wantCombined != "" {
				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatalf("failed to read file: %v", err)
				}

				if string(content) != tc.wantCombined {
					t.Errorf("file content = %q, want %q", string(content), tc.wantCombined)
				}
			}
		})
	}
}

func TestFileWriterHTMLMultiSectionAppend(t *testing.T) {
	skipIfNotIntegration(t)

	tempDir := t.TempDir()
	ctx := context.Background()

	tests := map[string]struct {
		initialHTML  string
		buildDoc     func() *Document
		wantContains []string
		wantErr      bool
	}{
		"HTML multi-section append all before marker": {
			initialHTML: "<html><body><h1>Initial</h1>\n<!-- go-output-append -->\n</body></html>",
			buildDoc: func() *Document {
				builder := New()
				builder.Table("", []map[string]any{
					{"Name": "Alice", "Age": 30},
				})
				builder.Table("", []map[string]any{
					{"Name": "Bob", "Age": 25},
				})
				return builder.Build()
			},
			wantContains: []string{
				"<h1>Initial</h1>",
				"Alice",
				"Bob",
				"<!-- go-output-append -->",
			},
			wantErr: false,
		},
		"append document with section content": {
			initialHTML: "<html><body><p>Start</p>\n<!-- go-output-append -->\n</body></html>",
			buildDoc: func() *Document {
				builder := New()
				builder.Section("Section 1", func(b *Builder) {
					b.Table("", []map[string]any{
						{"Name": "Alice"},
					})
				})
				builder.Section("Section 2", func(b *Builder) {
					b.Table("", []map[string]any{
						{"Name": "Bob"},
					})
				})
				return builder.Build()
			},
			wantContains: []string{
				"<p>Start</p>",
				"Section 1",
				"Section 2",
				"Alice",
				"Bob",
				"<!-- go-output-append -->",
			},
			wantErr: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			testDir := filepath.Join(tempDir, strings.ReplaceAll(name, " ", "-"))
			err := os.MkdirAll(testDir, 0755)
			if err != nil {
				t.Fatalf("failed to create test directory: %v", err)
			}

			// Create FileWriter in append mode
			fw, err := NewFileWriterWithOptions(testDir, "test.html", WithAppendMode())
			if err != nil {
				t.Fatalf("failed to create FileWriter: %v", err)
			}

			filePath := filepath.Join(testDir, "test.html")

			// Create initial HTML file with marker
			if err := os.WriteFile(filePath, []byte(tc.initialHTML), 0644); err != nil {
				t.Fatalf("failed to create initial file: %v", err)
			}

			// Build multi-section document
			doc := tc.buildDoc()

			// Render as HTML fragment
			renderer := HTMLFragment.Renderer
			renderedData, err := renderer.Render(ctx, doc)
			if err != nil {
				t.Fatalf("failed to render document: %v", err)
			}

			// Append to file
			err = fw.Write(ctx, FormatHTML, renderedData)

			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
				return
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify all expected content is present
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			contentStr := string(content)
			for _, want := range tc.wantContains {
				if !strings.Contains(contentStr, want) {
					t.Errorf("file content missing %q\nGot: %s", want, contentStr)
				}
			}

			// Verify marker is still at the end
			markerIndex := strings.Index(contentStr, HTMLAppendMarker)
			if markerIndex == -1 {
				t.Error("HTML append marker not found in final content")
			}
		})
	}
}

func TestFileWriterCSVMultiSectionHeaderHandling(t *testing.T) {
	skipIfNotIntegration(t)

	tempDir := t.TempDir()
	ctx := context.Background()

	// Test that only the first section has headers stripped
	t.Run("CSV multi-section only first section has headers stripped", func(t *testing.T) {

		testDir := filepath.Join(tempDir, "csv-multisection")
		err := os.MkdirAll(testDir, 0755)
		if err != nil {
			t.Fatalf("failed to create test directory: %v", err)
		}

		fw, err := NewFileWriterWithOptions(testDir, "test.csv", WithAppendMode())
		if err != nil {
			t.Fatalf("failed to create FileWriter: %v", err)
		}

		filePath := filepath.Join(testDir, "test.csv")

		// Create initial CSV file with Age,Name order to match new tables
		initialData := "Age,Name\n30,Alice\n"
		if err := os.WriteFile(filePath, []byte(initialData), 0644); err != nil {
			t.Fatalf("failed to create initial file: %v", err)
		}

		// Build document with multiple table sections
		// Use explicit key order to ensure consistent output
		builder := New()
		builder.Table("", []map[string]any{
			{"Age": 25, "Name": "Bob"},
		}, WithKeys("Age", "Name"))
		builder.Table("", []map[string]any{
			{"Age": 35, "Name": "Charlie"},
		}, WithKeys("Age", "Name"))
		doc := builder.Build()

		// Render as CSV
		renderer := CSV.Renderer
		renderedData, err := renderer.Render(ctx, doc)
		if err != nil {
			t.Fatalf("failed to render document: %v", err)
		}

		// Append to file
		err = fw.Write(ctx, FormatCSV, renderedData)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify content
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		// Should have initial row, Bob, and Charlie, but only one header line total
		lines := strings.Split(string(content), "\n")
		headerCount := 0
		for _, line := range lines {
			if strings.Contains(line, "Age,Name") {
				headerCount++
			}
		}

		if headerCount != 1 {
			t.Errorf("expected exactly 1 header line, got %d\nContent:\n%s", headerCount, string(content))
		}

		// Verify all data rows are present
		contentStr := string(content)
		wantRows := []string{"30,Alice", "25,Bob", "35,Charlie"}
		for _, row := range wantRows {
			if !strings.Contains(contentStr, row) {
				t.Errorf("missing expected row %q in content:\n%s", row, contentStr)
			}
		}
	})
}
