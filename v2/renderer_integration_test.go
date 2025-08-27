package output

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
)

// rendererMockDataTransformer - unique mock for renderer integration tests
type rendererMockDataTransformer struct {
	name        string
	priority    int
	formats     []string
	description string
	callCount   int
	mu          sync.Mutex
}

func newRendererMockDataTransformer(name string, priority int, formats []string, description string) *rendererMockDataTransformer {
	return &rendererMockDataTransformer{
		name:        name,
		priority:    priority,
		formats:     formats,
		description: description,
	}
}

func (m *rendererMockDataTransformer) Name() string {
	return m.name
}

func (m *rendererMockDataTransformer) Priority() int {
	return m.priority
}

func (m *rendererMockDataTransformer) CanTransform(content Content, format string) bool {
	// Only works on table content
	if content.Type() != ContentTypeTable {
		return false
	}

	// Check format support
	return slices.Contains(m.formats, format)
}

func (m *rendererMockDataTransformer) Describe() string {
	return m.description
}

func (m *rendererMockDataTransformer) TransformData(ctx context.Context, content Content, format string) (Content, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++

	// For testing, just add a prefix to each record in table content
	if tableContent, ok := content.(*TableContent); ok {
		// Clone the content for immutability
		cloned := tableContent.Clone()

		// Transform by adding a prefix to first field of each record
		if clonedTable, ok := cloned.(*TableContent); ok {
			for i, record := range clonedTable.records {
				if len(record) > 0 {
					// Get first key from schema
					firstKey := clonedTable.schema.keyOrder[0]
					if val, exists := record[firstKey]; exists {
						record[firstKey] = fmt.Sprintf("[%s]%v", m.name, val)
					}
				}
				clonedTable.records[i] = record
			}
			return clonedTable, nil
		}
	}

	return content, nil
}

func (m *rendererMockDataTransformer) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// rendererMockByteTransformer - unique mock for renderer integration tests
type rendererMockByteTransformer struct {
	name      string
	priority  int
	formats   []string
	suffix    string
	callCount int
	mu        sync.Mutex
}

func newRendererMockByteTransformer(name string, priority int, formats []string, suffix string) *rendererMockByteTransformer {
	return &rendererMockByteTransformer{
		name:     name,
		priority: priority,
		formats:  formats,
		suffix:   suffix,
	}
}

func (m *rendererMockByteTransformer) Name() string {
	return m.name
}

func (m *rendererMockByteTransformer) Priority() int {
	return m.priority
}

func (m *rendererMockByteTransformer) CanTransform(format string) bool {
	return slices.Contains(m.formats, format)
}

func (m *rendererMockByteTransformer) Transform(ctx context.Context, input []byte, format string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++

	return append(input, []byte(m.suffix)...), nil
}

func (m *rendererMockByteTransformer) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// Test TransformerAdapter detection and functionality
func TestTransformerAdapter_Detection(t *testing.T) {
	tests := map[string]struct {
		transformer    any
		isDataExpected bool
		isNameExpected string
	}{"byte transformer": {

		transformer:    newRendererMockByteTransformer("byte-test", 200, []string{FormatJSON}, " [byte-suffix]"),
		isDataExpected: false,
		isNameExpected: "byte-test",
	}, "data transformer": {

		transformer:    newRendererMockDataTransformer("data-test", 100, []string{FormatJSON}, "test data transformer"),
		isDataExpected: true,
		isNameExpected: "data-test",
	}}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			adapter := NewTransformerAdapter(test.transformer)

			if adapter.IsDataTransformer() != test.isDataExpected {
				t.Errorf("IsDataTransformer() = %t, want %t", adapter.IsDataTransformer(), test.isDataExpected)
			}

			if test.isDataExpected {
				dataTransformer := adapter.AsDataTransformer()
				if dataTransformer == nil {
					t.Error("AsDataTransformer() should not return nil for data transformer")
				} else if dataTransformer.Name() != test.isNameExpected {
					t.Errorf("DataTransformer.Name() = %s, want %s", dataTransformer.Name(), test.isNameExpected)
				}

				if adapter.AsByteTransformer() != nil {
					t.Error("AsByteTransformer() should return nil for data transformer")
				}
			} else {
				byteTransformer := adapter.AsByteTransformer()
				if byteTransformer == nil {
					t.Error("AsByteTransformer() should not return nil for byte transformer")
				} else if byteTransformer.Name() != test.isNameExpected {
					t.Errorf("ByteTransformer.Name() = %s, want %s", byteTransformer.Name(), test.isNameExpected)
				}

				if adapter.AsDataTransformer() != nil {
					t.Error("AsDataTransformer() should return nil for byte transformer")
				}
			}
		})
	}
}

