package output

import (
	"reflect"
	"testing"
)

// TestWithTextStyle verifies that WithTextStyle sets the complete style
func TestWithTextStyle(t *testing.T) {
	tests := map[string]struct {
		style TextStyle
	}{"complete style": {

		style: TextStyle{
			Bold:   true,
			Italic: true,
			Color:  "red",
			Size:   16,
			Header: true,
		},
	}, "empty style": {

		style: TextStyle{},
	}, "partial style": {

		style: TextStyle{
			Bold:  true,
			Color: "blue",
		},
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tc := &textConfig{}
			opt := WithTextStyle(tt.style)
			opt(tc)

			if !reflect.DeepEqual(tc.style, tt.style) {
				t.Errorf("style = %+v, want %+v", tc.style, tt.style)
			}
		})
	}
}

// TestWithBold verifies bold option
func TestWithBold(t *testing.T) {
	tests := []struct {
		name string
		bold bool
	}{
		{"set bold true", true},
		{"set bold false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := &textConfig{
				style: TextStyle{Italic: true}, // Pre-existing style
			}
			opt := WithBold(tt.bold)
			opt(tc)

			if tc.style.Bold != tt.bold {
				t.Errorf("Bold = %v, want %v", tc.style.Bold, tt.bold)
			}
			// Ensure other properties are preserved
			if !tc.style.Italic {
				t.Error("Italic should be preserved")
			}
		})
	}
}

// TestWithItalic verifies italic option
func TestWithItalic(t *testing.T) {
	tests := []struct {
		name   string
		italic bool
	}{
		{"set italic true", true},
		{"set italic false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := &textConfig{
				style: TextStyle{Bold: true}, // Pre-existing style
			}
			opt := WithItalic(tt.italic)
			opt(tc)

			if tc.style.Italic != tt.italic {
				t.Errorf("Italic = %v, want %v", tc.style.Italic, tt.italic)
			}
			// Ensure other properties are preserved
			if !tc.style.Bold {
				t.Error("Bold should be preserved")
			}
		})
	}
}

// TestWithColor verifies color option
func TestWithColor(t *testing.T) {
	tests := []struct {
		name  string
		color string
	}{
		{"red color", "red"},
		{"hex color", "#FF0000"},
		{"rgb color", "rgb(255,0,0)"},
		{"empty color", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := &textConfig{
				style: TextStyle{Bold: true},
			}
			opt := WithColor(tt.color)
			opt(tc)

			if tc.style.Color != tt.color {
				t.Errorf("Color = %q, want %q", tc.style.Color, tt.color)
			}
			// Ensure other properties are preserved
			if !tc.style.Bold {
				t.Error("Bold should be preserved")
			}
		})
	}
}

// TestWithSize verifies size option
func TestWithSize(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{"normal size", 12},
		{"large size", 24},
		{"zero size", 0},
		{"negative size", -1}, // Should be allowed, validation elsewhere
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := &textConfig{
				style: TextStyle{Color: "blue"},
			}
			opt := WithSize(tt.size)
			opt(tc)

			if tc.style.Size != tt.size {
				t.Errorf("Size = %d, want %d", tc.style.Size, tt.size)
			}
			// Ensure other properties are preserved
			if tc.style.Color != "blue" {
				t.Error("Color should be preserved")
			}
		})
	}
}

// TestWithHeader verifies header option for v1 compatibility
func TestWithHeader(t *testing.T) {
	tests := []struct {
		name   string
		header bool
	}{
		{"set header true", true},
		{"set header false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := &textConfig{
				style: TextStyle{Bold: true, Size: 18},
			}
			opt := WithHeader(tt.header)
			opt(tc)

			if tc.style.Header != tt.header {
				t.Errorf("Header = %v, want %v", tc.style.Header, tt.header)
			}
			// Ensure other properties are preserved
			if !tc.style.Bold {
				t.Error("Bold should be preserved")
			}
			if tc.style.Size != 18 {
				t.Error("Size should be preserved")
			}
		})
	}
}

