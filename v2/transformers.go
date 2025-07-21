package output

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

// EmojiTransformer converts text-based indicators to emoji equivalents
type EmojiTransformer struct{}

// Name returns the transformer name
func (e *EmojiTransformer) Name() string {
	return "emoji"
}

// Priority returns the transformer priority (lower = earlier)
func (e *EmojiTransformer) Priority() int {
	return 100
}

// CanTransform checks if this transformer applies to the given format
func (e *EmojiTransformer) CanTransform(format string) bool {
	// Apply emoji transformation to all text-based formats
	return format == FormatTable || format == FormatMarkdown || format == FormatHTML || format == FormatCSV
}

// Transform converts text indicators to emoji
func (e *EmojiTransformer) Transform(ctx context.Context, input []byte, format string) ([]byte, error) {
	output := string(input)

	// Common emoji substitutions based on v1 patterns
	replacements := map[string]string{
		"!!":  "🚨",
		"OK":  "✅",
		"Yes": "✅",
		"No":  "❌",
		"":    "ℹ️", // For info contexts
	}

	// Apply replacements
	for text, emoji := range replacements {
		if text != "" { // Skip empty string for now
			output = strings.ReplaceAll(output, text, emoji)
		}
	}

	// Handle boolean-like patterns
	output = regexp.MustCompile(`\btrue\b`).ReplaceAllString(output, "✅")
	output = regexp.MustCompile(`\bfalse\b`).ReplaceAllString(output, "❌")

	return []byte(output), nil
}

// ColorTransformer adds ANSI color codes to output
type ColorTransformer struct {
	scheme ColorScheme
}

// ColorScheme defines color configuration
type ColorScheme struct {
	Success string // Color for positive/success values
	Warning string // Color for warning values
	Error   string // Color for error/failure values
	Info    string // Color for informational values
}

// DefaultColorScheme returns a default color scheme
func DefaultColorScheme() ColorScheme {
	return ColorScheme{
		Success: "green",
		Warning: "yellow",
		Error:   "red",
		Info:    "blue",
	}
}

// NewColorTransformer creates a new color transformer with default scheme
func NewColorTransformer() *ColorTransformer {
	return &ColorTransformer{
		scheme: DefaultColorScheme(),
	}
}

// NewColorTransformerWithScheme creates a color transformer with custom scheme
func NewColorTransformerWithScheme(scheme ColorScheme) *ColorTransformer {
	return &ColorTransformer{
		scheme: scheme,
	}
}

// Name returns the transformer name
func (c *ColorTransformer) Name() string {
	return "color"
}

// Priority returns the transformer priority (lower = earlier)
func (c *ColorTransformer) Priority() int {
	return 200
}

// CanTransform checks if this transformer applies to the given format
func (c *ColorTransformer) CanTransform(format string) bool {
	// Only apply colors to terminal/console outputs
	return format == FormatTable
}

// Transform adds ANSI color codes to the output
func (c *ColorTransformer) Transform(ctx context.Context, input []byte, format string) ([]byte, error) {
	output := string(input)

	// Apply colors based on content patterns
	if strings.Contains(output, "✅") || strings.Contains(output, "Yes") || strings.Contains(output, "true") {
		green := color.New(color.FgGreen).Add(color.Bold)
		output = green.Sprint(output)
	} else if strings.Contains(output, "❌") || strings.Contains(output, "No") || strings.Contains(output, "false") {
		red := color.New(color.FgRed).Add(color.Bold)
		output = red.Sprint(output)
	} else if strings.Contains(output, "🚨") || strings.Contains(output, "!!") {
		red := color.New(color.FgRed).Add(color.Bold)
		output = red.Sprint(output)
	} else if strings.Contains(output, "ℹ️") {
		blue := color.New(color.FgBlue)
		output = blue.Sprint(output)
	}

	return []byte(output), nil
}

// SortTransformer sorts table data by a specified key
type SortTransformer struct {
	key       string
	ascending bool
}

