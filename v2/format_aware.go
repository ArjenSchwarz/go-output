package output

import (
	"context"
	"strings"
)

// FormatDetector provides advanced format detection capabilities
type FormatDetector struct{}

// NewFormatDetector creates a new format detetcor
func NewFormatDetector() *FormatDetector {
	return &FormatDetector{}
}

// IsTextBasedFormat checks if a format supports text-based transformations
func (fd *FormatDetector) IsTextBasedFormat(format string) bool {
	textFormats := []string{"table", "markdown", "html", "csv", "yaml"}
	for _, tf := range textFormats {
		if format == tf {
			return true
		}
	}
	return false
}

// IsStructuredFormat checks if a format is structured (like JSON, YAML)
func (fd *FormatDetector) IsStructuredFormat(format string) bool {
	structuredFormats := []string{"json", "yaml"}
	for _, sf := range structuredFormats {
		if format == sf {
			return true
		}
	}
	return false
}

// IsTabularFormat checks if a format represents tabular data
func (fd *FormatDetector) IsTabularFormat(format string) bool {
	tabularFormats := []string{"table", "csv", "html", "markdown"}
	for _, tf := range tabularFormats {
		if format == tf {
			return true
		}
	}
	return false
}

// IsGraphFormat checks if a format is for graph/diagram output
func (fd *FormatDetector) IsGraphFormat(format string) bool {
	graphFormats := []string{"dot", "mermaid", "drawio"}
	for _, gf := range graphFormats {
		if format == gf {
			return true
		}
	}
	return false
}

// SupportsColors checks if a format supports ANSI color codes
func (fd *FormatDetector) SupportsColors(format string) bool {
	// Only terminal/console formats support ANSI colors
	return format == "table"
}

// SupportsEmoji checks if a format supports emoji characters
func (fd *FormatDetector) SupportsEmoji(format string) bool {
	// Most text-based formats support emoji except structured data formats
	return fd.IsTextBasedFormat(format) && !fd.IsStructuredFormat(format)
}

// RequiresEscaping checks if a format requires special character escaping
func (fd *FormatDetector) RequiresEscaping(format string) bool {
	escapingFormats := map[string]bool{
		"html":     true,
		"markdown": true,
		"csv":      true,
		"json":     true,
		"yaml":     true,
	}
	return escapingFormats[format]
}

// SupportsSorting checks if a format supports data sorting
func (fd *FormatDetector) SupportsSorting(format string) bool {
	return fd.IsTabularFormat(format)
}

// SupportsLineSplitting checks if a format supports line splitting transformations
func (fd *FormatDetector) SupportsLineSplitting(format string) bool {
	return fd.IsTabularFormat(format)
}

// FormatAwareTransformer wraps existing transformers with enhanced format detection
type FormatAwareTransformer struct {
	transformer Transformer
	detetcor    *FormatDetector
}

// NewFormatAwareTransformer wraps a transformer with format awareness
func NewFormatAwareTransformer(transformer Transformer) *FormatAwareTransformer {
	return &FormatAwareTransformer{
		transformer: transformer,
		detetcor:    NewFormatDetector(),
	}
}

// Name returns the underlying transformer name
func (fat *FormatAwareTransformer) Name() string {
	return fat.transformer.Name()
}

// Priority returns the underlying transformer priority
func (fat *FormatAwareTransformer) Priority() int {
	return fat.transformer.Priority()
}

// CanTransform provides enhanced format detection
func (fat *FormatAwareTransformer) CanTransform(format string) bool {
	// Delegate to the underlying transformer first
	if !fat.transformer.CanTransform(format) {
		return false
	}

	// Add additional format-specific logic based on transformer type
	switch fat.transformer.Name() {
	case "emoji":
		return fat.detetcor.SupportsEmoji(format)
	case "color":
		return fat.detetcor.SupportsColors(format)
	case "sort":
		return fat.detetcor.SupportsSorting(format)
	case "linesplit":
		return fat.detetcor.SupportsLineSplitting(format)
	case "remove-colors":
		// Color removal is needed for all formats when writing to files
		return true
	default:
		return true
	}
}

// Transform applies format-aware transformation while preserving original data integrity
func (fat *FormatAwareTransformer) Transform(ctx context.Context, input []byte, format string) ([]byte, error) {
	// Create a copy of input to ensure we don't modify original document data
	inputCopy := make([]byte, len(input))
	copy(inputCopy, input)

	// Apply the transformation to the copy
	return fat.transformer.Transform(ctx, inputCopy, format)
}

// EnhancedEmojiTransformer provides format-specific emoji transformations
type EnhancedEmojiTransformer struct {
	*EmojiTransformer
	detetcor *FormatDetector
}

