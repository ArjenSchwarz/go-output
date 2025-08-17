# Transformation Pipeline Decision Log

## Overview
This document tracks key decisions made during the transformation pipeline enhancement design and implementation, including the rationale behind each decision.

## Decisions

### 1. Hybrid Approach: Data and Byte Transformers
**Date:** 2025-08-17
**Decision:** Implement both DataTransformer interface (for structured data) and maintain existing Transformer interface (for bytes)
**Rationale:** 
- Provides backward compatibility with zero breaking changes
- Allows gradual migration from byte to data transformers
- Enables format-specific styling operations to continue using byte transformers
- Gives users flexibility to choose the appropriate transformation level
**Trade-offs:**
- Increased complexity with two transformer types
- Need clear documentation on when to use each type

### 2. Pipeline API as Separate Addition
**Date:** 2025-08-17
**Decision:** Add Pipeline() method to Document rather than replacing existing transformation system
**Rationale:**
- Maintains immutability of Document after Build()
- Provides clean, fluent API for complex operations
- Allows optimization of operation chains
- Keeps simple transformations simple while enabling complex ones
**Trade-offs:**
- Multiple ways to achieve similar results
- Potential user confusion about which approach to use

### 3. Lazy Evaluation in Pipeline
**Date:** 2025-08-17
**Decision:** Pipeline operations are collected and optimized before execution
**Rationale:**
- Enables operation reordering for performance (e.g., filter before sort)
- Reduces data passes through optimization
- Allows for future parallel execution
- Provides opportunity for query planning
**Trade-offs:**
- More complex implementation
- Debugging may be harder with optimized execution

### 4. Immutable Document Transformation
**Date:** 2025-08-17
**Decision:** Pipeline operations return new Document instances rather than modifying existing ones
**Rationale:**
- Consistent with v2's immutability design principle
- Prevents unexpected side effects
- Enables transformation chains without data corruption
- Allows keeping original document for comparison
**Trade-offs:**
- Higher memory usage with document copies
- Need efficient copying mechanisms

### 5. Schema Preservation and Evolution
**Date:** 2025-08-17
**Decision:** Transformations preserve original schema with ability to evolve it (add columns)
**Rationale:**
- Maintains key order preservation (critical v2 feature)
- Allows calculated fields and aggregations
- Ensures format renderers work correctly
- Provides predictable output structure
**Trade-offs:**
- Schema updates need careful handling
- Some operations may require schema rebuilding

### 6. Format-Aware Context in All Transformers
**Date:** 2025-08-17
**Decision:** Pass format information to both data and byte transformers
**Rationale:**
- Enables format-specific optimizations
- Allows transformers to adapt behavior per format
- Supports format-specific features (e.g., HTML classes, Markdown syntax)
- Maintains consistency across transformer types
**Trade-offs:**
- Slightly more complex transformer interface
- Transformers need to handle format parameter

### 7. No Breaking Changes Policy
**Date:** 2025-08-17
**Decision:** All enhancements must be additive with zero breaking changes
**Rationale:**
- Ensures smooth upgrade path for existing users
- Maintains trust in library stability
- Allows incremental adoption of new features
- Follows semantic versioning principles
**Trade-offs:**
- Constrains design choices
- May lead to some API duplication

### 8. Built-in vs Custom Operations
**Date:** 2025-08-17
**Decision:** Provide core operations built-in with mechanism for custom operations
**Rationale:**
- Common operations (filter, sort, limit) available immediately
- Extensibility for domain-specific transformations
- Reduces external dependencies
- Provides consistent operation interface
**Trade-offs:**
- Need to decide which operations are "core"
- Custom operation API needs careful design

## User Decisions (2025-08-17)

After review and discussion, the following decisions have been made:

### 1. Performance Expectations
**Decision:** Performance is not a primary concern for this feature.
**Context:** Most datasets are small, and transformations won't be the main time sink.
**Implementation:** Set realistic expectations without aggressive optimization targets. Acceptable overhead of less than 2x compared to manual manipulation.

### 2. Security and Sandboxing
**Decision:** Not required in initial implementation.
**Context:** This can be addressed in future improvements if needed.
**Implementation:** Defer resource limits, timeouts, and sandboxing to future versions.

### 3. Error Handling Strategy
**Decision:** Fail fast - stop at the first error.
**Context:** Clear, immediate feedback is preferred over batch error collection.
**Implementation:** Pipeline operations will halt and return error context immediately upon failure.

### 4. Implementation Priority
**Decision:** Start with basic operations, but plan to implement all main features.
**Context:** Basic operations (Filter, Sort, Limit) provide immediate value.
**Implementation:** Phase 1 will focus on core operations with other features following quickly.

### 5. Filter Expression Language
**Decision:** Use Go predicate functions only.
**Context:** Maintains simplicity and type safety, aligns with Go idioms.
**Implementation:** Use func(Record) bool predicates for all filtering operations. Expression-based filters may be added in future versions if needed.

### 6. Deferred Features
The following are explicitly deferred to future versions:
- Streaming support for infinite data sources
- Cross-table join operations
- Expression-based filter language (may add later)
- Caching of transformation results
- Parallel execution optimizations
- Security sandboxing and resource limits

## Design Review Decisions (2025-08-17)

After critical review by design-critic and peer-review-validator agents:

### 1. Dual Transformer System Justified
**Decision:** Maintain dual transformer approach despite complexity concerns
**Rationale:** 
- Data and byte transformers serve fundamentally different purposes
- Clear separation between data manipulation and presentation styling
- Enables backward compatibility without compromise
- Both transformer types have clear, non-overlapping use cases

### 2. Enhanced Error Handling and Validation
**Decision:** Add Pipeline.Validate() and enhanced error context
**Context:** Address concerns about debugging and type safety
**Implementation:**
- Pre-execution validation catches issues early
- Mixed content type handling defined explicitly
- Clear error messages with operation context

### 3. Resource Limits from Day One
**Decision:** Include basic resource management in initial implementation
**Context:** Prevent runaway operations without full sandboxing
**Implementation:**
- Operation count limits
- Execution timeouts via context
- Memory usage monitoring (future)

### 4. Schema Evolution Complexity Acknowledged
**Decision:** Implement with extensive testing and validation
**Context:** Schema evolution is complex but necessary
**Implementation:**
- Explicit handling of field conflicts
- Bounds checking for insertions
- Key order preservation guarantees

### 5. Documentation-First Approach
**Decision:** Emphasize clear documentation and examples
**Context:** Complexity requires excellent user guidance
**Focus Areas:**
- When to use each transformer type
- Immutability behavior
- Performance expectations
- Migration patterns

## Implementation Phases

### Phase 1: Core Implementation
Based on user feedback, implement all main features together:
- DataTransformer interface for structured data operations
- Maintain existing Transformer interface for backward compatibility
- Pipeline() method on Document with fluent API
- Basic operations: Filter (with predicates), Sort, Limit
- Aggregation operations: GroupBy, Sum, Count, Average, Min, Max
- Calculated fields with AddColumn
- Renderer integration to detect and apply appropriate transformer type
- Fail-fast error handling with detailed context
- Comprehensive documentation and examples

### Phase 2: Future Enhancements (If Needed)
- Expression-based filter language (SQL-like syntax)
- Performance optimizations and parallel execution
- Caching strategies for transformation results
- Security sandboxing and resource limits
- Cross-table join operations
- Streaming support for large datasets
- Window functions for time-series data