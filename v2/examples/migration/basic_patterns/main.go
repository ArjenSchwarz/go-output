package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	output "github.com/ArjenSchwarz/go-output/v2"
)

func main() {
	fmt.Println("=== Basic Migration Patterns (v1 → v2) ===")
	fmt.Println("This example shows how to convert common v1 patterns to v2")
	fmt.Println()

	// Sample data used in all examples
	userData := []map[string]any{
		{"ID": 1, "Name": "Alice Johnson", "Email": "alice@company.com", "Department": "Engineering", "Status": "Active"},
		{"ID": 2, "Name": "Bob Smith", "Email": "bob@company.com", "Department": "Marketing", "Status": "Active"},
		{"ID": 3, "Name": "Carol Davis", "Email": "carol@company.com", "Department": "Engineering", "Status": "Inactive"},
	}

	// Pattern 1: Simple Table Output
	fmt.Println("Pattern 1: Simple Table Output")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	fmt.Println("\nv1 Code:")
	fmt.Println(`// v1 - Global state, unpredictable key ordering
output := &format.OutputArray{}
output.AddContents(userData)
output.Write()`)

	fmt.Println("\nv2 Equivalent:")
	
	// v2 Implementation
	doc1 := output.New().
		Table("", userData). // Auto-detects schema but order may vary
		Build()

	out1 := output.NewOutput(
		output.WithFormat(output.Table),
		output.WithWriter(output.NewStdoutWriter()),
	)

	if err := out1.Render(context.Background(), doc1); err != nil {
		log.Fatalf("Pattern 1 failed: %v", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 60) + "\n")

	// Pattern 2: Key Ordering (Most Important Migration)
	fmt.Println("Pattern 2: Key Ordering Preservation")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	fmt.Println("\nv1 Code:")
	fmt.Println(`// v1 - Keys field for column ordering (but unreliable)
output := &format.OutputArray{
    Keys: []string{"Name", "Department", "Email", "Status"},
}
output.AddContents(userData)
output.Write()`)

	fmt.Println("\nv2 Equivalent (Exact Key Order Preserved):")
	
	// v2 Implementation - exact key preservation
	doc2 := output.New().
		Table("", userData, output.WithKeys("Name", "Department", "Email", "Status")).
		Build()

	out2 := output.NewOutput(
		output.WithFormat(output.Table),
		output.WithWriter(output.NewStdoutWriter()),
	)

	if err := out2.Render(context.Background(), doc2); err != nil {
		log.Fatalf("Pattern 2 failed: %v", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 60) + "\n")

	// Pattern 3: Multiple Tables (Buffer Pattern)
	fmt.Println("Pattern 3: Multiple Tables")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	fmt.Println("\nv1 Code:")
	fmt.Println(`// v1 - Required buffer pattern for multiple tables
output := &format.OutputArray{
    Keys: []string{"Name", "Email"},
}
output.AddContents(activeUsers)
output.AddToBuffer()

output.Keys = []string{"ID", "Department", "Status"} // Different keys!
output.AddContents(summaryData)
output.Write()`)

	// Simulate the different data sets
	activeUsers := []map[string]any{
		{"Name": "Alice Johnson", "Email": "alice@company.com"},
		{"Name": "Bob Smith", "Email": "bob@company.com"},
	}

	summaryData := []map[string]any{
		{"ID": 1, "Department": "Engineering", "Status": "2 active"},
		{"ID": 2, "Department": "Marketing", "Status": "1 active"},
	}

	fmt.Println("\nv2 Equivalent (Clean Multiple Tables):")
	
	// v2 Implementation - multiple tables with different schemas
	doc3 := output.New().
		Header("User Report").
		Table("Active Users", activeUsers, 
			output.WithKeys("Name", "Email")).
		Table("Department Summary", summaryData, 
			output.WithKeys("ID", "Department", "Status")). // Different keys!
		Build()

	out3 := output.NewOutput(
		output.WithFormat(output.Table),
		output.WithWriter(output.NewStdoutWriter()),
	)

	if err := out3.Render(context.Background(), doc3); err != nil {
		log.Fatalf("Pattern 3 failed: %v", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 60) + "\n")

	// Pattern 4: File Output
	fmt.Println("Pattern 4: File Output")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━")
	
	fmt.Println("\nv1 Code:")
	fmt.Println(`// v1 - Settings-based file output
settings := format.NewOutputSettings()
settings.OutputFile = "report.html"
settings.OutputFileFormat = "html"

output := &format.OutputArray{Settings: settings}
output.AddContents(userData)
output.Write()`)

	fmt.Println("\nv2 Equivalent:")

	// v2 Implementation - FileWriter pattern
	doc4 := output.New().
		Table("User Export", userData, 
			output.WithKeys("Name", "Department", "Status")).
		Build()

	fileWriter, err := output.NewFileWriter("./output", "report.{format}")
	if err != nil {
		log.Fatalf("Failed to create file writer: %v", err)
	}

	out4 := output.NewOutput(
		output.WithFormat(output.HTML),
		output.WithWriter(fileWriter),
	)

	if err := out4.Render(context.Background(), doc4); err != nil {
		log.Fatalf("Pattern 4 failed: %v", err)
	}

	fmt.Println("✅ File written to: ./output/report.html")
	fmt.Println("\n" + strings.Repeat("=", 60) + "\n")

	// Summary
	fmt.Println("Migration Benefits Demonstrated:")
	fmt.Println("✓ Exact key order preservation (major improvement over v1)")
	fmt.Println("✓ Clean multiple table handling (no buffer pattern needed)")
	fmt.Println("✓ Thread-safe operations (no global state)")
	fmt.Println("✓ Immutable documents (safe for concurrent use)")
	fmt.Println("✓ Explicit configuration (no hidden settings)")
	fmt.Println("✓ Better error handling with context")

	fmt.Println("\nKey Differences from v1:")
	fmt.Println("• Document-Builder pattern instead of global OutputArray")
	fmt.Println("• WithKeys() option instead of Keys field")
	fmt.Println("• Functional options instead of OutputSettings")
	fmt.Println("• Build() + Render() instead of Write()")
	fmt.Println("• Context required for cancellation support")
	fmt.Println("• Writers instead of file settings")
}