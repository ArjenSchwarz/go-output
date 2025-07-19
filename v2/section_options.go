package output

// sectionConfig holds configuration for section creation
type sectionConfig struct {
	level int
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

// ApplySectionOptions applies all options to the section configuration
func ApplySectionOptions(opts ...SectionOption) *sectionConfig {
	sc := &sectionConfig{
		level: 0, // Default to level 0 (top level)
	}
	for _, opt := range opts {
		opt(sc)
	}
	return sc
}
