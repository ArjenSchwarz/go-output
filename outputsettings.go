package format

import (
	"strings"

	"github.com/ArjenSchwarz/go-output/drawio"
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
	// The columns for graphical interfaces to show how parent-child relationships connect
	FromToColumns *FromToColumns
	// The type of Mermaid diagram
	MermaidSettings *mermaid.Settings
	// The name of the file the output should be saved to
	OutputFile string
	// The format of the output that should be used
	OutputFormat string
	// Store the output in the provided S3 bucket
	S3Bucket S3Output
	// For table heavy outputs, should there be extra spacing between tables
	SeparateTables bool
	// Does the output need to be appended to an existing file?
	ShouldAppend bool
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
	case "dot":
		return ","
	default:
		return ", "
	}
}
