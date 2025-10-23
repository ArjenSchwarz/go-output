package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	output "github.com/ArjenSchwarz/go-output/v2"
)

// This example demonstrates appending multi-section documents to a file.
//
// Key Points:
// - Documents can contain multiple sections (text, tables, raw content)
// - All sections are appended together in a single operation
// - For HTML format, all sections are inserted before the marker
// - Section boundaries are preserved using format-appropriate separators
// - Perfect for complex reports with mixed content types
func main() {
	ctx := context.Background()

	// Clean up any existing report file from previous runs
	reportFile := "./output/multi-section-report.html"
	_ = os.Remove(reportFile)

	// Create FileWriter with append mode enabled
	fw, err := output.NewFileWriterWithOptions(
		"./output",
		"multi-section-report.{ext}",
		output.WithAppendMode(),
	)
	if err != nil {
		log.Fatalf("Failed to create FileWriter: %v", err)
	}

	// Create output configuration for HTML format
	out := output.NewOutput(
		output.WithFormat(output.FormatHTML),
		output.WithWriter(fw),
	)

	// First report: Morning metrics with multiple sections
	fmt.Println("Creating initial report with multiple sections...")

	morningPerformance := []map[string]any{
		{"Metric": "CPU Usage", "Value": "45%", "Status": "Normal"},
		{"Metric": "Memory Usage", "Value": "62%", "Status": "Normal"},
		{"Metric": "Disk I/O", "Value": "120 MB/s", "Status": "Normal"},
	}

	morningIncidents := []map[string]any{
		{"Time": "08:15", "Severity": "Low", "Component": "Web Server", "Description": "Slow response time"},
		{"Time": "09:45", "Severity": "Medium", "Component": "Database", "Description": "Connection pool exhausted"},
	}

	doc := output.New().
		Text("System Monitoring Report - "+time.Now().Format("2006-01-02")).
		Text("Report Period: Morning (06:00 - 12:00)").
		Text("").
		Text("Performance Metrics:").
		Table("morning_perf", morningPerformance, output.WithKeys("Metric", "Value", "Status")).
		Text("").
		Text("Incidents:").
		Table("morning_incidents", morningIncidents, output.WithKeys("Time", "Severity", "Component", "Description")).
		Text("Morning Summary: 2 incidents recorded, system performance within normal range.").
		Build()

	if err := out.Render(ctx, doc); err != nil {
		log.Fatalf("Failed to render morning report: %v", err)
	}
	fmt.Println("✓ Morning report created with 2 tables and 5 text sections")

	// Second report: Afternoon update with multiple sections
	fmt.Println("\nAppending afternoon report with multiple sections...")

	afternoonPerformance := []map[string]any{
		{"Metric": "CPU Usage", "Value": "78%", "Status": "Elevated"},
		{"Metric": "Memory Usage", "Value": "85%", "Status": "Warning"},
		{"Metric": "Disk I/O", "Value": "245 MB/s", "Status": "Elevated"},
	}

	afternoonIncidents := []map[string]any{
		{"Time": "13:20", "Severity": "High", "Component": "Load Balancer", "Description": "Health check failures"},
		{"Time": "14:05", "Severity": "Low", "Component": "Cache", "Description": "Cache miss rate increased"},
		{"Time": "15:30", "Severity": "Medium", "Component": "API Gateway", "Description": "Rate limit exceeded"},
	}

	resolutionActions := []map[string]any{
		{"Action": "Scaled web servers from 3 to 5 instances", "Time": "13:25", "Result": "Success"},
		{"Action": "Flushed and warmed cache", "Time": "14:10", "Result": "Success"},
		{"Action": "Updated rate limit configuration", "Time": "15:35", "Result": "Success"},
	}

	doc = output.New().
		Text("").
		Text("Report Period: Afternoon (12:00 - 18:00)").
		Text("").
		Text("Performance Metrics:").
		Table("afternoon_perf", afternoonPerformance, output.WithKeys("Metric", "Value", "Status")).
		Text("").
		Text("Incidents:").
		Table("afternoon_incidents", afternoonIncidents, output.WithKeys("Time", "Severity", "Component", "Description")).
		Text("").
		Text("Resolution Actions:").
		Table("resolutions", resolutionActions, output.WithKeys("Action", "Time", "Result")).
		Text("Afternoon Summary: 3 incidents recorded and resolved. System performance elevated but stable.").
		Build()

	if err := out.Render(ctx, doc); err != nil {
		log.Fatalf("Failed to render afternoon report: %v", err)
	}
	fmt.Println("✓ Afternoon report appended with 3 tables and 6 text sections")

	// Third report: Evening summary with final statistics
	fmt.Println("\nAppending evening summary with multiple sections...")

	dailyStats := []map[string]any{
		{"Period": "Morning", "Incidents": "2", "Avg CPU": "45%", "Avg Memory": "62%"},
		{"Period": "Afternoon", "Incidents": "3", "Avg CPU": "78%", "Avg Memory": "85%"},
		{"Period": "Evening", "Incidents": "0", "Avg CPU": "52%", "Avg Memory": "68%"},
	}

	recommendations := []map[string]any{
		{"Priority": "High", "Recommendation": "Review load balancer health check configuration", "Category": "Infrastructure"},
		{"Priority": "Medium", "Recommendation": "Implement auto-scaling for web tier", "Category": "Capacity"},
		{"Priority": "Low", "Recommendation": "Optimize cache warming strategy", "Category": "Performance"},
	}

	doc = output.New().
		Text("").
		Text("Daily Summary").
		Text("=============").
		Text("").
		Text("Statistics by Period:").
		Table("daily_stats", dailyStats, output.WithKeys("Period", "Incidents", "Avg CPU", "Avg Memory")).
		Text("").
		Text("Recommendations:").
		Table("recommendations", recommendations, output.WithKeys("Priority", "Recommendation", "Category")).
		Text("").
		Text("Overall Assessment: 5 incidents occurred today, all successfully resolved.").
		Text("System showed elevated resource usage during afternoon peak but remained operational.").
		Text("Report generated at: "+time.Now().Format(time.RFC3339)).
		Build()

	if err := out.Render(ctx, doc); err != nil {
		log.Fatalf("Failed to render evening summary: %v", err)
	}
	fmt.Println("✓ Evening summary appended with 2 tables and 7 text sections")

	fmt.Println("\n✅ Success! Multi-section report created with multiple appends")
	fmt.Printf("Open in browser: open %s\n", reportFile)

	fmt.Println("\nThe HTML file contains:")
	fmt.Println("  • Full HTML page structure (created on first write)")
	fmt.Println("  • Morning report: 2 tables + 5 text sections")
	fmt.Println("  • Afternoon report: 3 tables + 6 text sections (appended)")
	fmt.Println("  • Evening summary: 2 tables + 7 text sections (appended)")
	fmt.Println("  • Total: 7 tables and 18 text sections")
	fmt.Println("  • <!-- go-output-append --> marker preserved at the end")

	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("  ✓ Multiple sections in each document")
	fmt.Println("  ✓ Mixed content types (text + tables)")
	fmt.Println("  ✓ All sections appended together atomically")
	fmt.Println("  ✓ HTML structure preserved across appends")

	// Read and display file size
	info, err := os.Stat(reportFile)
	if err != nil {
		log.Fatalf("Failed to stat report file: %v", err)
	}
	fmt.Printf("\nGenerated file size: %d bytes\n", info.Size())
}
