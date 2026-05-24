package output

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"strings"
	"testing"
)

func TestCSVRenderer_TableKeyOrderPreservation(t *testing.T) {
	tests := map[string]struct {
		keys     []string
		data     []map[string]any
		expected []string
	}{ // Expected key order in CSV header
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
		}, "reverse alphabetical order": {

			keys: []string{"zebra", "yellow", "alpha"},
			data: []map[string]any{
				{"alpha": "first", "yellow": "second", "zebra": "third"},
			},
			expected: []string{"zebra", "yellow", "alpha"},
		}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
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

// TestCSVRenderer_TransformationIntegration tests that CSVRenderer applies transformations
func TestCSVRenderer_TransformationIntegration(t *testing.T) {
	data := []Record{
		{"name": "Alice", "age": 30},
		{"name": "Bob", "age": 25},
		{"name": "Charlie", "age": 35},
	}

	doc := New().
		Table("test", data,
			WithKeys("name", "age"),
			WithTransformations(
				NewFilterOp(func(r Record) bool {
					return r["age"].(int) >= 30
				}),
			),
		).
		Build()

	renderer := &csvRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Parse CSV to verify filtering worked
	csvReader := csv.NewReader(strings.NewReader(string(result)))
	records, err := csvReader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	// Should have header + 2 filtered records
	if len(records) != 3 {
		t.Errorf("Expected 3 rows (header + 2 data), got %d", len(records))
	}
}

// csvFailWriter is an io.Writer that always fails. Because encoding/csv buffers
// data through a bufio.Writer, the failure typically surfaces only when the
// buffer is flushed rather than on the individual Write call.
type csvFailWriter struct{}

func (csvFailWriter) Write(p []byte) (int, error) {
	return 0, errors.New("forced write failure")
}

// TestCSVRenderer_RenderToFlushError verifies that RenderTo reports the error
// from an underlying writer that fails during flush (T-1186). csv.Writer.Write
// buffers data, so a writer failure may only be reported by csvWriter.Error()
// after Flush. Before the fix RenderTo returned nil for a failing writer.
func TestCSVRenderer_RenderToFlushError(t *testing.T) {
	doc := New().
		Table("users", []map[string]any{
			{"id": 1, "name": "Alice"},
			{"id": 2, "name": "Bob"},
		}, WithKeys("id", "name")).
		Build()

	renderer := &csvRenderer{}
	err := renderer.RenderTo(context.Background(), doc, csvFailWriter{})
	if err == nil {
		t.Fatal("expected RenderTo to return an error when the underlying writer fails on flush, got nil")
	}
	if !strings.Contains(err.Error(), "forced write failure") {
		t.Errorf("error = %v, want it to wrap the underlying %q failure", err, "forced write failure")
	}
}

// TestCSVRenderer_RenderToFlushErrorWithSection verifies the same flush-error
// detection for a document whose content comes from a section (a separate
// rendering path through renderDocumentCSVTo).
func TestCSVRenderer_RenderToFlushErrorWithSection(t *testing.T) {
	doc := New().
		Section("group", func(b *Builder) {
			b.Table("users", []map[string]any{
				{"id": 1, "name": "Alice"},
			}, WithKeys("id", "name"))
		}).
		Build()

	renderer := &csvRenderer{}
	err := renderer.RenderTo(context.Background(), doc, csvFailWriter{})
	if err == nil {
		t.Fatal("expected RenderTo to return an error for a failing writer with section content, got nil")
	}
	if !strings.Contains(err.Error(), "forced write failure") {
		t.Errorf("error = %v, want it to wrap the underlying %q failure", err, "forced write failure")
	}
}

// TestCSVRenderer_DeeplyNestedSectionTables is a regression test for T-1239.
// The CSV renderer flattens tables inside sections, but the original
// implementation only handled direct table children plus a single nested
// SectionContent level. Tables nested 3+ levels deep were silently dropped
// from the output. This test builds tables at several nesting depths and
// verifies each one's rows reach the CSV output.
func TestCSVRenderer_DeeplyNestedSectionTables(t *testing.T) {
	// newTable builds a single-row table whose only value identifies the depth
	// at which it lives, so we can assert each table reaches the output.
	newTable := func(t *testing.T, name string) *TableContent {
		t.Helper()
		table, err := NewTableContent(name, []map[string]any{{"name": name}}, WithKeys("name"))
		if err != nil {
			t.Fatalf("failed to create table %q: %v", name, err)
		}
		return table
	}

	tests := map[string]struct {
		// depth is the number of nested sections wrapping the table.
		// depth=1 -> section -> table (already worked before the fix)
		// depth=2 -> outer -> inner -> table (already worked before the fix)
		// depth=3 -> outer -> middle -> inner -> table (regression)
		depth int
		row   string
	}{
		"one level":    {depth: 1, row: "level-1"},
		"two levels":   {depth: 2, row: "level-2"},
		"three levels": {depth: 3, row: "deep-row"},
		"five levels":  {depth: 5, row: "level-5"},
	}

	for testName, tt := range tests {
		t.Run(testName, func(t *testing.T) {
			// Build the innermost section holding the table, then wrap it in
			// the requested number of outer sections.
			innermost := NewSectionContent("section-0")
			innermost.AddContent(newTable(t, tt.row))

			current := innermost
			for level := 1; level < tt.depth; level++ {
				outer := NewSectionContent("section")
				outer.AddContent(current)
				current = outer
			}

			doc := New().AddContent(current).Build()

			result, err := (&csvRenderer{}).Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			if !strings.Contains(string(result), tt.row) {
				t.Errorf("CSV output missing table nested %d level(s) deep: want it to contain %q, got %q",
					tt.depth, tt.row, string(result))
			}
		})
	}
}

// TestCSVRenderer_MultipleTablesAcrossNestingLevels verifies that tables at
// different depths within the same section hierarchy are all rendered, not
// just the shallowest ones. Regression test for T-1239.
func TestCSVRenderer_MultipleTablesAcrossNestingLevels(t *testing.T) {
	makeTable := func(t *testing.T, row string) *TableContent {
		t.Helper()
		table, err := NewTableContent(row, []map[string]any{{"name": row}}, WithKeys("name"))
		if err != nil {
			t.Fatalf("failed to create table %q: %v", row, err)
		}
		return table
	}

	// Structure: outer{tableA, middle{tableB, inner{tableC}}}
	inner := NewSectionContent("inner")
	inner.AddContent(makeTable(t, "row-c"))

	middle := NewSectionContent("middle")
	middle.AddContent(makeTable(t, "row-b"))
	middle.AddContent(inner)

	outer := NewSectionContent("outer")
	outer.AddContent(makeTable(t, "row-a"))
	outer.AddContent(middle)

	doc := New().AddContent(outer).Build()

	result, err := (&csvRenderer{}).Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	out := string(result)
	for _, want := range []string{"row-a", "row-b", "row-c"} {
		if !strings.Contains(out, want) {
			t.Errorf("CSV output missing table row %q across nesting levels, got %q", want, out)
		}
	}
}
