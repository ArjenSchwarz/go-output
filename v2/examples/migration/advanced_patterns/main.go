package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	output "github.com/ArjenSchwarz/go-output/v2"
)

func main() {
	fmt.Println("=== Advanced Migration Patterns (v1 → v2) ===")
	fmt.Println("This example shows migration of advanced v1 features to v2")
	fmt.Println()

	// Sample data for examples
	systemData := []map[string]any{
		{"Server": "web-01", "Status": "✅ Running", "CPU": "45%", "Memory": "2.1GB"},
		{"Server": "web-02", "Status": "✅ Running", "CPU": "38%", "Memory": "1.8GB"},
		{"Server": "db-01", "Status": "⚠️ Warning", "CPU": "78%", "Memory": "6.2GB"},
	}

	// Pattern 1: Output Settings Migration
	fmt.Println("Pattern 1: OutputSettings → Functional Options")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Println("\nv1 Code:")
	fmt.Println(`// v1 - Complex settings struct
settings := format.NewOutputSettings()
settings.OutputFormat = "table"
settings.OutputFile = "system_report.html"
settings.OutputFileFormat = "html" 
settings.TableStyle = "ColoredBright"
settings.UseEmoji = true
settings.UseColors = true
settings.SortKey = "Server"
settings.HasTOC = true

output := &format.OutputArray{
    Settings: settings,
    Keys: []string{"Server", "Status", "CPU", "Memory"},
}
output.AddContents(systemData)
output.Write()`)

	fmt.Println("\nv2 Equivalent:")

	// v2 Implementation - functional options
	doc1 := output.New().
		Header("System Status Report").
		Table("Server Status", systemData,
			output.WithKeys("Server", "Status", "CPU", "Memory")).
		Build()

	fileWriter, err := output.NewFileWriter("./output", "system_report.{format}")
	if err != nil {
		log.Fatalf("Failed to create file writer: %v", err)
	}

	out1 := output.NewOutput(
		// Multiple formats instead of separate settings
		output.WithFormats(output.TableWithStyle("ColoredBright"), output.HTML()),
		// Transformers replace boolean flags
		output.WithTransformers(
			output.NewEnhancedEmojiTransformer(),
			output.NewColorTransformer(),
			output.NewSortTransformer("Server", true), // ascending sort
		),
		// Writers replace file settings
		output.WithWriters(
			output.NewStdoutWriter(), // Console output
			fileWriter,               // File output
		),
		// v1 compatibility options
		output.WithTOC(true),
		output.WithTableStyle("ColoredBright"),
	)

	if err := out1.Render(context.Background(), doc1); err != nil {
		log.Fatalf("Pattern 1 failed: %v", err)
	}

	fmt.Println("✅ Generated: Console output + ./output/system_report.table + ./output/system_report.html")
	fmt.Println("\n" + strings.Repeat("=", 70) + "\n")

	// Pattern 2: Progress Indicators
	fmt.Println("Pattern 2: Progress Indicators")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Println("\nv1 Code:")
	fmt.Println(`// v1 - Settings-based progress
settings := format.NewOutputSettings()
settings.OutputFormat = "table"
settings.ProgressOptions = format.ProgressOptions{
    Color:         format.ProgressColorGreen,
    Status:        "Processing servers",
    TrackerLength: 50,
}

p := format.NewProgress(settings)
p.SetTotal(100)
for i := 0; i < 100; i++ {
    p.Increment(1)
    p.SetStatus(fmt.Sprintf("Processing server %d", i))
}
p.Complete()`)

	fmt.Println("\nv2 Equivalent (Format-aware Progress):")

	// v2 Implementation - enhanced progress system
	doc2 := output.New().
		Table("Processing Results", systemData,
			output.WithKeys("Server", "Status", "CPU", "Memory")).
		Build()

	// Format-aware progress creation
	progress := output.NewProgressForFormatName("table",
		output.WithProgressColor(output.ProgressColorGreen),
		output.WithProgressStatus("Processing servers"),
	)

	out2 := output.NewOutput(
		output.WithFormat(output.Table()),
		output.WithWriter(output.NewStdoutWriter()),
		output.WithProgress(progress),
	)

	// Simulate progress updates
	go simulateProgress(progress)

	if err := out2.Render(context.Background(), doc2); err != nil {
		log.Fatalf("Pattern 2 failed: %v", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 70) + "\n")

	// Pattern 3: Multiple Format Output
	fmt.Println("Pattern 3: Multiple Format Output")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Println("\nv1 Code:")
	fmt.Println(`// v1 - Required separate operations for each format
// JSON output
settings1 := format.NewOutputSettings()
settings1.OutputFormat = "json"
settings1.OutputFile = "data.json"
output1 := &format.OutputArray{Settings: settings1}
output1.AddContents(data)
output1.Write()

// CSV output  
settings2 := format.NewOutputSettings()
settings2.OutputFormat = "csv"
settings2.OutputFile = "data.csv"
output2 := &format.OutputArray{Settings: settings2}
output2.AddContents(data) // Duplicate data setup!
output2.Write()`)

	fmt.Println("\nv2 Equivalent (Single Operation):")

	// v2 Implementation - single document, multiple outputs
	doc3 := output.New().
		SetMetadata("export_type", "multi_format").
		Table("Server Export", systemData,
			output.WithKeys("Server", "Status", "CPU", "Memory")).
		Text("Export completed: " + time.Now().Format("2006-01-02 15:04:05")).
		Build()

	multiFileWriter, err := output.NewFileWriter("./output", "data.{format}")
	if err != nil {
		log.Fatalf("Failed to create multi file writer: %v", err)
	}

	out3 := output.NewOutput(
		// Multiple formats from single document
		output.WithFormats(output.JSON(), output.CSV(), output.HTML(), output.Table()),
		output.WithWriters(
			output.NewStdoutWriter(),
			multiFileWriter,
		),
	)

	if err := out3.Render(context.Background(), doc3); err != nil {
		log.Fatalf("Pattern 3 failed: %v", err)
	}

	fmt.Println("✅ Generated: data.json, data.csv, data.html + console output")
	fmt.Println("\n" + strings.Repeat("=", 70) + "\n")

	// Pattern 4: Complex Document Structure
	fmt.Println("Pattern 4: Complex Document with Mixed Content")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Println("\nv1 Code:")
	fmt.Println(`// v1 - Multiple separate operations
output1 := &format.OutputArray{}
output1.AddHeader("System Report")
output1.AddContents(serverData)
output1.Write()

output2 := &format.OutputArray{}  
output2.AddHeader("Summary")
output2.AddContents(summaryData)
output2.Write() // Separate render operations`)

	fmt.Println("\nv2 Equivalent (Unified Document):")

	summaryData := []map[string]any{
		{"Metric": "Total Servers", "Value": "3", "Status": "OK"},
		{"Metric": "Healthy Servers", "Value": "2", "Status": "OK"},
		{"Metric": "Warnings", "Value": "1", "Status": "WARN"},
		{"Metric": "Critical Issues", "Value": "0", "Status": "OK"},
	}

	// v2 Implementation - unified document with sections
	doc4 := output.New().
		SetMetadata("report_type", "system_health").
		Header("Comprehensive System Report").
		Text("Generated: "+time.Now().Format("2006-01-02 15:04:05")).
		Text("").
		Section("Server Status", func(b *output.Builder) {
			b.Text("Current status of all production servers:")
			b.Table("Servers", systemData,
				output.WithKeys("Server", "Status", "CPU", "Memory"))
			b.Text("⚠️ Note: db-01 requires attention due to high CPU usage")
		}).
		Section("Health Summary", func(b *output.Builder) {
			b.Text("Overall system health metrics:")
			b.Table("Metrics", summaryData,
				output.WithKeys("Metric", "Value", "Status"))
			b.Text("✅ System operational with minor warnings")
		}).
		Text("").
		Text("---").
		Text("Report generated by go-output v2").
		Build()

	out4 := output.NewOutput(
		output.WithFormats(
			output.MarkdownWithToC(true), // Markdown with table of contents
			output.TableWithStyle("ColoredBright"),
		),
		output.WithWriter(output.NewStdoutWriter()),
		output.WithFrontMatter(map[string]string{
			"title":   "System Report",
			"date":    time.Now().Format("2006-01-02"),
			"type":    "system_health",
			"version": "2.0",
		}),
	)

	if err := out4.Render(context.Background(), doc4); err != nil {
		log.Fatalf("Pattern 4 failed: %v", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 70) + "\n")

	// Summary
	fmt.Println("Advanced Migration Benefits:")
	fmt.Println("✓ Functional options replace complex settings structs")
	fmt.Println("✓ Format-aware progress indicators")
	fmt.Println("✓ Single document → multiple formats (eliminates duplication)")
	fmt.Println("✓ Unified document structure with sections")
	fmt.Println("✓ Enhanced table of contents and front matter support")
	fmt.Println("✓ Transformer pipeline for data processing")
	fmt.Println("✓ Better separation of concerns (build vs render)")

	fmt.Println("\nV2 Advantages over V1:")
	fmt.Println("• No global state or race conditions")
	fmt.Println("• Exact key order preservation")
	fmt.Println("• Thread-safe concurrent operations")
	fmt.Println("• Immutable documents")
	fmt.Println("• Composable transformers")
	fmt.Println("• Enhanced error handling")
	fmt.Println("• Better performance with streaming")
}

// simulateProgress mimics background processing for progress demo
func simulateProgress(progress output.Progress) {
	progress.SetTotal(50)
	for i := 0; i <= 50; i++ {
		time.Sleep(20 * time.Millisecond)
		progress.SetCurrent(i)
		if i%10 == 0 {
			progress.SetStatus(fmt.Sprintf("Processing... %d%%", (i*100)/50))
		}
	}
}
