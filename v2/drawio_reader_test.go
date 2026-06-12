package output

import (
	"encoding/csv"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// mustParseDrawIO parses input and fails the test on error.
func mustParseDrawIO(t *testing.T, input string) *ParsedDrawIO {
	t.Helper()
	got, err := ParseDrawIOCSV(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseDrawIOCSV() error = %v, want nil", err)
	}
	if got == nil {
		t.Fatal("ParseDrawIOCSV() = nil, want non-nil result")
	}
	return got
}

// TestParseDrawIOCSV_DirectiveKeys verifies every directive key the v2
// renderer emits maps onto its DrawIOHeader field (requirement 2.1, 2.2).
func TestParseDrawIOCSV_DirectiveKeys(t *testing.T) {
	tests := map[string]struct {
		directives string
		want       DrawIOHeader
	}{
		"label": {
			directives: "# label: %Name%\n",
			want:       DrawIOHeader{Label: "%Name%"},
		},
		"style": {
			directives: "# style: %Image%\n",
			want:       DrawIOHeader{Style: "%Image%"},
		},
		"identity": {
			directives: "# identity: ID\n",
			want:       DrawIOHeader{Identity: "ID"},
		},
		"parent": {
			directives: "# parent: Parent\n",
			want:       DrawIOHeader{Parent: "Parent"},
		},
		"parentstyle": {
			directives: "# parentstyle: rounded=1\n",
			want:       DrawIOHeader{ParentStyle: "rounded=1"},
		},
		"namespace": {
			directives: "# namespace: csvimport-\n",
			want:       DrawIOHeader{Namespace: "csvimport-"},
		},
		"connect": {
			directives: "# connect: {\"from\":\"From\",\"to\":\"Name\",\"invert\":true,\"label\":\"Uses\",\"style\":\"curved=1\"}\n",
			want: DrawIOHeader{Connections: []DrawIOConnection{
				{From: "From", To: "Name", Invert: true, Label: "Uses", Style: "curved=1"},
			}},
		},
		"height": {
			directives: "# height: auto\n",
			want:       DrawIOHeader{Height: "auto"},
		},
		"width": {
			directives: "# width: 80\n",
			want:       DrawIOHeader{Width: "80"},
		},
		"ignore": {
			directives: "# ignore: Image\n",
			want:       DrawIOHeader{Ignore: "Image"},
		},
		"nodespacing": {
			directives: "# nodespacing: 40\n",
			want:       DrawIOHeader{NodeSpacing: 40},
		},
		"levelspacing": {
			directives: "# levelspacing: 100\n",
			want:       DrawIOHeader{LevelSpacing: 100},
		},
		"edgespacing": {
			directives: "# edgespacing: 40\n",
			want:       DrawIOHeader{EdgeSpacing: 40},
		},
		"padding": {
			directives: "# padding: 15\n",
			want:       DrawIOHeader{Padding: 15},
		},
		"link": {
			directives: "# link: URL\n",
			want:       DrawIOHeader{Link: "URL"},
		},
		"left": {
			directives: "# left: X\n",
			want:       DrawIOHeader{Left: "X"},
		},
		"top": {
			directives: "# top: Y\n",
			want:       DrawIOHeader{Top: "Y"},
		},
		"layout": {
			directives: "# layout: horizontalflow\n",
			want:       DrawIOHeader{Layout: "horizontalflow"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := mustParseDrawIO(t, tc.directives+"Name\nalpha\n")
			if !reflect.DeepEqual(got.Header, tc.want) {
				t.Errorf("Header = %+v, want %+v", got.Header, tc.want)
			}
		})
	}
}

// TestParseDrawIOCSV_DirectiveGrammar verifies the directive grammar is a
// case-sensitive exact prefix `# key: ` with the value taken verbatim,
// untrimmed, to the end of the line (requirements 2.1, 2.4).
func TestParseDrawIOCSV_DirectiveGrammar(t *testing.T) {
	tests := map[string]struct {
		directives string
		want       DrawIOHeader
	}{
		"key is case-sensitive capitalized ignored": {
			directives: "# Label: x\n",
			want:       DrawIOHeader{},
		},
		"key is case-sensitive uppercase ignored": {
			directives: "# LABEL: x\n",
			want:       DrawIOHeader{},
		},
		"no space after hash ignored": {
			directives: "#label: x\n",
			want:       DrawIOHeader{},
		},
		"two spaces after hash ignored": {
			directives: "#  label: x\n",
			want:       DrawIOHeader{},
		},
		"no space after colon ignored": {
			directives: "# label:x\n",
			want:       DrawIOHeader{},
		},
		"value keeps leading whitespace verbatim": {
			directives: "# label:  padded\n",
			want:       DrawIOHeader{Label: " padded"},
		},
		"value keeps trailing whitespace verbatim": {
			directives: "# label: padded \n",
			want:       DrawIOHeader{Label: "padded "},
		},
		"value may contain colon-space": {
			directives: "# label: a: b\n",
			want:       DrawIOHeader{Label: "a: b"},
		},
		"value taken to end of line": {
			directives: "# style: shape=image;html=1\n",
			want:       DrawIOHeader{Style: "shape=image;html=1"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := mustParseDrawIO(t, tc.directives+"Name\nalpha\n")
			if !reflect.DeepEqual(got.Header, tc.want) {
				t.Errorf("Header = %+v, want %+v", got.Header, tc.want)
			}
		})
	}
}

// TestParseDrawIOCSV_CRLF verifies \r\n line endings are stripped while a
// lone \r is kept, matching encoding/csv semantics (design parsing pipeline).
func TestParseDrawIOCSV_CRLF(t *testing.T) {
	tests := map[string]struct {
		input       string
		wantHeader  DrawIOHeader
		wantColumns []string
		wantRecords []Record
	}{
		"crlf line endings stripped from directives and data": {
			input:       "# label: %Name%\r\nName\r\nalpha\r\n",
			wantHeader:  DrawIOHeader{Label: "%Name%"},
			wantColumns: []string{"Name"},
			wantRecords: []Record{{"Name": "alpha"}},
		},
		"lone carriage return kept in directive value": {
			input:       "# label: a\rb\nName\nalpha\n",
			wantHeader:  DrawIOHeader{Label: "a\rb"},
			wantColumns: []string{"Name"},
			wantRecords: []Record{{"Name": "alpha"}},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := mustParseDrawIO(t, tc.input)
			if !reflect.DeepEqual(got.Header, tc.wantHeader) {
				t.Errorf("Header = %+v, want %+v", got.Header, tc.wantHeader)
			}
			if !reflect.DeepEqual(got.Columns, tc.wantColumns) {
				t.Errorf("Columns = %v, want %v", got.Columns, tc.wantColumns)
			}
			if !reflect.DeepEqual(got.Records, tc.wantRecords) {
				t.Errorf("Records = %v, want %v", got.Records, tc.wantRecords)
			}
		})
	}
}