// NewSortTransformer creates a new sort transformer
func NewSortTransformer(key string, ascending bool) *SortTransformer {
	return &SortTransformer{
		key:       key,
		ascending: ascending,
	}
}

// NewSortTransformerAscending creates a sort transformer with ascending order
func NewSortTransformerAscending(key string) *SortTransformer {
	return NewSortTransformer(key, true)
}

// Name returns the transformer name
func (s *SortTransformer) Name() string {
	return "sort"
}

// Priority returns the transformer priority (lower = earlier)
func (s *SortTransformer) Priority() int {
	return 50 // Run early, before other transformations
}

// CanTransform checks if this transformer applies to the given format
func (s *SortTransformer) CanTransform(format string) bool {
	// Apply sorting to tabular formats
	return format == FormatTable || format == FormatCSV || format == FormatHTML || format == FormatMarkdown
}

// Transform sorts the tabular data by the specified key
func (s *SortTransformer) Transform(ctx context.Context, input []byte, format string) ([]byte, error) {
	if s.key == "" {
		return input, nil
	}

	content := string(input)
	lines := strings.Split(content, "\n")

	if len(lines) < 2 {
		return input, nil // Need at least header + 1 data row
	}

	// Find header line and data lines
	var headerLine string
	var dataLines []string
	var beforeHeader []string
	var afterData []string

	headerFound := false
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if !headerFound {
				beforeHeader = append(beforeHeader, lines[i])
			} else {
				afterData = append(afterData, lines[i])
			}
			continue
		}

		// Detect header line (contains separators or is first non-empty line)
		if !headerFound && (strings.Contains(line, "\t") || strings.Contains(line, ",") || strings.Contains(line, "|")) {
			headerLine = lines[i]
			headerFound = true
			continue
		}

		if headerFound {
			if strings.Contains(line, "\t") || strings.Contains(line, ",") || strings.Contains(line, "|") {
				dataLines = append(dataLines, lines[i])
			} else {
				afterData = append(afterData, lines[i])
			}
		} else {
			beforeHeader = append(beforeHeader, lines[i])
		}
	}

	if !headerFound || len(dataLines) == 0 {
		return input, nil
	}

	// Parse header to find sort column index
	separator := detectSeparator(headerLine)
	if separator == "" {
		return input, nil
	}
	headers := strings.Split(headerLine, separator)

	// Find the column index for our sort key
	sortColumnIndex := -1
	for i, header := range headers {
		if strings.TrimSpace(header) == s.key {
			sortColumnIndex = i
			break
		}
	}

	if sortColumnIndex == -1 {
		return input, nil // Sort key not found
	}

	// Sort the data lines
	sort.Slice(dataLines, func(i, j int) bool {
		rowI := strings.Split(dataLines[i], separator)
		rowJ := strings.Split(dataLines[j], separator)

		if sortColumnIndex >= len(rowI) || sortColumnIndex >= len(rowJ) {
			return false
		}

		valueI := strings.TrimSpace(rowI[sortColumnIndex])
		valueJ := strings.TrimSpace(rowJ[sortColumnIndex])

		// Try numeric comparison first
		if numI, errI := strconv.ParseFloat(valueI, 64); errI == nil {
			if numJ, errJ := strconv.ParseFloat(valueJ, 64); errJ == nil {
				if s.ascending {
					return numI < numJ
				}
				return numI > numJ
			}
		}

		// Fall back to string comparison
		if s.ascending {
			return valueI < valueJ
		}
		return valueI > valueJ
	})

	// Reconstruct the output
	var result []string
	result = append(result, beforeHeader...)
	result = append(result, headerLine)
	result = append(result, dataLines...)
	result = append(result, afterData...)

	return []byte(strings.Join(result, "\n")), nil
}

// LineSplitTransformer splits cells containing multiple values into separate rows
type LineSplitTransformer struct {
	separator string
}

// NewLineSplitTransformer creates a new line split transformer
func NewLineSplitTransformer(separator string) *LineSplitTransformer {
	if separator == "" {
		separator = "\n" // Default to newline
	}
	return &LineSplitTransformer{
		separator: separator,
	}
}

