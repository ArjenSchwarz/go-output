package output

import (
	"testing"
)

// TestWithLevel verifies the WithLevel option
func TestWithLevel(t *testing.T) {
	tests := map[string]struct {
		level         int
		expectedLevel int
	}{"large level":

	// Should remain at default

	{

		level:         100,
		expectedLevel: 100,
	}, "level 0": {

		level:         0,
		expectedLevel: 0,
	}, "level 1": {

		level:         1,
		expectedLevel: 1,
	}, "level 5": {

		level:         5,
		expectedLevel: 5,
	}, "negative level ignored": {

		level:         -1,
		expectedLevel: 0,
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			sc := &sectionConfig{level: 0} // Start with default
			opt := WithLevel(tt.level)
			opt(sc)

			if sc.level != tt.expectedLevel {
				t.Errorf("level = %d, want %d", sc.level, tt.expectedLevel)
			}
		})
	}
}

// TestApplySectionOptions verifies option application
func TestApplySectionOptions(t *testing.T) {
	tests := map[string]struct {
		opts          []SectionOption
		expectedLevel int
	}{"multiple level options - last wins": {

		opts: []SectionOption{
			WithLevel(1),
			WithLevel(2),
			WithLevel(3),
		},
		expectedLevel: 3,
	}, "negative level ignored": {

		opts: []SectionOption{
			WithLevel(2),
			WithLevel(-5), // Should be ignored
		},
		expectedLevel: 2,
	}, "no options uses default": {

		opts:          []SectionOption{},
		expectedLevel: 0,
	}, "single level option": {

		opts: []SectionOption{
			WithLevel(2),
		},
		expectedLevel: 2,
	}, "valid after invalid": {

		opts: []SectionOption{
			WithLevel(-1), // Ignored
			WithLevel(4),
		},
		expectedLevel: 4,
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			sc := ApplySectionOptions(tt.opts...)

			if sc.level != tt.expectedLevel {
				t.Errorf("level = %d, want %d", sc.level, tt.expectedLevel)
			}
		})
	}
}

// TestSectionOptionsEdgeCases tests edge cases
func TestSectionOptionsEdgeCases(t *testing.T) {
	// Test that negative levels don't modify existing level
	sc1 := &sectionConfig{level: 5}
	WithLevel(-10)(sc1)
	if sc1.level != 5 {
		t.Errorf("negative level should not modify existing level, got %d", sc1.level)
	}

	// Test zero is valid
	sc2 := &sectionConfig{level: 5}
	WithLevel(0)(sc2)
	if sc2.level != 0 {
		t.Errorf("level 0 should be valid, got %d", sc2.level)
	}

	// Test multiple applications
	sc3 := ApplySectionOptions(
		WithLevel(1),
		WithLevel(2),
		WithLevel(3),
		WithLevel(2),
		WithLevel(1),
		WithLevel(0),
	)
	if sc3.level != 0 {
		t.Errorf("expected final level 0, got %d", sc3.level)
	}
}
