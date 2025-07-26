package output

import (
	"context"
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestJSONRenderer_CollapsibleValue tests JSON rendering of CollapsibleValue
func TestJSONRenderer_CollapsibleValue(t *testing.T) {
	tests := []struct {
		name             string
		collapsibleVal   *DefaultCollapsibleValue
		expectedType     string
		expectedSummary  string
		expectedDetails  any
		expectedExpanded bool
		formatHints      map[string]any
	}{
		{
			name: "basic collapsible value",
			collapsibleVal: NewCollapsibleValue(
				"2 errors",
				[]string{"syntax error", "missing import"},
			),
			expectedType:     "collapsible",
			expectedSummary:  "2 errors",
			expectedDetails:  []string{"syntax error", "missing import"},
			expectedExpanded: false,
		},
		{
			name: "expanded collapsible value",
			collapsibleVal: NewCollapsibleValue(
				"Config details",
				map[string]any{"debug": true, "port": 8080},
				WithExpanded(true),
			),
			expectedType:     "collapsible",
			expectedSummary:  "Config details",
			expectedDetails:  map[string]any{"debug": true, "port": 8080},
			expectedExpanded: true,
		},
		{
			name: "collapsible with JSON format hints",
			collapsibleVal: NewCollapsibleValue(
				"File info",
				"/very/long/path/to/file.go",
				WithFormatHint(FormatJSON, map[string]any{"priority": "high", "category": "file"}),
			),
			expectedType:     "collapsible",
			expectedSummary:  "File info",
			expectedDetails:  "/very/long/path/to/file.go",
			expectedExpanded: false,
			formatHints:      map[string]any{"priority": "high", "category": "file"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test data with CollapsibleValue
			testData := []map[string]any{
				{
					"id":    1,
					"name":  "Test Item",
					"value": tt.collapsibleVal,
				},
			}

			// Create table with formatter that returns CollapsibleValue
			schema := []Field{
				{Name: "id", Type: "int"},
				{Name: "name", Type: "string"},
				{Name: "value", Type: "string", Formatter: func(val any) any {
					// Return the CollapsibleValue as-is for testing
					return val
				}},
			}

			doc := New().
				Table("test", testData, WithSchema(schema...)).
				Build()

			renderer := &jsonRenderer{}
			result, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			// Parse JSON result
			var parsed map[string]any
			if err := json.Unmarshal(result, &parsed); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			// Verify structure
			data, ok := parsed["data"].([]any)
			if !ok {
				t.Fatalf("Expected data array, got %T", parsed["data"])
			}

			if len(data) != 1 {
				t.Fatalf("Expected 1 record, got %d", len(data))
			}

			record, ok := data[0].(map[string]any)
			if !ok {
				t.Fatalf("Expected record map, got %T", data[0])
			}

			// Check the collapsible value structure
			valueField, ok := record["value"].(map[string]any)
			if !ok {
				t.Fatalf("Expected collapsible value map, got %T", record["value"])
			}

			// Verify type indicator (Requirement 4.1)
			if valueField["type"] != tt.expectedType {
				t.Errorf("Expected type %q, got %q", tt.expectedType, valueField["type"])
			}

			// Verify summary (Requirement 4.2)
			if valueField["summary"] != tt.expectedSummary {
				t.Errorf("Expected summary %q, got %q", tt.expectedSummary, valueField["summary"])
			}

			// Verify expanded state (Requirement 4.2)
			if valueField["expanded"] != tt.expectedExpanded {
				t.Errorf("Expected expanded %t, got %t", tt.expectedExpanded, valueField["expanded"])
			}

			// Verify details (Requirement 4.2)
			switch expectedDetails := tt.expectedDetails.(type) {
			case []string:
				detailsArray, ok := valueField["details"].([]any)
				if !ok {
					t.Fatalf("Expected details array, got %T", valueField["details"])
				}
				if len(detailsArray) != len(expectedDetails) {
					t.Errorf("Expected %d details, got %d", len(expectedDetails), len(detailsArray))
				}
				for i, expected := range expectedDetails {
					if detailsArray[i] != expected {
						t.Errorf("Expected detail[%d] %q, got %q", i, expected, detailsArray[i])
					}
				}
			case map[string]any:
				detailsMap, ok := valueField["details"].(map[string]any)
				if !ok {
					t.Fatalf("Expected details map, got %T", valueField["details"])
				}
				for k, expected := range expectedDetails {
					actual := detailsMap[k]
					// Handle JSON number conversion (int -> float64)
					if expectedInt, ok := expected.(int); ok {
						if actualFloat, ok := actual.(float64); ok {
							if float64(expectedInt) != actualFloat {
								t.Errorf("Expected detail[%s] %v, got %v", k, expected, actual)
							}
							continue
						}
					}
					if actual != expected {
						t.Errorf("Expected detail[%s] %v, got %v", k, expected, actual)
					}
				}
			case string:
				if valueField["details"] != expectedDetails {
					t.Errorf("Expected details %q, got %q", expectedDetails, valueField["details"])
				}
			}

			// Verify format hints if present (Requirement 4.3)
			if tt.formatHints != nil {
				for key, expectedValue := range tt.formatHints {
					if valueField[key] != expectedValue {
						t.Errorf("Expected format hint %s=%v, got %v", key, expectedValue, valueField[key])
					}
				}
			}
		})
	}
}

// TestYAMLRenderer_CollapsibleValue tests YAML rendering of CollapsibleValue
func TestYAMLRenderer_CollapsibleValue(t *testing.T) {
	tests := []struct {
		name             string
		collapsibleVal   *DefaultCollapsibleValue
		expectedSummary  string
		expectedDetails  any
		expectedExpanded bool
		formatHints      map[string]any
	}{
		{
			name: "basic collapsible value",
			collapsibleVal: NewCollapsibleValue(
				"3 warnings",
				[]string{"deprecated API", "unused variable", "missing docs"},
			),
			expectedSummary:  "3 warnings",
			expectedDetails:  []string{"deprecated API", "unused variable", "missing docs"},
			expectedExpanded: false,
		},
		{
			name: "expanded collapsible value with map",
			collapsibleVal: NewCollapsibleValue(
				"Server config",
				map[string]any{"host": "localhost", "port": 3000, "ssl": false},
				WithExpanded(true),
			),
			expectedSummary:  "Server config",
			expectedDetails:  map[string]any{"host": "localhost", "port": 3000, "ssl": false},
			expectedExpanded: true,
		},
		{
			name: "collapsible with YAML format hints",
			collapsibleVal: NewCollapsibleValue(
				"Database info",
				"postgresql://localhost:5432/mydb",
				WithFormatHint(FormatYAML, map[string]any{"secure": true, "pool_size": 10}),
			),
			expectedSummary:  "Database info",
			expectedDetails:  "postgresql://localhost:5432/mydb",
			expectedExpanded: false,
			formatHints:      map[string]any{"secure": true, "pool_size": 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test data with CollapsibleValue
			testData := []map[string]any{
				{
					"id":     1,
					"status": "active",
					"info":   tt.collapsibleVal,
				},
			}

			// Create table with formatter that returns CollapsibleValue
			schema := []Field{
				{Name: "id", Type: "int"},
				{Name: "status", Type: "string"},
				{Name: "info", Type: "string", Formatter: func(val any) any {
					// Return the CollapsibleValue as-is for testing
					return val
				}},
			}

			doc := New().
				Table("test", testData, WithSchema(schema...)).
				Build()

			renderer := &yamlRenderer{}
			result, err := renderer.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Render failed: %v", err)
			}

			// Parse YAML result
			var parsed map[string]any
			if err := yaml.Unmarshal(result, &parsed); err != nil {
				t.Fatalf("Failed to parse YAML: %v", err)
			}

			// Verify structure
			data, ok := parsed["data"].([]any)
			if !ok {
				t.Fatalf("Expected data array, got %T", parsed["data"])
			}

			if len(data) != 1 {
				t.Fatalf("Expected 1 record, got %d", len(data))
			}

			record, ok := data[0].(map[string]any)
			if !ok {
				t.Fatalf("Expected record map, got %T", data[0])
			}

			// Check the collapsible value structure
			infoField, ok := record["info"].(map[string]any)
			if !ok {
				t.Fatalf("Expected collapsible value map, got %T", record["info"])
			}

			// Verify summary (Requirement 5.1)
			if infoField["summary"] != tt.expectedSummary {
				t.Errorf("Expected summary %q, got %q", tt.expectedSummary, infoField["summary"])
			}

			// Verify expanded state (Requirement 5.1)
			if infoField["expanded"] != tt.expectedExpanded {
				t.Errorf("Expected expanded %t, got %t", tt.expectedExpanded, infoField["expanded"])
			}

			// Verify details (Requirement 5.1)
			switch expectedDetails := tt.expectedDetails.(type) {
			case []string:
				detailsArray, ok := infoField["details"].([]any)
				if !ok {
					t.Fatalf("Expected details array, got %T", infoField["details"])
				}
				if len(detailsArray) != len(expectedDetails) {
					t.Errorf("Expected %d details, got %d", len(expectedDetails), len(detailsArray))
				}
				for i, expected := range expectedDetails {
					if detailsArray[i] != expected {
						t.Errorf("Expected detail[%d] %q, got %q", i, expected, detailsArray[i])
					}
				}
			case map[string]any:
				detailsMap, ok := infoField["details"].(map[string]any)
				if !ok {
					t.Fatalf("Expected details map, got %T", infoField["details"])
				}
				for k, expected := range expectedDetails {
					if detailsMap[k] != expected {
						t.Errorf("Expected detail[%s] %v, got %v", k, expected, detailsMap[k])
					}
				}
			case string:
				if infoField["details"] != expectedDetails {
					t.Errorf("Expected details %q, got %q", expectedDetails, infoField["details"])
				}
			}

			// Verify format hints if present (Requirement 5.2)
			if tt.formatHints != nil {
				for key, expectedValue := range tt.formatHints {
					if infoField[key] != expectedValue {
						t.Errorf("Expected format hint %s=%v, got %v", key, expectedValue, infoField[key])
					}
				}
			}
		})
	}
}

