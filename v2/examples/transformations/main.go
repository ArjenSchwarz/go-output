package main

import (
	"context"
	"fmt"
	"log"
	"os"

	output "github.com/ArjenSchwarz/go-output/v2"
)

func main() {
	fmt.Println("=== Per-Content Transformations Examples ===\n")

	// Example 1: Basic filter + sort transformations
	basicFilterAndSort()

	// Example 2: Multiple tables with different transformations
	multipleTables()

	// Example 3: Dynamic transformation construction
	dynamicTransformations()

	// Example 4: Error handling
	errorHandling()
}

// Example 1: Basic filter + sort transformations on a table
func basicFilterAndSort() {
	fmt.Println("Example 1: Basic Filter + Sort")
	fmt.Println("--------------------------------")

	// Sample employee data
	employees := []map[string]any{
		{"name": "Alice", "department": "Engineering", "salary": 85000, "years": 5},
		{"name": "Bob", "department": "Marketing", "salary": 65000, "years": 2},
		{"name": "Charlie", "department": "Engineering", "salary": 95000, "years": 7},
		{"name": "Diana", "department": "Sales", "salary": 70000, "years": 3},
		{"name": "Eve", "department": "Engineering", "salary": 75000, "years": 4},
	}

	// Build document with transformations:
	// 1. Filter to only Engineering department
	// 2. Sort by salary (descending)
	doc := output.New().
		Text("Engineering Team - Sorted by Salary").
		Table("engineers", employees,
			output.WithKeys("name", "salary", "years"),
			output.WithTransformations(
				output.NewFilterOp(func(r output.Record) bool {
					return r["department"].(string) == "Engineering"
				}),
				output.NewSortOp(output.SortKey{
					Column:    "salary",
					Direction: output.Descending,
				}),
			),
		).
		Build()

	// Render to table format
	out := output.NewOutput(
		output.WithFormat(output.Table),
		output.WithWriter(output.NewStdoutWriter()),
	)

	if err := out.Render(context.Background(), doc); err != nil {
		log.Printf("Error rendering: %v\n", err)
	}

	fmt.Println("\n")
}

// Example 2: Multiple tables with different transformations
func multipleTables() {
	fmt.Println("Example 2: Multiple Tables with Different Transformations")
	fmt.Println("---------------------------------------------------------")

	// Sales data
	sales := []map[string]any{
		{"product": "Widget", "revenue": 50000, "units": 1000, "quarter": "Q1"},
		{"product": "Gadget", "revenue": 75000, "units": 1500, "quarter": "Q1"},
		{"product": "Doohickey", "revenue": 30000, "units": 600, "quarter": "Q1"},
		{"product": "Widget", "revenue": 55000, "units": 1100, "quarter": "Q2"},
		{"product": "Gadget", "revenue": 80000, "units": 1600, "quarter": "Q2"},
	}

	// Customer data
	customers := []map[string]any{
		{"name": "Acme Corp", "status": "active", "value": 100000},
		{"name": "TechStart Inc", "status": "inactive", "value": 50000},
		{"name": "Global Solutions", "status": "active", "value": 250000},
		{"name": "Small Business LLC", "status": "active", "value": 25000},
	}

	// Build document with different transformations per table
	doc := output.New().
		Text("Q1/Q2 Sales Report").
		// Top 3 products by revenue
		Table("top_products", sales,
			output.WithKeys("product", "revenue", "units"),
			output.WithTransformations(
				output.NewSortOp(output.SortKey{Column: "revenue", Direction: output.Descending}),
				output.NewLimitOp(3),
			),
		).
		// Active customers sorted by value
		Table("active_customers", customers,
			output.WithKeys("name", "value"),
			output.WithTransformations(
				output.NewFilterOp(func(r output.Record) bool {
					return r["status"].(string) == "active"
				}),
				output.NewSortOp(output.SortKey{Column: "value", Direction: output.Descending}),
			),
		).
		Build()

	// Render to JSON format
	out := output.NewOutput(
		output.WithFormat(output.JSON),
		output.WithWriter(output.NewStdoutWriter()),
	)

	if err := out.Render(context.Background(), doc); err != nil {
		log.Printf("Error rendering: %v\n", err)
	}

	fmt.Println("\n")
}

