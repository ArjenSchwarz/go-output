package mermaid

import "fmt"

// HTMLScript contains the HTML script tag for Mermaid.js
const HTMLScript = `<script src="https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js"></script>
	<script>
		mermaid.initialize({ startOnLoad: true });
	</script>`

var scriptset = false

// Settings contains configuration for Mermaid diagrams
type Settings struct {
	AddMarkdown   bool
	AddHTML       bool
	ChartType     string
	GanttSettings *GanttSettings
}

// GanttSettings contains configuration specific to Gantt charts
type GanttSettings struct {
	LabelColumn     string
	StartDateColumn string
	DurationColumn  string
	StatusColumn    string
}

// MarkdownHeader returns the markdown header for Mermaid diagrams
func (settings *Settings) MarkdownHeader() string {
	if settings.AddMarkdown {
		return "```mermaid\n"
	} else {
		return ""
	}
}

// MarkdownFooter returns the markdown footer for Mermaid diagrams
func (settings *Settings) MarkdownFooter() string {
	if settings.AddMarkdown {
		return "\n```"
	} else {
		return ""
	}
}

// HtmlHeader returns the HTML header for Mermaid diagrams
func (settings *Settings) HtmlHeader() string {
	if settings.AddHTML {
		if !scriptset {
			scriptset = true
			return fmt.Sprintf("%s\n<div class='mermaid'>\n", HTMLScript)
		}
		return "<div class='mermaid'>\n"
	} else {
		return ""
	}
}

// HtmlFooter returns the HTML footer for Mermaid diagrams
func (settings *Settings) HtmlFooter() string {
	if settings.AddHTML {
		return "</div>\n"
	} else {
		return ""
	}
}

// Header returns the appropriate header based on settings
func (settings *Settings) Header() string {
	if settings.AddMarkdown {
		return settings.MarkdownHeader()
	} else if settings.AddHTML {
		return settings.HtmlHeader()
	}
	return ""
}

// Footer returns the appropriate footer based on settings
func (settings *Settings) Footer() string {
	if settings.AddMarkdown {
		return settings.MarkdownFooter()
	} else if settings.AddHTML {
		return settings.HtmlFooter()
	}
	return ""
}
