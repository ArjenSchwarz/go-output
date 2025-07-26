package output

import (
	"reflect"
	"testing"
)

// TestBuilder_HeaderMethod tests the Header convenience method
func TestBuilder_HeaderMethod(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "simple header",
			text:     "Test Header",
			expected: "Test Header",
		},
		{
			name:     "empty header",
			text:     "",
			expected: "",
		},
		{
			name:     "header with special characters",
			text:     "Header: Test & Examples",
			expected: "Header: Test & Examples",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := New()
			result := builder.Header(tt.text)

			// Verify builder returns itself for chaining
			if result != builder {
				t.Error("Header() should return the builder for chaining")
			}

			// Build and check content
			doc := builder.Build()
			if len(doc.contents) != 1 {
				t.Fatalf("Expected 1 content, got %d", len(doc.contents))
			}

			// Verify it's a TextContent with header style
			textContent, ok := doc.contents[0].(*TextContent)
			if !ok {
				t.Fatalf("Expected TextContent, got %T", doc.contents[0])
			}

			if textContent.text != tt.expected {
				t.Errorf("text = %q, want %q", textContent.text, tt.expected)
			}

			if !textContent.style.Header {
				t.Error("Header style should be true")
			}
		})
	}
}

// TestBuilder_SectionMethod tests the Section method
func TestBuilder_SectionMethod(t *testing.T) {
	tests := []struct {
		name            string
		title           string
		opts            []SectionOption
		contentCount    int
		expectedLevel   int
		expectedContent []string
	}{
		{
			name:          "simple section",
			title:         "Test Section",
			opts:          []SectionOption{},
			contentCount:  2,
			expectedLevel: 0,
		},
		{
			name:          "section with level",
			title:         "Nested Section",
			opts:          []SectionOption{WithLevel(2)},
			contentCount:  1,
			expectedLevel: 2,
		},
		{
			name:          "empty section",
			title:         "Empty Section",
			opts:          []SectionOption{},
			contentCount:  0,
			expectedLevel: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := New()

			var sectionBuilder *Builder
			result := builder.Section(tt.title, func(b *Builder) {
				sectionBuilder = b
				// Add test content based on contentCount
				for i := 0; i < tt.contentCount; i++ {
					b.Text("Content " + string(rune('A'+i)))
				}
			}, tt.opts...)

			// Verify builder returns itself for chaining
			if result != builder {
				t.Error("Section() should return the builder for chaining")
			}

			// Verify section builder is different from main builder
			if sectionBuilder == builder {
				t.Error("Section should create a new sub-builder")
			}

			// Build and check content
			doc := builder.Build()
			if len(doc.contents) != 1 {
				t.Fatalf("Expected 1 content, got %d", len(doc.contents))
			}

			// Verify it's a SectionContent
			section, ok := doc.contents[0].(*SectionContent)
			if !ok {
				t.Fatalf("Expected SectionContent, got %T", doc.contents[0])
			}

			if section.title != tt.title {
				t.Errorf("title = %q, want %q", section.title, tt.title)
			}

			if section.level != tt.expectedLevel {
				t.Errorf("level = %d, want %d", section.level, tt.expectedLevel)
			}

			if len(section.contents) != tt.contentCount {
				t.Errorf("content count = %d, want %d", len(section.contents), tt.contentCount)
			}
		})
	}
}

// TestBuilder_Graph tests the Graph method
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

