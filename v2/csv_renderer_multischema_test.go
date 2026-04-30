package output

import (
	"context"
	"encoding/csv"
	"strings"
	"testing"
)

func TestCSVRenderer_MultipleTablesWithDifferentSchemas(t *testing.T) {
	doc := New().
		Table("employees", []map[string]any{
			{"Name": "Alice", "Department": "Engineering"},
			{"Name": "Bob", "Department": "Marketing"},
		}, WithKeys("Name", "Department")).
		Table("projects", []map[string]any{
			{"Project": "Alpha", "Status": "Active", "Budget": 100000},
			{"Project": "Beta", "Status": "Complete", "Budget": 50000},
		}, WithKeys("Project", "Status", "Budget")).
		Build()

	renderer := &csvRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	csvReader := csv.NewReader(strings.NewReader(string(result)))
	csvReader.FieldsPerRecord = -1 // allow variable field counts
	records, err := csvReader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	// Expected structure:
	// Row 0: Name,Department (header for table 1)
	// Row 1: Alice,Engineering
	// Row 2: Bob,Marketing
	// Row 3: (blank separator)
	// Row 4: Project,Status,Budget (header for table 2)
	// Row 5: Alpha,Active,100000
	// Row 6: Beta,Complete,50000

	// Find the second header row
	foundSecondHeader := false
	for _, row := range records {
		if len(row) >= 3 && row[0] == "Project" && row[1] == "Status" && row[2] == "Budget" {
			foundSecondHeader = true
			break
		}
	}

	if !foundSecondHeader {
		t.Errorf("Expected headers for second table with different schema, got:\n%s", string(result))
	}
}

func TestCSVRenderer_MultipleTablesWithSameSchema(t *testing.T) {
	doc := New().
		Table("team1", []map[string]any{
			{"Name": "Alice", "Score": 95},
		}, WithKeys("Name", "Score")).
		Table("team2", []map[string]any{
			{"Name": "Bob", "Score": 87},
		}, WithKeys("Name", "Score")).
		Build()

	renderer := &csvRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	csvReader := csv.NewReader(strings.NewReader(string(result)))
	csvReader.FieldsPerRecord = -1
	records, err := csvReader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	// With same schema, headers should only appear once
	headerCount := 0
	for _, row := range records {
		if len(row) >= 2 && row[0] == "Name" && row[1] == "Score" {
			headerCount++
		}
	}

	if headerCount != 1 {
		t.Errorf("Expected exactly 1 header row for tables with same schema, got %d.\nOutput:\n%s", headerCount, string(result))
	}
}
