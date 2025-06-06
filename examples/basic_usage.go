package main

import (
	format "github.com/ArjenSchwarz/go-output"
)

func main() {
	// Example 1: Basic JSON output
	basicJSONExample()

	// Example 2: Table output with styling
	tableExample()

	// Example 3: HTML output with multiple sections
	htmlExample()

	// Example 4: Mermaid flowchart
	mermaidExample()

	// Example 5: CSV output to file
	csvFileExample()
}

func basicJSONExample() {
	settings := format.NewOutputSettings()
	settings.SetOutputFormat("json")

	output := format.OutputArray{
		Settings: settings,
		Keys:     []string{"Name", "Department", "Salary", "Active"},
	}

	// Add employee data
	output.AddContents(map[string]interface{}{
		"Name":       "Alice Johnson",
		"Department": "Engineering",
		"Salary":     75000,
		"Active":     true,
	})

	output.AddContents(map[string]interface{}{
		"Name":       "Bob Smith",
		"Department": "Marketing",
		"Salary":     65000,
		"Active":     false,
	})

	output.AddContents(map[string]interface{}{
		"Name":       "Carol Davis",
		"Department": "Engineering",
		"Salary":     80000,
		"Active":     true,
	})

	output.Write()
}

func tableExample() {
	settings := format.NewOutputSettings()
	settings.SetOutputFormat("table")
	settings.Title = "Employee Report"
	settings.SortKey = "Name"
	settings.UseColors = true
	settings.TableStyle = format.TableStyles["ColoredBright"]

	output := format.OutputArray{
		Settings: settings,
		Keys:     []string{"Name", "Department", "Salary", "Active"},
	}

	// Add the same employee data
	employees := []map[string]interface{}{
		{
			"Name":       "Alice Johnson",
			"Department": "Engineering",
			"Salary":     75000,
			"Active":     true,
		},
		{
			"Name":       "Bob Smith",
			"Department": "Marketing",
			"Salary":     65000,
			"Active":     false,
		},
		{
			"Name":       "Carol Davis",
			"Department": "Engineering",
			"Salary":     80000,
			"Active":     true,
		},
	}

	for _, emp := range employees {
		output.AddContents(emp)
	}

	output.Write()
}

func htmlExample() {
	settings := format.NewOutputSettings()
	settings.SetOutputFormat("html")
	settings.Title = "Company Employee Report"
	settings.HasTOC = true
	settings.UseEmoji = true
	settings.OutputFile = "employee_report.html"

	output := format.OutputArray{
		Settings: settings,
		Keys:     []string{"Name", "Department", "Salary", "Active"},
	}

	// Engineering section
	output.AddHeader("Engineering Department")
	output.AddContents(map[string]interface{}{
		"Name":       "Alice Johnson",
		"Department": "Engineering",
		"Salary":     75000,
		"Active":     true,
	})
	output.AddContents(map[string]interface{}{
		"Name":       "Carol Davis",
		"Department": "Engineering",
		"Salary":     80000,
		"Active":     true,
	})
	output.AddToBuffer()

	// Marketing section
	output.AddHeader("Marketing Department")
	output.AddContents(map[string]interface{}{
		"Name":       "Bob Smith",
		"Department": "Marketing",
		"Salary":     65000,
		"Active":     false,
	})
	output.AddContents(map[string]interface{}{
		"Name":       "Dave Wilson",
		"Department": "Marketing",
		"Salary":     70000,
		"Active":     true,
	})
	output.AddToBuffer()

	output.Write()
}

func mermaidExample() {
	settings := format.NewOutputSettings()
	settings.SetOutputFormat("mermaid")
	settings.MermaidSettings.ChartType = "flowchart"
	settings.AddFromToColumns("Manager", "Employee")

	output := format.OutputArray{
		Settings: settings,
		Keys:     []string{"Manager", "Employee"},
	}

	// Define reporting relationships
	relationships := []map[string]interface{}{
		{"Manager": "CEO", "Employee": "CTO"},
		{"Manager": "CEO", "Employee": "VP Marketing"},
		{"Manager": "CTO", "Employee": "Alice Johnson"},
		{"Manager": "CTO", "Employee": "Carol Davis"},
		{"Manager": "VP Marketing", "Employee": "Bob Smith"},
		{"Manager": "VP Marketing", "Employee": "Dave Wilson"},
	}

	for _, rel := range relationships {
		output.AddContents(rel)
	}

	output.Write()
}

func csvFileExample() {
	settings := format.NewOutputSettings()
	settings.SetOutputFormat("csv")
	settings.OutputFile = "employees.csv"
	settings.SortKey = "Department"

	output := format.OutputArray{
		Settings: settings,
		Keys:     []string{"Name", "Department", "Salary", "Skills", "Active"},
	}

	// Add employees with skill arrays
	employees := []map[string]interface{}{
		{
			"Name":       "Alice Johnson",
			"Department": "Engineering",
			"Salary":     75000,
			"Skills":     []string{"Go", "Python", "Docker"},
			"Active":     true,
		},
		{
			"Name":       "Bob Smith",
			"Department": "Marketing",
			"Salary":     65000,
			"Skills":     []string{"SEO", "Analytics", "Content"},
			"Active":     false,
		},
		{
			"Name":       "Carol Davis",
			"Department": "Engineering",
			"Salary":     80000,
			"Skills":     []string{"JavaScript", "React", "AWS"},
			"Active":     true,
		},
		{
			"Name":       "Dave Wilson",
			"Department": "Marketing",
			"Salary":     70000,
			"Skills":     []string{"PPC", "Social Media", "Design"},
			"Active":     true,
		},
	}

	for _, emp := range employees {
		output.AddContents(emp)
	}

	output.Write()
}
