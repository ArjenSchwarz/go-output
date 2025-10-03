// Package icons provides AWS service icon support for Draw.io diagrams.
//
// This package enables users to create Draw.io diagrams with proper AWS service icons
// by providing access to a comprehensive set of AWS shapes. The icons are embedded at
// compile time for zero-dependency usage.
//
// # Basic Usage
//
//	import "github.com/ArjenSchwarz/go-output/v2/icons"
//
//	// Get a specific AWS icon
//	style, err := icons.GetAWSShape("Compute", "EC2")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// List all available service groups
//	groups := icons.AllAWSGroups()
//
//	// List shapes in a specific group
//	shapes, err := icons.AWSShapesInGroup("Compute")
//
//	// Check if a shape exists
//	if icons.HasAWSShape("Compute", "EC2") {
//	    // Use the shape
//	}
//
// # Integration with Draw.io
//
// AWS icons can be used with Draw.io diagrams by assigning icon styles to records
// and using placeholders in the DrawIOHeader.Style field:
//
//	// Prepare data with service information
//	data := []map[string]any{
//	    {"Name": "API Gateway", "Type": "APIGateway", "Group": "Networking"},
//	    {"Name": "Lambda Function", "Type": "Lambda", "Group": "Compute"},
//	    {"Name": "DynamoDB", "Type": "DynamoDB", "Group": "Database"},
//	}
//
//	// Add AWS icon styles to each record
//	for _, record := range data {
//	    style, err := icons.GetAWSShape(record["Group"].(string), record["Type"].(string))
//	    if err == nil {
//	        record["IconStyle"] = style
//	    }
//	}
//
//	// Create Draw.io content with placeholder for dynamic icons
//	header := DrawIOHeader{
//	    Style: "%IconStyle%",  // Placeholder replaced per-record
//	    Label: "%Name%",
//	}
//
// # Migration from v1
//
// The API has changed from v1 to return errors explicitly:
//
//	// v1 usage
//	style := drawio.GetAWSShape("Compute", "EC2") // returns empty string on error
//
//	// v2 usage
//	style, err := icons.GetAWSShape("Compute", "EC2") // returns error
//	if err != nil {
//	    // handle error
//	}
package icons

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
)

//go:embed aws.json
var awsRaw []byte

var awsShapes map[string]map[string]string

// init performs one-time JSON parsing at package initialization.
// If the embedded aws.json is malformed, the program will panic at startup.
// This is intentional - embedded data corruption is a build issue, not a runtime issue.
func init() {
	if err := json.Unmarshal(awsRaw, &awsShapes); err != nil {
		panic(fmt.Sprintf("icons: failed to parse embedded aws.json: %v", err))
	}
	// Free the raw JSON bytes after successful parsing to save memory
	awsRaw = nil
}

// GetAWSShape returns the Draw.io style string for a specific AWS service icon.
//
// The function performs case-sensitive matching on both group and title parameters.
// Common groups include "Compute", "Storage", "Database", "Analytics", etc.
// Use AllAWSGroups() to discover available groups and AWSShapesInGroup() to
// discover shapes within a group.
//
// Parameters:
//   - group: The AWS service group (e.g., "Compute", "Storage", "Analytics")
//   - title: The specific service name (e.g., "EC2", "S3", "Kinesis")
//
// Returns:
//   - string: Draw.io compatible style string that can be used in DrawIOHeader.Style
//   - error: Non-nil if group or title not found
//
// Example:
//
//	style, err := GetAWSShape("Compute", "EC2")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// Use style in Draw.io diagram header
func GetAWSShape(group, title string) (string, error) {
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

// AllAWSGroups returns all available AWS service groups in alphabetical order.
//
// Use this function to discover what service categories are available.
// Common groups include Analytics, Compute, Database, Networking, Security, Storage, etc.
//
// Example:
//
//	for _, group := range icons.AllAWSGroups() {
//	    fmt.Println(group)
//	}
func AllAWSGroups() []string {
	groups := make([]string, 0, len(awsShapes))
	for group := range awsShapes {
		groups = append(groups, group)
	}
	slices.Sort(groups)
	return groups
}

// AWSShapesInGroup returns all shape titles in a specific group in alphabetical order.
//
// Returns an error if the group does not exist. Use AllAWSGroups() to discover
// available groups.
//
// Example:
//
//	shapes, err := icons.AWSShapesInGroup("Compute")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, shape := range shapes {
//	    fmt.Println(shape)
//	}
func AWSShapesInGroup(group string) ([]string, error) {
	groupMap, ok := awsShapes[group]
	if !ok {
		return nil, fmt.Errorf("shape group %q not found", group)
	}

	shapes := make([]string, 0, len(groupMap))
	for title := range groupMap {
		shapes = append(shapes, title)
	}
	slices.Sort(shapes)
	return shapes, nil
}

// HasAWSShape checks if a specific shape exists.
//
// This is a convenience function that returns true if the shape exists,
// false otherwise. It does not return an error.
//
// Example:
//
//	if icons.HasAWSShape("Compute", "EC2") {
//	    style, _ := icons.GetAWSShape("Compute", "EC2")
//	    // Use the style
//	}
func HasAWSShape(group, title string) bool {
	groupMap, ok := awsShapes[group]
	if !ok {
		return false
	}
	_, ok = groupMap[title]
	return ok
}
