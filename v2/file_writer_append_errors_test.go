package output

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFileWriter_AppendErrorHandling tests error scenarios for append operations
func TestFileWriter_AppendErrorHandling(t *testing.T) {

	tests := map[string]struct {
		setupFile    func(t *testing.T, dir string) string // Returns filepath
		format       string
		data         []byte
		wantErrMsg   string // Expected substring in error message
		checkErrType func(t *testing.T, err error)
	}{
		"format mismatch includes expected and actual extensions": {
			setupFile: func(t *testing.T, dir string) string {
				t.Helper()
				// Create a .txt file but try to append JSON
				path := filepath.Join(dir, "test.txt")
				if err := os.WriteFile(path, []byte("initial content"), 0644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				return path
			},
			format:     FormatJSON,
			data:       []byte(`{"key":"value"}`),
			wantErrMsg: `expected "json" but file has "txt"`,
			checkErrType: func(t *testing.T, err error) {
				t.Helper()
				var we *WriteError
				if !errors.As(err, &we) {
					t.Errorf("expected WriteError, got %T", err)
				}
			},
		},
		"HTML marker missing error includes file path": {
			setupFile: func(t *testing.T, dir string) string {
				t.Helper()
				// Create HTML file without marker
				path := filepath.Join(dir, "test.html")
				html := `<html><body><h1>Test</h1></body></html>`
				if err := os.WriteFile(path, []byte(html), 0644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				return path
			},
			format:     FormatHTML,
			data:       []byte(`<p>New content</p>`),
			wantErrMsg: "HTML append marker not found in file:",
			checkErrType: func(t *testing.T, err error) {
				t.Helper()
				var we *WriteError
				if !errors.As(err, &we) {
					t.Errorf("expected WriteError, got %T", err)
				}
				if !strings.Contains(err.Error(), "test.html") {
					t.Errorf("error should contain file path, got: %v", err)
				}
			},
		},
		"file read error includes operation and path": {
			setupFile: func(t *testing.T, dir string) string {
				t.Helper()
				// Create a file with no read permissions
				path := filepath.Join(dir, "test.html")
				html := `<html><body><!-- go-output-append --></body></html>`
				if err := os.WriteFile(path, []byte(html), 0000); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				return path
			},
			format:     FormatHTML,
			data:       []byte(`<p>New content</p>`),
			wantErrMsg: "failed to read existing file:",
			checkErrType: func(t *testing.T, err error) {
				t.Helper()
				var we *WriteError
				if !errors.As(err, &we) {
					t.Errorf("expected WriteError, got %T", err)
				}
			},
		},
		"verify all errors wrapped with WriteError - format mismatch": {
			setupFile: func(t *testing.T, dir string) string {
				t.Helper()
				path := filepath.Join(dir, "test.csv")
				if err := os.WriteFile(path, []byte("Name,Age\nAlice,30\n"), 0644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				return path
			},
			format:     FormatJSON,
			data:       []byte(`{"key":"value"}`),
			wantErrMsg: "file extension mismatch",
			checkErrType: func(t *testing.T, err error) {
				t.Helper()
				var we *WriteError
				if !errors.As(err, &we) {
					t.Errorf("expected WriteError, got %T", err)
				}
				if we.Writer != "file" {
					t.Errorf("expected Writer='file', got %q", we.Writer)
				}
				if we.Format != FormatJSON {
					t.Errorf("expected Format='json', got %q", we.Format)
				}
			},
		},
		"verify all errors wrapped with WriteError - HTML marker missing": {
			setupFile: func(t *testing.T, dir string) string {
				t.Helper()
				path := filepath.Join(dir, "test.html")
				if err := os.WriteFile(path, []byte("<html><body></body></html>"), 0644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				return path
			},
			format:     FormatHTML,
			data:       []byte(`<p>content</p>`),
			wantErrMsg: "HTML append marker not found",
			checkErrType: func(t *testing.T, err error) {
				t.Helper()
				var we *WriteError
				if !errors.As(err, &we) {
					t.Errorf("expected WriteError, got %T", err)
				}
				if we.Writer != "file" {
					t.Errorf("expected Writer='file', got %q", we.Writer)
				}
				if we.Format != FormatHTML {
					t.Errorf("expected Format='html', got %q", we.Format)
				}
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			// Create temp directory
			dir := t.TempDir()

			// Setup test file
			filePath := tc.setupFile(t, dir)

			// Create FileWriter in append mode pointing to the file's directory
			fw, err := NewFileWriterWithOptions(
				dir,
				filepath.Base(filePath), // Use filename as pattern
				WithAppendMode(),
			)
			if err != nil {
				t.Fatalf("failed to create FileWriter: %v", err)
			}

			// Attempt to append
			err = fw.Write(context.Background(), tc.format, tc.data)

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

// TestFileWriter_AppendHTMLMarkerErrorMessage verifies HTML marker error includes marker format
func TestFileWriter_AppendHTMLMarkerErrorMessage(t *testing.T) {

	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.html")

	// Create HTML file without the marker
	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<h1>Existing Content</h1>
</body>
</html>`

	if err := os.WriteFile(filePath, []byte(html), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create FileWriter in append mode
	fw, err := NewFileWriterWithOptions(dir, "test.html", WithAppendMode())
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	// Attempt to append
	newContent := []byte(`<p>New paragraph</p>`)
	err = fw.Write(context.Background(), FormatHTML, newContent)

	// Verify error
	if err == nil {
		t.Fatal("expected error for missing HTML marker, got nil")
	}

	// Check error message includes file path
	if !strings.Contains(err.Error(), filePath) {
		t.Errorf("error should contain file path %q, got: %v", filePath, err)
	}

	// Check error message mentions the marker
	if !strings.Contains(err.Error(), "marker") {
		t.Errorf("error should mention 'marker', got: %v", err)
	}
}

// TestFileWriter_AppendFormatMismatchDetails verifies format mismatch error details
func TestFileWriter_AppendFormatMismatchDetails(t *testing.T) {

	tests := map[string]struct {
		fileExt      string
		format       string
		wantExpected string
		wantActual   string
	}{
		"JSON to CSV file": {
			fileExt:      "csv",
			format:       FormatJSON,
			wantExpected: "json",
			wantActual:   "csv",
		},
		"HTML to Markdown file": {
			fileExt:      "md",
			format:       FormatHTML,
			wantExpected: "html",
			wantActual:   "md",
		},
		"YAML to JSON file": {
			fileExt:      "json",
			format:       FormatYAML,
			wantExpected: "yaml",
			wantActual:   "json",
		},
		"Table to JSON file": {
			fileExt:      "json",
			format:       FormatTable,
			wantExpected: "txt", // FormatTable maps to "txt" extension
			wantActual:   "json",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			dir := t.TempDir()
			filename := "test." + tc.fileExt
			filePath := filepath.Join(dir, filename)

			// Create file with initial content
			if err := os.WriteFile(filePath, []byte("initial content"), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			// Create FileWriter in append mode
			fw, err := NewFileWriterWithOptions(dir, filename, WithAppendMode())
			if err != nil {
				t.Fatalf("failed to create FileWriter: %v", err)
			}

			// Attempt to append with wrong format
			err = fw.Write(context.Background(), tc.format, []byte("new data"))

			// Verify error
			if err == nil {
				t.Fatal("expected format mismatch error, got nil")
			}

			errMsg := err.Error()

			// Verify error contains expected extension
			if !strings.Contains(errMsg, tc.wantExpected) {
				t.Errorf("error should contain expected extension %q, got: %v", tc.wantExpected, errMsg)
			}

			// Verify error contains actual extension
			if !strings.Contains(errMsg, tc.wantActual) {
				t.Errorf("error should contain actual extension %q, got: %v", tc.wantActual, errMsg)
			}

			// Note: We don't verify filename in error because validateFormatMatch
			// doesn't include it, and that's acceptable - the error provides enough
			// context with writer name, format, expected ext, and actual ext
		})
	}
}

// TestFileWriter_AppendIOErrorsIncludeContext verifies I/O errors include operation context
func TestFileWriter_AppendIOErrorsIncludeContext(t *testing.T) {

	tests := map[string]struct {
		setupError    func(t *testing.T, dir string) string
		format        string
		wantOperation string // Expected operation in error (read/write/rename)
	}{
		"read error includes 'read' operation": {
			setupError: func(t *testing.T, dir string) string {
				t.Helper()
				filePath := filepath.Join(dir, "test.html")
				// Create HTML file with marker but no read permissions
				html := `<html><body><!-- go-output-append --></body></html>`
				if err := os.WriteFile(filePath, []byte(html), 0000); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				return filePath
			},
			format:        FormatHTML,
			wantOperation: "read",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			dir := t.TempDir()
			filePath := tc.setupError(t, dir)

			// Create FileWriter in append mode
			fw, err := NewFileWriterWithOptions(dir, filepath.Base(filePath), WithAppendMode())
			if err != nil {
				t.Fatalf("failed to create FileWriter: %v", err)
			}

			// Attempt to append
			err = fw.Write(context.Background(), tc.format, []byte("new content"))

			// Verify error
			if err == nil {
				t.Fatal("expected I/O error, got nil")
			}

			errMsg := err.Error()

			// Verify error mentions the operation
			if !strings.Contains(errMsg, tc.wantOperation) {
				t.Errorf("error should mention %q operation, got: %v", tc.wantOperation, errMsg)
			}

			// Verify error includes file path
			if !strings.Contains(errMsg, filePath) && !strings.Contains(errMsg, filepath.Base(filePath)) {
				t.Errorf("error should include file path or filename, got: %v", errMsg)
			}
		})
	}
}
