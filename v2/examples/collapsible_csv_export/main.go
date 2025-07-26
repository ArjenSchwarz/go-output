// Package main demonstrates CSV export with detail columns for spreadsheet analysis
// This example shows how the CSV renderer automatically creates detail columns
// for collapsible content, enabling analysis in Excel or other spreadsheet tools.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	output "github.com/ArjenSchwarz/go-output/v2"
)

func main() {
	fmt.Println("📊 CSV Export with Detail Columns for Spreadsheet Analysis")
	fmt.Println("=========================================================")

	// Example 1: Product Catalog Export
	fmt.Println("\n🛍️  Example 1: Product Catalog Export")
	generateProductCatalogCSV()

	// Example 2: Customer Analysis Export
	fmt.Println("\n👥 Example 2: Customer Analysis Export")
	generateCustomerAnalysisCSV()

	// Example 3: Server Metrics Export
	fmt.Println("\n🖥️  Example 3: Server Metrics Export")
	generateServerMetricsCSV()

	fmt.Println("\n✅ All CSV files generated successfully!")
	fmt.Println("💡 Open these files in Excel/Google Sheets to see collapsible data in separate columns")
}

func generateProductCatalogCSV() {
	// Product data with complex nested information
	products := []map[string]any{
		{
			"id":           "PROD-001",
			"name":         "Laptop Pro 15\"",
			"category":     "Electronics",
			"price":        1299.99,
			"in_stock":     true,
			"specifications": map[string]any{
				"cpu":     "Intel i7-13700H",
				"ram":     "16GB DDR4",
				"storage": "512GB SSD",
				"display": "15.6\" 2560x1600",
				"weight":  "1.8kg",
			},
			"reviews": []string{
				"Excellent performance for development work",
				"Great display quality and color accuracy",
				"Battery life could be better",
				"Value for money considering the specs",
			},
			"variants": []string{
				"16GB RAM / 512GB SSD - $1299.99",
				"32GB RAM / 1TB SSD - $1599.99",
				"16GB RAM / 1TB SSD - $1399.99",
			},
			"supplier_info": map[string]any{
				"name":         "TechCorp Ltd",
				"contact":      "supplier@techcorp.com",
				"lead_time":    "5-7 days",
				"min_quantity": 10,
			},
		},
		{
			"id":           "PROD-002", 
			"name":         "Wireless Headphones",
			"category":     "Audio",
			"price":        199.99,
			"in_stock":     true,
			"specifications": map[string]any{
				"type":         "Over-ear",
				"connectivity": "Bluetooth 5.0",
				"battery":      "30 hours",
				"noise_cancel": true,
				"weight":       "280g",
			},
			"reviews": []string{
				"Amazing noise cancellation",
				"Comfortable for long listening sessions",
				"Great sound quality across all frequencies",
			},
			"variants": []string{
				"Black - $199.99",
				"White - $199.99", 
				"Blue - $219.99 (Limited Edition)",
			},
			"supplier_info": map[string]any{
				"name":         "AudioMax Inc",
				"contact":      "orders@audiomax.com",
				"lead_time":    "3-5 days",
				"min_quantity": 25,
			},
		},
		{
			"id":           "PROD-003",
			"name":         "Smart Watch Series 8",
			"category":     "Wearables",
			"price":        399.99,
			"in_stock":     false,
			"specifications": map[string]any{
				"display":     "1.9\" AMOLED",
				"battery":     "7 days",
				"water_proof": "50m",
				"sensors":     "Heart rate, GPS, SpO2",
				"weight":      "45g",
			},
			"reviews": []string{
				"Accurate fitness tracking",
				"Battery life exceeds expectations",
				"Smooth interface and quick responses",
				"Limited app ecosystem compared to competitors",
			},
			"variants": []string{
				"GPS Only - $399.99",
				"GPS + Cellular - $499.99",
				"Sport Band - $399.99",
				"Leather Band - $449.99",
			},
			"supplier_info": map[string]any{
				"name":         "WearTech Solutions",
				"contact":      "supply@weartech.com", 
				"lead_time":    "10-14 days",
				"min_quantity": 5,
			},
		},
	}

	// Create document with collapsible formatters for complex data
	doc := output.New().
		Header("Product Catalog Export").
		Text("Complete product information with expandable details for spreadsheet analysis").
		Table("Product Catalog", products,
			output.WithSchema(
				output.Field{Name: "id", Type: "string"},
				output.Field{Name: "name", Type: "string"},
				output.Field{Name: "category", Type: "string"},
				output.Field{Name: "price", Type: "float"},
				output.Field{Name: "in_stock", Type: "bool"},
				output.Field{Name: "specifications", Type: "object", Formatter: output.JSONFormatter(80, output.WithExpanded(false))},
				output.Field{Name: "reviews", Type: "array", Formatter: output.ErrorListFormatter(output.WithExpanded(false))},
				output.Field{Name: "variants", Type: "array", Formatter: output.ErrorListFormatter(output.WithExpanded(false))},
				output.Field{Name: "supplier_info", Type: "object", Formatter: output.JSONFormatter(100, output.WithExpanded(false))},
			)).
		Build()

	exportToCSV(doc, "product_catalog.csv", "Product Catalog")
}

