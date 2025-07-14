package format

import (
	"strings"
	"testing"

	"github.com/ArjenSchwarz/go-output/drawio"
)

// TestOutputArray_DrawIOErrorHandling tests that drawio errors are properly handled
// instead of causing log.Fatal termination
func TestOutputArray_DrawIOErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "invalid file path should return error",
			filename:    "/invalid/path/test.csv",
			expectError: true,
			errorMsg:    "failed to create drawio CSV",
		},
		{
			name:        "valid stdout output should work",
			filename:    "",
			expectError: false,
			errorMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create OutputArray with drawio format
			settings := NewOutputSettings()
			settings.SetOutputFormat("drawio")
			settings.OutputFile = tt.filename

			// Set up DrawIO header (required for drawio format)
			settings.DrawIOHeader = drawio.DefaultHeader()

			output := OutputArray{
				Settings: settings,
				Keys:     []string{"Name", "Value"},
			}

			// Add some test data
			output.AddHolder(OutputHolder{
				Contents: map[string]interface{}{
					"Name":  "test1",
					"Value": "value1",
				},
			})

			// Test Write method - this should return error instead of calling log.Fatal
			err := output.Write()

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
		})
	}
}

// TestOutputArray_DrawIOPreviouslyFatalScenarios tests scenarios that would have
// previously caused log.Fatal and program termination
func TestOutputArray_DrawIOPreviouslyFatalScenarios(t *testing.T) {
	t.Run("file creation failure - previously fatal", func(t *testing.T) {
		// This scenario would have previously called log.Fatal and terminated the program
		settings := NewOutputSettings()
		settings.SetOutputFormat("drawio")
		settings.OutputFile = "/invalid/path/test.csv"

		// Set up DrawIO header (required for drawio format)
		settings.DrawIOHeader = drawio.DefaultHeader()

		output := OutputArray{
			Settings: settings,
			Keys:     []string{"Name", "Value"},
		}

		output.AddHolder(OutputHolder{
			Contents: map[string]interface{}{
				"Name":  "test",
				"Value": "value",
			},
		})

		// This should now return an error instead of terminating the program
		err := output.Write()

		if err == nil {
			t.Error("Expected error for invalid file path but got none")
		}

		// Verify the error is descriptive and includes context
		if !strings.Contains(err.Error(), "failed to create drawio CSV") {
			t.Errorf("Expected descriptive error message, got: %v", err)
		}

		// Verify it's a ProcessingError with the correct error code
		if procErr, ok := err.(ProcessingError); ok {
			if procErr.Code() != ErrFileWrite {
				t.Errorf("Expected error code %s, got %s", ErrFileWrite, procErr.Code())
			}
		} else {
			t.Error("Expected ProcessingError type")
		}
	})
}
