package format

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/ArjenSchwarz/go-output/errors"
	"github.com/ArjenSchwarz/go-output/mermaid"
	"github.com/emicklei/dot"
	"gopkg.in/yaml.v3"
)

const (
	chartTypeFlowchart  = "flowchart"
	chartTypePiechart   = "piechart"
	chartTypeGanttchart = "ganttchart"
)

// Enhanced format-specific error handling methods

// toJSONWithErrorHandling generates JSON with comprehensive error handling
func (output *OutputArray) toJSONWithErrorHandling() ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			panic(errors.NewProcessingError(
				errors.ErrTemplateRender,
				fmt.Sprintf("JSON generation panicked: %v", r),
			).WithSeverity(errors.SeverityFatal))
		}
	}()

	// Validate data before marshaling
	if err := output.validateJSONData(); err != nil {
		return nil, err
	}

	// Get raw contents
	contents := output.GetContentsMapRaw()

	// Attempt to marshal
	jsonBytes, err := json.Marshal(contents)
	if err != nil {
		return nil, errors.NewProcessingError(
			errors.ErrTemplateRender,
			"Failed to marshal data to JSON",
		).WithContext(errors.ErrorContext{
			Operation: "json_marshal",
			Field:     "contents",
			Value:     fmt.Sprintf("%T", contents),
		}).WithSuggestions(
			"Check for unserializable data types (functions, channels, etc.)",
			"Consider using custom JSON marshaling for complex types",
		).Wrap(err)
	}

	return jsonBytes, nil
}

// toYAMLWithErrorHandling generates YAML with comprehensive error handling
func (output *OutputArray) toYAMLWithErrorHandling() ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			panic(errors.NewProcessingError(
				errors.ErrTemplateRender,
				fmt.Sprintf("YAML generation panicked: %v", r),
			).WithSeverity(errors.SeverityFatal))
		}
	}()

	// Validate data before marshaling
	if err := output.validateYAMLData(); err != nil {
		return nil, err
	}

	// Get raw contents
	contents := output.GetContentsMapRaw()

	// Attempt to marshal
	yamlBytes, err := yaml.Marshal(contents)
	if err != nil {
		return nil, errors.NewProcessingError(
			errors.ErrTemplateRender,
			"Failed to marshal data to YAML",
		).WithContext(errors.ErrorContext{
			Operation: "yaml_marshal",
			Field:     "contents",
			Value:     fmt.Sprintf("%T", contents),
		}).WithSuggestions(
			"Check for unserializable data types",
			"Ensure all data is YAML-compatible",
		).Wrap(err)
	}

	return yamlBytes, nil
}

// toCSVWithErrorHandling generates CSV with comprehensive error handling
func (output *OutputArray) toCSVWithErrorHandling() ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			panic(errors.NewProcessingError(
				errors.ErrTemplateRender,
				fmt.Sprintf("CSV generation panicked: %v", r),
			).WithSeverity(errors.SeverityFatal))
		}
	}()

	// Validate CSV data
	if err := output.validateCSVData(); err != nil {
		return nil, err
	}

	// Use existing CSV generation but with error context
	result := output.toCSV()

	// Validate result
	if len(result) == 0 {
		return nil, errors.NewProcessingError(
			errors.ErrTemplateRender,
			"CSV generation produced empty output",
		).WithContext(errors.ErrorContext{
			Operation: "csv_generation",
			Field:     "output",
			Value:     len(output.Contents),
		}).WithSuggestions(
			"Check that data contains valid rows",
			"Verify that keys are properly set",
		)
	}

	return result, nil
}

// toTableWithErrorHandling generates table with comprehensive error handling
func (output *OutputArray) toTableWithErrorHandling() ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			panic(errors.NewProcessingError(
				errors.ErrTemplateRender,
				fmt.Sprintf("Table generation panicked: %v", r),
			).WithSeverity(errors.SeverityFatal))
		}
	}()

	// Validate table data
	if err := output.validateTableData(); err != nil {
		return nil, err
	}

	// Check column width settings
	if output.Settings.TableMaxColumnWidth < 1 {
		return nil, errors.NewValidationError(
			errors.ErrInvalidConfiguration,
			"Table maximum column width must be at least 1",
		).WithContext(errors.ErrorContext{
			Operation: "table_validation",
			Field:     "TableMaxColumnWidth",
			Value:     output.Settings.TableMaxColumnWidth,
		}).WithSuggestions(
			"Set TableMaxColumnWidth to a positive value",
		)
	}

	// Use existing table generation
	result := output.toTable()

	// Validate result
	if len(result) == 0 {
		return nil, errors.NewProcessingError(
			errors.ErrTemplateRender,
			"Table generation produced empty output",
		).WithContext(errors.ErrorContext{
			Operation: "table_generation",
		})
	}

	return result, nil
}

