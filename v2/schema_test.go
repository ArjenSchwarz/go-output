package output

import (
	"fmt"
	"reflect"
	"testing"
)

func TestSchemaKeyOrderPreservation(t *testing.T) {
	tests := map[string]struct {
		fields   []Field
		expected []string
	}{"all hidden fields": {

		fields: []Field{
			{Name: "Hidden1", Hidden: true},
			{Name: "Hidden2", Hidden: true},
		},
		expected: []string{},
	}, "basic field order": {

		fields: []Field{
			{Name: "Z"},
			{Name: "A"},
			{Name: "M"},
		},
		expected: []string{"Z", "A", "M"},
	}, "complex field order": {

		fields: []Field{
			{Name: "ID", Type: "int"},
			{Name: "Name", Type: "string"},
			{Name: "Email", Type: "string"},
			{Name: "Age", Type: "int"},
			{Name: "Status", Type: "bool"},
		},
		expected: []string{"ID", "Name", "Email", "Age", "Status"},
	}, "empty fields": {

		fields:   []Field{},
		expected: []string{},
	}, "fields with hidden": {

		fields: []Field{
			{Name: "First"},
			{Name: "Hidden", Hidden: true},
			{Name: "Last"},
		},
		expected: []string{"First", "Last"},
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			schema := NewSchemaFromFields(tt.fields)

			// Verify key order is preserved
			keyOrder := schema.GetKeyOrder()
			if !reflect.DeepEqual(keyOrder, tt.expected) {
				t.Errorf("GetKeyOrder() = %v, want %v", keyOrder, tt.expected)
			}

			// Verify extractKeyOrder directly
			extracted := extractKeyOrder(tt.fields)
			if !reflect.DeepEqual(extracted, tt.expected) {
				t.Errorf("extractKeyOrder() = %v, want %v", extracted, tt.expected)
			}
		})
	}
}

func TestSchemaExplicitKeyOrder(t *testing.T) {
	// Create schema with fields in one order
	fields := []Field{
		{Name: "A"},
		{Name: "B"},
		{Name: "C"},
	}
	schema := NewSchemaFromFields(fields)

	// Verify initial order
	if !reflect.DeepEqual(schema.GetKeyOrder(), []string{"A", "B", "C"}) {
		t.Errorf("Initial key order incorrect")
	}

	// Set explicit different order
	newOrder := []string{"C", "A", "B"}
	schema.SetKeyOrder(newOrder)

	// Verify new order is preserved
	if !reflect.DeepEqual(schema.GetKeyOrder(), newOrder) {
		t.Errorf("GetKeyOrder() = %v, want %v", schema.GetKeyOrder(), newOrder)
	}
}

func TestNewSchemaFromKeys(t *testing.T) {
	keys := []string{"Name", "Age", "Email", "Status"}
	schema := NewSchemaFromKeys(keys)

	// Verify fields were created
	if len(schema.Fields) != len(keys) {
		t.Errorf("Fields count = %d, want %d", len(schema.Fields), len(keys))
	}

	// Verify key order is preserved
	if !reflect.DeepEqual(schema.GetKeyOrder(), keys) {
		t.Errorf("GetKeyOrder() = %v, want %v", schema.GetKeyOrder(), keys)
	}

	// Verify field names match keys
	for i, key := range keys {
		if schema.Fields[i].Name != key {
			t.Errorf("Field[%d].Name = %s, want %s", i, schema.Fields[i].Name, key)
		}
	}
}

