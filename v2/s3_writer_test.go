package output

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
)

// mockS3Client is a mock implementation of S3Client for testing
type mockS3Client struct {
	putObjectFunc func(ctx context.Context, input *S3PutObjectInput) (*S3PutObjectOutput, error)
	calls         []S3PutObjectInput
	mu            sync.Mutex
}

func (m *mockS3Client) PutObject(ctx context.Context, input *S3PutObjectInput) (*S3PutObjectOutput, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Read the body to capture it
	if input.Body != nil {
		data, _ := io.ReadAll(input.Body)
		input.Body = strings.NewReader(string(data))
	}

	m.calls = append(m.calls, *input)

	if m.putObjectFunc != nil {
		return m.putObjectFunc(ctx, input)
	}

	return &S3PutObjectOutput{
		ETag:      "mock-etag",
		VersionID: "mock-version",
	}, nil
}

func (m *mockS3Client) getCalls() []S3PutObjectInput {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]S3PutObjectInput{}, m.calls...)
}

func TestNewS3Writer(t *testing.T) {
	client := &mockS3Client{}

	tests := []struct {
		name        string
		bucket      string
		keyPattern  string
		wantPattern string
	}{
		{
			name:        "with custom pattern",
			bucket:      "test-bucket",
			keyPattern:  "reports/{format}/data.{ext}",
			wantPattern: "reports/{format}/data.{ext}",
		},
		{
			name:        "with default pattern",
			bucket:      "test-bucket",
			keyPattern:  "",
			wantPattern: "output-{format}.{ext}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

	tests := []struct {
		name            string
		format          string
		data            []byte
		bucket          string
		keyPattern      string
		client          *mockS3Client
		wantErr         bool
		wantKey         string
		wantContentType string
	}{
		{
			name:            "successful write JSON",
			format:          FormatJSON,
			data:            testData,
			bucket:          "test-bucket",
			keyPattern:      "data/{format}.{ext}",
			client:          &mockS3Client{},
			wantErr:         false,
			wantKey:         "data/json.json",
			wantContentType: "application/json",
		},
		{
			name:            "successful write CSV",
			format:          FormatCSV,
			data:            testData,
			bucket:          "test-bucket",
			keyPattern:      "reports/{format}-output.{ext}",
			client:          &mockS3Client{},
			wantErr:         false,
			wantKey:         "reports/csv-output.csv",
			wantContentType: "text/csv",
		},
		{
			name:       "empty format",
			format:     "",
			data:       testData,
			bucket:     "test-bucket",
			keyPattern: "data/{format}.{ext}",
			client:     &mockS3Client{},
			wantErr:    true,
		},
		{
			name:       "nil data",
			format:     FormatJSON,
			data:       nil,
			bucket:     "test-bucket",
			keyPattern: "data/{format}.{ext}",
			client:     &mockS3Client{},
			wantErr:    true,
		},
		{
			name:       "empty bucket",
			format:     FormatJSON,
			data:       testData,
			bucket:     "",
			keyPattern: "data/{format}.{ext}",
			client:     &mockS3Client{},
			wantErr:    true,
		},
		{
			name:       "nil client",
			format:     FormatJSON,
			data:       testData,
			bucket:     "test-bucket",
			keyPattern: "data/{format}.{ext}",
			client:     nil,
			wantErr:    true,
		},
		{
			name:       "S3 error",
			format:     FormatJSON,
			data:       testData,
			bucket:     "test-bucket",
			keyPattern: "data/{format}.{ext}",
			client: &mockS3Client{
				putObjectFunc: func(ctx context.Context, input *S3PutObjectInput) (*S3PutObjectOutput, error) {
					return nil, errors.New("S3 error")
				},
			},
			wantErr: true,
		},
		{
			name:       "empty data is valid",
			format:     FormatJSON,
			data:       []byte{},
			bucket:     "test-bucket",
			keyPattern: "empty.{ext}",
			client:     &mockS3Client{},
			wantErr:    false,
			wantKey:    "empty.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
				if call.Body != nil {
					bodyData, _ := io.ReadAll(call.Body)
					if string(bodyData) != string(tt.data) {
						t.Errorf("S3 body = %q, want %q", bodyData, tt.data)
					}
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
		putObjectFunc: func(ctx context.Context, input *S3PutObjectInput) (*S3PutObjectOutput, error) {
			mu.Lock()
			callCount++
			mu.Unlock()
			return &S3PutObjectOutput{}, nil
		},
	}

	sw := NewS3Writer(client, "test-bucket", "concurrent/{format}.{ext}")
	ctx := context.Background()

	// Write concurrently
	var wg sync.WaitGroup
	formats := []string{FormatJSON, FormatYAML, FormatCSV, FormatHTML, FormatMarkdown}
	numWrites := 10

	for i := 0; i < numWrites; i++ {
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
	tests := []struct {
		name        string
		pattern     string
		format      string
		expectedKey string
	}{
		{
			name:        "leading slash removed",
			pattern:     "/data/{format}.{ext}",
			format:      FormatJSON,
			expectedKey: "data/json.json",
		},
		{
			name:        "double slashes cleaned",
			pattern:     "data//{format}//{ext}",
			format:      FormatJSON,
			expectedKey: "data/json/json",
		},
		{
			name:        "no placeholders",
			pattern:     "static-file.txt",
			format:      FormatJSON,
			expectedKey: "static-file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
