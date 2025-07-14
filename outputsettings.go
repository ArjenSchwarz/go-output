package format

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ArjenSchwarz/go-output/drawio"
	"github.com/ArjenSchwarz/go-output/errors"
	"github.com/ArjenSchwarz/go-output/mermaid"
	"github.com/jedib0t/go-pretty/v6/table"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

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
		TableMaxColumnWidth: 50,
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
	composite := errors.NewCompositeError()

	// Validate output format
	if err := settings.validateOutputFormat(); err != nil {
		composite.Add(err.(errors.ValidationError))
	}

	// Validate format-specific requirements
	if err := settings.validateFormatSpecificRequirements(); err != nil {
		composite.Add(err.(errors.ValidationError))
	}

	// Validate file output configuration
	if err := settings.validateFileOutput(); err != nil {
		composite.Add(err.(errors.ValidationError))
	}

	// Validate S3 configuration
	if err := settings.validateS3Configuration(); err != nil {
		composite.Add(err.(errors.ValidationError))
	}

	return composite.ErrorOrNil()
}

// validateOutputFormat validates the output format is supported
func (settings *OutputSettings) validateOutputFormat() error {
	validFormats := []string{
		"json", "csv", "table", "html", "markdown", "yaml",
		"mermaid", "drawio", "dot",
	}

	if settings.OutputFormat == "" {
		return errors.NewValidationError(
			errors.ErrMissingRequired,
			"OutputFormat is required",
		).WithSuggestions(
			fmt.Sprintf("Valid formats: %s", strings.Join(validFormats, ", ")),
		)
	}

	for _, valid := range validFormats {
		if settings.OutputFormat == valid {
			return nil
		}
	}

	return errors.NewValidationError(
		errors.ErrInvalidFormat,
		fmt.Sprintf("Invalid output format: %s", settings.OutputFormat),
	).WithSuggestions(
		fmt.Sprintf("Valid formats: %s", strings.Join(validFormats, ", ")),
	).WithContext(errors.ErrorContext{
		Operation: "format_validation",
		Field:     "OutputFormat",
		Value:     settings.OutputFormat,
	})
}

// validateFormatSpecificRequirements validates format-specific requirements
func (settings *OutputSettings) validateFormatSpecificRequirements() error {
	switch settings.OutputFormat {
	case "mermaid":
		if settings.FromToColumns == nil && settings.MermaidSettings == nil {
			return errors.NewValidationError(
				errors.ErrMissingRequired,
				"mermaid format requires FromToColumns or MermaidSettings",
			).WithSuggestions(
				"Use AddFromToColumns() to set source and target columns",
				"Or configure MermaidSettings for chart generation",
			)
		}
	case "drawio":
		if !settings.DrawIOHeader.IsSet() {
			return errors.NewValidationError(
				errors.ErrMissingRequired,
				"drawio format requires DrawIOHeader configuration",
			).WithSuggestions(
				"Configure DrawIOHeader before using drawio format",
			)
		}
	case "dot":
		if settings.FromToColumns == nil {
			return errors.NewValidationError(
				errors.ErrMissingRequired,
				"dot format requires FromToColumns configuration",
			).WithSuggestions(
				"Use AddFromToColumns() to set source and target columns",
			)
		}
	}
	return nil
}

// validateFileOutput validates file output configuration
func (settings *OutputSettings) validateFileOutput() error {
	if settings.OutputFile == "" {
		return nil // No file output, nothing to validate
	}

	// Check if directory exists and is writable
	if dir := filepath.Dir(settings.OutputFile); dir != "." {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return errors.NewValidationError(
				errors.ErrInvalidFilePath,
				fmt.Sprintf("Output directory does not exist: %s", dir),
			).WithSuggestions(
				"Create the directory before running the command",
				"Use a different output path",
			).WithContext(errors.ErrorContext{
				Operation: "file_validation",
				Field:     "OutputFile",
				Value:     settings.OutputFile,
			})
		}
	}

	return nil
}

// validateS3Configuration validates S3 bucket configuration
func (settings *OutputSettings) validateS3Configuration() error {
	if settings.S3Bucket.Bucket == "" {
		return nil // No S3 output, nothing to validate
	}

	if settings.S3Bucket.Path == "" {
		return errors.NewValidationError(
			errors.ErrMissingRequired,
			"S3 path is required when bucket is specified",
		).WithSuggestions(
			"Specify the S3 object key/path",
		).WithContext(errors.ErrorContext{
			Operation: "s3_validation",
			Field:     "S3Bucket.Path",
		})
	}

	if settings.S3Bucket.S3Client == nil {
		return errors.NewValidationError(
			errors.ErrMissingRequired,
			"S3Client is required when bucket is specified",
		).WithSuggestions(
			"Initialize S3Client before setting bucket configuration",
		).WithContext(errors.ErrorContext{
			Operation: "s3_validation",
			Field:     "S3Bucket.S3Client",
		})
	}

	return nil
}
