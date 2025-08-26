package output

import (
	"bytes"
	"context"
	"encoding/csv"
	"strings"
	"testing"
)

func TestCSVRenderer_TableKeyOrderPreservation(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		data     []map[string]any
		expected []string // Expected key order in CSV header
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
			name: "reverse alphabetical order",
			keys: []string{"zebra", "yellow", "alpha"},
			data: []map[string]any{
				{"alpha": "first", "yellow": "second", "zebra": "third"},
			},
			expected: []string{"zebra", "yellow", "alpha"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create table with explicit key order
			doc := New().
				Table("test", tt.data, WithKeys(tt.keys...)).
				Build()

			renderer := &csvRenderer{}
			result, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			// Parse CSV result
			csvReader := csv.NewReader(strings.NewReader(string(result)))
			records, err := csvReader.ReadAll()
			if err != nil {
				t.Fatalf("Failed to parse CSV: %v", err)
			}

			if len(records) == 0 {
				t.Fatal("Expected at least header row in CSV output")
			}

			// Check header row (first row) for key order
			header := records[0]
			if len(header) != len(tt.expected) {
				t.Fatalf("Expected %d columns, got %d", len(tt.expected), len(header))
			}

			for i, expectedKey := range tt.expected {
				if header[i] != expectedKey {
					t.Errorf("Key order mismatch at column %d: expected %q, got %q", i, expectedKey, header[i])
				}
			}

			// Verify data rows maintain the same key order
			for rowIdx := 1; rowIdx < len(records); rowIdx++ {
				row := records[rowIdx]
				if len(row) != len(tt.expected) {
					t.Errorf("Row %d has %d columns, expected %d", rowIdx, len(row), len(tt.expected))
				}
			}
		})
	}
}

func TestCSVRenderer_DataTypeHandling(t *testing.T) {
	testData := []map[string]any{
		{
			"string":     "hello world",
			"int":        42,
			"float":      3.14159,
			"bool_true":  true,
			"bool_false": false,
			"nil":        nil,
			"zero":       0,
			"empty":      "",
		},
	}

	doc := New().
		Table("types", testData, WithKeys("string", "int", "float", "bool_true", "bool_false", "nil", "zero", "empty")).
		Build()

	renderer := &csvRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Parse CSV result
	csvReader := csv.NewReader(strings.NewReader(string(result)))
	records, err := csvReader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	if len(records) < 2 {
		t.Fatal("Expected header row plus at least one data row")
	}

	dataRow := records[1]

	// Verify data types are formatted correctly
	expectedValues := []string{
		"hello world", // string
		"42",          // int
		"3.14159",     // float (or close to it)
		"true",        // bool_true
		"false",       // bool_false
		"",            // nil -> empty string
		"0",           // zero
		"",            // empty string
	}

	for i, expected := range expectedValues {
		if i >= len(dataRow) {
			t.Fatalf("Data row has fewer columns than expected")
		}
		actual := dataRow[i]

		// For float comparison, be more flexible
		if expected == "3.14159" && strings.HasPrefix(actual, "3.1415") {
			continue // Accept slight formatting differences
		}

		if actual != expected {
			t.Errorf("Column %d value mismatch: expected %q, got %q", i, expected, actual)
		}
	}
}

func TestCSVRenderer_SpecialCharacterEscaping(t *testing.T) {
	testData := []map[string]any{
		{
			"quotes":   `text with "quotes" inside`,
			"commas":   "text, with, commas",
			"newlines": "text\nwith\nnewlines",
			"tabs":     "text\twith\ttabs",
			"mixed":    "complex \"text\", with\nnewlines\tand\rcarriage returns",
		},
	}

	doc := New().
		Table("special", testData, WithKeys("quotes", "commas", "newlines", "tabs", "mixed")).
		Build()

	renderer := &csvRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Parse CSV result - this should succeed if escaping is correct
	csvReader := csv.NewReader(strings.NewReader(string(result)))
	records, err := csvReader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV (escaping issue): %v", err)
	}

	if len(records) < 2 {
		t.Fatal("Expected header row plus at least one data row")
	}

	dataRow := records[1]

	// Verify special characters are handled
	if !strings.Contains(dataRow[0], "quotes") {
		t.Error("Quotes field should contain the word 'quotes'")
	}
	if !strings.Contains(dataRow[1], "commas") {
		t.Error("Commas field should contain the word 'commas'")
	}
	// Newlines should be converted to spaces
	if strings.Contains(dataRow[2], "\n") {
		t.Error("Newlines should be converted to spaces")
	}
	if strings.Contains(dataRow[3], "\t") {
		t.Error("Tabs should be converted to spaces")
	}
}

func TestCSVRenderer_StreamingVsBuffered(t *testing.T) {
	// Create a test document with table data
	doc := New().
		Table("users", []map[string]any{
			{"id": 1, "name": "Alice", "score": 95.5},
			{"id": 2, "name": "Bob", "score": 87.2},
			{"id": 3, "name": "Charlie", "score": 92.1},
		}, WithKeys("id", "name", "score")).
		Build()

	renderer := &csvRenderer{}

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

	// Results should be identical for CSV
	if !bytes.Equal(bufferedResult, streamedResult) {
		t.Errorf("Buffered and streamed results differ:\nBuffered: %q\nStreamed: %q",
			string(bufferedResult), string(streamedResult))
	}

	// Both should be valid CSV
	for i, result := range [][]byte{bufferedResult, streamedResult} {
		csvReader := csv.NewReader(strings.NewReader(string(result)))
		records, err := csvReader.ReadAll()
		if err != nil {
			t.Errorf("Result %d is not valid CSV: %v", i, err)
		}
		if len(records) != 4 { // header + 3 data rows
			t.Errorf("Result %d has %d rows, expected 4", i, len(records))
		}
	}
}
