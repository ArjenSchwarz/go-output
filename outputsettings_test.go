package format

import (
	"reflect"
	"testing"

	"github.com/ArjenSchwarz/go-output/drawio"
	"github.com/jedib0t/go-pretty/v6/table"
)

func TestNewOutputSettings(t *testing.T) {
	tests := []struct {
		name string
		want *OutputSettings
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewOutputSettings(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewOutputSettings() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
