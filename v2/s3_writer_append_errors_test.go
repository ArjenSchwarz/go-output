package output

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// TestS3Writer_AppendErrorHandling tests error scenarios for S3 append operations
func TestS3Writer_AppendErrorHandling(t *testing.T) {

	tests := map[string]struct {
		setupClient  func() S3ClientAPI
		format       string
		data         []byte
		wantErrMsg   string
		checkErrType func(t *testing.T, err error)
	}{
		"S3 size limit error includes current and max size": {
			setupClient: func() S3ClientAPI {
				return &mockS3ClientWithAppend{
					getObjectFunc: func(ctx context.Context, input *s3.GetObjectInput, opts ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
						// Return object that exceeds size limit (default 100MB)
						size := int64(150 * 1024 * 1024) // 150MB
						return &s3.GetObjectOutput{
							Body:          io.NopCloser(bytes.NewReader([]byte("existing data"))),
							ContentLength: &size,
							ETag:          aws.String("etag123"),
						}, nil
					},
					putObjectFunc: func(ctx context.Context, input *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
						return &s3.PutObjectOutput{}, nil
					},
				}
			},
			format:     FormatJSON,
			data:       []byte(`{"key":"value"}`),
			wantErrMsg: "exceeds maximum append size",
			checkErrType: func(t *testing.T, err error) {
				t.Helper()
				var we *WriteError
				if !errors.As(err, &we) {
					t.Errorf("expected WriteError, got %T", err)
				}
				// Check that error includes both current and max sizes
				if !strings.Contains(err.Error(), "157286400") { // 150MB in bytes
					t.Errorf("error should include current size (150MB), got: %v", err)
				}
				if !strings.Contains(err.Error(), "104857600") { // 100MB in bytes (default max)
					t.Errorf("error should include max size (100MB), got: %v", err)
				}
			},
		},
		"S3 ETag mismatch error message clarity": {
			setupClient: func() S3ClientAPI {
				return &mockS3ClientWithAppend{
					getObjectFunc: func(ctx context.Context, input *s3.GetObjectInput, opts ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
						size := int64(100)
						return &s3.GetObjectOutput{
							Body:          io.NopCloser(bytes.NewReader([]byte("existing"))),
							ContentLength: &size,
							ETag:          aws.String("etag123"),
						}, nil
					},
					putObjectFunc: func(ctx context.Context, input *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
						// Simulate ETag mismatch by returning precondition failed error
						return nil, &mockPreconditionFailedError{
							message: "At least one of the pre-conditions you specified did not hold",
						}
					},
				}
			},
			format:     FormatJSON,
			data:       []byte(`{"key":"value"}`),
			wantErrMsg: "concurrent modification detected",
			checkErrType: func(t *testing.T, err error) {
				t.Helper()
				var we *WriteError
				if !errors.As(err, &we) {
					t.Errorf("expected WriteError, got %T", err)
				}
				// Check that error suggests retry
				if !strings.Contains(err.Error(), "retry") {
					t.Errorf("error should suggest retry, got: %v", err)
				}
			},
		},
		"verify all S3 errors wrapped with WriteError": {
			setupClient: func() S3ClientAPI {
				return &mockS3ClientWithAppend{
					getObjectFunc: func(ctx context.Context, input *s3.GetObjectInput, opts ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
						return nil, errors.New("network error")
					},
					putObjectFunc: func(ctx context.Context, input *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
						return &s3.PutObjectOutput{}, nil
					},
				}
			},
			format:     FormatJSON,
			data:       []byte(`{"key":"value"}`),
			wantErrMsg: "failed to download object",
			checkErrType: func(t *testing.T, err error) {
				t.Helper()
				var we *WriteError
				if !errors.As(err, &we) {
					t.Errorf("expected WriteError, got %T", err)
				}
				if we.Writer != "s3" {
					t.Errorf("expected Writer='s3', got %q", we.Writer)
				}
				if we.Format != FormatJSON {
					t.Errorf("expected Format='json', got %q", we.Format)
				}
			},
		},
		"HTML marker missing in S3 object": {
			setupClient: func() S3ClientAPI {
				return &mockS3ClientWithAppend{
					getObjectFunc: func(ctx context.Context, input *s3.GetObjectInput, opts ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
						// Return HTML without marker
						html := []byte(`<html><body><h1>Test</h1></body></html>`)
						size := int64(len(html))
						return &s3.GetObjectOutput{
							Body:          io.NopCloser(bytes.NewReader(html)),
							ContentLength: &size,
							ETag:          aws.String("etag123"),
						}, nil
					},
					putObjectFunc: func(ctx context.Context, input *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
						return &s3.PutObjectOutput{}, nil
					},
				}
			},
			format:     FormatHTML,
			data:       []byte(`<p>New content</p>`),
			wantErrMsg: "HTML append marker not found",
			checkErrType: func(t *testing.T, err error) {
				t.Helper()
				var we *WriteError
				if !errors.As(err, &we) {
					t.Errorf("expected WriteError, got %T", err)
				}
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			client := tc.setupClient()

			// Create S3Writer with append mode
			sw := NewS3WriterWithOptions(
				client,
				"test-bucket",
				"test-key",
				WithS3AppendMode(),
			)

			// Attempt to append
			err := sw.Write(context.Background(), tc.format, tc.data)

			// Verify error occurred
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			// Check error message
			if !strings.Contains(err.Error(), tc.wantErrMsg) {
				t.Errorf("expected error to contain %q, got: %v", tc.wantErrMsg, err)
			}

			// Run additional error type checks
			if tc.checkErrType != nil {
				tc.checkErrType(t, err)
			}
		})
	}
}

