package output

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileWriterHTMLAppendWithMarker(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx := context.Background()

	tests := map[string]struct {
		initialContent string
		appendContent  string
		wantErr        bool
		wantContent    string
	}{
		"append HTML content before marker": {
			initialContent: "<html><body><p>Initial</p>\n<!-- go-output-append -->\n</body></html>",
			appendContent:  "<p>Appended</p>",
			wantErr:        false,
			wantContent:    "<html><body><p>Initial</p>\n<p>Appended</p><!-- go-output-append -->\n</body></html>",
		},
		"append to HTML with multiple sections": {
			initialContent: "<html><body><section>First</section>\n<!-- go-output-append -->\n</body></html>",
			appendContent:  "<section>Second</section>",
			wantErr:        false,
			wantContent:    "<html><body><section>First</section>\n<section>Second</section><!-- go-output-append -->\n</body></html>",
		},
		"error when marker is missing": {
			initialContent: "<html><body><p>No marker</p></body></html>",
			appendContent:  "<p>Data</p>",
			wantErr:        true,
			wantContent:    "", // Not checked
		},
		"preserve content after marker": {
			initialContent: "<html><body><p>Content</p>\n<!-- go-output-append -->\n</body></html>",
			appendContent:  "<p>New</p>",
			wantErr:        false,
			wantContent:    "<html><body><p>Content</p>\n<p>New</p><!-- go-output-append -->\n</body></html>",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create a subdirectory for this test case to avoid conflicts
			testDir := filepath.Join(tempDir, name)
			err := os.MkdirAll(testDir, 0755)
			if err != nil {
				t.Fatalf("failed to create test directory: %v", err)
			}

			// Create FileWriter in append mode
			fw, err := NewFileWriterWithOptions(testDir, "test.html", WithAppendMode())
			if err != nil {
				t.Fatalf("failed to create FileWriter: %v", err)
			}

			// Create initial file with content
			filePath := filepath.Join(testDir, "test.html")
			err = os.WriteFile(filePath, []byte(tc.initialContent), 0644)
			if err != nil {
				t.Fatalf("failed to create initial file: %v", err)
			}

			// Append content
			err = fw.Write(ctx, FormatHTML, []byte(tc.appendContent))

			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
				return
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tc.wantErr {
				// Expected error, check that file wasn't modified
				content, err := os.ReadFile(filePath)
				if err != nil {
					t.Fatalf("failed to read file: %v", err)
				}
				if string(content) != tc.initialContent {
					t.Errorf("file should not be modified on error")
				}
				return
			}

			// Verify final content
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("failed to read final file: %v", err)
			}

			if string(content) != tc.wantContent {
				t.Errorf("content mismatch:\ngot:  %q\nwant: %q", string(content), tc.wantContent)
			}
		})
	}
}

func TestFileWriterHTMLAppendMarkerPreservation(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx := context.Background()

	tests := map[string]struct {
		initialMarker  string
		newContent     string
		wantMarker     string
		markerPosition string // "start" or "end" relative to new content
	}{
		"marker preserved after first append": {
			initialMarker:  "<!-- go-output-append -->",
			newContent:     "<p>First</p>",
			wantMarker:     "<!-- go-output-append -->",
			markerPosition: "end",
		},
		"marker preserved after multiple appends": {
			initialMarker:  "<!-- go-output-append -->",
			newContent:     "<p>Second</p>",
			wantMarker:     "<!-- go-output-append -->",
			markerPosition: "end",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create a subdirectory for this test case to avoid conflicts
			testDir := filepath.Join(tempDir, name)
			err := os.MkdirAll(testDir, 0755)
			if err != nil {
				t.Fatalf("failed to create test directory: %v", err)
			}

			fw, err := NewFileWriterWithOptions(testDir, "test.html", WithAppendMode())
			if err != nil {
				t.Fatalf("failed to create FileWriter: %v", err)
			}

			// Create initial file with marker
			filePath := filepath.Join(testDir, "test.html")
			initialContent := "<html><body>" + tc.initialMarker + "</body></html>"
			err = os.WriteFile(filePath, []byte(initialContent), 0644)
			if err != nil {
				t.Fatalf("failed to create initial file: %v", err)
			}

			// Append content
			err = fw.Write(ctx, FormatHTML, []byte(tc.newContent))
			if err != nil {
				t.Fatalf("failed to append: %v", err)
			}

			// Verify marker is still present
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			if !bytes.Contains(content, []byte(tc.wantMarker)) {
				t.Errorf("marker not found in content: %q", string(content))
			}

			// Verify marker is after new content
			if tc.markerPosition == "end" {
				markerIndex := bytes.Index(content, []byte(tc.wantMarker))
				contentIndex := bytes.Index(content, []byte(tc.newContent))
				if markerIndex < contentIndex {
					t.Errorf("marker should be after new content")
				}
			}
		})
	}
}

