package output

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// Regression tests for T-1508: MultiError source tracking panics on
// non-comparable errors.
//
// AddWithSource stored errors as keys in SourceMap (map[error]ErrorSource),
// and Error() looked errors up in that map. Both operations hash the error
// interface value, which panics with "hash of unhashable type" when the
// error's dynamic type is not comparable (e.g. a value-receiver struct
// containing a slice or map field). This is public API surface and also
// affects render error aggregation when a custom renderer, transformer, or
// writer returns such an error.

// sliceFieldError is a non-comparable error type: a value-receiver struct
// containing a slice field.
type sliceFieldError struct {
	parts []string
}

func (e sliceFieldError) Error() string { return strings.Join(e.parts, ": ") }

// mapFieldError is a non-comparable error type: a value-receiver struct
// containing a map field.
type mapFieldError struct {
	fields map[string]string
}

func (e mapFieldError) Error() string { return fmt.Sprintf("map error with %d fields", len(e.fields)) }

// dynamicPayloadError is statically comparable (its only field is an
// interface), but hashing it panics at runtime when the interface holds a
// non-comparable value. reflect.Type.Comparable reports true for this type;
// only a dynamic check (reflect.Value.Comparable) catches it.
type dynamicPayloadError struct {
	payload any
}

func (e dynamicPayloadError) Error() string { return fmt.Sprintf("payload: %v", e.payload) }

// TestMultiErrorAddWithSourceNonComparable verifies that AddWithSource does
// not panic for valid error values whose dynamic type cannot be used as a
// map key, and that their source metadata still appears in Error() output.
//
// Expected: no panic; source info rendered for every error.
// Actual (before fix): panic "hash of unhashable type" on AddWithSource.
func TestMultiErrorAddWithSourceNonComparable(t *testing.T) {
	tests := map[string]struct {
		err error
	}{
		"struct with slice field": {
			err: sliceFieldError{parts: []string{"render", "failed"}},
		},
		"struct with map field": {
			err: mapFieldError{fields: map[string]string{"key": "value"}},
		},
		"comparable struct holding non-comparable payload": {
			err: dynamicPayloadError{payload: []int{1, 2, 3}},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			multiErr := NewMultiError("render")
			multiErr.AddWithSource(tt.err, "renderer", map[string]any{
				"format": "json",
			})
			// Add a second error so Error() takes the multi-error path,
			// which performs the source lookup.
			multiErr.AddWithSource(errors.New("second failure"), "writer", map[string]any{
				"format": "csv",
			})

			got := multiErr.Error()

			if !strings.Contains(got, tt.err.Error()) {
				t.Errorf("Error() = %q, want it to contain %q", got, tt.err.Error())
			}
			if !strings.Contains(got, "component=renderer") || !strings.Contains(got, "format=json") {
				t.Errorf("Error() = %q, want source info component=renderer format=json for non-comparable error", got)
			}
			if !strings.Contains(got, "component=writer") {
				t.Errorf("Error() = %q, want source info component=writer for second error", got)
			}
			if len(multiErr.Errors) != 2 {
				t.Errorf("len(Errors) = %d, want 2", len(multiErr.Errors))
			}
		})
	}
}

// TestMultiErrorErrorFormattingNonComparableViaAdd verifies that Error()
// does not panic when a non-comparable error was added via plain Add().
// NewMultiError initializes SourceMap, and since Go 1.12 map lookups hash
// the key even on empty maps, so formatting alone panicked before the fix.
//
// Expected: no panic; both errors formatted.
// Actual (before fix): panic "hash of unhashable type" in Error().
func TestMultiErrorErrorFormattingNonComparableViaAdd(t *testing.T) {
	multiErr := NewMultiError("render")
	multiErr.Add(sliceFieldError{parts: []string{"boom"}})
	multiErr.Add(errors.New("other failure"))

	got := multiErr.Error()

	if !strings.Contains(got, "boom") {
		t.Errorf("Error() = %q, want it to contain %q", got, "boom")
	}
	if !strings.Contains(got, "other failure") {
		t.Errorf("Error() = %q, want it to contain %q", got, "other failure")
	}
}

