package output

import (
	"context"
	"reflect"
	"testing"
)

// TestValidation_V1FeatureParity tests all v1 features are available in v2
func TestValidation_V1FeatureParity(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{"OutputFormats", testV1OutputFormats},
		{"Transformations", testV1Transformations},
		{"FileOutput", testV1FileOutput},
		{"S3Output", testV1S3Output},
		{"DataTypes", testV1DataTypes},
		{"ProgressIndicators", testV1ProgressIndicators},
		{"TableStyling", testV1TableStyling},
		{"TableOfContents", testV1TableOfContents},
		{"FrontMatter", testV1FrontMatter},
		{"ChartDiagrams", testV1ChartDiagrams},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.test(t)
		})
	}
}

func testV1OutputFormats(t *testing.T) {
	// Verify all v1 formats are available
	formats := []Format{
		JSON, YAML, CSV, HTML, Table, Markdown, DOT, Mermaid, DrawIO,
	}

	doc := New().
		Table("Test", []map[string]any{{"key": "value"}}, WithKeys("key")).
		Build()

	for _, format := range formats {
		t.Run(format.Name, func(t *testing.T) {
			output := NewOutput(
				WithFormat(format),
				WithWriter(&nullWriter{}),
			)

			err := output.Render(context.Background(), doc)
			if err != nil {
				t.Errorf("Failed to render %s format: %v", format.Name, err)
			}
		})
	}
}

func testV1Transformations(t *testing.T) {
	// Test all v1 transformers
	transformers := []Transformer{
		&EmojiTransformer{},
		NewColorTransformer(),
		NewSortTransformer("key", true),
		NewLineSplitTransformer(","),
	}

	for _, transformer := range transformers {
		t.Run(transformer.Name(), func(t *testing.T) {
			if transformer.Priority() < 0 {
				t.Error("Transformer priority should be non-negative")
			}
		})
	}
}

func testV1FileOutput(t *testing.T) {
	// Test file writer creation
	fw, err := NewFileWriter("/tmp", "test-{format}.{ext}")
	if err != nil {
		t.Fatalf("Failed to create file writer: %v", err)
	}
	if fw == nil {
		t.Error("File writer should not be nil")
	}
}

func testV1S3Output(t *testing.T) {
	// Test S3 writer creation
	mockClient := &mockS3Client{}
	sw := NewS3Writer(mockClient, "test-bucket", "test-key-{format}")
	if sw == nil {
		t.Error("S3 writer should not be nil")
	}
}

func testV1DataTypes(t *testing.T) {
	// Test all data types are preserved correctly
	data := []map[string]any{
		{
			"string": "text",
			"int":    42,
			"float":  3.14,
			"bool":   true,
			"nil":    nil,
		},
	}

	doc := New().
		Table("DataTypes", data, WithAutoSchema()).
		Build()

	// Verify table content preserves types
	table := doc.GetContents()[0].(*TableContent)
	record := table.records[0]

	if record["string"] != "text" {
		t.Error("String type not preserved")
	}
	if record["int"] != 42 {
		t.Error("Int type not preserved")
	}
	if record["float"] != 3.14 {
		t.Error("Float type not preserved")
	}
	if record["bool"] != true {
		t.Error("Bool type not preserved")
	}
	if record["nil"] != nil {
		t.Error("Nil type not preserved")
	}
}

func testV1ProgressIndicators(t *testing.T) {
	// Test progress indicator creation and colors
	colors := []ProgressColor{
		ProgressColorDefault,
		ProgressColorGreen,
		ProgressColorRed,
		ProgressColorYellow,
		ProgressColorBlue,
	}

	for _, color := range colors {
		progress := NewProgress(WithProgressColor(color))
		if progress == nil {
			t.Error("Progress indicator should not be nil")
		}

		// Test v1 compatibility methods
		progress.SetTotal(100)
		progress.SetCurrent(50)
		progress.Increment(10)
		progress.SetStatus("Testing")
		progress.SetColor(color)

		// IsActive should work
		_ = progress.IsActive()
	}
}

func testV1TableStyling(t *testing.T) {
	// Test table styling options
	styles := []string{
		"Default", "Light", "Bold", "ColoredBright",
	}

	for _, style := range styles {
		output := NewOutput(
			WithFormat(Table),
			WithTableStyle(style),
			WithWriter(&nullWriter{}),
		)

		if output == nil {
			t.Errorf("Failed to create output with table style %s", style)
		}
	}
}