// toHTMLWithErrorHandling generates HTML with comprehensive error handling
func (output *OutputArray) toHTMLWithErrorHandling() ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			panic(errors.NewProcessingError(
				errors.ErrTemplateRender,
				fmt.Sprintf("HTML generation panicked: %v", r),
			).WithSeverity(errors.SeverityFatal))
		}
	}()

	// Validate HTML data
	if err := output.validateHTMLData(); err != nil {
		return nil, err
	}

	// Use existing HTML generation
	result := output.toHTML()

	// Validate result
	if len(result) == 0 {
		return nil, errors.NewProcessingError(
			errors.ErrTemplateRender,
			"HTML generation produced empty output",
		).WithContext(errors.ErrorContext{
			Operation: "html_generation",
		})
	}

	return result, nil
}

// toMermaidWithErrorHandling generates Mermaid with comprehensive error handling
func (output *OutputArray) toMermaidWithErrorHandling() ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			panic(errors.NewProcessingError(
				errors.ErrTemplateRender,
				fmt.Sprintf("Mermaid generation panicked: %v", r),
			).WithSeverity(errors.SeverityFatal))
		}
	}()

	// Validate mermaid configuration
	if err := output.validateMermaidConfiguration(); err != nil {
		return nil, err
	}

	var result []byte

	switch output.Settings.MermaidSettings.ChartType {
	case "", chartTypeFlowchart:
		result = output.generateFlowchartWithValidation()
	case chartTypePiechart:
		data, err := output.generatePiechartWithValidation()
		if err != nil {
			return nil, err
		}
		result = data
	case chartTypeGanttchart:
		data, err := output.generateGanttchartWithValidation()
		if err != nil {
			return nil, err
		}
		result = data
	default:
		return nil, errors.NewValidationError(
			errors.ErrInvalidConfiguration,
			fmt.Sprintf("Unsupported Mermaid chart type: %s", output.Settings.MermaidSettings.ChartType),
		).WithContext(errors.ErrorContext{
			Operation: "mermaid_chart_type_validation",
			Field:     "ChartType",
			Value:     output.Settings.MermaidSettings.ChartType,
		}).WithSuggestions(
			"Use supported chart types: flowchart, piechart, ganttchart",
			"Check MermaidSettings.ChartType configuration",
		)
	}

	// Validate result
	if len(result) == 0 {
		return nil, errors.NewProcessingError(
			errors.ErrTemplateRender,
			"Mermaid generation produced empty output",
		).WithContext(errors.ErrorContext{
			Operation: "mermaid_generation",
			Field:     "chart_type",
			Value:     output.Settings.MermaidSettings.ChartType,
		})
	}

	return result, nil
}

