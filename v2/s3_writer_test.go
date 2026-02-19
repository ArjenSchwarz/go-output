package output

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// mockS3Client is a mock implementation of S3ClientAPI for testing.
// It uses actual AWS SDK v2 types, ensuring full type compatibility.
type mockS3Client struct {
	putObjectFunc func(ctx context.Context, input *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	getObjectFunc func(ctx context.Context, input *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	calls         []capturedCall
	mu            sync.Mutex
}

// capturedCall stores a single PutObject call for verification
type capturedCall struct {
	Bucket      string
	Key         string
	Body        string // Captured as string for easier comparison
	ContentType string
}

func (m *mockS3Client) PutObject(ctx context.Context, input *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Capture the call details
	call := capturedCall{}
	if input.Bucket != nil {
		call.Bucket = *input.Bucket
	}
	if input.Key != nil {
		call.Key = *input.Key
	}
	if input.ContentType != nil {
		call.ContentType = *input.ContentType
	}
	if input.Body != nil {
		data, _ := io.ReadAll(input.Body)
		call.Body = string(data)
	}

	m.calls = append(m.calls, call)

	if m.putObjectFunc != nil {
		return m.putObjectFunc(ctx, input, optFns...)
	}

	etag := "mock-etag"
	versionId := "mock-version"
	return &s3.PutObjectOutput{
		ETag:      &etag,
		VersionId: &versionId,
	}, nil
}

func (m *mockS3Client) GetObject(ctx context.Context, input *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if m.getObjectFunc != nil {
		return m.getObjectFunc(ctx, input, optFns...)
	}

	// Default behavior: return not found error
	return nil, errors.New("object not found")
}

func (m *mockS3Client) getCalls() []capturedCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]capturedCall{}, m.calls...)
}

func TestNewS3Writer(t *testing.T) {
	client := &mockS3Client{}

	tests := map[string]struct {
		bucket      string
		keyPattern  string
		wantPattern string
	}{"with custom pattern": {

		bucket:      "test-bucket",
		keyPattern:  "reports/{format}/data.{ext}",
		wantPattern: "reports/{format}/data.{ext}",
	}, "with default pattern": {

		bucket:      "test-bucket",
		keyPattern:  "",
		wantPattern: "output-{format}.{ext}",
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			sw := NewS3Writer(client, tt.bucket, tt.keyPattern)

			if sw == nil {
				t.Fatal("NewS3Writer returned nil")
			}

			if sw.bucket != tt.bucket {
				t.Errorf("bucket = %q, want %q", sw.bucket, tt.bucket)
			}

			if sw.keyPattern != tt.wantPattern {
				t.Errorf("keyPattern = %q, want %q", sw.keyPattern, tt.wantPattern)
			}

			if sw.client != client {
				t.Error("client not set correctly")
			}
		})
	}
}

