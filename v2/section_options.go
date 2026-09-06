package output

// sectionConfig holds configuration for section creation
type sectionConfig struct {
	level           int
	transformations []Operation
}

// SectionOption configures section creation
type SectionOption func(*sectionConfig)

// WithLevel sets the hierarchical level of the section
func WithLevel(level int) SectionOption {
	return func(sc *sectionConfig) {
		if level >= 0 {
			sc.level = level
		}
	}
}

// WithSectionTransformations sets transformations for the section content.
// The operations are copied and nil entries are dropped, so later changes to
// the caller's slice cannot affect the content (T-1378).
func WithSectionTransformations(ops ...Operation) SectionOption {
	return func(sc *sectionConfig) {
		sc.transformations = cloneOperations(ops)
	}
}

// ApplySectionOptions applies all options to the section configuration.
// Nil options are ignored.
func ApplySectionOptions(opts ...SectionOption) *sectionConfig {
	sc := &sectionConfig{
		level: 0, // Default to level 0 (top level)
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(sc)
	}
	return sc
}
