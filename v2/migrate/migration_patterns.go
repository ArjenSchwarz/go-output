package migrate

// MigrationPattern represents a complete v1 to v2 migration pattern
type MigrationPattern struct {
	Name        string
	Description string
	V1Code      string
	V2Code      string
	Notes       string
}

// GetAllMigrationPatterns returns all documented migration patterns
func GetAllMigrationPatterns() []MigrationPattern {
	return []MigrationPattern{
		// Basic Output Patterns
		{
			Name:        "SimpleOutput",
			Description: "Basic output without key ordering",
			V1Code: `output := &format.OutputArray{}
output.AddContents(map[string]interface{}{
    "Name": "Alice",
    "Age": 30,
})
output.Write()`,
			V2Code: `ctx := context.Background()
doc := output.New().
    Table("", []map[string]interface{}{
        {"Name": "Alice", "Age": 30},
    }).
    Build()

output.NewOutput(output.WithFormat(output.Table)).Render(ctx, doc)`,
			Notes: "v2 requires explicit context and separates document building from rendering",
		},
		{
			Name:        "OutputWithKeys",
			Description: "Output with explicit key ordering",
			V1Code: `output := &format.OutputArray{
    Keys: []string{"ID", "Name", "Status"},
}
output.AddContents(data)
output.Write()`,
			V2Code: `ctx := context.Background()
doc := output.New().
    Table("", data, output.WithKeys("ID", "Name", "Status")).
    Build()

output.NewOutput(output.WithFormat(output.Table)).Render(ctx, doc)`,
			Notes: "Key order is preserved exactly as specified",
		},
		{
			Name:        "OutputWithSettings",
			Description: "Output with settings configuration",
			V1Code: `settings := format.NewOutputSettings()
settings.OutputFormat = "json"

output := &format.OutputArray{
    Settings: settings,
}
output.AddContents(data)
output.Write()`,
			V2Code: `ctx := context.Background()
doc := output.New().
    Table("", data).
    Build()

output.NewOutput(output.WithFormat(output.JSON)).Render(ctx, doc)`,
			Notes: "Settings are replaced with functional options",
		},

		// Multiple Tables Patterns
		{
			Name:        "MultipleTables",
			Description: "Multiple tables with different keys",
			V1Code: `output := &format.OutputArray{}

output.Keys = []string{"Name", "Email"}
output.AddContents(userData)
output.AddToBuffer()

output.Keys = []string{"ID", "Status", "Time"}
output.AddContents(statusData)
output.AddToBuffer()

output.Write()`,
			V2Code: `ctx := context.Background()
doc := output.New().
    Table("Users", userData, output.WithKeys("Name", "Email")).
    Table("Status", statusData, output.WithKeys("ID", "Status", "Time")).
    Build()

output.NewOutput(output.WithFormat(output.Table)).Render(ctx, doc)`,
			Notes: "Each table maintains its own key order independently",
		},
		{
			Name:        "TablesWithHeaders",
			Description: "Tables with header text",
			V1Code: `output := &format.OutputArray{}
output.AddHeader("User Report")
output.Keys = []string{"Name", "Role"}
output.AddContents(users)
output.AddHeader("System Status")
output.Keys = []string{"Service", "Status"}
output.AddContents(services)
output.Write()`,
			V2Code: `ctx := context.Background()
doc := output.New().
    Header("User Report").
    Table("", users, output.WithKeys("Name", "Role")).
    Header("System Status").
    Table("", services, output.WithKeys("Service", "Status")).
    Build()

output.NewOutput(output.WithFormat(output.Table)).Render(ctx, doc)`,
			Notes: "Headers are added as separate content items",
		},

		// File Output Patterns
		{
			Name:        "FileOutput",
			Description: "Output to file",
			V1Code: `settings := format.NewOutputSettings()
settings.OutputFile = "report.json"
settings.OutputFormat = "json"

output := &format.OutputArray{Settings: settings}
output.AddContents(data)
output.Write()`,
			V2Code: `ctx := context.Background()
doc := output.New().
    Table("", data).
    Build()

output.NewOutput(
    output.WithFormat(output.JSON),
    output.WithWriter(output.NewFileWriter(".", "report.json")),
).Render(ctx, doc)`,
			Notes: "File output uses dedicated FileWriter",
		},
		{
			Name:        "MultipleOutputFormats",
			Description: "Output to multiple formats",
			V1Code: `settings := format.NewOutputSettings()
settings.OutputFormat = "table"
settings.OutputFile = "report.html"
settings.OutputFileFormat = "html"

output := &format.OutputArray{Settings: settings}
output.AddContents(data)
output.Write()`,
			V2Code: `ctx := context.Background()
doc := output.New().
    Table("", data).
    Build()

output.NewOutput(
    output.WithFormat(output.Table),
    output.WithFormat(output.HTML),
    output.WithWriter(&output.StdoutWriter{}),
    output.WithWriter(output.NewFileWriter(".", "report.html")),
).Render(ctx, doc)`,
			Notes: "v2 supports multiple formats and destinations in one render call",
		},

		// S3 Output Pattern
		{
			Name:        "S3Output",
			Description: "Output to S3",
			V1Code: `settings := format.NewOutputSettings()
settings.OutputS3Bucket = "my-bucket"
settings.OutputS3Key = "reports/output.json"
settings.OutputFormat = "json"

output := &format.OutputArray{Settings: settings}
output.AddContents(data)
output.Write()`,
			V2Code: `ctx := context.Background()
doc := output.New().
    Table("", data).
    Build()

s3Client := s3.NewFromConfig(cfg)
output.NewOutput(
    output.WithFormat(output.JSON),
    output.WithWriter(output.NewS3Writer(s3Client, "my-bucket", "reports/output.json")),
).Render(ctx, doc)`,
			Notes: "S3 output requires AWS SDK client",
		},

		// Progress Patterns
		{
			Name:        "BasicProgress",
			Description: "Progress indicator",
			V1Code: `settings := format.NewOutputSettings()
p := format.NewProgress(settings)
p.SetTotal(100)
p.SetColor(format.ProgressColorGreen)

for i := 0; i < 100; i++ {
    p.Increment(1)
    p.SetStatus(fmt.Sprintf("Item %d", i))
}
p.Complete()`,
			V2Code: `p := output.NewProgress(output.Table,
    output.WithProgressColor(output.ProgressColorGreen),
)
p.SetTotal(100)

for i := 0; i < 100; i++ {
    p.Increment(1)
    p.SetStatus(fmt.Sprintf("Item %d", i))
}
p.Complete()`,
			Notes: "Progress API is mostly compatible, just different initialization",
		},
		{
			Name:        "ProgressWithOutput",
			Description: "Progress integrated with output",
			V1Code: `settings := format.NewOutputSettings()
settings.SetOutputFormat("table")
settings.ProgressOptions = format.ProgressOptions{
    Color: format.ProgressColorBlue,
    Status: "Loading",
    TrackerLength: 50,
}`,
			V2Code: `out := output.NewOutput(
    output.WithFormat(output.Table),
    output.WithProgress(output.NewProgress(output.Table,
        output.WithProgressColor(output.ProgressColorBlue),
        output.WithProgressStatus("Loading"),
        output.WithTrackerLength(50),
    )),
)`,
			Notes: "Progress is configured as an output option",
		},

		// Transform Patterns
		{
			Name:        "EmojiTransform",
			Description: "Emoji conversion",
			V1Code: `settings := format.NewOutputSettings()
settings.UseEmoji = true`,
			V2Code: `output.WithTransformer(&output.EmojiTransformer{})`,
			Notes:  "Emoji conversion is now a transformer",
		},
		{
			Name:        "ColorTransform",
			Description: "Color output",
			V1Code: `settings := format.NewOutputSettings()
settings.UseColors = true`,
			V2Code: `output.WithTransformer(&output.ColorTransformer{})`,
			Notes:  "Colors are applied via transformer",
		},
		{
			Name:        "SortTransform",
			Description: "Sort by key",
			V1Code: `settings := format.NewOutputSettings()
settings.SortKey = "Name"`,
			V2Code: `output.WithTransformer(&output.SortTransformer{
    Key: "Name",
    Ascending: true,
})`,
			Notes: "Sorting is now a transformer with more options",
		},
		{
			Name:        "LineSplitTransform",
			Description: "Line splitting",
			V1Code: `settings := format.NewOutputSettings()
settings.LineSplitColumn = "Tags"
settings.LineSplitSeparator = ","`,
			V2Code: `output.WithTransformer(&output.LineSplitTransformer{
    Column: "Tags",
    Separator: ",",
})`,
			Notes: "Line splitting is now a transformer",
		},

		// Table Styling Patterns
		{
			Name:        "TableStyle",
			Description: "Table styling",
			V1Code: `settings := format.NewOutputSettings()
settings.TableStyle = "ColoredBright"`,
			V2Code: `output.WithTableStyle("ColoredBright")`,
			Notes:  "Table styles are preserved as options",
		},

		// Markdown Features
		{
			Name:        "MarkdownTOC",
			Description: "Markdown table of contents",
			V1Code: `settings := format.NewOutputSettings()
settings.HasTOC = true
settings.OutputFormat = "markdown"`,
			V2Code: `output.NewOutput(
    output.WithFormat(output.Markdown),
    output.WithTOC(true),
)`,
			Notes: "TOC is enabled via option",
		},
		{
			Name:        "MarkdownFrontMatter",
			Description: "Markdown front matter",
			V1Code: `settings := format.NewOutputSettings()
settings.FrontMatter = map[string]string{
    "title": "Report",
    "date": "2024-01-01",
}
settings.OutputFormat = "markdown"`,
			V2Code: `output.NewOutput(
    output.WithFormat(output.Markdown),
    output.WithFrontMatter(map[string]string{
        "title": "Report",
        "date": "2024-01-01",
    }),
)`,
			Notes: "Front matter is passed as option",
		},

		// Chart and Diagram Patterns
		{
			Name:        "DOTGraph",
			Description: "DOT format graph",
			V1Code: `settings := format.NewOutputSettings()
settings.OutputFormat = "dot"
settings.DotFromColumn = "source"
settings.DotToColumn = "target"`,
			V2Code: `doc := output.New().
    Graph("Network", data, 
        output.WithFromTo("source", "target"),
    ).
    Build()

output.NewOutput(output.WithFormat(output.DOT)).Render(ctx, doc)`,
			Notes: "Graphs use dedicated Graph() builder method",
		},
		{
			Name:        "MermaidGantt",
			Description: "Mermaid Gantt chart",
			V1Code: `settings := format.NewOutputSettings()
settings.OutputFormat = "mermaid"
settings.MermaidSettings = &mermaid.Settings{
    ChartType: "gantt",
}`,
			V2Code: `doc := output.New().
    Chart("Timeline", ganttTasks,
        output.WithChartType("gantt"),
    ).
    Build()

output.NewOutput(output.WithFormat(output.Mermaid)).Render(ctx, doc)`,
			Notes: "Charts use dedicated Chart() builder method",
		},
		{
			Name:        "DrawIODiagram",
			Description: "Draw.io diagram",
			V1Code: `drawio.SetHeaderValues(drawio.Header{
    Label: "%Name%",
    Style: "shape=image;image=%Image%",
    Layout: "horizontalflow",
    NodeSpacing: 40,
    LevelSpacing: 100,
})`,
			V2Code: `doc := output.New().
    DrawIO("Architecture", nodes,
        output.WithDrawIOLabel("%Name%"),
        output.WithDrawIOStyle("shape=image;image=%Image%"),
        output.WithDrawIOLayout("horizontalflow"),
        output.WithDrawIOSpacing(40, 100, 20),
    ).
    Build()

output.NewOutput(output.WithFormat(output.DrawIO)).Render(ctx, doc)`,
			Notes: "Draw.io uses dedicated DrawIO() builder method with options",
		},

		// Complex Example
		{
			Name:        "CompleteExample",
			Description: "Complete migration example",
			V1Code: `settings := format.NewOutputSettings()
settings.OutputFormat = "table"
settings.OutputFile = "report.json"
settings.OutputFileFormat = "json"
settings.UseColors = true
settings.UseEmoji = true
settings.SortKey = "Name"
settings.TableStyle = "ColoredBright"

output := &format.OutputArray{
    Settings: settings,
    Keys: []string{"ID", "Name", "Status"},
}

output.AddHeader("User Report")
output.AddContents(userData)
output.Write()`,
			V2Code: `ctx := context.Background()

doc := output.New().
    Header("User Report").
    Table("", userData, output.WithKeys("ID", "Name", "Status")).
    Build()

out := output.NewOutput(
    // Formats
    output.WithFormat(output.Table),
    output.WithFormat(output.JSON),
    
    // Writers
    output.WithWriter(&output.StdoutWriter{}),
    output.WithWriter(output.NewFileWriter(".", "report.json")),
    
    // Transformers
    output.WithTransformer(&output.ColorTransformer{}),
    output.WithTransformer(&output.EmojiTransformer{}),
    output.WithTransformer(&output.SortTransformer{Key: "Name", Ascending: true}),
    
    // Styling
    output.WithTableStyle("ColoredBright"),
)

err := out.Render(ctx, doc)`,
			Notes: "v2 provides cleaner separation of concerns",
		},
	}
}