func TestFileWriterHTMLAppendEmptyContent(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx := context.Background()

	fw, err := NewFileWriterWithOptions(tempDir, "test.html", WithAppendMode())
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	// Create initial file with marker
	filePath := filepath.Join(tempDir, "test.html")
	initialContent := "<html><body><!-- go-output-append --></body></html>"
	err = os.WriteFile(filePath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to create initial file: %v", err)
	}

	// Append empty content
	err = fw.Write(ctx, FormatHTML, []byte{})
	if err != nil {
		t.Fatalf("failed to append empty content: %v", err)
	}

	// Verify file remains the same (marker position shouldn't change)
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if string(content) != initialContent {
		t.Errorf("appending empty content modified file:\ngot:  %q\nwant: %q", string(content), initialContent)
	}
}

func TestFileWriterHTMLAppendMultipleMarkers(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx := context.Background()

	testDir := filepath.Join(tempDir, "multimarker")
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	fw, err := NewFileWriterWithOptions(testDir, "test.html", WithAppendMode())
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	// Create file with multiple markers (only first one should be used)
	filePath := filepath.Join(testDir, "test.html")
	initialContent := "<html><body><!-- go-output-append --><p>Middle</p><!-- go-output-append --></body></html>"
	err = os.WriteFile(filePath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to create initial file: %v", err)
	}

	// Append content - should use first marker only
	err = fw.Write(ctx, FormatHTML, []byte("<p>New</p>"))
	if err != nil {
		t.Fatalf("failed to append: %v", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	// First marker should be replaced, second should remain
	if !bytes.Contains(content, []byte("<!-- go-output-append -->")) {
		t.Errorf("marker not found after append")
	}

	// Count markers - should have 2 markers (one after new content, one original second)
	markerCount := bytes.Count(content, []byte("<!-- go-output-append -->"))
	if markerCount != 2 {
		t.Errorf("expected 2 markers (one after new content, one original second), got %d", markerCount)
	}
}

func TestFileWriterHTMLAppendWithSpecialCharacters(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx := context.Background()

	tests := map[string]struct {
		content string
	}{
		"HTML entities": {
			content: "<p>&lt;script&gt; &amp; &quot;test&quot;</p>",
		},
		"UTF-8 characters": {
			content: "<p>Héllo Wörld 你好 🌍</p>",
		},
		"newlines in content": {
			content: "<p>Line 1\nLine 2\nLine 3</p>",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create a subdirectory for this test case to avoid conflicts
			testDir := filepath.Join(tempDir, name)
			err := os.MkdirAll(testDir, 0755)
			if err != nil {
				t.Fatalf("failed to create test directory: %v", err)
			}

			filePath := filepath.Join(testDir, "test.html")
			initialContent := "<html><body><!-- go-output-append --></body></html>"
			err = os.WriteFile(filePath, []byte(initialContent), 0644)
			if err != nil {
				t.Fatalf("failed to create initial file: %v", err)
			}

			// Create FileWriter with specific directory
			fw, err := NewFileWriterWithOptions(testDir, "test.html", WithAppendMode())
			if err != nil {
				t.Fatalf("failed to create FileWriter: %v", err)
			}

			// Append content with special characters
			err = fw.Write(ctx, FormatHTML, []byte(tc.content))
			if err != nil {
				t.Fatalf("failed to append: %v", err)
			}

			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			if !bytes.Contains(content, []byte(tc.content)) {
				t.Errorf("content not properly preserved:\ngot:  %q\nwant: %q", string(content), tc.content)
			}
		})
	}
}