// TestApplyTextOptions verifies multiple options are applied correctly
func TestApplyTextOptions(t *testing.T) {
	tests := map[string]struct {
		opts          []TextOption
		expectedStyle TextStyle
	}{"WithTextStyle overrides individual settings": {

		opts: []TextOption{
			WithBold(true),
			WithColor("red"),
			WithTextStyle(TextStyle{
				Italic: true,
				Size:   20,
			}),
		},
		expectedStyle: TextStyle{
			Italic: true,
			Size:   20,
		},
	}, "individual settings after WithTextStyle": {

		opts: []TextOption{
			WithTextStyle(TextStyle{
				Bold:   true,
				Italic: true,
				Color:  "red",
				Size:   16,
			}),
			WithBold(false),
			WithColor("blue"),
		},
		expectedStyle: TextStyle{
			Bold:   false,
			Italic: true,
			Color:  "blue",
			Size:   16,
		},
	}, "last option wins for same property": {

		opts: []TextOption{
			WithColor("red"),
			WithColor("blue"),
			WithColor("green"),
		},
		expectedStyle: TextStyle{Color: "green"},
	}, "multiple options combined": {

		opts: []TextOption{
			WithBold(true),
			WithItalic(true),
			WithColor("green"),
			WithSize(14),
			WithHeader(false),
		},
		expectedStyle: TextStyle{
			Bold:   true,
			Italic: true,
			Color:  "green",
			Size:   14,
			Header: false,
		},
	}, "no options uses defaults": {

		opts:          []TextOption{},
		expectedStyle: TextStyle{},
	}, "single option": {

		opts: []TextOption{
			WithBold(true),
		},
		expectedStyle: TextStyle{Bold: true},
	}}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tc := ApplyTextOptions(tt.opts...)

			if !reflect.DeepEqual(tc.style, tt.expectedStyle) {
				t.Errorf("style = %+v, want %+v", tc.style, tt.expectedStyle)
			}
		})
	}
}

// TestTextOptionsChaining verifies options can be chained effectively
func TestTextOptionsChaining(t *testing.T) {
	// Create a common header style
	headerStyle := []TextOption{
		WithBold(true),
		WithSize(18),
		WithHeader(true),
	}

	// Add color to header style
	redHeader := append(headerStyle, WithColor("red"))
	tc1 := ApplyTextOptions(redHeader...)

	expectedRed := TextStyle{
		Bold:   true,
		Size:   18,
		Header: true,
		Color:  "red",
	}
	if !reflect.DeepEqual(tc1.style, expectedRed) {
		t.Errorf("red header style = %+v, want %+v", tc1.style, expectedRed)
	}

	// Different color for another header
	blueHeader := append(headerStyle, WithColor("blue"))
	tc2 := ApplyTextOptions(blueHeader...)

	expectedBlue := TextStyle{
		Bold:   true,
		Size:   18,
		Header: true,
		Color:  "blue",
	}
	if !reflect.DeepEqual(tc2.style, expectedBlue) {
		t.Errorf("blue header style = %+v, want %+v", tc2.style, expectedBlue)
	}
}

// TestTextOptionsIndependence verifies options don't affect each other
func TestTextOptionsIndependence(t *testing.T) {
	// Create first configuration
	tc1 := ApplyTextOptions(
		WithBold(true),
		WithColor("red"),
	)

	// Create second configuration
	tc2 := ApplyTextOptions(
		WithItalic(true),
		WithColor("blue"),
	)

	// Verify they're independent
	if tc1.style.Italic {
		t.Error("tc1 should not have italic")
	}
	if tc2.style.Bold {
		t.Error("tc2 should not have bold")
	}
	if tc1.style.Color != "red" {
		t.Errorf("tc1 color = %q, want red", tc1.style.Color)
	}
	if tc2.style.Color != "blue" {
		t.Errorf("tc2 color = %q, want blue", tc2.style.Color)
	}
}
