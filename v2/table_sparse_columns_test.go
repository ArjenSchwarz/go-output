package output

import (
	"reflect"
	"strings"
	"testing"
)

// Regression tests for T-1576: auto schema omits later-only table columns.
//
// DetectSchemaFromData only inspected the first row for []Record,
// []map[string]any, and []any inputs. When later rows contained additional
// columns, the auto-detected schema excluded those keys and renderers
// silently dropped the later-only values. Detection must instead cover the
// union of columns across all rows, keeping the deterministic alphabetical
// ordering established for map inputs (T-1692).

func TestDetectSchemaFromData_UnionOfColumns(t *testing.T) {
	tests := map[string]struct {
		data any
		// wantKeys is the expected key order: the union of all row keys,
		// sorted alphabetically (map order is unrecoverable, see T-1692).
		wantKeys []string
		// wantTypes[i] is the expected type for wantKeys[i], detected from
		// the first row in which the key appears.
		wantTypes []string
	}{
		"map slice with later-only column": {
			data: []map[string]any{
				{"name": "Alice"},
				{"name": "Bob", "email": "bob@example.com"},
			},
			wantKeys:  []string{"email", "name"},
			wantTypes: []string{"string", "string"},
		},
		"record slice with later-only column": {
			data: []Record{
				{"name": "Alice"},
				{"name": "Bob", "email": "bob@example.com"},
			},
			wantKeys:  []string{"email", "name"},
			wantTypes: []string{"string", "string"},
		},
		"any slice with later-only column": {
			data: []any{
				map[string]any{"name": "Alice"},
				map[string]any{"name": "Bob", "email": "bob@example.com"},
			},
			wantKeys:  []string{"email", "name"},
			wantTypes: []string{"string", "string"},
		},
		"disjoint rows": {
			data: []map[string]any{
				{"zebra": 1},
				{"apple": true},
				{"mango": "m"},
			},
			wantKeys:  []string{"apple", "mango", "zebra"},
			wantTypes: []string{"bool", "string", "int"},
		},
		"column type comes from first row containing the key": {
			data: []map[string]any{
				{"id": 1},
				{"id": 2, "score": 3.5},
				{"id": 3, "score": "high"},
			},
			wantKeys:  []string{"id", "score"},
			wantTypes: []string{"int", "float"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			schema := DetectSchemaFromData(tt.data)

			if got := schema.GetKeyOrder(); !reflect.DeepEqual(got, tt.wantKeys) {
				t.Errorf("GetKeyOrder() = %v, want %v", got, tt.wantKeys)
			}

			var gotTypes []string
			for _, f := range schema.Fields {
				gotTypes = append(gotTypes, f.Type)
			}
			if !reflect.DeepEqual(gotTypes, tt.wantTypes) {
				t.Errorf("field types = %v, want %v", gotTypes, tt.wantTypes)
			}
		})
	}
}

// TestNewTableContent_SparseRowsRenderAllColumns is the reproduction from
// T-1576: with default auto schema, a column that first appears in a later
// row must still be rendered rather than silently dropped.
func TestNewTableContent_SparseRowsRenderAllColumns(t *testing.T) {
	table, err := NewTableContent("Sparse", []map[string]any{
		{"name": "Alice"},
		{"name": "Bob", "email": "bob@example.com"},
	})
	if err != nil {
		t.Fatalf("NewTableContent() error = %v", err)
	}

	out, err := table.AppendText(nil)
	if err != nil {
		t.Fatalf("AppendText() error = %v", err)
	}

	got := string(out)
	if !strings.Contains(got, "email") {
		t.Errorf("AppendText() output missing later-only column header %q:\n%s", "email", got)
	}
	if !strings.Contains(got, "bob@example.com") {
		t.Errorf("AppendText() output missing later-only value %q:\n%s", "bob@example.com", got)
	}
}

// TestBuilderTable_SparseRowsKeyOrderWarning verifies that when the union of
// sparse rows yields multiple columns, the alphabetical-order guess is
// reported via ErrTableKeyOrderGuessed even though the first row alone has a
// single column (no order to guess on its own).
func TestBuilderTable_SparseRowsKeyOrderWarning(t *testing.T) {
	builder := New()
	builder.Table("sparse", []map[string]any{
		{"name": "Alice"},
		{"name": "Bob", "email": "bob@example.com"},
	})

	if warnings := keyOrderWarnings(builder.Errors()); len(warnings) == 0 {
		t.Errorf("Errors() = %v, want a key order guess warning for multi-column union", builder.Errors())
	}
}
