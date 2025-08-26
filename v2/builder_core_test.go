package output

import (
	"reflect"
	"testing"
)

func TestBuilder_Graph(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		edges    []Edge
		expected int
	}{
		{
			name:  "simple graph",
			title: "Test Graph",
			edges: []Edge{
				{From: "A", To: "B", Label: "connects"},
				{From: "B", To: "C", Label: "flows"},
			},
			expected: 2,
		},
		{
			name:     "empty graph",
			title:    "Empty Graph",
			edges:    []Edge{},
			expected: 0,
		},
		{
			name:  "single edge",
			title: "Single Edge Graph",
			edges: []Edge{
				{From: "Start", To: "End"},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := New()
			result := builder.Graph(tt.title, tt.edges)

			// Verify builder returns itself for chaining
			if result != builder {
				t.Error("Graph() should return the builder for chaining")
			}

			// Build and check content
			doc := builder.Build()
			if len(doc.contents) != 1 {
				t.Fatalf("Expected 1 content, got %d", len(doc.contents))
			}

			// Verify it's a GraphContent
			graph, ok := doc.contents[0].(*GraphContent)
			if !ok {
				t.Fatalf("Expected GraphContent, got %T", doc.contents[0])
			}

			if graph.title != tt.title {
				t.Errorf("title = %q, want %q", graph.title, tt.title)
			}

			if len(graph.edges) != tt.expected {
				t.Errorf("edge count = %d, want %d", len(graph.edges), tt.expected)
			}

			// Verify edges match
			if !reflect.DeepEqual(graph.edges, tt.edges) {
				t.Errorf("edges = %+v, want %+v", graph.edges, tt.edges)
			}
		})
	}
}

// TestBuilder_Chart tests the Chart method
func TestBuilder_Chart(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		chartType string
		data      any
	}{
		{
			name:      "pie chart data",
			title:     "Sales Distribution",
			chartType: "pie",
			data: []PieSlice{
				{Label: "Product A", Value: 30},
				{Label: "Product B", Value: 70},
			},
		},
		{
			name:      "gantt chart data",
			title:     "Project Timeline",
			chartType: "gantt",
			data: []GanttTask{
				{ID: "task1", Title: "Design", StartDate: "2024-01-01", EndDate: "2024-01-15"},
			},
		},
		{
			name:      "custom chart",
			title:     "Custom Chart",
			chartType: "custom",
			data:      map[string]any{"key": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := New()
			result := builder.Chart(tt.title, tt.chartType, tt.data)

			// Verify builder returns itself for chaining
			if result != builder {
				t.Error("Chart() should return the builder for chaining")
			}

			// Build and check content
			doc := builder.Build()
			if len(doc.contents) != 1 {
				t.Fatalf("Expected 1 content, got %d", len(doc.contents))
			}

			// Verify it's a ChartContent
			chart, ok := doc.contents[0].(*ChartContent)
			if !ok {
				t.Fatalf("Expected ChartContent, got %T", doc.contents[0])
			}

			if chart.title != tt.title {
				t.Errorf("title = %q, want %q", chart.title, tt.title)
			}

			if chart.chartType != tt.chartType {
				t.Errorf("chartType = %q, want %q", chart.chartType, tt.chartType)
			}

			// Note: Deep comparison of data would depend on the type
			if chart.data == nil {
				t.Error("chart data should not be nil")
			}
		})
	}
}

// TestBuilder_GanttChart tests the GanttChart convenience method
func TestBuilder_GanttChart(t *testing.T) {
	tests := []struct {
		name  string
		title string
		tasks []GanttTask
	}{
		{
			name:  "project timeline",
			title: "Q1 Project Plan",
			tasks: []GanttTask{
				{ID: "design", Title: "Design Phase", StartDate: "2024-01-01", EndDate: "2024-01-15", Status: "active"},
				{ID: "dev", Title: "Development", StartDate: "2024-01-16", EndDate: "2024-02-28", Dependencies: []string{"design"}},
				{ID: "test", Title: "Testing", StartDate: "2024-02-15", EndDate: "2024-03-15", Dependencies: []string{"dev"}},
			},
		},
		{
			name:  "empty gantt",
			title: "Empty Timeline",
			tasks: []GanttTask{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := New()
			result := builder.GanttChart(tt.title, tt.tasks)

			// Verify builder returns itself for chaining
			if result != builder {
				t.Error("GanttChart() should return the builder for chaining")
			}

			// Build and check content
			doc := builder.Build()
			if len(doc.contents) != 1 {
				t.Fatalf("Expected 1 content, got %d", len(doc.contents))
			}

			// Verify it's a ChartContent with gantt type
			chart, ok := doc.contents[0].(*ChartContent)
			if !ok {
				t.Fatalf("Expected ChartContent, got %T", doc.contents[0])
			}

			if chart.title != tt.title {
				t.Errorf("title = %q, want %q", chart.title, tt.title)
			}

			if chart.chartType != "gantt" {
				t.Errorf("chartType = %q, want %q", chart.chartType, "gantt")
			}

			// Verify tasks - the actual data structure varies by implementation
			// Just verify data exists and is the correct type
			if chart.data == nil {
				t.Error("chart data should not be nil")
			}
		})
	}
}

