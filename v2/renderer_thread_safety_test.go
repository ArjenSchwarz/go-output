package output

import (
	"context"
	"sync"
	"testing"
)

// TestConcurrentRendering_SameDocument tests concurrent rendering of the same document with multiple goroutines
func TestConcurrentRendering_SameDocument(t *testing.T) {
	tests := map[string]struct {
		setupDoc      func() *Document
		numGoroutines int
	}{
		"simple table without transformations": {
			setupDoc: func() *Document {
				builder := New()
				builder.Table("users", []Record{
					{"name": "Alice", "age": 30},
					{"name": "Bob", "age": 25},
				}, WithKeys("name", "age"))
				return builder.Build()
			},
			numGoroutines: 10,
		},
		"table with filter transformation": {
			setupDoc: func() *Document {
				builder := New()
				builder.Table("users", []Record{
					{"name": "Alice", "age": 30},
					{"name": "Bob", "age": 25},
					{"name": "Charlie", "age": 35},
				},
					WithKeys("name", "age"),
					WithTransformations(
						NewFilterOp(func(r Record) bool {
							return r["age"].(int) >= 30
						}),
					),
				)
				return builder.Build()
			},
			numGoroutines: 10,
		},
		"table with multiple transformations": {
			setupDoc: func() *Document {
				builder := New()
				builder.Table("users", []Record{
					{"name": "Alice", "age": 30},
					{"name": "Bob", "age": 25},
					{"name": "Charlie", "age": 35},
					{"name": "David", "age": 28},
				},
					WithKeys("name", "age"),
					WithTransformations(
						NewFilterOp(func(r Record) bool {
							return r["age"].(int) >= 25
						}),
						NewSortOp(SortKey{Column: "age", Direction: Ascending}),
						NewLimitOp(2),
					),
				)
				return builder.Build()
			},
			numGoroutines: 10,
		},
		"multiple tables with different transformations": {
			setupDoc: func() *Document {
				builder := New()
				builder.Table("users", []Record{
					{"name": "Alice", "age": 30},
					{"name": "Bob", "age": 25},
				},
					WithKeys("name", "age"),
					WithTransformations(
						NewFilterOp(func(r Record) bool {
							return r["age"].(int) >= 30
						}),
					),
				)
				builder.Table("products", []Record{
					{"id": 1, "name": "Widget", "price": 19.99},
					{"id": 2, "name": "Gadget", "price": 29.99},
				},
					WithKeys("id", "name", "price"),
					WithTransformations(
						NewSortOp(SortKey{Column: "price", Direction: Descending}),
					),
				)
				return builder.Build()
			},
			numGoroutines: 10,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			doc := tc.setupDoc()
			ctx := context.Background()

			var wg sync.WaitGroup
			errChan := make(chan error, tc.numGoroutines)

			// Launch multiple goroutines that render the same document concurrently
			for range tc.numGoroutines {
				wg.Add(1)
				go func() {
					defer wg.Done()

					// Apply transformations to all content
					for _, content := range doc.GetContents() {
						_, err := applyContentTransformations(ctx, content)
						if err != nil {
							errChan <- err
							return
						}
					}
				}()
			}

			wg.Wait()
			close(errChan)

			// Check for any errors
			for err := range errChan {
				t.Errorf("Concurrent rendering failed: %v", err)
			}
		})
	}
}

// TestConcurrentRendering_DifferentContent tests concurrent rendering of different content with the same operations
func TestConcurrentRendering_DifferentContent(t *testing.T) {
	tests := map[string]struct {
		numContents   int
		numGoroutines int
	}{
		"10 tables with 10 goroutines": {
			numContents:   10,
			numGoroutines: 10,
		},
		"100 tables with 20 goroutines": {
			numContents:   100,
			numGoroutines: 20,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			// Create multiple content items with the same transformation operations
			filterOp := NewFilterOp(func(r Record) bool {
				return r["age"].(int) >= 30
			})
			sortOp := NewSortOp(SortKey{Column: "age", Direction: Ascending})

			contents := make([]Content, tc.numContents)
			for i := range tc.numContents {
				contents[i] = &TableContent{
					id:    GenerateID(),
					title: "Test Table",
					records: []Record{
						{"name": "Alice", "age": 30},
						{"name": "Bob", "age": 25},
						{"name": "Charlie", "age": 35},
					},
					schema: &Schema{
						Fields:   []Field{{Name: "name"}, {Name: "age"}},
						keyOrder: []string{"name", "age"},
					},
					transformations: []Operation{filterOp, sortOp},
				}
			}

			ctx := context.Background()
			var wg sync.WaitGroup
			errChan := make(chan error, tc.numGoroutines)

			// Process different content items concurrently
			for _, content := range contents {
				wg.Add(1)
				go func(c Content) {
					defer wg.Done()

					_, err := applyContentTransformations(ctx, c)
					if err != nil {
						errChan <- err
					}
				}(content)
			}

			wg.Wait()
			close(errChan)

			// Check for any errors
			for err := range errChan {
				t.Errorf("Concurrent rendering failed: %v", err)
			}
		})
	}
}

