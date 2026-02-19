package main

import (
	format "github.com/ArjenSchwarz/go-output"
	"github.com/ArjenSchwarz/go-output/mermaid"
)

func main() {
	// Test the fix for integer values in piecharts
	settings := format.NewOutputSettings()
	settings.SetOutputFormat("mermaid")
	settings.MermaidSettings = &mermaid.Settings{ChartType: "piechart"}
	settings.AddFromToColumns("Label", "Value")

	output := format.OutputArray{
		Settings: settings,
		Keys:     []string{"Label", "Value"},
	}

	// Add test data with integer values
	output.AddContents(map[string]interface{}{
		"Label": "Users",
		"Value": 42, // integer - this was the bug
	})
	
	output.AddContents(map[string]interface{}{
		"Label": "Items", 
		"Value": int64(100), // int64
	})
	
	output.AddContents(map[string]interface{}{
		"Label": "Score",
		"Value": 75.5, // float64
	})

	// This should now work with integer values
	output.Write()
}