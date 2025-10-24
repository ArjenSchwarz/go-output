package output

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestFileWriter_CrossPlatformPathHandling verifies path handling works across platforms
func TestFileWriter_CrossPlatformPathHandling(t *testing.T) {

	tests := map[string]struct {
		pattern      string
		format       string
		wantFilename string // Expected filename (platform-agnostic)
	}{
		"simple filename": {
			pattern:      "output.{ext}",
			format:       FormatJSON,
			wantFilename: "output.json",
		},
		"filename with extension": {
			pattern:      "report.{ext}",
			format:       FormatJSON,
			wantFilename: "report.json",
		},
		"subdirectory in pattern": {
			pattern:      "subdir/output.{ext}",
			format:       FormatHTML,
			wantFilename: filepath.Join("subdir", "output.html"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			dir := t.TempDir()

			// Create subdirectory if pattern contains one
			if strings.Contains(tc.pattern, "/") {
				subdir := filepath.Join(dir, "subdir")
				if err := os.MkdirAll(subdir, 0755); err != nil {
					t.Fatalf("failed to create subdirectory: %v", err)
				}
			}

			// Create FileWriter with append mode
			fw, err := NewFileWriterWithOptions(dir, tc.pattern, WithAppendMode())
			if err != nil {
				t.Fatalf("failed to create FileWriter: %v", err)
			}

			// Write data
			data := []byte("test content")
			if err := fw.Write(context.Background(), tc.format, data); err != nil {
				t.Fatalf("failed to write: %v", err)
			}

			// Verify file was created with correct name
			expectedPath := filepath.Join(dir, tc.wantFilename)
			if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
				t.Errorf("expected file %q to exist", expectedPath)
			}
		})
	}
}

// TestFileWriter_CrossPlatformPermissions verifies file permission handling across platforms
func TestFileWriter_CrossPlatformPermissions(t *testing.T) {

	// Skip on Windows as permission model is different
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix permission test on Windows")
	}

	tests := map[string]struct {
		permissions os.FileMode
		wantMode    os.FileMode // Expected file mode (Unix only)
	}{
		"default 0644 permissions": {
			permissions: 0644,
			wantMode:    0644,
		},
		"custom 0600 permissions": {
			permissions: 0600,
			wantMode:    0600,
		},
		"custom 0755 permissions": {
			permissions: 0755,
			wantMode:    0755,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			dir := t.TempDir()

			// Create FileWriter with custom permissions
			fw, err := NewFileWriterWithOptions(
				dir,
				"test.json",
				WithPermissions(tc.permissions),
			)
			if err != nil {
				t.Fatalf("failed to create FileWriter: %v", err)
			}

			// Write data
			data := []byte(`{"key":"value"}`)
			if err := fw.Write(context.Background(), FormatJSON, data); err != nil {
				t.Fatalf("failed to write: %v", err)
			}

			// Verify file permissions
			expectedPath := filepath.Join(dir, "test.json")
			info, err := os.Stat(expectedPath)
			if err != nil {
				t.Fatalf("failed to stat file: %v", err)
			}

			gotMode := info.Mode().Perm()
			if gotMode != tc.wantMode {
				t.Errorf("expected file mode %o, got %o", tc.wantMode, gotMode)
			}
		})
	}
}

// TestFileWriter_CSVLineEndingHandling verifies CRLF handling for CSV append
func TestFileWriter_CSVLineEndingHandling(t *testing.T) {

	tests := map[string]struct {
		initialData  string
		appendData   string
		wantCombined string
	}{
		"Unix LF line endings": {
			initialData:  "Name,Age\nAlice,30\n",
			appendData:   "Name,Age\nBob,25\n",
			wantCombined: "Name,Age\nAlice,30\nBob,25\n",
		},
		"Windows CRLF line endings": {
			initialData:  "Name,Age\r\nAlice,30\r\n",
			appendData:   "Name,Age\r\nBob,25\r\n",
			wantCombined: "Name,Age\r\nAlice,30\r\nBob,25\n", // Initial file CRLF preserved, appended data normalized
		},
		"Mixed line endings": {
			initialData:  "Name,Age\r\nAlice,30\n",
			appendData:   "Name,Age\nBob,25\r\n",
			wantCombined: "Name,Age\r\nAlice,30\nBob,25\n",
		},
		"Only CRLF initial data": {
			initialData:  "Name,Age\r\nAlice,30\r\n",
			appendData:   "Name,Age\nBob,25\n",
			wantCombined: "Name,Age\r\nAlice,30\r\nBob,25\n", // Initial file CRLF preserved
		},
		"Only CRLF append data": {
			initialData:  "Name,Age\nAlice,30\n",
			appendData:   "Name,Age\r\nBob,25\r\n",
			wantCombined: "Name,Age\nAlice,30\nBob,25\n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			dir := t.TempDir()
			filePath := filepath.Join(dir, "test.csv")

			// Write initial data
			if err := os.WriteFile(filePath, []byte(tc.initialData), 0644); err != nil {
				t.Fatalf("failed to write initial data: %v", err)
			}

			// Create FileWriter in append mode
			fw, err := NewFileWriterWithOptions(dir, "test.csv", WithAppendMode())
			if err != nil {
				t.Fatalf("failed to create FileWriter: %v", err)
			}

			// Append data
			if err := fw.Write(context.Background(), FormatCSV, []byte(tc.appendData)); err != nil {
				t.Fatalf("failed to append: %v", err)
			}

			// Read combined result
			got, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("failed to read combined file: %v", err)
			}

			if string(got) != tc.wantCombined {
				t.Errorf("combined data mismatch:\nwant: %q\ngot:  %q", tc.wantCombined, string(got))
			}
		})
	}
}

