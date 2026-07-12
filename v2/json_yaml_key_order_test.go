package output

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"testing"

	"gopkg.in/yaml.v3"
)

// Regression tests for T-1520: record objects in JSON/YAML output must
// serialize their keys in the user-specified order, not alphabetically.
// These tests assert the order of keys as they appear in the emitted bytes,
// not just the out-of-band schema.keys array. Before the fix, records were
// built as map[string]any (alphabetized by encoding/json) and the
// multi-content, section, and stream paths destroyed order again through
// map round trips.

// keyOrderRegressionKeys is deliberately non-alphabetical; alphabetized
// output would produce [alpha, mike, zebra] instead.
var keyOrderRegressionKeys = []string{"zebra", "alpha", "mike"}

func keyOrderRegressionRecords() []map[string]any {
	return []map[string]any{
		{"zebra": 1, "alpha": 2, "mike": 3},
		{"alpha": 5, "mike": 6, "zebra": 4},
	}
}

// jsonObjectKeyOrder returns the keys of a serialized JSON object in the
// order they appear in the raw bytes.
func jsonObjectKeyOrder(t *testing.T, raw []byte) []string {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		t.Fatalf("failed to read object start: %v", err)
	}
	if tok != json.Delim('{') {
		t.Fatalf("expected object start, got %v", tok)
	}
	var keys []string
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			t.Fatalf("failed to read key: %v", err)
		}
		key, ok := tok.(string)
		if !ok {
			t.Fatalf("expected string key, got %T", tok)
		}
		keys = append(keys, key)
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			t.Fatalf("failed to read value for key %q: %v", key, err)
		}
	}
	return keys
}

// jsonTableRecordKeyOrders extracts the serialized key order of every record
// in a rendered table object's "data" array.
func jsonTableRecordKeyOrders(t *testing.T, tableJSON []byte) [][]string {
	t.Helper()
	var table struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(tableJSON, &table); err != nil {
		t.Fatalf("failed to parse table JSON: %v", err)
	}
	if len(table.Data) == 0 {
		t.Fatal("expected data records in table JSON")
	}
	orders := make([][]string, 0, len(table.Data))
	for _, record := range table.Data {
		orders = append(orders, jsonObjectKeyOrder(t, record))
	}
	return orders
}

func assertRecordKeyOrders(t *testing.T, got [][]string, want []string) {
	t.Helper()
	for i, keys := range got {
		if !slices.Equal(keys, want) {
			t.Errorf("record %d serialized key order = %v, want %v", i, keys, want)
		}
	}
}

// yamlRecordKeyOrders parses YAML output and returns the key order of every
// record mapping found under a "data" sequence, in document order.
func yamlRecordKeyOrders(t *testing.T, out []byte) [][]string {
	t.Helper()
	var root yaml.Node
	if err := yaml.Unmarshal(out, &root); err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}
	var orders [][]string
	collectYAMLDataRecordOrders(&root, &orders)
	if len(orders) == 0 {
		t.Fatal("no data records found in YAML output")
	}
	return orders
}

func collectYAMLDataRecordOrders(node *yaml.Node, orders *[][]string) {
	if node == nil {
		return
	}
	if node.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(node.Content); i += 2 {
			key, value := node.Content[i], node.Content[i+1]
			if key.Value == "data" && value.Kind == yaml.SequenceNode {
				for _, record := range value.Content {
					if record.Kind != yaml.MappingNode {
						continue
					}
					var keys []string
					for k := 0; k+1 < len(record.Content); k += 2 {
						keys = append(keys, record.Content[k].Value)
					}
					*orders = append(*orders, keys)
				}
				continue
			}
			collectYAMLDataRecordOrders(value, orders)
		}
		return
	}
	for _, child := range node.Content {
		collectYAMLDataRecordOrders(child, orders)
	}
}

func TestJSONRenderer_RecordObjectsSerializeInKeyOrder(t *testing.T) {
	doc := New().
		Table("test", keyOrderRegressionRecords(), WithKeys(keyOrderRegressionKeys...)).
		Build()

	result, err := (&jsonRenderer{}).Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	assertRecordKeyOrders(t, jsonTableRecordKeyOrders(t, result), keyOrderRegressionKeys)
}

func TestJSONRenderer_MultiContentRecordObjectsSerializeInKeyOrder(t *testing.T) {
	doc := New().
		Table("first", keyOrderRegressionRecords(), WithKeys(keyOrderRegressionKeys...)).
		Table("second", keyOrderRegressionRecords(), WithKeys(keyOrderRegressionKeys...)).
		Build()

	result, err := (&jsonRenderer{}).Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	var contents []json.RawMessage
	if err := json.Unmarshal(result, &contents); err != nil {
		t.Fatalf("failed to parse multi-content JSON array: %v", err)
	}
	if len(contents) != 2 {
		t.Fatalf("expected 2 content elements, got %d", len(contents))
	}
	for i, content := range contents {
		t.Logf("checking content element %d", i)
		assertRecordKeyOrders(t, jsonTableRecordKeyOrders(t, content), keyOrderRegressionKeys)
	}
}