// toDotWithErrorHandling generates DOT with comprehensive error handling
func (output *OutputArray) toDotWithErrorHandling() ([]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			panic(errors.NewProcessingError(
				errors.ErrTemplateRender,
				fmt.Sprintf("DOT generation panicked: %v", r),
			).WithSeverity(errors.SeverityFatal))
		}
	}()

	// Validate DOT data
	if err := output.validateDotData(); err != nil {
		return nil, err
	}

	// Validate from-to columns configuration
	if output.Settings.FromToColumns == nil {
		return nil, errors.NewValidationError(
			errors.ErrMissingRequired,
			"DOT format requires FromToColumns configuration",
		).WithSuggestions(
			"Use AddFromToColumns() to set source and target columns",
		)
	}

	cleanedlist := output.splitFromToValues()

	// Validate that we have data to process
	if len(cleanedlist) == 0 {
		return nil, errors.NewValidationError(
			errors.ErrEmptyDataset,
			"No valid from-to relationships found for DOT generation",
		).WithContext(errors.ErrorContext{
			Operation: "dot_data_validation",
			Field:     "from_to_relationships",
			Value:     len(cleanedlist),
		}).WithSuggestions(
			"Ensure data contains valid from-to column values",
			"Check FromToColumns configuration matches data structure",
		)
	}

	// Create graph
	g := dot.NewGraph(dot.Directed)
	nodelist := make(map[string]dot.Node)

	// Validate and add nodes
	for i, cleaned := range cleanedlist {
		fromValue := strings.TrimSpace(cleaned.From)
		if fromValue == "" {
			return nil, errors.NewValidationError(
				errors.ErrConstraintViolation,
				fmt.Sprintf("Empty 'from' value at row %d", i),
			).WithContext(errors.ErrorContext{
				Operation: "dot_node_validation",
				Field:     "from_value",
				Index:     i,
				Value:     fromValue,
			}).WithSuggestions(
				"Ensure all 'from' values are non-empty",
				"Check data for missing or null values",
			)
		}

		if _, ok := nodelist[fromValue]; !ok {
			node := g.Node(fromValue)
			nodelist[fromValue] = node
		}
	}

	// Add edges
	for _, cleaned := range cleanedlist {
		toValue := strings.TrimSpace(cleaned.To)
		if toValue != "" {
			fromValue := strings.TrimSpace(cleaned.From)

			// Ensure target node exists
			if _, ok := nodelist[toValue]; !ok {
				node := g.Node(toValue)
				nodelist[toValue] = node
			}

			g.Edge(nodelist[fromValue], nodelist[toValue])
		}
	}

	result := []byte(g.String())

	// Validate result
	if len(result) == 0 {
		return nil, errors.NewProcessingError(
			errors.ErrTemplateRender,
			"DOT generation produced empty output",
		).WithContext(errors.ErrorContext{
			Operation: "dot_generation",
		})
	}

	return result, nil
}

// Data validation methods

// validateJSONData validates data for JSON marshaling
func (output *OutputArray) validateJSONData() error {
	if len(output.Contents) == 0 {
		return errors.NewValidationError(
			errors.ErrEmptyDataset,
			"No data available for JSON generation",
		).WithSuggestions(
			"Add data using AddContents() or AddHolder()",
		)
	}

	// Check for unserializable types
	for rowIndex, holder := range output.Contents {
		for key, value := range holder.Contents {
			if err := validateJSONSerializable(value); err != nil {
				return errors.NewValidationError(
					errors.ErrInvalidDataType,
					fmt.Sprintf("Unserializable data type in row %d, field %s", rowIndex, key),
				).WithContext(errors.ErrorContext{
					Operation: "json_data_validation",
					Field:     key,
					Index:     rowIndex,
					Value:     fmt.Sprintf("%T", value),
				}).WithSuggestions(
					"Remove or convert unserializable data types (functions, channels, etc.)",
					"Use string representation for complex types",
				).Wrap(err)
			}
		}
	}

	return nil
}

// validateYAMLData validates data for YAML marshaling
func (output *OutputArray) validateYAMLData() error {
	if len(output.Contents) == 0 {
		return errors.NewValidationError(
			errors.ErrEmptyDataset,
			"No data available for YAML generation",
		).WithSuggestions(
			"Add data using AddContents() or AddHolder()",
		)
	}

	// YAML has similar restrictions to JSON
	return output.validateJSONData()
}

// validateCSVData validates data for CSV generation
func (output *OutputArray) validateCSVData() error {
	if len(output.Keys) == 0 {
		return errors.NewValidationError(
			errors.ErrMissingRequired,
			"No column keys defined for CSV generation",
		).WithSuggestions(
			"Set column keys using Keys field",
		)
	}

	// CSV can handle most data types through string conversion
	return nil
}

// validateTableData validates data for table generation
func (output *OutputArray) validateTableData() error {
	if len(output.Keys) == 0 {
		return errors.NewValidationError(
			errors.ErrMissingRequired,
			"No column keys defined for table generation",
		).WithSuggestions(
			"Set column keys using Keys field",
		)
	}

	// Check for excessively long content that might break table formatting
	for i, holder := range output.Contents {
		for key, value := range holder.Contents {
			if str := output.toString(value); len(str) > 10000 {
				return errors.NewValidationError(
					errors.ErrConstraintViolation,
					fmt.Sprintf("Excessively long content in row %d, field %s (%d characters)", i, key, len(str)),
				).WithContext(errors.ErrorContext{
					Operation: "table_content_validation",
					Field:     key,
					Index:     i,
					Value:     len(str),
				}).WithSuggestions(
					"Truncate long content for table display",
					"Consider using a different output format for large content",
				)
			}
		}
	}

	return nil
}

