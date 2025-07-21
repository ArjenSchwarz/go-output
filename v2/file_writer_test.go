package output

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestNewFileWriter(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name    string
		dir     string
		pattern string
		wantErr bool
	}{
		{
			name:    "valid directory",
			dir:     tempDir,
			pattern: "test-{format}.{ext}",
			wantErr: false,
		},
		{
			name:    "create new directory",
			dir:     filepath.Join(tempDir, "new", "nested", "dir"),
			pattern: "",
			wantErr: false,
		},
		{
			name:    "default pattern",
			dir:     tempDir,
			pattern: "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fw, err := NewFileWriter(tt.dir, tt.pattern)
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

			if fw == nil {
				t.Error("FileWriter is nil")
				return
			}

			// Check directory was created
			if _, err := os.Stat(fw.dir); err != nil {
				t.Errorf("directory not created: %v", err)
			}

			// Check default pattern
			if tt.pattern == "" && fw.pattern != "output-{format}.{ext}" {
				t.Errorf("pattern = %q, want %q", fw.pattern, "output-{format}.{ext}")
			}
		})
	}
}

func TestFileWriterWrite(t *testing.T) {
	tempDir := t.TempDir()
	fw, err := NewFileWriter(tempDir, "test-{format}.{ext}")
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	ctx := context.Background()
	testData := []byte("test content")

	tests := []struct {
		name     string
		format   string
		data     []byte
		wantFile string
		wantErr  bool
	}{
		{
			name:     "write JSON file",
			format:   FormatJSON,
			data:     testData,
			wantFile: "test-json.json",
			wantErr:  false,
		},
		{
			name:     "write YAML file",
			format:   FormatYAML,
			data:     testData,
			wantFile: "test-yaml.yaml",
			wantErr:  false,
		},
		{
			name:     "write CSV file",
			format:   FormatCSV,
			data:     testData,
			wantFile: "test-csv.csv",
			wantErr:  false,
		},
		{
			name:    "empty format",
			format:  "",
			data:    testData,
			wantErr: true,
		},
		{
			name:    "nil data",
			format:  FormatJSON,
			data:    nil,
			wantErr: true,
		},
		{
			name:     "empty data is valid",
			format:   FormatJSON,
			data:     []byte{},
			wantFile: "test-json.json",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fw.Write(ctx, tt.format, tt.data)
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

			// Check file was created with correct content
			filePath := filepath.Join(tempDir, tt.wantFile)
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Errorf("failed to read file: %v", err)
				return
			}

			if string(content) != string(tt.data) {
				t.Errorf("file content = %q, want %q", content, tt.data)
			}
		})
	}
}

func TestFileWriterConcurrency(t *testing.T) {
	tempDir := t.TempDir()
	fw, err := NewFileWriter(tempDir, "concurrent-{format}.{ext}")
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	ctx := context.Background()
	formats := []string{FormatJSON, FormatYAML, FormatCSV, FormatHTML, FormatMarkdown}
	var wg sync.WaitGroup

	// Write files concurrently
	for i := 0; i < 10; i++ {
		for _, format := range formats {
			wg.Add(1)
			go func(idx int, fmt string) {
				defer wg.Done()
				data := []byte(strings.Repeat(fmt, idx+1))
				if err := fw.Write(ctx, fmt, data); err != nil {
					t.Errorf("concurrent write failed: %v", err)
				}
			}(i, format)
		}
	}

	wg.Wait()

	// Verify final files contain expected content
	for _, format := range formats {
		filename := fw.generateFilenameForTest(format)
		filePath := filepath.Join(tempDir, filename)

		if _, err := os.Stat(filePath); err != nil {
			t.Errorf("file %q not found: %v", filename, err)
		}
	}
}

// Helper method for testing
func (fw *FileWriter) generateFilenameForTest(format string) string {
	filename, _ := fw.generateFilename(format)
	return filename
}

func TestFileWriterSecurityValidation(t *testing.T) {
	tempDir := t.TempDir()
	fw, err := NewFileWriter(tempDir, "{format}")
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	ctx := context.Background()
	testData := []byte("test")

	// Test directory traversal attempts
	maliciousFormats := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\config\\sam",
		"/etc/passwd",
		"C:\\Windows\\System32\\drivers\\etc\\hosts",
		"subdir/../../../escaped",
		"./././../../../escaped",
	}

	for _, format := range maliciousFormats {
		t.Run("malicious_"+format, func(t *testing.T) {
			err := fw.Write(ctx, format, testData)
			if err == nil {
				t.Errorf("expected security error for format %q but got nil", format)
			}
		})
	}
}

func TestFileWriterContextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	fw, err := NewFileWriter(tempDir, "test-{format}.{ext}")
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = fw.Write(ctx, FormatJSON, []byte("test"))
	if err == nil {
		t.Error("expected context cancellation error")
	}

	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("error should mention context cancellation, got: %v", err)
	}
}

func TestFileWriterCustomExtensions(t *testing.T) {
	tempDir := t.TempDir()

	customExt := map[string]string{
		"config": "conf",
		"data":   "dat",
	}

	fw, err := NewFileWriterWithOptions(tempDir, "output-{format}.{ext}",
		WithExtensions(customExt))
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		format   string
		wantFile string
	}{
		{"config", "output-config.conf"},
		{"data", "output-data.dat"},
		{FormatJSON, "output-json.json"}, // Should still use default
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			err := fw.Write(ctx, tt.format, []byte("test"))
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			filePath := filepath.Join(tempDir, tt.wantFile)
			if _, err := os.Stat(filePath); err != nil {
				t.Errorf("expected file %q not found: %v", tt.wantFile, err)
			}
		})
	}
}

func TestFileWriterOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	fw, err := NewFileWriter(tempDir, "overwrite-test.txt")
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	ctx := context.Background()

	// Write initial content
	initialData := []byte("initial content")
	if err := fw.Write(ctx, "txt", initialData); err != nil {
		t.Fatalf("failed to write initial content: %v", err)
	}

	// Overwrite with new content
	newData := []byte("new content")
	if err := fw.Write(ctx, "txt", newData); err != nil {
		t.Fatalf("failed to overwrite content: %v", err)
	}

	// Verify file contains new content only
	filePath := filepath.Join(tempDir, "overwrite-test.txt")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if string(content) != string(newData) {
		t.Errorf("file content = %q, want %q", content, newData)
	}
}

func TestGenerateFilename(t *testing.T) {
	fw := &FileWriter{
		pattern:    "report-{format}.{ext}",
		extensions: defaultExtensions(),
	}

	tests := []struct {
		format   string
		expected string
	}{
		{FormatJSON, "report-json.json"},
		{FormatYAML, "report-yaml.yaml"},
		{FormatCSV, "report-csv.csv"},
		{"unknown", "report-unknown.unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			filename, err := fw.generateFilename(tt.format)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if filename != tt.expected {
				t.Errorf("filename = %q, want %q", filename, tt.expected)
			}
		})
	}
}
