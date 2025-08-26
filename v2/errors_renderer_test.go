package output

import (
	"errors"
	"strings"
	"testing"
)

func TestRenderError(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		content  Content
		cause    error
		expected string
	}{
		{
			name:     "with table content",
			format:   "json",
			content:  &TableContent{id: "test-123"},
			cause:    errors.New("serialization failed"),
			expected: "render failed; format=json; content_type=table; content_id=test-123; cause: serialization failed",
		},
		{
			name:     "with text content",
			format:   "html",
			content:  &TextContent{id: "text-456"},
			cause:    errors.New("encoding error"),
			expected: "render failed; format=html; content_type=text; content_id=text-456; cause: encoding error",
		},
		{
			name:     "with nil content",
			format:   "csv",
			content:  nil,
			cause:    errors.New("content missing"),
			expected: "render failed; format=csv; cause: content missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewRenderError(tt.format, tt.content, tt.cause)

			if err.Error() != tt.expected {
				t.Errorf("RenderError.Error() = %q, want %q", err.Error(), tt.expected)
			}

			if err.Format != tt.format {
				t.Errorf("RenderError.Format = %q, want %q", err.Format, tt.format)
			}

			if !errors.Is(err, tt.cause) {
				t.Errorf("RenderError should wrap the cause error")
			}
		})
	}
}
func TestEnhancedRenderError(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		renderer    string
		operation   string
		content     Content
		context     map[string]any
		cause       error
		expectParts []string
	}{
		{
			name:      "detailed render error",
			format:    "json",
			renderer:  "JSONRenderer",
			operation: "encode",
			content:   &TableContent{id: "test-123"},
			context:   map[string]any{"data_size": 1024, "encoding": "utf-8"},
			cause:     errors.New("json encoding failed"),
			expectParts: []string{
				"operation \"encode\" failed",
				"format=json",
				"renderer=JSONRenderer",
				"content_type=table",
				"content_id=test-123",
				"data_size=1024",
				"encoding=utf-8",
				"cause: json encoding failed",
			},
		},
		{
			name:    "minimal render error",
			format:  "csv",
			content: nil,
			cause:   errors.New("content missing"),
			expectParts: []string{
				"render failed",
				"format=csv",
				"cause: content missing",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err *RenderError
			if tt.renderer != "" && tt.operation != "" {
				err = NewRenderErrorWithDetails(tt.format, tt.renderer, tt.operation, tt.content, tt.cause)
			} else {
				err = NewRenderError(tt.format, tt.content, tt.cause)
			}

			// Add context if provided
			for k, v := range tt.context {
				err.AddContext(k, v)
			}

			errorStr := err.Error()
			for _, part := range tt.expectParts {
				if !strings.Contains(errorStr, part) {
					t.Errorf("RenderError.Error() should contain %q, got: %s", part, errorStr)
				}
			}

			// Test unwrapping
			if !errors.Is(err, tt.cause) {
				t.Errorf("RenderError should wrap the cause error")
			}
		})
	}
}

// TestWriterError tests the new WriterError type
func TestWriterError(t *testing.T) {
	tests := []struct {
		name        string
		writer      string
		format      string
		operation   string
		context     map[string]any
		cause       error
		expectParts []string
	}{
		{
			name:      "detailed writer error",
			writer:    "FileWriter",
			format:    "html",
			operation: "write",
			context:   map[string]any{"file_path": "/tmp/output.html", "data_size": 2048},
			cause:     errors.New("permission denied"),
			expectParts: []string{
				"operation \"write\" failed",
				"format=html",
				"writer=FileWriter",
				"file_path=/tmp/output.html",
				"data_size=2048",
				"cause: permission denied",
			},
		},
		{
			name:   "minimal writer error",
			writer: "S3Writer",
			format: "json",
			cause:  errors.New("network timeout"),
			expectParts: []string{
				"write failed",
				"format=json",
				"writer=S3Writer",
				"cause: network timeout",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err *WriterError
			if tt.operation != "" {
				err = NewWriterErrorWithDetails(tt.writer, tt.format, tt.operation, tt.cause)
			} else {
				err = NewWriterError(tt.writer, tt.format, tt.cause)
			}

			// Add context if provided
			for k, v := range tt.context {
				err.AddContext(k, v)
			}

			errorStr := err.Error()
			for _, part := range tt.expectParts {
				if !strings.Contains(errorStr, part) {
					t.Errorf("WriterError.Error() should contain %q, got: %s", part, errorStr)
				}
			}

			// Test unwrapping
			if !errors.Is(err, tt.cause) {
				t.Errorf("WriterError should wrap the cause error")
			}
		})
	}
}

// TestMultiErrorWithSourceTracking tests enhanced MultiError with source tracking
