# Per-Content Transformations Requirements

## Introduction

This feature enables developers to attach transformations directly to individual content items (tables, text, sections, etc.) at creation time. This addresses the limitation where current document workflows lack a way to apply different operations (sorting, filtering, etc.) to different tables in the same document.

The document-level pipeline API (Filter, Sort, Limit, etc.) will be deprecated and removed as part of this change, as per-content transformations provide a more natural and flexible model for real-world documents where each table has its own transformation requirements.

Per-content transformations execute during document rendering, applying the specified operations to transform the content before output. The feature is designed to support all content types (TableContent, TextContent, RawContent, SectionContent) to enable future transformation capabilities beyond just table operations.

## Requirements

### 1. Per-Content Transformation Configuration

**User Story:** As a developer, I want to specify transformations when creating content, so that each table or content item can have its own unique transformation logic.

**Acceptance Criteria:**

1. <a name="1.1"></a>The system SHALL provide a `WithTransformations()` option function for content creation that accepts one or more Operation instances
2. <a name="1.2"></a>The system SHALL store the provided transformations as part of the content structure (TableContent, TextContent, RawContent, SectionContent)
3. <a name="1.3"></a>The system SHALL support the same Operation types currently used (FilterOp, SortOp, LimitOp, GroupByOp, AddColumnOp)
4. <a name="1.4"></a>The system SHALL allow zero or more transformations to be specified per content item
5. <a name="1.5"></a>The system SHALL store operations directly without cloning or copying them
6. <a name="1.6"></a>The system SHALL support per-content transformations for all content types (TableContent, TextContent, RawContent, SectionContent)

### 2. Transformation Execution

**User Story:** As a developer, I want transformations to execute during rendering, so that content is transformed just before output.

**Acceptance Criteria:**

1. <a name="2.1"></a>The system SHALL execute per-content transformations during document rendering/output
2. <a name="2.2"></a>The system SHALL execute per-content transformations in the order they were specified in `WithTransformations()`
3. <a name="2.3"></a>The system SHALL apply transformations independently to each content item
4. <a name="2.4"></a>The system SHALL support documents where some content has transformations and other content does not
5. <a name="2.5"></a>The system SHALL skip transformations for content items where none are configured
6. <a name="2.6"></a>The system SHALL use the existing `CanTransform()` method to determine if an operation applies to a content type
7. <a name="2.7"></a>The system SHALL skip non-applicable operations without error (e.g., SortOp on TextContent)

### 3. Pipeline API Deprecation

**User Story:** As a developer, I want a clear migration path from document-level pipelines to per-content transformations, so that I can update my code with minimal disruption.

**Acceptance Criteria:**

1. <a name="3.1"></a>The system SHALL deprecate the document-level Pipeline API (Document.Pipeline(), Filter(), Sort(), etc.)
2. <a name="3.2"></a>The system SHALL mark deprecated methods with clear deprecation notices in documentation
3. <a name="3.3"></a>The system SHALL provide migration examples showing how to convert pipeline code to per-content transformations
4. <a name="3.4"></a>The system SHALL maintain backward compatibility with existing Pipeline API during the deprecation period
5. <a name="3.5"></a>The system SHALL document a timeline for Pipeline API removal in a future major version

### 4. Transformation Validation

**User Story:** As a developer, I want transformation errors to be reported clearly during rendering, so that I can diagnose and fix issues quickly.

**Acceptance Criteria:**

1. <a name="4.1"></a>The system SHALL validate transformations during rendering when they are applied
2. <a name="4.2"></a>The system SHALL call the `Validate()` method on each operation before applying it to check configuration (e.g., nil predicates, negative limits, empty column names)
3. <a name="4.3"></a>The system SHALL validate data-dependent constraints during rendering when data is available (e.g., column existence, data types)
4. <a name="4.4"></a>The system SHALL provide error messages that identify which content item and which transformation failed
5. <a name="4.5"></a>The system SHALL include the content ID and operation index in error messages for debugging
6. <a name="4.6"></a>The system SHALL stop rendering immediately on first validation or transformation error (fail-fast)

