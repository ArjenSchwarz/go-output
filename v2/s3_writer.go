package output

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
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

// S3GetObjectAPI defines the minimal interface for S3 GetObject operations.
// This interface is used for append mode support, allowing the S3Writer to
// download existing objects before modifying and re-uploading them.
//
// The interface is satisfied by:
//   - github.com/aws/aws-sdk-go-v2/service/s3 (*s3.Client)
//   - Mock implementations for testing (using the same AWS SDK types)
type S3GetObjectAPI interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

// S3ClientAPI combines Get and Put operations for full S3Writer functionality.
// When a client implements both interfaces, the S3Writer can support append mode
// using the download-modify-upload pattern.
//
// The interface is satisfied by:
//   - github.com/aws/aws-sdk-go-v2/service/s3 (*s3.Client) - implements both operations
//   - Clients implementing only S3PutObjectAPI can still be used but without append mode
type S3ClientAPI interface {
	S3PutObjectAPI
	S3GetObjectAPI
}

// S3Writer writes rendered output to S3
type S3Writer struct {
	baseWriter
	client        S3PutObjectAPI
	bucket        string
	keyPattern    string            // e.g., "reports/{format}/output.{ext}"
	contentTypes  map[string]string // format to content-type mapping
	appendMode    bool              // enable append mode (download-modify-upload pattern)
	maxAppendSize int64             // maximum object size for append operations (default 100MB)
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
		baseWriter:    baseWriter{name: "s3"},
		client:        client,
		bucket:        bucket,
		keyPattern:    keyPattern,
		contentTypes:  defaultContentTypes(),
		appendMode:    false,
		maxAppendSize: 104857600, // 100MB default
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

	// Handle append mode
	if sw.appendMode {
		return sw.appendToS3Object(ctx, format, key, data)
	}

	// Normal write (create/truncate)
	return sw.putS3Object(ctx, format, key, data)
}

// putS3Object performs a normal (non-append) write to S3
func (sw *S3Writer) putS3Object(ctx context.Context, format, key string, data []byte) error {
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

// appendToS3Object implements append mode using download-modify-upload pattern
func (sw *S3Writer) appendToS3Object(ctx context.Context, format, key string, newData []byte) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return sw.wrapError(format, ctx.Err())
	default:
	}

	// Check if client supports GetObject (required for append mode)
	getClient, ok := sw.client.(S3GetObjectAPI)
	if !ok {
		return sw.wrapError(format, fmt.Errorf("S3 client does not support GetObject (required for append mode)"))
	}

	// Validate new data size before making any API calls
	if int64(len(newData)) > sw.maxAppendSize {
		return sw.wrapError(format, fmt.Errorf("new data size %d exceeds maximum append size %d",
			len(newData), sw.maxAppendSize))
	}

	// Attempt to get existing object
	// Using GetObject alone (no HeadObject) reduces API calls from 2 to 1
	getInput := &s3.GetObjectInput{
		Bucket: &sw.bucket,
		Key:    &key,
	}

	getOutput, err := getClient.GetObject(ctx, getInput)
	if err != nil {
		// Check if object doesn't exist (NoSuchKey error)
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			// Object doesn't exist - create new
			return sw.putS3Object(ctx, format, key, newData)
		}
		return sw.wrapError(format, fmt.Errorf("failed to download object for append: %w", err))
	}
	defer getOutput.Body.Close()

	// Validate existing object size
	if getOutput.ContentLength != nil && *getOutput.ContentLength > sw.maxAppendSize {
		return sw.wrapError(format, fmt.Errorf("object size %d exceeds maximum append size %d",
			*getOutput.ContentLength, sw.maxAppendSize))
	}

	// Validate combined size
	if getOutput.ContentLength != nil {
		combinedSize := *getOutput.ContentLength + int64(len(newData))
		if combinedSize > sw.maxAppendSize {
			return sw.wrapError(format, fmt.Errorf("combined size %d would exceed maximum append size %d",
				combinedSize, sw.maxAppendSize))
		}
	}

	// Read existing content
	existingData, err := io.ReadAll(getOutput.Body)
	if err != nil {
		return sw.wrapError(format, fmt.Errorf("failed to read object content: %w", err))
	}

	// Combine data based on format
	combinedData, err := sw.combineData(format, existingData, newData)
	if err != nil {
		return sw.wrapError(format, err)
	}

	// Upload with conditional put (ETag check for optimistic locking)
	contentType := sw.getContentType(format)
	putInput := &s3.PutObjectInput{
		Bucket:      &sw.bucket,
		Key:         &key,
		Body:        bytes.NewReader(combinedData),
		ContentType: &contentType,
		IfMatch:     getOutput.ETag, // Fail if ETag changed (detects concurrent modification)
	}

	_, err = sw.client.PutObject(ctx, putInput)
	if err != nil {
		// Check for precondition failed (ETag mismatch)
		// S3 returns 412 status code when IfMatch fails
		// The error message typically contains "PreconditionFailed" or "At least one of the pre-conditions you specified did not hold"
		errMsg := err.Error()
		if strings.Contains(errMsg, "PreconditionFailed") || strings.Contains(errMsg, "pre-condition") || strings.Contains(errMsg, "412") {
			return sw.wrapError(format, fmt.Errorf("concurrent modification detected - retry the operation"))
		}
		return sw.wrapError(format, fmt.Errorf("failed to upload combined object: %w", err))
	}

	return nil
}

