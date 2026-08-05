package output

import "testing"

// expectNoPanicWithNilOption runs fn and fails the test if it panics.
// Regression tests for T-1444: nil functional options must be skipped,
// not invoked (which panics with a nil pointer dereference).
func expectNoPanicWithNilOption(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panicked with nil option: %v", r)
		}
	}()
	fn()
}

// TestNilFunctionalOptionsDoNotPanic verifies that every public constructor
// and option applicator ignores nil options instead of panicking (T-1444).
func TestNilFunctionalOptionsDoNotPanic(t *testing.T) {
	tests := map[string]func(){
		"NewTableContent": func() {
			data := []map[string]any{{"name": "test"}}
			if _, err := NewTableContent("x", data, nil); err != nil {
				panic(err)
			}
		},
		"ApplyTableOptions": func() {
			ApplyTableOptions(nil, WithKeys("a"))
		},
		"NewTextContent": func() {
			NewTextContent("x", nil)
		},
		"ApplyTextOptions": func() {
			ApplyTextOptions(nil)
		},
		"NewRawContent": func() {
			if _, err := NewRawContent(FormatHTML, []byte("<p>hi</p>"), nil); err != nil {
				panic(err)
			}
		},
		"ApplyRawOptions": func() {
			ApplyRawOptions(nil)
		},
		"NewSectionContent": func() {
			NewSectionContent("title", nil)
		},
		"ApplySectionOptions": func() {
			ApplySectionOptions(nil)
		},
		"NewOutput": func() {
			NewOutput(nil)
		},
		"NewCollapsibleValue": func() {
			NewCollapsibleValue("summary", "details", nil)
		},
		"NewCollapsibleSection": func() {
			NewCollapsibleSection("title", []Content{}, nil)
		},
		"NewProgress": func() {
			p := NewProgress(nil)
			_ = p.Close()
		},
		"NewAutoProgress": func() {
			p := NewAutoProgress(nil)
			_ = p.Close()
		},
		"NewPrettyProgress": func() {
			p := NewPrettyProgress(nil)
			_ = p.Close()
		},
		"NewProgressForFormat": func() {
			p := NewProgressForFormat(Table(), nil)
			_ = p.Close()
		},
		"NewProgressForFormatName": func() {
			p := NewProgressForFormatName(FormatTable, nil)
			_ = p.Close()
		},
		"NewProgressForFormats": func() {
			p := NewProgressForFormats([]Format{Table()}, nil)
			_ = p.Close()
		},
		"NewFileWriterWithOptions": func() {
			dir := t.TempDir()
			if _, err := NewFileWriterWithOptions(dir, "out.{ext}", nil); err != nil {
				panic(err)
			}
		},
		"NewS3WriterWithOptions": func() {
			NewS3WriterWithOptions(nil, "bucket", "key-{format}.{ext}", nil)
		},
		"NewDrawIOContent": func() {
			NewDrawIOContent("title", []Record{}, DrawIOHeader{}, nil)
		},
		"NewDrawIOContentFromTable": func() {
			NewDrawIOContentFromTable(nil, DrawIOHeader{}, nil)
		},
	}

	for name, fn := range tests {
		t.Run(name, func(t *testing.T) {
			expectNoPanicWithNilOption(t, fn)
		})
	}
}

// TestNilOptionsInterleavedWithRealOptions verifies that nil options are
// skipped while surrounding real options are still applied (T-1444).
func TestNilOptionsInterleavedWithRealOptions(t *testing.T) {
	t.Run("table keys preserved around nil", func(t *testing.T) {
		expectNoPanicWithNilOption(t, func() {
			data := []map[string]any{{"b": 2, "a": 1}}
			table, err := NewTableContent("x", data, nil, WithKeys("b", "a"), nil)
			if err != nil {
				t.Fatalf("NewTableContent() error = %v, want nil", err)
			}
			got := table.Schema().GetKeyOrder()
			want := []string{"b", "a"}
			if len(got) != len(want) {
				t.Fatalf("key order length = %d, want %d", len(got), len(want))
			}
			for i, key := range want {
				if got[i] != key {
					t.Errorf("key order[%d] = %q, want %q", i, got[i], key)
				}
			}
		})
	})

	t.Run("text style applied around nil", func(t *testing.T) {
		expectNoPanicWithNilOption(t, func() {
			text := NewTextContent("x", nil, WithBold(true), nil)
			if !text.Style().Bold {
				t.Error("Style().Bold = false, want true")
			}
		})
	})
}
