package main

import (
	"context"
	"log"

	output "github.com/ArjenSchwarz/go-output/v2"
)

func main() {
	// Sample data
	products := []map[string]any{
		{"ID": "P001", "Name": "Laptop", "Price": 999.99, "Stock": 15, "Category": "Electronics"},
		{"ID": "P002", "Name": "Mouse", "Price": 25.50, "Stock": 150, "Category": "Electronics"},
		{"ID": "P003", "Name": "Keyboard", "Price": 75.00, "Stock": 80, "Category": "Electronics"},
		{"ID": "P004", "Name": "Monitor", "Price": 299.99, "Stock": 25, "Category": "Electronics"},
	}

	// Create document with multiple content types
	doc := output.New().
		SetMetadata("generated_by", "inventory_system").
		SetMetadata("version", "2.0").
		Header("Product Inventory Report").
		Table("Products", products,
			output.WithKeys("ID", "Name", "Price", "Stock", "Category")).
		Text("Report Summary:").
		Text("- Total products: 4").
		Text("- Categories: Electronics").
		Text("- Low stock items: Monitor (25 units)").
		Build()

	// Configure output for multiple formats simultaneously
	// This demonstrates v2's ability to render one document to multiple formats
	fileWriter, err := output.NewFileWriter("./output", "inventory.{format}")
	if err != nil {
		log.Fatalf("Failed to create file writer: %v", err)
	}

	out := output.NewOutput(
		// Multiple formats - each will be rendered independently
		output.WithFormats(
			output.JSON(),  // Machine-readable data
			output.CSV(),   // Spreadsheet import
			output.HTML(),  // Web display
			output.Table(), // Console display
		),
		// Multiple writers - stdout and files
		output.WithWriters(
			output.NewStdoutWriter(), // Display in console
			fileWriter,               // Save to files
		),
	)

	// Single render call processes all format/writer combinations
	if err := out.Render(context.Background(), doc); err != nil {
		log.Fatalf("Failed to render document: %v", err)
	}

	log.Println("Successfully generated:")
	log.Println("- Console output (table format)")
	log.Println("- ./output/inventory.json")
	log.Println("- ./output/inventory.csv")
	log.Println("- ./output/inventory.html")
}
