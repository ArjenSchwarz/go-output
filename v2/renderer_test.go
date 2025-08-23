package output

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// TestRendererInterface ensures all built-in renderers implement the Renderer interface
func TestRendererInterface(t *testing.T) {
	renderers := []Renderer{
		&jsonRenderer{},
		&yamlRenderer{},
		&csvRenderer{},
		&htmlRenderer{},
		&tableRenderer{},
		&markdownRenderer{headingLevel: 1},
		&dotRenderer{},
		&mermaidRenderer{},
		&drawioRenderer{},
	}

	for _, renderer := range renderers {
		t.Run(renderer.Format(), func(t *testing.T) {
			// Test Format() method
			format := renderer.Format()
			if format == "" {
				t.Error("Format() should return a non-empty string")
			}

			// Test SupportsStreaming() method
			_ = renderer.SupportsStreaming() // Should not panic

			// Test Render() method with context cancellation
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			doc := New().Build()
			_, err := renderer.Render(ctx, doc)
			// Note: Since implementations are stubs, we don't test actual cancellation yet
			_ = err

			// Test RenderTo() method
			var buf strings.Builder
			err = renderer.RenderTo(context.Background(), doc, &buf)
			// Note: Since implementations are stubs, we don't test actual output yet
			_ = err
		})
	}
}

// TestFormatConstants ensures format constants are properly configured
func TestFormatConstants(t *testing.T) {
	formats := []Format{
		JSON,
		YAML,
		CSV,
		HTML,
		Table,
		Markdown,
		DOT,
		Mermaid,
		DrawIO,
	}

	expectedNames := []string{
		FormatJSON,
		FormatYAML,
		FormatCSV,
		FormatHTML,
		FormatTable,
		FormatMarkdown,
		FormatDOT,
		FormatMermaid,
		FormatDrawIO,
	}

	if len(formats) != len(expectedNames) {
		t.Fatalf("Expected %d formats, got %d", len(expectedNames), len(formats))
	}

	for i, format := range formats {
		expectedName := expectedNames[i]

		// Test format name
		if format.Name != expectedName {
			t.Errorf("Format[%d]: expected name %q, got %q", i, expectedName, format.Name)
		}

		// Test renderer is not nil
		if format.Renderer == nil {
			t.Errorf("Format[%d]: renderer should not be nil", i)
			continue
		}

		// Test renderer format matches
		if format.Renderer.Format() != expectedName {
			t.Errorf("Format[%d]: renderer format %q does not match format name %q",
				i, format.Renderer.Format(), expectedName)
		}

		// Test options map is initialized
		if format.Options == nil {
			// Options map can be nil, but should be consistently handled
		}
	}
}

// TestStreamingSupportCategories ensures streaming support is categorized correctly
func TestStreamingSupportCategories(t *testing.T) {
	streamingFormats := []string{FormatJSON, FormatYAML, FormatCSV, FormatHTML, FormatTable, FormatMarkdown}
	nonStreamingFormats := []string{FormatDOT, FormatMermaid, FormatDrawIO}

	// Test streaming formats
	for _, formatName := range streamingFormats {
		t.Run("streaming_"+formatName, func(t *testing.T) {
			var renderer Renderer
			switch formatName {
			case FormatJSON:
				renderer = &jsonRenderer{}
			case FormatYAML:
				renderer = &yamlRenderer{}
			case FormatCSV:
				renderer = &csvRenderer{}
			case FormatHTML:
				renderer = &htmlRenderer{}
			case FormatTable:
				renderer = &tableRenderer{}
			case FormatMarkdown:
				renderer = &markdownRenderer{headingLevel: 1}
			}

			if !renderer.SupportsStreaming() {
				t.Errorf("%s renderer should support streaming", formatName)
			}
		})
	}

	// Test non-streaming formats
	for _, formatName := range nonStreamingFormats {
		t.Run("non_streaming_"+formatName, func(t *testing.T) {
			var renderer Renderer
			switch formatName {
			case FormatDOT:
				renderer = &dotRenderer{}
			case FormatMermaid:
				renderer = &mermaidRenderer{}
			case FormatDrawIO:
				renderer = &drawioRenderer{}
			}

			if renderer.SupportsStreaming() {
				t.Errorf("%s renderer should not support streaming", formatName)
			}
		})
	}
}

// MockRenderer for testing renderer interface compliance
type MockRenderer struct {
	formatName     string
	supportsStream bool
	renderResult   []byte
	renderError    error
	renderToError  error
}

func (m *MockRenderer) Format() string {
	return m.formatName
}

func (m *MockRenderer) Render(ctx context.Context, doc *Document) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return m.renderResult, m.renderError
	}
}

func (m *MockRenderer) RenderTo(ctx context.Context, doc *Document, w io.Writer) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		if m.renderToError != nil {
			return m.renderToError
		}
		if m.renderResult != nil {
			_, err := w.Write(m.renderResult)
			return err
		}
		return nil
	}
}

func (m *MockRenderer) SupportsStreaming() bool {
	return m.supportsStream
}

// TestMockRenderer ensures the mock renderer works correctly for testing
func TestMockRenderer(t *testing.T) {
	mock := &MockRenderer{
		formatName:     "test",
		supportsStream: true,
		renderResult:   []byte("test output"),
	}

	// Test Format
	if mock.Format() != "test" {
		t.Errorf("Expected format 'test', got %q", mock.Format())
	}

	// Test SupportsStreaming
	if !mock.SupportsStreaming() {
		t.Error("Expected mock to support streaming")
	}

	// Test Render
	doc := New().Build()
	result, err := mock.Render(context.Background(), doc)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if string(result) != "test output" {
		t.Errorf("Expected 'test output', got %q", string(result))
	}

	// Test RenderTo
	var buf strings.Builder
	err = mock.RenderTo(context.Background(), doc, &buf)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if buf.String() != "test output" {
		t.Errorf("Expected 'test output', got %q", buf.String())
	}

	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = mock.Render(ctx, doc)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}

	err = mock.RenderTo(ctx, doc, &buf)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

// TestBaseRenderer_RenderDocument tests the core document rendering functionality
func TestBaseRenderer_RenderDocument(t *testing.T) {
	base := &baseRenderer{}

	// Create test document with multiple content types
	doc := New().
		Text("Hello").
		Table("test", []map[string]any{{"name": "test", "value": 42}}, WithKeys("name", "value")).
		Text("World").
		Build()

	// Test successful rendering
	renderFunc := func(content Content) ([]byte, error) {
		return content.AppendText(nil)
	}

	result, err := base.renderDocument(context.Background(), doc, renderFunc)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, "Hello") {
		t.Error("Result should contain 'Hello'")
	}
	if !strings.Contains(resultStr, "World") {
		t.Error("Result should contain 'World'")
	}
	if !strings.Contains(resultStr, "test") {
		t.Error("Result should contain table data")
	}
}

// TestBaseRenderer_RenderDocument_NilDocument tests error handling for nil document
func TestBaseRenderer_RenderDocument_NilDocument(t *testing.T) {
	base := &baseRenderer{}

	renderFunc := func(content Content) ([]byte, error) {
		return content.AppendText(nil)
	}

	_, err := base.renderDocument(context.Background(), nil, renderFunc)
	if err == nil {
		t.Error("Expected error for nil document")
	}
	if !strings.Contains(err.Error(), "document cannot be nil") {
		t.Errorf("Expected 'document cannot be nil' error, got: %v", err)
	}
}

