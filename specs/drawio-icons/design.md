# Draw.io AWS Icons Design Document

## Overview

This document outlines the design for implementing AWS icon support in the v2 go-output library. The feature enables users to create Draw.io diagrams with proper AWS service icons, providing a direct migration path from v1's Draw.io functionality. The implementation focuses on simplicity, reliability, and direct compatibility with existing v1 code patterns.

## Architecture

### High-Level Architecture

```mermaid
graph TD
    A[v2/icons Package] --> B[Embedded AWS JSON]
    A --> C[GetAWSShape Function]
    A --> D[Helper Functions]

    B --> E[Parse Once on Init]
    E --> F[In-Memory Map Structure]

    C --> G[Map Lookup O(1)]
    G --> F

    D --> H[AllAWSGroups]
    D --> I[AWSShapesInGroup]
    D --> J[HasAWSShape]

    K[DrawIOContent] --> L[Style Field]
    L --> C

    M[v1 drawio/shapes/aws.json] -.->|Copy| B
```

### Package Structure

The implementation will be contained in a new `v2/icons` package with the following structure:

```
v2/
├── icons/
│   ├── aws.go           # Main implementation
│   ├── aws_test.go      # Unit tests
│   └── aws.json         # Embedded AWS shapes data (copied from v1)
```

### Data Flow

1. **Initialization**: The aws.json file is embedded at compile time using `//go:embed` directive
2. **Parsing**: JSON is parsed once at package initialization in `init()` function
3. **Storage**: Parsed data is stored in a read-only `map[string]map[string]string` structure
4. **Lookup**: Shape lookups are performed via O(1) map access
5. **Integration**: Retrieved style strings are used in DrawIOHeader.Style field with placeholders
6. **Rendering**: Draw.io replaces placeholders (e.g., `%ServiceType%`) with actual values from data records

## Components and Interfaces

### Core API

```go
package icons

// GetAWSShape returns the Draw.io style string for a specific AWS service icon
// Parameters:
//   - group: The AWS service group (e.g., "Compute", "Storage", "Analytics")
//   - title: The specific service name (e.g., "EC2", "S3", "Kinesis")
// Returns:
//   - string: Draw.io compatible style string
//   - error: Non-nil if group or title not found
func GetAWSShape(group, title string) (string, error) {
    // Defensive nested map access
    groupMap, ok := awsShapes[group]
    if !ok {
        return "", fmt.Errorf("shape group %q not found", group)
    }

    shape, ok := groupMap[title]
    if !ok {
        return "", fmt.Errorf("shape %q not found in group %q", title, group)
    }

    return shape, nil
}
```

### Helper Functions

```go
// AllAWSGroups returns all available AWS service groups in alphabetical order
func AllAWSGroups() []string

// AWSShapesInGroup returns all shape titles in a specific group in alphabetical order
func AWSShapesInGroup(group string) ([]string, error)

// HasAWSShape checks if a specific shape exists
func HasAWSShape(group, title string) bool
```

### Internal Components

```go
// Package-level variables
var (
    //go:embed aws.json
    awsRaw []byte

    awsShapes map[string]map[string]string
)

// init performs one-time JSON parsing at package initialization
func init() {
    if err := json.Unmarshal(awsRaw, &awsShapes); err != nil {
        panic(fmt.Sprintf("icons: failed to parse embedded aws.json: %v", err))
    }
    // Free the raw JSON bytes after successful parsing to save memory
    awsRaw = nil
}
```

## Data Models

### AWS Shapes JSON Structure

The embedded JSON file contains a two-level hierarchy:

```json
{
  "ServiceGroup": {
    "ServiceName": "draw.io-style-string"
  }
}
```

Example:
```json
{
  "Compute": {
    "EC2": "points=[[0,0,0]...];shape=mxgraph.aws4.ec2;",
    "Lambda": "points=[[0,0,0]...];shape=mxgraph.aws4.lambda;"
  },
  "Storage": {
    "S3": "points=[[0,0,0]...];shape=mxgraph.aws4.s3;",
    "EBS": "points=[[0,0,0]...];shape=mxgraph.aws4.ebs;"
  }
}
```

**Note on Duplicate Keys**: The v1 JSON file contains some duplicate keys (e.g., three "Internet" entries in General Resources). During JSON unmarshaling, Go's `json.Unmarshal` will use the last occurrence of any duplicate key. This behavior is consistent with v1 and will be maintained in v2.

