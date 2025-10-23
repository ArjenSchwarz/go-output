package output

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileWriterHTMLRendering_NewFileGetsFullPage(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx := context.Background()

	fw, err := NewFileWriterWithOptions(tempDir, "test.html", WithAppendMode())
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	// Create a full HTML page (as would be rendered by HTML renderer with useTemplate=true)
	fullHTMLPage := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Test</title>
</head>
<body>
<p>Content</p>
<!-- go-output-append -->
</body>
</html>`

	// Write to new file
	err = fw.Write(ctx, FormatHTML, []byte(fullHTMLPage))
	if err != nil {
		t.Fatalf("failed to write: %v", err)
	}

	// Verify file exists and has the full page structure
	filePath := filepath.Join(tempDir, "test.html")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if !bytes.Contains(content, []byte("<!DOCTYPE html>")) {
		t.Errorf("DOCTYPE not found")
	}

	if !bytes.Contains(content, []byte("<html")) {
		t.Errorf("<html> tag not found")
	}

	if !bytes.Contains(content, []byte("</html>")) {
		t.Errorf("</html> tag not found")
	}

	if !bytes.Contains(content, []byte(HTMLAppendMarker)) {
		t.Errorf("append marker not found")
	}
}

func TestFileWriterHTMLRendering_ExistingFileExpectsFragment(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx := context.Background()

	fw, err := NewFileWriterWithOptions(tempDir, "test.html", WithAppendMode())
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	// Create initial HTML file with marker (as would be created in first write)
	filePath := filepath.Join(tempDir, "test.html")
	initialContent := `<!DOCTYPE html>
<html>
<head>
  <title>Test</title>
</head>
<body>
<p>First content</p>
<!-- go-output-append -->
</body>
</html>`

	err = os.WriteFile(filePath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to create initial file: %v", err)
	}

	// Append a fragment (no DOCTYPE, no html/body tags)
	fragment := "<p>Fragment content</p>"

	err = fw.Write(ctx, FormatHTML, []byte(fragment))
	if err != nil {
		t.Fatalf("failed to append fragment: %v", err)
	}

	// Verify fragment was inserted before marker
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if !bytes.Contains(content, []byte(fragment)) {
		t.Errorf("fragment not found in file")
	}

	if !bytes.Contains(content, []byte(HTMLAppendMarker)) {
		t.Errorf("marker not found after append")
	}

	// Fragment should be before marker
	fragmentIndex := bytes.Index(content, []byte(fragment))
	markerIndex := bytes.Index(content, []byte(HTMLAppendMarker))
	if fragmentIndex > markerIndex {
		t.Errorf("fragment should be before marker")
	}

	// Verify original DOCTYPE and html/body tags are still there
	if !bytes.Contains(content, []byte("<!DOCTYPE html>")) {
		t.Errorf("DOCTYPE should remain in file")
	}

	if !bytes.Contains(content, []byte("</html>")) {
		t.Errorf("closing </html> should remain")
	}
}

func TestFileWriterHTMLRendering_ModeSelectionBasedOnFileExistence(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx := context.Background()

	tests := map[string]struct {
		fileExists       bool
		createBefore     bool
		dataToWrite      string
		shouldHaveMarker bool
	}{
		"new file gets full page with marker": {
			fileExists:       false,
			createBefore:     false,
			dataToWrite:      "<!DOCTYPE html>\n<html><body><p>Content</p>\n<!-- go-output-append -->\n</body></html>",
			shouldHaveMarker: true,
		},
		"existing file accepts fragment with marker": {
			fileExists:       true,
			createBefore:     true,
			dataToWrite:      "<p>Fragment</p>",
			shouldHaveMarker: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fw, err := NewFileWriterWithOptions(tempDir, name+".html", WithAppendMode())
			if err != nil {
				t.Fatalf("failed to create FileWriter: %v", err)
			}

			filePath := filepath.Join(tempDir, name+".html")

			if tc.createBefore && tc.fileExists {
				// Pre-create file with marker
				initialContent := "<!DOCTYPE html>\n<html><body>\n<!-- go-output-append -->\n</body></html>"
				err = os.WriteFile(filePath, []byte(initialContent), 0644)
				if err != nil {
					t.Fatalf("failed to create initial file: %v", err)
				}
			}

			// Write data
			err = fw.Write(ctx, FormatHTML, []byte(tc.dataToWrite))
			if err != nil {
				t.Fatalf("failed to write: %v", err)
			}

			// Verify file exists
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("file not created: %v", err)
			}

			if tc.shouldHaveMarker && !bytes.Contains(content, []byte(HTMLAppendMarker)) {
				t.Errorf("marker should be present")
			}
		})
	}
}

func TestFileWriterHTMLRendering_NoPlacementErrorOnFragment(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx := context.Background()

	fw, err := NewFileWriterWithOptions(tempDir, "test.html", WithAppendMode())
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	// Create initial file with marker
	filePath := filepath.Join(tempDir, "test.html")
	initialContent := "<!DOCTYPE html>\n<html><body>\n<!-- go-output-append -->\n</body></html>"
	err = os.WriteFile(filePath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Append fragment (should NOT have html/body tags)
	fragment := "<p>Just content</p>"
	err = fw.Write(ctx, FormatHTML, []byte(fragment))
	if err != nil {
		t.Fatalf("failed to append: %v", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	// Verify fragment is there
	if !bytes.Contains(content, []byte(fragment)) {
		t.Errorf("fragment not found")
	}

	// Verify original structure is intact
	if !bytes.Contains(content, []byte("<!DOCTYPE html>")) {
		t.Errorf("DOCTYPE removed")
	}
}

func TestFileWriterHTMLRendering_MultipleAppends(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx := context.Background()

	fw, err := NewFileWriterWithOptions(tempDir, "test.html", WithAppendMode())
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	filePath := filepath.Join(tempDir, "test.html")

	// First write: full page with marker
	fullPage := "<!DOCTYPE html>\n<html><body>\n<!-- go-output-append -->\n</body></html>"
	err = fw.Write(ctx, FormatHTML, []byte(fullPage))
	if err != nil {
		t.Fatalf("first write failed: %v", err)
	}

	// Second write: fragment
	fragment1 := "<p>Fragment 1</p>"
	err = fw.Write(ctx, FormatHTML, []byte(fragment1))
	if err != nil {
		t.Fatalf("second write failed: %v", err)
	}

	// Third write: another fragment
	fragment2 := "<p>Fragment 2</p>"
	err = fw.Write(ctx, FormatHTML, []byte(fragment2))
	if err != nil {
		t.Fatalf("third write failed: %v", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	// Verify all fragments are present
	if !bytes.Contains(content, []byte(fragment1)) {
		t.Errorf("fragment 1 not found")
	}

	if !bytes.Contains(content, []byte(fragment2)) {
		t.Errorf("fragment 2 not found")
	}

	// Verify marker is still present (only once at the end)
	markerCount := bytes.Count(content, []byte(HTMLAppendMarker))
	if markerCount != 1 {
		t.Errorf("expected 1 marker, found %d", markerCount)
	}

	// Verify fragment order
	idx1 := bytes.Index(content, []byte(fragment1))
	idx2 := bytes.Index(content, []byte(fragment2))
	markerIdx := bytes.Index(content, []byte(HTMLAppendMarker))

	if !(idx1 < idx2 && idx2 < markerIdx) {
		t.Errorf("fragments not in correct order before marker")
	}
}

func TestFileWriterHTMLRendering_ValidatesHTMLStructure(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx := context.Background()

	tests := map[string]struct {
		initialContent string
		fragment       string
		wantErr        bool
	}{
		"valid HTML with marker": {
			initialContent: "<!DOCTYPE html>\n<html><body>\n<!-- go-output-append -->\n</body></html>",
			fragment:       "<p>Content</p>",
			wantErr:        false,
		},
		"missing marker error": {
			initialContent: "<!DOCTYPE html>\n<html><body>\n<p>No marker</p>\n</body></html>",
			fragment:       "<p>Content</p>",
			wantErr:        true,
		},
		"marker at different position": {
			initialContent: "<!DOCTYPE html>\n<!-- go-output-append -->\n<html><body></body></html>",
			fragment:       "<p>Content</p>",
			wantErr:        false, // Should find it even if not at end
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fw, err := NewFileWriterWithOptions(tempDir, name+".html", WithAppendMode())
			if err != nil {
				t.Fatalf("failed to create FileWriter: %v", err)
			}

			filePath := filepath.Join(tempDir, name+".html")
			err = os.WriteFile(filePath, []byte(tc.initialContent), 0644)
			if err != nil {
				t.Fatalf("failed to create file: %v", err)
			}

			err = fw.Write(ctx, FormatHTML, []byte(tc.fragment))

			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}

			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestFileWriterHTMLRendering_FragmentWithoutPageStructure(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx := context.Background()

	fw, err := NewFileWriterWithOptions(tempDir, "test.html", WithAppendMode())
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	filePath := filepath.Join(tempDir, "test.html")

	// Create initial full HTML page
	fullPage := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Test</title>
</head>
<body>
<h1>Original</h1>
<!-- go-output-append -->
</body>
</html>`

	err = os.WriteFile(filePath, []byte(fullPage), 0644)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Append fragment with multiple elements
	fragment := `<section>
  <h2>Section</h2>
  <p>Content</p>
  <ul>
    <li>Item 1</li>
    <li>Item 2</li>
  </ul>
</section>`

	err = fw.Write(ctx, FormatHTML, []byte(fragment))
	if err != nil {
		t.Fatalf("failed to append: %v", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	// Verify fragment is present with no duplication of page structure
	if !bytes.Contains(content, []byte("<h1>Original</h1>")) {
		t.Errorf("original content missing")
	}

	if !bytes.Contains(content, []byte("<h2>Section</h2>")) {
		t.Errorf("fragment content missing")
	}

	// Should only have one DOCTYPE and one html tag (from original)
	docTypeCount := bytes.Count(content, []byte("<!DOCTYPE"))
	htmlStartCount := bytes.Count(content, []byte("<html"))
	htmlEndCount := bytes.Count(content, []byte("</html>"))

	if docTypeCount != 1 {
		t.Errorf("expected 1 DOCTYPE, found %d", docTypeCount)
	}

	if htmlStartCount != 1 {
		t.Errorf("expected 1 <html tag, found %d", htmlStartCount)
	}

	if htmlEndCount != 1 {
		t.Errorf("expected 1 </html> tag, found %d", htmlEndCount)
	}
}

func TestFileWriterHTMLRendering_ConsecutiveFragments(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx := context.Background()

	fw, err := NewFileWriterWithOptions(tempDir, "test.html", WithAppendMode())
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	filePath := filepath.Join(tempDir, "test.html")

	// Create initial full HTML
	fullPage := "<!DOCTYPE html>\n<html><body>\n<!-- go-output-append -->\n</body></html>"
	err = os.WriteFile(filePath, []byte(fullPage), 0644)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Append multiple fragments consecutively
	fragments := []string{
		"<p>First fragment</p>",
		"<p>Second fragment</p>",
		"<p>Third fragment</p>",
	}

	for _, frag := range fragments {
		err := fw.Write(ctx, FormatHTML, []byte(frag))
		if err != nil {
			t.Fatalf("failed to append fragment: %v", err)
		}
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	// All fragments should be present
	for i, frag := range fragments {
		if !bytes.Contains(content, []byte(frag)) {
			t.Errorf("fragment %d missing: %s", i+1, frag)
		}
	}

	// Marker should be present exactly once
	markerCount := bytes.Count(content, []byte(HTMLAppendMarker))
	if markerCount != 1 {
		t.Errorf("expected 1 marker, found %d", markerCount)
	}

	// All fragments should be before marker
	for _, frag := range fragments {
		fragIdx := bytes.Index(content, []byte(frag))
		markerIdx := bytes.Index(content, []byte(HTMLAppendMarker))
		if fragIdx > markerIdx {
			t.Errorf("fragment %q should be before marker", frag)
		}
	}
}