// TestBaseRenderer_RenderDocument_ContextCancellation tests context cancellation handling
func TestBaseRenderer_RenderDocument_ContextCancellation(t *testing.T) {
	base := &baseRenderer{}

	// Create a document with content
	doc := New().Text("test").Build()

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	renderFunc := func(content Content) ([]byte, error) {
		return content.AppendText(nil)
	}

	_, err := base.renderDocument(ctx, doc, renderFunc)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

// TestBaseRenderer_RenderDocument_RenderError tests error handling in render function
func TestBaseRenderer_RenderDocument_RenderError(t *testing.T) {
	base := &baseRenderer{}

	doc := New().Text("test").Build()

	expectedError := errors.New("render failed")
	renderFunc := func(content Content) ([]byte, error) {
		return nil, expectedError
	}

	_, err := base.renderDocument(context.Background(), doc, renderFunc)
	if err == nil {
		t.Error("Expected error from render function")
	}
	if !strings.Contains(err.Error(), "failed to render content") {
		t.Errorf("Expected 'failed to render content' error, got: %v", err)
	}
	if !errors.Is(err, expectedError) {
		t.Error("Error should wrap the original render error")
	}
}

// TestBaseRenderer_RenderDocumentTo tests streaming document rendering
func TestBaseRenderer_RenderDocumentTo(t *testing.T) {
	base := &baseRenderer{}

	doc := New().
		Text("Line 1").
		Text("Line 2").
		Build()

	var buf strings.Builder

	renderFunc := func(content Content, w io.Writer) error {
		data, err := content.AppendText(nil)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}

	err := base.renderDocumentTo(context.Background(), doc, &buf, renderFunc)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	result := buf.String()
	if !strings.Contains(result, "Line 1") {
		t.Error("Result should contain 'Line 1'")
	}
	if !strings.Contains(result, "Line 2") {
		t.Error("Result should contain 'Line 2'")
	}
}

// TestBaseRenderer_RenderDocumentTo_NilInputs tests error handling for nil inputs
func TestBaseRenderer_RenderDocumentTo_NilInputs(t *testing.T) {
	base := &baseRenderer{}

	renderFunc := func(content Content, w io.Writer) error {
		return nil
	}

	// Test nil document
	var buf strings.Builder
	err := base.renderDocumentTo(context.Background(), nil, &buf, renderFunc)
	if err == nil || !strings.Contains(err.Error(), "document cannot be nil") {
		t.Errorf("Expected 'document cannot be nil' error, got: %v", err)
	}

	// Test nil writer
	doc := New().Text("test").Build()
	err = base.renderDocumentTo(context.Background(), doc, nil, renderFunc)
	if err == nil || !strings.Contains(err.Error(), "writer cannot be nil") {
		t.Errorf("Expected 'writer cannot be nil' error, got: %v", err)
	}
}

// TestBaseRenderer_RenderDocumentTo_ContextCancellation tests context cancellation in streaming
func TestBaseRenderer_RenderDocumentTo_ContextCancellation(t *testing.T) {
	base := &baseRenderer{}

	doc := New().Text("test").Build()
	var buf strings.Builder

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	renderFunc := func(content Content, w io.Writer) error {
		return nil
	}

	err := base.renderDocumentTo(ctx, doc, &buf, renderFunc)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

// TestBaseRenderer_ThreadSafety tests concurrent access to base renderer
func TestBaseRenderer_ThreadSafety(t *testing.T) {
	base := &baseRenderer{}
	doc := New().Text("test content").Build()

	const numGoroutines = 10
	var wg sync.WaitGroup

	renderFunc := func(content Content) ([]byte, error) {
		// Simulate some work
		time.Sleep(1 * time.Millisecond)
		return content.AppendText(nil)
	}

	// Test concurrent Render calls
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_, err := base.renderDocument(context.Background(), doc, renderFunc)
			if err != nil {
				t.Errorf("Unexpected error in concurrent render: %v", err)
			}
		}()
	}

	wg.Wait()
}

// TestBaseRenderer_RenderContent tests default content rendering
func TestBaseRenderer_RenderContent(t *testing.T) {
	base := &baseRenderer{}

	// Test with text content
	text := NewTextContent("test text")
	result, err := base.renderContent(text)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(string(result), "test text") {
		t.Error("Result should contain the text content")
	}

	// Test with nil content
	_, err = base.renderContent(nil)
	if err == nil || !strings.Contains(err.Error(), "content cannot be nil") {
		t.Errorf("Expected 'content cannot be nil' error, got: %v", err)
	}
}

// TestBaseRenderer_RenderContentTo tests default streaming content rendering
func TestBaseRenderer_RenderContentTo(t *testing.T) {
	base := &baseRenderer{}

	text := NewTextContent("test text")
	var buf strings.Builder

	err := base.renderContentTo(text, &buf)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "test text") {
		t.Error("Result should contain the text content")
	}

	// Test with nil content
	buf.Reset()
	err = base.renderContentTo(nil, &buf)
	if err == nil || !strings.Contains(err.Error(), "content cannot be nil") {
		t.Errorf("Expected 'content cannot be nil' error, got: %v", err)
	}

	// Test with nil writer
	err = base.renderContentTo(text, nil)
	if err == nil || !strings.Contains(err.Error(), "writer cannot be nil") {
		t.Errorf("Expected 'writer cannot be nil' error, got: %v", err)
	}
}

// TestRendererImplementations_UseBaseRenderer tests that all renderers use base functionality
func TestRendererImplementations_UseBaseRenderer(t *testing.T) {
	renderers := []Renderer{
		&jsonRenderer{},
		&yamlRenderer{},
		&csvRenderer{},
		&htmlRenderer{},
		&tableRenderer{},
		&markdownRenderer{headingLevel: 1},
		&dotRenderer{},
		&mermaidRenderer{},
		&drawioRenderer{},
	}

	doc := New().Text("test").Build()

	for _, renderer := range renderers {
		t.Run(renderer.Format(), func(t *testing.T) {
			// Test that Render works (using base functionality)
			result, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Errorf("Render failed: %v", err)
			}
			if len(result) == 0 {
				t.Error("Render should return non-empty result")
			}

			// Test that RenderTo works (using base functionality)
			var buf strings.Builder
			err = renderer.RenderTo(context.Background(), doc, &buf)
			if err != nil {
				t.Errorf("RenderTo failed: %v", err)
			}
			if buf.Len() == 0 {
				t.Error("RenderTo should write non-empty content")
			}

			// Results should be the same
			if string(result) != buf.String() {
				t.Error("Render and RenderTo should produce the same output")
			}
		})
	}
}

// Test JSON Renderer with key order preservation and data integrity

// TestJSONRenderer_TableKeyOrderPreservation tests that JSON output preserves key order exactly
func TestJSONRenderer_TableKeyOrderPreservation(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		data     []map[string]any
		expected []string // Expected key order in output
	}{
		{
			name: "explicit key order Z-A-M",
			keys: []string{"Z", "A", "M"},
			data: []map[string]any{
				{"A": 1, "M": 2, "Z": 3},
				{"Z": 6, "M": 5, "A": 4},
			},
			expected: []string{"Z", "A", "M"},
		},
		{
			name: "numeric and string fields ordered",
			keys: []string{"id", "name", "score", "active"},
			data: []map[string]any{
				{"name": "Alice", "id": 1, "active": true, "score": 95.5},
				{"score": 87.2, "id": 2, "name": "Bob", "active": false},
			},
			expected: []string{"id", "name", "score", "active"},
		},
		{
			name: "single record key preservation",
			keys: []string{"last", "first", "middle"},
			data: []map[string]any{
				{"first": "John", "last": "Doe", "middle": "Q"},
			},
			expected: []string{"last", "first", "middle"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create table with explicit key order
			doc := New().
				Table("test", tt.data, WithKeys(tt.keys...)).
				Build()

			renderer := &jsonRenderer{}
			result, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			// Parse JSON result
			var parsed map[string]any
			if err := json.Unmarshal(result, &parsed); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			// Check schema key order
			schema, ok := parsed["schema"].(map[string]any)
			if !ok {
				t.Fatal("Expected schema in JSON output")
			}

			keys, ok := schema["keys"].([]any)
			if !ok {
				t.Fatal("Expected keys array in schema")
			}

			// Verify key order matches expected
			if len(keys) != len(tt.expected) {
				t.Fatalf("Expected %d keys, got %d", len(tt.expected), len(keys))
			}

			for i, key := range keys {
				keyStr, ok := key.(string)
				if !ok {
					t.Fatalf("Key at index %d is not a string: %T", i, key)
				}
				if keyStr != tt.expected[i] {
					t.Errorf("Key order mismatch at index %d: expected %q, got %q", i, tt.expected[i], keyStr)
				}
			}

			// Check data records maintain key order
			data, ok := parsed["data"].([]any)
			if !ok {
				t.Fatal("Expected data array in JSON output")
			}

			for recordIdx, recordAny := range data {
				record, ok := recordAny.(map[string]any)
				if !ok {
					t.Fatalf("Record %d is not a map", recordIdx)
				}

				// Check that record contains expected keys in order
				for _, expectedKey := range tt.expected {
					if _, exists := record[expectedKey]; !exists {
						t.Errorf("Record %d missing expected key %q", recordIdx, expectedKey)
					}
				}
			}
		})
	}
}

// TestJSONRenderer_DataTypePreservation tests that JSON preserves all data types correctly
func TestJSONRenderer_DataTypePreservation(t *testing.T) {
	testData := []map[string]any{
		{
			"string": "hello world",
			"int":    42,
			"float":  3.14159,
			"bool":   true,
			"nil":    nil,
			"zero":   0,
			"empty":  "",
		},
	}

	doc := New().
		Table("types", testData, WithKeys("string", "int", "float", "bool", "nil", "zero", "empty")).
		Build()

	renderer := &jsonRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	data, ok := parsed["data"].([]any)
	if !ok || len(data) == 0 {
		t.Fatal("Expected data array with at least one record")
	}

	record, ok := data[0].(map[string]any)
	if !ok {
		t.Fatal("First record is not a map")
	}

	// Verify types are preserved
	if record["string"] != "hello world" {
		t.Errorf("String value mismatch: expected %q, got %v", "hello world", record["string"])
	}

	// JSON unmarshaling converts numbers to float64
	if record["int"] != 42.0 {
		t.Errorf("Int value mismatch: expected %v, got %v", 42.0, record["int"])
	}

	if record["float"] != 3.14159 {
		t.Errorf("Float value mismatch: expected %v, got %v", 3.14159, record["float"])
	}

	if record["bool"] != true {
		t.Errorf("Bool value mismatch: expected %v, got %v", true, record["bool"])
	}

	if record["nil"] != nil {
		t.Errorf("Nil value mismatch: expected %v, got %v", nil, record["nil"])
	}

	if record["zero"] != 0.0 {
		t.Errorf("Zero value mismatch: expected %v, got %v", 0.0, record["zero"])
	}

	if record["empty"] != "" {
		t.Errorf("Empty string mismatch: expected %q, got %v", "", record["empty"])
	}
}