// combineData combines existing and new data based on format
func (sw *S3Writer) combineData(format string, existing, new []byte) ([]byte, error) {
	switch format {
	case FormatHTML:
		return sw.combineHTMLData(existing, new)
	case FormatCSV:
		return sw.combineCSVData(existing, new)
	default:
		// Byte-level append
		return append(existing, new...), nil
	}
}

// combineHTMLData combines HTML content by inserting new data before the append marker
func (sw *S3Writer) combineHTMLData(existing, new []byte) ([]byte, error) {
	// Find marker using bytes.Index
	markerIndex := bytes.Index(existing, []byte(HTMLAppendMarker))
	if markerIndex == -1 {
		return nil, fmt.Errorf("HTML append marker not found in existing object")
	}

	// Build combined content: [existing before marker] + [new data] + [marker] + [remainder]
	var buf bytes.Buffer
	buf.Write(existing[:markerIndex])
	buf.Write(new)
	buf.WriteString(HTMLAppendMarker)
	buf.Write(existing[markerIndex+len(HTMLAppendMarker):])

	return buf.Bytes(), nil
}

// combineCSVData combines CSV content by stripping headers from new data
func (sw *S3Writer) combineCSVData(existing, new []byte) ([]byte, error) {
	// Normalize line endings (handle both LF and CRLF)
	new = bytes.ReplaceAll(new, []byte("\r\n"), []byte("\n"))

	// Strip first line (header) from new data
	lines := bytes.SplitN(new, []byte("\n"), 2)
	if len(lines) < 2 {
		// Only one line (or empty) - nothing to append after removing header
		return existing, nil
	}

	dataWithoutHeader := lines[1]

	// Ensure existing data ends with newline before appending
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		return append(append(existing, '\n'), dataWithoutHeader...), nil
	}

	return append(existing, dataWithoutHeader...), nil
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

// WithS3AppendMode enables append mode for the S3Writer.
// When enabled, the S3Writer uses a download-modify-upload pattern to append
// new content to existing S3 objects. This is designed for infrequent updates
// to small objects (e.g., sporadic NDJSON logging).
//
// IMPORTANT: The append operation is NOT atomic. The S3Writer uses ETag-based
// conditional updates to detect concurrent modifications, but there is no locking
// mechanism. If a concurrent modification is detected, an error is returned.
//
// Recommendation: Enable S3 versioning for data protection when using append mode.
func WithS3AppendMode() S3WriterOption {
	return func(sw *S3Writer) {
		sw.appendMode = true
	}
}

// WithMaxAppendSize sets the maximum object size for append operations.
// The S3Writer will return an error if attempting to append to an object
// that exceeds this size limit. This prevents memory exhaustion and excessive
// bandwidth usage during the download-modify-upload process.
//
// The default maximum size is 100MB (104857600 bytes).
// The maxSize parameter must be greater than 0.
func WithMaxAppendSize(maxSize int64) S3WriterOption {
	return func(sw *S3Writer) {
		if maxSize > 0 {
			sw.maxAppendSize = maxSize
		}
	}
}

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
