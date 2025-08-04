// Package main demonstrates how to use code fence wrapping with collapsible fields
// This example shows various ways to display code snippets, configurations, and
// structured data within collapsible sections with proper syntax highlighting.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	output "github.com/ArjenSchwarz/go-output/v2"
)

func main() {
	fmt.Println("📦 Collapsible Fields with Code Fences Example")
	fmt.Println("=============================================")

	// Example 1: Code Review Results
	fmt.Println("\n🔍 Example 1: Code Review Results")
	generateCodeReviewReport()

	// Example 2: Configuration Files
	fmt.Println("\n⚙️  Example 2: Configuration Files Display")
	generateConfigurationReport()

	// Example 3: API Response Examples
	fmt.Println("\n🌐 Example 3: API Response Examples")
	generateAPIExamplesReport()

	// Example 4: Error Logs with Stack Traces
	fmt.Println("\n🐛 Example 4: Error Logs with Stack Traces")
	generateErrorLogsReport()
}

func generateCodeReviewReport() {
	// Create custom formatter for code snippets
	codeSnippetFormatter := func(val any) any {
		if snippet, ok := val.(string); ok {
			return output.NewCollapsibleValue("View Code", snippet, output.WithCodeFences("go"))
		}
		return val
	}

	// Create custom formatter for suggestions
	suggestionFormatter := func(val any) any {
		if suggestions, ok := val.([]string); ok {
			return output.NewCollapsibleValue(fmt.Sprintf("%d suggestions", len(suggestions)),
				suggestions, output.WithExpanded(false))
		}
		return val
	}

	data := []map[string]any{
		{
			"file":     "auth/login.go",
			"line":     42,
			"issue":    "Potential SQL injection vulnerability",
			"severity": "High",
			"code": `func authenticateUser(username, password string) bool {
    query := fmt.Sprintf("SELECT * FROM users WHERE username='%s'", username)
    // This is vulnerable to SQL injection!
    rows, err := db.Query(query)
    // ... rest of function
}`,
			"suggestions": []string{
				"Use parameterized queries with placeholders",
				"Consider using an ORM like GORM",
				"Add input validation before query construction",
			},
		},
		{
			"file":     "handlers/api.go",
			"line":     128,
			"issue":    "Missing error handling",
			"severity": "Medium",
			"code": `resp, _ := http.Get(apiURL)
defer resp.Body.Close()
// Missing nil check for resp`,
			"suggestions": []string{
				"Always check errors from http.Get",
				"Check if resp is nil before using",
			},
		},
	}

	table, err := output.NewTableContent("Code Review Findings", data,
		output.WithSchema(
			output.Field{Name: "file", Type: "string"},
			output.Field{Name: "line", Type: "int"},
			output.Field{Name: "issue", Type: "string"},
			output.Field{Name: "severity", Type: "string"},
			output.Field{Name: "code", Type: "string", Formatter: codeSnippetFormatter},
			output.Field{Name: "suggestions", Type: "array", Formatter: suggestionFormatter},
		))
	if err != nil {
		log.Fatal(err)
	}

	// Render as both HTML and Markdown
	renderInBothFormats("code_review", table)
}

func generateConfigurationReport() {
	// Custom formatter for config content with appropriate language
	configFormatter := func(format string) func(any) any {
		return func(val any) any {
			if config, ok := val.(string); ok {
				return output.NewCollapsibleValue("View Configuration", config,
					output.WithCodeFences(format), output.WithExpanded(true))
			}
			return val
		}
	}

	data := []map[string]any{
		{
			"service": "nginx",
			"format":  "nginx",
			"config": `server {
    listen 80;
    server_name example.com;
    
    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
    
    location /api {
        proxy_pass http://localhost:8080;
        proxy_read_timeout 30s;
    }
}`,
		},
		{
			"service": "database",
			"format":  "yaml",
			"config": `database:
  host: localhost
  port: 5432
  name: production_db
  pool:
    max_connections: 100
    idle_timeout: 30s
  ssl:
    enabled: true
    cert_path: /etc/ssl/certs/db.crt`,
		},
		{
			"service": "application",
			"format":  "json",
			"config": `{
  "app": {
    "name": "MyAPI",
    "version": "2.1.0",
    "environment": "production",
    "features": {
      "auth": true,
      "rateLimit": {
        "enabled": true,
        "requests": 100,
        "window": "1m"
      }
    }
  }
}`,
		},
	}

	// Create table with dynamic formatters
	fields := []output.Field{
		{Name: "service", Type: "string"},
		{Name: "format", Type: "string"},
	}

	// Add config field with appropriate formatter based on format
	for _, row := range data {
		format := row["format"].(string)
		fields = append(fields[:2], output.Field{
			Name:      "config",
			Type:      "string",
			Formatter: configFormatter(format),
		})
		break // Just need to set it once
	}

	table, err := output.NewTableContent("Service Configurations", data, output.WithSchema(fields...))
	if err != nil {
		log.Fatal(err)
	}

	renderInBothFormats("configurations", table)
}

