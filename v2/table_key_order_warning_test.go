package output

import (
	"errors"
	"strings"
	"testing"
)

// Regression tests for T-1692: optionless builder.Table() silently
// alphabetizes columns, contradicting the "never alphabetized" guarantee.
//
// Go map iteration order is unrecoverable, so schema auto-detection falls
// back to alphabetical key order. That fallback must not be silent: when
// auto-detection has to guess key order for map data (no WithKeys/WithSchema),
// Builder.Table must record a non-fatal warning retrievable via Errors().
// The table is still added to the document — the warning is a signal, not a
// failure.

// keyOrderWarnings filters builder errors down to the key-order guess warning
// recorded by Builder.Table.
func keyOrderWarnings(errs []error) []error {
	var found []error
	for _, err := range errs {
		if errors.Is(err, ErrTableKeyOrderGuessed) {
			found = append(found, err)
		}
	}
	return found
}

func TestBuilderTable_KeyOrderGuessWarning(t *testing.T) {
	multiColumn := []map[string]any{
		{"zebra": 1, "apple": 2, "mango": 3},
	}

	tests := map[string]struct {
		data        any
		opts        []TableOption
		wantWarning bool
	}{
		"optionless map data records warning": {
			data:        multiColumn,
			wantWarning: true,
		},
		"explicit WithAutoSchema still guesses and records warning": {
			data:        multiColumn,
			opts:        []TableOption{WithAutoSchema()},
			wantWarning: true,
		},
		"Record slice without options records warning": {
			data:        []Record{{"zebra": 1, "apple": 2}},
			wantWarning: true,
		},
		"single map without options records warning": {
			data:        map[string]any{"zebra": 1, "apple": 2},
			wantWarning: true,
		},
		"WithKeys does not warn": {
			data:        multiColumn,
			opts:        []TableOption{WithKeys("zebra", "apple", "mango")},
			wantWarning: false,
		},
		"WithSchema does not warn": {
			data: multiColumn,
			opts: []TableOption{WithSchema(
				Field{Name: "zebra"}, Field{Name: "apple"}, Field{Name: "mango"},
			)},
			wantWarning: false,
		},
		"WithAutoSchemaOrdered does not warn": {
			data:        multiColumn,
			opts:        []TableOption{WithAutoSchemaOrdered("zebra", "apple", "mango")},
			wantWarning: false,
		},
		"WithAutoSchemaOrdered with partial keys does not warn": {
			// The unlisted columns (apple, mango) are appended after the
			// explicit key in alphabetical order. That remainder ordering is
			// the option's documented contract — the caller opted into
			// detection with a partial order — not a silent guess, so no
			// warning is recorded (T-1451).
			data:        multiColumn,
			opts:        []TableOption{WithAutoSchemaOrdered("zebra")},
			wantWarning: false,
		},
		"single-column map has no order to guess": {
			data:        []map[string]any{{"only": 1}},
			wantWarning: false,
		},
		"empty data has no order to guess": {
			data:        []map[string]any{},
			wantWarning: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			builder := New()
			builder.Table("test-table", tt.data, tt.opts...)

			warnings := keyOrderWarnings(builder.Errors())
			if tt.wantWarning && len(warnings) == 0 {
				t.Errorf("Errors() = %v, want a key order guess warning", builder.Errors())
			}
			if !tt.wantWarning && len(warnings) > 0 {
				t.Errorf("Errors() contains unexpected key order warning: %v", warnings)
			}

			// The warning must be non-fatal: the table is still added.
			doc := builder.Build()
			if got := len(doc.GetContents()); got != 1 {
				t.Errorf("len(GetContents()) = %d, want 1 (warning must not drop the table)", got)
			}
		})
	}
}

// TestBuilderTable_KeyOrderWarningNamesTable verifies the warning identifies
// which table triggered it, so multi-table documents remain debuggable.
func TestBuilderTable_KeyOrderWarningNamesTable(t *testing.T) {
	builder := New()
	builder.Table("ambiguous-columns", []map[string]any{{"b": 1, "a": 2}})

	warnings := keyOrderWarnings(builder.Errors())
	if len(warnings) != 1 {
		t.Fatalf("got %d key order warnings, want 1 (errors: %v)", len(warnings), builder.Errors())
	}
	if !strings.Contains(warnings[0].Error(), "ambiguous-columns") {
		t.Errorf("warning %q does not name the table %q", warnings[0], "ambiguous-columns")
	}
}
