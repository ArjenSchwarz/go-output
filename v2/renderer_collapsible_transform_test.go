package output

import (
	"context"
	"strings"
	"testing"
)

// collapsibleTransformSection builds an expanded collapsible section holding a
// table whose per-content transformations (filter + limit) must be applied at
// render time. The table has four rows:
//
//   - keep1 (keep = true)
//   - drop1 (keep = false, filtered out)
//   - keep2 (keep = true)
//   - keep3 (keep = true, removed by limit 2)
//
// After the filter (keep == true) and limit (2), only keep1 and keep2 survive.
func collapsibleTransformSection(t *testing.T) *DefaultCollapsibleSection {
	t.Helper()

	table, err := NewTableContent("Filtered", []map[string]any{
		{"name": "keep1", "keep": true},
		{"name": "drop1", "keep": false},
		{"name": "keep2", "keep": true},
		{"name": "keep3", "keep": true},
	},
		WithKeys("name", "keep"),
		WithTransformations(
			NewFilterOp(func(r Record) bool {
				kept, _ := r["keep"].(bool)
				return kept
			}),
			NewLimitOp(2),
		),
	)
	if err != nil {
		t.Fatalf("NewTableContent() error = %v", err)
	}

	return NewCollapsibleSection("Details", []Content{table}, WithSectionExpanded(true))
}

// assertTransformedRows verifies that rendered output contains only the rows
// surviving the filter+limit transformations of collapsibleTransformSection.
func assertTransformedRows(t *testing.T, renderer string, got []byte) {
	t.Helper()

	gotStr := string(got)
	for _, want := range []string{"keep1", "keep2"} {
		if !strings.Contains(gotStr, want) {
			t.Errorf("%s output missing expected row %q\nGot:\n%s", renderer, want, gotStr)
		}
	}
	for _, absent := range []string{"drop1", "keep3"} {
		if strings.Contains(gotStr, absent) {
			t.Errorf("%s output includes transformed-out row %q\nGot:\n%s", renderer, absent, gotStr)
		}
	}
}

// TestCSVRenderer_CollapsibleSectionAppliesTransformations is a regression
// test for T-1635: the CSV renderer must apply per-content transformations to
// content nested inside a DefaultCollapsibleSection, exactly as it already
// does for top-level content and for content in regular sections. Previously
// renderCollapsibleSectionCSV rendered nested tables directly, so filtered-out
// rows appeared in the output.
func TestCSVRenderer_CollapsibleSectionAppliesTransformations(t *testing.T) {
	doc := New().AddContent(collapsibleTransformSection(t)).Build()
	renderer := NewCSVRendererWithCollapsible(DefaultRendererConfig)

	got, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("csvRenderer.Render() error = %v", err)
	}
	assertTransformedRows(t, "CSV", got)
}

// TestTableRenderer_CollapsibleSectionAppliesTransformations is a regression
// test for T-1635 (consolidating T-1637): the terminal-table renderer must
// apply per-content transformations to content nested inside an expanded
// DefaultCollapsibleSection. Previously renderCollapsibleSection rendered
// nested tables directly, so filtered-out rows appeared in the output.
func TestTableRenderer_CollapsibleSectionAppliesTransformations(t *testing.T) {
	doc := New().AddContent(collapsibleTransformSection(t)).Build()
	renderer := NewTableRendererWithCollapsible("Default", DefaultRendererConfig)

	got, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("tableRenderer.Render() error = %v", err)
	}
	assertTransformedRows(t, "table", got)
}

// TestTableRenderer_NestedCollapsibleSectionAppliesTransformations covers the
// second call site of renderCollapsibleSection (T-1635): a collapsible section
// placed inside a regular SectionContent must also have its nested content
// transformed.
func TestTableRenderer_NestedCollapsibleSectionAppliesTransformations(t *testing.T) {
	section := NewSectionContent("Outer")
	section.AddContent(collapsibleTransformSection(t))
	doc := New().AddContent(section).Build()
	renderer := NewTableRendererWithCollapsible("Default", DefaultRendererConfig)

	got, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("tableRenderer.Render() error = %v", err)
	}
	assertTransformedRows(t, "table", got)
}
