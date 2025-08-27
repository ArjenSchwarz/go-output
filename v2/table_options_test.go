package output

import (
	"reflect"
	"testing"
)

func TestWithSchema(t *testing.T) {
	fields := []Field{
		{Name: "ID", Type: "int"},
		{Name: "Name", Type: "string"},
		{Name: "Email", Type: "string"},
	}

	tc := &tableConfig{}
	opt := WithSchema(fields...)
	opt(tc)

	// Verify schema was set
	if tc.schema == nil {
		t.Fatal("Expected schema to be set")
	}

	// Verify fields match
	if len(tc.schema.Fields) != len(fields) {
		t.Errorf("Schema fields count = %d, want %d", len(tc.schema.Fields), len(fields))
	}

	// Verify key order is preserved
	expectedOrder := []string{"ID", "Name", "Email"}
	if !reflect.DeepEqual(tc.schema.keyOrder, expectedOrder) {
		t.Errorf("Key order = %v, want %v", tc.schema.keyOrder, expectedOrder)
	}

	// Verify autoSchema is disabled
	if tc.autoSchema {
		t.Error("Expected autoSchema to be false when using WithSchema")
	}
}

func TestWithKeys(t *testing.T) {
	keys := []string{"Name", "Age", "Status"}

	tc := &tableConfig{}
	opt := WithKeys(keys...)
	opt(tc)

	// Verify keys were set
	if !reflect.DeepEqual(tc.keys, keys) {
		t.Errorf("Keys = %v, want %v", tc.keys, keys)
	}

	// Verify autoSchema is disabled
	if tc.autoSchema {
		t.Error("Expected autoSchema to be false when using WithKeys")
	}
}

func TestWithAutoSchema(t *testing.T) {
	tc := &tableConfig{}
	opt := WithAutoSchema()
	opt(tc)

	// Verify autoSchema is enabled
	if !tc.autoSchema {
		t.Error("Expected autoSchema to be true")
	}

	// Verify detectOrder is enabled
	if !tc.detectOrder {
		t.Error("Expected detectOrder to be true")
	}
}

func TestWithAutoSchemaOrdered(t *testing.T) {
	keys := []string{"Z", "A", "M"}

	tc := &tableConfig{}
	opt := WithAutoSchemaOrdered(keys...)
	opt(tc)

	// Verify autoSchema is enabled
	if !tc.autoSchema {
		t.Error("Expected autoSchema to be true")
	}

	// Verify keys were set
	if !reflect.DeepEqual(tc.keys, keys) {
		t.Errorf("Keys = %v, want %v", tc.keys, keys)
	}

	// Verify detectOrder is disabled
	if tc.detectOrder {
		t.Error("Expected detectOrder to be false when using custom order")
	}
}

func TestDetectSchemaFromData(t *testing.T) {
	tests := map[string]struct {
		data          any
		expectedKeys  []string
		expectedTypes []string
	}{"empty slice": {

		data:          []map[string]any{},
		expectedKeys:  []string{},
		expectedTypes: []string{},
	}, "single map": {

		data:          map[string]any{"id": int64(1), "value": 3.14, "data": nil},
		expectedKeys:  []string{"id", "value", "data"},
		expectedTypes: []string{"int", "float", "nil"},
	}, "slice of interface with map": {

		data: []any{
			map[string]any{"key": "value"},
		},
		expectedKeys:  []string{"key"},
		expectedTypes: []string{"string"},
	}, "slice of maps": {

		data: []map[string]any{
			{"name": "Alice", "age": 30, "active": true},
			{"name": "Bob", "age": 25, "active": false},
		},
		expectedKeys:  []string{"name", "age", "active"},
		expectedTypes: []string{"string", "int", "bool"},
	}, "unsupported type": {

		data:          "not a map or slice",
		expectedKeys:  []string{},
		expectedTypes: []string{},
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			schema := DetectSchemaFromData(tt.data)

			// Extract field names and types
			var keys []string
			var types []string
			for _, f := range schema.Fields {
				keys = append(keys, f.Name)
				types = append(types, f.Type)
			}

			// Note: Map iteration order is not guaranteed, so we need to handle this
			// For testing, we just verify the presence of expected keys/types
			if len(keys) != len(tt.expectedKeys) {
				t.Errorf("Key count = %d, want %d", len(keys), len(tt.expectedKeys))
			}

			// Verify all expected keys are present
			for i, expectedKey := range tt.expectedKeys {
				found := false
				for j, key := range keys {
					if key == expectedKey {
						found = true
						// Also check the type matches
						expectedType := tt.expectedTypes[i]
						actualType := types[j]
						if actualType != expectedType {
							t.Errorf("Type for %s = %s, want %s", key, actualType, expectedType)
						}
						break
					}
				}
				if !found {
					t.Errorf("Expected key %s not found", expectedKey)
				}
			}
		})
	}
}