// TestParseDrawIOCSV_Tolerance verifies unknown directives and non-directive
// comments are ignored, scalar duplicates are last-wins, and connect is
// append-all (requirements 2.2, 2.4, 2.5).
func TestParseDrawIOCSV_Tolerance(t *testing.T) {
	tests := map[string]struct {
		directives string
		want       DrawIOHeader
	}{
		"unknown directive key ignored": {
			directives: "# stylename: foo\n",
			want:       DrawIOHeader{},
		},
		"unrepresented drawio directive ignored": {
			directives: "# vars: {\"x\":1}\n",
			want:       DrawIOHeader{},
		},
		"comment without key-value structure ignored": {
			directives: "# just a free-form comment\n",
			want:       DrawIOHeader{},
		},
		"bare hash ignored": {
			directives: "#\n",
			want:       DrawIOHeader{},
		},
		"double hash ignored": {
			directives: "## label: x\n",
			want:       DrawIOHeader{},
		},
		"scalar duplicate last wins": {
			directives: "# label: first\n# label: second\n",
			want:       DrawIOHeader{Label: "second"},
		},
		"numeric duplicate last wins": {
			directives: "# padding: 1\n# padding: 2\n",
			want:       DrawIOHeader{Padding: 2},
		},
		"connect append-all preserves file order": {
			directives: "# connect: {\"from\":\"A\",\"to\":\"B\",\"invert\":false,\"label\":\"first\",\"style\":\"s1\"}\n" +
				"# connect: {\"from\":\"C\",\"to\":\"D\",\"invert\":true,\"label\":\"second\",\"style\":\"s2\"}\n",
			want: DrawIOHeader{Connections: []DrawIOConnection{
				{From: "A", To: "B", Invert: false, Label: "first", Style: "s1"},
				{From: "C", To: "D", Invert: true, Label: "second", Style: "s2"},
			}},
		},
		"connect ignores unknown json keys": {
			directives: "# connect: {\"from\":\"A\",\"to\":\"B\",\"invert\":false,\"label\":\"L\",\"style\":\"S\",\"extra\":\"ignored\"}\n",
			want: DrawIOHeader{Connections: []DrawIOConnection{
				{From: "A", To: "B", Invert: false, Label: "L", Style: "S"},
			}},
		},
		"connect omitted json keys zero-valued": {
			directives: "# connect: {\"from\":\"A\",\"to\":\"B\"}\n",
			want: DrawIOHeader{Connections: []DrawIOConnection{
				{From: "A", To: "B"},
			}},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := mustParseDrawIO(t, tc.directives+"Name\nalpha\n")
			if !reflect.DeepEqual(got.Header, tc.want) {
				t.Errorf("Header = %+v, want %+v", got.Header, tc.want)
			}
		})
	}
}

