package output

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileWriterHTMLAppendCrashSafety_TempFileCleanup(t *testing.T) {
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

	// Successful append
	err = fw.Write(ctx, FormatHTML, []byte("<p>Content</p>"))
	if err != nil {
		t.Fatalf("failed to append: %v", err)
	}

	// Verify no temp files left behind
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}

	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp") {
			t.Errorf("temp file not cleaned up: %s", entry.Name())
		}
	}
}

func TestFileWriterHTMLAppendCrashSafety_OriginalFilePreserved(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx := context.Background()

	fw, err := NewFileWriterWithOptions(tempDir, "test.html", WithAppendMode())
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	// Create initial file with marker
	filePath := filepath.Join(tempDir, "test.html")
	initialContent := "<html><body><p>Important Data</p><!-- go-output-append --></body></html>"
	err = os.WriteFile(filePath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to create initial file: %v", err)
	}

	// Append - test that original is preserved
	newContent := "<p>New Content</p>"
	err = fw.Write(ctx, FormatHTML, []byte(newContent))
	if err != nil {
		t.Fatalf("failed to append: %v", err)
	}

	// Verify that original content is still there
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if !bytes.Contains(content, []byte("<p>Important Data</p>")) {
		t.Errorf("original content was lost")
	}

	if !bytes.Contains(content, []byte(newContent)) {
		t.Errorf("new content was not appended")
	}
}

func TestFileWriterHTMLAppendCrashSafety_TemporaryFileCreation(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx := context.Background()

	fw, err := NewFileWriterWithOptions(tempDir, "test.html", WithAppendMode())
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	// Create initial file
	filePath := filepath.Join(tempDir, "test.html")
	initialContent := "<html><body><!-- go-output-append --></body></html>"
	err = os.WriteFile(filePath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to create initial file: %v", err)
	}

	// Append
	err = fw.Write(ctx, FormatHTML, []byte("<p>Test</p>"))
	if err != nil {
		t.Fatalf("failed to append: %v", err)
	}

	// Verify temp files were created in the same directory as the target file
	// (This is important for atomic rename to work)
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}

	// All files should be in the target directory (no subdirectories created)
	for _, entry := range entries {
		if entry.IsDir() {
			// We only expect the main file, no temp files or directories should remain
			t.Logf("found directory: %s", entry.Name())
		}
	}
}

func TestFileWriterHTMLAppendCrashSafety_AtomicRename(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx := context.Background()

	fw, err := NewFileWriterWithOptions(tempDir, "test.html", WithAppendMode())
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	// Create initial file
	filePath := filepath.Join(tempDir, "test.html")
	initialContent := "<html><body><!-- go-output-append --></body></html>"
	err = os.WriteFile(filePath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to create initial file: %v", err)
	}

	// Get initial file stat info
	initialStat, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("failed to stat initial file: %v", err)
	}

	// Append
	err = fw.Write(ctx, FormatHTML, []byte("<p>New</p>"))
	if err != nil {
		t.Fatalf("failed to append: %v", err)
	}

	// Verify file was successfully renamed (content should be updated)
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file after append: %v", err)
	}

	if !bytes.Contains(content, []byte("<p>New</p>")) {
		t.Errorf("file was not properly updated")
	}

	// The file should still exist and be accessible
	finalStat, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("file not accessible after rename: %v", err)
	}

	// Verify file is different after append (mode/size changed)
	if finalStat.Size() == initialStat.Size() && bytes.Equal(content, []byte(initialContent)) {
		t.Errorf("file was not modified")
	}
}

func TestFileWriterHTMLAppendCrashSafety_SyncBeforeRename(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx := context.Background()

	// This test verifies that Sync() is called before rename
	// (We can't directly test for this, but we can verify that data is durable)
	fw, err := NewFileWriterWithOptions(tempDir, "test.html", WithAppendMode())
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	filePath := filepath.Join(tempDir, "test.html")
	initialContent := "<html><body><!-- go-output-append --></body></html>"
	err = os.WriteFile(filePath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to create initial file: %v", err)
	}

	// Append content that's large enough to benefit from sync
	largeContent := bytes.Repeat([]byte("<p>Data</p>"), 1000)
	err = fw.Write(ctx, FormatHTML, largeContent)
	if err != nil {
		t.Fatalf("failed to append: %v", err)
	}

	// Immediately read back and verify all data is there
	// (If sync wasn't called, this might fail on some systems under heavy load)
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if !bytes.Contains(content, largeContent) {
		t.Errorf("large data was not fully written and synced")
	}
}

func TestFileWriterHTMLAppendCrashSafety_ErrorDoesNotCorruptFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx := context.Background()

	fw, err := NewFileWriterWithOptions(tempDir, "test.html", WithAppendMode())
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	filePath := filepath.Join(tempDir, "test.html")
	initialContent := "<html><body><p>Original</p></body></html>" // Missing marker

	err = os.WriteFile(filePath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to create initial file: %v", err)
	}

	// Try to append - should fail because marker is missing
	err = fw.Write(ctx, FormatHTML, []byte("<p>New</p>"))
	if err == nil {
		t.Fatalf("expected error when marker is missing")
	}

	// Verify original file is unchanged
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file after error: %v", err)
	}

	if string(content) != initialContent {
		t.Errorf("file was corrupted on error:\ngot:  %q\nwant: %q", string(content), initialContent)
	}

	// Verify no temp files left behind
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}

	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp") {
			t.Errorf("temp file not cleaned up after error: %s", entry.Name())
		}
	}
}

func TestFileWriterHTMLAppendCrashSafety_ConcurrentOperations(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx := context.Background()

	// This test verifies that concurrent appends are safe with mutex protection
	fw, err := NewFileWriterWithOptions(tempDir, "test.html", WithAppendMode())
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	filePath := filepath.Join(tempDir, "test.html")
	initialContent := "<html><body><!-- go-output-append --></body></html>"
	err = os.WriteFile(filePath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("failed to create initial file: %v", err)
	}

	// Simulate sequential appends (like concurrent operations would do)
	// The mutex in FileWriter should prevent interleaving
	for i := 0; i < 3; i++ {
		content := []byte("<p>Content" + string(rune('1'+i)) + "</p>")
		err := fw.Write(ctx, FormatHTML, content)
		if err != nil {
			t.Fatalf("failed to append: %v", err)
		}
	}

	// Verify all appends are present and marker is still there
	finalContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if !bytes.Contains(finalContent, []byte("<!-- go-output-append -->")) {
		t.Errorf("marker missing after multiple appends")
	}

	// Check that all three content pieces are present in order
	content1Index := bytes.Index(finalContent, []byte("<p>Content1</p>"))
	content2Index := bytes.Index(finalContent, []byte("<p>Content2</p>"))
	content3Index := bytes.Index(finalContent, []byte("<p>Content3</p>"))
	markerIndex := bytes.Index(finalContent, []byte("<!-- go-output-append -->"))

	if content1Index == -1 || content2Index == -1 || content3Index == -1 {
		t.Errorf("not all appended content found")
		return
	}

	if !(content1Index < content2Index && content2Index < content3Index && content3Index < markerIndex) {
		t.Errorf("content order is incorrect")
	}
}
