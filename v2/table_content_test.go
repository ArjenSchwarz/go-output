package output

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestTableContent_KeyOrderPreservation(t *testing.T) {
	tests := map[string]struct {
		keys     []string
		data     []map[string]any
		expected []string
	}{ // Expected column order in output
		"explicit key order Name-Age-Status": {

			keys: []string{"Name", "Age", "Status"},
			data: []map[string]any{
				{"Status": "Active", "Name": "Alice", "Age": 30},
				{"Age": 25, "Status": "Inactive", "Name": "Bob"},
			},
			expected: []string{"Name", "Age", "Status"},
		}, "explicit key order Z-A-M": {

			keys: []string{"Z", "A", "M"},
			data: []map[string]any{
				{"A": 1, "M": 2, "Z": 3},
				{"Z": 6, "M": 5, "A": 4},
			},
			expected: []string{"Z", "A", "M"},
		}, "reverse alphabetical": {

			keys: []string{"Z", "Y", "X", "A"},
			data: []map[string]any{
				{"A": 1, "X": 2, "Y": 3, "Z": 4},
			},
			expected: []string{"Z", "Y", "X", "A"},
		}, "single key": {

			keys: []string{"ID"},
			data: []map[string]any{
				{"ID": 1},
				{"ID": 2},
			},
			expected: []string{"ID"},
		}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			table, err := NewTableContent("Test Table", tt.data, WithKeys(tt.keys...))
			if err != nil {
				t.Fatalf("NewTableContent failed: %v", err)
			}

			// Verify key order is preserved in schema
			actualOrder := table.schema.GetKeyOrder()
			if !reflect.DeepEqual(actualOrder, tt.expected) {
				t.Errorf("key order = %v, want %v", actualOrder, tt.expected)
			}

			// Verify key order is preserved in text output
			var buf []byte
			result, err := table.AppendText(buf)
			if err != nil {
				t.Fatalf("AppendText failed: %v", err)
			}

			lines := strings.Split(string(result), "\n")
			if len(lines) < 3 { // title + header + at least one data row
				t.Fatalf("expected at least 3 lines, got %d", len(lines))
			}

			// Parse header line (line 1, after title)
			headerLine := lines[1]
			headers := strings.Split(headerLine, "\t")
			if !reflect.DeepEqual(headers, tt.expected) {
				t.Errorf("header order = %v, want %v", headers, tt.expected)
			}

			// Verify data rows follow the same order
			for i, dataLine := range lines[2:] {
				if dataLine == "" {
					continue // Skip empty lines
				}
				values := strings.Split(dataLine, "\t")
				if len(values) != len(tt.expected) {
					t.Errorf("row %d: expected %d values, got %d", i, len(tt.expected), len(values))
				}
			}
		})
	}
}

func TestTableContent_WithSchema(t *testing.T) {
	fields := []Field{
		{Name: "Priority", Type: "string"},
		{Name: "Task", Type: "string"},
		{Name: "Done", Type: "bool"},
	}

	data := []map[string]any{
		{"Task": "Implement TableContent", "Done": false, "Priority": "High"},
		{"Priority": "Medium", "Task": "Write tests", "Done": true},
	}

	table, err := NewTableContent("Tasks", data, WithSchema(fields...))
	if err != nil {
		t.Fatalf("NewTableContent failed: %v", err)
	}

	expectedOrder := []string{"Priority", "Task", "Done"}
	actualOrder := table.schema.GetKeyOrder()
	if !reflect.DeepEqual(actualOrder, expectedOrder) {
		t.Errorf("key order = %v, want %v", actualOrder, expectedOrder)
	}

	// Test text output
	var buf []byte
	result, err := table.AppendText(buf)
	if err != nil {
		t.Fatalf("AppendText failed: %v", err)
	}

	lines := strings.Split(string(result), "\n")
	headerLine := lines[1] // Skip title line
	headers := strings.Split(headerLine, "\t")
	if !reflect.DeepEqual(headers, expectedOrder) {
		t.Errorf("header order = %v, want %v", headers, expectedOrder)
	}
}

