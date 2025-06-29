package format

import (
	"bytes"
	"os"
	"testing"
)

func resetGlobals() {
	buffer.Reset()
	toc = nil
}

func TestOutputArray_AddHeader(t *testing.T) {
	tests := []struct {
		format   string
		expected string
		toc      string
	}{
		{"html", "<h2 id='header-example'>Header Example</h2>\n", "<a href='#header-example'>Header Example</a>"},
		{"table", "\nHeader Example\n", ""},
		{"markdown", "## Header Example\n", "[Header Example](#header-example)"},
	}
	for _, tt := range tests {
		resetGlobals()
		s := NewOutputSettings()
		s.OutputFormat = tt.format
		oa := OutputArray{Settings: s}
		oa.AddHeader("Header Example")
		if got := buffer.String(); got != tt.expected {
			t.Errorf("format %s got %q want %q", tt.format, got, tt.expected)
		}
		if tt.toc != "" {
			if len(toc) != 1 || toc[0] != tt.toc {
				t.Errorf("format %s toc got %v want %v", tt.format, toc, tt.toc)
			}
		} else if len(toc) != 0 {
			t.Errorf("format %s expected no toc entries", tt.format)
		}
	}
}

func TestOutputArray_AddToBuffer(t *testing.T) {
	s := NewOutputSettings()
	s.OutputFormat = "csv"
	oa := OutputArray{Settings: s, Keys: []string{"Name"}}
	oa.AddContents(map[string]interface{}{"Name": "A"})
	resetGlobals()
	oa.AddToBuffer()
	if !bytes.Equal(buffer.Bytes(), oa.toCSV()) {
		t.Errorf("csv AddToBuffer output mismatch")
	}
	s.OutputFormat = "table"
	buffer.Reset()
	oa.Settings = s
	oa.AddToBuffer()
	if !bytes.Equal(buffer.Bytes(), oa.toTable()) {
		t.Errorf("table AddToBuffer output mismatch")
	}
}

func TestOutputArray_HtmlTableOnly_NotEmpty(t *testing.T) {
	s := NewOutputSettings()
	s.OutputFormat = "html"
	oa := OutputArray{Settings: s, Keys: []string{"Name"}}
	oa.AddContents(map[string]interface{}{"Name": "item"})
	out := oa.HtmlTableOnly()
	if len(out) == 0 {
		t.Fatalf("expected html output")
	}
	if !bytes.Contains(out, []byte("<table")) {
		t.Errorf("expected html table content")
	}
}

func TestOutputArray_KeysAsInterface(t *testing.T) {
	oa := OutputArray{Keys: []string{"A", "B"}}
	got := oa.KeysAsInterface()
	if len(got) != 2 || got[0] != "A" || got[1] != "B" {
		t.Errorf("unexpected result %v", got)
	}
}

func TestOutputArray_ContentsAsInterfaces(t *testing.T) {
	s := NewOutputSettings()
	oa := OutputArray{Settings: s, Keys: []string{"A", "B"}}
	oa.AddContents(map[string]interface{}{"A": "one", "B": 2})
	result := oa.ContentsAsInterfaces()
	if len(result) != 1 || len(result[0]) != 2 || result[0][0] != "one" || result[0][1] != "2" {
		t.Errorf("unexpected contents %v", result)
	}
}

func TestOutputArray_AddHolderSorting(t *testing.T) {
	s := NewOutputSettings()
	s.SortKey = "name"
	oa := OutputArray{Settings: s, Keys: []string{"name"}}
	oa.AddContents(map[string]interface{}{"name": "b"})
	oa.AddContents(map[string]interface{}{"name": "a"})
	if oa.Contents[0].Contents["name"] != "a" {
		t.Errorf("expected sorted order, got %v", oa.Contents[0])
	}
}

func TestFormatNumber(t *testing.T) {
	if v := formatNumber(42); v != "42" {
		t.Errorf("expected 42 got %s", v)
	}
}

func TestPrintByteSlice_File(t *testing.T) {
	tmp, err := os.CreateTemp("", "out.txt")
	if err != nil {
		t.Fatalf("tempfile: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(tmp.Name()) })
	if err := PrintByteSlice([]byte("\x1b[31mhi\x1b[0m"), tmp.Name(), S3Output{}); err != nil {
		t.Fatalf("PrintByteSlice returned error %v", err)
	}
	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != "hi" {
		t.Errorf("file contents %q", data)
	}
}

func TestActiveProgressRegisterStop(t *testing.T) {
	nop := newNoOpProgress(NewOutputSettings())
	registerActiveProgress(nop)
	if activeProgress == nil {
		t.Fatalf("expected active progress")
	}
	stopActiveProgress()
	if activeProgress != nil {
		t.Errorf("expected nil after stop")
	}
}
