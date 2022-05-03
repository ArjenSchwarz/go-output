package format

import (
	"fmt"

	"github.com/fatih/color"
)

func (settings *OutputSettings) StringFailure(text interface{}) string {
	warnings := "!!"
	if settings.UseEmoji {
		warnings = "🚨"
	}
	want := fmt.Sprintf("\r\n%v %v %v\r\n\r\n", warnings, text, warnings)
	return settings.StringWarningInline(want)
}

func (settings *OutputSettings) StringWarning(text string) string {
	return settings.StringWarningInline(fmt.Sprintf("%v\r\n", text))
}

func (settings *OutputSettings) StringWarningInline(text string) string {
	if !settings.UseColors {
		return settings.StringBoldInline(fmt.Sprintf("%v", text))
	}
	red := color.New(color.FgRed)
	redbold := red.Add(color.Bold)
	return redbold.Sprintf("%v", text)
}

func (settings *OutputSettings) StringSuccess(text interface{}) string {
	green := color.New(color.FgGreen)
	greenbold := green.Add(color.Bold)
	greenbold.Println("")
	success := "OK"
	if settings.UseEmoji {
		success = "✅"
	}
	return settings.StringPositiveInline(fmt.Sprintf("%v %v\r\n\r\n", success, text))
}

func (settings *OutputSettings) StringPositive(text string) string {
	return settings.StringPositiveInline(fmt.Sprintf("%v\r\n", text))
}

func (settings *OutputSettings) StringPositiveInline(text string) string {
	if !settings.UseColors {
		return settings.StringBoldInline(fmt.Sprintf("%v", text))
	}
	green := color.New(color.FgGreen)
	greenbold := green.Add(color.Bold)
	return greenbold.Sprintf("%v", text)
}

func (settings *OutputSettings) StringInfo(text interface{}) string {
	fmt.Println("")
	info := ""
	if settings.UseEmoji {
		info = "ℹ️"
	}
	return fmt.Sprintf("%v  %v\r\n\r\n", info, text)
}

func (settings *OutputSettings) StringBold(text string) string {
	return settings.StringBoldInline(fmt.Sprintf("%v\r\n", text))
}

func (settings *OutputSettings) StringBoldInline(text string) string {
	bold := color.New(color.Bold)
	return bold.Sprintf("%v", text)
}
