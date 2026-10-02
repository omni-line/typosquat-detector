package report

import (
	"fmt"
	"strings"
	"unicode"
)

// clean makes untrusted manifest content safe to print to a terminal.
//
// Control characters (C0, DEL, C1 — including ESC and the 8-bit CSI) could
// inject terminal escape sequences, and format characters (bidi overrides,
// zero-width spaces) could make a malicious name render as a legitimate one.
// Both are shown as visible \uXXXX escapes rather than dropped, so a reviewer
// can see that something unusual is in the name.
func clean(s string) string {
	if !needsCleaning(s) {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		if r == unicode.ReplacementChar || unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			fmt.Fprintf(&b, `\u%04x`, r)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func needsCleaning(s string) bool {
	for i := 0; i < len(s); i++ {
		if c := s[i]; c < 0x20 || c >= 0x7f {
			return true
		}
	}
	return false
}

func cleanAll(in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = clean(s)
	}
	return out
}
