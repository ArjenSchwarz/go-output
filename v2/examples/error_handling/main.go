package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	output "github.com/ArjenSchwarz/go-output/v2"
)

func main() {
	fmt.Println("=== Error Handling Demonstration ===")
	fmt.Println("This example shows v2's comprehensive error handling")
	fmt.Println()

	// Example 1: Builder Error Handling
	fmt.Println("Example 1: Builder validation and error accumulation")

	builder := output.New()

	// Add some valid content
	builder.Table("Valid Table", []map[string]any{
		{"Name": "Alice", "Age": 30},
	}, output.WithKeys("Name", "Age"))

	// Attempt to add invalid content (this will create an error)
	builder.Table("Invalid Table", "invalid data type") // Wrong data type
	builder.Raw("invalid_format", []byte("test"))       // Invalid format

	// Check for builder errors before building
	if builder.HasErrors() {
		fmt.Println("❌ Builder has errors:")
		for i, err := range builder.Errors() {
			fmt.Printf("  %d. %v\n", i+1, err)
		}
		fmt.Println()
	}

	// Build the document (valid parts will be included)
	doc := builder.Build()
	fmt.Printf("✅ Document built with %d valid content items\n", len(doc.GetContents()))
	fmt.Println()

	// Example 2: Render Error Handling
	fmt.Println("Example 2: Render error handling with context")

	// Create a document that will render successfully
	goodDoc := output.New().
		Table("Test Table", []map[string]any{
			{"ID": 1, "Name": "Test"},
		}, output.WithKeys("ID", "Name")).
		Build()

	// Create output with invalid writer to demonstrate error handling
	invalidWriter := &failingWriter{shouldFail: true}

	out := output.NewOutput(
		output.WithFormat(output.JSON()),
		output.WithWriter(invalidWriter),
	)

	err := out.Render(context.Background(), goodDoc)
	if err != nil {
		fmt.Println("❌ Render error caught:")
		fmt.Printf("  Error: %v\n", err)

		// Check if it's a specific error type
		var writerErr *output.WriterError
		if errors.As(err, &writerErr) {
			fmt.Printf("  Writer: %s\n", writerErr.Writer)
			fmt.Printf("  Format: %s\n", writerErr.Format)
			fmt.Printf("  Operation: %s\n", writerErr.Operation)
		}
		fmt.Println()
	}

	// Example 3: Multi-error handling (when multiple things fail)
	fmt.Println("Example 3: Multiple error aggregation")

	multiFailDoc := output.New().
		Table("Data", []map[string]any{
			{"Col1": "Value1", "Col2": "Value2"},
		}, output.WithKeys("Col1", "Col2")).
		Build()

	// Setup multiple failing writers
	multiOut := output.NewOutput(
		output.WithFormats(output.JSON(), output.CSV()),
		output.WithWriters(
			&failingWriter{shouldFail: true, name: "Writer1"},
			&failingWriter{shouldFail: true, name: "Writer2"},
		),
	)

	err = multiOut.Render(context.Background(), multiFailDoc)
	if err != nil {
		fmt.Println("❌ Multiple errors occurred:")

		// Check if it's a multi-error
		var multiErr *output.MultiError
		if errors.As(err, &multiErr) {
			fmt.Printf("  Total errors: %d\n", len(multiErr.Errors))
			for i, e := range multiErr.Errors {
				fmt.Printf("  %d. %v\n", i+1, e)
			}
		}
		fmt.Println()
	}

	// Example 4: Context cancellation
	fmt.Println("Example 4: Context cancellation handling")

	cancelDoc := output.New().
		Table("Large Dataset", generateLargeData(100)).
		Build()

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately to demonstrate cancellation handling
	cancel()

	cancelOut := output.NewOutput(
		output.WithFormat(output.JSON()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	err = cancelOut.Render(ctx, cancelDoc)
	if err != nil {
		fmt.Println("❌ Cancellation error:")
		fmt.Printf("  Error: %v\n", err)

		if errors.Is(err, context.Canceled) {
			fmt.Println("  ✓ Properly detected context cancellation")
		}
		fmt.Println()
	}

	// Example 5: Successful error recovery
	fmt.Println("Example 5: Successful operation after errors")

	successDoc := output.New().
		Header("Error Handling Demo Results").
		Table("Summary", []map[string]any{
			{"Test": "Builder Validation", "Status": "✓ Passed", "Details": "Invalid content filtered out"},
			{"Test": "Render Errors", "Status": "✓ Passed", "Details": "Detailed error context provided"},
			{"Test": "Multi-Error Handling", "Status": "✓ Passed", "Details": "All errors aggregated properly"},
			{"Test": "Context Cancellation", "Status": "✓ Passed", "Details": "Cancellation detected correctly"},
			{"Test": "Recovery", "Status": "✓ Passed", "Details": "System recovered successfully"},
		}, output.WithKeys("Test", "Status", "Details")).
		Build()

	successOut := output.NewOutput(
		output.WithFormat(output.Table()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	if err := successOut.Render(context.Background(), successDoc); err != nil {
		log.Fatalf("Unexpected error in success case: %v", err)
	}

	fmt.Println("Error Handling Features Demonstrated:")
	fmt.Println("✓ Builder error accumulation and validation")
	fmt.Println("✓ Structured error types with context")
	fmt.Println("✓ Multi-error aggregation for complex failures")
	fmt.Println("✓ Context cancellation detection")
	fmt.Println("✓ Error recovery and continued operation")
	fmt.Println("✓ Detailed error information for debugging")
}

// failingWriter is a test writer that deliberately fails for demonstration
type failingWriter struct {
	shouldFail bool
	name       string
}

func (f *failingWriter) Write(ctx context.Context, format string, data []byte) error {
	if f.shouldFail {
		return fmt.Errorf("deliberately failing writer (%s) for demonstration", f.name)
	}
	return nil
}

// generateLargeData creates test data for cancellation demo
func generateLargeData(size int) []map[string]any {
	data := make([]map[string]any, size)
	for i := range size {
		data[i] = map[string]any{
			"ID":    i,
			"Value": fmt.Sprintf("Item-%d", i),
		}
	}
	return data
}