// TestMultiErrorSourceMapCompatibility verifies the exported SourceMap field
// keeps its existing behaviour for comparable errors: AddWithSource still
// populates it, and entries placed directly into SourceMap by callers are
// still rendered by Error().
func TestMultiErrorSourceMapCompatibility(t *testing.T) {
	t.Run("AddWithSource populates SourceMap for comparable errors", func(t *testing.T) {
		multiErr := NewMultiError("render")
		err := errors.New("comparable failure")
		multiErr.AddWithSource(err, "renderer", map[string]any{"format": "json"})

		source, ok := multiErr.SourceMap[err]
		if !ok {
			t.Fatalf("SourceMap missing entry for comparable error added via AddWithSource")
		}
		if source.Component != "renderer" {
			t.Errorf("SourceMap[err].Component = %q, want %q", source.Component, "renderer")
		}
	})

	t.Run("directly populated SourceMap entries are rendered", func(t *testing.T) {
		errA := errors.New("first failure")
		errB := errors.New("second failure")
		multiErr := &MultiError{
			Operation: "render",
			Errors:    []error{errA, errB},
			SourceMap: map[error]ErrorSource{
				errA: {Component: "renderer", Details: map[string]any{"format": "json"}},
			},
		}

		got := multiErr.Error()
		if !strings.Contains(got, "component=renderer") || !strings.Contains(got, "format=json") {
			t.Errorf("Error() = %q, want source info from directly populated SourceMap", got)
		}
	})
}

// TestMultiErrorInterleavedAddAndAddWithSource verifies positional source
// tracking stays aligned with Errors when Add() and AddWithSource() calls are
// interleaved: Add appends to Errors without recording a source, so the
// source recorded by a later AddWithSource must land on that error's own
// line in Error() output and not shift onto a neighbour.
func TestMultiErrorInterleavedAddAndAddWithSource(t *testing.T) {
	multiErr := NewMultiError("render")
	multiErr.Add(errors.New("first failure"))
	multiErr.AddWithSource(errors.New("second failure"), "renderer", map[string]any{"format": "json"})
	multiErr.Add(errors.New("third failure"))

	got := multiErr.Error()
	lines := strings.Split(got, "\n")

	var numbered []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "1.") || strings.HasPrefix(trimmed, "2.") || strings.HasPrefix(trimmed, "3.") {
			numbered = append(numbered, trimmed)
		}
	}
	if len(numbered) != 3 {
		t.Fatalf("Error() = %q, want 3 numbered error lines, got %d", got, len(numbered))
	}
	if strings.Contains(numbered[0], "component=") {
		t.Errorf("line 1 = %q, want no source info for error added via Add()", numbered[0])
	}
	if !strings.Contains(numbered[1], "second failure") || !strings.Contains(numbered[1], "component=renderer") {
		t.Errorf("line 2 = %q, want %q with source component=renderer", numbered[1], "second failure")
	}
	if strings.Contains(numbered[2], "component=") {
		t.Errorf("line 3 = %q, want no source info for error added via Add()", numbered[2])
	}
}

// TestMultiErrorDuplicateErrorKeepsBothSources locks down the behaviour
// documented in the changelog: the same error value added twice via
// AddWithSource keeps both source entries in Error() output, instead of the
// second overwriting the first as the map-keyed implementation did.
func TestMultiErrorDuplicateErrorKeepsBothSources(t *testing.T) {
	multiErr := NewMultiError("render")
	err := errors.New("shared failure")
	multiErr.AddWithSource(err, "renderer", map[string]any{"format": "json"})
	multiErr.AddWithSource(err, "writer", map[string]any{"format": "csv"})

	got := multiErr.Error()

	if !strings.Contains(got, "component=renderer") || !strings.Contains(got, "format=json") {
		t.Errorf("Error() = %q, want first source entry component=renderer format=json", got)
	}
	if !strings.Contains(got, "component=writer") || !strings.Contains(got, "format=csv") {
		t.Errorf("Error() = %q, want second source entry component=writer format=csv", got)
	}
}
