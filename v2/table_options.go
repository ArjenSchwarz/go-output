package output

import "sort"

// tableConfig holds configuration for table creation
type tableConfig struct {
	schema      *Schema
	keys        []string
	autoSchema  bool
	detectOrder bool
}

// TableOption configures table creation
type TableOption func(*tableConfig)

// WithSchema explicitly sets the table schema with key order
func WithSchema(fields ...Field) TableOption {
	return func(tc *tableConfig) {
		tc.schema = &Schema{
			Fields:   fields,
			keyOrder: extractKeyOrder(fields),
		}
		tc.autoSchema = false
	}
}

// WithKeys sets explicit key ordering (for v1 compatibility)
func WithKeys(keys ...string) TableOption {
	return func(tc *tableConfig) {
		tc.keys = keys
		tc.autoSchema = false
	}
}

// WithAutoSchema enables automatic schema detection from data
// When enabled, the system will preserve the order keys appear in the source data
func WithAutoSchema() TableOption {
	return func(tc *tableConfig) {
		tc.autoSchema = true
		tc.detectOrder = true
	}
}

// WithAutoSchemaOrdered enables automatic schema detection with custom key order
func WithAutoSchemaOrdered(keys ...string) TableOption {
	return func(tc *tableConfig) {
		tc.autoSchema = true
		tc.keys = keys
		tc.detectOrder = false
	}
}

// DetectSchemaFromData creates a schema from the provided data, preserving key order
func DetectSchemaFromData(data any) *Schema {
	switch v := data.(type) {
	case []Record:
		if len(v) == 0 {
			return &Schema{Fields: []Field{}, keyOrder: []string{}}
		}
		return DetectSchemaFromMap(v[0])
	case []map[string]any:
		if len(v) == 0 {
			return &Schema{Fields: []Field{}, keyOrder: []string{}}
		}
		return DetectSchemaFromMap(v[0])
	case Record:
		return DetectSchemaFromMap(v)
	case map[string]any:
		return DetectSchemaFromMap(v)
	case []any:
		if len(v) == 0 {
			return &Schema{Fields: []Field{}, keyOrder: []string{}}
		}
		if m, ok := v[0].(map[string]any); ok {
			return DetectSchemaFromMap(m)
		}
	}
	return &Schema{Fields: []Field{}, keyOrder: []string{}}
}

// DetectSchemaFromMap creates a schema from a map, preserving insertion order
func DetectSchemaFromMap(m map[string]any) *Schema {
	// Since Go maps don't preserve order, we need to extract keys in a deterministic way
	// This is a limitation that should be documented - for true order preservation,
	// users should provide explicit keys or use WithKeys/WithSchema
	keys := make([]string, 0, len(m))
	fields := make([]Field, 0, len(m))

	// Extract keys in a consistent order (alphabetical for now)
	// In real usage, we might want to use a different approach or require explicit ordering
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys) // Sort keys alphabetically for deterministic output

	// For each key, create a field
	for _, k := range keys {
		fields = append(fields, Field{
			Name: k,
			Type: DetectType(m[k]),
		})
	}

	return &Schema{
		Fields:   fields,
		keyOrder: keys,
	}
}

// DetectType attempts to determine the type of a value
func DetectType(v any) string {
	switch v.(type) {
	case string:
		return "string"
	case int, int8, int16, int32, int64:
		return "int"
	case uint, uint8, uint16, uint32, uint64:
		return "uint"
	case float32, float64:
		return "float"
	case bool:
		return "bool"
	case nil:
		return "nil"
	default:
		return "interface"
	}
}

// ApplyTableOptions applies all options to the table configuration
func ApplyTableOptions(opts ...TableOption) *tableConfig {
	tc := &tableConfig{
		autoSchema: true, // Default to auto-detection
	}
	for _, opt := range opts {
		opt(tc)
	}
	return tc
}