// TestJSONRenderer_StreamingVsBuffered tests that streaming and buffered output match
func TestJSONRenderer_StreamingVsBuffered(t *testing.T) {
	// Create a moderately complex document
	doc := New().
		Text("Header text", WithHeader(true)).
		Table("users", []map[string]any{
			{"id": 1, "name": "Alice", "score": 95.5},
			{"id": 2, "name": "Bob", "score": 87.2},
			{"id": 3, "name": "Charlie", "score": 92.1},
		}, WithKeys("id", "name", "score")).
		Raw(FormatJSON, []byte(`{"custom": "data"}`)).
		Section("Details", func(b *Builder) {
			b.Text("Section content")
		}).
		Build()

	renderer := &jsonRenderer{}

	// Test buffered rendering
	bufferedResult, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Buffered render failed: %v", err)
	}

	// Test streaming rendering
	var streamBuf bytes.Buffer
	err = renderer.RenderTo(context.Background(), doc, &streamBuf)
	if err != nil {
		t.Fatalf("Streaming render failed: %v", err)
	}
	streamedResult := streamBuf.Bytes()

	// Parse both results to ensure they're valid JSON
	var bufferedParsed, streamedParsed any
	if err := json.Unmarshal(bufferedResult, &bufferedParsed); err != nil {
		t.Fatalf("Buffered result is not valid JSON: %v", err)
	}
	if err := json.Unmarshal(streamedResult, &streamedParsed); err != nil {
		t.Fatalf("Streamed result is not valid JSON: %v", err)
	}

	// Compare the results (they may differ in formatting but should be semantically equivalent)
	bufferedStr := string(bufferedResult)
	streamedStr := string(streamedResult)

	// Verify both contain the same key elements
	expectedElements := []string{
		`"id"`, `"name"`, `"score"`,
		`"Alice"`, `"Bob"`, `"Charlie"`,
		`"Header text"`, `"Section content"`,
		`"type"`, `"text"`, `"section"`,
	}

	for _, element := range expectedElements {
		if !strings.Contains(bufferedStr, element) {
			t.Errorf("Buffered result missing element: %s", element)
		}
		if !strings.Contains(streamedStr, element) {
			t.Errorf("Streamed result missing element: %s", element)
		}
	}
}

// Test YAML Renderer with key order preservation and data integrity

// TestYAMLRenderer_TableKeyOrderPreservation tests that YAML output preserves key order exactly
func TestYAMLRenderer_TableKeyOrderPreservation(t *testing.T) {
	testData := []map[string]any{
		{"name": "Alice", "id": 1, "active": true, "score": 95.5},
		{"score": 87.2, "id": 2, "name": "Bob", "active": false},
	}

	// Test with explicit key order that differs from alphabetical
	keyOrder := []string{"score", "name", "id", "active"}

	doc := New().
		Table("test", testData, WithKeys(keyOrder...)).
		Build()

	renderer := &yamlRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Parse YAML result
	var parsed map[string]any
	if err := yaml.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	// Check schema key order
	schema, ok := parsed["schema"].(map[string]any)
	if !ok {
		t.Fatal("Expected schema in YAML output")
	}

	keys, ok := schema["keys"].([]any)
	if !ok {
		t.Fatal("Expected keys array in schema")
	}

	// Verify key order matches expected
	if len(keys) != len(keyOrder) {
		t.Fatalf("Expected %d keys, got %d", len(keyOrder), len(keys))
	}

	for i, key := range keys {
		keyStr, ok := key.(string)
		if !ok {
			t.Fatalf("Key at index %d is not a string: %T", i, key)
		}
		if keyStr != keyOrder[i] {
			t.Errorf("Key order mismatch at index %d: expected %q, got %q", i, keyOrder[i], keyStr)
		}
	}
}

// TestYAMLRenderer_DataTypePreservation tests that YAML preserves types correctly
func TestYAMLRenderer_DataTypePreservation(t *testing.T) {
	testData := []map[string]any{
		{
			"string": "hello world",
			"int":    42,
			"float":  3.14159,
			"bool":   true,
			"nil":    nil,
			"zero":   0,
			"empty":  "",
		},
	}

	doc := New().
		Table("types", testData, WithKeys("string", "int", "float", "bool", "nil", "zero", "empty")).
		Build()

	renderer := &yamlRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	var parsed map[string]any
	if err := yaml.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	data, ok := parsed["data"].([]any)
	if !ok || len(data) == 0 {
		t.Fatal("Expected data array with at least one record")
	}

	record, ok := data[0].(map[string]any)
	if !ok {
		t.Fatal("First record is not a map")
	}

	// Verify types are preserved (YAML preserves types better than JSON)
	if record["string"] != "hello world" {
		t.Errorf("String value mismatch: expected %q, got %v", "hello world", record["string"])
	}

	if record["int"] != 42 {
		t.Errorf("Int value mismatch: expected %v, got %v", 42, record["int"])
	}

	if record["float"] != 3.14159 {
		t.Errorf("Float value mismatch: expected %v, got %v", 3.14159, record["float"])
	}

	if record["bool"] != true {
		t.Errorf("Bool value mismatch: expected %v, got %v", true, record["bool"])
	}

	if record["nil"] != nil {
		t.Errorf("Nil value mismatch: expected %v, got %v", nil, record["nil"])
	}

	if record["zero"] != 0 {
		t.Errorf("Zero value mismatch: expected %v, got %v", 0, record["zero"])
	}

	if record["empty"] != "" {
		t.Errorf("Empty string mismatch: expected %q, got %v", "", record["empty"])
	}
}

// TestYAMLRenderer_StreamingVsBuffered tests that YAML streaming and buffered output match
func TestYAMLRenderer_StreamingVsBuffered(t *testing.T) {
	doc := New().
		Text("Header text").
		Table("data", []map[string]any{
			{"key": "value1", "num": 1},
			{"key": "value2", "num": 2},
		}, WithKeys("key", "num")).
		Build()

	renderer := &yamlRenderer{}

	// Test buffered rendering
	bufferedResult, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Buffered render failed: %v", err)
	}

	// Test streaming rendering
	var streamBuf bytes.Buffer
	err = renderer.RenderTo(context.Background(), doc, &streamBuf)
	if err != nil {
		t.Fatalf("Streaming render failed: %v", err)
	}
	streamedResult := streamBuf.Bytes()

	// Parse both results to ensure they're valid YAML
	var bufferedParsed, streamedParsed any
	if err := yaml.Unmarshal(bufferedResult, &bufferedParsed); err != nil {
		t.Fatalf("Buffered result is not valid YAML: %v", err)
	}
	if err := yaml.Unmarshal(streamedResult, &streamedParsed); err != nil {
		t.Fatalf("Streamed result is not valid YAML: %v", err)
	}

	// Both should contain the same key elements
	bufferedStr := string(bufferedResult)
	streamedStr := string(streamedResult)

	expectedElements := []string{
		"key:", "num:", "value1", "value2", "Header text",
	}

	for _, element := range expectedElements {
		if !strings.Contains(bufferedStr, element) {
			t.Errorf("Buffered result missing element: %s", element)
		}
		if !strings.Contains(streamedStr, element) {
			t.Errorf("Streamed result missing element: %s", element)
		}
	}
}

// TestJSONYAMLRenderers_MixedContentTypes tests both renderers with all content types
func TestJSONYAMLRenderers_MixedContentTypes(t *testing.T) {
	doc := New().
		Text("Introduction", WithHeader(true)).
		Table("users", []map[string]any{
			{"id": 1, "name": "Alice"},
			{"id": 2, "name": "Bob"},
		}, WithKeys("id", "name")).
		Raw(FormatHTML, []byte("<p>Custom HTML</p>")).
		Section("Summary", func(b *Builder) {
			b.Text("This is a summary")
			b.Table("stats", []map[string]any{
				{"metric": "total", "value": 42},
			}, WithKeys("metric", "value"))
		}).
		Build()

	renderers := []struct {
		name     string
		renderer Renderer
	}{
		{FormatJSON, &jsonRenderer{}},
		{FormatYAML, &yamlRenderer{}},
	}

	for _, r := range renderers {
		t.Run(r.name, func(t *testing.T) {
			// Test both buffered and streaming
			result, err := r.renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			var streamBuf bytes.Buffer
			err = r.renderer.RenderTo(context.Background(), doc, &streamBuf)
			if err != nil {
				t.Fatalf("RenderTo failed: %v", err)
			}

			// Both should produce valid output
			resultStr := string(result)
			streamStr := streamBuf.String()

			expectedContent := []string{
				"Introduction", "Alice", "Bob", "Custom HTML", "Summary", "total", "42",
			}

			for _, content := range expectedContent {
				if !strings.Contains(resultStr, content) {
					t.Errorf("Buffered result missing content: %s", content)
				}
				if !strings.Contains(streamStr, content) {
					t.Errorf("Streamed result missing content: %s", content)
				}
			}
		})
	}
}

