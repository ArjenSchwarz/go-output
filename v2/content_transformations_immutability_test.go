package output

import (
	"bytes"
	"context"
	"testing"
)

// Regression tests for T-1378: GetTransformations exposed the internal
// []Operation slice, and the text/raw/section transformation options stored
// the caller's variadic slice directly. Both let callers change which
// transformations a built document runs.

// transformationContentConstructors builds one content value of each type
// that stores transformations, attaching the given operations through the
// type's option function.
func transformationContentConstructors() map[string]func(t *testing.T, ops ...Operation) Content {
	return map[string]func(t *testing.T, ops ...Operation) Content{
		"table": func(t *testing.T, ops ...Operation) Content {
			t.Helper()
			table, err := NewTableContent("t", []map[string]any{{"name": "Alice"}},
				WithKeys("name"), WithTransformations(ops...))
			if err != nil {
				t.Fatalf("NewTableContent() error = %v", err)
			}
			return table
		},
		"text": func(t *testing.T, ops ...Operation) Content {
			t.Helper()
			return NewTextContent("text", WithTextTransformations(ops...))
		},
		"raw": func(t *testing.T, ops ...Operation) Content {
			t.Helper()
			raw, err := NewRawContent(FormatHTML, []byte("<b>x</b>"), WithRawTransformations(ops...))
			if err != nil {
				t.Fatalf("NewRawContent() error = %v", err)
			}
			return raw
		},
		"section": func(t *testing.T, ops ...Operation) Content {
			t.Helper()
			return NewSectionContent("section", WithSectionTransformations(ops...))
		},
	}
}

func operationNames(ops []Operation) []string {
	names := make([]string, 0, len(ops))
	for _, op := range ops {
		if op == nil {
			names = append(names, "<nil>")
			continue
		}
		names = append(names, op.Name())
	}
	return names
}

// TestGetTransformations_ReturnsDefensiveCopy verifies that replacing an
// entry in the slice returned by GetTransformations does not change the
// transformations stored on the content.
func TestGetTransformations_ReturnsDefensiveCopy(t *testing.T) {
	for name, newContent := range transformationContentConstructors() {
		t.Run(name, func(t *testing.T) {
			content := newContent(t, &mockOperation{name: "original"})

			got := content.GetTransformations()
			if len(got) != 1 {
				t.Fatalf("GetTransformations() len = %d, want 1", len(got))
			}
			got[0] = &mockOperation{name: "replaced"}

			after := content.GetTransformations()
			if len(after) != 1 {
				t.Fatalf("GetTransformations() len after mutation = %d, want 1", len(after))
			}
			if gotName, wantName := after[0].Name(), "original"; gotName != wantName {
				t.Errorf("GetTransformations()[0].Name() after mutating returned slice = %q, want %q", gotName, wantName)
			}
		})
	}
}

// TestGetTransformations_AbsentReturnsEmptyNonNil locks in the documented
// contract that content without transformations yields an empty, non-nil
// slice rather than nil.
func TestGetTransformations_AbsentReturnsEmptyNonNil(t *testing.T) {
	for name, newContent := range transformationContentConstructors() {
		t.Run(name, func(t *testing.T) {
			content := newContent(t)

			got := content.GetTransformations()
			if got == nil {
				t.Fatal("GetTransformations() = nil, want empty non-nil slice")
			}
			if len(got) != 0 {
				t.Errorf("GetTransformations() len = %d, want 0", len(got))
			}
		})
	}
}

// TestTransformationOptions_CopyCallerSlice verifies that mutating the slice
// passed to a transformation option after construction does not change the
// content's transformations.
func TestTransformationOptions_CopyCallerSlice(t *testing.T) {
	for name, newContent := range transformationContentConstructors() {
		t.Run(name, func(t *testing.T) {
			ops := []Operation{&mockOperation{name: "original"}}
			content := newContent(t, ops...)

			ops[0] = &mockOperation{name: "replaced"}

			got := content.GetTransformations()
			if len(got) != 1 {
				t.Fatalf("GetTransformations() len = %d, want 1", len(got))
			}
			if gotName, wantName := got[0].Name(), "original"; gotName != wantName {
				t.Errorf("GetTransformations()[0].Name() after mutating caller slice = %q, want %q", gotName, wantName)
			}
		})
	}
}

// TestTransformationOptions_DropNilOperations verifies that every
// transformation option filters nil operations the way WithTransformations
// already does for tables (T-1208).
func TestTransformationOptions_DropNilOperations(t *testing.T) {
	tests := map[string]struct {
		ops       []Operation
		wantNames []string
	}{
		"single nil": {
			ops:       []Operation{nil},
			wantNames: []string{},
		},
		"nil mixed with operations": {
			ops:       []Operation{nil, &mockOperation{name: "first"}, nil, &mockOperation{name: "second"}, nil},
			wantNames: []string{"first", "second"},
		},
	}

	for typeName, newContent := range transformationContentConstructors() {
		for name, tc := range tests {
			t.Run(typeName+"/"+name, func(t *testing.T) {
				content := newContent(t, tc.ops...)

				got := operationNames(content.GetTransformations())
				if len(got) != len(tc.wantNames) {
					t.Fatalf("GetTransformations() names = %v, want %v", got, tc.wantNames)
				}
				for i := range got {
					if got[i] != tc.wantNames[i] {
						t.Errorf("GetTransformations()[%d].Name() = %q, want %q", i, got[i], tc.wantNames[i])
					}
				}
			})
		}
	}
}

// TestBuiltDocument_TransformationsCannotBeSwapped is the end-to-end shape of
// the bug: swapping an operation in the slice obtained from a built document
// must not change what later renders produce.
func TestBuiltDocument_TransformationsCannotBeSwapped(t *testing.T) {
	data := []map[string]any{
		{"name": "Alice", "active": true},
		{"name": "Bob", "active": false},
	}
	onlyActive := NewFilterOp(func(r Record) bool { return r["active"] == true })
	doc := New().
		Table("users", data, WithKeys("name", "active"), WithTransformations(onlyActive)).
		Build()

	renderer := &jsonRenderer{}
	before, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render() before mutation error = %v", err)
	}
	if bytes.Contains(before, []byte("Bob")) {
		t.Fatalf("Render() before mutation = %s, want Bob filtered out", before)
	}

	ops := doc.GetContents()[0].GetTransformations()
	if len(ops) != 1 {
		t.Fatalf("GetTransformations() len = %d, want 1", len(ops))
	}
	ops[0] = NewFilterOp(func(Record) bool { return true })

	after, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render() after mutation error = %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("Render() after swapping returned operation = %s, want unchanged output %s", after, before)
	}
}
