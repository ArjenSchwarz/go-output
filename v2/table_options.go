package output

import (
	"slices"
	"sort"
)

// tableConfig holds configuration for table creation
type tableConfig struct {
	schema          *Schema
	keys            []string
	autoSchema      bool
	transformations []Operation
}

// TableOption configures table creation
type TableOption func(*tableConfig)

// WithSchema explicitly sets the table schema with key order.
// The fields are cloned so later caller mutations cannot change the schema.
func WithSchema(fields ...Field) TableOption {
	return func(tc *tableConfig) {
		tc.schema = &Schema{
			Fields:   slices.Clone(fields),
			keyOrder: extractKeyOrder(fields),
		}
		tc.autoSchema = false
	}
}

// WithKeys sets explicit key ordering (for v1 compatibility).
// The keys are cloned so later caller mutations cannot change the key order.
func WithKeys(keys ...string) TableOption {
	return func(tc *tableConfig) {
		tc.keys = slices.Clone(keys)
		tc.autoSchema = false
	}
}

// WithAutoSchema enables automatic schema detection from data. This is also
// the default behavior when no schema or keys are provided.
//
// For slice input the detected columns are the union of keys across all rows,
// so columns that first appear in a later row are included (T-1576).
//
// Map input has no recoverable key order: Go randomizes map iteration order,
// so the order keys appear in the source data cannot be preserved. Detection
// falls back to sorting the column names alphabetically (see
// DetectSchemaFromMap). Use WithKeys or WithSchema to control column order;
// Builder.Table records a non-fatal ErrTableKeyOrderGuessed warning when this
// fallback guessed an order (T-1692).
func WithAutoSchema() TableOption {
	return func(tc *tableConfig) {
		tc.autoSchema = true
	}
}

// WithAutoSchemaOrdered enables automatic schema detection with a custom key
// order. The schema fields are detected from the data exactly as WithAutoSchema
// does (for slice input the columns are the union of keys across all rows,
// T-1576). The listed keys become the first columns, in the order given; keys
// not present in the data are still included as untyped columns, matching
// WithKeys. All remaining detected columns are appended after them in
// alphabetical order — the deterministic detection order (see
// DetectSchemaFromMap) — so unlisted data columns are never dropped (T-1451).
//
// Because the caller explicitly opts into detection under this ordering
// contract, Builder.Table does not record an ErrTableKeyOrderGuessed warning
// for tables built with this option — the alphabetically appended remainder
// is documented behavior, not a guess.
//
// The keys are cloned so later caller mutations cannot change the key order.
func WithAutoSchemaOrdered(keys ...string) TableOption {
	return func(tc *tableConfig) {
		tc.autoSchema = true
		tc.keys = slices.Clone(keys)
	}
}

// WithTransformations attaches one or more operations to the table content.
// Operations execute during rendering in the order specified.
//
// Transformations enable filtering, sorting, limiting, grouping, and other operations
// to be applied to individual tables without affecting other content in the document.
//
// Thread Safety Requirements:
// Operations MUST be stateless and thread-safe. Do not create operations with:
//   - Mutable state modified during Apply()
//   - Closures capturing mutable variables by reference
//   - External side effects (file writes, network calls, etc.)
//
// Example usage:
//
//	builder.Table("users", userData,
//	    output.WithKeys("name", "email", "age"),
//	    output.WithTransformations(
//	        output.NewFilterOp(func(r Record) bool {
//	            return r["age"].(int) >= 18
//	        }),
//	        output.NewSortOp(output.SortKey{Column: "name", Direction: output.Ascending}),
//	        output.NewLimitOp(10),
//	    ),
//	)
//
// See v2/BEST_PRACTICES.md for safe operation patterns and v2/MIGRATION.md
// for examples migrating from the deprecated Pipeline API.
func WithTransformations(ops ...Operation) TableOption {
	return func(tc *tableConfig) {
		// Filter out nil operations so they are never stored. A nil Operation
		// would panic when its interface methods (Validate/Name/Apply) are
		// called during rendering. Skipping nil matches how the rest of the
		// transformation API handles nil inputs (e.g. TransformPipeline.Add).
		filtered := make([]Operation, 0, len(ops))
		for _, op := range ops {
			if op == nil {
				continue
			}
			filtered = append(filtered, op)
		}
		tc.transformations = filtered
	}
}

// DetectSchemaFromData creates a schema from the provided data. Slice input
// is scanned in full: the detected columns are the union of keys across all
// rows, so columns that first appear in a later row are included rather than
// silently dropped (T-1576). Non-map elements in []any input are ignored:
// the union covers the map elements only. All accepted shapes are map-based,
// and map input has no recoverable key order, so the detected column order is
// alphabetical (see DetectSchemaFromMap). Use WithKeys or WithSchema when
// column order matters.
func DetectSchemaFromData(data any) *Schema {
	switch v := data.(type) {
	case []Record:
		return detectSchemaFromMaps(v)
	case []map[string]any:
		return detectSchemaFromMaps(v)
	case Record:
		return DetectSchemaFromMap(v)
	case map[string]any:
		return DetectSchemaFromMap(v)
	case []any:
		// Non-map items are skipped here: the union is built from whatever
		// maps are present. convertToRecords rejects []any input containing
		// non-map elements, so a schema detected from such input is
		// discarded before rendering anyway.
		items := make([]map[string]any, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				items = append(items, m)
			}
		}
		return detectSchemaFromMaps(items)
	}
	return &Schema{Fields: []Field{}, keyOrder: []string{}}
}

// DetectSchemaFromMap creates a schema from a map. Go map iteration order is
// randomized and the original insertion order is unrecoverable, so keys are
// sorted alphabetically as a deterministic fallback — the source key order is
// NOT preserved. Callers that need a specific column order must use WithKeys
// or WithSchema (T-1692).
func DetectSchemaFromMap(m map[string]any) *Schema {
	return detectSchemaFromMaps([]map[string]any{m})
}

// detectSchemaFromMaps creates a schema covering the union of keys across all
// rows, so columns that first appear after row 0 are not dropped (T-1576).
// Keys are sorted alphabetically for deterministic output (map key order is
// unrecoverable, see DetectSchemaFromMap). Each column's type is detected
// from its value in the first row that contains the key.
func detectSchemaFromMaps[M ~map[string]any](rows []M) *Schema {
	types := make(map[string]string)
	for _, row := range rows {
		for k, val := range row {
			if _, seen := types[k]; !seen {
				types[k] = DetectType(val)
			}
		}
	}
	keys := make([]string, 0, len(types))
	for k := range types {
		keys = append(keys, k)
	}
	sort.Strings(keys) // map key order is unrecoverable; alphabetize for deterministic output

	fields := make([]Field, 0, len(keys))
	for _, k := range keys {
		fields = append(fields, Field{
			Name: k,
			Type: types[k],
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

// ApplyTableOptions applies all options to the table configuration.
// Nil options are ignored.
func ApplyTableOptions(opts ...TableOption) *tableConfig {
	tc := &tableConfig{
		autoSchema: true, // Default to auto-detection
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(tc)
	}
	return tc
}
