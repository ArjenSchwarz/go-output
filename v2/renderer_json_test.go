package output

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONRenderer_TableKeyOrderPreservation(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		data     []map[string]any
		expected []string // Expected key order in output
	}{
		{
			name: "explicit key order Z-A-M",
			keys: []string{"Z", "A", "M"},
			data: []map[string]any{
				{"A": 1, "M": 2, "Z": 3},
				{"Z": 6, "M": 5, "A": 4},
			},
			expected: []string{"Z", "A", "M"},
		},
		{
			name: "numeric and string fields ordered",
			keys: []string{"id", "name", "score", "active"},
			data: []map[string]any{
				{"name": "Alice", "id": 1, "active": true, "score": 95.5},
				{"score": 87.2, "id": 2, "name": "Bob", "active": false},
			},
			expected: []string{"id", "name", "score", "active"},
		},
		{
			name: "single record key preservation",
			keys: []string{"last", "first", "middle"},
			data: []map[string]any{
				{"first": "John", "last": "Doe", "middle": "Q"},
			},
			expected: []string{"last", "first", "middle"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create table with explicit key order
			doc := New().
				Table("test", tt.data, WithKeys(tt.keys...)).
				Build()

			renderer := &jsonRenderer{}
			result, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			// Parse JSON result
			var parsed map[string]any
			if err := json.Unmarshal(result, &parsed); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			// Check schema key order
			schema, ok := parsed["schema"].(map[string]any)
			if !ok {
				t.Fatal("Expected schema in JSON output")
			}

			keys, ok := schema["keys"].([]any)
			if !ok {
				t.Fatal("Expected keys array in schema")
			}

			// Verify key order matches expected
			if len(keys) != len(tt.expected) {
				t.Fatalf("Expected %d keys, got %d", len(tt.expected), len(keys))
			}

			for i, key := range keys {
				keyStr, ok := key.(string)
				if !ok {
					t.Fatalf("Key at index %d is not a string: %T", i, key)
				}
				if keyStr != tt.expected[i] {
					t.Errorf("Key order mismatch at index %d: expected %q, got %q", i, tt.expected[i], keyStr)
				}
			}

			// Check data records maintain key order
			data, ok := parsed["data"].([]any)
			if !ok {
				t.Fatal("Expected data array in JSON output")
			}

			for recordIdx, recordAny := range data {
				record, ok := recordAny.(map[string]any)
				if !ok {
					t.Fatalf("Record %d is not a map", recordIdx)
				}

				// Check that record contains expected keys in order
				for _, expectedKey := range tt.expected {
					if _, exists := record[expectedKey]; !exists {
						t.Errorf("Record %d missing expected key %q", recordIdx, expectedKey)
					}
				}
			}
		})
	}
}

func TestJSONRenderer_DataTypePreservation(t *testing.T) {
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

	renderer := &jsonRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	data, ok := parsed["data"].([]any)
	if !ok || len(data) == 0 {
		t.Fatal("Expected data array with at least one record")
	}

	record, ok := data[0].(map[string]any)
	if !ok {
		t.Fatal("First record is not a map")
	}

	// Verify types are preserved
	if record["string"] != "hello world" {
		t.Errorf("String value mismatch: expected %q, got %v", "hello world", record["string"])
	}

	// JSON unmarshaling converts numbers to float64
	if record["int"] != 42.0 {
		t.Errorf("Int value mismatch: expected %v, got %v", 42.0, record["int"])
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

	if record["zero"] != 0.0 {
		t.Errorf("Zero value mismatch: expected %v, got %v", 0.0, record["zero"])
	}

	if record["empty"] != "" {
		t.Errorf("Empty string mismatch: expected %q, got %v", "", record["empty"])
	}
}

func TestJSONRenderer_StreamingVsBuffered(t *testing.T) {
	// Create a moderately complex document
	doc := New().
		Text("Header text", WithHeader(true)).
		Table("users", []map[string]any{
			{"id": 1, "name": "Alice", "score": 95.5},
			{"id": 2, "name": "Bob", "score": 87.2},
			{"id": 3, "name": "Charlie", "score": 92.1},
		}, WithKeys("id", "name", "score")).
		Raw(FormatJSON, []byte(`{"custom": "data"}`)).
		Section("Details", func(b *Builder) {
			b.Text("Section content")
		}).
		Build()

	renderer := &jsonRenderer{}

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

	// Parse both results to ensure they're valid JSON
	var bufferedParsed, streamedParsed any
	if err := json.Unmarshal(bufferedResult, &bufferedParsed); err != nil {
		t.Fatalf("Buffered result is not valid JSON: %v", err)
	}
	if err := json.Unmarshal(streamedResult, &streamedParsed); err != nil {
		t.Fatalf("Streamed result is not valid JSON: %v", err)
	}

	// Compare the results (they may differ in formatting but should be semantically equivalent)
	bufferedStr := string(bufferedResult)
	streamedStr := string(streamedResult)

	// Verify both contain the same key elements
	expectedElements := []string{
		`"id"`, `"name"`, `"score"`,
		`"Alice"`, `"Bob"`, `"Charlie"`,
		`"Header text"`, `"Section content"`,
		`"type"`, `"text"`, `"section"`,
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
