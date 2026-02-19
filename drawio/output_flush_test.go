package drawio

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

// TestCreateCSV_BufferFlush tests that CSV output is properly flushed to file
// This test demonstrates the potential buffering issue
func TestCreateCSV_BufferFlush(t *testing.T) {
	// Create minimal test data
	header := DefaultHeader()
	headerRow := []string{"Name"}
	contents := []map[string]string{
		{"Name": "Test"},
	}

	filename := "test_flush.csv"
	defer os.Remove(filename) // Clean up

	// Call CreateCSV
	CreateCSV(header, headerRow, contents, filename)

	// Immediately read the file to check if data was flushed
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Verify the file contains expected content
	contentStr := string(content)
	if contentStr == "" {
		t.Error("File is empty - buffer was not flushed")
	}

	// Check for header content
	if !strings.Contains(contentStr, "# label: %Name%") {
		t.Error("Header content missing - buffer was not flushed")
	}

	// Check for CSV data
	if !strings.Contains(contentStr, "Name") {
		t.Error("CSV header missing - buffer was not flushed")
	}

	if !strings.Contains(contentStr, "Test") {
		t.Error("CSV data missing - buffer was not flushed")
	}
}

// TestCreateCSV_BufferFlushManual demonstrates the exact issue by manually testing buffer behavior
func TestCreateCSV_BufferFlushManual(t *testing.T) {
	// This test demonstrates what happens when we don't flush the bufio.Writer
	filename := "test_manual_flush.csv"
	defer os.Remove(filename)

	// Create file and buffered writer (simulating the issue)
	file, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}

	bufferedWriter := bufio.NewWriter(file)

	// Write some data
	_, err = bufferedWriter.WriteString("test data")
	if err != nil {
		t.Fatal(err)
	}

	// Close file WITHOUT flushing the buffer (this simulates the bug)
	file.Close()

	// Try to read the file - it should be empty or incomplete
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}

	// This demonstrates the issue - data might not be written
	t.Logf("Content without flush: %q (length: %d)", string(content), len(content))

	// Now test with proper flushing
	filename2 := "test_manual_flush2.csv"
	defer os.Remove(filename2)

	file2, err := os.Create(filename2)
	if err != nil {
		t.Fatal(err)
	}

	bufferedWriter2 := bufio.NewWriter(file2)

	// Write the same data
	_, err = bufferedWriter2.WriteString("test data")
	if err != nil {
		t.Fatal(err)
	}

	// Properly flush before closing
	err = bufferedWriter2.Flush()
	if err != nil {
		t.Fatal(err)
	}
	file2.Close()

	// Read the file - it should contain the data
	content2, err := os.ReadFile(filename2)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Content with flush: %q (length: %d)", string(content2), len(content2))

	// The flushed version should have the data
	if string(content2) != "test data" {
		t.Error("Flushed content should contain 'test data'")
	}
}