### 5. Error Handling During Rendering

**User Story:** As a developer, I want clear error messages when transformations fail during rendering, so that I can diagnose and fix issues quickly.

**Acceptance Criteria:**

1. <a name="5.1"></a>The system SHALL use fail-fast error handling where rendering stops immediately on first transformation error
2. <a name="5.2"></a>The system SHALL return an error that identifies the failing content and operation
3. <a name="5.3"></a>The system SHALL include the content type, content ID, and operation index in error context
4. <a name="5.4"></a>The system SHALL propagate context cancellation errors appropriately

### 6. Content Cloning Behavior

**User Story:** As a developer, I want content cloning to preserve transformation configuration, so that transformations remain attached through the entire processing lifecycle.

**Acceptance Criteria:**

1. <a name="6.1"></a>The system SHALL preserve per-content transformations when content is cloned
2. <a name="6.2"></a>The system SHALL reference the same operation instances in both original and cloned content
3. <a name="6.3"></a>The system SHALL preserve transformation order during cloning
4. <a name="6.4"></a>The system SHALL apply the same cloning behavior to all content types that support transformations

### 7. Thread Safety

**User Story:** As a developer working with concurrent document processing, I want per-content transformations to be thread-safe, so that parallel rendering is safe.

**Acceptance Criteria:**

1. <a name="7.1"></a>The system SHALL ensure per-content transformations do not introduce data races
2. <a name="7.2"></a>The system SHALL maintain the existing thread-safety guarantees of the Document and Content types
3. <a name="7.3"></a>The system SHALL ensure concurrent execution of different content transformations is safe
4. <a name="7.4"></a>The system SHALL document thread-safety requirements for operations, including that operations MUST be stateless (no mutable fields modified during Apply())
5. <a name="7.5"></a>The system SHALL document that operations MUST NOT capture mutable variables in closures (e.g., loop variables captured by reference)
6. <a name="7.6"></a>The system SHALL document that operations MUST be safe for concurrent execution from multiple goroutines
7. <a name="7.7"></a>The system SHALL provide examples of safe and unsafe operation patterns in documentation
8. <a name="7.8"></a>The system SHOULD provide runtime validation in test/debug mode to detect stateful operations (e.g., calling Apply() twice and comparing results)
9. <a name="7.9"></a>The system SHALL document that operations containing mutable state or external side effects are not safe for per-content transformations

### 8. Performance and Resource Management

**User Story:** As a developer building documents with many transformations, I want predictable performance and resource usage, so that my application scales reliably.

**Acceptance Criteria:**

1. <a name="8.1"></a>The system SHALL document the performance characteristics of per-content transformations, including memory overhead and execution time complexity
2. <a name="8.2"></a>The system SHALL support documents with at least 100 content items each having up to 10 transformations without degradation
3. <a name="8.3"></a>The system SHALL execute transformations lazily during rendering rather than eagerly during Build()
4. <a name="8.4"></a>The system SHALL respect existing context cancellation and timeout mechanisms during transformation execution
5. <a name="8.5"></a>The system SHALL provide guidance on transformation complexity limits and performance best practices in documentation
6. <a name="8.6"></a>The system SHALL document that each operation clones content during Apply(), and chained operations result in multiple clones

### 9. API Design and Usability

**User Story:** As a developer, I want an intuitive API for specifying per-content transformations, so that the feature is easy to discover and use correctly.

**Acceptance Criteria:**

1. <a name="9.1"></a>The system SHALL follow the existing functional options pattern used in v2 for configuration
2. <a name="9.2"></a>The system SHALL provide clear documentation examples showing per-content transformation usage
3. <a name="9.3"></a>The system SHALL name the option function consistently with existing patterns (e.g., `WithTransformations()`)
4. <a name="9.4"></a>The system SHALL support variadic arguments in `WithTransformations()` to accept multiple operations
5. <a name="9.5"></a>The system SHALL allow chaining `WithTransformations()` with other content options like `WithKeys()` and `WithSchema()` for tables
6. <a name="9.6"></a>The system SHALL provide examples of common transformation patterns (filtering, sorting, limiting) in documentation