func generateCustomerAnalysisCSV() {
	// Customer data with purchase history and behavior analysis
	customers := []map[string]any{
		{
			"customer_id":     "CUST-12345",
			"name":           "Alice Johnson",
			"email":          "alice@example.com",
			"registration":   "2023-03-15",
			"total_orders":   12,
			"lifetime_value": 2847.50,
			"segment":        "Premium",
			"purchase_history": []string{
				"2024-01-10: Laptop Pro 15\" - $1299.99",
				"2023-12-20: Wireless Mouse - $49.99", 
				"2023-11-15: External Monitor - $299.99",
				"2023-10-05: Keyboard Mechanical - $149.99",
				"2023-09-22: Webcam HD - $89.99",
			},
			"preferences": map[string]any{
				"categories": []string{"Electronics", "Accessories"},
				"price_range": "$100-$1500",
				"communication": "Email",
				"promotion_opt_in": true,
			},
			"support_tickets": []string{
				"2024-01-12: Laptop warranty question - Resolved",
				"2023-12-22: Shipping delay inquiry - Resolved",
				"2023-11-20: Product compatibility question - Resolved",
			},
		},
		{
			"customer_id":     "CUST-67890",
			"name":           "Bob Smith",
			"email":          "bob@example.com",
			"registration":   "2023-08-22",
			"total_orders":   5,
			"lifetime_value": 892.45,
			"segment":        "Regular",
			"purchase_history": []string{
				"2024-01-08: Wireless Headphones - $199.99",
				"2023-12-15: Phone Case - $24.99",
				"2023-11-30: Charging Cable - $19.99",
				"2023-10-20: Screen Protector - $12.99",
			},
			"preferences": map[string]any{
				"categories": []string{"Audio", "Mobile Accessories"},
				"price_range": "$10-$200",
				"communication": "SMS",
				"promotion_opt_in": false,
			},
			"support_tickets": []string{
				"2024-01-09: Headphones pairing issue - Resolved",
				"2023-12-16: Return request for defective item - Resolved",
			},
		},
		{
			"customer_id":     "CUST-54321",
			"name":           "Carol Davis",
			"email":          "carol@example.com",
			"registration":   "2023-01-10",
			"total_orders":   28,
			"lifetime_value": 5420.75,
			"segment":        "VIP",
			"purchase_history": []string{
				"2024-01-14: Smart Watch Series 8 - $399.99",
				"2024-01-05: Laptop Pro 15\" - $1299.99",
				"2023-12-28: Wireless Headphones - $199.99",
				"2023-12-20: External SSD - $149.99",
				"2023-12-15: Monitor Stand - $79.99",
				"... and 23 more orders",
			},
			"preferences": map[string]any{
				"categories": []string{"Electronics", "Wearables", "Computing"},
				"price_range": "$50-$2000",
				"communication": "Phone",
				"promotion_opt_in": true,
			},
			"support_tickets": []string{
				"2024-01-15: VIP shipping upgrade request - Resolved",
				"2024-01-06: Bulk order discount inquiry - Resolved",
				"2023-12-29: Product recommendation request - Resolved",
			},
		},
	}

	doc := output.New().
		Header("Customer Analysis Export").
		Text("Customer data with purchase history and behavioral insights").
		Table("Customer Analysis", customers,
			output.WithSchema(
				output.Field{Name: "customer_id", Type: "string"},
				output.Field{Name: "name", Type: "string"},
				output.Field{Name: "email", Type: "string"},
				output.Field{Name: "registration", Type: "string"},
				output.Field{Name: "total_orders", Type: "int"},
				output.Field{Name: "lifetime_value", Type: "float"},
				output.Field{Name: "segment", Type: "string"},
				output.Field{Name: "purchase_history", Type: "array", Formatter: output.ErrorListFormatter(output.WithExpanded(false))},
				output.Field{Name: "preferences", Type: "object", Formatter: output.JSONFormatter(120, output.WithExpanded(false))},
				output.Field{Name: "support_tickets", Type: "array", Formatter: output.ErrorListFormatter(output.WithExpanded(false))},
			)).
		Build()

	exportToCSV(doc, "customer_analysis.csv", "Customer Analysis")
}

