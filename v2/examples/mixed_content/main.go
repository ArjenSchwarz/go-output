package main

import (
	"context"
	"log"

	output "github.com/ArjenSchwarz/go-output/v2"
)

func main() {
	// This example demonstrates v2's ability to mix different content types
	// in a single document with proper organization and formatting

	// Sample data for different content types
	servers := []map[string]any{
		{"Name": "web-01", "Status": "Running", "CPU": "45%", "Memory": "2.1GB", "Uptime": "15 days"},
		{"Name": "web-02", "Status": "Running", "CPU": "38%", "Memory": "1.8GB", "Uptime": "15 days"},
		{"Name": "db-01", "Status": "Warning", "CPU": "72%", "Memory": "6.2GB", "Uptime": "8 days"},
	}

	metrics := []map[string]any{
		{"Metric": "Response Time", "Value": "245ms", "Threshold": "500ms", "Status": "OK"},
		{"Metric": "Error Rate", "Value": "0.02%", "Threshold": "1.0%", "Status": "OK"},
		{"Metric": "Throughput", "Value": "1,250 req/s", "Threshold": "1,000 req/s", "Status": "Good"},
	}

	incidents := []map[string]any{
		{"ID": "INC-001", "Severity": "High", "Status": "Resolved", "Duration": "45 min"},
		{"ID": "INC-002", "Severity": "Medium", "Status": "In Progress", "Duration": "15 min"},
	}

	// Create a comprehensive document with mixed content types
	doc := output.New().
		// Document header
		Header("System Health Dashboard").
		Text("Generated: 2024-01-15 14:30:00 UTC").
		Text("").

		// Executive summary section
		Section("Executive Summary", func(b *output.Builder) {
			b.Text("• System Status: Operational with 1 warning")
			b.Text("• Total Servers: 3 (2 running normally, 1 high CPU)")
			b.Text("• Active Incidents: 1 in progress")
			b.Text("• Overall Health: 95%")
		}).

		// Server status section with table
		Section("Server Status", func(b *output.Builder) {
			b.Text("Current status of all production servers:")
			b.Table("Production Servers", servers,
				output.WithKeys("Name", "Status", "CPU", "Memory", "Uptime"))
			b.Text("⚠️  Warning: db-01 showing high CPU usage")
		}).

		// Performance metrics section
		Section("Performance Metrics", func(b *output.Builder) {
			b.Text("Key performance indicators for the last 24 hours:")
			b.Table("Metrics", metrics,
				output.WithKeys("Metric", "Value", "Threshold", "Status"))
			b.Text("✅ All metrics within acceptable thresholds")
		}).

		// Incident tracking section
		Section("Incident Management", func(b *output.Builder) {
			b.Table("Recent Incidents", incidents,
				output.WithKeys("ID", "Severity", "Status", "Duration"))
			b.Text("")
			b.Text("Incident Details:")
			b.Text("• INC-001: Database connection timeout - Resolved by restarting connection pool")
			b.Text("• INC-002: High CPU on db-01 - Investigation ongoing")
		}).

		// Raw HTML for web output (format-specific content)
		Raw("html", []byte(`
			<div style="background: #f0f8ff; padding: 10px; margin: 10px 0; border-left: 4px solid #0066cc;">
				<strong>Note:</strong> This dashboard is updated every 5 minutes. 
				For real-time monitoring, please check the ops dashboard.
			</div>
		`)).

		// Footer
		Text("").
		Text("---").
		Text("Dashboard v2.0 | Contact: ops-team@company.com").
		Build()

	// Configure output for multiple formats to show content adaptation
	out := output.NewOutput(
		output.WithFormats(
			output.Table,    // Console-friendly
			output.HTML,     // Web-friendly with raw HTML included
			output.Markdown, // Documentation-friendly
		),
		output.WithWriter(output.NewStdoutWriter()),
	)

	if err := out.Render(context.Background(), doc); err != nil {
		log.Fatalf("Failed to render mixed content document: %v", err)
	}

	log.Println("\nMixed Content Features Demonstrated:")
	log.Println("✓ Headers and formatted text")
	log.Println("✓ Hierarchical sections with nested content")
	log.Println("✓ Multiple tables with different schemas")
	log.Println("✓ Format-specific raw content (HTML)")
	log.Println("✓ Proper content ordering and spacing")
	log.Println("✓ Adaptive rendering across formats")
}