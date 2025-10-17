package output

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3PutObjectAPI defines the minimal interface for S3 PutObject operations.
// This interface uses actual AWS SDK v2 types, making it directly compatible
// with *s3.Client without requiring any adapter or type conversion.
//
// The interface is satisfied by:
//   - github.com/aws/aws-sdk-go-v2/service/s3 (*s3.Client)
//   - Mock implementations for testing (using the same AWS SDK types)
//
// Example usage with AWS SDK v2:
//
//	import (
//	    "github.com/aws/aws-sdk-go-v2/config"
//	    "github.com/aws/aws-sdk-go-v2/service/s3"
//	    "github.com/ArjenSchwarz/go-output/v2"
//	)
//
//	cfg, _ := config.LoadDefaultConfig(context.TODO())
//	s3Client := s3.NewFromConfig(cfg)
//	writer := output.NewS3Writer(s3Client, "my-bucket", "path/to/file.json")
type S3PutObjectAPI interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

// S3Writer writes rendered output to S3
type S3Writer struct {
	baseWriter
	client       S3PutObjectAPI
	bucket       string
	keyPattern   string            // e.g., "reports/{format}/output.{ext}"
	contentTypes map[string]string // format to content-type mapping
}

// NewS3Writer creates a new S3Writer that works with AWS SDK v2 s3.Client.
// The client parameter should be an *s3.Client from github.com/aws/aws-sdk-go-v2/service/s3.
//
// Example:
//
//	import (
//	    "github.com/aws/aws-sdk-go-v2/config"
//	    "github.com/aws/aws-sdk-go-v2/service/s3"
//	    "github.com/ArjenSchwarz/go-output/v2"
//	)
//
//	cfg, _ := config.LoadDefaultConfig(context.TODO())
//	s3Client := s3.NewFromConfig(cfg)
//	writer := output.NewS3Writer(s3Client, "my-bucket", "reports/{format}.{ext}")
func NewS3Writer(client S3PutObjectAPI, bucket, keyPattern string) *S3Writer {
	if keyPattern == "" {
		keyPattern = "output-{format}.{ext}"
	}

	return &S3Writer{
		baseWriter:   baseWriter{name: "s3"},
		client:       client,
		bucket:       bucket,
		keyPattern:   keyPattern,
		contentTypes: defaultContentTypes(),
	}
}

// Write implements the Writer interface
func (sw *S3Writer) Write(ctx context.Context, format string, data []byte) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return sw.wrapError(format, ctx.Err())
	default:
	}

	// Validate input
	if err := sw.validateInput(format, data); err != nil {
		return err
	}

	// Validate S3 configuration
	if sw.client == nil {
		return sw.wrapError(format, fmt.Errorf("S3 client is not configured"))
	}

	if sw.bucket == "" {
		return sw.wrapError(format, fmt.Errorf("S3 bucket is not specified"))
	}

	// Generate S3 key from pattern
	key, err := sw.generateKey(format)
	if err != nil {
		return sw.wrapError(format, err)
	}

	// Determine content type
	contentType := sw.getContentType(format)

	// Create input using actual AWS SDK v2 types
	input := &s3.PutObjectInput{
		Bucket:      &sw.bucket,
		Key:         &key,
		Body:        bytes.NewReader(data),
		ContentType: &contentType,
	}

	// Upload to S3
	output, err := sw.client.PutObject(ctx, input)
	if err != nil {
		return sw.wrapError(format, fmt.Errorf("failed to upload to S3: %w", err))
	}

	// Log success (in production, this might use a proper logger)
	_ = output // Use output to avoid unused variable warning

	return nil
}

// SetContentType sets a custom content type for a format
func (sw *S3Writer) SetContentType(format, contentType string) {
	sw.contentTypes[format] = contentType
}

// generateKey generates an S3 key from the pattern
func (sw *S3Writer) generateKey(format string) (string, error) {
	// Get extension for format
	ext := format // Default to format itself
	if e, ok := defaultExtensions()[format]; ok {
		ext = e
	}

	// Replace placeholders in pattern
	key := sw.keyPattern
	key = strings.ReplaceAll(key, "{format}", format)
	key = strings.ReplaceAll(key, "{ext}", ext)

	// Clean the key (remove any double slashes, etc.)
	key = path.Clean(key)

	// Ensure key doesn't start with /
	key = strings.TrimPrefix(key, "/")

	return key, nil
}

// getContentType returns the content type for a format
func (sw *S3Writer) getContentType(format string) string {
	if ct, ok := sw.contentTypes[format]; ok {
		return ct
	}
	return "application/octet-stream" // Default
}

// defaultContentTypes returns default format to content-type mappings
func defaultContentTypes() map[string]string {
	return map[string]string{
		"json":     "application/json",
		"yaml":     "application/x-yaml",
		"yml":      "application/x-yaml",
		"csv":      "text/csv",
		"html":     "text/html",
		"table":    "text/plain",
		"markdown": "text/markdown",
		"dot":      "text/vnd.graphviz",
		"mermaid":  "text/plain",
		"drawio":   "text/csv",
	}
}

// S3WriterOption configures an S3Writer
type S3WriterOption func(*S3Writer)

// WithContentTypes sets custom content types
func WithContentTypes(contentTypes map[string]string) S3WriterOption {
	return func(sw *S3Writer) {
		maps.Copy(sw.contentTypes, contentTypes)
	}
}

// NewS3WriterWithOptions creates an S3Writer with options
func NewS3WriterWithOptions(client S3PutObjectAPI, bucket, keyPattern string, opts ...S3WriterOption) *S3Writer {
	sw := NewS3Writer(client, bucket, keyPattern)

	for _, opt := range opts {
		opt(sw)
	}

	return sw
}
