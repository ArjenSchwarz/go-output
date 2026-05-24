package output

import "slices"

// Schema defines table structure with explicit key ordering
type Schema struct {
	Fields   []Field
	keyOrder []string // Preserves exact key order
}

// Field defines a table field
type Field struct {
	Name      string
	Type      string
	Formatter func(any) any // Enhanced to support CollapsibleValue returns
	Hidden    bool
}

// GetKeyOrder returns a copy of the preserved key order for the schema.
// A copy is returned so callers cannot mutate the schema's internal key order.
func (s *Schema) GetKeyOrder() []string {
	if s.keyOrder != nil {
		return slices.Clone(s.keyOrder)
	}
	// If keyOrder is not set, extract from fields
	return extractKeyOrder(s.Fields)
}

// SetKeyOrder explicitly sets the key order for the schema.
// The provided slice is cloned so later caller mutations cannot change the schema.
func (s *Schema) SetKeyOrder(keys []string) {
	s.keyOrder = slices.Clone(keys)
}

// extractKeyOrder preserves the exact order of fields
func extractKeyOrder(fields []Field) []string {
	keys := make([]string, 0, len(fields))
	for _, f := range fields {
		if !f.Hidden {
			keys = append(keys, f.Name)
		}
	}
	return keys
}

// NewSchemaFromFields creates a schema from field definitions with preserved order.
// The provided slice is cloned so later caller mutations cannot change the schema.
func NewSchemaFromFields(fields []Field) *Schema {
	return &Schema{
		Fields:   slices.Clone(fields),
		keyOrder: extractKeyOrder(fields),
	}
}

// NewSchemaFromKeys creates a schema from simple key list.
// The provided slice is cloned so later caller mutations cannot change the schema.
func NewSchemaFromKeys(keys []string) *Schema {
	fields := make([]Field, len(keys))
	for i, key := range keys {
		fields[i] = Field{Name: key}
	}
	return &Schema{
		Fields:   fields,
		keyOrder: slices.Clone(keys),
	}
}

// FindField looks up a field by name
func (s *Schema) FindField(name string) *Field {
	for i := range s.Fields {
		if s.Fields[i].Name == name {
			return &s.Fields[i]
		}
	}
	return nil
}

// HasField checks if a field exists in the schema
func (s *Schema) HasField(name string) bool {
	return s.FindField(name) != nil
}

// VisibleFieldCount returns the number of non-hidden fields
func (s *Schema) VisibleFieldCount() int {
	count := 0
	for _, f := range s.Fields {
		if !f.Hidden {
			count++
		}
	}
	return count
}

// GetFieldNames returns the field names in their preserved order
func (s *Schema) GetFieldNames() []string {
	return s.GetKeyOrder()
}

// clone returns a defensive copy of the schema with independent slices.
// Field values are copied (formatters are shared, as functions are immutable).
func (s *Schema) clone() *Schema {
	if s == nil {
		return nil
	}
	return &Schema{
		Fields:   slices.Clone(s.Fields),
		keyOrder: slices.Clone(s.keyOrder),
	}
}