// TestJSONYAMLRenderers_CollapsibleKeyOrderPreservation tests that key order is maintained with collapsible values
func TestJSONYAMLRenderers_CollapsibleKeyOrderPreservation(t *testing.T) {
	// Create test data with CollapsibleValue in specific key order
	testData := []map[string]any{
		{
			"zebra":  "last",
			"alpha":  NewCollapsibleValue("Short", "This is a longer description"),
			"middle": "center",
		},
	}

	keyOrder := []string{"zebra", "alpha", "middle"}

	schema := []Field{
		{Name: "zebra", Type: "string"},
		{Name: "alpha", Type: "string", Formatter: func(val any) any { return val }},
		{Name: "middle", Type: "string"},
	}

	doc := New().
		Table("test", testData, WithSchema(schema...)).
		Build()

	t.Run("JSON key order preservation", func(t *testing.T) {
		renderer := &jsonRenderer{}
		result, err := renderer.Render(context.Background(), doc)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}

		// Parse and verify key order
		var parsed map[string]any
		if err := json.Unmarshal(result, &parsed); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		schema, ok := parsed["schema"].(map[string]any)
		if !ok {
			t.Fatalf("Expected schema map")
		}

		keys, ok := schema["keys"].([]any)
		if !ok {
			t.Fatalf("Expected keys array")
		}

		for i, expectedKey := range keyOrder {
			if keys[i] != expectedKey {
				t.Errorf("Expected key[%d] %s, got %s", i, expectedKey, keys[i])
			}
		}

		// Verify collapsible value exists and maintains structure
		data := parsed["data"].([]any)
		record := data[0].(map[string]any)
		alphaValue := record["alpha"].(map[string]any)

		if alphaValue["type"] != "collapsible" {
			t.Errorf("Expected collapsible type, got %v", alphaValue["type"])
		}
	})

	t.Run("YAML key order preservation", func(t *testing.T) {
		renderer := &yamlRenderer{}
		result, err := renderer.Render(context.Background(), doc)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}

		// Parse and verify key order
		var parsed map[string]any
		if err := yaml.Unmarshal(result, &parsed); err != nil {
			t.Fatalf("Failed to parse YAML: %v", err)
		}

		schema, ok := parsed["schema"].(map[string]any)
		if !ok {
			t.Fatalf("Expected schema map")
		}

		keys, ok := schema["keys"].([]any)
		if !ok {
			t.Fatalf("Expected keys array")
		}

		for i, expectedKey := range keyOrder {
			if keys[i] != expectedKey {
				t.Errorf("Expected key[%d] %s, got %s", i, expectedKey, keys[i])
			}
		}

		// Verify collapsible value exists and maintains structure
		data := parsed["data"].([]any)
		record := data[0].(map[string]any)
		alphaValue := record["alpha"].(map[string]any)

		if alphaValue["summary"] != "Short" {
			t.Errorf("Expected summary 'Short', got %v", alphaValue["summary"])
		}
	})
}

