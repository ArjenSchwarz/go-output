package format

import (
	"testing"

	"github.com/ArjenSchwarz/go-output/drawio"
	"github.com/ArjenSchwarz/go-output/mermaid"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// TestOutputSettings_Validate tests the main Validate method
func TestOutputSettings_Validate(t *testing.T) {
	tests := []struct {
		name        string
		settings    *OutputSettings
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid default settings",
			settings: NewOutputSettings(),
			wantErr:  false,
		},
		{
			name: "valid json format",
			settings: func() *OutputSettings {
				s := NewOutputSettings()
				s.SetOutputFormat("json")
				return s
			}(),
			wantErr: false,
		},
		{
			name: "invalid format",
			settings: &OutputSettings{
				OutputFormat: "invalid",
			},
			wantErr:     true,
			errContains: "invalid output format",
		},
		{
			name: "mermaid without configuration",
			settings: &OutputSettings{
				OutputFormat: "mermaid",
			},
			wantErr:     true,
			errContains: "mermaid format requires FromToColumns or MermaidSettings",
		},
		{
			name: "dot without configuration",
			settings: &OutputSettings{
				OutputFormat: "dot",
			},
			wantErr:     true,
			errContains: "dot format requires FromToColumns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.settings.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("OutputSettings.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errContains != "" {
				if !containsString(err.Error(), tt.errContains) {
					t.Errorf("OutputSettings.Validate() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

// TestOutputSettings_validateOutputFormat tests output format validation
func TestOutputSettings_validateOutputFormat(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		wantErr     bool
		errContains string
	}{
		{"empty format (defaults to json)", "", false, ""},
		{"valid json", "json", false, ""},
		{"valid csv", "csv", false, ""},
		{"valid html", "html", false, ""},
		{"valid table", "table", false, ""},
		{"valid markdown", "markdown", false, ""},
		{"valid mermaid", "mermaid", false, ""},
		{"valid drawio", "drawio", false, ""},
		{"valid dot", "dot", false, ""},
		{"valid yaml", "yaml", false, ""},
		{"invalid format", "invalid", true, "invalid output format"},
		{"case sensitive invalid", "JSON", true, "invalid output format"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := &OutputSettings{OutputFormat: tt.format}
			err := settings.validateOutputFormat()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateOutputFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errContains != "" {
				if !containsString(err.Error(), tt.errContains) {
					t.Errorf("validateOutputFormat() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

// TestOutputSettings_validateMermaidRequirements tests mermaid format validation
func TestOutputSettings_validateMermaidRequirements(t *testing.T) {
	tests := []struct {
		name            string
		fromToColumns   *FromToColumns
		mermaidSettings *mermaid.Settings
		wantErr         bool
		errContains     string
	}{
		{
			name:            "no configuration",
			fromToColumns:   nil,
			mermaidSettings: nil,
			wantErr:         true,
			errContains:     "mermaid format requires FromToColumns or MermaidSettings",
		},
		{
			name: "valid FromToColumns",
			fromToColumns: &FromToColumns{
				From: "source",
				To:   "target",
			},
			mermaidSettings: nil,
			wantErr:         false,
		},
		{
			name:            "valid MermaidSettings",
			fromToColumns:   nil,
			mermaidSettings: &mermaid.Settings{},
			wantErr:         false,
		},
		{
			name: "both configurations",
			fromToColumns: &FromToColumns{
				From: "source",
				To:   "target",
			},
			mermaidSettings: &mermaid.Settings{},
			wantErr:         false,
		},
		{
			name: "invalid FromToColumns - empty From",
			fromToColumns: &FromToColumns{
				From: "",
				To:   "target",
			},
			mermaidSettings: nil,
			wantErr:         true,
			errContains:     "FromToColumns.From cannot be empty",
		},
		{
			name: "invalid FromToColumns - empty To",
			fromToColumns: &FromToColumns{
				From: "source",
				To:   "",
			},
			mermaidSettings: nil,
			wantErr:         true,
			errContains:     "FromToColumns.To cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := &OutputSettings{
				FromToColumns:   tt.fromToColumns,
				MermaidSettings: tt.mermaidSettings,
			}
			err := settings.validateMermaidRequirements()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMermaidRequirements() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errContains != "" {
				if !containsString(err.Error(), tt.errContains) {
					t.Errorf("validateMermaidRequirements() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

// TestOutputSettings_validateDotRequirements tests dot format validation
func TestOutputSettings_validateDotRequirements(t *testing.T) {
	tests := []struct {
		name          string
		fromToColumns *FromToColumns
		wantErr       bool
		errContains   string
	}{
		{
			name:          "no FromToColumns",
			fromToColumns: nil,
			wantErr:       true,
			errContains:   "dot format requires FromToColumns",
		},
		{
			name: "valid FromToColumns",
			fromToColumns: &FromToColumns{
				From: "source",
				To:   "target",
			},
			wantErr: false,
		},
		{
			name: "invalid FromToColumns - empty From",
			fromToColumns: &FromToColumns{
				From: "",
				To:   "target",
			},
			wantErr:     true,
			errContains: "FromToColumns.From cannot be empty",
		},
		{
			name: "invalid FromToColumns - empty To",
			fromToColumns: &FromToColumns{
				From: "source",
				To:   "",
			},
			wantErr:     true,
			errContains: "FromToColumns.To cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := &OutputSettings{
				FromToColumns: tt.fromToColumns,
			}
			err := settings.validateDotRequirements()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDotRequirements() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errContains != "" {
				if !containsString(err.Error(), tt.errContains) {
					t.Errorf("validateDotRequirements() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

// TestOutputSettings_validateDrawIORequirements tests draw.io format validation
func TestOutputSettings_validateDrawIORequirements(t *testing.T) {
	tests := []struct {
		name         string
		drawIOHeader drawio.Header
		wantErr      bool
		errContains  string
	}{
		{
			name:         "empty DrawIOHeader",
			drawIOHeader: drawio.Header{},
			wantErr:      true,
			errContains:  "drawio format requires DrawIOHeader",
		},
		// Note: We can't easily test a valid DrawIOHeader without knowing the internal structure
		// This would require examining the drawio package to understand what makes a header "set"
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := &OutputSettings{
				DrawIOHeader: tt.drawIOHeader,
			}
			err := settings.validateDrawIORequirements()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDrawIORequirements() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errContains != "" {
				if !containsString(err.Error(), tt.errContains) {
					t.Errorf("validateDrawIORequirements() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

// TestOutputSettings_validateFileOutput tests file output validation
func TestOutputSettings_validateFileOutput(t *testing.T) {
	tests := []struct {
		name             string
		outputFile       string
		outputFileFormat string
		wantErr          bool
		errContains      string
	}{
		{
			name:       "no file output",
			outputFile: "",
			wantErr:    false,
		},
		{
			name:       "valid file path",
			outputFile: "output.json",
			wantErr:    false,
		},
		{
			name:       "valid file path with directory",
			outputFile: "/path/to/output.csv",
			wantErr:    false,
		},
		{
			name:        "file path with control characters",
			outputFile:  "output\x00.json",
			wantErr:     true,
			errContains: "file path contains invalid characters",
		},
		{
			name:        "whitespace only file path",
			outputFile:  "   ",
			wantErr:     true,
			errContains: "file path cannot be empty or whitespace only",
		},
		{
			name:             "valid file format",
			outputFile:       "output.txt",
			outputFileFormat: "json",
			wantErr:          false,
		},
		{
			name:             "invalid file format",
			outputFile:       "output.txt",
			outputFileFormat: "invalid",
			wantErr:          true,
			errContains:      "invalid output file format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := &OutputSettings{
				OutputFile:       tt.outputFile,
				OutputFileFormat: tt.outputFileFormat,
			}
			err := settings.validateFileOutput()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFileOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errContains != "" {
				if !containsString(err.Error(), tt.errContains) {
					t.Errorf("validateFileOutput() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

// TestOutputSettings_validateS3Configuration tests S3 configuration validation
func TestOutputSettings_validateS3Configuration(t *testing.T) {
	tests := []struct {
		name        string
		s3Bucket    S3Output
		wantErr     bool
		errContains string
	}{
		{
			name:     "no S3 configuration",
			s3Bucket: S3Output{},
			wantErr:  false,
		},
		{
			name: "missing S3 client",
			s3Bucket: S3Output{
				Bucket: "test-bucket",
			},
			wantErr:     true,
			errContains: "S3 client is required",
		},
		{
			name: "valid S3 configuration",
			s3Bucket: S3Output{
				S3Client: &s3.Client{},
				Bucket:   "test-bucket",
				Path:     "path/to/file",
			},
			wantErr: false,
		},
		{
			name: "bucket name too short",
			s3Bucket: S3Output{
				S3Client: &s3.Client{},
				Bucket:   "ab",
			},
			wantErr:     true,
			errContains: "S3 bucket name must be between 3 and 63 characters",
		},
		{
			name: "bucket name too long",
			s3Bucket: S3Output{
				S3Client: &s3.Client{},
				Bucket:   "this-is-a-very-long-bucket-name-that-exceeds-the-maximum-allowed-length-for-s3-bucket-names",
			},
			wantErr:     true,
			errContains: "S3 bucket name must be between 3 and 63 characters",
		},
		{
			name: "bucket name with consecutive dots",
			s3Bucket: S3Output{
				S3Client: &s3.Client{},
				Bucket:   "test..bucket",
			},
			wantErr:     true,
			errContains: "S3 bucket name contains invalid character patterns",
		},
		{
			name: "bucket name starting with dot",
			s3Bucket: S3Output{
				S3Client: &s3.Client{},
				Bucket:   ".test-bucket",
			},
			wantErr:     true,
			errContains: "S3 bucket name cannot start or end with dots or hyphens",
		},
		{
			name: "bucket name ending with hyphen",
			s3Bucket: S3Output{
				S3Client: &s3.Client{},
				Bucket:   "test-bucket-",
			},
			wantErr:     true,
			errContains: "S3 bucket name cannot start or end with dots or hyphens",
		},
		{
			name: "S3 path with control characters",
			s3Bucket: S3Output{
				S3Client: &s3.Client{},
				Bucket:   "test-bucket",
				Path:     "path\x00/to/file",
			},
			wantErr:     true,
			errContains: "S3 path contains invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := &OutputSettings{
				S3Bucket: tt.s3Bucket,
			}
			err := settings.validateS3Configuration()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateS3Configuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errContains != "" {
				if !containsString(err.Error(), tt.errContains) {
					t.Errorf("validateS3Configuration() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

// TestOutputSettings_validateSettingCombinations tests incompatible setting combinations
func TestOutputSettings_validateSettingCombinations(t *testing.T) {
	tests := []struct {
		name        string
		settings    *OutputSettings
		wantErr     bool
		errContains string
	}{
		{
			name:     "no conflicts",
			settings: &OutputSettings{},
			wantErr:  false,
		},
		{
			name: "file and S3 output conflict",
			settings: &OutputSettings{
				OutputFile: "output.json",
				S3Bucket: S3Output{
					Bucket: "test-bucket",
				},
			},
			wantErr:     true,
			errContains: "cannot specify both file output and S3 output",
		},
		{
			name: "TableMaxColumnWidth with incompatible format - warning",
			settings: &OutputSettings{
				OutputFormat:        "json",
				TableMaxColumnWidth: 100, // Non-default value
			},
			wantErr:     true, // This generates a warning, which is still an error
			errContains: "TableMaxColumnWidth setting is not applicable for json format",
		},
		{
			name: "TableMaxColumnWidth with table format - valid",
			settings: &OutputSettings{
				OutputFormat:        "table",
				TableMaxColumnWidth: 100,
			},
			wantErr: false,
		},
		{
			name: "SeparateTables with incompatible format - warning",
			settings: &OutputSettings{
				OutputFormat:   "json",
				SeparateTables: true,
			},
			wantErr:     true, // This generates a warning, which is still an error
			errContains: "SeparateTables setting is not applicable for json format",
		},
		{
			name: "SeparateTables with markdown format - valid",
			settings: &OutputSettings{
				OutputFormat:   "markdown",
				SeparateTables: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.settings.validateSettingCombinations()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSettingCombinations() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errContains != "" {
				if !containsString(err.Error(), tt.errContains) {
					t.Errorf("validateSettingCombinations() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

// TestOutputSettings_validateFromToColumns tests FromToColumns validation
func TestOutputSettings_validateFromToColumns(t *testing.T) {
	tests := []struct {
		name          string
		fromToColumns *FromToColumns
		wantErr       bool
		errContains   string
	}{
		{
			name: "valid FromToColumns",
			fromToColumns: &FromToColumns{
				From: "source",
				To:   "target",
			},
			wantErr: false,
		},
		{
			name: "valid FromToColumns with label",
			fromToColumns: &FromToColumns{
				From:  "source",
				To:    "target",
				Label: "relationship",
			},
			wantErr: false,
		},
		{
			name: "empty From field",
			fromToColumns: &FromToColumns{
				From: "",
				To:   "target",
			},
			wantErr:     true,
			errContains: "FromToColumns.From cannot be empty",
		},
		{
			name: "empty To field",
			fromToColumns: &FromToColumns{
				From: "source",
				To:   "",
			},
			wantErr:     true,
			errContains: "FromToColumns.To cannot be empty",
		},
		{
			name: "both fields empty",
			fromToColumns: &FromToColumns{
				From: "",
				To:   "",
			},
			wantErr:     true,
			errContains: "FromToColumns.From cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := &OutputSettings{
				FromToColumns: tt.fromToColumns,
			}
			err := settings.validateFromToColumns()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFromToColumns() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errContains != "" {
				if !containsString(err.Error(), tt.errContains) {
					t.Errorf("validateFromToColumns() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
