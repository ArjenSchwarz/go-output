package output

import (
	"context"
	"strings"
	"testing"
)

// These tests are regressions for T-1601.
//
// Bug: a custom per-content Operation can return (nil, nil) from Apply.
// applyContentTransformations accepted that result and assigned it to the
// running content without validation, so renderers received a nil Content.
// The CSV renderer's type switch sent the nil Content to its default branch,
// which called content.AppendText(nil) on the nil interface and panicked.
// In the nested-section path the nil was silently dropped instead.
//
// Expected: a nil Content result from any per-content operation is rejected
// centrally in applyContentTransformations and surfaced as a normal
// transformation error naming the operation, before renderer-specific code
// sees it (mirrors the T-1438 guard on the DataTransformer path).

// nilResultOperation returns a mock operation whose Apply returns (nil, nil),
// breaking the Operation contract.
func nilResultOperation(name string) *mockTransformOperation {
	return &mockTransformOperation{
		name: name,
		applyFunc: func(ctx context.Context, content Content) (Content, error) {
			return nil, nil
		},
	}
}

// nilOpTestTable builds a document whose single table carries the given
// operations as per-content transformations.
func nilOpTestTable(ops ...Operation) *Document {
	return New().
		Table("test", []Record{
			{"id": 1, "status": "active"},
			{"id": 2, "status": "inactive"},
		}, WithKeys("id", "status"), WithTransformations(ops...)).
		Build()
}

// TestApplyContentTransformationsNilOperationResult verifies the central
// guard: an operation returning (nil, nil) must produce an error, not a nil
// Content result.
func TestApplyContentTransformationsNilOperationResult(t *testing.T) {
	nilOp := nilResultOperation("nil-result-op")
	content := nilOpTestTable(nilOp).GetContents()[0]

	result, err := applyContentTransformations(context.Background(), content)

	if err == nil {
		t.Fatalf("expected error for operation returning nil content, got nil error with result %v", result)
	}
	if result != nil {
		t.Errorf("expected nil result alongside error, got %v", result)
	}
	if !strings.Contains(err.Error(), "nil-result-op") {
		t.Errorf("expected error to name the operation %q, got %q", "nil-result-op", err.Error())
	}
	if !strings.Contains(err.Error(), "returned nil content") {
		t.Errorf("expected error to state nil content was returned, got %q", err.Error())
	}
}

// TestApplyContentTransformationsNilResultStopsChain verifies that a nil
// result from one operation is not passed as input to the next operation.
func TestApplyContentTransformationsNilResultStopsChain(t *testing.T) {
	nilOp := nilResultOperation("nil-result-op")
	nextOp := &mockTransformOperation{name: "next-op"}
	content := nilOpTestTable(nilOp, nextOp).GetContents()[0]

	_, err := applyContentTransformations(context.Background(), content)

	if err == nil {
		t.Fatal("expected error for operation returning nil content, got nil")
	}
	if nextOp.applyCalls != 0 {
		t.Errorf("expected subsequent operation not to run after nil result, got %d Apply calls", nextOp.applyCalls)
	}
}

// TestCSVRendererNilOperationResult verifies the CSV renderer returns a
// transformation error instead of panicking when a top-level content's
// operation returns (nil, nil). Before the fix this panicked with a nil
// pointer dereference in the type switch's default branch (AppendText on a
// nil Content).
func TestCSVRendererNilOperationResult(t *testing.T) {
	doc := nilOpTestTable(nilResultOperation("nil-result-op"))
	renderer := &csvRenderer{}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Render panicked with %v — expected transformation error", r)
		}
	}()

	output, err := renderer.Render(context.Background(), doc)

	if err == nil {
		t.Fatalf("expected transformation error, got nil error with output %q", string(output))
	}
	if !strings.Contains(err.Error(), "nil-result-op") {
		t.Errorf("expected error to name the operation %q, got %q", "nil-result-op", err.Error())
	}
}

// TestCSVRendererNilOperationResultInSection verifies the nested-section
// path: before the fix a nil transformed Content inside a section was
// silently dropped; it must surface the same transformation error as the
// top-level path.
func TestCSVRendererNilOperationResultInSection(t *testing.T) {
	doc := New().
		Section("outer", func(b *Builder) {
			b.Table("nested", []Record{
				{"id": 1, "status": "active"},
			}, WithKeys("id", "status"), WithTransformations(nilResultOperation("nil-result-op")))
		}).
		Build()
	renderer := &csvRenderer{}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Render panicked with %v — expected transformation error", r)
		}
	}()

	output, err := renderer.Render(context.Background(), doc)

	if err == nil {
		t.Fatalf("expected transformation error from nested content, got nil error with output %q", string(output))
	}
	if !strings.Contains(err.Error(), "nil-result-op") {
		t.Errorf("expected error to name the operation %q, got %q", "nil-result-op", err.Error())
	}
}
