package output

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"
)

// mockDataTransformer implements DataTransformer for testing
type mockDataTransformer struct {
	name        string
	priority    int
	formats     []string
	transformFn func(ctx context.Context, content Content, format string) (Content, error)
	description string
}

func (m *mockDataTransformer) Name() string {
	return m.name
}

func (m *mockDataTransformer) TransformData(ctx context.Context, content Content, format string) (Content, error) {
	if m.transformFn != nil {
		return m.transformFn(ctx, content, format)
	}
	return content, nil
}

func (m *mockDataTransformer) CanTransform(content Content, format string) bool {
	if len(m.formats) == 0 {
		return true
	}
	return slices.Contains(m.formats, format)
}

func (m *mockDataTransformer) Priority() int {
	return m.priority
}

func (m *mockDataTransformer) Describe() string {
	if m.description != "" {
		return m.description
	}
	return fmt.Sprintf("Mock data transformer %s", m.name)
}

func TestDataTransformerInterface(t *testing.T) {
	t.Run("Name returns correct name", func(t *testing.T) {
		transformer := &mockDataTransformer{name: "test-transformer"}
		if got := transformer.Name(); got != "test-transformer" {
			t.Errorf("Name() = %v, want %v", got, "test-transformer")
		}
	})

	t.Run("Priority returns correct priority", func(t *testing.T) {
		transformer := &mockDataTransformer{priority: 100}
		if got := transformer.Priority(); got != 100 {
			t.Errorf("Priority() = %v, want %v", got, 100)
		}
	})

	t.Run("Describe returns description", func(t *testing.T) {
		transformer := &mockDataTransformer{
			name:        "test",
			description: "Test data transformer",
		}
		if got := transformer.Describe(); got != "Test data transformer" {
			t.Errorf("Describe() = %v, want %v", got, "Test data transformer")
		}
	})

	t.Run("CanTransform checks format support", func(t *testing.T) {
		transformer := &mockDataTransformer{
			formats: []string{"json", "yaml"},
		}

		// Create a mock content
		content := &TableContent{
			records: []Record{},
			schema:  &Schema{},
		}

		if !transformer.CanTransform(content, "json") {
			t.Error("CanTransform(content, 'json') should return true")
		}

		if !transformer.CanTransform(content, "yaml") {
			t.Error("CanTransform(content, 'yaml') should return true")
		}

		if transformer.CanTransform(content, "csv") {
			t.Error("CanTransform(content, 'csv') should return false")
		}
	})

	t.Run("TransformData modifies content", func(t *testing.T) {
		originalContent := &TableContent{
			records: []Record{
				{"name": "Alice", "age": 30},
			},
			schema: &Schema{
				Fields: []Field{
					{Name: "name", Type: "string"},
					{Name: "age", Type: "int"},
				},
				keyOrder: []string{"name", "age"},
			},
		}

		transformer := &mockDataTransformer{
			transformFn: func(ctx context.Context, content Content, format string) (Content, error) {
				// Filter out records
				if tc, ok := content.(*TableContent); ok {
					filtered := &TableContent{
						records: []Record{},
						schema:  tc.schema,
					}
					for _, r := range tc.records {
						if age, ok := r["age"].(int); ok && age >= 25 {
							filtered.records = append(filtered.records, r)
						}
					}
					return filtered, nil
				}
				return content, nil
			},
		}

		ctx := context.Background()
		result, err := transformer.TransformData(ctx, originalContent, "json")
		if err != nil {
			t.Fatalf("TransformData() error = %v", err)
		}

		if tc, ok := result.(*TableContent); ok {
			if len(tc.records) != 1 {
				t.Errorf("Expected 1 record after filtering, got %d", len(tc.records))
			}
		} else {
			t.Error("Result is not TableContent")
		}
	})
}

func TestTransformContext(t *testing.T) {
	t.Run("TransformContext initialization", func(t *testing.T) {
		doc := &Document{
			metadata: map[string]any{"title": "Test Doc"},
		}

		ctx := &TransformContext{
			Format:   "json",
			Document: doc,
			Metadata: map[string]any{"key": "value"},
			Stats: TransformStats{
				InputRecords:  100,
				OutputRecords: 50,
				FilteredCount: 50,
				Duration:      time.Second,
			},
		}

		if ctx.Format != "json" {
			t.Errorf("Format = %v, want %v", ctx.Format, "json")
		}

		if ctx.Document != doc {
			t.Error("Document reference mismatch")
		}

		if ctx.Metadata["key"] != "value" {
			t.Error("Metadata not properly set")
		}

		if ctx.Stats.InputRecords != 100 {
			t.Errorf("Stats.InputRecords = %v, want %v", ctx.Stats.InputRecords, 100)
		}

		if ctx.Stats.Duration != time.Second {
			t.Errorf("Stats.Duration = %v, want %v", ctx.Stats.Duration, time.Second)
		}
	})

	t.Run("TransformStats tracking", func(t *testing.T) {
		stats := TransformStats{
			InputRecords:  1000,
			OutputRecords: 800,
			FilteredCount: 200,
			Duration:      2 * time.Second,
			Operations: []OperationStat{
				{Name: "filter", Duration: time.Second, RecordsProcessed: 1000},
				{Name: "sort", Duration: time.Second, RecordsProcessed: 800},
			},
		}

		if len(stats.Operations) != 2 {
			t.Errorf("Expected 2 operations, got %d", len(stats.Operations))
		}

		if stats.Operations[0].Name != "filter" {
			t.Errorf("First operation should be 'filter', got %s", stats.Operations[0].Name)
		}

		if stats.Operations[1].RecordsProcessed != 800 {
			t.Errorf("Second operation should process 800 records, got %d", stats.Operations[1].RecordsProcessed)
		}
	})
}

