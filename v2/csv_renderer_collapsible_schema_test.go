package output

import (
	"context"
	"encoding/csv"
	"strings"
	"testing"
)

// TestCSVRenderer_CollapsibleSectionMultipleSchemas reproduces T-1315.
//
// A DefaultCollapsibleSection that contains multiple TableContent entries with
// different key orders previously wrote headers only for the first table
// (renderCollapsibleSectionCSV tracked a single boolean hasWrittenHeaders).
// Later tables then rendered their rows without their own headers, producing
// malformed CSV where later values appear under the previous table's columns.
//
// Expected: like the top-level CSV path, headers are (re)written whenever a
// table's key order differs from the previously rendered table.
func TestCSVRenderer_CollapsibleSectionMultipleSchemas(t *testing.T) {
	employees, err := NewTableContent("employees", []map[string]any{
		{"Name": "Alice", "Department": "Engineering"},
		{"Name": "Bob", "Department": "Marketing"},
	}, WithKeys("Name", "Department"))
	if err != nil {
		t.Fatalf("failed to create employees table: %v", err)
	}

	projects, err := NewTableContent("projects", []map[string]any{
		{"Project": "Alpha", "Status": "Active", "Budget": 100000},
		{"Project": "Beta", "Status": "Complete", "Budget": 50000},
	}, WithKeys("Project", "Status", "Budget"))
	if err != nil {
		t.Fatalf("failed to create projects table: %v", err)
	}

	section := NewCollapsibleSection("Quarterly Report", []Content{employees, projects})
	doc := New().AddContent(section).Build()

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

	// The second table has a different key order, so its header row must be
	// present in the output.
	foundFirstHeader := false
	foundSecondHeader := false
	for _, row := range records {
		if len(row) >= 2 && row[0] == "Name" && row[1] == "Department" {
			foundFirstHeader = true
		}
		if len(row) >= 3 && row[0] == "Project" && row[1] == "Status" && row[2] == "Budget" {
			foundSecondHeader = true
		}
	}

	if !foundFirstHeader {
		t.Errorf("Expected headers for first table inside collapsible section, got:\n%s", string(result))
	}
	if !foundSecondHeader {
		t.Errorf("Expected headers for second table with different schema inside collapsible section, got:\n%s", string(result))
	}
}

// TestCSVRenderer_CollapsibleSectionSameSchema ensures the fix does not cause
// redundant header rows when consecutive tables share a key order. Headers
// should be written exactly once for the contiguous run of same-schema tables.
func TestCSVRenderer_CollapsibleSectionSameSchema(t *testing.T) {
	team1, err := NewTableContent("team1", []map[string]any{
		{"Name": "Alice", "Score": 95},
	}, WithKeys("Name", "Score"))
	if err != nil {
		t.Fatalf("failed to create team1 table: %v", err)
	}

	team2, err := NewTableContent("team2", []map[string]any{
		{"Name": "Bob", "Score": 87},
	}, WithKeys("Name", "Score"))
	if err != nil {
		t.Fatalf("failed to create team2 table: %v", err)
	}

	section := NewCollapsibleSection("Scores", []Content{team1, team2})
	doc := New().AddContent(section).Build()

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

	headerCount := 0
	for _, row := range records {
		if len(row) >= 2 && row[0] == "Name" && row[1] == "Score" {
			headerCount++
		}
	}

	if headerCount != 1 {
		t.Errorf("Expected exactly 1 header row for same-schema tables in collapsible section, got %d.\nOutput:\n%s", headerCount, string(result))
	}
}
