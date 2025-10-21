package output

// rawConfig holds configuration for raw content creation
type rawConfig struct {
	validateFormat  bool
	preserveData    bool
	transformations []Operation
}

// RawOption configures raw content creation
type RawOption func(*rawConfig)

// WithFormatValidation enables or disables format validation
func WithFormatValidation(validate bool) RawOption {
	return func(rc *rawConfig) {
		rc.validateFormat = validate
	}
}

// WithDataPreservation enables or disables data preservation (copying)
func WithDataPreservation(preserve bool) RawOption {
	return func(rc *rawConfig) {
		rc.preserveData = preserve
	}
}

// WithRawTransformations sets transformations for the raw content
func WithRawTransformations(ops ...Operation) RawOption {
	return func(rc *rawConfig) {
		rc.transformations = ops
	}
}

// ApplyRawOptions applies all options to the raw content configuration
func ApplyRawOptions(opts ...RawOption) *rawConfig {
	rc := &rawConfig{
		validateFormat: true, // Default to validating formats
		preserveData:   true, // Default to preserving data by copying
	}
	for _, opt := range opts {
		opt(rc)
	}
	return rc
}