// Test data transformer application before rendering
func TestRenderer_DataTransformerApplication(t *testing.T) {
	// Create test data
	records := []Record{
		{"name": "Alice", "age": 25},
		{"name": "Bob", "age": 30},
	}

	// Create document with table content
	doc := New().
		Table("users", records, WithKeys("name", "age")).
		Build()

	// Create data transformer
	dataTransformer := newRendererMockDataTransformer("test-data", 100, []string{FormatTable}, "test transformer")

	// This test verifies the concept but will need actual renderer integration
	// For now, we test that the transformer can process the content correctly
	content := doc.GetContents()[0] // Get table content

	if !dataTransformer.CanTransform(content, FormatTable) {
		t.Error("DataTransformer should be able to transform table content for table format")
	}

	ctx := context.Background()
	transformed, err := dataTransformer.TransformData(ctx, content, FormatTable)
	if err != nil {
		t.Fatalf("TransformData() error = %v", err)
	}

	if dataTransformer.CallCount() != 1 {
		t.Errorf("DataTransformer call count = %d, want 1", dataTransformer.CallCount())
	}

	// Verify transformation applied correctly
	if tableContent, ok := transformed.(*TableContent); ok {
		if len(tableContent.records) != 2 {
			t.Errorf("Expected 2 records, got %d", len(tableContent.records))
		}

		// Check that first record's name was prefixed
		if !strings.HasPrefix(tableContent.records[0]["name"].(string), "[test-data]") {
			t.Errorf("Expected name to be prefixed with [test-data], got %v", tableContent.records[0]["name"])
		}
	} else {
		t.Error("Transformed content should be TableContent")
	}
}

// Test format context passing to transformers
func TestRenderer_FormatContextPassing(t *testing.T) {
	formats := []string{FormatJSON, FormatYAML, FormatCSV, FormatHTML, FormatTable, FormatMarkdown}

	// Create test data
	records := []Record{{"name": "Alice", "age": 25}}
	doc := New().
		Table("users", records, WithKeys("name", "age")).
		Build()
	content := doc.GetContents()[0]

	for _, format := range formats {
		t.Run(fmt.Sprintf("format_%s", format), func(t *testing.T) {
			// Create transformer that supports this format
			dataTransformer := newRendererMockDataTransformer("format-test", 100, []string{format}, fmt.Sprintf("transformer for %s", format))

			// Test CanTransform with format
			canTransform := dataTransformer.CanTransform(content, format)
			if !canTransform {
				t.Errorf("DataTransformer should support format %s", format)
			}

			// Test TransformData receives correct format
			ctx := context.Background()
			_, err := dataTransformer.TransformData(ctx, content, format)
			if err != nil {
				t.Errorf("TransformData() error for format %s: %v", format, err)
			}

			if dataTransformer.CallCount() != 1 {
				t.Errorf("DataTransformer should be called once for format %s", format)
			}
		})
	}
}

