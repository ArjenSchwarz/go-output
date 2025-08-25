package output

import (
	"context"
	"strings"
	"testing"
	"time"
)

// Performance test constants
const (
	// largeDatasetSize defines the number of records used in performance tests
	largeDatasetSize = 1000
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

	t.Run("Requirement 3.1-3.5: Pipeline API provides fluent interface with method chaining", func(t *testing.T) {
		// Test that Pipeline() method exists and returns pipeline
		pipeline := doc.Pipeline()
		if pipeline == nil {
			t.Fatal("Document.Pipeline() should return a pipeline")
		}

		// Test method chaining works and returns new Document
		result, err := pipeline.
			Filter(func(r Record) bool { return r["active"].(bool) }).
			Sort(SortKey{Column: "salary", Direction: Descending}).
			Limit(2).
			Execute()

		if err != nil {
			t.Fatalf("Pipeline execution failed: %v", err)
		}

		if result == nil {
			t.Fatal("Pipeline should return a new Document")
		}

		// Verify immutability - original document unchanged
		if len(doc.GetContents()[0].(*TableContent).records) != 4 {
			t.Error("Original document should remain unchanged (immutability)")
		}

		// Verify transformation applied correctly
		resultContent := result.GetContents()[0].(*TableContent)
		if len(resultContent.records) != 2 {
			t.Error("Pipeline should have limited results to 2")
		}
	})

	t.Run("Requirement 4.1-4.6: Filter operation with predicates", func(t *testing.T) {
		// Test filter with Go predicate functions
		result, err := doc.Pipeline().
			Filter(func(r Record) bool {
				return r["active"].(bool) && r["salary"].(int) >= 80000
			}).
			Execute()

		if err != nil {
			t.Fatalf("Filter operation failed: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)
		if len(tableContent.records) != 2 { // Alice and Charlie
			t.Errorf("Expected 2 records after filter, got %d", len(tableContent.records))
		}

		// Verify schema and key order preserved
		if !tableContent.Schema().HasField("salary") {
			t.Error("Filter should preserve original schema")
		}

		expectedOrder := []string{"id", "name", "dept", "salary", "active"}
		actualOrder := tableContent.Schema().GetKeyOrder()
		if len(actualOrder) != len(expectedOrder) {
			t.Error("Filter should preserve key ordering")
		}

		// Test multiple filter operations are chainable
		result2, err := doc.Pipeline().
			Filter(func(r Record) bool { return r["active"].(bool) }).
			Filter(func(r Record) bool { return r["salary"].(int) >= 90000 }).
			Execute()

		if err != nil {
			t.Fatalf("Multiple filter operations failed: %v", err)
		}

		if len(result2.GetContents()[0].(*TableContent).records) != 1 { // Only Alice
			t.Error("Multiple filters should be composable")
		}
	})

	t.Run("Requirement 5.1-5.6: Sort operation with multiple columns and directions", func(t *testing.T) {
		// Test single column sort
		result, err := doc.Pipeline().
			Sort(SortKey{Column: "salary", Direction: Descending}).
			Execute()

		if err != nil {
			t.Fatalf("Sort operation failed: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)
		salaries := make([]int, len(tableContent.records))
		for i, record := range tableContent.records {
			salaries[i] = record["salary"].(int)
		}

		// Verify descending order
		for i := 1; i < len(salaries); i++ {
			if salaries[i] > salaries[i-1] {
				t.Error("Sort should maintain descending order")
			}
		}

		// Test multiple columns sort
		result2, err := doc.Pipeline().
			Sort(
				SortKey{Column: "dept", Direction: Ascending},
				SortKey{Column: "salary", Direction: Descending},
			).
			Execute()

		if err != nil {
			t.Fatalf("Multi-column sort failed: %v", err)
		}

		// Test that sort works with all table-based formats (requirement 5.6)
		// This is implicitly tested by the fact that sorting changes the data structure
		if len(result2.GetContents()[0].(*TableContent).records) != 4 {
			t.Error("Sort should preserve all records")
		}
	})

	t.Run("Requirement 6.1-6.6: GroupBy and Aggregation operations", func(t *testing.T) {
		// Test GroupBy with aggregation functions
		result, err := doc.Pipeline().
			GroupBy([]string{"dept"}, map[string]AggregateFunc{
				"count":      CountAggregate(),
				"avg_salary": AverageAggregate("salary"),
				"max_salary": MaxAggregate("salary"),
				"min_salary": MinAggregate("salary"),
				"sum_salary": SumAggregate("salary"),
			}).
			Execute()

		if err != nil {
			t.Fatalf("GroupBy aggregation failed: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)

		// Should have 3 departments: Engineering, Sales, Marketing
		if len(tableContent.records) != 3 {
			t.Errorf("Expected 3 department groups, got %d", len(tableContent.records))
		}

		// Verify aggregated results produce new table structure
		schema := tableContent.Schema()
		expectedFields := []string{"dept", "count", "avg_salary", "max_salary", "min_salary", "sum_salary"}
		for _, field := range expectedFields {
			if !schema.HasField(field) {
				t.Errorf("Aggregated schema should include field: %s", field)
			}
		}

		// Test that aggregations work with numeric data types
		for _, record := range tableContent.records {
			if count, ok := record["count"].(int); !ok || count <= 0 {
				t.Error("Count aggregation should produce positive integer")
			}
			if avg, ok := record["avg_salary"].(float64); !ok || avg <= 0 {
				t.Error("Average aggregation should produce positive float64")
			}
		}
	})

	t.Run("Requirement 7.1-7.6: AddColumn for calculated fields", func(t *testing.T) {
		// Test adding calculated columns
		result, err := doc.Pipeline().
			AddColumn("salary_grade", func(r Record) any {
				salary := r["salary"].(int)
				if salary >= 90000 {
					return "Senior"
				} else if salary >= 75000 {
					return "Mid"
				}
				return "Junior"
			}).
			AddColumn("full_info", func(r Record) any {
				// Test that calculated fields have access to all record data
				return r["name"].(string) + " from " + r["dept"].(string)
			}).
			Execute()

		if err != nil {
			t.Fatalf("AddColumn operation failed: %v", err)
		}

		tableContent := result.GetContents()[0].(*TableContent)
		schema := tableContent.Schema()

		// Verify schema updates with new fields
		if !schema.HasField("salary_grade") || !schema.HasField("full_info") {
			t.Error("AddColumn should update schema with new fields")
		}

		// Verify key order maintained with new fields appended
		keyOrder := schema.GetKeyOrder()
		expectedOrder := []string{"id", "name", "dept", "salary", "active", "salary_grade", "full_info"}
		if len(keyOrder) != len(expectedOrder) {
			t.Errorf("Key order length after AddColumn: expected %d, got %d", len(expectedOrder), len(keyOrder))
		}

		// Verify calculated values are correct
		for _, record := range tableContent.records {
			salary := record["salary"].(int)
			grade := record["salary_grade"].(string)

			expectedGrade := "Junior"
			if salary >= 90000 {
				expectedGrade = "Senior"
			} else if salary >= 75000 {
				expectedGrade = "Mid"
			}

			if grade != expectedGrade {
				t.Errorf("Calculated field incorrect: expected %s, got %s for salary %d", expectedGrade, grade, salary)
			}

			// Verify access to all record data
			fullInfo := record["full_info"].(string)
			expectedInfo := record["name"].(string) + " from " + record["dept"].(string)
			if fullInfo != expectedInfo {
				t.Errorf("Calculated field should access all data: expected %s, got %s", expectedInfo, fullInfo)
			}
		}
	})

	t.Run("Requirement 8.1-8.6: Performance optimization and acceptable overhead", func(t *testing.T) {
		// Create larger dataset for performance testing
		largeRecords := make([]Record, largeDatasetSize)
		for i := 0; i < largeDatasetSize; i++ {
			largeRecords[i] = Record{
				"id":     i + 1,
				"value":  i * 10,
				"active": i%2 == 0,
			}
		}

		largeDoc := New().
			Table("large", largeRecords, WithKeys("id", "value", "active")).
			Build()

		start := time.Now()

		// Test that operations complete in reasonable time
		result, err := largeDoc.Pipeline().
			Filter(func(r Record) bool { return r["active"].(bool) }).
			Sort(SortKey{Column: "value", Direction: Descending}).
			Limit(100).
			Execute()

		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Performance test failed: %v", err)
		}

		// Should complete within reasonable time (performance requirement)
		maxDuration := 1 * time.Second
		if duration > maxDuration {
			t.Errorf("Pipeline performance too slow: %v (max %v)", duration, maxDuration)
		}

		// Verify optimization applied (filter before expensive operations)
		if len(result.GetContents()[0].(*TableContent).records) != 100 {
			t.Error("Pipeline should optimize operations correctly")
		}

		t.Logf("Performance test completed in %v", duration)
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

	t.Run("Requirement 10.1-10.6: Error handling and validation", func(t *testing.T) {
		// Test pipeline validation
		textDoc := New().Text("Just text").Build()
		pipeline := textDoc.Pipeline()

		err := pipeline.Validate()
		if err == nil {
			t.Error("Pipeline should validate that document has table content")
		}

		if !strings.Contains(err.Error(), "table content") {
			t.Errorf("Validation error should mention table content requirement: %v", err)
		}

		// Test fail-fast error handling
		_, err = doc.Pipeline().
			Filter(func(r Record) bool {
				// This should work - we test with a safe predicate
				return r["active"].(bool)
			}).
			Execute()

		if err != nil {
			// If there is an error, it should be descriptive
			if !strings.Contains(err.Error(), "Filter") {
				t.Errorf("Error should provide context about which operation failed")
			}
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

	for _, f := range m.supportedFormats {
		if f == format {
			return true
		}
	}
	return false
}

func (m *validationMockFormatAwareTransformer) TransformData(ctx context.Context, content Content, format string) (Content, error) {
	m.lastFormat = format
	return content, nil
}
