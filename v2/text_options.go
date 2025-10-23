package output

// textConfig holds configuration for text creation
type textConfig struct {
	style           TextStyle
	transformations []Operation
}

// TextOption configures text creation
type TextOption func(*textConfig)

// WithTextStyle sets the complete text style
func WithTextStyle(style TextStyle) TextOption {
	return func(tc *textConfig) {
		tc.style = style
	}
}

// WithBold sets bold styling
func WithBold(bold bool) TextOption {
	return func(tc *textConfig) {
		tc.style.Bold = bold
	}
}

// WithItalic sets italic styling
func WithItalic(italic bool) TextOption {
	return func(tc *textConfig) {
		tc.style.Italic = italic
	}
}

// WithColor sets text color
func WithColor(color string) TextOption {
	return func(tc *textConfig) {
		tc.style.Color = color
	}
}

// WithSize sets text size
func WithSize(size int) TextOption {
	return func(tc *textConfig) {
		tc.style.Size = size
	}
}

// WithHeader marks text as a header (for v1 AddHeader compatibility)
func WithHeader(header bool) TextOption {
	return func(tc *textConfig) {
		tc.style.Header = header
	}
}

// WithTextTransformations sets transformations for the text content
func WithTextTransformations(ops ...Operation) TextOption {
	return func(tc *textConfig) {
		tc.transformations = ops
	}
}

// ApplyTextOptions applies all options to the text configuration
func ApplyTextOptions(opts ...TextOption) *textConfig {
	tc := &textConfig{
		style: TextStyle{}, // Default empty style
	}
	for _, opt := range opts {
		opt(tc)
	}
	return tc
}
