package output

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"unicode"

	"pgregory.net/rapid"
)

// This file holds the property-based round-trip tests for the draw.io CSV
// writer/parser pair (requirements 3.1, 3.2, 4.7) plus an example-based
// golden round-trip as an anchor against generator blind spots.
//
// Generator constraints encode documented carve-outs, not bugs:
//   - no '\n' or '\r' in header (directive) string fields: the draw.io CSV
//     header format cannot represent line breaks in directive values
//     (requirements Out of Scope)
//   - unique column names: duplicates correctly fail with
//     ErrDrawIODuplicateColumn (requirement 4.6)
//   - record keys are a subset of the columns: keys outside the column list
//     are intentionally not rendered (WithDrawIOColumns silent-drop contract)
//   - column names must stay distinct after CRLF normalization: parsing
//     normalizes "\r\n" to "\n" inside quoted fields (Decision 7), so two
//     columns differing only by CR would collide on re-parse

// drawioRoundTripTB is the subset of testing.T/rapid.T the round-trip
// helpers need.
type drawioRoundTripTB interface {
	Helper()
	Fatalf(format string, args ...any)
}

// renderDrawIORoundTrip renders header/columns/records through the drawio
// renderer using the round-trip contract from the design: a document built
// with NewDrawIOContent and WithDrawIOColumns(parsed.Columns...).
func renderDrawIORoundTrip(t drawioRoundTripTB, header DrawIOHeader, columns []string, records []Record) []byte {
	t.Helper()
	doc := New().
		AddContent(NewDrawIOContent("round-trip", records, header, WithDrawIOColumns(columns...))).
		Build()
	out, err := (&drawioRenderer{}).Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Render() error = %v, want nil", err)
	}
	return out
}

// parseDrawIORoundTrip parses rendered output, failing the test on error:
// renderer-produced output must always parse back (requirement 3.1).
func parseDrawIORoundTrip(t drawioRoundTripTB, data []byte, stage string) *ParsedDrawIO {
	t.Helper()
	parsed, err := ParseDrawIOCSV(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ParseDrawIOCSV() %s parse error = %v, want nil\ninput:\n%s", stage, err, data)
	}
	return parsed
}

// drawioStringGen generates printable strings biased toward the characters
// the round-trip requirements call out: quotes, backslashes, commas, '#',
// directive punctuation, and HTML-escape candidates. extra adds runes (e.g.
// '\n' or '\r' for record values) beyond the printable base set.
func drawioStringGen(extra ...rune) *rapid.Generator[string] {
	runes := append([]rune{'"', '\\', ',', '#', ':', ' ', '%', '&', '<', '>'}, extra...)
	return rapid.StringOf(rapid.RuneFrom(runes, unicode.L, unicode.N, unicode.P, unicode.S))
}

// drawioHeaderGen generates an arbitrary DrawIOHeader. String fields stay
// free of line breaks (documented carve-out); numeric fields include zero
// and negative values, which the writer suppresses in both renders.
func drawioHeaderGen() *rapid.Generator[DrawIOHeader] {
	return rapid.Custom(func(t *rapid.T) DrawIOHeader {
		str := drawioStringGen()
		num := rapid.IntRange(-2, 200)
		h := DrawIOHeader{
			Label:        str.Draw(t, "label"),
			Style:        str.Draw(t, "style"),
			Ignore:       str.Draw(t, "ignore"),
			Link:         str.Draw(t, "link"),
			Layout:       rapid.OneOf(rapid.SampledFrom([]string{"", DrawIOLayoutNone, DrawIOLayoutAuto, DrawIOLayoutHorizontalFlow}), str).Draw(t, "layout"),
			NodeSpacing:  num.Draw(t, "nodespacing"),
			LevelSpacing: num.Draw(t, "levelspacing"),
			EdgeSpacing:  num.Draw(t, "edgespacing"),
			Padding:      num.Draw(t, "padding"),
			Parent:       str.Draw(t, "parent"),
			ParentStyle:  str.Draw(t, "parentstyle"),
			Height:       str.Draw(t, "height"),
			Width:        str.Draw(t, "width"),
			Left:         str.Draw(t, "left"),
			Top:          str.Draw(t, "top"),
			Identity:     str.Draw(t, "identity"),
			Namespace:    str.Draw(t, "namespace"),
		}
		for i := range rapid.IntRange(0, 3).Draw(t, "connectionCount") {
			h.Connections = append(h.Connections, DrawIOConnection{
				From:   str.Draw(t, fmt.Sprintf("conn%d.from", i)),
				To:     str.Draw(t, fmt.Sprintf("conn%d.to", i)),
				Invert: rapid.Bool().Draw(t, fmt.Sprintf("conn%d.invert", i)),
				Label:  str.Draw(t, fmt.Sprintf("conn%d.label", i)),
				Style:  str.Draw(t, fmt.Sprintf("conn%d.style", i)),
			})
		}
		return h
	})
}

