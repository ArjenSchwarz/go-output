package output

import (
	"context"
	"strings"
	"testing"
)

// TestMermaidGanttFormat_HTML verifies that Gantt charts render correctly in HTML format
// with proper mermaid class and syntax
func TestMermaidGanttFormat_HTML(t *testing.T) {
	// Create a realistic Gantt chart based on user's example
	doc := New().
		GanttChart("transit-gateway-prod-sorpets-rt - Create event - Started 2023-12-05T16:22:32+11:00", []GanttTask{
			{
				Title:     "Stack REVIEW_IN_PROGRESS",
				StartDate: "2023-12-05",
				Duration:  "0s",
			},
			{
				Title:     "Stack CREATE_IN_PROGRESS",
				StartDate: "2023-12-05",
				Duration:  "0s",
			},
			{
				Title:     "StageRouteTable",
				StartDate: "2023-12-05",
				Duration:  "7s",
			},
		}).
		Build()

	renderer := &htmlRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := string(result)

	// Verify correct HTML format
	checks := map[string]string{
		"Has mermaid pre tag":     `<pre class="mermaid">`,
		"Has gantt chart type":    "gantt",
		"Has proper dateFormat":   "dateFormat YYYY-MM-DD",
		"Has chart title":         "title transit-gateway-prod-sorpets-rt",
		"Has first task":          "Stack REVIEW_IN_PROGRESS",
		"Has closing pre tag":     "</pre>",
		"Not plain text format":   "Chart Type:", // Should NOT contain this
		"Proper Gantt task lines": ":",           // Tasks should have colons
	}

	for name, expected := range checks {
		if name == "Not plain text format" {
			if strings.Contains(output, expected) {
				t.Errorf("%s: Output should NOT contain plain text %q", name, expected)
			}
		} else {
			if !strings.Contains(output, expected) {
				t.Errorf("%s: Output should contain %q\nGot:\n%s", name, expected, output)
			}
		}
	}

	// Verify proper mermaid syntax structure (not plain text)
	lines := strings.Split(output, "\n")
	foundGantt := false
	foundTitle := false
	foundDateFormat := false
	foundTask := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "gantt" {
			foundGantt = true
		}
		if strings.HasPrefix(trimmed, "title ") {
			foundTitle = true
		}
		if strings.HasPrefix(trimmed, "dateFormat ") {
			foundDateFormat = true
		}
		// Task format: "Task Name :date, duration"
		if strings.Contains(trimmed, " :") && strings.Contains(trimmed, "2023-12-05") {
			foundTask = true
		}
	}

	if !foundGantt {
		t.Error("Missing 'gantt' line in proper mermaid format")
	}
	if !foundTitle {
		t.Error("Missing 'title' line in proper mermaid format")
	}
	if !foundDateFormat {
		t.Error("Missing 'dateFormat' line in proper mermaid format")
	}
	if !foundTask {
		t.Error("Missing properly formatted task line (should be 'Task :date, duration')")
	}
}