func generateServerMetricsCSV() {
	// Server monitoring data with detailed metrics
	servers := []map[string]any{
		{
			"server_id":    "SRV-WEB-01",
			"hostname":     "web-prod-01.company.com",
			"environment":  "Production",
			"role":         "Web Server",
			"status":       "Healthy",
			"cpu_usage":    23.5,
			"memory_usage": 68.2,
			"disk_usage":   45.8,
			"network_io":   "125 MB/s",
			"alerts": []string{
				"2024-01-15 14:20:00: Memory usage above 80% for 5 minutes",
				"2024-01-15 13:45:00: High network latency detected",
			},
			"performance_metrics": map[string]any{
				"requests_per_second": 450,
				"response_time_avg": "85ms",
				"error_rate": "0.02%",
				"uptime": "99.95%",
			},
			"configurations": map[string]any{
				"nginx_version": "1.21.6",
				"php_version": "8.1.0",
				"ssl_cert_expiry": "2024-06-15",
				"backup_frequency": "daily",
			},
		},
		{
			"server_id":    "SRV-DB-01",
			"hostname":     "db-prod-01.company.com",
			"environment":  "Production",
			"role":         "Database",
			"status":       "Warning",
			"cpu_usage":    78.9,
			"memory_usage": 85.4,
			"disk_usage":   72.3,
			"network_io":   "89 MB/s", 
			"alerts": []string{
				"2024-01-15 14:25:00: High CPU usage sustained for 10 minutes",
				"2024-01-15 14:15:00: Memory usage above 85%",
				"2024-01-15 13:30:00: Slow query detected: avg execution time 2.5s",
			},
			"performance_metrics": map[string]any{
				"queries_per_second": 125,
				"avg_query_time": "45ms",
				"connection_count": "95/100",
				"replication_lag": "50ms",
			},
			"configurations": map[string]any{
				"postgresql_version": "14.6",
				"max_connections": 100,
				"shared_buffers": "2GB",
				"backup_frequency": "hourly",
			},
		},
		{
			"server_id":    "SRV-CACHE-01",
			"hostname":     "cache-prod-01.company.com", 
			"environment":  "Production",
			"role":         "Cache",
			"status":       "Healthy",
			"cpu_usage":    15.2,
			"memory_usage": 45.7,
			"disk_usage":   12.8,
			"network_io":   "67 MB/s",
			"alerts": []string{},
			"performance_metrics": map[string]any{
				"hit_rate": "94.8%",
				"operations_per_second": 2500,
				"avg_response_time": "0.5ms",
				"evicted_keys": 125,
			},
			"configurations": map[string]any{
				"redis_version": "7.0.5",
				"max_memory": "4GB",
				"persistence": "RDB + AOF",
				"backup_frequency": "daily",
			},
		},
	}

	doc := output.New().
		Header("Server Metrics Export").
		Text("Server monitoring data with detailed performance metrics and configurations").
		Table("Server Metrics", servers,
			output.WithSchema(
				output.Field{Name: "server_id", Type: "string"},
				output.Field{Name: "hostname", Type: "string"},
				output.Field{Name: "environment", Type: "string"},
				output.Field{Name: "role", Type: "string"},
				output.Field{Name: "status", Type: "string"},
				output.Field{Name: "cpu_usage", Type: "float"},
				output.Field{Name: "memory_usage", Type: "float"},
				output.Field{Name: "disk_usage", Type: "float"},
				output.Field{Name: "network_io", Type: "string"},
				output.Field{Name: "alerts", Type: "array", Formatter: output.ErrorListFormatter(output.WithExpanded(false))},
				output.Field{Name: "performance_metrics", Type: "object", Formatter: output.JSONFormatter(150, output.WithExpanded(false))},
				output.Field{Name: "configurations", Type: "object", Formatter: output.JSONFormatter(150, output.WithExpanded(false))},
			)).
		Build()

	exportToCSV(doc, "server_metrics.csv", "Server Metrics")
}

func exportToCSV(doc *output.Document, filename, description string) {
	// Create CSV renderer with collapsible support
	csvRenderer := output.NewCSVRendererWithCollapsible(output.DefaultRendererConfig)
	csvFormat := output.Format{
		Name:     "csv",
		Renderer: csvRenderer,
	}

	// Create file writer
	outputDir := "csv_exports"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Printf("Warning: Could not create output directory: %v", err)
		outputDir = "." // Fall back to current directory
	}

	fileWriter, err := output.NewFileWriter(outputDir, filename)
	if err != nil {
		log.Fatalf("Failed to create file writer for %s: %v", filename, err)
	}

	// Create output with CSV format and file writer
	out := output.NewOutput(
		output.WithFormat(csvFormat),
		output.WithWriter(fileWriter),
	)

	ctx := context.Background()
	err = out.Render(ctx, doc)
	if err != nil {
		log.Fatalf("Failed to export %s: %v", description, err)
	}

	fmt.Printf("✅ %s exported to %s/%s\n", description, outputDir, filename)
	
	// Also show a preview in the console
	fmt.Printf("📄 Preview of %s (showing structure with detail columns):\n", filename)
	
	// Create a console output to show the structure
	consoleOut := output.NewOutput(
		output.WithFormat(csvFormat),
		output.WithWriter(output.NewStdoutWriter()),
	)
	
	// Show just the header and first row for preview
	previewDoc := output.New().
		Text(fmt.Sprintf("Preview of %s:", description)).
		Text("(Full data exported to file with separate detail columns)").
		Build()
		
	consoleOut.Render(ctx, previewDoc)
	fmt.Println("💡 Open the CSV file in Excel to see the collapsible data in separate columns")
	fmt.Println(strings.Repeat("-", 80))
}

// Helper function to simulate strings package
var strings = struct {
	Repeat func(string, int) string
}{
	Repeat: func(s string, count int) string {
		result := ""
		for i := 0; i < count; i++ {
			result += s
		}
		return result
	},
}