// TestBuilder_AddCollapsibleSection tests the AddCollapsibleSection method
func TestBuilder_AddCollapsibleSection(t *testing.T) {
	tests := []struct {
		name             string
		title            string
		content          []Content
		opts             []CollapsibleSectionOption
		expectedLevel    int
		expectedContent  int
		expectedExpanded bool
	}{
		{
			name:  "simple collapsible section",
			title: "Expandable Details",
			content: []Content{
				NewTextContent("Detail text 1"),
				NewTextContent("Detail text 2"),
			},
			opts:             []CollapsibleSectionOption{},
			expectedLevel:    0,
			expectedContent:  2,
			expectedExpanded: false,
		},
		{
			name:  "expanded section with level",
			title: "Important Information",
			content: []Content{
				NewTextContent("Critical details"),
			},
			opts: []CollapsibleSectionOption{
				WithSectionExpanded(true),
				WithSectionLevel(2),
			},
			expectedLevel:    2,
			expectedContent:  1,
			expectedExpanded: true,
		},
		{
			name:             "empty collapsible section",
			title:            "No Details",
			content:          []Content{},
			opts:             []CollapsibleSectionOption{},
			expectedLevel:    0,
			expectedContent:  0,
			expectedExpanded: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := New()
			result := builder.AddCollapsibleSection(tt.title, tt.content, tt.opts...)

			// Verify builder returns itself for chaining
			if result != builder {
				t.Error("AddCollapsibleSection() should return the builder for chaining")
			}

			// Build and check content
			doc := builder.Build()
			if len(doc.contents) != 1 {
				t.Fatalf("Expected 1 content, got %d", len(doc.contents))
			}

			// Verify it's a DefaultCollapsibleSection
			section, ok := doc.contents[0].(*DefaultCollapsibleSection)
			if !ok {
				t.Fatalf("Expected DefaultCollapsibleSection, got %T", doc.contents[0])
			}

			if section.Title() != tt.title {
				t.Errorf("title = %q, want %q", section.Title(), tt.title)
			}

			if section.Level() != tt.expectedLevel {
				t.Errorf("level = %d, want %d", section.Level(), tt.expectedLevel)
			}

			if section.IsExpanded() != tt.expectedExpanded {
				t.Errorf("expanded = %t, want %t", section.IsExpanded(), tt.expectedExpanded)
			}

			if len(section.Content()) != tt.expectedContent {
				t.Errorf("content count = %d, want %d", len(section.Content()), tt.expectedContent)
			}
		})
	}
}

// TestBuilder_AddCollapsibleTable tests the AddCollapsibleTable method
func TestBuilder_AddCollapsibleTable(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		tableData []map[string]any
		opts      []CollapsibleSectionOption
	}{
		{
			name:  "simple collapsible table",
			title: "User Data",
			tableData: []map[string]any{
				{"name": "Alice", "age": 30},
				{"name": "Bob", "age": 25},
			},
			opts: []CollapsibleSectionOption{},
		},
		{
			name:  "expanded collapsible table with level",
			title: "Critical Issues",
			tableData: []map[string]any{
				{"severity": "high", "count": 5},
				{"severity": "medium", "count": 10},
			},
			opts: []CollapsibleSectionOption{
				WithSectionExpanded(true),
				WithSectionLevel(1),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create table content first
			table, err := NewTableContent("Test Table", tt.tableData)
			if err != nil {
				t.Fatalf("Failed to create table: %v", err)
			}

			builder := New()
			result := builder.AddCollapsibleTable(tt.title, table, tt.opts...)

			// Verify builder returns itself for chaining
			if result != builder {
				t.Error("AddCollapsibleTable() should return the builder for chaining")
			}

			// Build and check content
			doc := builder.Build()
			if len(doc.contents) != 1 {
				t.Fatalf("Expected 1 content, got %d", len(doc.contents))
			}

			// Verify it's a DefaultCollapsibleSection containing the table
			section, ok := doc.contents[0].(*DefaultCollapsibleSection)
			if !ok {
				t.Fatalf("Expected DefaultCollapsibleSection, got %T", doc.contents[0])
			}

			if section.Title() != tt.title {
				t.Errorf("title = %q, want %q", section.Title(), tt.title)
			}

			// Verify it contains exactly one content item (the table)
			sectionContent := section.Content()
			if len(sectionContent) != 1 {
				t.Fatalf("Expected 1 content in section, got %d", len(sectionContent))
			}

			// Verify the content is a table
			_, ok = sectionContent[0].(*TableContent)
			if !ok {
				t.Fatalf("Expected TableContent in section, got %T", sectionContent[0])
			}
		})
	}
}

