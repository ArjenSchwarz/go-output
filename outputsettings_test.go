package format

import (
	"testing"

	"github.com/ArjenSchwarz/go-output/drawio"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jedib0t/go-pretty/v6/table"
)

// TestNewOutputSettings verifies the default values returned by
// NewOutputSettings and that progress output can be disabled through the
// environment variable.
func TestNewOutputSettings(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		t.Setenv("GO_OUTPUT_PROGRESS", "")
		got := NewOutputSettings()
		if got.TableStyle.Name != table.StyleDefault.Name {
			t.Errorf("expected default table style")
		}
		if got.TableMaxColumnWidth != 50 {
			t.Errorf("expected default max width 50, got %d", got.TableMaxColumnWidth)
		}
		if got.MermaidSettings == nil {
			t.Errorf("expected MermaidSettings to be initialised")
		}
		if !got.ProgressEnabled {
			t.Errorf("progress should be enabled by default")
		}
	})

	t.Run("disabled via env", func(t *testing.T) {
		t.Setenv("GO_OUTPUT_PROGRESS", "false")
		got := NewOutputSettings()
		if got.ProgressEnabled {
			t.Errorf("progress should be disabled when env var is false")
		}
	})
}

// TestOutputSettings_AddFromToColumns verifies that calling AddFromToColumns
// stores the provided source and destination column names.
func TestOutputSettings_AddFromToColumns(t *testing.T) {
	type fields struct {
		DrawIOHeader        drawio.Header
		FromToColumns       *FromToColumns
		OutputFile          string
		OutputFormat        string
		SeparateTables      bool
		ShouldAppend        bool
		SortKey             string
		TableMaxColumnWidth int
		TableStyle          table.Style
		Title               string
		UseColors           bool
		UseEmoji            bool
	}
	type args struct {
		from string
		to   string
	}
	basicresult := FromToColumns{
		From: "Source",
		To:   "Target",
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{"Add columns", fields{FromToColumns: &basicresult}, args{from: "Source", to: "Target"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := &OutputSettings{
				DrawIOHeader:        tt.fields.DrawIOHeader,
				FromToColumns:       tt.fields.FromToColumns,
				OutputFile:          tt.fields.OutputFile,
				OutputFormat:        tt.fields.OutputFormat,
				SeparateTables:      tt.fields.SeparateTables,
				ShouldAppend:        tt.fields.ShouldAppend,
				SortKey:             tt.fields.SortKey,
				TableMaxColumnWidth: tt.fields.TableMaxColumnWidth,
				TableStyle:          tt.fields.TableStyle,
				Title:               tt.fields.Title,
				UseColors:           tt.fields.UseColors,
				UseEmoji:            tt.fields.UseEmoji,
			}
			settings.AddFromToColumns(tt.args.from, tt.args.to)
			if settings.FromToColumns.From != tt.args.from {
				t.Errorf("OutputSettings.AddFromToColumns(From) = \r\n%v, want \r\n%v", string(settings.FromToColumns.From), string(tt.args.from))
			}
			if settings.FromToColumns.To != tt.args.to {
				t.Errorf("OutputSettings.AddFromToColumns(To) = \r\n%v, want \r\n%v", string(settings.FromToColumns.To), string(tt.args.to))
			}
		})
	}
}

// TestOutputSettings_SetOutputFormat checks that SetOutputFormat normalises the
// provided string and stores it in the OutputSettings.
func TestOutputSettings_SetOutputFormat(t *testing.T) {
	type fields struct {
		DrawIOHeader        drawio.Header
		FromToColumns       *FromToColumns
		OutputFile          string
		OutputFormat        string
		SeparateTables      bool
		ShouldAppend        bool
		SortKey             string
		TableMaxColumnWidth int
		TableStyle          table.Style
		Title               string
		UseColors           bool
		UseEmoji            bool
	}
	type args struct {
		format string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{"Test lowercase", fields{OutputFormat: "fake"}, args{format: "fake"}},
		{"Test uppercase converted", fields{OutputFormat: "FAKE"}, args{format: "fake"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := &OutputSettings{
				DrawIOHeader:        tt.fields.DrawIOHeader,
				FromToColumns:       tt.fields.FromToColumns,
				OutputFile:          tt.fields.OutputFile,
				OutputFormat:        tt.fields.OutputFormat,
				SeparateTables:      tt.fields.SeparateTables,
				ShouldAppend:        tt.fields.ShouldAppend,
				SortKey:             tt.fields.SortKey,
				TableMaxColumnWidth: tt.fields.TableMaxColumnWidth,
				TableStyle:          tt.fields.TableStyle,
				Title:               tt.fields.Title,
				UseColors:           tt.fields.UseColors,
				UseEmoji:            tt.fields.UseEmoji,
			}
			settings.SetOutputFormat(tt.args.format)
			if settings.OutputFormat != tt.args.format {
				t.Errorf("OutputSettings.SetOutputFormat() = \r\n%v, want \r\n%v", string(settings.OutputFormat), string(tt.args.format))
			}
		})
	}
}