func generateAPIExamplesReport() {
	// Custom formatter for API responses
	responseFormatter := func(val any) any {
		if resp, ok := val.(string); ok {
			return output.NewCollapsibleValue("Response Example", resp,
				output.WithCodeFences("json"), output.WithExpanded(false))
		}
		return val
	}

	// Custom formatter for curl commands
	curlFormatter := func(val any) any {
		if cmd, ok := val.(string); ok {
			return output.NewCollapsibleValue("cURL Command", cmd,
				output.WithCodeFences("bash"))
		}
		return val
	}

	data := []map[string]any{
		{
			"endpoint":    "GET /api/users/{id}",
			"description": "Retrieve user by ID",
			"curl": `curl -X GET "https://api.example.com/users/123" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Accept: application/json"`,
			"response": `{
  "id": 123,
  "username": "johndoe",
  "email": "john@example.com",
  "profile": {
    "firstName": "John",
    "lastName": "Doe",
    "avatar": "https://example.com/avatars/123.jpg"
  },
  "createdAt": "2024-01-15T10:30:00Z",
  "lastLogin": "2024-03-20T14:22:00Z"
}`,
		},
		{
			"endpoint":    "POST /api/users",
			"description": "Create new user",
			"curl": `curl -X POST "https://api.example.com/users" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "newuser",
    "email": "new@example.com",
    "password": "SecurePass123!"
  }'`,
			"response": `{
  "id": 456,
  "username": "newuser",
  "email": "new@example.com",
  "profile": {
    "firstName": "",
    "lastName": "",
    "avatar": null
  },
  "createdAt": "2024-03-21T09:15:00Z",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}`,
		},
	}

	table, err := output.NewTableContent("API Endpoint Examples", data,
		output.WithSchema(
			output.Field{Name: "endpoint", Type: "string"},
			output.Field{Name: "description", Type: "string"},
			output.Field{Name: "curl", Type: "string", Formatter: curlFormatter},
			output.Field{Name: "response", Type: "string", Formatter: responseFormatter},
		))
	if err != nil {
		log.Fatal(err)
	}

	renderInBothFormats("api_examples", table)
}

func generateErrorLogsReport() {
	// Custom formatter for stack traces
	stackTraceFormatter := func(val any) any {
		if trace, ok := val.([]string); ok {
			// Join the trace lines and format as Go code
			traceStr := ""
			for _, line := range trace {
				traceStr += line + "\n"
			}
			return output.NewCollapsibleValue(fmt.Sprintf("Stack Trace (%d frames)", len(trace)),
				traceStr, output.WithCodeFences(""), output.WithMaxLength(500))
		}
		return val
	}

	// Custom formatter for error context
	contextFormatter := func(val any) any {
		if ctx, ok := val.(map[string]any); ok {
			return output.NewCollapsibleValue("Error Context", ctx,
				output.WithCodeFences("json"))
		}
		return val
	}

	data := []map[string]any{
		{
			"timestamp": "2024-03-21 10:45:23",
			"level":     "ERROR",
			"message":   "Database connection failed",
			"stackTrace": []string{
				"goroutine 42 [running]:",
				"main.connectDB(0xc0000a6000, 0xc0000a8000)",
				"    /app/db/connection.go:45 +0x123",
				"main.initializeServices()",
				"    /app/main.go:78 +0x45",
				"main.main()",
				"    /app/main.go:32 +0x89",
			},
			"context": map[string]any{
				"host":        "db.example.com",
				"port":        5432,
				"database":    "production",
				"retry_count": 3,
				"last_error":  "connection refused",
			},
		},
		{
			"timestamp": "2024-03-21 11:02:45",
			"level":     "PANIC",
			"message":   "Nil pointer dereference",
			"stackTrace": []string{
				"panic: runtime error: invalid memory address or nil pointer dereference",
				"[signal SIGSEGV: segmentation violation code=0x1 addr=0x0 pc=0x123456]",
				"",
				"goroutine 1 [running]:",
				"main.processRequest(0x0)",
				"    /app/handlers/request.go:123 +0x89",
				"net/http.HandlerFunc.ServeHTTP(0x123456, 0x789abc, 0xc0000a6000, 0xc0000a8000)",
				"    /usr/local/go/src/net/http/server.go:2084 +0x44",
			},
			"context": map[string]any{
				"request_id": "550e8400-e29b-41d4-a716-446655440000",
				"user_id":    nil,
				"endpoint":   "/api/process",
				"method":     "POST",
			},
		},
	}

	table, err := output.NewTableContent("Application Error Logs", data,
		output.WithSchema(
			output.Field{Name: "timestamp", Type: "string"},
			output.Field{Name: "level", Type: "string"},
			output.Field{Name: "message", Type: "string"},
			output.Field{Name: "stackTrace", Type: "array", Formatter: stackTraceFormatter},
			output.Field{Name: "context", Type: "object", Formatter: contextFormatter},
		))
	if err != nil {
		log.Fatal(err)
	}

	renderInBothFormats("error_logs", table)
}

func renderInBothFormats(name string, content output.Content) {
	doc := output.New().AddContent(content).Build()

	// Render as HTML
	htmlRenderer := output.NewHTMLRendererWithCollapsible(output.DefaultRendererConfig)
	htmlOutput, err := htmlRenderer.Render(context.Background(), doc)
	if err != nil {
		log.Fatal(err)
	}

	htmlFile := fmt.Sprintf("%s.html", name)
	if err := os.WriteFile(htmlFile, htmlOutput, 0644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  ✅ Generated %s\n", htmlFile)

	// Render as Markdown
	mdRenderer := output.NewMarkdownRendererWithCollapsible(output.DefaultRendererConfig)
	mdOutput, err := mdRenderer.Render(context.Background(), doc)
	if err != nil {
		log.Fatal(err)
	}

	mdFile := fmt.Sprintf("%s.md", name)
	if err := os.WriteFile(mdFile, mdOutput, 0644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  ✅ Generated %s\n", mdFile)
}