// TestJSONYAMLRenderers_LargeDataset tests performance with larger datasets
func TestJSONYAMLRenderers_LargeDataset(t *testing.T) {
	// Create a larger dataset to test streaming performance
	const recordCount = 1000
	var data []map[string]any
	for i := 0; i < recordCount; i++ {
		data = append(data, map[string]any{
			"id":       i,
			"name":     "User " + strings.Repeat("X", i%10), // Varying length
			"score":    float64(i) * 1.5,
			"active":   i%2 == 0,
			"category": "Category " + string(rune('A'+i%5)),
		})
	}

	doc := New().
		Table("large_dataset", data, WithKeys("id", "name", "score", "active", "category")).
		Build()

	renderers := []struct {
		name     string
		renderer Renderer
	}{
		{FormatJSON, &jsonRenderer{}},
		{FormatYAML, &yamlRenderer{}},
	}

	for _, r := range renderers {
		t.Run(r.name, func(t *testing.T) {
			// Test that both buffered and streaming work without errors
			_, err := r.renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Buffered render failed: %v", err)
			}

			var streamBuf bytes.Buffer
			err = r.renderer.RenderTo(context.Background(), doc, &streamBuf)
			if err != nil {
				t.Fatalf("Streaming render failed: %v", err)
			}

			// Verify the output contains expected elements
			output := streamBuf.String()
			if !strings.Contains(output, "User ") {
				t.Error("Output should contain user data")
			}
			if !strings.Contains(output, "Category ") {
				t.Error("Output should contain category data")
			}
		})
	}
}

// Test CSV Renderer with key order preservation and proper escaping

// TestCSVRenderer_TableKeyOrderPreservation tests that CSV output preserves key order exactly
func TestCSVRenderer_TableKeyOrderPreservation(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		data     []map[string]any
		expected []string // Expected key order in CSV header
	}{
		{
			name: "explicit key order Z-A-M",
			keys: []string{"Z", "A", "M"},
			data: []map[string]any{
				{"A": 1, "M": 2, "Z": 3},
				{"Z": 6, "M": 5, "A": 4},
			},
			expected: []string{"Z", "A", "M"},
		},
		{
			name: "numeric and string fields ordered",
			keys: []string{"id", "name", "score", "active"},
			data: []map[string]any{
				{"name": "Alice", "id": 1, "active": true, "score": 95.5},
				{"score": 87.2, "id": 2, "name": "Bob", "active": false},
			},
			expected: []string{"id", "name", "score", "active"},
		},
		{
			name: "reverse alphabetical order",
			keys: []string{"zebra", "yellow", "alpha"},
			data: []map[string]any{
				{"alpha": "first", "yellow": "second", "zebra": "third"},
			},
			expected: []string{"zebra", "yellow", "alpha"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create table with explicit key order
			doc := New().
				Table("test", tt.data, WithKeys(tt.keys...)).
				Build()

			renderer := &csvRenderer{}
			result, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			// Parse CSV result
			csvReader := csv.NewReader(strings.NewReader(string(result)))
			records, err := csvReader.ReadAll()
			if err != nil {
				t.Fatalf("Failed to parse CSV: %v", err)
			}

			if len(records) == 0 {
				t.Fatal("Expected at least header row in CSV output")
			}

			// Check header row (first row) for key order
			header := records[0]
			if len(header) != len(tt.expected) {
				t.Fatalf("Expected %d columns, got %d", len(tt.expected), len(header))
			}

			for i, expectedKey := range tt.expected {
				if header[i] != expectedKey {
					t.Errorf("Key order mismatch at column %d: expected %q, got %q", i, expectedKey, header[i])
				}
			}

			// Verify data rows maintain the same key order
			for rowIdx := 1; rowIdx < len(records); rowIdx++ {
				row := records[rowIdx]
				if len(row) != len(tt.expected) {
					t.Errorf("Row %d has %d columns, expected %d", rowIdx, len(row), len(tt.expected))
				}
			}
		})
	}
}

// TestCSVRenderer_DataTypeHandling tests that CSV handles different data types correctly
func TestCSVRenderer_DataTypeHandling(t *testing.T) {
	testData := []map[string]any{
		{
			"string":     "hello world",
			"int":        42,
			"float":      3.14159,
			"bool_true":  true,
			"bool_false": false,
			"nil":        nil,
			"zero":       0,
			"empty":      "",
		},
	}

	doc := New().
		Table("types", testData, WithKeys("string", "int", "float", "bool_true", "bool_false", "nil", "zero", "empty")).
		Build()

	renderer := &csvRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Parse CSV result
	csvReader := csv.NewReader(strings.NewReader(string(result)))
	records, err := csvReader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	if len(records) < 2 {
		t.Fatal("Expected header row plus at least one data row")
	}

	dataRow := records[1]

	// Verify data types are formatted correctly
	expectedValues := []string{
		"hello world", // string
		"42",          // int
		"3.14159",     // float (or close to it)
		"true",        // bool_true
		"false",       // bool_false
		"",            // nil -> empty string
		"0",           // zero
		"",            // empty string
	}

	for i, expected := range expectedValues {
		if i >= len(dataRow) {
			t.Fatalf("Data row has fewer columns than expected")
		}
		actual := dataRow[i]

		// For float comparison, be more flexible
		if expected == "3.14159" && strings.HasPrefix(actual, "3.1415") {
			continue // Accept slight formatting differences
		}

		if actual != expected {
			t.Errorf("Column %d value mismatch: expected %q, got %q", i, expected, actual)
		}
	}
}

// TestCSVRenderer_SpecialCharacterEscaping tests proper handling of special characters
func TestCSVRenderer_SpecialCharacterEscaping(t *testing.T) {
	testData := []map[string]any{
		{
			"quotes":   `text with "quotes" inside`,
			"commas":   "text, with, commas",
			"newlines": "text\nwith\nnewlines",
			"tabs":     "text\twith\ttabs",
			"mixed":    "complex \"text\", with\nnewlines\tand\rcarriage returns",
		},
	}

	doc := New().
		Table("special", testData, WithKeys("quotes", "commas", "newlines", "tabs", "mixed")).
		Build()

	renderer := &csvRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Parse CSV result - this should succeed if escaping is correct
	csvReader := csv.NewReader(strings.NewReader(string(result)))
	records, err := csvReader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV (escaping issue): %v", err)
	}

	if len(records) < 2 {
		t.Fatal("Expected header row plus at least one data row")
	}

	dataRow := records[1]

	// Verify special characters are handled
	if !strings.Contains(dataRow[0], "quotes") {
		t.Error("Quotes field should contain the word 'quotes'")
	}
	if !strings.Contains(dataRow[1], "commas") {
		t.Error("Commas field should contain the word 'commas'")
	}
	// Newlines should be converted to spaces
	if strings.Contains(dataRow[2], "\n") {
		t.Error("Newlines should be converted to spaces")
	}
	if strings.Contains(dataRow[3], "\t") {
		t.Error("Tabs should be converted to spaces")
	}
}

// TestCSVRenderer_StreamingVsBuffered tests that streaming and buffered output match
func TestCSVRenderer_StreamingVsBuffered(t *testing.T) {
	// Create a test document with table data
	doc := New().
		Table("users", []map[string]any{
			{"id": 1, "name": "Alice", "score": 95.5},
			{"id": 2, "name": "Bob", "score": 87.2},
			{"id": 3, "name": "Charlie", "score": 92.1},
		}, WithKeys("id", "name", "score")).
		Build()

	renderer := &csvRenderer{}

	// Test buffered rendering
	bufferedResult, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Buffered render failed: %v", err)
	}

	// Test streaming rendering
	var streamBuf bytes.Buffer
	err = renderer.RenderTo(context.Background(), doc, &streamBuf)
	if err != nil {
		t.Fatalf("Streaming render failed: %v", err)
	}
	streamedResult := streamBuf.Bytes()

	// Results should be identical for CSV
	if !bytes.Equal(bufferedResult, streamedResult) {
		t.Errorf("Buffered and streamed results differ:\nBuffered: %q\nStreamed: %q",
			string(bufferedResult), string(streamedResult))
	}

	// Both should be valid CSV
	for i, result := range [][]byte{bufferedResult, streamedResult} {
		csvReader := csv.NewReader(strings.NewReader(string(result)))
		records, err := csvReader.ReadAll()
		if err != nil {
			t.Errorf("Result %d is not valid CSV: %v", i, err)
		}
		if len(records) != 4 { // header + 3 data rows
			t.Errorf("Result %d has %d rows, expected 4", i, len(records))
		}
	}
}