func TestS3WriterWrite(t *testing.T) {
	ctx := context.Background()
	testData := []byte("test content")

	tests := map[string]struct {
		format          string
		data            []byte
		bucket          string
		keyPattern      string
		client          *mockS3Client
		wantErr         bool
		wantKey         string
		wantContentType string
	}{"S3 error": {

		format:     FormatJSON,
		data:       testData,
		bucket:     "test-bucket",
		keyPattern: "data/{format}.{ext}",
		client: &mockS3Client{
			putObjectFunc: func(ctx context.Context, input *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				return nil, errors.New("S3 error")
			},
		},
		wantErr: true,
	}, "empty bucket": {

		format:     FormatJSON,
		data:       testData,
		bucket:     "",
		keyPattern: "data/{format}.{ext}",
		client:     &mockS3Client{},
		wantErr:    true,
	}, "empty data is valid": {

		format:     FormatJSON,
		data:       []byte{},
		bucket:     "test-bucket",
		keyPattern: "empty.{ext}",
		client:     &mockS3Client{},
		wantErr:    false,
		wantKey:    "empty.json",
	}, "empty format": {

		format:     "",
		data:       testData,
		bucket:     "test-bucket",
		keyPattern: "data/{format}.{ext}",
		client:     &mockS3Client{},
		wantErr:    true,
	}, "nil client": {

		format:     FormatJSON,
		data:       testData,
		bucket:     "test-bucket",
		keyPattern: "data/{format}.{ext}",
		client:     nil,
		wantErr:    true,
	}, "nil data": {

		format:     FormatJSON,
		data:       nil,
		bucket:     "test-bucket",
		keyPattern: "data/{format}.{ext}",
		client:     &mockS3Client{},
		wantErr:    true,
	}, "successful write CSV": {

		format:          FormatCSV,
		data:            testData,
		bucket:          "test-bucket",
		keyPattern:      "reports/{format}-output.{ext}",
		client:          &mockS3Client{},
		wantErr:         false,
		wantKey:         "reports/csv-output.csv",
		wantContentType: "text/csv",
	}, "successful write JSON": {

		format:          FormatJSON,
		data:            testData,
		bucket:          "test-bucket",
		keyPattern:      "data/{format}.{ext}",
		client:          &mockS3Client{},
		wantErr:         false,
		wantKey:         "data/json.json",
		wantContentType: "application/json",
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var sw *S3Writer
			if tt.client != nil {
				sw = NewS3Writer(tt.client, tt.bucket, tt.keyPattern)
			} else {
				sw = &S3Writer{
					baseWriter:   baseWriter{name: "s3"},
					bucket:       tt.bucket,
					keyPattern:   tt.keyPattern,
					contentTypes: defaultContentTypes(),
				}
			}

			err := sw.Write(ctx, tt.format, tt.data)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify S3 client was called correctly
			if tt.client != nil {
				calls := tt.client.getCalls()
				if len(calls) != 1 {
					t.Fatalf("expected 1 S3 call, got %d", len(calls))
				}

				call := calls[0]
				if call.Bucket != tt.bucket {
					t.Errorf("S3 bucket = %q, want %q", call.Bucket, tt.bucket)
				}

				if call.Key != tt.wantKey {
					t.Errorf("S3 key = %q, want %q", call.Key, tt.wantKey)
				}

				if tt.wantContentType != "" && call.ContentType != tt.wantContentType {
					t.Errorf("S3 content type = %q, want %q", call.ContentType, tt.wantContentType)
				}

				// Verify body content
				if call.Body != string(tt.data) {
					t.Errorf("S3 body = %q, want %q", call.Body, string(tt.data))
				}
			}
		})
	}
}

func TestS3WriterContextCancellation(t *testing.T) {
	client := &mockS3Client{}
	sw := NewS3Writer(client, "test-bucket", "data.{ext}")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := sw.Write(ctx, FormatJSON, []byte("test"))
	if err == nil {
		t.Error("expected context cancellation error")
	}

	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("error should mention context cancellation, got: %v", err)
	}

	// S3 should not have been called
	calls := client.getCalls()
	if len(calls) > 0 {
		t.Error("S3 should not be called after context cancellation")
	}
}

func TestS3WriterCustomContentTypes(t *testing.T) {
	client := &mockS3Client{}
	customTypes := map[string]string{
		"config": "application/x-config",
		"data":   "application/x-custom-data",
	}

	sw := NewS3WriterWithOptions(client, "test-bucket", "file.{ext}",
		WithContentTypes(customTypes))

	ctx := context.Background()

	tests := []struct {
		format          string
		wantContentType string
	}{
		{"config", "application/x-config"},
		{"data", "application/x-custom-data"},
		{FormatJSON, "application/json"},        // Should still use default
		{"unknown", "application/octet-stream"}, // Default for unknown
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			client.calls = nil // Reset calls

			err := sw.Write(ctx, tt.format, []byte("test"))
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			calls := client.getCalls()
			if len(calls) != 1 {
				t.Fatalf("expected 1 S3 call, got %d", len(calls))
			}

			if calls[0].ContentType != tt.wantContentType {
				t.Errorf("content type = %q, want %q", calls[0].ContentType, tt.wantContentType)
			}
		})
	}
}

func TestS3WriterConcurrency(t *testing.T) {
	var mu sync.Mutex
	var callCount int

	client := &mockS3Client{
		putObjectFunc: func(ctx context.Context, input *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			mu.Lock()
			callCount++
			mu.Unlock()
			return &s3.PutObjectOutput{}, nil
		},
	}

	sw := NewS3Writer(client, "test-bucket", "concurrent/{format}.{ext}")
	ctx := context.Background()

	// Write concurrently
	var wg sync.WaitGroup
	formats := []string{FormatJSON, FormatYAML, FormatCSV, FormatHTML, FormatMarkdown}
	numWrites := 10

	for i := range numWrites {
		for _, format := range formats {
			wg.Add(1)
			go func(f string, n int) {
				defer wg.Done()
				data := []byte(strings.Repeat(f, n+1))
				if err := sw.Write(ctx, f, data); err != nil {
					t.Errorf("concurrent write failed: %v", err)
				}
			}(format, i)
		}
	}

	wg.Wait()

	// Verify all writes completed
	expectedCount := numWrites * len(formats)
	if callCount != expectedCount {
		t.Errorf("call count = %d, want %d", callCount, expectedCount)
	}
}

