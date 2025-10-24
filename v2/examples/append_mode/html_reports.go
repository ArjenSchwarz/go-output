package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	output "github.com/ArjenSchwarz/go-output/v2"
)

// This example demonstrates appending HTML reports using the comment marker system.
//
// Key Points:
// - First write creates a full HTML page with <!-- go-output-append --> marker
// - Subsequent writes insert HTML fragments before the marker
// - The marker preserves the HTML structure across multiple appends
// - This is perfect for building daily reports, dashboard updates, or incremental documentation
func main() {
	ctx := context.Background()

	// Clean up any existing report file from previous runs
	reportFile := "./output/daily-report.html"
	_ = os.Remove(reportFile)

	// Create FileWriter with append mode enabled
	fw, err := output.NewFileWriterWithOptions(
		"./output",
		"daily-report.{ext}",
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

	// First report section - creates the HTML page with marker
	fmt.Println("Creating initial HTML report...")
	morningData := []map[string]any{
		{"Time": "09:00", "Event": "Server Start", "Status": "Success", "Duration": "2.3s"},
		{"Time": "09:15", "Event": "Database Migration", "Status": "Success", "Duration": "45.1s"},
		{"Time": "09:20", "Event": "Cache Warming", "Status": "Success", "Duration": "12.7s"},
	}

	doc := output.New().
		Text("Daily Operations Report - "+time.Now().Format("2006-01-02")).
		Text("Morning Operations (09:00 - 12:00)").
		Table("morning", morningData, output.WithKeys("Time", "Event", "Status", "Duration")).
		Build()

	if err := out.Render(ctx, doc); err != nil {
		log.Fatalf("Failed to render morning report: %v", err)
	}
	fmt.Println("✓ Morning report created")

	// Second report section - appends afternoon activities
	fmt.Println("Appending afternoon report...")
	afternoonData := []map[string]any{
		{"Time": "13:00", "Event": "API Deploy", "Status": "Success", "Duration": "8.2s"},
		{"Time": "14:30", "Event": "Load Test", "Status": "Warning", "Duration": "180.5s"},
		{"Time": "15:45", "Event": "Security Scan", "Status": "Success", "Duration": "95.3s"},
	}

	doc = output.New().
		Text("Afternoon Operations (13:00 - 17:00)").
		Table("afternoon", afternoonData, output.WithKeys("Time", "Event", "Status", "Duration")).
		Build()

	if err := out.Render(ctx, doc); err != nil {
		log.Fatalf("Failed to render afternoon report: %v", err)
	}
	fmt.Println("✓ Afternoon report appended")

	// Third report section - appends evening summary
	fmt.Println("Appending evening summary...")
	summaryData := []map[string]any{
		{"Metric": "Total Operations", "Value": "6", "Target": "5", "Status": "Above Target"},
		{"Metric": "Success Rate", "Value": "83%", "Target": "95%", "Status": "Below Target"},
		{"Metric": "Avg Duration", "Value": "57.4s", "Target": "60s", "Status": "On Target"},
		{"Metric": "Warnings", "Value": "1", "Target": "0", "Status": "Needs Review"},
	}

	doc = output.New().
		Text("Daily Summary").
		Table("summary", summaryData, output.WithKeys("Metric", "Value", "Target", "Status")).
		Text("Report generated at: " + time.Now().Format(time.RFC3339)).
		Build()

	if err := out.Render(ctx, doc); err != nil {
		log.Fatalf("Failed to render summary: %v", err)
	}
	fmt.Println("✓ Evening summary appended")

	fmt.Println("\nSuccess! HTML report created with multiple appends")
	fmt.Printf("Open in browser: open %s\n", reportFile)
	fmt.Println("\nThe HTML file contains:")
	fmt.Println("  1. Full HTML page structure (created on first write)")
	fmt.Println("  2. Morning operations table")
	fmt.Println("  3. Afternoon operations table (appended)")
	fmt.Println("  4. Evening summary table (appended)")
	fmt.Println("  5. <!-- go-output-append --> marker at the end")
	fmt.Println("\nAll content is inserted before the marker, preserving HTML structure.")

	// Read and display file size
	info, err := os.Stat(reportFile)
	if err != nil {
		log.Fatalf("Failed to stat report file: %v", err)
	}
	fmt.Printf("\nGenerated file size: %d bytes\n", info.Size())
}