// NewEnhancedEmojiTransformer creates an enhanced emoji transformer
func NewEnhancedEmojiTransformer() *EnhancedEmojiTransformer {
	return &EnhancedEmojiTransformer{
		EmojiTransformer: &EmojiTransformer{},
		detetcor:         NewFormatDetector(),
	}
}

// CanTransform provides enhanced format detection for emoji
func (eet *EnhancedEmojiTransformer) CanTransform(format string) bool {
	return eet.detetcor.SupportsEmoji(format)
}

// Transform applies format-specific emoji transformations
func (eet *EnhancedEmojiTransformer) Transform(ctx context.Context, input []byte, format string) ([]byte, error) {
	// Create a copy to preserve original data
	inputCopy := make([]byte, len(input))
	copy(inputCopy, input)

	// Check if this format supports emoji
	if !eet.CanTransform(format) {
		return inputCopy, nil
	}

	output := string(inputCopy)

	// Format-specific emoji substitutions
	switch format {
	case "markdown":
		// In markdown, be more conservative with emoji to maintain readability
		output = strings.ReplaceAll(output, "!!", "⚠️")
		output = strings.ReplaceAll(output, "OK", "✅")
	case FormatHTML:
		// In HTML, use emoji but ensure proper encoding
		output = strings.ReplaceAll(output, "!!", "&#x1F6A8;") // 🚨
		output = strings.ReplaceAll(output, "OK", "&#x2705;")  // ✅
		output = strings.ReplaceAll(output, "Yes", "&#x2705;") // ✅
		output = strings.ReplaceAll(output, "No", "&#x274C;")  // ❌
	default:
		// Default behavior for table, csv, etc.
		return eet.EmojiTransformer.Transform(ctx, inputCopy, format)
	}

	return []byte(output), nil
}

// EnhancedColorTransformer provides format-specific color handling
type EnhancedColorTransformer struct {
	*ColorTransformer
	detetcor *FormatDetector
}

// NewEnhancedColorTransformer creates an enhanced color transformer
func NewEnhancedColorTransformer() *EnhancedColorTransformer {
	return &EnhancedColorTransformer{
		ColorTransformer: NewColorTransformer(),
		detetcor:         NewFormatDetector(),
	}
}

// CanTransform checks if colors are supported for the format
func (etc *EnhancedColorTransformer) CanTransform(format string) bool {
	return etc.detetcor.SupportsColors(format)
}

// Transform applies format-specific color transformations
func (etc *EnhancedColorTransformer) Transform(ctx context.Context, input []byte, format string) ([]byte, error) {
	// Create a copy to preserve original data
	inputCopy := make([]byte, len(input))
	copy(inputCopy, input)

	// Only apply colors to terminal formats
	if !etc.detetcor.SupportsColors(format) {
		return inputCopy, nil
	}

	return etc.ColorTransformer.Transform(ctx, inputCopy, format)
}

// EnhancedSortTransformer provides format-specific sorting
type EnhancedSortTransformer struct {
	*SortTransformer
	detetcor *FormatDetector
}

// NewEnhancedSortTransformer creates an enhanced sort transformer
func NewEnhancedSortTransformer(key string, ascending bool) *EnhancedSortTransformer {
	return &EnhancedSortTransformer{
		SortTransformer: NewSortTransformer(key, ascending),
		detetcor:        NewFormatDetector(),
	}
}

// CanTransform checks if sorting is supported for the format
func (est *EnhancedSortTransformer) CanTransform(format string) bool {
	return est.detetcor.SupportsSorting(format)
}

// Transform applies format-specific sorting
func (est *EnhancedSortTransformer) Transform(ctx context.Context, input []byte, format string) ([]byte, error) {
	// Create a copy to preserve original data
	inputCopy := make([]byte, len(input))
	copy(inputCopy, input)

	if !est.detetcor.SupportsSorting(format) {
		return inputCopy, nil
	}

	return est.SortTransformer.Transform(ctx, inputCopy, format)
}

// DataIntegrityValidator ensures transformers don't modify original document data
type DataIntegrityValidator struct {
	originalData []byte
}

// NewDataIntegrityValidator creates a validator for the original data
func NewDataIntegrityValidator(originalData []byte) *DataIntegrityValidator {
	// Create a deep copy of the original data
	copy := make([]byte, len(originalData))
	copy = append(copy[:0], originalData...)
	return &DataIntegrityValidator{
		originalData: copy,
	}
}

// ValidateIntegrity checks that the original data hasn't been modified
func (div *DataIntegrityValidator) ValidateIntegrity(currentData []byte) bool {
	if len(div.originalData) != len(currentData) {
		return false
	}

	for i, b := range div.originalData {
		if currentData[i] != b {
			return false
		}
	}

	return true
}

// GetOriginalData returns a copy of the original data
func (div *DataIntegrityValidator) GetOriginalData() []byte {
	copy := make([]byte, len(div.originalData))
	copy = append(copy[:0], div.originalData...)
	return copy
}
