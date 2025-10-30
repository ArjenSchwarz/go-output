package main

import (
	"context"
	"fmt"
	"log"
	"time"

	output "github.com/ArjenSchwarz/go-output/v2"
)

func main() {
	fmt.Println("=== Complete Migration Example ===")
	fmt.Println("Real-world v1 → v2 migration with performance comparison")
	fmt.Println()

	// This example shows a complete, realistic migration from a hypothetical
	// v1 application that generates an infrastructure monitoring report

	// Simulate realistic data sets
	servers := generateServerData()
	databases := generateDatabaseData()
	metrics := generateMetricsData()
	incidents := generateIncidentData()

	fmt.Println("📊 Generating Infrastructure Report")
	fmt.Println("   • Servers:", len(servers))
	fmt.Println("   • Databases:", len(databases))
	fmt.Println("   • Metrics:", len(metrics))
	fmt.Println("   • Incidents:", len(incidents))
	fmt.Println()

	// Show what the v1 code would have looked like
	fmt.Println("🕰️  V1 Implementation (Hypothetical):")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	showV1HypotheticalCode()
	fmt.Println()

	// Now show the v2 implementation
	fmt.Println("🚀 V2 Implementation (Actual):")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Time the v2 implementation
	start := time.Now()

	// Create comprehensive document using v2 API
	doc := output.New().
		SetMetadata("report_type", "infrastructure").
		SetMetadata("generated_at", time.Now().Unix()).
		SetMetadata("version", "2.0").

		// Document header
		Header("Infrastructure Monitoring Report").
		Text("Generated: "+time.Now().Format("Monday, January 2, 2006 at 15:04:05 MST")).
		Text("Coverage: Production Environment").
		Text("").

		// Executive summary
		Section("Executive Summary", func(b *output.Builder) {
			b.Text("📈 **System Health**: 94% (Excellent)")
			b.Text("🖥️  **Total Servers**: 25 (23 healthy, 2 warnings)")
			b.Text("🗄️  **Databases**: 8 (7 operational, 1 maintenance)")
			b.Text("⚠️  **Active Incidents**: 2 (1 high, 1 medium priority)")
			b.Text("🎯 **SLA Compliance**: 99.7%")
		}).

		// Server infrastructure section
		Section("Server Infrastructure", func(b *output.Builder) {
			b.Text("Current status of all production servers ordered by criticality:")
			b.Table("Production Servers", servers,
				// Exact key ordering: critical info first
				output.WithKeys("Name", "Status", "CPU", "Memory", "Disk", "Network", "Uptime", "Location"))
			b.Text("")
			b.Text("🔍 **Analysis:**")
			b.Text("• High CPU usage on app-server-03 requires investigation")
			b.Text("• Memory utilization within normal parameters")
			b.Text("• All servers reporting healthy network connectivity")
		}).

		// Database section
		Section("Database Systems", func(b *output.Builder) {
			b.Text("Database cluster status and performance metrics:")
			b.Table("Database Instances", databases,
				// Different key ordering for databases
				output.WithKeys("Instance", "Type", "Status", "Connections", "CPU", "Memory", "Storage", "Replication"))
			b.Text("")
			b.Text("🔍 **Database Health:**")
			b.Text("• Primary databases operating normally")
			b.Text("• Replica lag within acceptable limits (<100ms)")
			b.Text("• Scheduled maintenance: db-replica-02 (Sunday 02:00-04:00)")
		}).

		// Performance metrics section
		Section("Performance Metrics", func(b *output.Builder) {
			b.Text("Key performance indicators for the last 24 hours:")
			b.Table("System Metrics", metrics,
				output.WithKeys("Metric", "Current", "Average", "Peak", "Threshold", "Status"))
			b.Text("")
			b.Text("📊 **Performance Summary:**")
			b.Text("• Response times consistently below SLA thresholds")
			b.Text("• Error rates at historical lows (0.02%)")
			b.Text("• Throughput increased 15% compared to last month")
		}).

		// Incident management
		Section("Incident Management", func(b *output.Builder) {
			b.Table("Active Incidents", incidents,
				output.WithKeys("ID", "Priority", "Component", "Status", "Duration", "Assignee"))
			b.Text("")
			b.Text("🚨 **Incident Details:**")
			b.Text("• INC-2024-0157: High CPU on app-server-03 - Under investigation")
			b.Text("• INC-2024-0156: Database connection spike - Monitoring")
			b.Text("• Recent Resolution: INC-2024-0155 - Network latency (Resolved)")
		}).

		// Raw HTML for web dashboard integration
		Raw("html", []byte(`
			<div class="alert alert-info">
				<h4>🔗 Integration Links</h4>
				<ul>
					<li><a href="/grafana/infrastructure">Live Grafana Dashboard</a></li>
					<li><a href="/prometheus/alerts">Prometheus Alert Manager</a></li>
					<li><a href="/kibana/logs">Log Analysis (Kibana)</a></li>
				</ul>
			</div>
		`)).

		// Footer
		Text("").
		Text("─────────────────────────────────────────────────────").
		Text("🏢 Infrastructure Team | 📧 ops@company.com | 📱 +1-555-OPS-TEAM").
		Text("🔄 Next report: " + time.Now().Add(24*time.Hour).Format("Monday, January 2, 2006")).
		Build()

	// Create comprehensive output configuration
	fileWriter, err := output.NewFileWriter("./output", "infrastructure_report.{format}")
	if err != nil {
		log.Fatalf("Failed to create file writer: %v", err)
	}

	// Configure multiple formats and transformations
	out := output.NewOutput(
		// Multiple output formats for different audiences
		output.WithFormats(
			output.MarkdownWithOptions(true, map[string]string{ // Technical team
				"title":       "Infrastructure Report",
				"date":        time.Now().Format("2006-01-02"),
				"team":        "Infrastructure",
				"environment": "Production",
			}),
			output.HTML(),                          // Web dashboard
			output.JSON(),                          // API consumption
			output.TableWithStyle("ColoredBright"), // Console/terminal
		),

		// Data transformations
		output.WithTransformers(
			output.NewEnhancedEmojiTransformer(), // Convert emoji codes
			output.NewColorTransformer(),         // Add colors
		),

		// Multiple output destinations
		output.WithWriters(
			output.NewStdoutWriter(), // Console display
			fileWriter,               // File export
		),

		// Progress tracking for large datasets
		output.WithProgress(output.NewProgressForFormats(
			[]output.Format{output.Markdown(), output.HTML(), output.JSON(), output.Table()},
			output.WithProgressColor(output.ProgressColorBlue),
			output.WithProgressStatus("Generating infrastructure report"),
		)),
	)

	// Render the complete report
	if err := out.Render(context.Background(), doc); err != nil {
		log.Fatalf("Failed to render infrastructure report: %v", err)
	}

	duration := time.Since(start)

	fmt.Printf("\n✅ Report Generation Complete (%v)\n", duration)
	fmt.Println("Generated files:")
	fmt.Println("📄 ./output/infrastructure_report.md   - Technical documentation")
	fmt.Println("🌐 ./output/infrastructure_report.html - Web dashboard")
	fmt.Println("📊 ./output/infrastructure_report.json - API/automation")
	fmt.Println("💻 Console output                      - Terminal display")
	fmt.Println()

	// Performance and feature comparison
	showMigrationComparison(duration)
}

