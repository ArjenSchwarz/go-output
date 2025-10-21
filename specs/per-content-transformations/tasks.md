---
references:
    - specs/per-content-transformations/requirements.md
    - specs/per-content-transformations/design.md
    - specs/per-content-transformations/decision_log.md
---
# Per-Content Transformations Implementation Tasks

## Phase 1: Core Infrastructure Setup

- [x] 1. Extend Content interface with transformation support
  - Add Clone() method to Content interface signature
  - Add GetTransformations() []Operation method to Content interface signature
  - Ensure interface compiles (implementation comes in next tasks)
  - Requirements: [1.2](requirements.md#1.2), [1.6](requirements.md#1.6)
  - References: v2/content.go

## Phase 2: TableContent with Transformations (TDD)

- [ ] 2. Write tests for TableContent transformation storage
  - Write tests for WithTransformations() option storing operations
  - Write tests for GetTransformations() returning stored operations
  - Write tests for Clone() preserving transformations
  - Write tests for transformation order preservation
  - Write tests for zero transformations (empty slice)
  - Write tests for operation instances being shared (not cloned)
  - All tests should fail initially (red phase)
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2), [1.4](requirements.md#1.4), [1.5](requirements.md#1.5), [6.1](requirements.md#6.1), [6.2](requirements.md#6.2), [6.3](requirements.md#6.3), [9.1](requirements.md#9.1), [9.3](requirements.md#9.3), [9.4](requirements.md#9.4), [9.5](requirements.md#9.5)
  - References: v2/table_content_test.go

- [ ] 3. Implement TableContent transformations support
  - Add transformations []Operation field to TableContent struct
  - Implement GetTransformations() to return the transformations slice
  - Create WithTransformations(ops ...Operation) TableOption function
  - Update tableConfig struct to include transformations field
  - Update Clone() method to preserve transformations (shallow copy of operation references)
  - Ensure transformations are stored without cloning operation instances
  - Support variadic arguments for multiple operations
  - Allow chaining with other table options like WithKeys() and WithSchema()
  - Run tests - all should pass (green phase)
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2), [1.4](requirements.md#1.4), [1.5](requirements.md#1.5), [6.1](requirements.md#6.1), [6.2](requirements.md#6.2), [6.3](requirements.md#6.3), [9.1](requirements.md#9.1), [9.3](requirements.md#9.3), [9.4](requirements.md#9.4), [9.5](requirements.md#9.5)
  - References: v2/table_content.go

- [ ] 4. Write tests for TextContent transformation storage
  - Write tests for WithTransformations() option storing operations
  - Write tests for GetTransformations() returning stored operations
  - Write tests for Clone() preserving transformations
  - Write tests for transformation order preservation
  - All tests should fail initially (red phase)
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2), [1.4](requirements.md#1.4), [1.5](requirements.md#1.5), [1.6](requirements.md#1.6), [6.1](requirements.md#6.1), [6.2](requirements.md#6.2), [9.1](requirements.md#9.1), [9.3](requirements.md#9.3), [9.4](requirements.md#9.4)
  - References: v2/text_content_test.go

- [ ] 5. Implement TextContent transformations support
  - Add transformations []Operation field to TextContent struct
  - Implement GetTransformations() to return the transformations slice
  - Create WithTransformations(ops ...Operation) TextOption function
  - Update textConfig struct to include transformations field
  - Implement Clone() method to preserve transformations
  - Run tests - all should pass (green phase)
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2), [1.4](requirements.md#1.4), [1.5](requirements.md#1.5), [1.6](requirements.md#1.6), [6.1](requirements.md#6.1), [6.2](requirements.md#6.2), [9.1](requirements.md#9.1), [9.3](requirements.md#9.3), [9.4](requirements.md#9.4)
  - References: v2/text_content.go

- [ ] 6. Write tests for RawContent transformation storage
  - Write tests for WithTransformations() option storing operations
  - Write tests for GetTransformations() returning stored operations
  - Write tests for Clone() preserving transformations
  - All tests should fail initially (red phase)
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2), [1.4](requirements.md#1.4), [1.5](requirements.md#1.5), [1.6](requirements.md#1.6), [6.1](requirements.md#6.1), [6.2](requirements.md#6.2), [9.1](requirements.md#9.1), [9.3](requirements.md#9.3), [9.4](requirements.md#9.4)
  - References: v2/raw_content_test.go

- [ ] 7. Implement RawContent transformations support
  - Add transformations []Operation field to RawContent struct
  - Implement GetTransformations() to return the transformations slice
  - Create WithTransformations(ops ...Operation) RawOption function
  - Update rawConfig struct to include transformations field
  - Implement Clone() method to preserve transformations
  - Run tests - all should pass (green phase)
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2), [1.4](requirements.md#1.4), [1.5](requirements.md#1.5), [1.6](requirements.md#1.6), [6.1](requirements.md#6.1), [6.2](requirements.md#6.2), [9.1](requirements.md#9.1), [9.3](requirements.md#9.3), [9.4](requirements.md#9.4)
  - References: v2/raw_content.go

- [ ] 8. Write tests for SectionContent transformation storage
  - Write tests for WithTransformations() option storing operations
  - Write tests for GetTransformations() returning stored operations
  - Write tests for Clone() preserving transformations
  - Write tests for nested content cloning
  - All tests should fail initially (red phase)
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2), [1.4](requirements.md#1.4), [1.5](requirements.md#1.5), [1.6](requirements.md#1.6), [6.1](requirements.md#6.1), [6.2](requirements.md#6.2), [9.1](requirements.md#9.1), [9.3](requirements.md#9.3), [9.4](requirements.md#9.4)
  - References: v2/section_content_test.go

- [ ] 9. Implement SectionContent transformations support
  - Add transformations []Operation field to SectionContent struct
  - Implement GetTransformations() to return the transformations slice
  - Create WithTransformations(ops ...Operation) SectionOption function
  - Update sectionConfig struct to include transformations field
  - Implement Clone() method to preserve transformations
  - Handle nested content cloning correctly
  - Run tests - all should pass (green phase)
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2), [1.4](requirements.md#1.4), [1.5](requirements.md#1.5), [1.6](requirements.md#1.6), [6.1](requirements.md#6.1), [6.2](requirements.md#6.2), [9.1](requirements.md#9.1), [9.3](requirements.md#9.3), [9.4](requirements.md#9.4)
  - References: v2/section_content.go

## Phase 3: Transformation Execution (TDD)

- [ ] 10. Write tests for transformation execution helper
  - Write tests for applyContentTransformations() with no transformations
  - Write tests for transformations executing in sequence
  - Write tests for validation being called before Apply()
  - Write tests for context cancellation before operations
  - Write tests for error messages including content ID and operation index
  - Write tests for cloning preserving immutability
  - Write tests for lazy execution (transformations don't run during Build())
  - All tests should fail initially (red phase)
  - Requirements: [2.1](requirements.md#2.1), [2.2](requirements.md#2.2), [2.3](requirements.md#2.3), [2.5](requirements.md#2.5), [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.3](requirements.md#4.3), [4.4](requirements.md#4.4), [4.5](requirements.md#4.5), [4.6](requirements.md#4.6), [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3), [5.4](requirements.md#5.4), [8.3](requirements.md#8.3), [8.4](requirements.md#8.4)
  - References: v2/renderer_test.go

- [ ] 11. Implement transformation execution helper
  - Create applyContentTransformations(ctx context.Context, content Content) (Content, error) helper function
  - Check if content has transformations via GetTransformations()
  - Clone content once at the start to preserve immutability
  - Apply transformations in sequence, validating each before execution
  - Check context cancellation before each operation (ctx.Err())
  - Include content ID, operation index, and operation name in error messages
  - Return transformed content or error with clear context
  - Run tests - all should pass (green phase)
  - Requirements: [2.1](requirements.md#2.1), [2.2](requirements.md#2.2), [2.3](requirements.md#2.3), [2.5](requirements.md#2.5), [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.3](requirements.md#4.3), [4.4](requirements.md#4.4), [4.5](requirements.md#4.5), [4.6](requirements.md#4.6), [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3), [5.4](requirements.md#5.4), [8.3](requirements.md#8.3), [8.4](requirements.md#8.4)
  - References: v2/renderer.go

## Phase 4: Renderer Integration (TDD)

- [ ] 12. Write tests for JSONRenderer transformation integration
  - Write tests for JSONRenderer calling applyContentTransformations()
  - Write tests for fail-fast error handling
  - Write tests for context cancellation propagation
  - Write tests for original document remaining unchanged
  - Write tests for mixed content (with/without transformations)
  - All tests should fail initially (red phase)
  - Requirements: [2.1](requirements.md#2.1), [2.3](requirements.md#2.3), [2.4](requirements.md#2.4), [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3), [5.4](requirements.md#5.4)
  - References: v2/renderer_json_test.go

- [ ] 13. Integrate transformation execution into JSONRenderer
  - Update JSONRenderer.Render() to call applyContentTransformations() for each content item
  - Handle transformation errors with fail-fast behavior
  - Propagate context cancellation errors appropriately
  - Ensure original document content remains unchanged
  - Run tests - all should pass (green phase)
  - Requirements: [2.1](requirements.md#2.1), [2.3](requirements.md#2.3), [2.4](requirements.md#2.4), [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3), [5.4](requirements.md#5.4)
  - References: v2/renderer_json.go

- [ ] 14. Write tests for YAMLRenderer transformation integration
  - Write tests for YAMLRenderer calling applyContentTransformations()
  - Write tests for fail-fast error handling
  - Write tests for context cancellation propagation
  - Write tests for original document remaining unchanged
  - All tests should fail initially (red phase)
  - Requirements: [2.1](requirements.md#2.1), [2.3](requirements.md#2.3), [2.4](requirements.md#2.4), [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3), [5.4](requirements.md#5.4)
  - References: v2/renderer_yaml_test.go

- [ ] 15. Integrate transformation execution into YAMLRenderer
  - Update YAMLRenderer.Render() to call applyContentTransformations() for each content item
  - Handle transformation errors with fail-fast behavior
  - Propagate context cancellation errors appropriately
  - Ensure original document content remains unchanged
  - Run tests - all should pass (green phase)
  - Requirements: [2.1](requirements.md#2.1), [2.3](requirements.md#2.3), [2.4](requirements.md#2.4), [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3), [5.4](requirements.md#5.4)
  - References: v2/renderer_yaml.go

- [ ] 16. Write tests for remaining renderers transformation integration
  - Write tests for CSVRenderer calling applyContentTransformations()
  - Write tests for TableRenderer calling applyContentTransformations()
  - Write tests for MarkdownRenderer calling applyContentTransformations()
  - Write tests for HTMLRenderer calling applyContentTransformations()
  - Write tests for consistent error handling across all renderers
  - All tests should fail initially (red phase)
  - Requirements: [2.1](requirements.md#2.1), [2.3](requirements.md#2.3), [2.4](requirements.md#2.4), [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3)
  - References: v2/renderer_csv_test.go, v2/renderer_table_test.go, v2/renderer_markdown_test.go, v2/renderer_html_test.go

- [ ] 17. Integrate transformation execution into remaining renderers
  - Update CSVRenderer.Render() to call applyContentTransformations()
  - Update TableRenderer.Render() to call applyContentTransformations()
  - Update MarkdownRenderer.Render() to call applyContentTransformations()
  - Update HTMLRenderer.Render() to call applyContentTransformations()
  - Ensure consistent error handling across all renderers
  - Run tests - all should pass (green phase)
  - Requirements: [2.1](requirements.md#2.1), [2.3](requirements.md#2.3), [2.4](requirements.md#2.4), [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3)
  - References: v2/renderer_csv.go, v2/renderer_table.go, v2/renderer_markdown.go, v2/renderer_html.go

## Phase 5: Advanced Testing (TDD)

- [ ] 18. Write tests for validation error handling
  - Write tests for configuration errors (nil predicates, negative limits)
  - Write tests for data-dependent errors (missing columns, type mismatches)
  - Write tests for error messages with content ID and operation index
  - Write tests for validation stopping rendering immediately
  - All tests should fail initially (red phase)
  - Requirements: [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.3](requirements.md#4.3), [4.4](requirements.md#4.4), [4.5](requirements.md#4.5), [4.6](requirements.md#4.6), [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3)
  - References: v2/renderer_test.go

- [ ] 19. Implement validation error handling
  - Ensure Validate() is called before Apply() in applyContentTransformations()
  - Ensure validation errors include content ID and operation index
  - Ensure validation errors stop rendering immediately (fail-fast)
  - Handle both configuration and data-dependent errors
  - Run tests - all should pass (green phase)
  - Requirements: [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.3](requirements.md#4.3), [4.4](requirements.md#4.4), [4.5](requirements.md#4.5), [4.6](requirements.md#4.6), [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3)
  - References: v2/renderer.go

- [ ] 20. Write tests for context cancellation handling
  - Write tests for context cancellation detected before operations
  - Write tests for rendering stopping when context is cancelled
  - Write tests for context.Canceled and context.DeadlineExceeded propagation
  - Write tests for cancellation error messages with context
  - All tests should fail initially (red phase)
  - Requirements: [5.4](requirements.md#5.4), [8.4](requirements.md#8.4)
  - References: v2/renderer_test.go

- [ ] 21. Implement context cancellation handling
  - Ensure ctx.Err() is checked before each operation in applyContentTransformations()
  - Ensure context cancellation stops rendering immediately
  - Ensure context errors are properly wrapped with content context
  - Run tests - all should pass (green phase)
  - Requirements: [5.4](requirements.md#5.4), [8.4](requirements.md#8.4)
  - References: v2/renderer.go

## Phase 6: Thread Safety & Performance (TDD)

- [ ] 22. Write thread safety tests
  - Write tests for concurrent rendering of same document with multiple goroutines
  - Write tests for concurrent rendering of different content with same operations
  - Write tests for cloned content independence (mutations don't affect original)
  - Write tests for operation safety during concurrent use
  - Configure tests to run with -race flag
  - All tests should fail initially (red phase)
  - Requirements: [7.1](requirements.md#7.1), [7.2](requirements.md#7.2), [7.3](requirements.md#7.3), [7.6](requirements.md#7.6), [8.3](requirements.md#8.3)
  - References: v2/renderer_test.go, v2/content_test.go

- [ ] 23. Ensure thread safety implementation
  - Verify Content interface implementations are thread-safe
  - Verify Clone() creates independent copies
  - Verify operations don't mutate shared state
  - Run thread safety tests with -race flag - all should pass (green phase)
  - Requirements: [7.1](requirements.md#7.1), [7.2](requirements.md#7.2), [7.3](requirements.md#7.3), [7.6](requirements.md#7.6), [8.3](requirements.md#8.3)
  - References: v2/content.go, v2/table_content.go, v2/text_content.go, v2/raw_content.go, v2/section_content.go

- [ ] 24. Write tests for ValidateStatelessOperation() utility
  - Write tests for utility detecting non-deterministic operations
  - Write tests for utility passing on deterministic operations
  - Write tests demonstrating utility usage with example operations
  - All tests should fail initially (red phase)
  - Requirements: [7.4](requirements.md#7.4), [7.5](requirements.md#7.5), [7.7](requirements.md#7.7), [7.8](requirements.md#7.8), [7.9](requirements.md#7.9)
  - References: v2/testing_utils_test.go

- [ ] 25. Implement ValidateStatelessOperation() testing utility
  - Implement ValidateStatelessOperation(t *testing.T, op Operation, testContent Content) helper
  - Apply operation twice to cloned content
  - Compare results using reflect.DeepEqual()
  - Document limitations: only detects output non-determinism, not hidden state mutations
  - Add godoc with usage examples
  - Run tests - all should pass (green phase)
  - Requirements: [7.4](requirements.md#7.4), [7.5](requirements.md#7.5), [7.7](requirements.md#7.7), [7.8](requirements.md#7.8), [7.9](requirements.md#7.9)
  - References: v2/testing_utils.go

- [ ] 26. Write performance benchmarks
  - Write benchmark for 100 content items with 10 transformations each
  - Write benchmark for 1000 records per table
  - Write benchmark measuring memory overhead of transformation storage
  - Write benchmark measuring transformation execution time
  - Set performance target: 100 items × 10 transformations without degradation
  - Benchmarks establish baseline (no pass/fail yet)
  - Requirements: [8.1](requirements.md#8.1), [8.2](requirements.md#8.2), [8.5](requirements.md#8.5), [8.6](requirements.md#8.6)
  - References: v2/renderer_benchmark_test.go

- [ ] 27. Optimize performance if needed
  - Run benchmarks and analyze results
  - Optimize if performance doesn't meet requirements
  - Re-run benchmarks to verify improvements
  - Document performance characteristics
  - Verify benchmarks meet requirements (green phase)
  - Requirements: [8.1](requirements.md#8.1), [8.2](requirements.md#8.2), [8.5](requirements.md#8.5), [8.6](requirements.md#8.6)
  - References: v2/renderer.go, v2/content.go

## Phase 7: Integration & Examples

- [ ] 28. Write integration tests
  - Write test: Build document with transformations → Render to JSON
  - Write test: Build document with transformations → Render to YAML
  - Write test: Multiple tables with different transformations
  - Write test: Mixed content types (table, text, section) with transformations
  - Write test: Filter + sort + limit transformation chains
  - Write test: Original document data unchanged after rendering
  - All tests should fail initially (red phase)
  - Requirements: [2.1](requirements.md#2.1), [2.2](requirements.md#2.2), [2.3](requirements.md#2.3), [2.4](requirements.md#2.4), [2.5](requirements.md#2.5), [8.3](requirements.md#8.3)
  - References: v2/integration_test.go

- [ ] 29. Fix any integration test failures
  - Run integration tests and fix any failures
  - Ensure end-to-end workflows work correctly
  - Verify transformations work across all renderers
  - Run tests - all should pass (green phase)
  - Requirements: [2.1](requirements.md#2.1), [2.2](requirements.md#2.2), [2.3](requirements.md#2.3), [2.4](requirements.md#2.4), [2.5](requirements.md#2.5), [8.3](requirements.md#8.3)
  - References: v2/renderer.go, v2/content.go

- [ ] 30. Create example code demonstrating per-content transformations
  - Create example: basic filter + sort transformations on a table
  - Create example: multiple tables with different transformations
  - Create example: dynamic transformation construction
  - Create example: error handling
  - Add examples to package documentation
  - Verify examples compile and run correctly
  - Requirements: [9.2](requirements.md#9.2), [9.6](requirements.md#9.6)
  - References: v2/examples/transformations_example.go

## Phase 8: Pipeline Deprecation & Documentation

- [ ] 31. Mark Pipeline API as deprecated
  - Add deprecation comments to Document.Pipeline() method
  - Add deprecation comments to all pipeline operations (Filter, Sort, Limit, etc.)
  - Include clear guidance to use WithTransformations() instead
  - Document timeline for removal in future major version
  - Run existing pipeline tests to ensure backward compatibility
  - Requirements: [3.1](requirements.md#3.1), [3.2](requirements.md#3.2), [3.4](requirements.md#3.4), [3.5](requirements.md#3.5)
  - References: v2/pipeline.go, v2/document.go

- [ ] 32. Create migration examples from Pipeline API
  - Document before/after examples showing pipeline conversion
  - Show how to migrate global transformations to per-content
  - Show how to migrate dynamic transformation construction
  - Create migration guide document
  - Add migration examples to package documentation
  - Create runnable migration example code
  - Requirements: [3.3](requirements.md#3.3)
  - References: v2/MIGRATION.md, v2/examples/migration_example.go

- [ ] 33. Update package documentation
  - Update package-level godoc to describe per-content transformations
  - Document WithTransformations() option for all content types
  - Document thread-safety requirements for operations (stateless, no mutable state)
  - Document closure safety best practices
  - Document performance characteristics and complexity
  - Add example code to godoc
  - Verify documentation with go doc commands
  - Requirements: [7.4](requirements.md#7.4), [7.5](requirements.md#7.5), [7.6](requirements.md#7.6), [7.7](requirements.md#7.7), [8.1](requirements.md#8.1), [8.5](requirements.md#8.5), [8.6](requirements.md#8.6), [9.2](requirements.md#9.2), [9.6](requirements.md#9.6)
  - References: v2/doc.go, v2/table_content.go, v2/operations.go

- [ ] 34. Create best practices guide
  - Document safe vs unsafe operation patterns with examples
  - Provide examples of closure capture best practices
  - Document transformation complexity limits
  - Provide guidance on migration from pipeline API
  - Document common pitfalls and how to avoid them
  - Include code examples for each best practice
  - Requirements: [7.5](requirements.md#7.5), [7.7](requirements.md#7.7), [8.5](requirements.md#8.5)
  - References: v2/BEST_PRACTICES.md

## Phase 9: Final Validation

- [ ] 35. Run final validation
  - Run all tests (unit + integration) with coverage: make test-coverage
  - Run tests with -race flag: make test-race
  - Run golangci-lint: make lint
  - Run go fmt: make fmt
  - Verify all requirements covered by tests (check coverage report)
  - Verify performance benchmarks meet requirements
  - Verify all tests pass and code quality checks succeed
  - Requirements: [7.1](requirements.md#7.1), [7.2](requirements.md#7.2), [7.3](requirements.md#7.3), [8.1](requirements.md#8.1), [8.2](requirements.md#8.2)
  - References: Makefile
