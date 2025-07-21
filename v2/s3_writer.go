package output

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"strings"
)

// S3Client defines the interface for S3 operations
// This allows for easy mocking in tests
type S3Client interface {
	PutObject(ctx context.Context, input *S3PutObjectInput) (*S3PutObjectOutput, error)
}

// S3PutObjectInput represents the input for S3 PutObject operation
type S3PutObjectInput struct {
	Bucket      string
	Key         string
	Body        io.Reader
	ContentType string
}

// S3PutObjectOutput represents the output from S3 PutObject operation
type S3PutObjectOutput struct {
	ETag      string
	VersionID string
}

// S3Writer writes rendered output to S3
type S3Writer struct {
	baseWriter
	client       S3Client
	bucket       string
	keyPattern   string            // e.g., "reports/{format}/output.{ext}"
	contentTypes map[string]string // format to content-type mapping
}

// NewS3Writer creates a new S3Writer
func NewS3Writer(client S3Client, bucket, keyPattern string) *S3Writer {
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

	// Create input for S3 PutObject
	input := &S3PutObjectInput{
		Bucket:      sw.bucket,
		Key:         key,
		Body:        bytes.NewReader(data),
		ContentType: contentType,
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
		for format, ct := range contentTypes {
			sw.contentTypes[format] = ct
		}
	}
}

// NewS3WriterWithOptions creates an S3Writer with options
func NewS3WriterWithOptions(client S3Client, bucket, keyPattern string, opts ...S3WriterOption) *S3Writer {
	sw := NewS3Writer(client, bucket, keyPattern)

	for _, opt := range opts {
		opt(sw)
	}

	return sw
}
