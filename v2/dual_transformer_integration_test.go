package output

import (
	"context"
	"slices"
	"strings"
	"testing"
)

// Test end-to-end dual transformer system integration
func TestDualTransformerSystem_Integration(t *testing.T) {
	// Create test document with table content
	records := []Record{
		{"name": "Alice", "age": 25, "status": "active"},
		{"name": "Bob", "age": 30, "status": "inactive"},
		{"name": "Charlie", "age": 35, "status": "active"},
	}

	doc := New().
		Table("users", records, WithKeys("name", "age", "status")).
		Build()

	// Create data transformer (filters and modifies data)
	dataTransformer := &testDataTransformer{
		name:     "test-filter",
		priority: 100,
		formats:  []string{FormatHTML},
	}

	// Create byte transformer (adds suffix)
	byteTransformer := &testByteTransformer{
		name:     "test-suffix",
		priority: 200,
		formats:  []string{FormatHTML},
		suffix:   "\n<!-- Processed by byte transformer -->",
	}

	// Create HTML renderer with transformers configured
	renderer := HTML.Renderer

	// Manually configure the baseRenderer with transformers
	// (This simulates proper configuration that would be done through constructor options)
	if htmlRenderer, ok := renderer.(*htmlRenderer); ok {
		htmlRenderer.baseRenderer.config = RendererConfig{
			DataTransformers: []*TransformerAdapter{
				NewTransformerAdapter(dataTransformer),
			},
			ByteTransformers: NewTransformPipeline(),
		}
		htmlRenderer.baseRenderer.config.ByteTransformers.Add(byteTransformer)
	}

	// Render the document
	ctx := context.Background()
	output, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	outputStr := string(output)

	// Verify data transformation was applied (only active users should remain)
	if strings.Contains(outputStr, "Bob") {
		t.Error("Data transformer should have filtered out inactive user Bob")
	}

	if !strings.Contains(outputStr, "Alice") {
		t.Error("Data transformer should have kept active user Alice")
	}

	if !strings.Contains(outputStr, "Charlie") {
		t.Error("Data transformer should have kept active user Charlie")
	}

	// Verify data transformer modified the names
	if !strings.Contains(outputStr, "[FILTERED]Alice") {
		t.Error("Data transformer should have prefixed Alice's name with [FILTERED]")
	}

	if !strings.Contains(outputStr, "[FILTERED]Charlie") {
		t.Error("Data transformer should have prefixed Charlie's name with [FILTERED]")
	}

	// Verify byte transformation was applied
	if !strings.Contains(outputStr, "<!-- Processed by byte transformer -->") {
		t.Error("Byte transformer should have added its suffix")
	}

	// Verify transformers were called
	if dataTransformer.callCount != 1 {
		t.Errorf("Data transformer should have been called once, got %d", dataTransformer.callCount)
	}

	if byteTransformer.callCount != 1 {
		t.Errorf("Byte transformer should have been called once, got %d", byteTransformer.callCount)
	}
}

// Test that data transformers are not applied for unsupported formats
func TestDualTransformerSystem_FormatFiltering(t *testing.T) {
	// Create test document
	records := []Record{{"name": "Alice", "age": 25}}
	doc := New().
		Table("users", records, WithKeys("name", "age")).
		Build()

	// Create data transformer that only supports JSON
	dataTransformer := &testDataTransformer{
		name:     "json-only",
		priority: 100,
		formats:  []string{FormatJSON}, // Only supports JSON
	}

	// Create HTML renderer with data transformer
	renderer := HTML.Renderer
	if htmlRenderer, ok := renderer.(*htmlRenderer); ok {
		htmlRenderer.baseRenderer.config = RendererConfig{
			DataTransformers: []*TransformerAdapter{
				NewTransformerAdapter(dataTransformer),
			},
			ByteTransformers: NewTransformPipeline(),
		}
	}

	// Render as HTML (should not apply data transformer)
	ctx := context.Background()
	output, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	outputStr := string(output)

	// Verify data transformer was NOT applied (name should not be prefixed)
	if strings.Contains(outputStr, "[FILTERED]") {
		t.Error("Data transformer should not have been applied for unsupported format")
	}

	// Verify transformer was not called
	if dataTransformer.callCount != 0 {
		t.Errorf("Data transformer should not have been called for unsupported format, got %d calls", dataTransformer.callCount)
	}
}