// TestParseDrawIOCSV_DataSection verifies BOM tolerance, blank-line skipping,
// empty-cell fill, file-order preservation, zero-value headers, and the
// zero-data-rows case (requirements 1.1, 1.3, 1.4, 1.6, 2.3).
func TestParseDrawIOCSV_DataSection(t *testing.T) {
	tests := map[string]struct {
		input       string
		wantHeader  DrawIOHeader
		wantColumns []string
		wantRecords []Record
	}{
		"zero-value header when directives absent": {
			input:       "Name,Image\nalpha,a.png\n",
			wantHeader:  DrawIOHeader{},
			wantColumns: []string{"Name", "Image"},
			wantRecords: []Record{{"Name": "alpha", "Image": "a.png"}},
		},
		"utf8 bom tolerated before directives": {
			input:       "\ufeff# label: %Name%\nName\nalpha\n",
			wantHeader:  DrawIOHeader{Label: "%Name%"},
			wantColumns: []string{"Name"},
			wantRecords: []Record{{"Name": "alpha"}},
		},
		"utf8 bom tolerated before column header": {
			input:       "\ufeffName\nalpha\n",
			wantHeader:  DrawIOHeader{},
			wantColumns: []string{"Name"},
			wantRecords: []Record{{"Name": "alpha"}},
		},
		"blank lines skipped everywhere": {
			input:       "\n# label: x\n\nName\n\nalpha\n\nbeta\n\n",
			wantHeader:  DrawIOHeader{Label: "x"},
			wantColumns: []string{"Name"},
			wantRecords: []Record{{"Name": "alpha"}, {"Name": "beta"}},
		},
		"empty cells materialize as empty string": {
			input:       "A,B,C\n1,,3\n,,\n",
			wantHeader:  DrawIOHeader{},
			wantColumns: []string{"A", "B", "C"},
			wantRecords: []Record{
				{"A": "1", "B": "", "C": "3"},
				{"A": "", "B": "", "C": ""},
			},
		},
		"columns and records preserved in file order": {
			input:       "Zebra,Alpha,Mango\nz1,a1,m1\nz2,a2,m2\nz3,a3,m3\n",
			wantHeader:  DrawIOHeader{},
			wantColumns: []string{"Zebra", "Alpha", "Mango"},
			wantRecords: []Record{
				{"Zebra": "z1", "Alpha": "a1", "Mango": "m1"},
				{"Zebra": "z2", "Alpha": "a2", "Mango": "m2"},
				{"Zebra": "z3", "Alpha": "a3", "Mango": "m3"},
			},
		},
		"zero data rows returns columns and no records": {
			input:       "# label: %Name%\nName,Image\n",
			wantHeader:  DrawIOHeader{Label: "%Name%"},
			wantColumns: []string{"Name", "Image"},
			wantRecords: []Record{},
		},
		"no trailing newline on final data row": {
			input:       "Name\nalpha",
			wantHeader:  DrawIOHeader{},
			wantColumns: []string{"Name"},
			wantRecords: []Record{{"Name": "alpha"}},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := mustParseDrawIO(t, tc.input)
			if !reflect.DeepEqual(got.Header, tc.wantHeader) {
				t.Errorf("Header = %+v, want %+v", got.Header, tc.wantHeader)
			}
			if !reflect.DeepEqual(got.Columns, tc.wantColumns) {
				t.Errorf("Columns = %v, want %v", got.Columns, tc.wantColumns)
			}
			if len(got.Records) != len(tc.wantRecords) {
				t.Fatalf("len(Records) = %d, want %d", len(got.Records), len(tc.wantRecords))
			}
			for i, want := range tc.wantRecords {
				if !reflect.DeepEqual(got.Records[i], want) {
					t.Errorf("Records[%d] = %v, want %v", i, got.Records[i], want)
				}
			}
		})
	}
}

