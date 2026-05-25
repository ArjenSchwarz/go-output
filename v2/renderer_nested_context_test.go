package output

import (
	"context"
	"errors"
	"testing"
)

// T-1253 regression tests: nested renderers (sections and collapsible sections)
// must thread the active render context into applyContentTransformations so that
// cancelled/timed-out renders stop consistently at all depths, not just for
// top-level content.
//
// The top-level render loops already short-circuit on an already-cancelled
// context before they reach the nested section helpers, so an
// already-cancelled context cannot prove the nested bug for every format.
// Instead these tests cancel the context mid-render from inside the first
// nested content's transformation, then assert that a *second* nested content
// observes the cancellation. With the bug present, the section/collapsible
// helpers passed context.Background() to applyContentTransformations, so the
// second nested content never saw the cancellation and the render completed.
// With the fix, the caller's context is threaded through and the render stops
// with context.Canceled.

// cancellingOperation cancels the supplied context the first time it is applied.
// It is used as the first nested content's transformation to simulate a render
// being cancelled partway through.
type cancellingOperation struct {
	cancel context.CancelFunc
}

func (c *cancellingOperation) Name() string { return "cancel" }

func (c *cancellingOperation) Validate() error { return nil }

func (c *cancellingOperation) Apply(_ context.Context, content Content) (Content, error) {
	c.cancel()
	return content, nil
}

func (c *cancellingOperation) CanOptimize(Operation) bool { return false }

// newTableWithTransformation builds a table content carrying the supplied
// transformations. Transformations are required so applyContentTransformations
// performs its per-operation ctx.Err() check during rendering.
func newTableWithTransformation(t *testing.T, ops ...Operation) *TableContent {
	t.Helper()
	table, err := NewTableContent(
		"nested",
		[]map[string]any{{"name": "Alice"}},
		WithKeys("name"),
		WithTransformations(ops...),
	)
	if err != nil {
		t.Fatalf("failed to build table content: %v", err)
	}
	return table
}

// nestedContents returns two table contents: the first cancels the context when
// its transformation is applied, the second carries a no-op transformation so
// applyContentTransformations checks ctx.Err() and surfaces the cancellation.
func nestedContents(t *testing.T, cancel context.CancelFunc) []Content {
	t.Helper()
	first := newTableWithTransformation(t, &cancellingOperation{cancel: cancel})
	second := newTableWithTransformation(t, &mockTransformOperation{name: "noop"})
	return []Content{first, second}
}

func sectionWithNestedContents(t *testing.T, cancel context.CancelFunc) *SectionContent {
	t.Helper()
	section := NewSectionContent("section")
	for _, c := range nestedContents(t, cancel) {
		section.AddContent(c)
	}
	return section
}

func collapsibleWithNestedContents(t *testing.T, cancel context.CancelFunc) *DefaultCollapsibleSection {
	t.Helper()
	return NewCollapsibleSection("collapsible", nestedContents(t, cancel))
}

func TestNestedRendererPropagatesCancelledContext(t *testing.T) {
	type builder func(t *testing.T, cancel context.CancelFunc) Content

	sectionBuilder := func(t *testing.T, cancel context.CancelFunc) Content {
		return sectionWithNestedContents(t, cancel)
	}
	collapsibleBuilder := func(t *testing.T, cancel context.CancelFunc) Content {
		return collapsibleWithNestedContents(t, cancel)
	}

	tests := map[string]struct {
		renderer Renderer
		build    builder
	}{
		"json section":                 {renderer: &jsonRenderer{}, build: sectionBuilder},
		"json collapsible section":     {renderer: &jsonRenderer{}, build: collapsibleBuilder},
		"yaml section":                 {renderer: &yamlRenderer{}, build: sectionBuilder},
		"yaml collapsible section":     {renderer: &yamlRenderer{}, build: collapsibleBuilder},
		"html section":                 {renderer: &htmlRenderer{}, build: sectionBuilder},
		"html collapsible section":     {renderer: &htmlRenderer{}, build: collapsibleBuilder},
		"markdown section":             {renderer: &markdownRenderer{}, build: sectionBuilder},
		"markdown collapsible section": {renderer: &markdownRenderer{}, build: collapsibleBuilder},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			doc := New().AddContent(tc.build(t, cancel)).Build()

			_, err := tc.renderer.Render(ctx, doc)
			if err == nil {
				t.Fatalf("expected render to fail with a context error, got nil")
			}
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("expected context.Canceled, got %v", err)
			}
		})
	}
}