// drawioColumnsGen generates 1..4 column names, unique after CRLF
// normalization. allowCR includes '\r' for the idempotency property; the
// byte-identity property (requirement 3.2) is restricted to CR-free values.
func drawioColumnsGen(allowCR bool) *rapid.Generator[[]string] {
	extra := []rune{'\n'}
	if allowCR {
		extra = append(extra, '\r')
	}
	return rapid.SliceOfNDistinct(drawioStringGen(extra...), 1, 4, func(s string) string {
		return strings.ReplaceAll(s, "\r\n", "\n")
	})
}

// drawioRecordsGen generates 0..4 records whose keys are a subset of
// columns and whose values are strings (the only value type the parser can
// produce, so the only one the round-trip contract covers).
func drawioRecordsGen(t *rapid.T, columns []string, allowCR bool) []Record {
	extra := []rune{'\n'}
	if allowCR {
		extra = append(extra, '\r')
	}
	valueGen := drawioStringGen(extra...)
	count := rapid.IntRange(0, 4).Draw(t, "recordCount")
	records := make([]Record, 0, count)
	for i := range count {
		record := make(Record)
		for j, col := range columns {
			if rapid.Bool().Draw(t, fmt.Sprintf("record%d.has%d", i, j)) {
				record[col] = valueGen.Draw(t, fmt.Sprintf("record%d.value%d", i, j))
			}
		}
		records = append(records, record)
	}
	return records
}

// TestPropertyDrawIORoundTripIdempotency verifies requirement 3.1: for any
// renderable header/columns/records, render → parse → render → parse →
// render produces byte-equal output from the first re-render onward.
func TestPropertyDrawIORoundTripIdempotency(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		header := drawioHeaderGen().Draw(rt, "header")
		columns := drawioColumnsGen(true).Draw(rt, "columns")
		records := drawioRecordsGen(rt, columns, true)

		first := renderDrawIORoundTrip(rt, header, columns, records)
		parsed1 := parseDrawIORoundTrip(rt, first, "first")
		second := renderDrawIORoundTrip(rt, parsed1.Header, parsed1.Columns, parsed1.Records)
		parsed2 := parseDrawIORoundTrip(rt, second, "second")
		third := renderDrawIORoundTrip(rt, parsed2.Header, parsed2.Columns, parsed2.Records)

		if !bytes.Equal(second, third) {
			rt.Fatalf("round-trip not idempotent\nsecond render:\n%q\nthird render:\n%q", second, third)
		}
	})
}

// TestPropertyDrawIORoundTripByteIdentity verifies requirement 3.2: for
// CR-free values, the first re-render is already byte-identical to the
// renderer's original output.
func TestPropertyDrawIORoundTripByteIdentity(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		header := drawioHeaderGen().Draw(rt, "header")
		columns := drawioColumnsGen(false).Draw(rt, "columns")
		records := drawioRecordsGen(rt, columns, false)

		first := renderDrawIORoundTrip(rt, header, columns, records)
		parsed := parseDrawIORoundTrip(rt, first, "first")
		second := renderDrawIORoundTrip(rt, parsed.Header, parsed.Columns, parsed.Records)

		if !bytes.Equal(first, second) {
			rt.Fatalf("re-render not byte-identical\nfirst render:\n%q\nsecond render:\n%q", first, second)
		}
	})
}

