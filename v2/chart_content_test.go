package output

import (
	"testing"
)

func TestChartContent_Creation(t *testing.T) {
	tests := map[string]struct {
		title     string
		chartType string
		data      any
	}{"gantt chart": {

		title:     "Project Timeline",
		chartType: ChartTypeGantt,
		data: &GanttData{
			DateFormat: "YYYY-MM-DD",
			AxisFormat: "%Y-%m-%d",
			Tasks: []GanttTask{
				{Title: "Task 1", StartDate: "2024-01-01", Duration: "3d"},
			},
		},
	}, "pie chart": {

		title:     "Market Share",
		chartType: ChartTypePie,
		data: &PieData{
			ShowData: true,
			Slices: []PieSlice{
				{Label: "Product A", Value: 45.5},
				{Label: "Product B", Value: 30.2},
			},
		},
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			chart := NewChartContent(tt.title, tt.chartType, tt.data)

			if chart.GetTitle() != tt.title {
				t.Errorf("GetTitle() = %v, want %v", chart.GetTitle(), tt.title)
			}

			if chart.GetChartType() != tt.chartType {
				t.Errorf("GetChartType() = %v, want %v", chart.GetChartType(), tt.chartType)
			}

			if chart.Type() != ContentTypeRaw {
				t.Errorf("Type() = %v, want %v", chart.Type(), ContentTypeRaw)
			}

			if chart.ID() == "" {
				t.Error("ID() returned empty string")
			}
		})
	}
}

func TestNewGanttChart(t *testing.T) {
	tasks := []GanttTask{
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
	}

	chart := NewGanttChart("Project Timeline", tasks)

	if chart.GetChartType() != ChartTypeGantt {
		t.Errorf("GetChartType() = %v, want %v", chart.GetChartType(), ChartTypeGantt)
	}

	ganttData, ok := chart.GetData().(*GanttData)
	if !ok {
		t.Fatal("GetData() did not return *GanttData")
	}

	if len(ganttData.Tasks) != 2 {
		t.Errorf("len(Tasks) = %v, want %v", len(ganttData.Tasks), 2)
	}

	if ganttData.Tasks[0].Title != "Design Phase" {
		t.Errorf("Tasks[0].Title = %v, want %v", ganttData.Tasks[0].Title, "Design Phase")
	}
}

func TestNewPieChart(t *testing.T) {
	slices := []PieSlice{
		{Label: "Chrome", Value: 65.2},
		{Label: "Firefox", Value: 18.8},
		{Label: "Safari", Value: 9.6},
		{Label: "Edge", Value: 4.1},
		{Label: "Other", Value: 2.3},
	}

	chart := NewPieChart("Browser Market Share", slices, true)

	if chart.GetChartType() != ChartTypePie {
		t.Errorf("GetChartType() = %v, want %v", chart.GetChartType(), ChartTypePie)
	}

	pieData, ok := chart.GetData().(*PieData)
	if !ok {
		t.Fatal("GetData() did not return *PieData")
	}

	if !pieData.ShowData {
		t.Error("ShowData should be true")
	}

	if len(pieData.Slices) != 5 {
		t.Errorf("len(Slices) = %v, want %v", len(pieData.Slices), 5)
	}

	if pieData.Slices[0].Label != "Chrome" {
		t.Errorf("Slices[0].Label = %v, want %v", pieData.Slices[0].Label, "Chrome")
	}

	if pieData.Slices[0].Value != 65.2 {
		t.Errorf("Slices[0].Value = %v, want %v", pieData.Slices[0].Value, 65.2)
	}
}

func TestChartContent_AppendText(t *testing.T) {
	tests := map[string]struct {
		chart    *ChartContent
		contains []string
	}{"gantt chart text": {

		chart: NewGanttChart("Project Timeline", []GanttTask{
			{Title: "Task 1", StartDate: "2024-01-01", Duration: "3d"},
			{Title: "Task 2", StartDate: "2024-01-05", Duration: "2d"},
		}),
		contains: []string{"Project Timeline", "Chart Type: gantt", "Task 1", "Task 2"},
	}, "pie chart text": {

		chart: NewPieChart("Market Share", []PieSlice{
			{Label: "Product A", Value: 45.5},
			{Label: "Product B", Value: 30.2},
		}, true),
		contains: []string{"Market Share", "Chart Type: pie", "Product A: 45.50", "Product B: 30.20"},
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var buf []byte
			result, err := tt.chart.AppendText(buf)
			if err != nil {
				t.Fatalf("AppendText() error = %v", err)
			}

			output := string(result)
			for _, expected := range tt.contains {
				if !contains(output, expected) {
					t.Errorf("AppendText() output should contain %q, got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestBuilder_ChartMethods(t *testing.T) {
	builder := New()

	// Test generic Chart method
	builder.Chart("Custom Chart", "custom", map[string]any{"data": "test"})

	// Test GanttChart method
	ganttTasks := []GanttTask{
		{Title: "Task 1", StartDate: "2024-01-01", Duration: "3d"},
	}
	builder.GanttChart("Project Timeline", ganttTasks)

	// Test PieChart method
	pieSlices := []PieSlice{
		{Label: "A", Value: 50},
		{Label: "B", Value: 50},
	}
	builder.PieChart("Distribution", pieSlices, true)

	doc := builder.Build()
	contents := doc.GetContents()

	if len(contents) != 3 {
		t.Errorf("Expected 3 contents, got %d", len(contents))
	}

	// Verify first content is generic chart
	chart1, ok := contents[0].(*ChartContent)
	if !ok {
		t.Error("First content should be ChartContent")
	} else if chart1.GetChartType() != "custom" {
		t.Errorf("First chart type = %v, want custom", chart1.GetChartType())
	}

	// Verify second content is Gantt chart
	chart2, ok := contents[1].(*ChartContent)
	if !ok {
		t.Error("Second content should be ChartContent")
	} else if chart2.GetChartType() != ChartTypeGantt {
		t.Errorf("Second chart type = %v, want %v", chart2.GetChartType(), ChartTypeGantt)
	}

	// Verify third content is pie chart
	chart3, ok := contents[2].(*ChartContent)
	if !ok {
		t.Error("Third content should be ChartContent")
	} else if chart3.GetChartType() != ChartTypePie {
		t.Errorf("Third chart type = %v, want %v", chart3.GetChartType(), ChartTypePie)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}
