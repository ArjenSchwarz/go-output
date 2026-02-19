# Bugfix Report: Mermaid Piechart Ignores Int Values

**Bug ID:** T-143  
**Date:** 2026-02-19  
**Severity:** Medium  
**Status:** Fixed  

## Problem Description

The Mermaid piechart functionality in `output.go` only handled `float64` values when converting data for chart generation. Integer values (`int`, `int64`, `int32`, etc.) and `float32` values were silently treated as 0, resulting in empty or incorrect pie charts when users passed integer counts.

### Expected Behavior
When users provide integer values for piechart data, these values should be properly converted to numeric format and displayed in the generated Mermaid piechart.

### Actual Behavior
- Integer values were silently treated as 0
- Pie charts appeared empty when all values were integers
- Pie charts showed incorrect proportions when mixing integers and floats
- No error messages were generated (silent failure)

### Symptoms
- Empty pie charts when using integer data
- Incorrect proportions in mixed-type scenarios
- Silent failure with no error indication

## Root Cause Analysis

### Technical Analysis
The issue was in the `toMermaid()` function in `output.go` (lines 691-695). The type switch only handled the `float64` case:

```go
var value float64
switch converted := holder.Contents[output.Settings.FromToColumns.To].(type) {
case float64:
    value = converted
}
```

When the value was any other numeric type, the switch fell through and `value` remained 0.

### Five Whys Analysis
1. **Why do integer values appear as 0 in piecharts?** → Because the type switch only handles `float64` type.
2. **Why does the type switch only handle float64?** → Because the developer only implemented the most common case.
3. **Why wasn't this caught during development?** → Because there were no tests covering integer input scenarios.
4. **Why is there inconsistency with other numeric handling?** → Because the piechart logic was implemented separately without reusing existing patterns.
5. **Why wasn't the existing pattern used?** → Because the piechart needs numeric values (not strings) but the conversion wasn't implemented comprehensively.

**Root Cause:** Incomplete type switch implementation that didn't handle all Go numeric types, unlike other parts of the codebase that correctly handle all numeric types.

## Solution

### Fix Applied
Replaced the incomplete type switch with a comprehensive one that handles all Go numeric types:

```go
var value float64
switch converted := holder.Contents[output.Settings.FromToColumns.To].(type) {
case int:
    value = float64(converted)
case int8:
    value = float64(converted)
case int16:
    value = float64(converted)
case int32:
    value = float64(converted)
case int64:
    value = float64(converted)
case uint:
    value = float64(converted)
case uint8:
    value = float64(converted)
case uint16:
    value = float64(converted)
case uint32:
    value = float64(converted)
case uint64:
    value = float64(converted)
case float32:
    value = float64(converted)
case float64:
    value = converted
}
```

### Why This Fixes the Issue
- Handles all Go numeric types consistently with the rest of the codebase
- Converts all numeric types to float64 as expected by the mermaid library
- Eliminates the silent failure where non-float64 values become 0
- Maintains backward compatibility (no breaking changes)

## Testing

### Regression Tests Added
Created comprehensive regression tests in `output_mermaid_piechart_test.go`:

1. **TestMermaidPiechartIntegerValues**: Tests individual numeric types
   - `int`, `int32`, `int64`, `float32`, `float64`
   - Verifies each type produces correct output

2. **TestMermaidPiechartMixedTypes**: Tests mixed numeric types in single chart
   - Combines `int`, `int64`, and `float64` values
   - Ensures all values appear correctly in output

### Test Results
- **Before fix**: All integer and float32 types showed as "0.00"
- **After fix**: All numeric types display correct values
- **Full test suite**: All existing tests continue to pass

### Manual Verification
Tested with the existing `test/test_fix.go` file:
- **Before**: Output showed all zeros
- **After**: Output correctly shows "Users: 42.00", "Items: 100.00", "Score: 75.50"

## Impact Assessment

### Positive Impact
- Fixes silent data loss for integer piechart values
- Improves consistency across the codebase
- Maintains backward compatibility
- No performance impact

### Risk Assessment
- **Low risk**: Minimal code change with comprehensive test coverage
- **No breaking changes**: Existing float64 usage continues to work
- **Potential precision loss**: Large int64/uint64 values may lose precision when converted to float64, but this is acceptable for chart display purposes

## Files Modified

1. **output.go**: Fixed type switch in `toMermaid()` piechart case (lines 691-695)
2. **output_mermaid_piechart_test.go**: Added comprehensive regression tests
3. **test/test_fix.go**: Fixed import and type references

## Verification Steps

1. ✅ Regression tests pass
2. ✅ Full test suite passes  
3. ✅ Manual testing with mixed numeric types works
4. ✅ Code formatting and linting clean
5. ✅ No breaking changes to existing functionality

## Future Prevention

- The regression tests will prevent this issue from recurring
- Consider adding integration tests for all chart types with various data types
- Document expected data types in API documentation

---

**Fixed by:** Claude (AI Assistant)  
**Reviewed by:** [Pending]  
**Deployed:** [Pending]