func TestJSONRenderer_SectionTableRecordObjectsSerializeInKeyOrder(t *testing.T) {
	doc := New().
		Section("wrapper", func(b *Builder) {
			b.Table("nested", keyOrderRegressionRecords(), WithKeys(keyOrderRegressionKeys...))
		}).
		Build()

	result, err := (&jsonRenderer{}).Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	var section struct {
		Contents []json.RawMessage `json:"contents"`
	}
	if err := json.Unmarshal(result, &section); err != nil {
		t.Fatalf("failed to parse section JSON: %v", err)
	}
	if len(section.Contents) != 1 {
		t.Fatalf("expected 1 nested content, got %d", len(section.Contents))
	}
	assertRecordKeyOrders(t, jsonTableRecordKeyOrders(t, section.Contents[0]), keyOrderRegressionKeys)
}

func TestJSONRenderer_CollapsibleSectionTableRecordObjectsSerializeInKeyOrder(t *testing.T) {
	doc := New().
		CollapsibleSection("wrapper", func(b *Builder) {
			b.Table("nested", keyOrderRegressionRecords(), WithKeys(keyOrderRegressionKeys...))
		}).
		Build()

	result, err := (&jsonRenderer{}).Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	var section struct {
		Content []json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(result, &section); err != nil {
		t.Fatalf("failed to parse collapsible section JSON: %v", err)
	}
	if len(section.Content) != 1 {
		t.Fatalf("expected 1 nested content, got %d", len(section.Content))
	}
	assertRecordKeyOrders(t, jsonTableRecordKeyOrders(t, section.Content[0]), keyOrderRegressionKeys)
}

func TestJSONRenderer_StreamTableRecordObjectsSerializeInKeyOrder(t *testing.T) {
	table, err := NewTableContent("test", keyOrderRegressionRecords(), WithKeys(keyOrderRegressionKeys...))
	if err != nil {
		t.Fatalf("NewTableContent failed: %v", err)
	}

	var buf bytes.Buffer
	if err := (&jsonRenderer{}).renderTableContentJSONStream(table, &buf); err != nil {
		t.Fatalf("renderTableContentJSONStream failed: %v", err)
	}

	assertRecordKeyOrders(t, jsonTableRecordKeyOrders(t, buf.Bytes()), keyOrderRegressionKeys)
}

func TestYAMLRenderer_RecordMappingsSerializeInKeyOrder(t *testing.T) {
	doc := New().
		Table("test", keyOrderRegressionRecords(), WithKeys(keyOrderRegressionKeys...)).
		Build()

	result, err := (&yamlRenderer{}).Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	assertRecordKeyOrders(t, yamlRecordKeyOrders(t, result), keyOrderRegressionKeys)
}

func TestYAMLRenderer_MultiContentRecordMappingsSerializeInKeyOrder(t *testing.T) {
	doc := New().
		Table("first", keyOrderRegressionRecords(), WithKeys(keyOrderRegressionKeys...)).
		Table("second", keyOrderRegressionRecords(), WithKeys(keyOrderRegressionKeys...)).
		Build()

	result, err := (&yamlRenderer{}).Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	orders := yamlRecordKeyOrders(t, result)
	if len(orders) != 4 {
		t.Fatalf("expected 4 records across 2 tables, got %d", len(orders))
	}
	assertRecordKeyOrders(t, orders, keyOrderRegressionKeys)
}

func TestYAMLRenderer_SectionTableRecordMappingsSerializeInKeyOrder(t *testing.T) {
	doc := New().
		Section("wrapper", func(b *Builder) {
			b.Table("nested", keyOrderRegressionRecords(), WithKeys(keyOrderRegressionKeys...))
		}).
		Build()

	result, err := (&yamlRenderer{}).Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	assertRecordKeyOrders(t, yamlRecordKeyOrders(t, result), keyOrderRegressionKeys)
}

func TestYAMLRenderer_CollapsibleSectionTableRecordMappingsSerializeInKeyOrder(t *testing.T) {
	doc := New().
		CollapsibleSection("wrapper", func(b *Builder) {
			b.Table("nested", keyOrderRegressionRecords(), WithKeys(keyOrderRegressionKeys...))
		}).
		Build()

	result, err := (&yamlRenderer{}).Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	assertRecordKeyOrders(t, yamlRecordKeyOrders(t, result), keyOrderRegressionKeys)
}

func TestYAMLRenderer_StreamTableRecordMappingsSerializeInKeyOrder(t *testing.T) {
	table, err := NewTableContent("test", keyOrderRegressionRecords(), WithKeys(keyOrderRegressionKeys...))
	if err != nil {
		t.Fatalf("NewTableContent failed: %v", err)
	}

	var buf bytes.Buffer
	if err := (&yamlRenderer{}).renderTableContentYAMLStream(table, &buf); err != nil {
		t.Fatalf("renderTableContentYAMLStream failed: %v", err)
	}

	assertRecordKeyOrders(t, yamlRecordKeyOrders(t, buf.Bytes()), keyOrderRegressionKeys)
}