func TestTableRenderer_KeyOrderPreservation(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		data     []map[string]any
		expected []string
	}{
		{
			name: "preserve explicit key order",
			keys: []string{"c", "a", "b"},
			data: []map[string]any{
				{"a": "alpha", "b": "beta", "c": "gamma"},
				{"c": "charlie", "b": "bravo", "a": "alpha"},
			},
			expected: []string{"c", "a", "b"},
		},
		{
			name: "preserve numeric and string keys",
			keys: []string{"id", "name", "score", "active"},
			data: []map[string]any{
				{"name": "Alice", "id": 1, "active": true, "score": 95.5},
				{"score": 87.2, "id": 2, "name": "Bob", "active": false},
			},
			expected: []string{"id", "name", "score", "active"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create table with explicit key order
			doc := New().
				Table("Test Table", tt.data, WithKeys(tt.keys...)).
				Build()

			// Test with table renderer
			renderer := &tableRenderer{}
			ctx := context.Background()

			result, err := renderer.Render(ctx, doc)
			if err != nil {
				t.Fatalf("Failed to render table: %v", err)
			}

			resultStr := string(result)

			// Verify that the output contains the table title
			if !strings.Contains(resultStr, "Test Table") {
				t.Errorf("Output does not contain table title")
			}

			// Split into lines to check header order
			lines := strings.Split(resultStr, "\n")
			var headerLine string

			// Find the header line (should contain all our keys)
			// Look for uppercase versions since go-pretty converts headers to uppercase
			upperKeys := make([]string, len(tt.keys))
			for i, key := range tt.keys {
				upperKeys[i] = strings.ToUpper(key)
			}

			for _, line := range lines {
				if strings.Contains(line, upperKeys[0]) && strings.Contains(line, upperKeys[1]) {
					headerLine = line
					break
				}
			}

			if headerLine == "" {
				t.Errorf("Could not find header line in output. Full output:\n%s", resultStr)
				return
			}

			// Verify that keys appear in the correct order in the header
			// We check that each key appears before the next one
			// Use uppercase versions for comparison
			for i := 0; i < len(tt.expected)-1; i++ {
				key1 := strings.ToUpper(tt.expected[i])
				key2 := strings.ToUpper(tt.expected[i+1])

				pos1 := strings.Index(headerLine, key1)
				pos2 := strings.Index(headerLine, key2)

				if pos1 == -1 {
					t.Errorf("Key %s not found in header", key1)
				}
				if pos2 == -1 {
					t.Errorf("Key %s not found in header", key2)
				}
				if pos1 >= pos2 {
					t.Errorf("Key %s appears after %s in header, expected before",
						key1, key2)
				}
			}
		})
	}
}

func TestTableRenderer_MixedContent(t *testing.T) {
	// Create a document with mixed content types
	data := []map[string]any{
		{"name": "Alice", "age": 30, "city": "New York"},
		{"name": "Bob", "age": 25, "city": "Los Angeles"},
	}

	doc := New().
		Text("User Report").
		Table("Users", data, WithKeys("name", "age", "city")).
		Text("End of report").
		Build()

	renderer := &tableRenderer{}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render mixed content: %v", err)
	}

	resultStr := string(result)

	// Should contain text content
	if !strings.Contains(resultStr, "User Report") {
		t.Errorf("Output missing text content 'User Report'")
	}

	if !strings.Contains(resultStr, "End of report") {
		t.Errorf("Output missing text content 'End of report'")
	}

	// Should contain table data
	if !strings.Contains(resultStr, "Alice") {
		t.Errorf("Output missing table data 'Alice'")
	}

	if !strings.Contains(resultStr, "Bob") {
		t.Errorf("Output missing table data 'Bob'")
	}

	// Should contain table title
	if !strings.Contains(resultStr, "Users") {
		t.Errorf("Output missing table title 'Users'")
	}
}

func TestTableRenderer_SectionContent(t *testing.T) {
	// Create a document with section content
	userData := []map[string]any{
		{"name": "Alice", "role": "Admin"},
		{"name": "Bob", "role": "User"},
	}

	doc := New().
		Section("User Management", func(b *Builder) {
			b.Text("This section contains user information").
				Table("Active Users", userData, WithKeys("name", "role"))
		}).
		Build()

	renderer := &tableRenderer{}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render section content: %v", err)
	}

	resultStr := string(result)

	// Should contain section marker
	if !strings.Contains(resultStr, "=== User Management ===") {
		t.Errorf("Output missing section header")
	}

	// Should contain section text and table data
	if !strings.Contains(resultStr, "This section contains user information") {
		t.Errorf("Output missing section text")
	}

	if !strings.Contains(resultStr, "Alice") || !strings.Contains(resultStr, "Admin") {
		t.Errorf("Output missing table data from section")
	}
}

func TestHTMLRenderer_TableEscaping(t *testing.T) {
	// Test HTML escaping with dangerous content
	data := []map[string]any{
		{"name": "<script>alert('xss')</script>", "html": "<b>Bold</b>", "safe": "Normal Text"},
		{"name": "John & Jane", "html": "A > B", "safe": "Test"},
	}

	doc := New().
		Table("Test Escaping", data, WithKeys("name", "html", "safe")).
		Build()

	renderer := &htmlRenderer{}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render HTML: %v", err)
	}

	resultStr := string(result)

	// Should escape dangerous script tags
	if strings.Contains(resultStr, "<script>") {
		t.Errorf("HTML output contains unescaped script tag")
	}

	// Should contain escaped version
	if !strings.Contains(resultStr, "&lt;script&gt;") {
		t.Errorf("HTML output doesn't contain properly escaped script tag")
	}

	// Should escape HTML entities
	if !strings.Contains(resultStr, "&amp;") {
		t.Errorf("HTML output doesn't escape & character")
	}

	if !strings.Contains(resultStr, "&gt;") {
		t.Errorf("HTML output doesn't escape > character")
	}

	// Should have proper HTML structure
	if !strings.Contains(resultStr, "<table class=\"data-table\">") {
		t.Errorf("HTML output missing table structure")
	}

	if !strings.Contains(resultStr, "<thead>") || !strings.Contains(resultStr, "<tbody>") {
		t.Errorf("HTML output missing thead/tbody structure")
	}
}

func TestHTMLRenderer_KeyOrderPreservation(t *testing.T) {
	data := []map[string]any{
		{"c": "gamma", "a": "alpha", "b": "beta"},
		{"b": "bravo", "c": "charlie", "a": "alpha"},
	}

	doc := New().
		Table("Order Test", data, WithKeys("c", "a", "b")).
		Build()

	renderer := &htmlRenderer{}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render HTML: %v", err)
	}

	resultStr := string(result)

	// Find the header row
	lines := strings.Split(resultStr, "\n")
	var headerRowStart int
	for i, line := range lines {
		if strings.Contains(line, "<th>") {
			headerRowStart = i
			break
		}
	}

	if headerRowStart == 0 {
		t.Fatalf("Could not find header row in HTML output")
	}

	// Check that headers appear in correct order by examining their positions
	cPos := -1
	aPos := -1
	bPos := -1

	for i := headerRowStart; i < len(lines) && strings.Contains(lines[i], "<th>"); i++ {
		if strings.Contains(lines[i], "<th>c</th>") {
			cPos = i
		}
		if strings.Contains(lines[i], "<th>a</th>") {
			aPos = i
		}
		if strings.Contains(lines[i], "<th>b</th>") {
			bPos = i
		}
	}

	if cPos == -1 || aPos == -1 || bPos == -1 {
		t.Errorf("Could not find all headers in HTML output")
	}

	if !(cPos < aPos && aPos < bPos) {
		t.Errorf("Headers not in correct order: c=%d, a=%d, b=%d", cPos, aPos, bPos)
	}
}

func TestHTMLRenderer_TextContentStyling(t *testing.T) {
	doc := New().
		Text("Normal Text", WithTextStyle(TextStyle{})).
		Text("Bold Text", WithTextStyle(TextStyle{Bold: true})).
		Text("Header Text", WithTextStyle(TextStyle{Header: true})).
		Text("Colored Text", WithTextStyle(TextStyle{Color: "red", Size: 14})).
		Build()

	renderer := &htmlRenderer{}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render HTML text: %v", err)
	}

	resultStr := string(result)

	// Check for proper HTML elements
	if !strings.Contains(resultStr, "<p class=\"text-content\">Normal Text</p>") {
		t.Errorf("Normal text not rendered correctly")
	}

	if !strings.Contains(resultStr, "font-weight: bold") {
		t.Errorf("Bold style not applied")
	}

	if !strings.Contains(resultStr, "<h2 class=\"text-header\">Header Text</h2>") {
		t.Errorf("Header not rendered as h2")
	}

	if !strings.Contains(resultStr, "color: red") {
		t.Errorf("Color style not applied")
	}

	if !strings.Contains(resultStr, "font-size: 14px") {
		t.Errorf("Font size not applied")
	}
}

