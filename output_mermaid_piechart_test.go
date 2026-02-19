package format

import (
	"strings"
	"testing"

	"github.com/ArjenSchwarz/go-output/mermaid"
)

// TestMermaidPiechartIntegerValues tests that piechart handles integer values correctly
// This is a regression test for T-143: Mermaid piechart ignores int values
func TestMermaidPiechartIntegerValues(t *testing.T) {
	tests := map[string]struct {
		value    interface{}
		expected string // Expected to be present in mermaid output
	}{
		"int": {
			value:    42,
			expected: "42",
		},
		"int64": {
			value:    int64(100),
			expected: "100",
		},
		"int32": {
			value:    int32(75),
			expected: "75",
		},
		"float64": {
			value:    75.5,
			expected: "75.5",
		},
		"float32": {
			value:    float32(25.25),
			expected: "25.25",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			settings := NewOutputSettings()
			settings.SetOutputFormat("mermaid")
			settings.MermaidSettings = &mermaid.Settings{ChartType: "piechart"}
			settings.AddFromToColumns("Label", "Value")

			output := OutputArray{
				Settings: settings,
				Keys:     []string{"Label", "Value"},
			}

			output.AddContents(map[string]interface{}{
				"Label": "Test",
				"Value": tc.value,
			})

			result := output.toMermaid()
			resultStr := string(result)

			// The mermaid output should contain the numeric value
			if !strings.Contains(resultStr, tc.expected) {
				t.Errorf("Expected mermaid output to contain %q, but got: %s", tc.expected, resultStr)
			}

			// The output should not be empty (which happens when values are treated as 0)
			if len(strings.TrimSpace(resultStr)) == 0 {
				t.Error("Mermaid output should not be empty")
			}
		})
	}
}

// TestMermaidPiechartMixedTypes tests that piechart handles mixed numeric types correctly
func TestMermaidPiechartMixedTypes(t *testing.T) {
	settings := NewOutputSettings()
	settings.SetOutputFormat("mermaid")
	settings.MermaidSettings = &mermaid.Settings{ChartType: "piechart"}
	settings.AddFromToColumns("Label", "Value")

	output := OutputArray{
		Settings: settings,
		Keys:     []string{"Label", "Value"},
	}

	// Add mixed numeric types
	output.AddContents(map[string]interface{}{
		"Label": "Users",
		"Value": 42, // int
	})

	output.AddContents(map[string]interface{}{
		"Label": "Items",
		"Value": int64(100), // int64
	})

	output.AddContents(map[string]interface{}{
		"Label": "Score",
		"Value": 75.5, // float64
	})

	result := output.toMermaid()
	resultStr := string(result)

	// All values should be present in the output
	expectedValues := []string{"42", "100", "75.5"}
	for _, expected := range expectedValues {
		if !strings.Contains(resultStr, expected) {
			t.Errorf("Expected mermaid output to contain %q, but got: %s", expected, resultStr)
		}
	}
}