func TestTableContent_MixedTableScenarios(t *testing.T) {
	// Test multiple tables with different key orders in the same document
	table1Data := []map[string]any{
		{"Name": "Alice", "Email": "alice@example.com"},
	}

	table2Data := []map[string]any{
		{"ID": 1, "Status": "Active", "Time": "2024-01-01"},
	}

	table1, err := NewTableContent("Users", table1Data, WithKeys("Name", "Email"))
	if err != nil {
		t.Fatalf("NewTableContent table1 failed: %v", err)
	}

	table2, err := NewTableContent("Status", table2Data, WithKeys("ID", "Status", "Time"))
	if err != nil {
		t.Fatalf("NewTableContent table2 failed: %v", err)
	}

	// Verify each table maintains its own key order
	expectedOrder1 := []string{"Name", "Email"}
	expectedOrder2 := []string{"ID", "Status", "Time"}

	actualOrder1 := table1.schema.GetKeyOrder()
	actualOrder2 := table2.schema.GetKeyOrder()

	if !reflect.DeepEqual(actualOrder1, expectedOrder1) {
		t.Errorf("table1 key order = %v, want %v", actualOrder1, expectedOrder1)
	}

	if !reflect.DeepEqual(actualOrder2, expectedOrder2) {
		t.Errorf("table2 key order = %v, want %v", actualOrder2, expectedOrder2)
	}

	// Verify they produce different outputs
	var buf1, buf2 []byte
	result1, err := table1.AppendText(buf1)
	if err != nil {
		t.Fatalf("table1 AppendText failed: %v", err)
	}

	result2, err := table2.AppendText(buf2)
	if err != nil {
		t.Fatalf("table2 AppendText failed: %v", err)
	}

	if string(result1) == string(result2) {
		t.Error("expected different outputs for different tables")
	}
}

func TestTableContent_Implementation(t *testing.T) {
	data := []map[string]any{
		{"Name": "Alice", "Age": 30, "Status": "Active"},
	}

	table, err := NewTableContent("Test", data, WithKeys("Name", "Age", "Status"))
	if err != nil {
		t.Fatalf("NewTableContent failed: %v", err)
	}

	// Test Content interface implementation
	if table.Type() != ContentTypeTable {
		t.Errorf("Type() = %v, want %v", table.Type(), ContentTypeTable)
	}

	if table.ID() == "" {
		t.Error("ID() should return a non-empty string")
	}

	if table.Title() != "Test" {
		t.Errorf("Title() = %q, want %q", table.Title(), "Test")
	}

	// Test encoding interfaces
	var textBuf []byte
	textResult, err := table.AppendText(textBuf)
	if err != nil {
		t.Fatalf("AppendText failed: %v", err)
	}
	if len(textResult) == 0 {
		t.Error("AppendText should return non-empty result")
	}

	var binBuf []byte
	binResult, err := table.AppendBinary(binBuf)
	if err != nil {
		t.Fatalf("AppendBinary failed: %v", err)
	}
	if len(binResult) == 0 {
		t.Error("AppendBinary should return non-empty result")
	}
}

func TestTableContent_DataTypes(t *testing.T) {
	tests := map[string]struct {
		data any
		want error
	}{"single map[string]any": {

		data: map[string]any{"A": 1, "B": 2},
		want: nil,
	}, "slice of Record": {

		data: []Record{
			{"A": 1, "B": 2},
		},
		want: nil,
	}, "slice of any with maps": {

		data: []any{
			map[string]any{"A": 1, "B": 2},
		},
		want: nil,
	}, "slice of map[string]any": {

		data: []map[string]any{
			{"A": 1, "B": 2},
		},
		want: nil,
	}, "unsupported type": {

		data: "invalid",
		want: fmt.Errorf("some error"), // convertToRecords will return an error
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := NewTableContent("Test", tt.data)
			if (err != nil) != (tt.want != nil) {
				t.Errorf("NewTableContent() error = %v, want error = %v", err, tt.want != nil)
			}
		})
	}
}

func TestTableContent_EmptyData(t *testing.T) {
	emptyData := []map[string]any{}
	table, err := NewTableContent("Empty", emptyData, WithKeys("A", "B"))
	if err != nil {
		t.Fatalf("NewTableContent with empty data failed: %v", err)
	}

	expectedOrder := []string{"A", "B"}
	actualOrder := table.schema.GetKeyOrder()
	if !reflect.DeepEqual(actualOrder, expectedOrder) {
		t.Errorf("key order = %v, want %v", actualOrder, expectedOrder)
	}

	// Test output with empty data
	var buf []byte
	result, err := table.AppendText(buf)
	if err != nil {
		t.Fatalf("AppendText failed: %v", err)
	}

	lines := strings.Split(string(result), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines (title + header), got %d", len(lines))
	}

	headerLine := lines[1]
	headers := strings.Split(headerLine, "\t")
	if !reflect.DeepEqual(headers, expectedOrder) {
		t.Errorf("header order = %v, want %v", headers, expectedOrder)
	}
}

func TestTableContent_ThreadSafety(t *testing.T) {
	// Test that Records() returns a copy and doesn't allow external modification
	data := []map[string]any{
		{"A": 1, "B": 2},
	}

	table, err := NewTableContent("Test", data, WithKeys("A", "B"))
	if err != nil {
		t.Fatalf("NewTableContent failed: %v", err)
	}

	records1 := table.Records()
	records2 := table.Records()

	// Modify one of the returned copies
	records1[0]["A"] = 999

	// Verify the other copy is unaffected
	if records2[0]["A"] == 999 {
		t.Error("Records() should return independent copies")
	}

	// Verify original table is unaffected
	records3 := table.Records()
	if records3[0]["A"] == 999 {
		t.Error("External modification affected original table")
	}
}
