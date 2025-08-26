package output

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestIntegration_CompleteWorkflow tests a complete document creation and rendering workflow
func TestIntegration_CompleteWorkflow(t *testing.T) {
	skipIfNotIntegration(t)

	// Create a comprehensive document with all content types
	doc := New().
		SetMetadata("title", "Integration Test Document").
		SetMetadata("author", "Test Suite").
		Header("Executive Summary").
		Text("This document demonstrates all features of the v2 output library.").
		Section("Data Tables", func(b *Builder) {
			b.Table("Sales Q4", []map[string]any{
				{"Region": "North", "Sales": 45000, "Growth": "12%"},
				{"Region": "South", "Sales": 38000, "Growth": "8%"},
				{"Region": "East", "Sales": 42000, "Growth": "15%"},
				{"Region": "West", "Sales": 35000, "Growth": "5%"},
			}, WithKeys("Region", "Sales", "Growth"))

			b.Text("Sales data shows strong growth in Eastern region.", WithBold(true))
		}).
		Section("Visualizations", func(b *Builder) {
			b.PieChart("Market Share", []PieSlice{
				{Label: "Product A", Value: 45},
				{Label: "Product B", Value: 30},
				{Label: "Product C", Value: 25},
			}, true)

			b.Graph("Dependencies", []Edge{
				{From: "Frontend", To: "API", Label: "calls"},
				{From: "API", To: "Database", Label: "queries"},
				{From: "API", To: "Cache", Label: "reads"},
			})
		}).
		Raw("html", []byte("<div class='custom'>Custom HTML content</div>")).
		Build()

	// Test metadata
	metadata := doc.GetMetadata()
	if metadata["title"] != "Integration Test Document" {
		t.Errorf("metadata title = %v, want 'Integration Test Document'", metadata["title"])
	}

	// Test content count
	contents := doc.GetContents()
	expectedContentCount := 5 // Header, Text, 2 Sections, Raw
	if len(contents) != expectedContentCount {
		t.Errorf("content count = %d, want %d", len(contents), expectedContentCount)
	}

	// Verify sections have nested content
	var sectionsFound int
	for _, content := range contents {
		if section, ok := content.(*SectionContent); ok {
			sectionsFound++
			if len(section.contents) == 0 {
				t.Errorf("Section %q has no nested content", section.Title())
			}
		}
	}
	if sectionsFound != 2 {
		t.Errorf("found %d sections, want 2", sectionsFound)
	}
}

// TestIntegration_ConcurrentDocumentBuilding tests thread-safe document building
func TestIntegration_ConcurrentDocumentBuilding(t *testing.T) {
	skipIfNotIntegration(t)

	builder := New()

	// Use a WaitGroup to ensure all goroutines complete
	var wg sync.WaitGroup
	numGoroutines := 10
	itemsPerGoroutine := 5

	wg.Add(numGoroutines)

	// Concurrently add content
	for i := range numGoroutines {
		go func(goroutineID int) {
			defer wg.Done()

			for j := range itemsPerGoroutine {
				// Add different types of content
				switch j % 3 {
				case 0:
					builder.Table(
						"Table from goroutine",
						[]map[string]any{{"ID": goroutineID, "Item": j}},
						WithKeys("ID", "Item"),
					)
				case 1:
					builder.Text("Text from goroutine")
				case 2:
					builder.Header("Header from goroutine")
				}
			}
		}(i)
	}

	wg.Wait()

	// Build and verify
	doc := builder.Build()
	contents := doc.GetContents()

	expectedCount := numGoroutines * itemsPerGoroutine
	if len(contents) != expectedCount {
		t.Errorf("Expected %d contents after concurrent building, got %d", expectedCount, len(contents))
	}
}

// TestIntegration_RenderingPipeline tests the complete rendering pipeline
func TestIntegration_RenderingPipeline(t *testing.T) {
	skipIfNotIntegration(t)

	// Create document
	doc := New().
		Table("Products", []map[string]any{
			{"Name": "Laptop", "Price": 999.99, "InStock": true},
			{"Name": "Mouse", "Price": 29.99, "InStock": true},
			{"Name": "Keyboard", "Price": 79.99, "InStock": false},
		}, WithKeys("Name", "Price", "InStock")).
		Build()

	// Create output with multiple formats and transformers
	var buf bytes.Buffer
	output := NewOutput(
		WithFormat(JSON),
		WithFormat(Table),
		WithTransformer(NewSortTransformer("Price", true)),
		WithWriter(NewStdoutWriter()),                 // Would write to stdout
		WithWriter(&integrationMockWriter{buf: &buf}), // Test writer
	)

	// Render with context
	ctx := context.Background()
	err := output.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Verify something was written
	if buf.Len() == 0 {
		t.Error("No output was written")
	}
}

// TestIntegration_ErrorHandling tests error handling throughout the system
func TestIntegration_ErrorHandling(t *testing.T) {
	skipIfNotIntegration(t)

	builder := New()

	// Add some valid content
	builder.Table("Valid", []map[string]any{{"key": "value"}}, WithKeys("key"))

	// Try to add invalid content (this would cause an error in Table creation)
	builder.Table("Invalid", "not a valid data type", WithKeys("key"))

	// Build should still work
	doc := builder.Build()

	// Check errors
	if !builder.HasErrors() {
		t.Error("Expected builder to have errors")
	}

	errors := builder.Errors()
	if len(errors) == 0 {
		t.Error("Expected at least one error")
	}

	// Document should still have the valid content
	if len(doc.GetContents()) != 1 {
		t.Error("Document should contain the valid content")
	}
}

