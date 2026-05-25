package output

import (
	"context"
	"strings"
	"testing"
)

// Regression tests for T-1223: Collapsible renderer constructors ignore transformer config.
//
// RendererConfig carries DataTransformers and ByteTransformers, and
// baseRenderer.renderDocumentWithFormat reads them from baseRenderer.config.
// Before the fix, the New*RendererWithCollapsible constructors stored the
// supplied config only in the renderer's collapsibleConfig field and left
// baseRenderer.config at its zero value, so transformers passed through the
// constructor were silently ignored.
//
// Only the HTML renderer routes its Render call through
// renderDocumentWithFormat (the path that reads baseRenderer.config). The
// expected behaviour after the fix is that transformers supplied via
// NewHTMLRendererWithCollapsible actually run.

// newConstructorTransformerDoc builds a small table document used by the
// regression tests below.
func newConstructorTransformerDoc() *Document {
	records := []Record{
		{"name": "Alice", "status": "active"},
		{"name": "Bob", "status": "inactive"},
	}
	return New().
		Table("users", records, WithKeys("name", "status")).
		Build()
}

// TestNewHTMLRendererWithCollapsible_AppliesDataTransformer verifies that a
// data transformer supplied via the constructor's RendererConfig actually runs.
func TestNewHTMLRendererWithCollapsible_AppliesDataTransformer(t *testing.T) {
	dataTransformer := &testDataTransformer{
		name:     "test-filter",
		priority: 100,
		formats:  []string{FormatHTML},
	}

	config := DefaultRendererConfig
	config.DataTransformers = []*TransformerAdapter{
		NewTransformerAdapter(dataTransformer),
	}
	config.ByteTransformers = NewTransformPipeline()

	renderer := NewHTMLRendererWithCollapsible(config)

	output, err := renderer.Render(context.Background(), newConstructorTransformerDoc())
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	out := string(output)

	// Expected: the data transformer filters out the inactive user and
	// prefixes the active user's name.
	if dataTransformer.callCount != 1 {
		t.Errorf("data transformer should have run once, got %d calls", dataTransformer.callCount)
	}
	if strings.Contains(out, "Bob") {
		t.Error("data transformer should have filtered out inactive user Bob")
	}
	if !strings.Contains(out, "[FILTERED]Alice") {
		t.Errorf("data transformer should have prefixed Alice; output: %s", out)
	}
}

// TestNewHTMLRendererWithCollapsible_AppliesByteTransformer verifies that a
// byte transformer supplied via the constructor's RendererConfig actually runs.
func TestNewHTMLRendererWithCollapsible_AppliesByteTransformer(t *testing.T) {
	byteTransformer := &testByteTransformer{
		name:     "test-suffix",
		priority: 200,
		formats:  []string{FormatHTML},
		suffix:   "\n<!-- Processed by byte transformer -->",
	}

	config := DefaultRendererConfig
	config.DataTransformers = nil
	config.ByteTransformers = NewTransformPipeline()
	config.ByteTransformers.Add(byteTransformer)

	renderer := NewHTMLRendererWithCollapsible(config)

	output, err := renderer.Render(context.Background(), newConstructorTransformerDoc())
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	out := string(output)

	if byteTransformer.callCount != 1 {
		t.Errorf("byte transformer should have run once, got %d calls", byteTransformer.callCount)
	}
	if !strings.Contains(out, "<!-- Processed by byte transformer -->") {
		t.Errorf("byte transformer should have appended its suffix; output: %s", out)
	}
}
