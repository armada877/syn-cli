package tui

import (
	"charm.land/lipgloss/v2"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"
)

type Theme struct {
	Accent, Selection, Muted, Emphasis, Error color.Color
	Mode                                      string
	NoColor                                   bool
}

type Styles struct{ Selected, Focus, Read, Unread, Error, Emphasis lipgloss.Style }

func defaultTheme() Theme { return Theme{Accent: lipgloss.BrightBlue, Error: lipgloss.Red} }

func themePath(home string) string {
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".local", "state", "omarchy", "current", "theme", "syn.toml")
}

func loadTheme(path string, noColor bool) Theme {
	theme := defaultTheme()
	theme.NoColor = noColor
	if noColor {
		return theme
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return theme
	}
	values := parseThemeFile(string(data))
	theme.Mode = values["mode"]
	if theme.Mode != "dark" && theme.Mode != "light" {
		theme.Mode = ""
	}
	theme.Emphasis, _ = themeColor(values, "bright_foreground", "foreground")
	theme.Muted, _ = themeColor(values, "muted")
	if c, ok := themeColor(values, "error"); ok {
		theme.Error = c
	}
	if c, ok := themeColor(values, "accent"); ok {
		usable := theme.Emphasis == nil || colorDistance(c, theme.Emphasis) >= 64
		if bg, ok := themeColor(values, "background"); ok {
			reference, ok := themeColor(values, "bright_blue", "blue")
			if !ok {
				reference = lipgloss.BrightBlue
			}
			usable = usable && (contrastRatio(c, bg) >= 5 || contrastRatio(c, bg) >= contrastRatio(reference, bg))
		}
		if usable {
			theme.Accent = c
		}
	}
	if c, ok := themeColor(values, "selection"); ok && contrastRatio(theme.Accent, c) >= 4.5 {
		theme.Selection = c
	}
	return theme
}

func makeStyles(theme Theme) Styles {
	plain := lipgloss.NewStyle()
	if theme.NoColor {
		return Styles{plain, plain, plain, plain, plain, plain}
	}
	selected := plain.Foreground(theme.Accent).Bold(true)
	if theme.Selection != nil {
		selected = selected.Background(theme.Selection)
	}
	read := plain.Faint(true)
	if theme.Muted != nil {
		read = plain.Foreground(theme.Muted)
	}
	emphasis := plain.Bold(true)
	if theme.Emphasis != nil {
		emphasis = emphasis.Foreground(theme.Emphasis)
	}
	return Styles{selected, plain.Foreground(theme.Accent).Bold(true), read, plain.Foreground(theme.Accent), plain.Foreground(theme.Error), emphasis}
}

func parseThemeFile(data string) map[string]string {
	values := map[string]string{}
	for _, line := range strings.Split(data, "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) < 2 || (value[0] != '\'' && value[0] != '"') {
			continue
		}
		end := strings.IndexByte(value[1:], value[0])
		if end < 0 {
			continue
		}
		tail := strings.TrimSpace(value[end+2:])
		if tail != "" && !strings.HasPrefix(tail, "#") {
			continue
		}
		values[strings.TrimSpace(key)] = value[1 : end+1]
	}
	return values
}

func contrastRatio(a, b color.Color) float64 {
	luminance := func(c color.Color) float64 {
		r, g, b, _ := c.RGBA()
		linear := func(v uint32) float64 {
			x := float64(v>>8) / 255
			if x <= 0.04045 {
				return x / 12.92
			}
			return math.Pow((x+0.055)/1.055, 2.4)
		}
		return 0.2126*linear(r) + 0.7152*linear(g) + 0.0722*linear(b)
	}
	x, y := luminance(a), luminance(b)
	return (math.Max(x, y) + 0.05) / (math.Min(x, y) + 0.05)
}

func colorDistance(a, b color.Color) float64 {
	ar, ag, ab, _ := a.RGBA()
	br, bg, bb, _ := b.RGBA()
	d := func(x, y uint32) float64 { return float64(x>>8) - float64(y>>8) }
	return math.Sqrt(d(ar, br)*d(ar, br) + d(ag, bg)*d(ag, bg) + d(ab, bb)*d(ab, bb))
}

func themeColor(values map[string]string, keys ...string) (color.Color, bool) {
	for _, key := range keys {
		if hex, ok := values[key]; ok && isHexColor(hex) {
			return lipgloss.Color(hex), true
		}
	}
	return nil, false
}

func isHexColor(s string) bool {
	if !strings.HasPrefix(s, "#") {
		return false
	}
	digits := s[1:]
	if len(digits) != 3 && len(digits) != 6 {
		return false
	}
	for _, r := range digits {
		isDigit := r >= '0' && r <= '9'
		isLower := r >= 'a' && r <= 'f'
		isUpper := r >= 'A' && r <= 'F'
		if !isDigit && !isLower && !isUpper {
			return false
		}
	}
	return true
}