func TestSchemaFieldLookup(t *testing.T) {
	formatter := func(v any) any {
		return fmt.Sprintf("formatted-%v", v)
	}

	fields := []Field{
		{Name: "ID", Type: "int"},
		{Name: "Name", Type: "string", Formatter: formatter},
		{Name: "Hidden", Hidden: true},
	}
	schema := NewSchemaFromFields(fields)

	tests := []struct {
		name      string
		fieldName string
		found     bool
		fieldType string
		hidden    bool
	}{
		{"existing field", "ID", true, "int", false},
		{"field with formatter", "Name", true, "string", false},
		{"hidden field", "Hidden", true, "", true},
		{"non-existent field", "Missing", false, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := schema.FindField(tt.fieldName)
			hasField := schema.HasField(tt.fieldName)

			if tt.found {
				if field == nil {
					t.Errorf("FindField(%s) returned nil, expected field", tt.fieldName)
				}
				if !hasField {
					t.Errorf("HasField(%s) returned false, expected true", tt.fieldName)
				}
				if field != nil {
					if field.Type != tt.fieldType {
						t.Errorf("Field.Type = %s, want %s", field.Type, tt.fieldType)
					}
					if field.Hidden != tt.hidden {
						t.Errorf("Field.Hidden = %v, want %v", field.Hidden, tt.hidden)
					}
					if tt.fieldName == "Name" && field.Formatter == nil {
						t.Error("Expected formatter to be set for Name field")
					}
				}
			} else {
				if field != nil {
					t.Errorf("FindField(%s) returned field, expected nil", tt.fieldName)
				}
				if hasField {
					t.Errorf("HasField(%s) returned true, expected false", tt.fieldName)
				}
			}
		})
	}
}

func TestSchemaVisibleFieldCount(t *testing.T) {
	tests := map[string]struct {
		fields   []Field
		expected int
	}{"all hidden": {

		fields: []Field{
			{Name: "A", Hidden: true},
			{Name: "B", Hidden: true},
		},
		expected: 0,
	}, "all visible": {

		fields: []Field{
			{Name: "A"},
			{Name: "B"},
			{Name: "C"},
		},
		expected: 3,
	}, "no fields": {

		fields:   []Field{},
		expected: 0,
	}, "some hidden": {

		fields: []Field{
			{Name: "A"},
			{Name: "B", Hidden: true},
			{Name: "C"},
			{Name: "D", Hidden: true},
		},
		expected: 2,
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			schema := NewSchemaFromFields(tt.fields)
			count := schema.VisibleFieldCount()
			if count != tt.expected {
				t.Errorf("VisibleFieldCount() = %d, want %d", count, tt.expected)
			}
		})
	}
}

func TestSchemaKeyOrderConsistency(t *testing.T) {
	// Test that key order is consistent across multiple calls
	fields := []Field{
		{Name: "Zebra"},
		{Name: "Apple"},
		{Name: "Mango"},
		{Name: "Banana"},
	}
	schema := NewSchemaFromFields(fields)

	// Get key order multiple times
	order1 := schema.GetKeyOrder()
	order2 := schema.GetKeyOrder()
	order3 := schema.GetKeyOrder()

	// All should be identical
	if !reflect.DeepEqual(order1, order2) || !reflect.DeepEqual(order2, order3) {
		t.Error("Key order is not consistent across multiple calls")
	}

	// And should match the original field order
	expected := []string{"Zebra", "Apple", "Mango", "Banana"}
	if !reflect.DeepEqual(order1, expected) {
		t.Errorf("Key order = %v, want %v", order1, expected)
	}
}

func TestFormatterFunction(t *testing.T) {
	// Test custom formatter that returns string (backward compatibility)
	upperFormatter := func(v any) any {
		if str, ok := v.(string); ok {
			return "UPPER:" + str
		}
		return fmt.Sprintf("%v", v)
	}

	field := Field{
		Name:      "TestField",
		Type:      "string",
		Formatter: upperFormatter,
	}

	// Test formatter with string
	result := field.Formatter("hello")
	if result != "UPPER:hello" {
		t.Errorf("Formatter result = %v, want UPPER:hello", result)
	}

	// Test formatter with non-string
	result = field.Formatter(123)
	if result != "123" {
		t.Errorf("Formatter result = %v, want 123", result)
	}
}
