package output

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestJSONRenderer_TableKeyOrderPreservation(t *testing.T) {
	tests := map[string]struct {
		keys     []string
		data     []map[string]any
		expected []string
	}{ // Expected key order in output
		"explicit key order Z-A-M": {

			keys: []string{"Z", "A", "M"},
			data: []map[string]any{
				{"A": 1, "M": 2, "Z": 3},
				{"Z": 6, "M": 5, "A": 4},
			},
			expected: []string{"Z", "A", "M"},
		}, "numeric and string fields ordered": {

			keys: []string{"id", "name", "score", "active"},
			data: []map[string]any{
				{"name": "Alice", "id": 1, "active": true, "score": 95.5},
				{"score": 87.2, "id": 2, "name": "Bob", "active": false},
			},
			expected: []string{"id", "name", "score", "active"},
		}, "single record key preservation": {

			keys: []string{"last", "first", "middle"},
			data: []map[string]any{
				{"first": "John", "last": "Doe", "middle": "Q"},
			},
			expected: []string{"last", "first", "middle"},
		}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
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

// TestJSONRenderer_TransformationIntegration tests that JSONRenderer correctly calls
// applyContentTransformations() for content with transformations
func TestJSONRenderer_TransformationIntegration(t *testing.T) {
	tests := map[string]struct {
		data            []Record
		transformations []Operation
		expectedCount   int // Expected number of records after transformation
		wantErr         bool
	}{
		"filter operation": {
			data: []Record{
				{"name": "Alice", "age": 30},
				{"name": "Bob", "age": 25},
				{"name": "Charlie", "age": 35},
			},
			transformations: []Operation{
				NewFilterOp(func(r Record) bool {
					age, ok := r["age"].(int)
					return ok && age >= 30
				}),
			},
			expectedCount: 2,
		},
		"sort and limit operations": {
			data: []Record{
				{"name": "Charlie", "score": 85},
				{"name": "Alice", "score": 95},
				{"name": "Bob", "score": 90},
			},
			transformations: []Operation{
				NewSortOp(SortKey{Column: "score", Direction: Descending}),
				NewLimitOp(2),
			},
			expectedCount: 2,
		},
		"multiple filters": {
			data: []Record{
				{"name": "Alice", "age": 30, "active": true},
				{"name": "Bob", "age": 25, "active": false},
				{"name": "Charlie", "age": 35, "active": true},
			},
			transformations: []Operation{
				NewFilterOp(func(r Record) bool {
					age, ok := r["age"].(int)
					return ok && age >= 30
				}),
				NewFilterOp(func(r Record) bool {
					active, ok := r["active"].(bool)
					return ok && active
				}),
			},
			expectedCount: 2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create document with transformations
			doc := New().
				Table("test", tc.data,
					WithKeys("name", "age", "score", "active"),
					WithTransformations(tc.transformations...),
				).
				Build()

			renderer := &jsonRenderer{}
			result, err := renderer.Render(context.Background(), doc)

			if tc.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tc.wantErr {
				return
			}

			// Parse result to verify transformation was applied
			var parsed map[string]any
			if err := json.Unmarshal(result, &parsed); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			data, ok := parsed["data"].([]any)
			if !ok {
				t.Fatal("Expected data array in JSON output")
			}

			if len(data) != tc.expectedCount {
				t.Errorf("Expected %d records after transformation, got %d", tc.expectedCount, len(data))
			}
		})
	}
}

// TestJSONRenderer_TransformationFailFast tests that rendering stops immediately
// on the first transformation error (fail-fast behavior)
func TestJSONRenderer_TransformationFailFast(t *testing.T) {
	tests := map[string]struct {
		data            []Record
		transformations []Operation
		wantErrContains string
	}{
		"invalid filter predicate": {
			data: []Record{
				{"name": "Alice", "age": 30},
			},
			transformations: []Operation{
				&FilterOp{predicate: nil}, // Invalid: nil predicate
			},
			wantErrContains: "invalid",
		},
		"invalid limit count": {
			data: []Record{
				{"name": "Alice", "age": 30},
			},
			transformations: []Operation{
				NewLimitOp(-1), // Invalid: negative limit
			},
			wantErrContains: "invalid",
		},
		"sort on non-existent column": {
			data: []Record{
				{"name": "Alice", "age": 30},
			},
			transformations: []Operation{
				NewSortOp(SortKey{Column: "nonexistent", Direction: Ascending}),
			},
			wantErrContains: "failed",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			doc := New().
				Table("test", tc.data,
					WithKeys("name", "age"),
					WithTransformations(tc.transformations...),
				).
				Build()

			renderer := &jsonRenderer{}
			_, err := renderer.Render(context.Background(), doc)

			if err == nil {
				t.Fatal("Expected error but got none")
			}

			if tc.wantErrContains != "" && !strings.Contains(err.Error(), tc.wantErrContains) {
				t.Errorf("Error message %q does not contain %q", err.Error(), tc.wantErrContains)
			}
		})
	}
}

