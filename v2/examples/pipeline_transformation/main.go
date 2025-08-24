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
	fmt.Println("🚀 Go-Output v2 Data Transformation Pipeline Examples")
	fmt.Println(strings.Repeat("=", 60))

	// Generate sample sales data
	salesData := generateSalesData(100)

	// Example 1: Basic Pipeline Operations
	fmt.Println("\n📊 Example 1: Basic Pipeline Operations")
	basicPipelineExample(salesData)

	// Example 2: Complex Analytics Pipeline
	fmt.Println("\n📈 Example 2: Complex Analytics Pipeline")
	complexAnalyticsExample(salesData)

	// Example 3: Aggregation and Reporting
	fmt.Println("\n📋 Example 3: Aggregation and Reporting")
	aggregationExample(salesData)

	// Example 4: Combining Pipeline with Byte Transformers
	fmt.Println("\n🎨 Example 4: Combining Pipeline with Styling")
	combinedTransformationExample(salesData)

	// Example 5: Error Handling and Performance Stats
	fmt.Println("\n🔍 Example 5: Error Handling and Performance")
	errorHandlingExample(salesData)
}

// generateSalesData creates realistic sample data for demonstrations
func generateSalesData(count int) []map[string]any {
	regions := []string{"North", "South", "East", "West", "Central"}
	salespeople := []string{"Alice Johnson", "Bob Smith", "Carol Davis", "David Wilson", "Emma Brown", "Frank Miller", "Grace Lee", "Henry Taylor"}

	data := make([]map[string]any, count)
	baseDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < count; i++ {
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

// basicPipelineExample demonstrates fundamental pipeline operations
func basicPipelineExample(salesData []map[string]any) {
	// Create initial document
	doc := output.New().
		Table("All Sales", salesData,
			output.WithKeys("id", "salesperson", "region", "amount", "status", "date")).
		Build()

	fmt.Printf("📄 Original data: %d records\n", len(salesData))

	// Apply pipeline transformations
	transformedDoc, err := doc.Pipeline().
		// Filter for completed high-value sales
		Filter(func(r output.Record) bool {
			return r["status"] == "completed" && r["amount"].(float64) > 20000
		}).
		// Sort by amount (highest first)
		SortBy("amount", output.Descending).
		// Get top 10 results
		Limit(10).
		Execute()

	if err != nil {
		log.Fatalf("Pipeline execution failed: %v", err)
	}

	// Output results
	out := output.NewOutput(
		output.WithFormat(output.Table),
		output.WithWriter(output.NewStdoutWriter()),
	)

	if err := out.Render(context.Background(), transformedDoc); err != nil {
		log.Fatal(err)
	}

	// Show transformation statistics
	if stats, found := transformedDoc.GetTransformStats(); found {
		fmt.Printf("📊 Transformation completed in %v\n", stats.Duration)
		fmt.Printf("   Input: %d records → Output: %d records\n", stats.InputRecords, stats.OutputRecords)
		fmt.Printf("   Filtered: %d records\n", stats.FilteredCount)
	} else {
		fmt.Printf("📊 Transformation completed successfully!\n")
	}
}

// complexAnalyticsExample shows advanced pipeline operations
func complexAnalyticsExample(salesData []map[string]any) {
	doc := output.New().
		Table("Sales Analysis", salesData,
			output.WithKeys("salesperson", "region", "amount", "date", "status", "product")).
		Build()

	// Complex analytical pipeline
	analyticsDoc, err := doc.Pipeline().
		// Focus on completed sales
		Filter(func(r output.Record) bool {
			return r["status"] == "completed"
		}).
		// Add calculated fields
		AddColumn("quarter", func(r output.Record) any {
			date := r["date"].(time.Time)
			quarter := (date.Month()-1)/3 + 1
			return fmt.Sprintf("Q%d", quarter)
		}).
		AddColumn("commission", func(r output.Record) any {
			amount := r["amount"].(float64)
			return amount * 0.05 // 5% commission
		}).
		AddColumn("tier", func(r output.Record) any {
			amount := r["amount"].(float64)
			if amount > 40000 {
				return "🥇 Premium"
			} else if amount > 20000 {
				return "🥈 Standard"
			}
			return "🥉 Basic"
		}).
		AddColumn("days_since_sale", func(r output.Record) any {
			saleDate := r["date"].(time.Time)
			return int(time.Since(saleDate).Hours() / 24)
		}).
		// Sort by commission (highest first)
		SortBy("commission", output.Descending).
		// Get top 15 for detailed analysis
		Limit(15).
		Execute()

	if err != nil {
		log.Fatalf("Analytics pipeline failed: %v", err)
	}

	// Show results in both table and JSON format
	out := output.NewOutput(
		output.WithFormats(output.Table, output.JSON),
		output.WithWriter(output.NewStdoutWriter()),
	)

	if err := out.Render(context.Background(), analyticsDoc); err != nil {
		log.Fatal(err)
	}

	// Performance statistics
	if stats, found := analyticsDoc.GetTransformStats(); found {
		fmt.Printf("\n🔍 Pipeline Performance:\n")
		for _, opStat := range stats.Operations {
			fmt.Printf("   %s: %v (%d records processed)\n",
				opStat.Name, opStat.Duration, opStat.RecordsProcessed)
		}
	} else {
		fmt.Printf("\n🔍 Analytics pipeline completed successfully!\n")
	}
}

// aggregationExample demonstrates GroupBy and aggregate functions
func aggregationExample(salesData []map[string]any) {
	doc := output.New().
		Table("Regional Sales", salesData,
			output.WithKeys("region", "salesperson", "amount", "status")).
		Build()

	// Group by region and calculate aggregate statistics
	regionReport, err := doc.Pipeline().
		// Only include completed sales
		Filter(func(r output.Record) bool {
			return r["status"] == "completed"
		}).
		// Group by region with multiple aggregates
		GroupBy(
			[]string{"region"},
			map[string]output.AggregateFunc{
				"total_sales": output.SumAggregate("amount"),
				"avg_sale":    output.AverageAggregate("amount"),
				"max_sale":    output.MaxAggregate("amount"),
				"min_sale":    output.MinAggregate("amount"),
				"sale_count":  output.CountAggregate(),
				"unique_reps": uniqueSalespeopleAggregate,
			},
		).
		// Sort by total sales (highest first)
		SortBy("total_sales", output.Descending).
		Execute()

	if err != nil {
		log.Fatalf("Aggregation pipeline failed: %v", err)
	}

	fmt.Println("🌍 Regional Performance Summary:")

	out := output.NewOutput(
		output.WithFormat(output.Table),
		output.WithWriter(output.NewStdoutWriter()),
	)

	if err := out.Render(context.Background(), regionReport); err != nil {
		log.Fatal(err)
	}

	// Show quarterly breakdown
	fmt.Println("\n📅 Quarterly Breakdown:")
	quarterlyDoc, err := doc.Pipeline().
		Filter(func(r output.Record) bool {
			return r["status"] == "completed"
		}).
		AddColumn("quarter", func(r output.Record) any {
			date := r["date"].(time.Time)
			quarter := (date.Month()-1)/3 + 1
			year := date.Year()
			return fmt.Sprintf("%d-Q%d", year, quarter)
		}).
		GroupBy(
			[]string{"quarter", "region"},
			map[string]output.AggregateFunc{
				"total_sales": output.SumAggregate("amount"),
				"sale_count":  output.CountAggregate(),
			},
		).
		SortBy("quarter", output.Ascending).
		Execute()

	if err != nil {
		log.Fatalf("Quarterly analysis failed: %v", err)
	}

	if err := out.Render(context.Background(), quarterlyDoc); err != nil {
		log.Fatal(err)
	}
}

// combinedTransformationExample shows data pipeline + byte transformers
func combinedTransformationExample(salesData []map[string]any) {
	doc := output.New().
		Table("Performance Report", salesData,
			output.WithKeys("salesperson", "region", "amount", "status")).
		Build()

	// Step 1: Data transformations with pipeline
	transformedDoc, err := doc.Pipeline().
		Filter(func(r output.Record) bool {
			return r["status"] == "completed"
		}).
		AddColumn("performance", func(r output.Record) any {
			amount := r["amount"].(float64)
			if amount > 40000 {
				return "excellent"
			} else if amount > 25000 {
				return "good"
			} else if amount > 15000 {
				return "average"
			}
			return "below_target"
		}).
		AddColumn("bonus_eligible", func(r output.Record) any {
			amount := r["amount"].(float64)
			if amount > 30000 {
				return "yes"
			}
			return "no"
		}).
		SortBy("amount", output.Descending).
		Limit(20).
		Execute()

	if err != nil {
		log.Fatalf("Combined transformation failed: %v", err)
	}

	fmt.Println("🎯 Top Performers with Styling:")

	// Step 2: Apply visual styling with byte transformers
	out := output.NewOutput(
		output.WithFormat(output.Table),
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

	if err := out.Render(context.Background(), transformedDoc); err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n💡 This example demonstrates:")
	fmt.Println("   • Data operations (filter, sort, calculate) done by pipeline")
	fmt.Println("   • Visual styling (colors, emoji) done by byte transformers")
	fmt.Println("   • Best of both worlds: structured data manipulation + presentation")
}

// errorHandlingExample demonstrates pipeline error handling
func errorHandlingExample(salesData []map[string]any) {
	doc := output.New().
		Table("Error Handling Demo", salesData,
			output.WithKeys("id", "salesperson", "amount")).
		Build()

	fmt.Println("⚠️  Demonstrating Error Handling:")

	// First demonstrate a successful error-free pipeline
	successfulDoc, err := doc.Pipeline().
		Filter(func(r output.Record) bool {
			// Safe type assertion with fallback
			if amount, ok := r["amount"].(float64); ok {
				return amount > 15000
			}
			return false // Safely handle missing or wrong type
		}).
		SortBy("amount", output.Descending).
		Limit(3).
		Execute()

	if err != nil {
		fmt.Printf("❌ Unexpected error in safe pipeline: %v\n", err)
	} else {
		fmt.Printf("✅ Safe pipeline completed successfully!\n")
		fmt.Printf("   Result: Filtered to high-value sales and limited to top 3\n")
	}

	// Now demonstrate what would happen with unsafe type assertions
	fmt.Println("\n🚨 Common Type Assertion Pitfalls (Avoided):")
	fmt.Println("   • Always use two-value type assertions: value, ok := r[\"field\"].(Type)")
	fmt.Println("   • Check if field exists before type assertion")
	fmt.Println("   • Provide fallback values for missing/wrong types")
	fmt.Println("   • Example: r[\"field\"].(string) can panic if field is nil or wrong type")

	// Create a pipeline with timeout to demonstrate context cancellation
	fmt.Println("\n⏱️  Demonstrating Context Timeout Handling:")

	// Use a very short timeout to trigger cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	_, err = doc.Pipeline().
		Filter(func(r output.Record) bool {
			// Add a small delay to ensure timeout is hit
			time.Sleep(1 * time.Millisecond)
			return r["status"] == "completed"
		}).
		ExecuteContext(ctx)

	// Handle the pipeline error with detailed context
	if err != nil {
		var pipelineErr *output.PipelineError
		if AsError(err, &pipelineErr) {
			fmt.Printf("❌ Context Timeout Error Caught:\n")
			fmt.Printf("   Operation: %s\n", pipelineErr.Operation)
			fmt.Printf("   Stage: %d\n", pipelineErr.Stage)
			fmt.Printf("   Cause: %v\n", pipelineErr.Cause)

			// Show context if available
			if len(pipelineErr.Context) > 0 {
				fmt.Printf("   Context:\n")
				for k, v := range pipelineErr.Context {
					fmt.Printf("     %s: %v\n", k, v)
				}
			}
		} else {
			fmt.Printf("❌ Context timeout error: %v\n", err)
		}
	} else {
		fmt.Printf("⚠️  Expected timeout error but none occurred\n")
	}

	// Now demonstrate successful pipeline with performance stats using the safe result from earlier
	fmt.Println("\n✅ Successful Pipeline with Performance Tracking:")
	
	// Use the successfulDoc from the earlier safe pipeline, not the timeout one

	// Show detailed performance statistics from the safe successful pipeline
	if stats, found := successfulDoc.GetTransformStats(); found {
		fmt.Printf("📊 Performance Statistics:\n")
		fmt.Printf("   Total Duration: %v\n", stats.Duration)
		fmt.Printf("   Input Records: %d\n", stats.InputRecords)
		fmt.Printf("   Output Records: %d\n", stats.OutputRecords)
		fmt.Printf("   Records Filtered: %d\n", stats.FilteredCount)
		fmt.Printf("   Filter Efficiency: %.1f%%\n",
			float64(stats.OutputRecords)/float64(stats.InputRecords)*100)

		fmt.Printf("\n🔍 Operation Breakdown:\n")
		for i, opStat := range stats.Operations {
			fmt.Printf("   %d. %s: %v (%d records)\n",
				i+1, opStat.Name, opStat.Duration, opStat.RecordsProcessed)
		}
	} else {
		fmt.Printf("📊 Pipeline completed successfully!\n")
	}

	out := output.NewOutput(
		output.WithFormat(output.Table),
		output.WithWriter(output.NewStdoutWriter()),
	)

	if err := out.Render(context.Background(), successfulDoc); err != nil {
		log.Fatal(err)
	}
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

// Helper function for error handling (simplified version of errors.As)
func AsError(err error, target **output.PipelineError) bool {
	if pe, ok := err.(*output.PipelineError); ok {
		*target = pe
		return true
	}
	return false
}
