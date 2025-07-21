package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMigrator_IntegrationTests tests the migration tool with complete v1 code examples
func TestMigrator_IntegrationTests(t *testing.T) {
	tests := []struct {
		name             string
		v1Code           string
		expectedPatterns []string
		expectedRules    []string
		shouldSucceed    bool
	}{
		{
			name: "Basic table output pattern",
			v1Code: `package main

import (
	"github.com/ArjenSchwarz/go-output/format"
)

func main() {
	output := &format.OutputArray{
		Keys: []string{"Name", "Age", "Status"},
	}
	
	data := map[string]interface{}{
		"Name":   "Alice",
		"Age":    30,
		"Status": "Active",
	}
	
	output.AddContents(data)
	output.Write()
}`,
			expectedPatterns: []string{
				"V1ImportStatement",
				"OutputArrayDeclaration",
				"KeysFieldAssignment",
				"AddContentsCall",
				"WriteCall",
			},
			expectedRules: []string{
				"UpdateImportStatement",
				"ReplaceOutputArrayDeclaration",
				"ReplaceAddContents",
				"ReplaceWriteCall",
			},
			shouldSucceed: true,
		},
		{
			name: "Multiple tables with different keys",
			v1Code: `package main

import (
	"github.com/ArjenSchwarz/go-output/format"
)

func main() {
	output := &format.OutputArray{}
	
	// First table
	output.Keys = []string{"Name", "Email"}
	output.AddContents(userData)
	output.AddToBuffer()
	
	// Second table
	output.Keys = []string{"ID", "Status", "Time"}
	output.AddContents(statusData)
	output.AddToBuffer()
	
	output.Write()
}`,
			expectedPatterns: []string{
				"V1ImportStatement",
				"OutputArrayDeclaration",
				"KeysFieldAssignment",
				"AddContentsCall",
				"AddToBufferCall",
				"WriteCall",
			},
			expectedRules: []string{
				"UpdateImportStatement",
				"ReplaceOutputArrayDeclaration",
				"ReplaceAddContents",
				"ReplaceAddToBuffer",
				"ReplaceWriteCall",
				"ReplaceKeysAssignment",
			},
			shouldSucceed: true,
		},
		{
			name: "OutputSettings configuration",
			v1Code: `package main

import (
	"github.com/ArjenSchwarz/go-output/format"
)

func main() {
	settings := format.NewOutputSettings()
	settings.OutputFormat = "table"
	settings.UseEmoji = true
	settings.UseColors = true
	settings.TableStyle = "ColoredBright"
	settings.HasTOC = true
	
	output := &format.OutputArray{
		Settings: settings,
	}
	
	output.AddContents(data)
	output.Write()
}`,
			expectedPatterns: []string{
				"V1ImportStatement",
				"OutputSettingsUsage",
				"OutputArrayDeclaration",
				"SettingsFieldAssignment",
				"AddContentsCall",
				"WriteCall",
			},
			expectedRules: []string{
				"UpdateImportStatement",
				"ReplaceOutputSettings",
				"ReplaceOutputArrayDeclaration",
				"ReplaceAddContents",
				"ReplaceWriteCall",
			},
			shouldSucceed: true,
		},
		{
			name: "Progress indicator usage",
			v1Code: `package main

import (
	"github.com/ArjenSchwarz/go-output/format"
)

func main() {
	settings := format.NewOutputSettings()
	settings.OutputFormat = "table"
	
	p := format.NewProgress(settings)
	p.SetTotal(100)
	p.SetColor(format.ProgressColorGreen)
	
	for i := 0; i < 100; i++ {
		p.Increment(1)
		p.SetStatus("Processing...")
	}
	
	p.Complete()
}`,
			expectedPatterns: []string{
				"V1ImportStatement",
				"OutputSettingsUsage",
				"ProgressCreation",
			},
			expectedRules: []string{
				"UpdateImportStatement",
				"ReplaceOutputSettings",
				"ReplaceProgressCreation",
			},
			shouldSucceed: true,
		},
		{
			name: "Header and mixed content",
			v1Code: `package main

import (
	"github.com/ArjenSchwarz/go-output/format"
)

func main() {
	output := &format.OutputArray{}
	
	output.AddHeader("User Report")
	output.Keys = []string{"Name", "Age"}
	output.AddContents(userData)
	
	output.AddHeader("Status Report") 
	output.Keys = []string{"ID", "Status"}
	output.AddContents(statusData)
	
	output.Write()
}`,
			expectedPatterns: []string{
				"V1ImportStatement",
				"OutputArrayDeclaration",
				"AddHeaderCall",
				"KeysFieldAssignment",
				"AddContentsCall",
				"WriteCall",
			},
			expectedRules: []string{
				"UpdateImportStatement",
				"ReplaceOutputArrayDeclaration",
				"ReplaceAddHeader",
				"ReplaceKeysAssignment",
				"ReplaceAddContents",
				"ReplaceWriteCall",
			},
			shouldSucceed: true,
		},
	}

	migrator := New()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tempFile, err := createTempFile(tt.v1Code)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tempFile)

			// Perform migration
			result, err := migrator.MigrateFile(tempFile)

			if tt.shouldSucceed && err != nil {
				t.Fatalf("Migration failed unexpectedly: %v", err)
			}

			if !tt.shouldSucceed && err == nil {
				t.Fatalf("Migration should have failed but succeeded")
			}

			if !tt.shouldSucceed {
				return // Skip further checks for expected failures
			}

			// Verify expected patterns were found
			for _, expectedPattern := range tt.expectedPatterns {
				found := false
				for _, foundPattern := range result.PatternsFound {
					if foundPattern == expectedPattern {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected pattern %s not found. Found: %v", expectedPattern, result.PatternsFound)
				}
			}

			// Verify expected rules were applied
			for _, expectedRule := range tt.expectedRules {
				found := false
				for _, appliedRule := range result.RulesApplied {
					if appliedRule == expectedRule {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected rule %s not applied. Applied: %v", expectedRule, result.RulesApplied)
				}
			}

			// Verify transformed code contains v2 imports
			if !strings.Contains(result.TransformedFile, "github.com/ArjenSchwarz/go-output/v2") {
				t.Error("Transformed code should contain v2 import")
			}

			// Verify transformed code doesn't contain v1 patterns
			if strings.Contains(result.TransformedFile, "OutputArray") {
				t.Error("Transformed code should not contain OutputArray")
			}

			// Log the transformation for manual inspection
			t.Logf("Patterns found: %v", result.PatternsFound)
			t.Logf("Rules applied: %v", result.RulesApplied)
			t.Logf("Transformed code:\n%s", result.TransformedFile)
		})
	}
}