// TestS3Writer_AppendSizeLimitErrorDetails verifies size limit error includes details
func TestS3Writer_AppendSizeLimitErrorDetails(t *testing.T) {

	tests := map[string]struct {
		objectSize     int64
		maxAppendSize  int64
		wantCurrentStr string
		wantMaxStr     string
	}{
		"default 100MB limit exceeded": {
			objectSize:     150 * 1024 * 1024, // 150MB
			maxAppendSize:  100 * 1024 * 1024, // 100MB (default)
			wantCurrentStr: "157286400",       // 150MB in bytes
			wantMaxStr:     "104857600",       // 100MB in bytes
		},
		"custom 50MB limit exceeded": {
			objectSize:     75 * 1024 * 1024, // 75MB
			maxAppendSize:  50 * 1024 * 1024, // 50MB
			wantCurrentStr: "78643200",       // 75MB in bytes
			wantMaxStr:     "52428800",       // 50MB in bytes
		},
		"custom 10MB limit exceeded": {
			objectSize:     15 * 1024 * 1024, // 15MB
			maxAppendSize:  10 * 1024 * 1024, // 10MB
			wantCurrentStr: "15728640",       // 15MB in bytes
			wantMaxStr:     "10485760",       // 10MB in bytes
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			client := &mockS3ClientWithAppend{
				getObjectFunc: func(ctx context.Context, input *s3.GetObjectInput, opts ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					return &s3.GetObjectOutput{
						Body:          io.NopCloser(bytes.NewReader([]byte("existing data"))),
						ContentLength: &tc.objectSize,
						ETag:          aws.String("etag123"),
					}, nil
				},
				putObjectFunc: func(ctx context.Context, input *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
					return &s3.PutObjectOutput{}, nil
				},
			}

			// Create S3Writer with append mode and custom size limit
			sw := NewS3WriterWithOptions(
				client,
				"test-bucket",
				"test-key",
				WithS3AppendMode(),
				WithMaxAppendSize(tc.maxAppendSize),
			)

			// Attempt to append
			err := sw.Write(context.Background(), FormatJSON, []byte(`{"key":"value"}`))

			// Verify error
			if err == nil {
				t.Fatal("expected size limit error, got nil")
			}

			errMsg := err.Error()

			// Verify error includes current size
			if !strings.Contains(errMsg, tc.wantCurrentStr) {
				t.Errorf("error should include current size %s, got: %v", tc.wantCurrentStr, errMsg)
			}

			// Verify error includes max size
			if !strings.Contains(errMsg, tc.wantMaxStr) {
				t.Errorf("error should include max size %s, got: %v", tc.wantMaxStr, errMsg)
			}
		})
	}
}