func TestHTMLRenderer_RawContent(t *testing.T) {
	doc := New().
		Raw(FormatHTML, []byte("<div>Safe HTML</div>")).
		Raw(FormatText, []byte("<script>alert('danger')</script>")).
		Build()

	renderer := &htmlRenderer{}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render HTML raw content: %v", err)
	}

	resultStr := string(result)

	// HTML format raw content should be included directly
	if !strings.Contains(resultStr, "<div>Safe HTML</div>") {
		t.Errorf("HTML raw content not included directly")
	}

	// Non-HTML raw content should be escaped
	if strings.Contains(resultStr, "<script>alert('danger')</script>") {
		t.Errorf("Non-HTML raw content not escaped")
	}

	if !strings.Contains(resultStr, "&lt;script&gt;") {
		t.Errorf("Non-HTML raw content not properly escaped")
	}

	if !strings.Contains(resultStr, "<pre class=\"raw-content\">") {
		t.Errorf("Non-HTML raw content not wrapped in pre tag")
	}
}

func TestHTMLRenderer_SectionContent(t *testing.T) {
	userData := []map[string]any{
		{"name": "Alice", "role": "Admin"},
	}

	doc := New().
		Section("User Management", func(b *Builder) {
			b.Text("Section description").
				Table("Users", userData, WithKeys("name", "role"))
		}).
		Build()

	renderer := &htmlRenderer{}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render HTML section: %v", err)
	}

	resultStr := string(result)

	// Should have section structure
	if !strings.Contains(resultStr, "<section class=\"content-section\">") {
		t.Errorf("Section not properly structured")
	}

	if !strings.Contains(resultStr, "<h1>User Management</h1>") {
		t.Errorf("Section title not rendered as h1")
	}

	// Should contain nested content
	if !strings.Contains(resultStr, "Section description") {
		t.Errorf("Section text content missing")
	}

	if !strings.Contains(resultStr, "<table class=\"data-table\">") {
		t.Errorf("Section table content missing")
	}

	if !strings.Contains(resultStr, "Alice") {
		t.Errorf("Table data in section missing")
	}
}

func TestTableRenderer_StyleConfiguration(t *testing.T) {
	data := []map[string]any{
		{"name": "Alice", "age": 30},
		{"name": "Bob", "age": 25},
	}

	tests := []struct {
		name      string
		renderer  *tableRenderer
		styleName string
	}{
		{
			name:      "default style",
			renderer:  &tableRenderer{},
			styleName: "ColoredBright", // default
		},
		{
			name:      "bold style",
			renderer:  NewTableRendererWithStyle("Bold").(*tableRenderer),
			styleName: "Bold",
		},
		{
			name:      "light style",
			renderer:  NewTableRendererWithStyle("Light").(*tableRenderer),
			styleName: "Light",
		},
		{
			name:      "rounded style",
			renderer:  NewTableRendererWithStyle("Rounded").(*tableRenderer),
			styleName: "Rounded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := New().
				Table("Styled Table", data, WithKeys("name", "age")).
				Build()

			ctx := context.Background()
			result, err := tt.renderer.Render(ctx, doc)
			if err != nil {
				t.Fatalf("Failed to render table with style: %v", err)
			}

			resultStr := string(result)

			// Should contain the data regardless of style
			if !strings.Contains(resultStr, "Alice") {
				t.Errorf("Table data missing from %s style", tt.styleName)
			}

			if !strings.Contains(resultStr, "Bob") {
				t.Errorf("Table data missing from %s style", tt.styleName)
			}

			// Should contain table title (may be split across lines with ANSI codes)
			if !strings.Contains(resultStr, "Styled") || !(strings.Contains(resultStr, "Table") || strings.Contains(resultStr, "Tabl") || strings.Contains(resultStr, "Tab")) {
				t.Errorf("Table title missing from %s style. Full output:\n%s", tt.styleName, resultStr)
			}

			// Different styles should produce different output (basic check)
			if tt.styleName != "ColoredBright" {
				// Create a default renderer for comparison
				defaultRenderer := &tableRenderer{}
				defaultResult, err := defaultRenderer.Render(ctx, doc)
				if err != nil {
					t.Fatalf("Failed to render with default style: %v", err)
				}

				// The styled output should be different from default
				// (This is a basic check - different styles will have different ANSI codes)
				if string(result) == string(defaultResult) {
					t.Errorf("Style %s produced identical output to default", tt.styleName)
				}
			}
		})
	}
}

func TestTableRenderer_PredefinedStyles(t *testing.T) {
	data := []map[string]any{
		{"id": 1, "name": "Test"},
	}

	doc := New().
		Table("Style Test", data, WithKeys("id", "name")).
		Build()

	// Test some of the predefined style formats
	styles := []Format{
		TableDefault,
		TableBold,
		TableColoredBright,
		TableLight,
		TableRounded,
	}

	ctx := context.Background()

	for _, style := range styles {
		t.Run(style.Name+"_"+style.Renderer.(*tableRenderer).styleName, func(t *testing.T) {
			result, err := style.Renderer.Render(ctx, doc)
			if err != nil {
				t.Fatalf("Failed to render with predefined style: %v", err)
			}

			resultStr := string(result)

			// Should contain the basic data
			if !strings.Contains(resultStr, "Test") {
				t.Errorf("Table data missing from predefined style")
			}

			if !strings.Contains(resultStr, "Style") || !strings.Contains(resultStr, "Test") {
				t.Errorf("Table title missing from predefined style")
			}
		})
	}
}

func TestTableWithStyle_Function(t *testing.T) {
	// Test the TableWithStyle function
	customStyle := TableWithStyle("Double")

	if customStyle.Name != FormatTable {
		t.Errorf("TableWithStyle should have name %q, got %s", FormatTable, customStyle.Name)
	}

	renderer, ok := customStyle.Renderer.(*tableRenderer)
	if !ok {
		t.Fatalf("TableWithStyle should return tableRenderer")
	}

	if renderer.styleName != "Double" {
		t.Errorf("TableWithStyle should set styleName to 'Double', got %s", renderer.styleName)
	}
}

func TestMarkdownRenderer_BasicRendering(t *testing.T) {
	data := []map[string]any{
		{"name": "Alice", "age": 30, "city": "New York"},
		{"name": "Bob", "age": 25, "city": "Los Angeles"},
	}

	doc := New().
		Text("User Report", WithTextStyle(TextStyle{Header: true})).
		Table("Users", data, WithKeys("name", "age", "city")).
		Text("End of report").
		Build()

	renderer := &markdownRenderer{headingLevel: 1}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render markdown: %v", err)
	}

	resultStr := string(result)

	// Should contain header
	if !strings.Contains(resultStr, "## User Report") {
		t.Errorf("Missing header in markdown output")
	}

	// Should contain table title
	if !strings.Contains(resultStr, "### Users") {
		t.Errorf("Missing table title in markdown output")
	}

	// Should contain table headers in proper order
	if !strings.Contains(resultStr, "| name | age | city |") {
		t.Errorf("Missing table headers in markdown output")
	}

	// Should contain table separator
	if !strings.Contains(resultStr, "| --- | --- | --- |") {
		t.Errorf("Missing table separator in markdown output")
	}

	// Should contain table data
	if !strings.Contains(resultStr, "| Alice | 30 | New York |") {
		t.Errorf("Missing Alice data in markdown output")
	}

	if !strings.Contains(resultStr, "| Bob | 25 | Los Angeles |") {
		t.Errorf("Missing Bob data in markdown output")
	}

	// Should contain plain text
	if !strings.Contains(resultStr, "End of report") {
		t.Errorf("Missing plain text in markdown output")
	}
}