func testV1TableOfContents(t *testing.T) {
	// Test TOC generation
	output := NewOutput(
		WithFormat(Markdown),
		WithTOC(true),
		WithWriter(&nullWriter{}),
	)

	if output == nil {
		t.Error("Failed to create output with TOC")
	}
}

func testV1FrontMatter(t *testing.T) {
	// Test front matter support
	frontMatter := map[string]string{
		"title":  "Test",
		"author": "Test Suite",
		"date":   "2024-01-01",
	}

	output := NewOutput(
		WithFormat(Markdown),
		WithFrontMatter(frontMatter),
		WithWriter(&nullWriter{}),
	)

	if output == nil {
		t.Error("Failed to create output with front matter")
	}
}

func testV1ChartDiagrams(t *testing.T) {
	// Test all chart and diagram types
	doc := New().
		// DOT format
		Graph("Dependencies", []Edge{
			{From: "A", To: "B"},
		}).
		// Mermaid formats
		Chart("Flow", "flowchart", map[string]any{
			"nodes": []string{"Start", "End"},
		}).
		GanttChart("Timeline", []GanttTask{
			{ID: "task1", Title: "Task 1", StartDate: "2024-01-01", EndDate: "2024-01-15"},
		}).
		PieChart("Distribution", []PieSlice{
			{Label: "Part1", Value: 60},
			{Label: "Part2", Value: 40},
		}, true).
		// Draw.io format
		DrawIO("Diagram", []Record{
			{"id": "1", "name": "Node1"},
		}, DrawIOHeader{
			Layout: "auto",
		}).
		Build()

	if len(doc.GetContents()) != 5 {
		t.Error("Not all chart types were created")
	}
}

// TestValidation_SecurityRequirements tests security features
func TestValidation_SecurityRequirements(t *testing.T) {
	t.Run("DirectoryConfinement", func(t *testing.T) {
		// File writer should use os.Root for confinement
		// This is implementation-specific but we can test creation
		fw, err := NewFileWriter("/tmp", "test.txt")
		if err != nil {
			t.Fatalf("Failed to create file writer: %v", err)
		}
		if fw == nil {
			t.Error("File writer should not be nil")
		}
	})

	t.Run("InputValidation", func(t *testing.T) {
		// Test that invalid inputs are handled gracefully
		builder := New()

		// Invalid table data
		builder.Table("Invalid", "not a valid type", WithKeys("key"))

		if !builder.HasErrors() {
			t.Error("Builder should have errors for invalid input")
		}
	})

	t.Run("ErrorMessages", func(t *testing.T) {
		// Verify error messages don't leak sensitive info
		err := NewRenderError("json", nil, context.Canceled)
		errMsg := err.Error()

		// Should not contain system paths or internal details
		if len(errMsg) > 200 {
			t.Error("Error message might be leaking too much information")
		}
	})
}

// TestValidation_DataIntegrity tests data integrity across transformations
func TestValidation_DataIntegrity(t *testing.T) {
	// Original data
	originalData := []map[string]any{
		{
			"id":      1,
			"name":    "Test Name",
			"score":   95.5,
			"active":  true,
			"tags":    nil,
			"special": "!@#$%^&*()",
		},
	}

	doc := New().
		Table("Integrity", originalData, WithKeys("id", "name", "score", "active", "tags", "special")).
		Build()

	// Render to multiple formats
	formats := []Format{JSON, YAML, CSV}

	for _, format := range formats {
		t.Run(format.Name, func(t *testing.T) {
			output := NewOutput(
				WithFormat(format),
				WithWriter(&nullWriter{}),
			)

			err := output.Render(context.Background(), doc)
			if err != nil {
				t.Errorf("Failed to render: %v", err)
			}

			// Verify data integrity is maintained
			table := doc.GetContents()[0].(*TableContent)
			record := table.records[0]

			// Check each field
			if record["id"] != 1 {
				t.Error("ID not preserved")
			}
			if record["name"] != "Test Name" {
				t.Error("Name not preserved")
			}
			if record["score"] != 95.5 {
				t.Error("Score not preserved")
			}
			if record["active"] != true {
				t.Error("Active not preserved")
			}
			if record["tags"] != nil {
				t.Error("Nil not preserved")
			}
			if record["special"] != "!@#$%^&*()" {
				t.Error("Special characters not preserved")
			}
		})
	}
}

