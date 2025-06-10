package format

import "strings"

// NewProgress returns a progress implementation based on the output format and settings.
func NewProgress(settings *OutputSettings) Progress {
	if settings == nil {
		settings = NewOutputSettings()
	}
	if !settings.ProgressEnabled {
		return newNoOpProgress(settings)
	}
	switch strings.ToLower(settings.OutputFormat) {
	case "json", "yaml", "csv", "dot":
		return newNoOpProgress(settings)
	case "table", "markdown", "html":
		return newPrettyProgress(settings)
	default:
		return newNoOpProgress(settings)
	}
}