func TestGenerateKey(t *testing.T) {
	sw := &S3Writer{
		keyPattern: "reports/{format}/output.{ext}",
	}

	tests := []struct {
		format      string
		expectedKey string
	}{
		{FormatJSON, "reports/json/output.json"},
		{FormatCSV, "reports/csv/output.csv"},
		{"unknown", "reports/unknown/output.unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			key, err := sw.generateKey(tt.format)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if key != tt.expectedKey {
				t.Errorf("key = %q, want %q", key, tt.expectedKey)
			}
		})
	}
}

func TestGenerateKeyEdgeCases(t *testing.T) {
	tests := map[string]struct {
		pattern     string
		format      string
		expectedKey string
	}{"double slashes cleaned": {

		pattern:     "data//{format}//{ext}",
		format:      FormatJSON,
		expectedKey: "data/json/json",
	}, "leading slash removed": {

		pattern:     "/data/{format}.{ext}",
		format:      FormatJSON,
		expectedKey: "data/json.json",
	}, "no placeholders": {

		pattern:     "static-file.txt",
		format:      FormatJSON,
		expectedKey: "static-file.txt",
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			sw := &S3Writer{keyPattern: tt.pattern}
			key, err := sw.generateKey(tt.format)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if key != tt.expectedKey {
				t.Errorf("key = %q, want %q", key, tt.expectedKey)
			}
		})
	}
}

