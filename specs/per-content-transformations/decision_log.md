# Decision Log: Per-Content Transformations

## Overview

This document tracks key decisions made during the requirements phase for the per-content transformations feature.

---

## Decision 1: Remove Document-Level Pipeline Operations

**Date:** 2025-01-19

**Context:** Initial design included both per-content transformations and document-level pipeline operations, with per-content executing first.

**Decision:** Deprecate and remove document-level pipeline operations entirely.

**Rationale:**
- Per-content transformations provide a more natural model for real-world documents where each table has unique requirements
- Dual transformation systems (per-content + global) created complexity and optimization conflicts
- The pipeline approach was not yet heavily used in v2
- Simpler mental model: transformations belong to content, not documents

**Consequences:**
- Breaking change requiring migration path
- Clearer architecture with single transformation approach
- Loss of global optimization across content items (acceptable trade-off)

**Alternatives Considered:**
- Tagged operations (operations with content filters)
- Keep both systems (rejected due to complexity)

---

## Decision 2: Store Operations Directly Without Cloning

**Date:** 2025-01-19

**Context:** Initial requirements mandated deep copying operations to prevent shared state issues.

**Decision:** Store operation references directly without cloning.

**Rationale:**
- Deep copying Go functions/closures is technically impossible without serialization
- Operations in v2 are already designed as value types or stateless function holders
- Cloning adds complexity and memory overhead
- Thread-safety is enforced through documentation and operation design, not copying

**Consequences:**
- Operations must be stateless and safe for concurrent use (documented requirement)
- Closure capture becomes a developer responsibility with clear documentation
- Memory efficient (single operation instance shared across uses)
- Risk of shared mutable state if operations aren't designed correctly

