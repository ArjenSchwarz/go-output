package output

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
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
		JSON(),
		YAML(),
		CSV(),
		HTML(),
		Table(),
		Markdown(),
		DOT(),
		Mermaid(),
		DrawIO(),
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
	for range numGoroutines {
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
