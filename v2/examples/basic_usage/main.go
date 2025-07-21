package main

import (
	"context"
	"log"

	output "github.com/ArjenSchwarz/go-output/v2"
)

func main() {
	// Sample data - simulating user records
	users := []map[string]any{
		{"Name": "Alice Johnson", "Age": 30, "Department": "Engineering", "Status": "Active"},
		{"Name": "Bob Smith", "Age": 25, "Department": "Marketing", "Status": "Active"},
		{"Name": "Carol Davis", "Age": 35, "Department": "Engineering", "Status": "Inactive"},
		{"Name": "David Wilson", "Age": 28, "Department": "Sales", "Status": "Active"},
	}

	// Create a document using the builder pattern
	// Key feature: WithKeys preserves exact column ordering
	doc := output.New().
		Text("Company Employee Report").
		Text("Generated: " + "2024-01-15").
		Table("Employees", users, output.WithKeys("Name", "Department", "Age", "Status")).
		Text("Total employees: 4").
		Build()

	// Create output configuration
	// This example outputs to stdout in table format
	out := output.NewOutput(
		output.WithFormat(output.Table),
		output.WithWriter(output.NewStdoutWriter()),
	)

	// Render the document
	if err := out.Render(context.Background(), doc); err != nil {
		log.Fatalf("Failed to render document: %v", err)
	}
}