package output

import (
	"context"
	"errors"
	"maps"
	"testing"
)

// mockTransformOperation is a test implementation of Operation with tracking
type mockTransformOperation struct {
	name          string
	validateErr   error
	applyErr      error
	applyFunc     func(ctx context.Context, content Content) (Content, error)
	validateCalls int
	applyCalls    int
}

func (m *mockTransformOperation) Name() string {
	return m.name
}

func (m *mockTransformOperation) Validate() error {
	m.validateCalls++
	return m.validateErr
}

func (m *mockTransformOperation) Apply(ctx context.Context, content Content) (Content, error) {
	m.applyCalls++
	if m.applyFunc != nil {
		return m.applyFunc(ctx, content)
	}
	if m.applyErr != nil {
		return nil, m.applyErr
	}
	return content, nil
}

func (m *mockTransformOperation) CanOptimize(with Operation) bool {
	return false
}

// TestApplyContentTransformations_NoTransformations tests that content without transformations is returned unchanged
func TestApplyContentTransformations_NoTransformations(t *testing.T) {
	tests := map[string]struct {
		content Content
	}{
		"table content": {
			content: &TableContent{
				id:              "test-table",
				title:           "Test Table",
				records:         []Record{{"name": "Alice"}},
				transformations: nil,
			},
		},
		"text content": {
			content: &TextContent{
				id:              "test-text",
				text:            "Hello",
				transformations: nil,
			},
		},
		"empty transformations slice": {
			content: &TableContent{
				id:              "test-table",
				title:           "Test Table",
				records:         []Record{{"name": "Alice"}},
				transformations: []Operation{},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			result, err := applyContentTransformations(ctx, tc.content)

			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if result == nil {
				t.Error("Expected result to be non-nil")
			}

			// Verify content ID is preserved
			if result.ID() != tc.content.ID() {
				t.Errorf("Expected ID %s, got %s", tc.content.ID(), result.ID())
			}
		})
	}
}

// TestApplyContentTransformations_SequentialExecution tests that transformations execute in order
func TestApplyContentTransformations_SequentialExecution(t *testing.T) {
	executionOrder := []string{}

	op1 := &mockTransformOperation{
		name: "operation1",
		applyFunc: func(ctx context.Context, content Content) (Content, error) {
			executionOrder = append(executionOrder, "op1")
			return content, nil
		},
	}

	op2 := &mockTransformOperation{
		name: "operation2",
		applyFunc: func(ctx context.Context, content Content) (Content, error) {
			executionOrder = append(executionOrder, "op2")
			return content, nil
		},
	}

	op3 := &mockTransformOperation{
		name: "operation3",
		applyFunc: func(ctx context.Context, content Content) (Content, error) {
			executionOrder = append(executionOrder, "op3")
			return content, nil
		},
	}

	content := &TableContent{
		id:              "test-table",
		title:           "Test Table",
		records:         []Record{{"name": "Alice"}},
		transformations: []Operation{op1, op2, op3},
	}

	ctx := context.Background()
	_, err := applyContentTransformations(ctx, content)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify execution order
	if len(executionOrder) != 3 {
		t.Errorf("Expected 3 operations to execute, got %d", len(executionOrder))
	}

	expectedOrder := []string{"op1", "op2", "op3"}
	for i, expected := range expectedOrder {
		if i >= len(executionOrder) || executionOrder[i] != expected {
			t.Errorf("Expected operation %d to be %s, got %v", i, expected, executionOrder)
		}
	}
}

// TestApplyContentTransformations_ValidationCalled tests that Validate() is called before Apply()
func TestApplyContentTransformations_ValidationCalled(t *testing.T) {
	tests := map[string]struct {
		operation   *mockTransformOperation
		expectError bool
		errorMsg    string
	}{
		"validation succeeds": {
			operation: &mockTransformOperation{
				name:        "valid-op",
				validateErr: nil,
			},
			expectError: false,
		},
		"validation fails": {
			operation: &mockTransformOperation{
				name:        "invalid-op",
				validateErr: errors.New("validation error"),
			},
			expectError: true,
			errorMsg:    "validation error",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			content := &TableContent{
				id:              "test-table",
				title:           "Test Table",
				records:         []Record{{"name": "Alice"}},
				transformations: []Operation{tc.operation},
			}

			ctx := context.Background()
			_, err := applyContentTransformations(ctx, content)

			// Verify Validate() was called
			if tc.operation.validateCalls != 1 {
				t.Errorf("Expected Validate() to be called once, called %d times", tc.operation.validateCalls)
			}

			if tc.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				if err != nil && !errors.Is(err, tc.operation.validateErr) {
					// Check if error message contains validation error
					if tc.errorMsg != "" && !containsString(err.Error(), tc.errorMsg) {
						t.Errorf("Expected error to contain %q, got: %v", tc.errorMsg, err)
					}
				}

				// Verify Apply() was NOT called after validation failure
				if tc.operation.applyCalls != 0 {
					t.Errorf("Expected Apply() not to be called after validation failure, called %d times", tc.operation.applyCalls)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}

				// Verify Apply() was called after validation success
				if tc.operation.applyCalls != 1 {
					t.Errorf("Expected Apply() to be called once, called %d times", tc.operation.applyCalls)
				}
			}
		})
	}
}

