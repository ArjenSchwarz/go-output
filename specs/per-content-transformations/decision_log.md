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

## Decision 6: Support All Content Types

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

## Decision 7: Lazy Execution During Rendering

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

## Decision 8: Measurable Performance Requirements

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
3. **Partial rendering API**: How should partial mode be configured (per-document? per-renderer?)?
4. **Context propagation**: How is context threaded through transformation chains?
5. **Error message security**: What heuristics determine when to include data samples?

---

## Design Phase Decisions

### Decision 9: Operation Factory Functions for Closure Safety

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

### Decision 10: Runtime Stateless Validation

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

### Decision 11: Partial Rendering API Configuration

**Date:** 2025-01-19

**Context:** Need to specify where partial rendering mode is configured.

**Decision:** Configure failure mode at renderer level via functional options, not per-document.

**Rationale:**
- Matches Go idioms (http.Server, sql.DB configure at creation)
- Single configuration point, clear behavior per render call
- Allows different renderers with different modes for same document
- Aligns with Go resilience library patterns

**Implementation:**
```go
renderer := NewJSONRenderer(
    WithFailureMode(PartialRender),
)
```

**Consequences:**
- Renderer creation includes configuration
- Clear separation: document has data, renderer has behavior
- Can render same document with different failure modes

**Alternatives Considered:**
- Per-document configuration (rejected: document shouldn't know rendering behavior)
- Per-transformation configuration (rejected: too fine-grained)

---

### Decision 12: Context Propagation Design

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
// Check BEFORE operation
select {
case <-ctx.Done():
    return nil, ctx.Err()
default:
}

// Perform operation without context checks in hot loops
// NOTE: Context checks in comparators create massive overhead
// (10,000+ channel operations for sorting 10,000 records)
sort.SliceStable(records, func(i, j int) bool {
    return compare(records[i], records[j]) < 0
})

// Check AFTER operation
select {
case <-ctx.Done():
    return nil, ctx.Err()
default:
}
```

**Consequences:**
- Context passed from renderer → transformation chain → operations
- Cancellation detected between operations (responsive enough for typical workloads)
- No performance overhead in hot loops (comparators, predicates, filters)
- Compatible with existing context patterns
- No additional goroutine overhead

**Revision Notes:**
- Original design included periodic checks during long operations
- Peer review identified this creates severe performance overhead
- Revised to check only before/after operations, not during execution

---

### Decision 13: Error Message Security Heuristics

**Date:** 2025-01-19

**Context:** Need heuristics for when to include data samples in errors without exposing PII.

**Decision:** Conservative approach - no data samples by default, opt-in via environment variable with automatic redaction.

**Rationale:**
- OWASP guidelines: never log PII (health, government IDs, financial data)
- Secure by default principle
- Debug mode with `GO_OUTPUT_DEBUG=true` for troubleshooting
- Automatic redaction of known sensitive patterns (password, token, ssn, etc.)

**Implementation:**
```go
// Default: no data samples
if os.Getenv("GO_OUTPUT_DEBUG") == "true" {
    sanitized := redactSensitiveFields(record)
    return fmt.Errorf("%w (sample: %v)", err, sanitized)
}
```

**Consequences:**
- Production: no data exposure risk
- Development: enable debugging with environment variable
- Automatic protection against common PII patterns
- Can be extended with custom sensitive field lists

**Alternatives Considered:**
- Always include samples (rejected: security risk)
- Per-operation opt-in (rejected: too verbose)
- Regex-based detection (deferred: simpler field name matching sufficient)

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

### Revision 3: Remove Go 1.21 Complexity (Major Revision)

**Problem:** Decision 9 created complex factory patterns to work around Go 1.21 loop variable capture bug, but project uses Go 1.24+ where this is fixed.

**Solution:** Simplified Decision 1 to basic closure best practices without Go 1.21 workarounds.

**Impact:**
- Removed unnecessary complexity from design
- Simplified API and developer experience
- Documentation focuses on clear intent, not outdated workarounds

### Revision 4: Stateless Validation Limitations (Major Revision)

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
- Updated Decision 2 in design.md

### Revision 5: Reframe Error Message Security (Major Revision)

**Problem:** Decision 13 positioned field name redaction as a security feature, but it's not comprehensive PII protection.

**Solution:** Reframed Decision 5 as "Debug Hints in Error Messages" with clear limitations:

- **Primary goal**: Development debugging aid
- **Not a security feature**: Field name matching cannot reliably detect all PII
- **Production safety**: Disabled by default
- **Documentation emphasis**: This is convenience tooling, not a security boundary

**Impact:**
- Honest about capabilities and limitations
- Developers understand this helps debugging, not production security
- Production systems should never enable GO_OUTPUT_DEBUG regardless of redaction

### API Stability

All revisions maintain public API stability:
- New TransformableContent interface is additive (Content interface unchanged)
- Context propagation changes are internal implementation details
- Error message changes are internal formatting, not API changes
- No breaking changes to planned transformation API

---

## Open Questions for User Validation

None - all design questions resolved through research and decision-making process.