// TestJSONYAMLRenderers_LargeDataset tests streaming capabilities with collapsible values
func TestJSONYAMLRenderers_CollapsibleStreaming(t *testing.T) {
	// Create test data with many collapsible values
	const numRecords = 1000
	testData := make([]map[string]any, numRecords)

	for i := 0; i < numRecords; i++ {
		testData[i] = map[string]any{
			"id": i,
			"error": NewCollapsibleValue(
				"Error details",
				[]string{"error 1", "error 2", "error 3"},
			),
		}
	}

	schema := []Field{
		{Name: "id", Type: "int"},
		{Name: "error", Type: "string", Formatter: func(val any) any { return val }},
	}

	doc := New().
		Table("large_test", testData, WithSchema(schema...)).
		Build()

	t.Run("JSON streaming with collapsible values", func(t *testing.T) {
		renderer := &jsonRenderer{}
		result, err := renderer.Render(context.Background(), doc)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}

		// Verify result is valid JSON
		var parsed map[string]any
		if err := json.Unmarshal(result, &parsed); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		// Verify we have the correct number of records
		data := parsed["data"].([]any)
		if len(data) != numRecords {
			t.Errorf("Expected %d records, got %d", numRecords, len(data))
		}

		// Verify first and last records have collapsible structure
		firstRecord := data[0].(map[string]any)
		lastRecord := data[numRecords-1].(map[string]any)

		for _, record := range []map[string]any{firstRecord, lastRecord} {
			errorValue := record["error"].(map[string]any)
			if errorValue["type"] != "collapsible" {
				t.Errorf("Expected collapsible type in record")
			}
		}
	})

	t.Run("YAML streaming with collapsible values", func(t *testing.T) {
		renderer := &yamlRenderer{}
		result, err := renderer.Render(context.Background(), doc)
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}

		// Verify result is valid YAML
		var parsed map[string]any
		if err := yaml.Unmarshal(result, &parsed); err != nil {
			t.Fatalf("Failed to parse YAML: %v", err)
		}

		// Verify we have the correct number of records
		data := parsed["data"].([]any)
		if len(data) != numRecords {
			t.Errorf("Expected %d records, got %d", numRecords, len(data))
		}

		// Verify first and last records have collapsible structure
		firstRecord := data[0].(map[string]any)
		lastRecord := data[numRecords-1].(map[string]any)

		for _, record := range []map[string]any{firstRecord, lastRecord} {
			errorValue := record["error"].(map[string]any)
			if errorValue["summary"] != "Error details" {
				t.Errorf("Expected summary 'Error details' in record")
			}
		}
	})
}