// validateHTMLData validates data for HTML generation
func (output *OutputArray) validateHTMLData() error {
	// HTML uses table generation, so validate table data
	return output.validateTableData()
}

// validateMermaidConfiguration validates mermaid configuration
func (output *OutputArray) validateMermaidConfiguration() error {
	if output.Settings.MermaidSettings == nil {
		output.Settings.MermaidSettings = &mermaid.Settings{ChartType: "flowchart"}
	}

	chartType := output.Settings.MermaidSettings.ChartType
	if chartType == "" {
		chartType = chartTypeFlowchart
	}

	switch chartType {
	case "flowchart":
		if output.Settings.FromToColumns == nil {
			return errors.NewValidationError(
				errors.ErrMissingRequired,
				"Flowchart requires FromToColumns configuration",
			).WithSuggestions(
				"Use AddFromToColumns() to set source and target columns",
			)
		}
	case chartTypePiechart:
		if output.Settings.FromToColumns == nil {
			return errors.NewValidationError(
				errors.ErrMissingRequired,
				"Piechart requires FromToColumns configuration (label and value columns)",
			).WithSuggestions(
				"Use AddFromToColumns() to set label and value columns",
			)
		}
	case chartTypeGanttchart:
		if output.Settings.MermaidSettings.GanttSettings == nil {
			return errors.NewValidationError(
				errors.ErrMissingRequired,
				"Gantt chart requires GanttSettings configuration",
			).WithSuggestions(
				"Configure MermaidSettings.GanttSettings with required columns",
			)
		}
	}

	return nil
}

// validateDotData validates data for DOT generation
func (output *OutputArray) validateDotData() error {
	if output.Settings.FromToColumns == nil {
		return errors.NewValidationError(
			errors.ErrMissingRequired,
			"DOT format requires FromToColumns configuration",
		).WithSuggestions(
			"Use AddFromToColumns() to set source and target columns",
		)
	}

	if len(output.Contents) == 0 {
		return errors.NewValidationError(
			errors.ErrEmptyDataset,
			"No data available for DOT generation",
		).WithSuggestions(
			"Add data using AddContents() or AddHolder()",
		)
	}

	return nil
}

// Helper methods