func TestDetectType(t *testing.T) {
	tests := []struct {
		value    any
		expected string
	}{
		{"hello", "string"},
		{42, "int"},
		{int8(8), "int"},
		{int16(16), "int"},
		{int32(32), "int"},
		{int64(64), "int"},
		{uint(42), "uint"},
		{uint8(8), "uint"},
		{uint16(16), "uint"},
		{uint32(32), "uint"},
		{uint64(64), "uint"},
		{3.14, "float"},
		{float32(3.14), "float"},
		{true, "bool"},
		{false, "bool"},
		{nil, "nil"},
		{struct{}{}, "interface"},
		{[]int{1, 2, 3}, "interface"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := DetectType(tt.value)
			if result != tt.expected {
				t.Errorf("DetectType(%T) = %s, want %s", tt.value, result, tt.expected)
			}
		})
	}
}

func TestApplyTableOptions(t *testing.T) {
	// Test with no options (defaults)
	tc := ApplyTableOptions()
	if !tc.autoSchema {
		t.Error("Expected autoSchema to be true by default")
	}

	// Test with multiple options
	keys := []string{"A", "B", "C"}
	fields := []Field{{Name: "X"}, {Name: "Y"}}

	tc = ApplyTableOptions(
		WithKeys(keys...),
		WithSchema(fields...),
		WithAutoSchema(),
	)

	// Last option should win for conflicting settings
	if !tc.autoSchema {
		t.Error("Expected autoSchema to be true (last option)")
	}
	if !tc.detectOrder {
		t.Error("Expected detectOrder to be true")
	}

	// Schema should be from WithSchema (applied before WithAutoSchema)
	if tc.schema == nil {
		t.Error("Expected schema to be set")
	}
}

func TestOptionsPreserveKeyOrder(t *testing.T) {
	// Test that all option methods preserve key order as specified

	// Test 1: WithSchema preserves field order
	fields := []Field{
		{Name: "Zebra"},
		{Name: "Apple"},
		{Name: "Mango"},
	}
	tc1 := &tableConfig{}
	WithSchema(fields...)(tc1)

	expectedOrder := []string{"Zebra", "Apple", "Mango"}
	if !reflect.DeepEqual(tc1.schema.keyOrder, expectedOrder) {
		t.Errorf("WithSchema key order = %v, want %v", tc1.schema.keyOrder, expectedOrder)
	}

	// Test 2: WithKeys preserves key order
	keys := []string{"Z", "A", "M", "B"}
	tc2 := &tableConfig{}
	WithKeys(keys...)(tc2)

	if !reflect.DeepEqual(tc2.keys, keys) {
		t.Errorf("WithKeys order = %v, want %v", tc2.keys, keys)
	}

	// Test 3: WithAutoSchemaOrdered preserves key order
	orderedKeys := []string{"Third", "First", "Second"}
	tc3 := &tableConfig{}
	WithAutoSchemaOrdered(orderedKeys...)(tc3)

	if !reflect.DeepEqual(tc3.keys, orderedKeys) {
		t.Errorf("WithAutoSchemaOrdered keys = %v, want %v", tc3.keys, orderedKeys)
	}
}
