// Package pypi parses requirements files and pyproject.toml manifests.
//
// The pyproject parser is a deliberately small TOML subset reader: it only
// needs table boundaries, string arrays, and simple key = value lines, and it
// must never fail hard on TOML features it does not understand.
package pypi

import (
	"regexp"
	"strings"

	"github.com/omni-line/typosquat-detector/internal/manifest"
)

var (
	reqNameRe  = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9._-]*)\s*(?:\[[^\]]*\])?\s*(.*)$`)
	commentRe  = regexp.MustCompile(`(^|\s)#.*$`)
	sepRe      = regexp.MustCompile(`[-_.]+`)
	tableRe    = regexp.MustCompile(`(?m)^[ \t]*\[\[?([^\[\]\n]+)\]\]?[ \t]*(?:#.*)?$`)
	arrayKeyRe = regexp.MustCompile(`(?m)^[ \t]*("?)([A-Za-z0-9_.-]+)("?)[ \t]*=[ \t]*\[`)
	poetryKVRe = regexp.MustCompile(`(?m)^[ \t]*["']?([A-Za-z0-9][A-Za-z0-9._-]*)["']?[ \t]*=[ \t]*(.*)$`)
	tomlStrRe  = regexp.MustCompile(`^["']([^"']*)["']`)
	versionRe  = regexp.MustCompile(`\bversion\s*=\s*["']([^"']*)["']`)
	localDepRe = regexp.MustCompile(`\b(path|git|url|file)\s*=`)
)

