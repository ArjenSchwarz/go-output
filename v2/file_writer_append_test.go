package output

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileWriterWithAppendMode(t *testing.T) {
	skipIfNotIntegration(t)

	tempDir := t.TempDir()

	tests := map[string]struct {
		wantAppendMode bool
		opts           []FileWriterOption
	}{
		"default append mode disabled": {
			wantAppendMode: false,
			opts:           []FileWriterOption{},
		},
		"append mode enabled": {
			wantAppendMode: true,
			opts:           []FileWriterOption{WithAppendMode()},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			fw, err := NewFileWriterWithOptions(tempDir, "test-{format}.{ext}", tc.opts...)
			if err != nil {
				t.Fatalf("failed to create FileWriter: %v", err)
			}

			if fw.appendMode != tc.wantAppendMode {
				t.Errorf("appendMode = %v, want %v", fw.appendMode, tc.wantAppendMode)
			}
		})
	}
}

func TestFileWriterWithPermissions(t *testing.T) {
	skipIfNotIntegration(t)

	tempDir := t.TempDir()

	tests := map[string]struct {
		permissions os.FileMode
		opts        []FileWriterOption
	}{
		"default permissions": {
			permissions: 0644,
			opts:        []FileWriterOption{},
		},
		"custom permissions": {
			permissions: 0600,
			opts:        []FileWriterOption{WithPermissions(0600)},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			fw, err := NewFileWriterWithOptions(tempDir, "test-{format}.{ext}", tc.opts...)
			if err != nil {
				t.Fatalf("failed to create FileWriter: %v", err)
			}

			if fw.permissions != tc.permissions {
				t.Errorf("permissions = %o, want %o", fw.permissions, tc.permissions)
			}
		})
	}
}

func TestFileWriterAppendByteLevel(t *testing.T) {
	skipIfNotIntegration(t)

	tempDir := t.TempDir()
	fw, err := NewFileWriterWithOptions(tempDir, "test-{format}.{ext}", WithAppendMode())
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	ctx := context.Background()

	tests := map[string]struct {
		initialData   []byte
		appendData    []byte
		wantCombined  string
		wantErr       bool
		createInitial bool
	}{
		"append to new file": {
			initialData:   nil,
			appendData:    []byte("first"),
			wantCombined:  "first",
			wantErr:       false,
			createInitial: false,
		},
		"append to existing file": {
			initialData:   []byte("initial"),
			appendData:    []byte("appended"),
			wantCombined:  "initialappended",
			wantErr:       false,
			createInitial: true,
		},
		"multiple appends": {
			initialData:   []byte("a"),
			appendData:    []byte("b"),
			wantCombined:  "ab",
			wantErr:       false,
			createInitial: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			filename := "test-" + name + ".json"
			filepath := filepath.Join(fw.dir, filename)

			// Create initial file if needed
			if tc.createInitial && tc.initialData != nil {
				if err := os.WriteFile(filepath, tc.initialData, 0644); err != nil {
					t.Fatalf("failed to create initial file: %v", err)
				}
			}

			// Perform append
			err := fw.appendByteLevel(ctx, filepath, tc.appendData)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify file content
			content, err := os.ReadFile(filepath)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			if string(content) != tc.wantCombined {
				t.Errorf("file content = %q, want %q", string(content), tc.wantCombined)
			}
		})
	}
}