// TestParseDrawIOCSV_DirectiveErrors verifies malformed recognized directives
// fail with ErrDrawIODirective carrying line and directive context, validated
// on every occurrence even when superseded (requirements 4.1, 4.2, 2.5).
func TestParseDrawIOCSV_DirectiveErrors(t *testing.T) {
	tests := map[string]struct {
		input        string
		wantContains []string
	}{
		"connect value not json": {
			input:        "# connect: notjson\nName\nalpha\n",
			wantContains: []string{"line 1", "connect"},
		},
		"connect value wrong json type": {
			input:        "# connect: {\"from\":1}\nName\nalpha\n",
			wantContains: []string{"line 1", "connect"},
		},
		"connect malformed even when another connect is valid": {
			input: "# connect: {\"from\":\"A\",\"to\":\"B\",\"invert\":false,\"label\":\"L\",\"style\":\"S\"}\n" +
				"# connect: {broken\nName\nalpha\n",
			wantContains: []string{"line 2", "connect"},
		},
		"nodespacing non-integer": {
			input:        "# nodespacing: wide\nName\nalpha\n",
			wantContains: []string{"line 1", "nodespacing"},
		},
		"levelspacing non-integer": {
			input:        "# levelspacing: 10.5\nName\nalpha\n",
			wantContains: []string{"line 1", "levelspacing"},
		},
		"edgespacing non-integer": {
			input:        "# edgespacing: \nName\nalpha\n",
			wantContains: []string{"line 1", "edgespacing"},
		},
		"padding non-integer": {
			input:        "# padding: 5px\nName\nalpha\n",
			wantContains: []string{"line 1", "padding"},
		},
		"padding value not trimmed before validation": {
			input:        "# padding:  5\nName\nalpha\n",
			wantContains: []string{"line 1", "padding"},
		},
		"superseded malformed numeric still errors": {
			input:        "# padding: nope\n# padding: 5\nName\nalpha\n",
			wantContains: []string{"line 1", "padding"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := ParseDrawIOCSV(strings.NewReader(tc.input))
			if !errors.Is(err, ErrDrawIODirective) {
				t.Fatalf("ParseDrawIOCSV() error = %v, want errors.Is ErrDrawIODirective", err)
			}
			for _, want := range tc.wantContains {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err.Error(), want)
				}
			}
		})
	}
}

// TestParseDrawIOCSV_NoColumnHeader verifies empty or comment/blank-only
// input fails with ErrDrawIONoColumnHeader (requirement 4.4).
func TestParseDrawIOCSV_NoColumnHeader(t *testing.T) {
	tests := map[string]string{
		"empty input":         "",
		"blank lines only":    "\n\n\n",
		"comments only":       "# label: x\n# style: y\n",
		"comments and blanks": "\n# label: x\n\n# free comment\n\n",
		"bom only":            "\ufeff",
	}

	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := ParseDrawIOCSV(strings.NewReader(input))
			if !errors.Is(err, ErrDrawIONoColumnHeader) {
				t.Fatalf("ParseDrawIOCSV() error = %v, want errors.Is ErrDrawIONoColumnHeader", err)
			}
		})
	}
}

