// Package main demonstrates terminal analysis tool with expandable section reports
// This example shows how to use CollapsibleSection to create hierarchical reports
// with progressive disclosure for terminal-based analysis tools.
package main

import (
	"context"
	"fmt"
	"log"

	output "github.com/ArjenSchwarz/go-output/v2"
)

func main() {
	fmt.Println("🖥️  Terminal Analysis Tool with Expandable Sections")
	fmt.Println("==================================================")

	// Example 1: System Performance Analysis
	fmt.Println("\n📊 Example 1: System Performance Analysis")
	generateSystemAnalysisReport()

	// Example 2: Code Quality Report with Nested Sections
	fmt.Println("\n📈 Example 2: Code Quality Report with Nested Sections")
	generateCodeQualityReport()

	// Example 3: Infrastructure Health Check
	fmt.Println("\n🏗️  Example 3: Infrastructure Health Check")
	generateInfrastructureReport()
}

func generateSystemAnalysisReport() {
	// Create performance data
	cpuData := []map[string]any{
		{"process": "node", "cpu": 45.2, "memory": 512, "threads": 8},
		{"process": "postgres", "cpu": 23.1, "memory": 256, "threads": 12},
		{"process": "nginx", "cpu": 5.8, "memory": 64, "threads": 4},
	}

	memoryData := []map[string]any{
		{"component": "Application", "used": 2048, "available": 6144, "percent": 25.0},
		{"component": "Database", "used": 1536, "available": 4608, "percent": 25.0},
		{"component": "Cache", "used": 512, "available": 1536, "percent": 25.0},
	}

	diskData := []map[string]any{
		{"mount": "/", "used": 45, "total": 100, "percent": 45.0},
		{"mount": "/var/log", "used": 8, "total": 20, "percent": 40.0},
		{"mount": "/tmp", "used": 2, "total": 10, "percent": 20.0},
	}

	// Create table content
	cpuTable, _ := output.NewTableContent("CPU Usage by Process", cpuData,
		output.WithSchema(
			output.Field{Name: "process", Type: "string"},
			output.Field{Name: "cpu", Type: "float"},
			output.Field{Name: "memory", Type: "int"},
			output.Field{Name: "threads", Type: "int"},
		))

	memoryTable, _ := output.NewTableContent("Memory Usage by Component", memoryData,
		output.WithKeys("component", "used", "available", "percent"))

	diskTable, _ := output.NewTableContent("Disk Usage by Mount", diskData,
		output.WithKeys("mount", "used", "total", "percent"))

	// Create collapsible sections
	performanceSection := output.NewCollapsibleSection("Performance Metrics",
		[]output.Content{
			cpuTable,
			memoryTable,
			diskTable,
		},
		output.WithSectionExpanded(true),
		output.WithSectionLevel(1),
	)

	// Create alerts section
	alertsData := []map[string]any{
		{
			"severity": "Warning",
			"component": "CPU",
			"message": "High CPU usage detected on node process",
			"details": []string{
				"Current usage: 45.2%",
				"Threshold: 40.0%",
				"Duration: 15 minutes",
				"Suggested action: Check for memory leaks or optimize queries",
			},
		},
		{
			"severity": "Info",
			"component": "Disk",
			"message": "Disk usage approaching threshold on root partition",
			"details": []string{
				"Current usage: 45%",
				"Threshold: 50%",
				"Estimated time to full: 2 days",
				"Suggested action: Clean up log files or expand storage",
			},
		},
	}

	alertsTable, _ := output.NewTableContent("System Alerts", alertsData,
		output.WithSchema(
			output.Field{Name: "severity", Type: "string"},
			output.Field{Name: "component", Type: "string"},
			output.Field{Name: "message", Type: "string"},
			output.Field{Name: "details", Type: "array", Formatter: output.ErrorListFormatter(output.WithExpanded(false))},
		))

	alertsSection := output.NewCollapsibleSection("System Alerts",
		[]output.Content{alertsTable},
		output.WithSectionExpanded(false),
		output.WithSectionLevel(1),
	)

	// Build complete report
	doc := output.New().
		Header("System Performance Analysis Report").
		Text("Analysis completed at 2024-01-15 14:30:00 UTC").
		Text("Overall system health: **GOOD** with 2 items requiring attention").
		AddCollapsibleSection("Detailed Analysis", []output.Content{
			performanceSection,
			alertsSection,
		}, output.WithSectionExpanded(true)).
		Text("**Summary**: System is performing well with minor optimization opportunities.").
		Build()

	// Render as table format for terminal display
	renderReport(doc, output.Table, "System Performance")
}