// showV1HypotheticalCode demonstrates what the equivalent v1 code would have looked like
func showV1HypotheticalCode() {
	v1Code := `// V1 - Multiple separate operations, global state
settings1 := format.NewOutputSettings()
settings1.OutputFormat = "table"  
settings1.OutputFile = "servers.html"
settings1.OutputFileFormat = "html"
settings1.TableStyle = "ColoredBright"
settings1.UseEmoji = true
settings1.UseColors = true

// Server report (separate operation)
serverOutput := &format.OutputArray{
    Settings: settings1,
    Keys: []string{"Name", "Status", "CPU", "Memory", "Disk", "Network", "Uptime", "Location"},
}
serverOutput.AddHeader("Production Servers")
serverOutput.AddContents(servers)
serverOutput.Write()

// Database report (separate operation, different keys)
settings2 := format.NewOutputSettings() // Duplicate configuration!
settings2.OutputFormat = "json"
settings2.OutputFile = "databases.json"
dbOutput := &format.OutputArray{
    Settings: settings2,  
    Keys: []string{"Instance", "Type", "Status", "Connections", "CPU", "Memory"},
}
dbOutput.AddContents(databases)
dbOutput.Write()

// Metrics report (third separate operation)
settings3 := format.NewOutputSettings() // More duplication!
settings3.OutputFormat = "markdown"
settings3.OutputFile = "metrics.md"
metricsOutput := &format.OutputArray{
    Settings: settings3,
    Keys: []string{"Metric", "Current", "Average", "Peak", "Threshold", "Status"},
}
metricsOutput.AddContents(metrics)
metricsOutput.Write()

// Problems with V1 approach:
// ❌ Global state causes race conditions
// ❌ Unpredictable key ordering due to map iteration
// ❌ Multiple operations for single logical report
// ❌ Configuration duplication
// ❌ No mixed content support
// ❌ Limited error handling
// ❌ No progress tracking
// ❌ Separate operations = inconsistent data`

	fmt.Print(v1Code)
}

