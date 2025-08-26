package output

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

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

func TestJSONYAMLRenderers_LargeDataset(t *testing.T) {
	// Create a larger dataset to test streaming performance
	const recordCount = 1000
	var data []map[string]any
	for i := range recordCount {
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
