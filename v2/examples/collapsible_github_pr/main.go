// Package main demonstrates GitHub PR comment generation with collapsible error details
// This example shows how to use the collapsible content system to create GitHub PR comments
// that provide summary information with expandable details for code analysis results.
package main

import (
	"context"
	"fmt"
	"log"

	output "github.com/ArjenSchwarz/go-output/v2"
)

func main() {
	fmt.Println("🔍 GitHub PR Comment Generation with Collapsible Content")
	fmt.Println("========================================================")

	// Example 1: Code Analysis Results for PR Comment
	fmt.Println("\n📝 Example 1: Code Analysis Results")
	generateCodeAnalysisComment()

	// Example 2: Test Results with Collapsible Failures
	fmt.Println("\n🧪 Example 2: Test Results with Collapsible Failures")
	generateTestResultsComment()

	// Example 3: Security Scan Results
	fmt.Println("\n🔒 Example 3: Security Scan Results")
	generateSecurityScanComment()
}

func generateCodeAnalysisComment() {
	// Simulate code analysis results with various issues
	analysisData := []map[string]any{
		{
			"file":        "/src/components/UserProfile.tsx",
			"lines":       142,
			"errors":      []string{"Missing import for React", "Unused variable 'userData'", "Type annotation missing for 'props'"},
			"warnings":    []string{"Deprecated method usage", "Consider using useCallback for optimization"},
			"suggestions": []string{"Add TypeScript strict mode", "Consider memoization for expensive calculations"},
			"complexity":  8.5,
		},
		{
			"file":        "/src/utils/api-helpers.ts",
			"lines":       87,
			"errors":      []string{"Unhandled promise rejection"},
			"warnings":    []string{"Consider adding retry logic", "Missing error boundary"},
			"suggestions": []string{"Add request timeout", "Implement circuit breaker pattern"},
			"complexity":  6.2,
		},
		{
			"file":        "/src/hooks/useDataFetching.ts",
			"lines":       156,
			"errors":      []string{},
			"warnings":    []string{"Large hook, consider splitting", "Missing cleanup in useEffect"},
			"suggestions": []string{"Extract custom hooks", "Add AbortController for cleanup"},
			"complexity":  9.1,
		},
	}

	// Create document with collapsible error details
	doc := output.New().
		SetMetadata("title", "Code Analysis Results").
		Header("🔍 Code Analysis Report").
		Text("Analysis completed for **3 files** with mixed results.", output.WithBold(true)).
		Text("⚠️ **2 files** have errors requiring attention").
		Table("File Analysis Results", analysisData,
			output.WithSchema(
				output.Field{Name: "file", Type: "string", Formatter: output.FilePathFormatter(30)},
				output.Field{Name: "lines", Type: "int"},
				output.Field{Name: "errors", Type: "array", Formatter: output.ErrorListFormatter(output.WithCollapsibleExpanded(false))},
				output.Field{Name: "warnings", Type: "array", Formatter: output.ErrorListFormatter(output.WithCollapsibleExpanded(false))},
				output.Field{Name: "suggestions", Type: "array", Formatter: output.ErrorListFormatter(output.WithCollapsibleExpanded(false))},
				output.Field{Name: "complexity", Type: "float"},
			)).
		Text("**Next Steps:**").
		Text("1. Fix all error-level issues before merge").
		Text("2. Consider addressing high-complexity files (>9.0)").
		Text("3. Review suggestions for performance improvements").
		Build()

	// Render as Markdown for GitHub
	renderDocument(doc, output.Markdown, "github-pr-analysis.md")
}