// TestMermaidGanttFormat_Markdown verifies that Gantt charts render correctly in Markdown format
// with proper code fence
func TestMermaidGanttFormat_Markdown(t *testing.T) {
	// Create same Gantt chart as HTML test
	doc := New().
		GanttChart("transit-gateway-prod-sorpets-rt - Create event", []GanttTask{
			{
				Title:     "Stack REVIEW_IN_PROGRESS",
				StartDate: "2023-12-05",
				Duration:  "0s",
			},
			{
				Title:     "Stack CREATE_IN_PROGRESS",
				StartDate: "2023-12-05",
				Duration:  "0s",
			},
			{
				Title:     "StageRouteTable",
				StartDate: "2023-12-05",
				Duration:  "7s",
			},
		}).
		Build()

	renderer := &markdownRenderer{headingLevel: 1}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := string(result)

	// Verify correct Markdown format with code fence
	checks := map[string]string{
		"Has mermaid code fence opening": "```mermaid",
		"Has gantt chart type":           "gantt",
		"Has proper dateFormat":          "dateFormat YYYY-MM-DD",
		"Has chart title":                "title transit-gateway-prod-sorpets-rt",
		"Has first task":                 "Stack REVIEW_IN_PROGRESS",
		"Has code fence closing":         "```",
		"Not plain text format":          "Chart Type:", // Should NOT contain this
	}

	for name, expected := range checks {
		if name == "Not plain text format" {
			if strings.Contains(output, expected) {
				t.Errorf("%s: Output should NOT contain plain text %q", name, expected)
			}
		} else {
			if !strings.Contains(output, expected) {
				t.Errorf("%s: Output should contain %q\nGot:\n%s", name, expected, output)
			}
		}
	}

	// Verify code fence structure
	if !strings.HasPrefix(strings.TrimSpace(output), "```mermaid") {
		t.Error("Output should start with ```mermaid code fence")
	}
	if !strings.HasSuffix(strings.TrimSpace(output), "```") {
		t.Error("Output should end with ``` closing fence")
	}

	// Count backticks - should have opening and closing
	backtickCount := strings.Count(output, "```")
	if backtickCount != 2 {
		t.Errorf("Expected exactly 2 code fence markers (```), got %d", backtickCount)
	}

	// Verify no plain text "Visual timeline" or "Chart Type" format
	plainTextIndicators := []string{
		"Visual timeline",
		"Chart Type:",
		"Chart Type: gantt",
	}
	for _, indicator := range plainTextIndicators {
		if strings.Contains(output, indicator) {
			t.Errorf("Output contains plain text format indicator %q, should be mermaid syntax", indicator)
		}
	}
}

// TestMermaidTaskFormat verifies proper Gantt task format
// Format should be: "Task Title :status, id, date, duration"
// NOT: "Task Title (date, duration)"
func TestMermaidTaskFormat(t *testing.T) {
	tests := map[string]struct {
		task             GanttTask
		expectedFragment string // Part of the expected task line
		notExpected      string // Should NOT appear
	}{
		"task with status and ID": {
			task: GanttTask{
				ID:        "task1",
				Title:     "Design Phase",
				StartDate: "2024-01-01",
				Duration:  "5d",
				Status:    "done",
			},
			expectedFragment: "Design Phase :done, task1, 2024-01-01, 5d",
			notExpected:      "Design Phase (2024-01-01, 5d)", // Wrong format
		},
		"task without status": {
			task: GanttTask{
				Title:     "Development",
				StartDate: "2024-01-08",
				Duration:  "10d",
			},
			expectedFragment: "Development :2024-01-08, 10d",
			notExpected:      "Development (2024-01-08, 10d)", // Wrong format
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			doc := New().
				GanttChart("Test", []GanttTask{tc.task}).
				Build()

			// Test both HTML and Markdown
			t.Run("HTML", func(t *testing.T) {
				renderer := &htmlRenderer{}
				result, err := renderer.Render(context.Background(), doc)
				if err != nil {
					t.Fatalf("HTML Render() error = %v", err)
				}
				output := string(result)

				if !strings.Contains(output, tc.expectedFragment) {
					t.Errorf("HTML output should contain proper task format %q\nGot:\n%s",
						tc.expectedFragment, output)
				}
				if strings.Contains(output, tc.notExpected) {
					t.Errorf("HTML output should NOT contain wrong format %q\nGot:\n%s",
						tc.notExpected, output)
				}
			})

			t.Run("Markdown", func(t *testing.T) {
				renderer := &markdownRenderer{headingLevel: 1}
				result, err := renderer.Render(context.Background(), doc)
				if err != nil {
					t.Fatalf("Markdown Render() error = %v", err)
				}
				output := string(result)

				if !strings.Contains(output, tc.expectedFragment) {
					t.Errorf("Markdown output should contain proper task format %q\nGot:\n%s",
						tc.expectedFragment, output)
				}
				if strings.Contains(output, tc.notExpected) {
					t.Errorf("Markdown output should NOT contain wrong format %q\nGot:\n%s",
						tc.notExpected, output)
				}
			})
		})
	}
}
