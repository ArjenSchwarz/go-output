# Decision Log - Draw.io AWS Icons Feature

## Feature Overview
Implementing AWS icon support for Draw.io diagrams in v2 of go-output library.

## Key Decisions

### 1. Scope Limitation to AWS Icons Only
**Date:** 2025-09-21
**Decision:** Focus exclusively on AWS icons without building an extensible icon registry system
**Rationale:**
- User has not needed other icon sets in 5 years of using v1
- Avoids overengineering and unnecessary complexity
- Can be extended later if real requirements emerge
- Maintains simplicity and reduces maintenance burden

### 2. Simple Direct Port from v1
**Date:** 2025-09-21
**Decision:** Port the v1 implementation directly with minimal modifications
**Rationale:**
- v1 implementation is proven and works well (only 49 lines of code)
- User's existing code depends on this exact functionality
- No performance issues reported with v1 approach
- Simplicity is valued over hypothetical extensibility

### 3. Error Handling Approach
**Date:** 2025-09-21
**Decision:** Return errors when shapes are not found (not empty strings)
**Rationale:**
- User explicitly requested error returns
- Better for debugging and problem diagnosis
- More idiomatic Go pattern
- Aligns with v2's general error handling philosophy

### 4. Package Location
**Date:** 2025-09-21
**Decision:** Implement in new `v2/icons` package
**Rationale:**
- Clean separation of concerns
- Leaves room for future icon-related functionality if needed
- Easy to find and understand
- Follows v2's modular package structure

### 5. No Lazy Loading or Complex Caching
**Date:** 2025-09-21
**Decision:** Parse JSON once at package initialization
**Rationale:**
- AWS shapes JSON is only ~200KB
- 600 services is a small dataset for modern systems
- Startup time impact is negligible
- Simpler implementation with fewer edge cases
- v1 approach has proven sufficient

### 6. API Design
**Date:** 2025-09-21
**Decision:** Mirror v1's `GetAWSShape(group, title string) (string, error)` signature
**Rationale:**
- User explicitly wants GetAWSShape functionality mirrored
- Enables easier migration from v1
- Familiar API for existing users
- Clear and simple interface

### 7. Helper Functions Included
**Date:** 2025-09-21
**Decision:** Add helper functions for discovering available icons
**Rationale:**
- Useful for development and debugging
- Low implementation cost
- Improves developer experience
- Already partially exists in v1 (AllAWSShapes)

## Rejected Alternatives

### Extensible Icon Registry System
**Rejected because:**
- No proven demand from users
- Adds significant complexity without clear value
- YAGNI principle - can be added later if needed
- Would require more testing and maintenance

### Runtime Icon Loading
**Rejected because:**
- User is satisfied with compile-time embedding
- Adds complexity around file paths, URLs, error handling
- No use case presented for dynamic icon updates
- Security implications of loading external resources

### Backward Compatible Empty String Returns
**Rejected because:**
- User explicitly wants errors returned
- Error returns are more idiomatic Go
- Better for debugging when icons are missing
- v2 doesn't maintain backward compatibility anyway

## Implementation Notes

- Copy aws.json file from v1 to v2/icons/aws.json
- Use go:embed directive for zero-dependency inclusion
- Initialize map once at package startup
- Ensure thread-safety with read-only access after init
- Include comprehensive tests matching v1 test coverage

## Future Considerations

If icon extensibility is needed in the future:
- Can add an IconProvider interface without breaking existing API
- Can add registration functions alongside existing GetAWSShape
- Current simple implementation doesn't prevent future extension
- Wait for concrete requirements before adding complexity