// TestApplyContentTransformations_ContextCancellation tests context cancellation handling
func TestApplyContentTransformations_ContextCancellation(t *testing.T) {
	tests := map[string]struct {
		setupContext func() context.Context
		expectError  bool
		errorMsg     string
	}{
		"already cancelled context": {
			setupContext: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			expectError: true,
			errorMsg:    "context canceled",
		},
		"valid context": {
			setupContext: func() context.Context {
				return context.Background()
			},
			expectError: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			op := &mockTransformOperation{
				name: "test-op",
			}

			content := &TableContent{
				id:              "test-table",
				title:           "Test Table",
				records:         []Record{{"name": "Alice"}},
				transformations: []Operation{op},
			}

			ctx := tc.setupContext()
			_, err := applyContentTransformations(ctx, content)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error due to context cancellation, got nil")
				}
				if err != nil && !containsString(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error to contain %q, got: %v", tc.errorMsg, err)
				}

				// Verify operation was not executed
				if op.validateCalls > 0 {
					t.Error("Expected Validate() not to be called after context cancellation")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestApplyContentTransformations_ErrorMessages tests error message formatting
func TestApplyContentTransformations_ErrorMessages(t *testing.T) {
	tests := map[string]struct {
		operations      []Operation
		contentID       string
		expectedInError []string
	}{
		"validation error includes content ID and operation index": {
			operations: []Operation{
				&mockTransformOperation{name: "op1"},
				&mockTransformOperation{name: "op2", validateErr: errors.New("invalid config")},
			},
			contentID: "test-content-123",
			expectedInError: []string{
				"test-content-123",
				"1", // operation index
				"op2",
				"invalid",
			},
		},
		"apply error includes content ID and operation index": {
			operations: []Operation{
				&mockTransformOperation{name: "op1"},
				&mockTransformOperation{name: "op2"},
				&mockTransformOperation{name: "op3", applyErr: errors.New("apply failed")},
			},
			contentID: "test-content-456",
			expectedInError: []string{
				"test-content-456",
				"2", // operation index (zero-based)
				"op3",
				"failed",
			},
		},
		"first operation error": {
			operations: []Operation{
				&mockTransformOperation{name: "op1", validateErr: errors.New("first error")},
			},
			contentID: "test-content-789",
			expectedInError: []string{
				"test-content-789",
				"0", // operation index
				"op1",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			content := &TableContent{
				id:              tc.contentID,
				title:           "Test Table",
				records:         []Record{{"name": "Alice"}},
				transformations: tc.operations,
			}

			ctx := context.Background()
			_, err := applyContentTransformations(ctx, content)

			if err == nil {
				t.Error("Expected error, got nil")
				return
			}

			errorMsg := err.Error()
			for _, expected := range tc.expectedInError {
				if !containsString(errorMsg, expected) {
					t.Errorf("Expected error message to contain %q, got: %v", expected, errorMsg)
				}
			}
		})
	}
}

// TestApplyContentTransformations_Immutability tests that original content is not modified
func TestApplyContentTransformations_Immutability(t *testing.T) {
	originalRecords := []Record{
		{"name": "Alice", "age": 30},
		{"name": "Bob", "age": 25},
	}

	// Create a deep copy for comparison
	expectedRecords := make([]Record, len(originalRecords))
	for i, rec := range originalRecords {
		expectedRecords[i] = make(Record)
		maps.Copy(expectedRecords[i], rec)
	}

	op := &mockTransformOperation{
		name: "modify-op",
		applyFunc: func(ctx context.Context, content Content) (Content, error) {
			// Try to modify the content (should work on clone, not original)
			if tc, ok := content.(*TableContent); ok {
				records := tc.Records()
				if len(records) > 0 {
					records[0]["modified"] = true
				}
			}
			return content, nil
		},
	}

	content := &TableContent{
		id:              "test-table",
		title:           "Test Table",
		records:         originalRecords,
		transformations: []Operation{op},
	}

	ctx := context.Background()
	result, err := applyContentTransformations(ctx, content)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	// Verify original content records are unchanged
	if len(content.records) != len(expectedRecords) {
		t.Errorf("Original content record count changed: expected %d, got %d", len(expectedRecords), len(content.records))
	}

	for i, rec := range content.records {
		if i >= len(expectedRecords) {
			break
		}

		// Check that "modified" key was not added to original
		if _, exists := rec["modified"]; exists {
			t.Error("Original content was modified - immutability violated")
		}

		// Verify original values are intact
		for k, v := range expectedRecords[i] {
			if rec[k] != v {
				t.Errorf("Original content field %q changed: expected %v, got %v", k, v, rec[k])
			}
		}
	}
}

// TestApplyContentTransformations_LazyExecution tests that transformations don't run during Build()
func TestApplyContentTransformations_LazyExecution(t *testing.T) {
	executed := false
	op := &mockTransformOperation{
		name: "lazy-op",
		applyFunc: func(ctx context.Context, content Content) (Content, error) {
			executed = true
			return content, nil
		},
	}

	// Create a builder and add content with transformations
	builder := New()
	data := []Record{{"name": "Alice"}}

	builder.Table("test", data,
		WithKeys("name"),
		WithTransformations(op),
	)

	// Build the document
	doc := builder.Build()

	// At this point, transformations should NOT have executed
	if executed {
		t.Error("Transformation executed during Build() - should be lazy (execute during rendering)")
	}

	// Verify document was created
	if doc == nil {
		t.Fatal("Expected document to be non-nil")
	}

	contents := doc.GetContents()
	if len(contents) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(contents))
	}

	// Now apply transformations (simulating rendering)
	ctx := context.Background()
	_, err := applyContentTransformations(ctx, contents[0])

	if err != nil {
		t.Errorf("Expected no error during transformation application, got: %v", err)
	}

	// NOW the transformation should have executed
	if !executed {
		t.Error("Transformation did not execute when explicitly applied")
	}
}

// TestApplyContentTransformations_MultipleTransformations tests chaining multiple operations
func TestApplyContentTransformations_MultipleTransformations(t *testing.T) {
	// Track the order and content state through transformations
	callOrder := []string{}

	op1 := &mockTransformOperation{
		name: "op1",
		applyFunc: func(ctx context.Context, content Content) (Content, error) {
			callOrder = append(callOrder, "op1-apply")
			return content, nil
		},
	}

	op2 := &mockTransformOperation{
		name: "op2",
		applyFunc: func(ctx context.Context, content Content) (Content, error) {
			callOrder = append(callOrder, "op2-apply")
			return content, nil
		},
	}

	op3 := &mockTransformOperation{
		name: "op3",
		applyFunc: func(ctx context.Context, content Content) (Content, error) {
			callOrder = append(callOrder, "op3-apply")
			return content, nil
		},
	}

	content := &TableContent{
		id:              "test-table",
		title:           "Test Table",
		records:         []Record{{"name": "Alice"}},
		transformations: []Operation{op1, op2, op3},
	}

	ctx := context.Background()
	result, err := applyContentTransformations(ctx, content)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	// Verify all operations were called in order
	expectedOrder := []string{"op1-apply", "op2-apply", "op3-apply"}
	if len(callOrder) != len(expectedOrder) {
		t.Errorf("Expected %d operations, got %d", len(expectedOrder), len(callOrder))
	}

	for i, expected := range expectedOrder {
		if i >= len(callOrder) || callOrder[i] != expected {
			t.Errorf("Call %d: expected %s, got %v", i, expected, callOrder)
		}
	}

	// Verify validation was called for each operation before apply
	if op1.validateCalls != 1 {
		t.Errorf("Expected op1.Validate() called once, got %d", op1.validateCalls)
	}
	if op2.validateCalls != 1 {
		t.Errorf("Expected op2.Validate() called once, got %d", op2.validateCalls)
	}
	if op3.validateCalls != 1 {
		t.Errorf("Expected op3.Validate() called once, got %d", op3.validateCalls)
	}
}

// TestApplyContentTransformations_StopOnFirstError tests fail-fast behavior
func TestApplyContentTransformations_StopOnFirstError(t *testing.T) {
	op1 := &mockTransformOperation{name: "op1"}
	op2 := &mockTransformOperation{name: "op2", applyErr: errors.New("op2 failed")}
	op3 := &mockTransformOperation{name: "op3"} // Should not be reached

	content := &TableContent{
		id:              "test-table",
		title:           "Test Table",
		records:         []Record{{"name": "Alice"}},
		transformations: []Operation{op1, op2, op3},
	}

	ctx := context.Background()
	_, err := applyContentTransformations(ctx, content)

	if err == nil {
		t.Error("Expected error from op2, got nil")
	}

	// Verify op1 was executed
	if op1.applyCalls != 1 {
		t.Errorf("Expected op1 to be called once, got %d", op1.applyCalls)
	}

	// Verify op2 was executed
	if op2.applyCalls != 1 {
		t.Errorf("Expected op2 to be called once, got %d", op2.applyCalls)
	}

	// Verify op3 was NOT executed (fail-fast)
	if op3.validateCalls > 0 || op3.applyCalls > 0 {
		t.Error("Expected op3 not to be executed after op2 failed (fail-fast)")
	}
}

// containsString checks if a string contains a substring (helper function)
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || stringContains(s, substr))
}

// stringContains is a simple substring check
func stringContains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
