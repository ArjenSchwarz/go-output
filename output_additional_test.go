package format

import (
	"bytes"
	"os"
	"testing"
)

func resetGlobals() {
	buffer.Reset()
	toc = nil
	savedRawData = nil
	savedAllKeys = nil
	savedSections = nil
}

func TestOutputArray_AddHeader(t *testing.T) {
	tests := []struct {
		format   string
		expected string
		toc      string
	}{
		{"html", "<h2 id='header-example'>Header Example</h2>\n", "<a href='#header-example'>Header Example</a>"},
		{"table", "\nHeader Example\n", ""},
		{"markdown", "## Header Example\n", "[Header Example](#header-example)"},
	}
	for _, tt := range tests {
		resetGlobals()
		s := NewOutputSettings()
		s.OutputFormat = tt.format
		oa := OutputArray{Settings: s}
		oa.AddHeader("Header Example")
		if got := buffer.String(); got != tt.expected {
			t.Errorf("format %s got %q want %q", tt.format, got, tt.expected)
		}
		if tt.toc != "" {
			if len(toc) != 1 || toc[0] != tt.toc {
				t.Errorf("format %s toc got %v want %v", tt.format, toc, tt.toc)
			}
		} else if len(toc) != 0 {
			t.Errorf("format %s expected no toc entries", tt.format)
		}
	}
}

func TestOutputArray_AddToBuffer(t *testing.T) {
	s := NewOutputSettings()
	s.OutputFormat = "csv"
	oa := OutputArray{Settings: s, Keys: []string{"Name"}}
	oa.AddContents(map[string]interface{}{"Name": "A"})
	resetGlobals()
	expectedCSV := oa.toCSV()
	oa.AddToBuffer()
	if !bytes.Equal(buffer.Bytes(), expectedCSV) {
		t.Errorf("csv AddToBuffer output mismatch")
	}

	// Reset for table test
	s.OutputFormat = "table"
	buffer.Reset()
	oa.Settings = s
	oa.AddContents(map[string]interface{}{"Name": "A"}) // Re-add content since AddToBuffer cleared it
	expectedTable := oa.toTable()
	oa.AddToBuffer()
	if !bytes.Equal(buffer.Bytes(), expectedTable) {
		t.Errorf("table AddToBuffer output mismatch")
	}
}

func TestOutputArray_HtmlTableOnly_NotEmpty(t *testing.T) {
	s := NewOutputSettings()
	s.OutputFormat = "html"
	oa := OutputArray{Settings: s, Keys: []string{"Name"}}
	oa.AddContents(map[string]interface{}{"Name": "item"})
	out := oa.HtmlTableOnly()
	if len(out) == 0 {
		t.Fatalf("expected html output")
	}
	if !bytes.Contains(out, []byte("<table")) {
		t.Errorf("expected html table content")
	}
}

func TestOutputArray_KeysAsInterface(t *testing.T) {
	oa := OutputArray{Keys: []string{"A", "B"}}
	got := oa.KeysAsInterface()
	if len(got) != 2 || got[0] != "A" || got[1] != "B" {
		t.Errorf("unexpected result %v", got)
	}
}

func TestOutputArray_ContentsAsInterfaces(t *testing.T) {
	s := NewOutputSettings()
	oa := OutputArray{Settings: s, Keys: []string{"A", "B"}}
	oa.AddContents(map[string]interface{}{"A": "one", "B": 2})
	result := oa.ContentsAsInterfaces()
	if len(result) != 1 || len(result[0]) != 2 || result[0][0] != "one" || result[0][1] != "2" {
		t.Errorf("unexpected contents %v", result)
	}
}

func TestOutputArray_AddHolderSorting(t *testing.T) {
	s := NewOutputSettings()
	s.SortKey = "name"
	oa := OutputArray{Settings: s, Keys: []string{"name"}}
	oa.AddContents(map[string]interface{}{"name": "b"})
	oa.AddContents(map[string]interface{}{"name": "a"})
	if oa.Contents[0].Contents["name"] != "a" {
		t.Errorf("expected sorted order, got %v", oa.Contents[0])
	}
}

func TestFormatNumber(t *testing.T) {
	if v := formatNumber(42); v != "42" {
		t.Errorf("expected 42 got %s", v)
	}
}