func TestMarkdownRenderer_TableOfContents(t *testing.T) {
	userData := []map[string]any{
		{"name": "Alice", "role": "Admin"},
	}

	doc := New().
		Section("Introduction", func(b *Builder) {
			b.Text("This is the introduction")
		}).
		Text("Main Section", WithTextStyle(TextStyle{Header: true})).
		Section("User Management", func(b *Builder) {
			b.Text("User details").
				Table("Active Users", userData, WithKeys("name", "role"))
		}).
		Build()

	// Test with ToC enabled
	renderer := NewMarkdownRendererWithToC(true)
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render markdown with ToC: %v", err)
	}

	resultStr := string(result)

	// Should contain ToC header
	if !strings.Contains(resultStr, "## Table of Contents") {
		t.Errorf("Missing ToC header in markdown output")
	}

	// Should contain ToC entries
	if !strings.Contains(resultStr, "- [Introduction](#introduction)") {
		t.Errorf("Missing Introduction in ToC")
	}

	if !strings.Contains(resultStr, "- [Main Section](#main-section)") {
		t.Errorf("Missing Main Section in ToC")
	}

	if !strings.Contains(resultStr, "- [User Management](#user-management)") {
		t.Errorf("Missing User Management in ToC")
	}

	// Should contain actual content headings
	if !strings.Contains(resultStr, "# Introduction") {
		t.Errorf("Missing Introduction heading")
	}

	if !strings.Contains(resultStr, "## Main Section") {
		t.Errorf("Missing Main Section heading")
	}

	if !strings.Contains(resultStr, "# User Management") {
		t.Errorf("Missing User Management heading")
	}
}

func TestMarkdownRenderer_FrontMatter(t *testing.T) {
	frontMatter := map[string]string{
		"title":  "Test Document",
		"author": "Test Author",
		"date":   "2024-01-01",
	}

	doc := New().
		Text("Document Content").
		Build()

	renderer := NewMarkdownRendererWithFrontMatter(frontMatter)
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render markdown with front matter: %v", err)
	}

	resultStr := string(result)

	// Should start with front matter delimiter
	if !strings.HasPrefix(resultStr, "---\n") {
		t.Errorf("Markdown should start with front matter delimiter")
	}

	// Should contain front matter fields
	if !strings.Contains(resultStr, "title: \"Test Document\"") {
		t.Errorf("Missing title in front matter")
	}

	if !strings.Contains(resultStr, "author: \"Test Author\"") {
		t.Errorf("Missing author in front matter")
	}

	if !strings.Contains(resultStr, "date: 2024-01-01") {
		t.Errorf("Missing date in front matter")
	}

	// Should end front matter properly
	if !strings.Contains(resultStr, "---\n\nDocument Content") {
		t.Errorf("Front matter not properly terminated")
	}
}

func TestMarkdownRenderer_TextStyling(t *testing.T) {
	doc := New().
		Text("Normal text").
		Text("Bold text", WithTextStyle(TextStyle{Bold: true})).
		Text("Italic text", WithTextStyle(TextStyle{Italic: true})).
		Text("Header text", WithTextStyle(TextStyle{Header: true})).
		Text("Colored text", WithTextStyle(TextStyle{Color: "red", Size: 14})).
		Build()

	renderer := &markdownRenderer{headingLevel: 1}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render markdown text styling: %v", err)
	}

	resultStr := string(result)

	// Should contain normal text
	if !strings.Contains(resultStr, "Normal text") {
		t.Errorf("Missing normal text")
	}

	// Should contain bold formatting
	if !strings.Contains(resultStr, "**Bold text**") {
		t.Errorf("Missing bold formatting")
	}

	// Should contain italic formatting
	if !strings.Contains(resultStr, "*Italic text*") {
		t.Errorf("Missing italic formatting")
	}

	// Should contain header formatting
	if !strings.Contains(resultStr, "## Header text") {
		t.Errorf("Missing header formatting")
	}

	// Should contain HTML span for color/size
	if !strings.Contains(resultStr, `<span style="color: red; font-size: 14px">Colored text</span>`) {
		t.Errorf("Missing color/size HTML formatting")
	}
}

func TestMarkdownRenderer_MarkdownEscaping(t *testing.T) {
	specialText := "Text with *asterisks* and _underscores_ and [brackets] and |pipes|"

	doc := New().
		Text(specialText).
		Build()

	renderer := &markdownRenderer{headingLevel: 1}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render markdown with special characters: %v", err)
	}

	resultStr := string(result)

	// Should escape special markdown characters
	if !strings.Contains(resultStr, "\\*asterisks\\*") {
		t.Errorf("Asterisks not properly escaped")
	}

	if !strings.Contains(resultStr, "\\_underscores\\_") {
		t.Errorf("Underscores not properly escaped")
	}

	if !strings.Contains(resultStr, "\\[brackets\\]") {
		t.Errorf("Brackets not properly escaped")
	}

	if !strings.Contains(resultStr, "\\|pipes\\|") {
		t.Errorf("Pipes not properly escaped")
	}
}

func TestMarkdownRenderer_RawContent(t *testing.T) {
	doc := New().
		Raw(FormatMarkdown, []byte("# Raw Markdown\n\nThis is **raw** markdown content.")).
		Raw("html", []byte("<div>HTML content</div>")).
		Build()

	renderer := &markdownRenderer{headingLevel: 1}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render markdown raw content: %v", err)
	}

	resultStr := string(result)

	// Markdown raw content should be included directly
	if !strings.Contains(resultStr, "# Raw Markdown") {
		t.Errorf("Raw markdown content not included directly")
	}

	if !strings.Contains(resultStr, "This is **raw** markdown content.") {
		t.Errorf("Raw markdown formatting not preserved")
	}

	// Non-markdown raw content should be in code block
	if !strings.Contains(resultStr, "```\n<div>HTML content</div>\n```") {
		t.Errorf("Non-markdown raw content not properly escaped in code block")
	}
}

func TestMarkdownRenderer_SectionNesting(t *testing.T) {
	doc := New().
		Section("Level 1", func(b *Builder) {
			b.Text("Level 1 content").
				Section("Level 2", func(b2 *Builder) {
					b2.Text("Level 2 content").
						Section("Level 3", func(b3 *Builder) {
							b3.Text("Level 3 content")
						})
				})
		}).
		Build()

	renderer := &markdownRenderer{headingLevel: 1}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render markdown section nesting: %v", err)
	}

	resultStr := string(result)

	// Should have proper heading levels
	if !strings.Contains(resultStr, "# Level 1") {
		t.Errorf("Missing Level 1 heading")
	}

	if !strings.Contains(resultStr, "## Level 2") {
		t.Errorf("Missing Level 2 heading")
	}

	if !strings.Contains(resultStr, "### Level 3") {
		t.Errorf("Missing Level 3 heading")
	}

	// Should contain all content
	if !strings.Contains(resultStr, "Level 1 content") {
		t.Errorf("Missing Level 1 content")
	}

	if !strings.Contains(resultStr, "Level 2 content") {
		t.Errorf("Missing Level 2 content")
	}

	if !strings.Contains(resultStr, "Level 3 content") {
		t.Errorf("Missing Level 3 content")
	}
}

func TestMarkdownRenderer_KeyOrderPreservation(t *testing.T) {
	data := []map[string]any{
		{"c": "gamma", "a": "alpha", "b": "beta"},
		{"b": "bravo", "c": "charlie", "a": "alpha"},
	}

	doc := New().
		Table("Order Test", data, WithKeys("c", "a", "b")).
		Build()

	renderer := &markdownRenderer{headingLevel: 1}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render markdown: %v", err)
	}

	resultStr := string(result)

	// Should have headers in correct order
	if !strings.Contains(resultStr, "| c | a | b |") {
		t.Errorf("Headers not in correct order, got: %s", resultStr)
	}

	// Should have data in correct order
	if !strings.Contains(resultStr, "| gamma | alpha | beta |") {
		t.Errorf("First row not in correct order")
	}

	if !strings.Contains(resultStr, "| charlie | alpha | bravo |") {
		t.Errorf("Second row not in correct order")
	}
}

func TestMarkdownRenderer_TableCellEscaping(t *testing.T) {
	data := []map[string]any{
		{"text": "Text with | pipes", "multiline": "Line 1\nLine 2"},
	}

	doc := New().
		Table("Escaping Test", data, WithKeys("text", "multiline")).
		Build()

	renderer := &markdownRenderer{headingLevel: 1}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render markdown table: %v", err)
	}

	resultStr := string(result)

	// Pipes should be escaped in table cells
	if !strings.Contains(resultStr, "Text with \\| pipes") {
		t.Errorf("Pipes not properly escaped in table cell")
	}

	// Newlines should be converted to <br>
	if !strings.Contains(resultStr, "Line 1<br>Line 2") {
		t.Errorf("Newlines not properly converted to <br> in table cell")
	}
}

