# Draw.io AWS Icons Support for v2

## Introduction

The v1 version of go-output includes built-in AWS service shapes for Draw.io integration, allowing users to create architecture diagrams with proper AWS icons. This feature is currently missing in v2, preventing users from migrating applications that rely on this capability. This document outlines the requirements for implementing AWS icon support in v2.

## Requirements

### 1. AWS Icons Support

**User Story:** As a developer migrating from v1, I want access to the same AWS shapes functionality that was available in v1, so that my existing Draw.io diagrams continue to work.

**Acceptance Criteria:**
1.1. The system SHALL include the complete AWS shapes dataset from v1 (embedded aws.json file)
1.2. The system SHALL provide a `GetAWSShape(group, title string) (string, error)` function that mirrors v1 functionality
1.3. The system SHALL organize AWS shapes by service categories exactly as v1 (Analytics, Compute, Storage, Security Identity Compliance, General Resources, etc.)
1.4. The system SHALL embed the AWS shapes JSON at compile time using `go:embed` for zero-dependency usage
1.5. The system SHALL return an error when a shape is not found (not an empty string)
1.6. The system SHALL provide clear error messages: `fmt.Errorf("shape group %q not found", group)` for missing groups and `fmt.Errorf("shape %q not found in group %q", title, group)` for missing shapes
1.7. The system SHALL use exact case-sensitive matching for group and title parameters (matching v1 behavior)

### 2. Implementation Location

**User Story:** As a developer, I want the AWS shapes functionality in a logical location within v2, so that I can easily find and use it.

**Acceptance Criteria:**
2.1. The system SHALL implement AWS shapes in a new `v2/icons` package
2.2. The system SHALL expose the primary function as `icons.GetAWSShape(group, title string) (string, error)`
2.3. The system SHALL include the embedded aws.json file in the icons package (copied from v1's drawio/shapes/aws.json)
2.4. The system SHALL parse the JSON using sync.Once for lazy initialization with error handling (not in init())
2.5. The system SHALL be thread-safe for concurrent lookups using read-only map access after initialization

### 3. Integration with Draw.io

**User Story:** As a developer, I want to use AWS icons in my Draw.io diagrams through the existing v2 Draw.io API, so that I can create professional architecture diagrams.

**Acceptance Criteria:**
3.1. The system SHALL work with the existing DrawIOHeader structure's Style field
3.2. The system SHALL return Draw.io-compatible style strings that can be used directly in the Style field
3.3. The system SHALL maintain the exact style string format from v1 (no modifications)
3.4. The system SHALL support usage patterns like assigning different icons to different node types in diagrams

### 4. Helper Functions

**User Story:** As a developer, I want helper functions to explore available AWS icons, so that I can discover what's available.

**Acceptance Criteria:**
4.1. The system SHALL provide `AllAWSGroups() []string` to list all available service groups in alphabetical order
4.2. The system SHALL provide `AWSShapesInGroup(group string) ([]string, error)` to list all shapes in a group in alphabetical order
4.3. The system SHALL provide `HasAWSShape(group, title string) bool` to check if a shape exists
4.4. The system SHALL handle JSON parsing errors in helper functions by returning empty slices/false and logging the error

### 5. Performance

**User Story:** As a developer working with diagrams, I want icon lookups to be fast, so that diagram generation remains performant.

**Acceptance Criteria:**
5.1. The system SHALL parse the embedded JSON only once at startup
5.2. The system SHALL use a map-based lookup for O(1) access time
5.3. The system SHALL not re-parse JSON on each lookup
5.4. The system SHALL handle ~600 AWS services without performance degradation

### 6. Testing

**User Story:** As a maintainer, I want comprehensive tests for AWS icons, so that I can ensure the feature works correctly.

**Acceptance Criteria:**
6.1. The system SHALL include unit tests for successful shape lookups
6.2. The system SHALL include unit tests for missing group error cases
6.3. The system SHALL include unit tests for missing shape within group error cases
6.4. The system SHALL include tests for helper functions including alphabetical ordering
6.5. The system SHALL include tests verifying thread safety using Go's race detector (`go test -race`)
6.6. The system SHALL include an integration test showing usage with Draw.io content
6.7. The system SHALL include tests for JSON parsing error handling

### 7. Documentation

**User Story:** As a developer, I want clear documentation on using AWS icons, so that I can quickly integrate them into my code.

**Acceptance Criteria:**
7.1. The system SHALL provide godoc comments for all exported functions
7.2. The system SHALL include a usage example in the package documentation
7.3. The system SHALL reference the helper functions for discovering available groups (not list all ~600 services in docs)
7.4. The system SHALL include migration notes for v1 users explaining the change from `drawio.GetAWSShape()` to `icons.GetAWSShape()`

## Design Decisions

Based on user feedback and requirements analysis:

1. **Scope:** Focus solely on AWS icons for now. No extensible registry system.
2. **API:** Mirror v1's `GetAWSShape` function with error returns
3. **No over-engineering:** Simple map lookup, no lazy loading, no complex caching
4. **Error handling:** Return errors (not empty strings) when shapes are not found
5. **Future extensibility:** Can be added later if needed, but not part of initial implementation