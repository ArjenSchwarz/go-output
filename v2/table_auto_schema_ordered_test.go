package output

import (
	"reflect"
	"testing"
)

// Regression tests for T-1451: WithAutoSchemaOrdered skips schema detection.
//
// WithAutoSchemaOrdered is documented as enabling automatic schema detection
// with a custom key order. The option sets both autoSchema and keys, but the
// option-precedence switch in newTableContent checked len(tc.keys) > 0 before
// tc.autoSchema, so the schema was built from the explicit keys alone and
// DetectSchemaFromData was never called. Any data field not listed in the
// custom order was silently dropped from the schema and the rendered output.
//
// Fixed behaviour: detection runs over the data (union of columns across all
// rows, per T-1576) and the resulting columns are ordered as explicit keys
// first (in the given order), with the remaining detected fields appended
// alphabetically (the deterministic detection order, per T-1692).

func TestWithAutoSchemaOrdered_SchemaDetection(t *testing.T) {
	tests := map[string]struct {
		data any
		keys []string
		want []string
	}{
		"ticket reproduction: unlisted field is detected": {
			data: []map[string]any{{"name": "Alice", "age": 30}},
			keys: []string{"name"},
			want: []string{"name", "age"},
		},
		"explicit keys first then remainder alphabetical": {
			data: []map[string]any{{"delta": 1, "bravo": 2, "alpha": 3, "charlie": 4}},
			keys: []string{"delta", "bravo"},
			want: []string{"delta", "bravo", "alpha", "charlie"},
		},
		"all fields listed keeps given order": {
			data: []map[string]any{{"zebra": 1, "apple": 2, "mango": 3}},
			keys: []string{"zebra", "apple", "mango"},
			want: []string{"zebra", "apple", "mango"},
		},
		"explicit key missing from data is kept like WithKeys": {
			data: []map[string]any{{"alpha": 1, "bravo": 2}},
			keys: []string{"missing"},
			want: []string{"missing", "alpha", "bravo"},
		},
		"union of columns across rows is detected (T-1576)": {
			data: []map[string]any{
				{"name": "Alice"},
				{"name": "Bob", "age": 25},
			},
			keys: []string{"name"},
			want: []string{"name", "age"},
		},
		"duplicate explicit keys are deduplicated": {
			data: []map[string]any{{"name": "Alice", "age": 30}},
			keys: []string{"name", "name"},
			want: []string{"name", "age"},
		},
		"Record slice input": {
			data: []Record{{"name": "Alice", "age": 30, "city": "Sydney"}},
			keys: []string{"city"},
			want: []string{"city", "age", "name"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			table, err := NewTableContent("test", tt.data, WithAutoSchemaOrdered(tt.keys...))
			if err != nil {
				t.Fatalf("NewTableContent() error = %v", err)
			}

			got := table.Schema().GetKeyOrder()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("key order = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestWithAutoSchemaOrdered_PreservesDetectedTypes verifies that explicitly
// listed keys keep the field type detected from the data instead of being
// replaced by untyped placeholder fields.
func TestWithAutoSchemaOrdered_PreservesDetectedTypes(t *testing.T) {
	table, err := NewTableContent("users",
		[]map[string]any{{"name": "Alice", "age": 30, "active": true}},
		WithAutoSchemaOrdered("age", "name"))
	if err != nil {
		t.Fatalf("NewTableContent() error = %v", err)
	}

	schema := table.Schema()
	wantTypes := map[string]string{
		"age":    "int",
		"name":   "string",
		"active": "bool",
	}
	for fieldName, wantType := range wantTypes {
		field := schema.FindField(fieldName)
		if field == nil {
			t.Fatalf("FindField(%q) = nil, want field present", fieldName)
		}
		if field.Type != wantType {
			t.Errorf("field %q type = %q, want %q", fieldName, field.Type, wantType)
		}
	}
}

// TestWithAutoSchemaOrdered_ZeroKeys locks in the documented degradation:
// with no keys there is no ordering contract, so the option behaves exactly
// like WithAutoSchema — detection runs and all columns are alphabetized.
// (The matching warning semantics — ErrTableKeyOrderGuessed IS recorded in
// this case — are asserted in TestBuilderTable_KeyOrderGuessWarning.)
func TestWithAutoSchemaOrdered_ZeroKeys(t *testing.T) {
	table, err := NewTableContent("test",
		[]map[string]any{{"zebra": 1, "apple": 2, "mango": 3}},
		WithAutoSchemaOrdered())
	if err != nil {
		t.Fatalf("NewTableContent() error = %v", err)
	}

	got := table.Schema().GetKeyOrder()
	want := []string{"apple", "mango", "zebra"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("key order = %v, want %v (alphabetical, like WithAutoSchema)", got, want)
	}
}

// TestWithAutoSchemaOrdered_DoesNotRetainCallerSlice verifies the option
// clones the caller's key slice, matching the defensive-copy convention of
// WithKeys and WithSchema (T-1086).
func TestWithAutoSchemaOrdered_DoesNotRetainCallerSlice(t *testing.T) {
	keys := []string{"name"}
	table, err := NewTableContent("users",
		[]map[string]any{{"name": "Alice", "age": 30}},
		WithAutoSchemaOrdered(keys...))
	if err != nil {
		t.Fatalf("NewTableContent() error = %v", err)
	}

	// Mutate the caller's backing array after building the table.
	keys[0] = "MUTATED"

	got := table.Schema().GetKeyOrder()
	want := []string{"name", "age"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("after mutating caller slice, key order = %v, want %v", got, want)
	}
}

// TestTableOptionPrecedence locks in the option-precedence switch in
// newTableContent so its cases cannot be accidentally reordered: an explicit
// schema always wins, detection-with-keys comes next, then keys alone. It
// also pins the documented side effect of the T-1451 fix: because the options
// mutate shared tableConfig state, combining WithKeys(...) with a later
// WithAutoSchema() behaves like WithAutoSchemaOrdered(...) instead of the
// trailing WithAutoSchema() being silently ignored (see CHANGELOG).
func TestTableOptionPrecedence(t *testing.T) {
	data := []map[string]any{{"name": "Alice", "zebra": 1, "apple": 2}}

	tests := map[string]struct {
		opts []TableOption
		want []string
	}{
		"WithSchema wins over later WithAutoSchemaOrdered": {
			opts: []TableOption{
				WithSchema(Field{Name: "name"}),
				WithAutoSchemaOrdered("zebra"),
			},
			want: []string{"name"},
		},
		"WithSchema wins over earlier WithAutoSchemaOrdered": {
			opts: []TableOption{
				WithAutoSchemaOrdered("zebra"),
				WithSchema(Field{Name: "name"}),
			},
			want: []string{"name"},
		},
		"WithKeys then WithAutoSchema behaves like WithAutoSchemaOrdered": {
			opts: []TableOption{
				WithKeys("name"),
				WithAutoSchema(),
			},
			want: []string{"name", "apple", "zebra"},
		},
		"WithAutoSchema then WithKeys behaves like WithKeys alone": {
			// WithKeys clears autoSchema, so no detection runs and unlisted
			// data columns are dropped — plain WithKeys semantics.
			opts: []TableOption{
				WithAutoSchema(),
				WithKeys("name"),
			},
			want: []string{"name"},
		},
		"WithAutoSchemaOrdered then WithKeys behaves like WithKeys alone": {
			opts: []TableOption{
				WithAutoSchemaOrdered("zebra"),
				WithKeys("name"),
			},
			want: []string{"name"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			table, err := NewTableContent("test", data, tt.opts...)
			if err != nil {
				t.Fatalf("NewTableContent() error = %v", err)
			}

			got := table.Schema().GetKeyOrder()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("key order = %v, want %v", got, tt.want)
			}
		})
	}
}
