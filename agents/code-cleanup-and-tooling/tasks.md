# Code Cleanup and Tooling Implementation Tasks

## Implementation Plan

This document outlines the implementation tasks for the code cleanup and tooling feature. Each task is designed to be executed by a coding agent and focuses on writing, modifying, or testing code.

### 1. Apply Automated Code Modernization

- [x] **1.1 Write unit tests to validate modernization doesn't break functionality**
  - Create comprehensive test suite that captures current behavior
  - Focus on areas with the most modernize suggestions (performance optimizations, modern idioms)
  - Reference: Requirements 1.4, 3.2

- [x] **1.2 Execute modernize tool with -fix flag on v2 directory**
  - Run `modernize -fix ./...` in the v2 directory
  - Apply all 118 identified improvements automatically
  - Reference: Requirements 1.1

- [x] **1.3 Format all modernized code**
  - Run `go fmt ./...` on v2 directory
  - Ensure consistent formatting after modernization
  - Reference: Requirements 1.3

- [x] **1.4 Verify tests pass after modernization**
  - Run entire test suite to confirm no breaking changes
  - Fix any test failures caused by modernization
  - Reference: Requirements 1.4, 5.2

- [x] **1.5 Apply modernization to example directories**
  - Run modernize tool on each directory in v2/examples/*/
  - Format each example directory with go fmt
  - Reference: Design Phase 1 Step 3

### 2. Implement Test Organization Helper Functions

- [ ] **2.1 Create test helper for integration test separation**
  - Write `skipIfNotIntegration` helper function
  - Use environment variable INTEGRATION=1 check
  - Reference: Requirements 2.1, Design Integration Test Separation

- [ ] **2.2 Write tests for the integration test helper**
  - Test behavior with and without INTEGRATION environment variable
  - Verify proper skip messages
  - Reference: Requirements 2.1

### 3. Split Large Test Files

- [ ] **3.1 Identify test files exceeding 800 lines**
  - Use find_long_files tool to locate test files over 800 lines in v2 directory
  - Review identified files to determine logical boundaries for splitting
  - Reference: Requirements 2.4

- [ ] **3.2 Split pipeline_test.go by operation type**
  - Create pipeline_filter_test.go for filter operations
  - Create pipeline_sort_test.go for sorting operations
  - Create pipeline_aggregate_test.go for aggregation operations
  - Create pipeline_transform_test.go for transformation operations
  - Create pipeline_validation_test.go for validation and error cases
  - Reference: Requirements 2.4, Design Test File Splitting Strategy

- [ ] **3.3 Split renderer_test.go by renderer type**
  - Create renderer_json_test.go for JSON rendering tests
  - Create renderer_yaml_test.go for YAML rendering tests
  - Create renderer_html_test.go for HTML rendering tests
  - Create renderer_markdown_test.go for Markdown rendering tests
  - Reference: Requirements 2.4, Design Test File Splitting Strategy

- [ ] **3.4 Split other large test files by logical functionality**
  - Split operations_test.go by operation category if > 800 lines
  - Split errors_test.go by error type if > 800 lines
  - Split progress_test.go by progress feature if > 800 lines
  - Reference: Requirements 2.4

- [ ] **3.5 Verify all tests pass after splitting**
  - Run full test suite to ensure no tests were lost or broken
  - Confirm each new test file follows naming conventions
  - Reference: Requirements 2.3, 2.4

### 4. Create Makefile with Developer Tooling

- [ ] **4.1 Create base Makefile structure with help system**
  - Implement help target that documents all available targets
  - Set up proper .PHONY declarations
  - Reference: Requirements 4.12

- [ ] **4.2 Implement test targets**
  - Create `test` target for unit tests
  - Create `test-integration` target with INTEGRATION=1
  - Create `test-all` target combining both
  - Create `test-coverage` target with coverage report generation
  - Reference: Requirements 4.1, 4.2, 4.3, 4.4

- [ ] **4.3 Implement code quality targets**
  - Create `lint` target running golangci-lint
  - Create `fmt` target for v2 and all example directories
  - Create `modernize` target running the modernize tool
  - Reference: Requirements 4.5, 4.6, 4.10

- [ ] **4.4 Implement development utility targets**
  - Create `mod-tidy` target for go mod tidy
  - Create `benchmark` target for performance tests
  - Create `clean` target to remove generated files and test caches
  - Reference: Requirements 4.7, 4.8, 4.11

- [ ] **4.5 Implement composite check target**
  - Create `check` target that runs fmt, lint, and tests in sequence
  - Ensure proper error propagation between steps
  - Reference: Requirements 4.9

- [ ] **4.6 Write tests to verify Makefile targets work correctly**
  - Create script to test each Makefile target
  - Verify targets fail appropriately on errors
  - Reference: Requirements 4.1-4.12

### 5. Update Integration Test Markers

- [ ] **5.1 Identify existing integration tests in the codebase**
  - Search for tests that require external resources or full system setup
  - Create list of tests to be marked as integration tests
  - Reference: Requirements 2.1

- [ ] **5.2 Add skipIfNotIntegration calls to integration tests**
  - Modify identified integration tests to use the helper function
  - Ensure skip messages are descriptive
  - Reference: Requirements 2.1, Design Integration Test Separation

- [ ] **5.3 Verify integration test separation works correctly**
  - Run tests without INTEGRATION=1 and verify integration tests skip
  - Run tests with INTEGRATION=1 and verify all tests run
  - Reference: Requirements 2.1

### 6. Apply Test Best Practices

- [ ] **6.1 Convert tests to use map-based table-driven pattern**
  - Identify tests using slice-based tables
  - Convert to map[string]struct pattern for better test isolation
  - Reference: Requirements 2.5, Design Table-Driven Tests

- [ ] **6.2 Apply got/want naming conventions**
  - Update test variable names to use consistent got/want pattern
  - Ensure error messages clearly show expected vs actual
  - Reference: Requirements 2.5

- [ ] **6.3 Add descriptive test names**
  - Review test names for clarity and descriptiveness
  - Update test names to clearly indicate what is being tested
  - Reference: Requirements 2.3, 2.5

### 7. Update Documentation

- [ ] **7.1 Update CLAUDE.md with new development commands**
  - Add Makefile targets to development commands section
  - Document integration test separation strategy
  - Reference: Requirements 2.2, Design Phase 4

- [ ] **7.2 Create or update README with testing strategy**
  - Document how to run different test types
  - Explain test organization and file structure
  - Add quick start guide for contributors
  - Reference: Requirements 2.2, Design Phase 4

### 8. Final Validation

- [ ] **8.1 Run complete validation suite**
  - Execute `make check` to run full validation
  - Verify all golangci-lint checks pass
  - Confirm all tests pass including integration tests
  - Reference: Requirements 5.1, 5.2

- [ ] **8.2 Verify benchmark modernization**
  - Run benchmarks to ensure they work with b.Loop() pattern
  - Compare performance metrics before and after changes
  - Reference: Requirements 3.1, 3.2

- [ ] **8.3 Validate API compatibility**
  - Create test to verify v2 API remains unchanged
  - Ensure no breaking changes were introduced
  - Reference: Requirements 1.2, Design Backward Compatibility