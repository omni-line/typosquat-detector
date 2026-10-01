package pypi

import (
	"bufio"
	"regexp"
	"strings"

	"github.com/omni-line/typosquat-detector/internal/manifest"
)

var (
	reqNameRe = regexp.MustCompile(`(?i)^\s*([A-Za-z0-9][A-Za-z0-9._-]*)(?:\[[^\]]*\])?\s*(.*)$`)
	markerRe  = regexp.MustCompile(`\s*;.*$`)
	sepRe     = regexp.MustCompile(`[-_.]+`)
	tableRe   = regexp.MustCompile(`(?m)^\s*\[([^\]]+)\]\s*$`)
)

// ParseRequirements parses requirements.txt bytes.
func ParseRequirements(data []byte) ([]manifest.Dependency, error) {
	seen := map[string]struct{}{}
	var out []manifest.Dependency
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(line, "://") || strings.HasPrefix(lower, "git+") {
			continue
		}
		line = markerRe.ReplaceAllString(line, "")
		name, ver, ok := splitReq(strings.TrimSpace(line))
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
	return out, sc.Err()
}

func splitReq(line string) (name, version string, ok bool) {
	m := reqNameRe.FindStringSubmatch(line)
	if m == nil {
		return "", "", false
	}
	rest := strings.TrimSpace(m[2])
	if strings.HasPrefix(rest, "@") {
		return "", "", false
	}
	version = strings.TrimSpace(strings.TrimLeft(rest, "=<~>! "))
	return m[1], version, true
}

// ParsePyProject extracts PEP 621 project dependencies.
func ParsePyProject(data []byte) ([]manifest.Dependency, error) {
	text := string(data)
	sections := splitSections(text)
	seen := map[string]struct{}{}
	var out []manifest.Dependency
	add := func(raw, group string, line int) {
		raw = strings.Trim(strings.TrimSpace(raw), `"'`)
		raw = markerRe.ReplaceAllString(raw, "")
		name, ver, ok := splitReq(strings.TrimSpace(raw))
		if !ok || strings.EqualFold(name, "python") {
			return
		}
		key := Normalize(name)
		if _, dup := seen[key]; dup {
			return
		}
		seen[key] = struct{}{}
		out = append(out, manifest.Dependency{Name: name, Version: ver, Group: group, Line: line})
	}
	if body, ok := sections["project"]; ok {
		for _, item := range namedArray(body, "dependencies") {
			add(item.s, "dependencies", item.line)
		}
	}
	if body, ok := sections["project.optional-dependencies"]; ok {
		for _, arr := range allArrays(body) {
			for _, item := range arr {
				add(item.s, "optional-dependencies", item.line)
			}
		}
	}
	if out == nil {
		out = []manifest.Dependency{}
	}
	return out, nil
}

type item struct {
	s    string
	line int
}

func splitSections(text string) map[string]string {
	out := map[string]string{}
	locs := tableRe.FindAllStringSubmatchIndex(text, -1)
	for i, loc := range locs {
		name := text[loc[2]:loc[3]]
		start := loc[1]
		end := len(text)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		out[name] = text[start:end]
	}
	return out
}

func namedArray(body, key string) []item {
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + `\s*=\s*\[`)
	loc := re.FindStringIndex(body)
	if loc == nil {
		return nil
	}
	idx := strings.Index(body[loc[0]:], "[")
	items, ok := readStringArray(body[loc[0]+idx:], manifest.LineAt([]byte(body), loc[0]+idx))
	if !ok {
		return nil
	}
	return items
}

func allArrays(body string) [][]item {
	var out [][]item
	re := regexp.MustCompile(`(?m)^\s*[A-Za-z0-9_-]+\s*=\s*\[`)
	locs := re.FindAllStringIndex(body, -1)
	for _, loc := range locs {
		idx := strings.Index(body[loc[0]:loc[1]], "[")
		items, ok := readStringArray(body[loc[0]+idx:], manifest.LineAt([]byte(body), loc[0]+idx))
		if ok {
			out = append(out, items)
		}
	}
	return out
}

func readStringArray(s string, baseLine int) ([]item, bool) {
	if !strings.HasPrefix(s, "[") {
		return nil, false
	}
	depth := 0
	inStr := false
	var quote rune
	escape := false
	var cur strings.Builder
	var items []item
	line := baseLine
	strStartLine := baseLine
	for _, r := range s {
		if r == '\n' {
			line++
		}
		if inStr {
			if escape {
				cur.WriteRune(r)
				escape = false
				continue
			}
			if r == '\\' {
				escape = true
				continue
			}
			if r == quote {
				inStr = false
				items = append(items, item{s: cur.String(), line: strStartLine})
				cur.Reset()
				continue
			}
			cur.WriteRune(r)
			continue
		}
		switch r {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return items, true
			}
		case '"', '\'':
			inStr = true
			quote = r
			strStartLine = line
		}
	}
	return nil, false
}

// Normalize applies PEP 503-ish normalization.
func Normalize(name string) string {
	return sepRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-")
}