// TestPropertyParseDrawIOCSVNoPanic verifies requirement 4.7 for arbitrary
// byte input: the parser returns a result or an error, and never panics.
func TestPropertyParseDrawIOCSVNoPanic(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		data := rapid.SliceOf(rapid.Byte()).Draw(rt, "data")
		parsed, err := ParseDrawIOCSV(bytes.NewReader(data))
		if (parsed == nil) == (err == nil) {
			rt.Fatalf("ParseDrawIOCSV() = (%v, %v), want exactly one of result and error non-nil", parsed, err)
		}
	})
}

// TestPropertyParseDrawIODirectivesNoPanic verifies requirement 4.7 on the
// directive-handling paths: a generator biased toward `# connect:` and
// `# padding:` lines makes sure the JSON unmarshal and Atoi branches are
// actually exercised with malformed values, asserting error-or-success.
func TestPropertyParseDrawIODirectivesNoPanic(t *testing.T) {
	connectValueGen := rapid.OneOf(
		rapid.SampledFrom([]string{
			`{"from":"a","to":"b","invert":false,"label":"l","style":"s"}`,
			`{"from": 5}`,
			`{"invert":"yes"}`,
			`{"from":"a"`,
			`"unterminated`,
			`{`, `}`, `[]`, `[1,2]`, `null`, `true`, `NaN`, ``,
		}),
		rapid.String(),
	)
	paddingValueGen := rapid.OneOf(
		rapid.SampledFrom([]string{
			"42", "-7", "0", "", " 42", "42 ", "4.5", "1e3", "0x10", "x",
			"999999999999999999999999999999",
		}),
		rapid.String(),
	)

	rapid.Check(t, func(rt *rapid.T) {
		var sb strings.Builder
		lineCount := rapid.IntRange(1, 8).Draw(rt, "lineCount")
		for i := range lineCount {
			switch rapid.IntRange(0, 2).Draw(rt, fmt.Sprintf("kind%d", i)) {
			case 0:
				sb.WriteString("# connect: " + connectValueGen.Draw(rt, fmt.Sprintf("connect%d", i)) + "\n")
			case 1:
				sb.WriteString("# padding: " + paddingValueGen.Draw(rt, fmt.Sprintf("padding%d", i)) + "\n")
			default:
				sb.WriteString(rapid.String().Draw(rt, fmt.Sprintf("line%d", i)) + "\n")
			}
		}
		if rapid.Bool().Draw(rt, "appendData") {
			sb.WriteString("a,b\n1,2\n")
		}

		parsed, err := ParseDrawIOCSV(strings.NewReader(sb.String()))
		if (parsed == nil) == (err == nil) {
			rt.Fatalf("ParseDrawIOCSV() = (%v, %v), want exactly one of result and error non-nil\ninput:\n%q", parsed, err, sb.String())
		}
	})
}

// drawioGoldenFile is a realistic awstools-style draw.io CSV file
// (connections, identity, parent, layout none with left/top), with directive
// lines in the renderer's emission order so byte identity (requirement 3.2)
// can be asserted against the first re-render.
const drawioGoldenFile = `# label: %Name%
# style: %Image%
# identity: ID
# parent: Parent
# parentstyle: swimlane;whiteSpace=wrap;html=1;childLayout=stackLayout;horizontal=1;horizontalStack=0;resizeParent=1;resizeLast=0;collapsible=1;
# namespace: awstools-
# connect: {"from":"DrawSource","to":"ID","invert":false,"label":"Connection","style":"curved=1;endArrow=blockThin;endFill=1;fontSize=11;"}
# connect: {"from":"DrawTarget","to":"ID","invert":true,"label":"","style":"dashed=1;endArrow=blockThin;endFill=1;fontSize=11;"}
# height: 78
# width: 78
# ignore: ID,Image,DrawSource,DrawTarget,Connection,X,Y
# nodespacing: 40
# levelspacing: 100
# edgespacing: 40
# link: Link
# left: X
# top: Y
# layout: none
ID,Name,Image,Parent,DrawSource,DrawTarget,Connection,Link,X,Y
tgw-123,Transit Gateway,aws.network.transit_gateway,,,,,https://example.com/tgw-123,40,40
vpc-1,"Production VPC, primary",aws.network.vpc,tgw-123,tgw-123,,attached,https://example.com/vpc-1,120,40
vpc-2,Development VPC,aws.network.vpc,tgw-123,tgw-123,vpc-1,peered,,200,120
`

