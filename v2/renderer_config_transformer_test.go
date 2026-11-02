package output

import (
	"bytes"
	"strings"
	"testing"
)

// Regression tests for T-1718: Markdown and streaming HTML renderers ignored
// RendererConfig.DataTransformers and RendererConfig.ByteTransformers.
//
// Before the fix, markdownRenderer.Render called renderDocumentMarkdown
// directly and htmlRenderer.RenderTo used baseRenderer.renderDocumentTo; both
// paths skipped the renderer-level transformers that the buffered HTML path
// (renderDocumentWithFormat) applies. The same configured report therefore
// produced different content depending on format and on Render vs RenderTo.

// newTransformerConfig returns a RendererConfig with a filtering data
// transformer and a suffix byte transformer, both limited to format.
func newTransformerConfig(format, suffix string) (RendererConfig, *testDataTransformer, *testByteTransformer) {
	dataTransformer := &testDataTransformer{
		name:     "test-filter",
		priority: 100,
		formats:  []string{format},
	}
	byteTransformer := &testByteTransformer{
		name:     "test-suffix",
		priority: 200,
		formats:  []string{format},
		suffix:   suffix,
	}

	config := DefaultRendererConfig
	config.DataTransformers = []*TransformerAdapter{NewTransformerAdapter(dataTransformer)}
	config.ByteTransformers = NewTransformPipeline()
	config.ByteTransformers.Add(byteTransformer)
	return config, dataTransformer, byteTransformer
}

// assertTransformed checks the effect of the filter data transformer and the
// suffix byte transformer on rendered output.
func assertTransformed(t *testing.T, out, suffix string) {
	t.Helper()
	if strings.Contains(out, "Bob") {
		t.Errorf("data transformer should have filtered out inactive user Bob; output:\n%s", out)
	}
	// Markdown escapes the brackets, so accept either form.
	if !strings.Contains(out, "[FILTERED]Alice") && !strings.Contains(out, `\[FILTERED\]Alice`) {
		t.Errorf("data transformer should have prefixed Alice; output:\n%s", out)
	}
	if !strings.HasSuffix(out, suffix) {
		t.Errorf("byte transformer should have appended %q; output:\n%s", suffix, out)
	}
}

func TestNewMarkdownRendererWithCollapsible_AppliesTransformers(t *testing.T) {
	const suffix = "\n<!-- markdown byte transformer -->"

	tests := map[string]struct {
		render func(r Renderer) (string, error)
	}{
		"Render": {
			render: func(r Renderer) (string, error) {
				out, err := r.Render(t.Context(), newConstructorTransformerDoc())
				return string(out), err
			},
		},
		"RenderTo": {
			render: func(r Renderer) (string, error) {
				var buf bytes.Buffer
				err := r.RenderTo(t.Context(), newConstructorTransformerDoc(), &buf)
				return buf.String(), err
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			config, dataTransformer, byteTransformer := newTransformerConfig(FormatMarkdown, suffix)

			got, err := tc.render(NewMarkdownRendererWithCollapsible(config))
			if err != nil {
				t.Fatalf("render failed: %v", err)
			}

			if dataTransformer.callCount != 1 {
				t.Errorf("data transformer calls = %d, want 1", dataTransformer.callCount)
			}
			if byteTransformer.callCount != 1 {
				t.Errorf("byte transformer calls = %d, want 1", byteTransformer.callCount)
			}
			assertTransformed(t, got, suffix)
		})
	}
}

func TestNewHTMLRendererWithCollapsible_RenderToAppliesTransformers(t *testing.T) {
	const suffix = "\n<!-- html byte transformer -->"

	config, dataTransformer, byteTransformer := newTransformerConfig(FormatHTML, suffix)
	renderer := NewHTMLRendererWithCollapsible(config)

	var buf bytes.Buffer
	if err := renderer.RenderTo(t.Context(), newConstructorTransformerDoc(), &buf); err != nil {
		t.Fatalf("RenderTo failed: %v", err)
	}
	got := buf.String()

	if dataTransformer.callCount != 1 {
		t.Errorf("data transformer calls = %d, want 1", dataTransformer.callCount)
	}
	if byteTransformer.callCount != 1 {
		t.Errorf("byte transformer calls = %d, want 1", byteTransformer.callCount)
	}
	assertTransformed(t, got, suffix)

	// Streaming output must match buffered output for the same config.
	want, err := renderer.Render(t.Context(), newConstructorTransformerDoc())
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if got != string(want) {
		t.Errorf("RenderTo output differs from Render output\nRenderTo:\n%s\nRender:\n%s", got, want)
	}
}

