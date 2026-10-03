// Package gomod parses require directives from go.mod files.
package gomod

import (
	"bufio"
	"strings"
	"unicode"

	"github.com/omni-line/typosquat-detector/internal/manifest"
)

// Parse extracts module paths from require directives in go.mod bytes.
func Parse(data []byte) ([]manifest.Dependency, error) {
	seen := map[string]struct{}{}
	var out []manifest.Dependency
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	lineNo := 0
	inRequire := false
	for sc.Scan() {
		lineNo++
		line := stripComment(sc.Text())
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if inRequire {
			if trimmed == ")" {
				inRequire = false
				continue
			}
			path, ver, ok := parseRequireEntry(trimmed)
			if !ok {
				continue
			}
			if _, dup := seen[path]; dup {
				continue
			}
			seen[path] = struct{}{}
			out = append(out, manifest.Dependency{Name: path, Version: ver, Group: "require", Line: lineNo})
			continue
		}
		lower := strings.ToLower(trimmed)
		switch {
		case lower == "require (":
			inRequire = true
		case strings.HasPrefix(lower, "require "):
			rest := strings.TrimSpace(trimmed[len("require"):])
			if rest == "(" {
				inRequire = true
				continue
			}
			path, ver, ok := parseRequireEntry(rest)
			if !ok {
				continue
			}
			if _, dup := seen[path]; dup {
				continue
			}
			seen[path] = struct{}{}
			out = append(out, manifest.Dependency{Name: path, Version: ver, Group: "require", Line: lineNo})
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []manifest.Dependency{}
	}
	return out, nil
}

func stripComment(line string) string {
	inQuote := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		if c == '"' {
			inQuote = !inQuote
			continue
		}
		if !inQuote && c == '/' && i+1 < len(line) && line[i+1] == '/' {
			return line[:i]
		}
	}
	return line
}

func parseRequireEntry(s string) (path, version string, ok bool) {
	s = strings.TrimSpace(s)
	fields := strings.Fields(s)
	if len(fields) < 2 {
		return "", "", false
	}
	path, version = fields[0], fields[1]
	if path == "=>" || version == "=>" || !looksLikeModulePath(path) {
		return "", "", false
	}
	return path, version, true
}

func looksLikeModulePath(p string) bool {
	if p == "" || strings.Contains(p, "://") || !strings.Contains(p, ".") {
		return false
	}
	for _, r := range p {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// Normalize returns the module path trimmed (paths are case-sensitive).
func Normalize(name string) string {
	return strings.TrimSpace(name)
}