// TestClonedContent_Independence tests that cloned content is independent and mutations don't affect original
func TestClonedContent_Independence(t *testing.T) {
	tests := map[string]struct {
		createContent  func() Content
		mutateClone    func(Content) error
		verifyOriginal func(*testing.T, Content)
	}{
		"table content clone independence": {
			createContent: func() Content {
				return &TableContent{
					id:    "original",
					title: "Original Table",
					records: []Record{
						{"name": "Alice", "age": 30},
						{"name": "Bob", "age": 25},
					},
					schema: &Schema{
						Fields:   []Field{{Name: "name"}, {Name: "age"}},
						keyOrder: []string{"name", "age"},
					},
					transformations: []Operation{
						NewFilterOp(func(r Record) bool {
							return r["age"].(int) >= 30
						}),
					},
				}
			},
			mutateClone: func(c Content) error {
				// Mutate the clone's records
				tc := c.(*TableContent)
				if len(tc.records) > 0 {
					tc.records[0]["name"] = "Modified"
					tc.records[0]["age"] = 999
				}
				return nil
			},
			verifyOriginal: func(t *testing.T, c Content) {
				tc := c.(*TableContent)
				if len(tc.records) == 0 {
					t.Error("Original content should have records")
					return
				}
				if tc.records[0]["name"] != "Alice" {
					t.Errorf("Original record name = %v, want Alice", tc.records[0]["name"])
				}
				if tc.records[0]["age"] != 30 {
					t.Errorf("Original record age = %v, want 30", tc.records[0]["age"])
				}
			},
		},
		"text content clone independence": {
			createContent: func() Content {
				return &TextContent{
					id:    "original-text",
					text:  "Original Text",
					style: TextStyle{Bold: true},
					transformations: []Operation{
						&mockTransformOperation{name: "test-op"},
					},
				}
			},
			mutateClone: func(c Content) error {
				// Mutate the clone
				tc := c.(*TextContent)
				tc.text = "Modified Text"
				tc.style.Bold = false
				return nil
			},
			verifyOriginal: func(t *testing.T, c Content) {
				tc := c.(*TextContent)
				if tc.text != "Original Text" {
					t.Errorf("Original text = %v, want 'Original Text'", tc.text)
				}
				if !tc.style.Bold {
					t.Error("Original style should be bold")
				}
			},
		},
		"transformations independence": {
			createContent: func() Content {
				return &TableContent{
					id:    "test",
					title: "Test",
					records: []Record{
						{"name": "Alice", "age": 30},
					},
					schema: &Schema{
						Fields:   []Field{{Name: "name"}, {Name: "age"}},
						keyOrder: []string{"name", "age"},
					},
					transformations: []Operation{
						NewFilterOp(func(r Record) bool { return true }),
						NewSortOp(SortKey{Column: "age", Direction: Ascending}),
					},
				}
			},
			mutateClone: func(c Content) error {
				// Try to append to transformations slice
				tc := c.(*TableContent)
				// This should not affect the original
				tc.transformations = append(tc.transformations, NewLimitOp(5))
				return nil
			},
			verifyOriginal: func(t *testing.T, c Content) {
				tc := c.(*TableContent)
				if len(tc.transformations) != 2 {
					t.Errorf("Original should have 2 transformations, got %d", len(tc.transformations))
				}
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			original := tc.createContent()
			clone := original.Clone()

			// Mutate the clone
			if err := tc.mutateClone(clone); err != nil {
				t.Fatalf("Failed to mutate clone: %v", err)
			}

			// Verify original is unchanged
			tc.verifyOriginal(t, original)
		})
	}
}

