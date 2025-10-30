package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	output "github.com/ArjenSchwarz/go-output/v2"
)

func main() {
	fmt.Println("🚀 Go-Output v2 Per-Content Transformation Examples")
	fmt.Println(strings.Repeat("=", 60))

	// Generate sample sales data
	salesData := generateSalesData(100)

	// Example 1: Basic Per-Content Operations
	fmt.Println("\n📊 Example 1: Basic Per-Content Transformations")
	basicTransformationExample(salesData)

	// Example 2: Complex Analytics with Transformations
	fmt.Println("\n📈 Example 2: Complex Analytics Transformations")
	complexAnalyticsExample(salesData)

	// Example 3: Aggregation and Reporting
	fmt.Println("\n📋 Example 3: Aggregation and Reporting")
	aggregationExample(salesData)

	// Example 4: Combining Transformations with Byte Transformers
	fmt.Println("\n🎨 Example 4: Combining Transformations with Styling")
	combinedTransformationExample(salesData)

	// Example 5: Multiple Tables with Different Transformations
	fmt.Println("\n🔍 Example 5: Multiple Tables with Different Transformations")
	multipleTablesExample(salesData)
}

// generateSalesData creates realistic sample data for demonstrations
func generateSalesData(count int) []map[string]any {
	regions := []string{"North", "South", "East", "West", "Central"}
	salespeople := []string{"Alice Johnson", "Bob Smith", "Carol Davis", "David Wilson", "Emma Brown", "Frank Miller", "Grace Lee", "Henry Taylor"}

	data := make([]map[string]any, count)
	baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := range count {
		amount := rand.Float64()*50000 + 5000 // $5,000 to $55,000

		// Weight status towards completed for realistic data
		var status string
		statusRand := rand.Float64()
		if statusRand < 0.7 {
			status = "completed"
		} else if statusRand < 0.9 {
			status = "pending"
		} else {
			status = "cancelled"
		}

		data[i] = map[string]any{
			"id":          fmt.Sprintf("SALE-%04d", i+1),
			"salesperson": salespeople[rand.Intn(len(salespeople))],
			"region":      regions[rand.Intn(len(regions))],
			"amount":      amount,
			"date":        baseDate.AddDate(0, 0, rand.Intn(365)),
			"status":      status,
			"product":     fmt.Sprintf("Product-%d", rand.Intn(10)+1),
		}
	}

	return data
}