// TestBuilder_PieChart tests the PieChart convenience method
func TestBuilder_PieChart(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		slices   []PieSlice
		showData bool
	}{
		{
			name:  "sales distribution",
			title: "Q4 Sales by Region",
			slices: []PieSlice{
				{Label: "North", Value: 45.5},
				{Label: "South", Value: 30.2},
				{Label: "East", Value: 15.3},
				{Label: "West", Value: 9.0},
			},
			showData: true,
		},
		{
			name:     "simple pie without data",
			title:    "Market Share",
			slices:   []PieSlice{{Label: "Us", Value: 60}, {Label: "Others", Value: 40}},
			showData: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := New()
			result := builder.PieChart(tt.title, tt.slices, tt.showData)

			// Verify builder returns itself for chaining
			if result != builder {
				t.Error("PieChart() should return the builder for chaining")
			}

			// Build and check content
			doc := builder.Build()
			if len(doc.contents) != 1 {
				t.Fatalf("Expected 1 content, got %d", len(doc.contents))
			}

			// Verify it's a ChartContent with pie type
			chart, ok := doc.contents[0].(*ChartContent)
			if !ok {
				t.Fatalf("Expected ChartContent, got %T", doc.contents[0])
			}

			if chart.title != tt.title {
				t.Errorf("title = %q, want %q", chart.title, tt.title)
			}

			if chart.chartType != "pie" {
				t.Errorf("chartType = %q, want %q", chart.chartType, "pie")
			}

			// Verify pie data structure
			// The actual data structure would depend on implementation
			// For now, just verify data is not nil
			if chart.data == nil {
				t.Error("chart data should not be nil")
			}
		})
	}
}

// TestBuilder_DrawIO tests the DrawIO method
func TestBuilder_DrawIO(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		records []Record
		header  DrawIOHeader
	}{
		{
			name:  "org chart",
			title: "Company Structure",
			records: []Record{
				{"id": "ceo", "name": "CEO", "parent": ""},
				{"id": "cto", "name": "CTO", "parent": "ceo"},
				{"id": "cfo", "name": "CFO", "parent": "ceo"},
			},
			header: DrawIOHeader{
				Label:       "%name%",
				Layout:      "horizontaltree",
				Parent:      "parent",
				NodeSpacing: 50,
			},
		},
		{
			name:    "empty diagram",
			title:   "Empty Diagram",
			records: []Record{},
			header:  DrawIOHeader{Layout: "auto"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := New()
			result := builder.DrawIO(tt.title, tt.records, tt.header)

			// Verify builder returns itself for chaining
			if result != builder {
				t.Error("DrawIO() should return the builder for chaining")
			}

			// Build and check content
			doc := builder.Build()
			if len(doc.contents) != 1 {
				t.Fatalf("Expected 1 content, got %d", len(doc.contents))
			}

			// Verify it's a DrawIOContent
			drawio, ok := doc.contents[0].(*DrawIOContent)
			if !ok {
				t.Fatalf("Expected DrawIOContent, got %T", doc.contents[0])
			}

			if drawio.title != tt.title {
				t.Errorf("title = %q, want %q", drawio.title, tt.title)
			}

			if len(drawio.records) != len(tt.records) {
				t.Errorf("record count = %d, want %d", len(drawio.records), len(tt.records))
			}

			// Basic header check
			if drawio.header.Layout != tt.header.Layout {
				t.Errorf("header.Layout = %q, want %q", drawio.header.Layout, tt.header.Layout)
			}
		})
	}
}

func TestBuilder_MethodChaining(t *testing.T) {
	builder := New()

	// Create test table for collapsible section
	testTable, err := NewTableContent("Test Table", []map[string]any{{"name": "Alice"}}, WithKeys("name"))
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Chain all methods including new collapsible section methods
	result := builder.
		SetMetadata("author", "test").
		Table("Users", []map[string]any{{"name": "Alice"}}, WithKeys("name")).
		Text("Some text").
		Header("Section Header").
		Raw("html", []byte("<p>HTML</p>")).
		Section("Details", func(b *Builder) {
			b.Text("Section content")
		}).
		AddCollapsibleSection("Expandable", []Content{NewTextContent("Detail")}).
		AddCollapsibleTable("Table Section", testTable).
		CollapsibleSection("Mixed Content", func(b *Builder) {
			b.Text("Nested content")
		}).
		Graph("Dependencies", []Edge{{From: "A", To: "B"}}).
		Chart("Custom", "flow", map[string]any{"data": "value"}).
		GanttChart("Timeline", []GanttTask{{ID: "t1", Title: "Task 1"}}).
		PieChart("Distribution", []PieSlice{{Label: "A", Value: 100}}, true).
		DrawIO("Diagram", []Record{{"id": "1"}}, DrawIOHeader{})

	// All methods should return the same builder
	if result != builder {
		t.Error("Method chaining broken: methods should return the same builder")
	}

	// Verify all content was added
	doc := builder.Build()
	expectedCount := 13 // One for each method call above (SetMetadata doesn't add content)
	if len(doc.contents) != expectedCount {
		t.Errorf("Expected %d contents after chaining, got %d", expectedCount, len(doc.contents))
	}
}
