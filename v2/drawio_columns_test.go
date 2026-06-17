package output

import (
	"slices"
	"testing"
)

// TestWithDrawIOColumns_NewDrawIOContent verifies that the WithDrawIOColumns
// option sets an explicit column order on content built via NewDrawIOContent.
func TestWithDrawIOColumns_NewDrawIOContent(t *testing.T) {
	records := []Record{
		{"Beta": "1", "Alpha": "2"},
	}

	content := NewDrawIOContent("ordered", records, DrawIOHeader{}, WithDrawIOColumns("Beta", "Alpha"))

	got := content.GetColumns()
	want := []string{"Beta", "Alpha"}
	if !slices.Equal(got, want) {
		t.Errorf("GetColumns() = %v, want %v", got, want)
	}
}

// TestWithDrawIOColumns_BuilderDrawIO verifies that Builder.DrawIO forwards
// DrawIOOption values to the underlying NewDrawIOContent call.
func TestWithDrawIOColumns_BuilderDrawIO(t *testing.T) {
	doc := New().
		DrawIO("forwarded", []Record{{"Beta": "1", "Alpha": "2"}}, DrawIOHeader{}, WithDrawIOColumns("Beta", "Alpha")).
		Build()

	contents := doc.GetContents()
	if len(contents) != 1 {
		t.Fatalf("len(GetContents()) = %d, want 1", len(contents))
	}

	drawioContent, ok := contents[0].(*DrawIOContent)
	if !ok {
		t.Fatalf("content type = %T, want *DrawIOContent", contents[0])
	}

	got := drawioContent.GetColumns()
	want := []string{"Beta", "Alpha"}
	if !slices.Equal(got, want) {
		t.Errorf("GetColumns() = %v, want %v", got, want)
	}
}

// TestDrawIOContent_GetColumnsDefensiveCopy verifies that mutating the slice
// returned by GetColumns does not affect the content's internal state.
func TestDrawIOContent_GetColumnsDefensiveCopy(t *testing.T) {
	content := NewDrawIOContent("copy", nil, DrawIOHeader{}, WithDrawIOColumns("Beta", "Alpha"))

	got := content.GetColumns()
	if len(got) != 2 {
		t.Fatalf("GetColumns() returned %d columns, want 2", len(got))
	}

	got[0] = "mutated"

	want := []string{"Beta", "Alpha"}
	if again := content.GetColumns(); !slices.Equal(again, want) {
		t.Errorf("after mutating returned slice, GetColumns() = %v, want %v", again, want)
	}
}

// TestDrawIOContent_CloneCopiesColumns verifies that Clone deep-copies the
// columns slice so the clone and the original do not share backing storage.
func TestDrawIOContent_CloneCopiesColumns(t *testing.T) {
	content := NewDrawIOContent("clone", nil, DrawIOHeader{}, WithDrawIOColumns("Beta", "Alpha"))

	clone, ok := content.Clone().(*DrawIOContent)
	if !ok {
		t.Fatalf("Clone() type = %T, want *DrawIOContent", content.Clone())
	}

	want := []string{"Beta", "Alpha"}
	if !slices.Equal(clone.columns, want) {
		t.Fatalf("clone columns = %v, want %v", clone.columns, want)
	}

	clone.columns[0] = "mutated"

	if !slices.Equal(content.columns, want) {
		t.Errorf("mutating clone columns changed original: got %v, want %v", content.columns, want)
	}
}

// TestNewDrawIOContentFromTable_CapturesSchemaOrder verifies that the
// from-table constructor records the table schema's field order (Decision 13)
// instead of leaving columns empty for the renderer to alphabetize.
func TestNewDrawIOContentFromTable_CapturesSchemaOrder(t *testing.T) {
	records := []Record{
		{"Name": "Server1", "Type": "Web"},
	}
	table, err := NewTableContent("table", records, WithKeys("Type", "Name"))
	if err != nil {
		t.Fatalf("NewTableContent failed: %v", err)
	}

	content := NewDrawIOContentFromTable(table, DrawIOHeader{})

	got := content.GetColumns()
	want := []string{"Type", "Name"}
	if !slices.Equal(got, want) {
		t.Errorf("GetColumns() = %v, want %v", got, want)
	}
}

// TestNewDrawIOContentFromTable_NilTableNilColumns verifies the nil-table
// path leaves the columns slice nil (Decision 13).
func TestNewDrawIOContentFromTable_NilTableNilColumns(t *testing.T) {
	content := NewDrawIOContentFromTable(nil, DrawIOHeader{})

	if got := content.GetColumns(); got != nil {
		t.Errorf("GetColumns() = %v, want nil", got)
	}
}
