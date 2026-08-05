package output

import (
	"context"
	"strings"
	"testing"
)

// Regression tests for T-1438: the data transformer path in
// baseRenderer.applyDataTransformers panicked on nil *TransformerAdapter
// entries in RendererConfig.DataTransformers, and silently propagated a nil
// Content when a DataTransformer returned (nil, nil), causing a later panic
// in applyContentTransformations.

// nilResultDataTransformer returns (nil, nil) from TransformData, simulating
// a buggy custom transformer.
type nilResultDataTransformer struct{}

func (n *nilResultDataTransformer) Name() string     { return "nil-result" }
func (n *nilResultDataTransformer) Priority() int    { return 100 }
func (n *nilResultDataTransformer) Describe() string { return "returns nil content with nil error" }

func (n *nilResultDataTransformer) CanTransform(content Content, format string) bool {
	return content.Type() == ContentTypeTable && format == FormatHTML
}

func (n *nilResultDataTransformer) TransformData(ctx context.Context, content Content, format string) (Content, error) {
	return nil, nil
}

func TestApplyDataTransformersNilAdapterEntry(t *testing.T) {
	tests := map[string]struct {
		transformers    []*TransformerAdapter
		wantInOutput    string
		wantErrContains string
	}{
		"only nil adapter entry": {
			transformers: []*TransformerAdapter{nil},
			wantInOutput: "Alice",
		},
		"nil adapter entry alongside valid transformer": {
			transformers: []*TransformerAdapter{
				nil,
				NewTransformerAdapter(&testDataTransformer{
					name:     "valid",
					priority: 100,
					formats:  []string{FormatHTML},
					prefix:   "[OK]",
				}),
			},
			wantInOutput: "[OK]Alice",
		},
		"nil adapter entry alongside erroring transformer": {
			transformers: []*TransformerAdapter{
				nil,
				NewTransformerAdapter(&rendererFailingDataTransformer{
					name:     "failing",
					priority: 100,
					formats:  []string{FormatHTML},
				}),
			},
			wantErrContains: "simulated failure in transformer failing",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			doc := New().
				Table("users", []Record{{"name": "Alice"}}, WithKeys("name")).
				Build()

			renderer := NewHTMLRendererWithCollapsible(RendererConfig{
				DataTransformers: tc.transformers,
			})

			// Expected: nil adapter entries are ignored, so rendering
			// either succeeds or surfaces the remaining transformer's
			// error. Actual (before fix): panic via nil pointer
			// dereference in TransformerAdapter.IsDataTransformer.
			got, err := renderer.Render(context.Background(), doc)
			if tc.wantErrContains != "" {
				if err == nil {
					t.Fatalf("Render() error = nil, want error containing %q", tc.wantErrContains)
				}
				if !strings.Contains(err.Error(), tc.wantErrContains) {
					t.Errorf("Render() error = %q, want it to contain %q", err, tc.wantErrContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("Render() error = %v, want nil", err)
			}
			if !strings.Contains(string(got), tc.wantInOutput) {
				t.Errorf("Render() output does not contain %q, got: %s", tc.wantInOutput, got)
			}
		})
	}
}

func TestApplyDataTransformersNilTransformedContent(t *testing.T) {
	doc := New().
		Table("users", []Record{{"name": "Alice"}}, WithKeys("name")).
		Build()

	renderer := NewHTMLRendererWithCollapsible(RendererConfig{
		DataTransformers: []*TransformerAdapter{
			NewTransformerAdapter(&nilResultDataTransformer{}),
		},
	})

	// Expected: a transformer returning (nil, nil) yields a transformation
	// error before the transformed document is constructed. Actual (before
	// fix): nil Content enters the document and rendering panics in
	// applyContentTransformations.
	_, err := renderer.Render(context.Background(), doc)
	if err == nil {
		t.Fatal("Render() error = nil, want error for nil transformed content")
	}
	if !strings.Contains(err.Error(), "nil-result") {
		t.Errorf("Render() error = %q, want it to reference transformer name %q", err, "nil-result")
	}
}
