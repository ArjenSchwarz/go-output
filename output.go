package format

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/emicklei/dot"
	"github.com/gosimple/slug"
	"github.com/jedib0t/go-pretty/v6/table"
	"gopkg.in/yaml.v3"

	"github.com/ArjenSchwarz/go-output/drawio"
	"github.com/ArjenSchwarz/go-output/mermaid"
	"github.com/ArjenSchwarz/go-output/templates"
)

var buffer bytes.Buffer
var toc []string

// Store raw data for dual output format support
var savedRawData []OutputHolder
var savedAllKeys []string

// Store section structure for table-based formats
type SavedSection struct {
	Title    string
	Keys     []string
	Contents []OutputHolder
}

var savedSections []SavedSection

// OutputHolder holds key-value pairs that belong together in the output
type OutputHolder struct {
	Contents map[string]interface{}
}

// OutputArray holds all the different OutputHolders that will be provided as
// output, as well as the keys (headers) that will actually need to be printed
type OutputArray struct {
	Settings *OutputSettings
	Contents []OutputHolder
	Keys     []string
}

// GetContentsMap returns a stringmap of the output contents
func (output OutputArray) GetContentsMap() []map[string]string {
	total := make([]map[string]string, 0, len(output.Contents))
	for _, holder := range output.Contents {
		values := make(map[string]string)
		for _, key := range output.Keys {
			if val, ok := holder.Contents[key]; ok {
				values[key] = output.toString(val)
			}
		}
		total = append(total, values)
	}
	return total
}

// GetContentsMapWithAllKeys returns a stringmap of the output contents using all collected keys
func (output OutputArray) GetContentsMapWithAllKeys() []map[string]string {
	total := make([]map[string]string, 0, len(output.Contents))
	for _, holder := range output.Contents {
		values := make(map[string]string)
		for _, key := range savedAllKeys {
			if val, ok := holder.Contents[key]; ok {
				values[key] = output.toString(val)
			}
		}
		total = append(total, values)
	}
	return total
}

// GetContentsMapRaw returns a interface map of the output contents
func (output OutputArray) GetContentsMapRaw() []map[string]interface{} {
	total := make([]map[string]interface{}, 0, len(output.Contents))
	for _, holder := range output.Contents {
		values := make(map[string]interface{})
		for _, key := range output.Keys {
			if val, ok := holder.Contents[key]; ok {
				values[key] = val
			}
		}
		total = append(total, values)
	}
	return total
}

// GetContentsMapRawWithAllKeys returns a interface map of the output contents using all collected keys
func (output OutputArray) GetContentsMapRawWithAllKeys() []map[string]interface{} {
	total := make([]map[string]interface{}, 0, len(output.Contents))
	for _, holder := range output.Contents {
		values := make(map[string]interface{})
		for _, key := range savedAllKeys {
			if val, ok := holder.Contents[key]; ok {
				values[key] = val
			}
		}
		total = append(total, values)
	}
	return total
}

