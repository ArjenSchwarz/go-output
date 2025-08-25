# Code Cleanup and Tooling Requirements

## Introduction

This feature encompasses comprehensive code modernization, test improvements, and developer tooling enhancements for the go-output v2 library. The goal is to leverage modern Go 1.24+ features, improve test organization based on current best practices, and provide robust developer tooling through a Makefile.

## Requirements

### 1. Code Modernization

**User Story:** As a developer, I want the codebase to use modern Go idioms and features, so that the code is more efficient, readable, and maintainable.

**Acceptance Criteria:**
1.1. The system SHALL apply all modernize tool suggestions using `modernize -fix ./...`
1.2. The system SHALL ensure all changes maintain backward compatibility with the v2 API
1.3. The system SHALL run `go fmt` after all modernization changes
1.4. The system SHALL verify that all tests pass after modernization

### 2. Test Structure and Organization Improvements

**User Story:** As a developer, I want tests organized according to Go best practices, so that they are maintainable, efficient, and provide clear feedback.

**Acceptance Criteria:**
2.1. The system SHALL separate unit tests from integration tests using environment variables (e.g., `INTEGRATION` environment variable)
2.2. The system SHALL document the integration test separation strategy in the project README or CLAUDE.md
2.3. The system SHALL maintain test files adjacent to source files following `*_test.go` naming convention
2.4. The system SHALL split test files larger than 800 lines by logical functionality (e.g., `handler_auth_test.go`, `handler_validation_test.go`)
2.5. The system SHALL use descriptive test names and `got`/`want` naming conventions where applicable

### 3. Benchmark Modernization

**User Story:** As a developer, I want benchmarks using modern Go 1.24 patterns, so that performance measurements are accurate and immune to compiler optimizations.

**Acceptance Criteria:**
3.1. The system SHALL modernize benchmarks as part of the overall modernize tool application
3.2. The system SHALL verify that all benchmarks continue to run successfully after modernization

### 4. Developer Tooling via Makefile

**User Story:** As a developer, I want a comprehensive Makefile with common development commands, so that I can efficiently run tests, linting, and other development tasks.

**Acceptance Criteria:**
4.1. The Makefile SHALL provide a `test` target that runs unit tests
4.2. The Makefile SHALL provide a `test-integration` target that runs integration tests (with INTEGRATION=1)
4.3. The Makefile SHALL provide a `test-all` target that runs both unit and integration tests
4.4. The Makefile SHALL provide a `test-coverage` target that generates coverage reports
4.5. The Makefile SHALL provide a `lint` target that runs golangci-lint
4.6. The Makefile SHALL provide a `fmt` target that runs go fmt on v2 code and all example directories
4.7. The Makefile SHALL provide a `mod-tidy` target that runs go mod tidy
4.8. The Makefile SHALL provide a `benchmark` target for running performance tests
4.9. The Makefile SHALL provide a `check` target that runs fmt, lint, and tests in sequence
4.10. The Makefile SHALL provide a `modernize` target that runs the modernize tool
4.11. The Makefile SHALL provide a `clean` target to remove generated files and test caches
4.12. The Makefile SHALL include help documentation for each target

### 5. Code Quality Standards

**User Story:** As a maintainer, I want consistent code quality standards applied throughout the codebase, so that contributions are uniform and professional.

**Acceptance Criteria:**
5.1. The system SHALL pass all golangci-lint checks after modernization
5.2. The system SHALL maintain all existing tests passing after changes
5.3. The system SHALL use modern Go constructs as applied by the modernize tool

## Success Criteria

- All modernize tool suggestions are implemented using `modernize -fix`
- All tests pass after modernization
- Makefile provides easy access to both unit and integration tests
- Integration test separation is clearly documented
- Code maintains v2 API compatibility

## Technical Constraints

- Must maintain compatibility with Go 1.24+
- Must not break existing v2 API
- Changes must be limited to the v2 directory only

## Excluded from Scope

- Changes to v1 code (parent directory)
- API breaking changes
- New features beyond cleanup and tooling
- Performance optimization beyond what modernize provides