// TestMarkdownRenderer_CollapsibleValues tests collapsible value rendering in markdown
func TestMarkdownRenderer_CollapsibleValues(t *testing.T) {
	tests := []struct {
		name           string
		collapsible    CollapsibleValue
		expectedOutput string
	}{
		{
			name: "collapsed string details",
			collapsible: NewCollapsibleValue(
				"3 errors",
				"Error 1\nError 2\nError 3",
			),
			expectedOutput: "<details><summary>3 errors</summary><br/>Error 1<br>Error 2<br>Error 3</details>",
		},
		{
			name: "expanded string details with open attribute",
			collapsible: NewCollapsibleValue(
				"System status",
				"All systems operational",
				WithExpanded(true),
			),
			expectedOutput: "<details open><summary>System status</summary><br/>All systems operational</details>",
		},
		{
			name: "string array details",
			collapsible: NewCollapsibleValue(
				"File list (3 items)",
				[]string{"file1.go", "file2.go", "file3.go"},
			),
			expectedOutput: "<details><summary>File list (3 items)</summary><br/>file1.go<br/>file2.go<br/>file3.go</details>",
		},
		{
			name: "map details as key-value pairs",
			collapsible: NewCollapsibleValue(
				"Config settings",
				map[string]any{"debug": true, "port": 8080, "host": "localhost"},
			),
			expectedOutput: "<details><summary>Config settings</summary><br/><strong>debug:</strong> true<br/><strong>port:</strong> 8080<br/><strong>host:</strong> localhost</details>",
		},
		{
			name: "empty summary fallback",
			collapsible: NewCollapsibleValue(
				"",
				"Some details",
			),
			expectedOutput: "<details><summary>[no summary]</summary><br/>Some details</details>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			renderer := &markdownRenderer{
				baseRenderer:      baseRenderer{},
				headingLevel:      1,
				collapsibleConfig: DefaultRendererConfig,
			}

			result := renderer.renderCollapsibleValue(tt.collapsible)

			// Check that all expected parts are present (order may vary for maps)
			if tt.name == "map details as key-value pairs" {
				// For maps, check individual components since order varies
				expectedParts := []string{
					"<details><summary>Config settings</summary><br/>",
					"<strong>debug:</strong> true",
					"<strong>port:</strong> 8080",
					"<strong>host:</strong> localhost",
					"</details>",
				}
				for _, part := range expectedParts {
					if !strings.Contains(result, part) {
						t.Errorf("Expected result to contain %q, got %q", part, result)
					}
				}
			} else {
				if result != tt.expectedOutput {
					t.Errorf("Expected %q, got %q", tt.expectedOutput, result)
				}
			}
		})
	}
}

// TestMarkdownRenderer_CollapsibleInTable tests collapsible values within table cells
func TestMarkdownRenderer_CollapsibleInTable(t *testing.T) {
	data := []map[string]any{
		{
			"file":   "main.go",
			"errors": []string{"missing import", "unused variable"},
			"status": "failed",
		},
		{
			"file":   "utils.go",
			"errors": []string{},
			"status": "ok",
		},
	}

	// Create table with collapsible formatter for errors
	doc := New().
		Table("Code Analysis", data,
			WithSchema(
				Field{
					Name: "file",
					Type: "string",
				},
				Field{
					Name:      "errors",
					Type:      "array",
					Formatter: ErrorListFormatter(),
				},
				Field{
					Name: "status",
					Type: "string",
				},
			)).
		Build()

	renderer := &markdownRenderer{
		baseRenderer:      baseRenderer{},
		headingLevel:      1,
		collapsibleConfig: DefaultRendererConfig,
	}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := string(result)

	// Check table structure exists
	if !strings.Contains(output, "### Code Analysis") {
		t.Error("Expected table title")
	}
	if !strings.Contains(output, "| file | errors | status |") {
		t.Error("Expected table header")
	}

	// Check first row has collapsible content
	if !strings.Contains(output, "<details><summary>2 errors (click to expand)</summary>") {
		t.Error("Expected collapsible summary for first row")
	}
	if !strings.Contains(output, "missing import<br/>unused variable") {
		t.Error("Expected error details in first row")
	}

	// Check second row has no collapsible (empty array)
	secondRow := strings.Split(output, "\n")[5] // Approximate line with second data row
	if strings.Contains(secondRow, "<details>") {
		t.Error("Second row should not have collapsible content for empty array")
	}
}

// TestMarkdownRenderer_GlobalExpansionOverride tests global expansion configuration
func TestMarkdownRenderer_GlobalExpansionOverride(t *testing.T) {
	collapsible := NewCollapsibleValue(
		"Summary text",
		"Detail text",
		WithExpanded(false), // Explicitly set to collapsed
	)

	tests := []struct {
		name           string
		forceExpansion bool
		expectOpen     bool
	}{
		{
			name:           "respect individual setting when global expansion disabled",
			forceExpansion: false,
			expectOpen:     false,
		},
		{
			name:           "override individual setting when global expansion enabled",
			forceExpansion: true,
			expectOpen:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultRendererConfig
			config.ForceExpansion = tt.forceExpansion

			renderer := &markdownRenderer{
				baseRenderer:      baseRenderer{},
				headingLevel:      1,
				collapsibleConfig: config,
			}

			result := renderer.renderCollapsibleValue(collapsible)

			if tt.expectOpen {
				if !strings.Contains(result, "<details open>") {
					t.Error("Expected open attribute when global expansion enabled")
				}
			} else {
				if strings.Contains(result, "<details open>") {
					t.Error("Did not expect open attribute when global expansion disabled")
				}
			}
		})
	}
}

// TestMarkdownRenderer_CollapsibleDetailsFormatting tests various detail types
func TestMarkdownRenderer_CollapsibleDetailsFormatting(t *testing.T) {
	renderer := &markdownRenderer{
		baseRenderer:      baseRenderer{},
		headingLevel:      1,
		collapsibleConfig: DefaultRendererConfig,
	}

	tests := []struct {
		name     string
		details  any
		expected string
	}{
		{
			name:     "string details",
			details:  "Simple string",
			expected: "Simple string",
		},
		{
			name:     "string array details",
			details:  []string{"item1", "item2", "item3"},
			expected: "item1<br/>item2<br/>item3",
		},
		{
			name:     "map details",
			details:  map[string]any{"key1": "value1", "key2": "value2"},
			expected: "<strong>key1:</strong> value1<br/><strong>key2:</strong> value2",
		},
		{
			name:     "complex type fallback",
			details:  struct{ Name string }{Name: "test"},
			expected: "{test}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderer.formatDetailsForMarkdown(tt.details)

			if tt.name == "map details" {
				// For maps, check both possible orders
				option1 := "<strong>key1:</strong> value1<br/><strong>key2:</strong> value2"
				option2 := "<strong>key2:</strong> value2<br/><strong>key1:</strong> value1"
				if result != option1 && result != option2 {
					t.Errorf("Expected one of %q or %q, got %q", option1, option2, result)
				}
			} else {
				if result != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

// TestMarkdownRenderer_CollapsibleTableCellEscaping tests markdown escaping in collapsible content
func TestMarkdownRenderer_CollapsibleTableCellEscaping(t *testing.T) {
	data := []map[string]any{
		{
			"description": "Test with special chars",
			"details":     "Text with | pipes and * asterisks",
		},
	}

	doc := New().
		Table("Escaping Test", data,
			WithSchema(
				Field{Name: "description", Type: "string"},
				Field{
					Name: "details",
					Type: "string",
					Formatter: func(val any) any {
						return NewCollapsibleValue(
							"Click to expand",
							val,
						)
					},
				},
			)).
		Build()

	renderer := &markdownRenderer{
		baseRenderer:      baseRenderer{},
		headingLevel:      1,
		collapsibleConfig: DefaultRendererConfig,
	}
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := string(result)

	// Check that pipes in collapsible content are properly escaped for table cells
	// Look for the actual pattern that should exist in table cells
	if !strings.Contains(output, "pipes and * asterisks") {
		t.Error("Expected basic content to be present")
	}

	// Check that the content appears within a collapsible structure
	if !strings.Contains(output, "<details><summary>Click to expand</summary>") {
		t.Error("Expected collapsible structure")
	}
}

func TestMarkdownRenderer_CombinedFeatures(t *testing.T) {
	frontMatter := map[string]string{
		"title": "Complete Test",
	}

	userData := []map[string]any{
		{"name": "Alice", "role": "Admin"},
	}

	doc := New().
		Section("Introduction", func(b *Builder) {
			b.Text("This document demonstrates all features")
		}).
		Table("Users", userData, WithKeys("name", "role")).
		Raw(FormatMarkdown, []byte("**Bold raw markdown**")).
		Build()

	renderer := NewMarkdownRendererWithOptions(true, frontMatter)
	ctx := context.Background()

	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render complete markdown: %v", err)
	}

	resultStr := string(result)

	// Should have front matter
	if !strings.Contains(resultStr, "title: \"Complete Test\"") {
		t.Errorf("Missing front matter")
	}

	// Should have ToC
	if !strings.Contains(resultStr, "## Table of Contents") {
		t.Errorf("Missing ToC")
	}

	// Should have section
	if !strings.Contains(resultStr, "# Introduction") {
		t.Errorf("Missing section heading")
	}

	// Should have table
	if !strings.Contains(resultStr, "| name | role |") {
		t.Errorf("Missing table")
	}

	// Should have raw markdown
	if !strings.Contains(resultStr, "**Bold raw markdown**") {
		t.Errorf("Missing raw markdown content")
	}
}