// TestMigrator_DirectoryMigration tests migrating entire directories
func TestMigrator_DirectoryMigration(t *testing.T) {
	// Create temporary directory with multiple Go files
	tempDir, err := createTempDirectory()
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create multiple test files
	files := map[string]string{
		"main.go": `package main

import "github.com/ArjenSchwarz/go-output/format"

func main() {
	output := &format.OutputArray{}
	output.AddContents(data)
	output.Write()
}`,
		"report.go": `package main

import "github.com/ArjenSchwarz/go-output/format"

func generateReport() {
	settings := format.NewOutputSettings()
	settings.UseEmoji = true
	
	output := &format.OutputArray{Settings: settings}
	output.Keys = []string{"Name", "Value"}
	output.AddContents(reportData)
	output.Write()
}`,
		"utils.go": `package main

// This file has no v1 usage and should be unchanged
func helper() string {
	return "unchanged"
}`,
	}

	for filename, content := range files {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Perform directory migration
	migrator := New()
	results, err := migrator.MigrateDirectory(tempDir)
	if err != nil {
		t.Fatalf("Directory migration failed: %v", err)
	}

	// Should have results for all .go files
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Check that files with v1 usage were transformed
	for _, result := range results {
		filename := filepath.Base(result.OriginalFile)

		switch filename {
		case "main.go", "report.go":
			// These should have patterns found
			if len(result.PatternsFound) == 0 {
				t.Errorf("File %s should have patterns found", filename)
			}
		case "utils.go":
			// This should have no patterns
			if len(result.PatternsFound) != 0 {
				t.Errorf("File %s should have no patterns found, got: %v", filename, result.PatternsFound)
			}
		}
	}
}

// TestMigrator_ErrorHandling tests error conditions
func TestMigrator_ErrorHandling(t *testing.T) {
	migrator := New()

	// Test with non-existent file
	_, err := migrator.MigrateFile("/nonexistent/file.go")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Test with invalid Go code
	invalidCode := `package main
this is not valid go code`

	tempFile, err := createTempFile(invalidCode)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile)

	_, err = migrator.MigrateFile(tempFile)
	if err == nil {
		t.Error("Expected error for invalid Go code")
	}
}

// TestAdvancedPatternDetection tests the advanced pattern detection
func TestAdvancedPatternDetection(t *testing.T) {
	v1Code := `package main

import "github.com/ArjenSchwarz/go-output/format"

func main() {
	settings := format.NewOutputSettings()
	settings.OutputFormat = "table"
	settings.UseEmoji = true
	
	output := &format.OutputArray{Settings: settings}
	output.Keys = []string{"Name", "Age"}
	output.AddContents(userData)
	output.AddToBuffer()
	
	output.Keys = []string{"ID", "Status"} 
	output.AddContents(statusData)
	output.Write()
}`

	tempFile, err := createTempFile(v1Code)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile)

	migrator := New()
	result, err := migrator.MigrateFile(tempFile)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Should detect complex patterns
	complexPatterns := []string{
		"OutputArrayDeclaration",
		"OutputSettingsUsage",
		"KeysFieldAssignment",
		"AddContentsCall",
		"AddToBufferCall",
		"WriteCall",
	}

	for _, expectedPattern := range complexPatterns {
		found := false
		for _, foundPattern := range result.PatternsFound {
			if foundPattern == expectedPattern {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Complex pattern %s not detected", expectedPattern)
		}
	}
}

// Helper functions

func createTempFile(content string) (string, error) {
	tempFile, err := os.CreateTemp("", "migrate_test_*.go")
	if err != nil {
		return "", err
	}

	if _, err := tempFile.WriteString(content); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return "", err
	}

	tempFile.Close()
	return tempFile.Name(), nil
}

func createTempDirectory() (string, error) {
	return os.MkdirTemp("", "migrate_test_dir_*")
}
