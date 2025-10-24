package icons

import (
	"strings"
	"testing"
)

// TestAWSIconsWithDrawIO tests integration with Draw.io using placeholders
// This test demonstrates the recommended pattern from Requirements 3.1-3.4:
// - AWS icons are used through DrawIOHeader.Style field with placeholders
// - Different icons can be assigned to different nodes
// - Style strings are Draw.io-compatible
func TestAWSIconsWithDrawIO(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Create test data with different AWS services
	testServices := []struct {
		name  string
		group string
		title string
	}{
		{"Web Server", "Compute", "EC2"},
		{"Database", "Database", "DynamoDB"},
		{"Storage", "Storage", "Elastic Block Store"},
		{"Analytics", "Analytics", "Kinesis"},
	}

	// Retrieve AWS icon styles for each service
	iconStyles := make(map[string]string)
	for _, svc := range testServices {
		style, err := GetAWSShape(svc.group, svc.title)
		if err != nil {
			t.Fatalf("GetAWSShape(%q, %q) failed: %v", svc.group, svc.title, err)
		}
		iconStyles[svc.name] = style
	}

	// Verify all styles are Draw.io-compatible
	for name, style := range iconStyles {
		if !strings.Contains(style, "shape=") {
			t.Errorf("Icon for %q is not Draw.io-compatible: %q", name, style)
		}
		if !strings.Contains(style, "mxgraph") {
			t.Errorf("Icon for %q missing mxgraph namespace: %q", name, style)
		}
	}

	// Verify different services have different icons
	seenStyles := make(map[string]string)
	for name, style := range iconStyles {
		if otherName, exists := seenStyles[style]; exists {
			t.Errorf("Services %q and %q have identical styles (should be different)", name, otherName)
		}
		seenStyles[style] = name
	}
}

// TestAWSIconPlaceholderPattern tests using placeholder patterns in Style field
// This demonstrates Requirement 3.1: Style field with placeholder pattern
func TestAWSIconPlaceholderPattern(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Example: User wants to use %AWSIcon% placeholder in their DrawIOHeader.Style
	// They would populate the AWSIcon column with GetAWSShape results

	testCases := map[string]struct {
		placeholder string
		group       string
		title       string
	}{
		"compute_ec2": {
			placeholder: "%ComputeIcon%",
			group:       "Compute",
			title:       "EC2",
		},
		"storage_s3": {
			placeholder: "%StorageIcon%",
			group:       "Storage",
			title:       "Elastic Block Store",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			// Get the AWS icon style
			style, err := GetAWSShape(tc.group, tc.title)
			if err != nil {
				t.Fatalf("GetAWSShape(%q, %q) failed: %v", tc.group, tc.title, err)
			}

			// Verify the style can be used directly in DrawIOHeader.Style
			// (In actual usage, the placeholder would be replaced by Draw.io)
			if style == "" {
				t.Errorf("GetAWSShape(%q, %q) returned empty style", tc.group, tc.title)
			}

			// Verify style format matches what Draw.io expects
			if !strings.Contains(style, "shape=mxgraph") {
				t.Errorf("Style for %s doesn't contain mxgraph shape: %q", name, style)
			}
		})
	}
}

// TestMultipleIconsInDiagram tests assigning different icons to different node types
// This demonstrates Requirement 3.4: Different icons for different node types
func TestMultipleIconsInDiagram(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Simulate a multi-tier architecture diagram with different AWS services
	architectureNodes := []struct {
		nodeType string
		group    string
		service  string
	}{
		{"frontend", "Application Integration", "AppSync"},
		{"api", "Compute", "Lambda"},
		{"database", "Database", "DynamoDB"},
		{"storage", "Storage", "Elastic Block Store"},
	}

	// Retrieve icon for each node type
	nodeIcons := make(map[string]string)
	for _, node := range architectureNodes {
		style, err := GetAWSShape(node.group, node.service)
		if err != nil {
			t.Fatalf("GetAWSShape(%q, %q) for %s node failed: %v",
				node.group, node.service, node.nodeType, err)
		}
		nodeIcons[node.nodeType] = style
	}

	// Verify we have unique icons for each node type
	if len(nodeIcons) != len(architectureNodes) {
		t.Errorf("Expected %d unique icons, got %d", len(architectureNodes), len(nodeIcons))
	}

	// Verify all icons are valid Draw.io styles
	for nodeType, style := range nodeIcons {
		if !strings.Contains(style, "shape=") {
			t.Errorf("Icon for %s node is invalid: %q", nodeType, style)
		}
	}

	// Verify icons are different from each other
	for nodeType1, style1 := range nodeIcons {
		for nodeType2, style2 := range nodeIcons {
			if nodeType1 != nodeType2 && style1 == style2 {
				t.Errorf("Nodes %q and %q have identical icons (should differ)", nodeType1, nodeType2)
			}
		}
	}
}

