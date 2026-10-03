// Package cargo parses Cargo.toml dependency tables.
package cargo

import (
	"bufio"
	"regexp"
	"strings"

	"github.com/omni-line/typosquat-detector/internal/manifest"
)

var (
	tableRe = regexp.MustCompile(`(?i)^\s*\[([^\]]+)\]\s*$`)
	depRe   = regexp.MustCompile(`(?i)^\s*([A-Za-z0-9_-]+)\s*=\s*(.+)$`)
)

var depTables = map[string]string{
	"dependencies":       "dependencies",
	"dev-dependencies":   "dev-dependencies",
	"build-dependencies": "build-dependencies",
}

// Parse extracts crates.io-resolvable dependencies from Cargo.toml bytes.
func Parse(data []byte) ([]manifest.Dependency, error) {
	seen := map[string]struct{}{}
	var out []manifest.Dependency
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	lineNo := 0
	group := ""
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if m := tableRe.FindStringSubmatch(line); m != nil {
			name := strings.ToLower(strings.TrimSpace(m[1]))
			// Skip target-specific / workspace nested tables for v1.
			if g, ok := depTables[name]; ok {
				group = g
			} else {
				group = ""
			}
			continue
		}
		if group == "" {
			continue
		}
		m := depRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name, spec := m[1], strings.TrimSpace(m[2])
		if isNonRegistry(spec) {
			continue
		}
		key := Normalize(name)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, manifest.Dependency{
			Name:    name,
			Version: versionOf(spec),
			Group:   group,
			Line:    lineNo,
		})
	}
	return out, sc.Err()
}

func isNonRegistry(spec string) bool {
	lower := strings.ToLower(spec)
	if strings.Contains(lower, "path") || strings.Contains(lower, "git") {
		return true
	}
	return strings.Contains(lower, "registry") && !strings.Contains(lower, "crates-io")
}

func versionOf(spec string) string {
	spec = strings.TrimSpace(spec)
	if strings.HasPrefix(spec, "\"") || strings.HasPrefix(spec, "'") {
		return strings.Trim(spec, `"'`)
	}
	if i := strings.Index(spec, "version"); i >= 0 {
		rest := spec[i+len("version"):]
		rest = strings.TrimLeft(rest, " =")
		if len(rest) > 0 && (rest[0] == '"' || rest[0] == '\'') {
			q := rest[0]
			if j := strings.IndexByte(rest[1:], q); j >= 0 {
				return rest[1 : 1+j]
			}
		}
	}
	return ""
}

// Normalize lowercases crate names (crates.io identity).
func Normalize(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
