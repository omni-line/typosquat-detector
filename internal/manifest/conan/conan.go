// Package conan parses conanfile.txt [requires] entries.
package conan

import (
	"bufio"
	"strings"

	"github.com/omni-line/typosquat-detector/internal/manifest"
)

// Parse extracts recipe names from conanfile.txt [requires] / [tool_requires].
func Parse(data []byte) ([]manifest.Dependency, error) {
	seen := map[string]struct{}{}
	var out []manifest.Dependency
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	lineNo := 0
	section := ""
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.Trim(line, "[]"))
			continue
		}
		if section != "requires" && section != "tool_requires" && section != "build_requires" {
			continue
		}
		name, ver := splitRef(line)
		if name == "" {
			continue
		}
		key := Normalize(name)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, manifest.Dependency{
			Name:    name,
			Version: ver,
			Group:   section,
			Line:    lineNo,
		})
	}
	return out, sc.Err()
}

func splitRef(s string) (name, version string) {
	// name/version[@user/channel][#rrev]
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	if i := strings.IndexByte(s, '#'); i >= 0 {
		s = s[:i]
	}
	if i := strings.IndexByte(s, '@'); i >= 0 {
		s = s[:i]
	}
	if i := strings.IndexByte(s, '/'); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}

// Normalize lowercases Conan recipe names.
func Normalize(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