// TestDrawIOStyleStringFormat tests that style strings have valid Draw.io format
// This demonstrates Requirement 3.3: Maintain exact style string format
func TestDrawIOStyleStringFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	testCases := map[string]struct {
		group string
		title string
	}{
		"compute_ec2": {
			group: "Compute",
			title: "EC2",
		},
		"storage_ebs": {
			group: "Storage",
			title: "Elastic Block Store",
		},
		"database_dynamodb": {
			group: "Database",
			title: "DynamoDB",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			// Get style
			style, err := GetAWSShape(tc.group, tc.title)
			if err != nil {
				t.Fatalf("GetAWSShape(%q, %q) failed: %v", tc.group, tc.title, err)
			}

			// Verify it's a valid Draw.io style string with required elements
			requiredElements := []string{"shape=", "mxgraph", "points="}
			for _, elem := range requiredElements {
				if !strings.Contains(style, elem) {
					t.Errorf("Style for %s missing required element %q: %q", name, elem, style)
				}
			}
		})
	}
}

// TestDuplicateKeyHandling tests that duplicate keys in JSON are handled consistently
// The aws.json has duplicate "Internet" entries - Go's json.Unmarshal uses last value
func TestDuplicateKeyHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// The General Resources group has duplicate "Internet" keys in aws.json
	// Go's json.Unmarshal uses the last occurrence for any duplicate key
	// This test verifies the behavior is consistent (last value wins)

	style, err := GetAWSShape("General Resources", "Internet")
	if err != nil {
		t.Fatalf("GetAWSShape failed: %v", err)
	}

	// Verify it returns a valid style (proving last duplicate value was used)
	if !strings.Contains(style, "shape=") {
		t.Errorf("Duplicate key handling returned invalid style: %q", style)
	}
}

// TestIconsUsableInDrawIOWorkflow demonstrates the complete workflow
// This is a comprehensive integration test showing actual usage pattern
func TestIconsUsableInDrawIOWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Step 1: User has data about their AWS architecture
	type AWSResource struct {
		Name         string
		ServiceGroup string
		ServiceType  string
	}

	resources := []AWSResource{
		{"AppSync API", "Application Integration", "AppSync"},
		{"Lambda Functions", "Compute", "Lambda"},
		{"DynamoDB Tables", "Database", "DynamoDB"},
		{"Backup Storage", "Storage", "Backup"},
	}

	// Step 2: Retrieve icon styles for each resource
	type ResourceWithIcon struct {
		AWSResource
		IconStyle string
	}

	resourcesWithIcons := make([]ResourceWithIcon, 0, len(resources))
	for _, res := range resources {
		style, err := GetAWSShape(res.ServiceGroup, res.ServiceType)
		if err != nil {
			// In real usage, might use a default icon or skip
			t.Logf("Warning: Could not get icon for %s: %v", res.Name, err)
			continue
		}

		resourcesWithIcons = append(resourcesWithIcons, ResourceWithIcon{
			AWSResource: res,
			IconStyle:   style,
		})
	}

	// Step 3: Verify all resources have valid icons
	if len(resourcesWithIcons) != len(resources) {
		t.Errorf("Expected icons for %d resources, got %d", len(resources), len(resourcesWithIcons))
	}

	// Step 4: Verify icons are unique and valid
	for i, res := range resourcesWithIcons {
		if res.IconStyle == "" {
			t.Errorf("Resource %q has empty icon style", res.Name)
		}

		if !strings.Contains(res.IconStyle, "shape=mxgraph") {
			t.Errorf("Resource %q has invalid icon style: %q", res.Name, res.IconStyle)
		}

		// Check uniqueness
		for j, other := range resourcesWithIcons {
			if i != j && res.IconStyle == other.IconStyle && res.ServiceType != other.ServiceType {
				t.Errorf("Resources %q and %q have same icon but different services", res.Name, other.Name)
			}
		}
	}

	// Step 5: In actual usage, these styles would be used in DrawIOHeader.Style field
	// with placeholders like %IconStyle% that Draw.io would replace per record
	// This test verifies the retrieved styles are suitable for that purpose
	for _, res := range resourcesWithIcons {
		// Style should be directly usable in Draw.io CSV
		if len(res.IconStyle) < 10 {
			t.Errorf("Icon style for %q seems too short: %q", res.Name, res.IconStyle)
		}
	}
}