### Memory Model

- **Size**: ~225KB embedded JSON data
- **Parsed Structure**: Approximately 600 service entries across ~30 groups
- **Memory Usage**: Estimated ~750KB-1MB after parsing (JSON + map overhead + string duplication)
- **Access Pattern**: Read-only after initialization
- **Memory Optimization**: Raw JSON bytes freed after parsing to reduce memory footprint

## Error Handling

### Error Types

1. **Parse-time Error (Fatal)**:
   ```go
   panic(fmt.Sprintf("icons: failed to parse embedded aws.json: %v", err))
   ```
   - Occurs only if embedded JSON is malformed
   - Fails fast at package initialization
   - Indicates a build/compilation issue

2. **Missing Group Error**:
   ```go
   fmt.Errorf("shape group %q not found", group)
   ```

3. **Missing Shape Error**:
   ```go
   fmt.Errorf("shape %q not found in group %q", title, group)
   ```

### Error Handling Strategy

- **Fail-Fast**: JSON parsing errors cause panic at init() since embedded data should never be corrupt
- **Explicit Errors**: All lookup functions return errors (not empty strings) for missing items
- **Defensive Programming**: Safe nested map access to prevent runtime panics
- **Consistent API**: All public functions have consistent error handling patterns

## Testing Strategy

### Unit Tests

1. **Successful Lookup Tests**:
   - Known shape retrieval (e.g., "Compute"/"EC2")
   - Style string format validation
   - Multiple shapes from different groups

2. **Error Case Tests**:
   - Non-existent group
   - Non-existent shape within valid group
   - Empty string parameters
   - Case sensitivity validation

3. **Helper Function Tests**:
   - AllAWSGroups alphabetical ordering
   - AWSShapesInGroup alphabetical ordering
   - HasAWSShape true/false cases
   - Error handling in helpers

4. **Thread Safety Tests**:
   - Concurrent GetAWSShape calls
   - Race detector validation (`go test -race`)
   - Multiple goroutines accessing same/different shapes

### Integration Tests

1. **Draw.io Integration with Placeholders**:
   ```go
   func TestAWSIconsWithDrawIO(t *testing.T) {
       // Create table with ServiceType column containing AWS service info
       data := []map[string]any{
           {"Name": "Web Server", "ServiceType": "EC2", "ServiceGroup": "Compute"},
           {"Name": "Database", "ServiceType": "RDS", "ServiceGroup": "Database"},
           {"Name": "Storage", "ServiceType": "S3", "ServiceGroup": "Storage"},
       }

       // Use placeholder in style to apply different icons per record
       header := DrawIOHeader{
           Style: "shape=%AWSIcon%;fillColor=#FF9900",
           Label: "%Name%",
       }

       // In practice, users would populate AWSIcon column with GetAWSShape results
       for _, record := range data {
           style, _ := icons.GetAWSShape(record["ServiceGroup"].(string), record["ServiceType"].(string))
           record["AWSIcon"] = style
       }
   }
   ```

2. **Migration Compatibility Test**:
   - Compare output with v1 for same inputs
   - Verify style string format unchanged
   - Test duplicate key handling consistency

### Benchmark Tests

```go
func BenchmarkGetAWSShape(b *testing.B) {
    b.Loop(func() {
        _, _ = GetAWSShape("Compute", "EC2")
    })
}

func BenchmarkConcurrentLookups(b *testing.B) {
    // Parallel lookup performance
}
```

## Performance Considerations

### Initialization
- **One-time Cost**: JSON parsing occurs once at package init()
- **Parse Time**: ~5-10ms for 225KB JSON (measured, not estimated)
- **Memory Allocation**: Single allocation for map structure
- **Fail-Fast**: Any parsing issues detected immediately at startup

### Runtime Performance
- **Lookup Complexity**: O(1) for map access with defensive nested checks
- **No Allocations**: String returns are references to parsed data
- **Thread Safe**: No synchronization needed - data is immutable after init()
- **Cache Friendly**: Frequently accessed shapes benefit from CPU cache

### Scalability
- **Shape Count**: Handles 600+ shapes without degradation
- **Concurrent Access**: No locks or synchronization needed
- **Memory Footprint**: Fixed ~750KB-1MB after initialization