func TestFileWriterFormatValidation(t *testing.T) {
	skipIfNotIntegration(t)

	tempDir := t.TempDir()
	fw, err := NewFileWriterWithOptions(tempDir, "test-{format}.{ext}")
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	tests := map[string]struct {
		format      string
		filepath    string
		wantErr     bool
		errContains string
	}{
		"matching extension": {
			format:   FormatJSON,
			filepath: filepath.Join(fw.dir, "data.json"),
			wantErr:  false,
		},
		"mismatched extension": {
			format:      FormatJSON,
			filepath:    filepath.Join(fw.dir, "data.yaml"),
			wantErr:     true,
			errContains: "file extension mismatch",
		},
		"no extension": {
			format:   FormatJSON,
			filepath: filepath.Join(fw.dir, "data"),
			wantErr:  false,
		},
		"csv format": {
			format:   FormatCSV,
			filepath: filepath.Join(fw.dir, "data.csv"),
			wantErr:  false,
		},
		"csv with wrong extension": {
			format:      FormatCSV,
			filepath:    filepath.Join(fw.dir, "data.json"),
			wantErr:     true,
			errContains: "file extension mismatch",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			err := fw.validateFormatMatch(tc.format, tc.filepath)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("error = %v, want it to contain %q", err, tc.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestFileWriterAppendWithFormatValidation(t *testing.T) {
	skipIfNotIntegration(t)

	tempDir := t.TempDir()
	fw, err := NewFileWriterWithOptions(tempDir, "test-{format}.{ext}", WithAppendMode())
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	ctx := context.Background()

	tests := map[string]struct {
		format      string
		filename    string
		data        []byte
		wantErr     bool
		errContains string
	}{
		"append to matching file": {
			format:   FormatJSON,
			filename: "data.json",
			data:     []byte(`{}`),
			wantErr:  false,
		},
		"append to mismatched file": {
			format:      FormatJSON,
			filename:    "data.yaml",
			data:        []byte(`{}`),
			wantErr:     true,
			errContains: "mismatch",
		},
		"append to file with no extension": {
			format:   FormatJSON,
			filename: "data",
			data:     []byte(`{}`),
			wantErr:  false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			filepath := filepath.Join(fw.dir, tc.filename)

			// Create initial file
			if err := os.WriteFile(filepath, []byte("initial"), 0644); err != nil {
				t.Fatalf("failed to create file: %v", err)
			}

			// Try to append
			err := fw.appendToFile(ctx, tc.format, filepath, tc.data)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("error = %v, want it to contain %q", err, tc.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestFileWriterDisallowUnsafeAppend(t *testing.T) {
	skipIfNotIntegration(t)

	tempDir := t.TempDir()
	fw, err := NewFileWriterWithOptions(
		tempDir,
		"test-{format}.{ext}",
		WithAppendMode(),
		WithDisallowUnsafeAppend(),
	)
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	ctx := context.Background()

	tests := map[string]struct {
		format   string
		filename string
		wantErr  bool
	}{
		"json append disallowed": {
			format:   FormatJSON,
			filename: "data.json",
			wantErr:  true,
		},
		"yaml append disallowed": {
			format:   FormatYAML,
			filename: "data.yaml",
			wantErr:  true,
		},
		"csv append allowed": {
			format:   FormatCSV,
			filename: "data.csv",
			wantErr:  false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			filepath := filepath.Join(fw.dir, tc.filename)

			// Create initial file
			if err := os.WriteFile(filepath, []byte("initial"), 0644); err != nil {
				t.Fatalf("failed to create file: %v", err)
			}

			// Try to append
			err := fw.appendToFile(ctx, tc.format, filepath, []byte("appended"))
			if tc.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestFileWriterCSVHeaderSkipping(t *testing.T) {
	skipIfNotIntegration(t)

	tempDir := t.TempDir()
	fw, err := NewFileWriterWithOptions(tempDir, "test-{format}.{ext}", WithAppendMode())
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	ctx := context.Background()

	tests := map[string]struct {
		initialData  string
		appendData   string
		wantCombined string
		wantErr      bool
	}{
		"header is stripped when appending": {
			initialData:  "Name,Age\nAlice,30\n",
			appendData:   "Name,Age\nBob,25\n",
			wantCombined: "Name,Age\nAlice,30\nBob,25\n",
			wantErr:      false,
		},
		"unix LF line endings": {
			initialData:  "Name,Age\nAlice,30\n",
			appendData:   "Name,Age\nBob,25\n",
			wantCombined: "Name,Age\nAlice,30\nBob,25\n",
			wantErr:      false,
		},
		"windows CRLF line endings": {
			initialData:  "Name,Age\r\nAlice,30\r\n",
			appendData:   "Name,Age\r\nBob,25\r\n",
			wantCombined: "Name,Age\r\nAlice,30\r\nBob,25\n",
			wantErr:      false,
		},
		"mixed line endings": {
			initialData:  "Name,Age\r\nAlice,30\n",
			appendData:   "Name,Age\nBob,25\r\n",
			wantCombined: "Name,Age\r\nAlice,30\nBob,25\n",
			wantErr:      false,
		},
		"header-only CSV appends nothing": {
			initialData:  "Name,Age\nAlice,30\n",
			appendData:   "Name,Age\n",
			wantCombined: "Name,Age\nAlice,30\n",
			wantErr:      false,
		},
		"empty CSV data": {
			initialData:  "Name,Age\nAlice,30\n",
			appendData:   "",
			wantCombined: "Name,Age\nAlice,30\n",
			wantErr:      false,
		},
		"multiple data rows": {
			initialData:  "Name,Age\nAlice,30\n",
			appendData:   "Name,Age\nBob,25\nCharlie,35\n",
			wantCombined: "Name,Age\nAlice,30\nBob,25\nCharlie,35\n",
			wantErr:      false,
		},
		"CRLF header only": {
			initialData:  "Name,Age\r\nAlice,30\r\n",
			appendData:   "Name,Age\r\n",
			wantCombined: "Name,Age\r\nAlice,30\r\n",
			wantErr:      false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			filename := "test-csv-" + strings.ReplaceAll(name, " ", "-") + ".csv"
			filepath := filepath.Join(fw.dir, filename)

			// Create initial file
			if err := os.WriteFile(filepath, []byte(tc.initialData), 0644); err != nil {
				t.Fatalf("failed to create initial file: %v", err)
			}

			// Append CSV data (should strip headers)
			err := fw.appendCSVWithoutHeaders(ctx, filepath, []byte(tc.appendData))
			if tc.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify combined content
			content, err := os.ReadFile(filepath)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			if string(content) != tc.wantCombined {
				t.Errorf("file content = %q, want %q", string(content), tc.wantCombined)
			}
		})
	}
}