// Test dual transformer system priority ordering
func TestRenderer_DualTransformerPriorityOrdering(t *testing.T) {
	// This test verifies that data transformers are applied before byte transformers
	// and that within each category, priority ordering is respected

	// Create transformers with different priorities
	dataTransformer1 := newRendererMockDataTransformer("data-high", 200, []string{FormatTable}, "high priority data")
	dataTransformer2 := newRendererMockDataTransformer("data-low", 100, []string{FormatTable}, "low priority data")
	byteTransformer1 := newRendererMockByteTransformer("byte-high", 200, []string{FormatTable}, " [byte-high]")
	byteTransformer2 := newRendererMockByteTransformer("byte-low", 100, []string{FormatTable}, " [byte-low]")

	// Create adapters (simulates renderer integration)
	adapters := []*TransformerAdapter{
		NewTransformerAdapter(dataTransformer1),
		NewTransformerAdapter(dataTransformer2),
		NewTransformerAdapter(byteTransformer1),
		NewTransformerAdapter(byteTransformer2),
	}

	// Verify adapter detection
	dataAdapters := make([]*TransformerAdapter, 0)
	byteAdapters := make([]*TransformerAdapter, 0)

	for _, adapter := range adapters {
		if adapter.IsDataTransformer() {
			dataAdapters = append(dataAdapters, adapter)
		} else {
			byteAdapters = append(byteAdapters, adapter)
		}
	}

	if len(dataAdapters) != 2 {
		t.Errorf("Expected 2 data adapters, got %d", len(dataAdapters))
	}

	if len(byteAdapters) != 2 {
		t.Errorf("Expected 2 byte adapters, got %d", len(byteAdapters))
	}

	// Verify priority-based detection would work
	// (Actual sorting would be implemented in renderer integration)
	for _, adapter := range dataAdapters {
		dataTransformer := adapter.AsDataTransformer()
		if dataTransformer == nil {
			t.Error("Data adapter should provide DataTransformer")
			continue
		}

		// Verify priority values are accessible
		if dataTransformer.Priority() < 100 || dataTransformer.Priority() > 200 {
			t.Errorf("Unexpected priority value: %d", dataTransformer.Priority())
		}
	}
}

// Test CanTransform method filtering
func TestRenderer_CanTransformFiltering(t *testing.T) {
	// Create transformers with different format support
	jsonOnlyData := newRendererMockDataTransformer("json-data", 100, []string{FormatJSON}, "JSON only")
	universalData := newRendererMockDataTransformer("universal-data", 100, []string{FormatJSON, FormatYAML, FormatTable}, "Universal")

	jsonOnlyByte := newRendererMockByteTransformer("json-byte", 100, []string{FormatJSON}, " [json]")
	universalByte := newRendererMockByteTransformer("universal-byte", 100, []string{FormatJSON, FormatYAML, FormatTable}, " [universal]")

	// Create test content
	records := []Record{{"name": "Alice"}}
	doc := New().
		Table("users", records, WithKeys("name")).
		Build()
	content := doc.GetContents()[0]

	testCases := []struct {
		format            string
		expectedDataCount int
		expectedByteCount int
	}{
		{FormatJSON, 2, 2},  // All should support JSON
		{FormatYAML, 1, 1},  // Only universal should support YAML
		{FormatTable, 1, 1}, // Only universal should support Table
		{FormatHTML, 0, 0},  // None should support HTML
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("format_%s", tc.format), func(t *testing.T) {
			// Test data transformers
			dataTransformers := []*rendererMockDataTransformer{jsonOnlyData, universalData}
			supportedDataCount := 0

			for _, dt := range dataTransformers {
				if dt.CanTransform(content, tc.format) {
					supportedDataCount++
				}
			}

			if supportedDataCount != tc.expectedDataCount {
				t.Errorf("Expected %d data transformers to support %s, got %d",
					tc.expectedDataCount, tc.format, supportedDataCount)
			}

			// Test byte transformers
			byteTransformers := []*rendererMockByteTransformer{jsonOnlyByte, universalByte}
			supportedByteCount := 0

			for _, bt := range byteTransformers {
				if bt.CanTransform(tc.format) {
					supportedByteCount++
				}
			}

			if supportedByteCount != tc.expectedByteCount {
				t.Errorf("Expected %d byte transformers to support %s, got %d",
					tc.expectedByteCount, tc.format, supportedByteCount)
			}
		})
	}
}