// TestHTMLRenderTo_DataTransformerOnly checks the streaming path
// with data transformers and no byte transformers (no buffering needed).
func TestHTMLRenderTo_DataTransformerOnly(t *testing.T) {
	config, dataTransformer, _ := newTransformerConfig(FormatHTML, "")
	config.ByteTransformers = nil
	renderer := NewHTMLRendererWithCollapsible(config)

	var buf bytes.Buffer
	if err := renderer.RenderTo(t.Context(), newConstructorTransformerDoc(), &buf); err != nil {
		t.Fatalf("RenderTo failed: %v", err)
	}
	if dataTransformer.callCount != 1 {
		t.Errorf("data transformer calls = %d, want 1", dataTransformer.callCount)
	}
	assertTransformed(t, buf.String(), "")
}

// Regression tests for T-2322 (RendererConfig part): the RendererConfig-taking
// renderer constructors stored the config by value, leaving its
// DataTransformers slice, HTMLCSSClasses map and ByteTransformers pipeline
// aliased to the caller. Mutating the caller's config after construction
// changed the already-constructed renderer's output.

func TestRendererConstructors_DoNotAliasCallerConfig(t *testing.T) {
	tests := map[string]struct {
		format      string
		constructor func(RendererConfig) Renderer
	}{
		"markdown": {format: FormatMarkdown, constructor: NewMarkdownRendererWithCollapsible},
		"html":     {format: FormatHTML, constructor: NewHTMLRendererWithCollapsible},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			noop := &testDataTransformer{name: "noop", formats: []string{"none"}}
			config := DefaultRendererConfig
			config.DataTransformers = []*TransformerAdapter{NewTransformerAdapter(noop)}
			config.ByteTransformers = NewTransformPipeline()

			renderer := tc.constructor(config)

			// Mutate the caller's config after construction: replace the
			// slice element and add a byte transformer to the pipeline.
			config.DataTransformers[0] = NewTransformerAdapter(&testDataTransformer{
				name:     "test-filter",
				priority: 100,
				formats:  []string{tc.format},
			})
			config.ByteTransformers.Add(&testByteTransformer{
				name:    "late-suffix",
				formats: []string{tc.format},
				suffix:  "LATE-SUFFIX",
			})

			out, err := renderer.Render(t.Context(), newConstructorTransformerDoc())
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}
			got := string(out)

			if strings.Contains(got, "FILTERED") || !strings.Contains(got, "Bob") {
				t.Errorf("caller's DataTransformers mutation leaked into renderer; output:\n%s", got)
			}
			if strings.Contains(got, "LATE-SUFFIX") {
				t.Errorf("caller's ByteTransformers mutation leaked into renderer; output:\n%s", got)
			}
		})
	}
}

func TestNewHTMLRendererWithCollapsible_DoesNotAliasCSSClasses(t *testing.T) {
	config := DefaultRendererConfig
	config.HTMLCSSClasses = map[string]string{
		keyDetails: "orig-cell",
		keySummary: "orig-summary",
		keyContent: "orig-details",
	}
	renderer := NewHTMLRendererWithCollapsible(config)

	config.HTMLCSSClasses[keyDetails] = "mutated-cell"

	doc := New().
		Table("t", []Record{{"v": NewCollapsibleValue("sum", "detail")}}, WithKeys("v")).
		Build()
	out, err := renderer.Render(t.Context(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	got := string(out)

	if strings.Contains(got, "mutated-cell") {
		t.Errorf("caller's HTMLCSSClasses mutation leaked into renderer; output:\n%s", got)
	}
	if !strings.Contains(got, "orig-cell") {
		t.Errorf("renderer should keep the CSS classes supplied at construction; output:\n%s", got)
	}
}

func TestCloneRendererConfig_NilFields(t *testing.T) {
	got := cloneRendererConfig(RendererConfig{MaxDetailLength: 7})
	if got.DataTransformers != nil || got.HTMLCSSClasses != nil || got.ByteTransformers != nil {
		t.Errorf("nil reference fields should stay nil, got %+v", got)
	}
	if got.MaxDetailLength != 7 {
		t.Errorf("MaxDetailLength = %d, want 7", got.MaxDetailLength)
	}
}
