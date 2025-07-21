package migrate

import (
	"fmt"
	"strings"
)

// Example represents a migration example for documentation
type Example struct {
	Title       string
	Description string
	V1Code      string
	V2Code      string
	Notes       []string
}

// GetMigrationExamples returns categorized migration examples for documentation
func GetMigrationExamples() map[string][]Example {
	return map[string][]Example{
		"Basic Usage": {
			{
				Title:       "Simple Table Output",
				Description: "Basic table output without configuration",
				V1Code: `output := &format.OutputArray{}
output.AddContents(data)
output.Write()`,
				V2Code: `ctx := context.Background()
doc := output.New().
    Table("", data).
    Build()

output.NewOutput(output.WithFormat(output.Table)).Render(ctx, doc)`,
				Notes: []string{
					"Context is now required for all operations",
					"Document building is separate from rendering",
				},
			},
			{
				Title:       "Table with Key Ordering",
				Description: "Preserve specific column order",
				V1Code: `output := &format.OutputArray{
    Keys: []string{"ID", "Name", "Status"},
}
output.AddContents(data)
output.Write()`,
				V2Code: `doc := output.New().
    Table("", data, output.WithKeys("ID", "Name", "Status")).
    Build()`,
				Notes: []string{
					"Key order is preserved exactly as specified",
					"Each table can have different key ordering",
				},
			},
		},
		"Configuration": {
			{
				Title:       "Output Format Configuration",
				Description: "Configure output format and options",
				V1Code: `settings := format.NewOutputSettings()
settings.OutputFormat = "json"
settings.UseEmoji = true
settings.UseColors = true`,
				V2Code: `out := output.NewOutput(
    output.WithFormat(output.JSON),
    output.WithTransformer(&output.EmojiTransformer{}),
    output.WithTransformer(&output.ColorTransformer{}),
)`,
				Notes: []string{
					"Settings replaced with functional options",
					"Features are now transformers",
				},
			},
			{
				Title:       "File Output",
				Description: "Write output to file",
				V1Code: `settings.OutputFile = "report.csv"
settings.OutputFormat = "csv"`,
				V2Code: `output.WithFormat(output.CSV),
output.WithWriter(output.NewFileWriter(".", "report.csv"))`,
				Notes: []string{
					"File output uses dedicated Writer",
					"Can output to multiple destinations",
				},
			},
		},
		"Advanced Features": {
			{
				Title:       "Progress Indicators",
				Description: "Show progress during processing",
				V1Code: `p := format.NewProgress(settings)
p.SetTotal(100)
p.SetColor(format.ProgressColorGreen)`,
				V2Code: `p := output.NewProgress(output.Table,
    output.WithProgressColor(output.ProgressColorGreen),
)
p.SetTotal(100)`,
				Notes: []string{
					"Progress is format-aware",
					"Color constants moved to output package",
				},
			},
			{
				Title:       "Multiple Tables",
				Description: "Output multiple tables with different schemas",
				V1Code: `output.Keys = []string{"Name", "Email"}
output.AddContents(users)
output.AddToBuffer()

output.Keys = []string{"Service", "Status"}
output.AddContents(services)
output.Write()`,
				V2Code: `doc := output.New().
    Table("Users", users, output.WithKeys("Name", "Email")).
    Table("Services", services, output.WithKeys("Service", "Status")).
    Build()`,
				Notes: []string{
					"No need for AddToBuffer()",
					"Each table has independent schema",
				},
			},
		},
	}
}

// GenerateExampleMarkdown generates markdown documentation for examples
func GenerateExampleMarkdown() string {
	var sb strings.Builder
	examples := GetMigrationExamples()

	sb.WriteString("# Migration Examples\n\n")
	sb.WriteString("This document provides practical examples of migrating from go-output v1 to v2.\n\n")

	categories := []string{"Basic Usage", "Configuration", "Advanced Features"}

	for _, category := range categories {
		if exampleList, ok := examples[category]; ok {
			sb.WriteString(fmt.Sprintf("## %s\n\n", category))

			for _, example := range exampleList {
				sb.WriteString(fmt.Sprintf("### %s\n\n", example.Title))
				sb.WriteString(fmt.Sprintf("%s\n\n", example.Description))

				sb.WriteString("**Before (v1):**\n```go\n")
				sb.WriteString(example.V1Code)
				sb.WriteString("\n```\n\n")

				sb.WriteString("**After (v2):**\n```go\n")
				sb.WriteString(example.V2Code)
				sb.WriteString("\n```\n\n")

				if len(example.Notes) > 0 {
					sb.WriteString("**Notes:**\n")
					for _, note := range example.Notes {
						sb.WriteString(fmt.Sprintf("- %s\n", note))
					}
					sb.WriteString("\n")
				}
			}
		}
	}

	return sb.String()
}

// GetQuickReference returns a quick reference card for migration
func GetQuickReference() string {
	return `# go-output v2 Migration Quick Reference

## Import Changes
| v1 | v2 |
|---|---|
| github.com/ArjenSchwarz/go-output/format | github.com/ArjenSchwarz/go-output/v2 |

## Common Replacements
| v1 | v2 |
|---|---|
| &format.OutputArray{} | output.New() |
| output.AddContents(data) | .Table("", data) |
| output.Write() | output.NewOutput().Render(ctx, doc) |
| output.Keys = [...] | output.WithKeys(...) |
| output.AddHeader("text") | .Header("text") |
| output.AddToBuffer() | (not needed) |

## Settings to Options
| v1 Setting | v2 Option |
|---|---|
| settings.OutputFormat = "json" | output.WithFormat(output.JSON) |
| settings.OutputFile = "file.ext" | output.WithWriter(output.NewFileWriter(".", "file.ext")) |
| settings.UseEmoji = true | output.WithTransformer(&output.EmojiTransformer{}) |
| settings.UseColors = true | output.WithTransformer(&output.ColorTransformer{}) |
| settings.SortKey = "Name" | output.WithTransformer(&output.SortTransformer{Key: "Name"}) |
| settings.TableStyle = "style" | output.WithTableStyle("style") |
| settings.HasTOC = true | output.WithTOC(true) |

## Progress Changes
| v1 | v2 |
|---|---|
| format.NewProgress(settings) | output.NewProgress(format, options...) |
| format.ProgressColorGreen | output.ProgressColorGreen |

## Key Concepts
1. **Context Required**: All rendering requires context.Context
2. **Builder Pattern**: Build documents, then render them
3. **No Global State**: Each instance is independent
4. **Functional Options**: Configure with WithXxx() functions
5. **Type Safety**: Compile-time format checking
`
}
