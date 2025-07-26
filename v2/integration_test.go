package output

import (
	"strings"
	"testing"
)

func TestIntegration_DocumentWithTables(t *testing.T) {
	// Create a document with multiple tables demonstrating key order preservation
	doc := New().
		Table("Users", []map[string]any{
			{"Name": "Alice", "Email": "alice@example.com", "Role": "Admin"},
			{"Name": "Bob", "Email": "bob@example.com", "Role": "User"},
		}, WithKeys("Name", "Email", "Role")).
		Table("Tasks", []map[string]any{
			{"Priority": "High", "Task": "Implement TableContent", "Done": false},
			{"Priority": "Medium", "Task": "Write tests", "Done": true},
		}, WithKeys("Priority", "Task", "Done")).
		Build()

	// Verify document contains expected content
	contents := doc.GetContents()
	if len(contents) != 2 {
		t.Fatalf("expected 2 contents, got %d", len(contents))
	}

	// Verify first table (Users)
	userTable, ok := contents[0].(*TableContent)
	if !ok {
		t.Fatal("first content should be TableContent")
	}

	expectedUserOrder := []string{"Name", "Email", "Role"}
	actualUserOrder := userTable.Schema().GetKeyOrder()
	if len(actualUserOrder) != len(expectedUserOrder) {
		t.Errorf("user table key count = %d, want %d", len(actualUserOrder), len(expectedUserOrder))
	}
	for i, key := range expectedUserOrder {
		if i >= len(actualUserOrder) || actualUserOrder[i] != key {
			t.Errorf("user table key[%d] = %q, want %q", i,
				func() string {
					if i < len(actualUserOrder) {
						return actualUserOrder[i]
					} else {
						return "missing"
					}
				}(),
				key)
		}
	}

	// Verify second table (Tasks)
	taskTable, ok := contents[1].(*TableContent)
	if !ok {
		t.Fatal("second content should be TableContent")
	}

	expectedTaskOrder := []string{"Priority", "Task", "Done"}
	actualTaskOrder := taskTable.Schema().GetKeyOrder()
	if len(actualTaskOrder) != len(expectedTaskOrder) {
		t.Errorf("task table key count = %d, want %d", len(actualTaskOrder), len(expectedTaskOrder))
	}
	for i, key := range expectedTaskOrder {
		if i >= len(actualTaskOrder) || actualTaskOrder[i] != key {
			t.Errorf("task table key[%d] = %q, want %q", i,
				func() string {
					if i < len(actualTaskOrder) {
						return actualTaskOrder[i]
					} else {
						return "missing"
					}
				}(),
				key)
		}
	}

	// Test text output for both tables
	var userBuf []byte
	userResult, err := userTable.AppendText(userBuf)
	if err != nil {
		t.Fatalf("user table AppendText failed: %v", err)
	}

	userLines := strings.Split(string(userResult), "\n")
	if len(userLines) < 3 {
		t.Fatalf("user table expected at least 3 lines, got %d", len(userLines))
	}

	userHeaders := strings.Split(userLines[1], "\t") // Skip title line
	if len(userHeaders) != len(expectedUserOrder) {
		t.Errorf("user table header count = %d, want %d", len(userHeaders), len(expectedUserOrder))
	}
	for i, header := range expectedUserOrder {
		if i >= len(userHeaders) || userHeaders[i] != header {
			t.Errorf("user table header[%d] = %q, want %q", i,
				func() string {
					if i < len(userHeaders) {
						return userHeaders[i]
					} else {
						return "missing"
					}
				}(),
				header)
		}
	}

	var taskBuf []byte
	taskResult, err := taskTable.AppendText(taskBuf)
	if err != nil {
		t.Fatalf("task table AppendText failed: %v", err)
	}

	taskLines := strings.Split(string(taskResult), "\n")
	if len(taskLines) < 3 {
		t.Fatalf("task table expected at least 3 lines, got %d", len(taskLines))
	}

	taskHeaders := strings.Split(taskLines[1], "\t") // Skip title line
	if len(taskHeaders) != len(expectedTaskOrder) {
		t.Errorf("task table header count = %d, want %d", len(taskHeaders), len(expectedTaskOrder))
	}
	for i, header := range expectedTaskOrder {
		if i >= len(taskHeaders) || taskHeaders[i] != header {
			t.Errorf("task table header[%d] = %q, want %q", i,
				func() string {
					if i < len(taskHeaders) {
						return taskHeaders[i]
					} else {
						return "missing"
					}
				}(),
				header)
		}
	}
}

func TestIntegration_SchemaWithFormatters(t *testing.T) {
	// Test custom formatter functionality
	fields := []Field{
		{
			Name: "Amount",
			Type: "float",
			Formatter: func(v any) any {
				if f, ok := v.(float64); ok {
					return "$" + formatValue(f)
				}
				return formatValue(v)
			},
		},
		{Name: "Description", Type: "string"},
	}

	data := []map[string]any{
		{"Amount": 123.45, "Description": "Payment"},
		{"Amount": 67.89, "Description": "Refund"},
	}

	table, err := NewTableContent("Transactions", data, WithSchema(fields...))
	if err != nil {
		t.Fatalf("NewTableContent failed: %v", err)
	}

	var buf []byte
	result, err := table.AppendText(buf)
	if err != nil {
		t.Fatalf("AppendText failed: %v", err)
	}

	output := string(result)
	if !strings.Contains(output, "$123.45") {
		t.Error("expected formatted amount $123.45 in output")
	}
	if !strings.Contains(output, "$67.89") {
		t.Error("expected formatted amount $67.89 in output")
	}

	// Verify key order is preserved
	expectedOrder := []string{"Amount", "Description"}
	actualOrder := table.Schema().GetKeyOrder()
	if len(actualOrder) != len(expectedOrder) {
		t.Errorf("key count = %d, want %d", len(actualOrder), len(expectedOrder))
	}
	for i, key := range expectedOrder {
		if i >= len(actualOrder) || actualOrder[i] != key {
			t.Errorf("key[%d] = %q, want %q", i,
				func() string {
					if i < len(actualOrder) {
						return actualOrder[i]
					} else {
						return "missing"
					}
				}(),
				key)
		}
	}
}

func TestIntegration_ThreadSafeDocument(t *testing.T) {
	// Test that document building is thread-safe
	builder := New()

	// Simulate concurrent table additions
	done := make(chan bool, 2)

	go func() {
		builder.Table("Table1", []map[string]any{
			{"A": 1, "B": 2},
		}, WithKeys("A", "B"))
		done <- true
	}()

	go func() {
		builder.Table("Table2", []map[string]any{
			{"X": 10, "Y": 20},
		}, WithKeys("X", "Y"))
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	doc := builder.Build()
	contents := doc.GetContents()

	if len(contents) != 2 {
		t.Fatalf("expected 2 contents, got %d", len(contents))
	}

	// Verify both tables were added correctly
	for i, content := range contents {
		table, ok := content.(*TableContent)
		if !ok {
			t.Fatalf("content[%d] should be TableContent", i)
		}

		if table.Title() == "" {
			t.Errorf("content[%d] should have a title", i)
		}

		if len(table.Schema().GetKeyOrder()) == 0 {
			t.Errorf("content[%d] should have keys", i)
		}
	}
}