func generateTestResultsComment() {
	// Simulate test results with detailed failure information
	testData := []map[string]any{
		{
			"suite":   "Authentication Tests",
			"passed":  12,
			"failed":  2,
			"skipped": 1,
			"failures": []string{
				"test_login_with_invalid_credentials: Expected 401, got 500",
				"test_token_refresh: Token refresh failed after 3 attempts",
			},
			"duration": "2.3s",
		},
		{
			"suite":    "API Integration Tests",
			"passed":   8,
			"failed":   0,
			"skipped":  0,
			"failures": []string{},
			"duration": "1.8s",
		},
		{
			"suite":   "UI Component Tests",
			"passed":  45,
			"failed":  1,
			"skipped": 3,
			"failures": []string{
				"test_modal_keyboard_navigation: Modal did not close on Escape key",
			},
			"duration": "5.2s",
		},
	}

	doc := output.New().
		Header("🧪 Test Results Summary").
		Text("Test suite completed with **1 suite** containing failures.", output.WithColor("red")).
		Table("Test Suite Results", testData,
			output.WithSchema(
				output.Field{Name: "suite", Type: "string"},
				output.Field{Name: "passed", Type: "int"},
				output.Field{Name: "failed", Type: "int"},
				output.Field{Name: "skipped", Type: "int"},
				output.Field{Name: "failures", Type: "array", Formatter: output.ErrorListFormatter(output.WithCollapsibleExpanded(false))},
				output.Field{Name: "duration", Type: "string"},
			)).
		Text("**Overall:** 65 passed, 3 failed, 4 skipped").
		Build()

	renderDocument(doc, output.Markdown, "github-pr-tests.md")
}

func generateSecurityScanComment() {
	// Simulate security scan results
	securityData := []map[string]any{
		{
			"category":    "Dependencies",
			"severity":    "High",
			"count":       2,
			"issues":      []string{"lodash@4.17.20: Prototype pollution vulnerability", "axios@0.21.0: Server-side request forgery"},
			"remediation": "Run 'npm audit fix' to update vulnerable dependencies",
		},
		{
			"category":    "Code Quality",
			"severity":    "Medium",
			"count":       5,
			"issues":      []string{"Hardcoded API key in config.js", "SQL query without parameterization", "Missing CSRF protection", "Weak password validation", "Unencrypted sensitive data storage"},
			"remediation": "Review code security practices and implement secure coding guidelines",
		},
		{
			"category":    "Infrastructure",
			"severity":    "Low",
			"count":       1,
			"issues":      []string{"HTTP instead of HTTPS for API endpoint"},
			"remediation": "Update API endpoint configuration to use HTTPS",
		},
	}

	doc := output.New().
		Header("🔒 Security Scan Results").
		Text("Security scan identified **8 issues** across 3 categories.", output.WithColor("orange")).
		Text("⚠️ **High severity** issues require immediate attention.").
		Table("Security Issues by Category", securityData,
			output.WithSchema(
				output.Field{Name: "category", Type: "string"},
				output.Field{Name: "severity", Type: "string"},
				output.Field{Name: "count", Type: "int"},
				output.Field{Name: "issues", Type: "array", Formatter: output.ErrorListFormatter(output.WithCollapsibleExpanded(false))},
				output.Field{Name: "remediation", Type: "string"},
			)).
		Text("**Priority Order:**").
		Text("1. **High**: Update vulnerable dependencies immediately").
		Text("2. **Medium**: Implement secure coding practices").
		Text("3. **Low**: Configure HTTPS endpoints").
		Build()

	renderDocument(doc, output.Markdown, "github-pr-security.md")
}

func renderDocument(doc *output.Document, format output.Format, filename string) {
	// Create output with GitHub-optimized markdown
	out := output.NewOutput(
		output.WithFormat(format),
		output.WithWriter(output.NewStdoutWriter()),
	)

	ctx := context.Background()
	err := out.Render(ctx, doc)
	if err != nil {
		log.Fatalf("Failed to render document: %v", err)
	}

	fmt.Printf("\n💾 Content saved as %s\n", filename)
	fmt.Println(strings.Repeat("-", 50))
}

// Helper function to simulate strings package
var strings = struct {
	Repeat func(string, int) string
}{
	Repeat: func(s string, count int) string {
		result := ""
		for range count {
			result += s
		}
		return result
	},
}