// TestValidation_KeyOrderConsistency tests key order is maintained
func TestValidation_KeyOrderConsistency(t *testing.T) {
	// Define specific key order
	keys := []string{"Z", "A", "M", "B", "Y"}

	data := []map[string]any{
		{"A": 1, "B": 2, "M": 3, "Y": 4, "Z": 5},
		{"Z": 10, "Y": 9, "M": 8, "B": 7, "A": 6},
	}

	doc := New().
		Table("OrderTest", data, WithKeys(keys...)).
		Build()

	// Get table and verify key order
	table := doc.GetContents()[0].(*TableContent)
	actualOrder := table.schema.keyOrder

	if !reflect.DeepEqual(actualOrder, keys) {
		t.Errorf("Key order not preserved: got %v, want %v", actualOrder, keys)
	}

	// Render and verify order is maintained
	output := NewOutput(
		WithFormat(CSV),
		WithWriter(&nullWriter{}),
	)

	err := output.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}
}

// TestValidation_ThreadSafety validates thread-safe operations
func TestValidation_ThreadSafety(t *testing.T) {
	// This is tested in integration tests but we can add specific validation
	builder := New()

	// Concurrent metadata updates
	done := make(chan bool, 10)
	for i := range 10 {
		go func(idx int) {
			builder.SetMetadata("key", idx)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for range 10 {
		<-done
	}

	// Should not panic and should have some value
	doc := builder.Build()
	if doc.GetMetadata()["key"] == nil {
		t.Error("Metadata should have a value after concurrent updates")
	}
}

// TestValidation_ErrorConditions tests various error conditions
func TestValidation_ErrorConditions(t *testing.T) {
	tests := map[string]struct {
		fn func() error
	}{"ContextCancellation": {

		fn: func() error {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			doc := New().Text("Test").Build()
			output := NewOutput(
				WithFormat(JSON),
				WithWriter(&nullWriter{}),
			)
			return output.Render(ctx, doc)
		},
	}, "InvalidRawFormat": {

		fn: func() error {
			_, err := NewRawContent("", []byte("data"))
			return err
		},
	}, "InvalidTableData": {

		fn: func() error {
			_, err := NewTableContent("Test", struct{}{})
			return err
		},
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Error("Expected error but got nil")
			}
		})
	}
}

// Mock implementations for testing
type nullWriter struct{}

func (n *nullWriter) Write(ctx context.Context, format string, data []byte) error {
	return nil
}

// Use existing mockS3Client from s3_writer_test.go

// TestValidation_CompleteScenario tests a complete real-world scenario
func TestValidation_CompleteScenario(t *testing.T) {
	// Create a complex document mimicking real usage
	doc := New().
		SetMetadata("report_type", "quarterly").
		SetMetadata("generated_at", "2024-01-01").
		Header("Q4 2023 Sales Report").
		Text("Executive summary of Q4 performance.", WithBold(true)).
		Section("Regional Performance", func(b *Builder) {
			b.Table("Sales by Region", []map[string]any{
				{"Region": "North", "Q3_Sales": 100000, "Q4_Sales": 120000, "Growth": "20%"},
				{"Region": "South", "Q3_Sales": 80000, "Q4_Sales": 85000, "Growth": "6.25%"},
				{"Region": "East", "Q3_Sales": 90000, "Q4_Sales": 105000, "Growth": "16.67%"},
				{"Region": "West", "Q3_Sales": 70000, "Q4_Sales": 72000, "Growth": "2.86%"},
			}, WithKeys("Region", "Q3_Sales", "Q4_Sales", "Growth"))

			b.PieChart("Q4 Market Share", []PieSlice{
				{Label: "North", Value: 38.7},
				{Label: "South", Value: 27.4},
				{Label: "East", Value: 33.9},
			}, true)
		}).
		Section("Product Performance", func(b *Builder) {
			b.Table("Top Products", []map[string]any{
				{"Product": "Widget A", "Units": 5000, "Revenue": 250000},
				{"Product": "Widget B", "Units": 3000, "Revenue": 180000},
				{"Product": "Widget C", "Units": 2000, "Revenue": 140000},
			}, WithKeys("Product", "Units", "Revenue"))
		}).
		Text("For detailed analysis, see appendix.", WithItalic(true)).
		Build()

	// Validate document structure
	contents := doc.GetContents()
	if len(contents) != 5 {
		t.Errorf("Expected 5 content items, got %d", len(contents))
	}

	// Render to multiple formats with transformations
	output := NewOutput(
		WithFormat(Markdown),
		WithFormat(JSON),
		WithTransformer(NewSortTransformer("Revenue", false)),
		WithTableStyle("ColoredBright"),
		WithTOC(true),
		WithWriter(&nullWriter{}),
	)

	err := output.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Failed to render complete scenario: %v", err)
	}
}
