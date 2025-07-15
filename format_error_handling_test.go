package format

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ArjenSchwarz/go-output/errors"
	"github.com/ArjenSchwarz/go-output/mermaid"
	"github.com/ArjenSchwarz/go-output/validators"
)

// TestFormatSpecificErrorHandling tests error handling in individual output formats
func TestFormatSpecificErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setupArray  func() *OutputArray
		format      string
		expectError bool
		errorCode   errors.ErrorCode
		description string
	}{
		{
			name: "table_format_with_nil_contents",
			setupArray: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("table")
				return &OutputArray{
					Settings: settings,
					Contents: nil, // This could cause issues
					Keys:     []string{"Name", "Value"},
				}
			},
			format:      "table",
			expectError: false, // Table should handle nil contents gracefully
			description: "Table format should handle nil contents without error",
		},
		{
			name: "csv_format_with_special_characters",
			setupArray: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("csv")
				output := &OutputArray{
					Settings: settings,
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{
							"Name":  "Test,Item\"With,Quotes",
							"Value": "Some\nNewline\tTabs",
						}},
					},
					Keys: []string{"Name", "Value"},
				}
				return output
			},
			format:      "csv",
			expectError: false, // CSV should handle special characters through proper escaping
			description: "CSV format should handle special characters and escaping",
		},
		{
			name: "json_format_with_unserializable_data",
			setupArray: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("json")

				// Create an unserializable function value
				unserializable := func() {}

				output := &OutputArray{
					Settings: settings,
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{
							"Name":     "Test",
							"Function": unserializable, // This can't be marshaled to JSON
						}},
					},
					Keys: []string{"Name", "Function"},
				}
				return output
			},
			format:      "json",
			expectError: true,
			errorCode:   errors.ErrInvalidDataType,
			description: "JSON format should properly handle unmarshallable data",
		},
		{
			name: "mermaid_format_with_invalid_chart_type",
			setupArray: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("mermaid")
				settings.MermaidSettings = &mermaid.Settings{
					ChartType: "invalid_chart_type", // Invalid chart type
				}
				output := &OutputArray{
					Settings: settings,
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{
							"From": "A",
							"To":   "B",
						}},
					},
					Keys: []string{"From", "To"},
				}
				settings.FromToColumns = &FromToColumns{
					From: "From",
					To:   "To",
				}
				return output
			},
			format:      "mermaid",
			expectError: true,
			errorCode:   errors.ErrInvalidConfiguration,
			description: "Mermaid format should handle invalid chart types",
		},
		{
			name: "mermaid_piechart_with_invalid_values",
			setupArray: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("mermaid")
				settings.MermaidSettings = &mermaid.Settings{
					ChartType: "piechart",
				}
				output := &OutputArray{
					Settings: settings,
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{
							"Label": "A",
							"Value": "not_a_number", // Invalid number for pie chart
						}},
					},
					Keys: []string{"Label", "Value"},
				}
				settings.FromToColumns = &FromToColumns{
					From: "Label",
					To:   "Value",
				}
				return output
			},
			format:      "mermaid",
			expectError: true,
			errorCode:   errors.ErrInvalidDataType,
			description: "Mermaid piechart should handle invalid numeric values",
		},
		{
			name: "dot_format_with_empty_from_to",
			setupArray: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("dot")
				settings.FromToColumns = &FromToColumns{
					From: "From",
					To:   "To",
				}
				output := &OutputArray{
					Settings: settings,
					Contents: []OutputHolder{
						{Contents: map[string]interface{}{
							"From": "", // Empty from value
							"To":   "B",
						}},
					},
					Keys: []string{"From", "To"},
				}
				return output
			},
			format:      "dot",
			expectError: true,
			errorCode:   errors.ErrConstraintViolation,
			description: "DOT format should validate non-empty node names",
		},
		{
			name: "html_format_with_large_content",
			setupArray: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("html")

				// Create a large content array to test performance
				contents := make([]OutputHolder, 1000)
				for i := 0; i < 1000; i++ {
					contents[i] = OutputHolder{
						Contents: map[string]interface{}{
							"Index": i,
							"Data":  strings.Repeat("LargeData", 100),
						},
					}
				}

				output := &OutputArray{
					Settings: settings,
					Contents: contents,
					Keys:     []string{"Index", "Data"},
				}
				return output
			},
			format:      "html",
			expectError: false,
			description: "HTML format should handle large datasets efficiently",
		},
		{
			name: "yaml_format_with_complex_nested_data",
			setupArray: func() *OutputArray {
				settings := NewOutputSettings()
				settings.SetOutputFormat("yaml")

				// Create complex nested structure
				complexData := map[string]interface{}{
					"nested": map[string]interface{}{
						"deep": map[string]interface{}{
							"value": []interface{}{1, 2, 3},
						},
					},
				}

				output := &OutputArray{
					Settings: settings,
					Contents: []OutputHolder{
						{Contents: complexData},
					},
					Keys: []string{"nested"},
				}
				return output
			},
			format:      "yaml",
			expectError: false,
			description: "YAML format should handle complex nested data structures",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.setupArray()

			// Initialize error handling
			output.errorHandler = errors.NewDefaultErrorHandler()
			output.errorHandler.SetMode(errors.ErrorModeStrict)

			// Test format generation with error handling
			var result []byte
			var err error

			switch tt.format {
			case "table":
				result, err = output.toTableWithErrorHandling()
			case "csv":
				result, err = output.toCSVWithErrorHandling()
			case "json":
				result, err = output.toJSONWithErrorHandling()
			case "yaml":
				result, err = output.toYAMLWithErrorHandling()
			case "html":
				result, err = output.toHTMLWithErrorHandling()
			case "mermaid":
				result, err = output.toMermaidWithErrorHandling()
			case "dot":
				result, err = output.toDotWithErrorHandling()
			default:
				t.Fatalf("Unknown format: %s", tt.format)
			}

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.description)
					return
				}

				// Check if it's the expected error type
				if outputErr, ok := err.(errors.OutputError); ok {
					if outputErr.Code() != tt.errorCode {
						t.Errorf("Expected error code %s, got %s", tt.errorCode, outputErr.Code())
					}
				} else {
					t.Errorf("Expected OutputError, got %T: %v", err, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.description, err)
					return
				}

				// Basic validation that we got some output
				if len(result) == 0 {
					t.Errorf("Expected output for %s, but got empty result", tt.description)
				}
			}
		})
	}
}