// TestDrawIORoundTripGoldenFile is the example-based anchor for the
// round-trip properties: parse the golden file, verify the interesting
// fields, then assert first-re-render byte identity (3.2) and second/third
// render idempotency (3.1).
func TestDrawIORoundTripGoldenFile(t *testing.T) {
	parsed, err := ParseDrawIOCSV(strings.NewReader(drawioGoldenFile))
	if err != nil {
		t.Fatalf("ParseDrawIOCSV() error = %v, want nil", err)
	}

	if parsed.Header.Identity != "ID" {
		t.Errorf("Header.Identity = %q, want %q", parsed.Header.Identity, "ID")
	}
	if parsed.Header.Parent != "Parent" {
		t.Errorf("Header.Parent = %q, want %q", parsed.Header.Parent, "Parent")
	}
	if parsed.Header.Layout != DrawIOLayoutNone {
		t.Errorf("Header.Layout = %q, want %q", parsed.Header.Layout, DrawIOLayoutNone)
	}
	if parsed.Header.Left != "X" || parsed.Header.Top != "Y" {
		t.Errorf("Header.Left/Top = %q/%q, want X/Y", parsed.Header.Left, parsed.Header.Top)
	}
	if len(parsed.Header.Connections) != 2 {
		t.Fatalf("len(Header.Connections) = %d, want 2", len(parsed.Header.Connections))
	}
	if got := parsed.Header.Connections[0]; got.From != "DrawSource" || got.Invert {
		t.Errorf("Connections[0] = %+v, want From=DrawSource Invert=false", got)
	}
	if got := parsed.Header.Connections[1]; got.From != "DrawTarget" || !got.Invert {
		t.Errorf("Connections[1] = %+v, want From=DrawTarget Invert=true", got)
	}

	wantColumns := []string{"ID", "Name", "Image", "Parent", "DrawSource", "DrawTarget", "Connection", "Link", "X", "Y"}
	if fmt.Sprint(parsed.Columns) != fmt.Sprint(wantColumns) {
		t.Errorf("Columns = %v, want %v", parsed.Columns, wantColumns)
	}
	if len(parsed.Records) != 3 {
		t.Fatalf("len(Records) = %d, want 3", len(parsed.Records))
	}
	if got := parsed.Records[1]["Name"]; got != "Production VPC, primary" {
		t.Errorf("Records[1][Name] = %q, want %q", got, "Production VPC, primary")
	}
	if got := parsed.Records[0]["DrawSource"]; got != "" {
		t.Errorf("Records[0][DrawSource] = %q, want empty string", got)
	}

	// Byte identity (3.2): the golden file is CR-free renderer-shaped
	// output, so the first re-render must reproduce it exactly.
	first := renderDrawIORoundTrip(t, parsed.Header, parsed.Columns, parsed.Records)
	if string(first) != drawioGoldenFile {
		t.Errorf("first re-render differs from golden file\ngot:\n%s\nwant:\n%s", first, drawioGoldenFile)
	}

	// Idempotency (3.1): subsequent parse/render cycles are stable.
	reparsed := parseDrawIORoundTrip(t, first, "golden re-render")
	second := renderDrawIORoundTrip(t, reparsed.Header, reparsed.Columns, reparsed.Records)
	if !bytes.Equal(first, second) {
		t.Errorf("second render differs from first\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}