func TestTransformerAdapter(t *testing.T) {
	t.Run("Detect DataTransformer", func(t *testing.T) {
		dataTransformer := &mockDataTransformer{name: "data-transformer"}

		adapter := &TransformerAdapter{transformer: dataTransformer}

		if !adapter.IsDataTransformer() {
			t.Error("IsDataTransformer() should return true for DataTransformer")
		}

		if adapter.AsDataTransformer() == nil {
			t.Error("AsDataTransformer() should return the transformer")
		}

		if adapter.AsByteTransformer() != nil {
			t.Error("AsByteTransformer() should return nil for DataTransformer")
		}
	})

	t.Run("Detect byte Transformer", func(t *testing.T) {
		byteTransformer := &mockByteTransformer{name: "byte-transformer"}

		adapter := &TransformerAdapter{transformer: byteTransformer}

		if adapter.IsDataTransformer() {
			t.Error("IsDataTransformer() should return false for byte Transformer")
		}

		if adapter.AsDataTransformer() != nil {
			t.Error("AsDataTransformer() should return nil for byte Transformer")
		}

		if adapter.AsByteTransformer() == nil {
			t.Error("AsByteTransformer() should return the transformer")
		}
	})
}

// mockByteTransformer implements the existing byte Transformer interface for testing
type mockByteTransformer struct {
	name     string
	priority int
	formats  []string
}

func (m *mockByteTransformer) Name() string {
	return m.name
}

func (m *mockByteTransformer) Transform(ctx context.Context, input []byte, format string) ([]byte, error) {
	return input, nil
}

func (m *mockByteTransformer) CanTransform(format string) bool {
	if len(m.formats) == 0 {
		return true
	}
	return slices.Contains(m.formats, format)
}

func (m *mockByteTransformer) Priority() int {
	return m.priority
}

func TestTransformableContent(t *testing.T) {
	t.Run("Clone creates deep copy", func(t *testing.T) {
		original := &TableContent{
			records: []Record{
				{"id": 1, "name": "Alice"},
				{"id": 2, "name": "Bob"},
			},
			schema: &Schema{
				Fields: []Field{
					{Name: "id", Type: "int"},
					{Name: "name", Type: "string"},
				},
				keyOrder: []string{"id", "name"},
			},
		}

		// Test that TransformableContent interface is implemented
		var tc TransformableContent = original
		cloned := tc.Clone()

		// Verify it's a different instance
		if cloned == original {
			t.Error("Clone should return a new instance")
		}

		// Verify content is the same
		clonedTable := cloned.(*TableContent)
		if len(clonedTable.records) != len(original.records) {
			t.Error("Cloned content should have same number of records")
		}
	})

	t.Run("Transform applies function to content", func(t *testing.T) {
		content := &TableContent{
			records: []Record{
				{"value": 10},
				{"value": 20},
			},
			schema: &Schema{
				Fields: []Field{
					{Name: "value", Type: "int"},
				},
				keyOrder: []string{"value"},
			},
		}

		var tc TransformableContent = content
		err := tc.Transform(func(data any) (any, error) {
			if records, ok := data.([]Record); ok {
				for i := range records {
					if val, ok := records[i]["value"].(int); ok {
						records[i]["value"] = val * 2
					}
				}
				return records, nil
			}
			return nil, fmt.Errorf("unexpected data type")
		})

		if err != nil {
			t.Fatalf("Transform() error = %v", err)
		}

		// Verify transformation was applied
		if content.records[0]["value"] != 20 {
			t.Errorf("First record value should be 20, got %v", content.records[0]["value"])
		}
		if content.records[1]["value"] != 40 {
			t.Errorf("Second record value should be 40, got %v", content.records[1]["value"])
		}
	})
}

