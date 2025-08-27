package output

import (
	"context"
	"strings"
	"testing"
)

func TestMermaidRenderer_ChartSupport(t *testing.T) {
	tests := map[string]struct {
		chart       *ChartContent
		contains    []string
		notContains []string
	}{"gantt chart rendering": {

		chart: NewGanttChart("Project Timeline", []GanttTask{
			{
				ID:        "task1",
				Title:     "Design Phase",
				StartDate: "2024-01-01",
				Duration:  "5d",
				Status:    "done",
				Section:   "Planning",
			},
			{
				ID:        "task2",
				Title:     "Development Phase",
				StartDate: "2024-01-08",
				Duration:  "10d",
				Status:    "active",
				Section:   "Implementation",
			},
		}),
		contains: []string{
			"gantt",
			"title Project Timeline",
			"dateFormat YYYY-MM-DD",
			"section Planning",
			"Design Phase :done, task1, 2024-01-01, 5d",
			"section Implementation",
			"Development Phase :active, task2, 2024-01-08, 10d",
		},
		notContains: []string{
			"graph TD",
			"pie",
		},
	}, "pie chart rendering": {

		chart: NewPieChart("Browser Market Share", []PieSlice{
			{Label: "Chrome", Value: 65.2},
			{Label: "Firefox", Value: 18.8},
			{Label: "Safari", Value: 9.6},
		}, true),
		contains: []string{
			"pie showData",
			"title Browser Market Share",
			`"Chrome" : 65.20`,
			`"Firefox" : 18.80`,
			`"Safari" : 9.60`,
		},
		notContains: []string{
			"gantt",
			"graph TD",
		},
	}, "pie chart without data": {

		chart: NewPieChart("Simple Pie", []PieSlice{
			{Label: "A", Value: 50},
			{Label: "B", Value: 50},
		}, false),
		contains: []string{
			"pie\n", // pie without showData
			"title Simple Pie",
			`"A" : 50.00`,
			`"B" : 50.00`,
		},
		notContains: []string{
			"showData",
			"gantt",
		},
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			renderer := &mermaidRenderer{}
			doc := New().AddContent(tt.chart).Build()

			result, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			output := string(result)

			// Check for expected content
			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Output should contain %q, got:\n%s", expected, output)
				}
			}

			// Check for content that should not be present
			for _, notExpected := range tt.notContains {
				if strings.Contains(output, notExpected) {
					t.Errorf("Output should not contain %q, got:\n%s", notExpected, output)
				}
			}
		})
	}
}

func TestMermaidRenderer_MixedContent(t *testing.T) {
	// Test document with both flowchart and specialized charts
	doc := New().
		GanttChart("Project", []GanttTask{
			{Title: "Task 1", StartDate: "2024-01-01", Duration: "3d"},
		}).
		Graph("Flow", []Edge{
			{From: "A", To: "B", Label: "connects"},
		}).
		PieChart("Data", []PieSlice{
			{Label: "X", Value: 60},
			{Label: "Y", Value: 40},
		}, false).
		Build()

	renderer := &mermaidRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := string(result)

	// Should contain all chart types
	expectedParts := []string{
		"gantt",
		"title Project",
		"Task 1",
		"graph TD",
		"A -->|connects| B",
		"pie",
		"title Data",
		`"X" : 60.00`,
	}

	for _, expected := range expectedParts {
		if !strings.Contains(output, expected) {
			t.Errorf("Mixed content should contain %q, got:\n%s", expected, output)
		}
	}
}

func TestMermaidRenderer_GanttSections(t *testing.T) {
	tasks := []GanttTask{
		{Title: "Task 1", StartDate: "2024-01-01", Duration: "3d", Section: "Phase 1"},
		{Title: "Task 2", StartDate: "2024-01-05", Duration: "2d", Section: "Phase 1"},
		{Title: "Task 3", StartDate: "2024-01-08", Duration: "4d", Section: "Phase 2"},
		{Title: "Task 4", StartDate: "2024-01-10", Duration: "1d"}, // No section
	}

	chart := NewGanttChart("Multi-Phase Project", tasks)
	doc := New().AddContent(chart).Build()

	renderer := &mermaidRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := string(result)

	// Should contain both sections
	expectedSections := []string{
		"section Phase 1",
		"section Phase 2",
		"section Tasks", // Default section for Task 4
	}

	for _, expected := range expectedSections {
		if !strings.Contains(output, expected) {
			t.Errorf("Output should contain section %q, got:\n%s", expected, output)
		}
	}

	// Verify tasks are in correct sections
	if !strings.Contains(output, "Task 1") || !strings.Contains(output, "Task 2") {
		t.Error("Phase 1 tasks should be present")
	}
	if !strings.Contains(output, "Task 3") {
		t.Error("Phase 2 tasks should be present")
	}
	if !strings.Contains(output, "Task 4") {
		t.Error("Default section task should be present")
	}
}

func TestMermaidRenderer_GanttTaskProperties(t *testing.T) {
	tasks := []GanttTask{
		{
			ID:        "design",
			Title:     "Design Work",
			StartDate: "2024-01-01",
			Duration:  "5d",
			Status:    "done",
		},
		{
			Title:     "Development",
			StartDate: "2024-01-08",
			EndDate:   "2024-01-15", // Using EndDate instead of Duration
			Status:    "active",
		},
		{
			Title:     "Testing",
			StartDate: "2024-01-16",
			// No duration or end date
		},
	}

	chart := NewGanttChart("Task Properties Test", tasks)
	doc := New().AddContent(chart).Build()

	renderer := &mermaidRenderer{}
	result, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	output := string(result)

	// Check specific task formatting
	expectedTaskFormats := []string{
		"Design Work :done, design, 2024-01-01, 5d",
		"Development :active, 2024-01-08, 2024-01-15",
		"Testing :2024-01-16", // No status, ID, duration, or end date
	}

	for _, expected := range expectedTaskFormats {
		if !strings.Contains(output, expected) {
			t.Errorf("Output should contain task format %q, got:\n%s", expected, output)
		}
	}
}

func TestMermaidRenderer_EmptyCharts(t *testing.T) {
	tests := map[string]struct {
		chart *ChartContent
	}{"empty gantt": {

		chart: NewGanttChart("Empty Project", []GanttTask{}),
	}, "empty pie": {

		chart: NewPieChart("Empty Pie", []PieSlice{}, false),
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			doc := New().AddContent(tt.chart).Build()
			renderer := &mermaidRenderer{}

			result, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			output := string(result)

			// Should still have proper chart structure even if empty
			if tt.chart.GetChartType() == ChartTypeGantt {
				if !strings.Contains(output, "gantt") {
					t.Error("Empty Gantt should still have gantt header")
				}
			} else if tt.chart.GetChartType() == ChartTypePie {
				if !strings.Contains(output, "pie") {
					t.Error("Empty pie should still have pie header")
				}
			}
		})
	}
}