func generateCodeQualityReport() {
	// Create code metrics data
	qualityData := []map[string]any{
		{
			"metric": "Test Coverage",
			"value": "78%",
			"target": "80%",
			"status": "Below Target",
			"files": []string{
				"src/auth/login.ts: 65%",
				"src/utils/validation.ts: 45%", 
				"src/components/Modal.tsx: 90%",
				"src/hooks/useApi.ts: 72%",
			},
		},
		{
			"metric": "Code Complexity",
			"value": "7.2",
			"target": "< 10.0",
			"status": "Good",
			"files": []string{
				"src/services/userService.ts: 12.1 (high)",
				"src/utils/dataProcessor.ts: 8.9",
				"src/components/Dashboard.tsx: 6.5",
			},
		},
		{
			"metric": "Duplication",
			"value": "5.2%",
			"target": "< 5.0%",
			"status": "Above Target",
			"files": []string{
				"src/components/Button.tsx and IconButton.tsx",
				"src/utils/formatters.ts: repeated date logic",
				"src/hooks/useForm.ts: validation duplicates",
			},
		},
	}

	// Dependency analysis
	depsData := []map[string]any{
		{
			"category": "Outdated",
			"count": 12,
			"details": []string{
				"react: 17.0.2 → 18.2.0 (major)",
				"typescript: 4.5.0 → 5.0.0 (major)",
				"eslint: 8.0.0 → 8.45.0 (minor)",
				"@types/node: 16.0.0 → 20.0.0 (major)",
			},
		},
		{
			"category": "Vulnerabilities",
			"count": 3,
			"details": []string{
				"lodash: CVE-2021-23337 (high)",
				"minimist: CVE-2020-7598 (moderate)",
				"node-forge: CVE-2022-24771 (moderate)",
			},
		},
	}

	// Create tables
	qualityTable, _ := output.NewTableContent("Code Quality Metrics", qualityData,
		output.WithSchema(
			output.Field{Name: "metric", Type: "string"},
			output.Field{Name: "value", Type: "string"},
			output.Field{Name: "target", Type: "string"},
			output.Field{Name: "status", Type: "string"},
			output.Field{Name: "files", Type: "array", Formatter: output.ErrorListFormatter(output.WithExpanded(false))},
		))

	depsTable, _ := output.NewTableContent("Dependency Analysis", depsData,
		output.WithSchema(
			output.Field{Name: "category", Type: "string"},
			output.Field{Name: "count", Type: "int"},
			output.Field{Name: "details", Type: "array", Formatter: output.ErrorListFormatter(output.WithExpanded(false))},
		))

	// Create nested sections
	qualitySection := output.NewCollapsibleSection("Code Quality Metrics",
		[]output.Content{qualityTable},
		output.WithSectionLevel(2),
		output.WithSectionExpanded(true),
	)

	dependencySection := output.NewCollapsibleSection("Dependency Analysis", 
		[]output.Content{depsTable},
		output.WithSectionLevel(2),
		output.WithSectionExpanded(false),
	)

	// Build report with nested collapsible sections
	doc := output.New().
		Header("Code Quality Analysis Report").
		Text("**Project**: E-commerce Platform").
		Text("**Analyzed**: 127 files, 15,482 lines of code").
		AddCollapsibleSection("Quality Assessment", []output.Content{
			qualitySection,
			dependencySection,
		}, output.WithSectionExpanded(true), output.WithSectionLevel(1)).
		Text("**Recommendations**:").
		Text("1. Increase test coverage for authentication modules").
		Text("2. Refactor high-complexity service files").
		Text("3. Update major dependencies and fix vulnerabilities").
		Build()

	renderReport(doc, output.Table, "Code Quality")
}

