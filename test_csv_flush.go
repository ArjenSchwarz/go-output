package main

import (
	"os"
	"github.com/ArjenSchwarz/go-output/drawio"
)

func main() {
	// Create test data
	header := drawio.DefaultHeader()
	headerRow := []string{"Name", "Type", "Location"}
	contents := []map[string]string{
		{"Name": "Server1", "Type": "Web Server", "Location": "US-East"},
		{"Name": "Server2", "Type": "Database", "Location": "US-West"},
	}
	
	// Test with file output
	filename := "test_output.csv"
	drawio.CreateCSV(header, headerRow, contents, filename)
	
	// Check if file was created and has content
	if info, err := os.Stat(filename); err != nil {
		panic("File was not created: " + err.Error())
	} else {
		println("File created with size:", info.Size())
	}
	
	// Read and display content
	content, err := os.ReadFile(filename)
	if err != nil {
		panic("Could not read file: " + err.Error())
	}
	
	println("File content:")
	println(string(content))
	
	// Clean up
	os.Remove(filename)
}