// Write will provide the output as configured in the configuration
func (output OutputArray) Write() {
	stopActiveProgress()
	var result []byte
	var savedBufferContent []byte
	var savedContents []OutputHolder

	// Save buffer content and data if they exist for potential file output use
	if buffer.Len() > 0 {
		savedBufferContent = make([]byte, buffer.Len())
		copy(savedBufferContent, buffer.Bytes())
		// Note: Contents might be nil due to AddToBuffer clearing them
		// We'll need to extract data from buffer for different format generation
	}

	// Save contents for file output regeneration if needed
	if output.Contents != nil {
		savedContents = make([]OutputHolder, len(output.Contents))
		copy(savedContents, output.Contents)
	}

	// Handle stdout output
	switch output.Settings.OutputFormat {
	case "csv":
		if buffer.Len() == 0 {
			result = output.toCSV()
		} else {
			result = buffer.Bytes()
		}
	case "html":
		if buffer.Len() == 0 {
			result = output.toHTML()
		} else {
			result = output.bufferToHTML()
		}
	case "table":
		if buffer.Len() == 0 {
			result = output.toTable()
		} else {
			result = buffer.Bytes()
		}
	case "markdown":
		if buffer.Len() == 0 {
			result = output.toMarkdown()
		} else {
			result = output.bufferToMarkdown()
		}
	case "mermaid":
		if output.Settings.FromToColumns == nil && output.Settings.MermaidSettings == nil {
			log.Fatal("This command doesn't currently support the mermaid output format")
		}
		result = output.toMermaid()
	case "drawio":
		if !output.Settings.DrawIOHeader.IsSet() {
			log.Fatal("This command doesn't currently support the drawio output format")
		}
		drawio.CreateCSV(output.Settings.DrawIOHeader, output.Keys, output.GetContentsMap(), output.Settings.OutputFile)
	case "dot":
		if output.Settings.FromToColumns == nil {
			log.Fatal("This command doesn't currently support the dot output format")
		}
		result = output.toDot()
	case "yaml":
		if buffer.Len() == 0 {
			result = output.toYAML()
		} else {
			result = buffer.Bytes()
		}
	default:
		if buffer.Len() == 0 {
			result = output.toJSON()
		} else {
			result = buffer.Bytes()
		}
	}
	if len(result) != 0 {
		err := PrintByteSlice(result, "", output.Settings.S3Bucket)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	// Handle file output with proper dual format support
	if output.Settings.OutputFile != "" {
		if output.Settings.OutputFileFormat == "" {
			output.Settings.OutputFileFormat = output.Settings.OutputFormat
		}

		// Check if we need to generate different format for file or if we have multi-key sections
		needsDifferentFormat := len(savedBufferContent) > 0 &&
			output.Settings.OutputFileFormat != output.Settings.OutputFormat &&
			!output.supportsBufferConversion(output.Settings.OutputFileFormat)

		needsAllKeysGeneration := len(savedAllKeys) > 0 && len(savedRawData) > 0

		if needsDifferentFormat || needsAllKeysGeneration {
			// Generate content in the file format using saved raw data
			if len(savedRawData) > 0 {
				// Create a temporary OutputArray with saved data
				// Use all collected keys if available, otherwise fall back to current keys
				keysToUse := output.Keys
				if len(savedAllKeys) > 0 {
					keysToUse = savedAllKeys
				}
				tempOutput := OutputArray{
					Settings: output.Settings,
					Contents: savedRawData,
					Keys:     keysToUse,
				}

				switch output.Settings.OutputFileFormat {
				case "csv":
					if len(savedSections) > 0 {
						result = tempOutput.toCSVWithSections()
					} else {
						result = tempOutput.toCSV()
					}
				case "html":
					if len(savedSections) > 0 {
						result = tempOutput.toHTMLWithSections()
					} else {
						result = tempOutput.toHTML()
					}
				case "table":
					if len(savedSections) > 0 {
						result = tempOutput.toTableWithSections()
					} else {
						result = tempOutput.toTable()
					}
				case "markdown":
					if len(savedSections) > 0 {
						result = tempOutput.toMarkdownWithSections()
					} else {
						result = tempOutput.toMarkdown()
					}
				case "mermaid":
					if output.Settings.FromToColumns == nil && output.Settings.MermaidSettings == nil {
						log.Fatal("This command doesn't currently support the mermaid output format")
					}
					result = tempOutput.toMermaid()
				case "drawio":
					if !output.Settings.DrawIOHeader.IsSet() {
						log.Fatal("This command doesn't currently support the drawio output format")
					}
					if len(savedAllKeys) > 0 {
						drawio.CreateCSV(output.Settings.DrawIOHeader, savedAllKeys, tempOutput.GetContentsMapWithAllKeys(), output.Settings.OutputFile)
					} else {
						drawio.CreateCSV(output.Settings.DrawIOHeader, tempOutput.Keys, tempOutput.GetContentsMap(), output.Settings.OutputFile)
					}
					return
				case "dot":
					if output.Settings.FromToColumns == nil {
						log.Fatal("This command doesn't currently support the dot output format")
					}
					result = tempOutput.toDot()
				case "yaml":
					if len(savedAllKeys) > 0 {
						result = tempOutput.toYAMLWithAllKeys()
					} else {
						result = tempOutput.toYAML()
					}
				default:
					if len(savedAllKeys) > 0 {
						result = tempOutput.toJSONWithAllKeys()
					} else {
						result = tempOutput.toJSON()
					}
				}
			} else {
				// Fallback to existing behavior if no saved data
				switch output.Settings.OutputFileFormat {
				case "csv":
					result = output.toCSV()
				case "html":
					result = output.toHTML()
				case "table":
					result = output.toTable()
				case "markdown":
					result = output.toMarkdown()
				case "mermaid":
					if output.Settings.FromToColumns == nil && output.Settings.MermaidSettings == nil {
						log.Fatal("This command doesn't currently support the mermaid output format")
					}
					result = output.toMermaid()
				case "drawio":
					if !output.Settings.DrawIOHeader.IsSet() {
						log.Fatal("This command doesn't currently support the drawio output format")
					}
					drawio.CreateCSV(output.Settings.DrawIOHeader, output.Keys, output.GetContentsMap(), output.Settings.OutputFile)
					return
				case "dot":
					if output.Settings.FromToColumns == nil {
						log.Fatal("This command doesn't currently support the dot output format")
					}
					result = output.toDot()
				case "yaml":
					result = output.toYAML()
				default:
					result = output.toJSON()
				}
			}
		} else {
			// Use existing logic for file output
			switch output.Settings.OutputFileFormat {
			case "csv":
				if buffer.Len() == 0 {
					result = output.toCSV()
				} else {
					result = buffer.Bytes()
				}
			case "html":
				if buffer.Len() == 0 {
					result = output.toHTML()
				} else {
					result = output.bufferToHTML()
				}
			case "table":
				if buffer.Len() == 0 {
					result = output.toTable()
				} else {
					result = buffer.Bytes()
				}
			case "markdown":
				if buffer.Len() == 0 {
					result = output.toMarkdown()
				} else {
					result = output.bufferToMarkdown()
				}
			case "mermaid":
				if output.Settings.FromToColumns == nil && output.Settings.MermaidSettings == nil {
					log.Fatal("This command doesn't currently support the mermaid output format")
				}
				result = output.toMermaid()
			case "drawio":
				if !output.Settings.DrawIOHeader.IsSet() {
					log.Fatal("This command doesn't currently support the drawio output format")
				}
				drawio.CreateCSV(output.Settings.DrawIOHeader, output.Keys, output.GetContentsMap(), output.Settings.OutputFile)
				return
			case "dot":
				if output.Settings.FromToColumns == nil {
					log.Fatal("This command doesn't currently support the dot output format")
				}
				result = output.toDot()
			case "yaml":
				if buffer.Len() == 0 {
					result = output.toYAML()
				} else {
					result = buffer.Bytes()
				}
			default:
				if buffer.Len() == 0 {
					result = output.toJSON()
				} else {
					result = buffer.Bytes()
				}
			}
		}

		if len(result) != 0 {
			err := PrintByteSlice(result, output.Settings.OutputFile, output.Settings.S3Bucket)
			if err != nil {
				log.Fatal(err.Error())
			}
		}
	}

	// Reset buffer and saved raw data after all output operations
	buffer.Reset()
	savedRawData = nil
	savedAllKeys = nil
	savedSections = nil
}

// supportsBufferConversion checks if a format supports converting buffer content
// rather than regenerating from raw data
func (output OutputArray) supportsBufferConversion(format string) bool {
	// Only HTML and Markdown support buffer conversion, and only when the source format
	// is also HTML or Markdown respectively. If converting from table to HTML/Markdown,
	// we need to regenerate from raw data.
	switch format {
	case "html":
		return output.Settings.OutputFormat == "html" // Only if source is also HTML
	case "markdown":
		return output.Settings.OutputFormat == "markdown" // Only if source is also Markdown
	default:
		return false // Other formats like json, yaml, csv need to be regenerated from raw data
	}
}

func (output OutputArray) toCSV() []byte {
	tableBuf := new(bytes.Buffer)
	t := output.buildTable()
	t.SetOutputMirror(tableBuf)
	t.RenderCSV()
	return tableBuf.Bytes()
}

func (output OutputArray) toJSON() []byte {
	jsonString, _ := json.Marshal(output.GetContentsMapRaw())
	return jsonString
}

// toJSONWithAllKeys generates JSON using all collected keys from AddToBuffer operations
func (output OutputArray) toJSONWithAllKeys() []byte {
	total := make([]map[string]interface{}, 0, len(output.Contents))
	for _, holder := range output.Contents {
		values := make(map[string]interface{})
		for _, key := range savedAllKeys {
			if val, ok := holder.Contents[key]; ok {
				values[key] = val
			}
		}
		total = append(total, values)
	}
	jsonString, _ := json.Marshal(total)
	return jsonString
}

func (output OutputArray) toCSVWithAllKeys() []byte {
	tableBuf := new(bytes.Buffer)
	t := output.buildTableWithAllKeys()
	t.SetOutputMirror(tableBuf)
	t.RenderCSV()
	return tableBuf.Bytes()
}

func (output OutputArray) toHTMLWithAllKeys() []byte {
	var baseTemplate string
	if output.Settings.ShouldAppend {
		originalfile, err := os.ReadFile(output.Settings.OutputFile)
		if err != nil {
			panic(err)
		}
		baseTemplate = string(originalfile)
	} else {
		b := template.New("base")
		b, _ = b.Parse(templates.BaseHTMLTemplate)
		baseBuf := new(bytes.Buffer)
		err := b.Execute(baseBuf, output)
		if err != nil {
			panic(err)
		}
		baseTemplate = baseBuf.String()
	}
	t := output.buildTableWithAllKeys()
	tableBuf := new(bytes.Buffer)
	t.SetOutputMirror(tableBuf)
	t.SetHTMLCSSClass("responstable")
	t.RenderHTML()
	tableBuf.Write([]byte("<div id='end'></div>")) // Add the placeholder
	return []byte(strings.Replace(baseTemplate, "<div id='end'></div>", tableBuf.String(), 1))
}

func (output OutputArray) toTableWithAllKeys() []byte {
	tableBuf := new(bytes.Buffer)
	if output.Settings.SeparateTables {
		tableBuf.WriteString("\n")
	}
	t := output.buildTableWithAllKeys()
	t.SetOutputMirror(tableBuf)
	t.SetStyle(output.Settings.TableStyle)
	t.Render()
	if output.Settings.SeparateTables {
		tableBuf.WriteString("\n")
	}
	return tableBuf.Bytes()
}

func (output OutputArray) toMarkdownWithAllKeys() []byte {
	t := output.buildTableWithAllKeys()
	tableBuf := new(bytes.Buffer)
	t.SetOutputMirror(tableBuf)
	t.RenderMarkdown()
	tableBuf.WriteString("\n")
	return tableBuf.Bytes()
}

func (output OutputArray) toYAMLWithAllKeys() []byte {
	jsonString, _ := yaml.Marshal(output.GetContentsMapRawWithAllKeys())
	return jsonString
}

func (output OutputArray) toMarkdownWithSections() []byte {
	var result bytes.Buffer

	for _, section := range savedSections {
		// Add section header
		if section.Title != "" {
			result.WriteString(fmt.Sprintf("## %s\n", section.Title))
		}

		// Create a temporary OutputArray for this section
		// Clear the title since we're adding it manually
		tempSettings := *output.Settings
		tempSettings.Title = ""
		tempOutput := OutputArray{
			Settings: &tempSettings,
			Contents: section.Contents,
			Keys:     section.Keys,
		}

		// Generate table for this section
		t := tempOutput.buildTable()
		tableBuf := new(bytes.Buffer)
		t.SetOutputMirror(tableBuf)
		t.RenderMarkdown()
		result.Write(tableBuf.Bytes())
		result.WriteString("\n")
	}

	return result.Bytes()
}

func (output OutputArray) toHTMLWithSections() []byte {
	var baseTemplate string
	if output.Settings.ShouldAppend {
		originalfile, err := os.ReadFile(output.Settings.OutputFile)
		if err != nil {
			panic(err)
		}
		baseTemplate = string(originalfile)
	} else {
		b := template.New("base")
		b, _ = b.Parse(templates.BaseHTMLTemplate)
		baseBuf := new(bytes.Buffer)
		err := b.Execute(baseBuf, output)
		if err != nil {
			panic(err)
		}
		baseTemplate = baseBuf.String()
	}

	var result bytes.Buffer

	for _, section := range savedSections {
		// Add section header
		if section.Title != "" {
			result.WriteString(fmt.Sprintf("<h2>%s</h2>\n", section.Title))
		}

		// Create a temporary OutputArray for this section
		// Clear the title since we're adding it manually
		tempSettings := *output.Settings
		tempSettings.Title = ""
		tempOutput := OutputArray{
			Settings: &tempSettings,
			Contents: section.Contents,
			Keys:     section.Keys,
		}

		// Generate table for this section
		t := tempOutput.buildTable()
		tableBuf := new(bytes.Buffer)
		t.SetOutputMirror(tableBuf)
		t.SetHTMLCSSClass("responstable")
		t.RenderHTML()
		result.Write(tableBuf.Bytes())
		result.WriteString("\n")
	}

	result.Write([]byte("<div id='end'></div>")) // Add the placeholder
	return []byte(strings.Replace(baseTemplate, "<div id='end'></div>", result.String(), 1))
}

func (output OutputArray) toCSVWithSections() []byte {
	var result bytes.Buffer

	for i, section := range savedSections {
		// Add section header as a comment or separator
		if section.Title != "" {
			result.WriteString(fmt.Sprintf("# %s\n", section.Title))
		}

		// Create a temporary OutputArray for this section
		// Clear the title since we're adding it manually
		tempSettings := *output.Settings
		tempSettings.Title = ""
		tempOutput := OutputArray{
			Settings: &tempSettings,
			Contents: section.Contents,
			Keys:     section.Keys,
		}

		// Generate CSV for this section
		sectionCSV := tempOutput.toCSV()
		result.Write(sectionCSV)

		// Add separator between sections
		if i < len(savedSections)-1 {
			result.WriteString("\n")
		}
	}

	return result.Bytes()
}

func (output OutputArray) toTableWithSections() []byte {
	var result bytes.Buffer

	for _, section := range savedSections {
		// Add section header
		if section.Title != "" {
			result.WriteString(fmt.Sprintf("\n%s\n", section.Title))
		}

		// Create a temporary OutputArray for this section
		// Clear the title since we're adding it manually
		tempSettings := *output.Settings
		tempSettings.Title = ""
		tempOutput := OutputArray{
			Settings: &tempSettings,
			Contents: section.Contents,
			Keys:     section.Keys,
		}

		// Generate table for this section
		sectionTable := tempOutput.toTable()
		result.Write(sectionTable)
	}

	return result.Bytes()
}

func (output OutputArray) toYAML() []byte {
	jsonString, _ := yaml.Marshal(output.GetContentsMapRaw())
	return jsonString
}

func (output OutputArray) toDot() []byte {
	cleanedlist := output.splitFromToValues()

	g := dot.NewGraph(dot.Directed)

	nodelist := make(map[string]dot.Node)

	// Step 1: Put all nodes in the list
	for _, cleaned := range cleanedlist {
		if _, ok := nodelist[cleaned.From]; !ok {
			node := g.Node(cleaned.From)
			nodelist[cleaned.From] = node
		}
	}

	// Step 2: Add all the edges/connections
	for _, cleaned := range cleanedlist {
		if cleaned.To != "" {
			g.Edge(nodelist[cleaned.From], nodelist[cleaned.To])
		}
	}
	return []byte(g.String())
}

func (output OutputArray) toMermaid() []byte {
	switch output.Settings.MermaidSettings.ChartType {
	case "":
		fallthrough
	case "flowchart":
		mermaid := mermaid.NewFlowchart(output.Settings.MermaidSettings)
		cleanedlist := output.splitFromToValues()
		// Add nodes
		for _, cleaned := range cleanedlist {
			mermaid.AddBasicNode(cleaned.From)
		}
		for _, cleaned := range cleanedlist {
			if cleaned.To != "" {
				mermaid.AddEdgeByNames(cleaned.From, cleaned.To)
			}
		}
		return []byte(mermaid.RenderString())
	case "piechart":
		mermaid := mermaid.NewPiechart(output.Settings.MermaidSettings)
		for _, holder := range output.Contents {
			label := output.toString(holder.Contents[output.Settings.FromToColumns.From])
			var value float64
			switch converted := holder.Contents[output.Settings.FromToColumns.To].(type) {
			case int:
				value = float64(converted)
			case int8:
				value = float64(converted)
			case int16:
				value = float64(converted)
			case int32:
				value = float64(converted)
			case int64:
				value = float64(converted)
			case uint:
				value = float64(converted)
			case uint8:
				value = float64(converted)
			case uint16:
				value = float64(converted)
			case uint32:
				value = float64(converted)
			case uint64:
				value = float64(converted)
			case float32:
				value = float64(converted)
			case float64:
				value = converted
			}
			mermaid.AddValue(label, value)
		}
		return []byte(mermaid.RenderString())
	case "ganttchart":
		chart := mermaid.NewGanttchart(output.Settings.MermaidSettings)
		chart.Title = output.Settings.Title
		section := chart.GetDefaultSection()
		for _, holder := range output.Contents {
			startdate := output.toString(holder.Contents[chart.Settings.GanttSettings.StartDateColumn])
			duration := output.toString(holder.Contents[chart.Settings.GanttSettings.DurationColumn])
			label := output.toString(holder.Contents[chart.Settings.GanttSettings.LabelColumn])
			status := output.toString(holder.Contents[chart.Settings.GanttSettings.StatusColumn])
			section.AddTask(label, startdate, duration, status)
		}
		return []byte(chart.RenderString())
	}
	return []byte("")
}

type fromToValues struct {
	From string
	To   string
}

func (output OutputArray) splitFromToValues() []fromToValues {
	resultList := make([]fromToValues, 0)
	for _, holder := range output.Contents {
		for _, tovalue := range strings.Split(output.toString(holder.Contents[output.Settings.FromToColumns.To]), ",") {
			values := fromToValues{
				From: output.toString(holder.Contents[output.Settings.FromToColumns.From]),
				To:   tovalue,
			}
			resultList = append(resultList, values)
		}
	}
	return resultList
}

func (output *OutputArray) AddHeader(header string) {
	switch output.Settings.OutputFormat {
	case "html":
		id := slug.Make(header)
		buffer.Write([]byte(fmt.Sprintf("<h2 id='%s'>%s</h2>\n", id, header)))
		toc = append(toc, fmt.Sprintf("<a href='#%s'>%s</a>", id, header))
	case "table":
		buffer.Write([]byte(fmt.Sprintf("\n%s\n", header)))
	case "markdown":
		buffer.Write([]byte(fmt.Sprintf("## %s\n", header)))
		id := slug.Make(header)
		toc = append(toc, fmt.Sprintf("[%s](#%s)", header, id))
	}
}

func (output *OutputArray) AddToBuffer() {
	// Save raw data for dual output format support before clearing contents
	if output.Contents != nil {
		var sectionContents []OutputHolder
		for _, content := range output.Contents {
			// Create a deep copy of the content
			copiedContent := OutputHolder{
				Contents: make(map[string]interface{}),
			}
			for k, v := range content.Contents {
				copiedContent.Contents[k] = v
				// Collect all unique keys for cross-section JSON generation
				keyExists := false
				for _, existingKey := range savedAllKeys {
					if existingKey == k {
						keyExists = true
						break
					}
				}
				if !keyExists {
					savedAllKeys = append(savedAllKeys, k)
				}
			}
			savedRawData = append(savedRawData, copiedContent)
			sectionContents = append(sectionContents, copiedContent)
		}

		// Save section structure for table-based formats
		// Extract the actual keys from this section's data
		sectionKeys := make(map[string]bool)
		for _, content := range sectionContents {
			for key := range content.Contents {
				sectionKeys[key] = true
			}
		}

		// Convert map to slice
		var keys []string
		for key := range sectionKeys {
			keys = append(keys, key)
		}

		section := SavedSection{
			Title:    output.Settings.Title,
			Keys:     keys,
			Contents: sectionContents,
		}
		savedSections = append(savedSections, section)
	}

	switch output.Settings.OutputFormat {
	case "csv":
		buffer.Write(output.toCSV())
	case "html":
		buffer.Write(output.HtmlTableOnly())
	case "table":
		buffer.Write(output.toTable())
	case "markdown":
		buffer.Write(output.toMarkdown())
	case "mermaid":
		// if output.Settings.FromToColumns == nil {
		// 	log.Fatal("This command doesn't currently support the mermaid output format")
		// }
		buffer.Write(output.toMermaid())
	case "drawio":
		// if !output.Settings.DrawIOHeader.IsSet() {
		// 	log.Fatal("This command doesn't currently support the drawio output format")
		// }
		// drawio.CreateCSV(output.Settings.DrawIOHeader, output.Keys, output.GetContentsMap(), output.Settings.OutputFile)
	case "dot":
		if output.Settings.FromToColumns == nil {
			log.Fatal("This command doesn't currently support the dot output format")
		}
		buffer.Write(output.toDot())
	default:
		buffer.Write(output.toJSON())
	}
	// Clear contents after adding to buffer to prepare for next section
	output.Contents = nil
}

func (output OutputArray) bufferToHTML() []byte {
	var baseTemplate string
	if output.Settings.ShouldAppend {
		originalfile, err := os.ReadFile(output.Settings.OutputFile)
		if err != nil {
			panic(err)
		}
		baseTemplate = string(originalfile)
	} else {
		b := template.New("base")
		b, _ = b.Parse(templates.BaseHTMLTemplate)
		baseBuf := new(bytes.Buffer)
		err := b.Execute(baseBuf, output)
		if err != nil {
			panic(err)
		}
		baseTemplate = baseBuf.String()
		tocstring := ""
		if output.Settings.Title != "" {
			tocstring = fmt.Sprintf("<h1>%s</h1>", output.Settings.Title)
		}
		if output.Settings.HasTOC {
			tocstring += "<h2>Table of Contents</h2>\n<ul id='tableofcontent'>\n"
			for _, item := range toc {
				tocstring += fmt.Sprintf("<li>%s</li>\n", item)
			}
			tocstring += "</ul>"
		}
		tocstring += "\n<div id='end'></div>"
		baseTemplate = strings.Replace(baseTemplate, "<div id='end'></div>", tocstring, 1)
	}
	buffer.Write([]byte("<div id='end'></div>")) // Add the placeholder
	return []byte(strings.Replace(baseTemplate, "<div id='end'></div>", buffer.String(), 1))

}

func (output OutputArray) bufferToMarkdown() []byte {
	headerstring := ""
	if len(output.Settings.FrontMatter) != 0 {
		headerstring = "---\n"
		for key, value := range output.Settings.FrontMatter {
			headerstring += fmt.Sprintf("%s: %s\n", key, value)
		}
		headerstring += "---\n"
	}
	if output.Settings.Title != "" {
		headerstring += fmt.Sprintf("# %s\n\n", output.Settings.Title)
	}
	if output.Settings.HasTOC {
		headerstring += "## Table of Contents\n"
		for _, item := range toc {
			headerstring += fmt.Sprintf("* %s \n", item)
		}
		headerstring += "\n"
	}
	return []byte(headerstring + buffer.String())
}

func (output OutputArray) toHTML() []byte {
	var baseTemplate string
	if output.Settings.ShouldAppend {
		originalfile, err := os.ReadFile(output.Settings.OutputFile)
		if err != nil {
			panic(err)
		}
		baseTemplate = string(originalfile)
	} else {
		b := template.New("base")
		b, _ = b.Parse(templates.BaseHTMLTemplate)
		baseBuf := new(bytes.Buffer)
		err := b.Execute(baseBuf, output)
		if err != nil {
			panic(err)
		}
		baseTemplate = baseBuf.String()
	}
	t := output.buildTable()
	tableBuf := new(bytes.Buffer)
	t.SetOutputMirror(tableBuf)
	t.SetHTMLCSSClass("responstable")
	t.RenderHTML()
	tableBuf.Write([]byte("<div id='end'></div>")) // Add the placeholder
	return []byte(strings.Replace(baseTemplate, "<div id='end'></div>", tableBuf.String(), 1))
}

// HtmlTableOnly returns a byte array containing an HTML table of the OutputArray
func (output OutputArray) HtmlTableOnly() []byte {
	t := output.buildTable()
	tableBuf := new(bytes.Buffer)
	t.SetOutputMirror(tableBuf)
	t.SetHTMLCSSClass("responstable")
	t.RenderHTML()
	return tableBuf.Bytes()
}

func (output OutputArray) toTable() []byte {
	tableBuf := new(bytes.Buffer)
	if output.Settings.SeparateTables {
		tableBuf.WriteString("\n")
	}
	t := output.buildTable()
	t.SetOutputMirror(tableBuf)
	t.SetStyle(output.Settings.TableStyle)
	t.Render()
	if output.Settings.SeparateTables {
		tableBuf.WriteString("\n")
	}
	return tableBuf.Bytes()
}

func (output OutputArray) toMarkdown() []byte {
	t := output.buildTable()
	tableBuf := new(bytes.Buffer)
	t.SetOutputMirror(tableBuf)
	t.RenderMarkdown()
	tableBuf.WriteString("\n")
	return tableBuf.Bytes()
}

func (output OutputArray) buildTable() table.Writer {
	t := table.NewWriter()
	if output.Settings.Title != "" {
		// Ugly hack because go-pretty uses a h1 (#) for the table title in Markdown
		if (output.Settings.OutputFormat == "markdown") && buffer.Len() != 0 {
			buffer.WriteString(fmt.Sprintf("#### %s\n\n", output.Settings.Title))
		} else {
			t.SetTitle(output.Settings.Title)
		}
	}
	var target io.Writer
	// var err error
	// pretend it's stdout when writing a bucket to prevent files from being created
	if output.Settings.OutputFile == "" || output.Settings.S3Bucket.Bucket != "" {
		target = os.Stdout
	} else {
		//Always create if append flag isn't provided
		if !output.Settings.ShouldAppend {
			target, _ = os.Create(output.Settings.OutputFile)
		} else {
			target, _ = os.OpenFile(output.Settings.OutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		}
	}
	t.SetOutputMirror(target)
	t.AppendHeader(output.KeysAsInterface())
	for _, cont := range output.ContentsAsInterfaces() {
		t.AppendRow(cont)
	}
	columnConfigs := make([]table.ColumnConfig, 0)
	for _, key := range output.Keys {
		columnConfig := table.ColumnConfig{
			Name:     key,
			WidthMin: 6,
			WidthMax: output.Settings.TableMaxColumnWidth,
		}
		columnConfigs = append(columnConfigs, columnConfig)
	}
	t.SetColumnConfigs(columnConfigs)
	return t
}

func (output OutputArray) buildTableWithAllKeys() table.Writer {
	t := table.NewWriter()
	if output.Settings.Title != "" {
		// Ugly hack because go-pretty uses a h1 (#) for the table title in Markdown
		if (output.Settings.OutputFormat == "markdown") && buffer.Len() != 0 {
			buffer.WriteString(fmt.Sprintf("#### %s\n\n", output.Settings.Title))
		} else {
			t.SetTitle(output.Settings.Title)
		}
	}
	var target io.Writer
	// var err error
	// pretend it's stdout when writing a bucket to prevent files from being created
	if output.Settings.OutputFile == "" || output.Settings.S3Bucket.Bucket != "" {
		target = os.Stdout
	} else {
		//Always create if append flag isn't provided
		if !output.Settings.ShouldAppend {
			target, _ = os.Create(output.Settings.OutputFile)
		} else {
			target, _ = os.OpenFile(output.Settings.OutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		}
	}
	t.SetOutputMirror(target)
	t.AppendHeader(output.KeysAsInterfaceWithAllKeys())
	for _, cont := range output.ContentsAsInterfacesWithAllKeys() {
		t.AppendRow(cont)
	}
	columnConfigs := make([]table.ColumnConfig, 0)
	for _, key := range savedAllKeys {
		columnConfig := table.ColumnConfig{
			Name:     key,
			WidthMin: 6,
			WidthMax: output.Settings.TableMaxColumnWidth,
		}
		columnConfigs = append(columnConfigs, columnConfig)
	}
	t.SetColumnConfigs(columnConfigs)
	return t
}

func (output *OutputArray) KeysAsInterface() []interface{} {
	b := make([]interface{}, len(output.Keys))
	for i := range output.Keys {
		b[i] = output.Keys[i]
	}

	return b
}

func (output *OutputArray) KeysAsInterfaceWithAllKeys() []interface{} {
	b := make([]interface{}, len(savedAllKeys))
	for i := range savedAllKeys {
		b[i] = savedAllKeys[i]
	}

	return b
}

func (output *OutputArray) ContentsAsInterfaces() [][]interface{} {
	total := make([][]interface{}, 0)

	for _, holder := range output.Contents {
		values := make([]interface{}, len(output.Keys))
		for counter, key := range output.Keys {
			if val, ok := holder.Contents[key]; ok {
				values[counter] = output.toString(val)
			}
		}
		total = append(total, values)
	}
	return total
}

func (output *OutputArray) ContentsAsInterfacesWithAllKeys() [][]interface{} {
	total := make([][]interface{}, 0)

	for _, holder := range output.Contents {
		values := make([]interface{}, len(savedAllKeys))
		for counter, key := range savedAllKeys {
			if val, ok := holder.Contents[key]; ok {
				values[counter] = output.toString(val)
			}
		}
		total = append(total, values)
	}
	return total
}

// PrintByteSlice prints the provided contents to stdout or the provided filepath
func PrintByteSlice(contents []byte, outputFile string, targetBucket S3Output) (err error) {
	stopActiveProgress()
	var target io.Writer
	// Remove the bash colours from output files
	if outputFile != "" {
		re := regexp.MustCompile(`\x1B\[([0-9]{1,3}(;[0-9]{1,3})*)?[mGK]`) // source: https://stackoverflow.com/questions/17998978/removing-colors-from-output
		contents = re.ReplaceAll(contents, []byte(""))
	}
	if targetBucket.Bucket != "" {
		s3params := s3.PutObjectInput{
			Bucket: &targetBucket.Bucket,
			Key:    &targetBucket.Path,
			Body:   bytes.NewReader(contents),
		}
		_, err = targetBucket.S3Client.PutObject(context.TODO(), &s3params)
		return err
	}
	if outputFile == "" {
		target = os.Stdout
	} else {
		f, errCreate := os.Create(outputFile)
		if errCreate != nil {
			return errCreate
		}
		target = f
		defer func() {
			if cerr := f.Close(); err == nil && cerr != nil {
				err = cerr
			}
		}()
	}
	w := bufio.NewWriter(target)
	_, err = w.Write(contents)
	if err != nil {
		return err
	}
	err = w.Flush()
	return err
}

// AddHolder adds the provided OutputHolder to the OutputArray
func (output *OutputArray) AddHolder(holder OutputHolder) {
	var contents []OutputHolder
	if output.Contents != nil {
		contents = output.Contents
	}
	contents = append(contents, holder)
	if output.Settings.SortKey != "" {
		sort.Slice(contents,
			func(i, j int) bool {
				return output.toString(contents[i].Contents[output.Settings.SortKey]) < output.toString(contents[j].Contents[output.Settings.SortKey])
			})
	}
	output.Contents = contents
}

// AddContents adds the provided map[string]interface{} to the OutputHolder and that in turn to the OutputArray
func (output *OutputArray) AddContents(contents map[string]interface{}) {
	holder := OutputHolder{Contents: contents}
	output.AddHolder(holder)
}

// toString converts the provided interface value into a string.
func (output *OutputArray) toString(val interface{}) string {
	switch converted := val.(type) {
	case []string:
		return strings.Join(converted, output.Settings.GetSeparator())
	case bool:
		if converted {
			if output.Settings.UseEmoji {
				return "✅"
			}
			return "Yes"
		}
		if output.Settings.UseEmoji {
			return "❌"
		}
		return "No"
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return formatNumber(val)
	}
	return fmt.Sprintf("%v", val)
}

// formatNumber provides consistent string formatting for numeric values.
func formatNumber(val interface{}) string {
	return fmt.Sprintf("%v", val)
}