func generateInfrastructureReport() {
	// Service health data
	servicesData := []map[string]any{
		{
			"service": "API Gateway",
			"status": "Healthy",
			"uptime": "99.9%",
			"response_time": "45ms",
			"errors": []string{},
		},
		{
			"service": "Auth Service",
			"status": "Warning",
			"uptime": "98.2%",
			"response_time": "120ms",
			"errors": []string{
				"2024-01-15 14:25:00: Rate limit exceeded",
				"2024-01-15 13:45:00: Database connection timeout",
			},
		},
		{
			"service": "Payment Service",
			"status": "Critical",
			"uptime": "95.1%",
			"response_time": "250ms",
			"errors": []string{
				"2024-01-15 14:20:00: Payment processor unavailable",
				"2024-01-15 14:15:00: SSL certificate validation failed",
				"2024-01-15 14:10:00: High memory usage detected",
			},
		},
	}

	// Database health
	dbData := []map[string]any{
		{
			"database": "Primary (PostgreSQL)",
			"status": "Healthy",
			"connections": "45/100",
			"replication_lag": "0ms",
			"issues": []string{},
		},
		{
			"database": "Cache (Redis)",
			"status": "Warning", 
			"connections": "80/100",
			"replication_lag": "N/A",
			"issues": []string{
				"Memory usage: 85% (threshold: 80%)",
				"Evicted keys: 1,250 in last hour",
			},
		},
	}

	// Create tables
	servicesTable, _ := output.NewTableContent("Service Health Status", servicesData,
		output.WithSchema(
			output.Field{Name: "service", Type: "string"},
			output.Field{Name: "status", Type: "string"},
			output.Field{Name: "uptime", Type: "string"},
			output.Field{Name: "response_time", Type: "string"},
			output.Field{Name: "errors", Type: "array", Formatter: output.ErrorListFormatter(output.WithExpanded(false))},
		))

	dbTable, _ := output.NewTableContent("Database Health", dbData,
		output.WithSchema(
			output.Field{Name: "database", Type: "string"},
			output.Field{Name: "status", Type: "string"},
			output.Field{Name: "connections", Type: "string"},
			output.Field{Name: "replication_lag", Type: "string"},
			output.Field{Name: "issues", Type: "array", Formatter: output.ErrorListFormatter(output.WithExpanded(false))},
		))

	// Build infrastructure health report
	doc := output.New().
		Header("Infrastructure Health Check").
		Text("**Environment**: Production").
		Text("**Check Time**: 2024-01-15 14:30:00 UTC").
		Text("**Overall Status**: ⚠️ **WARNING** - Payment service requires immediate attention").
		AddCollapsibleSection("Service Status", []output.Content{
			servicesTable,
		}, output.WithSectionExpanded(true)).
		AddCollapsibleSection("Database Status", []output.Content{
			dbTable,
		}, output.WithSectionExpanded(false)).
		Text("**Action Items**:").
		Text("1. 🔴 **URGENT**: Investigate payment service SSL issues").
		Text("2. 🟡 **MEDIUM**: Scale Redis cache or optimize memory usage").
		Text("3. 🟡 **MEDIUM**: Review auth service rate limiting configuration").
		Build()

	renderReport(doc, output.Table, "Infrastructure Health")
}

func renderReport(doc *output.Document, format output.Format, reportName string) {
	// Create output with custom table renderer configuration for better terminal display
	tableRenderer := output.NewTableRendererWithCollapsible("Default", output.RendererConfig{
		ForceExpansion:       false, // Allow collapsed sections by default
		TableHiddenIndicator: "[click to expand]",
	})
	
	customFormat := output.Format{
		Name:     "table",
		Renderer: tableRenderer,
	}

	out := output.NewOutput(
		output.WithFormat(customFormat),
		output.WithWriter(output.NewStdoutWriter()),
	)

	ctx := context.Background()
	err := out.Render(ctx, doc)
	if err != nil {
		log.Fatalf("Failed to render %s report: %v", reportName, err)
	}

	fmt.Printf("\n✅ %s report rendered successfully\n", reportName)
	fmt.Println("💡 Tip: In a real terminal tool, you could add --expand flag to show all details")
	fmt.Println(strings.Repeat("=", 80))
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