package output

import (
	"context"
	"strings"
	"testing"
)

func TestIntegration_DocumentWithTables(t *testing.T) {
	skipIfNotIntegration(t)

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
	skipIfNotIntegration(t)

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
	skipIfNotIntegration(t)

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

// TestIntegration_TransformationsToJSON tests per-content transformations with JSON rendering
func TestIntegration_TransformationsToJSON(t *testing.T) {
	skipIfNotIntegration(t)

	// Create data
	data := []map[string]any{
		{"name": "Charlie", "age": 35, "city": "Chicago"},
		{"name": "Alice", "age": 30, "city": "Austin"},
		{"name": "Bob", "age": 25, "city": "Boston"},
		{"name": "Diana", "age": 40, "city": "Denver"},
	}

	// Build document with transformations
	builder := New()
	builder.Table("users", data,
		WithKeys("name", "age", "city"),
		WithTransformations(
			NewFilterOp(func(r Record) bool {
				return r["age"].(int) >= 30
			}),
			NewSortOp(SortKey{Column: "name", Direction: Ascending}),
		),
	)

	doc := builder.Build()

	// Render to JSON
	renderer := &jsonRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := string(result)

	// Verify that filtering worked (only age >= 30)
	if !strings.Contains(output, "Alice") {
		t.Error("expected Alice in output")
	}
	if !strings.Contains(output, "Charlie") {
		t.Error("expected Charlie in output")
	}
	if !strings.Contains(output, "Diana") {
		t.Error("expected Diana in output")
	}
	if strings.Contains(output, "Bob") {
		t.Error("Bob should be filtered out (age 25)")
	}

	// Verify that sorting worked (Alice, Charlie, Diana)
	alicePos := strings.Index(output, "Alice")
	charliePos := strings.Index(output, "Charlie")
	dianaPos := strings.Index(output, "Diana")

	if alicePos == -1 || charliePos == -1 || dianaPos == -1 {
		t.Fatal("expected all filtered users in output")
	}

	if !(alicePos < charliePos && charliePos < dianaPos) {
		t.Error("expected users to be sorted alphabetically: Alice, Charlie, Diana")
	}
}

// TestIntegration_TransformationsToYAML tests per-content transformations with YAML rendering
func TestIntegration_TransformationsToYAML(t *testing.T) {
	skipIfNotIntegration(t)

	// Create data
	data := []map[string]any{
		{"product": "Widget", "price": 25.50, "stock": 100},
		{"product": "Gadget", "price": 45.00, "stock": 50},
		{"product": "Doohickey", "price": 15.75, "stock": 200},
	}

	// Build document with transformations
	builder := New()
	builder.Table("products", data,
		WithKeys("product", "price", "stock"),
		WithTransformations(
			NewSortOp(SortKey{Column: "price", Direction: Descending}),
			NewLimitOp(2),
		),
	)

	doc := builder.Build()

	// Render to YAML
	renderer := &yamlRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := string(result)

	// Verify that sorting and limiting worked (top 2 by price)
	if !strings.Contains(output, "Gadget") {
		t.Error("expected Gadget in output (highest price)")
	}
	if !strings.Contains(output, "Widget") {
		t.Error("expected Widget in output (second highest price)")
	}
	if strings.Contains(output, "Doohickey") {
		t.Error("Doohickey should be limited out (lowest price)")
	}
}

// TestIntegration_MultipleTables_DifferentTransformations tests multiple tables with different transformations
func TestIntegration_MultipleTables_DifferentTransformations(t *testing.T) {
	skipIfNotIntegration(t)

	// Create data for users table
	users := []map[string]any{
		{"name": "Alice", "role": "Admin", "salary": 80000},
		{"name": "Bob", "role": "User", "salary": 50000},
		{"name": "Charlie", "role": "Admin", "salary": 90000},
	}

	// Create data for tasks table
	tasks := []map[string]any{
		{"task": "Write tests", "priority": 1, "done": true},
		{"task": "Fix bugs", "priority": 2, "done": false},
		{"task": "Add feature", "priority": 1, "done": false},
		{"task": "Review code", "priority": 3, "done": true},
	}

	// Build document with different transformations per table
	builder := New()
	builder.Table("admins", users,
		WithKeys("name", "salary"),
		WithTransformations(
			NewFilterOp(func(r Record) bool {
				return r["role"].(string) == "Admin"
			}),
			NewSortOp(SortKey{Column: "salary", Direction: Descending}),
		),
	)
	builder.Table("pending_tasks", tasks,
		WithKeys("priority", "task"),
		WithTransformations(
			NewFilterOp(func(r Record) bool {
				return !r["done"].(bool)
			}),
			NewSortOp(SortKey{Column: "priority", Direction: Ascending}),
		),
	)

	doc := builder.Build()

	// Render to JSON
	renderer := &jsonRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := string(result)

	// Verify admins table (only Admin role, sorted by salary desc)
	if !strings.Contains(output, "Charlie") {
		t.Error("expected Charlie in admins (Admin role)")
	}
	if !strings.Contains(output, "Alice") {
		t.Error("expected Alice in admins (Admin role)")
	}
	if strings.Contains(output, "Bob") {
		t.Error("Bob should be filtered out (User role)")
	}

	// Verify pending_tasks table (only undone tasks)
	if !strings.Contains(output, "Fix bugs") {
		t.Error("expected 'Fix bugs' in pending tasks")
	}
	if !strings.Contains(output, "Add feature") {
		t.Error("expected 'Add feature' in pending tasks")
	}
	if strings.Contains(output, "Write tests") {
		t.Error("'Write tests' should be filtered out (done)")
	}
	if strings.Contains(output, "Review code") {
		t.Error("'Review code' should be filtered out (done)")
	}
}

// TestIntegration_MixedContent_WithAndWithoutTransformations tests mixed content types
func TestIntegration_MixedContent_WithAndWithoutTransformations(t *testing.T) {
	skipIfNotIntegration(t)

	// Create data
	data := []map[string]any{
		{"name": "Alice", "score": 95},
		{"name": "Bob", "score": 85},
		{"name": "Charlie", "score": 90},
	}

	// Build document with mixed content
	builder := New()
	builder.Text("Top Performers Report")
	builder.Table("top_performers", data,
		WithKeys("name", "score"),
		WithTransformations(
			NewFilterOp(func(r Record) bool {
				return r["score"].(int) >= 90
			}),
			NewSortOp(SortKey{Column: "score", Direction: Descending}),
		),
	)
	builder.Text("End of Report")

	doc := builder.Build()

	// Render to JSON
	renderer := &jsonRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := string(result)

	// Verify structure
	if !strings.Contains(output, "Top Performers Report") {
		t.Error("expected header text in output")
	}
	if !strings.Contains(output, "End of Report") {
		t.Error("expected footer text in output")
	}

	// Verify transformations worked on table
	if !strings.Contains(output, "Alice") {
		t.Error("expected Alice in output (score 95)")
	}
	if !strings.Contains(output, "Charlie") {
		t.Error("expected Charlie in output (score 90)")
	}
	if strings.Contains(output, "Bob") {
		t.Error("Bob should be filtered out (score 85)")
	}
}

// TestIntegration_TransformationChains_FilterSortLimit tests complex transformation chains
func TestIntegration_TransformationChains_FilterSortLimit(t *testing.T) {
	skipIfNotIntegration(t)

	// Create data with 10 records
	data := []map[string]any{
		{"id": 1, "value": 10, "active": true},
		{"id": 2, "value": 25, "active": false},
		{"id": 3, "value": 30, "active": true},
		{"id": 4, "value": 15, "active": true},
		{"id": 5, "value": 40, "active": false},
		{"id": 6, "value": 35, "active": true},
		{"id": 7, "value": 20, "active": true},
		{"id": 8, "value": 45, "active": false},
		{"id": 9, "value": 50, "active": true},
		{"id": 10, "value": 5, "active": true},
	}

	// Build document with transformation chain: filter → sort → limit
	builder := New()
	builder.Table("top_active", data,
		WithKeys("id", "value", "active"),
		WithTransformations(
			NewFilterOp(func(r Record) bool {
				return r["active"].(bool)
			}),
			NewSortOp(SortKey{Column: "value", Direction: Descending}),
			NewLimitOp(3),
		),
	)

	doc := builder.Build()

	// Render to JSON
	renderer := &jsonRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	output := string(result)

	// Should have top 3 active records by value: id=9 (50), id=6 (35), id=3 (30)
	if !strings.Contains(output, `"id": 9`) {
		t.Error("expected id 9 in output (value 50, active)")
	}
	if !strings.Contains(output, `"id": 6`) {
		t.Error("expected id 6 in output (value 35, active)")
	}
	if !strings.Contains(output, `"id": 3`) {
		t.Error("expected id 3 in output (value 30, active)")
	}

	// Should NOT have lower values or inactive records
	if strings.Contains(output, `"id": 1`) {
		t.Error("id 1 should be limited out (lower value)")
	}
	if strings.Contains(output, `"id": 2`) {
		t.Error("id 2 should be filtered out (inactive)")
	}
	if strings.Contains(output, `"id": 7`) {
		t.Error("id 7 should be limited out (lower value)")
	}
}

// TestIntegration_OriginalDataUnchanged tests that original document data is unchanged after rendering
func TestIntegration_OriginalDataUnchanged(t *testing.T) {
	skipIfNotIntegration(t)

	// Create data
	originalData := []map[string]any{
		{"name": "Alice", "age": 30},
		{"name": "Bob", "age": 25},
		{"name": "Charlie", "age": 35},
	}

	// Build document with transformations
	builder := New()
	builder.Table("users", originalData,
		WithKeys("name", "age"),
		WithTransformations(
			NewFilterOp(func(r Record) bool {
				return r["age"].(int) >= 30
			}),
		),
	)

	doc := builder.Build()

	// Render once
	renderer := &jsonRenderer{}
	_, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("First render failed: %v", err)
	}

	// Verify original data is unchanged
	contents := doc.GetContents()
	if len(contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(contents))
	}

	table, ok := contents[0].(*TableContent)
	if !ok {
		t.Fatal("expected TableContent")
	}

	records := table.Records()
	if len(records) != 3 {
		t.Errorf("expected original 3 records, got %d", len(records))
	}

	// Verify all original records still present
	names := make(map[string]bool)
	for _, record := range records {
		names[record["name"].(string)] = true
	}

	if !names["Alice"] || !names["Bob"] || !names["Charlie"] {
		t.Error("original data was modified - some records missing")
	}

	// Render again - should produce same result
	result2, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Second render failed: %v", err)
	}

	output2 := string(result2)
	if !strings.Contains(output2, "Alice") {
		t.Error("second render should still have Alice")
	}
	if strings.Contains(output2, "Bob") {
		t.Error("second render should still filter out Bob")
	}
}
