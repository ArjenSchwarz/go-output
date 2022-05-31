package mermaid

import "fmt"

const HTMLScript = `<script src="https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js"></script>
	<script>
		mermaid.initialize({ startOnLoad: true });
	</script>`

var scriptset = false

type Settings struct {
	AddMarkdown   bool
	AddHTML       bool
	ChartType     string
	GanttSettings *GanttSettings
}

type GanttSettings struct {
	LabelColumn     string
	StartDateColumn string
	DurationColumn  string
	StatusColumn    string
}

func (settings *Settings) MarkdownHeader() string {
	if settings.AddMarkdown {
		return "```mermaid\n"
	} else {
		return ""
	}
}

func (settings *Settings) MarkdownFooter() string {
	if settings.AddMarkdown {
		return "\n```"
	} else {
		return ""
	}
}

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

func (settings *Settings) HtmlFooter() string {
	if settings.AddHTML {
		return "</div>\n"
	} else {
		return ""
	}
}

func (settings *Settings) Header() string {
	if settings.AddMarkdown {
		return settings.MarkdownHeader()
	} else if settings.AddHTML {
		return settings.HtmlHeader()
	}
	return ""
}

func (settings *Settings) Footer() string {
	if settings.AddMarkdown {
		return settings.MarkdownFooter()
	} else if settings.AddHTML {
		return settings.HtmlFooter()
	}
	return ""
}
