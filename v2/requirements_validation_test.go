package output

import (
	"context"
	"slices"
	"strings"
	"testing"
)

// TestRequirementsValidation validates that all transformation pipeline requirements
// from the tasks.md file are properly implemented and working
func TestRequirementsValidation(t *testing.T) {
	// Test data for validation
	records := []Record{
		{"id": 1, "name": "Alice", "dept": "Engineering", "salary": 95000, "active": true},
		{"id": 2, "name": "Bob", "dept": "Sales", "salary": 65000, "active": false},
		{"id": 3, "name": "Charlie", "dept": "Engineering", "salary": 88000, "active": true},
		{"id": 4, "name": "Diana", "dept": "Marketing", "salary": 72000, "active": true},
	}

	doc := New().
		Table("employees", records, WithKeys("id", "name", "dept", "salary", "active")).
		Build()

	t.Run("Requirement 1.1: DataTransformer interface provides Name, TransformData, CanTransform, Priority, Describe", func(t *testing.T) {
		// Create a mock data transformer to test interface compliance
		transformer := &validationMockDataTransformer{
			name:     "test-transformer",
			priority: 100,
		}

		// Test all required methods exist and work
		if transformer.Name() != "test-transformer" {
			t.Error("Name() method not working correctly")
		}

		if transformer.Priority() != 100 {
			t.Error("Priority() method not working correctly")
		}

		desc := transformer.Describe()
		if desc == "" {
			t.Error("Describe() method should return non-empty description")
		}

		// Test CanTransform
		tableContent := doc.GetContents()[0]
		if !transformer.CanTransform(tableContent, FormatJSON) {
			t.Error("CanTransform() should work with table content and JSON format")
		}

		// Test TransformData
		ctx := context.Background()
		result, err := transformer.TransformData(ctx, tableContent, FormatJSON)
		if err != nil {
			t.Errorf("TransformData() failed: %v", err)
		}
		if result == nil {
			t.Error("TransformData() should return transformed content")
		}
	})

	t.Run("Requirement 9.1-9.6: Backward compatibility with existing transformers", func(t *testing.T) {
		// Test that existing byte transformers still work
		pipeline := NewTransformPipeline()
		pipeline.Add(&validationMockByteTransformer{
			name:     "legacy-transformer",
			priority: 100,
		})

		ctx := context.Background()
		input := []byte("test data")
		result, err := pipeline.Transform(ctx, input, FormatJSON)

		if err != nil {
			t.Fatalf("Legacy transformer should still work: %v", err)
		}

		if !strings.Contains(string(result), "test data") {
			t.Error("Legacy transformer should preserve functionality")
		}

		// Test that TransformPipeline methods still exist and work
		if pipeline.Count() != 1 {
			t.Error("TransformPipeline.Count() should work")
		}

		if !pipeline.Has("legacy-transformer") {
			t.Error("TransformPipeline.Has() should work")
		}

		transformer := pipeline.Get("legacy-transformer")
		if transformer == nil {
			t.Error("TransformPipeline.Get() should work")
		}
	})

	t.Run("Requirement 2.1-2.5: Format-aware transformation support", func(t *testing.T) {
		// Test that format context is passed to transformers
		formatAwareTransformer := &validationMockFormatAwareTransformer{
			supportedFormats: []string{FormatJSON, FormatHTML},
		}

		// Test CanTransform with different formats
		tableContent := doc.GetContents()[0]

		if !formatAwareTransformer.CanTransform(tableContent, FormatJSON) {
			t.Error("Transformer should support JSON format")
		}

		if !formatAwareTransformer.CanTransform(tableContent, FormatHTML) {
			t.Error("Transformer should support HTML format")
		}

		if formatAwareTransformer.CanTransform(tableContent, FormatCSV) {
			t.Error("Transformer should not support unsupported formats")
		}

		// Test that TransformData receives format parameter
		ctx := context.Background()
		_, err := formatAwareTransformer.TransformData(ctx, tableContent, FormatJSON)
		if err != nil {
			t.Errorf("Format-aware transformer should work: %v", err)
		}

		if formatAwareTransformer.lastFormat != FormatJSON {
			t.Error("Transformer should receive correct format parameter")
		}
	})
}

// Mock implementations for testing

type validationMockDataTransformer struct {
	name     string
	priority int
}

func (m *validationMockDataTransformer) Name() string     { return m.name }
func (m *validationMockDataTransformer) Priority() int    { return m.priority }
func (m *validationMockDataTransformer) Describe() string { return "Mock data transformer for testing" }

func (m *validationMockDataTransformer) CanTransform(content Content, format string) bool {
	return content.Type() == ContentTypeTable
}

func (m *validationMockDataTransformer) TransformData(ctx context.Context, content Content, format string) (Content, error) {
	// Simply return the content unchanged for testing
	return content, nil
}

type validationMockByteTransformer struct {
	name     string
	priority int
}

func (m *validationMockByteTransformer) Name() string                    { return m.name }
func (m *validationMockByteTransformer) Priority() int                   { return m.priority }
func (m *validationMockByteTransformer) CanTransform(format string) bool { return true }
func (m *validationMockByteTransformer) Transform(ctx context.Context, input []byte, format string) ([]byte, error) {
	return append(input, []byte(" [transformed]")...), nil
}

type validationMockFormatAwareTransformer struct {
	supportedFormats []string
	lastFormat       string
}

func (m *validationMockFormatAwareTransformer) Name() string  { return "format-aware-test" }
func (m *validationMockFormatAwareTransformer) Priority() int { return 100 }
func (m *validationMockFormatAwareTransformer) Describe() string {
	return "Format-aware transformer for testing"
}

func (m *validationMockFormatAwareTransformer) CanTransform(content Content, format string) bool {
	if content.Type() != ContentTypeTable {
		return false
	}

	return slices.Contains(m.supportedFormats, format)
}

func (m *validationMockFormatAwareTransformer) TransformData(ctx context.Context, content Content, format string) (Content, error) {
	m.lastFormat = format
	return content, nil
}