// NewLineSplitTransformerDefault creates a line split transformer with default newline separator
func NewLineSplitTransformerDefault() *LineSplitTransformer {
	return NewLineSplitTransformer("\n")
}

// Name returns the transformer name
func (l *LineSplitTransformer) Name() string {
	return "linesplit"
}

// Priority returns the transformer priority (lower = earlier)
func (l *LineSplitTransformer) Priority() int {
	return 150
}

// CanTransform checks if this transformer applies to the given format
func (l *LineSplitTransformer) CanTransform(format string) bool {
	// Apply line splitting to tabular formats
	return format == FormatTable || format == FormatCSV || format == FormatHTML || format == FormatMarkdown
}

// Transform splits multi-line cells into separate rows
func (l *LineSplitTransformer) Transform(ctx context.Context, input []byte, format string) ([]byte, error) {
	content := string(input)
	lines := strings.Split(content, "\n")

	if len(lines) < 2 {
		return input, nil
	}

	var result []string
	var columnSeparator string

	// Detect the column separator from first data line
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			result = append(result, line)
			continue
		}

		if columnSeparator == "" {
			columnSeparator = detectSeparator(line)
		}

		if columnSeparator == "" {
			result = append(result, line)
			continue
		}

		// Check if any cell contains our separator
		cells := strings.Split(line, columnSeparator)
		needsSplitting := false
		maxSplits := 1

		for _, cell := range cells {
			if strings.Contains(cell, l.separator) {
				needsSplitting = true
				splits := strings.Split(cell, l.separator)
				if len(splits) > maxSplits {
					maxSplits = len(splits)
				}
			}
		}

		if !needsSplitting {
			result = append(result, line)
			continue
		}

		// Split the row into multiple rows
		for i := 0; i < maxSplits; i++ {
			var newCells []string
			for _, cell := range cells {
				if strings.Contains(cell, l.separator) {
					splits := strings.Split(cell, l.separator)
					if i < len(splits) {
						newCells = append(newCells, strings.TrimSpace(splits[i]))
					} else {
						newCells = append(newCells, "")
					}
				} else {
					// For cells without separator, show value in first row, empty in subsequent rows
					if i == 0 {
						newCells = append(newCells, cell)
					} else {
						newCells = append(newCells, "")
					}
				}
			}
			result = append(result, strings.Join(newCells, columnSeparator))
		}
	}

	return []byte(strings.Join(result, "\n")), nil
}

// RemoveColorsTransformer removes ANSI color codes from output (for file output)
type RemoveColorsTransformer struct{}

// NewRemoveColorsTransformer creates a new color removal transformer
func NewRemoveColorsTransformer() *RemoveColorsTransformer {
	return &RemoveColorsTransformer{}
}

// Name returns the transformer name
func (r *RemoveColorsTransformer) Name() string {
	return "remove-colors"
}

// Priority returns the transformer priority (lower = earlier)
func (r *RemoveColorsTransformer) Priority() int {
	return 1000 // Run very late, after color addition
}

// CanTransform checks if this transformer applies to the given format
func (r *RemoveColorsTransformer) CanTransform(format string) bool {
	// Remove colors from all formats when writing to files
	return true
}

// Transform removes ANSI color codes from the output
func (r *RemoveColorsTransformer) Transform(ctx context.Context, input []byte, format string) ([]byte, error) {
	// Remove ANSI escape sequences (colors)
	re := regexp.MustCompile(`\x1B\[([0-9]{1,3}(;[0-9]{1,3})*)?[mGK]`)
	output := re.ReplaceAll(input, []byte(""))
	return output, nil
}

// detectSeparator detects the separator used in a line
func detectSeparator(line string) string {
	if strings.Contains(line, "\t") {
		return "\t"
	}
	if strings.Contains(line, ",") {
		return ","
	}
	if strings.Contains(line, "|") {
		return "|"
	}
	return ""
}
