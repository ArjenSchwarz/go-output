package format

import (
	"fmt"

	"github.com/fatih/color"
)

// StringFailure returns a formatted failure message
func (settings *OutputSettings) StringFailure(text interface{}) string {
	warnings := "!!"
	if settings.UseEmoji {
		warnings = "🚨"
	}
	want := fmt.Sprintf("\r\n%v %v %v\r\n\r\n", warnings, text, warnings)
	return settings.StringWarningInline(want)
}

// StringWarning returns a formatted warning message
func (settings *OutputSettings) StringWarning(text string) string {
	return settings.StringWarningInline(fmt.Sprintf("%v\r\n", text))
}

// StringWarningInline returns a formatted inline warning message
func (settings *OutputSettings) StringWarningInline(text string) string {
	if !settings.UseColors {
		return settings.StringBoldInline(fmt.Sprintf("%v", text))
	}
	red := color.New(color.FgRed)
	redbold := red.Add(color.Bold)
	return redbold.Sprintf("%v", text)
}

// StringSuccess returns a formatted success message
func (settings *OutputSettings) StringSuccess(text interface{}) string {
	green := color.New(color.FgGreen)
	greenbold := green.Add(color.Bold)
	_, _ = greenbold.Println("")
	success := "OK"
	if settings.UseEmoji {
		success = "✅"
	}
	return settings.StringPositiveInline(fmt.Sprintf("%v %v\r\n\r\n", success, text))
}

// StringPositive returns a formatted positive message
func (settings *OutputSettings) StringPositive(text string) string {
	return settings.StringPositiveInline(fmt.Sprintf("%v\r\n", text))
}

// StringPositiveInline returns a formatted inline positive message
func (settings *OutputSettings) StringPositiveInline(text string) string {
	if !settings.UseColors {
		return settings.StringBoldInline(fmt.Sprintf("%v", text))
	}
	green := color.New(color.FgGreen)
	greenbold := green.Add(color.Bold)
	return greenbold.Sprintf("%v", text)
}

// StringInfo returns a formatted info message
func (settings *OutputSettings) StringInfo(text interface{}) string {
	fmt.Println("")
	info := ""
	if settings.UseEmoji {
		info = "ℹ️"
	}
	return fmt.Sprintf("%v  %v\r\n\r\n", info, text)
}

// StringBold returns a formatted bold message
func (settings *OutputSettings) StringBold(text string) string {
	return settings.StringBoldInline(fmt.Sprintf("%v\r\n", text))
}

// StringBoldInline returns a formatted inline bold message
func (settings *OutputSettings) StringBoldInline(text string) string {
	bold := color.New(color.Bold)
	return bold.Sprintf("%v", text)
}