**Alternatives Considered:**
- Deep copy via reflection (rejected: doesn't work with functions)
- Factory function pattern (rejected: less ergonomic)
- Immutable operation builder pattern (deferred to v3 consideration)

---

## Decision 3: Validation During Build() Phase

**Date:** 2025-01-19

**Context:** Needed to specify when transformation validation occurs.

**Decision:** Perform static configuration validation during Build(), data-dependent validation during rendering.

**Rationale:**
- Build() is the natural point for early error detection
- Static configuration errors (nil predicates, negative limits) can be caught without data
- Data-dependent errors (column existence) require actual data and must wait for rendering
- Aligns with existing Builder pattern error collection via HasErrors()/Errors()

**Consequences:**
- Two-phase validation: configuration early, data late
- Build() can complete with errors (queryable via API)
- Developers get early feedback on configuration issues
- Data shape errors appear at render time (documented trade-off)

**Alternatives Considered:**
- Eager validation during WithTransformations() (rejected: too early, no error collection)
- All validation at render time (rejected: late error detection)
- Schema-based validation at Build() (rejected: doesn't handle dynamic transformations)

---

## Decision 4: Configurable Failure Behavior

**Date:** 2025-01-19

**Context:** Need to specify what happens when a transformation fails during rendering.

**Decision:** Provide two modes - fail-fast (default) and partial rendering.

**Rationale:**
- Different use cases have different needs:
  - Reports: partial output may be valuable
  - Transactions: all-or-nothing is required
- Fail-fast as default provides safe, predictable behavior
- Partial rendering as opt-in supports resilience scenarios
- Aligns with context cancellation patterns

**Consequences:**
- Additional configuration API surface
- More complex error handling implementation
- Flexibility for different use cases
- Clear documentation needed on mode selection

**Alternatives Considered:**
- Always fail-fast (rejected: too rigid for some use cases)
- Always partial (rejected: unsafe default)
- Per-content error handlers (rejected: too complex)

---

## Decision 5: Thread-Safety Through Documentation

**Date:** 2025-01-19

**Context:** Operations may be used concurrently when rendering multiple content items.

**Decision:** Require operations to be stateless via documentation, with optional runtime validation in test mode.

**Rationale:**
- Go cannot enforce statelessness at compile time
- Operations are already designed to be stateless in v2
- Runtime validation in tests can catch violations during development
- Documentation with clear examples prevents most issues
- Follows Go's pragmatic approach (race detector, vet, etc.)

**Consequences:**
- Thread-safety is a contract, not a compile-time guarantee
- Requires clear documentation with safe/unsafe patterns
- Runtime validation adds test-time overhead (but catches bugs)
- Developer responsibility to follow guidelines

**Alternatives Considered:**
- Marker interface for stateless operations (rejected: doesn't actually enforce)
- Mutex-wrapped operations (rejected: performance overhead)
- Pure data operations only (deferred to v3)

---

## Decision 6: Extend Content Interface for Transformations

**Date:** 2025-01-21

**Context:** Should transformations use a separate TransformableContent interface or extend the existing Content interface?

**Decision:** Extend the existing Content interface to include Clone() and GetTransformations() methods.

**Rationale:**
- Every content type is transformable - no need for a separate interface
- Simpler mental model: transformations are a fundamental capability of content
- Eliminates unnecessary type assertions and interface checks
- Content already has ID(), Title(), Type() - transformation methods fit naturally
- Clone() already exists in all content types, just not formalized in interface

**Consequences:**
- Content interface gains two new methods
- All content types must implement GetTransformations() and Clone()
- No TransformableContent interface needed
- Type assertions eliminated from rendering code

**Alternatives Considered:**
- Separate TransformableContent interface (rejected: unnecessary abstraction)
- No interface changes, use type assertions (rejected: less clean)

---

## Decision 7: Support All Content Types

**Date:** 2025-01-19

**Context:** Should transformations be limited to TableContent or support all content types?

**Decision:** Design system to support all content types (Table, Text, Raw, Section), with initial implementation focused on table operations.

**Rationale:**
- Future-proofs the API for text transformations (trim, case conversion, etc.)
- Architectural consistency across content types
- Minimal additional complexity (operation `CanTransform()` already handles type checking)
- Operations naturally skip non-applicable content types

**Consequences:**
- API accepts transformations for all content types
- Current operations only work on tables (documented)
- Future operations can add text/raw/section support without API changes
- CanTransform() and skip-non-applicable logic required

**Alternatives Considered:**
- Table-only (rejected: future API breaking change needed)
- Separate transformation types per content type (rejected: complexity)

---

## Decision 8: Single Validation Phase During Rendering

**Date:** 2025-01-21

**Context:** Should validation occur at build-time, render-time, or both?

**Decision:** Validate transformations only during rendering when they are applied.

**Rationale:**
- Simpler mental model: Build() constructs, Render() validates and transforms
- Most validation errors require data context (column existence, type compatibility)
- Static validation adds complexity without proportional benefit
- Eliminates dual error handling paths
- All errors discovered at the point they matter

**Consequences:**
- Build() never fails due to transformations
- No Builder.HasErrors() or Builder.Errors() for transformations
- All validation happens inline during rendering
- Configuration and data errors have consistent handling
- Simpler Builder implementation

**Alternatives Considered:**
- Build-time static validation (rejected: adds complexity, limited value)
- Dual validation (rejected: two error paths to handle)

---

## Decision 9: Fail-Fast Error Handling Only

**Date:** 2025-01-21

**Context:** Should rendering support partial rendering mode or only fail-fast?

**Decision:** Always fail-fast - stop rendering immediately on first transformation error.

**Rationale:**
- Simpler implementation and API surface
- Partial rendering adds configuration complexity for unproven use case
- Fail-fast is correct default for data integrity
- Users needing resilience can catch errors themselves
- Most use cases want all-or-nothing rendering
- No evidence of need for partial rendering

**Consequences:**
- No FailureMode configuration needed
- No PartialRenderError type
- No skip tracking or error collection
- Clear, predictable behavior
- Less code to write and maintain

**Alternatives Considered:**
- Configurable failure modes (rejected: premature flexibility)
- Always partial (rejected: unsafe default)

---

## Decision 10: Lazy Execution During Rendering

**Date:** 2025-01-19

**Context:** When should transformations be applied - during Build() or during rendering?

**Decision:** Execute transformations lazily during rendering/output.

**Rationale:**
- Preserves original data in document (transformations don't mutate)
- Allows context-aware rendering with cancellation
- Defers expensive operations until actually needed
- Supports potential caching/memoization in future

**Consequences:**
- Build() is fast (no transformation execution)
- Rendering includes transformation time
- Transformation errors appear at render time
- Multiple renders may re-execute transformations (unless cached)

**Alternatives Considered:**
- Eager execution during Build() (rejected: slower builds, loses original data)
- Hybrid caching approach (deferred to implementation)

---

## Decision 11: Measurable Performance Requirements

**Date:** 2025-01-19

**Context:** Initial requirements had vague performance claims ("hundreds of items").

**Decision:** Specify concrete minimums: 100 content items with 10 transformations each.

**Rationale:**
- Testable and verifiable requirement
- Based on realistic document sizes (reports, dashboards)
- Provides performance budget for implementation
- Allows regression testing

**Consequences:**
- Implementation must meet specified minimums
- Performance testing required as part of acceptance
- Documented limit helps users understand scale

**Alternatives Considered:**
- No specific limits (rejected: untestable)
- Higher limits (1000+) (rejected: over-specification)
- Configurable limits (deferred to implementation)

---

## Questions Deferred to Design Phase

1. **Operation factory functions**: Should we provide builder patterns to avoid closure capture issues?
2. **Runtime stateless validation**: What specific checks should the debug mode perform?
3. **Context propagation**: How is context threaded through transformation chains?

---

## Design Phase Decisions

### Decision 12: Operation Factory Functions for Closure Safety

**Date:** 2025-01-19

**Context:** Go 1.21 has loop variable capture issue. Operations with closures can accidentally capture loop variables by reference.

**Decision:** Provide safe constructor patterns through factory functions with explicit parameters.

**Rationale:**
- Go 1.22+ fixed loop variable capture, but project uses Go 1.21
- Factory functions with explicit field parameters avoid capture issues
- Documented safe/unsafe patterns guide developers
- Works with existing operation constructors (NewFilterOp, NewSortOp)

**Implementation Example:**
```go
// SAFE: Factory with explicit parameters
func NewFilterByField(field string, predicate func(any) bool) *FilterOp {
    return &FilterOp{
        predicate: func(r Record) bool {
            return predicate(r[field])
        },
    }
}
```

**Consequences:**
- Developers have safe patterns to follow
- Documentation includes prominent warnings
- Compatible with existing API
- Future Go 1.22+ upgrade eliminates the issue

---

### Decision 13: Runtime Stateless Validation

**Date:** 2025-01-19

**Context:** Operations must be stateless for thread safety, but Go cannot enforce at compile time.

**Decision:** Provide testing utility `ValidateStatelessOperation()` for test/debug mode validation.

**Rationale:**
- No automatic runtime detection for pure functions in Go
- Testing utility allows developers to verify statelessness
- Fails tests if operation produces different results on repeated calls
- Follows Go's pragmatic testing approach (race detector, vet)

**Implementation:**
```go
func ValidateStatelessOperation(t *testing.T, op Operation, testContent Content) {
    // Apply twice, compare results with DeepEqual
}
```

**Consequences:**
- Developers must add validation to test suites
- Catches stateful operations during development
- No runtime overhead in production
- Documentation encourages testing utility use

---

### Decision 14: Context Propagation Design

**Date:** 2025-01-19
**Revised:** 2025-01-20 (removed context checks from hot loops)

**Context:** How to thread context through transformation chains for cancellation.

**Decision:** Pass context explicitly through operation chain, check before and after operations (not during).

**Rationale:**
- Follows Go blog guidelines: "pass Context explicitly as first parameter"
- Context checks before/after operations enable responsive cancellation
- Avoids performance overhead of checks in hot loops (comparators, predicates)
- Works with existing Operation.Apply(ctx, content) signature
- Supports per-renderer timeouts

**Implementation:**
```go
// Check once before operation
if err := ctx.Err(); err != nil {
    return nil, err
}

// Perform operation without context checks in hot loops
// NOTE: Context checks in comparators create massive overhead
// (10,000+ channel operations for sorting 10,000 records)
sort.SliceStable(records, func(i, j int) bool {
    return compare(records[i], records[j]) < 0
})
```

**Consequences:**
- Context passed from renderer → transformation chain → operations
- Cancellation detected between operations (responsive enough for typical workloads)
- No performance overhead in hot loops (comparators, predicates, filters)
- Compatible with existing context patterns
- No additional goroutine overhead

**Revision Notes:**
- Original design included checks before and after operations
- Simplified to single check before operation only
- Cancellation happens between operations in chain, providing sufficient responsiveness

---

## Design Revisions After Peer Review

**Date:** 2025-01-20

**Context:** Initial design document underwent peer review (design-critic + peer-review-validator) which identified 2 blocking issues and 3 major revisions needed.

### Revision 1: TransformableContent Interface (Blocking Issue)

**Problem:** Design referenced `Clone()` method on Content interface, but Content interface doesn't include Clone().

**Solution:** Created explicit `TransformableContent` interface that extends `Content`:

```go
type TransformableContent interface {
    Content
    Clone() Content
    GetTransformations() []Operation
}
```

**Impact:**
- Makes transformation contract explicit without breaking existing Content interface
- All content types (Table, Text, Raw, Section) implement TransformableContent
- Backward compatible: existing Content implementations unaffected

### Revision 2: Context Propagation Performance Fix (Blocking Issue)

**Problem:** Decision 12 showed context checks inside sort comparator functions, creating 10,000+ channel operations for sorting 10,000 records.

**Solution:** Move context checks to before/after operations only:

```go
// Check BEFORE operation
select {
case <-ctx.Done():
    return nil, ctx.Err()
default:
}

// Perform operation without context checks in hot loops
sort.SliceStable(records, func(i, j int) bool {
    return o.compareRecords(records[i], records[j]) < 0
})

// Check AFTER operation
select {
case <-ctx.Done():
    return nil, ctx.Err()
default:
}
```

**Impact:**
- Eliminates massive performance overhead in comparators and predicates
- Still responsive to cancellation (checks between operations)
- Updated Decision 12 in decision_log.md

### Revision 3: Expand Content Interface Instead of New Interface (Major Revision)

**Problem:** Design created TransformableContent interface, but this adds unnecessary abstraction.

**Solution:** Extend existing Content interface with Clone() and GetTransformations() methods.

**Impact:**
- Simpler architecture with single Content interface
- All content is transformable without type assertions
- Clone() formalized in interface (already existed in implementations)
- Documented in Decision 6

### Revision 4: Single Validation Phase (Major Revision)

**Problem:** Build-time and render-time validation created dual error handling paths.

**Solution:** Validate only during rendering when both configuration and data are available.

**Impact:**
- Simpler mental model
- No Builder.HasErrors() or Builder.Errors() needed
- All validation errors discovered at render time
- Documented in Decision 8

### Revision 5: Remove Partial Rendering Mode (Major Revision)

**Problem:** Partial rendering adds configuration complexity without proven use case.

**Solution:** Always fail-fast - stop immediately on first error.

**Impact:**
- No FailureMode configuration
- No PartialRenderError type
- Simpler renderer implementation
- Documented in Decision 9

### Revision 6: Simplify Context Checks (Minor Revision)

**Problem:** Checking context before and after operations is redundant.

**Solution:** Check context once before each operation only.

**Impact:**
- Simpler implementation
- Same responsiveness (checks between operations)
- Updated Decision 14

### Revision 7: Remove Go 1.21 Complexity (Major Revision)

**Problem:** Decision 9 created complex factory patterns to work around Go 1.21 loop variable capture bug, but project uses Go 1.24+ where this is fixed.

**Solution:** Simplified Decision 1 to basic closure best practices without Go 1.21 workarounds.

**Impact:**
- Removed unnecessary complexity from design
- Simplified API and developer experience
- Documentation focuses on clear intent, not outdated workarounds

### Revision 8: Stateless Validation Limitations (Major Revision)

**Problem:** Decision 10 claimed `ValidateStatelessOperation()` detects statelessness, but it only detects output determinism.

**Solution:** Added explicit limitations documentation:

```go
// NOTE: This checks if operations produce consistent results, but does NOT detect:
// - Hidden state mutations that don't affect output
// - External side effects (file writes, network calls)
// - Non-deterministic operations that happen to match twice
```

**Impact:**
- Sets correct expectations for testing utility
- Developers understand what is and isn't validated
- Updated Decision 3 in design.md

### Revision 9: Remove Debug Data Samples (Major Revision)

**Date:** 2025-01-21

**Problem:** Debug data samples with field redaction created security complexity without proportional value.

**Decision:** Remove debug data samples feature entirely.

**Rationale:**
- Security anti-pattern: error messages shouldn't include data
- Field name redaction gives false confidence
- Clear error messages with row/column context are sufficient
- Developers can examine input data separately if needed
- Eliminates environment variable configuration and global state

**Impact:**
- Simpler error handling implementation
- No GO_OUTPUT_DEBUG environment variable
- No sensitive field redaction logic
- Clearer separation: errors describe what failed, not data contents
- Less code, fewer security concerns

---

## Code Simplifier Review Decisions

**Date:** 2025-01-21

**Context:** code-simplifier agent reviewed the design and identified overcomplexity.

**Decisions Made:**
1. ✅ **Extend Content interface** - No separate TransformableContent interface (Decision 6)
2. ✅ **Single validation phase** - Only during rendering (Decision 8)
3. ✅ **Fail-fast only** - No partial rendering mode (Decision 9)
4. ✅ **Remove debug data samples** - Security concern without proportional value (Revision 9)
5. ✅ **Simplify context checks** - Single check before operations (Revision 6)
6. ⚠️ **Keep CanOptimize()** - For future optimization potential (retained)

**Not Implemented:**
- Removing FormatAwareOperation interface (deferred: may be needed)
- Removing ApplyWithFormat() method (deferred: cross-format consistency)

**Impact:**
- 30-40% reduction in design complexity
- Clearer mental model for developers
- Less code to write and maintain
- Same functionality for real-world use cases

---

### API Stability

All revisions maintain public API stability:
- Content interface extended with new methods (additive change)
- Context propagation changes are internal implementation details
- Error message changes are internal formatting, not API changes
- No breaking changes to planned transformation API

---

## Decision 12: Type-Specific Transformation Function Names

**Date:** 2025-01-23

**Context:** Design document specified a single `WithTransformations()` function name across all content types for consistency. However, implementation uses type-specific names: `WithTransformations()`, `WithTextTransformations()`, `WithRawTransformations()`, and `WithSectionTransformations()`.

**Decision:** Use type-specific function names for transformation options rather than a uniform `WithTransformations()` across all content types.

**Rationale:**
- Go does not support function overloading - all functions in a package must have unique names
- Functions like `WithTransformations()` must have distinct signatures or names to work with different content types
- Type-specific names provide better clarity about which content type is being configured
- Consistent with Go idioms (e.g., `strings.Builder`, `json.Decoder` - type-specific APIs)
- IDE autocomplete shows relevant options for each content type

**Implementation:**
```go
// Table content transformations
output.WithTransformations(ops...)

// Text content transformations
output.WithTextTransformations(ops...)

// Raw content transformations
output.WithRawTransformations(ops...)

// Section content transformations
output.WithSectionTransformations(ops...)
```

**Consequences:**
- Slightly more verbose than uniform naming
- Better type safety - compiler catches mismatched content/options
- Clearer documentation - each function documents its specific content type
- No ambiguity about which content type an option applies to

**Alternatives Considered:**
- Uniform `WithTransformations()` with type assertions (rejected: not possible in Go without overloading)
- Generic function with type parameters (rejected: excessive complexity for simple use case)
- Interface-based approach with type switches (rejected: less ergonomic)

---

## Open Questions for User Validation

None - all design questions resolved through research and decision-making process.
