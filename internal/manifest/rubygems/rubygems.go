// Package rubygems parses Gemfile gem declarations.
package rubygems

import (
	"bufio"
	"regexp"
	"strings"

	"github.com/omni-line/typosquat-detector/internal/manifest"
)

var gemRe = regexp.MustCompile(`(?i)^\s*gem\s+['"]([^'"]+)['"]\s*(?:,(.*))?$`)

// Parse extracts rubygems.org-resolvable gems from a Gemfile.
func Parse(data []byte) ([]manifest.Dependency, error) {
	seen := map[string]struct{}{}
	var out []manifest.Dependency
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m := gemRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name, rest := m[1], m[2]
		lower := strings.ToLower(rest)
		if strings.Contains(lower, "git:") || strings.Contains(lower, "github:") ||
			strings.Contains(lower, "path:") || strings.Contains(lower, "source:") {
			continue
		}
		key := Normalize(name)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, manifest.Dependency{
			Name:    name,
			Version: firstQuoted(rest),
			Group:   "gem",
			Line:    lineNo,
		})
	}
	return out, sc.Err()
}

func firstQuoted(s string) string {
	s = strings.TrimSpace(s)
	for _, q := range []byte{'"', '\''} {
		if i := strings.IndexByte(s, q); i >= 0 {
			if j := strings.IndexByte(s[i+1:], q); j >= 0 {
				return s[i+1 : i+1+j]
			}
		}
	}
	return ""
}

// Normalize lowercases gem names.
func Normalize(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