// validateJSONSerializable checks if a value can be serialized to JSON
func validateJSONSerializable(value interface{}) error {
	if value == nil {
		return nil
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Func, reflect.Chan, reflect.UnsafePointer:
		return fmt.Errorf("unserializable type: %T", value)
	case reflect.Map:
		// Check map keys and values
		for _, key := range v.MapKeys() {
			if err := validateJSONSerializable(key.Interface()); err != nil {
				return err
			}
			if err := validateJSONSerializable(v.MapIndex(key).Interface()); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		// Check slice/array elements
		for i := 0; i < v.Len(); i++ {
			if err := validateJSONSerializable(v.Index(i).Interface()); err != nil {
				return err
			}
		}
	case reflect.Struct:
		// Check struct fields (JSON marshaling handles this, but we can add specific checks)
		if !v.CanInterface() {
			return fmt.Errorf("unexported struct fields cannot be serialized")
		}
	}

	return nil
}

// generateFlowchartWithValidation generates flowchart with validation
func (output *OutputArray) generateFlowchartWithValidation() []byte {
	mermaidChart := mermaid.NewFlowchart(output.Settings.MermaidSettings)
	cleanedlist := output.splitFromToValues()

	// Add nodes
	for _, cleaned := range cleanedlist {
		if strings.TrimSpace(cleaned.From) != "" {
			mermaidChart.AddBasicNode(cleaned.From)
		}
	}

	// Add edges
	for _, cleaned := range cleanedlist {
		if cleaned.To != "" && strings.TrimSpace(cleaned.From) != "" {
			mermaidChart.AddEdgeByNames(cleaned.From, cleaned.To)
		}
	}

	return []byte(mermaidChart.RenderString())
}

// generatePiechartWithValidation generates piechart with validation
func (output *OutputArray) generatePiechartWithValidation() ([]byte, error) {
	mermaidChart := mermaid.NewPiechart(output.Settings.MermaidSettings)

	for i, holder := range output.Contents {
		label := output.toString(holder.Contents[output.Settings.FromToColumns.From])

		// Validate and convert value
		valueInterface, exists := holder.Contents[output.Settings.FromToColumns.To]
		if !exists {
			return nil, errors.NewValidationError(
				errors.ErrMissingColumn,
				fmt.Sprintf("Missing value column '%s' in row %d", output.Settings.FromToColumns.To, i),
			).WithContext(errors.ErrorContext{
				Operation: "piechart_value_validation",
				Field:     output.Settings.FromToColumns.To,
				Index:     i,
			})
		}

		var value float64
		var err error

		switch converted := valueInterface.(type) {
		case float64:
			value = converted
		case float32:
			value = float64(converted)
		case int, int8, int16, int32, int64:
			value = float64(reflect.ValueOf(converted).Int())
		case uint, uint8, uint16, uint32, uint64:
			value = float64(reflect.ValueOf(converted).Uint())
		case string:
			value, err = strconv.ParseFloat(converted, 64)
			if err != nil {
				return nil, errors.NewValidationError(
					errors.ErrInvalidDataType,
					fmt.Sprintf("Cannot convert value '%s' to number in row %d", converted, i),
				).WithContext(errors.ErrorContext{
					Operation: "piechart_value_conversion",
					Field:     output.Settings.FromToColumns.To,
					Index:     i,
					Value:     converted,
				}).WithSuggestions(
					"Ensure value column contains numeric data",
					"Convert string values to numbers before adding to data",
				).Wrap(err)
			}
		default:
			return nil, errors.NewValidationError(
				errors.ErrInvalidDataType,
				fmt.Sprintf("Unsupported value type %T in row %d", valueInterface, i),
			).WithContext(errors.ErrorContext{
				Operation: "piechart_value_type_validation",
				Field:     output.Settings.FromToColumns.To,
				Index:     i,
				Value:     fmt.Sprintf("%T", valueInterface),
			}).WithSuggestions(
				"Use numeric types (int, float64) for piechart values",
			)
		}

		mermaidChart.AddValue(label, value)
	}

	return []byte(mermaidChart.RenderString()), nil
}

// generateGanttchartWithValidation generates gantt chart with validation
func (output *OutputArray) generateGanttchartWithValidation() ([]byte, error) {
	chart := mermaid.NewGanttchart(output.Settings.MermaidSettings)
	chart.Title = output.Settings.Title

	ganttSettings := output.Settings.MermaidSettings.GanttSettings
	if ganttSettings == nil {
		return nil, errors.NewValidationError(
			errors.ErrMissingRequired,
			"GanttSettings is required for gantt chart generation",
		).WithSuggestions(
			"Configure MermaidSettings.GanttSettings with required columns",
		)
	}

	section := chart.GetDefaultSection()

	for i, holder := range output.Contents {
		// Validate required fields
		requiredFields := map[string]string{
			"StartDateColumn": ganttSettings.StartDateColumn,
			"DurationColumn":  ganttSettings.DurationColumn,
			"LabelColumn":     ganttSettings.LabelColumn,
			"StatusColumn":    ganttSettings.StatusColumn,
		}

		for fieldName, columnName := range requiredFields {
			if columnName == "" {
				return nil, errors.NewValidationError(
					errors.ErrMissingRequired,
					fmt.Sprintf("GanttSettings.%s is required", fieldName),
				).WithSuggestions(
					"Configure all required gantt chart columns",
				)
			}

			if _, exists := holder.Contents[columnName]; !exists {
				return nil, errors.NewValidationError(
					errors.ErrMissingColumn,
					fmt.Sprintf("Missing required column '%s' in row %d", columnName, i),
				).WithContext(errors.ErrorContext{
					Operation: "gantt_column_validation",
					Field:     columnName,
					Index:     i,
				})
			}
		}

		startdate := output.toString(holder.Contents[ganttSettings.StartDateColumn])
		duration := output.toString(holder.Contents[ganttSettings.DurationColumn])
		label := output.toString(holder.Contents[ganttSettings.LabelColumn])
		status := output.toString(holder.Contents[ganttSettings.StatusColumn])

		section.AddTask(label, startdate, duration, status)
	}

	return []byte(chart.RenderString()), nil
}