// showMigrationComparison highlights the benefits of v2
func showMigrationComparison(v2Duration time.Duration) {
	fmt.Println("📈 Migration Benefits Analysis:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	fmt.Println("🎯 **Key Improvements:**")
	fmt.Printf("   ⚡ Performance: Single operation (%v total)\n", v2Duration)
	fmt.Println("   🎨 Visual: Exact key order preservation")
	fmt.Println("   🔒 Safety: Thread-safe, no race conditions")
	fmt.Println("   📐 Structure: Unified document with sections")
	fmt.Println("   🎛️  Configuration: Functional options vs structs")
	fmt.Println("   🚦 Progress: Format-aware progress indicators")
	fmt.Println("   📊 Output: Multiple formats from single document")
	fmt.Println("   🛡️  Errors: Structured error handling")
	fmt.Println("   🧩 Content: Mixed content types (tables, text, raw)")
	fmt.Println()

	fmt.Println("📊 **Feature Comparison:**")
	fmt.Println("   Feature                  | V1        | V2")
	fmt.Println("   -------------------------|-----------|----------")
	fmt.Println("   Key Order Preservation   | ❌ Random | ✅ Exact")
	fmt.Println("   Thread Safety           | ❌ Races  | ✅ Safe")
	fmt.Println("   Mixed Content           | ❌ No     | ✅ Yes")
	fmt.Println("   Multiple Tables         | ⚠️ Buffer | ✅ Native")
	fmt.Println("   Progress Tracking       | ⚠️ Basic  | ✅ Enhanced")
	fmt.Println("   Error Context           | ❌ Minimal| ✅ Rich")
	fmt.Println("   Document Structure      | ❌ Flat   | ✅ Hierarchical")
	fmt.Println("   Configuration           | ⚠️ Struct | ✅ Functional")
	fmt.Println()

	fmt.Println("💡 **Migration Effort:**")
	fmt.Println("   🕐 Time: ~2-4 hours for typical application")
	fmt.Println("   🔧 Automation: 80% convertible with migration tool")
	fmt.Println("   📚 Learning: Familiar patterns, cleaner API")
	fmt.Println("   🧪 Testing: Better testability with immutable documents")
	fmt.Println("   ⬆️ Upgrade: Zero-downtime deployment (separate module)")
}

// Data generation functions for realistic example
func generateServerData() []map[string]any {
	return []map[string]any{
		{"Name": "web-server-01", "Status": "✅ Healthy", "CPU": "23%", "Memory": "1.2GB", "Disk": "45%", "Network": "1Gbps", "Uptime": "47d", "Location": "us-east-1a"},
		{"Name": "web-server-02", "Status": "✅ Healthy", "CPU": "31%", "Memory": "1.4GB", "Disk": "42%", "Network": "1Gbps", "Uptime": "47d", "Location": "us-east-1b"},
		{"Name": "app-server-01", "Status": "✅ Healthy", "CPU": "45%", "Memory": "3.2GB", "Disk": "38%", "Network": "1Gbps", "Uptime": "35d", "Location": "us-east-1a"},
		{"Name": "app-server-02", "Status": "✅ Healthy", "CPU": "52%", "Memory": "3.8GB", "Disk": "41%", "Network": "1Gbps", "Uptime": "35d", "Location": "us-east-1b"},
		{"Name": "app-server-03", "Status": "⚠️ Warning", "CPU": "78%", "Memory": "4.1GB", "Disk": "44%", "Network": "1Gbps", "Uptime": "12d", "Location": "us-east-1c"},
		{"Name": "cache-server-01", "Status": "✅ Healthy", "CPU": "15%", "Memory": "8.2GB", "Disk": "25%", "Network": "1Gbps", "Uptime": "72d", "Location": "us-east-1a"},
	}
}

func generateDatabaseData() []map[string]any {
	return []map[string]any{
		{"Instance": "db-primary-01", "Type": "PostgreSQL", "Status": "✅ Primary", "Connections": "145/200", "CPU": "34%", "Memory": "12.8GB", "Storage": "67%", "Replication": "Sync"},
		{"Instance": "db-replica-01", "Type": "PostgreSQL", "Status": "✅ Replica", "Connections": "89/200", "CPU": "18%", "Memory": "11.2GB", "Storage": "67%", "Replication": "45ms"},
		{"Instance": "db-replica-02", "Type": "PostgreSQL", "Status": "🔧 Maintenance", "Connections": "0/200", "CPU": "5%", "Memory": "2.1GB", "Storage": "67%", "Replication": "Offline"},
		{"Instance": "cache-redis-01", "Type": "Redis", "Status": "✅ Active", "Connections": "1,247/10,000", "CPU": "8%", "Memory": "4.7GB", "Storage": "23%", "Replication": "Master"},
	}
}

func generateMetricsData() []map[string]any {
	return []map[string]any{
		{"Metric": "Response Time", "Current": "185ms", "Average": "201ms", "Peak": "312ms", "Threshold": "500ms", "Status": "✅ Good"},
		{"Metric": "Throughput", "Current": "1,847 req/s", "Average": "1,654 req/s", "Peak": "2,103 req/s", "Threshold": "2,500 req/s", "Status": "✅ Good"},
		{"Metric": "Error Rate", "Current": "0.02%", "Average": "0.03%", "Peak": "0.08%", "Threshold": "1.00%", "Status": "✅ Excellent"},
		{"Metric": "CPU Usage", "Current": "42%", "Average": "38%", "Peak": "67%", "Threshold": "80%", "Status": "✅ Good"},
		{"Metric": "Memory Usage", "Current": "67%", "Average": "61%", "Peak": "78%", "Threshold": "85%", "Status": "✅ Good"},
		{"Metric": "Disk I/O", "Current": "234 IOPS", "Average": "198 IOPS", "Peak": "445 IOPS", "Threshold": "1000 IOPS", "Status": "✅ Good"},
	}
}

func generateIncidentData() []map[string]any {
	return []map[string]any{
		{"ID": "INC-2024-0157", "Priority": "🔴 High", "Component": "app-server-03", "Status": "🔍 Investigating", "Duration": "2h 15m", "Assignee": "ops-team"},
		{"ID": "INC-2024-0156", "Priority": "🟡 Medium", "Component": "db-primary-01", "Status": "📊 Monitoring", "Duration": "45m", "Assignee": "db-team"},
	}
}