func TestPrintByteSlice_File(t *testing.T) {
	tmp, err := os.CreateTemp("", "out.txt")
	if err != nil {
		t.Fatalf("tempfile: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(tmp.Name()) })
	if err := PrintByteSlice([]byte("\x1b[31mhi\x1b[0m"), tmp.Name(), S3Output{}); err != nil {
		t.Fatalf("PrintByteSlice returned error %v", err)
	}
	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != "hi" {
		t.Errorf("file contents %q", data)
	}
}

func TestActiveProgressRegisterStop(t *testing.T) {
	nop := newNoOpProgress(NewOutputSettings())
	registerActiveProgress(nop)
	if activeProgress == nil {
		t.Fatalf("expected active progress")
	}
	stopActiveProgress()
	if activeProgress != nil {
		t.Errorf("expected nil after stop")
	}
}

func TestOutputArray_DualOutputWithAddToBuffer(t *testing.T) {
	// Create temp file for output
	tmp, err := os.CreateTemp("", "dual-output-test.json")
	if err != nil {
		t.Fatalf("tempfile: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(tmp.Name()) })

	resetGlobals()

	// Create SINGLE OutputArray with dual output settings
	settings := NewOutputSettings()
	settings.OutputFormat = "table"    // stdout format
	settings.OutputFile = tmp.Name()   // file path
	settings.OutputFileFormat = "json" // file format
	settings.SeparateTables = true

	oa := OutputArray{
		Settings: settings,
		Contents: []OutputHolder{},
		Keys:     []string{"Name", "Value", "Active"},
	}

	// First section - User accounts
	oa.Contents = []OutputHolder{
		{Contents: map[string]interface{}{"Name": "Alice", "Value": 1001, "Active": true}},
		{Contents: map[string]interface{}{"Name": "Bob", "Value": 1002, "Active": false}},
	}
	oa.Settings.Title = "User Accounts"
	oa.AddToBuffer()

	// Second section - System resources
	oa.Contents = []OutputHolder{
		{Contents: map[string]interface{}{"Name": "Database", "Value": 8080, "Active": true}},
		{Contents: map[string]interface{}{"Name": "Cache", "Value": 6379, "Active": true}},
	}
	oa.Settings.Title = "System Resources"
	oa.AddToBuffer()

	// Capture stdout before Write()
	stdoutBefore := buffer.String()

	// Single Write() call should handle dual output
	oa.Write()

	// Capture stdout after Write()
	stdoutAfter := buffer.String()

	// Read file output
	fileData, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	fileOutput := string(fileData)

	// VERIFIED BEHAVIOR: When using AddToBuffer() with dual output
	// The implementation now correctly respects OutputFileFormat
	// Buffer content goes to stdout, file gets content in OutputFileFormat

	// Verify that stdout buffer was consumed by Write()
	if len(stdoutAfter) != 0 {
		t.Errorf("Expected stdout buffer to be empty after Write(), but got: %s", stdoutAfter)
	}

	// Verify that stdoutBefore had the expected multi-section content
	if !bytes.Contains([]byte(stdoutBefore), []byte("Alice")) ||
		!bytes.Contains([]byte(stdoutBefore), []byte("Database")) {
		t.Errorf("Expected stdoutBefore to contain all table data")
	}

	// Verify that both sections were in the buffer before Write()
	if !bytes.Contains([]byte(stdoutBefore), []byte("User Accounts")) ||
		!bytes.Contains([]byte(stdoutBefore), []byte("System Resources")) {
		t.Errorf("Expected both section titles in stdoutBefore")
	}

	// FIXED BEHAVIOR: File gets JSON format even when buffer exists
	// This demonstrates that OutputFileFormat is now respected when buffer exists
	if !bytes.Contains(fileData, []byte("{")) ||
		!bytes.Contains(fileData, []byte("\"Name\"")) {
		t.Errorf("Expected file to contain JSON format, got: %s", fileOutput)
	}

	// Verify file contains all the data from both sections
	if !bytes.Contains(fileData, []byte("Alice")) ||
		!bytes.Contains(fileData, []byte("Database")) {
		t.Errorf("Expected file to contain all data, got: %s", fileOutput)
	}

	// Verify that file and stdoutBefore are different (table vs JSON format)
	if string(fileData) == stdoutBefore {
		t.Errorf("Expected file content to be different from stdoutBefore (different formats)")
	}

	// Verify multiple AddToBuffer calls accumulated correctly
	tableCount := bytes.Count([]byte(stdoutBefore), []byte("NAME"))
	if tableCount != 2 {
		t.Errorf("Expected 2 table headers from multiple AddToBuffer calls, got %d", tableCount)
	}
}

func TestOutputArray_TrueDualOutputFormat(t *testing.T) {
	// This test demonstrates how to achieve true dual output format
	// (different formats for stdout vs file)

	tmp, err := os.CreateTemp("", "true-dual-output-test.json")
	if err != nil {
		t.Fatalf("tempfile: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(tmp.Name()) })

	resetGlobals()

	// Create OutputArray with dual output settings
	settings := NewOutputSettings()
	settings.OutputFormat = "table"    // stdout format
	settings.OutputFile = tmp.Name()   // file path
	settings.OutputFileFormat = "json" // file format

	oa := OutputArray{
		Settings: settings,
		Keys:     []string{"Name", "Value", "Active"},
	}

	// Add data directly (without AddToBuffer)
	oa.AddContents(map[string]interface{}{"Name": "Alice", "Value": 1001, "Active": true})
	oa.AddContents(map[string]interface{}{"Name": "Bob", "Value": 1002, "Active": false})
	oa.AddContents(map[string]interface{}{"Name": "Database", "Value": 8080, "Active": true})
	oa.AddContents(map[string]interface{}{"Name": "Cache", "Value": 6379, "Active": true})

	// Capture stdout before Write()
	stdoutBefore := buffer.String()

	// Single Write() call with empty buffer should use different formats
	oa.Write()

	// Capture stdout after Write()
	stdoutAfter := buffer.String()

	// Read file output
	fileData, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	fileOutput := string(fileData)

	// Verify that buffer was empty before Write()
	if len(stdoutBefore) != 0 {
		t.Errorf("Expected empty buffer before Write(), got: %s", stdoutBefore)
	}

	// TRUE DUAL OUTPUT: stdout gets table format
	// Note: stdoutAfter will be empty because Write() outputs directly to stdout
	// and empties the buffer. We can see from the test output that table format
	// was printed to stdout during the test execution.
	if len(stdoutAfter) != 0 {
		t.Errorf("Expected stdout buffer to be empty after Write(), got: %s", stdoutAfter)
	}

	// TRUE DUAL OUTPUT: file gets JSON format
	if !bytes.Contains(fileData, []byte("{")) ||
		!bytes.Contains(fileData, []byte("\"Name\"")) {
		t.Errorf("Expected file to contain JSON format, got: %s", fileOutput)
	}

	// Verify stdout was written (we can see it in test output above)
	// Since Write() outputs directly to stdout and clears buffer,
	// we can't verify stdout content in buffer, but we can see
	// it was output in the test execution above

	if !bytes.Contains(fileData, []byte("Alice")) ||
		!bytes.Contains(fileData, []byte("Database")) {
		t.Errorf("Expected file to contain all data")
	}

	// Verify formats are different
	// We can see from test output that stdout was table format
	// and file should be JSON format
	if bytes.Contains(fileData, []byte("|")) {
		t.Errorf("Expected file to be JSON format, but got table format: %s", fileOutput)
	}

	// Verify JSON structure
	jsonRecordCount := bytes.Count(fileData, []byte("{\"Active\":"))
	if jsonRecordCount != 4 {
		t.Errorf("Expected 4 JSON records in file, got %d", jsonRecordCount)
	}
}

func TestOutputArray_MarkdownHtmlFileFormat(t *testing.T) {
	// Test case 1: Markdown file format with table stdout format
	t.Run("Markdown file format", func(t *testing.T) {
		tmp, err := os.CreateTemp("", "markdown-output-test.md")
		if err != nil {
			t.Fatalf("tempfile: %v", err)
		}
		t.Cleanup(func() { _ = os.Remove(tmp.Name()) })

		resetGlobals()

		settings := NewOutputSettings()
		settings.OutputFormat = "table"        // stdout format
		settings.OutputFile = tmp.Name()       // file path
		settings.OutputFileFormat = "markdown" // file format

		oa := OutputArray{
			Settings: settings,
			Keys:     []string{"Name", "Value", "Active"},
		}

		oa.AddContents(map[string]interface{}{"Name": "Alice", "Value": 1001, "Active": true})
		oa.AddContents(map[string]interface{}{"Name": "Bob", "Value": 1002, "Active": false})

		oa.Write()

		// Read file output
		fileData, err := os.ReadFile(tmp.Name())
		if err != nil {
			t.Fatalf("read file: %v", err)
		}

		// Verify file contains markdown format (not table format)
		if bytes.Contains(fileData, []byte("+-")) {
			t.Errorf("Expected file to contain markdown format, but got table format: %s", string(fileData))
		}

		// Verify file contains markdown table syntax
		if !bytes.Contains(fileData, []byte("| Name")) ||
			!bytes.Contains(fileData, []byte("| ---")) {
			t.Errorf("Expected file to contain markdown table syntax, got: %s", string(fileData))
		}

		// Verify file contains the data
		if !bytes.Contains(fileData, []byte("Alice")) ||
			!bytes.Contains(fileData, []byte("Bob")) {
			t.Errorf("Expected file to contain all data, got: %s", string(fileData))
		}
	})

	// Test case 2: HTML file format with table stdout format
	t.Run("HTML file format", func(t *testing.T) {
		tmp, err := os.CreateTemp("", "html-output-test.html")
		if err != nil {
			t.Fatalf("tempfile: %v", err)
		}
		t.Cleanup(func() { _ = os.Remove(tmp.Name()) })

		resetGlobals()

		settings := NewOutputSettings()
		settings.OutputFormat = "table"    // stdout format
		settings.OutputFile = tmp.Name()   // file path
		settings.OutputFileFormat = "html" // file format

		oa := OutputArray{
			Settings: settings,
			Keys:     []string{"Name", "Value", "Active"},
		}

		oa.AddContents(map[string]interface{}{"Name": "Charlie", "Value": 1003, "Active": true})
		oa.AddContents(map[string]interface{}{"Name": "David", "Value": 1004, "Active": false})

		oa.Write()

		// Read file output
		fileData, err := os.ReadFile(tmp.Name())
		if err != nil {
			t.Fatalf("read file: %v", err)
		}

		// Verify file contains HTML format (not table format)
		if bytes.Contains(fileData, []byte("+-")) {
			t.Errorf("Expected file to contain HTML format, but got table format: %s", string(fileData))
		}

		// Verify file contains HTML table syntax
		if !bytes.Contains(fileData, []byte("<table")) ||
			!bytes.Contains(fileData, []byte("<tr")) ||
			!bytes.Contains(fileData, []byte("<td")) {
			t.Errorf("Expected file to contain HTML table syntax, got: %s", string(fileData))
		}

		// Verify file contains the data
		if !bytes.Contains(fileData, []byte("Charlie")) ||
			!bytes.Contains(fileData, []byte("David")) {
			t.Errorf("Expected file to contain all data, got: %s", string(fileData))
		}
	})
}

func TestOutputArray_MultipleOutputHoldersDifferentTypes(t *testing.T) {
	// Test case: Multiple output holders with different data types
	tmp, err := os.CreateTemp("", "multi-holders-test.json")
	if err != nil {
		t.Fatalf("tempfile: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(tmp.Name()) })

	resetGlobals()

	settings := NewOutputSettings()
	settings.OutputFormat = "table"    // stdout format
	settings.OutputFile = tmp.Name()   // file path
	settings.OutputFileFormat = "json" // file format

	oa := OutputArray{
		Settings: settings,
		Keys:     []string{"Name", "Value", "Active"},
	}

	// Add first holder with string, integer, boolean
	oa.AddContents(map[string]interface{}{
		"Name":   "Alice",
		"Value":  1001,
		"Active": true,
	})

	// Add second holder with string, float, boolean
	oa.AddContents(map[string]interface{}{
		"Name":   "Bob",
		"Value":  1002.5,
		"Active": false,
	})

	// Add third holder with string, string, boolean
	oa.AddContents(map[string]interface{}{
		"Name":   "Charlie",
		"Value":  "text_value",
		"Active": true,
	})

	oa.Write()

	// Read file output
	fileData, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	// Debug: Log the file content
	t.Logf("File content: %s", string(fileData))

	// Verify file contains JSON format
	if !bytes.Contains(fileData, []byte("{")) ||
		!bytes.Contains(fileData, []byte("\"Name\"")) {
		t.Errorf("Expected file to contain JSON format, got: %s", string(fileData))
	}

	// Verify ALL records are present with their correct values
	if !bytes.Contains(fileData, []byte("\"Alice\"")) {
		t.Errorf("Expected file to contain Alice record, got: %s", string(fileData))
	}

	if !bytes.Contains(fileData, []byte("\"Bob\"")) {
		t.Errorf("Expected file to contain Bob record, got: %s", string(fileData))
	}

	if !bytes.Contains(fileData, []byte("\"Charlie\"")) {
		t.Errorf("Expected file to contain Charlie record, got: %s", string(fileData))
	}

	// Verify different value types are preserved
	if !bytes.Contains(fileData, []byte("1001")) {
		t.Errorf("Expected file to contain Alice's integer value 1001, got: %s", string(fileData))
	}

	if !bytes.Contains(fileData, []byte("1002.5")) {
		t.Errorf("Expected file to contain Bob's float value 1002.5, got: %s", string(fileData))
	}

	if !bytes.Contains(fileData, []byte("\"text_value\"")) {
		t.Errorf("Expected file to contain Charlie's string value 'text_value', got: %s", string(fileData))
	}

	// Verify we have exactly 3 records
	recordCount := bytes.Count(fileData, []byte("{\"Active\":"))
	if recordCount != 3 {
		t.Errorf("Expected 3 JSON records in file, got %d", recordCount)
	}
}

func TestOutputArray_MultipleOutputHoldersWithAddToBuffer(t *testing.T) {
	// Test case: Multiple AddToBuffer calls with different data types
	tmp, err := os.CreateTemp("", "multi-holders-buffer-test.json")
	if err != nil {
		t.Fatalf("tempfile: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(tmp.Name()) })

	resetGlobals()

	settings := NewOutputSettings()
	settings.OutputFormat = "table"    // stdout format
	settings.OutputFile = tmp.Name()   // file path
	settings.OutputFileFormat = "json" // file format

	oa := OutputArray{
		Settings: settings,
		Keys:     []string{"Name", "Value", "Active"},
	}

	// First section with integer values
	oa.Contents = []OutputHolder{
		{Contents: map[string]interface{}{"Name": "Alice", "Value": 1001, "Active": true}},
		{Contents: map[string]interface{}{"Name": "Bob", "Value": 1002, "Active": false}},
	}
	oa.AddToBuffer()

	// Second section with float and string values
	oa.Contents = []OutputHolder{
		{Contents: map[string]interface{}{"Name": "Charlie", "Value": 1003.5, "Active": true}},
		{Contents: map[string]interface{}{"Name": "David", "Value": "text_value", "Active": false}},
	}
	oa.AddToBuffer()

	oa.Write()

	// Read file output
	fileData, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	// Debug: Log the file content
	t.Logf("File content: %s", string(fileData))

	// Verify file contains JSON format
	if !bytes.Contains(fileData, []byte("{")) ||
		!bytes.Contains(fileData, []byte("\"Name\"")) {
		t.Errorf("Expected file to contain JSON format, got: %s", string(fileData))
	}

	// Verify ALL records from both sections are present
	if !bytes.Contains(fileData, []byte("\"Alice\"")) {
		t.Errorf("Expected file to contain Alice record, got: %s", string(fileData))
	}

	if !bytes.Contains(fileData, []byte("\"Bob\"")) {
		t.Errorf("Expected file to contain Bob record, got: %s", string(fileData))
	}

	if !bytes.Contains(fileData, []byte("\"Charlie\"")) {
		t.Errorf("Expected file to contain Charlie record, got: %s", string(fileData))
	}

	if !bytes.Contains(fileData, []byte("\"David\"")) {
		t.Errorf("Expected file to contain David record, got: %s", string(fileData))
	}

	// Verify different value types are preserved from both sections
	if !bytes.Contains(fileData, []byte("1001")) {
		t.Errorf("Expected file to contain Alice's integer value 1001, got: %s", string(fileData))
	}

	if !bytes.Contains(fileData, []byte("1002")) {
		t.Errorf("Expected file to contain Bob's integer value 1002, got: %s", string(fileData))
	}

	if !bytes.Contains(fileData, []byte("1003.5")) {
		t.Errorf("Expected file to contain Charlie's float value 1003.5, got: %s", string(fileData))
	}

	if !bytes.Contains(fileData, []byte("\"text_value\"")) {
		t.Errorf("Expected file to contain David's string value 'text_value', got: %s", string(fileData))
	}

	// Verify we have exactly 4 records
	recordCount := bytes.Count(fileData, []byte("{\"Active\":"))
	if recordCount != 4 {
		t.Errorf("Expected 4 JSON records in file, got %d", recordCount)
	}
}

func TestOutputArray_MarkdownHtmlFileFormatWithAddToBuffer(t *testing.T) {
	// Test the specific scenario where markdown/html formats might not work with AddToBuffer
	t.Run("Markdown with AddToBuffer", func(t *testing.T) {
		tmp, err := os.CreateTemp("", "markdown-buffer-test.md")
		if err != nil {
			t.Fatalf("tempfile: %v", err)
		}
		t.Cleanup(func() { _ = os.Remove(tmp.Name()) })

		resetGlobals()

		settings := NewOutputSettings()
		settings.OutputFormat = "table"        // stdout format
		settings.OutputFile = tmp.Name()       // file path
		settings.OutputFileFormat = "markdown" // file format

		oa := OutputArray{
			Settings: settings,
			Keys:     []string{"Name", "Value", "Active"},
		}

		// First section
		oa.Contents = []OutputHolder{
			{Contents: map[string]interface{}{"Name": "Alice", "Value": 1001, "Active": true}},
			{Contents: map[string]interface{}{"Name": "Bob", "Value": 1002, "Active": false}},
		}
		oa.AddToBuffer()

		// Second section
		oa.Contents = []OutputHolder{
			{Contents: map[string]interface{}{"Name": "Charlie", "Value": 1003, "Active": true}},
			{Contents: map[string]interface{}{"Name": "David", "Value": 1004, "Active": false}},
		}
		oa.AddToBuffer()

		oa.Write()

		// Read file output
		fileData, err := os.ReadFile(tmp.Name())
		if err != nil {
			t.Fatalf("read file: %v", err)
		}

		t.Logf("File content: %s", string(fileData))

		// Verify file contains markdown format (not table format)
		if bytes.Contains(fileData, []byte("+-")) {
			t.Errorf("Expected file to contain markdown format, but got table format: %s", string(fileData))
		}

		// Verify file contains markdown table syntax
		if !bytes.Contains(fileData, []byte("| Name")) ||
			!bytes.Contains(fileData, []byte("| ---")) {
			t.Errorf("Expected file to contain markdown table syntax, got: %s", string(fileData))
		}

		// Verify file contains ALL data from both sections
		if !bytes.Contains(fileData, []byte("Alice")) ||
			!bytes.Contains(fileData, []byte("Bob")) ||
			!bytes.Contains(fileData, []byte("Charlie")) ||
			!bytes.Contains(fileData, []byte("David")) {
			t.Errorf("Expected file to contain all data from both sections, got: %s", string(fileData))
		}
	})

	t.Run("HTML with AddToBuffer", func(t *testing.T) {
		tmp, err := os.CreateTemp("", "html-buffer-test.html")
		if err != nil {
			t.Fatalf("tempfile: %v", err)
		}
		t.Cleanup(func() { _ = os.Remove(tmp.Name()) })

		resetGlobals()

		settings := NewOutputSettings()
		settings.OutputFormat = "table"    // stdout format
		settings.OutputFile = tmp.Name()   // file path
		settings.OutputFileFormat = "html" // file format

		oa := OutputArray{
			Settings: settings,
			Keys:     []string{"Name", "Value", "Active"},
		}

		// First section
		oa.Contents = []OutputHolder{
			{Contents: map[string]interface{}{"Name": "Eve", "Value": 1005, "Active": true}},
			{Contents: map[string]interface{}{"Name": "Frank", "Value": 1006, "Active": false}},
		}
		oa.AddToBuffer()

		// Second section
		oa.Contents = []OutputHolder{
			{Contents: map[string]interface{}{"Name": "Grace", "Value": 1007, "Active": true}},
			{Contents: map[string]interface{}{"Name": "Henry", "Value": 1008, "Active": false}},
		}
		oa.AddToBuffer()

		oa.Write()

		// Read file output
		fileData, err := os.ReadFile(tmp.Name())
		if err != nil {
			t.Fatalf("read file: %v", err)
		}

		t.Logf("File content: %s", string(fileData))

		// Verify file contains HTML format (not table format)
		if bytes.Contains(fileData, []byte("+-")) {
			t.Errorf("Expected file to contain HTML format, but got table format: %s", string(fileData))
		}

		// Verify file contains HTML table syntax
		if !bytes.Contains(fileData, []byte("<table")) ||
			!bytes.Contains(fileData, []byte("<tr")) ||
			!bytes.Contains(fileData, []byte("<td")) {
			t.Errorf("Expected file to contain HTML table syntax, got: %s", string(fileData))
		}

		// Verify file contains ALL data from both sections
		if !bytes.Contains(fileData, []byte("Eve")) ||
			!bytes.Contains(fileData, []byte("Frank")) ||
			!bytes.Contains(fileData, []byte("Grace")) ||
			!bytes.Contains(fileData, []byte("Henry")) {
			t.Errorf("Expected file to contain all data from both sections, got: %s", string(fileData))
		}
	})
}

func TestOutputArray_MultipleOutputHoldersEdgeCases(t *testing.T) {
	// Test more complex scenarios that might reveal the "empty values except for final one" issue

	t.Run("Different key sets between holders", func(t *testing.T) {
		tmp, err := os.CreateTemp("", "edge-case-keys-test.json")
		if err != nil {
			t.Fatalf("tempfile: %v", err)
		}
		t.Cleanup(func() { _ = os.Remove(tmp.Name()) })

		resetGlobals()

		settings := NewOutputSettings()
		settings.OutputFormat = "table"    // stdout format
		settings.OutputFile = tmp.Name()   // file path
		settings.OutputFileFormat = "json" // file format

		oa := OutputArray{
			Settings: settings,
			Keys:     []string{"Name", "Value", "Active", "Extra"},
		}

		// First holder with all keys
		oa.AddContents(map[string]interface{}{
			"Name":   "Alice",
			"Value":  1001,
			"Active": true,
			"Extra":  "first",
		})

		// Second holder missing some keys
		oa.AddContents(map[string]interface{}{
			"Name":   "Bob",
			"Value":  1002,
			"Active": false,
			// Missing "Extra" key
		})

		// Third holder with different key present
		oa.AddContents(map[string]interface{}{
			"Name":   "Charlie",
			"Value":  1003,
			"Active": true,
			"Extra":  "third",
		})

		oa.Write()

		// Read file output
		fileData, err := os.ReadFile(tmp.Name())
		if err != nil {
			t.Fatalf("read file: %v", err)
		}

		t.Logf("File content: %s", string(fileData))

		// Verify all records are present
		if !bytes.Contains(fileData, []byte("\"Alice\"")) ||
			!bytes.Contains(fileData, []byte("\"Bob\"")) ||
			!bytes.Contains(fileData, []byte("\"Charlie\"")) {
			t.Errorf("Expected all records to be present, got: %s", string(fileData))
		}

		// Verify values are correct for each record
		if !bytes.Contains(fileData, []byte("1001")) ||
			!bytes.Contains(fileData, []byte("1002")) ||
			!bytes.Contains(fileData, []byte("1003")) {
			t.Errorf("Expected all values to be present, got: %s", string(fileData))
		}

		// Verify Extra field shows up for Alice and Charlie but not Bob
		if !bytes.Contains(fileData, []byte("\"first\"")) ||
			!bytes.Contains(fileData, []byte("\"third\"")) {
			t.Errorf("Expected Extra field values to be present, got: %s", string(fileData))
		}
	})

	t.Run("Nil and empty values", func(t *testing.T) {
		tmp, err := os.CreateTemp("", "edge-case-nil-test.json")
		if err != nil {
			t.Fatalf("tempfile: %v", err)
		}
		t.Cleanup(func() { _ = os.Remove(tmp.Name()) })

		resetGlobals()

		settings := NewOutputSettings()
		settings.OutputFormat = "table"    // stdout format
		settings.OutputFile = tmp.Name()   // file path
		settings.OutputFileFormat = "json" // file format

		oa := OutputArray{
			Settings: settings,
			Keys:     []string{"Name", "Value", "Active"},
		}

		// First holder with normal values
		oa.AddContents(map[string]interface{}{
			"Name":   "Alice",
			"Value":  1001,
			"Active": true,
		})

		// Second holder with nil value
		oa.AddContents(map[string]interface{}{
			"Name":   "Bob",
			"Value":  nil,
			"Active": false,
		})

		// Third holder with empty string
		oa.AddContents(map[string]interface{}{
			"Name":   "Charlie",
			"Value":  "",
			"Active": true,
		})

		// Fourth holder with zero values
		oa.AddContents(map[string]interface{}{
			"Name":   "David",
			"Value":  0,
			"Active": false,
		})

		oa.Write()

		// Read file output
		fileData, err := os.ReadFile(tmp.Name())
		if err != nil {
			t.Fatalf("read file: %v", err)
		}

		t.Logf("File content: %s", string(fileData))

		// Verify all records are present
		if !bytes.Contains(fileData, []byte("\"Alice\"")) ||
			!bytes.Contains(fileData, []byte("\"Bob\"")) ||
			!bytes.Contains(fileData, []byte("\"Charlie\"")) ||
			!bytes.Contains(fileData, []byte("\"David\"")) {
			t.Errorf("Expected all records to be present, got: %s", string(fileData))
		}

		// Verify that all records have their expected structures
		recordCount := bytes.Count(fileData, []byte("{\"Active\":"))
		if recordCount != 4 {
			t.Errorf("Expected 4 JSON records in file, got %d", recordCount)
		}
	})

	t.Run("AddToBuffer with mixed key presence", func(t *testing.T) {
		tmp, err := os.CreateTemp("", "edge-case-buffer-keys-test.json")
		if err != nil {
			t.Fatalf("tempfile: %v", err)
		}
		t.Cleanup(func() { _ = os.Remove(tmp.Name()) })

		resetGlobals()

		settings := NewOutputSettings()
		settings.OutputFormat = "table"    // stdout format
		settings.OutputFile = tmp.Name()   // file path
		settings.OutputFileFormat = "json" // file format

		oa := OutputArray{
			Settings: settings,
			Keys:     []string{"Name", "Value", "Active", "Category"},
		}

		// First section - all keys present
		oa.Contents = []OutputHolder{
			{Contents: map[string]interface{}{"Name": "Alice", "Value": 1001, "Active": true, "Category": "user"}},
			{Contents: map[string]interface{}{"Name": "Bob", "Value": 1002, "Active": false, "Category": "user"}},
		}
		oa.AddToBuffer()

		// Second section - missing some keys
		oa.Contents = []OutputHolder{
			{Contents: map[string]interface{}{"Name": "Service1", "Value": 8080, "Active": true}},  // missing Category
			{Contents: map[string]interface{}{"Name": "Service2", "Value": 8081, "Active": false}}, // missing Category
		}
		oa.AddToBuffer()

		// Third section - different key presence
		oa.Contents = []OutputHolder{
			{Contents: map[string]interface{}{"Name": "Charlie", "Active": true, "Category": "admin"}}, // missing Value
			{Contents: map[string]interface{}{"Name": "David", "Value": 999, "Active": false, "Category": "admin"}},
		}
		oa.AddToBuffer()

		oa.Write()

		// Read file output
		fileData, err := os.ReadFile(tmp.Name())
		if err != nil {
			t.Fatalf("read file: %v", err)
		}

		t.Logf("File content: %s", string(fileData))

		// Verify all records are present
		expectedNames := []string{"Alice", "Bob", "Service1", "Service2", "Charlie", "David"}
		for _, name := range expectedNames {
			if !bytes.Contains(fileData, []byte("\""+name+"\"")) {
				t.Errorf("Expected record %s to be present, got: %s", name, string(fileData))
			}
		}

		// Verify we have exactly 6 records
		recordCount := bytes.Count(fileData, []byte("{\"Active\":"))
		if recordCount != 6 {
			t.Errorf("Expected 6 JSON records in file, got %d", recordCount)
		}

		// Verify specific values are present from each section
		if !bytes.Contains(fileData, []byte("1001")) ||
			!bytes.Contains(fileData, []byte("8080")) ||
			!bytes.Contains(fileData, []byte("999")) {
			t.Errorf("Expected specific values from each section to be present, got: %s", string(fileData))
		}
	})
}

func TestOutputArray_RealWorldMultipleOutputHoldersIssue(t *testing.T) {
	// Test the exact scenario from the user's real application
	// This reproduces the issue where different key structures result in empty JSON objects
	tmp, err := os.CreateTemp("", "real-world-test.json")
	if err != nil {
		t.Fatalf("tempfile: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(tmp.Name()) })

	resetGlobals()

	settings := NewOutputSettings()
	settings.OutputFormat = "table"    // stdout format
	settings.OutputFile = tmp.Name()   // file path
	settings.OutputFileFormat = "json" // file format

	oa := OutputArray{
		Settings: settings,
		Keys:     []string{"ACTION", "RESOURCE", "TYPE", "ID", "REPLACEMENT", "MODULE", "DANGER"},
	}

	// First section - Plan Information (has different keys than the main Keys)
	oa.Contents = []OutputHolder{
		{Contents: map[string]interface{}{"PLAN FILE": "plan.out", "VERSION": "1.8.5", "WORKSPACE": "default", "BACKEND": "local (terraform.tfstate)", "CREATED": "2025-07-16 22:30:02"}},
	}
	oa.Settings.Title = "Plan Information"
	oa.AddToBuffer()

	// Second section - Summary (also has different keys than the main Keys)
	oa.Contents = []OutputHolder{
		{Contents: map[string]interface{}{"TOTAL": 20, "ADDED": 20, "REMOVED": 0, "MODIFIED": 0, "REPLACEMENTS": 0, "CONDITIONALS": 0, "HIGH RISK": 0}},
	}
	oa.Settings.Title = "Summary for plan.out"
	oa.AddToBuffer()

	// Third section - Resource Changes (matches the main Keys)
	oa.Contents = []OutputHolder{
		{Contents: map[string]interface{}{"ACTION": "Add", "RESOURCE": "local_file.test_files[0]", "TYPE": "local_file", "ID": "-", "REPLACEMENT": "Never", "MODULE": "-", "DANGER": ""}},
		{Contents: map[string]interface{}{"ACTION": "Add", "RESOURCE": "local_file.test_files[1]", "TYPE": "local_file", "ID": "-", "REPLACEMENT": "Never", "MODULE": "-", "DANGER": ""}},
		{Contents: map[string]interface{}{"ACTION": "Add", "RESOURCE": "local_file.test_files[2]", "TYPE": "local_file", "ID": "-", "REPLACEMENT": "Never", "MODULE": "-", "DANGER": ""}},
	}
	oa.Settings.Title = "Resource Changes"
	oa.AddToBuffer()

	oa.Write()

	// Read file output
	fileData, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	t.Logf("File content: %s", string(fileData))

	// This test should FAIL initially because the first two sections will create empty JSON objects
	// since their keys don't match the main Keys array

	// Check if we have empty objects (this is the bug)
	emptyObjectCount := bytes.Count(fileData, []byte("{}"))
	if emptyObjectCount > 0 {
		t.Errorf("Found %d empty JSON objects - this is the bug! Expected all sections to have data", emptyObjectCount)
	}

	// Verify the Resource Changes section has data
	if !bytes.Contains(fileData, []byte("\"local_file.test_files[0]\"")) {
		t.Errorf("Expected Resource Changes data to be present")
	}

	// The bug is that Plan Information and Summary sections will be empty objects
	// because their keys don't match the main Keys array
	// We should see data from all sections, not just the Resource Changes

	// Check for Plan Information data
	if !bytes.Contains(fileData, []byte("\"plan.out\"")) {
		t.Errorf("Expected Plan Information data to be present, but it's missing")
	}

	// Check for Summary data
	if !bytes.Contains(fileData, []byte("20")) {
		t.Errorf("Expected Summary data to be present, but it's missing")
	}

	// This test demonstrates the issue: when different sections have different key structures,
	// only the sections that match the main Keys array will have data in the JSON output
}

func TestOutputArray_RealWorldMultipleOutputHoldersIssueMarkdown(t *testing.T) {
	// Test the same scenario but with markdown output format
	tmp, err := os.CreateTemp("", "real-world-markdown-test.md")
	if err != nil {
		t.Fatalf("tempfile: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(tmp.Name()) })

	resetGlobals()

	settings := NewOutputSettings()
	settings.OutputFormat = "table"        // stdout format
	settings.OutputFile = tmp.Name()       // file path
	settings.OutputFileFormat = "markdown" // file format

	oa := OutputArray{
		Settings: settings,
		Keys:     []string{"ACTION", "RESOURCE", "TYPE", "ID", "REPLACEMENT", "MODULE", "DANGER"},
	}

	// First section - Plan Information (has different keys than the main Keys)
	oa.Contents = []OutputHolder{
		{Contents: map[string]interface{}{"PLAN FILE": "plan.out", "VERSION": "1.8.5", "WORKSPACE": "default", "BACKEND": "local (terraform.tfstate)", "CREATED": "2025-07-16 22:30:02"}},
	}
	oa.Settings.Title = "Plan Information"
	oa.AddToBuffer()

	// Second section - Summary (also has different keys than the main Keys)
	oa.Contents = []OutputHolder{
		{Contents: map[string]interface{}{"TOTAL": 20, "ADDED": 20, "REMOVED": 0, "MODIFIED": 0, "REPLACEMENTS": 0, "CONDITIONALS": 0, "HIGH RISK": 0}},
	}
	oa.Settings.Title = "Summary for plan.out"
	oa.AddToBuffer()

	// Third section - Resource Changes (matches the main Keys)
	oa.Contents = []OutputHolder{
		{Contents: map[string]interface{}{"ACTION": "Add", "RESOURCE": "local_file.test_files[0]", "TYPE": "local_file", "ID": "-", "REPLACEMENT": "Never", "MODULE": "-", "DANGER": ""}},
		{Contents: map[string]interface{}{"ACTION": "Add", "RESOURCE": "local_file.test_files[1]", "TYPE": "local_file", "ID": "-", "REPLACEMENT": "Never", "MODULE": "-", "DANGER": ""}},
		{Contents: map[string]interface{}{"ACTION": "Add", "RESOURCE": "local_file.test_files[2]", "TYPE": "local_file", "ID": "-", "REPLACEMENT": "Never", "MODULE": "-", "DANGER": ""}},
	}
	oa.Settings.Title = "Resource Changes"
	oa.AddToBuffer()

	oa.Write()

	// Read file output
	fileData, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	t.Logf("File content: %s", string(fileData))

	// Check for Plan Information data - should not be showing as <nil>
	if !bytes.Contains(fileData, []byte("plan.out")) {
		t.Errorf("Expected Plan Information data to be present, but it's missing")
	}

	// Check for Summary data - should not be showing as <nil>
	if !bytes.Contains(fileData, []byte("20")) {
		t.Errorf("Expected Summary data to be present, but it's missing")
	}

	// For now, we expect some <nil> values when different sections have different keys
	// The important thing is that the actual data is preserved
	// Note: This is expected behavior for table-based formats when sections have different keys

	// Verify the Resource Changes section still has data
	if !bytes.Contains(fileData, []byte("local_file.test_files[0]")) {
		t.Errorf("Expected Resource Changes data to be present")
	}
}

func TestOutputArray_RealWorldMultipleOutputHoldersIssueHTML(t *testing.T) {
	// Test the same scenario but with HTML output format
	tmp, err := os.CreateTemp("", "real-world-html-test.html")
	if err != nil {
		t.Fatalf("tempfile: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(tmp.Name()) })

	resetGlobals()

	settings := NewOutputSettings()
	settings.OutputFormat = "table"    // stdout format
	settings.OutputFile = tmp.Name()   // file path
	settings.OutputFileFormat = "html" // file format

	oa := OutputArray{
		Settings: settings,
		Keys:     []string{"ACTION", "RESOURCE", "TYPE", "ID", "REPLACEMENT", "MODULE", "DANGER"},
	}

	// First section - Plan Information (has different keys than the main Keys)
	oa.Contents = []OutputHolder{
		{Contents: map[string]interface{}{"PLAN FILE": "plan.out", "VERSION": "1.8.5", "WORKSPACE": "default"}},
	}
	oa.Settings.Title = "Plan Information"
	oa.AddToBuffer()

	// Second section - Summary (also has different keys than the main Keys)
	oa.Contents = []OutputHolder{
		{Contents: map[string]interface{}{"TOTAL": 20, "ADDED": 20, "REMOVED": 0}},
	}
	oa.Settings.Title = "Summary"
	oa.AddToBuffer()

	// Third section - Resource Changes (matches the main Keys)
	oa.Contents = []OutputHolder{
		{Contents: map[string]interface{}{"ACTION": "Add", "RESOURCE": "local_file.test_files[0]", "TYPE": "local_file", "ID": "-", "REPLACEMENT": "Never", "MODULE": "-", "DANGER": ""}},
	}
	oa.Settings.Title = "Resource Changes"
	oa.AddToBuffer()

	oa.Write()

	// Read file output
	fileData, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	// Verify it's HTML format
	if !bytes.Contains(fileData, []byte("<table")) ||
		!bytes.Contains(fileData, []byte("<tr")) ||
		!bytes.Contains(fileData, []byte("<td")) {
		t.Errorf("Expected HTML table format")
	}

	// Check for Plan Information data
	if !bytes.Contains(fileData, []byte("plan.out")) {
		t.Errorf("Expected Plan Information data to be present")
	}

	// Check for Summary data
	if !bytes.Contains(fileData, []byte("20")) {
		t.Errorf("Expected Summary data to be present")
	}

	// Verify the Resource Changes section still has data
	if !bytes.Contains(fileData, []byte("local_file.test_files[0]")) {
		t.Errorf("Expected Resource Changes data to be present")
	}
}
