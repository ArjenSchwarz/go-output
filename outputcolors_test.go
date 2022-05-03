package format

import (
	"testing"

	"github.com/fatih/color"
)

func TestOutputSettings_StringFailure(t *testing.T) {
	type args struct {
		text string
	}
	red := color.New(color.FgRed)
	redbold := red.Add(color.Bold)
	bold := color.New(color.Bold)
	tests := []struct {
		name     string
		settings *OutputSettings
		args     args
		want     string
	}{
		{"WithColorsNoEmoji", &OutputSettings{
			UseColors: true,
			UseEmoji:  false,
		}, args{text: "random text"}, redbold.Sprint("\r\n!! random text !!\r\n\r\n")},
		{"NoColorsNoEmoji", &OutputSettings{
			UseColors: false,
			UseEmoji:  false,
		}, args{text: "random text"}, bold.Sprint("\r\n!! random text !!\r\n\r\n")},
		{"WithColorsWithEmoji", &OutputSettings{
			UseColors: true,
			UseEmoji:  true,
		}, args{text: "random text"}, redbold.Sprint("\r\n🚨 random text 🚨\r\n\r\n")},
		{"NoColorsNoEmoji", &OutputSettings{
			UseColors: false,
			UseEmoji:  true,
		}, args{text: "random text"}, bold.Sprint("\r\n🚨 random text 🚨\r\n\r\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.settings.StringFailure(tt.args.text); got != tt.want {
				t.Errorf("OutputSettings.StringWarning() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOutputSettings_StringWarning(t *testing.T) {
	type args struct {
		text string
	}
	red := color.New(color.FgRed)
	redbold := red.Add(color.Bold)
	bold := color.New(color.Bold)
	tests := []struct {
		name     string
		settings *OutputSettings
		args     args
		want     string
	}{
		{"WithColors", &OutputSettings{
			UseColors: true,
		}, args{text: "random text"}, redbold.Sprint("random text\r\n")},
		{"NoColors", &OutputSettings{
			UseColors: false,
		}, args{text: "random text"}, bold.Sprint("random text\r\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.settings.StringWarning(tt.args.text); got != tt.want {
				t.Errorf("OutputSettings.StringWarning() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOutputSettings_StringWarningInline(t *testing.T) {
	type args struct {
		text string
	}
	red := color.New(color.FgRed)
	redbold := red.Add(color.Bold)
	bold := color.New(color.Bold)
	tests := []struct {
		name     string
		settings *OutputSettings
		args     args
		want     string
	}{
		{"WithColors", &OutputSettings{
			UseColors: true,
		}, args{text: "random text"}, redbold.Sprint("random text")},
		{"NoColors", &OutputSettings{
			UseColors: false,
		}, args{text: "random text"}, bold.Sprint("random text")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.settings.StringWarningInline(tt.args.text); got != tt.want {
				t.Errorf("OutputSettings.StringWarningInline() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOutputSettings_StringSuccess(t *testing.T) {
	type args struct {
		text string
	}
	green := color.New(color.FgGreen)
	greenbold := green.Add(color.Bold)
	bold := color.New(color.Bold)
	tests := []struct {
		name     string
		settings *OutputSettings
		args     args
		want     string
	}{
		{"WithColorsNoEmoji", &OutputSettings{
			UseColors: true,
			UseEmoji:  false,
		}, args{text: "random text"}, greenbold.Sprint("OK random text\r\n\r\n")},
		{"NoColorsNoEmoji", &OutputSettings{
			UseColors: false,
			UseEmoji:  false,
		}, args{text: "random text"}, bold.Sprint("OK random text\r\n\r\n")},
		{"WithColorsWithEmoji", &OutputSettings{
			UseColors: true,
			UseEmoji:  true,
		}, args{text: "random text"}, greenbold.Sprint("✅ random text\r\n\r\n")},
		{"NoColorsWithEmoji", &OutputSettings{
			UseColors: false,
			UseEmoji:  true,
		}, args{text: "random text"}, bold.Sprint("✅ random text\r\n\r\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.settings.StringSuccess(tt.args.text); got != tt.want {
				t.Errorf("OutputSettings.StringSuccess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOutputSettings_StringPositive(t *testing.T) {
	type args struct {
		text string
	}
	green := color.New(color.FgGreen)
	greenbold := green.Add(color.Bold)
	bold := color.New(color.Bold)
	tests := []struct {
		name     string
		settings *OutputSettings
		args     args
		want     string
	}{
		{"WithColors", &OutputSettings{
			UseColors: true,
		}, args{text: "random text"}, greenbold.Sprint("random text\r\n")},
		{"NoColors", &OutputSettings{
			UseColors: false,
		}, args{text: "random text"}, bold.Sprint("random text\r\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.settings.StringPositive(tt.args.text); got != tt.want {
				t.Errorf("OutputSettings.StringPositive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOutputSettings_StringPositiveInline(t *testing.T) {
	type args struct {
		text string
	}
	green := color.New(color.FgGreen)
	greenbold := green.Add(color.Bold)
	bold := color.New(color.Bold)
	tests := []struct {
		name     string
		settings *OutputSettings
		args     args
		want     string
	}{
		{"WithColors", &OutputSettings{
			UseColors: true,
		}, args{text: "random text"}, greenbold.Sprint("random text")},
		{"NoColors", &OutputSettings{
			UseColors: false,
		}, args{text: "random text"}, bold.Sprint("random text")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.settings.StringPositiveInline(tt.args.text); got != tt.want {
				t.Errorf("OutputSettings.StringPositiveInline() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOutputSettings_StringInfo(t *testing.T) {
	type args struct {
		text string
	}
	tests := []struct {
		name     string
		settings *OutputSettings
		args     args
		want     string
	}{
		{"withEmoji", &OutputSettings{
			UseEmoji: true,
		}, args{text: "random text"}, "ℹ️  random text\r\n\r\n"},
		{"noEmoji", &OutputSettings{
			UseEmoji: false,
		}, args{text: "random text"}, "  random text\r\n\r\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.settings.StringInfo(tt.args.text); got != tt.want {
				t.Errorf("OutputSettings.StringInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOutputSettings_StringBold(t *testing.T) {
	type args struct {
		text string
	}
	bold := color.New(color.Bold)
	tests := []struct {
		name     string
		settings *OutputSettings
		args     args
		want     string
	}{
		{"WithColors", &OutputSettings{
			UseColors: true,
		}, args{text: "random text"}, bold.Sprint("random text\r\n")},
		{"NoColors", &OutputSettings{
			UseColors: false,
		}, args{text: "random text"}, bold.Sprint("random text\r\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.settings.StringBold(tt.args.text); got != tt.want {
				t.Errorf("OutputSettings.StringBold() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOutputSettings_StringBoldInline(t *testing.T) {
	type args struct {
		text string
	}
	bold := color.New(color.Bold)
	tests := []struct {
		name     string
		settings *OutputSettings
		args     args
		want     string
	}{
		{"WithColors", &OutputSettings{
			UseColors: true,
		}, args{text: "random text"}, bold.Sprint("random text")},
		{"NoColors", &OutputSettings{
			UseColors: false,
		}, args{text: "random text"}, bold.Sprint("random text")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.settings.StringBoldInline(tt.args.text); got != tt.want {
				t.Errorf("OutputSettings.StringBoldInline() = %v, want %v", got, tt.want)
			}
		})
	}
}
