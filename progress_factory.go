package format

import "strings"

// NewProgress returns a progress implementation based on the output format and
// settings. Use the returned Progress instance to update progress from your
// application. The implementation chosen depends on the configured output
// format.
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