// ParseRequirements parses requirements.txt bytes.
func ParseRequirements(data []byte) ([]manifest.Dependency, error) {
	seen := map[string]struct{}{}
	var out []manifest.Dependency
	lines := strings.Split(string(data), "\n")
	for i := 0; i < len(lines); i++ {
		lineNo := i + 1
		line := strings.TrimRight(lines[i], "\r")
		for strings.HasSuffix(line, `\`) && i+1 < len(lines) {
			i++
			line = strings.TrimSuffix(line, `\`) + " " + strings.TrimRight(lines[i], "\r")
		}
		line = strings.TrimSpace(commentRe.ReplaceAllString(line, ""))
		if line == "" || strings.HasPrefix(line, "-") {
			continue
		}
		if strings.Contains(line, "://") || strings.HasPrefix(strings.ToLower(line), "git+") {
			continue
		}
		name, ver, ok := splitReq(line)
		if !ok {
			continue
		}
		key := Normalize(name)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, manifest.Dependency{Name: name, Version: ver, Group: "requirements", Line: lineNo})
	}
	return out, nil
}

// splitReq parses a PEP 508 requirement into name and version spec. Direct
// references ("name @ url") are skipped because they never hit the index.
func splitReq(req string) (name, version string, ok bool) {
	req = strings.TrimSpace(req)
	if i := strings.IndexByte(req, ';'); i >= 0 {
		req = req[:i]
	}
	if i := strings.Index(req, " --"); i >= 0 {
		req = req[:i]
	}
	m := reqNameRe.FindStringSubmatch(strings.TrimSpace(req))
	if m == nil {
		return "", "", false
	}
	rest := strings.TrimSpace(m[2])
	if strings.HasPrefix(rest, "@") {
		return "", "", false
	}
	if rest != "" && !strings.ContainsAny(rest[:1], "=<>~!(,") {
		return "", "", false
	}
	version = strings.Trim(rest, "=<~>!() ")
	return m[1], version, true
}

type section struct {
	name   string
	body   string
	offset int
}

// ParsePyProject extracts dependencies from PEP 621 ([project]), PEP 518
// ([build-system]), PEP 735 ([dependency-groups]) and Poetry tables.
func ParsePyProject(data []byte) ([]manifest.Dependency, error) {
	text := string(data)
	seen := map[string]struct{}{}
	out := []manifest.Dependency{}
	lineOf := func(off int) int { return manifest.LineAt(data, off) }
	addReq := func(raw, group string, off int) {
		name, ver, ok := splitReq(raw)
		if !ok || strings.EqualFold(name, "python") {
			return
		}
		key := Normalize(name)
		if _, dup := seen[key]; dup {
			return
		}
		seen[key] = struct{}{}
		out = append(out, manifest.Dependency{Name: name, Version: ver, Group: group, Line: lineOf(off)})
	}

	for _, sec := range splitSections(text) {
		switch {
		case sec.name == "project":
			for _, it := range namedArray(sec, "dependencies") {
				addReq(it.s, "dependencies", it.off)
			}
		case sec.name == "project.optional-dependencies":
			for _, it := range allArrays(sec) {
				addReq(it.s, "optional-dependencies", it.off)
			}
		case sec.name == "build-system":
			for _, it := range namedArray(sec, "requires") {
				addReq(it.s, "build-system", it.off)
			}
		case sec.name == "dependency-groups":
			for _, it := range allArrays(sec) {
				addReq(it.s, "dependency-groups", it.off)
			}
		case isPoetrySection(sec.name):
			for _, d := range poetryDeps(sec) {
				if strings.EqualFold(d.Name, "python") {
					continue
				}
				key := Normalize(d.Name)
				if _, dup := seen[key]; dup {
					continue
				}
				seen[key] = struct{}{}
				d.Line = lineOf(d.Line)
				out = append(out, d)
			}
		}
	}
	return out, nil
}

func isPoetrySection(name string) bool {
	if name == "tool.poetry.dependencies" || name == "tool.poetry.dev-dependencies" {
		return true
	}
	return strings.HasPrefix(name, "tool.poetry.group.") && strings.HasSuffix(name, ".dependencies")
}

// poetryDeps parses `name = "spec"` / `name = { version = "..." }` lines.
// Line holds the absolute byte offset; the caller converts it.
func poetryDeps(sec section) []manifest.Dependency {
	var out []manifest.Dependency
	for _, loc := range poetryKVRe.FindAllStringSubmatchIndex(sec.body, -1) {
		name := sec.body[loc[2]:loc[3]]
		value := strings.TrimSpace(sec.body[loc[4]:loc[5]])
		var version string
		switch {
		case strings.HasPrefix(value, "{"):
			if localDepRe.MatchString(value) {
				continue
			}
			if m := versionRe.FindStringSubmatch(value); m != nil {
				version = m[1]
			}
		case strings.HasPrefix(value, "["):
		default:
			if m := tomlStrRe.FindStringSubmatch(value); m != nil {
				version = m[1]
			}
		}
		out = append(out, manifest.Dependency{
			Name:    name,
			Version: strings.Trim(version, "=<~>!^ "),
			Group:   "poetry",
			Line:    sec.offset + loc[0],
		})
	}
	return out
}

func splitSections(text string) []section {
	locs := tableRe.FindAllStringSubmatchIndex(text, -1)
	out := make([]section, 0, len(locs))
	for i, loc := range locs {
		start := loc[1]
		end := len(text)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		out = append(out, section{
			name:   tableName(text[loc[2]:loc[3]]),
			body:   text[start:end],
			offset: start,
		})
	}
	return out
}

// tableName canonicalizes `[ tool . "poetry" ]` to "tool.poetry".
func tableName(raw string) string {
	parts := strings.Split(raw, ".")
	for i, p := range parts {
		parts[i] = strings.Trim(strings.TrimSpace(p), `"'`)
	}
	return strings.Join(parts, ".")
}

type item struct {
	s   string
	off int
}

func namedArray(sec section, key string) []item {
	for _, loc := range arrayKeyRe.FindAllStringSubmatchIndex(sec.body, -1) {
		if sec.body[loc[4]:loc[5]] != key {
			continue
		}
		open := loc[1] - 1
		items, ok := readStringArray(sec.body[open:])
		if !ok {
			return nil
		}
		return shift(items, sec.offset+open)
	}
	return nil
}

func allArrays(sec section) []item {
	var out []item
	for _, loc := range arrayKeyRe.FindAllStringSubmatchIndex(sec.body, -1) {
		open := loc[1] - 1
		if items, ok := readStringArray(sec.body[open:]); ok {
			out = append(out, shift(items, sec.offset+open)...)
		}
	}
	return out
}

func shift(items []item, by int) []item {
	for i := range items {
		items[i].off += by
	}
	return items
}

// readStringArray reads a TOML array starting at s[0] == '[' and returns its
// top-level string elements with byte offsets relative to s. Comments and
// strings inside inline tables (e.g. PEP 735 {include-group = "x"}) are
// ignored.
func readStringArray(s string) ([]item, bool) {
	if !strings.HasPrefix(s, "[") {
		return nil, false
	}
	var (
		depth, braces int
		inStr         bool
		inComment     bool
		escape        bool
		quote         byte
		start         int
		cur           strings.Builder
		items         []item
	)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inComment {
			if c == '\n' {
				inComment = false
			}
			continue
		}
		if inStr {
			switch {
			case escape:
				cur.WriteByte(c)
				escape = false
			case c == '\\' && quote == '"':
				escape = true
			case c == quote:
				inStr = false
				if braces == 0 {
					items = append(items, item{s: cur.String(), off: start})
				}
				cur.Reset()
			default:
				cur.WriteByte(c)
			}
			continue
		}
		switch c {
		case '#':
			inComment = true
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return items, true
			}
		case '{':
			braces++
		case '}':
			braces--
		case '"', '\'':
			inStr = true
			quote = c
			start = i
		}
	}
	return nil, false
}

// Normalize applies PEP 503 normalization.
func Normalize(name string) string {
	return sepRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-")
}