// TestOutputSettings_NeedsFromToColumns verifies which output formats require
// the presence of from/to column information.
func TestOutputSettings_NeedsFromToColumns(t *testing.T) {
	type fields struct {
		DrawIOHeader        drawio.Header
		FromToColumns       *FromToColumns
		OutputFile          string
		OutputFormat        string
		SeparateTables      bool
		ShouldAppend        bool
		SortKey             string
		TableMaxColumnWidth int
		TableStyle          table.Style
		Title               string
		UseColors           bool
		UseEmoji            bool
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{"Is dot", fields{OutputFormat: "dot"}, true},
		{"Is mermaid", fields{OutputFormat: "mermaid"}, true},
		{"Is json", fields{OutputFormat: "json"}, false},
		{"Is table", fields{OutputFormat: "table"}, false},
		{"Is csv", fields{OutputFormat: "csv"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := &OutputSettings{
				DrawIOHeader:        tt.fields.DrawIOHeader,
				FromToColumns:       tt.fields.FromToColumns,
				OutputFile:          tt.fields.OutputFile,
				OutputFormat:        tt.fields.OutputFormat,
				SeparateTables:      tt.fields.SeparateTables,
				ShouldAppend:        tt.fields.ShouldAppend,
				SortKey:             tt.fields.SortKey,
				TableMaxColumnWidth: tt.fields.TableMaxColumnWidth,
				TableStyle:          tt.fields.TableStyle,
				Title:               tt.fields.Title,
				UseColors:           tt.fields.UseColors,
				UseEmoji:            tt.fields.UseEmoji,
			}
			if got := settings.NeedsFromToColumns(); got != tt.want {
				t.Errorf("OutputSettings.NeedsFromToColumns() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestOutputSettings_GetSeparator ensures that the correct value separator is
// returned for each supported output format.
func TestOutputSettings_GetSeparator(t *testing.T) {
	type fields struct {
		DrawIOHeader        drawio.Header
		FromToColumns       *FromToColumns
		OutputFile          string
		OutputFormat        string
		SeparateTables      bool
		ShouldAppend        bool
		SortKey             string
		TableMaxColumnWidth int
		TableStyle          table.Style
		Title               string
		UseColors           bool
		UseEmoji            bool
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{"Is dot", fields{OutputFormat: "dot"}, ","},
		{"Is mermaid", fields{OutputFormat: "mermaid"}, ", "},
		{"Is json", fields{OutputFormat: "json"}, ", "},
		{"Is table", fields{OutputFormat: "table"}, "\n"},
		{"Is markdown", fields{OutputFormat: "markdown"}, "\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := &OutputSettings{
				DrawIOHeader:        tt.fields.DrawIOHeader,
				FromToColumns:       tt.fields.FromToColumns,
				OutputFile:          tt.fields.OutputFile,
				OutputFormat:        tt.fields.OutputFormat,
				SeparateTables:      tt.fields.SeparateTables,
				ShouldAppend:        tt.fields.ShouldAppend,
				SortKey:             tt.fields.SortKey,
				TableMaxColumnWidth: tt.fields.TableMaxColumnWidth,
				TableStyle:          tt.fields.TableStyle,
				Title:               tt.fields.Title,
				UseColors:           tt.fields.UseColors,
				UseEmoji:            tt.fields.UseEmoji,
			}
			if got := settings.GetSeparator(); got != tt.want {
				t.Errorf("OutputSettings.GetSeparator() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestOutputSettings_AddFromToColumnsWithLabel verifies that from, to and label
// values are stored when provided together.
func TestOutputSettings_AddFromToColumnsWithLabel(t *testing.T) {
	settings := &OutputSettings{}
	settings.AddFromToColumnsWithLabel("src", "dst", "label")
	if settings.FromToColumns == nil {
		t.Fatalf("FromToColumns should not be nil")
	}
	if settings.FromToColumns.From != "src" || settings.FromToColumns.To != "dst" || settings.FromToColumns.Label != "label" {
		t.Errorf("unexpected FromToColumns %+v", settings.FromToColumns)
	}
}

// TestOutputSettings_GetDefaultExtension confirms that GetDefaultExtension
// returns the expected file extension for a given format.
func TestOutputSettings_GetDefaultExtension(t *testing.T) {
	tests := []struct {
		name   string
		format string
		want   string
	}{
		{"markdown", "markdown", ".md"},
		{"table", "table", ".txt"},
		{"json", "json", ".json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := &OutputSettings{OutputFormat: tt.format}
			if got := settings.GetDefaultExtension(); got != tt.want {
				t.Errorf("GetDefaultExtension() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestOutputSettings_SetS3Bucket ensures that the S3 bucket configuration is
// correctly stored on the settings struct.
func TestOutputSettings_SetS3Bucket(t *testing.T) {
	client := &s3.Client{}
	settings := &OutputSettings{}
	settings.SetS3Bucket(client, "bucket", "path")
	if settings.S3Bucket.S3Client != client || settings.S3Bucket.Bucket != "bucket" || settings.S3Bucket.Path != "path" {
		t.Errorf("unexpected S3Bucket %+v", settings.S3Bucket)
	}
}

// TestOutputSettings_EnableDisableProgress verifies that enabling and disabling
// progress updates the ProgressEnabled flag accordingly.
func TestOutputSettings_EnableDisableProgress(t *testing.T) {
	s := &OutputSettings{}
	s.ProgressEnabled = false
	s.EnableProgress()
	if !s.ProgressEnabled {
		t.Errorf("expected progress enabled")
	}
	s.DisableProgress()
	if s.ProgressEnabled {
		t.Errorf("expected progress disabled")
	}
}

// TestTableStyles asserts that the custom style map contains the expected
// entry referencing go-pretty's Bold style.
func TestTableStyles(t *testing.T) {
	if style, ok := TableStyles["Bold"]; !ok || style.Name != table.StyleBold.Name {
		t.Errorf("expected Bold style to match table.StyleBold")
	}
}
