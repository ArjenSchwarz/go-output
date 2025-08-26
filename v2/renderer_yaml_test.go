package output

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestYAMLRenderer_TableKeyOrderPreservation(t *testing.T) {
	testData := []map[string]any{
		{"name": "Alice", "id": 1, "active": true, "score": 95.5},
		{"score": 87.2, "id": 2, "name": "Bob", "active": false},
	}

	// Test with explicit key order that differs from alphabetical
	keyOrder := []string{"score", "name", "id", "active"}

	doc := New().
		Table("test", testData, WithKeys(keyOrder...)).
		Build()

	renderer := &yamlRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Parse YAML result
	var parsed map[string]any
	if err := yaml.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	// Check schema key order
	schema, ok := parsed["schema"].(map[string]any)
	if !ok {
		t.Fatal("Expected schema in YAML output")
	}

	keys, ok := schema["keys"].([]any)
	if !ok {
		t.Fatal("Expected keys array in schema")
	}

	// Verify key order matches expected
	if len(keys) != len(keyOrder) {
		t.Fatalf("Expected %d keys, got %d", len(keyOrder), len(keys))
	}

	for i, key := range keys {
		keyStr, ok := key.(string)
		if !ok {
			t.Fatalf("Key at index %d is not a string: %T", i, key)
		}
		if keyStr != keyOrder[i] {
			t.Errorf("Key order mismatch at index %d: expected %q, got %q", i, keyOrder[i], keyStr)
		}
	}
}

func TestYAMLRenderer_DataTypePreservation(t *testing.T) {
	testData := []map[string]any{
		{
			"string": "hello world",
			"int":    42,
			"float":  3.14159,
			"bool":   true,
			"nil":    nil,
			"zero":   0,
			"empty":  "",
		},
	}

	doc := New().
		Table("types", testData, WithKeys("string", "int", "float", "bool", "nil", "zero", "empty")).
		Build()

	renderer := &yamlRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	var parsed map[string]any
	if err := yaml.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	data, ok := parsed["data"].([]any)
	if !ok || len(data) == 0 {
		t.Fatal("Expected data array with at least one record")
	}

	record, ok := data[0].(map[string]any)
	if !ok {
		t.Fatal("First record is not a map")
	}

	// Verify types are preserved (YAML preserves types better than JSON)
	if record["string"] != "hello world" {
		t.Errorf("String value mismatch: expected %q, got %v", "hello world", record["string"])
	}

	if record["int"] != 42 {
		t.Errorf("Int value mismatch: expected %v, got %v", 42, record["int"])
	}

	if record["float"] != 3.14159 {
		t.Errorf("Float value mismatch: expected %v, got %v", 3.14159, record["float"])
	}

	if record["bool"] != true {
		t.Errorf("Bool value mismatch: expected %v, got %v", true, record["bool"])
	}

	if record["nil"] != nil {
		t.Errorf("Nil value mismatch: expected %v, got %v", nil, record["nil"])
	}

	if record["zero"] != 0 {
		t.Errorf("Zero value mismatch: expected %v, got %v", 0, record["zero"])
	}

	if record["empty"] != "" {
		t.Errorf("Empty string mismatch: expected %q, got %v", "", record["empty"])
	}
}

func TestYAMLRenderer_StreamingVsBuffered(t *testing.T) {
	doc := New().
		Text("Header text").
		Table("data", []map[string]any{
			{"key": "value1", "num": 1},
			{"key": "value2", "num": 2},
		}, WithKeys("key", "num")).
		Build()

	renderer := &yamlRenderer{}

	// Test buffered rendering
	bufferedResult, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Buffered render failed: %v", err)
	}

	// Test streaming rendering
	var streamBuf bytes.Buffer
	err = renderer.RenderTo(context.Background(), doc, &streamBuf)
	if err != nil {
		t.Fatalf("Streaming render failed: %v", err)
	}
	streamedResult := streamBuf.Bytes()

	// Parse both results to ensure they're valid YAML
	var bufferedParsed, streamedParsed any
	if err := yaml.Unmarshal(bufferedResult, &bufferedParsed); err != nil {
		t.Fatalf("Buffered result is not valid YAML: %v", err)
	}
	if err := yaml.Unmarshal(streamedResult, &streamedParsed); err != nil {
		t.Fatalf("Streamed result is not valid YAML: %v", err)
	}

	// Both should contain the same key elements
	bufferedStr := string(bufferedResult)
	streamedStr := string(streamedResult)

	expectedElements := []string{
		"key:", "num:", "value1", "value2", "Header text",
	}

	for _, element := range expectedElements {
		if !strings.Contains(bufferedStr, element) {
			t.Errorf("Buffered result missing element: %s", element)
		}
		if !strings.Contains(streamedStr, element) {
			t.Errorf("Streamed result missing element: %s", element)
		}
	}
}
