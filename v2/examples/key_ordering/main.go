package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	output "github.com/ArjenSchwarz/go-output/v2"
)

func main() {
	fmt.Println("=== Key Ordering Demonstration ===")
	fmt.Println("This example shows how v2 preserves exact user-specified key ordering")
	fmt.Println()

	// Same data, different key orders to demonstrate preservation
	salesData := []map[string]any{
		{"Product": "Widget A", "Q1": 100, "Q2": 150, "Q3": 120, "Q4": 180, "Total": 550},
		{"Product": "Widget B", "Q1": 80, "Q2": 90, "Q3": 85, "Q4": 95, "Total": 350},
		{"Product": "Widget C", "Q1": 200, "Q2": 180, "Q3": 220, "Q4": 240, "Total": 840},
	}

	// Example 1: Business-friendly ordering (Product first, then quarters, then total)
	fmt.Println("Example 1: Business View (Product → Quarters → Total)")
	doc1 := output.New().
		Table("Sales Report - Business View", salesData,
			output.WithKeys("Product", "Q1", "Q2", "Q3", "Q4", "Total")).
		Build()

	out1 := output.NewOutput(
		output.WithFormat(output.Table()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	if err := out1.Render(context.Background(), doc1); err != nil {
		log.Fatalf("Failed to render document 1: %v", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 60) + "\n")

	// Example 2: Financial ordering (Total first for quick scanning)
	fmt.Println("Example 2: Financial View (Total → Product → Quarters)")
	doc2 := output.New().
		Table("Sales Report - Financial View", salesData,
			output.WithKeys("Total", "Product", "Q1", "Q2", "Q3", "Q4")).
		Build()

	out2 := output.NewOutput(
		output.WithFormat(output.Table()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	if err := out2.Render(context.Background(), doc2); err != nil {
		log.Fatalf("Failed to render document 2: %v", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 60) + "\n")

	// Example 3: Multiple tables with different key orders in same document
	fmt.Println("Example 3: Multiple Tables with Different Key Orders")

	customerData := []map[string]any{
		{"ID": "C001", "Name": "ACME Corp", "Revenue": 50000, "Region": "North"},
		{"ID": "C002", "Name": "Beta Inc", "Revenue": 75000, "Region": "South"},
	}

	doc3 := output.New().
		Header("Quarterly Business Report").
		// Sales table: Product-focused ordering
		Table("Sales by Product", salesData,
			output.WithKeys("Product", "Q1", "Q2", "Q3", "Q4", "Total")).
		Text("---").
		// Customer table: Different ordering entirely
		Table("Top Customers", customerData,
			output.WithKeys("Name", "Region", "Revenue", "ID")). // Name first, ID last
		Build()

	out3 := output.NewOutput(
		output.WithFormat(output.Table()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	if err := out3.Render(context.Background(), doc3); err != nil {
		log.Fatalf("Failed to render document 3: %v", err)
	}

	fmt.Println("\nKey Ordering Features Demonstrated:")
	fmt.Println("✓ Each table preserves exact user-specified key order")
	fmt.Println("✓ No alphabetical sorting or reordering")
	fmt.Println("✓ Multiple tables can have completely different key orders")
	fmt.Println("✓ Key order is consistent across all output formats")
	fmt.Println()
	fmt.Println("This addresses a major limitation in v1 where key ordering")
	fmt.Println("was unpredictable due to Go map iteration randomness.")
}