// TestParseDrawIOCSV_TrailingDirective verifies recognized directive lines
// after the column header row fail with ErrDrawIOTrailingDirective, taking
// precedence over field-count errors (requirement 4.5, Decision 9).
func TestParseDrawIOCSV_TrailingDirective(t *testing.T) {
	tests := map[string]struct {
		input        string
		wantContains []string
	}{
		"directive after data rows": {
			input:        "Name\nalpha\n# label: x\n",
			wantContains: []string{"line 3", "label"},
		},
		"second diagram block detected": {
			input:        "# label: a\nName\nalpha\n# label: b\nName\nbeta\n",
			wantContains: []string{"line 4", "label"},
		},
		"precedence over field-count mismatch on same line": {
			input:        "A,B\n1,2\n# layout: auto\n",
			wantContains: []string{"line 3", "layout"},
		},
		"whole-section precedence over earlier field-count mismatch": {
			input:        "A,B\n1,2,3\n# layout: auto\n",
			wantContains: []string{"line 3", "layout"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := ParseDrawIOCSV(strings.NewReader(tc.input))
			if !errors.Is(err, ErrDrawIOTrailingDirective) {
				t.Fatalf("ParseDrawIOCSV() error = %v, want errors.Is ErrDrawIOTrailingDirective", err)
			}
			for _, want := range tc.wantContains {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err.Error(), want)
				}
			}
		})
	}
}

// TestParseDrawIOCSV_DuplicateColumns verifies duplicate column header names
// fail with ErrDrawIODuplicateColumn (requirement 4.6).
func TestParseDrawIOCSV_DuplicateColumns(t *testing.T) {
	_, err := ParseDrawIOCSV(strings.NewReader("A,B,A\n1,2,3\n"))
	if !errors.Is(err, ErrDrawIODuplicateColumn) {
		t.Fatalf("ParseDrawIOCSV() error = %v, want errors.Is ErrDrawIODuplicateColumn", err)
	}
	if !strings.Contains(err.Error(), `"A"`) {
		t.Errorf("error %q does not name the duplicate column", err.Error())
	}
}

// TestParseDrawIOCSV_FieldCountMismatch verifies malformed CSV passes through
// as a *csv.ParseError whose line numbers refer to the whole file, not just
// the data section (requirement 4.3).
func TestParseDrawIOCSV_FieldCountMismatch(t *testing.T) {
	_, err := ParseDrawIOCSV(strings.NewReader("# label: x\nA,B\n1,2,3\n"))
	if err == nil {
		t.Fatal("ParseDrawIOCSV() error = nil, want field-count error")
	}
	var pe *csv.ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("ParseDrawIOCSV() error = %v, want errors.As *csv.ParseError", err)
	}
	if pe.Line != 3 {
		t.Errorf("ParseError.Line = %d, want 3 (file line of the bad row)", pe.Line)
	}
	if !strings.Contains(err.Error(), "line 3") {
		t.Errorf("error %q does not contain file line context %q", err.Error(), "line 3")
	}
}

// TestParseDrawIOCSV_DataEdgeCases verifies leading-# data rows parse as data
// (requirement 1.5) and quoted fields with commas, quotes, and newlines parse
// correctly, including multi-line quoted column header names (requirement 3.4).
func TestParseDrawIOCSV_DataEdgeCases(t *testing.T) {
	tests := map[string]struct {
		input       string
		wantColumns []string
		wantRecords []Record
	}{
		"leading hash data row parses as data": {
			input:       "Name\n#alpha\n",
			wantColumns: []string{"Name"},
			wantRecords: []Record{{"Name": "#alpha"}},
		},
		"unrecognized directive-like data row parses as data": {
			input:       "Name\n# vars: x\n",
			wantColumns: []string{"Name"},
			wantRecords: []Record{{"Name": "# vars: x"}},
		},
		"quoted directive lookalike parses as data": {
			input:       "Name\n\"# label: x\"\n",
			wantColumns: []string{"Name"},
			wantRecords: []Record{{"Name": "# label: x"}},
		},
		"quoted fields with commas quotes and newlines": {
			input:       "Name,Desc\n\"a,b\",\"say \"\"hi\"\"\nsecond line\"\n",
			wantColumns: []string{"Name", "Desc"},
			wantRecords: []Record{{"Name": "a,b", "Desc": "say \"hi\"\nsecond line"}},
		},
		"multi-line quoted column header name": {
			input:       "\"Na\nme\",X\n1,2\n",
			wantColumns: []string{"Na\nme", "X"},
			wantRecords: []Record{{"Na\nme": "1", "X": "2"}},
		},
		"directive lookalike inside multi-line quoted field is data": {
			input:       "Name\n\"start\n# label: x\nend\"\n",
			wantColumns: []string{"Name"},
			wantRecords: []Record{{"Name": "start\n# label: x\nend"}},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := mustParseDrawIO(t, tc.input)
			if !reflect.DeepEqual(got.Columns, tc.wantColumns) {
				t.Errorf("Columns = %v, want %v", got.Columns, tc.wantColumns)
			}
			if !reflect.DeepEqual(got.Records, tc.wantRecords) {
				t.Errorf("Records = %v, want %v", got.Records, tc.wantRecords)
			}
		})
	}
}

