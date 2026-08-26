package output

import (
	"context"
	"strings"
	"testing"
)

// Regression tests for T-1472: collapsible sections panic on nil nested
// content.
//
// NewCollapsibleSection copied the caller-provided []Content verbatim,
// preserving nil entries. DefaultCollapsibleSection.AppendText and Clone then
// dereferenced them and panicked, as did the collapsible-section paths of the
// Markdown, HTML, JSON, YAML, CSV, and table renderers. The table helpers
// (NewCollapsibleTable, NewCollapsibleMultiTable) could smuggle a typed-nil
// *TableContent through the same constructor.
//
// Expected behaviour: nil nested content is dropped at construction
// (consistent with SectionContent.AddContent), and section methods plus
// renderers tolerate malformed sections without panicking.

// TestNewCollapsibleSectionDropsNilContent verifies the constructor filters
// nil entries out of the nested content instead of storing them.
func TestNewCollapsibleSectionDropsNilContent(t *testing.T) {
	tests := map[string]struct {
		content []Content
		want    int
	}{
		"nil slice": {
			content: nil,
			want:    0,
		},
		"single nil entry": {
			content: []Content{nil},
			want:    0,
		},
		"nil entries around valid content": {
			content: []Content{nil, NewTextContent("kept"), nil},
			want:    1,
		},
		"all valid content": {
			content: []Content{NewTextContent("a"), NewTextContent("b")},
			want:    2,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			section := NewCollapsibleSection("section", tt.content)
			if got := len(section.Content()); got != tt.want {
				t.Errorf("len(Content()) = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestCollapsibleTableHelpersDropNilTables verifies the table helpers drop
// nil *TableContent values instead of wrapping them into typed-nil interface
// entries that later dereferences cannot detect.
func TestCollapsibleTableHelpersDropNilTables(t *testing.T) {
	validTable, err := NewTableContent("data", []map[string]any{{"k": "v"}}, WithKeys("k"))
	if err != nil {
		t.Fatalf("NewTableContent() error = %v", err)
	}

	t.Run("NewCollapsibleTable with nil table", func(t *testing.T) {
		section := NewCollapsibleTable("empty", nil)
		if got := len(section.Content()); got != 0 {
			t.Errorf("len(Content()) = %d, want 0", got)
		}
		// Clone previously panicked on the typed-nil entry.
		if clone := section.Clone(); clone == nil {
			t.Error("Clone() = nil, want non-nil")
		}
	})

	t.Run("NewCollapsibleMultiTable with nil tables", func(t *testing.T) {
		section := NewCollapsibleMultiTable("mixed", []*TableContent{nil, validTable, nil})
		if got := len(section.Content()); got != 1 {
			t.Errorf("len(Content()) = %d, want 1", got)
		}
	})

	t.Run("Builder AddCollapsibleTable with nil table", func(t *testing.T) {
		doc := New().AddCollapsibleTable("empty", nil).Build()
		if got := len(doc.GetContents()); got != 1 {
			t.Fatalf("len(GetContents()) = %d, want 1", got)
		}
	})
}

// TestCollapsibleDelegationRoutesDropNilContent verifies the entry points
// that delegate to NewCollapsibleSection inherit its nil filter, as the
// changelog claims: NewCollapsibleReport and Builder.AddCollapsibleSection.
func TestCollapsibleDelegationRoutesDropNilContent(t *testing.T) {
	t.Run("NewCollapsibleReport", func(t *testing.T) {
		section := NewCollapsibleReport("report", []Content{nil, NewTextContent("kept"), nil})
		if got := len(section.Content()); got != 1 {
			t.Errorf("len(Content()) = %d, want 1", got)
		}
	})

	t.Run("Builder AddCollapsibleSection", func(t *testing.T) {
		doc := New().AddCollapsibleSection("section", []Content{nil}).Build()
		contents := doc.GetContents()
		if len(contents) != 1 {
			t.Fatalf("len(GetContents()) = %d, want 1", len(contents))
		}
		section, ok := contents[0].(*DefaultCollapsibleSection)
		if !ok {
			t.Fatalf("content type = %T, want *DefaultCollapsibleSection", contents[0])
		}
		if got := len(section.Content()); got != 0 {
			t.Errorf("len(Content()) = %d, want 0", got)
		}
	})
}

// TestCollapsibleSectionNilContentMethodsNoPanic verifies AppendText and
// Clone tolerate a malformed section whose content slice contains nil
// entries. The section is built via a struct literal because the constructor
// now filters nils; these are the defence-in-depth guards.
func TestCollapsibleSectionNilContentMethodsNoPanic(t *testing.T) {
	section := &DefaultCollapsibleSection{
		id:          "malformed",
		title:       "malformed",
		content:     []Content{nil, NewTextContent("kept")},
		formatHints: make(map[string]map[string]any),
	}

	t.Run("AppendText", func(t *testing.T) {
		got, err := section.AppendText(nil)
		if err != nil {
			t.Fatalf("AppendText() error = %v, want nil", err)
		}
		if len(got) == 0 {
			t.Error("AppendText() produced no output, want section text")
		}
	})

	t.Run("Clone", func(t *testing.T) {
		clone := section.Clone()
		cloned, ok := clone.(*DefaultCollapsibleSection)
		if !ok {
			t.Fatalf("Clone() type = %T, want *DefaultCollapsibleSection", clone)
		}
		if got := len(cloned.Content()); got != 1 {
			t.Errorf("len(Clone().Content()) = %d, want 1 (nil dropped)", got)
		}
	})
}

// TestRenderCollapsibleSectionNilContent is the ticket's reproduction: a
// section constructed with a nil entry must render through every public
// format path and clone without panicking.
func TestRenderCollapsibleSectionNilContent(t *testing.T) {
	section := NewCollapsibleSection("bad", []Content{nil}, WithSectionExpanded(true))
	doc := New().AddContent(section).Build()

	formats := map[string]Format{
		"markdown": Markdown(),
		"html":     HTML(),
		"json":     JSON(),
		"yaml":     YAML(),
		"csv":      CSV(),
		"table":    Table(),
	}

	for name, format := range formats {
		t.Run(name, func(t *testing.T) {
			if _, err := format.Renderer.Render(context.Background(), doc); err != nil {
				t.Errorf("Render() error = %v, want nil", err)
			}
		})
	}

	if clone := section.Clone(); clone == nil {
		t.Error("Clone() = nil, want non-nil")
	}
}

// TestCSVCollapsibleContentNumberingSkipsNilEntries verifies the CSV
// renderer numbers "# Content N" metadata rows by rendered entries, not the
// raw slice index, so a nil-skipped entry (T-1472 malformed section) cannot
// produce gaps like "Content 1", "Content 3".
func TestCSVCollapsibleContentNumberingSkipsNilEntries(t *testing.T) {
	section := &DefaultCollapsibleSection{
		id:              "malformed",
		title:           "malformed",
		content:         []Content{nil, NewTextContent("first"), NewTextContent("second")},
		defaultExpanded: true,
		formatHints:     make(map[string]map[string]any),
	}
	doc := New().AddContent(section).Build()

	output, err := CSV().Renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}

	got := string(output)
	for _, want := range []string{"# Content 1: text", "# Content 2: text"} {
		if !strings.Contains(got, want) {
			t.Errorf("Render() output missing %q\noutput:\n%s", want, got)
		}
	}
	if strings.Contains(got, "# Content 3:") {
		t.Errorf("Render() output contains %q, want contiguous numbering\noutput:\n%s", "# Content 3:", got)
	}
}

// TestRenderersTolerateMalformedCollapsibleSection injects a section whose
// content slice holds a nil entry (bypassing the constructor filter via a
// struct literal) and verifies every renderer's collapsible path skips the
// nil instead of panicking, still rendering the valid sibling content.
func TestRenderersTolerateMalformedCollapsibleSection(t *testing.T) {
	section := &DefaultCollapsibleSection{
		id:              "malformed",
		title:           "malformed",
		content:         []Content{nil, NewTextContent("survivor")},
		defaultExpanded: true,
		formatHints:     make(map[string]map[string]any),
	}
	doc := New().AddContent(section).Build()

	formats := map[string]Format{
		"markdown": Markdown(),
		"html":     HTML(),
		"json":     JSON(),
		"yaml":     YAML(),
		"csv":      CSV(),
		"table":    Table(),
	}

	for name, format := range formats {
		t.Run(name, func(t *testing.T) {
			output, err := format.Renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render() error = %v, want nil", err)
			}
			if len(output) == 0 {
				t.Error("Render() produced no output, want section output")
			}
		})
	}
}
