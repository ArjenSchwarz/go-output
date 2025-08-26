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
	fmt.Println("=== Progress Tracking Demonstration ===")
	fmt.Println("This example shows v2's enhanced progress indicators")
	fmt.Println()

	// Simulate a large dataset processing scenario
	largeDataset := generateLargeDataset(1000)

	// Example 1: Auto-selected progress based on format
	fmt.Println("Example 1: Format-aware progress (Table format = visual progress)")

	doc1 := output.New().
		Header("Large Dataset Processing").
		Table("Sample Data", largeDataset[:5], // Show only first 5 rows
			output.WithKeys("ID", "Name", "Category", "Value", "Status")).
		Text(fmt.Sprintf("Processing %d total records...", len(largeDataset))).
		Build()

	// For table format, this will show visual progress bar
	progress1 := output.NewProgressForFormatName("table",
		output.WithProgressColor(output.ProgressColorBlue),
		output.WithProgressStatus("Processing large dataset"),
	)

	out1 := output.NewOutput(
		output.WithFormat(output.Table),
		output.WithWriter(output.NewStdoutWriter()),
		output.WithProgress(progress1),
	)

	// Simulate progress updates during rendering
	go simulateProgressUpdates(progress1, 100, 50*time.Millisecond)

	if err := out1.Render(context.Background(), doc1); err != nil {
		log.Fatalf("Failed to render with progress: %v", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")

	// Example 2: Multiple format progress selection
	fmt.Println("Example 2: Multi-format progress (JSON + Table = smart selection)")

	doc2 := output.New().
		Table("Export Data", largeDataset[:10],
			output.WithKeys("ID", "Name", "Value")).
		Build()

	// For multiple formats, progress system intelligently selects approach
	progress2 := output.NewProgressForFormats(
		[]output.Format{output.JSON, output.Table},
		output.WithProgressColor(output.ProgressColorGreen),
		output.WithProgressStatus("Multi-format export"),
	)

	fileWriter, err := output.NewFileWriter("./output", "export.{format}")
	if err != nil {
		log.Fatalf("Failed to create file writer: %v", err)
	}

	out2 := output.NewOutput(
		output.WithFormats(output.JSON, output.Table),
		output.WithWriters(
			output.NewStdoutWriter(),
			fileWriter,
		),
		output.WithProgress(progress2),
	)

	// Simulate longer operation
	go simulateProgressUpdates(progress2, 200, 25*time.Millisecond)

	if err := out2.Render(context.Background(), doc2); err != nil {
		log.Fatalf("Failed to render multi-format: %v", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 50) + "\n")

	// Example 3: Manual progress control
	fmt.Println("Example 3: Manual progress control with status updates")

	doc3 := output.New().
		Header("Processing Pipeline").
		Table("Results", largeDataset[:3],
			output.WithKeys("ID", "Name", "Status")).
		Build()

	progress3 := output.NewProgressForFormatName("table",
		output.WithProgressColor(output.ProgressColorYellow),
		output.WithProgressStatus("Initializing"),
	)

	out3 := output.NewOutput(
		output.WithFormat(output.Table),
		output.WithWriter(output.NewStdoutWriter()),
		output.WithProgress(progress3),
	)

	// Manual progress control with detailed status
	go func() {
		time.Sleep(100 * time.Millisecond)
		progress3.SetTotal(4)
		progress3.SetStatus("Loading data...")
		progress3.SetCurrent(1)

		time.Sleep(200 * time.Millisecond)
		progress3.SetStatus("Validating records...")
		progress3.SetCurrent(2)

		time.Sleep(200 * time.Millisecond)
		progress3.SetStatus("Applying transformations...")
		progress3.SetCurrent(3)

		time.Sleep(200 * time.Millisecond)
		progress3.SetStatus("Generating output...")
		progress3.SetCurrent(4)

		time.Sleep(100 * time.Millisecond)
		// Progress will be completed by the render process
	}()

	if err := out3.Render(context.Background(), doc3); err != nil {
		log.Fatalf("Failed to render with manual progress: %v", err)
	}

	fmt.Println("\nProgress Features Demonstrated:")
	fmt.Println("✓ Format-aware progress selection (visual for table, no-op for JSON)")
	fmt.Println("✓ Multi-format intelligent progress handling")
	fmt.Println("✓ Manual progress control with custom status messages")
	fmt.Println("✓ Color-coded progress indicators")
	fmt.Println("✓ Thread-safe progress updates")
	fmt.Println("✓ Automatic completion and cleanup")
}

// generateLargeDataset creates a sample dataset for demonstration
func generateLargeDataset(size int) []map[string]any {
	dataset := make([]map[string]any, size)
	categories := []string{"Electronics", "Books", "Clothing", "Home", "Sports"}
	statuses := []string{"Active", "Pending", "Completed", "Cancelled"}

	for i := range size {
		dataset[i] = map[string]any{
			"ID":       fmt.Sprintf("REC-%04d", i+1),
			"Name":     fmt.Sprintf("Item %d", i+1),
			"Category": categories[i%len(categories)],
			"Value":    float64(100 + (i*13)%1000),
			"Status":   statuses[i%len(statuses)],
		}
	}

	return dataset
}

// simulateProgressUpdates mimics a background process updating progress
func simulateProgressUpdates(progress output.Progress, total int, delay time.Duration) {
	progress.SetTotal(total)

	for i := 0; i <= total; i++ {
		time.Sleep(delay)
		progress.SetCurrent(i)

		// Update status at key milestones
		if i == total/4 {
			progress.SetStatus("25% complete")
		} else if i == total/2 {
			progress.SetStatus("50% complete")
		} else if i == 3*total/4 {
			progress.SetStatus("75% complete")
		} else if i == total {
			progress.SetStatus("Finalizing...")
		}
	}
}
