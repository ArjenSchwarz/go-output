package output

import (
	"context"
	"regexp"
	"slices"
	"strings"
)

// FormatDetector provides advanced format detection capabilities
type FormatDetector struct{}

// NewFormatDetector creates a new format detector
func NewFormatDetector() *FormatDetector {
	return &FormatDetector{}
}

// IsTextBasedFormat checks if a format supports text-based transformations
func (fd *FormatDetector) IsTextBasedFormat(format string) bool {
	textFormats := []string{FormatTable, FormatMarkdown, FormatHTML, FormatCSV, FormatYAML}
	return slices.Contains(textFormats, format)
}

// IsStructuredFormat checks if a format is structured (like JSON, YAML)
func (fd *FormatDetector) IsStructuredFormat(format string) bool {
	structuredFormats := []string{FormatJSON, FormatYAML}
	return slices.Contains(structuredFormats, format)
}

// IsTabularFormat checks if a format represents tabular data that the
// byte-level tabular transformers can parse (CSV via encoding/csv, otherwise
// tab/comma/pipe separated lines). HTML renders tables but uses markup rather
// than those separators, so it is deliberately excluded — sorting on HTML was
// a silent no-op and line splitting could corrupt the markup (T-1510).
func (fd *FormatDetector) IsTabularFormat(format string) bool {
	tabularFormats := []string{FormatTable, FormatCSV, FormatMarkdown}
	return slices.Contains(tabularFormats, format)
}

// IsGraphFormat checks if a format is for graph/diagram output
func (fd *FormatDetector) IsGraphFormat(format string) bool {
	graphFormats := []string{FormatDOT, FormatMermaid, FormatDrawIO}
	return slices.Contains(graphFormats, format)
}

// SupportsColors checks if a format supports ANSI color codes
func (fd *FormatDetector) SupportsColors(format string) bool {
	// Only terminal/console formats support ANSI colors
	return format == FormatTable
}

// SupportsEmoji checks if a format supports emoji characters
func (fd *FormatDetector) SupportsEmoji(format string) bool {
	// Most text-based formats support emoji except structured data formats
	return fd.IsTextBasedFormat(format) && !fd.IsStructuredFormat(format)
}

