package drawio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateCSV_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		setupFunc   func(string) error
		expectError bool
		errorMsg    string
	}{
		{
			name:        "invalid file path",
			filename:    "/invalid/path/test.csv",
			setupFunc:   nil,
			expectError: true,
			errorMsg:    "failed to create file",
		},
		{
			name:     "permission denied",
			filename: "/root/test.csv",
			setupFunc: func(filename string) error {
				// Try to create a file in a restricted directory
				return nil
			},
			expectError: true,
			errorMsg:    "failed to create file",
		},
		{
			name:        "valid file creation",
			filename:    "",
			setupFunc:   nil,
			expectError: false,
			errorMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test data
			header := Header{}
			headerRow := []string{"Name", "Value"}
			contents := []map[string]string{
				{"Name": "test1", "Value": "value1"},
				{"Name": "test2", "Value": "value2"},
			}

			// Setup if needed
			if tt.setupFunc != nil {
				if err := tt.setupFunc(tt.filename); err != nil {
					t.Skipf("Setup failed: %v", err)
				}
			}

			// Test the function
			err := CreateCSV(header, headerRow, contents, tt.filename)

			// Verify results
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}

			// Cleanup
			if tt.filename != "" && tt.filename != "/invalid/path/test.csv" && tt.filename != "/root/test.csv" {
				os.Remove(tt.filename)
			}
		})
	}
}

func TestCreateCSV_ValidFileCreation(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "drawio_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filename := filepath.Join(tempDir, "test.csv")
	header := Header{}
	headerRow := []string{"Name", "Value", "Description"}
	contents := []map[string]string{
		{"Name": "item1", "Value": "100", "Description": "First item"},
		{"Name": "item2", "Value": "200", "Description": "Second item"},
	}

	err = CreateCSV(header, headerRow, contents, filename)
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	// Verify file was created and has content
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Error("Expected file to be created but it doesn't exist")
	}

	// Read and verify content
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Errorf("Failed to read created file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "Name,Value,Description") {
		t.Error("Expected header row in CSV content")
	}
	if !strings.Contains(contentStr, "item1,100,First item") {
		t.Error("Expected first data row in CSV content")
	}
}

func TestGetContentsFromFile_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		fileContent string
		setupFunc   func(string, string) error
		expectError bool
		errorMsg    string
	}{
		{
			name:        "file does not exist",
			filename:    "nonexistent.csv",
			fileContent: "",
			setupFunc:   nil,
			expectError: true,
			errorMsg:    "failed to read file",
		},
		{
			name:        "invalid CSV content",
			filename:    "invalid.csv",
			fileContent: "Name,Value\n\"unclosed quote,value",
			setupFunc: func(filename, content string) error {
				return os.WriteFile(filename, []byte(content), 0644)
			},
			expectError: true,
			errorMsg:    "failed to parse CSV",
		},
		{
			name:        "empty file",
			filename:    "empty.csv",
			fileContent: "",
			setupFunc: func(filename, content string) error {
				return os.WriteFile(filename, []byte(content), 0644)
			},
			expectError: true,
			errorMsg:    "CSV file",
		},
		{
			name:        "valid CSV file",
			filename:    "valid.csv",
			fileContent: "Name,Value\ntest1,value1\ntest2,value2",
			setupFunc: func(filename, content string) error {
				return os.WriteFile(filename, []byte(content), 0644)
			},
			expectError: false,
			errorMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup file if needed
			if tt.setupFunc != nil {
				if err := tt.setupFunc(tt.filename, tt.fileContent); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
				defer os.Remove(tt.filename)
			}

			// Test the function
			headers, contents, err := getContentsFromFile(tt.filename)

			// Verify results
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if len(headers) == 0 {
					t.Error("Expected headers but got empty slice")
				}
				if len(contents) == 0 {
					t.Error("Expected contents but got empty slice")
				}
			}
		})
	}
}

func TestGetHeaderAndContentsFromFile_ErrorHandling(t *testing.T) {
	// Test error propagation from getContentsFromFile
	_, _, err := GetHeaderAndContentsFromFile("nonexistent.csv")
	if err == nil {
		t.Error("Expected error for nonexistent file but got none")
	}
	if !strings.Contains(err.Error(), "failed to read file") {
		t.Errorf("Expected error message about reading file, got: %v", err)
	}
}

func TestGetContentsFromFileAsStringMaps_ErrorHandling(t *testing.T) {
	// Test error propagation from getContentsFromFile
	_, err := GetContentsFromFileAsStringMaps("nonexistent.csv")
	if err == nil {
		t.Error("Expected error for nonexistent file but got none")
	}
	if !strings.Contains(err.Error(), "failed to read file") {
		t.Errorf("Expected error message about reading file, got: %v", err)
	}
}

func TestGetContentsFromFileAsStringMaps_ValidFile(t *testing.T) {
	// Create a temporary file for testing
	tempDir, err := os.MkdirTemp("", "drawio_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	filename := filepath.Join(tempDir, "test.csv")
	content := "Name,Value,Description\nitem1,100,First item\nitem2,200,Second item"

	err = os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result, err := GetContentsFromFileAsStringMaps(filename)
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 rows but got %d", len(result))
	}

	if result[0]["Name"] != "item1" {
		t.Errorf("Expected first row Name to be 'item1', got '%s'", result[0]["Name"])
	}
	if result[0]["Value"] != "100" {
		t.Errorf("Expected first row Value to be '100', got '%s'", result[0]["Value"])
	}
}

// TestCreateCSV_PreviouslyFatalScenarios tests scenarios that would have previously caused log.Fatal
func TestCreateCSV_PreviouslyFatalScenarios(t *testing.T) {
	t.Run("file creation failure - previously fatal", func(t *testing.T) {
		// This would have previously called log.Fatal and terminated the program
		err := CreateCSV(Header{}, []string{"test"}, []map[string]string{}, "/invalid/path/file.csv")

		// Now it should return an error instead of terminating
		if err == nil {
			t.Error("Expected error for invalid file path but got none")
		}

		// Verify the error is descriptive
		if !strings.Contains(err.Error(), "failed to create file") {
			t.Errorf("Expected descriptive error message, got: %v", err)
		}
	})

	t.Run("CSV parsing failure - previously fatal", func(t *testing.T) {
		// Create a file with invalid CSV content
		tempDir, err := os.MkdirTemp("", "drawio_test")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		filename := filepath.Join(tempDir, "invalid.csv")
		invalidContent := "Name,Value\n\"unclosed quote,value"

		err = os.WriteFile(filename, []byte(invalidContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// This would have previously called log.Fatal and terminated the program
		_, _, err = getContentsFromFile(filename)

		// Now it should return an error instead of terminating
		if err == nil {
			t.Error("Expected error for invalid CSV but got none")
		}

		// Verify the error is descriptive
		if !strings.Contains(err.Error(), "failed to parse CSV") {
			t.Errorf("Expected descriptive error message, got: %v", err)
		}
	})
}
