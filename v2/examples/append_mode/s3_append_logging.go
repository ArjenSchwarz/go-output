package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	output "github.com/ArjenSchwarz/go-output/v2"
)

// This example demonstrates S3 append mode with ETag-based conflict detection.
//
// Key Points:
// - S3 append uses download-modify-upload pattern (not atomic)
// - ETag-based conditional updates detect concurrent modifications
// - If ETag changes between download and upload, an error is returned
// - Suitable for infrequent updates to small objects (e.g., sporadic logging)
// - MaxAppendSize limit (default 100MB) prevents memory/bandwidth issues
//
// Prerequisites:
// - AWS credentials configured (environment variables, ~/.aws/credentials, or IAM role)
// - S3 bucket with write permissions
// - Optionally enable S3 versioning for data protection
//
// Note: This example requires AWS SDK v2 and valid AWS credentials to run.
func main() {
	ctx := context.Background()

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(cfg)

	// Configure S3 bucket and key
	bucket := "my-logs-bucket"     // Change to your bucket name
	keyPattern := "logs/app.{ext}" // Will become logs/app.json

	fmt.Println("S3 Append Mode Example")
	fmt.Println("======================")
	fmt.Printf("Bucket: %s\n", bucket)
	fmt.Printf("Key pattern: %s\n\n", keyPattern)

	// Create S3Writer with append mode and size limit
	sw := output.NewS3WriterWithOptions(
		s3Client,
		bucket,
		keyPattern,
		output.WithS3AppendMode(),
		output.WithMaxAppendSize(10*1024*1024), // 10MB limit for this example
	)

	// Create output configuration for JSON format (NDJSON logging)
	out := output.NewOutput(
		output.WithFormat(output.FormatJSON),
		output.WithWriter(sw),
	)

	// Simulate multiple log entries over time
	logEntries := []map[string]any{
		{
			"timestamp": time.Now().Format(time.RFC3339),
			"level":     "INFO",
			"service":   "api-gateway",
			"message":   "Service started",
			"version":   "2.1.0",
		},
		{
			"timestamp": time.Now().Add(time.Minute).Format(time.RFC3339),
			"level":     "INFO",
			"service":   "api-gateway",
			"message":   "Health check passed",
			"checks":    5,
		},
		{
			"timestamp":  time.Now().Add(2 * time.Minute).Format(time.RFC3339),
			"level":      "WARNING",
			"service":    "api-gateway",
			"message":    "High latency detected",
			"latency_ms": 850,
		},
		{
			"timestamp": time.Now().Add(3 * time.Minute).Format(time.RFC3339),
			"level":     "ERROR",
			"service":   "api-gateway",
			"message":   "Database connection timeout",
			"retry":     3,
		},
	}

	// Write each log entry separately
	// Each write will download, append, and upload the S3 object
	fmt.Println("Appending log entries to S3...")
	for i, entry := range logEntries {
		doc := output.New().
			Table("log", []map[string]any{entry}).
			Build()

		fmt.Printf("\n[%d/%d] Appending: %s - %s\n", i+1, len(logEntries), entry["level"], entry["message"])

		err := out.Render(ctx, doc)
		if err != nil {
			// Check for specific error types
			fmt.Printf("❌ Failed to append log entry: %v\n", err)

			// In production, you would implement retry logic here
			// For concurrent modification errors, wait and retry
			// For size limit errors, rotate the log file
			if isETagConflict(err) {
				fmt.Println("\n⚠️  ETag Conflict Detected!")
				fmt.Println("This means another process modified the S3 object between download and upload.")
				fmt.Println("In production, implement retry logic with exponential backoff.")
			} else if isSizeLimitError(err) {
				fmt.Println("\n⚠️  Size Limit Exceeded!")
				fmt.Println("The S3 object exceeds the MaxAppendSize limit.")
				fmt.Println("Consider rotating to a new log file or increasing the limit.")
			}

			continue
		}

		fmt.Println("✓ Successfully appended to S3")

		// Add delay between writes to simulate real-world usage
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("\n✅ Success! All log entries appended to S3")
	fmt.Printf("\nView logs: aws s3 cp s3://%s/logs/app.json -\n", bucket)
	fmt.Println("\nIMPORTANT NOTES:")
	fmt.Println("  - S3 append is NOT atomic (download-modify-upload)")
	fmt.Println("  - ETag checks detect conflicts but don't prevent them")
	fmt.Println("  - Enable S3 versioning for data protection")
	fmt.Println("  - Best for infrequent updates to small objects")
	fmt.Println("  - Not suitable for high-frequency concurrent writes")
}

// isETagConflict checks if an error is due to ETag conflict
func isETagConflict(err error) bool {
	if err == nil {
		return false
	}
	// The error message from S3Writer contains "concurrent modification detected"
	return strings.Contains(err.Error(), "concurrent modification")
}

// isSizeLimitError checks if an error is due to size limit exceeded
func isSizeLimitError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "exceeds maximum append size")
}
