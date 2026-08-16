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

// WithDataPreservation enables or disables data preservation (copying).
//
// When enabled (the default), NewRawContent copies the input byte slice so
// later modification of the caller's slice cannot affect the content. When
// disabled, the content stores the caller's slice directly, skipping the copy
// as a performance opt-out for large payloads. In that case the caller must
// not modify the slice after passing it in: the content aliases it, so any
// mutation becomes visible in subsequent renders and breaks the immutability
// guarantees of built documents. RawContent.Data still returns a copy and
// Clone still deep-copies regardless of this option.
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

// ApplyRawOptions applies all options to the raw content configuration.
// Nil options are ignored.
func ApplyRawOptions(opts ...RawOption) *rawConfig {
	rc := &rawConfig{
		validateFormat: true, // Default to validating formats
		preserveData:   true, // Default to preserving data by copying
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(rc)
	}
	return rc
}
