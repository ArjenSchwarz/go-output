package output

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Sentinel errors returned by ParseDrawIOCSV/ParseDrawIOFile. All are wrapped
// with additional line/directive context via fmt.Errorf("%w: ...") so callers
// can discriminate with errors.Is while messages stay informative.
var (
	// ErrDrawIONoColumnHeader indicates the input contains no column header
	// row (empty input, or comments and blank lines only). Requirement 4.4.
	ErrDrawIONoColumnHeader = errors.New("drawio csv: no column header row")
	// ErrDrawIOTrailingDirective indicates a recognized directive line was
	// found after the column header row, e.g. the start of a second diagram
	// block. Requirement 4.5.
	ErrDrawIOTrailingDirective = errors.New("drawio csv: directive after column header row")
	// ErrDrawIODuplicateColumn indicates the column header row contains a
	// duplicate column name. Requirement 4.6.
	ErrDrawIODuplicateColumn = errors.New("drawio csv: duplicate column name")
	// ErrDrawIODirective indicates a recognized directive has an invalid
	// value: unparseable connect JSON or a non-integer numeric value.
	// Requirements 4.1 and 4.2.
	ErrDrawIODirective = errors.New("drawio csv: invalid directive")
)

// ParsedDrawIO holds the result of parsing a draw.io CSV file.
type ParsedDrawIO struct {
	Header  DrawIOHeader // zero-valued fields for absent directives
	Columns []string     // column header row, file order
	Records []Record     // file order; every column present, "" for empty cells
}

// drawioUTF8BOM is the UTF-8 byte order mark tolerated at the start of input.
const drawioUTF8BOM = "\xef\xbb\xbf"

// Directive keys referenced from more than one place in the parser.
const (
	drawioKeyLabel        = "label"
	drawioKeyNodeSpacing  = "nodespacing"
	drawioKeyLevelSpacing = "levelspacing"
	drawioKeyEdgeSpacing  = "edgespacing"
	drawioKeyPadding      = "padding"
)

// drawioDirectiveKeys is the set of directive keys the v2 renderer emits,
// matched case-sensitively (requirement 2.1).
var drawioDirectiveKeys = map[string]bool{
	drawioKeyLabel:        true,
	"style":               true,
	"identity":            true,
	"parent":              true,
	"parentstyle":         true,
	"namespace":           true,
	"connect":             true,
	"height":              true,
	"width":               true,
	"ignore":              true,
	drawioKeyNodeSpacing:  true,
	drawioKeyLevelSpacing: true,
	drawioKeyEdgeSpacing:  true,
	drawioKeyPadding:      true,
	"link":                true,
	"left":                true,
	"top":                 true,
	"layout":              true,
}

// ParseDrawIOCSV parses draw.io CSV (as written by the drawio renderer) from
// r into its header directives, column order, and data records. Absent
// directives leave the corresponding DrawIOHeader field zero-valued; the
// DefaultDrawIOHeader values are never substituted.
func ParseDrawIOCSV(r io.Reader) (*ParsedDrawIO, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("drawio csv: reading input: %w", err)
	}

	// Strip an optional UTF-8 byte order mark (requirement 1.6).
	text := strings.TrimPrefix(string(data), drawioUTF8BOM)
	lines := splitDrawIOLines(text)

	// Directive pre-pass: consume directive/comment/blank lines until the
	// first non-comment, non-blank line, which is the column header row.
	header := DrawIOHeader{}
	headerIdx := -1
	for i, line := range lines {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			if err := applyDrawIODirective(&header, line, i+1); err != nil {
				return nil, err
			}
			continue
		}
		headerIdx = i
		break
	}
	if headerIdx == -1 {
		return nil, fmt.Errorf("%w: input contains only directives, comments, or blank lines", ErrDrawIONoColumnHeader)
	}

	// Quote-parity scan of the data section: detect recognized directive
	// lines after the column header row (a second diagram block) before csv
	// parsing, so this error takes precedence over field-count errors. The
	// state is seeded at the start of the header line so continuation lines
	// of a multi-line quoted column header are never tested as boundaries.
	inQuote := false
	for i := headerIdx; i < len(lines); i++ {
		line := lines[i]
		if i > headerIdx && !inQuote {
			if key, _, ok := matchDrawIODirective(line); ok {
				return nil, fmt.Errorf("%w: line %d: %q (multiple diagram blocks are not supported)", ErrDrawIOTrailingDirective, i+1, key)
			}
		}
		if strings.Count(line, `"`)%2 == 1 {
			inQuote = !inQuote
		}
	}

	// CSV parse of column header row + data section. No Comment mode (data
	// rows starting with '#' survive), FieldsPerRecord enforcement via the
	// first record, TrimLeadingSpace stays false (values verbatim).
	reader := csv.NewReader(strings.NewReader(strings.Join(lines[headerIdx:], "\n")))
	columns, err := reader.Read()
	if err != nil {
		return nil, adjustDrawIOCSVError(err, headerIdx)
	}

	seen := make(map[string]bool, len(columns))
	for _, col := range columns {
		if seen[col] {
			return nil, fmt.Errorf("%w: %q", ErrDrawIODuplicateColumn, col)
		}
		seen[col] = true
	}

	records := make([]Record, 0)
	for {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, adjustDrawIOCSVError(err, headerIdx)
		}
		record := make(Record, len(columns))
		for j, col := range columns {
			record[col] = row[j]
		}
		records = append(records, record)
	}

	return &ParsedDrawIO{Header: header, Columns: columns, Records: records}, nil
}