// TestParseDrawIOCSV_ParityVsCSVQuoteDisagreement pins which error fires when
// the parity scan's quote model (parity counting) and csv's (quotes special
// only at field start) disagree on malformed input: a bare quote mid-field
// makes the parity scan treat following lines as quoted, so the trailing
// directive is not flagged and csv's bare-quote ParseError surfaces instead.
// Both paths fail loudly (design: whole-section scan precedence note).
func TestParseDrawIOCSV_ParityVsCSVQuoteDisagreement(t *testing.T) {
	input := "A,B\nx\"q,b\n# label: z\n"
	_, err := ParseDrawIOCSV(strings.NewReader(input))
	if err == nil {
		t.Fatal("ParseDrawIOCSV() error = nil, want bare-quote csv error")
	}
	if errors.Is(err, ErrDrawIOTrailingDirective) {
		t.Fatalf("ParseDrawIOCSV() error = %v, want csv error, not ErrDrawIOTrailingDirective", err)
	}
	var pe *csv.ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("ParseDrawIOCSV() error = %v, want errors.As *csv.ParseError", err)
	}
}

// errDrawIOReader returns its payload, then a non-EOF error mid-stream.
type errDrawIOReader struct {
	data string
	err  error
	done bool
}

func (r *errDrawIOReader) Read(p []byte) (int, error) {
	if !r.done {
		r.done = true
		return copy(p, r.data), nil
	}
	return 0, r.err
}

// TestParseDrawIOCSV_ReaderError verifies a mid-stream non-EOF read failure
// surfaces as a returned error, never a panic (requirement 4.7).
func TestParseDrawIOCSV_ReaderError(t *testing.T) {
	wantErr := errors.New("disk exploded")
	_, err := ParseDrawIOCSV(&errDrawIOReader{data: "Name\nal", err: wantErr})
	if !errors.Is(err, wantErr) {
		t.Fatalf("ParseDrawIOCSV() error = %v, want errors.Is the reader error", err)
	}
}

// TestParseDrawIOFile verifies the file convenience wrapper: happy path and
// wrapped os.Open errors (requirements 1.2, 4.7).
func TestParseDrawIOFile(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "diagram.csv")
		content := "# label: %Name%\nName,Image\nalpha,a.png\n"
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("writing fixture: %v", err)
		}
		got, err := ParseDrawIOFile(path)
		if err != nil {
			t.Fatalf("ParseDrawIOFile() error = %v, want nil", err)
		}
		want := &ParsedDrawIO{
			Header:  DrawIOHeader{Label: "%Name%"},
			Columns: []string{"Name", "Image"},
			Records: []Record{{"Name": "alpha", "Image": "a.png"}},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("ParseDrawIOFile() = %+v, want %+v", got, want)
		}
	})

	t.Run("missing file wraps os.Open error", func(t *testing.T) {
		_, err := ParseDrawIOFile(filepath.Join(t.TempDir(), "nope.csv"))
		if !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("ParseDrawIOFile() error = %v, want errors.Is fs.ErrNotExist", err)
		}
	})
}