// TestS3Writer_AppendETagConflictSuggestsRetry verifies ETag conflict error suggests retry
func TestS3Writer_AppendETagConflictSuggestsRetry(t *testing.T) {

	tests := map[string]struct {
		putError    error
		wantRetry   bool
		wantMessage string
	}{
		"412 status code suggests retry": {
			putError: &mockPreconditionFailedError{
				message: "Status Code: 412, Request ID: abc123",
			},
			wantRetry:   true,
			wantMessage: "concurrent modification detected - retry the operation",
		},
		"PreconditionFailed error suggests retry": {
			putError: &mockPreconditionFailedError{
				message: "PreconditionFailed: At least one of the pre-conditions you specified did not hold",
			},
			wantRetry:   true,
			wantMessage: "concurrent modification detected - retry the operation",
		},
		"pre-condition text suggests retry": {
			putError: &mockPreconditionFailedError{
				message: "The pre-condition specified in one or more headers did not hold",
			},
			wantRetry:   true,
			wantMessage: "concurrent modification detected - retry the operation",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			client := &mockS3ClientWithAppend{
				getObjectFunc: func(ctx context.Context, input *s3.GetObjectInput, opts ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
					size := int64(100)
					return &s3.GetObjectOutput{
						Body:          io.NopCloser(bytes.NewReader([]byte("existing"))),
						ContentLength: &size,
						ETag:          aws.String("etag123"),
					}, nil
				},
				putObjectFunc: func(ctx context.Context, input *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
					return nil, tc.putError
				},
			}

			sw := NewS3WriterWithOptions(
				client,
				"test-bucket",
				"test-key",
				WithS3AppendMode(),
			)

			// Attempt to append
			err := sw.Write(context.Background(), FormatJSON, []byte(`{"key":"value"}`))

			// Verify error
			if err == nil {
				t.Fatal("expected ETag conflict error, got nil")
			}

			errMsg := err.Error()

			// Verify error message
			if !strings.Contains(errMsg, tc.wantMessage) {
				t.Errorf("expected error message to contain %q, got: %v", tc.wantMessage, errMsg)
			}

			// Verify error suggests retry
			if tc.wantRetry && !strings.Contains(errMsg, "retry") {
				t.Errorf("error should suggest retry, got: %v", errMsg)
			}
		})
	}
}

// mockPreconditionFailedError simulates S3 precondition failed error
type mockPreconditionFailedError struct {
	message string
}

func (e *mockPreconditionFailedError) Error() string {
	return e.message
}

// mockS3ClientWithAppend is a mock S3 client that supports both Get and Put operations
type mockS3ClientWithAppend struct {
	getObjectFunc func(ctx context.Context, input *s3.GetObjectInput, opts ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	putObjectFunc func(ctx context.Context, input *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

func (m *mockS3ClientWithAppend) GetObject(ctx context.Context, input *s3.GetObjectInput, opts ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if m.getObjectFunc != nil {
		return m.getObjectFunc(ctx, input, opts...)
	}
	return nil, errors.New("GetObject not implemented")
}

func (m *mockS3ClientWithAppend) PutObject(ctx context.Context, input *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if m.putObjectFunc != nil {
		return m.putObjectFunc(ctx, input, opts...)
	}
	return nil, errors.New("PutObject not implemented")
}

// TestS3Writer_AppendObjectNotFoundCreatesNew verifies that non-existent objects are created
func TestS3Writer_AppendObjectNotFoundCreatesNew(t *testing.T) {

	putCalled := false
	client := &mockS3ClientWithAppend{
		getObjectFunc: func(ctx context.Context, input *s3.GetObjectInput, opts ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			// Simulate object not found
			return nil, &types.NoSuchKey{
				Message: aws.String("The specified key does not exist."),
			}
		},
		putObjectFunc: func(ctx context.Context, input *s3.PutObjectInput, opts ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			putCalled = true
			// Verify no IfMatch (ETag) is set for new objects
			if input.IfMatch != nil {
				t.Errorf("expected IfMatch to be nil for new objects, got: %v", *input.IfMatch)
			}
			return &s3.PutObjectOutput{}, nil
		},
	}

	sw := NewS3WriterWithOptions(
		client,
		"test-bucket",
		"test-key",
		WithS3AppendMode(),
	)

	// Attempt to append to non-existent object
	err := sw.Write(context.Background(), FormatJSON, []byte(`{"key":"value"}`))

	// Verify no error
	if err != nil {
		t.Fatalf("expected no error for non-existent object, got: %v", err)
	}

	// Verify PutObject was called
	if !putCalled {
		t.Error("expected PutObject to be called for non-existent object")
	}
}