// Example 3: Dynamic transformation construction
func dynamicTransformations() {
	fmt.Println("Example 3: Dynamic Transformation Construction")
	fmt.Println("----------------------------------------------")

	// Sample data
	data := []map[string]any{
		{"id": 1, "priority": "high", "status": "open", "assignee": "Alice"},
		{"id": 2, "priority": "low", "status": "closed", "assignee": "Bob"},
		{"id": 3, "priority": "high", "status": "open", "assignee": "Charlie"},
		{"id": 4, "priority": "medium", "status": "open", "assignee": "Diana"},
		{"id": 5, "priority": "low", "status": "open", "assignee": "Eve"},
	}

	// Simulate user preferences
	filterByPriority := "high"
	filterByStatus := "open"
	sortByAssignee := true
	limitResults := 10

	// Build transformations dynamically
	var transformations []output.Operation

	// Add filter for priority if specified
	if filterByPriority != "" {
		transformations = append(transformations, output.NewFilterOp(func(r output.Record) bool {
			return r["priority"].(string) == filterByPriority
		}))
	}

	// Add filter for status if specified
	if filterByStatus != "" {
		transformations = append(transformations, output.NewFilterOp(func(r output.Record) bool {
			return r["status"].(string) == filterByStatus
		}))
	}

	// Add sorting if requested
	if sortByAssignee {
		transformations = append(transformations, output.NewSortOp(
			output.SortKey{Column: "assignee", Direction: output.Ascending},
		))
	}

	// Add limit if specified
	if limitResults > 0 {
		transformations = append(transformations, output.NewLimitOp(limitResults))
	}

	// Build document with dynamic transformations
	doc := output.New().
		Text(fmt.Sprintf("Filtered Tasks: %s priority, %s status", filterByPriority, filterByStatus)).
		Table("tasks", data,
			output.WithKeys("id", "priority", "assignee"),
			output.WithTransformations(transformations...),
		).
		Build()

	// Render to table format
	out := output.NewOutput(
		output.WithFormat(output.Table),
		output.WithWriter(output.NewStdoutWriter()),
	)

	if err := out.Render(context.Background(), doc); err != nil {
		log.Printf("Error rendering: %v\n", err)
	}

	fmt.Println("\n")
}

// Example 4: Error handling
func errorHandling() {
	fmt.Println("Example 4: Error Handling")
	fmt.Println("-------------------------")

	data := []map[string]any{
		{"name": "Alice", "score": 95},
		{"name": "Bob", "score": 85},
		{"name": "Charlie", "score": 90},
	}

	// Example 4a: Invalid sort column (will fail during rendering)
	fmt.Println("4a. Attempting to sort by non-existent column:")
	doc := output.New().
		Table("students", data,
			output.WithKeys("name", "score"),
			output.WithTransformations(
				// This will fail because "grade" doesn't exist
				output.NewSortOp(output.SortKey{Column: "grade", Direction: output.Ascending}),
			),
		).
		Build()

	out := output.NewOutput(
		output.WithFormat(output.Table),
		output.WithWriter(output.NewStdoutWriter()),
	)

	if err := out.Render(context.Background(), doc); err != nil {
		fmt.Printf("✓ Expected error caught: %v\n\n", err)
	} else {
		fmt.Println("✗ Expected error but got none\n")
	}

	// Example 4b: Invalid operation configuration (negative limit)
	fmt.Println("4b. Attempting to use negative limit:")
	invalidOp := output.NewLimitOp(-5)
	if err := invalidOp.Validate(); err != nil {
		fmt.Printf("✓ Validation error caught: %v\n\n", err)
	} else {
		fmt.Println("✗ Expected validation error but got none\n")
	}

	// Example 4c: Successful rendering with proper error checking
	fmt.Println("4c. Successful rendering with proper transformations:")
	doc = output.New().
		Table("students", data,
			output.WithKeys("name", "score"),
			output.WithTransformations(
				output.NewFilterOp(func(r output.Record) bool {
					return r["score"].(int) >= 90
				}),
				output.NewSortOp(output.SortKey{Column: "score", Direction: output.Descending}),
			),
		).
		Build()

	if err := out.Render(context.Background(), doc); err != nil {
		fmt.Printf("✗ Unexpected error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✓ All examples completed!")
}