func TestDataTransformerPriority(t *testing.T) {
	t.Run("Transformers are applied in priority order", func(t *testing.T) {
		order := []string{}

		t1 := &mockDataTransformer{
			name:     "first",
			priority: 10,
			transformFn: func(ctx context.Context, content Content, format string) (Content, error) {
				order = append(order, "first")
				return content, nil
			},
		}

		t2 := &mockDataTransformer{
			name:     "second",
			priority: 20,
			transformFn: func(ctx context.Context, content Content, format string) (Content, error) {
				order = append(order, "second")
				return content, nil
			},
		}

		t3 := &mockDataTransformer{
			name:     "third",
			priority: 5,
			transformFn: func(ctx context.Context, content Content, format string) (Content, error) {
				order = append(order, "third")
				return content, nil
			},
		}

		// Create a pipeline and add transformers
		transformers := []DataTransformer{t1, t2, t3}

		// Sort by priority
		for i := 0; i < len(transformers)-1; i++ {
			for j := i + 1; j < len(transformers); j++ {
				if transformers[i].Priority() > transformers[j].Priority() {
					transformers[i], transformers[j] = transformers[j], transformers[i]
				}
			}
		}

		// Apply transformers
		var content Content = &TableContent{records: []Record{}}
		ctx := context.Background()
		for _, t := range transformers {
			content, _ = t.TransformData(ctx, content, "json")
		}

		// Verify order
		if len(order) != 3 {
			t.Fatalf("Expected 3 transformations, got %d", len(order))
		}

		expectedOrder := []string{"third", "first", "second"}
		for i, name := range expectedOrder {
			if order[i] != name {
				t.Errorf("Expected %s at position %d, got %s", name, i, order[i])
			}
		}
	})
}

