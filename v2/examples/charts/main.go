package main

import (
	"context"
	"log"
	"time"

	output "github.com/ArjenSchwarz/go-output/v2"
)

func main() {
	// This example demonstrates v2's chart and graph capabilities
	// including Mermaid, DOT, and Draw.io formats

	// Example 1: Project Timeline (Gantt Chart)
	projectTasks := []output.GanttTask{
		{
			ID:           "planning",
			Title:        "Project Planning",
			StartDate:    "2024-01-01",
			EndDate:      "2024-01-15",
			Dependencies: []string{},
			Status:       "done",
		},
		{
			ID:           "design",
			Title:        "System Design",
			StartDate:    "2024-01-16",
			EndDate:      "2024-02-15",
			Dependencies: []string{"planning"},
			Status:       "done",
		},
		{
			ID:           "backend",
			Title:        "Backend Development",
			StartDate:    "2024-02-01",
			EndDate:      "2024-03-15",
			Dependencies: []string{"design"},
			Status:       "active",
		},
		{
			ID:           "frontend",
			Title:        "Frontend Development",
			StartDate:    "2024-02-15",
			EndDate:      "2024-04-01",
			Dependencies: []string{"design"},
			Status:       "active",
		},
		{
			ID:           "testing",
			Title:        "Integration Testing",
			StartDate:    "2024-03-15",
			EndDate:      "2024-04-15",
			Dependencies: []string{"backend", "frontend"},
			Status:       "crit",
		},
		{
			ID:           "deployment",
			Title:        "Deployment",
			StartDate:    "2024-04-15",
			EndDate:      "2024-04-30",
			Dependencies: []string{"testing"},
			Status:       "",
		},
	}

	// Example 2: Budget Distribution (Pie Chart)
	budgetSlices := []output.PieSlice{
		{Label: "Development", Value: 45.0},
		{Label: "Testing", Value: 20.0},
		{Label: "Infrastructure", Value: 15.0},
		{Label: "Marketing", Value: 12.0},
		{Label: "Support", Value: 8.0},
	}

	// Example 3: System Architecture (Flow Chart)
	systemEdges := []output.Edge{
		{From: "User", To: "Load Balancer", Label: "HTTPS"},
		{From: "Load Balancer", To: "Web Server 1", Label: ""},
		{From: "Load Balancer", To: "Web Server 2", Label: ""},
		{From: "Web Server 1", To: "App Server", Label: "API"},
		{From: "Web Server 2", To: "App Server", Label: "API"},
		{From: "App Server", To: "Database", Label: "SQL"},
		{From: "App Server", To: "Cache", Label: "Redis"},
		{From: "App Server", To: "Queue", Label: "Jobs"},
	}

	// Example 4: Draw.io Network Diagram Data
	networkComponents := []output.Record{
		{
			"Name":        "Load Balancer",
			"Type":        "nginx",
			"IP":          "10.0.1.10",
			"Connections": "web-server-1,web-server-2",
			"Status":      "active",
		},
		{
			"Name":        "Web Server 1",
			"Type":        "apache",
			"IP":          "10.0.1.20",
			"Connections": "app-server",
			"Status":      "active",
		},
		{
			"Name":        "Web Server 2",
			"Type":        "apache",
			"IP":          "10.0.1.21",
			"Connections": "app-server",
			"Status":      "active",
		},
		{
			"Name":        "App Server",
			"Type":        "nodejs",
			"IP":          "10.0.2.10",
			"Connections": "database,cache",
			"Status":      "active",
		},
		{
			"Name":        "Database",
			"Type":        "postgresql",
			"IP":          "10.0.3.10",
			"Connections": "",
			"Status":      "active",
		},
		{
			"Name":        "Cache",
			"Type":        "redis",
			"IP":          "10.0.3.20",
			"Connections": "",
			"Status":      "active",
		},
	}

	drawioHeader := output.DrawIOHeader{
		Label: "%Name%\\n%Type%\\n%IP%",
		Style: "shape=%Image%;fillColor=#dae8fc;strokeColor=#6c8ebf;",
		Connections: []output.DrawIOConnection{
			{From: "Name", To: "Connections", Label: "connects to", Style: "curved=1"},
		},
		Layout:       "horizontalflow",
		NodeSpacing:  100,
		LevelSpacing: 150,
		EdgeSpacing:  50,
	}

	// Create comprehensive document with all chart types
	doc := output.New().
		Header("Project Dashboard - Charts & Diagrams").
		Text("Generated: "+time.Now().Format("2006-01-02 15:04:05")).
		Text("").

		// Section 1: Project Timeline
		Section("Project Timeline", func(b *output.Builder) {
			b.Text("Current project status and timeline:")
			b.GanttChart("Development Timeline", projectTasks)
			b.Text("📅 Current phase: Development (Backend & Frontend in parallel)")
			b.Text("⚠️  Critical path: Testing phase is critical for delivery date")
		}).

		// Section 2: Budget Analysis
		Section("Budget Distribution", func(b *output.Builder) {
			b.Text("Project budget allocation by category:")
			b.PieChart("Budget Breakdown", budgetSlices, true)
			b.Text("💰 Total budget: $500,000")
			b.Text("📊 Largest allocation: Development (45%)")
		}).

		// Section 3: System Architecture
		Section("System Architecture", func(b *output.Builder) {
			b.Text("High-level system component relationships:")
			b.Graph("System Flow", systemEdges)
			b.Text("🏗️  Architecture: Load-balanced web tier with shared backend")
			b.Text("🔄 Data flow: User → LB → Web → App → DB/Cache")
		}).

		// Section 4: Infrastructure Diagram
		Section("Infrastructure Diagram", func(b *output.Builder) {
			b.Text("Network topology and component details:")
			b.DrawIO("Network Infrastructure", networkComponents, drawioHeader)
			b.Text("🌐 Network: 10.0.0.0/16 with segmented subnets")
			b.Text("✅ All components: Active and healthy")
		}).
		Text("").
		Text("---").
		Text("Dashboard generated by go-output v2 | Supports Mermaid, DOT, and Draw.io formats").
		Build()

	// Create file writer for saving diagram files
	fileWriter, err := output.NewFileWriter("./output", "charts.{format}")
	if err != nil {
		log.Fatalf("Failed to create file writer: %v", err)
	}

	// Configure output for multiple diagram formats
	out := output.NewOutput(
		output.WithFormats(
			output.Table,   // Console overview
			output.Mermaid, // Mermaid diagrams (.mmd files)
			output.DOT,     // Graphviz diagrams (.dot files)
			output.DrawIO,  // Draw.io CSV import (.csv files)
		),
		output.WithWriters(
			output.NewStdoutWriter(), // Display summary
			fileWriter,               // Save diagram files
		),
	)

	if err := out.Render(context.Background(), doc); err != nil {
		log.Fatalf("Failed to render charts document: %v", err)
	}

	log.Println("\n=== Chart Generation Complete ===")
	log.Println("Generated files:")
	log.Println("• ./output/charts.mmd  - Mermaid diagrams (paste into Mermaid Live)")
	log.Println("• ./output/charts.dot  - Graphviz DOT file (render with: dot -Tpng -o chart.png charts.dot)")
	log.Println("• ./output/charts.csv  - Draw.io import file (File → Import → CSV in Draw.io)")
	log.Println()
	log.Println("Chart Features Demonstrated:")
	log.Println("✓ Gantt charts for project timelines")
	log.Println("✓ Pie charts for data distribution")
	log.Println("✓ Flow charts for system architecture")
	log.Println("✓ Draw.io CSV format with layout configuration")
	log.Println("✓ Multiple diagram formats from single document")
}