// Test priority ordering of transformers
func TestDualTransformerSystem_PriorityOrdering(t *testing.T) {
	records := []Record{{"name": "Alice"}}
	doc := New().
		Table("users", records, WithKeys("name")).
		Build()

	// Create data transformers with different priorities
	highPriorityTransformer := &testDataTransformer{
		name:     "high-priority",
		priority: 50, // Lower number = higher priority
		formats:  []string{FormatHTML},
		prefix:   "[HIGH]",
	}

	lowPriorityTransformer := &testDataTransformer{
		name:     "low-priority",
		priority: 200, // Higher number = lower priority
		formats:  []string{FormatHTML},
		prefix:   "[LOW]",
	}

	renderer := HTML.Renderer
	if htmlRenderer, ok := renderer.(*htmlRenderer); ok {
		htmlRenderer.baseRenderer.config = RendererConfig{
			DataTransformers: []*TransformerAdapter{
				NewTransformerAdapter(lowPriorityTransformer),  // Add low priority first
				NewTransformerAdapter(highPriorityTransformer), // Add high priority second
			},
			ByteTransformers: NewTransformPipeline(),
		}
	}

	ctx := context.Background()
	output, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	outputStr := string(output)

	// High priority transformer should apply first, so we should see [LOW][HIGH]Alice
	// (each transformer prefixes the existing name, so final result has transformers in reverse order)
	if !strings.Contains(outputStr, "[LOW][HIGH]Alice") {
		t.Errorf("Expected [LOW][HIGH]Alice, but got: %s", outputStr)
	}
}

// testDataTransformer for integration testing
type testDataTransformer struct {
	name      string
	priority  int
	formats   []string
	prefix    string
	callCount int
}

func (t *testDataTransformer) Name() string     { return t.name }
func (t *testDataTransformer) Priority() int    { return t.priority }
func (t *testDataTransformer) Describe() string { return "test data transformer" }

func (t *testDataTransformer) CanTransform(content Content, format string) bool {
	if content.Type() != ContentTypeTable {
		return false
	}

	return slices.Contains(t.formats, format)
}

func (t *testDataTransformer) TransformData(ctx context.Context, content Content, format string) (Content, error) {
	t.callCount++

	tableContent, ok := content.(*TableContent)
	if !ok {
		return content, nil
	}

	// Clone the content
	cloned := tableContent.Clone().(*TableContent)

	// Filter and modify records
	filteredRecords := make([]Record, 0)

	for _, record := range cloned.records {
		// Filter: only keep "active" status records (for test-filter)
		// Or keep all records for priority test
		if t.name == "test-filter" {
			if status, ok := record["status"]; ok && status != "active" {
				continue // Skip inactive records
			}
		}

		// Modify: add prefix to name
		if name, ok := record["name"]; ok {
			prefix := t.prefix
			if prefix == "" {
				prefix = "[FILTERED]"
			}
			record["name"] = prefix + name.(string)
		}

		filteredRecords = append(filteredRecords, record)
	}

	cloned.records = filteredRecords
	return cloned, nil
}

// testByteTransformer for integration testing
type testByteTransformer struct {
	name      string
	priority  int
	formats   []string
	suffix    string
	callCount int
}

func (t *testByteTransformer) Name() string  { return t.name }
func (t *testByteTransformer) Priority() int { return t.priority }

func (t *testByteTransformer) CanTransform(format string) bool {
	return slices.Contains(t.formats, format)
}

func (t *testByteTransformer) Transform(ctx context.Context, input []byte, format string) ([]byte, error) {
	t.callCount++
	return append(input, []byte(t.suffix)...), nil
}
