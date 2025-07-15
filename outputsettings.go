package format

import (
	"fmt"
	"os"
	"strings"

	"github.com/ArjenSchwarz/go-output/drawio"
	"github.com/ArjenSchwarz/go-output/mermaid"
	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// DefaultTableMaxColumnWidth is the default maximum column width for table output
const DefaultTableMaxColumnWidth = 50

// TableStyles is a lookup map for getting the table styles based on a string
var TableStyles = map[string]table.Style{
	"Default":                    table.StyleDefault,
	"Bold":                       table.StyleBold,
	"ColoredBright":              table.StyleColoredBright,
	"ColoredDark":                table.StyleColoredDark,
	"ColoredBlackOnBlueWhite":    table.StyleColoredBlackOnBlueWhite,
	"ColoredBlackOnCyanWhite":    table.StyleColoredBlackOnCyanWhite,
	"ColoredBlackOnGreenWhite":   table.StyleColoredBlackOnGreenWhite,
	"ColoredBlackOnMagentaWhite": table.StyleColoredBlackOnMagentaWhite,
	"ColoredBlackOnYellowWhite":  table.StyleColoredBlackOnYellowWhite,
	"ColoredBlackOnRedWhite":     table.StyleColoredBlackOnRedWhite,
	"ColoredBlueWhiteOnBlack":    table.StyleColoredBlueWhiteOnBlack,
	"ColoredCyanWhiteOnBlack":    table.StyleColoredCyanWhiteOnBlack,
	"ColoredGreenWhiteOnBlack":   table.StyleColoredGreenWhiteOnBlack,
	"ColoredMagentaWhiteOnBlack": table.StyleColoredMagentaWhiteOnBlack,
	"ColoredRedWhiteOnBlack":     table.StyleColoredRedWhiteOnBlack,
	"ColoredYellowWhiteOnBlack":  table.StyleColoredYellowWhiteOnBlack,
}

type OutputSettings struct {
	// Defines whether a table of contents should be added
	HasTOC bool
	// The header information for a draw.io/diagrams.net CSV import
	DrawIOHeader drawio.Header
	// FrontMatter can be provided for a Markdown output
	FrontMatter map[string]string
	// The columns for graphical interfaces to show how parent-child relationships connect
	FromToColumns *FromToColumns
	// The type of Mermaid diagram
	MermaidSettings *mermaid.Settings
	// The name of the file the output should be saved to
	OutputFile string
	// The format of the output file
	OutputFileFormat string
	// The format of the output that should be used
	OutputFormat string
	// Store the output in the provided S3 bucket
	S3Bucket S3Output
	// For table heavy outputs, should there be extra spacing between tables
	SeparateTables bool
	// Does the output need to be appended to an existing file?
	ShouldAppend bool
	// For columnar outputs (table, html, csv, markdown) split rows with multiple values into separate rows
	SplitLines bool
	// The key the output should be sorted by
	SortKey string
	// For tables, how wide can a table be?
	TableMaxColumnWidth int
	// The style of the table
	TableStyle table.Style
	// The title of the output
	Title string
	// Should colors be shown in the output
	UseColors bool
	// Should emoji be shown in the output
	UseEmoji bool
	// Should progress output be shown
	ProgressEnabled bool
	// ProgressOptions configures the progress indicator
	ProgressOptions ProgressOptions
}

type S3Output struct {
	S3Client *s3.Client
	Bucket   string
	Path     string
}

// FromToColumns is used to set the From and To columns for graphical output formats
type FromToColumns struct {
	From  string
	To    string
	Label string
}

// NewOutputSettings creates and returns a new OutputSettings object with some default values
func NewOutputSettings() *OutputSettings {
	settings := OutputSettings{
		TableStyle:          table.StyleDefault,
		TableMaxColumnWidth: DefaultTableMaxColumnWidth,
		MermaidSettings:     &mermaid.Settings{},
	}
	env := strings.ToLower(os.Getenv("GO_OUTPUT_PROGRESS"))
	if env == "false" {
		settings.ProgressEnabled = false
	} else {
		settings.ProgressEnabled = true
	}
	return &settings
}

// AddFromToColumns sets from to columns for graphical formats
func (settings *OutputSettings) AddFromToColumns(from string, to string) {
	result := FromToColumns{
		From: from,
		To:   to,
	}
	settings.FromToColumns = &result
}

// AddFromToColumns sets from to columns for graphical formats
func (settings *OutputSettings) AddFromToColumnsWithLabel(from string, to string, label string) {
	result := FromToColumns{
		From:  from,
		To:    to,
		Label: label,
	}
	settings.FromToColumns = &result
}

// SetOutputFormat sets the expected output format
func (settings *OutputSettings) SetOutputFormat(format string) {
	settings.OutputFormat = strings.ToLower(format)
}

func (settings *OutputSettings) GetDefaultExtension() string {
	switch settings.OutputFormat {
	case "markdown":
		return ".md"
	case "table":
		return ".txt"
	default:
		return "." + settings.OutputFormat
	}
}

func (settings *OutputSettings) SetS3Bucket(client *s3.Client, bucket string, path string) {
	settings.S3Bucket = S3Output{
		S3Client: client,
		Bucket:   bucket,
		Path:     path,
	}
}

// NeedsFromToColumns verifies if a format requires from and to columns to be set
func (settings *OutputSettings) NeedsFromToColumns() bool {
	if settings.OutputFormat == "dot" || settings.OutputFormat == "mermaid" {
		return true
	}
	return false
}

func (settings *OutputSettings) GetSeparator() string {
	switch settings.OutputFormat {
	case "table":
		return "\n"
	case "markdown":
		return "\n"
	case "csv":
		return "\n"
	case "dot":
		return ","
	default:
		return ", "
	}
}

// EnableProgress turns on progress output.
func (settings *OutputSettings) EnableProgress() {
	settings.ProgressEnabled = true
}

// DisableProgress turns off progress output.
func (settings *OutputSettings) DisableProgress() {
	settings.ProgressEnabled = false
}

// Validate performs comprehensive validation of OutputSettings
func (settings *OutputSettings) Validate() error {
	// Start with simple validation to avoid potential issues
	if err := settings.validateOutputFormat(); err != nil {
		return err
	}

	// Validate format-specific requirements
	if err := settings.validateFormatRequirements(); err != nil {
		return err
	}

	// Validate file output configuration
	if err := settings.validateFileOutput(); err != nil {
		return err
	}

	// Validate S3 configuration
	if err := settings.validateS3Configuration(); err != nil {
		return err
	}

	// Validate incompatible setting combinations
	if err := settings.validateSettingCombinations(); err != nil {
		return err
	}

	return nil
}

// validateOutputFormat validates the output format is supported
func (settings *OutputSettings) validateOutputFormat() error {
	if settings.OutputFormat == "" {
		// Empty format defaults to JSON, which is valid
		return nil
	}

	validFormats := []string{"json", "csv", "html", "table", "markdown", "mermaid", "drawio", "dot", "yaml"}
	for _, format := range validFormats {
		if settings.OutputFormat == format {
			return nil
		}
	}

	return NewErrorBuilder(ErrInvalidFormat, fmt.Sprintf("invalid output format: %s", settings.OutputFormat)).
		WithField("OutputFormat").
		WithValue(settings.OutputFormat).
		WithOperation("format validation").
		WithSuggestions(
			fmt.Sprintf("Valid formats: %s", strings.Join(validFormats, ", ")),
			"Use SetOutputFormat() to set a valid format",
		).
		Build()
}

// validateFormatRequirements validates format-specific requirements
func (settings *OutputSettings) validateFormatRequirements() error {
	composite := NewCompositeError()

	switch settings.OutputFormat {
	case "mermaid":
		if err := settings.validateMermaidRequirements(); err != nil {
			composite.Add(err)
		}
	case "dot":
		if err := settings.validateDotRequirements(); err != nil {
			composite.Add(err)
		}
	case "drawio":
		if err := settings.validateDrawIORequirements(); err != nil {
			composite.Add(err)
		}
	}

	return composite.ErrorOrNil()
}

// validateMermaidRequirements validates mermaid format requirements
func (settings *OutputSettings) validateMermaidRequirements() error {
	if settings.FromToColumns == nil && settings.MermaidSettings == nil {
		return NewErrorBuilder(ErrMissingRequired, "mermaid format requires FromToColumns or MermaidSettings configuration").
			WithField("FromToColumns/MermaidSettings").
			WithOperation("mermaid validation").
			WithSuggestions(
				"Use AddFromToColumns() to set source and target columns for relationship diagrams",
				"Or configure MermaidSettings for chart generation",
				"Example: settings.AddFromToColumns(\"source\", \"target\")",
			).
			Build()
	}

	// Validate FromToColumns if present
	if settings.FromToColumns != nil {
		if err := settings.validateFromToColumns(); err != nil {
			return err
		}
	}

	return nil
}

// validateDotRequirements validates dot format requirements
func (settings *OutputSettings) validateDotRequirements() error {
	if settings.FromToColumns == nil {
		return NewErrorBuilder(ErrMissingRequired, "dot format requires FromToColumns configuration").
			WithField("FromToColumns").
			WithOperation("dot validation").
			WithSuggestions(
				"Use AddFromToColumns() to set source and target columns",
				"Example: settings.AddFromToColumns(\"from\", \"to\")",
			).
			Build()
	}

	return settings.validateFromToColumns()
}

// validateDrawIORequirements validates draw.io format requirements
func (settings *OutputSettings) validateDrawIORequirements() error {
	if !settings.DrawIOHeader.IsSet() {
		return NewErrorBuilder(ErrMissingRequired, "drawio format requires DrawIOHeader configuration").
			WithField("DrawIOHeader").
			WithOperation("drawio validation").
			WithSuggestions(
				"Configure DrawIOHeader with appropriate settings for CSV import",
				"Set header information for draw.io/diagrams.net compatibility",
			).
			Build()
	}

	return nil
}

// validateFromToColumns validates FromToColumns configuration
func (settings *OutputSettings) validateFromToColumns() error {
	if settings.FromToColumns.From == "" {
		return NewErrorBuilder(ErrMissingRequired, "FromToColumns.From cannot be empty").
			WithField("FromToColumns.From").
			WithOperation("from-to columns validation").
			WithSuggestions(
				"Specify the source column name",
				"Example: settings.AddFromToColumns(\"source\", \"target\")",
			).
			Build()
	}

	if settings.FromToColumns.To == "" {
		return NewErrorBuilder(ErrMissingRequired, "FromToColumns.To cannot be empty").
			WithField("FromToColumns.To").
			WithOperation("from-to columns validation").
			WithSuggestions(
				"Specify the target column name",
				"Example: settings.AddFromToColumns(\"source\", \"target\")",
			).
			Build()
	}

	return nil
}

// validateFileOutput validates file output configuration
func (settings *OutputSettings) validateFileOutput() error {
	if settings.OutputFile == "" {
		return nil // No file output configured, nothing to validate
	}

	composite := NewCompositeError()

	// Validate file path
	if err := settings.validateFilePath(); err != nil {
		composite.Add(err)
	}

	// Validate file format compatibility
	if err := settings.validateFileFormat(); err != nil {
		composite.Add(err)
	}

	return composite.ErrorOrNil()
}

// validateFilePath validates the output file path
func (settings *OutputSettings) validateFilePath() error {
	// Check for invalid characters in file path
	invalidChars := []string{"\x00", "\x01", "\x02", "\x03", "\x04", "\x05", "\x06", "\x07", "\x08", "\x0b", "\x0c", "\x0e", "\x0f"}
	for _, char := range invalidChars {
		if strings.Contains(settings.OutputFile, char) {
			return NewErrorBuilder(ErrInvalidFilePath, "file path contains invalid characters").
				WithField("OutputFile").
				WithValue(settings.OutputFile).
				WithOperation("file path validation").
				WithSuggestions(
					"Remove control characters from the file path",
					"Use only printable characters in file names",
				).
				Build()
		}
	}

	// Check for empty file name
	if strings.TrimSpace(settings.OutputFile) == "" {
		return NewErrorBuilder(ErrInvalidFilePath, "file path cannot be empty or whitespace only").
			WithField("OutputFile").
			WithValue(settings.OutputFile).
			WithOperation("file path validation").
			WithSuggestions(
				"Provide a valid file path",
				"Example: \"output.json\" or \"/path/to/output.csv\"",
			).
			Build()
	}

	return nil
}

// validateFileFormat validates file format compatibility
func (settings *OutputSettings) validateFileFormat() error {
	if settings.OutputFileFormat == "" {
		return nil // Will default to OutputFormat, which is already validated
	}

	// Validate OutputFileFormat is a supported format
	validFormats := []string{"json", "csv", "html", "table", "markdown", "mermaid", "drawio", "dot", "yaml"}
	isValid := false
	for _, format := range validFormats {
		if settings.OutputFileFormat == format {
			isValid = true
			break
		}
	}

	if !isValid {
		return NewErrorBuilder(ErrInvalidFormat, fmt.Sprintf("invalid output file format: %s", settings.OutputFileFormat)).
			WithField("OutputFileFormat").
			WithValue(settings.OutputFileFormat).
			WithOperation("file format validation").
			WithSuggestions(
				fmt.Sprintf("Valid file formats: %s", strings.Join(validFormats, ", ")),
				"Leave OutputFileFormat empty to use the same format as OutputFormat",
			).
			Build()
	}

	return nil
}

// validateS3Configuration validates S3 output configuration
func (settings *OutputSettings) validateS3Configuration() error {
	if settings.S3Bucket.Bucket == "" {
		return nil // No S3 output configured, nothing to validate
	}

	composite := NewCompositeError()

	// Validate S3 client
	if settings.S3Bucket.S3Client == nil {
		composite.Add(NewErrorBuilder(ErrInvalidS3Config, "S3 client is required when S3 bucket is specified").
			WithField("S3Bucket.S3Client").
			WithOperation("S3 validation").
			WithSuggestions(
				"Initialize S3 client before setting S3 bucket",
				"Use SetS3Bucket() method with a valid S3 client",
			).
			Build())
	}

	// Validate bucket name
	if err := settings.validateS3BucketName(); err != nil {
		composite.Add(err)
	}

	// Validate S3 path
	if err := settings.validateS3Path(); err != nil {
		composite.Add(err)
	}

	return composite.ErrorOrNil()
}

// validateS3BucketName validates S3 bucket name according to AWS rules
func (settings *OutputSettings) validateS3BucketName() error {
	bucket := settings.S3Bucket.Bucket

	// Basic length check
	if len(bucket) < 3 || len(bucket) > 63 {
		return NewErrorBuilder(ErrInvalidS3Config, "S3 bucket name must be between 3 and 63 characters").
			WithField("S3Bucket.Bucket").
			WithValue(bucket).
			WithOperation("S3 bucket validation").
			WithSuggestions(
				"Use a bucket name between 3 and 63 characters",
				"Follow AWS S3 bucket naming conventions",
			).
			Build()
	}

	// Check for invalid characters and patterns
	if strings.Contains(bucket, "..") || strings.Contains(bucket, ".-") || strings.Contains(bucket, "-.") {
		return NewErrorBuilder(ErrInvalidS3Config, "S3 bucket name contains invalid character patterns").
			WithField("S3Bucket.Bucket").
			WithValue(bucket).
			WithOperation("S3 bucket validation").
			WithSuggestions(
				"Avoid consecutive dots (..) and dot-dash (.-) or dash-dot (-.) patterns",
				"Use only lowercase letters, numbers, dots, and hyphens",
			).
			Build()
	}

	// Check start/end characters
	if strings.HasPrefix(bucket, ".") || strings.HasSuffix(bucket, ".") ||
		strings.HasPrefix(bucket, "-") || strings.HasSuffix(bucket, "-") {
		return NewErrorBuilder(ErrInvalidS3Config, "S3 bucket name cannot start or end with dots or hyphens").
			WithField("S3Bucket.Bucket").
			WithValue(bucket).
			WithOperation("S3 bucket validation").
			WithSuggestions(
				"Start and end bucket names with lowercase letters or numbers",
				"Remove leading/trailing dots and hyphens",
			).
			Build()
	}

	return nil
}

// validateS3Path validates S3 object path
func (settings *OutputSettings) validateS3Path() error {
	if settings.S3Bucket.Path == "" {
		return nil // Empty path is valid (will use root)
	}

	path := settings.S3Bucket.Path

	// Check for invalid characters
	invalidChars := []string{"\x00", "\x01", "\x02", "\x03", "\x04", "\x05", "\x06", "\x07", "\x08", "\x0b", "\x0c", "\x0e", "\x0f"}
	for _, char := range invalidChars {
		if strings.Contains(path, char) {
			return NewErrorBuilder(ErrInvalidS3Config, "S3 path contains invalid characters").
				WithField("S3Bucket.Path").
				WithValue(path).
				WithOperation("S3 path validation").
				WithSuggestions(
					"Remove control characters from the S3 path",
					"Use only printable characters in S3 object keys",
				).
				Build()
		}
	}

	return nil
}

// validateSettingCombinations validates incompatible setting combinations
func (settings *OutputSettings) validateSettingCombinations() error {
	// Validate file output and S3 output combination
	if settings.OutputFile != "" && settings.S3Bucket.Bucket != "" {
		return NewErrorBuilder(ErrIncompatibleConfig, "cannot specify both file output and S3 output").
			WithField("OutputFile/S3Bucket").
			WithOperation("setting combination validation").
			WithSuggestions(
				"Choose either file output or S3 output, not both",
				"Remove OutputFile to use S3 output only",
				"Remove S3Bucket configuration to use file output only",
			).
			Build()
	}

	// Validate table-specific settings with non-table formats
	if settings.OutputFormat != "table" && settings.OutputFormat != "html" && settings.OutputFormat != "markdown" {
		// Only warn if TableMaxColumnWidth has been explicitly set to a non-default value
		if settings.TableMaxColumnWidth != 0 && settings.TableMaxColumnWidth != DefaultTableMaxColumnWidth {
			return NewErrorBuilder(ErrIncompatibleConfig, fmt.Sprintf("TableMaxColumnWidth setting is not applicable for %s format", settings.OutputFormat)).
				WithField("TableMaxColumnWidth").
				WithValue(settings.TableMaxColumnWidth).
				WithOperation("setting combination validation").
				WithSuggestions(
					fmt.Sprintf("TableMaxColumnWidth only applies to table, html, and markdown formats, not %s", settings.OutputFormat),
					"Remove TableMaxColumnWidth setting or change to a compatible format",
				).
				WithSeverity(SeverityWarning).
				Build()
		}
	}

	// Validate SeparateTables with non-applicable formats
	if settings.SeparateTables && settings.OutputFormat != "table" && settings.OutputFormat != "html" && settings.OutputFormat != "markdown" {
		return NewErrorBuilder(ErrIncompatibleConfig, fmt.Sprintf("SeparateTables setting is not applicable for %s format", settings.OutputFormat)).
			WithField("SeparateTables").
			WithValue(settings.SeparateTables).
			WithOperation("setting combination validation").
			WithSuggestions(
				fmt.Sprintf("SeparateTables only applies to table, html, and markdown formats, not %s", settings.OutputFormat),
				"Set SeparateTables to false or change to a compatible format",
			).
			WithSeverity(SeverityWarning).
			Build()
	}

	return nil
}