func TestS3WriterAppend_CreateNewObject(t *testing.T) {
	if testing.Short() && os.Getenv("INTEGRATION") == "" {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	testData := []byte("first write")

	client := &mockS3Client{
		getObjectFunc: func(ctx context.Context, input *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			// Simulate object not found
			return nil, &types.NoSuchKey{
				Message: aws.String("The specified key does not exist."),
			}
		},
	}

	sw := NewS3WriterWithOptions(client, "test-bucket", "data.{ext}", WithS3AppendMode())

	err := sw.Write(ctx, FormatJSON, testData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify PutObject was called
	calls := client.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 PutObject call, got %d", len(calls))
	}

	if calls[0].Body != string(testData) {
		t.Errorf("body = %q, want %q", calls[0].Body, string(testData))
	}
}

func TestS3WriterAppend_ExistingObject(t *testing.T) {
	if testing.Short() && os.Getenv("INTEGRATION") == "" {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	existingData := []byte("existing data\n")
	newData := []byte("new data\n")

	etag := "test-etag-123"
	contentLength := int64(len(existingData))

	client := &mockS3Client{
		getObjectFunc: func(ctx context.Context, input *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			return &s3.GetObjectOutput{
				Body:          io.NopCloser(bytes.NewReader(existingData)),
				ETag:          &etag,
				ContentLength: &contentLength,
			}, nil
		},
	}

	sw := NewS3WriterWithOptions(client, "test-bucket", "data.{ext}", WithS3AppendMode())

	err := sw.Write(ctx, FormatJSON, newData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify PutObject was called with combined data
	calls := client.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 PutObject call, got %d", len(calls))
	}

	wantBody := string(existingData) + string(newData)
	if calls[0].Body != wantBody {
		t.Errorf("body = %q, want %q", calls[0].Body, wantBody)
	}
}

func TestS3WriterAppend_SizeExceedsLimit(t *testing.T) {
	if testing.Short() && os.Getenv("INTEGRATION") == "" {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	newData := []byte("new data")

	// Object size exceeds limit
	etag := "test-etag"
	contentLength := int64(200 * 1024 * 1024) // 200MB

	client := &mockS3Client{
		getObjectFunc: func(ctx context.Context, input *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			return &s3.GetObjectOutput{
				Body:          io.NopCloser(bytes.NewReader([]byte{})),
				ETag:          &etag,
				ContentLength: &contentLength,
			}, nil
		},
	}

	sw := NewS3WriterWithOptions(client, "test-bucket", "data.{ext}",
		WithS3AppendMode(),
		WithMaxAppendSize(100*1024*1024)) // 100MB limit

	err := sw.Write(ctx, FormatJSON, newData)
	if err == nil {
		t.Fatal("expected error for oversized object, got nil")
	}

	if !strings.Contains(err.Error(), "exceeds maximum append size") {
		t.Errorf("error should mention size limit, got: %v", err)
	}
}

func TestS3WriterAppend_ConcurrentModification(t *testing.T) {
	if testing.Short() && os.Getenv("INTEGRATION") == "" {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	existingData := []byte("existing data")
	newData := []byte("new data")

	etag := "test-etag-123"
	contentLength := int64(len(existingData))

	client := &mockS3Client{
		getObjectFunc: func(ctx context.Context, input *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			return &s3.GetObjectOutput{
				Body:          io.NopCloser(bytes.NewReader(existingData)),
				ETag:          &etag,
				ContentLength: &contentLength,
			}, nil
		},
		putObjectFunc: func(ctx context.Context, input *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			// Simulate ETag mismatch (concurrent modification)
			return nil, errors.New("PreconditionFailed: At least one of the pre-conditions you specified did not hold")
		},
	}

	sw := NewS3WriterWithOptions(client, "test-bucket", "data.{ext}", WithS3AppendMode())

	err := sw.Write(ctx, FormatJSON, newData)
	if err == nil {
		t.Fatal("expected error for concurrent modification, got nil")
	}

	if !strings.Contains(err.Error(), "concurrent modification detected") {
		t.Errorf("error should mention concurrent modification, got: %v", err)
	}
}

func TestS3WriterAppend_HTMLFormat(t *testing.T) {
	if testing.Short() && os.Getenv("INTEGRATION") == "" {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	existingHTML := []byte("<html><body><p>Existing content</p><!-- go-output-append --></body></html>")
	newHTML := []byte("<p>New content</p>")

	etag := "test-etag"
	contentLength := int64(len(existingHTML))

	client := &mockS3Client{
		getObjectFunc: func(ctx context.Context, input *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			return &s3.GetObjectOutput{
				Body:          io.NopCloser(bytes.NewReader(existingHTML)),
				ETag:          &etag,
				ContentLength: &contentLength,
			}, nil
		},
	}

	sw := NewS3WriterWithOptions(client, "test-bucket", "data.{ext}", WithS3AppendMode())

	err := sw.Write(ctx, FormatHTML, newHTML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify marker is preserved and new content inserted before it
	calls := client.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 PutObject call, got %d", len(calls))
	}

	body := calls[0].Body
	if !strings.Contains(body, "Existing content") {
		t.Error("combined HTML should contain existing content")
	}
	if !strings.Contains(body, "New content") {
		t.Error("combined HTML should contain new content")
	}
	if !strings.Contains(body, HTMLAppendMarker) {
		t.Error("combined HTML should contain append marker")
	}

	// Verify marker comes after new content
	markerIdx := strings.Index(body, HTMLAppendMarker)
	newContentIdx := strings.Index(body, "New content")
	if markerIdx < newContentIdx {
		t.Error("marker should come after new content")
	}
}

func TestS3WriterAppend_CSVFormat(t *testing.T) {
	if testing.Short() && os.Getenv("INTEGRATION") == "" {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	existingCSV := []byte("Name,Age\nAlice,30\n")
	newCSV := []byte("Name,Age\nBob,25\n")

	etag := "test-etag"
	contentLength := int64(len(existingCSV))

	client := &mockS3Client{
		getObjectFunc: func(ctx context.Context, input *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			return &s3.GetObjectOutput{
				Body:          io.NopCloser(bytes.NewReader(existingCSV)),
				ETag:          &etag,
				ContentLength: &contentLength,
			}, nil
		},
	}

	sw := NewS3WriterWithOptions(client, "test-bucket", "data.{ext}", WithS3AppendMode())

	err := sw.Write(ctx, FormatCSV, newCSV)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify header was stripped from new data
	calls := client.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 PutObject call, got %d", len(calls))
	}

	wantBody := "Name,Age\nAlice,30\nBob,25\n"
	if calls[0].Body != wantBody {
		t.Errorf("body = %q, want %q", calls[0].Body, wantBody)
	}
}

func TestS3WriterAppendModeConfiguration(t *testing.T) {
	client := &mockS3Client{}

	tests := map[string]struct {
		options           []S3WriterOption
		wantAppendMode    bool
		wantMaxAppendSize int64
	}{
		"default configuration has append disabled": {
			options:           nil,
			wantAppendMode:    false,
			wantMaxAppendSize: 104857600, // 100MB default
		},
		"WithS3AppendMode enables append": {
			options:           []S3WriterOption{WithS3AppendMode()},
			wantAppendMode:    true,
			wantMaxAppendSize: 104857600,
		},
		"WithMaxAppendSize sets custom size": {
			options:           []S3WriterOption{WithMaxAppendSize(50 * 1024 * 1024)},
			wantAppendMode:    false,
			wantMaxAppendSize: 50 * 1024 * 1024,
		},
		"combined options": {
			options: []S3WriterOption{
				WithS3AppendMode(),
				WithMaxAppendSize(10 * 1024 * 1024),
			},
			wantAppendMode:    true,
			wantMaxAppendSize: 10 * 1024 * 1024,
		},
		"WithMaxAppendSize with zero is ignored": {
			options:           []S3WriterOption{WithMaxAppendSize(0)},
			wantAppendMode:    false,
			wantMaxAppendSize: 104857600, // Should remain default
		},
		"WithMaxAppendSize with negative is ignored": {
			options:           []S3WriterOption{WithMaxAppendSize(-100)},
			wantAppendMode:    false,
			wantMaxAppendSize: 104857600, // Should remain default
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			sw := NewS3WriterWithOptions(client, "test-bucket", "data.{ext}", tc.options...)

			if sw.appendMode != tc.wantAppendMode {
				t.Errorf("appendMode = %v, want %v", sw.appendMode, tc.wantAppendMode)
			}

			if sw.maxAppendSize != tc.wantMaxAppendSize {
				t.Errorf("maxAppendSize = %d, want %d", sw.maxAppendSize, tc.wantMaxAppendSize)
			}
		})
	}
}

// TestS3Writer_CSVAppendNewlineHandling tests the regression for T-80
// Ensures CSV append properly handles existing data without trailing newlines
func TestS3Writer_CSVAppendNewlineHandling(t *testing.T) {
	tests := map[string]struct {
		existingCSV string
		newCSV      string
		wantResult  string
	}{
		"existing CSV without trailing newline": {
			existingCSV: "name,age\nAlice,30\nBob,25",
			newCSV:      "name,age\nCharlie,35\nDiana,28",
			wantResult:  "name,age\nAlice,30\nBob,25\nCharlie,35\nDiana,28",
		},
		"existing CSV with trailing newline": {
			existingCSV: "name,age\nAlice,30\nBob,25\n",
			newCSV:      "name,age\nCharlie,35\nDiana,28",
			wantResult:  "name,age\nAlice,30\nBob,25\nCharlie,35\nDiana,28",
		},
		"empty existing CSV": {
			existingCSV: "",
			newCSV:      "name,age\nCharlie,35",
			wantResult:  "Charlie,35",
		},
		"new CSV with only header": {
			existingCSV: "name,age\nAlice,30",
			newCSV:      "name,age",
			wantResult:  "name,age\nAlice,30",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var capturedBody string

			client := &mockS3ClientWithAppend{
				getObjectFunc: func(ctx context.Context, input *s3.GetObjectInput, opts ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					size := int64(len(tc.existingCSV))
					return &s3.GetObjectOutput{
						Body:          io.NopCloser(strings.NewReader(tc.existingCSV)),
						ContentLength: &size,
						ETag:          aws.String("etag123"),
					}, nil
				},
				putObjectFunc: func(ctx context.Context, input *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
					// Capture the body for verification
					if input.Body != nil {
						bodyBytes, err := io.ReadAll(input.Body)
						if err != nil {
							t.Fatalf("failed to read body: %v", err)
						}
						capturedBody = string(bodyBytes)
					}
					return &s3.PutObjectOutput{}, nil
				},
			}

			sw := NewS3WriterWithOptions(
				client,
				"test-bucket",
				"test.csv",
				WithS3AppendMode(),
			)

			// Perform the append operation
			err := sw.Write(context.Background(), FormatCSV, []byte(tc.newCSV))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify the result
			if capturedBody != tc.wantResult {
				t.Errorf("CSV append result mismatch:\nwant: %q\ngot:  %q", tc.wantResult, capturedBody)
			}
		})
	}
}