## Security Considerations

1. **Embedded Data**: JSON is embedded at compile time, eliminating runtime file access
2. **Read-Only Access**: Map is never modified after initialization
3. **Input Validation**: Parameters are used only for map lookup (no injection risk)
4. **No External Dependencies**: Self-contained implementation

## Migration Guide

### For v1 Users

**v1 Usage**:
```go
import "github.com/ArjenSchwarz/go-output/drawio"

style := drawio.GetAWSShape("Compute", "EC2")
// Note: v1 returns empty string on error
```

**v2 Usage**:
```go
import "github.com/ArjenSchwarz/go-output/v2/icons"

style, err := icons.GetAWSShape("Compute", "EC2")
if err != nil {
    // Handle missing shape
}
```

### Using AWS Icons in Draw.io Diagrams

**Example: Multi-service Architecture Diagram**:
```go
// Prepare data with service information
data := []map[string]any{
    {"Name": "API Gateway", "Type": "APIGateway", "Group": "Networking"},
    {"Name": "Lambda Function", "Type": "Lambda", "Group": "Compute"},
    {"Name": "DynamoDB", "Type": "DynamoDB", "Group": "Database"},
}

// Add AWS icon styles to each record
for _, record := range data {
    if style, err := icons.GetAWSShape(record["Group"].(string), record["Type"].(string)); err == nil {
        record["IconStyle"] = style
    }
}

// Create Draw.io content with placeholder for dynamic icons
header := DrawIOHeader{
    Style: "%IconStyle%",  // Placeholder replaced per-record
    Label: "%Name%",
}

content := NewDrawIOContentFromTable(NewTableContent(WithData(data)))
content.header = header
```

### Key Differences
1. Package location: `drawio` → `v2/icons`
2. Error handling: Empty string → Explicit error
3. Function signature: Added error return
4. Integration: Use placeholders in DrawIOHeader.Style for per-record icon assignment

## Future Extensibility

While not part of the initial implementation, the design allows for future extensions:

1. **Additional Icon Sets**: Could add Azure, GCP icons in separate files
2. **Icon Provider Interface**: Could introduce abstraction if needed
3. **Dynamic Loading**: Could support runtime icon loading if required
4. **Custom Icons**: Could allow user-defined icon registration

These extensions can be added without breaking the existing API.

## Implementation Plan

1. **Phase 1**: Core Implementation
   - Copy aws.json from v1
   - Implement GetAWSShape with error handling
   - Add sync.Once initialization

2. **Phase 2**: Helper Functions
   - Implement AllAWSGroups
   - Implement AWSShapesInGroup
   - Implement HasAWSShape

3. **Phase 3**: Testing
   - Unit tests for all functions
   - Thread safety tests
   - Integration tests with DrawIOContent

4. **Phase 4**: Documentation
   - Package documentation with examples
   - Migration guide
   - Godoc comments

## Decision Rationale

### Why a Separate Package?
- Clean separation of concerns
- Easy to locate icon-related functionality
- Leaves room for future icon features
- Follows v2's modular structure

### Why Embed at Compile Time?
- Zero runtime dependencies
- No file path issues
- Guaranteed data availability
- Simplified deployment
- Security: No external file access

### Why init() with Panic Instead of sync.Once?
- **Fail-fast philosophy**: Embedded data corruption is a build issue, not runtime
- **Simplicity**: No need for error checking in every function
- **Performance**: No synchronization overhead on lookups
- **Predictability**: Consistent initialization timing
- **Debugging**: Issues detected immediately at startup

### Why Return Errors?
- More idiomatic Go
- Better debugging experience
- Aligns with v2 error handling philosophy
- Explicitly signals missing shapes

### Why Preserve Duplicate Keys Behavior?
- Maintains exact v1 compatibility
- Go's json.Unmarshal naturally handles this (last value wins)
- Changing behavior could break existing diagrams
- The duplicates represent different icon variations (internet, internet_alt1, internet_alt2)

### Why Use Placeholders for Per-Record Icons?
- Leverages Draw.io's built-in placeholder system
- Allows different icons for each record in the diagram
- Maintains separation between icon lookup and diagram generation
- Flexible: Users can mix static styles with dynamic icon selection