// TestFileWriter_PathSeparatorHandling verifies filepath.Join usage across platforms
func TestFileWriter_PathSeparatorHandling(t *testing.T) {

	dir := t.TempDir()

	// Create subdirectory structure
	subdir := filepath.Join(dir, "reports", "daily")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	// Create FileWriter with nested path
	pattern := filepath.Join("reports", "daily", "output.{ext}")
	fw, err := NewFileWriterWithOptions(dir, pattern)
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	// Write data
	data := []byte(`{"test":"data"}`)
	if err := fw.Write(context.Background(), FormatJSON, data); err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Verify file was created with correct path separators
	expectedPath := filepath.Join(dir, "reports", "daily", "output.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected file %q to exist (using platform-specific path separators)", expectedPath)
	}
}

// TestFileWriter_UTF8Encoding verifies UTF-8 content handling across platforms
func TestFileWriter_UTF8Encoding(t *testing.T) {

	tests := map[string]struct {
		initialData string
		appendData  string
		format      string
	}{
		"UTF-8 emoji in JSON": {
			initialData: `{"message":"Hello 👋"}`,
			appendData:  `{"message":"世界 🌍"}`,
			format:      FormatJSON,
		},
		"UTF-8 Chinese characters in text": {
			initialData: "你好世界\n",
			appendData:  "こんにちは\n",
			format:      FormatTable,
		},
		"UTF-8 special characters in CSV": {
			initialData: "Name,Description\nTest,Café ☕\n",
			appendData:  "Name,Description\nExample,Naïve 🎭\n",
			format:      FormatCSV,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			dir := t.TempDir()
			// Get the correct extension for the format
			ext := defaultExtensions()[tc.format]
			filename := "test." + ext
			filePath := filepath.Join(dir, filename)

			// Write initial UTF-8 data
			if err := os.WriteFile(filePath, []byte(tc.initialData), 0644); err != nil {
				t.Fatalf("failed to write initial data: %v", err)
			}

			// Create FileWriter in append mode
			fw, err := NewFileWriterWithOptions(dir, filename, WithAppendMode())
			if err != nil {
				t.Fatalf("failed to create FileWriter: %v", err)
			}

			// Append UTF-8 data
			if err := fw.Write(context.Background(), tc.format, []byte(tc.appendData)); err != nil {
				t.Fatalf("failed to append: %v", err)
			}

			// Read and verify UTF-8 content is preserved
			got, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			// Verify both initial and append data are present
			combined := string(got)
			if !strings.Contains(combined, tc.initialData) {
				t.Errorf("combined data missing initial content: %q", combined)
			}
			// For formats with headers (CSV), append data may have headers stripped
			if tc.format != FormatCSV {
				if !strings.Contains(combined, tc.appendData) {
					t.Errorf("combined data missing append content: %q", combined)
				}
			}
		})
	}
}

// TestFileWriter_WindowsReservedNames documents Windows reserved filename behavior
func TestFileWriter_WindowsReservedNames(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test on non-Windows platform")
	}

	// Document Windows reserved names behavior
	// Note: This test documents expected failures on Windows
	reservedNames := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4",
		"LPT1", "LPT2", "LPT3",
	}

	dir := t.TempDir()

	for _, reserved := range reservedNames {
		t.Run(reserved, func(t *testing.T) {
			fw, err := NewFileWriterWithOptions(dir, reserved)
			if err != nil {
				t.Fatalf("failed to create FileWriter: %v", err)
			}

			// Attempt to write - this should fail on Windows
			err = fw.Write(context.Background(), FormatJSON, []byte(`{"test":"data"}`))

			// Document that Windows reserves these names
			if err == nil {
				t.Logf("Note: Windows reserved name %q was successfully created (unexpected)", reserved)
			} else {
				t.Logf("Note: Windows reserved name %q failed as expected: %v", reserved, err)
			}
		})
	}
}