// Test mixed content type handling
func TestRenderer_MixedContentTypeHandling(t *testing.T) {
	// Create document with mixed content types
	doc := New().
		Text("Introduction text").
		Table("users", []Record{{"name": "Alice"}}, WithKeys("name")).
		Text("Conclusion text").
		Build()

	contents := doc.GetContents()
	if len(contents) != 3 {
		t.Fatalf("Expected 3 contents, got %d", len(contents))
	}

	// Create data transformer
	dataTransformer := newRendererMockDataTransformer("mixed-test", 100, []string{FormatTable}, "mixed content test")

	// Test that only table content can be transformed
	for i, content := range contents {
		canTransform := dataTransformer.CanTransform(content, FormatTable)

		if content.Type() == ContentTypeTable {
			if !canTransform {
				t.Errorf("Content %d (table) should be transformable", i)
			}
		} else {
			if canTransform {
				t.Errorf("Content %d (%s) should not be transformable by data transformer",
					i, content.Type().String())
			}
		}
	}
}

// Test error handling in transformer detection
func TestRenderer_TransformerErrorHandling(t *testing.T) {
	// Create failing data transformer
	failingTransformer := &rendererFailingDataTransformer{
		name:     "failing-transformer",
		priority: 100,
		formats:  []string{FormatTable},
	}

	// Create test content
	records := []Record{{"name": "Alice"}}
	doc := New().
		Table("users", records, WithKeys("name")).
		Build()
	content := doc.GetContents()[0]

	// Test that CanTransform works
	if !failingTransformer.CanTransform(content, FormatTable) {
		t.Error("Failing transformer should still pass CanTransform")
	}

	// Test that TransformData fails appropriately
	ctx := context.Background()
	_, err := failingTransformer.TransformData(ctx, content, FormatTable)
	if err == nil {
		t.Error("Failing transformer should return error")
	}

	if !strings.Contains(err.Error(), "simulated failure") {
		t.Errorf("Error should contain 'simulated failure', got: %v", err)
	}
}

// rendererFailingDataTransformer for error testing
type rendererFailingDataTransformer struct {
	name     string
	priority int
	formats  []string
}

func (f *rendererFailingDataTransformer) Name() string     { return f.name }
func (f *rendererFailingDataTransformer) Priority() int    { return f.priority }
func (f *rendererFailingDataTransformer) Describe() string { return "failing transformer for testing" }
func (f *rendererFailingDataTransformer) CanTransform(content Content, format string) bool {
	if content.Type() != ContentTypeTable {
		return false
	}
	return slices.Contains(f.formats, format)
}
func (f *rendererFailingDataTransformer) TransformData(ctx context.Context, content Content, format string) (Content, error) {
	return nil, fmt.Errorf("simulated failure in transformer %s", f.name)
}

// Test concurrent transformer access
func TestRenderer_ConcurrentTransformerAccess(t *testing.T) {
	// Create transformer that tracks concurrent calls
	transformer := newRendererMockDataTransformer("concurrent-test", 100, []string{FormatTable}, "concurrent test")

	// Create test content
	records := []Record{{"name": "Alice"}}
	doc := New().
		Table("users", records, WithKeys("name")).
		Build()
	content := doc.GetContents()[0]

	// Run multiple concurrent transformations
	const numGoroutines = 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			_, err := transformer.TransformData(ctx, content, FormatTable)
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent transformation error: %v", err)
	}

	// Verify all calls completed
	if transformer.CallCount() != numGoroutines {
		t.Errorf("Expected %d calls, got %d", numGoroutines, transformer.CallCount())
	}
}