// TestBuilder_CollapsibleSection tests the CollapsibleSection method with sub-builder
func TestBuilder_CollapsibleSection(t *testing.T) {
	tests := []struct {
		name            string
		title           string
		opts            []CollapsibleSectionOption
		contentCount    int
		expectedLevel   int
		expectedContent []string
	}{
		{
			name:          "simple collapsible section with sub-builder",
			title:         "Analysis Results",
			opts:          []CollapsibleSectionOption{},
			contentCount:  2,
			expectedLevel: 0,
		},
		{
			name:          "expanded section with level and sub-builder",
			title:         "Important Findings",
			opts:          []CollapsibleSectionOption{WithSectionLevel(1), WithSectionExpanded(true)},
			contentCount:  3,
			expectedLevel: 1,
		},
		{
			name:          "empty collapsible section",
			title:         "No Results",
			opts:          []CollapsibleSectionOption{},
			contentCount:  0,
			expectedLevel: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := New()

			var sectionBuilder *Builder
			result := builder.CollapsibleSection(tt.title, func(b *Builder) {
				sectionBuilder = b
				// Add test content based on contentCount
				for i := 0; i < tt.contentCount; i++ {
					b.Text("Content " + string(rune('A'+i)))
				}
			}, tt.opts...)

			// Verify builder returns itself for chaining
			if result != builder {
				t.Error("CollapsibleSection() should return the builder for chaining")
			}

			// Verify section builder is different from main builder
			if sectionBuilder == builder {
				t.Error("CollapsibleSection should create a new sub-builder")
			}

			// Build and check content
			doc := builder.Build()
			if len(doc.contents) != 1 {
				t.Fatalf("Expected 1 content, got %d", len(doc.contents))
			}

			// Verify it's a DefaultCollapsibleSection
			section, ok := doc.contents[0].(*DefaultCollapsibleSection)
			if !ok {
				t.Fatalf("Expected DefaultCollapsibleSection, got %T", doc.contents[0])
			}

			if section.Title() != tt.title {
				t.Errorf("title = %q, want %q", section.Title(), tt.title)
			}

			if section.Level() != tt.expectedLevel {
				t.Errorf("level = %d, want %d", section.Level(), tt.expectedLevel)
			}

			if len(section.Content()) != tt.contentCount {
				t.Errorf("content count = %d, want %d", len(section.Content()), tt.contentCount)
			}
		})
	}
}

// TestBuilder_CollapsibleSection_Mixed_Content tests collapsible sections with mixed content types
func TestBuilder_CollapsibleSection_Mixed_Content(t *testing.T) {
	builder := New()

	// Create a collapsible section with mixed content
	result := builder.CollapsibleSection("Analysis Report", func(b *Builder) {
		b.Text("Summary: Found multiple issues")
		b.Table("Issues", []map[string]any{
			{"type": "error", "count": 5},
			{"type": "warning", "count": 10},
		}, WithKeys("type", "count"))
		b.Text("See detailed breakdown above")
	}, WithSectionExpanded(false), WithSectionLevel(2))

	// Verify method chaining
	if result != builder {
		t.Error("CollapsibleSection() should return the builder for chaining")
	}

	// Build and verify
	doc := builder.Build()
	if len(doc.contents) != 1 {
		t.Fatalf("Expected 1 content, got %d", len(doc.contents))
	}

	section, ok := doc.contents[0].(*DefaultCollapsibleSection)
	if !ok {
		t.Fatalf("Expected DefaultCollapsibleSection, got %T", doc.contents[0])
	}

	if section.Title() != "Analysis Report" {
		t.Errorf("title = %q, want %q", section.Title(), "Analysis Report")
	}

	if section.Level() != 2 {
		t.Errorf("level = %d, want %d", section.Level(), 2)
	}

	if section.IsExpanded() != false {
		t.Errorf("expanded = %t, want %t", section.IsExpanded(), false)
	}

	// Verify content types
	content := section.Content()
	if len(content) != 3 {
		t.Fatalf("Expected 3 content items, got %d", len(content))
	}

	// Check content types
	_, ok1 := content[0].(*TextContent)
	_, ok2 := content[1].(*TableContent)
	_, ok3 := content[2].(*TextContent)

	if !ok1 || !ok2 || !ok3 {
		t.Error("Content types don't match expected: text, table, text")
	}
}

// TestBuilder_MethodChaining tests that all methods can be chained
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
