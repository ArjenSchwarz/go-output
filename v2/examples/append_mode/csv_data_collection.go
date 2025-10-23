package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	output "github.com/ArjenSchwarz/go-output/v2"
)

// This example demonstrates appending CSV data with automatic header handling.
//
// Key Points:
// - First write creates the CSV file with headers
// - Subsequent writes skip headers and append only data rows
// - This maintains valid CSV structure across multiple appends
// - Perfect for continuous data collection, batch processing, or scheduled exports
func main() {
	ctx := context.Background()

	// Clean up any existing CSV file from previous runs
	csvFile := "./output/sensor-readings.csv"
	_ = os.Remove(csvFile)

	// Create FileWriter with append mode enabled
	fw, err := output.NewFileWriterWithOptions(
		"./output",
		"sensor-readings.{ext}",
		output.WithAppendMode(),
	)
	if err != nil {
		log.Fatalf("Failed to create FileWriter: %v", err)
	}

	// Create output configuration for CSV format
	out := output.NewOutput(
		output.WithFormat(output.FormatCSV),
		output.WithWriter(fw),
	)

	// Simulate collecting sensor data in batches throughout the day
	batches := []struct {
		time     string
		readings []map[string]any
	}{
		{
			time: "Morning (06:00-09:00)",
			readings: []map[string]any{
				{"Timestamp": "2024-01-15 06:00:00", "Sensor": "TempSensor01", "Location": "Server Room", "Value": "21.5", "Unit": "°C"},
				{"Timestamp": "2024-01-15 07:30:00", "Sensor": "TempSensor01", "Location": "Server Room", "Value": "22.1", "Unit": "°C"},
				{"Timestamp": "2024-01-15 09:00:00", "Sensor": "TempSensor01", "Location": "Server Room", "Value": "23.4", "Unit": "°C"},
			},
		},
		{
			time: "Midday (12:00-14:00)",
			readings: []map[string]any{
				{"Timestamp": "2024-01-15 12:00:00", "Sensor": "TempSensor01", "Location": "Server Room", "Value": "24.8", "Unit": "°C"},
				{"Timestamp": "2024-01-15 13:00:00", "Sensor": "TempSensor01", "Location": "Server Room", "Value": "25.2", "Unit": "°C"},
				{"Timestamp": "2024-01-15 14:00:00", "Sensor": "TempSensor01", "Location": "Server Room", "Value": "24.9", "Unit": "°C"},
			},
		},
		{
			time: "Afternoon (15:00-18:00)",
			readings: []map[string]any{
				{"Timestamp": "2024-01-15 15:00:00", "Sensor": "TempSensor01", "Location": "Server Room", "Value": "24.3", "Unit": "°C"},
				{"Timestamp": "2024-01-15 16:30:00", "Sensor": "TempSensor01", "Location": "Server Room", "Value": "23.7", "Unit": "°C"},
				{"Timestamp": "2024-01-15 18:00:00", "Sensor": "TempSensor01", "Location": "Server Room", "Value": "22.9", "Unit": "°C"},
			},
		},
		{
			time: "Evening (20:00-22:00)",
			readings: []map[string]any{
				{"Timestamp": "2024-01-15 20:00:00", "Sensor": "TempSensor01", "Location": "Server Room", "Value": "22.1", "Unit": "°C"},
				{"Timestamp": "2024-01-15 21:00:00", "Sensor": "TempSensor01", "Location": "Server Room", "Value": "21.8", "Unit": "°C"},
				{"Timestamp": "2024-01-15 22:00:00", "Sensor": "TempSensor01", "Location": "Server Room", "Value": "21.5", "Unit": "°C"},
			},
		},
	}

	// Process each batch
	totalReadings := 0
	for batchNum, batch := range batches {
		fmt.Printf("Processing batch %d: %s...\n", batchNum+1, batch.time)

		// Create document with batch readings
		// Note: We use WithKeys to ensure consistent column order
		doc := output.New().
			Table("readings", batch.readings, output.WithKeys("Timestamp", "Sensor", "Location", "Value", "Unit")).
			Build()

		// Render and append to CSV file
		if err := out.Render(ctx, doc); err != nil {
			log.Fatalf("Failed to append batch %d: %v", batchNum+1, err)
		}

		totalReadings += len(batch.readings)
		fmt.Printf("✓ Appended %d readings (Total: %d)\n", len(batch.readings), totalReadings)

		// Small delay to simulate real-world batch processing
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("\nSuccess! CSV file created with automatic header handling")
	fmt.Printf("View the CSV file: cat %s\n", csvFile)
	fmt.Println("\nThe CSV file contains:")
	fmt.Println("  - Header row (written once on first batch)")
	fmt.Printf("  - %d data rows (appended across %d batches)\n", totalReadings, len(batches))
	fmt.Println("\nHeaders were automatically skipped on batches 2-4 to maintain CSV structure.")

	// Read and display the CSV content
	content, err := os.ReadFile(csvFile)
	if err != nil {
		log.Fatalf("Failed to read CSV file: %v", err)
	}

	fmt.Println("\nGenerated CSV content (first 10 lines):")
	fmt.Println("----------------------------------------")
	lines := 0
	for i, b := range content {
		if b == '\n' {
			lines++
			if lines >= 10 {
				fmt.Print(string(content[:i+1]))
				fmt.Println("... (remaining lines omitted)")
				break
			}
		}
	}
	if lines < 10 {
		fmt.Print(string(content))
	}
	fmt.Println("----------------------------------------")
	fmt.Printf("\nTotal file size: %d bytes\n", len(content))
}
