package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	output "github.com/ArjenSchwarz/go-output/v2"
)

// This example demonstrates appending JSON logs in NDJSON (Newline-Delimited JSON) format.
// NDJSON is a convenient format for log files where each line is a separate, self-contained JSON object.
//
// Key Points:
// - WithAppendMode() enables byte-level appending to JSON files
// - Each write appends a new JSON object to the file
// - The result is NDJSON format: {"event":"start"}\n{"event":"end"}\n
// - This is NOT valid JSON for standard parsers, but is perfect for log streaming
func main() {
	ctx := context.Background()

	// Clean up any existing log file from previous runs
	logFile := "./output/app-log.json"
	_ = os.Remove(logFile)

	// Create FileWriter with append mode enabled
	fw, err := output.NewFileWriterWithOptions(
		"./output",
		"app-log.{ext}",
		output.WithAppendMode(),
	)
	if err != nil {
		log.Fatalf("Failed to create FileWriter: %v", err)
	}

	// Create output configuration for JSON format
	out := output.NewOutput(
		output.WithFormat(output.FormatJSON),
		output.WithWriter(fw),
	)

	// Simulate application lifecycle with multiple log events
	events := []map[string]any{
		{"timestamp": time.Now().Format(time.RFC3339), "level": "INFO", "event": "application_start", "version": "1.0.0"},
		{"timestamp": time.Now().Add(time.Second).Format(time.RFC3339), "level": "INFO", "event": "database_connected", "host": "localhost:5432"},
		{"timestamp": time.Now().Add(2 * time.Second).Format(time.RFC3339), "level": "WARNING", "event": "high_memory_usage", "memory_mb": 856},
		{"timestamp": time.Now().Add(3 * time.Second).Format(time.RFC3339), "level": "INFO", "event": "api_request", "method": "GET", "path": "/api/users", "duration_ms": 45},
		{"timestamp": time.Now().Add(4 * time.Second).Format(time.RFC3339), "level": "ERROR", "event": "database_timeout", "query": "SELECT * FROM users"},
		{"timestamp": time.Now().Add(5 * time.Second).Format(time.RFC3339), "level": "INFO", "event": "application_shutdown", "uptime_seconds": 300},
	}

	// Write each event as a separate document
	// Each write appends to the file, creating NDJSON format
	for i, event := range events {
		doc := output.New().
			Table("log", []map[string]any{event}).
			Build()

		if err := out.Render(ctx, doc); err != nil {
			log.Fatalf("Failed to log event %d: %v", i+1, err)
		}

		fmt.Printf("Logged event %d: %s\n", i+1, event["event"])
	}

	fmt.Println("\nSuccess! Log file created with NDJSON format")
	fmt.Printf("View the log file: cat %s\n", logFile)
	fmt.Println("\nEach line is a separate JSON object that can be processed independently.")
	fmt.Println("This format is ideal for log streaming, line-by-line processing, and tools like jq:")
	fmt.Printf("  jq -s '.' %s  # Convert to JSON array\n", logFile)
	fmt.Printf("  jq 'select(.level==\"ERROR\")' %s  # Filter errors\n", logFile)

	// Read and display the file content
	content, err := os.ReadFile(logFile)
	if err != nil {
		log.Fatalf("Failed to read log file: %v", err)
	}

	fmt.Println("\nGenerated NDJSON content:")
	fmt.Println("----------------------------------------")
	fmt.Println(string(content))
	fmt.Println("----------------------------------------")
}
