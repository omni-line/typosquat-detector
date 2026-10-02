package report

import (
	"fmt"
	"os"
	"strings"
)

// ColorMode selects when ANSI colors are emitted.
type ColorMode string

// Color modes.
const (
	ColorAuto   ColorMode = "auto"
	ColorAlways ColorMode = "always"
	ColorNever  ColorMode = "never"
)

// Palette wraps strings in ANSI color codes when enabled.
type Palette struct{ enabled bool }

// NewPalette resolves mode against the environment (NO_COLOR, FORCE_COLOR)
// and whether the destination is a terminal.
func NewPalette(mode ColorMode, isTTY bool) Palette {
	switch mode {
	case ColorAlways:
		return Palette{enabled: true}
	case ColorNever:
		return Palette{enabled: false}
	default:
		if strings.TrimSpace(os.Getenv("NO_COLOR")) != "" {
			return Palette{enabled: false}
		}
		v := strings.ToLower(strings.TrimSpace(os.Getenv("FORCE_COLOR")))
		if v == "1" || v == "true" || v == "yes" {
			return Palette{enabled: true}
		}
		return Palette{enabled: isTTY}
	}
}

func (p Palette) wrap(code, s string) string {
	if !p.enabled || s == "" {
		return s
	}
	return "\033[" + code + "m" + s + "\033[0m"
}

// Style helpers.
func (p Palette) Bold(s string) string       { return p.wrap("1", s) }
func (p Palette) Dim(s string) string        { return p.wrap("2", s) }
func (p Palette) Red(s string) string        { return p.wrap("31", s) }
func (p Palette) Green(s string) string      { return p.wrap("32", s) }
func (p Palette) Yellow(s string) string     { return p.wrap("33", s) }
func (p Palette) Cyan(s string) string       { return p.wrap("36", s) }
func (p Palette) BoldRed(s string) string    { return p.wrap("1;31", s) }
func (p Palette) BoldGreen(s string) string  { return p.wrap("1;32", s) }
func (p Palette) BoldCyan(s string) string   { return p.wrap("1;36", s) }
func (p Palette) BoldYellow(s string) string { return p.wrap("1;33", s) }

// Enabled reports whether colors are emitted.
func (p Palette) Enabled() bool { return p.enabled }

// ParseColorMode parses a --color value.
func ParseColorMode(s string) (ColorMode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "auto":
		return ColorAuto, nil
	case "always", "on", "yes", "true", "1":
		return ColorAlways, nil
	case "never", "off", "no", "false", "0":
		return ColorNever, nil
	default:
		return ColorAuto, fmt.Errorf("invalid --color %q (want auto|always|never)", s)
	}
}