// TestFormatSpecificBehavior tests format-specific transformer behavior
func TestFormatSpecificBehavior(t *testing.T) {
	t.Run("Transformer receives correct format parameter", func(t *testing.T) {
		receivedFormat := ""
		transformer := &mockDataTransformer{
			transformFn: func(ctx context.Context, content Content, format string) (Content, error) {
				receivedFormat = format
				return content, nil
			},
		}

		content := &TableContent{
			records: []Record{{"test": "value"}},
			schema:  &Schema{Fields: []Field{{Name: "test"}}, keyOrder: []string{"test"}},
		}

		formats := []string{FormatJSON, FormatYAML, FormatCSV, FormatHTML, FormatTable, FormatMarkdown}
		for _, expectedFormat := range formats {
			receivedFormat = "" // Reset
			_, err := transformer.TransformData(context.Background(), content, expectedFormat)
			if err != nil {
				t.Fatalf("TransformData() error = %v", err)
			}
			if receivedFormat != expectedFormat {
				t.Errorf("Expected format %s, received %s", expectedFormat, receivedFormat)
			}
		}
	})

	t.Run("CanTransform works with all supported formats", func(t *testing.T) {
		// Test format-specific transformer
		jsonYamlTransformer := &mockDataTransformer{
			formats: []string{FormatJSON, FormatYAML},
		}

		tabularTransformer := &mockDataTransformer{
			formats: []string{FormatTable, FormatCSV, FormatHTML, FormatMarkdown},
		}

		content := &TableContent{
			records: []Record{{"test": "value"}},
			schema:  &Schema{Fields: []Field{{Name: "test"}}, keyOrder: []string{"test"}},
		}

		// Test JSON/YAML transformer
		if !jsonYamlTransformer.CanTransform(content, FormatJSON) {
			t.Error("JSON/YAML transformer should support JSON")
		}
		if !jsonYamlTransformer.CanTransform(content, FormatYAML) {
			t.Error("JSON/YAML transformer should support YAML")
		}
		if jsonYamlTransformer.CanTransform(content, FormatCSV) {
			t.Error("JSON/YAML transformer should not support CSV")
		}

		// Test tabular transformer
		if !tabularTransformer.CanTransform(content, FormatTable) {
			t.Error("Tabular transformer should support Table")
		}
		if !tabularTransformer.CanTransform(content, FormatCSV) {
			t.Error("Tabular transformer should support CSV")
		}
		if tabularTransformer.CanTransform(content, FormatJSON) {
			t.Error("Tabular transformer should not support JSON")
		}
	})

	t.Run("Format-specific behavior changes based on format", func(t *testing.T) {
		// Transformer that behaves differently for different formats
		formatAwareTransformer := &mockDataTransformer{
			transformFn: func(ctx context.Context, content Content, format string) (Content, error) {
				if tc, ok := content.(*TableContent); ok {
					newRecords := make([]Record, len(tc.records))
					for i, record := range tc.records {
						newRecord := make(Record)
						for k, v := range record {
							// Transform values based on format
							switch format {
							case FormatJSON:
								newRecord[k] = fmt.Sprintf("json_%v", v)
							case FormatCSV:
								newRecord[k] = fmt.Sprintf("csv_%v", v)
							case FormatHTML:
								newRecord[k] = fmt.Sprintf("<span>%v</span>", v)
							default:
								newRecord[k] = v
							}
						}
						newRecords[i] = newRecord
					}
					return &TableContent{
						records: newRecords,
						schema:  tc.schema,
					}, nil
				}
				return content, nil
			},
		}

		originalContent := &TableContent{
			records: []Record{{"name": "test", "value": "data"}},
			schema: &Schema{
				Fields:   []Field{{Name: "name"}, {Name: "value"}},
				keyOrder: []string{"name", "value"},
			},
		}

		ctx := context.Background()

		// Test JSON format
		jsonResult, err := formatAwareTransformer.TransformData(ctx, originalContent, FormatJSON)
		if err != nil {
			t.Fatalf("TransformData() error for JSON = %v", err)
		}
		if tc, ok := jsonResult.(*TableContent); ok {
			if tc.records[0]["name"] != "json_test" {
				t.Errorf("Expected JSON formatted value 'json_test', got %v", tc.records[0]["name"])
			}
		}

		// Test CSV format
		csvResult, err := formatAwareTransformer.TransformData(ctx, originalContent, FormatCSV)
		if err != nil {
			t.Fatalf("TransformData() error for CSV = %v", err)
		}
		if tc, ok := csvResult.(*TableContent); ok {
			if tc.records[0]["name"] != "csv_test" {
				t.Errorf("Expected CSV formatted value 'csv_test', got %v", tc.records[0]["name"])
			}
		}

		// Test HTML format
		htmlResult, err := formatAwareTransformer.TransformData(ctx, originalContent, FormatHTML)
		if err != nil {
			t.Fatalf("TransformData() error for HTML = %v", err)
		}
		if tc, ok := htmlResult.(*TableContent); ok {
			if tc.records[0]["name"] != "<span>test</span>" {
				t.Errorf("Expected HTML formatted value '<span>test</span>', got %v", tc.records[0]["name"])
			}
		}
	})

	t.Run("Multiple format-specific transformers in sequence", func(t *testing.T) {
		transformOrder := []string{}

		jsonTransformer := &mockDataTransformer{
			name:    "json-transformer",
			formats: []string{FormatJSON},
			transformFn: func(ctx context.Context, content Content, format string) (Content, error) {
				transformOrder = append(transformOrder, "json")
				return content, nil
			},
		}

		csvTransformer := &mockDataTransformer{
			name:    "csv-transformer",
			formats: []string{FormatCSV},
			transformFn: func(ctx context.Context, content Content, format string) (Content, error) {
				transformOrder = append(transformOrder, "csv")
				return content, nil
			},
		}

		universalTransformer := &mockDataTransformer{
			name: "universal-transformer",
			transformFn: func(ctx context.Context, content Content, format string) (Content, error) {
				transformOrder = append(transformOrder, "universal")
				return content, nil
			},
		}

		content := &TableContent{
			records: []Record{{"test": "value"}},
			schema:  &Schema{Fields: []Field{{Name: "test"}}, keyOrder: []string{"test"}},
		}

		transformers := []DataTransformer{jsonTransformer, csvTransformer, universalTransformer}

		// Test JSON format - only JSON and universal transformers should apply
		transformOrder = []string{} // Reset
		ctx := context.Background()
		var result Content = content
		for _, transformer := range transformers {
			if transformer.CanTransform(result, FormatJSON) {
				result, _ = transformer.TransformData(ctx, result, FormatJSON)
			}
		}

		expectedJSONOrder := []string{"json", "universal"}
		if len(transformOrder) != len(expectedJSONOrder) {
			t.Fatalf("Expected %d transformations for JSON, got %d", len(expectedJSONOrder), len(transformOrder))
		}
		for i, expected := range expectedJSONOrder {
			if transformOrder[i] != expected {
				t.Errorf("Expected %s at position %d for JSON, got %s", expected, i, transformOrder[i])
			}
		}

		// Test CSV format - only CSV and universal transformers should apply
		transformOrder = []string{} // Reset
		result = content
		for _, transformer := range transformers {
			if transformer.CanTransform(result, FormatCSV) {
				result, _ = transformer.TransformData(ctx, result, FormatCSV)
			}
		}

		expectedCSVOrder := []string{"csv", "universal"}
		if len(transformOrder) != len(expectedCSVOrder) {
			t.Fatalf("Expected %d transformations for CSV, got %d", len(expectedCSVOrder), len(transformOrder))
		}
		for i, expected := range expectedCSVOrder {
			if transformOrder[i] != expected {
				t.Errorf("Expected %s at position %d for CSV, got %s", expected, i, transformOrder[i])
			}
		}
	})
}