// TestConcurrentCloning tests that cloning is safe when done concurrently
func TestConcurrentCloning(t *testing.T) {
	tests := map[string]struct {
		createContent func() Content
		numGoroutines int
	}{
		"table content concurrent cloning": {
			createContent: func() Content {
				return &TableContent{
					id:    "test",
					title: "Test Table",
					records: []Record{
						{"name": "Alice", "age": 30},
						{"name": "Bob", "age": 25},
					},
					schema: &Schema{
						Fields:   []Field{{Name: "name"}, {Name: "age"}},
						keyOrder: []string{"name", "age"},
					},
					transformations: []Operation{
						NewFilterOp(func(r Record) bool { return true }),
					},
				}
			},
			numGoroutines: 100,
		},
		"text content concurrent cloning": {
			createContent: func() Content {
				return &TextContent{
					id:    "test-text",
					text:  "Test Text",
					style: TextStyle{Bold: true},
					transformations: []Operation{
						&mockTransformOperation{name: "test-op"},
					},
				}
			},
			numGoroutines: 100,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			original := tc.createContent()
			var wg sync.WaitGroup
			errChan := make(chan error, tc.numGoroutines)

			// Clone concurrently
			for range tc.numGoroutines {
				wg.Add(1)
				go func() {
					defer wg.Done()

					clone := original.Clone()
					if clone == nil {
						errChan <- NewValidationError("clone", clone, "clone returned nil")
						return
					}

					if clone.ID() != original.ID() {
						errChan <- NewValidationError("clone.ID", clone.ID(), "clone ID does not match original")
					}
				}()
			}

			wg.Wait()
			close(errChan)

			// Check for any errors
			for err := range errChan {
				t.Errorf("Concurrent cloning failed: %v", err)
			}
		})
	}
}

// TestOperationSafety_ConcurrentExecution tests that operations are safe for concurrent execution
func TestOperationSafety_ConcurrentExecution(t *testing.T) {
	tests := map[string]struct {
		createOp      func() Operation
		createContent func() Content
		numGoroutines int
	}{
		"filter operation concurrent execution": {
			createOp: func() Operation {
				return NewFilterOp(func(r Record) bool {
					return r["age"].(int) >= 30
				})
			},
			createContent: func() Content {
				return &TableContent{
					id:    GenerateID(),
					title: "Test",
					records: []Record{
						{"name": "Alice", "age": 30},
						{"name": "Bob", "age": 25},
						{"name": "Charlie", "age": 35},
					},
					schema: &Schema{
						Fields:   []Field{{Name: "name"}, {Name: "age"}},
						keyOrder: []string{"name", "age"},
					},
				}
			},
			numGoroutines: 50,
		},
		"sort operation concurrent execution": {
			createOp: func() Operation {
				return NewSortOp(SortKey{Column: "age", Direction: Ascending})
			},
			createContent: func() Content {
				return &TableContent{
					id:    GenerateID(),
					title: "Test",
					records: []Record{
						{"name": "Alice", "age": 30},
						{"name": "Bob", "age": 25},
						{"name": "Charlie", "age": 35},
					},
					schema: &Schema{
						Fields:   []Field{{Name: "name"}, {Name: "age"}},
						keyOrder: []string{"name", "age"},
					},
				}
			},
			numGoroutines: 50,
		},
		"limit operation concurrent execution": {
			createOp: func() Operation {
				return NewLimitOp(2)
			},
			createContent: func() Content {
				return &TableContent{
					id:    GenerateID(),
					title: "Test",
					records: []Record{
						{"name": "Alice", "age": 30},
						{"name": "Bob", "age": 25},
						{"name": "Charlie", "age": 35},
					},
					schema: &Schema{
						Fields:   []Field{{Name: "name"}, {Name: "age"}},
						keyOrder: []string{"name", "age"},
					},
				}
			},
			numGoroutines: 50,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {

			op := tc.createOp()
			ctx := context.Background()
			var wg sync.WaitGroup
			errChan := make(chan error, tc.numGoroutines)

			// Execute the same operation concurrently on different content
			for range tc.numGoroutines {
				wg.Add(1)
				go func() {
					defer wg.Done()

					content := tc.createContent()
					_, err := op.Apply(ctx, content)
					if err != nil {
						errChan <- err
					}
				}()
			}

			wg.Wait()
			close(errChan)

			// Check for any errors
			for err := range errChan {
				t.Errorf("Concurrent operation execution failed: %v", err)
			}
		})
	}
}
