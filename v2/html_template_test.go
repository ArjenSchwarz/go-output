package output

import (
	"testing"
)

func TestDefaultHTMLTemplateDefaults(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		template *HTMLTemplate
		field    string
		want     any
	}{
		"default Title": {
			template: DefaultHTMLTemplate,
			field:    "Title",
			want:     "Output Report",
		},
		"default Charset": {
			template: DefaultHTMLTemplate,
			field:    "Charset",
			want:     "UTF-8",
		},
		"default Language": {
			template: DefaultHTMLTemplate,
			field:    "Language",
			want:     "en",
		},
		"default Viewport": {
			template: DefaultHTMLTemplate,
			field:    "Viewport",
			want:     "width=device-width, initial-scale=1.0",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var got any
			switch tc.field {
			case "Title":
				got = tc.template.Title
			case "Charset":
				got = tc.template.Charset
			case "Language":
				got = tc.template.Language
			case "Viewport":
				got = tc.template.Viewport
			}

			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMinimalHTMLTemplateDefaults(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		field string
		want  any
	}{
		"Title": {
			field: "Title",
			want:  "Output Report",
		},
		"Charset": {
			field: "Charset",
			want:  "UTF-8",
		},
		"Language": {
			field: "Language",
			want:  "en",
		},
		"Viewport": {
			field: "Viewport",
			want:  "width=device-width, initial-scale=1.0",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var got any
			switch tc.field {
			case "Title":
				got = MinimalHTMLTemplate.Title
			case "Charset":
				got = MinimalHTMLTemplate.Charset
			case "Language":
				got = MinimalHTMLTemplate.Language
			case "Viewport":
				got = MinimalHTMLTemplate.Viewport
			}

			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMermaidHTMLTemplateDefaults(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		field string
		want  any
	}{
		"Title": {
			field: "Title",
			want:  "Diagram Output",
		},
		"Charset": {
			field: "Charset",
			want:  "UTF-8",
		},
		"Language": {
			field: "Language",
			want:  "en",
		},
		"Viewport": {
			field: "Viewport",
			want:  "width=device-width, initial-scale=1.0",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var got any
			switch tc.field {
			case "Title":
				got = MermaidHTMLTemplate.Title
			case "Charset":
				got = MermaidHTMLTemplate.Charset
			case "Language":
				got = MermaidHTMLTemplate.Language
			case "Viewport":
				got = MermaidHTMLTemplate.Viewport
			}

			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCustomHTMLTemplateOverrides(t *testing.T) {
	t.Parallel()

	custom := &HTMLTemplate{
		Title:    "Custom Title",
		Charset:  "ISO-8859-1",
		Language: "de",
		Viewport: "width=800px",
	}

	tests := map[string]struct {
		got  string
		want string
	}{
		"custom Title": {
			got:  custom.Title,
			want: "Custom Title",
		},
		"custom Charset": {
			got:  custom.Charset,
			want: "ISO-8859-1",
		},
		"custom Language": {
			got:  custom.Language,
			want: "de",
		},
		"custom Viewport": {
			got:  custom.Viewport,
			want: "width=800px",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.got != tc.want {
				t.Errorf("got %q, want %q", tc.got, tc.want)
			}
		})
	}
}

func TestEmptyHTMLTemplateFieldHandling(t *testing.T) {
	t.Parallel()

	empty := &HTMLTemplate{}

	tests := map[string]struct {
		field string
		check func() bool
	}{
		"empty Title": {
			field: "Title",
			check: func() bool { return empty.Title == "" },
		},
		"empty Charset": {
			field: "Charset",
			check: func() bool { return empty.Charset == "" },
		},
		"empty Language": {
			field: "Language",
			check: func() bool { return empty.Language == "" },
		},
		"empty Viewport": {
			field: "Viewport",
			check: func() bool { return empty.Viewport == "" },
		},
		"empty CSS": {
			field: "CSS",
			check: func() bool { return empty.CSS == "" },
		},
		"nil ExternalCSS": {
			field: "ExternalCSS",
			check: func() bool { return empty.ExternalCSS == nil },
		},
		"nil MetaTags": {
			field: "MetaTags",
			check: func() bool { return empty.MetaTags == nil },
		},
		"nil ThemeOverrides": {
			field: "ThemeOverrides",
			check: func() bool { return empty.ThemeOverrides == nil },
		},
		"nil BodyAttrs": {
			field: "BodyAttrs",
			check: func() bool { return empty.BodyAttrs == nil },
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if !tc.check() {
				t.Errorf("expected empty field %q", tc.field)
			}
		})
	}
}