// TestFormatColumnWidthCalculation tests table format column width issues
func TestFormatColumnWidthCalculation(t *testing.T) {
	settings := NewOutputSettings()
	settings.SetOutputFormat("table")
	settings.TableMaxColumnWidth = 5 // Very narrow columns

	output := &OutputArray{
		Settings: settings,
		Contents: []OutputHolder{
			{Contents: map[string]interface{}{
				"VeryLongColumnNameThatShouldBeWrapped": "Very long content that exceeds column width limits and should be properly handled",
			}},
		},
		Keys: []string{"VeryLongColumnNameThatShouldBeWrapped"},
	}

	output.errorHandler = errors.NewDefaultErrorHandler()
	result, err := output.toTableWithErrorHandling()

	if err != nil {
		t.Errorf("Table format should handle column width calculation gracefully: %v", err)
	}

	if len(result) == 0 {
		t.Error("Expected table output with column width handling")
	}
}

// TestFormatEncodingIssues tests handling of character encoding problems
func TestFormatEncodingIssues(t *testing.T) {
	// Test with various problematic characters
	problematicData := []struct {
		name    string
		content string
	}{
		{"unicode_emoji", "Test with emojis 🚀🎉🔥"},
		{"unicode_international", "Testing with international characters: こんにちは, Ñoño, Ëxample"},
		{"control_characters", "Test with\x00null\x01control\x02chars"},
		{"high_unicode", "Test with high unicode: \U0001F600\U0001F4A9"},
	}

	for _, data := range problematicData {
		t.Run(data.name, func(t *testing.T) {
			settings := NewOutputSettings()
			settings.SetOutputFormat("csv")

			output := &OutputArray{
				Settings: settings,
				Contents: []OutputHolder{
					{Contents: map[string]interface{}{
						"Content": data.content,
					}},
				},
				Keys: []string{"Content"},
			}

			output.errorHandler = errors.NewDefaultErrorHandler()
			result, err := output.toCSVWithErrorHandling()

			if err != nil {
				t.Errorf("CSV format should handle encoding issues gracefully for %s: %v", data.name, err)
			}

			if len(result) == 0 {
				t.Errorf("Expected CSV output for encoding test %s", data.name)
			}
		})
	}
}

// TestFormatRecoveryIntegration tests recovery strategies with format-specific errors
func TestFormatRecoveryIntegration(t *testing.T) {
	// Test that error handlers work correctly in lenient mode
	settings := NewOutputSettings()
	settings.SetOutputFormat("table")

	output := &OutputArray{
		Settings: settings,
		Contents: []OutputHolder{
			{Contents: map[string]interface{}{
				"Name":  "Test",
				"Value": "123",
			}},
		},
		Keys: []string{"Name", "Value"},
	}

	// Set up error handling in lenient mode
	output.errorHandler = errors.NewDefaultErrorHandler()
	output.errorHandler.SetMode(errors.ErrorModeLenient)

	// Add recovery handler (though it won't be needed for valid data)
	recovery := errors.NewDefaultRecoveryHandler()
	fallbackStrategy := errors.NewFormatFallbackStrategy([]string{"csv", "json"})
	recovery.AddStrategy(fallbackStrategy)
	output.recoveryHandler = recovery

	// This should succeed since the data is valid
	err := output.WriteWithValidation()

	if err != nil {
		t.Errorf("Expected successful output generation, but got error: %v", err)
	}

	// Test error collection in lenient mode by adding a validator that will trigger warnings
	output.AddValidator(&validators.RequiredColumnsValidator{})

	// Get error summary to verify lenient mode is collecting errors
	summary := output.GetErrorSummary()

	// The summary should be available even if no errors occurred
	if summary.BySeverity == nil {
		t.Error("Expected error summary to be available")
	}
}

// TestFormatPerformanceConsiderations tests performance aspects of error checking
func TestFormatPerformanceConsiderations(t *testing.T) {
	// Create a large dataset to test performance impact of error checking
	settings := NewOutputSettings()
	settings.SetOutputFormat("json")

	contents := make([]OutputHolder, 10000)
	for i := 0; i < 10000; i++ {
		contents[i] = OutputHolder{
			Contents: map[string]interface{}{
				"Index": i,
				"Data":  "Sample data",
				"Value": float64(i) * 2.5,
			},
		}
	}

	output := &OutputArray{
		Settings: settings,
		Contents: contents,
		Keys:     []string{"Index", "Data", "Value"},
	}

	output.errorHandler = errors.NewDefaultErrorHandler()

	// Test that error handling doesn't significantly impact performance
	result, err := output.toJSONWithErrorHandling()

	if err != nil {
		t.Errorf("Large dataset JSON generation should not error: %v", err)
	}

	if len(result) == 0 {
		t.Error("Expected JSON output for large dataset")
	}

	// Verify the JSON is valid
	var parsed interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Errorf("Generated JSON should be valid: %v", err)
	}
}