// RequiresEscaping checks if a format requires special character escaping
func (fd *FormatDetector) RequiresEscaping(format string) bool {
	escapingFormats := map[string]bool{
		FormatHTML:     true,
		FormatMarkdown: true,
		FormatCSV:      true,
		FormatJSON:     true,
		FormatYAML:     true,
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
	detector    *FormatDetector
}

// NewFormatAwareTransformer wraps a transformer with format awareness. It
// returns nil when the supplied transformer is nil — including a typed nil
// such as a nil concrete pointer boxed into the interface (T-1649) — since
// wrapping nil would only defer a nil-pointer panic to the wrapper's
// Name/Priority/CanTransform/Transform methods.
func NewFormatAwareTransformer(transformer Transformer) *FormatAwareTransformer {
	if isNilValue(transformer) {
		return nil
	}
	return &FormatAwareTransformer{
		transformer: transformer,
		detector:    NewFormatDetector(),
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
		return fat.detector.SupportsEmoji(format)
	case transformerNameColor:
		return fat.detector.SupportsColors(format)
	case "sort":
		return fat.detector.SupportsSorting(format)
	case "linesplit":
		return fat.detector.SupportsLineSplitting(format)
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
	detector *FormatDetector
}

// Format-specific replacement tables for EnhancedEmojiTransformer. Word-based
// indicators are matched with word boundaries so that only standalone
// indicators are converted, mirroring the base EmojiTransformer semantics from
// T-1267. Without boundaries, ordinary text such as "Notes" or "BROKEN" would
// be corrupted into "&#x274C;tes"/"BR✅EN" (see T-1509). Compiled once at
// package load since the patterns are constant.
//
// Keep the indicator words in sync with emojiIndicatorReplacements and the
// ColorTransformer indicator patterns (both in transformers.go); "OK" is
// deliberately absent from the color patterns — see the comment there.
var enhancedEmojiReplacements = map[string][]struct {
	pattern     *regexp.Regexp
	replacement string
}{
	FormatMarkdown: {
		// In markdown, be more conservative with emoji to maintain readability
		{regexp.MustCompile(`\bOK\b`), "✅"},
	},
	FormatHTML: {
		// In HTML, use emoji but ensure proper encoding
		{regexp.MustCompile(`\bOK\b`), "&#x2705;"},  // ✅
		{regexp.MustCompile(`\bYes\b`), "&#x2705;"}, // ✅
		{regexp.MustCompile(`\bNo\b`), "&#x274C;"},  // ❌
	},
}

// NewEnhancedEmojiTransformer creates an enhanced emoji transformer
func NewEnhancedEmojiTransformer() *EnhancedEmojiTransformer {
	return &EnhancedEmojiTransformer{
		EmojiTransformer: &EmojiTransformer{},
		detector:         NewFormatDetector(),
	}
}

// CanTransform provides enhanced format detection for emoji
func (eet *EnhancedEmojiTransformer) CanTransform(format string) bool {
	return eet.detector.SupportsEmoji(format)
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

	// Format-specific emoji substitutions. The "!!" indicator is punctuation,
	// so word boundaries (\b) do not apply; it is replaced as a plain
	// substring. Word-based indicators are handled by the boundary-aware
	// enhancedEmojiReplacements table below.
	switch format {
	case FormatMarkdown:
		output = strings.ReplaceAll(output, "!!", "⚠️")
	case FormatHTML:
		output = strings.ReplaceAll(output, "!!", "&#x1F6A8;") // 🚨
	default:
		// Default behavior for table, csv, etc.
		return eet.EmojiTransformer.Transform(ctx, inputCopy, format)
	}

	for _, r := range enhancedEmojiReplacements[format] {
		output = r.pattern.ReplaceAllString(output, r.replacement)
	}

	return []byte(output), nil
}

// EnhancedColorTransformer provides format-specific color handling
type EnhancedColorTransformer struct {
	*ColorTransformer
	detector *FormatDetector
}

// NewEnhancedColorTransformer creates an enhanced color transformer
func NewEnhancedColorTransformer() *EnhancedColorTransformer {
	return &EnhancedColorTransformer{
		ColorTransformer: NewColorTransformer(),
		detector:         NewFormatDetector(),
	}
}

// CanTransform checks if colors are supported for the format
func (etc *EnhancedColorTransformer) CanTransform(format string) bool {
	return etc.detector.SupportsColors(format)
}

// Transform applies format-specific color transformations
func (etc *EnhancedColorTransformer) Transform(ctx context.Context, input []byte, format string) ([]byte, error) {
	// Create a copy to preserve original data
	inputCopy := make([]byte, len(input))
	copy(inputCopy, input)

	// Only apply colors to terminal formats
	if !etc.detector.SupportsColors(format) {
		return inputCopy, nil
	}

	return etc.ColorTransformer.Transform(ctx, inputCopy, format)
}

// EnhancedSortTransformer provides format-specific sorting
type EnhancedSortTransformer struct {
	*SortTransformer
	detector *FormatDetector
}

// NewEnhancedSortTransformer creates an enhanced sort transformer
func NewEnhancedSortTransformer(key string, ascending bool) *EnhancedSortTransformer {
	return &EnhancedSortTransformer{
		SortTransformer: NewSortTransformer(key, ascending),
		detector:        NewFormatDetector(),
	}
}

// CanTransform checks if sorting is supported for the format
func (est *EnhancedSortTransformer) CanTransform(format string) bool {
	return est.detector.SupportsSorting(format)
}

// Transform applies format-specific sorting
func (est *EnhancedSortTransformer) Transform(ctx context.Context, input []byte, format string) ([]byte, error) {
	// Create a copy to preserve original data
	inputCopy := make([]byte, len(input))
	copy(inputCopy, input)

	if !est.detector.SupportsSorting(format) {
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
	dataCopy := make([]byte, len(originalData))
	dataCopy = append(dataCopy[:0], originalData...)
	return &DataIntegrityValidator{
		originalData: dataCopy,
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
	dataCopy := make([]byte, len(div.originalData))
	dataCopy = append(dataCopy[:0], div.originalData...)
	return dataCopy
}