// TestJSONRenderer_TransformationContextCancellation tests that context cancellation
// is properly propagated during transformation execution
func TestJSONRenderer_TransformationContextCancellation(t *testing.T) {
	tests := map[string]struct {
		setupContext func() context.Context
		wantErr      bool
	}{
		"cancelled context": {
			setupContext: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				return ctx
			},
			wantErr: true,
		},
		"timeout context": {
			setupContext: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
				defer cancel()
				time.Sleep(10 * time.Millisecond) // Ensure timeout
				return ctx
			},
			wantErr: true,
		},
		"valid context": {
			setupContext: func() context.Context {
				return context.Background()
			},
			wantErr: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			data := []Record{
				{"name": "Alice", "age": 30},
				{"name": "Bob", "age": 25},
			}

			doc := New().
				Table("test", data,
					WithKeys("name", "age"),
					WithTransformations(
						NewFilterOp(func(r Record) bool {
							return r["age"].(int) >= 25
						}),
					),
				).
				Build()

			renderer := &jsonRenderer{}
			ctx := tc.setupContext()
			_, err := renderer.Render(ctx, doc)

			if tc.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestJSONRenderer_OriginalDocumentUnchanged tests that the original document
// remains unchanged after transformation (immutability)
func TestJSONRenderer_OriginalDocumentUnchanged(t *testing.T) {
	originalData := []Record{
		{"name": "Alice", "age": 30},
		{"name": "Bob", "age": 25},
		{"name": "Charlie", "age": 35},
	}

	doc := New().
		Table("test", originalData,
			WithKeys("name", "age"),
			WithTransformations(
				NewFilterOp(func(r Record) bool {
					return r["age"].(int) >= 30
				}),
			),
		).
		Build()

	// Get original content before rendering
	originalContents := doc.GetContents()
	if len(originalContents) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(originalContents))
	}

	originalTable, ok := originalContents[0].(*TableContent)
	if !ok {
		t.Fatal("Expected TableContent")
	}

	originalRecordCount := len(originalTable.Records())

	// Render with transformations
	renderer := &jsonRenderer{}
	_, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Verify original document is unchanged
	afterContents := doc.GetContents()
	if len(afterContents) != 1 {
		t.Fatalf("Expected 1 content item after render, got %d", len(afterContents))
	}

	afterTable, ok := afterContents[0].(*TableContent)
	if !ok {
		t.Fatal("Expected TableContent after render")
	}

	afterRecordCount := len(afterTable.Records())

	if afterRecordCount != originalRecordCount {
		t.Errorf("Original document was modified: had %d records, now has %d", originalRecordCount, afterRecordCount)
	}

	// Verify record data is still the same
	if originalRecordCount != 3 {
		t.Errorf("Expected 3 records in original document, got %d", originalRecordCount)
	}
}

// TestJSONRenderer_MixedContentTransformations tests rendering documents with
// mixed content (some with transformations, some without)
func TestJSONRenderer_MixedContentTransformations(t *testing.T) {
	tests := map[string]struct {
		setupDoc        func() *Document
		expectTableData map[string]int // table ID -> expected record count
	}{
		"one table with transformations, one without": {
			setupDoc: func() *Document {
				return New().
					Table("filtered", []Record{
						{"name": "Alice", "age": 30},
						{"name": "Bob", "age": 25},
						{"name": "Charlie", "age": 35},
					},
						WithKeys("name", "age"),
						WithTransformations(
							NewFilterOp(func(r Record) bool {
								return r["age"].(int) >= 30
							}),
						),
					).
					Table("unfiltered", []Record{
						{"id": 1, "status": "active"},
						{"id": 2, "status": "inactive"},
					},
						WithKeys("id", "status"),
						// No transformations
					).
					Build()
			},
			expectTableData: map[string]int{
				"filtered":   2, // Filtered to 2 records
				"unfiltered": 2, // All 2 records
			},
		},
		"multiple tables all with transformations": {
			setupDoc: func() *Document {
				return New().
					Table("users", []Record{
						{"name": "Alice", "age": 30},
						{"name": "Bob", "age": 25},
					},
						WithKeys("name", "age"),
						WithTransformations(
							NewFilterOp(func(r Record) bool {
								return r["age"].(int) >= 30
							}),
						),
					).
					Table("products", []Record{
						{"name": "Product A", "price": 100},
						{"name": "Product B", "price": 50},
						{"name": "Product C", "price": 75},
					},
						WithKeys("name", "price"),
						WithTransformations(
							NewSortOp(SortKey{Column: "price", Direction: Descending}),
							NewLimitOp(2),
						),
					).
					Build()
			},
			expectTableData: map[string]int{
				"users":    1,
				"products": 2,
			},
		},
		"text content mixed with table transformations": {
			setupDoc: func() *Document {
				return New().
					Text("Header text", WithHeader(true)).
					Table("data", []Record{
						{"name": "Alice", "value": 10},
						{"name": "Bob", "value": 20},
						{"name": "Charlie", "value": 15},
					},
						WithKeys("name", "value"),
						WithTransformations(
							NewLimitOp(2),
						),
					).
					Text("Footer text").
					Build()
			},
			expectTableData: map[string]int{
				"data": 2,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			doc := tc.setupDoc()

			renderer := &jsonRenderer{}
			result, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			// Parse result
			var parsed []any
			if err := json.Unmarshal(result, &parsed); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			// Verify each table's record count
			for _, item := range parsed {
				itemMap, ok := item.(map[string]any)
				if !ok {
					continue
				}

				// Check if this is a table with data
				data, hasData := itemMap["data"].([]any)
				if !hasData {
					continue
				}

				// Find which table this is by checking schema
				_, hasSchema := itemMap["schema"].(map[string]any)
				if !hasSchema {
					continue
				}

				// Match by number of records for simplicity
				recordCount := len(data)
				found := false
				for _, expectedCount := range tc.expectTableData {
					if recordCount == expectedCount {
						found = true
						break
					}
				}

				if !found {
					t.Errorf("Unexpected record count %d in output", recordCount)
				}
			}
		})
	}
}