// ParseDrawIOFile opens path and parses it with ParseDrawIOCSV.
func ParseDrawIOFile(path string) (*ParsedDrawIO, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("drawio csv: %w", err)
	}
	defer func() { _ = f.Close() }()
	return ParseDrawIOCSV(f)
}

// adjustDrawIOCSVError shifts a *csv.ParseError's line numbers from
// data-section coordinates (the csv.Reader only sees the input from the
// column header row onward) to whole-file coordinates so the reported line
// context matches the file (requirement 4.3). Non-ParseError errors pass
// through unchanged.
func adjustDrawIOCSVError(err error, headerIdx int) error {
	var pe *csv.ParseError
	if errors.As(err, &pe) {
		pe.Line += headerIdx
		pe.StartLine += headerIdx
	}
	return err
}

// splitDrawIOLines splits input into physical lines on '\n', stripping a
// preceding '\r' (CRLF) while keeping lone '\r' characters — matching
// encoding/csv semantics so directives and data normalize identically on
// CRLF files.
func splitDrawIOLines(s string) []string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSuffix(line, "\r")
	}
	return lines
}

// matchDrawIODirective reports whether line matches the directive grammar:
// the exact prefix "# " + recognized key + ": ", with the value taken
// verbatim (untrimmed) to the end of the line. Keys are case-sensitive.
func matchDrawIODirective(line string) (key, value string, ok bool) {
	rest, found := strings.CutPrefix(line, "# ")
	if !found {
		return "", "", false
	}
	key, value, found = strings.Cut(rest, ": ")
	if !found || !drawioDirectiveKeys[key] {
		return "", "", false
	}
	return key, value, true
}

// applyDrawIODirective applies a single comment line from the header block to
// h. Lines that do not match a recognized directive are ignored (requirement
// 2.4). Scalar directives are last-wins; connect is append-all (requirements
// 2.2, 2.5). Numeric and connect values are validated on every occurrence,
// even when later superseded (requirements 4.1, 4.2).
func applyDrawIODirective(h *DrawIOHeader, line string, lineNum int) error {
	key, value, ok := matchDrawIODirective(line)
	if !ok {
		return nil
	}
	switch key {
	case drawioKeyLabel:
		h.Label = value
	case "style":
		h.Style = value
	case "identity":
		h.Identity = value
	case "parent":
		h.Parent = value
	case "parentstyle":
		h.ParentStyle = value
	case "namespace":
		h.Namespace = value
	case "connect":
		var conn drawioConnectionJSON
		if err := json.Unmarshal([]byte(value), &conn); err != nil {
			return fmt.Errorf("%w: line %d: connect: %v", ErrDrawIODirective, lineNum, err)
		}
		h.Connections = append(h.Connections, DrawIOConnection(conn))
	case "height":
		h.Height = value
	case "width":
		h.Width = value
	case "ignore":
		h.Ignore = value
	case drawioKeyNodeSpacing, drawioKeyLevelSpacing, drawioKeyEdgeSpacing, drawioKeyPadding:
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("%w: line %d: %s: %q is not an integer", ErrDrawIODirective, lineNum, key, value)
		}
		switch key {
		case drawioKeyNodeSpacing:
			h.NodeSpacing = n
		case drawioKeyLevelSpacing:
			h.LevelSpacing = n
		case drawioKeyEdgeSpacing:
			h.EdgeSpacing = n
		case drawioKeyPadding:
			h.Padding = n
		}
	case "link":
		h.Link = value
	case "left":
		h.Left = value
	case "top":
		h.Top = value
	case "layout":
		h.Layout = value
	}
	return nil
}

// drawioConnectionJSON pins the draw.io CSV wire format for `# connect:`
// directive values (Decision 14): keys from/to/invert/label/style in that
// order, all always present. The public DrawIOConnection stays untagged so
// JSON/YAML document renderers keep their exported-field key casing; this
// private mirror confines the wire format to the drawio CSV writer and
// parser.
type drawioConnectionJSON struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Invert bool   `json:"invert"` // no omitempty: "invert":false is load-bearing for byte identity (requirement 3.5)
	Label  string `json:"label"`
	Style  string `json:"style"`
}