// basicTransformationExample demonstrates fundamental per-content transformations
func basicTransformationExample(salesData []map[string]any) {
	fmt.Printf("📄 Original data: %d records\n", len(salesData))

	// Create document with transformations attached to the table
	doc := output.New().
		Table("Top High-Value Sales", salesData,
			output.WithKeys("id", "salesperson", "region", "amount", "status", "date"),
			output.WithTransformations(
				// Filter for completed high-value sales
				output.NewFilterOp(func(r output.Record) bool {
					return r["status"] == "completed" && r["amount"].(float64) > 20000
				}),
				// Sort by amount (highest first)
				output.NewSortOp(output.SortKey{Column: "amount", Direction: output.Descending}),
				// Get top 10 results
				output.NewLimitOp(10),
			),
		).
		Build()

	// Output results - transformations apply automatically during rendering
	out := output.NewOutput(
		output.WithFormat(output.Table()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	if err := out.Render(context.Background(), doc); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("📊 Transformations applied during rendering (filter → sort → limit)\n")
}

// complexAnalyticsExample shows advanced per-content transformations
func complexAnalyticsExample(salesData []map[string]any) {
	// Create document with complex transformations attached to the table
	doc := output.New().
		Table("Sales Analysis with Calculated Fields", salesData,
			output.WithKeys("salesperson", "region", "amount", "date", "status", "product", "quarter", "commission", "tier", "days_since_sale"),
			output.WithTransformations(
				// Focus on completed sales
				output.NewFilterOp(func(r output.Record) bool {
					return r["status"] == "completed"
				}),
				// Add calculated fields
				output.NewAddColumnOp("quarter", func(r output.Record) any {
					date := r["date"].(time.Time)
					quarter := (date.Month()-1)/3 + 1
					return fmt.Sprintf("Q%d", quarter)
				}, nil),
				output.NewAddColumnOp("commission", func(r output.Record) any {
					amount := r["amount"].(float64)
					return amount * 0.05 // 5% commission
				}, nil),
				output.NewAddColumnOp("tier", func(r output.Record) any {
					amount := r["amount"].(float64)
					if amount > 40000 {
						return "🥇 Premium"
					} else if amount > 20000 {
						return "🥈 Standard"
					}
					return "🥉 Basic"
				}, nil),
				output.NewAddColumnOp("days_since_sale", func(r output.Record) any {
					saleDate := r["date"].(time.Time)
					return int(time.Since(saleDate).Hours() / 24)
				}, nil),
				// Sort by commission (highest first)
				output.NewSortOp(output.SortKey{Column: "commission", Direction: output.Descending}),
				// Get top 15 for detailed analysis
				output.NewLimitOp(15),
			),
		).
		Build()

	// Show results in both table and JSON format
	out := output.NewOutput(
		output.WithFormats(output.Table(), output.JSON()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	if err := out.Render(context.Background(), doc); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n🔍 Complex transformations applied: filter → add 4 columns → sort → limit\n")
}

// aggregationExample demonstrates GroupBy and aggregate functions with per-content transformations
func aggregationExample(salesData []map[string]any) {
	builder := output.New()

	// Regional performance with aggregations
	builder.Table("Regional Performance Summary", salesData,
		output.WithKeys("region", "total_sales", "avg_sale", "max_sale", "min_sale", "sale_count", "unique_reps"),
		output.WithTransformations(
			// Only include completed sales
			output.NewFilterOp(func(r output.Record) bool {
				return r["status"] == "completed"
			}),
			// Group by region with multiple aggregates
			output.NewGroupByOp(
				[]string{"region"},
				map[string]output.AggregateFunc{
					"total_sales": output.SumAggregate("amount"),
					"avg_sale":    output.AverageAggregate("amount"),
					"max_sale":    output.MaxAggregate("amount"),
					"min_sale":    output.MinAggregate("amount"),
					"sale_count":  output.CountAggregate(),
					"unique_reps": uniqueSalespeopleAggregate,
				},
			),
			// Sort by total sales (highest first)
			output.NewSortOp(output.SortKey{Column: "total_sales", Direction: output.Descending}),
		),
	)

	// Quarterly breakdown with region
	builder.Table("Quarterly Breakdown by Region", salesData,
		output.WithKeys("quarter", "region", "total_sales", "sale_count"),
		output.WithTransformations(
			output.NewFilterOp(func(r output.Record) bool {
				return r["status"] == "completed"
			}),
			output.NewAddColumnOp("quarter", func(r output.Record) any {
				date := r["date"].(time.Time)
				quarter := (date.Month()-1)/3 + 1
				year := date.Year()
				return fmt.Sprintf("%d-Q%d", year, quarter)
			}, nil),
			output.NewGroupByOp(
				[]string{"quarter", "region"},
				map[string]output.AggregateFunc{
					"total_sales": output.SumAggregate("amount"),
					"sale_count":  output.CountAggregate(),
				},
			),
			output.NewSortOp(output.SortKey{Column: "quarter", Direction: output.Ascending}),
		),
	)

	doc := builder.Build()

	fmt.Println("🌍 Multiple aggregation tables with different transformations:")

	out := output.NewOutput(
		output.WithFormat(output.Table()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	if err := out.Render(context.Background(), doc); err != nil {
		log.Fatal(err)
	}
}

// combinedTransformationExample shows per-content transformations + byte transformers
func combinedTransformationExample(salesData []map[string]any) {
	// Create document with data transformations attached
	doc := output.New().
		Table("Top Performers with Styling", salesData,
			output.WithKeys("salesperson", "region", "amount", "status", "performance", "bonus_eligible"),
			output.WithTransformations(
				output.NewFilterOp(func(r output.Record) bool {
					return r["status"] == "completed"
				}),
				output.NewAddColumnOp("performance", func(r output.Record) any {
					amount := r["amount"].(float64)
					if amount > 40000 {
						return "excellent"
					} else if amount > 25000 {
						return "good"
					} else if amount > 15000 {
						return "average"
					}
					return "below_target"
				}, nil),
				output.NewAddColumnOp("bonus_eligible", func(r output.Record) any {
					amount := r["amount"].(float64)
					if amount > 30000 {
						return "yes"
					}
					return "no"
				}, nil),
				output.NewSortOp(output.SortKey{Column: "amount", Direction: output.Descending}),
				output.NewLimitOp(20),
			),
		).
		Build()

	fmt.Println("🎯 Top Performers with Styling:")

	// Apply visual styling with byte transformers during rendering
	out := output.NewOutput(
		output.WithFormat(output.Table()),
		// Add color coding based on performance levels
		output.WithTransformer(output.NewColorTransformerWithScheme(output.ColorScheme{
			Success: "excellent",    // Green
			Warning: "good",         // Yellow
			Info:    "average",      // Blue
			Error:   "below_target", // Red
		})),
		// Convert yes/no to emoji
		output.WithTransformer(&output.EmojiTransformer{}),
		output.WithWriter(output.NewStdoutWriter()),
	)

	if err := out.Render(context.Background(), doc); err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n💡 This example demonstrates:")
	fmt.Println("   • Data operations (filter, sort, calculate) done by per-content transformations")
	fmt.Println("   • Visual styling (colors, emoji) done by byte transformers")
	fmt.Println("   • Best of both worlds: structured data manipulation + presentation")
}

// multipleTablesExample demonstrates the key benefit of per-content transformations:
// different tables in the same document can have different transformations
func multipleTablesExample(salesData []map[string]any) {
	builder := output.New()

	// Table 1: Top performers (high-value completed sales)
	builder.Table("🏆 Top Performers", salesData,
		output.WithKeys("salesperson", "region", "amount", "date"),
		output.WithTransformations(
			output.NewFilterOp(func(r output.Record) bool {
				return r["status"] == "completed" && r["amount"].(float64) > 30000
			}),
			output.NewSortOp(output.SortKey{Column: "amount", Direction: output.Descending}),
			output.NewLimitOp(5),
		),
	)

	// Table 2: Recent pending sales (different filter, different sort)
	builder.Table("⏳ Recent Pending Sales", salesData,
		output.WithKeys("id", "salesperson", "amount", "date", "product"),
		output.WithTransformations(
			output.NewFilterOp(func(r output.Record) bool {
				return r["status"] == "pending"
			}),
			output.NewSortOp(output.SortKey{Column: "date", Direction: output.Descending}),
			output.NewLimitOp(10),
		),
	)

	// Table 3: Regional summary (aggregation)
	builder.Table("🌍 Regional Summary", salesData,
		output.WithKeys("region", "completed_sales", "total_amount", "avg_amount"),
		output.WithTransformations(
			output.NewFilterOp(func(r output.Record) bool {
				return r["status"] == "completed"
			}),
			output.NewGroupByOp(
				[]string{"region"},
				map[string]output.AggregateFunc{
					"completed_sales": output.CountAggregate(),
					"total_amount":    output.SumAggregate("amount"),
					"avg_amount":      output.AverageAggregate("amount"),
				},
			),
			output.NewSortOp(output.SortKey{Column: "total_amount", Direction: output.Descending}),
		),
	)

	// Table 4: Cancelled sales analysis (different focus entirely)
	builder.Table("❌ Cancelled Sales Analysis", salesData,
		output.WithKeys("salesperson", "region", "amount", "product"),
		output.WithTransformations(
			output.NewFilterOp(func(r output.Record) bool {
				return r["status"] == "cancelled"
			}),
			output.NewSortOp(output.SortKey{Column: "amount", Direction: output.Descending}),
		),
	)

	doc := builder.Build()

	fmt.Println("📊 Multiple tables, each with its own transformations:")
	fmt.Println("   • Top Performers: filter completed > $30k, sort by amount desc, limit 5")
	fmt.Println("   • Pending Sales: filter pending, sort by date desc, limit 10")
	fmt.Println("   • Regional Summary: filter completed, group by region with aggregates")
	fmt.Println("   • Cancelled Sales: filter cancelled, sort by amount desc")
	fmt.Println()

	out := output.NewOutput(
		output.WithFormat(output.Table()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	if err := out.Render(context.Background(), doc); err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n💡 Key Advantage: Each table has its own transformation logic!")
	fmt.Println("   This was not possible with the old Pipeline API which applied globally.")
}

// Custom aggregate function to count unique salespeople
func uniqueSalespeopleAggregate(records []output.Record, field string) any {
	seen := make(map[string]bool)
	for _, record := range records {
		if salesperson, ok := record["salesperson"].(string); ok {
			seen[salesperson] = true
		}
	}
	return len(seen)
}
