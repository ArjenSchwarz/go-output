package main

import (
	"context"
	"fmt"
	"log"

	output "github.com/ArjenSchwarz/go-output/v2"
)

// User represents a user record
type User struct {
	Name  string
	Email string
	Age   int
}

// Product represents a product record
type Product struct {
	ID    int
	Name  string
	Price float64
}

func main() {
	fmt.Println("=== Pipeline API vs Per-Content Transformations ===\n")

	// Sample data
	users := []output.Record{
		{"name": "Alice", "email": "alice@example.com", "age": 30},
		{"name": "Bob", "email": "bob@example.com", "age": 17},
		{"name": "Charlie", "email": "charlie@example.com", "age": 25},
		{"name": "Diana", "email": "diana@example.com", "age": 35},
		{"name": "Eve", "email": "eve@example.com", "age": 16},
	}

	products := []output.Record{
		{"id": 1, "name": "Widget A", "price": 29.99},
		{"id": 2, "name": "Widget B", "price": 49.99},
		{"id": 3, "name": "Widget C", "price": 19.99},
		{"id": 4, "name": "Widget D", "price": 99.99},
		{"id": 5, "name": "Widget E", "price": 39.99},
	}

	// Demonstrate the old way (deprecated)
	fmt.Println("### OLD WAY (Deprecated Pipeline API) ###")
	oldWay(users)

	fmt.Println("\n### NEW WAY (Per-Content Transformations) ###")
	newWay(users, products)

	fmt.Println("\n### ADVANCED: Multiple Tables with Different Transformations ###")
	multipleTablesExample(users, products)

	fmt.Println("\n### DYNAMIC TRANSFORMATION CONSTRUCTION ###")
	dynamicTransformations(users, true, true, 3)
}

// oldWay demonstrates the deprecated Pipeline API
func oldWay(users []output.Record) {
	builder := output.New()
	builder.Table("users", users, output.WithKeys("name", "email", "age"))

	doc := builder.Build()

	// Apply transformations using deprecated pipeline
	transformed, err := doc.Pipeline().
		Filter(func(r output.Record) bool {
			return r["age"].(int) >= 18
		}).
		Sort(output.SortKey{Column: "name", Direction: output.Ascending}).
		Execute()

	if err != nil {
		log.Fatalf("Pipeline error: %v", err)
	}

	// Render result
	out := output.NewOutput(
		output.WithFormat(output.JSON),
		output.WithWriter(output.NewStdoutWriter()),
	)
	if err := out.Render(context.Background(), transformed); err != nil {
		log.Fatalf("Render error: %v", err)
	}
}

// newWay demonstrates per-content transformations
func newWay(users, products []output.Record) {
	builder := output.New()

	// Attach transformations directly to the table
	builder.Table("users", users,
		output.WithKeys("name", "email", "age"),
		output.WithTransformations(
			output.NewFilterOp(func(r output.Record) bool {
				return r["age"].(int) >= 18
			}),
			output.NewSortOp(output.SortKey{Column: "name", Direction: output.Ascending}),
		),
	)

	doc := builder.Build()

	// Transformations apply automatically during rendering
	out := output.NewOutput(
		output.WithFormat(output.JSON),
		output.WithWriter(output.NewStdoutWriter()),
	)
	if err := out.Render(context.Background(), doc); err != nil {
		log.Fatalf("Render error: %v", err)
	}
}

// multipleTablesExample shows how per-content transformations work with multiple tables
func multipleTablesExample(users, products []output.Record) {
	builder := output.New()

	// Each table gets its own transformations
	builder.Table("adult_users", users,
		output.WithKeys("name", "email", "age"),
		output.WithTransformations(
			output.NewFilterOp(func(r output.Record) bool {
				return r["age"].(int) >= 18
			}),
			output.NewSortOp(output.SortKey{Column: "name", Direction: output.Ascending}),
		),
	)

	builder.Table("top_products", products,
		output.WithKeys("id", "name", "price"),
		output.WithTransformations(
			output.NewSortOp(output.SortKey{Column: "price", Direction: output.Descending}),
			output.NewLimitOp(3), // Top 3 products by price
		),
	)

	doc := builder.Build()

	out := output.NewOutput(
		output.WithFormat(output.JSON),
		output.WithWriter(output.NewStdoutWriter()),
	)
	if err := out.Render(context.Background(), doc); err != nil {
		log.Fatalf("Render error: %v", err)
	}
}

// dynamicTransformations shows how to build transformations conditionally
func dynamicTransformations(users []output.Record, filterAdults, sortByName bool, limit int) {
	builder := output.New()

	var transformations []output.Operation

	if filterAdults {
		transformations = append(transformations, output.NewFilterOp(func(r output.Record) bool {
			return r["age"].(int) >= 18
		}))
	}

	if sortByName {
		transformations = append(transformations, output.NewSortOp(
			output.SortKey{Column: "name", Direction: output.Ascending},
		))
	}

	if limit > 0 {
		transformations = append(transformations, output.NewLimitOp(limit))
	}

	builder.Table("users", users,
		output.WithKeys("name", "email", "age"),
		output.WithTransformations(transformations...),
	)

	doc := builder.Build()

	out := output.NewOutput(
		output.WithFormat(output.JSON),
		output.WithWriter(output.NewStdoutWriter()),
	)
	if err := out.Render(context.Background(), doc); err != nil {
		log.Fatalf("Render error: %v", err)
	}
}