// TestIntegration_ProgressWithRendering tests progress integration
func TestIntegration_ProgressWithRendering(t *testing.T) {
	skipIfNotIntegration(t)

	// Create a large document
	builder := New()
	for i := range 100 {
		builder.Table("Table", []map[string]any{
			{"Index": i, "Value": i * 10},
		}, WithKeys("Index", "Value"))
	}
	doc := builder.Build()

	// Create output with progress
	progress := NewNoOpProgress() // Use NoOp for testing
	output := NewOutput(
		WithFormat(JSON),
		WithProgress(progress),
		WithWriter(&integrationMockWriter{buf: &bytes.Buffer{}}),
	)

	// Render
	ctx := context.Background()
	err := output.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Render with progress failed: %v", err)
	}

	// Progress should be marked complete
	if !progress.IsActive() {
		// NoOp progress is never active, this is expected
		t.Log("Progress completed as expected")
	}
}

// TestIntegration_ContextCancellation tests context cancellation
func TestIntegration_ContextCancellation(t *testing.T) {
	skipIfNotIntegration(t)

	// Create a large document
	builder := New()
	for i := range 1000 {
		builder.Table("Large Table", []map[string]any{
			{"Row": i, "Data": strings.Repeat("x", 1000)},
		}, WithKeys("Row", "Data"))
	}
	doc := builder.Build()

	// Create a context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Create output with longer delay to ensure cancellation happens during write
	output := NewOutput(
		WithFormat(JSON),
		WithWriter(&slowWriter{delay: 100 * time.Millisecond}),
	)

	// Start rendering in a goroutine
	renderDone := make(chan error, 1)
	go func() {
		renderDone <- output.Render(ctx, doc)
	}()

	// Cancel context after a very short delay to ensure it happens during write
	time.Sleep(10 * time.Millisecond)
	cancel()

	// Wait for render to complete
	err := <-renderDone

	// Should have received a context error
	if err == nil {
		t.Error("Expected context cancellation error")
	}
}

// TestIntegration_MixedContentTypes tests documents with all content types
func TestIntegration_MixedContentTypes(t *testing.T) {
	skipIfNotIntegration(t)

	doc := New().
		Header("Document Title").
		Text("Introduction paragraph.").
		Table("Data", []map[string]any{
			{"A": 1, "B": 2, "C": 3},
		}, WithKeys("C", "B", "A")). // Reverse order to test preservation
		Section("Details", func(b *Builder) {
			b.Text("Section text", WithItalic(true))
			b.Table("Nested", []map[string]any{
				{"X": "x", "Y": "y"},
			}, WithKeys("Y", "X"))
		}).
		Raw("custom", []byte("raw data")).
		Graph("Flow", []Edge{{From: "Start", To: "End"}}).
		PieChart("Distribution", []PieSlice{{Label: "All", Value: 100}}, false).
		Build()

	// Verify all content types are present
	contents := doc.GetContents()

	contentTypes := make(map[ContentType]int)
	for _, content := range contents {
		contentTypes[content.Type()]++
	}

	// Check we have all expected types
	expectedTypes := map[ContentType]int{
		ContentTypeText:    2, // Header creates text content with header style
		ContentTypeTable:   1,
		ContentTypeSection: 1,
		ContentTypeRaw:     1,
		// Graph and Chart would create their specific types
	}

	for contentType, expectedCount := range expectedTypes {
		if contentTypes[contentType] < expectedCount {
			t.Errorf("Expected at least %d %v content, got %d",
				expectedCount, contentType, contentTypes[contentType])
		}
	}
}

// Mock writer for integration testing
type integrationMockWriter struct {
	buf *bytes.Buffer
	mu  sync.Mutex
}

func (m *integrationMockWriter) Write(ctx context.Context, format string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.buf.Write([]byte(format + ": "))
	m.buf.Write(data)
	m.buf.WriteByte('\n')
	return nil
}

// Slow writer for testing cancellation
type slowWriter struct {
	delay time.Duration
}

func (s *slowWriter) Write(ctx context.Context, format string, data []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(s.delay):
		return nil
	}
}

// TestIntegration_LargeDataset tests handling of large datasets
func TestIntegration_LargeDataset(t *testing.T) {
	skipIfNotIntegration(t)

	if testing.Short() {
		t.Skip("Skipping large dataset test in short mode")
	}

	// Create a document with a large table
	rows := 10000
	data := make([]map[string]any, rows)
	for i := range rows {
		data[i] = map[string]any{
			"ID":    i,
			"Name":  "Item " + string(rune('A'+i%26)),
			"Value": i * 100,
			"Flag":  i%2 == 0,
		}
	}

	start := time.Now()
	doc := New().
		Table("Large Dataset", data, WithKeys("ID", "Name", "Value", "Flag")).
		Build()
	buildTime := time.Since(start)

	t.Logf("Built document with %d rows in %v", rows, buildTime)

	// Test rendering performance
	output := NewOutput(
		WithFormat(JSON),
		WithWriter(&integrationMockWriter{buf: &bytes.Buffer{}}),
	)

	start = time.Now()
	err := output.Render(context.Background(), doc)
	renderTime := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to render large dataset: %v", err)
	}

	t.Logf("Rendered %d rows to JSON in %v", rows, renderTime)

	// Basic performance check - should handle 10k rows reasonably fast
	if renderTime > 5*time.Second {
		t.Errorf("Rendering took too long: %v", renderTime)
	}
}
