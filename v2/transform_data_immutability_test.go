package output

import (
	"errors"
	"testing"
)

// Regression tests for T-1677: TableContent.Transform mutates built documents.
//
// Documents are documented as immutable after Build(), but Transform used to
// mutate tc.records in place on content shared with a built document
// (Document.GetContents returns the same Content pointers). These tests verify
// that Transform refuses to mutate content attached to a document, that
// Clone() remains the sanctioned escape hatch, and that a failed transform
// cannot corrupt a table's records.

// doubleValues is a TransformFunc that doubles the "value" column in place.
func doubleValues(data any) (any, error) {
	records, ok := data.([]Record)
	if !ok {
		return nil, errors.New("unexpected data type")
	}
	for i := range records {
		if v, ok := records[i]["value"].(int); ok {
			records[i]["value"] = v * 2
		}
	}
	return records, nil
}

func TestTransformCannotMutateBuiltDocument(t *testing.T) {
	buildDoc := func(t *testing.T) (*Document, *TableContent) {
		t.Helper()
		doc := New().
			Table("test", []Record{
				{"value": 10},
				{"value": 20},
			}, WithKeys("value")).
			Build()
		table, ok := doc.GetContents()[0].(*TableContent)
		if !ok {
			t.Fatalf("expected *TableContent, got %T", doc.GetContents()[0])
		}
		return doc, table
	}

	assertUnchanged := func(t *testing.T, table *TableContent) {
		t.Helper()
		records := table.Records()
		if got, want := records[0]["value"], 10; got != want {
			t.Errorf("document record 0 value = %v, want %v (document was mutated after Build)", got, want)
		}
		if got, want := records[1]["value"], 20; got != want {
			t.Errorf("document record 1 value = %v, want %v (document was mutated after Build)", got, want)
		}
	}

	t.Run("Transform on table from built document returns error", func(t *testing.T) {
		doc, table := buildDoc(t)

		err := table.Transform(doubleValues)
		if err == nil {
			t.Error("Transform() on content attached to a document should return an error, got nil")
		}

		// The document must render the original data regardless.
		_ = doc
		assertUnchanged(t, table)
	})

	t.Run("Transform on table nested in section of built document returns error", func(t *testing.T) {
		doc := New().
			Section("section", func(b *Builder) {
				b.Table("test", []Record{{"value": 10}, {"value": 20}}, WithKeys("value"))
			}).
			Build()

		section, ok := doc.GetContents()[0].(*SectionContent)
		if !ok {
			t.Fatalf("expected *SectionContent, got %T", doc.GetContents()[0])
		}
		table, ok := section.contents[0].(*TableContent)
		if !ok {
			t.Fatalf("expected *TableContent, got %T", section.contents[0])
		}

		if err := table.Transform(doubleValues); err == nil {
			t.Error("Transform() on nested content attached to a document should return an error, got nil")
		}
		assertUnchanged(t, table)
	})

	t.Run("Transform on table inside collapsible section returns error", func(t *testing.T) {
		inner, err := NewTableContent("test", []Record{{"value": 10}, {"value": 20}}, WithKeys("value"))
		if err != nil {
			t.Fatalf("NewTableContent() error = %v", err)
		}

		doc := New().
			AddCollapsibleTable("collapsed", inner).
			Build()

		if err := inner.Transform(doubleValues); err == nil {
			t.Error("Transform() on table wrapped in a document's collapsible section should return an error, got nil")
		}
		_ = doc
		assertUnchanged(t, inner)
	})

	t.Run("Clone of document table remains transformable", func(t *testing.T) {
		_, table := buildDoc(t)

		clone, ok := table.Clone().(*TableContent)
		if !ok {
			t.Fatalf("expected *TableContent clone, got %T", table.Clone())
		}

		if err := clone.Transform(doubleValues); err != nil {
			t.Fatalf("Transform() on clone should succeed, got error: %v", err)
		}

		cloneRecords := clone.Records()
		if got, want := cloneRecords[0]["value"], 20; got != want {
			t.Errorf("clone record 0 value = %v, want %v", got, want)
		}

		// The original document content must be untouched.
		assertUnchanged(t, table)
	})

	t.Run("standalone table remains transformable", func(t *testing.T) {
		table, err := NewTableContent("test", []Record{{"value": 10}}, WithKeys("value"))
		if err != nil {
			t.Fatalf("NewTableContent() error = %v", err)
		}

		if err := table.Transform(doubleValues); err != nil {
			t.Fatalf("Transform() on standalone table should succeed, got error: %v", err)
		}
		if got, want := table.Records()[0]["value"], 20; got != want {
			t.Errorf("standalone record value = %v, want %v", got, want)
		}
	})

	t.Run("failed transform leaves table unchanged", func(t *testing.T) {
		table, err := NewTableContent("test", []Record{{"value": 10}}, WithKeys("value"))
		if err != nil {
			t.Fatalf("NewTableContent() error = %v", err)
		}

		transformErr := table.Transform(func(data any) (any, error) {
			// Mutate the records we were handed, then fail. The mutation must
			// not leak into the table: Transform hands the function a copy.
			records := data.([]Record)
			records[0]["value"] = 999
			return nil, errors.New("boom")
		})
		if transformErr == nil {
			t.Fatal("Transform() should propagate the function error, got nil")
		}

		if got, want := table.Records()[0]["value"], 10; got != want {
			t.Errorf("record value after failed transform = %v, want %v (partial mutation leaked)", got, want)
		}
	})
}

// TestAddContentTypedNilDoesNotPanic verifies that sealContents tolerates
// typed-nil content pointers. Builder.AddContent's nil check only catches an
// untyped nil interface, so a typed nil like (*SectionContent)(nil) reaches
// sealContents, which must not dereference it (regression: the initial T-1677
// sealing walk panicked here, while main stored the typed nil untouched).
func TestAddContentTypedNilDoesNotPanic(t *testing.T) {
	tests := map[string]Content{
		"typed-nil *TableContent":              (*TableContent)(nil),
		"typed-nil *SectionContent":            (*SectionContent)(nil),
		"typed-nil *DefaultCollapsibleSection": (*DefaultCollapsibleSection)(nil),
		"typed-nil table nested in collapsible section": NewCollapsibleSection(
			"section", []Content{(*TableContent)(nil)},
		),
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			New().AddContent(content)
		})
	}
}
