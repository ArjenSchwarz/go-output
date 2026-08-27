package output

import (
	"context"
	"testing"
)

// Regression tests for T-1510: the byte-level tabular transformers falsely
// advertised HTML support. SortTransformer and LineSplitTransformer only
// understand CSV (via encoding/csv) or line-based tab/comma/pipe table text.
// Rendered HTML tables use none of those separators, so sorting was a no-op
// despite the advertised support, and line splitting could corrupt raw HTML
// whenever cell content contained the configured separator. The predicates
// (including FormatDetector's tabular family and the format-aware wrappers)
// must therefore report HTML as unsupported so the pipeline never runs these
// transformers on HTML output.

func TestTabularTransformersDoNotAdvertiseHTMLSupport(t *testing.T) {
	detector := NewFormatDetector()

	tests := map[string]func() bool{
		"SortTransformer.CanTransform": func() bool {
			return NewSortTransformer("name", true).CanTransform(FormatHTML)
		},
		"LineSplitTransformer.CanTransform": func() bool {
			return NewLineSplitTransformerDefault().CanTransform(FormatHTML)
		},
		"FormatDetector.IsTabularFormat": func() bool {
			return detector.IsTabularFormat(FormatHTML)
		},
		"FormatDetector.SupportsSorting": func() bool {
			return detector.SupportsSorting(FormatHTML)
		},
		"FormatDetector.SupportsLineSplitting": func() bool {
			return detector.SupportsLineSplitting(FormatHTML)
		},
		"EnhancedSortTransformer.CanTransform": func() bool {
			return NewEnhancedSortTransformer("name", true).CanTransform(FormatHTML)
		},
		"FormatAwareTransformer(sort).CanTransform": func() bool {
			return NewFormatAwareTransformer(NewSortTransformer("name", true)).CanTransform(FormatHTML)
		},
		"FormatAwareTransformer(linesplit).CanTransform": func() bool {
			return NewFormatAwareTransformer(NewLineSplitTransformerDefault()).CanTransform(FormatHTML)
		},
	}

	for name, predicate := range tests {
		t.Run(name, func(t *testing.T) {
			if got := predicate(); got != false {
				t.Errorf("%s(%s) = %t, want false: byte-level tabular transformers cannot parse HTML", name, FormatHTML, got)
			}
		})
	}
}

// TestTransformPipeline_LineSplitLeavesHTMLIntact demonstrates the corruption
// the false advertisement enabled: with LineSplitTransformer registered, the
// pipeline previously ran it on rendered HTML, where a comma inside cell text
// was mistaken for a column separator and a cell containing the line separator
// was split across mangled rows. HTML output must pass through unchanged.
func TestTransformPipeline_LineSplitLeavesHTMLIntact(t *testing.T) {
	input := "<table>\n" +
		"<tr><th>Name</th><th>Skills</th></tr>\n" +
		"<tr><td>Alice</td><td>Java;Go, senior</td></tr>\n" +
		"</table>"

	pipeline := NewTransformPipeline()
	pipeline.Add(NewLineSplitTransformer(";"))

	got, err := pipeline.Transform(context.Background(), []byte(input), FormatHTML)
	if err != nil {
		t.Fatalf("TransformPipeline.Transform() error = %v", err)
	}

	if string(got) != input {
		t.Errorf("TransformPipeline.Transform() corrupted HTML output\n got = %q\nwant = %q", string(got), input)
	}
}

// TestTransformPipeline_SortLeavesHTMLIntact covers the sort side at pipeline
// level: sort must not run on HTML at all, so the bytes pass through unchanged.
func TestTransformPipeline_SortLeavesHTMLIntact(t *testing.T) {
	input := "<table>\n" +
		"<tr><th>Name</th><th>Age</th></tr>\n" +
		"<tr><td>Charlie</td><td>30</td></tr>\n" +
		"<tr><td>Alice</td><td>25</td></tr>\n" +
		"</table>"

	pipeline := NewTransformPipeline()
	pipeline.Add(NewSortTransformer("Name", true))

	got, err := pipeline.Transform(context.Background(), []byte(input), FormatHTML)
	if err != nil {
		t.Fatalf("TransformPipeline.Transform() error = %v", err)
	}

	if string(got) != input {
		t.Errorf("TransformPipeline.Transform() modified HTML output\n got = %q\nwant = %q", string(got), input)
	}
}

// TestEnhancedSortTransformer_Transform_HTMLPassthrough verifies the Enhanced
// layer's internal gate: Transform on HTML returns the input unchanged rather
// than attempting a byte-level sort.
func TestEnhancedSortTransformer_Transform_HTMLPassthrough(t *testing.T) {
	transformer := NewEnhancedSortTransformer("Name", true)
	input := "<table>\n<tr><th>Name</th></tr>\n<tr><td>Bob, junior</td></tr>\n</table>"

	got, err := transformer.Transform(context.Background(), []byte(input), FormatHTML)
	if err != nil {
		t.Fatalf("EnhancedSortTransformer.Transform() error = %v", err)
	}

	if string(got) != input {
		t.Errorf("EnhancedSortTransformer.Transform() modified HTML output\n got = %q\nwant = %q", string(got), input)
	}
}
